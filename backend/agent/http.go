package agent

// HTTP entrypoint used by external LLM agents (Claude Code, Gemini) to drive
// the page. Local-only, no auth: the listener is bound to localhost.
//
// Contract:
//   POST /agent  { Actions: [ { ID, Method, Args } ] }
// Stops on the first failing action. Always returns the post-action page
// snapshot so the caller can decide its next step without a second round-trip.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type AgentAction struct {
	ID     string
	Method string
	Args   []any
}

type AgentActionResult struct {
	OK    bool
	Value json.RawMessage `json:",omitempty"`
	Error string          `json:",omitempty"`
}

type AgentRequest struct {
	Actions []AgentAction
}

type AgentResponse struct {
	Results []AgentActionResult
	Page    PageContent
}

// HandleAgentGet serves the read-only side-channel queries (currently only
// `?get=menu`). The POST /agent endpoint stays the place to drive the page;
// this one exists so the menu structure can be fetched without dumping it
// into the HTML snapshot every action call returns.
func HandleAgentGet(w http.ResponseWriter, r *http.Request) {
	if !IsConnected() {
		writeJSONError(w, http.StatusServiceUnavailable, "no agent client connected")
		return
	}
	switch r.URL.Query().Get("get") {
	case "menu":
		menu, err := GetMenu(r.Context())
		if err != nil {
			writeJSONError(w, http.StatusBadGateway, "menu fetch failed: "+err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(struct{ Menu []AgentMenuGroup }{menu})
	default:
		writeJSONError(w, http.StatusBadRequest, "missing or unsupported `get` query (expected ?get=menu)")
	}
}

// HandleAgentHTTP runs a batch of actions sequentially against the connected
// browser and returns a fresh page snapshot. Designed to be mounted at /agent.
func HandleAgentHTTP(w http.ResponseWriter, r *http.Request) {
	if !IsConnected() {
		writeJSONError(w, http.StatusServiceUnavailable, "no agent client connected")
		return
	}

	var req AgentRequest
	if r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
			return
		}
	}

	ctx := r.Context()
	results := make([]AgentActionResult, 0, len(req.Actions))
	for _, action := range req.Actions {
		raw, err := runAction(ctx, action)
		if err != nil {
			results = append(results, AgentActionResult{OK: false, Error: err.Error()})
			break // stop on first error
		}
		results = append(results, AgentActionResult{OK: true, Value: raw})
	}

	// Always include the post-action page snapshot so the caller can use
	// `{actions: []}` as a "give me the current page" call too.
	page, err := GetPageContent(ctx)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "page fetch failed: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(AgentResponse{Results: results, Page: page})
}

// runAction resolves the target handle from the action id and forwards the call
// to the browser via the existing WS RPC. The `navigate` method is special-
// cased: it is a global page action, not a method on a registered handle, so
// it bypasses id resolution and goes straight to the SPA router.
func runAction(ctx context.Context, a AgentAction) (json.RawMessage, error) {
	if a.Method == "" {
		return nil, errors.New("missing method")
	}
	if a.Method == "navigate" {
		route, err := navigateRouteArg(a.Args)
		if err != nil {
			return nil, err
		}
		if err := Navigate(ctx, route); err != nil {
			return nil, err
		}
		return json.RawMessage("null"), nil
	}
	handleID, method, args, err := resolveTarget(a.ID, a.Method, a.Args)
	if err != nil {
		return nil, err
	}
	return InvokeRaw(ctx, handleID, method, args)
}

// navigateRouteArg pulls the route string from a navigate action's args.
// Accepts either a single positional string ("/comercial/...") or an object
// with a `Route` field for callers that prefer named args.
func navigateRouteArg(args []any) (string, error) {
	if len(args) == 0 {
		return "", errors.New("navigate requires a route argument")
	}
	switch v := args[0].(type) {
	case string:
		if v == "" {
			return "", errors.New("navigate route is empty")
		}
		return v, nil
	case map[string]any:
		if r, ok := v["Route"].(string); ok && r != "" {
			return r, nil
		}
		if r, ok := v["route"].(string); ok && r != "" {
			return r, nil
		}
	}
	return "", fmt.Errorf("navigate: invalid route arg %T", args[0])
}

// childAliasMethods maps the cell-facing method name (the one printed in the
// HTML snapshot's `methods="..."`) to the Table handle's internal routing
// variant. Calls on a composite id with one of these methods are rewritten:
// `setValue("38:101", v)` becomes `setValueChild(101, v)` on Table 38, with
// the parent prefix stripped from the id arg. Methods not listed here keep
// their original name and receive the full composite id (e.g.
// SearchCard.remove("58:235"), Table.select("38:100")).
var childAliasMethods = map[string]string{
	"setValue":   "setValueChild",
	"search":     "searchChild",
	"getOptions": "getOptionsChild",
	"click":      "clickChild",
}

// resolveTarget maps an action id to a registered handle id, the method to
// invoke, and the args to pass. Plain ids ("58") target the handle directly.
// Composite ids ("<handle>:<child>", e.g. "58:235" for an Option or "38:101"
// for a table cell) route to the parent handle. For methods listed in
// childAliasMethods the parent prefix is stripped and the method renamed to
// its `*Child` variant; otherwise the full composite is passed as args[0] so
// the handle's method can split it (e.g. SearchCard.remove).
func resolveTarget(id, method string, args []any) (int, string, []any, error) {
	if id == "" {
		return 0, "", nil, errors.New("missing id")
	}
	if colon := strings.IndexByte(id, ':'); colon >= 0 {
		handleID, err := strconv.Atoi(id[:colon])
		if err != nil {
			return 0, "", nil, fmt.Errorf("invalid handle id %q: %w", id[:colon], err)
		}
		childPart := id[colon+1:]
		if alias, ok := childAliasMethods[method]; ok {
			childID, err := strconv.Atoi(childPart)
			if err != nil {
				return 0, "", nil, fmt.Errorf("invalid child id %q: %w", childPart, err)
			}
			merged := make([]any, 0, len(args)+1)
			merged = append(merged, childID)
			merged = append(merged, args...)
			return handleID, alias, merged, nil
		}
		merged := make([]any, 0, len(args)+1)
		merged = append(merged, id)
		merged = append(merged, args...)
		return handleID, method, merged, nil
	}
	handleID, err := strconv.Atoi(id)
	if err != nil {
		return 0, "", nil, fmt.Errorf("invalid id %q: %w", id, err)
	}
	return handleID, method, args, nil
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct{ Error string }{msg})
}
