package main

import (
	"image/color"
	"strings"
	"testing"
)

func TestParseCSSColorHex(t *testing.T) {
	c := parseCSSColor("#ff0000")
	if c == nil || c.R != 255 || c.G != 0 || c.B != 0 || c.A != 255 {
		t.Errorf("expected red, got %+v", c)
	}
	c = parseCSSColor("#abc")
	if c == nil || c.R != 0xaa || c.G != 0xbb || c.B != 0xcc {
		t.Errorf("expected short hex expansion, got %+v", c)
	}
}

func TestParseCSSColorNamed(t *testing.T) {
	c := parseCSSColor("blue")
	want := color.NRGBA{0, 0, 255, 255}
	if c == nil || *c != want {
		t.Errorf("expected blue, got %+v", c)
	}
	if parseCSSColor("notacolor") != nil {
		t.Errorf("expected nil for unknown color")
	}
}

func TestParseFontSize(t *testing.T) {
	if size, ok := parseFontSize("24px"); !ok || size != 24 {
		t.Errorf("expected 24px, got %v %v", size, ok)
	}
	if size, ok := parseFontSize("large"); !ok || size != 19 {
		t.Errorf("expected large=19, got %v %v", size, ok)
	}
	if _, ok := parseFontSize("bogus"); ok {
		t.Errorf("expected false for bogus size")
	}
}

func TestParseStyleString(t *testing.T) {
	sp := parseStyle("color: #ff0000; font-size: 20px; background-color: yellow")
	if sp.Color == nil || sp.Color.R != 255 {
		t.Errorf("color not parsed: %+v", sp.Color)
	}
	if !sp.HasFontSize || sp.FontSizeSp != 20 {
		t.Errorf("font size not parsed: %+v", sp)
	}
	if sp.Background == nil || sp.Background.B != 0 || sp.Background.G != 255 {
		t.Errorf("background not parsed: %+v", sp.Background)
	}
}

func TestStyleAttrInParseHTML(t *testing.T) {
	doc := `<p style="color: #00ff00">green text</p>`
	_, elems := parseHTML(strings.NewReader(doc))
	found := false
	for _, e := range elems {
		if e.Style.Color != nil && e.Style.Color.G == 255 && strings.Contains(e.Text, "green") {
			found = true
		}
	}
	if !found {
		t.Errorf("style attribute not applied to paragraph")
	}
}

func TestIsDownloadableFile(t *testing.T) {
	tests := []struct {
		ct       string
		cd       string
		expected bool
	}{
		{"text/html; charset=utf-8", "", false},
		{"application/xhtml+xml", "", false},
		{"application/pdf", "", true},
		{"application/zip", "", true},
		{"", "attachment; filename=report.pdf", true},
		{"audio/mpeg", "", true},
		{"text/plain", "", false},
	}
	for _, tt := range tests {
		got := isDownloadableFile(tt.ct, tt.cd)
		if got != tt.expected {
			t.Errorf("isDownloadableFile(%q, %q) = %v; want %v", tt.ct, tt.cd, got, tt.expected)
		}
	}
}
