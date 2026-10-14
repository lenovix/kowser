package main

import (
	"strings"
	"unicode"

	"gioui.org/unit"
)

// BoxType indicates whether a layout box is block-level, inline, or anonymous.
type BoxType int

const (
	BoxBlock BoxType = iota
	BoxInline
	BoxAnonymous
)

// Edges represents the 4-sided geometry (top, right, bottom, left).
type Edges struct {
	Top    float32
	Right  float32
	Bottom float32
	Left   float32
}

// Dimensions represents the total geometry of a box in pixels.
type Dimensions struct {
	X      float32
	Y      float32
	Width  float32
	Height float32

	Margin  Edges
	Padding Edges
	Border  Edges
}

// BorderWidth returns the top border width (or 0 if not set).
func (d Dimensions) BorderWidth() float32 {
	return d.Border.Top
}

// MarginBox returns the outer bounding rect including margins.
func (d Dimensions) MarginBox() (x, y, w, h float32) {
	x = d.X - d.Margin.Left
	y = d.Y - d.Margin.Top
	w = d.Width + d.Margin.Left + d.Margin.Right
	h = d.Height + d.Margin.Top + d.Margin.Bottom
	return
}

// PaddingBox returns the rect including content + padding.
func (d Dimensions) PaddingBox() (x, y, w, h float32) {
	x = d.X - d.Padding.Left
	y = d.Y - d.Padding.Top
	w = d.Width + d.Padding.Left + d.Padding.Right
	h = d.Height + d.Padding.Top + d.Padding.Bottom
	return
}

// InlineWord represents a single word measured for line-breaking.
type InlineWord struct {
	Text   string
	Width  float32
	Node   *Node
	IsLink bool
	Href   string
}

// LineBox represents a single horizontal line of wrapped inline text.
type LineBox struct {
	Y      float32
	Height float32
	Words  []InlineWord
}

// LayoutBox is a node in the layout geometry tree.
type LayoutBox struct {
	Type     BoxType
	Node     *Node
	Dims     Dimensions
	Children []*LayoutBox
	Lines    []LineBox // For inline formatting context
}

// BuildLayoutTree builds the box tree from the DOM tree, respecting display:none.
func BuildLayoutTree(node *Node) *LayoutBox {
	if node == nil {
		return nil
	}

	// Respect display: none
	if node.Computed != nil && node.Computed.Display == DisplayNone {
		return nil
	}

	box := &LayoutBox{
		Node: node,
	}

	if node.Type == NodeElement {
		if node.Computed != nil && node.Computed.Display == DisplayInline {
			box.Type = BoxInline
		} else {
			box.Type = BoxBlock
		}
	} else if node.Type == NodeText {
		// Whitespace-only text nodes directly under block elements do not generate layout boxes
		if strings.TrimSpace(node.Value) == "" {
			if node.Parent == nil || node.Parent.Computed == nil || node.Parent.Computed.Display == DisplayBlock {
				return nil
			}
		}
		box.Type = BoxInline
	} else {
		box.Type = BoxBlock
	}

	for c := node.FirstChild; c != nil; c = c.NextSibling {
		childBox := BuildLayoutTree(c)
		if childBox != nil {
			box.Children = append(box.Children, childBox)
		}
	}

	return box
}

// ComputeLayout calculates geometry (X, Y, Width, Height) for the whole box tree.
// containingWidth is the viewport width available for block layout.
func ComputeLayout(root *LayoutBox, containingWidth float32) {
	if root == nil {
		return
	}

	root.Dims.X = 0
	root.Dims.Y = 0
	root.Dims.Width = containingWidth

	layoutBlock(root, containingWidth, 0)
}

// Approximate text measurement: average character width ~0.55 * fontSize in px.
func estimateCharWidth(fontSize unit.Sp) float32 {
	fs := float32(fontSize)
	if fs <= 0 {
		fs = 16
	}
	return fs * 0.55
}

func estimateLineHeight(fontSize unit.Sp) float32 {
	fs := float32(fontSize)
	if fs <= 0 {
		fs = 16
	}
	return fs * 1.35
}

// layoutBlock lays out a block-level box and its children vertically.
// cursorY is the incoming vertical coordinate, returns new vertical coordinate.
func layoutBlock(box *LayoutBox, containingWidth float32, cursorY float32) float32 {
	node := box.Node

	// Resolve margins & paddings from ComputedStyle
	if node != nil && node.Computed != nil {
		style := node.Computed
		box.Dims.Margin.Top = style.MarginTop
		box.Dims.Margin.Bottom = style.MarginBottom
		box.Dims.Margin.Left = style.MarginLeft
		box.Dims.Margin.Right = style.MarginRight

		box.Dims.Padding.Top = style.PaddingTop
		box.Dims.Padding.Bottom = style.PaddingBottom
		box.Dims.Padding.Left = style.PaddingLeft
		box.Dims.Padding.Right = style.PaddingRight

		// Width calculation: if explicitly set in style, use it; else fill containing block
		if style.Width > 0 {
			box.Dims.Width = style.Width
		} else {
			box.Dims.Width = containingWidth - box.Dims.Margin.Left - box.Dims.Margin.Right - box.Dims.Padding.Left - box.Dims.Padding.Right
			if box.Dims.Width < 0 {
				box.Dims.Width = 0
			}
		}
	} else {
		box.Dims.Width = containingWidth
	}

	// Apply margin top to cursor
	box.Dims.X = box.Dims.Margin.Left + box.Dims.Padding.Left
	cursorY += box.Dims.Margin.Top
	box.Dims.Y = cursorY
	contentStartY := cursorY + box.Dims.Padding.Top

	// Check if this box directly contains inline children (e.g. text/links inside <p> or <h1>)
	hasInlineChildren := false
	hasBlockChildren := false
	for _, c := range box.Children {
		if c.Type == BoxInline {
			hasInlineChildren = true
		} else if c.Type == BoxBlock {
			hasBlockChildren = true
		}
	}

	currentY := contentStartY

	if hasInlineChildren && !hasBlockChildren {
		// Pure inline formatting context: perform word-wrapping
		currentY = layoutInlineContext(box, contentStartY)
	} else {
		// Block formatting context: layout children vertically with margin collapsing
		var prevBottomMargin float32 = 0
		for _, child := range box.Children {
			// Margin collapsing between siblings: max(prevMarginBottom, nextMarginTop)
			nextTopMargin := float32(0)
			if child.Node != nil && child.Node.Computed != nil {
				nextTopMargin = child.Node.Computed.MarginTop
			}

			collapsedMargin := max(prevBottomMargin, nextTopMargin)
			// currentY includes prevBottomMargin from previous child's return value.
			// layoutBlock will add nextTopMargin again, so subtract it here
			// to avoid double counting: input = current - prevBottom + collapsed - nextTop
			currentY = currentY - prevBottomMargin + collapsedMargin - nextTopMargin

			currentY = layoutBlock(child, box.Dims.Width, currentY)

			if child.Node != nil && child.Node.Computed != nil {
				prevBottomMargin = child.Node.Computed.MarginBottom
			} else {
				prevBottomMargin = 0
			}
		}
	}

	// Compute box height
	contentHeight := currentY - contentStartY
	if node != nil && node.Computed != nil && node.Computed.Height > 0 {
		box.Dims.Height = node.Computed.Height
	} else {
		box.Dims.Height = contentHeight
	}

	totalY := box.Dims.Y + box.Dims.Height + box.Dims.Padding.Bottom + box.Dims.Margin.Bottom
	return totalY
}

// layoutInlineContext collects all text/words inside inline children and wraps them into lines.
func layoutInlineContext(box *LayoutBox, startY float32) float32 {
	fontSize := unit.Sp(16)
	if box.Node != nil && box.Node.Computed != nil && box.Node.Computed.FontSize > 0 {
		fontSize = box.Node.Computed.FontSize
	}

	charW := estimateCharWidth(fontSize)
	lineH := estimateLineHeight(fontSize)
	availWidth := box.Dims.Width

	var words []InlineWord
	collectInlineWords(box, &words, fontSize)

	if len(words) == 0 {
		return startY
	}

	// Word wrapping algorithm (greedy break at word boundary)
	var lines []LineBox
	curLine := LineBox{
		Y:      startY,
		Height: lineH,
	}
	var curLineWidth float32 = 0
	spaceWidth := charW * 0.8

	for _, w := range words {
		neededWidth := w.Width
		if len(curLine.Words) > 0 {
			neededWidth += spaceWidth
		}

		if curLineWidth+neededWidth > availWidth && len(curLine.Words) > 0 {
			// Line break: finalize current line
			lines = append(lines, curLine)
			curLine = LineBox{
				Y:      curLine.Y + curLine.Height,
				Height: lineH,
				Words:  []InlineWord{w},
			}
			curLineWidth = w.Width
		} else {
			curLine.Words = append(curLine.Words, w)
			curLineWidth += neededWidth
		}
	}

	if len(curLine.Words) > 0 {
		lines = append(lines, curLine)
	}

	box.Lines = lines
	totalHeight := float32(len(lines)) * lineH
	return startY + totalHeight
}

func collectInlineWords(box *LayoutBox, words *[]InlineWord, inheritedFontSize unit.Sp) {
	node := box.Node
	if node == nil {
		return
	}

	fs := inheritedFontSize
	if node.Computed != nil && node.Computed.FontSize > 0 {
		fs = node.Computed.FontSize
	}
	charW := estimateCharWidth(fs)

	if node.Type == NodeText {
		raw := strings.TrimSpace(node.Value)
		if raw != "" {
			parts := strings.FieldsFunc(raw, unicode.IsSpace)
			isLink := false
			href := ""
			for p := node.Parent; p != nil; p = p.Parent {
				if p.Tag == "a" {
					isLink = true
					href = p.Attribute("href")
					break
				}
			}

			for _, p := range parts {
				w := InlineWord{
					Text:   p,
					Width:  float32(len(p)) * charW,
					Node:   node,
					IsLink: isLink,
					Href:   href,
				}
				*words = append(*words, w)
			}
		}
		return
	}

	for _, child := range box.Children {
		collectInlineWords(child, words, fs)
	}
}

func max(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
