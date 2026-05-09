package agent

import (
	"strings"
	"testing"
)

func TestParsePageHTML_StripsAttributesExceptAriaAndRole(t *testing.T) {
	in := `<div class="x" id="y" role="button" aria-label="ok" onclick="boom()">hi</div>`
	out, err := ParsePageHTML(in, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `role="button"`) || !strings.Contains(out, `aria-label="ok"`) {
		t.Fatalf("expected role + aria-label preserved, got:\n%s", out)
	}
	for _, banned := range []string{"class=", "id=", "onclick="} {
		if strings.Contains(out, banned) {
			t.Fatalf("expected %q stripped, got:\n%s", banned, out)
		}
	}
}

func TestParsePageHTML_DropsEmptyChains(t *testing.T) {
	in := `<section><div><div><div></div></div></div></section>`
	out, err := ParsePageHTML(in, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out) != "" {
		t.Fatalf("expected empty output, got:\n%s", out)
	}
}

func TestParsePageHTML_CollapsesWrapperChain(t *testing.T) {
	in := `<div><div><div><div>hello</div></div></div></div>`
	out, err := ParsePageHTML(in, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Expect at most one wrapper between the outer text-bearing context
	// (the implicit body) and the innermost element with text.
	if strings.Count(out, "<div>") > 2 {
		t.Fatalf("expected at most 2 <div> wrappers, got:\n%s", out)
	}
	if !strings.Contains(out, "hello") {
		t.Fatalf("expected text preserved, got:\n%s", out)
	}
}

func TestParsePageHTML_KeepsOneWrapperBetweenTextLayers(t *testing.T) {
	in := `<div>hello<div><div><div>world</div></div></div></div>`
	out, err := ParsePageHTML(in, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "hello") || !strings.Contains(out, "world") {
		t.Fatalf("expected both text nodes preserved, got:\n%s", out)
	}
	// Between hello (in outer div) and world (innermost) only one wrapper allowed.
	// Outer div + 1 wrapper + innermost div = 3 opening div tags total.
	if strings.Count(out, "<div") != 3 {
		t.Fatalf("expected exactly 3 <div tags, got %d:\n%s", strings.Count(out, "<div"), out)
	}
}

func TestParsePageHTML_DropsScriptsAndComments(t *testing.T) {
	in := `<div>keep<script>evil()</script><!-- gone --><style>.x{}</style></div>`
	out, err := ParsePageHTML(in, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, banned := range []string{"script", "style", "evil", "gone"} {
		if strings.Contains(out, banned) {
			t.Fatalf("expected %q stripped, got:\n%s", banned, out)
		}
	}
	if !strings.Contains(out, "keep") {
		t.Fatalf("expected text preserved, got:\n%s", out)
	}
}

func TestParsePageHTML_IndentsNestedContent(t *testing.T) {
	in := `<section><h1>Title</h1><p>Body text</p></section>`
	out, err := ParsePageHTML(in, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "\n  <h1>") {
		t.Fatalf("expected indented children, got:\n%s", out)
	}
}
