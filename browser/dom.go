package main

import (
	"strings"
)

// NodeType represents the type of a DOM node.
type NodeType int

const (
	NodeDocument NodeType = iota
	NodeElement
	NodeText
	NodeComment
)

// Node represents a single node in the DOM tree.
type Node struct {
	Type       NodeType
	Tag        string            // lowercase tag name for Element nodes (e.g. "div", "p", "a")
	Value      string            // text content for Text nodes
	Attributes map[string]string // key-value pairs for attributes (lowercase keys)

	// Computed styles from CSS cascade engine
	Computed *ComputedStyle

	// Tree topology pointers
	Parent      *Node
	FirstChild  *Node
	LastChild   *Node
	PrevSibling *Node
	NextSibling *Node

	// Slice helper
	Children []*Node
}

// NewDocument creates a root Document node.
func NewDocument() *Node {
	return &Node{
		Type:       NodeDocument,
		Attributes: make(map[string]string),
	}
}

// NewElement creates an element node with a specified tag and optional attributes.
func NewElement(tag string, attrs map[string]string) *Node {
	if attrs == nil {
		attrs = make(map[string]string)
	}
	return &Node{
		Type:       NodeElement,
		Tag:        strings.ToLower(strings.TrimSpace(tag)),
		Attributes: attrs,
	}
}

// NewTextNode creates a text node.
func NewTextNode(val string) *Node {
	return &Node{
		Type:       NodeText,
		Value:      val,
		Attributes: make(map[string]string),
	}
}

// AppendChild adds a child node to the parent and maintains tree topology.
func (n *Node) AppendChild(child *Node) {
	if child == nil {
		return
	}
	child.Parent = n

	if n.FirstChild == nil {
		n.FirstChild = child
		n.LastChild = child
	} else {
		prev := n.LastChild
		prev.NextSibling = child
		child.PrevSibling = prev
		n.LastChild = child
	}

	n.Children = append(n.Children, child)
}

// Walk traverses the subtree in pre-order. If fn returns false, traversal terminates.
func (n *Node) Walk(fn func(curr *Node) bool) bool {
	if n == nil {
		return true
	}
	if !fn(n) {
		return false
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if !c.Walk(fn) {
			return false
		}
	}
	return true
}

// TextContent returns the combined text of this node and all descendants,
// excluding contents within <script> and <style> tags.
func (n *Node) TextContent() string {
	if n == nil {
		return ""
	}
	if n.Type == NodeText {
		return n.Value
	}
	if n.Type == NodeElement && (n.Tag == "script" || n.Tag == "style" || n.Tag == "noscript") {
		return ""
	}

	var sb strings.Builder
	var collect func(curr *Node)
	collect = func(curr *Node) {
		if curr == nil {
			return
		}
		if curr.Type == NodeText {
			sb.WriteString(curr.Value)
			sb.WriteString(" ")
			return
		}
		if curr.Type == NodeElement && (curr.Tag == "script" || curr.Tag == "style" || curr.Tag == "noscript") {
			return
		}
		for c := curr.FirstChild; c != nil; c = c.NextSibling {
			collect(c)
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		collect(c)
	}
	return strings.TrimSpace(sb.String())
}

// Attribute returns the attribute value by key (case-insensitive).
func (n *Node) Attribute(key string) string {
	if n.Attributes == nil {
		return ""
	}
	return n.Attributes[strings.ToLower(key)]
}

// HasAttribute checks if an attribute exists.
func (n *Node) HasAttribute(key string) bool {
	if n.Attributes == nil {
		return false
	}
	_, ok := n.Attributes[strings.ToLower(key)]
	return ok
}

// GetElementById returns the first descendant with the given ID.
func (n *Node) GetElementById(id string) *Node {
	var found *Node
	n.Walk(func(curr *Node) bool {
		if curr.Type == NodeElement && curr.Attribute("id") == id {
			found = curr
			return false
		}
		return true
	})
	return found
}

// GetElementsByTagName returns all descendants with the given tag name.
func (n *Node) GetElementsByTagName(tag string) []*Node {
	tag = strings.ToLower(strings.TrimSpace(tag))
	var res []*Node
	n.Walk(func(curr *Node) bool {
		if curr.Type == NodeElement && (curr.Tag == tag || tag == "*") {
			res = append(res, curr)
		}
		return true
	})
	return res
}

// GetElementsByClassName returns all descendants containing the given class name.
func (n *Node) GetElementsByClassName(className string) []*Node {
	className = strings.TrimSpace(className)
	var res []*Node
	n.Walk(func(curr *Node) bool {
		if curr.Type == NodeElement && curr.HasClass(className) {
			res = append(res, curr)
		}
		return true
	})
	return res
}

// HasClass returns true if the element has the specified class.
func (n *Node) HasClass(className string) bool {
	classes := strings.Fields(n.Attribute("class"))
	for _, c := range classes {
		if c == className {
			return true
		}
	}
	return false
}

// matchesSimpleSelector checks if a node matches a single compound selector part
// like "div", "#header", ".btn", or "div.btn#submit".
func (n *Node) matchesSimpleSelector(sel string) bool {
	if n.Type != NodeElement {
		return false
	}
	sel = strings.TrimSpace(sel)
	if sel == "" || sel == "*" {
		return true
	}

	// Tokenize selector for tag, id, and classes
	var tag, id string
	var classes []string

	var token strings.Builder
	mode := "tag" // "tag", "id", "class"

	flush := func() {
		str := token.String()
		if str != "" {
			switch mode {
			case "tag":
				tag = strings.ToLower(str)
			case "id":
				id = str
			case "class":
				classes = append(classes, str)
			}
		}
		token.Reset()
	}

	for _, ch := range sel {
		if ch == '#' {
			flush()
			mode = "id"
		} else if ch == '.' {
			flush()
			mode = "class"
		} else {
			token.WriteRune(ch)
		}
	}
	flush()

	if tag != "" && n.Tag != tag {
		return false
	}
	if id != "" && n.Attribute("id") != id {
		return false
	}
	for _, c := range classes {
		if !n.HasClass(c) {
			return false
		}
	}
	return true
}

// QuerySelector returns the first node matching the selector.
// Supports space-separated descendant combinators (e.g. "div.main #item a").
func (n *Node) QuerySelector(selector string) *Node {
	all := n.QuerySelectorAll(selector)
	if len(all) > 0 {
		return all[0]
	}
	return nil
}

// QuerySelectorAll returns all nodes matching the selector.
// Supports descendant combinators like "div.container p.text a".
func (n *Node) QuerySelectorAll(selector string) []*Node {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return nil
	}

	parts := strings.Fields(selector)
	if len(parts) == 0 {
		return nil
	}

	var candidates []*Node
	n.Walk(func(curr *Node) bool {
		if curr.Type == NodeElement {
			candidates = append(candidates, curr)
		}
		return true
	})

	var matchChain func(node *Node, partIdx int) bool
	matchChain = func(node *Node, partIdx int) bool {
		if partIdx < 0 {
			return true
		}
		if node == nil || node.Type != NodeElement {
			return false
		}
		if !node.matchesSimpleSelector(parts[partIdx]) {
			return false
		}
		if partIdx == 0 {
			return true
		}
		// Check ancestor chain for previous parts
		for p := node.Parent; p != nil; p = p.Parent {
			if matchChain(p, partIdx-1) {
				return true
			}
		}
		return false
	}

	var matched []*Node
	lastIdx := len(parts) - 1
	for _, cand := range candidates {
		if cand.matchesSimpleSelector(parts[lastIdx]) {
			if matchChain(cand, lastIdx) {
				matched = append(matched, cand)
			}
		}
	}

	return matched
}
