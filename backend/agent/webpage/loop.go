// Package webpage hosts the agentic loop for the page builder. It is a second
// loop, parallel to the chat loop in backend/agent (chat_loop.go), dedicated to
// the builder's two modes: "build page" (rewrite/add whole-page sections) and
// "edit section" (rewrite the selected section). The chat session in package
// agent routes modes 2/3 here; mode 1 ("ask") keeps the chat loop.
//
// Dependency direction is one-way: agent → webpage. The session functionality
// this loop needs is declared as the Sink interface below (satisfied
// structurally by *agent.AgentSession), so this package never imports agent
// and there is no import cycle.
package webpage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"app/agent/llm"
	"app/core"
)

// Builder agent modes. Single source of truth, shared with package agent
// (which imports this package to route) and mirrored by the frontend's
// webpage-builder route, which sends these IDs on the wire.
const (
	ModeAsk         = 1 // default chat loop — NOT handled here
	ModeBuildPage   = 2 // construir página — whole-page sections in the context
	ModeEditSection = 3 // editar sección — only the selected section in the context
)

// maxBuilderIterations caps the tool-call round-trips per turn. Builder turns
// are typically a few generate_svg / find_image calls then apply_sections, so
// the cap is generous headroom, not an expected limit.
const maxBuilderIterations = 12

// maxAestheticRevisions caps how many times the design critic can bounce a
// result back for rework before we apply it anyway. One round catches the
// common misses (tiny image, no padding) without risking a critic↔model
// ping-pong that eats the iteration budget.
const maxAestheticRevisions = 1

// maxContentRevisions caps how many times the deterministic content-preservation
// gate can bounce a result back before we apply it anyway. It is higher than the
// aesthetic budget because preserving the user's content is a hard correctness
// requirement, not a quality nudge — give the model more chances to comply.
const maxContentRevisions = 2

// Reasoning budgets. The main loop leaves reasoning unset so each model's
// registry entry decides its own effort (llm.Models — muse-spark reasons at
// medium, hy3-preview stays at low+exclude). The subagents and the critic pin
// their budget instead: their work is mechanical or latency-sensitive, so they
// must stay cheap regardless of which model the user picked. llm.Chat maps
// these onto whatever shape the active provider expects (Meta's flat
// `reasoning_effort`, OpenRouter's nested `reasoning`).
//
// Model requirements for the builder: it must honor a reasoning budget (an
// early test with DeepSeek V4 Flash ignored effort:low and reasoned to a huge
// default — a single generate_svg took ~68s) and work with tool_choice:"auto"
// (hy3-preview's OpenRouter routing rejects "required"; the loop never uses it).
var (
	// subagentNoReasoning: generate_svg and image-select are mechanical (emit
	// markup / pick an index) — no chain-of-thought. Disabled outright. On Meta
	// this lands on the cheapest accepted budget instead, since Muse Spark
	// rejects a fully disabled one (see llm.metaReasoningEffort).
	subagentNoReasoning = &llm.ReasoningOptions{Enabled: boolPtr(false)}
	// criticReasoning: the aesthetic critic weighs the layout but must stay fast.
	criticReasoning = &llm.ReasoningOptions{Effort: "low", Exclude: true}
)

func boolPtr(b bool) *bool { return &b }

// SectionEdit is one section the agent applies back to the builder. Only the
// HTML is carried — the builder targets which section by mode (the selected
// one for edit-section; the whole ordered list replaces the page for
// build-page), so no id is needed.
type SectionEdit struct {
	HTML string `json:"html"`
	// CSS is optional raw CSS the agent authored for this section (gradients,
	// clip-path, keyframes…), using its own class names applied in HTML. The
	// frontend scopes it to page-unique `.x{n}` classes. Empty for most sections.
	CSS string `json:"css"`
	// SourceID maps a returned section back to its "=== SECTION N ===" number in
	// the build-page context, so the content gate can require unchanged sections
	// to come back verbatim. 0 = a brand-new section. Build-page only; the
	// frontend applies sections positionally and ignores this field.
	SourceID int `json:"sourceId"`
}

// Sink is the subset of the chat session the webpage loop needs to push events
// back to the browser over the shared SSE stream. Declared here (not imported
// from agent) to keep the dependency one-directional.
type Sink interface {
	// PushStatus emits a transient progress bubble (state: "thinking"|"acting").
	PushStatus(state, label string, step, maxSteps int)
	// PushReply delivers final assistant text to the chat widget. A zero
	// timestamp lets the frontend stamp it on arrival.
	PushReply(message, summary string, timestamp int64)
	// PushSections delivers the edited sections (+ any generated SVG bodies) so
	// the builder can parse them back into its AST. modeID tells the builder how
	// to apply them (replace selected section vs. replace whole page).
	PushSections(modeID int, sections []SectionEdit, svgs map[string]string, message, summary string, timestamp int64)
}

// builderTurn carries the per-turn state shared across the loop and its tool
// dispatch: the LLM client + resolved model (also used for subagents), and the
// SVG bodies generated this turn (keyed by sprite id) that ship with the final
// apply_sections payload.
type builderTurn struct {
	sink    Sink
	modeID  int
	client  *llm.Client
	modelID string
	svgs    map[string]string
	svgSeq  int
	log     *turnLog

	// Content-preservation state, set by the classifier in RunTurn before the
	// loop and read by the content gate (verifyContent) on apply_sections.
	policy         ContentPolicy          // edit-section: per-dimension policy
	sourceContent  SectionContent         // edit-section: the original section's content
	pageOp         pageClassification     // build-page: which sections may change
	sourceSections map[int]SectionContent // build-page: id → original content
}

// Package-level LLM client, cached so a missing provider key surfaces once.
var (
	llmClientOnce sync.Once
	llmClient     *llm.Client
	llmClientErr  error
)

func getLLMClient() (*llm.Client, error) {
	llmClientOnce.Do(func() { llmClient, llmClientErr = llm.NewClient() })
	return llmClient, llmClientErr
}

// RunTurn is the entry point of the page-builder loop. pageContext carries the
// builder's current section(s) serialized to HTML — the whole page for
// ModeBuildPage, the selected section for ModeEditSection. The loop ends when
// the model calls apply_sections (which pushes the edits back to the builder).
func RunTurn(ctx context.Context, sink Sink, modeID int, userText, modelHash, pageContext string) error {
	client, err := getLLMClient()
	if err != nil {
		return fmt.Errorf("llm client unavailable: %w", err)
	}
	// The builder follows the shared chat model picker. An empty hash means the
	// user never chose one — fall back to the client's default model, same as
	// the chat loop does.
	modelID := ""
	modelHash = strings.TrimSpace(modelHash)
	if modelHash != "" {
		modelConfig, ok := llm.LookupModelHash(modelHash)
		if !ok {
			return fmt.Errorf("modelo de agente no válido: %s", modelHash)
		}
		modelID = modelConfig.ID
	}
	activeModel := modelID
	if activeModel == "" {
		activeModel = client.Model
	}
	core.Log("agent.webpage RunTurn mode::", modeID, " model::", activeModel,
		" picker_hash::", modelHash,
		" prompt_bytes::", len(userText), " context_bytes::", len(pageContext))

	tlog := newTurnLog(modeID, activeModel)
	tlog.meta(modeID, userText, pageContext)

	turn := &builderTurn{sink: sink, modeID: modeID, client: client, modelID: modelID, svgs: map[string]string{}, log: tlog}

	// Intent classification (runs before the loop): decide what the user wants
	// changed so we can both instruct the model and deterministically verify it
	// preserved everything else. relevant=false aborts the turn.
	sink.PushStatus("thinking", "Interpretando…", 1, maxBuilderIterations)
	constraints, abort := turn.classifyTurn(ctx, userText, pageContext)
	if abort {
		sink.PushReply(notInterpretableReply, "", 0)
		return nil
	}

	messages := []llm.Message{
		{Role: "system", Content: systemPrompt(modeID) + constraints},
		{Role: "user", Content: buildUserMessage(modeID, userText, pageContext)},
	}

	sink.PushStatus("thinking", "Pensando…", 1, maxBuilderIterations)

	// revisionsDone counts how many times the aesthetic critic has bounced the
	// result back for rework. Once it hits maxAestheticRevisions we apply the
	// next apply_sections verbatim — the critic is a quality nudge, not a gate
	// that can wedge the turn.
	revisionsDone := 0
	// contentRevisionsDone counts content-preservation bounces (its own budget,
	// see maxContentRevisions) so a stubborn model can't wedge the turn.
	contentRevisionsDone := 0

	for iter := 0; iter < maxBuilderIterations; iter++ {
		// Reasoning is left unset on purpose: llm.Chat fills it from the active
		// model's registry entry, so each model plans at the effort it was
		// tuned for instead of a single hardcoded budget.
		req := llm.ChatRequest{
			Model:      modelID,
			Messages:   messages,
			Tools:      builderTools,
			ToolChoice: "auto",
		}
		resp, err := client.Chat(ctx, req)
		tlog.exchange(fmt.Sprintf("main_iter%d", iter+1), req, resp, err)
		if err != nil {
			return fmt.Errorf("llm chat: %w", err)
		}
		choice := resp.Choices[0]
		core.Log("agent.webpage iteration iter::", iter+1, " finish::", choice.FinishReason,
			" tool_calls::", len(choice.Message.ToolCalls), " text_bytes::", len(choice.Message.Content))

		// No tool call: model replied in plain text instead of apply_sections.
		// Treat as the final reply (no edits to apply) rather than erroring.
		if len(choice.Message.ToolCalls) == 0 {
			text := strings.TrimSpace(choice.Message.Content)
			if text == "" {
				return errors.New("el modelo devolvió un mensaje vacío sin llamar apply_sections")
			}
			sink.PushReply(text, "", 0)
			return nil
		}

		messages = append(messages, choice.Message)
		// Handle apply_sections after the other calls so every tool_call_id in
		// this assistant message gets a matching tool reply (OpenAI contract),
		// even if the critic sends the turn back for one more revision.
		var applyCall *llm.ToolCall
		for i := range choice.Message.ToolCalls {
			call := choice.Message.ToolCalls[i]
			if call.Function.Name == ApplySectionsToolName {
				applyCall = &choice.Message.ToolCalls[i]
				continue
			}
			sink.PushStatus("acting", statusLabelFor(call), iter+1, maxBuilderIterations)
			result := turn.dispatchTool(ctx, call)
			tlog.tool(call.Function.Name, call.Function.Arguments, result)
			// OpenAI contract: every tool_call_id needs a matching tool message.
			messages = append(messages, llm.Message{Role: "tool", ToolCallID: call.ID, Content: result})
		}

		if applyCall != nil {
			// Content-preservation gate (hard, runs before the soft aesthetic
			// critic so a violation bounces cheaply): reject edits that drop or
			// change content the classifier said to preserve.
			if contentRevisionsDone < maxContentRevisions {
				if violations := turn.verifyContent(*applyCall); len(violations) > 0 {
					contentRevisionsDone++
					feedback := contentViolationFeedback(violations)
					core.Log("agent.webpage apply_sections content gate veto mode::", modeID, " violations::", len(violations))
					tlog.tool("apply_sections_content_rejected", applyCall.Function.Arguments, feedback)
					messages = append(messages, llm.Message{Role: "tool", ToolCallID: applyCall.ID, Content: feedback})
					sink.PushStatus("thinking", "Corrigiendo el contenido…", iter+2, maxBuilderIterations)
					continue
				}
			}
			// Run the aesthetic critic unless the revision budget is spent.
			runCritic := revisionsDone < maxAestheticRevisions
			applied, critique, err := turn.applySections(ctx, *applyCall, runCritic)
			if err != nil {
				return err
			}
			if applied {
				return nil
			}
			// Critic rejected: feed the critique back as the tool result for the
			// apply_sections call and let the model rework + re-apply.
			revisionsDone++
			tlog.tool("apply_sections_rejected", applyCall.Function.Arguments, critique)
			messages = append(messages, llm.Message{Role: "tool", ToolCallID: applyCall.ID, Content: critique})
			sink.PushStatus("thinking", "Revisando el diseño…", iter+2, maxBuilderIterations)
			continue
		}

		sink.PushStatus("thinking", "Pensando…", iter+2, maxBuilderIterations)
	}
	return fmt.Errorf("el agente del builder excedió %d iteraciones sin llamar apply_sections", maxBuilderIterations)
}

// applySections decodes the terminator call and, unless the aesthetic critic
// vetoes it, pushes the edited sections (and the SVG bodies generated this turn)
// to the builder. Returns applied=true when the edits shipped; applied=false
// with a critique string when the critic wants a revision (the loop feeds that
// back to the model). runCritic=false skips the review (revision budget spent).
func (t *builderTurn) applySections(ctx context.Context, call llm.ToolCall, runCritic bool) (applied bool, critique string, retErr error) {
	var args struct {
		Message  string        `json:"message"`
		Summary  string        `json:"summary"`
		Sections []SectionEdit `json:"sections"`
	}
	if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
		return false, "", fmt.Errorf("parse apply_sections args: %w (raw=%s)", err, call.Function.Arguments)
	}
	if len(args.Sections) == 0 {
		return false, "", errors.New("apply_sections recibió 0 secciones")
	}

	// Aesthetic gate: a design critic inspects the proposed HTML. On a veto we
	// return its critique so the model reworks and re-applies — we do NOT push.
	if runCritic {
		t.sink.PushStatus("acting", "Revisando el diseño…", 0, maxBuilderIterations)
		if ok, fb := t.reviewAesthetics(ctx, args.Sections); !ok {
			core.Log("agent.webpage apply_sections critic veto mode::", t.modeID, " feedback_bytes::", len(fb))
			return false, fb, nil
		}
	}

	core.Log("agent.webpage apply_sections mode::", t.modeID, " sections::", len(args.Sections), " svgs::", len(t.svgs))
	if t.log != nil {
		var b strings.Builder
		fmt.Fprintf(&b, "=== APPLY_SECTIONS (turn end) ===\nmessage : %s\nsummary : %s\nsections: %d\nsvgs    : %d\n",
			args.Message, args.Summary, len(args.Sections), len(t.svgs))
		for i, s := range args.Sections {
			fmt.Fprintf(&b, "\n----- section[%d] HTML -----\n%s\n", i, s.HTML)
			if strings.TrimSpace(s.CSS) != "" {
				fmt.Fprintf(&b, "\n----- section[%d] CSS -----\n%s\n", i, s.CSS)
			}
		}
		for id, body := range t.svgs {
			fmt.Fprintf(&b, "\n----- svg %s -----\n%s\n", id, body)
		}
		t.log.note("apply_sections", b.String())
	}
	t.sink.PushSections(t.modeID, args.Sections, t.svgs, args.Message, args.Summary, 0)
	return true, "", nil
}

// reviewAesthetics runs the design critic over the proposed sections. It returns
// ok=true to ship as-is, or ok=false with an actionable critique (already framed
// as a JSON tool result) for the model to act on. Fails open: any critic error
// approves the result rather than blocking the turn.
func (t *builderTurn) reviewAesthetics(ctx context.Context, sections []SectionEdit) (ok bool, critique string) {
	var b strings.Builder
	b.WriteString("Review the aesthetic quality of this section HTML before it ships.\n\n")
	for i, s := range sections {
		fmt.Fprintf(&b, "----- Section %d HTML -----\n%s\n", i+1, s.HTML)
		if strings.TrimSpace(s.CSS) != "" {
			fmt.Fprintf(&b, "----- Section %d custom CSS -----\n%s\n", i+1, s.CSS)
		}
		b.WriteByte('\n')
	}

	// Deterministic structural checks the LLM tends to miss (ImageEffect height /
	// fit contract). Surfaced as authoritative observations the critic must fold in.
	if obs := staticLintSections(sections); len(obs) > 0 {
		b.WriteString("----- STATIC LINTER OBSERVATIONS (STATIC CHECK NO-PASS) -----\n")
		b.WriteString("A deterministic structural linter flagged the issues below. They are provably true from the markup — treat them as authoritative and INCLUDE each one in your REVISE verdict (you may add your own aesthetic notes too):\n")
		for _, o := range obs {
			fmt.Fprintf(&b, "  - %s\n", o)
		}
		b.WriteByte('\n')
	}

	out, err := t.runSubagent(ctx, "aesthetic_review", criticReasoning, aestheticReviewSystemPrompt, b.String())
	if err != nil {
		core.Log("agent.webpage aesthetic_review error::", err)
		return true, "" // fail open
	}
	trimmed := strings.TrimSpace(out)
	core.Log("agent.webpage aesthetic_review verdict::", core.StrCut(trimmed, 200))
	// Verdict contract: starts with "OK" to approve, else "REVISE: <feedback>".
	if strings.HasPrefix(strings.ToUpper(trimmed), "OK") {
		return true, ""
	}
	feedback := strings.TrimSpace(trimmed)
	if idx := strings.IndexByte(feedback, ':'); idx >= 0 && idx <= 8 {
		feedback = strings.TrimSpace(feedback[idx+1:])
	}
	if feedback == "" {
		return true, "" // empty critique ⇒ nothing actionable, ship it
	}
	return false, toolJSON(map[string]any{
		"approved":     false,
		"designReview": feedback,
		"instruction":  "A design reviewer found aesthetic issues in your HTML. Apply ONLY these fixes and re-call apply_sections. CRITICAL: keep everything else exactly as it was — every existing image, icon (<Icon …>), text and element must stay. Do NOT remove or drop anything that the review did not mention.",
	})
}

// runSubagent performs one stand-alone LLM call with no tools — used by
// generate_svg, find_image's selection step, and the aesthetic critic. The
// caller passes the reasoning budget: DeepSeek V4 Flash ignores effort:low and
// reasons to a huge default budget (a single generate_svg was seen at ~68s /
// 5800 output tokens), so subagents that don't need chain-of-thought disable it
// outright and the ones that do cap it with an explicit max_tokens.
func (t *builderTurn) runSubagent(ctx context.Context, label string, reasoning *llm.ReasoningOptions, systemPromptText, userPrompt string) (string, error) {
	req := llm.ChatRequest{
		Model:     t.modelID,
		Reasoning: reasoning,
		Messages: []llm.Message{
			{Role: "system", Content: systemPromptText},
			{Role: "user", Content: userPrompt},
		},
	}
	resp, err := t.client.Chat(ctx, req)
	t.log.exchange("subagent_"+label, req, resp, err)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

// buildUserMessage appends the builder's per-turn context to the user's request.
// The frontend owns the framing of pageContext (it knows the mode, and includes the
// current color palette + the section HTML, each clearly labeled), so the backend
// just concatenates rather than re-labeling.
func buildUserMessage(modeID int, userText, pageContext string) string {
	if strings.TrimSpace(pageContext) == "" {
		return userText
	}
	return fmt.Sprintf("%s\n\n%s", userText, pageContext)
}

// statusLabelFor maps a tool call to a Spanish progress label for the widget.
func statusLabelFor(call llm.ToolCall) string {
	switch call.Function.Name {
	case GenerateSVGToolName:
		return "Generando un ícono…"
	case FindImageToolName:
		return "Buscando una imagen…"
	default:
		return "Trabajando…"
	}
}

// toolJSON / toolErrorJSON marshal a tool result for the model; errors use a
// stable {"error":"..."} shape so the model can recover.
func toolJSON(v any) string {
	body, err := json.Marshal(v)
	if err != nil {
		return toolErrorJSON(fmt.Errorf("marshal tool result: %w", err))
	}
	return string(body)
}

func toolErrorJSON(err error) string {
	body, _ := json.Marshal(map[string]string{"error": err.Error()})
	return string(body)
}
