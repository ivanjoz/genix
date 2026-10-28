package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"app/core"
	"genix-agent/llm"
	"genix-agent/salestools"
)

// The browser tools of a turn (get_page, get_menu, navigate, invoke_batch) and the helpers that
// shape their results for the model. The turn itself runs on the Eino harness (execution_turn.go).

// dispatchTool runs one tool call against the connected browser tab. Page
// results keep JSON metadata and raw snapshot text in separate sections;
// failures stay JSON so the model can recover from them.
func (s *AgentSession) dispatchTool(ctx context.Context, call llm.ToolCall, mutationRoutes []string) string {
	core.Log("agent.chat-ws dispatchTool tab::", shortTabID(s.TabID), " name::", call.Function.Name, " args::", core.StrCut(call.Function.Arguments, 200), " page_connected::", IsConnected(s.TabID))
	switch call.Function.Name {
	case llm.GetPageToolName:
		page, err := GetPageContent(ctx, s.TabID)
		if err != nil {
			core.Log("agent.chat-ws get_page error tab::", shortTabID(s.TabID), " err::", err)
			return toolErrorJSON(err)
		}
		// Parsed HTML is far smaller and easier for the model to read than
		// the raw DOM snapshot the browser sent; same parse runs on the Page
		// returned by navigate/invoke_batch via compactPage. Components are
		// dropped because every field (id/type/label/methods/options) is
		// already encoded in the rewritten tags.
		compactPage(&page)
		core.Log("agent.chat-ws get_page ok tab::", shortTabID(s.TabID), " html_bytes::", len(page.HTML))
		return pageToolResult(map[string]any{"ok": true}, page.HTML)

	case llm.GetMenuToolName:
		menu, err := GetMenu(ctx, s.TabID)
		if err != nil {
			core.Log("agent.chat-ws get_menu error tab::", shortTabID(s.TabID), " err::", err)
			return toolErrorJSON(err)
		}
		core.Log("agent.chat-ws get_menu ok tab::", shortTabID(s.TabID), " groups::", len(menu))
		return FormatMenuTSV(menu)

	case llm.NavigateToolName:
		var args struct {
			Route             string `json:"route"`
			ReturnPageContent bool   `json:"returnPageContent"`
		}
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
			return toolErrorJSON(fmt.Errorf("decode args: %w", err))
		}
		if args.Route == "" {
			return toolErrorJSON(errors.New("route is required"))
		}
		menu, err := GetMenu(ctx, s.TabID)
		if err != nil {
			return toolErrorJSON(fmt.Errorf("validate accessible route: %w", err))
		}
		if !menuContainsRoute(menu, args.Route) {
			core.Log("agent.chat-ws navigate rejected tab::", shortTabID(s.TabID), " route::", args.Route)
			return toolErrorJSON(fmt.Errorf("route %q is not in the current accessible menu", args.Route))
		}
		navigateResult, err := NavigateWithPage(ctx, s.TabID, args.Route, args.ReturnPageContent)
		if err != nil {
			core.Log("agent.chat-ws navigate error tab::", shortTabID(s.TabID), " route::", args.Route, " err::", err)
			return toolErrorJSON(err)
		}
		// Track the route so later get_page status labels can name it.
		s.setCurrentRoute(args.Route)
		compactPage(navigateResult.Page)
		body := toolJSON(map[string]any{"ok": true, "route": args.Route})
		if args.ReturnPageContent {
			body = pageToolResult(map[string]any{"ok": true, "route": args.Route}, pageHTML(navigateResult.Page))
		}
		core.Log("agent.chat-ws navigate ok tab::", shortTabID(s.TabID), " route::", args.Route, " return_page::", args.ReturnPageContent)
		return body

	case llm.InvokeBatchToolName:
		if !routeAllowed(s.CurrentRoute(), mutationRoutes) {
			core.Log("agent.chat-ws invoke_batch rejected tab::", shortTabID(s.TabID), " current_route::", s.CurrentRoute(), " allowed_routes::", strings.Join(mutationRoutes, ","))
			return toolErrorJSON(fmt.Errorf("page actions are not allowed on route %q; navigate to the discovered target first", s.CurrentRoute()))
		}
		// The LLM-facing shape uses a string `ID` so the model can pass the id
		// straight from the snapshot — plain ("38") or composite ("38:100").
		// resolveTarget mirrors what the HTTP /agent endpoint does: split the
		// composite into parent HandleID + child arg, and rewrite
		// setValue/search/getOptions/click to their *Child variant for cells.
		var args struct {
			Invocations []struct {
				ID     string `json:"ID"`
				Method string `json:"Method"`
				Args   []any  `json:"Args"`
			} `json:"invocations"`
			ReturnPageContent bool `json:"returnPageContent"`
		}
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
			return toolErrorJSON(fmt.Errorf("decode args: %w", err))
		}
		if len(args.Invocations) == 0 {
			return toolErrorJSON(errors.New("invocations must not be empty"))
		}
		resolved := make([]InvokePayload, 0, len(args.Invocations))
		for i, inv := range args.Invocations {
			if inv.Method == "" {
				return toolErrorJSON(fmt.Errorf("invocation[%d]: missing Method", i))
			}
			handleID, method, callArgs, err := resolveTarget(inv.ID, inv.Method, inv.Args)
			if err != nil {
				return toolErrorJSON(fmt.Errorf("invocation[%d]: %w", i, err))
			}
			resolved = append(resolved, InvokePayload{HandleID: handleID, Method: method, Args: callArgs})
		}
		result, err := InvokeBatch(ctx, s.TabID, resolved, args.ReturnPageContent)
		if err != nil {
			core.Log("agent.chat-ws invoke_batch error tab::", shortTabID(s.TabID), " invocations::", len(resolved), " err::", err)
			return toolErrorJSON(err)
		}
		compactPage(result.Page)
		tsvifyOptionValues(&result)
		body := toolJSON(struct{ Results []InvocationResult }{Results: result.Results})
		if args.ReturnPageContent {
			body = pageToolResult(struct{ Results []InvocationResult }{Results: result.Results}, pageHTML(result.Page))
		}
		core.Log("agent.chat-ws invoke_batch ok tab::", shortTabID(s.TabID), " invocations::", len(resolved), " results::", len(result.Results), " return_page::", args.ReturnPageContent)
		return body

	default:
		core.Log("agent.chat-ws unknown tool tab::", shortTabID(s.TabID), " name::", call.Function.Name)
		return toolErrorJSON(fmt.Errorf("unknown tool %q — only get_page/get_menu/navigate/invoke_batch/finish are available", call.Function.Name))
	}
}

func routeAllowed(currentRoute string, allowedRoutes []string) bool {
	currentRoute = strings.TrimSpace(currentRoute)
	if currentRoute == "" {
		return false
	}
	for _, allowedRoute := range allowedRoutes {
		if currentRoute == strings.TrimSpace(allowedRoute) {
			return true
		}
	}
	return false
}

func menuContainsRoute(menu []AgentMenuGroup, expectedRoute string) bool {
	expectedRoute = strings.TrimSpace(expectedRoute)
	for _, group := range menu {
		for _, option := range group.Options {
			if strings.TrimSpace(option.Route) == expectedRoute {
				return true
			}
		}
	}
	return false
}

// completeTurn persists the agent's reply and emits the `agentReply` wire
// event. Pulled out so both the `finish` branch and the plain-text fallback
// share the same exit path.
func (s *AgentSession) completeTurn(message, summary string, tokens int32) error {
	return s.completeAnswer(salestools.Answer{Message: message, Summary: summary}, tokens)
}

// completeAnswer closes the turn with an answer that may carry tables and charts. They go to the
// browser, which stores them with the reply row; the server keeps the message and the summary,
// which is all a later turn reads.
func (s *AgentSession) completeAnswer(answer salestools.Answer, tokens int32) error {
	ts, err := saveAgentMessage(s, answer.Message, answer.Summary, tokens)
	if err != nil {
		return fmt.Errorf("persist agent message: %w", err)
	}
	s.sendJSON(ChatTypeAgentReply, ChatAgentReply{
		Message:   answer.Message,
		Summary:   answer.Summary,
		Timestamp: ts,
		Tables:    answer.Tables,
		Charts:    answer.Charts,
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

	case salestools.ToolName:
		return "Consultando las ventas…"

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

// actionLogNote wraps the prior turn's `summary`. After every agent reply that has one, the
// replayed history carries it, so the model on the next turn knows what was already done and doesn't
// redo it (the "si guardalo" bug: the model re-ran setValue on every field before clicking save).
//
// It wraps the prior turn's `summary` in the marker the system
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

// stripPageSnapshotFromToolResult rewrites a tool result so it loses its
// grammar and PAGE SNAPSHOT section while keeping the metadata intact.
// Every tool result but the latest is kept in this form (prunePageToolRounds).
// Tool results without a snapshot are returned unchanged.
func stripPageSnapshotFromToolResult(content string) string {
	metadata, _, found := splitPageToolResult(content)
	if !found {
		return content
	}
	return metadata
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

func toolResultErrorSummary(content string) (string, bool) {
	if metadata, _, found := splitPageToolResult(content); found {
		content = metadata
	}
	var decoded struct {
		Error   string `json:"error"`
		Results []struct {
			OK    bool
			Error string
		}
	}
	if err := json.Unmarshal([]byte(content), &decoded); err != nil {
		return "", false
	}
	if summary := compactToolError(decoded.Error); summary != "" {
		return summary, true
	}
	for _, result := range decoded.Results {
		if result.OK {
			continue
		}
		if summary := compactToolError(result.Error); summary != "" {
			return summary, true
		}
		return "La acción no se completó.", true
	}
	return "", false
}

func compactToolError(message string) string {
	message = strings.Join(strings.Fields(message), " ")
	const maximumRunes = 240
	characters := []rune(message)
	if len(characters) > maximumRunes {
		message = string(characters[:maximumRunes]) + "…"
	}
	return message
}

// tsvifyOptionValues rewrites InvocationResult values that decode as
// []AgentOption into a compact TSV string ("ID\tValue\n<id>\t<label>\n..."),
// so the LLM sees option lists from Select.search / Select.getOptions as a
// table instead of a JSON array (much cheaper in tokens and easier for small
// models to scan). Values that don't match the shape are left untouched.
// Mutates result in place.
func tsvifyOptionValues(result *InvokeBatchResult) {
	if result == nil {
		return
	}
	for i := range result.Results {
		r := &result.Results[i]
		if !r.OK || len(r.Value) == 0 {
			continue
		}
		var opts []AgentOption
		if err := json.Unmarshal(r.Value, &opts); err != nil || len(opts) == 0 {
			continue
		}
		// Guard against false positives — every entry must look like an
		// option record (both ID and Value populated).
		looksLikeOptions := true
		for _, o := range opts {
			if o.ID == nil || o.Value == nil {
				looksLikeOptions = false
				break
			}
		}
		if !looksLikeOptions {
			continue
		}
		var b strings.Builder
		b.WriteString("ID\tValue")
		for _, o := range opts {
			b.WriteByte('\n')
			fmt.Fprintf(&b, "%v\t%v", o.ID, o.Value)
		}
		encoded, err := json.Marshal(b.String())
		if err != nil {
			continue
		}
		r.Value = encoded
	}
}

// compactPage runs ParsePageHTML in-place on a Page snapshot returned by
// navigate / invoke_batch when returnPageContent is set. Without this the
// LLM receives the raw browser-sanitised DOM (every CSS class intact, the
// chat widget unfiltered, etc.) — get_page's branch already parses; this
// brings the other two paths to parity. nil-safe so callers don't need to
// guard. Parse errors leave the original HTML untouched, matching the
// fallback shape get_page uses.
func compactPage(page *PageContent) {
	if page == nil {
		return
	}
	parsed, err := ParsePageHTML(page.HTML, page.Components)
	if err != nil {
		return
	}
	page.HTML = parsed
	limitedHTML, limitedRows := limitTableRows(page.HTML, pageSnapshotMaxTableRows)
	page.HTML = limitedHTML
	trimmedHTML, trimmedRows := trimTableRowsToBytes(page.HTML, pageSnapshotMaxBytes)
	page.HTML = trimmedHTML
	if limitedRows+trimmedRows > 0 {
		core.Log("agent.page-snapshot rows limited::", limitedRows, " rows byte-trimmed::", trimmedRows,
			" html_bytes::", len(page.HTML), " rows_per_table::", pageSnapshotMaxTableRows, " byte_limit::", pageSnapshotMaxBytes)
	}
	// Components is redundant once parsed: id / type / label / methods / inline
	// options all live on the rewritten HTML tags. Drop it from the LLM-facing
	// envelope to save tokens. HTTP / external callers skip compactPage and
	// still receive the full Components array.
	page.Components = nil
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
