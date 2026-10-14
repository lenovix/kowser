package main

import (
	"image/color"
	"strings"
	"testing"

	"gioui.org/unit"
)

func TestParseCSSBasic(t *testing.T) {
	css := `
	/* Comments should be stripped */
	h1 {
		color: red;
		font-size: 32px;
	}
	.container {
		background-color: #00ff00;
		padding: 10px;
	}
	#main-header {
		color: blue;
		margin-top: 20px;
	}
	`
	sheet := ParseCSS(css)
	if len(sheet.Rules) != 3 {
		t.Fatalf("expected 3 rules, got %d", len(sheet.Rules))
	}

	// Rule 1: h1
	r0 := sheet.Rules[0]
	if r0.Selector != "h1" || r0.Specificity != (Specificity{A: 0, B: 0, C: 1}) {
		t.Errorf("r0 mismatch: %+v", r0)
	}

	// Rule 2: .container
	r1 := sheet.Rules[1]
	if r1.Selector != ".container" || r1.Specificity != (Specificity{A: 0, B: 1, C: 0}) {
		t.Errorf("r1 mismatch: %+v", r1)
	}

	// Rule 3: #main-header
	r2 := sheet.Rules[2]
	if r2.Selector != "#main-header" || r2.Specificity != (Specificity{A: 1, B: 0, C: 0}) {
		t.Errorf("r2 mismatch: %+v", r2)
	}
}

func TestSpecificityComparison(t *testing.T) {
	tag := Specificity{A: 0, B: 0, C: 1}
	class := Specificity{A: 0, B: 1, C: 0}
	id := Specificity{A: 1, B: 0, C: 0}

	if !tag.Less(class) {
		t.Errorf("tag should have less specificity than class")
	}
	if !class.Less(id) {
		t.Errorf("class should have less specificity than id")
	}
	if !tag.Less(id) {
		t.Errorf("tag should have less specificity than id")
	}
}

func TestComputeStylesCascade(t *testing.T) {
	htmlText := `
	<html>
	<head>
		<style>
			div { color: black; font-size: 14px; }
			.alert { color: orange; font-weight: bold; }
			#critical { color: red; font-size: 20px; }
		</style>
	</head>
	<body>
		<div id="first">Normal Div</div>
		<div class="alert" id="second">Alert Div</div>
		<div class="alert" id="critical">Critical Div</div>
		<div style="color: blue;" id="override">Inline Override Div</div>
	</body>
	</html>
	`
	root, err := ParseHTMLToDOM(strings.NewReader(htmlText))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ComputeStyles(root)

	// Check normal div (matches 'div' tag)
	first := root.GetElementById("first")
	if first == nil || first.Computed == nil {
		t.Fatalf("first element or computed style missing")
	}
	if first.Computed.Color != (color.NRGBA{R: 0, G: 0, B: 0, A: 255}) {
		t.Errorf("first color mismatch: got %+v", first.Computed.Color)
	}
	if first.Computed.FontSize != unit.Sp(14) {
		t.Errorf("first font size mismatch: got %v", first.Computed.FontSize)
	}

	// Check alert div (matches 'div' and '.alert' -> .alert wins for color)
	second := root.GetElementById("second")
	if second == nil || second.Computed == nil {
		t.Fatalf("second element missing")
	}
	if second.Computed.Color != (color.NRGBA{R: 255, G: 165, B: 0, A: 255}) {
		t.Errorf("second color mismatch (should be orange): got %+v", second.Computed.Color)
	}
	if !second.Computed.IsBold {
		t.Errorf("second should be bold")
	}

	// Check critical div (matches 'div', '.alert', and '#critical' -> #critical wins for color & font-size)
	critical := root.GetElementById("critical")
	if critical == nil || critical.Computed == nil {
		t.Fatalf("critical element missing")
	}
	if critical.Computed.Color != (color.NRGBA{R: 255, G: 0, B: 0, A: 255}) {
		t.Errorf("critical color mismatch (should be red): got %+v", critical.Computed.Color)
	}
	if critical.Computed.FontSize != unit.Sp(20) {
		t.Errorf("critical font size mismatch: got %v", critical.Computed.FontSize)
	}
	if !critical.Computed.IsBold {
		t.Errorf("critical should inherit or keep bold from .alert")
	}

	// Check inline style override (style="color: blue;")
	override := root.GetElementById("override")
	if override == nil || override.Computed == nil {
		t.Fatalf("override element missing")
	}
	if override.Computed.Color != (color.NRGBA{R: 0, G: 0, B: 255, A: 255}) {
		t.Errorf("override color mismatch (should be blue): got %+v", override.Computed.Color)
	}
}

func TestStyleInheritance(t *testing.T) {
	htmlText := `
	<html>
	<head>
		<style>
			.parent-box { color: purple; font-size: 18px; }
		</style>
	</head>
	<body>
		<div class="parent-box">
			<p id="child-p">Child text should inherit color and size</p>
		</div>
	</body>
	</html>
	`
	root, err := ParseHTMLToDOM(strings.NewReader(htmlText))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ComputeStyles(root)

	child := root.GetElementById("child-p")
	if child == nil || child.Computed == nil {
		t.Fatalf("child element missing")
	}
	// Inherited color should be purple: {128, 0, 128, 255}
	if child.Computed.Color != (color.NRGBA{R: 128, G: 0, B: 128, A: 255}) {
		t.Errorf("child failed to inherit color: got %+v", child.Computed.Color)
	}
	if child.Computed.FontSize != unit.Sp(18) {
		t.Errorf("child failed to inherit font size: got %v", child.Computed.FontSize)
	}
}
