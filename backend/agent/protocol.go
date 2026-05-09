package agent

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
	CmdScreenshot     = "screenshot"
	CmdAgentList      = "agentList"
	CmdAgentDescribe  = "agentDescribe"
	// CmdAgentInvoke calls a method on a registered handle. The browser dispatches
	// to handle[Method](...Args). See frontend/ui-components/AGENTIC_COMPONENTS.md
	// for the supported method names per component type.
	CmdAgentInvoke = "agent.invoke"
)

// Reply types sent by the frontend.
const (
	TypeResult = "result"
	TypeError  = "error"
	TypeReady  = "ready" // sent by client right after connect
)

// Unsolicited event types pushed by the frontend (no matching request).
const (
	EventPageContent = "pageContent"
)

type AgentComponentInfo struct {
	ID      int
	Type    string
	Label   string
	Methods []string
}

type PageContent struct {
	Components []AgentComponentInfo
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
type InvokePayload struct {
	HandleID int
	Method   string
	Args     []any
}

type AgentListFilter struct {
	Type  string
	Label string
}
