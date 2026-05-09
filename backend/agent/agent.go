package agent

import (
	"context"
	"encoding/json"
	"time"
)

// Public Go API. The backend (the agent) calls these to drive the browser.
// Each call blocks until the browser replies, the context is cancelled, or
// the default timeout elapses.

const defaultTimeout = 30 * time.Second

func ctxWithDefault(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, defaultTimeout)
}

// GetPageContent asks the browser for the live agent registry + sanitized HTML.
func GetPageContent(ctx context.Context) (PageContent, error) {
	ctx, cancel := ctxWithDefault(ctx)
	defer cancel()
	var out PageContent
	err := request(ctx, CmdGetPageContent, nil, &out)
	return out, err
}

// Screenshot grabs a frame from the active getDisplayMedia stream.
// First call in a session triggers a permission prompt in the browser.
func Screenshot(ctx context.Context) (ScreenshotResult, error) {
	// Screenshot may need extra time the very first call (user must accept the
	// share-screen prompt), so widen the default budget.
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
		defer cancel()
	}
	var out ScreenshotResult
	err := request(ctx, CmdScreenshot, nil, &out)
	return out, err
}

// ListComponents enumerates registered agent handles.
func ListComponents(ctx context.Context, filter *AgentListFilter) ([]AgentComponentInfo, error) {
	ctx, cancel := ctxWithDefault(ctx)
	defer cancel()
	var out []AgentComponentInfo
	err := request(ctx, CmdAgentList, filter, &out)
	return out, err
}

// Invoke calls a method on a registered handle. Pass nil/empty Args for methods
// that take no parameters. Result is JSON-decoded into out (pass nil if you
// don't need the reply body, e.g. for void methods).
func Invoke(ctx context.Context, handleID int, method string, args []any, out any) error {
	ctx, cancel := ctxWithDefault(ctx)
	defer cancel()
	return request(ctx, CmdAgentInvoke, InvokePayload{HandleID: handleID, Method: method, Args: args}, out)
}

// InvokeRaw is like Invoke but returns the raw JSON of the result envelope so
// the caller can decode dynamically-typed responses themselves.
func InvokeRaw(ctx context.Context, handleID int, method string, args []any) (json.RawMessage, error) {
	ctx, cancel := ctxWithDefault(ctx)
	defer cancel()
	var raw json.RawMessage
	err := request(ctx, CmdAgentInvoke, InvokePayload{HandleID: handleID, Method: method, Args: args}, &raw)
	return raw, err
}
