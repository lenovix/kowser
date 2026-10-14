package main

import (
	"strings"
	"testing"
)

func TestParseHTMLAdvanced(t *testing.T) {
	htmlSample := `
	<!DOCTYPE html>
	<html>
	<head><title>Test Page Title</title></head>
	<body>
		<h1>Main Title</h1>
		<img src="/images/branding/googlelogo.png" alt="Google Logo" />
		<form action="/search" method="GET">
			<input name="q" />
		</form>
		<ul>
			<li>Bullet item one</li>
		</ul>
		<p>Normal text with <strong>bold text</strong>.</p>
	</body>
	</html>`

	title, elems := parseHTML(strings.NewReader(htmlSample))
	if title != "Test Page Title" {
		t.Errorf("expected title 'Test Page Title', got %q", title)
	}

	var foundForm, foundHeading, foundBullet, foundBold, foundImage bool
	for _, e := range elems {
		switch e.Type {
		case TypeImage:
			if e.Src == "/images/branding/googlelogo.png" && e.Text == "Google Logo" {
				foundImage = true
			}
		case TypeForm:
			foundForm = true
		case TypeHeading:
			if e.Level == 1 && e.Text == "Main Title" {
				foundHeading = true
			}
		case TypeText:
			if e.IsBullet && strings.Contains(e.Text, "Bullet item one") {
				foundBullet = true
			}
			if e.IsBold && strings.Contains(e.Text, "bold text") {
				foundBold = true
			}
		}
	}

	if !foundImage {
		t.Errorf("image element not parsed properly")
	}
	if !foundForm {
		t.Errorf("form element not parsed")
	}
	if !foundHeading {
		t.Errorf("h1 heading not parsed")
	}
	if !foundBullet {
		t.Errorf("li bullet not parsed")
	}
	if !foundBold {
		t.Errorf("strong bold text not parsed")
	}
}

func TestBookmarkToggle(t *testing.T) {
	app := &BrowserApp{
		currentURL: "https://news.ycombinator.com",
		pageTitle:  "Hacker News",
	}

	if app.isCurrentBookmarked() {
		t.Errorf("expected initially not bookmarked")
	}

	app.toggleBookmark()
	if !app.isCurrentBookmarked() {
		t.Errorf("expected bookmarked after toggle")
	}

	app.toggleBookmark()
	if app.isCurrentBookmarked() {
		t.Errorf("expected not bookmarked after second toggle")
	}
}
