package main

import (
	"io"
	"strings"

	"golang.org/x/net/html"
)

// ParseHTMLToDOM parses an HTML document from the given reader and returns
// the root Document node as a *Node tree.
func ParseHTMLToDOM(r io.Reader) (*Node, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}
	return convertHTMLNode(doc), nil
}

// convertHTMLNode recursively converts golang.org/x/net/html.Node into *Node.
func convertHTMLNode(n *html.Node) *Node {
	if n == nil {
		return nil
	}

	var node *Node

	switch n.Type {
	case html.DocumentNode:
		node = NewDocument()
	case html.ElementNode:
		attrs := make(map[string]string)
		for _, a := range n.Attr {
			attrs[strings.ToLower(a.Key)] = a.Val
		}
		node = NewElement(n.Data, attrs)
	case html.TextNode:
		node = NewTextNode(n.Data)
	case html.CommentNode:
		node = &Node{
			Type:       NodeComment,
			Value:      n.Data,
			Attributes: make(map[string]string),
		}
	default:
		node = NewDocument()
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		child := convertHTMLNode(c)
		if child != nil {
			node.AppendChild(child)
		}
	}

	return node
}
