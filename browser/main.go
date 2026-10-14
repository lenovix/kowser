package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
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
	"gioui.org/op/paint"
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
	TypeImage
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
	Src      string
	IsBold   bool
	IsBullet bool
	Click    widget.Clickable
	Form     *FormElement
}

type BookmarkItem struct {
	Title string           `json:"title"`
	URL   string           `json:"url"`
	Click widget.Clickable `json:"-"`
}

type HistoryState struct {
	History []string `json:"history"`
	Index   int      `json:"index"`
}

type Tab struct {
	ID           int
	Title        string
	CurrentURL   string
	History      []string
	HistoryIdx   int
	PageElements []*PageElement
	StatusText   string
	Loading      bool
	CancelFetch  context.CancelFunc
	ContentList  widget.List

	TabBtn   widget.Clickable
	CloseBtn widget.Clickable
}

type BrowserApp struct {
	theme       *material.Theme
	urlEditor   widget.Editor
	goBtn       widget.Clickable
	backBtn     widget.Clickable
	fwdBtn      widget.Clickable
	reloadBtn   widget.Clickable
	homeBtn     widget.Clickable
	bookmarkBtn widget.Clickable
	newTabBtn   widget.Clickable

	tabBar      widget.List
	bookmarkBar widget.List

	client *http.Client

	mu         sync.Mutex
	tabs       []*Tab
	activeTab  int
	nextTabID  int
	imgCacheMu sync.Mutex
	imgCache   map[string]image.Image
	window     *app.Window
	bookmarks  []*BookmarkItem
}

const defaultHomeURL = "https://www.google.com"

func historyFilePath() string {
	return filepath.Join(".", "history.json")
}

func (b *BrowserApp) loadHistory() {
	data, err := os.ReadFile(historyFilePath())
	if err != nil {
		return
	}
	var state HistoryState
	if err := json.Unmarshal(data, &state); err == nil && len(state.History) > 0 {
		idx := state.Index
		if idx < 0 || idx >= len(state.History) {
			idx = len(state.History) - 1
		}
		if len(b.tabs) > 0 {
			b.tabs[0].History = state.History
			b.tabs[0].HistoryIdx = idx
			b.tabs[0].CurrentURL = state.History[idx]
			b.tabs[0].Title = state.History[idx]
		}
	}
}

func (b *BrowserApp) saveHistory() {
	tab := b.getActiveTab()
	if tab == nil {
		return
	}
	state := HistoryState{
		History: tab.History,
		Index:   tab.HistoryIdx,
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err == nil {
		_ = os.WriteFile(historyFilePath(), data, 0644)
	}
}

func bookmarksFilePath() string {
	return filepath.Join(".", "bookmarks.json")
}

func (b *BrowserApp) loadBookmarks() {
	data, err := os.ReadFile(bookmarksFilePath())
	if err != nil {
		b.bookmarks = []*BookmarkItem{
			{Title: "Google", URL: "https://www.google.com"},
		}
		return
	}
	var bms []*BookmarkItem
	if err := json.Unmarshal(data, &bms); err == nil && len(bms) > 0 {
		b.bookmarks = bms
	} else {
		b.bookmarks = []*BookmarkItem{
			{Title: "Google", URL: "https://www.google.com"},
		}
	}
}

func (b *BrowserApp) saveBookmarks() {
	data, err := json.MarshalIndent(b.bookmarks, "", "  ")
	if err == nil {
		_ = os.WriteFile(bookmarksFilePath(), data, 0644)
	}
}

func (b *BrowserApp) isCurrentBookmarked() bool {
	tab := b.getActiveTab()
	if tab == nil || tab.CurrentURL == "" {
		return false
	}
	for _, bm := range b.bookmarks {
		if bm.URL == tab.CurrentURL {
			return true
		}
	}
	return false
}

func (b *BrowserApp) toggleBookmark() {
	tab := b.getActiveTab()
	if tab == nil || tab.CurrentURL == "" {
		return
	}
	for i, bm := range b.bookmarks {
		if bm.URL == tab.CurrentURL {
			b.bookmarks = append(b.bookmarks[:i], b.bookmarks[i+1:]...)
			b.saveBookmarks()
			return
		}
	}
	title := tab.Title
	if title == "" {
		title = tab.CurrentURL
	}
	b.bookmarks = append(b.bookmarks, &BookmarkItem{
		Title: title,
		URL:   tab.CurrentURL,
	})
	b.saveBookmarks()
}

func (b *BrowserApp) getImage(rawURL string) image.Image {
	b.imgCacheMu.Lock()
	defer b.imgCacheMu.Unlock()
	return b.imgCache[rawURL]
}

func (b *BrowserApp) fetchImageAsync(rawURL string) {
	b.imgCacheMu.Lock()
	if _, exists := b.imgCache[rawURL]; exists {
		b.imgCacheMu.Unlock()
		return
	}
	b.imgCacheMu.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
		if err != nil {
			return
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Kowser/1.0")

		resp, err := b.client.Do(req)
		if err != nil {
			return
		}
		defer resp.Body.Close()

		data, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
		if err != nil {
			return
		}

		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return
		}

		b.imgCacheMu.Lock()
		b.imgCache[rawURL] = img
		b.imgCacheMu.Unlock()

		b.mu.Lock()
		w := b.window
		b.mu.Unlock()
		if w != nil {
			w.Invalidate()
		}
	}()
}

func (b *BrowserApp) newTab(initialURL string) *Tab {
	b.nextTabID++
	tab := &Tab{
		ID:         b.nextTabID,
		Title:      "New Tab",
		CurrentURL: initialURL,
		HistoryIdx: -1,
		StatusText: "Ready.",
		ContentList: widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
	}
	return tab
}

func (b *BrowserApp) getActiveTab() *Tab {
	if len(b.tabs) == 0 {
		return nil
	}
	if b.activeTab < 0 || b.activeTab >= len(b.tabs) {
		b.activeTab = 0
	}
	return b.tabs[b.activeTab]
}

func (b *BrowserApp) closeTab(idx int) {
	if len(b.tabs) <= 1 {
		// Jangan tutup jika sisa 1 tab, reset ke default
		tab := b.tabs[0]
		if tab.CancelFetch != nil {
			tab.CancelFetch()
		}
		tab.Title = "New Tab"
		tab.CurrentURL = defaultHomeURL
		tab.History = nil
		tab.HistoryIdx = -1
		tab.PageElements = nil
		tab.StatusText = "Ready."
		b.urlEditor.SetText(defaultHomeURL)
		return
	}

	target := b.tabs[idx]
	if target.CancelFetch != nil {
		target.CancelFetch()
	}

	b.tabs = append(b.tabs[:idx], b.tabs[idx+1:]...)
	if b.activeTab >= len(b.tabs) {
		b.activeTab = len(b.tabs) - 1
	}

	curr := b.getActiveTab()
	if curr != nil {
		b.urlEditor.SetText(curr.CurrentURL)
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
		tabBar: widget.List{
			List: layout.List{
				Axis: layout.Horizontal,
			},
		},
		bookmarkBar: widget.List{
			List: layout.List{
				Axis: layout.Horizontal,
			},
		},
		imgCache: make(map[string]image.Image),
	}

	initialTab := b.newTab(defaultHomeURL)
	b.tabs = append(b.tabs, initialTab)
	b.activeTab = 0

	b.loadHistory()
	b.loadBookmarks()

	initialURL := defaultHomeURL
	if initialTab.CurrentURL != "" {
		initialURL = initialTab.CurrentURL
	}

	b.urlEditor.SingleLine = true
	b.urlEditor.Submit = true
	b.urlEditor.SetText(initialURL)

	go func() {
		w := new(app.Window)
		b.mu.Lock()
		b.window = w
		b.mu.Unlock()
		w.Option(app.Title("Go Browser"), app.Size(unit.Dp(960), unit.Dp(680)))
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
	defer b.mu.Unlock()

	// Handle new tab button
	if b.newTabBtn.Clicked(gtx) {
		t := b.newTab(defaultHomeURL)
		b.tabs = append(b.tabs, t)
		b.activeTab = len(b.tabs) - 1
		b.urlEditor.SetText(defaultHomeURL)
		go b.fetchURL(t, defaultHomeURL, true, w)
		return
	}

	// Handle tab switching & tab closing
	for i, t := range b.tabs {
		if t.CloseBtn.Clicked(gtx) {
			b.closeTab(i)
			w.Invalidate()
			return
		}
		if t.TabBtn.Clicked(gtx) {
			b.activeTab = i
			b.urlEditor.SetText(t.CurrentURL)
			w.Option(app.Title(t.Title))
			w.Invalidate()
			return
		}
	}

	active := b.getActiveTab()
	if active == nil {
		return
	}

	canBack := active.HistoryIdx > 0
	canFwd := active.HistoryIdx < len(active.History)-1
	curr := active.CurrentURL

	if canBack && b.backBtn.Clicked(gtx) {
		b.navigateHistory(active, -1, w)
		return
	}
	if canFwd && b.fwdBtn.Clicked(gtx) {
		b.navigateHistory(active, 1, w)
		return
	}
	if b.reloadBtn.Clicked(gtx) {
		target := curr
		if target == "" {
			target = strings.TrimSpace(b.urlEditor.Text())
		}
		if target != "" {
			target = normalizeURL(target)
			go b.fetchURL(active, target, false, w)
		}
		return
	}
	if b.homeBtn.Clicked(gtx) {
		b.urlEditor.SetText(defaultHomeURL)
		go b.fetchURL(active, defaultHomeURL, true, w)
		return
	}
	if b.bookmarkBtn.Clicked(gtx) {
		b.toggleBookmark()
		w.Invalidate()
		return
	}

	// Handle bookmarks bar clicks
	for _, bm := range b.bookmarks {
		if bm.Click.Clicked(gtx) {
			b.urlEditor.SetText(bm.URL)
			go b.fetchURL(active, bm.URL, true, w)
			return
		}
	}

	// Handle address bar submit
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
			go b.fetchURL(active, target, true, w)
		}
		return
	}

	// Handle elements click in current active tab
	elems := active.PageElements
	for _, elem := range elems {
		if elem.Type == TypeLink && elem.Click.Clicked(gtx) {
			target := resolveURL(curr, elem.Href)
			b.urlEditor.SetText(target)
			go b.fetchURL(active, target, true, w)
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
					go b.fetchURL(active, target, true, w)
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

func (b *BrowserApp) navigateHistory(tab *Tab, delta int, w *app.Window) {
	newIdx := tab.HistoryIdx + delta
	if newIdx < 0 || newIdx >= len(tab.History) {
		return
	}
	tab.HistoryIdx = newIdx
	target := tab.History[tab.HistoryIdx]
	b.saveHistory()

	if b.getActiveTab() == tab {
		b.urlEditor.SetText(target)
	}
	go b.fetchURL(tab, target, false, w)
}

func (b *BrowserApp) fetchURL(tab *Tab, rawURL string, recordHistory bool, w *app.Window) {
	b.mu.Lock()
	if tab.CancelFetch != nil {
		tab.CancelFetch()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	tab.CancelFetch = cancel
	tab.Loading = true
	tab.StatusText = "Loading: " + rawURL
	tab.PageElements = nil
	b.mu.Unlock()
	w.Invalidate()

	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil {
		b.setErrorState(tab, "Invalid URL: "+err.Error(), rawURL, w)
		return
	}

	req, err := http.NewRequestWithContext(ctx, "GET", parsedURL.String(), nil)
	if err != nil {
		b.setErrorState(tab, "Request error: "+err.Error(), rawURL, w)
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Kowser/1.0")

	resp, err := b.client.Do(req)
	if err != nil {
		if ctx.Err() == context.Canceled {
			return
		}
		b.setErrorState(tab, "Fetch failed: "+err.Error(), rawURL, w)
		return
	}
	defer resp.Body.Close()

	title, elems := parseHTML(resp.Body)

	for _, elem := range elems {
		if elem.Type == TypeImage && elem.Src != "" {
			elem.Src = resolveURL(resp.Request.URL.String(), elem.Src)
			b.fetchImageAsync(elem.Src)
		}
	}

	b.mu.Lock()
	finalURL := resp.Request.URL.String()
	tab.CurrentURL = finalURL
	tab.Loading = false
	if title == "" {
		title = finalURL
	}
	tab.Title = title
	tab.StatusText = fmt.Sprintf("HTTP %d %s | %d elements", resp.StatusCode, resp.Status, len(elems))
	tab.PageElements = elems

	if recordHistory {
		if tab.HistoryIdx >= 0 && tab.HistoryIdx < len(tab.History) {
			tab.History = append(tab.History[:tab.HistoryIdx+1], finalURL)
		} else {
			tab.History = append(tab.History, finalURL)
		}
		tab.HistoryIdx = len(tab.History) - 1
		b.saveHistory()
	}
	isActive := (b.getActiveTab() == tab)
	b.mu.Unlock()

	if isActive {
		w.Option(app.Title(title))
		b.urlEditor.SetText(finalURL)
	}
	w.Invalidate()
}

func (b *BrowserApp) setErrorState(tab *Tab, msg string, targetURL string, w *app.Window) {
	b.mu.Lock()
	tab.Loading = false
	tab.StatusText = msg
	tab.PageElements = []*PageElement{
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
			case "img":
				src := ""
				alt := ""
				for _, attr := range n.Attr {
					if attr.Key == "src" {
						src = attr.Val
					} else if attr.Key == "alt" {
						alt = attr.Val
					}
				}
				if src != "" {
					elems = append(elems, &PageElement{
						Type: TypeImage,
						Src:  src,
						Text: alt,
					})
				}
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
					if c.Type == html.ElementNode && (c.Data == "p" || c.Data == "div" || c.Data == "h1" || c.Data == "h2" || c.Data == "h3" || c.Data == "a" || c.Data == "ul" || c.Data == "ol" || c.Data == "form" || c.Data == "li" || c.Data == "strong" || c.Data == "b" || c.Data == "img") {
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
			return b.layoutTabBar(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return b.layoutTopBar(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return b.layoutBookmarkBar(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return b.layoutStatusBar(gtx)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return b.layoutContent(gtx)
		}),
	)
}

func (b *BrowserApp) layoutTabBar(gtx layout.Context) layout.Dimensions {
	b.mu.Lock()
	tabs := b.tabs
	activeIdx := b.activeTab
	b.mu.Unlock()

	activeBg := color.NRGBA{R: 240, G: 240, B: 240, A: 255}
	inactiveBg := color.NRGBA{R: 215, G: 215, B: 215, A: 255}
	closeRed := color.NRGBA{R: 200, G: 40, B: 40, A: 255}

	inset := layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(2), Left: unit.Dp(6), Right: unit.Dp(6)}
	return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return material.List(b.theme, &b.tabBar).Layout(gtx, len(tabs), func(gtx layout.Context, i int) layout.Dimensions {
					t := tabs[i]
					isActive := (i == activeIdx)

					tabInset := layout.Inset{Right: unit.Dp(4)}
					return tabInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								btn := material.Clickable(gtx, &t.TabBtn, func(gtx layout.Context) layout.Dimensions {
									pad := layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4), Left: unit.Dp(8), Right: unit.Dp(4)}
									return pad.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										title := t.Title
										if len(title) > 16 {
											title = title[:14] + ".."
										}
										lbl := material.Caption(b.theme, title)
										if isActive {
											lbl.Color = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
										} else {
											lbl.Color = color.NRGBA{R: 90, G: 90, B: 90, A: 255}
										}
										return lbl.Layout(gtx)
									})
								})
								_ = isActive
								_ = activeBg
								_ = inactiveBg
								return btn
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								cBtn := material.Clickable(gtx, &t.CloseBtn, func(gtx layout.Context) layout.Dimensions {
									cPad := layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(2), Right: unit.Dp(6)}
									return cPad.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										cLbl := material.Caption(b.theme, "x")
										cLbl.Color = closeRed
										return cLbl.Layout(gtx)
									})
								})
								return cBtn
							}),
						)
					})
				})
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				btn := material.Button(b.theme, &b.newTabBtn, "+")
				return btn.Layout(gtx)
			}),
		)
	})
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
				b.mu.Lock()
				bookmarked := b.isCurrentBookmarked()
				b.mu.Unlock()
				label := "★"
				if !bookmarked {
					label = "☆"
				}
				btn := material.Button(b.theme, &b.bookmarkBtn, label)
				return btn.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				btnText := "Go"
				tab := b.getActiveTab()
				if tab != nil && tab.Loading {
					btnText = "..."
				}
				btn := material.Button(b.theme, &b.goBtn, btnText)
				return btn.Layout(gtx)
			}),
		)
	})
}

func (b *BrowserApp) layoutBookmarkBar(gtx layout.Context) layout.Dimensions {
	b.mu.Lock()
	bms := b.bookmarks
	b.mu.Unlock()

	if len(bms) == 0 {
		return layout.Dimensions{}
	}

	inset := layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(4), Left: unit.Dp(8), Right: unit.Dp(8)}
	return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return material.List(b.theme, &b.bookmarkBar).Layout(gtx, len(bms), func(gtx layout.Context, i int) layout.Dimensions {
			bm := bms[i]
			btnInset := layout.Inset{Right: unit.Dp(8)}
			return btnInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				btn := material.Clickable(gtx, &bm.Click, func(gtx layout.Context) layout.Dimensions {
					lbl := material.Caption(b.theme, bm.Title)
					lbl.Color = color.NRGBA{R: 100, G: 100, B: 100, A: 255}
					return lbl.Layout(gtx)
				})
				return btn
			})
		})
	})
}

func (b *BrowserApp) layoutStatusBar(gtx layout.Context) layout.Dimensions {
	inset := layout.Inset{Left: unit.Dp(8), Right: unit.Dp(8), Bottom: unit.Dp(4)}
	return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		b.mu.Lock()
		tab := b.getActiveTab()
		txt := ""
		if tab != nil {
			txt = tab.StatusText
		}
		b.mu.Unlock()
		lbl := material.Caption(b.theme, txt)
		return lbl.Layout(gtx)
	})
}

func (b *BrowserApp) layoutContent(gtx layout.Context) layout.Dimensions {
	b.mu.Lock()
	tab := b.getActiveTab()
	if tab == nil {
		b.mu.Unlock()
		return layout.Dimensions{}
	}
	elems := tab.PageElements
	b.mu.Unlock()

	if len(elems) == 0 {
		inset := layout.UniformInset(unit.Dp(16))
		return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return material.Body2(b.theme, "No content loaded.").Layout(gtx)
		})
	}

	blue := color.NRGBA{R: 26, G: 115, B: 232, A: 255}
	red := color.NRGBA{R: 220, G: 50, B: 50, A: 255}

	return material.List(b.theme, &tab.ContentList).Layout(gtx, len(elems), func(gtx layout.Context, i int) layout.Dimensions {
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
			case TypeImage:
				img := b.getImage(elem.Src)
				if img != nil {
					imgWidget := widget.Image{
						Src:      paint.NewImageOp(img),
						Fit:      widget.ScaleDown,
						Position: layout.NW,
					}
					origConstraints := gtx.Constraints
					gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(240))
					gtx.Constraints.Max.X = gtx.Dp(unit.Dp(360))
					dims := imgWidget.Layout(gtx)
					gtx.Constraints = origConstraints
					return dims
				}
				alt := elem.Text
				if alt == "" {
					alt = "[Image]"
				} else {
					alt = "[Image: " + alt + "]"
				}
				lbl := material.Caption(b.theme, alt)
				lbl.Color = color.NRGBA{R: 128, G: 128, B: 128, A: 255}
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
