package agent

import (
	"context"
	"time"
)

// Public Go API. The backend (the agent) calls these to drive one browser
// tab. Each call blocks until that tab replies, the context is cancelled, or
// the default timeout elapses. `tab` is the TabID the frontend sent on
// connect; callers that don't have one can use ResolveTab to pick the unique
// connected tab.

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
func GetPageContent(ctx context.Context, tab string) (PageContent, error) {
	ctx, cancel := ctxWithDefault(ctx)
	defer cancel()
	var out PageContent
	err := request(ctx, tab, CmdGetPageContent, nil, &out)
	return out, err
}

// Screenshot renders the page DOM directly via modern-screenshot. No user
// prompt, no browser banner — only the page content is captured (no
// surrounding browser chrome or other tabs). This is the default capture path
// used by public users since it requires no permission.
func Screenshot(ctx context.Context, tab string) (ScreenshotResult, error) {
	ctx, cancel := ctxWithDefault(ctx)
	defer cancel()
	var out ScreenshotResult
	err := request(ctx, tab, CmdScreenshot, nil, &out)
	return out, err
}

// ScreenshotReal grabs a frame from the active getDisplayMedia stream — the
// pixel-real view including any cross-origin content the DOM renderer can't
// reach. First call in a session triggers a permission prompt in the browser.
func ScreenshotReal(ctx context.Context, tab string) (ScreenshotResult, error) {
	// May need extra time the very first call (user must accept the
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
	err := request(ctx, tab, CmdScreenshotReal, nil, &out)
	return out, err
}

// ListComponents enumerates registered agent handles.
func ListComponents(ctx context.Context, tab string, filter *AgentListFilter) ([]AgentComponentInfo, error) {
	ctx, cancel := ctxWithDefault(ctx)
	defer cancel()
	var out []AgentComponentInfo
	err := request(ctx, tab, CmdAgentList, filter, &out)
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
func InvokeBatch(ctx context.Context, tab string, invocations []InvokePayload, returnPageContent bool) (InvokeBatchResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); !ok {
		budget := defaultTimeout + time.Duration(len(invocations))*500*time.Millisecond
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, budget)
		defer cancel()
	}
	var out InvokeBatchResult
	payload := InvokeBatchPayload{Invocations: invocations, ReturnPageContent: returnPageContent}
	if err := request(ctx, tab, CmdAgentInvoke, payload, &out); err != nil {
		return out, err
	}
	return out, nil
}

// GetMenu returns the side-menu structure (groups + accessible options) from
// the connected browser. Used by GET /agent?get=menu to surface routes
// without leaking the menu DOM into the page snapshot.
func GetMenu(ctx context.Context, tab string) ([]AgentMenuGroup, error) {
	ctx, cancel := ctxWithDefault(ctx)
	defer cancel()
	var out []AgentMenuGroup
	err := request(ctx, tab, CmdGetMenu, nil, &out)
	if err != nil {
		return out, err
	}
	if err := AttachMenuDescriptions(out); err != nil {
		return nil, err
	}
	return out, nil
}

// Navigate asks the browser to change the SPA route. Used by the `navigate`
// action so agents can move between pages by passing a route from GetMenu.
func Navigate(ctx context.Context, tab, route string) error {
	_, err := NavigateWithPage(ctx, tab, route, false)
	return err
}

func NavigateWithPage(ctx context.Context, tab, route string, returnPageContent bool) (NavigateResult, error) {
	ctx, cancel := ctxWithDefault(ctx)
	defer cancel()
	var out NavigateResult
	err := request(ctx, tab, CmdNavigate, NavigatePayload{Route: route, ReturnPageContent: returnPageContent}, &out)
	return out, err
}
