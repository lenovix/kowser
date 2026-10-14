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
	tab := &Tab{
		CurrentURL: "https://news.ycombinator.com",
		Title:      "Hacker News",
	}
	app := &BrowserApp{
		tabs:      []*Tab{tab},
		activeTab: 0,
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

func TestMultiTabManagement(t *testing.T) {
	app := &BrowserApp{}
	t1 := app.newTab("https://www.google.com")
	t2 := app.newTab("https://go.dev")
	app.tabs = []*Tab{t1, t2}
	app.activeTab = 0

	if len(app.tabs) != 2 {
		t.Fatalf("expected 2 tabs, got %d", len(app.tabs))
	}
	if app.getActiveTab().CurrentURL != "https://www.google.com" {
		t.Errorf("expected active tab 0")
	}

	// Switch tab
	app.activeTab = 1
	if app.getActiveTab().CurrentURL != "https://go.dev" {
		t.Errorf("expected active tab 1")
	}

	// Close tab 1
	app.closeTab(1)
	if len(app.tabs) != 1 {
		t.Fatalf("expected 1 tab remaining, got %d", len(app.tabs))
	}
	if app.getActiveTab().CurrentURL != "https://www.google.com" {
		t.Errorf("expected remaining tab to be google.com")
	}

	// Close last remaining tab: should reset to default instead of empty slice
	app.closeTab(0)
	if len(app.tabs) != 1 {
		t.Fatalf("closing last tab should keep 1 reset tab, got %d", len(app.tabs))
	}
	if app.getActiveTab().CurrentURL != defaultHomeURL {
		t.Errorf("expected reset tab to point to defaultHomeURL")
	}
}
