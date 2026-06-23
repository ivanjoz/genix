package webpage

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"app/agent/llm"
)

// sourceHTMLMarker is the line the frontend writes just before the section HTML
// in the per-turn context (see buildAgentContext). Everything after it is the
// raw HTML the agent edits — and the ground truth the content gate compares to.
const sourceHTMLMarker = "--- HTML of"

// sectionMarkerPrefix labels each section in build-page context: "=== SECTION 1 ===".
// The classifier references these numbers and the agent echoes them as SourceID.
const sectionMarkerPrefix = "=== SECTION "

// allKeepPolicy locks every content dimension — used for build-page sections the
// user did not ask to change.
var allKeepPolicy = ContentPolicy{Text: PolicyKeep, Images: PolicyKeep, Icons: PolicyKeep}

// classifyTurn runs the mode-specific classifier, stores the preservation state
// on the turn, and returns the constraint block to append to the system prompt.
// abort=true means the request was uninterpretable (caller replies and stops).
func (t *builderTurn) classifyTurn(ctx context.Context, userText, pageContext string) (constraints string, abort bool) {
	sourceHTML := extractSourceHTML(pageContext)
	if t.modeID == ModeBuildPage {
		verdict := t.classifyPage(ctx, userText, sourceHTML)
		if !verdict.Relevant {
			return "", true
		}
		t.pageOp = verdict
		t.sourceSections = splitSourceSections(sourceHTML)
		return pageConstraintBlock(verdict), false
	}
	// ModeEditSection.
	verdict, policy := t.classifyEdit(ctx, userText, sourceHTML)
	if !verdict.Relevant {
		return "", true
	}
	t.policy = policy
	if nodes, err := ParseHTMLToAST(sourceHTML); err == nil {
		t.sourceContent = ExtractSectionContent(nodes)
	}
	return editConstraintBlock(policy), false
}

// verifyContent parses the apply_sections payload and checks it against the
// turn's preservation policy. An empty result means the edit is allowed.
func (t *builderTurn) verifyContent(call llm.ToolCall) []string {
	var args struct {
		Sections []SectionEdit `json:"sections"`
	}
	if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
		return nil // malformed args are handled (and errored) by applySections
	}
	if t.modeID == ModeBuildPage {
		return t.verifyPageContent(args.Sections)
	}
	if len(args.Sections) == 0 {
		return nil
	}
	nodes, err := ParseHTMLToAST(args.Sections[0].HTML)
	if err != nil {
		return nil // unparseable HTML surfaces elsewhere; don't block on it here
	}
	return VerifySectionContent(t.sourceContent, ExtractSectionContent(nodes), t.policy)
}

// verifyPageContent enforces the build-page section plan: sections the request
// did not target must come back unchanged, removed sections must be gone, and
// new sections are allowed only when the user asked for them.
func (t *builderTurn) verifyPageContent(sections []SectionEdit) []string {
	if strings.EqualFold(t.pageOp.Operation, "rewrite") {
		return nil // whole-page redo — nothing to preserve
	}
	modify := intSet(t.pageOp.ModifySectionIDs)
	remove := intSet(t.pageOp.RemoveSectionIDs)

	var violations []string
	returned := map[int]bool{}
	for _, s := range sections {
		if s.SourceID == 0 { // a brand-new section
			if !t.pageOp.AddSections {
				violations = append(violations, "you returned a new section but the request did not ask to add sections — remove it.")
			}
			continue
		}
		source, known := t.sourceSections[s.SourceID]
		if !known {
			violations = append(violations, fmt.Sprintf("section %d does not exist — use the exact \"=== SECTION N ===\" numbers, or 0 for a new section.", s.SourceID))
			continue
		}
		if returned[s.SourceID] {
			violations = append(violations, fmt.Sprintf("section %d was returned more than once.", s.SourceID))
			continue
		}
		returned[s.SourceID] = true
		if remove[s.SourceID] {
			violations = append(violations, fmt.Sprintf("section %d should be removed — omit it from your output, do not return it.", s.SourceID))
			continue
		}
		if modify[s.SourceID] {
			continue // the user asked to change this one — unverified
		}
		// "keep" section: must come back with identical content.
		nodes, err := ParseHTMLToAST(s.HTML)
		if err != nil {
			continue
		}
		for _, v := range VerifySectionContent(source, ExtractSectionContent(nodes), allKeepPolicy) {
			violations = append(violations, fmt.Sprintf("section %d: %s", s.SourceID, v))
		}
	}
	// Every "keep" section must still be present (dropping it loses content).
	for id := range t.sourceSections {
		if modify[id] || remove[id] || returned[id] {
			continue
		}
		violations = append(violations, fmt.Sprintf("section %d is missing from your output — return it unchanged.", id))
	}
	return violations
}

// contentViolationFeedback frames the violations as a tool result the model can
// act on: fix ONLY these, keep everything else, then re-call apply_sections.
func contentViolationFeedback(violations []string) string {
	return toolJSON(map[string]any{
		"approved":     false,
		"contentCheck": violations,
		"instruction":  "Your edit changed content the user did not ask to change. Fix ONLY the issues listed in contentCheck and re-call apply_sections. Restore every flagged text, image (src) and icon (<Icon svg=…>) EXACTLY; do not touch anything else.",
	})
}

// extractSourceHTML returns the raw section HTML embedded in the per-turn
// context, i.e. everything after the sourceHTMLMarker line. Falls back to the
// whole context when the marker is absent (older framing / direct calls).
func extractSourceHTML(pageContext string) string {
	idx := strings.Index(pageContext, sourceHTMLMarker)
	if idx < 0 {
		return pageContext
	}
	rest := pageContext[idx:]
	if nl := strings.IndexByte(rest, '\n'); nl >= 0 {
		return strings.TrimSpace(rest[nl+1:])
	}
	return ""
}

// splitSourceSections splits build-page source HTML on the "=== SECTION N ==="
// markers into id → content fingerprint, for per-section verification.
func splitSourceSections(html string) map[int]SectionContent {
	out := map[int]SectionContent{}
	lines := strings.Split(html, "\n")
	currentID := 0
	var body strings.Builder
	flush := func() {
		if currentID == 0 {
			return
		}
		if nodes, err := ParseHTMLToAST(body.String()); err == nil {
			out[currentID] = ExtractSectionContent(nodes)
		}
		body.Reset()
	}
	for _, line := range lines {
		if id, ok := parseSectionMarker(line); ok {
			flush()
			currentID = id
			continue
		}
		body.WriteString(line)
		body.WriteByte('\n')
	}
	flush()
	return out
}

// parseSectionMarker returns the N from a "=== SECTION N ===" line.
func parseSectionMarker(line string) (int, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, sectionMarkerPrefix) {
		return 0, false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(trimmed, sectionMarkerPrefix))
	rest = strings.TrimSpace(strings.TrimSuffix(rest, "="))
	n, err := strconv.Atoi(strings.TrimSpace(strings.TrimRight(rest, "= ")))
	if err != nil {
		return 0, false
	}
	return n, true
}

func intSet(ids []int) map[int]bool {
	set := make(map[int]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set
}

// editConstraintBlock renders the per-dimension preservation rules appended to
// the ModeEditSection system prompt. Returns "" when nothing is locked (a full
// rewrite), so the prompt stays clean.
func editConstraintBlock(p ContentPolicy) string {
	if p.Text == PolicyReplace && p.Images == PolicyReplace && p.Icons == PolicyReplace {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\nPRESERVATION CONSTRAINTS THIS TURN (a deterministic check WILL REJECT your output if you break them):\n")
	b.WriteString("  - TEXT: " + dimensionRule(p.Text, "text", "wording") + "\n")
	b.WriteString("  - IMAGES: " + dimensionRule(p.Images, "image", "src") + "\n")
	b.WriteString("  - ICONS: " + dimensionRule(p.Icons, "icon", "<Icon> glyph") + "\n")
	b.WriteString("Layout, classes and styling are always free to change — only the content above is gated.")
	return b.String()
}

// dimensionRule is the human instruction for one policy value.
func dimensionRule(policy, noun, what string) string {
	switch policy {
	case PolicyAdd:
		return fmt.Sprintf("you MAY ADD %s, but every existing %s must stay EXACTLY as-is.", noun+"s", noun)
	case PolicyModify:
		return fmt.Sprintf("you may change the %s as the user asked.", noun)
	case PolicyReplace:
		return fmt.Sprintf("you may freely change or replace the %s.", noun)
	default: // keep
		return fmt.Sprintf("KEEP every existing %s (%s) EXACTLY — do not add, remove, reword or replace any.", noun, what)
	}
}

// pageConstraintBlock renders the build-page section plan into the system prompt.
func pageConstraintBlock(v pageClassification) string {
	var b strings.Builder
	b.WriteString("\n\nPRESERVATION CONSTRAINTS THIS TURN (a deterministic check WILL REJECT your output if you break them):\n")
	if strings.EqualFold(v.Operation, "rewrite") {
		b.WriteString("  - Whole-page rewrite: you may redo every section.\n")
		b.WriteString("  - Still set sourceId on each returned section (its \"=== SECTION N ===\" number, or 0 for a new section).")
		return b.String()
	}
	b.WriteString("  - Change ONLY the sections listed below; return EVERY OTHER section UNCHANGED and verbatim.\n")
	b.WriteString("  - You may MODIFY sections: " + idListOrNone(v.ModifySectionIDs) + "\n")
	b.WriteString("  - REMOVE (omit from output) sections: " + idListOrNone(v.RemoveSectionIDs) + "\n")
	if v.AddSections {
		b.WriteString("  - You MAY add one or more new sections (set their sourceId to 0).\n")
	} else {
		b.WriteString("  - Do NOT add any new section this turn.\n")
	}
	b.WriteString("  - On EVERY returned section set sourceId to its \"=== SECTION N ===\" number (0 for a new section).")
	return b.String()
}

// idListOrNone renders an id list for the prompt, sorted for stable output.
func idListOrNone(ids []int) string {
	if len(ids) == 0 {
		return "(none)"
	}
	sorted := append([]int(nil), ids...)
	sort.Ints(sorted)
	parts := make([]string, len(sorted))
	for i, id := range sorted {
		parts[i] = strconv.Itoa(id)
	}
	return strings.Join(parts, ", ")
}
