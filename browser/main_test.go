package main

import (
	"strings"
	"testing"
)

func TestParseHTML(t *testing.T) {
	htmlSample := `
	<!DOCTYPE html>
	<html>
	<head><title>Google</title><style>.hidden{display:none}</style></head>
	<body>
		<script>console.log("ignore")</script>
		<h1>Google Search</h1>
		<a href="https://about.google">About Google</a>
		<p>Search the world's information.</p>
	</body>
	</html>`

	_, elems := parseHTML(strings.NewReader(htmlSample))
	if len(elems) == 0 {
		t.Fatalf("expected non-empty elements")
	}

	var foundHeading, foundLink, foundText bool
	for _, e := range elems {
		if e.Type == TypeHeading && strings.Contains(e.Text, "Google Search") {
			foundHeading = true
		}
		if e.Type == TypeLink && e.Href == "https://about.google" && strings.Contains(e.Text, "About Google") {
			foundLink = true
		}
		if e.Type == TypeText && strings.Contains(e.Text, "Search the world") {
			foundText = true
		}
	}

	if !foundHeading {
		t.Errorf("heading not extracted properly")
	}
	if !foundLink {
		t.Errorf("link not extracted properly")
	}
	if !foundText {
		t.Errorf("paragraph text not extracted properly")
	}
}

func TestResolveURL(t *testing.T) {
	tests := []struct {
		base     string
		ref      string
		expected string
	}{
		{"https://google.com", "/search?q=test", "https://google.com/search?q=test"},
		{"https://google.com/a/b", "c", "https://google.com/a/c"},
		{"https://google.com", "https://example.com", "https://example.com"},
	}

	for _, tt := range tests {
		got := resolveURL(tt.base, tt.ref)
		if got != tt.expected {
			t.Errorf("resolveURL(%q, %q) = %q; want %q", tt.base, tt.ref, got, tt.expected)
		}
	}
}
