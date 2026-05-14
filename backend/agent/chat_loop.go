package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"app/agent/llm"
	"app/core"
	"app/types"
)

// RunTurn is the per-user-message entry point of the agentic loop. The model
// is offered the page-driving tools (`get_page`, `get_menu`, `navigate`,
// `invoke_batch`) plus the terminator `finish`. Each iteration is one
// OpenRouter call; tool calls dispatch through the existing /ws/agent bridge
// and feed their result back as a `tool` message on the next iteration. The
// loop terminates when the model calls `finish` or hits the iteration cap.

const (
	// historyTurnTail caps how many persisted messages we attach to each LLM
	// call. Keeps prompt tokens bounded as conversations grow.
	historyTurnTail = 5
	// maxLoopIterations is the safety cap on tool-call/result round-trips per
	// user turn. Tuned for the typical "get_page → navigate → invoke_batch →
	// finish" arc with a few retries; bump if real usage shows it bites.
	maxLoopIterations = 8
	// toolResultMaxBytes truncates a tool result before it goes back into the
	// conversation. Page HTML can be tens of KB and we don't want one call to
	// blow the model's context window. The truncation marker tells the model
	// the result was clipped so it can ask the user or retry differently.
	toolResultMaxBytes = 24_000
	// keepRecentToolRounds caps how many (assistant tool_calls + tool result)
	// pairs survive between iterations of the same turn. A get_page snapshot
	// can be ~10K tokens; replaying every prior snapshot on every iteration
	// blows the context window fast (turns observed at 20K+ in/iteration).
	// Keeping only the most recent rounds means the model still has the
	// latest page state but stale snapshots are dropped — if it needs older
	// information it can call get_page() again.
	keepRecentToolRounds = 2
)

// Package-level LLM client. NewClient is cheap but we cache it so a missing
// OPENROUTER_KEY surfaces once instead of on every turn.
var (
	llmClientOnce sync.Once
	llmClient     *llm.Client
	llmClientErr  error
)

func getLLMClient() (*llm.Client, error) {
	llmClientOnce.Do(func() {
		llmClient, llmClientErr = llm.NewClient()
	})
	return llmClient, llmClientErr
}

// RunTurn persists the user's message, drives the LLM loop until `finish` is
// called (or the cap is hit), persists the agent's reply, and pushes the
// reply over the chat WS. Caller is responsible for the InFlight guard.
func (s *AgentSession) RunTurn(ctx context.Context, userText string) error {
	client, err := getLLMClient()
	if err != nil {
		return fmt.Errorf("llm client unavailable: %w", err)
	}

	if _, err := saveMessage(s, RoleUser, userText, "", 0); err != nil {
		return fmt.Errorf("persist user message: %w", err)
	}

	history, err := loadLastN(s, historyTurnTail)
	if err != nil {
		return fmt.Errorf("load history: %w", err)
	}

	messages := buildChatMessages(history)
	// baseLen is the size of the prompt before any tool round is appended —
	// system + replayed history + (the user message of this turn is in the
	// replayed history, since we saved it just above). Tool round pruning
	// only touches indices >= baseLen.
	baseLen := len(messages)
	var totalTokens int32

	s.pushStatus("thinking", "Pensando…", 1, maxLoopIterations)

	for iter := 0; iter < maxLoopIterations; iter++ {
		messages = pruneToolRounds(messages, baseLen, keepRecentToolRounds)
		// Reasoning defaults (effort/exclude) come from the per-model registry
		// in llm/models.go — Client.Chat fills them in if we leave them nil.
		// That way switching models via OPENROUTER_MODEL just works without
		// touching the loop body.
		resp, err := client.Chat(ctx, llm.ChatRequest{
			Messages:   messages,
			Tools:      llm.ChatTools,
			ToolChoice: "auto",
		})
		if err != nil {
			return fmt.Errorf("openrouter chat: %w", err)
		}
		totalTokens += resp.Usage.TotalTokens

		choice := resp.Choices[0]

		// Branch 1: model called a tool.
		if len(choice.Message.ToolCalls) > 0 {
			call := choice.Message.ToolCalls[0]
			if call.Function.Name == llm.FinishToolName {
				message, summary, parseErr := parseFinishArgs(call.Function.Arguments)
				if parseErr != nil {
					return fmt.Errorf("parse finish args: %w (raw=%s)", parseErr, call.Function.Arguments)
				}
				return s.completeTurn(message, summary, totalTokens)
			}
			// Page tool — emit a friendly progress label, run it via /ws/agent,
			// and feed the JSON-encoded result back as a `tool` message. The
			// model decides what to do next on the following iteration.
			s.pushStatus("acting", s.statusLabelFor(call), iter+1, maxLoopIterations)
			toolResult := s.dispatchTool(ctx, call)
			messages = append(messages, choice.Message, llm.Message{
				Role:       "tool",
				ToolCallID: call.ID,
				Content:    clipForLLM(toolResult),
			})
			// Brief "thinking" status between tool result and the next LLM call
			// so the widget doesn't show a stale label while we wait.
			s.pushStatus("thinking", "Pensando…", iter+2, maxLoopIterations)
			continue
		}

		// Branch 2: model emitted plain assistant text. Per the prompt this
		// shouldn't happen, but treating it as a final answer is the
		// user-friendly recovery — better than telling them "agent broke."
		text := strings.TrimSpace(choice.Message.Content)
		if text == "" {
			return errors.New("openrouter returned empty assistant message and no tool call")
		}
		core.Log("agent.chat-ws plain-text reply (no finish call) tab::", s.TabID)
		return s.completeTurn(text, "", totalTokens)
	}

	return fmt.Errorf("agent exceeded %d iterations without calling finish", maxLoopIterations)
}

// dispatchTool runs one tool call against the connected browser tab and
// returns the result as a JSON string for the LLM. Failures are returned as
// JSON `{"error":"..."}` so the model can recover rather than seeing a raw
// Go error string — same shape as the success case, just an extra key.
func (s *AgentSession) dispatchTool(ctx context.Context, call llm.ToolCall) string {
	core.Log("agent.chat-ws dispatchTool tab::", s.TabID, " name::", call.Function.Name, " args::", core.StrCut(call.Function.Arguments, 200))
	switch call.Function.Name {
	case llm.GetPageToolName:
		page, err := GetPageContent(ctx, s.TabID)
		if err != nil {
			return toolErrorJSON(err)
		}
		// Parsed HTML is far smaller and easier for the model to read than the
		// raw DOM snapshot the browser sent.
		parsedHTML, perr := ParsePageHTML(page.HTML, page.Components)
		if perr != nil {
			parsedHTML = page.HTML
		}
		return toolJSON(map[string]any{
			"components": page.Components,
			"html":       parsedHTML,
		})

	case llm.GetMenuToolName:
		menu, err := GetMenu(ctx, s.TabID)
		if err != nil {
			return toolErrorJSON(err)
		}
		return toolJSON(map[string]any{"groups": menu})

	case llm.NavigateToolName:
		var args struct {
			Route string `json:"route"`
		}
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
			return toolErrorJSON(fmt.Errorf("decode args: %w", err))
		}
		if args.Route == "" {
			return toolErrorJSON(errors.New("route is required"))
		}
		if err := Navigate(ctx, s.TabID, args.Route); err != nil {
			return toolErrorJSON(err)
		}
		// Track the route so later get_page status labels can name it.
		s.setCurrentRoute(args.Route)
		return toolJSON(map[string]any{"ok": true, "route": args.Route})

	case llm.InvokeBatchToolName:
		var args struct {
			Invocations []InvokePayload `json:"invocations"`
		}
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
			return toolErrorJSON(fmt.Errorf("decode args: %w", err))
		}
		if len(args.Invocations) == 0 {
			return toolErrorJSON(errors.New("invocations must not be empty"))
		}
		results, err := InvokeBatch(ctx, s.TabID, args.Invocations)
		if err != nil {
			return toolErrorJSON(err)
		}
		return toolJSON(map[string]any{"results": results})

	default:
		core.Log("agent.chat-ws unknown tool tab::", s.TabID, " name::", call.Function.Name)
		return toolErrorJSON(fmt.Errorf("unknown tool %q — only get_page/get_menu/navigate/invoke_batch/finish are available", call.Function.Name))
	}
}

// completeTurn persists the agent's reply and emits the `agentReply` wire
// event. Pulled out so both the `finish` branch and the plain-text fallback
// share the same exit path.
func (s *AgentSession) completeTurn(message, summary string, tokens int32) error {
	ts, err := saveMessage(s, RoleAgent, message, summary, tokens)
	if err != nil {
		return fmt.Errorf("persist agent message: %w", err)
	}
	s.sendJSON(ChatTypeAgentReply, ChatAgentReply{
		Message:   message,
		Summary:   summary,
		Timestamp: ts,
	})
	return nil
}

// pushStatus emits an agentStatus event so the widget can render a transient
// progress bubble while the loop iterates. `Label` is the human string the
// widget shows (e.g. "Consultando menú…"); `State` is the coarse phase the
// label belongs to (thinking|acting) for any future styling. Best-effort:
// the status stream is a UX nicety, not load-bearing.
func (s *AgentSession) pushStatus(state, label string, step, maxSteps int) {
	s.sendJSON(ChatTypeAgentStatus, ChatAgentStatus{
		State: state, Label: label, Step: step, MaxSteps: maxSteps,
	})
}

// statusLabelFor produces a Spanish progress label for a non-finish tool
// call. Method on AgentSession so labels can name the current SPA route
// (e.g. "Leyendo información de '/negocio/productos'…"). Unknown tools and
// unmapped methods fall back to a generic label.
func (s *AgentSession) statusLabelFor(call llm.ToolCall) string {
	switch call.Function.Name {
	case llm.GetPageToolName:
		if route := s.CurrentRoute(); route != "" {
			return "Leyendo información de '" + route + "'…"
		}
		return "Leyendo información de la página actual…"

	case llm.GetMenuToolName:
		return "Listando páginas disponibles…"

	case llm.NavigateToolName:
		var args struct {
			Route string `json:"route"`
		}
		_ = json.Unmarshal([]byte(call.Function.Arguments), &args)
		if args.Route != "" {
			return "Navegando hacia '" + args.Route + "'…"
		}
		return "Navegando…"

	case llm.InvokeBatchToolName:
		var args struct {
			Invocations []struct {
				Method string `json:"Method"`
			} `json:"invocations"`
		}
		_ = json.Unmarshal([]byte(call.Function.Arguments), &args)
		switch len(args.Invocations) {
		case 0:
			return "Ejecutando acción…"
		case 1:
			return methodLabel(args.Invocations[0].Method)
		default:
			return fmt.Sprintf("Ejecutando %d acciones…", len(args.Invocations))
		}

	default:
		return "Trabajando…"
	}
}

// methodLabel maps a single component method name to a Spanish progress
// verb. Aliases (e.g. setValueChild → "Llenando un campo") share a label
// with their parent variant so the UX is uniform. Unknown methods fall
// through to a generic "Ejecutando $method…".
func methodLabel(method string) string {
	switch method {
	case "click", "clickChild":
		return "Haciendo click…"
	case "setValue", "setValueChild":
		return "Llenando un campo…"
	case "select":
		return "Seleccionando una opción…"
	case "getOptions", "getOptionsChild":
		return "Consultando opciones…"
	case "search", "searchChild":
		return "Buscando…"
	case "save":
		return "Guardando…"
	case "remove":
		return "Eliminando…"
	case "getValue":
		return "Leyendo valor…"
	case "":
		return "Ejecutando acción…"
	default:
		return "Ejecutando " + method + "…"
	}
}

// buildChatMessages converts persisted history into the LLM wire shape. The
// system prompt always leads; user/agent rows are mapped to user/assistant
// roles. After every agent row that has a non-trivial summary, we splice in
// an extra `system` message with the action log so the model on the *next*
// turn knows what was already done and doesn't redo it (e.g. re-filling a
// form whose values were set in the prior turn — see the "si guardalo" bug
// where the model re-ran setValue on every field before clicking save).
//
// Empty messages are dropped defensively (shouldn't exist, but a stray
// empty row from a future feature shouldn't poison the prompt).
func buildChatMessages(history []types.AgentMessage) []llm.Message {
	out := make([]llm.Message, 0, len(history)+1)
	out = append(out, llm.Message{Role: "system", Content: llm.SystemPromptChat})
	for _, row := range history {
		if row.Message == "" {
			continue
		}
		switch row.Role {
		case RoleUser:
			out = append(out, llm.Message{Role: "user", Content: row.Message})
		case RoleAgent:
			out = append(out, llm.Message{Role: "assistant", Content: row.Message})
			if note := actionLogNote(row.Summary); note != "" {
				out = append(out, llm.Message{Role: "system", Content: note})
			}
		}
	}
	return out
}

// actionLogNote wraps the prior turn's `summary` in the marker the system
// prompt told the model to expect. Returns empty when the summary is empty
// or matches the "no actions" sentinel, so a chat-only turn doesn't bloat
// the next prompt with a useless note.
func actionLogNote(summary string) string {
	s := strings.TrimSpace(summary)
	if s == "" {
		return ""
	}
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "sin acciones") || strings.HasPrefix(lower, "no actions") {
		return ""
	}
	return "[Acciones ya realizadas en esta conversación] " + s +
		" — No repitas estas acciones. Si necesitas verificar el estado actual, llama a get_page()."
}

// parseFinishArgs decodes the `finish` tool arguments. OpenAI's contract
// delivers them as a JSON-encoded string, so we Unmarshal twice-removed.
func parseFinishArgs(raw string) (message, summary string, err error) {
	var args struct {
		Message string `json:"message"`
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return "", "", err
	}
	return strings.TrimSpace(args.Message), strings.TrimSpace(args.Summary), nil
}

// pruneToolRounds drops all but the most recent `keep` tool rounds from the
// tail of `messages`. A round starts at an `assistant` message with one or
// more `tool_calls` and ends just before the next `assistant` (or the end
// of the slice). All `tool` messages belonging to that assistant stay with
// it — the OpenAI/OpenRouter contract requires every tool_call_id to be
// answered by a matching `tool` message in the same prompt, so we can never
// drop a tool reply without dropping its originating assistant message too.
//
// Everything below baseLen (system + replayed history) is untouched.
// keep <= 0 collapses every tool round, leaving only the base prompt.
func pruneToolRounds(messages []llm.Message, baseLen, keep int) []llm.Message {
	if baseLen >= len(messages) {
		return messages
	}
	// Find the start index of each round inside the appended tail.
	roundStarts := make([]int, 0, 4)
	for i := baseLen; i < len(messages); i++ {
		if messages[i].Role == "assistant" && len(messages[i].ToolCalls) > 0 {
			roundStarts = append(roundStarts, i)
		}
	}
	if len(roundStarts) <= keep {
		return messages
	}
	// Drop everything from the first preserved round backwards, keeping the
	// base prompt intact. roundStarts[len-keep] is the first index we want
	// to preserve.
	cutFrom := roundStarts[len(roundStarts)-keep]
	dropped := cutFrom - baseLen
	pruned := make([]llm.Message, 0, baseLen+(len(messages)-cutFrom))
	pruned = append(pruned, messages[:baseLen]...)
	pruned = append(pruned, messages[cutFrom:]...)
	core.Log("agent.chat-loop pruned tool rounds; dropped::", dropped, " kept::", len(pruned)-baseLen, " rounds_total::", len(roundStarts))
	return pruned
}

// toolJSON marshals a tool result for the LLM. Falls back to a stable error
// shape if marshal fails — the loop should never wedge over an encoding bug.
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

// clipForLLM truncates oversized tool results so a giant page snapshot can't
// blow the model's context window. The marker tells the model the output was
// clipped — it'll usually ask the user for guidance instead of looping.
func clipForLLM(s string) string {
	if len(s) <= toolResultMaxBytes {
		return s
	}
	return s[:toolResultMaxBytes] + "\n…[truncated: result exceeded " + fmt.Sprint(toolResultMaxBytes) + " bytes]"
}
