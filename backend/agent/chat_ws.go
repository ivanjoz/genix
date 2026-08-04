package agent

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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
	// ModeID is the agent mode the user is in (1 ask, 2 build page, 3 edit
	// section); Context carries mode-specific payload such as the builder's
	// sections serialized to HTML (whole page for mode 2, selected section for 3).
	ModeID  int
	Context string
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
// the same TabID identifies both the turn stream (so the loop can issue tool
// calls through it) and the chat session. The session outlives any single turn:
// it holds the route the agent last knew about and guards against overlapping
// turns on the same tab.
type AgentSession struct {
	CompanyID int32
	UserID    int32
	TabID     string
	SessionID int64 // unix seconds when the chat session was created

	inFlight atomic.Bool

	// sink is the turn stream chat events are written to, installed by
	// HandleTurn for the turn's duration. Deliberately not resolved through the
	// `clients` map: with a dev page-bridge stream open, commands ride that
	// stream while events must still go down this turn's response body.
	sinkMu sync.Mutex
	sink   *clientConn

	// currentRoute is the SPA path the user is on, used to enrich progress
	// labels (e.g. "Leyendo /negocio/productos…"). Re-seeded from each turn
	// request and updated whenever a navigate tool dispatch succeeds.
	routeMu      sync.RWMutex
	currentRoute string
}

// setSink installs cc as the destination for this turn's chat events.
func (s *AgentSession) setSink(cc *clientConn) {
	s.sinkMu.Lock()
	s.sink = cc
	s.sinkMu.Unlock()
}

// clearSink removes cc, but only if it is still the installed sink — a turn
// that unwinds late must not detach its successor's stream.
func (s *AgentSession) clearSink(cc *clientConn) {
	s.sinkMu.Lock()
	if s.sink == cc {
		s.sink = nil
	}
	s.sinkMu.Unlock()
}

func (s *AgentSession) eventSink() *clientConn {
	s.sinkMu.Lock()
	defer s.sinkMu.Unlock()
	return s.sink
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
// can switch company or navigate by hand between turns, and unlike the old
// page-load stream a turn request always reports the live values. Sessions
// persist across turns; history lives in ScyllaDB regardless.
func ensureChatSession(tab string, companyID, userID int32, path string) *AgentSession {
	chatSessionsMu.Lock()
	s := chatSessions[tab]
	if s == nil {
		s = &AgentSession{
			TabID:     tab,
			SessionID: time.Now().Unix(),
		}
		chatSessions[tab] = s
		core.Log("agent.chat session created tab::", shortTabID(tab), " company::", companyID, " user::", userID, " path::", path)
	}
	chatSessionsMu.Unlock()

	s.CompanyID = companyID
	s.UserID = userID
	if strings.TrimSpace(path) != "" {
		s.setCurrentRoute(path)
	}
	return s
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
	core.Log("agent.chat userMessage tab::", shortTabID(s.TabID), " bytes::", len(text), " model_hash::", msg.ModelHash, " mode::", msg.ModeID, " context_bytes::", len(msg.Context), " page_connected::", IsConnected(s.TabID), " connected_tabs::", strings.Join(shortConnectedTabs(), ","))

	// Route by mode: the builder's "build page" / "edit section" modes run the
	// page-builder loop (which needs msg.Context — the sections as HTML);
	// everything else (mode 1 "ask", and any unknown mode) runs the default
	// chat loop.
	switch msg.ModeID {
	case webpage.ModeBuildPage, webpage.ModeEditSection:
		return webpage.RunTurn(ctx, s, msg.ModeID, text, msg.ModelHash, msg.Context)
	default:
		return s.RunTurn(ctx, text, msg.ModelHash)
	}
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
	cc := s.eventSink()
	if cc == nil {
		core.Log("agent.chat no stream tab::", shortTabID(s.TabID), " type::", kind)
		return
	}
	if err := cc.push(env); err != nil {
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

func (s *AgentSession) PushReply(message, summary string, timestamp int64) {
	s.sendJSON(ChatTypeAgentReply, ChatAgentReply{Message: message, Summary: summary, Timestamp: timestamp})
}

func (s *AgentSession) PushSections(modeID int, sections []webpage.SectionEdit, svgs map[string]string, message, summary string, timestamp int64) {
	s.sendJSON(ChatTypeAgentSections, ChatAgentSections{
		ModeID: modeID, Sections: sections, Svgs: svgs, Message: message, Summary: summary, Timestamp: timestamp,
	})
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
