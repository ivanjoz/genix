# Plan: Replace agent WebSockets with SSE + POST

Status: **IMPLEMENTED** — unified single-stream design (§3.1) shipped. WS code,
`/ws/*` routes, and the `coder/websocket` dependency removed; `ws.ts` renamed to
`sse.ts`. Backend builds; frontend `svelte-check` clean for the touched files.

## 1. Why

The two agent channels are WebSockets today. We want to drop WS in favor of
**Server-Sent Events (SSE)** for the backend→browser push direction and plain
**POST** requests for the browser→backend direction. Motivations: SSE rides
ordinary HTTP/1.1 (no upgrade, no proxy quirks, auto-reconnect built into
`EventSource`), and POST is the transport the rest of the app already uses, so
auth/CORS/proxy handling is uniform.

## 2. What exists today

Two WS connections, both keyed by a frontend-minted `TabID`:

| Channel | File (BE / FE) | Server→browser | Browser→server |
|---|---|---|---|
| Page bridge `/ws/agent` | `ws.go` / `core/agent/ws.ts` | RPC commands (`{ID,Type,Payload}`), waits for reply | replies `{ID,Type:result\|error}`, unsolicited `ready`, `pageContent` |
| Chat `/ws/agent-chat` | `chat_ws.go` / `core/agent/AgentChat.svelte` | `agentReply`, `agentStatus`, `agentError` (many per turn) | `userMessage` |

Key backend mechanics that must be preserved:

- **Page bridge is request/reply RPC.** `ws.go:request()` mints a per-tab
  monotonic `ID`, stores a `pendingReply` channel in `cc.pending[ID]`, writes
  the command, and blocks on the channel until the browser's reply arrives in
  `handleIncoming`. Public API (`InvokeBatch`, `Navigate`, `GetPageContent`,
  `Screenshot`, `ListComponents`, `GetMenu`) all funnel through `request()`.
- **Per-tab isolation**: each tab has its own `idCount` and `pending` map.
  Last-connection-wins per tab (`registerClient` closes the previous conn and
  fails its pending). Sibling tabs untouched.
- **`WaitForClient` / `clientWake`**: chat turns block until the page bridge
  for the tab is connected.
- **Large replies**: screenshots/pageContent are multi-MB. Today the WS read
  limit is bumped to 32 MB. These travel **browser→backend**, so under the new
  design they ride POST bodies (no SSE size concern — SSE only carries the
  small outbound commands).
- **Unsolicited browser→backend events**: `ready` (connect proof) and
  `pageContent` (test push) arrive with no pending waiter.
- **Chat turn lifecycle**: `onUserMessage` runs `RunTurn` in a goroutine with
  an independent 5-min context, emitting many `agentStatus` pushes then a final
  `agentReply`/`agentError`. `inFlight` bounds one turn per session.

## 3. Target design

### 3.1 Decision: unify the two channels into ONE SSE stream per tab

**Recommended.** Both channels are already keyed by the same `TabID` and the
page bridge SSE is open for the whole page lifetime. Routing chat events down
the same stream removes a whole connection + lifecycle and matches the
"minimize code" rule. The chat widget keeps POSTing user messages and just
subscribes to the shared stream, filtering for chat-typed events.

```
GET  /agent/stream?tab=&company=&user=&path=     # SSE: ALL server→browser push
POST /agent/in?tab=                              # browser→backend: replies, ready,
                                                 #   pageContent, userMessage
```

SSE event envelope (one JSON object per `data:` line):

```
event: message
data: {"ID":7,"Type":"invoke","Payload":{...}}      # page command (has ID)
data: {"Type":"agentStatus","Payload":{...}}         # chat push (no ID)
data: {"Type":"agentReply","Payload":{...}}
```

POST `/agent/in` body — discriminated by `Type`:

```
{"ID":7,"Type":"result","Payload":{...}}   # reply to command 7  -> pending[7]
{"ID":7,"Type":"error","Payload":{...}}    # reply error
{"Type":"ready","Payload":{...}}           # connect proof (log only)
{"Type":"pageContent","Payload":{...}}     # unsolicited test push
{"Type":"userMessage","Payload":{...}}     # starts a chat turn (async)
```

POST returns `200 {}` immediately for fire-and-forget kinds; replies just feed
the pending channel.

> Alternative (not recommended): keep two separate SSE streams
> (`/agent/page/stream` + `/agent/chat/stream`) and two POST endpoints. More
> code, no real benefit since the TabID already unifies them. **Need your call
> on unify vs. keep-separate before I start.**

### 3.2 Backend connection model

Replace `*websocket.Conn` on `clientConn` with an SSE writer abstraction:

```go
type clientConn struct {
    tab       string
    companyID, userID int32
    send      chan []byte   // buffered; SSE goroutine drains -> http.Flusher
    idCount   atomic.Uint64
    pendMu    sync.Mutex
    pending   map[uint64]*pendingReply
}
```

- `GET /agent/stream` handler: set SSE headers (`Content-Type:
  text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`),
  `registerClient(cc)`, then loop: `select { case b := <-cc.send: write+flush;
  case <-ticker.C: write keepalive comment ":\n"; case <-ctx.Done(): return }`.
  Periodic `: ping` comment every ~20s keeps intermediaries from idling out.
- `request()` changes only its write step: instead of `conn.Write`, push the
  marshaled command onto `cc.send` (non-blocking with a write timeout via a
  `select` + `time.After`). Everything else (pending map, reply wait, errors)
  is unchanged.
- `handleIncoming` logic moves into the **POST `/agent/in`** handler: parse the
  same envelope, route `result`/`error` to `pending[ID]`, handle `ready` /
  `pageContent` / `userMessage` as today.
- `registerClient`/`unregisterClient`/`failAllPending`/`WaitForClient`/
  `IsConnected`/`ConnectedTabs`/`ResolveTab` stay almost verbatim — they key on
  the registry, not the transport. "Connected" now means "has a live SSE
  stream". Replaced-stream semantics: new `GET /agent/stream` for an existing
  tab closes the old stream (close its `send` chan / cancel ctx) and fails its
  pending, same as today.

### 3.3 Chat on the unified stream

- `AgentSession` no longer owns a `chatConn`. Its `sendJSON` writes onto the
  same per-tab `cc.send` channel (look up `clientConn` by TabID). So chat and
  page push share one ordered stream.
- `userMessage` arrives via POST `/agent/in` → `LookupChatSession(tab)` (create
  the session lazily on first message, seeded with company/user/path) →
  `onUserMessage` runs the turn in a goroutine exactly as now.
- `inFlight` guard, independent 5-min context, and status/reply emission are
  unchanged in `chat_loop.go` apart from where `sendJSON` writes.

> Open question: chat session currently seeds `company/user/path` from the chat
> WS upgrade query. With unify, those come from the `GET /agent/stream` query
> (page bridge) instead, and `path` updates via navigate as today. Confirm
> that's acceptable (it is, since both already share the tab).

### 3.4 Frontend

- **`core/agent/ws.ts` → `core/agent/sse.ts`** (rename; "minimize code"):
  - Open `new EventSource(`${base}/agent/stream?tab=&company=&user=&path=`)`.
  - `onmessage`: parse envelope. If it has an `ID` → it's a page command: run
    via `runCommand` (unchanged `commands.ts`) and POST the reply to
    `/agent/in`. If `Type` is a chat type → dispatch to the chat widget (see
    below). `EventSource` auto-reconnects, so drop the manual
    `scheduleReconnect` backoff and the `open/close/error` socket plumbing.
  - `ready` and `pageContent` test push become POSTs to `/agent/in`.
  - `sendReply` becomes a `fetch(POST /agent/in, {ID,Type,Payload})`.
- **`AgentChat.svelte`**: drop its own `WebSocket`. Subscribe to the shared
  stream (a tiny pub/sub exported from `sse.ts`, or a `CustomEvent` on
  `window`) for `agentReply`/`agentStatus`/`agentError`. `sendMessage` becomes
  `fetch(POST /agent/in, {Type:"userMessage", Payload:{...}})`. The
  `wsReady`/`waitOpen` handshake is replaced by "stream is open" state from
  `sse.ts`; optimistic row + busy handling stay.
- `agentWsBase()` (`ws→` scheme swap) → `agentHttpBase()` (plain http base);
  SSE/POST use the same origin as the rest of the API.

### 3.5 Routing (`backend/main.go`)

```
- mux.HandleFunc("/ws/agent", agent.HandleWebSocket)
- mux.HandleFunc("/ws/agent-chat", agent.HandleChatWebSocket)
+ mux.HandleFunc("GET /agent/stream", agent.HandleStream)
+ mux.HandleFunc("POST /agent/in", agent.HandleIn)
```

`POST /agent` and `GET /agent` (the external LLM HTTP API in `http.go`) are
**unchanged** — they already use the internal `request()` RPC, which keeps
working once its transport is SSE.

## 4. Files to change

| File | Change |
|---|---|
| `backend/agent/ws.go` | Replace WS conn with SSE `send` chan + `HandleStream`; move `handleIncoming` into POST path; keep registry/pending/request logic |
| `backend/agent/chat_ws.go` | Drop `chatConn`; `sendJSON` writes to shared `send` chan; lazy session creation from POST `userMessage`; add `HandleIn` dispatch (or put in ws.go) |
| `backend/agent/chat_loop.go` | No logic change; verify `sendJSON`/`sendStatus` still compile against new session |
| `backend/main.go` | Swap the two `/ws/*` routes for `GET /agent/stream` + `POST /agent/in` |
| `frontend/core/agent/ws.ts` → `sse.ts` | `EventSource` + POST reply; export stream pub/sub; rename `agentWsBase`→`agentHttpBase` |
| `frontend/core/agent/AgentChat.svelte` | POST user messages; subscribe to shared stream; drop WS lifecycle |
| `backend/agent/HTTP_API.md` | Update "sits on top of WebSocket bridge" wording to SSE |
| `backend/agent/AGENTIC_LOOP_DESIGN.md` | Update transport references if any |
| `go.mod` | Remove `github.com/coder/websocket` once no references remain |

Grep for stragglers: `rg "coder/websocket|/ws/agent|WebSocket|agentWsBase|EventSource"`.

## 5. Risks / edge cases

1. **`http.Flusher` required** — the SSE handler must assert `w.(http.Flusher)`
   and flush after every event; without it events buffer. Verify the prod
   server / any reverse proxy doesn't buffer SSE (disable proxy buffering for
   `/agent/stream`). The dev proxy (`frontend/scripts/proxy-server.js`) needs
   to pass SSE through unbuffered — check it.
2. **`EventSource` is GET-only, no custom headers** — auth must travel as query
   params (as today) or cookies. No regression vs. current WS query-param auth.
3. **Reconnect storms** — `EventSource` auto-reconnects; a flapping stream will
   re-`registerClient` repeatedly. Keep the last-wins close + `failAllPending`
   so in-flight RPCs fail fast instead of hanging.
4. **Ordering** — a single `send` chan per tab preserves ordering of chat
   statuses + page commands; keep one writer goroutine per stream.
5. **Backpressure** — buffered `send` chan; on overflow, drop the stream
   (close + let client reconnect) rather than block `request()`.
6. **Keepalive** — emit `:` comment pings (~20s) so idle streams survive proxy
   timeouts.
7. **Lazy chat session** — first `userMessage` POST may arrive before any
   session exists; create it on demand keyed by tab/company/user/path.

## 6. Suggested step order (each independently testable)

1. Backend: add `HandleStream` + `HandleIn`, refactor `clientConn` to `send`
   chan, route page RPC through it. Keep WS routes temporarily for A/B.
2. Move chat onto the shared stream; lazy session from POST.
3. Frontend: `sse.ts` page bridge via `EventSource` + POST replies.
4. Frontend: `AgentChat.svelte` onto POST + shared stream.
5. Delete WS handlers, `/ws/*` routes, `coder/websocket` dep; update docs.
6. Verify: external `POST /agent` curl flow still drives the page; chat turn
   shows statuses + reply; reconnect after backend restart; multi-tab.

## 7. Questions for you before I start

1. **Unify into one stream (recommended) or keep two separate SSE+POST pairs?**
2. OK to **rename** `ws.ts`→`sse.ts` and `agentWsBase`→`agentHttpBase`, and to
   fully delete the WS code + `coder/websocket` (no back-compat, per AGENTS.md)?
3. Any deployment constraint on SSE (CDN/reverse proxy in front of the prod
   server that might buffer `text/event-stream`)?
