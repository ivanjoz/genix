package agent

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"app/core"
)

// Chat is the user↔agent channel for the in-app widget. It shares the per-tab
// SSE+POST bridge (ws.go): user messages arrive on `POST /agent/in` with
// `Type: userMessage`, and agent replies/status/errors are pushed back down
// the same tab's SSE stream the page-driving commands use. There is no
// separate chat connection — the shared TabID unifies both.

// Wire envelope. Mirrors protocol.go style — capitalized field names, no json
// tags. `Type` discriminates the union; `Payload` is decoded per-type.
type chatEnvelope struct {
	Type    string          `json:"Type"`
	Payload json.RawMessage `json:"Payload,omitempty"`
}

// Chat message Types.
const (
	ChatTypeUserMessage = "userMessage"
	ChatTypeAgentReply  = "agentReply"
	ChatTypeAgentStatus = "agentStatus"
	ChatTypeAgentError  = "agentError"
)

type ChatUserMessage struct {
	Message   string
	ModelHash string
	Timestamp int64
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

// AgentSession is one chat conversation. There is at most one per browser tab —
// the same TabID identifies both the page-bridge stream (so the loop can issue
// tool calls through it) and the chat session.
type AgentSession struct {
	CompanyID int32
	UserID    int32
	TabID     string
	SessionID int64 // unix seconds when the chat session was created

	inFlight atomic.Bool

	// currentRoute is the SPA path the user is on, used to enrich progress
	// labels (e.g. "Leyendo /negocio/productos…"). Seeded from the tab's
	// stream path and updated whenever a navigate tool dispatch succeeds.
	// Manual user navigation between turns is NOT tracked — best-effort.
	routeMu      sync.RWMutex
	currentRoute string
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

// ensureChatSession returns the tab's session, creating it on first use. The
// session is seeded with company/user/path from the tab's live stream (opened
// at page load) so the first status label can name the current route. Sessions
// persist across stream reconnects; history lives in ScyllaDB regardless.
func ensureChatSession(tab string) *AgentSession {
	chatSessionsMu.Lock()
	defer chatSessionsMu.Unlock()
	if s := chatSessions[tab]; s != nil {
		return s
	}
	var companyID, userID int32
	var path string
	if cc := lookupClient(tab); cc != nil {
		companyID, userID, path = cc.companyID, cc.userID, cc.path
	}
	s := &AgentSession{
		CompanyID:    companyID,
		UserID:       userID,
		TabID:        tab,
		SessionID:    time.Now().Unix(),
		currentRoute: path,
	}
	chatSessions[tab] = s
	core.Log("agent.chat session created tab::", shortTabID(tab), " company::", companyID, " user::", userID, " path::", path)
	return s
}

// onUserMessage runs the agentic loop for one user turn. Decoupled from the
// inbound POST via a goroutine so the POST returns immediately and a slow
// OpenRouter call doesn't hold the request open. Concurrency is bounded per
// session by inFlight (second user message while still working → agentError).
func (s *AgentSession) onUserMessage(_ context.Context, msg ChatUserMessage) {
	text := strings.TrimSpace(msg.Message)
	if text == "" {
		s.sendError("empty message")
		return
	}
	if !s.inFlight.CompareAndSwap(false, true) {
		core.Log("agent.chat busy tab::", shortTabID(s.TabID), " incoming_bytes::", len(text))
		s.sendError("a previous turn is still running")
		return
	}
	core.Log("agent.chat userMessage tab::", shortTabID(s.TabID), " bytes::", len(text), " model_hash::", msg.ModelHash, " page_connected::", IsConnected(s.TabID), " connected_tabs::", strings.Join(shortConnectedTabs(), ","))

	go func() {
		defer s.inFlight.Store(false)
		// Independent context: the inbound POST returns immediately, so its
		// request context is already gone. We let the turn finish and persist;
		// the user just won't see the reply if they walked away. 5 min hard cap.
		runCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := s.RunTurn(runCtx, text, msg.ModelHash); err != nil {
			core.Log("agent.chat RunTurn error tab::", shortTabID(s.TabID), " err::", err)
			s.sendError(err.Error())
		}
	}()
}

// sendJSON pushes a chat event down the tab's shared SSE stream. A missing
// stream (browser closed/reloaded mid-turn) just drops the event — the turn
// still finishes and persists.
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
	cc := lookupClient(s.TabID)
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

func shortTabID(tabID string) string {
	const visibleTail = 6
	tabID = strings.TrimSpace(tabID)
	if len(tabID) <= visibleTail {
		return tabID
	}
	// Keep only the tail so concurrent tab logs stay distinguishable without noisy UUIDs.
	return tabID[len(tabID)-visibleTail:]
}
