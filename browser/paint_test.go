package main

import (
	"image"
	"strings"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

func TestPaintLayoutBasicDocument(t *testing.T) {
	html := `
	<html>
	<head>
		<style>
			h1 { color: red; font-size: 24px; }
			p { color: black; font-size: 16px; margin-top: 10px; }
			a { color: blue; }
		</style>
	</head>
	<body>
		<h1>Page Heading</h1>
		<p>This is a paragraph with a <a href="https://example.com/target">click link</a> inside it.</p>
	</body>
	</html>`

	root, err := ParseHTMLToDOM(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	ComputeStyles(root)

	box := BuildLayoutTree(root)
	ComputeLayout(box, 600)

	var ops op.Ops
	gtx := layout.Context{
		Ops: &ops,
		Constraints: layout.Constraints{
			Max: image.Pt(600, 800),
		},
	}

	th := material.NewTheme()
	pc := &PaintContext{
		Theme:   th,
		ScrollY: 0,
		BaseURL: "https://example.com",
		Links:   make(map[*Node]*widget.Clickable),
	}

	totalH, clicked := PaintLayout(gtx, box, pc)
	if totalH <= 0 {
		t.Errorf("expected positive total height from paint pass, got %v", totalH)
	}
	if clicked != "" {
		t.Errorf("expected no click without user input, got %q", clicked)
	}

	// Verify links clickable map was populated for the link node
	linkNodes := root.GetElementsByTagName("a")
	if len(linkNodes) != 1 {
		t.Fatalf("expected 1 link node, got %d", len(linkNodes))
	}
	if pc.Links[linkNodes[0]] == nil {
		t.Errorf("expected clickable widget allocated for link node")
	}
}
