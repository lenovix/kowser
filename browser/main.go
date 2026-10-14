package main

import (
	"context"
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

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
	TypeForm
	TypeError
)

type FormElement struct {
	Action string
	Method string
	Query  widget.Editor
	Submit widget.Clickable
}

type PageElement struct {
	Type     ElementType
	Level    int // For headings 1-6
	Text     string
	Href     string
	IsBold   bool
	IsBullet bool
	Click    widget.Clickable
	Form     *FormElement
}

type HistoryState struct {
	History []string `json:"history"`
	Index   int      `json:"index"`
}

type BrowserApp struct {
	theme       *material.Theme
	urlEditor   widget.Editor
	goBtn       widget.Clickable
	backBtn     widget.Clickable
	fwdBtn      widget.Clickable
	reloadBtn   widget.Clickable
	homeBtn     widget.Clickable
	contentList widget.List

	client *http.Client

	mu           sync.Mutex
	statusText   string
	pageTitle    string
	currentURL   string
	history      []string
	historyIdx   int
	pageElements []*PageElement
	loading      bool
	cancelFetch  context.CancelFunc
}

const defaultHomeURL = "https://www.google.com"

func historyFilePath() string {
	return filepath.Join(".", "history.json")
}

func (b *BrowserApp) loadHistory() {
	data, err := os.ReadFile(historyFilePath())
	if err != nil {
		b.historyIdx = -1
		return
	}
	var state HistoryState
	if err := json.Unmarshal(data, &state); err == nil && len(state.History) > 0 {
		b.history = state.History
		b.historyIdx = state.Index
		if b.historyIdx < 0 || b.historyIdx >= len(b.history) {
			b.historyIdx = len(b.history) - 1
		}
	} else {
		b.historyIdx = -1
	}
}

func (b *BrowserApp) saveHistory() {
	state := HistoryState{
		History: b.history,
		Index:   b.historyIdx,
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err == nil {
		_ = os.WriteFile(historyFilePath(), data, 0644)
	}
}

func main() {
	jar, _ := cookiejar.New(nil)
	b := &BrowserApp{
		theme: material.NewTheme(),
		client: &http.Client{
			Jar:     jar,
			Timeout: 15 * time.Second,
		},
		contentList: widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
		statusText: "Ready. Enter URL or search.",
		pageTitle:  "Go Browser",
	}

	b.loadHistory()

	initialURL := defaultHomeURL
	if b.historyIdx >= 0 && b.historyIdx < len(b.history) {
		initialURL = b.history[b.historyIdx]
	}

	b.urlEditor.SingleLine = true
	b.urlEditor.Submit = true
	b.urlEditor.SetText(initialURL)

	go func() {
		w := new(app.Window)
		w.Option(app.Title(b.pageTitle), app.Size(unit.Dp(900), unit.Dp(650)))
		var ops op.Ops
		for {
			switch e := w.Event().(type) {
			case app.DestroyEvent:
				b.mu.Lock()
				b.saveHistory()
				b.mu.Unlock()
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
	curr := b.currentURL
	b.mu.Unlock()

	if canBack && b.backBtn.Clicked(gtx) {
		b.navigateHistory(-1, w)
		return
	}
	if canFwd && b.fwdBtn.Clicked(gtx) {
		b.navigateHistory(1, w)
		return
	}
	if b.reloadBtn.Clicked(gtx) {
		target := curr
		if target == "" {
			target = strings.TrimSpace(b.urlEditor.Text())
		}
		if target != "" {
			target = normalizeURL(target)
			go b.fetchURL(target, false, w)
		}
		return
	}
	if b.homeBtn.Clicked(gtx) {
		b.urlEditor.SetText(defaultHomeURL)
		go b.fetchURL(defaultHomeURL, true, w)
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
	b.mu.Unlock()

	for _, elem := range elems {
		if elem.Type == TypeLink && elem.Click.Clicked(gtx) {
			target := resolveURL(curr, elem.Href)
			b.urlEditor.SetText(target)
			go b.fetchURL(target, true, w)
			break
		}
		if elem.Type == TypeForm && elem.Form != nil {
			formSubmit := false
			if fev, fok := elem.Form.Query.Update(gtx); fok {
				if _, isSub := fev.(widget.SubmitEvent); isSub {
					formSubmit = true
				}
			}
			if elem.Form.Submit.Clicked(gtx) || formSubmit {
				qVal := strings.TrimSpace(elem.Form.Query.Text())
				actionURL := elem.Form.Action
				if actionURL == "" {
					actionURL = curr
				}
				actionURL = resolveURL(curr, actionURL)

				u, err := url.Parse(actionURL)
				if err == nil {
					q := u.Query()
					q.Set("q", qVal)
					u.RawQuery = q.Encode()
					target := u.String()
					b.urlEditor.SetText(target)
					go b.fetchURL(target, true, w)
					break
				}
			}
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
	b.saveHistory()
	b.mu.Unlock()

	b.urlEditor.SetText(target)
	go b.fetchURL(target, false, w)
}

func (b *BrowserApp) fetchURL(rawURL string, recordHistory bool, w *app.Window) {
	b.mu.Lock()
	if b.cancelFetch != nil {
		b.cancelFetch()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	b.cancelFetch = cancel
	b.loading = true
	b.statusText = "Loading: " + rawURL
	b.pageElements = nil
	b.mu.Unlock()
	w.Invalidate()

	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil {
		b.setErrorState("Invalid URL: "+err.Error(), rawURL, w)
		return
	}

	req, err := http.NewRequestWithContext(ctx, "GET", parsedURL.String(), nil)
	if err != nil {
		b.setErrorState("Request error: "+err.Error(), rawURL, w)
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Kowser/1.0")

	resp, err := b.client.Do(req)
	if err != nil {
		if ctx.Err() == context.Canceled {
			return // Superseded by new request
		}
		b.setErrorState("Fetch failed: "+err.Error(), rawURL, w)
		return
	}
	defer resp.Body.Close()

	title, elems := parseHTML(resp.Body)

	b.mu.Lock()
	finalURL := resp.Request.URL.String()
	b.currentURL = finalURL
	b.loading = false
	if title == "" {
		title = finalURL
	}
	b.pageTitle = title
	b.statusText = fmt.Sprintf("HTTP %d %s | %d elements", resp.StatusCode, resp.Status, len(elems))
	b.pageElements = elems

	if recordHistory {
		if b.historyIdx >= 0 && b.historyIdx < len(b.history) {
			b.history = append(b.history[:b.historyIdx+1], finalURL)
		} else {
			b.history = append(b.history, finalURL)
		}
		b.historyIdx = len(b.history) - 1
		b.saveHistory()
	}
	b.mu.Unlock()

	w.Option(app.Title(title))
	b.urlEditor.SetText(finalURL)
	w.Invalidate()
}

func (b *BrowserApp) setErrorState(msg string, targetURL string, w *app.Window) {
	b.mu.Lock()
	b.loading = false
	b.statusText = msg
	b.pageElements = []*PageElement{
		{
			Type:  TypeError,
			Text:  "Failed to load page: " + targetURL,
			Level: 1,
		},
		{
			Type: TypeError,
			Text: msg,
		},
	}
	b.mu.Unlock()
	w.Invalidate()
}

func parseHTML(r io.Reader) (string, []*PageElement) {
	doc, err := html.Parse(r)
	if err != nil {
		return "", nil
	}

	var title string
	var elems []*PageElement
	const maxElements = 5000

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if len(elems) >= maxElements {
			return
		}

		if n.Type == html.ElementNode {
			switch n.Data {
			case "title":
				title = strings.TrimSpace(nodeText(n))
				return
			case "script", "style", "noscript", "svg":
				return
			case "form":
				action := ""
				method := "GET"
				for _, a := range n.Attr {
					if a.Key == "action" {
						action = a.Val
					} else if a.Key == "method" {
						method = strings.ToUpper(a.Val)
					}
				}
				form := &FormElement{
					Action: action,
					Method: method,
				}
				form.Query.SingleLine = true
				form.Query.Submit = true
				elems = append(elems, &PageElement{
					Type: TypeForm,
					Form: form,
				})
				// Continue inside form so inner text or links also render
			case "h1", "h2", "h3", "h4", "h5", "h6":
				level := int(n.Data[1] - '0')
				txt := strings.TrimSpace(nodeText(n))
				if txt != "" {
					elems = append(elems, &PageElement{
						Type:  TypeHeading,
						Level: level,
						Text:  txt,
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
			case "li":
				txt := strings.TrimSpace(nodeText(n))
				if txt != "" {
					elems = append(elems, &PageElement{
						Type:     TypeText,
						Text:     txt,
						IsBullet: true,
					})
				}
				return
			case "strong", "b":
				txt := strings.TrimSpace(nodeText(n))
				if txt != "" {
					elems = append(elems, &PageElement{
						Type:   TypeText,
						Text:   txt,
						IsBold: true,
					})
				}
				return
			case "p", "div", "span":
				hasBlockChild := false
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					if c.Type == html.ElementNode && (c.Data == "p" || c.Data == "div" || c.Data == "h1" || c.Data == "h2" || c.Data == "h3" || c.Data == "a" || c.Data == "ul" || c.Data == "ol" || c.Data == "form" || c.Data == "li" || c.Data == "strong" || c.Data == "b") {
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
	return title, elems
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
			layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				btn := material.Button(b.theme, &b.reloadBtn, "R")
				return btn.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				btn := material.Button(b.theme, &b.homeBtn, "H")
				return btn.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				ed := material.Editor(b.theme, &b.urlEditor, "Enter URL or search...")
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
	red := color.NRGBA{R: 220, G: 50, B: 50, A: 255}

	return material.List(b.theme, &b.contentList).Layout(gtx, len(elems), func(gtx layout.Context, i int) layout.Dimensions {
		elem := elems[i]
		inset := layout.Inset{Top: unit.Dp(3), Bottom: unit.Dp(3), Left: unit.Dp(12), Right: unit.Dp(12)}

		return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			switch elem.Type {
			case TypeForm:
				if elem.Form == nil {
					return layout.Dimensions{}
				}
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						ed := material.Editor(b.theme, &elem.Form.Query, "Search Google...")
						return ed.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						btn := material.Button(b.theme, &elem.Form.Submit, "Search")
						return btn.Layout(gtx)
					}),
				)
			case TypeError:
				lbl := material.Body1(b.theme, elem.Text)
				lbl.Color = red
				return lbl.Layout(gtx)
			case TypeHeading:
				switch elem.Level {
				case 1:
					return material.H4(b.theme, elem.Text).Layout(gtx)
				case 2:
					return material.H5(b.theme, elem.Text).Layout(gtx)
				default:
					return material.H6(b.theme, elem.Text).Layout(gtx)
				}
			case TypeLink:
				btn := material.Clickable(gtx, &elem.Click, func(gtx layout.Context) layout.Dimensions {
					lbl := material.Body1(b.theme, elem.Text)
					lbl.Color = blue
					return lbl.Layout(gtx)
				})
				return btn
			default:
				txt := elem.Text
				if elem.IsBullet {
					txt = "• " + txt
				}
				lbl := material.Body1(b.theme, txt)
				return lbl.Layout(gtx)
			}
		})
	})
}
