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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type AgentAction struct {
	ID     string
	Method string
	Args   []any
}

type AgentRequest struct {
	Actions []AgentAction
}

type AgentResponse struct {
	Results []InvocationResult
	Page    PageContent
}

// HandleAgentGet serves the read-only side-channel queries (currently only
// `?get=menu`). The POST /agent endpoint stays the place to drive the page;
// this one exists so the menu structure can be fetched without dumping it
// into the HTML snapshot every action call returns.
func HandleAgentGet(w http.ResponseWriter, r *http.Request) {
	tab, err := ResolveTab(r.URL.Query().Get("tab"))
	if err != nil {
		writeJSONError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	switch r.URL.Query().Get("get") {
	case "menu":
		menu, err := GetMenu(r.Context(), tab)
		if err != nil {
			writeJSONError(w, http.StatusBadGateway, "menu fetch failed: "+err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(struct{ Menu []AgentMenuGroup }{menu})
	case "screenshot":
		writeScreenshot(w, r, tab, Screenshot, "genix-agent-screenshot.png")
	case "screenshot-real":
		writeScreenshot(w, r, tab, ScreenshotReal, "genix-agent-screenshot-real.png")
	default:
		writeJSONError(w, http.StatusBadRequest, "missing or unsupported `get` query (expected ?get=menu|screenshot|screenshot-real)")
	}
}

// HandleAgentHTTP groups the request's actions into invoke batches separated
// by navigate calls (and pre-flight errors), dispatches each batch as a single
// WS round-trip, and returns one InvocationResult per executed action plus a
// fresh page snapshot. Stops on first error, same as before — only the
// transport changed.
func HandleAgentHTTP(w http.ResponseWriter, r *http.Request) {
	tab, err := ResolveTab(r.URL.Query().Get("tab"))
	if err != nil {
		writeJSONError(w, http.StatusServiceUnavailable, err.Error())
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
	results := runActions(ctx, tab, req.Actions)

	// Always include the post-action page snapshot so the caller can use
	// `{actions: []}` as a "give me the current page" call too.
	page, err := GetPageContent(ctx, tab)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "page fetch failed: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(AgentResponse{Results: results, Page: page})
}

// runActions walks the request actions, buffering contiguous invokes into a
// batch and flushing the batch around navigates and pre-flight errors.
// Returns the result list, truncated at the first failure.
func runActions(ctx context.Context, tab string, actions []AgentAction) []InvocationResult {
	results := make([]InvocationResult, 0, len(actions))
	pending := make([]InvokePayload, 0)

	// flushPending dispatches the buffered batch (if any), appends its
	// results, and reports whether the batch fully succeeded. A transport
	// failure or any non-OK invocation result terminates the run.
	flushPending := func() bool {
		if len(pending) == 0 {
			return true
		}
		batch, err := InvokeBatch(ctx, tab, pending)
		pending = pending[:0]
		if err != nil {
			results = append(results, InvocationResult{OK: false, Error: err.Error()})
			return false
		}
		for _, r := range batch {
			results = append(results, r)
			if !r.OK {
				return false
			}
		}
		return true
	}

	for _, action := range actions {
		if action.Method == "" {
			if !flushPending() {
				return results
			}
			results = append(results, InvocationResult{OK: false, Error: "missing method"})
			return results
		}
		if action.Method == "navigate" {
			if !flushPending() {
				return results
			}
			route, err := navigateRouteArg(action.Args)
			if err != nil {
				results = append(results, InvocationResult{OK: false, Error: err.Error()})
				return results
			}
			if err := Navigate(ctx, tab, route); err != nil {
				results = append(results, InvocationResult{OK: false, Error: err.Error()})
				return results
			}
			results = append(results, InvocationResult{OK: true, Value: json.RawMessage("null")})
			continue
		}
		handleID, method, args, err := resolveTarget(action.ID, action.Method, action.Args)
		if err != nil {
			if !flushPending() {
				return results
			}
			results = append(results, InvocationResult{OK: false, Error: err.Error()})
			return results
		}
		pending = append(pending, InvokePayload{HandleID: handleID, Method: method, Args: args})
	}
	flushPending()
	return results
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

// writeScreenshot runs `capture` (Screenshot or ScreenshotReal), decodes the
// base64 result to a stable PNG file under the OS temp dir, and returns the
// path + dimensions. The two screenshot variants write to distinct filenames
// so an agent can hold both captures side-by-side without one overwriting the
// other. LLM agents read the path to view the image, since base64 strings in
// JSON aren't natively viewable.
func writeScreenshot(
	w http.ResponseWriter,
	r *http.Request,
	tab string,
	capture func(context.Context, string) (ScreenshotResult, error),
	filename string,
) {
	shot, err := capture(r.Context(), tab)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "screenshot fetch failed: "+err.Error())
		return
	}
	raw, err := base64.StdEncoding.DecodeString(shot.Base64)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "screenshot decode failed: "+err.Error())
		return
	}
	path := filepath.Join(os.TempDir(), filename)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "screenshot write failed: "+err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Path   string
		MIME   string
		Width  int
		Height int
	}{path, shot.MIME, shot.Width, shot.Height})
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct{ Error string }{msg})
}
