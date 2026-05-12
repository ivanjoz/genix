package agent

import (
	"context"
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

// InvokeBatch sends a list of invocations to the browser in a single WS
// request. The browser runs them sequentially with a 250ms gap between each,
// stops on the first failure, and returns one InvocationResult per invocation
// that actually ran (so a 5-element batch failing at index 2 returns 3
// results: two OK and one Error). Caller-side stop-on-error logic just
// checks the last entry's OK flag.
//
// The context budget is widened past the default 30s to accommodate the
// per-invocation gap: ~500ms of headroom is added per invocation.
func InvokeBatch(ctx context.Context, invocations []InvokePayload) ([]InvocationResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); !ok {
		budget := defaultTimeout + time.Duration(len(invocations))*500*time.Millisecond
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, budget)
		defer cancel()
	}
	var out []InvocationResult
	if err := request(ctx, CmdAgentInvoke, invocations, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetMenu returns the side-menu structure (groups + accessible options) from
// the connected browser. Used by GET /agent?get=menu to surface routes
// without leaking the menu DOM into the page snapshot.
func GetMenu(ctx context.Context) ([]AgentMenuGroup, error) {
	ctx, cancel := ctxWithDefault(ctx)
	defer cancel()
	var out []AgentMenuGroup
	err := request(ctx, CmdGetMenu, nil, &out)
	return out, err
}

// Navigate asks the browser to change the SPA route. Used by the `navigate`
// action so agents can move between pages by passing a route from GetMenu.
func Navigate(ctx context.Context, route string) error {
	ctx, cancel := ctxWithDefault(ctx)
	defer cancel()
	return request(ctx, CmdNavigate, NavigatePayload{Route: route}, nil)
}
