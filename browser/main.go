package main

import (
	"fmt"
	"io"
	"net/http"
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

type BrowserApp struct {
	theme       *material.Theme
	urlEditor   widget.Editor
	goBtn       widget.Clickable
	contentList widget.List

	mu          sync.Mutex
	statusText  string
	pageElements []string
	loading     bool
}

func main() {
	b := &BrowserApp{
		theme: material.NewTheme(),
		contentList: widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
		statusText: "Ready. Enter URL and click Go.",
	}
	b.urlEditor.SingleLine = true
	b.urlEditor.Submit = true
	b.urlEditor.SetText("https://www.google.com")

	go func() {
		w := new(app.Window)
		w.Option(app.Title("Go Browser MVP"), app.Size(unit.Dp(800), unit.Dp(600)))
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
	submit := false
	if ev, ok := b.urlEditor.Update(gtx); ok {
		if _, isSubmit := ev.(widget.SubmitEvent); isSubmit {
			submit = true
		}
	}
	if b.goBtn.Clicked(gtx) || submit {
		target := strings.TrimSpace(b.urlEditor.Text())
		if target != "" {
			if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
				target = "https://" + target
				b.urlEditor.SetText(target)
			}
			go b.fetchURL(target, w)
		}
	}
}

func (b *BrowserApp) fetchURL(rawURL string, w *app.Window) {
	b.mu.Lock()
	b.loading = true
	b.statusText = "Loading: " + rawURL
	b.pageElements = nil
	b.mu.Unlock()
	w.Invalidate()

	_, err := url.ParseRequestURI(rawURL)
	if err != nil {
		b.mu.Lock()
		b.loading = false
		b.statusText = "Invalid URL: " + err.Error()
		b.mu.Unlock()
		w.Invalidate()
		return
	}

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		b.mu.Lock()
		b.loading = false
		b.statusText = "Request error: " + err.Error()
		b.mu.Unlock()
		w.Invalidate()
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) GoBrowser/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		b.mu.Lock()
		b.loading = false
		b.statusText = "Fetch failed: " + err.Error()
		b.mu.Unlock()
		w.Invalidate()
		return
	}
	defer resp.Body.Close()

	elems := extractTextAndLinks(resp.Body)

	b.mu.Lock()
	b.loading = false
	b.statusText = fmt.Sprintf("HTTP %d %s | Elements: %d", resp.StatusCode, resp.Status, len(elems))
	b.pageElements = elems
	b.mu.Unlock()
	w.Invalidate()
}

func extractTextAndLinks(r io.Reader) []string {
	var lines []string
	z := html.NewTokenizer(r)
	skip := false

	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			return lines
		case html.StartTagToken:
			tn, _ := z.TagName()
			tag := string(tn)
			if tag == "script" || tag == "style" || tag == "noscript" {
				skip = true
			}
		case html.EndTagToken:
			tn, _ := z.TagName()
			tag := string(tn)
			if tag == "script" || tag == "style" || tag == "noscript" {
				skip = false
			}
		case html.TextToken:
			if !skip {
				txt := strings.TrimSpace(string(z.Text()))
				if txt != "" {
					lines = append(lines, txt)
				}
			}
		}
	}
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
	elems := make([]string, len(b.pageElements))
	copy(elems, b.pageElements)
	b.mu.Unlock()

	if len(elems) == 0 {
		inset := layout.UniformInset(unit.Dp(16))
		return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return material.Body2(b.theme, "No content loaded.").Layout(gtx)
		})
	}

	return material.List(b.theme, &b.contentList).Layout(gtx, len(elems), func(gtx layout.Context, i int) layout.Dimensions {
		inset := layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(8), Right: unit.Dp(8)}
		return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return material.Body1(b.theme, elems[i]).Layout(gtx)
		})
	})
}
