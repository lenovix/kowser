package main

import (
	"image/color"
	"sort"
	"strconv"
	"strings"

	"gioui.org/unit"
)

// DisplayType represents the CSS display property.
type DisplayType int

const (
	DisplayBlock DisplayType = iota
	DisplayInline
	DisplayNone
)

// Specificity represents CSS selector specificity: (ID, Class, Tag).
type Specificity struct {
	A int // IDs (#id)
	B int // Classes (.class, attributes)
	C int // Tags (div, p)
}

func (s Specificity) Less(other Specificity) bool {
	if s.A != other.A {
		return s.A < other.A
	}
	if s.B != other.B {
		return s.B < other.B
	}
	return s.C < other.C
}

// CSSDeclaration represents a single CSS property-value pair.
type CSSDeclaration struct {
	Property string
	Value    string
}

// CSSRule represents a single rule: selector + declarations + calculated specificity.
type CSSRule struct {
	Selector    string
	Declarations []CSSDeclaration
	Specificity  Specificity
	Order        int // Rule index in stylesheet for tie-breaking
}

// Stylesheet represents a collection of parsed CSS rules.
type Stylesheet struct {
	Rules []CSSRule
}

// ComputedStyle holds all computed visual values for a DOM node.
type ComputedStyle struct {
	Display         DisplayType
	Color           color.NRGBA
	BackgroundColor *color.NRGBA
	FontSize        unit.Sp
	IsBold          bool
	IsItalic        bool
	Width           float32
	Height          float32
	MarginTop       float32
	MarginBottom    float32
	MarginLeft      float32
	MarginRight     float32
	PaddingTop      float32
	PaddingBottom   float32
	PaddingLeft     float32
	PaddingRight    float32
}

// DefaultStyle returns standard user-agent default styles.
func DefaultStyle(tag string) ComputedStyle {
	cs := ComputedStyle{
		Display:   DisplayBlock,
		Color:     color.NRGBA{R: 0, G: 0, B: 0, A: 255}, // default black text
		FontSize:  unit.Sp(16),
		MarginTop: 0,
	}

	switch strings.ToLower(tag) {
	case "a", "span", "b", "strong", "i", "em":
		cs.Display = DisplayInline
	case "script", "style", "head", "noscript":
		cs.Display = DisplayNone
	}

	switch strings.ToLower(tag) {
	case "h1":
		cs.FontSize = unit.Sp(32)
		cs.IsBold = true
		cs.MarginTop = 16
		cs.MarginBottom = 16
	case "h2":
		cs.FontSize = unit.Sp(24)
		cs.IsBold = true
		cs.MarginTop = 14
		cs.MarginBottom = 14
	case "h3":
		cs.FontSize = unit.Sp(20)
		cs.IsBold = true
		cs.MarginTop = 12
		cs.MarginBottom = 12
	case "h4":
		cs.FontSize = unit.Sp(16)
		cs.IsBold = true
	case "h5":
		cs.FontSize = unit.Sp(14)
		cs.IsBold = true
	case "h6":
		cs.FontSize = unit.Sp(12)
		cs.IsBold = true
	case "p":
		cs.MarginTop = 12
		cs.MarginBottom = 12
	case "b", "strong":
		cs.IsBold = true
	case "i", "em":
		cs.IsItalic = true
	case "a":
		cs.Color = color.NRGBA{R: 26, G: 115, B: 232, A: 255} // Google blue
	}

	return cs
}

// ParseCSS parses raw CSS text (e.g. from <style> tags) into a Stylesheet.
func ParseCSS(cssText string) Stylesheet {
	var sheet Stylesheet
	order := 0

	// Strip CSS comments /* ... */
	cssText = stripCSSComments(cssText)

	// Split by '}'
	rawRules := strings.Split(cssText, "}")
	for _, rawRule := range rawRules {
		rawRule = strings.TrimSpace(rawRule)
		if rawRule == "" {
			continue
		}
		parts := strings.SplitN(rawRule, "{", 2)
		if len(parts) != 2 {
			continue
		}

		selList := strings.Split(parts[0], ",")
		body := parts[1]
		decls := parseDeclarations(body)
		if len(decls) == 0 {
			continue
		}

		for _, sel := range selList {
			sel = strings.TrimSpace(sel)
			if sel == "" {
				continue
			}
			spec := calculateSpecificity(sel)
			sheet.Rules = append(sheet.Rules, CSSRule{
				Selector:     sel,
				Declarations: decls,
				Specificity:  spec,
				Order:        order,
			})
			order++
		}
	}
	return sheet
}

func stripCSSComments(text string) string {
	var sb strings.Builder
	for {
		start := strings.Index(text, "/*")
		if start == -1 {
			sb.WriteString(text)
			break
		}
		sb.WriteString(text[:start])
		end := strings.Index(text[start+2:], "*/")
		if end == -1 {
			break
		}
		text = text[start+2+end+2:]
	}
	return sb.String()
}

func parseDeclarations(body string) []CSSDeclaration {
	var decls []CSSDeclaration
	lines := strings.Split(body, ";")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		kv := strings.SplitN(line, ":", 2)
		if len(kv) != 2 {
			continue
		}
		p := strings.TrimSpace(strings.ToLower(kv[0]))
		v := strings.TrimSpace(kv[1])
		if p != "" && v != "" {
			decls = append(decls, CSSDeclaration{Property: p, Value: v})
		}
	}
	return decls
}

// calculateSpecificity computes (ID, Class, Tag) counts for a selector.
func calculateSpecificity(selector string) Specificity {
	var spec Specificity
	parts := strings.Fields(selector)
	for _, part := range parts {
		if strings.HasPrefix(part, "#") {
			spec.A++
		} else if strings.HasPrefix(part, ".") {
			spec.B++
		} else {
			// Check compound parts like div.myclass#myid
			subParts := tokenizeCompoundSelector(part)
			for _, sp := range subParts {
				if strings.HasPrefix(sp, "#") {
					spec.A++
				} else if strings.HasPrefix(sp, ".") {
					spec.B++
				} else if sp != "*" && sp != "" {
					spec.C++
				}
			}
		}
	}
	return spec
}

func tokenizeCompoundSelector(s string) []string {
	var res []string
	var cur strings.Builder
	for _, r := range s {
		if r == '#' || r == '.' {
			if cur.Len() > 0 {
				res = append(res, cur.String())
				cur.Reset()
			}
		}
		cur.WriteRune(r)
	}
	if cur.Len() > 0 {
		res = append(res, cur.String())
	}
	return res
}

// ComputeStyles performs CSS cascade on an entire DOM tree:
// 1. Collects all <style> blocks.
// 2. Applies rules by specificity and document order.
// 3. Applies inline style="..." overrides (highest specificity).
// 4. Inherits inheritable properties (color, font-size) from parent to child.
func ComputeStyles(root *Node) {
	if root == nil {
		return
	}

	// 1. Collect all stylesheets from <style> tags
	var allRules []CSSRule
	root.Walk(func(n *Node) bool {
		if n.Type == NodeElement && n.Tag == "style" {
			cssText := n.TextContent()
			if cssText == "" && len(n.Children) > 0 {
				for _, c := range n.Children {
					if c.Type == NodeText {
						cssText += c.Value
					}
				}
			}
			sheet := ParseCSS(cssText)
			allRules = append(allRules, sheet.Rules...)
		}
		return true
	})

	// 2. Walk and compute style for each node
	computeNodeStyleRecursive(root, allRules, nil)
}

func computeNodeStyleRecursive(node *Node, rules []CSSRule, parentStyle *ComputedStyle) {
	if node == nil {
		return
	}

	if node.Type == NodeElement {
		style := DefaultStyle(node.Tag)

		// Inherit properties from parent if not root
		if parentStyle != nil {
			style.Color = parentStyle.Color
			style.FontSize = parentStyle.FontSize
		}

		// Find matching stylesheet rules
		var matched []CSSRule
		for _, r := range rules {
			if nodeMatchesSelector(node, r.Selector) {
				matched = append(matched, r)
			}
		}

		// Sort matched rules by Specificity, then Order
		sort.SliceStable(matched, func(i, j int) bool {
			if matched[i].Specificity.Less(matched[j].Specificity) {
				return true
			}
			if matched[j].Specificity.Less(matched[i].Specificity) {
				return false
			}
			return matched[i].Order < matched[j].Order
		})

		// Apply matched rules declarations in order
		for _, r := range matched {
			for _, decl := range r.Declarations {
				applyDeclaration(&style, decl.Property, decl.Value)
			}
		}

		// Inline style attribute has highest priority
		if inline := node.Attribute("style"); inline != "" {
			inlineDecls := parseDeclarations(inline)
			for _, decl := range inlineDecls {
				applyDeclaration(&style, decl.Property, decl.Value)
			}
		}

		node.Computed = &style

		// Recurse children with current style as parentStyle
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			computeNodeStyleRecursive(c, rules, &style)
		}
	} else {
		// Non-element inherits parent style
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			computeNodeStyleRecursive(c, rules, parentStyle)
		}
	}
}

func nodeMatchesSelector(n *Node, selector string) bool {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return false
	}
	parts := strings.Fields(selector)
	if len(parts) == 0 {
		return false
	}
	lastIdx := len(parts) - 1
	if !n.matchesSimpleSelector(parts[lastIdx]) {
		return false
	}
	if lastIdx == 0 {
		return true
	}
	// Verify ancestor chain for previous parts
	currAncestor := n.Parent
	for pIdx := lastIdx - 1; pIdx >= 0; pIdx-- {
		found := false
		for currAncestor != nil {
			if currAncestor.matchesSimpleSelector(parts[pIdx]) {
				found = true
				currAncestor = currAncestor.Parent
				break
			}
			currAncestor = currAncestor.Parent
		}
		if !found {
			return false
		}
	}
	return true
}

func applyDeclaration(s *ComputedStyle, prop, val string) {
	val = strings.TrimSpace(val)
	switch prop {
	case "color":
		if col := parseCSSColor(val); col != nil {
			s.Color = *col
		}
	case "background-color", "background":
		s.BackgroundColor = parseCSSColor(val)
	case "font-size":
		if sz, ok := parseFontSize(val); ok {
			s.FontSize = sz
		}
	case "font-weight":
		if strings.Contains(val, "bold") || val == "700" || val == "800" || val == "900" {
			s.IsBold = true
		} else if val == "normal" || val == "400" {
			s.IsBold = false
		}
	case "font-style":
		if strings.Contains(val, "italic") {
			s.IsItalic = true
		} else if val == "normal" {
			s.IsItalic = false
		}
	case "display":
		switch strings.ToLower(val) {
		case "none":
			s.Display = DisplayNone
		case "inline":
			s.Display = DisplayInline
		case "block":
			s.Display = DisplayBlock
		}
	case "width":
		if px := parsePixelValue(val); px >= 0 {
			s.Width = px
		}
	case "height":
		if px := parsePixelValue(val); px >= 0 {
			s.Height = px
		}
	case "margin":
		v := parsePixelValue(val)
		if v >= 0 {
			s.MarginTop = v
			s.MarginBottom = v
			s.MarginLeft = v
			s.MarginRight = v
		}
	case "margin-top":
		if v := parsePixelValue(val); v >= 0 {
			s.MarginTop = v
		}
	case "margin-bottom":
		if v := parsePixelValue(val); v >= 0 {
			s.MarginBottom = v
		}
	case "margin-left":
		if v := parsePixelValue(val); v >= 0 {
			s.MarginLeft = v
		}
	case "margin-right":
		if v := parsePixelValue(val); v >= 0 {
			s.MarginRight = v
		}
	case "padding":
		v := parsePixelValue(val)
		if v >= 0 {
			s.PaddingTop = v
			s.PaddingBottom = v
			s.PaddingLeft = v
			s.PaddingRight = v
		}
	case "padding-top":
		if v := parsePixelValue(val); v >= 0 {
			s.PaddingTop = v
		}
	case "padding-bottom":
		if v := parsePixelValue(val); v >= 0 {
			s.PaddingBottom = v
		}
	case "padding-left":
		if v := parsePixelValue(val); v >= 0 {
			s.PaddingLeft = v
		}
	case "padding-right":
		if v := parsePixelValue(val); v >= 0 {
			s.PaddingRight = v
		}
	}
}

func parsePixelValue(val string) float32 {
	val = strings.TrimSpace(strings.ToLower(val))
	val = strings.TrimSuffix(val, "px")
	if v, err := strconv.ParseFloat(val, 32); err == nil {
		return float32(v)
	}
	return -1
}
