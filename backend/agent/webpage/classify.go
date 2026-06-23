package webpage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"app/core"
)

// classifyMaxAttempts caps how many times we re-ask the classifier when its
// reply is unparseable or the LLM call errors. The classifier is a single cheap
// call; one retry covers a transient bad reply. On exhaustion the caller falls
// back to the safest verdict (lock everything — see classifyEdit/classifyPage).
const classifyMaxAttempts = 2

// classifyReasoning: the intent classifier weighs the request but emits a tiny
// JSON object — no chain-of-thought, disabled outright like the other mechanical
// subagents (see subagentNoReasoning).
var classifyReasoning = subagentNoReasoning

// editClassification is the classifier verdict for ModeEditSection: whether the
// request is interpretable, the overall scope, and the per-dimension policy that
// the verifier enforces. Missing fields default to the safest value (keep).
type editClassification struct {
	Relevant bool   `json:"relevant"`
	Reason   string `json:"reason"`
	Scope    string `json:"scope"` // tweak | extend | rewrite
	Text     string `json:"text"`
	Images   string `json:"images"`
	Icons    string `json:"icons"`
}

// pageClassification is the classifier verdict for ModeBuildPage: which sections
// the request targets, so the verifier can require every other section to come
// back unchanged.
type pageClassification struct {
	Relevant         bool   `json:"relevant"`
	Reason           string `json:"reason"`
	Operation        string `json:"operation"` // partial | rewrite
	ModifySectionIDs []int  `json:"modifySectionIds"`
	RemoveSectionIDs []int  `json:"removeSectionIds"`
	AddSections      bool   `json:"addSections"`
}

// notInterpretableReply is pushed to the user (in Spanish) and the turn aborts
// when the classifier judges the request nonsense / unrelated to page design.
const notInterpretableReply = "No se pudo interpretar la instrucción"

const editClassifierSystemPrompt = `You classify a user's edit request for ONE website section before an HTML editor agent acts on it. You receive the request and the section's current HTML. Your job is to decide what the user wants changed so the editor changes ONLY that and preserves everything else.

Reply with ONLY a JSON object — no prose, no markdown fences:
{"relevant":true,"scope":"tweak|extend|rewrite","text":"keep|add|modify|replace","images":"keep|add|modify|replace","icons":"keep|add|modify|replace"}
On an uninterpretable request reply instead: {"relevant":false,"reason":"<short>"}

Rules:
- relevant=false ONLY when the request is nonsense or unrelated to designing/editing this web page section. A vague but on-topic request ("make it nicer", "more modern") is relevant.
- For each dimension pick: keep = user does NOT want it changed (this is the DEFAULT for anything the request does not mention); add = keep what exists and add more; modify = change existing values; replace = free rein, existing may be dropped.
- text = the visible wording. images = <img>/<ImageEffect> photos. icons = <Icon> SVG glyphs.
- scope: tweak = small change; extend = add while keeping current; rewrite = redo most of the section (then treat every dimension as replace).
- When unsure, prefer "keep" — losing content the user wanted is worse than being too conservative.`

const pageClassifierSystemPrompt = `You classify a user's request for a multi-section web page before an HTML editor agent acts on it. The current sections are given, each prefixed with a line "=== SECTION N ===". Decide which sections the request targets so the editor preserves every other section unchanged.

Reply with ONLY a JSON object — no prose, no markdown fences:
{"relevant":true,"operation":"partial","modifySectionIds":[..],"removeSectionIds":[..],"addSections":false}
On an uninterpretable request reply instead: {"relevant":false,"reason":"<short>"}

Rules:
- relevant=false ONLY when the request is nonsense or unrelated to building this page.
- operation="rewrite" ONLY when the user wants the WHOLE page redone; then the id lists are ignored.
- operation="partial" otherwise: put in modifySectionIds / removeSectionIds ONLY the section numbers the request actually targets. Every section you do NOT list MUST be preserved unchanged. addSections=true when the user wants one or more brand-new sections.
- When unsure, list FEWER sections — preserving a section is safer than letting it be rewritten.`

// classifyEdit runs the ModeEditSection classifier. It returns the verdict and
// the derived ContentPolicy. On unrecoverable failure it logs and returns the
// safest fallback: relevant=true with every dimension locked to keep.
func (t *builderTurn) classifyEdit(ctx context.Context, userText, sectionHTML string) (editClassification, ContentPolicy) {
	prompt := fmt.Sprintf("User request:\n%s\n\n--- Section HTML ---\n%s", userText, sectionHTML)
	var verdict editClassification
	if err := t.classifyInto(ctx, "classify_edit", editClassifierSystemPrompt, prompt, &verdict); err != nil {
		core.Log("agent.webpage classify_edit fallback (lock all) err::", err)
		return editClassification{Relevant: true, Scope: "tweak"}, ContentPolicy{Text: PolicyKeep, Images: PolicyKeep, Icons: PolicyKeep}
	}
	return verdict, editPolicy(verdict)
}

// classifyPage runs the ModeBuildPage classifier. On unrecoverable failure it
// returns the safest fallback: a partial op with no modify/remove ids, so every
// section must be preserved verbatim.
func (t *builderTurn) classifyPage(ctx context.Context, userText, pageHTML string) pageClassification {
	prompt := fmt.Sprintf("User request:\n%s\n\n--- Current page sections ---\n%s", userText, pageHTML)
	var verdict pageClassification
	if err := t.classifyInto(ctx, "classify_page", pageClassifierSystemPrompt, prompt, &verdict); err != nil {
		core.Log("agent.webpage classify_page fallback (lock all) err::", err)
		return pageClassification{Relevant: true, Operation: "partial"}
	}
	return verdict
}

// editPolicy turns the classifier verdict into the enforced ContentPolicy. A
// rewrite scope relaxes every dimension to replace; any unset/unknown dimension
// defaults to keep (the safe lock).
func editPolicy(v editClassification) ContentPolicy {
	if strings.EqualFold(v.Scope, "rewrite") {
		return ContentPolicy{Text: PolicyReplace, Images: PolicyReplace, Icons: PolicyReplace}
	}
	return ContentPolicy{
		Text:   normalizePolicy(v.Text),
		Images: normalizePolicy(v.Images),
		Icons:  normalizePolicy(v.Icons),
	}
}

// normalizePolicy maps a raw classifier value to a known policy, defaulting
// anything unrecognized to keep.
func normalizePolicy(p string) string {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case PolicyAdd:
		return PolicyAdd
	case PolicyModify:
		return PolicyModify
	case PolicyReplace:
		return PolicyReplace
	default:
		return PolicyKeep
	}
}

// classifyInto calls the classifier subagent (with retries) and unmarshals its
// JSON reply into out. Returns an error only when every attempt fails.
func (t *builderTurn) classifyInto(ctx context.Context, label, systemPrompt, userPrompt string, out any) error {
	var lastErr error
	for attempt := 1; attempt <= classifyMaxAttempts; attempt++ {
		raw, err := t.runSubagent(ctx, label, classifyReasoning, systemPrompt, userPrompt)
		if err != nil {
			lastErr = err
			continue
		}
		jsonText := extractJSONObject(raw)
		if jsonText == "" {
			lastErr = fmt.Errorf("classifier reply has no JSON object: %s", core.StrCut(raw, 200))
			continue
		}
		if err := json.Unmarshal([]byte(jsonText), out); err != nil {
			lastErr = fmt.Errorf("classifier JSON parse: %w (raw=%s)", err, core.StrCut(jsonText, 200))
			continue
		}
		core.Log("agent.webpage", label, "verdict::", core.StrCut(jsonText, 200))
		return nil
	}
	return lastErr
}

// extractJSONObject returns the substring from the first '{' to the last '}',
// tolerating markdown fences or stray prose around the classifier's JSON.
func extractJSONObject(s string) string {
	start := strings.IndexByte(s, '{')
	end := strings.LastIndexByte(s, '}')
	if start < 0 || end <= start {
		return ""
	}
	return s[start : end+1]
}
