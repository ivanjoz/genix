package agent

import "encoding/json"

// JSON wire protocol shared with the local frontend client.
// The backend acts as the agent: it issues commands; the browser executes them
// and replies with a result keyed by the same id.
// All fields are emitted with their Go names (capitalized) — no json tags.

type Message struct {
	ID      uint64
	Type    string
	Payload any
}

// Command types issued by the backend.
const (
	CmdGetPageContent = "getPageContent"
	// CmdScreenshot renders the page DOM via modern-screenshot — no permission
	// prompt, default capture for public users.
	CmdScreenshot = "screenshot"
	// CmdScreenshotReal captures pixel-real screen via getDisplayMedia. Needs
	// user permission and shows the browser's "is sharing your screen" banner.
	CmdScreenshotReal = "screenshotReal"
	CmdAgentList      = "agentList"
	CmdAgentDescribe  = "agentDescribe"
	// CmdAgentInvoke calls a method on a registered handle. The browser dispatches
	// to handle[Method](...Args). See frontend/ui-components/AGENTIC_COMPONENTS.md
	// for the supported method names per component type.
	CmdAgentInvoke = "agent.invoke"
	// CmdGetMenu returns the side-menu structure (groups + options with route)
	// the user has access to. Replaces the per-option agent handles that used
	// to appear in the HTML snapshot.
	CmdGetMenu = "getMenu"
	// CmdNavigate triggers a client-side route change in the browser. Used by
	// the `navigate` action so the agent can move between pages without
	// addressing a specific menu handle.
	CmdNavigate = "navigate"
	// CmdGetAgentContext retrieves builder state only after the global classifier
	// has selected and validated a live context scope.
	CmdGetAgentContext = "getAgentContext"
)

// Reply types sent by the frontend.
const (
	TypeResult = "result"
	TypeError  = "error"
	TypeReady  = "ready" // sent by client right after connect
)

// TypeBridgeReady is the first frame the server sends on a freshly opened
// stream, once the tab is routable. Both transports emit it — this backend in
// HandleStream and the SSE bridge in its own /sse handler — so the client can
// wait for it before starting a turn regardless of which one it connected to.
const TypeBridgeReady = "bridgeReady"

// Unsolicited event types pushed by the frontend (no matching request).
const (
	EventPageContent = "pageContent"
)

type AgentComponentInfo struct {
	ID      int
	Type    string
	Label   string
	Methods []string
	// Options is the inline option list for a Select handle whose total option
	// count is small enough to embed directly in the HTML snapshot (currently
	// ≤12). Frontend probes via getOptions(13) and only attaches when the
	// list is exhaustive. nil/empty for every other handle type.
	Options []AgentOption `json:",omitempty"`
}

type PageContent struct {
	Components []AgentComponentInfo `json:",omitempty"`
	HTML       string
}

type AgentOption struct {
	ID    any
	Value any
}

type ScreenshotResult struct {
	MIME   string
	Base64 string
	Width  int
	Height int
}

// InvokePayload calls handle[Method] with the given Args on the browser side.
// The WS `agent.invoke` command carries a `[]InvokePayload` so the browser can
// execute the full batch sequentially (with its own 250ms gap between each)
// and return one result per invocation in a single round-trip.
type InvokePayload struct {
	HandleID int
	Method   string
	Args     []any
}

type InvokeBatchPayload struct {
	Invocations       []InvokePayload
	ReturnPageContent bool
}

type InvokeBatchResult struct {
	Results []InvocationResult
	Page    *PageContent `json:",omitempty"`
}

// InvocationResult is one entry in the reply array for the batched
// `agent.invoke` command. Also reused as the HTTP-level result element so
// external callers see the same shape the WS protocol uses.
type InvocationResult struct {
	OK    bool
	Value json.RawMessage `json:",omitempty"`
	Error string          `json:",omitempty"`
}

type AgentListFilter struct {
	Type  string
	Label string
}

// AgentMenuOption is one entry inside a side-menu group. Route is the SPA
// path the agent passes to the `navigate` action.
type AgentMenuOption struct {
	Name        string
	Route       string
	Description string `json:",omitempty"`
}

// AgentMenuGroup is a side-menu section (CONFIGURACIÓN, NEGOCIO, …) with the
// options the user has access to.
type AgentMenuGroup struct {
	ID      int
	Name    string
	Options []AgentMenuOption
}

// NavigatePayload is the body sent to the browser for CmdNavigate.
type NavigatePayload struct {
	Route             string
	ReturnPageContent bool
}

type NavigateResult struct {
	Route string
	Page  *PageContent `json:",omitempty"`
}

type AgentContextPayload struct {
	Scope string
}

type AgentContextResult struct {
	SurfaceKind       string
	PageID            string
	Route             string
	Scope             string
	SelectedSectionID string `json:",omitempty"`
	Content           string
}
