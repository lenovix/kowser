package main

import (
	"strings"
	"testing"
)

func TestExtractTextAndLinks(t *testing.T) {
	htmlSample := `
	<!DOCTYPE html>
	<html>
	<head><title>Google</title><style>.hidden{display:none}</style></head>
	<body>
		<script>console.log("ignore")</script>
		<h1>Google Search</h1>
		<a href="https://about.google">About</a>
		<p>Search the world's information.</p>
	</body>
	</html>`

	elems := extractTextAndLinks(strings.NewReader(htmlSample))
	if len(elems) == 0 {
		t.Fatalf("expected non-empty elements")
	}

	joined := strings.Join(elems, " ")
	if !strings.Contains(joined, "Google Search") {
		t.Errorf("missing heading in output: %v", joined)
	}
	if strings.Contains(joined, "console.log") {
		t.Errorf("script tag text was not skipped: %v", joined)
	}
}
