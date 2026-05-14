package agent

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"app/core"

	"github.com/coder/websocket"
)

// /ws/agent-chat is the user↔agent chat channel for the in-app widget. It is
// distinct from /ws/agent: the latter drives the page (browser executes
// backend-issued commands); this one carries user prompts and agent replies
// in the opposite direction. The chat session never talks to the LLM directly
// from the read loop — RunTurn will (in step 4) call OpenRouter and use the
// existing /ws/agent client (looked up by the shared TabID) when it needs to
// act on the page.
//
// Auth at step 3: the upgrade trusts `?company=&user=&tab=` query params.
// Local-only HTTP listener so this is safe for development; real
// session-cookie or token validation lands in a later step.

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

// AgentSession is one chat WS conversation. There is at most one per browser
// tab — the same TabID identifies both the page-WS client (so the future loop
// can issue tool calls through it) and the chat-WS session.
type AgentSession struct {
	CompanyID int32
	UserID    int32
	TabID     string
	SessionID int64 // SUnixTime when the chat connection was opened

	chatConn *websocket.Conn
	writeM   sync.Mutex
	inFlight atomic.Bool

	// currentRoute is the SPA path the user is on, used to enrich progress
	// labels (e.g. "Leyendo /negocio/productos…"). Seeded from the `path`
	// query param on the chat WS upgrade and updated whenever a navigate
	// tool dispatch succeeds. Manual user navigation between turns is NOT
	// tracked — accuracy is best-effort.
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

// HandleChatWebSocket upgrades the request and runs the chat read loop. The
// frontend widget connects to this; the page WS (/ws/agent) is a separate
// connection from the same tab sharing the same TabID.
func HandleChatWebSocket(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	tab := strings.TrimSpace(q.Get("tab"))
	if tab == "" {
		http.Error(w, "missing ?tab=", http.StatusBadRequest)
		return
	}
	companyID := atoi32(q.Get("company"))
	userID := atoi32(q.Get("user"))
	initialPath := strings.TrimSpace(q.Get("path"))

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		core.Log("agent.chat-ws accept error::", err)
		return
	}

	s := &AgentSession{
		CompanyID:    companyID,
		UserID:       userID,
		TabID:        tab,
		SessionID:    time.Now().Unix(),
		chatConn:     conn,
		currentRoute: initialPath,
	}
	registerChatSession(s)
	core.Log("agent.chat-ws connected tab::", tab, " company::", companyID, " user::", userID, " path::", initialPath)

	defer func() {
		unregisterChatSession(s)
		_ = conn.Close(websocket.StatusNormalClosure, "bye")
		core.Log("agent.chat-ws disconnected tab::", tab)
	}()

	ctx := r.Context()
	for {
		mt, data, err := conn.Read(ctx)
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				core.Log("agent.chat-ws read end tab::", tab, " err::", err)
			}
			return
		}
		if mt != websocket.MessageText {
			continue
		}
		s.handleIncoming(ctx, data)
	}
}

func registerChatSession(s *AgentSession) {
	chatSessionsMu.Lock()
	prev := chatSessions[s.TabID]
	chatSessions[s.TabID] = s
	chatSessionsMu.Unlock()
	if prev != nil {
		_ = prev.chatConn.Close(websocket.StatusGoingAway, "replaced")
	}
}

func unregisterChatSession(s *AgentSession) {
	chatSessionsMu.Lock()
	if chatSessions[s.TabID] == s {
		delete(chatSessions, s.TabID)
	}
	chatSessionsMu.Unlock()
}

// LookupChatSession returns the AgentSession for tab, or nil if none open.
// Reserved for future RunTurn dispatch — unused in step 3.
func LookupChatSession(tab string) *AgentSession {
	chatSessionsMu.RLock()
	defer chatSessionsMu.RUnlock()
	return chatSessions[tab]
}

func (s *AgentSession) handleIncoming(ctx context.Context, raw []byte) {
	var env chatEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		core.Log("agent.chat-ws bad json tab::", s.TabID, " err::", err)
		return
	}
	switch env.Type {
	case ChatTypeUserMessage:
		var msg ChatUserMessage
		if err := json.Unmarshal(env.Payload, &msg); err != nil {
			s.sendError("invalid userMessage payload: " + err.Error())
			return
		}
		s.onUserMessage(ctx, msg)
	default:
		core.Log("agent.chat-ws unknown type tab::", s.TabID, " type::", env.Type)
	}
}

// onUserMessage runs the agentic loop for one user turn. Decoupled from the
// read loop via a goroutine so a slow OpenRouter call doesn't block the
// connection from receiving pings/close frames. Concurrency is bounded per
// session by inFlight (second user message while still working → agentError).
func (s *AgentSession) onUserMessage(_ context.Context, msg ChatUserMessage) {
	text := strings.TrimSpace(msg.Message)
	if text == "" {
		s.sendError("empty message")
		return
	}
	if !s.inFlight.CompareAndSwap(false, true) {
		s.sendError("a previous turn is still running")
		return
	}
	core.Log("agent.chat-ws userMessage tab::", s.TabID, " bytes::", len(text))

	go func() {
		defer s.inFlight.Store(false)
		// Independent context: the chat read loop's request context closes when
		// the WS disconnects, which would abort an in-flight OpenRouter call.
		// We'd rather let the turn finish and persist; the user just won't see
		// the reply if they walked away. 5 min hard cap as a safety net.
		runCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := s.RunTurn(runCtx, text); err != nil {
			core.Log("agent.chat-ws RunTurn error tab::", s.TabID, " err::", err)
			s.sendError(err.Error())
		}
	}()
}

func (s *AgentSession) sendJSON(kind string, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		core.Log("agent.chat-ws marshal payload tab::", s.TabID, " err::", err)
		return
	}
	env, err := json.Marshal(chatEnvelope{Type: kind, Payload: body})
	if err != nil {
		core.Log("agent.chat-ws marshal envelope tab::", s.TabID, " err::", err)
		return
	}
	s.writeM.Lock()
	defer s.writeM.Unlock()
	writeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.chatConn.Write(writeCtx, websocket.MessageText, env); err != nil {
		core.Log("agent.chat-ws write error tab::", s.TabID, " err::", err)
	}
}

func (s *AgentSession) sendError(msg string) {
	s.sendJSON(ChatTypeAgentError, ChatAgentError{Message: msg})
}
