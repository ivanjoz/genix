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
// to the browser via the existing WS RPC.
func runAction(ctx context.Context, a AgentAction) (json.RawMessage, error) {
	if a.Method == "" {
		return nil, errors.New("missing method")
	}
	handleID, args, err := resolveTarget(a.ID, a.Args)
	if err != nil {
		return nil, err
	}
	return InvokeRaw(ctx, handleID, a.Method, args)
}

// resolveTarget maps an action id to a registered handle id plus the args to
// pass. Composite ids ("<handle>:<child>", e.g. "58:235" for an Option, "7:12"
// for a table cell) route to the parent handle and prepend the full composite
// id as the first argument; the handle's method splits it as needed (see
// SearchCard.remove).
func resolveTarget(id string, args []any) (int, []any, error) {
	if id == "" {
		return 0, nil, errors.New("missing id")
	}
	if colon := strings.IndexByte(id, ':'); colon >= 0 {
		handleID, err := strconv.Atoi(id[:colon])
		if err != nil {
			return 0, nil, fmt.Errorf("invalid handle id %q: %w", id[:colon], err)
		}
		merged := make([]any, 0, len(args)+1)
		merged = append(merged, id)
		merged = append(merged, args...)
		return handleID, merged, nil
	}
	handleID, err := strconv.Atoi(id)
	if err != nil {
		return 0, nil, fmt.Errorf("invalid id %q: %w", id, err)
	}
	return handleID, args, nil
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct{ Error string }{msg})
}
