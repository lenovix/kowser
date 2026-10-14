package main

import (
	"fmt"
	"image/color"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"golang.org/x/net/html"
)

type ElementType int

const (
	TypeText ElementType = iota
	TypeHeading
	TypeLink
)

type PageElement struct {
	Type  ElementType
	Text  string
	Href  string
	Click widget.Clickable
}

type BrowserApp struct {
	theme       *material.Theme
	urlEditor   widget.Editor
	goBtn       widget.Clickable
	backBtn     widget.Clickable
	fwdBtn      widget.Clickable
	contentList widget.List

	client *http.Client

	mu           sync.Mutex
	statusText   string
	currentURL   string
	history      []string
	historyIdx   int
	pageElements []*PageElement
	loading      bool
}

func main() {
	jar, _ := cookiejar.New(nil)
	b := &BrowserApp{
		theme: material.NewTheme(),
		client: &http.Client{
			Jar: jar,
		},
		contentList: widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
		statusText: "Ready. Enter URL and click Go.",
		historyIdx: -1,
	}
	b.urlEditor.SingleLine = true
	b.urlEditor.Submit = true
	b.urlEditor.SetText("https://www.google.com")

	go func() {
		w := new(app.Window)
		w.Option(app.Title("Go Browser"), app.Size(unit.Dp(900), unit.Dp(650)))
		var ops op.Ops
		for {
			switch e := w.Event().(type) {
			case app.DestroyEvent:
				return
			case app.FrameEvent:
				gtx := app.NewContext(&ops, e)
				b.handleEvents(gtx, w)
				b.layout(gtx)
				e.Frame(gtx.Ops)
			}
		}
	}()
	app.Main()
}

func (b *BrowserApp) handleEvents(gtx layout.Context, w *app.Window) {
	b.mu.Lock()
	canBack := b.historyIdx > 0
	canFwd := b.historyIdx < len(b.history)-1
	b.mu.Unlock()

	if canBack && b.backBtn.Clicked(gtx) {
		b.navigateHistory(-1, w)
		return
	}
	if canFwd && b.fwdBtn.Clicked(gtx) {
		b.navigateHistory(1, w)
		return
	}

	submit := false
	if ev, ok := b.urlEditor.Update(gtx); ok {
		if _, isSubmit := ev.(widget.SubmitEvent); isSubmit {
			submit = true
		}
	}

	if b.goBtn.Clicked(gtx) || submit {
		target := strings.TrimSpace(b.urlEditor.Text())
		if target != "" {
			target = normalizeURL(target)
			b.urlEditor.SetText(target)
			go b.fetchURL(target, true, w)
		}
		return
	}

	b.mu.Lock()
	elems := b.pageElements
	curr := b.currentURL
	b.mu.Unlock()

	for _, elem := range elems {
		if elem.Type == TypeLink && elem.Click.Clicked(gtx) {
			target := resolveURL(curr, elem.Href)
			b.urlEditor.SetText(target)
			go b.fetchURL(target, true, w)
			break
		}
	}
}

func normalizeURL(target string) string {
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		return "https://" + target
	}
	return target
}

func resolveURL(baseRaw, refRaw string) string {
	base, err := url.Parse(baseRaw)
	if err != nil {
		return normalizeURL(refRaw)
	}
	ref, err := url.Parse(refRaw)
	if err != nil {
		return refRaw
	}
	return base.ResolveReference(ref).String()
}

func (b *BrowserApp) navigateHistory(delta int, w *app.Window) {
	b.mu.Lock()
	newIdx := b.historyIdx + delta
	if newIdx < 0 || newIdx >= len(b.history) {
		b.mu.Unlock()
		return
	}
	b.historyIdx = newIdx
	target := b.history[b.historyIdx]
	b.mu.Unlock()

	b.urlEditor.SetText(target)
	go b.fetchURL(target, false, w)
}

func (b *BrowserApp) fetchURL(rawURL string, recordHistory bool, w *app.Window) {
	b.mu.Lock()
	b.loading = true
	b.statusText = "Loading: " + rawURL
	b.pageElements = nil
	b.mu.Unlock()
	w.Invalidate()

	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil {
		b.mu.Lock()
		b.loading = false
		b.statusText = "Invalid URL: " + err.Error()
		b.mu.Unlock()
		w.Invalidate()
		return
	}

	req, err := http.NewRequest("GET", parsedURL.String(), nil)
	if err != nil {
		b.mu.Lock()
		b.loading = false
		b.statusText = "Request error: " + err.Error()
		b.mu.Unlock()
		w.Invalidate()
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) GoBrowser/1.0")

	resp, err := b.client.Do(req)
	if err != nil {
		b.mu.Lock()
		b.loading = false
		b.statusText = "Fetch failed: " + err.Error()
		b.mu.Unlock()
		w.Invalidate()
		return
	}
	defer resp.Body.Close()

	elems := parseHTML(resp.Body)

	b.mu.Lock()
	finalURL := resp.Request.URL.String()
	b.currentURL = finalURL
	b.loading = false
	b.statusText = fmt.Sprintf("HTTP %d | %d elements", resp.StatusCode, len(elems))
	b.pageElements = elems

	if recordHistory {
		if b.historyIdx >= 0 && b.historyIdx < len(b.history) {
			b.history = append(b.history[:b.historyIdx+1], finalURL)
		} else {
			b.history = append(b.history, finalURL)
		}
		b.historyIdx = len(b.history) - 1
	}
	b.mu.Unlock()

	b.urlEditor.SetText(finalURL)
	w.Invalidate()
}

func parseHTML(r io.Reader) []*PageElement {
	doc, err := html.Parse(r)
	if err != nil {
		return nil
	}

	var elems []*PageElement
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "script", "style", "noscript", "svg", "head":
				return
			case "h1", "h2", "h3", "h4", "h5", "h6":
				txt := strings.TrimSpace(nodeText(n))
				if txt != "" {
					elems = append(elems, &PageElement{
						Type: TypeHeading,
						Text: txt,
					})
				}
				return
			case "a":
				href := ""
				for _, attr := range n.Attr {
					if attr.Key == "href" {
						href = attr.Val
						break
					}
				}
				txt := strings.TrimSpace(nodeText(n))
				if txt != "" && href != "" && !strings.HasPrefix(href, "javascript:") {
					elems = append(elems, &PageElement{
						Type: TypeLink,
						Text: txt,
						Href: href,
					})
				}
				return
			case "p", "div", "li", "span":
				// If contains pure text with no child block tags, extract
				hasBlockChild := false
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					if c.Type == html.ElementNode && (c.Data == "p" || c.Data == "div" || c.Data == "h1" || c.Data == "h2" || c.Data == "h3" || c.Data == "a" || c.Data == "ul" || c.Data == "ol") {
						hasBlockChild = true
						break
					}
				}
				if !hasBlockChild {
					txt := strings.TrimSpace(nodeText(n))
					if txt != "" {
						elems = append(elems, &PageElement{
							Type: TypeText,
							Text: txt,
						})
						return
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}

	walk(doc)
	return elems
}

func nodeText(n *html.Node) string {
	if n == nil {
		return ""
	}
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && (c.Data == "script" || c.Data == "style") {
			continue
		}
		sb.WriteString(nodeText(c))
		sb.WriteString(" ")
	}
	return sb.String()
}

func (b *BrowserApp) layout(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return b.layoutTopBar(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return b.layoutStatusBar(gtx)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return b.layoutContent(gtx)
		}),
	)
}

func (b *BrowserApp) layoutTopBar(gtx layout.Context) layout.Dimensions {
	inset := layout.UniformInset(unit.Dp(8))
	return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				btn := material.Button(b.theme, &b.backBtn, "<")
				return btn.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				btn := material.Button(b.theme, &b.fwdBtn, ">")
				return btn.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				ed := material.Editor(b.theme, &b.urlEditor, "Enter URL...")
				return ed.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				btnText := "Go"
				if b.loading {
					btnText = "..."
				}
				btn := material.Button(b.theme, &b.goBtn, btnText)
				return btn.Layout(gtx)
			}),
		)
	})
}

func (b *BrowserApp) layoutStatusBar(gtx layout.Context) layout.Dimensions {
	inset := layout.Inset{Left: unit.Dp(8), Right: unit.Dp(8), Bottom: unit.Dp(4)}
	return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		b.mu.Lock()
		txt := b.statusText
		b.mu.Unlock()
		lbl := material.Caption(b.theme, txt)
		return lbl.Layout(gtx)
	})
}

func (b *BrowserApp) layoutContent(gtx layout.Context) layout.Dimensions {
	b.mu.Lock()
	elems := b.pageElements
	b.mu.Unlock()

	if len(elems) == 0 {
		inset := layout.UniformInset(unit.Dp(16))
		return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return material.Body2(b.theme, "No content loaded.").Layout(gtx)
		})
	}

	blue := color.NRGBA{R: 26, G: 115, B: 232, A: 255}

	return material.List(b.theme, &b.contentList).Layout(gtx, len(elems), func(gtx layout.Context, i int) layout.Dimensions {
		elem := elems[i]
		inset := layout.Inset{Top: unit.Dp(3), Bottom: unit.Dp(3), Left: unit.Dp(12), Right: unit.Dp(12)}

		return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			switch elem.Type {
			case TypeHeading:
				h := material.H6(b.theme, elem.Text)
				return h.Layout(gtx)
			case TypeLink:
				btn := material.Clickable(gtx, &elem.Click, func(gtx layout.Context) layout.Dimensions {
					lbl := material.Body1(b.theme, elem.Text)
					lbl.Color = blue
					return lbl.Layout(gtx)
				})
				return btn
			default:
				return material.Body1(b.theme, elem.Text).Layout(gtx)
			}
		})
	})
}
