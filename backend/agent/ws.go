package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"app/core"

	"github.com/coder/websocket"
)

// Single-client local-only websocket bridge between the backend (the agent)
// and one browser tab. Last connection wins.

type pendingReply struct {
	ch chan replyEnvelope
}

type replyEnvelope struct {
	kind    string // TypeResult | TypeError
	payload json.RawMessage
}

type clientConn struct {
	conn   *websocket.Conn
	writeM sync.Mutex
}

var (
	currentMu     sync.RWMutex
	currentClient *clientConn
	clientReady   = make(chan struct{}, 1) // signals "fresh client connected"

	idCounter atomic.Uint64

	pendingMu sync.Mutex
	pending   = map[uint64]*pendingReply{}
)

// HandleWebSocket upgrades the HTTP request and runs the read loop for the
// browser-side agent. Designed to be mounted at /ws/agent.
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Accept any origin: this server only listens on localhost during dev.
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		core.Log("agent.ws accept error::", err)
		return
	}
	// Disable read size cap so screenshot frames (~MBs) survive.
	conn.SetReadLimit(32 * 1024 * 1024)

	cc := &clientConn{conn: conn}
	swapClient(cc)
	core.Log("agent.ws client connected from", r.RemoteAddr)

	defer func() {
		clearClient(cc)
		_ = conn.Close(websocket.StatusNormalClosure, "bye")
		core.Log("agent.ws client disconnected")
	}()

	ctx := r.Context()
	for {
		mt, data, err := conn.Read(ctx)
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				core.Log("agent.ws read end::", err)
			}
			return
		}
		if mt != websocket.MessageText {
			continue
		}
		handleIncoming(data)
	}
}

// swapClient replaces the active client; existing pending requests are
// failed so the caller doesn't block forever waiting on a tab that's gone.
func swapClient(cc *clientConn) {
	currentMu.Lock()
	prev := currentClient
	currentClient = cc
	currentMu.Unlock()
	if prev != nil {
		_ = prev.conn.Close(websocket.StatusGoingAway, "replaced")
	}
	failAllPending(errors.New("client reconnected"))
	select {
	case clientReady <- struct{}{}:
	default:
	}
}

func clearClient(cc *clientConn) {
	currentMu.Lock()
	if currentClient == cc {
		currentClient = nil
	}
	currentMu.Unlock()
	failAllPending(errors.New("client disconnected"))
}

func failAllPending(err error) {
	pendingMu.Lock()
	defer pendingMu.Unlock()
	errPayload, _ := json.Marshal(struct{ Message string }{err.Error()})
	for id, p := range pending {
		select {
		case p.ch <- replyEnvelope{kind: TypeError, payload: errPayload}:
		default:
		}
		delete(pending, id)
	}
}

func handleIncoming(raw []byte) {
	// Parse only the envelope; payload stays as RawMessage to be decoded by caller.
	var env struct {
		ID      uint64
		Type    string
		Payload json.RawMessage
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		core.Log("agent.ws bad json::", err, string(raw))
		return
	}

	if env.Type == TypeReady {
		core.Log("agent.ws client signaled ready")
		return
	}

	// Unsolicited events (no matching request) — handled here, not via pending map.
	if env.Type == EventPageContent {
		var page PageContent
		if err := json.Unmarshal(env.Payload, &page); err != nil {
			core.Log("agent.ws pageContent decode error::", err)
			return
		}
		core.Log("agent.ws pageContent received, components::", len(page.Components), "html bytes::", len(page.HTML))
		fmt.Println("---- agent pageContent components ----")
		for _, c := range page.Components {
			fmt.Printf("  [%d] %s label=%q methods=%v\n", c.ID, c.Type, c.Label, c.Methods)
		}
		fmt.Println("---- end agent pageContent components ----")
		fmt.Println("---- raw HTML data-id occurrences ----")
		fmt.Println("  data-id count:", strings.Count(page.HTML, "data-id"))
		if idx := strings.Index(page.HTML, "data-id"); idx >= 0 {
			end := idx + 80
			if end > len(page.HTML) {
				end = len(page.HTML)
			}
			fmt.Printf("  first occurrence: %q\n", page.HTML[idx:end])
		}
		fmt.Println("---- end raw HTML data-id occurrences ----")
		parsed, err := ParsePageHTML(page.HTML, page.Components)
		if err != nil {
			core.Log("agent.ws pageContent parse error::", err)
			return
		}
		fmt.Println("---- agent pageContent HTML ----")
		fmt.Println(parsed)
		fmt.Println("---- end agent pageContent HTML ----")
		return
	}

	if env.ID == 0 {
		core.Log("agent.ws message without id, dropping::", env.Type)
		return
	}

	pendingMu.Lock()
	p, ok := pending[env.ID]
	if ok {
		delete(pending, env.ID)
	}
	pendingMu.Unlock()

	if !ok {
		core.Log("agent.ws reply with no waiter::", env.ID, env.Type)
		return
	}
	select {
	case p.ch <- replyEnvelope{kind: env.Type, payload: env.Payload}:
	default:
	}
}

// IsConnected reports whether there is an active browser client.
func IsConnected() bool {
	currentMu.RLock()
	defer currentMu.RUnlock()
	return currentClient != nil
}

// WaitForClient blocks until a client is connected or ctx is done.
func WaitForClient(ctx context.Context) error {
	if IsConnected() {
		return nil
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-clientReady:
			if IsConnected() {
				return nil
			}
		}
	}
}

// request is the low-level RPC: send a command, wait for the reply.
// The reply's payload is decoded into `out` on success.
func request(ctx context.Context, cmdType string, payload any, out any) error {
	currentMu.RLock()
	cc := currentClient
	currentMu.RUnlock()
	if cc == nil {
		return errors.New("no agent client connected")
	}

	id := idCounter.Add(1)
	msg := Message{ID: id, Type: cmdType, Payload: payload}
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal command: %w", err)
	}

	p := &pendingReply{ch: make(chan replyEnvelope, 1)}
	pendingMu.Lock()
	pending[id] = p
	pendingMu.Unlock()

	cleanup := func() {
		pendingMu.Lock()
		delete(pending, id)
		pendingMu.Unlock()
	}

	cc.writeM.Lock()
	writeCtx, cancelWrite := context.WithTimeout(ctx, 10*time.Second)
	err = cc.conn.Write(writeCtx, websocket.MessageText, data)
	cancelWrite()
	cc.writeM.Unlock()
	if err != nil {
		cleanup()
		return fmt.Errorf("ws write: %w", err)
	}

	select {
	case <-ctx.Done():
		cleanup()
		return ctx.Err()
	case env := <-p.ch:
		if env.kind == TypeError {
			var e struct {
				Message string
			}
			_ = json.Unmarshal(env.payload, &e)
			if e.Message == "" {
				e.Message = "agent error"
			}
			return errors.New(e.Message)
		}
		if out != nil && len(env.payload) > 0 {
			if err := json.Unmarshal(env.payload, out); err != nil {
				return fmt.Errorf("decode payload: %w", err)
			}
		}
		return nil
	}
}

// Drain ensures Reader/Writer are not leaked; coder/websocket auto-handles
// pings, but we keep this helper for documentation/symmetry.
var _ = io.Discard
