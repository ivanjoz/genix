package webpage

import (
	"fmt"
	"strings"
)

// staticLintSections parses each proposed section's HTML and returns a list of
// deterministic structural observations — issues that are provably true from the
// markup alone (not aesthetic judgment calls). They are surfaced to the
// aesthetic-review subagent under a "STATIC LINTER OBSERVATIONS" heading so the
// critic confirms them and folds them into its verdict.
//
// The checks target ImageEffect's render contract, where misuse is invisible in
// the markup but breaks the preview:
//   - rich mode (effect/layout/overlay children, non-fill) renders the photo as an
//     absolute layer, so the component needs an explicit height (aspectRatio or a
//     min-h-*/h-* on its OWN class) or it collapses to 0px;
//   - fill mode needs its immediate parent to carry real height;
//   - in any rich/fill mode `class` styles the wrapper box, so object-fit
//     utilities never reach the <img> (the `fit` attribute must be used instead).
//
// Best-effort: unparseable HTML yields no observations (fails open).
func staticLintSections(sections []SectionEdit) []string {
	var obs []string
	for i, s := range sections {
		nodes, err := ParseHTMLToAST(s.HTML)
		if err != nil {
			continue
		}
		label := fmt.Sprintf("Section %d", i+1)
		for _, n := range nodes {
			lintNode(n, nil, label, &obs)
		}
	}
	return obs
}

// lintNode walks the AST depth-first, carrying each node's parent so checks that
// depend on context (fill mode's parent height) can inspect it.
func lintNode(n, parent *HTMLASTNode, label string, obs *[]string) {
	if isImageEffectTag(n.TagName) {
		*obs = append(*obs, lintImageEffect(n, parent, label)...)
	}
	for _, c := range n.Children {
		lintNode(c, n, label, obs)
	}
}

func lintImageEffect(n, parent *HTMLASTNode, label string) []string {
	attr := func(k string) string { return strings.TrimSpace(n.Attributes[k]) }
	_, hasFill := n.Attributes["fill"] // boolean attribute: presence ⇒ fill mode
	effect := attr("effect")
	layout := attr("layout")
	aspect := attr("aspectratio") // tokenizer lowercases attribute keys
	classTokens := strings.Fields(attr("class"))
	// Whitespace-only text is dropped at parse time, so any child is real overlay content.
	hasOverlay := len(n.Children) > 0
	rich := effect != "" || layout != "" || hasOverlay

	var out []string

	// Check A — rich (non-fill) ImageEffect with no intrinsic height source.
	if rich && !hasFill {
		hasAspect := aspect != "" && aspect != "auto"
		if !hasAspect && !hasHeightClass(classTokens) {
			out = append(out, fmt.Sprintf(
				"%s: <ImageEffect> is in rich mode (%s) but has NO height source — no aspectRatio and no min-h-*/h-* on its own class. Its photo is an absolute layer, so the box collapses to 0px and the image is invisible (the editor still shows the image control). Fix: add aspectRatio=\"W/H\" or a min-h-* on the ImageEffect itself (a min-h on the PARENT does not count).",
				label, richReason(effect, layout, hasOverlay)))
		}
	}

	// Check B — fill ImageEffect whose immediate parent has no real height.
	if hasFill && parent != nil && !hasHeightClass(strings.Fields(strings.TrimSpace(parent.Attributes["class"]))) {
		out = append(out, fmt.Sprintf(
			"%s: <ImageEffect> uses fill mode (absolute background) but its immediate parent <%s> has no explicit height (min-h-*/h-*), so the background has nothing to fill and the image is invisible. Fix: give the parent a real height (and position:relative), or drop fill and size the ImageEffect itself.",
			label, parent.TagName))
	}

	// Check C — object-fit utility on a rich/fill ImageEffect; class styles the box, not the <img>.
	if rich || hasFill {
		if obj := firstObjectFitClass(classTokens); obj != "" {
			out = append(out, fmt.Sprintf(
				"%s: <ImageEffect> has class \"%s\", but in rich/fill mode `class` styles the wrapper box, NOT the <img>, so object-fit utilities never reach the image. Fix: set image fit via the fit attribute (fit=\"cover\" | \"contain\") instead of an object-* class.",
				label, obj))
		}
	}

	return out
}

func richReason(effect, layout string, hasOverlay bool) string {
	var parts []string
	if effect != "" {
		parts = append(parts, fmt.Sprintf("effect=%q", effect))
	}
	if layout != "" {
		parts = append(parts, fmt.Sprintf("layout=%q", layout))
	}
	if hasOverlay {
		parts = append(parts, "overlay children")
	}
	return strings.Join(parts, " + ")
}

// isImageEffectTag matches the component name loosely (case/space/underscore/hyphen
// insensitive), the same way the renderer's component registry resolves names.
func isImageEffectTag(tag string) bool { return normalizeComponentName(tag) == "imageeffect" }

// utilityPart strips Tailwind variant prefixes (md:, hover:, …) and returns the
// bare utility so prefix checks work on responsive/state-qualified tokens.
func utilityPart(token string) string {
	if i := strings.LastIndex(token, ":"); i >= 0 {
		return token[i+1:]
	}
	return token
}

// hasHeightClass reports whether any class token supplies a real height. h-full /
// h-auto / aspect-auto are excluded: they defer height to the parent or to nothing.
func hasHeightClass(tokens []string) bool {
	for _, t := range tokens {
		u := utilityPart(t)
		switch {
		case u == "h-full", u == "h-auto", u == "aspect-auto":
			continue
		case strings.HasPrefix(u, "min-h-"),
			strings.HasPrefix(u, "h-"),
			strings.HasPrefix(u, "aspect-"):
			return true
		}
	}
	return false
}

var objectFitClasses = map[string]bool{
	"object-cover": true, "object-contain": true, "object-fill": true,
	"object-none": true, "object-scale-down": true,
}

func firstObjectFitClass(tokens []string) string {
	for _, t := range tokens {
		if u := utilityPart(t); objectFitClasses[u] {
			return u
		}
	}
	return ""
}
