package webpage

import (
	"encoding/json"
	"strings"
	"testing"

	"app/agent/llm"
)

func TestExtractSourceHTMLAndSplit(t *testing.T) {
	ctx := "CURRENT COLOR PALETTE: 1=#fff\n\n--- HTML of the page's current sections ---\n" +
		"=== SECTION 1 ===\n<section><h1>One</h1></section>\n" +
		"=== SECTION 2 ===\n<section><img src=\"/two.jpg\"/></section>"

	html := extractSourceHTML(ctx)
	if !strings.HasPrefix(html, "=== SECTION 1 ===") {
		t.Fatalf("extractSourceHTML = %q, want it to start at the first section marker", html)
	}

	sections := splitSourceSections(html)
	if len(sections) != 2 {
		t.Fatalf("split = %d sections, want 2", len(sections))
	}
	if len(sections[1].Texts) != 1 || sections[1].Texts[0] != "One" {
		t.Fatalf("section 1 texts = %#v, want [One]", sections[1].Texts)
	}
	if len(sections[2].Images) != 1 || sections[2].Images[0] != "/two.jpg" {
		t.Fatalf("section 2 images = %#v, want [/two.jpg]", sections[2].Images)
	}
}

// applyCall builds a synthetic apply_sections tool call for the verifier.
func applyCall(sections []SectionEdit) llm.ToolCall {
	args, _ := json.Marshal(map[string]any{"sections": sections})
	return llm.ToolCall{Function: llm.ToolCallFunction{Name: ApplySectionsToolName, Arguments: string(args)}}
}

func TestVerifyPageContentPreservesUntargetedSections(t *testing.T) {
	src := splitSourceSections(
		"=== SECTION 1 ===\n<section><h1>Hero</h1></section>\n" +
			"=== SECTION 2 ===\n<section><p>About us</p></section>\n" +
			"=== SECTION 3 ===\n<section><p>Footer</p></section>")

	turn := &builderTurn{
		modeID:         ModeBuildPage,
		sourceSections: src,
		pageOp:         pageClassification{Operation: "partial", ModifySectionIDs: []int{1}},
	}

	// Legal: section 1 modified, 2 & 3 returned verbatim.
	ok := []SectionEdit{
		{SourceID: 1, HTML: "<section><h1>New Hero</h1></section>"},
		{SourceID: 2, HTML: "<section><p>About us</p></section>"},
		{SourceID: 3, HTML: "<section><p>Footer</p></section>"},
	}
	if v := turn.verifyContent(applyCall(ok)); len(v) != 0 {
		t.Fatalf("expected no violations, got %v", v)
	}

	// Illegal: section 2 (not targeted) had its text changed, and 3 is dropped.
	bad := []SectionEdit{
		{SourceID: 1, HTML: "<section><h1>New Hero</h1></section>"},
		{SourceID: 2, HTML: "<section><p>Changed copy</p></section>"},
	}
	v := turn.verifyContent(applyCall(bad))
	if len(v) < 2 {
		t.Fatalf("expected violations for changed section 2 + missing section 3, got %v", v)
	}
}

func TestVerifyPageContentRejectsUnwantedNewSection(t *testing.T) {
	src := splitSourceSections("=== SECTION 1 ===\n<section><h1>Hero</h1></section>")
	turn := &builderTurn{
		modeID:         ModeBuildPage,
		sourceSections: src,
		pageOp:         pageClassification{Operation: "partial", ModifySectionIDs: []int{1}, AddSections: false},
	}
	withNew := []SectionEdit{
		{SourceID: 1, HTML: "<section><h1>Hero</h1></section>"},
		{SourceID: 0, HTML: "<section><p>Surprise</p></section>"},
	}
	if v := turn.verifyContent(applyCall(withNew)); len(v) == 0 {
		t.Fatalf("expected a violation for an unrequested new section")
	}

	turn.pageOp.AddSections = true
	if v := turn.verifyContent(applyCall(withNew)); len(v) != 0 {
		t.Fatalf("addSections=true should permit a new section, got %v", v)
	}
}

func TestVerifyPageRewriteSkipsChecks(t *testing.T) {
	turn := &builderTurn{
		modeID:         ModeBuildPage,
		sourceSections: splitSourceSections("=== SECTION 1 ===\n<section><h1>Hero</h1></section>"),
		pageOp:         pageClassification{Operation: "rewrite"},
	}
	if v := turn.verifyContent(applyCall([]SectionEdit{{SourceID: 0, HTML: "<section>totally new</section>"}})); len(v) != 0 {
		t.Fatalf("rewrite should skip verification, got %v", v)
	}
}
