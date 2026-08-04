package agent

// Turn-scoped agent stream. One user turn is one `POST /agent/turn` whose
// response body stays open for the turn's duration and carries everything the
// turn produces:
//
//   - chat events (agentStatus / agentReply / agentError / agentSections),
//     which are `ID: 0` frames the widget consumes;
//   - page commands (navigate / agent.invoke / getPageContent / getMenu),
//     which are `ID > 0` frames the browser executes, replying on
//     `POST /agent/in?tab=` — the response body is one-way, so replies need
//     their own short request.
//
// Framing is identical to the page-bridge stream in ws.go (`data: <json>\n\n`),
// so the browser parses both the same way; only the lifetime differs. When the
// turn ends the body closes and nothing remains connected — this is what keeps
// idle tabs from holding a permanent stream open.

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"app/core"
)

// turnStreamTimeout caps a single turn end-to-end. Matches the old detached
// turn's budget: long enough for a dozen LLM round-trips, short enough that a
// wedged turn can't hold a connection and the session's inFlight guard forever.
const turnStreamTimeout = 5 * time.Minute

// disconnectGrace is how long HandleTurn waits for a cancelled turn to unwind
// after the client vanishes, so inFlight isn't released while the loop is still
// running and a retry can't overlap it.
const disconnectGrace = 10 * time.Second

// TurnRequest is the `POST /agent/turn` body. Message/ModelHash/Timestamp/
// ModeID/Context mirror ChatUserMessage; CompanyID/UserID/Path used to ride the
// page-bridge stream's query params and now come per-turn, which also means
// they reflect manual navigation and company switches between turns.
type TurnRequest struct {
	Message   string
	ModelHash string
	Timestamp int64
	ModeID    int
	Context   string

	CompanyID int32
	UserID    int32
	Path      string
}

// HandleTurn runs one chat turn and streams it back. Mounted at
// `POST /agent/turn`; the tab is identified by `?tab=<TabID>`.
func HandleTurn(w http.ResponseWriter, r *http.Request) {
	tab := strings.TrimSpace(r.URL.Query().Get("tab"))
	if tab == "" {
		writeJSONError(w, http.StatusBadRequest, "missing ?tab=")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	var req TurnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if strings.TrimSpace(req.Message) == "" {
		writeJSONError(w, http.StatusBadRequest, "empty message")
		return
	}

	s := ensureChatSession(tab, req.CompanyID, req.UserID, req.Path)
	// One turn per tab. Rejected before any streaming headers go out so the
	// browser sees a plain JSON 409 rather than an empty event stream.
	if !s.inFlight.CompareAndSwap(false, true) {
		core.Log("agent.turn busy tab::", shortTabID(tab), " incoming_bytes::", len(req.Message))
		writeJSONError(w, http.StatusConflict, "a previous turn is still running")
		return
	}
	defer s.inFlight.Store(false)

	cc := &clientConn{
		tab:       tab,
		companyID: req.CompanyID,
		userID:    req.UserID,
		path:      req.Path,
		send:      make(chan []byte, 64),
		closed:    make(chan struct{}),
		pending:   map[uint64]*pendingReply{},
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// Defeat reverse-proxy response buffering, otherwise every status event
	// batches up and lands at once when the turn ends.
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	// Register as the tab's command channel only when nothing else holds it.
	// A dev page-bridge stream (see HandleStream) may already be open for the
	// external HTTP driver; registerClient would evict it with a "replaced"
	// notice and make it rotate its tab id. Leaving it in place is harmless —
	// the browser executes commands identically whichever stream delivers
	// them, and replies come back over /agent/in either way.
	registered := false
	if lookupClient(tab) == nil {
		registerClient(cc)
		registered = true
	}
	s.setSink(cc)
	core.Log("agent.turn start tab::", shortTabID(tab), " company::", cc.companyID, " user::", cc.userID, " path::", cc.path, " mode::", req.ModeID, " owns_command_channel::", registered)

	defer func() {
		s.clearSink(cc)
		if registered {
			unregisterClient(cc)
		}
		cc.close()
		core.Log("agent.turn end tab::", shortTabID(tab))
	}()

	runCtx, cancel := context.WithTimeout(r.Context(), turnStreamTimeout)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := s.RunUserMessage(runCtx, ChatUserMessage{
			Message:   req.Message,
			ModelHash: req.ModelHash,
			Timestamp: req.Timestamp,
			ModeID:    req.ModeID,
			Context:   req.Context,
		}); err != nil {
			core.Log("agent.turn RunUserMessage error tab::", shortTabID(tab), " err::", err)
			s.sendError(err.Error())
		}
		// Queued, not written directly: the pump below owns the ResponseWriter.
		_ = cc.push([]byte(`{"Type":"` + ChatTypeTurnEnd + `"}`))
	}()

	// Keepalive still matters inside a turn — a slow LLM call can idle the
	// connection past a proxy's read timeout between two status events.
	keepalive := time.NewTicker(20 * time.Second)
	defer keepalive.Stop()

	for {
		select {
		case <-r.Context().Done():
			// Client vanished. The loop's tools execute in the browser, so
			// there is nothing left to run and nowhere to deliver a reply;
			// cancel rather than burn tokens on a turn no one will see. The
			// user message is already persisted, so history stays consistent.
			core.Log("agent.turn client gone tab::", shortTabID(tab))
			cancel()
			cc.close()
			select {
			case <-done:
			case <-time.After(disconnectGrace):
				core.Log("agent.turn unwind timeout tab::", shortTabID(tab))
			}
			return

		case <-done:
			// Drain whatever the turn queued right before finishing (its final
			// reply and the turnEnd marker) before closing the body.
			for {
				select {
				case frame := <-cc.send:
					if _, err := w.Write(frame); err != nil {
						return
					}
					flusher.Flush()
				default:
					return
				}
			}

		case frame := <-cc.send:
			if _, err := w.Write(frame); err != nil {
				core.Log("agent.turn write end tab::", shortTabID(tab), " err::", err)
				cancel()
				return
			}
			flusher.Flush()

		case <-keepalive.C:
			if _, err := io.WriteString(w, ": ping\n\n"); err != nil {
				cancel()
				return
			}
			flusher.Flush()
		}
	}
}
