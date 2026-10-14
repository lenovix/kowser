package main

import (
	"strings"
	"testing"
)

func buildSampleTree() *Node {
	doc := NewDocument()
	htmlEl := NewElement("html", nil)
	doc.AppendChild(htmlEl)

	head := NewElement("head", nil)
	body := NewElement("body", map[string]string{"id": "main-body", "class": "page dark"})
	htmlEl.AppendChild(head)
	htmlEl.AppendChild(body)

	title := NewElement("title", nil)
	title.AppendChild(NewTextNode("Test Page"))
	head.AppendChild(title)

	h1 := NewElement("h1", map[string]string{"id": "heading", "class": "title big"})
	h1.AppendChild(NewTextNode("Hello World"))
	body.AppendChild(h1)

	div := NewElement("div", map[string]string{"class": "container"})
	body.AppendChild(div)

	p := NewElement("p", map[string]string{"class": "text"})
	p.AppendChild(NewTextNode("Some "))
	strong := NewElement("strong", nil)
	strong.AppendChild(NewTextNode("bold"))
	p.AppendChild(strong)
	p.AppendChild(NewTextNode(" text"))
	div.AppendChild(p)

	a := NewElement("a", map[string]string{"href": "https://google.com", "id": "link1"})
	a.AppendChild(NewTextNode("Google"))
	div.AppendChild(a)

	return doc
}

func TestTreeTopology(t *testing.T) {
	doc := buildSampleTree()

	htmlEl := doc.FirstChild
	if htmlEl == nil || htmlEl.Tag != "html" {
		t.Fatalf("expected html as first child of document")
	}
	if doc.LastChild != htmlEl {
		t.Errorf("expected LastChild == FirstChild for single child")
	}
	if htmlEl.Parent != doc {
		t.Errorf("html parent should be document")
	}

	head := htmlEl.FirstChild
	body := htmlEl.LastChild
	if head.Tag != "head" || body.Tag != "body" {
		t.Fatalf("expected head and body, got %q and %q", head.Tag, body.Tag)
	}
	if head.NextSibling != body {
		t.Errorf("head.NextSibling should be body")
	}
	if body.PrevSibling != head {
		t.Errorf("body.PrevSibling should be head")
	}
	if len(htmlEl.Children) != 2 {
		t.Errorf("expected 2 children of html, got %d", len(htmlEl.Children))
	}
	if head.Parent != htmlEl || body.Parent != htmlEl {
		t.Errorf("head/body parent should be html")
	}
}

func TestWalkOrder(t *testing.T) {
	doc := buildSampleTree()
	var tags []string
	doc.Walk(func(n *Node) bool {
		if n.Type == NodeElement {
			tags = append(tags, n.Tag)
		}
		return true
	})
	expected := []string{"html", "head", "title", "body", "h1", "div", "p", "strong", "a"}
	if len(tags) != len(expected) {
		t.Fatalf("walk count mismatch: got %v, want %v", tags, expected)
	}
	for i := range expected {
		if tags[i] != expected[i] {
			t.Fatalf("walk order mismatch at %d: got %v, want %v", i, tags, expected)
		}
	}
}

func TestWalkEarlyExit(t *testing.T) {
	doc := buildSampleTree()
	count := 0
	doc.Walk(func(n *Node) bool {
		count++
		return count < 3
	})
	if count != 3 {
		t.Errorf("expected early exit at 3, got %d", count)
	}
}

func TestTextContent(t *testing.T) {
	doc := buildSampleTree()
	body := doc.GetElementById("main-body")
	got := body.TextContent()
	for _, want := range []string{"Hello World", "Some", "bold", "text", "Google"} {
		if !strings.Contains(got, want) {
			t.Errorf("TextContent missing %q in %q", want, got)
		}
	}
}

func TestTextContentSkipsScriptStyle(t *testing.T) {
	doc := NewDocument()
	body := NewElement("body", nil)
	doc.AppendChild(body)
	body.AppendChild(NewTextNode("visible"))
	script := NewElement("script", nil)
	script.AppendChild(NewTextNode("secret-var"))
	body.AppendChild(script)
	style := NewElement("style", nil)
	style.AppendChild(NewTextNode(".hidden{color:red}"))
	body.AppendChild(style)

	got := body.TextContent()
	if !strings.Contains(got, "visible") {
		t.Errorf("expected visible text, got %q", got)
	}
	if strings.Contains(got, "secret-var") || strings.Contains(got, "hidden") {
		t.Errorf("script/style leaked into TextContent: %q", got)
	}
}

func TestGetElementById(t *testing.T) {
	doc := buildSampleTree()
	n := doc.GetElementById("heading")
	if n == nil || n.Tag != "h1" {
		t.Fatalf("GetElementById(heading) failed")
	}
	if doc.GetElementById("nonexistent") != nil {
		t.Errorf("expected nil for missing id")
	}
}

func TestGetElementsByTagName(t *testing.T) {
	doc := buildSampleTree()
	ps := doc.GetElementsByTagName("p")
	if len(ps) != 1 {
		t.Errorf("expected 1 p, got %d", len(ps))
	}
	as := doc.GetElementsByTagName("a")
	if len(as) != 1 || as[0].Attribute("href") != "https://google.com" {
		t.Errorf("GetElementsByTagName(a) failed")
	}
}

func TestGetElementsByClassName(t *testing.T) {
	doc := buildSampleTree()
	bigs := doc.GetElementsByClassName("big")
	if len(bigs) != 1 || bigs[0].Tag != "h1" {
		t.Errorf("expected h1.big")
	}
	containers := doc.GetElementsByClassName("container")
	if len(containers) != 1 {
		t.Errorf("expected 1 .container")
	}
}

func TestQuerySelectorSimple(t *testing.T) {
	doc := buildSampleTree()
	if n := doc.QuerySelector("#heading"); n == nil || n.Tag != "h1" {
		t.Errorf("QuerySelector(#heading) failed")
	}
	if n := doc.QuerySelector(".container"); n == nil || n.Tag != "div" {
		t.Errorf("QuerySelector(.container) failed")
	}
	if n := doc.QuerySelector("strong"); n == nil {
		t.Errorf("QuerySelector(strong) failed")
	}
	if n := doc.QuerySelector("h1.title.big"); n == nil {
		t.Errorf("QuerySelector(h1.title.big) failed")
	}
}

func TestQuerySelectorDescendant(t *testing.T) {
	doc := buildSampleTree()
	n := doc.QuerySelector("div p")
	if n == nil || n.Tag != "p" {
		t.Fatalf("QuerySelector(div p) failed")
	}
	n = doc.QuerySelector("body div.container a")
	if n == nil || n.Attribute("id") != "link1" {
		t.Fatalf("QuerySelector(body div.container a) failed")
	}
	if doc.QuerySelector("span a") != nil {
		t.Errorf("expected nil for span a")
	}
}

func TestQuerySelectorAll(t *testing.T) {
	doc := buildSampleTree()
	all := doc.QuerySelectorAll("div a")
	if len(all) != 1 {
		t.Errorf("expected 1 match for 'div a', got %d", len(all))
	}
}

func TestParseHTMLToDOM(t *testing.T) {
	src := `<!DOCTYPE html><html><head><title>Hi</title></head><body><h1 id="t">Hey</h1><p class="x">para <b>bold</b></p></body></html>`
	root, err := ParseHTMLToDOM(strings.NewReader(src))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	h1 := root.GetElementById("t")
	if h1 == nil || h1.TextContent() != "Hey" {
		t.Errorf("h1 lookup failed, got %+v", h1)
	}
	titles := root.GetElementsByTagName("title")
	if len(titles) != 1 || titles[0].TextContent() != "Hi" {
		t.Errorf("title lookup failed")
	}
	ps := root.GetElementsByClassName("x")
	if len(ps) != 1 {
		t.Errorf("class lookup failed")
	}
	if root.QuerySelector("body p b") == nil {
		t.Errorf("descendant query failed")
	}
}

func TestFlattenDOMRoundtrip(t *testing.T) {
	src := `<html><body><h1>Title</h1><a href="/x">Link</a><p>Body text</p><img src="/i.png" alt="pic"><form action="/s"><input name="q"></form></body></html>`
	root, err := ParseHTMLToDOM(strings.NewReader(src))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	elems := flattenDOMToPageElements(root, "https://example.com")
	if len(elems) == 0 {
		t.Fatalf("expected non-empty elements")
	}
	var hasH, hasL, hasT, hasImg, hasForm bool
	for _, e := range elems {
		switch e.Type {
		case TypeHeading:
			if strings.Contains(e.Text, "Title") {
				hasH = true
			}
		case TypeLink:
			if e.Href == "https://example.com/x" {
				hasL = true
			}
		case TypeText:
			if strings.Contains(e.Text, "Body text") {
				hasT = true
			}
		case TypeImage:
			if strings.Contains(e.Src, "/i.png") {
				hasImg = true
			}
		case TypeForm:
			hasForm = true
		}
	}
	if !hasH || !hasL || !hasT || !hasImg || !hasForm {
		t.Errorf("roundtrip missing elements h=%v l=%v t=%v img=%v form=%v", hasH, hasL, hasT, hasImg, hasForm)
	}
}
