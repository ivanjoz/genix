package webpage

import (
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/html"
)

const htmlTextNodeTag = "#text"

// HTMLASTNode is a small backend-only view of the section HTML. It exists so
// the agent loop can compare semantic content before accepting rewritten HTML.
type HTMLASTNode struct {
	TagName    string
	Attributes map[string]string
	Text       string
	Children   []*HTMLASTNode
}

var htmlVoidTags = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true,
	"hr": true, "img": true, "input": true, "link": true, "meta": true,
	"source": true, "track": true, "wbr": true,
}

// ParseHTMLToAST parses a section HTML fragment into a structural AST. It uses
// the tokenizer instead of html.Parse so custom self-closing components such as
// <ImageEffect /> do not accidentally wrap following sibling nodes.
func ParseHTMLToAST(htmlText string) ([]*HTMLASTNode, error) {
	tokenizer := html.NewTokenizer(strings.NewReader(htmlText))
	root := &HTMLASTNode{TagName: "root"}
	stack := []*HTMLASTNode{root}

	for {
		tokenType := tokenizer.Next()
		switch tokenType {
		case html.ErrorToken:
			if err := tokenizer.Err(); err != nil && err != io.EOF {
				return nil, err
			}
			return root.Children, nil
		case html.TextToken:
			appendHTMLTextNode(stack[len(stack)-1], string(tokenizer.Text()))
		case html.StartTagToken, html.SelfClosingTagToken:
			rawToken := string(tokenizer.Raw())
			token := tokenizer.Token()
			node := &HTMLASTNode{
				TagName:    originalHTMLTagName(rawToken, token.Data),
				Attributes: htmlTokenAttributes(token),
			}
			parent := stack[len(stack)-1]
			parent.Children = append(parent.Children, node)
			if tokenType == html.StartTagToken && !htmlVoidTags[strings.ToLower(token.Data)] {
				stack = append(stack, node)
			}
		case html.EndTagToken:
			closeHTMLNode(&stack, tokenizer.Token().Data)
		}
	}
}

func appendHTMLTextNode(parent *HTMLASTNode, rawText string) {
	text := normalizeHTMLText(rawText)
	if text == "" {
		return
	}
	parent.Children = append(parent.Children, &HTMLASTNode{
		TagName: htmlTextNodeTag,
		Text:    text,
	})
}

func htmlTokenAttributes(token html.Token) map[string]string {
	if len(token.Attr) == 0 {
		return nil
	}
	attrs := make(map[string]string, len(token.Attr))
	for _, attr := range token.Attr {
		attrs[attr.Key] = attr.Val
	}
	return attrs
}

func closeHTMLNode(stack *[]*HTMLASTNode, tagName string) {
	lowerTag := strings.ToLower(tagName)
	for i := len(*stack) - 1; i > 0; i-- {
		if strings.ToLower((*stack)[i].TagName) == lowerTag {
			*stack = (*stack)[:i]
			return
		}
	}
}

func normalizeHTMLText(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

func originalHTMLTagName(rawToken string, fallback string) string {
	trimmedToken := strings.TrimSpace(rawToken)
	if !strings.HasPrefix(trimmedToken, "<") {
		return fallback
	}
	tagStart := 1
	if tagStart < len(trimmedToken) && trimmedToken[tagStart] == '/' {
		tagStart++
	}
	for tagStart < len(trimmedToken) && isHTMLNameBoundary(trimmedToken[tagStart]) {
		tagStart++
	}
	tagEnd := tagStart
	for tagEnd < len(trimmedToken) && !isHTMLNameBoundary(trimmedToken[tagEnd]) {
		tagEnd++
	}
	if tagEnd <= tagStart {
		return fallback
	}
	return trimmedToken[tagStart:tagEnd]
}

func isHTMLNameBoundary(char byte) bool {
	return char == ' ' || char == '\t' || char == '\n' || char == '\r' || char == '/' || char == '>'
}

// SectionContent is the deterministic content fingerprint of one section: the
// three kinds of content the user cares about preserving. Each slice is a
// multiset (order-independent, duplicates kept) so the verifier can assert a
// dimension was preserved without depending on where the agent placed it.
type SectionContent struct {
	Texts  []string // normalized text-node strings
	Images []string // src= URLs (on <img>, <ImageEffect>, or any node)
	Icons  []string // svg= ids on <Icon> nodes
}

// ExtractSectionContent walks the AST and collects the text/image/icon content.
// Identity mirrors the frontend's extractAssets: an image is any node carrying a
// src attribute; an icon is an <Icon> node's svg attribute.
func ExtractSectionContent(nodes []*HTMLASTNode) SectionContent {
	var content SectionContent
	var walk func(n *HTMLASTNode)
	walk = func(n *HTMLASTNode) {
		if n.TagName == htmlTextNodeTag {
			if n.Text != "" {
				content.Texts = append(content.Texts, n.Text)
			}
			return
		}
		if src := n.Attributes["src"]; src != "" {
			content.Images = append(content.Images, src)
		}
		if strings.EqualFold(n.TagName, "Icon") {
			if svg := n.Attributes["svg"]; svg != "" {
				content.Icons = append(content.Icons, svg)
			}
		}
		for _, child := range n.Children {
			walk(child)
		}
	}
	for _, n := range nodes {
		walk(n)
	}
	return content
}

// Content-preservation policy values for one dimension. keep/add are verified;
// modify/replace are intentionally NOT (the user asked for that change).
const (
	PolicyKeep    = "keep"    // multiset must be identical (no add/remove/change)
	PolicyAdd     = "add"     // old must be a subset of new (additions allowed)
	PolicyModify  = "modify"  // not verified
	PolicyReplace = "replace" // not verified
)

// ContentPolicy is the per-dimension verdict for a single section (mode 3, or a
// modified section in mode 2). Each field is one of the Policy* constants.
type ContentPolicy struct {
	Text   string `json:"text"`
	Images string `json:"images"`
	Icons  string `json:"icons"`
}

// VerifySectionContent compares old vs new content under the policy and returns
// a human-readable violation per dimension (empty slice = the edit is allowed).
// The messages name the exact offending items so they can be fed back to the
// model verbatim.
func VerifySectionContent(old, new SectionContent, p ContentPolicy) []string {
	var violations []string
	check := func(dimension, policy string, oldItems, newItems []string) {
		switch policy {
		case PolicyKeep:
			if removed := multisetDiff(oldItems, newItems); len(removed) > 0 {
				violations = append(violations, fmt.Sprintf(
					"%s policy is KEEP but you removed or changed: %s — restore them exactly.",
					dimension, strings.Join(removed, " | ")))
			}
			if added := multisetDiff(newItems, oldItems); len(added) > 0 {
				violations = append(violations, fmt.Sprintf(
					"%s policy is KEEP but you added: %s — the user did not ask to change %s.",
					dimension, strings.Join(added, " | "), dimension))
			}
		case PolicyAdd:
			if removed := multisetDiff(oldItems, newItems); len(removed) > 0 {
				violations = append(violations, fmt.Sprintf(
					"%s policy is ADD (additions only) but you removed or changed: %s — restore them.",
					dimension, strings.Join(removed, " | ")))
			}
		}
	}
	check("Text", p.Text, old.Texts, new.Texts)
	check("Images", p.Images, old.Images, new.Images)
	check("Icons", p.Icons, old.Icons, new.Icons)
	return violations
}

// multisetDiff returns the items present in want but missing from have, honoring
// multiplicity (an item needed twice but present once is still reported once).
func multisetDiff(want, have []string) []string {
	counts := make(map[string]int, len(have))
	for _, item := range have {
		counts[item]++
	}
	var missing []string
	for _, item := range want {
		if counts[item] > 0 {
			counts[item]--
			continue
		}
		missing = append(missing, item)
	}
	return missing
}
