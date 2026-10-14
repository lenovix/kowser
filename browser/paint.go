package main

import (
	"image"
	"image/color"
	"strings"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// PaintContext carries the state needed to paint a layout tree into Gio ops.
type PaintContext struct {
	Theme    *material.Theme
	ScrollY  float32
	BaseURL  string
	GetImage func(src string) image.Image
	Links    map[*Node]*widget.Clickable
}

// PaintLayout walks the layout box tree and emits Gio paint operations.
// It returns the total content height and, if a link was clicked this frame,
// the raw href destination.
func PaintLayout(gtx layout.Context, box *LayoutBox, pc *PaintContext) (float32, string) {
	if box == nil {
		return 0, ""
	}
	clicked := ""
	totalHeight := paintBox(gtx, box, 0, 0, pc, &clicked)
	return totalHeight, clicked
}

// paintBox paints one box and recurses into children.
// parentX/parentY are the accumulated absolute origins (pre-scroll for Y).
func paintBox(gtx layout.Context, box *LayoutBox, parentX, parentY float32, pc *PaintContext, clicked *string) float32 {
	if box == nil {
		return 0
	}

	node := box.Node
	absX := parentX + box.Dims.X
	absY := parentY + box.Dims.Y
	viewY := absY - pc.ScrollY

	// 1. Background (padding box) if the computed style sets one.
	if node != nil && node.Computed != nil && node.Computed.BackgroundColor != nil {
		bgW := box.Dims.Width + box.Dims.Padding.Left + box.Dims.Padding.Right
		bgH := box.Dims.Height + box.Dims.Padding.Top + box.Dims.Padding.Bottom
		if bgW > 0 && bgH > 0 {
			bgX := absX - box.Dims.Padding.Left
			bgY := viewY - box.Dims.Padding.Top

			stack := op.Offset(image.Pt(int(bgX), int(bgY))).Push(gtx.Ops)
			paint.FillShape(gtx.Ops, *node.Computed.BackgroundColor,
				clip.Rect{Max: image.Pt(int(bgW), int(bgH))}.Op())
			stack.Pop()
		}
	}

	// 2. Border (simple uniform width, solid color derived from foreground).
	if node != nil && node.Computed != nil && box.Dims.BorderWidth() > 0 {
		paintBorder(gtx, absX, viewY, box, node.Computed.Color)
	}

	// 3. Image boxes: <img> elements render the cached bitmap.
	if node != nil && node.Type == NodeElement && node.Tag == "img" {
		if src := node.Attribute("src"); src != "" && pc.GetImage != nil {
			if pc.BaseURL != "" {
				src = resolveURL(pc.BaseURL, src)
			}
			if img := pc.GetImage(src); img != nil {
				PaintImageBox(gtx, img, absX, viewY, box.Dims.Width, box.Dims.Height)
			} else if alt := node.Attribute("alt"); alt != "" {
				lbl := material.Caption(pc.Theme, "[Image: "+alt+"]")
				lbl.Color = color.NRGBA{R: 128, G: 128, B: 128, A: 255}
				stack := op.Offset(image.Pt(int(absX), int(viewY))).Push(gtx.Ops)
				lbl.Layout(gtx)
				stack.Pop()
			}
		}
	}

	// 4. Inline formatting context: paint wrapped word lines.
	if len(box.Lines) > 0 {
		paintLines(gtx, box, absX, pc, clicked)
	}

	// 5. Recurse into block children.
	maxBottom := absY + box.Dims.Height + box.Dims.Padding.Bottom + box.Dims.Margin.Bottom
	for _, child := range box.Children {
		bottom := paintBox(gtx, child, absX, absY, pc, clicked)
		if bottom > maxBottom {
			maxBottom = bottom
		}
	}

	return maxBottom
}

// paintBorder draws a simple 1px uniform border around the padding box.
func paintBorder(gtx layout.Context, absX, viewY float32, box *LayoutBox, col color.NRGBA) {
	w := box.Dims.BorderWidth()
	padX := absX - box.Dims.Padding.Left
	padY := viewY - box.Dims.Padding.Top
	totalW := box.Dims.Width + box.Dims.Padding.Left + box.Dims.Padding.Right
	totalH := box.Dims.Height + box.Dims.Padding.Top + box.Dims.Padding.Bottom

	bw := int(w)
	if bw < 1 {
		bw = 1
	}

	// Top + bottom
	stack := op.Offset(image.Pt(int(padX), int(padY))).Push(gtx.Ops)
	paint.FillShape(gtx.Ops, col, clip.Rect{Max: image.Pt(int(totalW), bw)}.Op())
	paint.FillShape(gtx.Ops, col, clip.Rect{
		Min: image.Pt(0, int(totalH) - bw),
		Max: image.Pt(int(totalW), int(totalH)),
	}.Op())
	// Left + right
	paint.FillShape(gtx.Ops, col, clip.Rect{Max: image.Pt(bw, int(totalH))}.Op())
	paint.FillShape(gtx.Ops, col, clip.Rect{
		Min: image.Pt(int(totalW) - bw, 0),
		Max: image.Pt(int(totalW), int(totalH)),
	}.Op())
	stack.Pop()
}

// paintLines renders each wrapped line of words at its computed Y offset.
func paintLines(gtx layout.Context, box *LayoutBox, absContentX float32, pc *PaintContext, clicked *string) {
	node := box.Node

	fontSize := unit.Sp(16)
	textColor := color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	if node != nil && node.Computed != nil {
		if node.Computed.FontSize > 0 {
			fontSize = node.Computed.FontSize
		}
		textColor = node.Computed.Color
	}

	charW := estimateCharWidth(fontSize)
	spaceW := charW * 0.8

	for _, line := range box.Lines {
		viewY := line.Y - pc.ScrollY
		xOff := float32(0)

		for wi, word := range line.Words {
			if wi > 0 {
				xOff += spaceW
			}

			wordX := absContentX + xOff
			wordColor := textColor

			if word.IsLink {
				wordColor = color.NRGBA{R: 26, G: 115, B: 232, A: 255}
				if linkNode := findAncestorTag(word.Node, "a"); linkNode != nil && linkNode.Computed != nil {
					wordColor = linkNode.Computed.Color
				}
			}

			stack := op.Offset(image.Pt(int(wordX), int(viewY))).Push(gtx.Ops)
			lbl := material.Label(pc.Theme, fontSize, word.Text)
			lbl.Color = wordColor
			lbl.Layout(gtx)
			stack.Pop()

			// Register hit area for clickable links.
			if word.IsLink {
				if linkNode := findAncestorTag(word.Node, "a"); linkNode != nil {
					h := estimateLineHeight(fontSize)
					stack := op.Offset(image.Pt(int(wordX), int(viewY))).Push(gtx.Ops)
					ck := pc.linkClickable(linkNode)
					rect := image.Rectangle{Max: image.Pt(int(word.Width)+4, int(h))}
					if ck.Clicked(gtx) {
						if href := linkNode.Attribute("href"); href != "" && !strings.HasPrefix(href, "javascript:") {
							*clicked = href
						}
					}
					_ = rect
					stack.Pop()
				}
			}

			xOff += word.Width
		}
	}
}

// linkClickable returns a stable Clickable for a link node, creating it once.
func (pc *PaintContext) linkClickable(n *Node) *widget.Clickable {
	if pc.Links == nil {
		return &widget.Clickable{}
	}
	if ck, ok := pc.Links[n]; ok && ck != nil {
		return ck
	}
	ck := &widget.Clickable{}
	pc.Links[n] = ck
	return ck
}

func findAncestorTag(n *Node, tag string) *Node {
	for p := n; p != nil; p = p.Parent {
		if p.Type == NodeElement && p.Tag == tag {
			return p
		}
	}
	return nil
}

// PaintImageBox paints a decoded image scaled to fit within maxW x maxH.
func PaintImageBox(gtx layout.Context, img image.Image, x, y, maxW, maxH float32) {
	if img == nil {
		return
	}
	bounds := img.Bounds()
	iw := float32(bounds.Dx())
	ih := float32(bounds.Dy())
	if iw <= 0 || ih <= 0 {
		return
	}
	if maxW <= 0 {
		maxW = iw
	}
	if maxH <= 0 {
		maxH = ih
	}

	scale := float32(1)
	if iw > maxW {
		scale = maxW / iw
	}
	if ih*scale > maxH {
		scale = maxH / ih
	}
	dw := iw * scale
	dh := ih * scale

	stack := op.Offset(image.Pt(int(x), int(y))).Push(gtx.Ops)
	defer stack.Pop()

	defer clip.Rect{Max: image.Pt(int(dw), int(dh))}.Push(gtx.Ops).Pop()
	paint.NewImageOp(img).Add(gtx.Ops)
	op.Affine(scaleAffine(scale)).Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
}

func scaleAffine(s float32) f32.Affine2D {
	return f32.Affine2D{}.Scale(f32.Point{}, f32.Pt(s, s))
}