package agent

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"app/agent/routing"
	"app/agent/webpage"
	"app/core"
)

// Chat is the user↔agent channel for the in-app widget. One turn is one
// `POST /agent/turn` (turn.go): the request carries the user message and the
// streamed response carries everything the turn produces — chat
// replies/status/errors AND the page-driving commands the loop's tools issue.
// The browser POSTs command replies back to `/agent/in`, correlated by the
// shared TabID. The stream lives exactly as long as the turn.

// Wire envelope. Mirrors protocol.go style — capitalized field names, no json
// tags. `Type` discriminates the union; `Payload` is decoded per-type.
type chatEnvelope struct {
	Type    string          `json:"Type"`
	Payload json.RawMessage `json:"Payload,omitempty"`
}

// Chat message Types. All are server→browser: the user's message rides the
// `POST /agent/turn` body, not a typed envelope.
const (
	ChatTypeAgentReply    = "agentReply"
	ChatTypeAgentStatus   = "agentStatus"
	ChatTypeAgentError    = "agentError"
	ChatTypeAgentSections = "agentSections" // page-builder edits to apply back into the builder
	// ChatTypeTurnEnd is the last frame of a turn stream. The body closing is
	// already the real signal; this lets the browser tell a finished turn apart
	// from a dropped connection.
	ChatTypeTurnEnd = "turnEnd"
)

type ChatUserMessage struct {
	Message   string
	ModelHash string
	Timestamp int64
	// ModeID is only a compact routing hint; live builder state is fetched after classification.
	ModeID      int
	Surface     routing.SurfaceContext
	AppLanguage routing.Language
}

type ChatAgentReply struct {
	Message   string
	Summary   string
	Timestamp int64
}

type ChatAgentStatus struct {
	State    string // "thinking" | "acting" | "idle"
	Label    string // human-readable progress text, e.g. "Consultando el menú…"
	Step     int
	MaxSteps int
}

type ChatAgentError struct {
	Message string
}

// ChatAgentSections carries the page-builder loop's edited sections back to the
// builder. ModeID tells the frontend how to apply them (replace the selected
// section vs. replace the whole page); Svgs holds the inline SVG bodies the
// turn generated, keyed by sprite id, to merge into the target SectionData.
type ChatAgentSections struct {
	ModeID    int
	Sections  []webpage.SectionEdit
	Svgs      map[string]string
	Message   string
	Summary   string
	Timestamp int64
}

// AgentSession is one chat conversation. There is at most one per browser tab —
// the same TabID identifies both the tab's stream (so the loop can issue tool
// calls through it) and the chat session. The session outlives any single turn:
// it holds the route the agent last knew about and guards against overlapping
// turns on the same tab.
type AgentSession struct {
	CompanyID int32
	UserID    int32
	TabID     string
	// ChannelToken is the wire name of this tab's stream (channel.go). Only the
	// bridge transport needs it; the local one addresses the tab directly.
	ChannelToken string
	// SessionID scopes the persisted history (chat_store.go). It comes from the
	// client, not from this process's clock: under Lambda two turns of the same
	// conversation can land on different execution environments, and a
	// locally-minted id would start the LLM's history from scratch each time.
	SessionID int64

	inFlight atomic.Bool
	// lastMessageTimestamp prevents instant fixed responses from reusing the
	// current user row's millisecond clustering key.
	lastMessageTimestamp atomic.Int64

	// currentRoute is the SPA path the user is on, used to enrich progress
	// labels (e.g. "Leyendo /negocio/productos…"). Re-seeded from each turn
	// request and updated whenever a navigate tool dispatch succeeds.
	routeMu      sync.RWMutex
	currentRoute string
}

// pushEvent delivers one already-framed chat event to the tab's stream. Under
// Lambda the stream belongs to the SSE bridge; otherwise it is the connection
// this process registered in ws.go. Either way a tab with no stream just drops
// the event — the turn still finishes and persists.
func (s *AgentSession) pushEvent(envelopeJSON []byte) error {
	if BridgeEnabled() {
		return bridgePublish(s.ChannelToken, envelopeJSON)
	}
	clientConnection := lookupClient(s.TabID)
	if clientConnection == nil {
		return errors.New("no hay stream conectado para el tab")
	}
	return clientConnection.push(envelopeJSON)
}

// CurrentRoute returns the last route the session knows about. Empty means
// the session hasn't been seeded yet and the agent has never navigated.
func (s *AgentSession) CurrentRoute() string {
	s.routeMu.RLock()
	defer s.routeMu.RUnlock()
	return s.currentRoute
}

func (s *AgentSession) setCurrentRoute(route string) {
	s.routeMu.Lock()
	s.currentRoute = strings.TrimSpace(route)
	s.routeMu.Unlock()
}

var (
	chatSessionsMu sync.RWMutex
	chatSessions   = map[string]*AgentSession{} // keyed by TabID
)

// ensureChatSession returns the tab's session, creating it on first use, and
// re-seeds the mutable context every turn carries. Re-seeding matters: the user
// can switch company or navigate by hand between turns, and a turn request
// always reports the live values. Sessions persist across turns; history lives
// in ScyllaDB regardless.
func ensureChatSession(tab, channelToken string, companyID, userID int32, sessionID int64, path string) *AgentSession {
	chatSessionsMu.Lock()
	s := chatSessions[tab]
	if s == nil {
		s = &AgentSession{TabID: tab}
		chatSessions[tab] = s
		core.Log("agent.chat session created tab::", shortTabID(tab), " company::", companyID, " user::", userID, " session::", sessionID, " path::", path)
	}
	chatSessionsMu.Unlock()

	s.CompanyID = companyID
	s.UserID = userID
	// Re-seeded per turn: switching company changes the channel even though the
	// tab, and therefore this session, stays the same.
	s.ChannelToken = channelToken
	if sessionID > 0 {
		s.SessionID = sessionID
	} else if s.SessionID == 0 {
		// Legacy/malformed client: fall back to a local id so the turn still runs,
		// at the cost of the LLM starting this conversation without history.
		s.SessionID = time.Now().Unix()
		core.Log("agent.chat sin SessionID del cliente, usando local tab::", shortTabID(tab), " session::", s.SessionID)
	}
	if strings.TrimSpace(path) != "" {
		s.setCurrentRoute(path)
	}
	return s
}

// lookupChatSession returns the tab's session without creating one. The bridge
// transport uses it to resolve the channel's company/user identity.
func lookupChatSession(tab string) *AgentSession {
	chatSessionsMu.RLock()
	defer chatSessionsMu.RUnlock()
	return chatSessions[tab]
}

// RunUserMessage runs the agentic loop for one user turn, blocking until the
// turn finishes. HandleTurn calls it on a goroutine and holds the response body
// open for its duration, so unlike the old fire-and-forget POST the caller owns
// both the lifetime and the inFlight guard.
func (s *AgentSession) RunUserMessage(ctx context.Context, msg ChatUserMessage) error {
	text := strings.TrimSpace(msg.Message)
	if text == "" {
		return errors.New("empty message")
	}
	core.Log("agent.chat userMessage tab::", shortTabID(s.TabID), " bytes::", len(text), " model_hash::", msg.ModelHash, " mode::", msg.ModeID, " surface::", msg.Surface.Kind, " page_connected::", IsConnected(s.TabID), " connected_tabs::", strings.Join(shortConnectedTabs(), ","))
	return s.runClassifiedTurn(ctx, msg, text)
}

// sendJSON pushes a chat event down the turn's response stream. A missing sink
// (no turn in flight, or the browser vanished mid-turn) just drops the event —
// the turn still finishes and persists.
func (s *AgentSession) sendJSON(kind string, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		core.Log("agent.chat marshal payload tab::", shortTabID(s.TabID), " err::", err)
		return
	}
	env, err := json.Marshal(chatEnvelope{Type: kind, Payload: body})
	if err != nil {
		core.Log("agent.chat marshal envelope tab::", shortTabID(s.TabID), " err::", err)
		return
	}
	if err := s.pushEvent(env); err != nil {
		core.Log("agent.chat push error tab::", shortTabID(s.TabID), " type::", kind, " err::", err)
		return
	}
	core.Log("agent.chat send tab::", shortTabID(s.TabID), " type::", kind, " payload_bytes::", len(body))
}

func (s *AgentSession) sendError(msg string) {
	s.sendJSON(ChatTypeAgentError, ChatAgentError{Message: msg})
}

// PushStatus and PushReply expose the session's event helpers to the webpage
// agentic loop, which lives in a sub-package and can't reach the unexported
// ones. Together they satisfy webpage.Sink.
func (s *AgentSession) PushStatus(state, label string, step, maxSteps int) {
	s.pushStatus(state, label, step, maxSteps)
}

func (s *AgentSession) PushReply(message, summary string, _ int64) error {
	return s.completeTurn(message, summary, 0)
}

func (s *AgentSession) PushSections(modeID int, sections []webpage.SectionEdit, svgs map[string]string, message, summary string, _ int64) error {
	timestamp, err := saveAgentMessage(s, message, summary, 0)
	if err != nil {
		return err
	}
	s.sendJSON(ChatTypeAgentSections, ChatAgentSections{
		ModeID: modeID, Sections: sections, Svgs: svgs, Message: message, Summary: summary, Timestamp: timestamp,
	})
	return nil
}

func shortTabID(tabID string) string {
	const visibleTail = 6
	tabID = strings.TrimSpace(tabID)
	if len(tabID) <= visibleTail {
		return tabID
	}
	// Keep only the tail so concurrent tab logs stay distinguishable without noisy UUIDs.
	return tabID[len(tabID)-visibleTail:]
}
