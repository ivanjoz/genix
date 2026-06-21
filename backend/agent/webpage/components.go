package webpage

import (
	_ "embed"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// BUILDER_COMPONENTS.md documents the custom HTML component tags the builder
// understands (ProductGrid, ImageEffect, Slider…). It is embedded and split per
// component so the builder agent can fetch one component's reference on demand
// via the get_component_docs tool — instead of carrying the whole doc in every
// system prompt.
//
//go:embed BUILDER_COMPONENTS.md
var builderComponentsDoc string

// componentDoc is one parsed "## <Name>" section of the reference.
type componentDoc struct {
	Name    string // canonical heading, e.g. "ProductGrid"
	Summary string // one-line description (first prose sentence) for the prompt listing
	Body    string // the full markdown section (heading + content), dividers stripped
}

var (
	componentsOnce  sync.Once
	componentsByKey map[string]componentDoc // keyed by normalized name (see normalizeComponentName)
	componentNames  []string                // canonical names, sorted
)

// nonAlphanumeric strips everything but [a-z0-9] so "ProductGrid",
// "product_grid" and "product grid" all normalize to "productgrid".
var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)

// normalizeComponentName makes lookups tolerant of case, spaces, and
// separators, so the agent can search loosely and still hit the right doc.
func normalizeComponentName(name string) string {
	return nonAlphanumeric.ReplaceAllString(strings.ToLower(strings.TrimSpace(name)), "")
}

// parseComponents splits the embedded reference into per-component sections.
// A section starts at a "## <Name>" heading and runs until the next one; the
// "---" dividers between sections are dropped. Text before the first heading
// (the general rules preamble) is ignored here — it isn't component-specific.
func parseComponents() {
	componentsByKey = map[string]componentDoc{}
	var current *componentDoc
	var body []string

	flush := func() {
		if current == nil {
			return
		}
		current.Body = strings.TrimSpace(strings.Join(body, "\n"))
		current.Summary = summaryFromBody(body)
		componentsByKey[normalizeComponentName(current.Name)] = *current
		componentNames = append(componentNames, current.Name)
	}

	for _, line := range strings.Split(builderComponentsDoc, "\n") {
		if name, ok := strings.CutPrefix(line, "## "); ok {
			flush()
			current = &componentDoc{Name: strings.TrimSpace(name)}
			body = []string{line}
			continue
		}
		if current == nil {
			continue // preamble before the first component
		}
		if strings.TrimSpace(line) == "---" {
			continue // section divider — not part of any component body
		}
		body = append(body, line)
	}
	flush()
	sort.Strings(componentNames)
}

// summaryFromBody derives a one-line description for the prompt listing: it
// joins the FIRST prose paragraph (consecutive non-empty prose lines after the
// heading — markdown hard-wraps a sentence across several) into one string,
// then returns just the first sentence. Joining before cutting is what keeps
// the sentence complete regardless of where the line wrap falls. Tables, code
// fences, blockquotes and headings are not prose and end the paragraph.
func summaryFromBody(body []string) string {
	var paragraph []string
	for _, line := range body {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "##") {
			continue // the section heading line itself
		}
		isProse := trimmed != "" && trimmed[0] != '|' && trimmed[0] != '#' && trimmed[0] != '>' && trimmed[0] != '`'
		if isProse {
			paragraph = append(paragraph, trimmed)
			continue
		}
		if len(paragraph) > 0 {
			break // a blank or non-prose line ends the first paragraph
		}
	}
	text := strings.Join(paragraph, " ")
	text = strings.ReplaceAll(text, "**", "")
	text = strings.ReplaceAll(text, "`", "")
	// First sentence: cut at the first period that ends a word (". ").
	if i := strings.Index(text, ". "); i >= 0 {
		return text[:i+1]
	}
	return text
}

// componentDocByName returns the doc for a component, matched loosely by name.
func componentDocByName(name string) (componentDoc, bool) {
	componentsOnce.Do(parseComponents)
	doc, ok := componentsByKey[normalizeComponentName(name)]
	return doc, ok
}

// componentNamesList returns the canonical component names, sorted. Used in
// not-found errors so the agent can pick a valid name.
func componentNamesList() []string {
	componentsOnce.Do(parseComponents)
	return componentNames
}

// componentDocsList returns every component doc (name + summary + body), sorted
// by name. Used to print the "Name: description" listing in the system prompt.
func componentDocsList() []componentDoc {
	componentsOnce.Do(parseComponents)
	docs := make([]componentDoc, 0, len(componentNames))
	for _, name := range componentNames {
		docs = append(docs, componentsByKey[normalizeComponentName(name)])
	}
	return docs
}
