package webpage

import (
	"strings"
	"testing"
)

// TestParseComponents checks the embedded reference splits into the expected
// component sections and that loose name matching works.
func TestParseComponents(t *testing.T) {
	names := componentNamesList()
	want := []string{
		"CategoryDescription", "ImageEffect", "ProductCard",
		"ProductGrid", "ProductsByCategory", "Slider", "TabbedLayer",
	}
	if len(names) != len(want) {
		t.Fatalf("got %d components %v, want %d %v", len(names), names, len(want), want)
	}
	for i, n := range want {
		if names[i] != n {
			t.Fatalf("component[%d] = %q, want %q (full: %v)", i, names[i], n, names)
		}
	}

	// Loose lookups all resolve to the same canonical doc.
	for _, alias := range []string{"ProductGrid", "productgrid", "product_grid", "product grid", "PRODUCT-GRID"} {
		doc, ok := componentDocByName(alias)
		if !ok {
			t.Fatalf("alias %q not found", alias)
		}
		if doc.Name != "ProductGrid" {
			t.Fatalf("alias %q resolved to %q, want ProductGrid", alias, doc.Name)
		}
	}

	// A real section carries its heading + the attribute table.
	doc, _ := componentDocByName("ImageEffect")
	if !strings.Contains(doc.Body, "## ImageEffect") || !strings.Contains(doc.Body, "duotone") {
		t.Fatalf("ImageEffect body missing expected content: %q", doc.Body)
	}

	// Every component gets a complete one-line summary for the prompt listing.
	for _, d := range componentDocsList() {
		if d.Summary == "" {
			t.Fatalf("component %q has no summary", d.Name)
		}
		if strings.Contains(d.Summary, "\n") {
			t.Fatalf("component %q summary is not one line: %q", d.Name, d.Summary)
		}
		// A complete first sentence ends with a period — never mid-sentence.
		if !strings.HasSuffix(d.Summary, ".") {
			t.Fatalf("component %q summary is not a complete sentence: %q", d.Name, d.Summary)
		}
		t.Logf("%s: %s", d.Name, d.Summary)
	}

	// Regression for the wrapped-sentence bug: a component whose first sentence
	// wraps across markdown hard-wraps must still get the full sentence, not the
	// first wrapped line. ImageEffect's first sentence wraps; assert it stays whole.
	img, _ := componentDocByName("ImageEffect")
	if !strings.HasSuffix(img.Summary, ".") || strings.Contains(img.Summary, "\n") {
		t.Fatalf("ImageEffect summary not a complete single sentence: %q", img.Summary)
	}

	// The agent's Iconify route is disabled: Icon must not be a documented component.
	if _, ok := componentDocByName("Icon"); ok {
		t.Fatal("Icon should not be a documented component (agent uses generate_svg only)")
	}

	// Misses report the available list rather than panicking.
	if _, ok := componentDocByName("DoesNotExist"); ok {
		t.Fatal("expected miss for unknown component")
	}
}
