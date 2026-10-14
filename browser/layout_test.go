package main

import (
	"strings"
	"testing"
)

func TestBuildLayoutTreeIgnoresDisplayNone(t *testing.T) {
	html := `
	<html>
	<head><style>.hidden { display: none; }</style></head>
	<body>
		<div id="v1">Visible 1</div>
		<div id="h1" class="hidden">Hidden</div>
		<div id="v2">Visible 2</div>
	</body>
	</html>`

	root, err := ParseHTMLToDOM(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	ComputeStyles(root)

	box := BuildLayoutTree(root)
	if box == nil {
		t.Fatalf("expected layout tree")
	}

	// Verify hidden div is not in layout tree
	var foundHidden bool
	var walk func(b *LayoutBox)
	walk = func(b *LayoutBox) {
		if b.Node != nil && b.Node.Attribute("id") == "h1" {
			foundHidden = true
		}
		for _, c := range b.Children {
			walk(c)
		}
	}
	walk(box)

	if foundHidden {
		t.Errorf("expected display:none node to be excluded from layout tree")
	}
}

func TestComputeLayoutBoxModelGeometry(t *testing.T) {
	html := `
	<html>
	<head>
		<style>
			#box1 {
				margin-top: 10px;
				margin-bottom: 20px;
				padding-left: 15px;
				padding-right: 15px;
				width: 300px;
				height: 100px;
			}
		</style>
	</head>
	<body>
		<div id="box1">Box Content</div>
	</body>
	</html>`

	root, err := ParseHTMLToDOM(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	ComputeStyles(root)

	layoutRoot := BuildLayoutTree(root)
	ComputeLayout(layoutRoot, 800)

	// Find box1
	var box1 *LayoutBox
	var find func(b *LayoutBox)
	find = func(b *LayoutBox) {
		if b.Node != nil && b.Node.Attribute("id") == "box1" {
			box1 = b
			return
		}
		for _, c := range b.Children {
			find(c)
		}
	}
	find(layoutRoot)

	if box1 == nil {
		t.Fatalf("box1 not found in layout tree")
	}

	if box1.Dims.Width != 300 {
		t.Errorf("expected explicit width 300, got %v", box1.Dims.Width)
	}
	if box1.Dims.Height != 100 {
		t.Errorf("expected explicit height 100, got %v", box1.Dims.Height)
	}
	if box1.Dims.Padding.Left != 15 || box1.Dims.Padding.Right != 15 {
		t.Errorf("expected padding 15, got %+v", box1.Dims.Padding)
	}
	if box1.Dims.Margin.Top != 10 || box1.Dims.Margin.Bottom != 20 {
		t.Errorf("expected margin 10/20, got %+v", box1.Dims.Margin)
	}
}

func TestWordWrappingBreaksLines(t *testing.T) {
	// A long paragraph that should wrap into multiple lines when width is small (e.g. 100px)
	html := `
	<html>
	<body>
		<p id="p1">This is a long sentence with many words that must wrap across multiple lines because the container is very narrow.</p>
	</body>
	</html>`

	root, err := ParseHTMLToDOM(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	ComputeStyles(root)

	layoutRoot := BuildLayoutTree(root)
	ComputeLayout(layoutRoot, 120) // Narrow container width: 120px

	var pBox *LayoutBox
	var find func(b *LayoutBox)
	find = func(b *LayoutBox) {
		if b.Node != nil && b.Node.Tag == "p" {
			pBox = b
			return
		}
		for _, c := range b.Children {
			find(c)
		}
	}
	find(layoutRoot)

	if pBox == nil {
		t.Fatalf("p box not found")
	}

	if len(pBox.Lines) <= 1 {
		t.Fatalf("expected paragraph to wrap into > 1 line, got %d lines", len(pBox.Lines))
	}

	// Verify each line has Y offset advancing
	for i := 1; i < len(pBox.Lines); i++ {
		if pBox.Lines[i].Y <= pBox.Lines[i-1].Y {
			t.Errorf("line %d Y (%v) should be greater than line %d Y (%v)", i, pBox.Lines[i].Y, i-1, pBox.Lines[i-1].Y)
		}
	}
}

func TestVerticalMarginCollapsing(t *testing.T) {
	html := `
	<html>
	<head>
		<style>
			#d1 { margin-bottom: 30px; height: 50px; }
			#d2 { margin-top: 20px; height: 50px; }
		</style>
	</head>
	<body>
		<div id="d1">Block 1</div>
		<div id="d2">Block 2</div>
	</body>
	</html>`

	root, err := ParseHTMLToDOM(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	ComputeStyles(root)

	layoutRoot := BuildLayoutTree(root)
	ComputeLayout(layoutRoot, 500)

	var d1, d2 *LayoutBox
	var find func(b *LayoutBox)
	find = func(b *LayoutBox) {
		if b.Node != nil {
			if b.Node.Attribute("id") == "d1" {
				d1 = b
			} else if b.Node.Attribute("id") == "d2" {
				d2 = b
			}
		}
		for _, c := range b.Children {
			find(c)
		}
	}
	find(layoutRoot)

	if d1 == nil || d2 == nil {
		t.Fatalf("d1 or d2 not found")
	}

	// In margin collapsing:
	// d1 bottom margin is 30px, d2 top margin is 20px.
	// Collapsed margin gap between d1 bottom and d2 top should be max(30, 20) = 30px.
	// d1 ends at d1.Dims.Y + d1.Dims.Height = d1.Dims.Y + 50.
	// d2.Dims.Y should equal d1.Dims.Y + 50 + 30 = d1.Dims.Y + 80.
	gap := d2.Dims.Y - (d1.Dims.Y + d1.Dims.Height)
	if gap != 30 {
		t.Errorf("expected collapsed gap of 30px, got %v (d1.Y=%v, d2.Y=%v)", gap, d1.Dims.Y, d2.Dims.Y)
	}
}
