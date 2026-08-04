# PLAN — Turn-scoped agent stream

Replace the always-on per-tab SSE stream with a stream whose lifetime **is the
turn**: one `POST /agent/turn` opens a streamed response that carries both chat
events and page commands, and closes when the agent finishes. No idle
connections, no reconnect loop.

## Why

`/agent/stream` is not a status channel — it is a bidirectional RPC where the
backend is the caller and the browser is the server. The chat loop's tools
(`get_page`, `get_menu`, `navigate`, `invoke_batch`) all execute in the browser
(`chat_loop.go:195-275` → `commands.ts:228`), so the backend must be able to
push requests into the page mid-turn and await POSTed replies.

None of that requires a connection *between* turns:

- `sendJSON` drops events when no stream exists; the turn still completes and
  persists to Scylla (`chat_ws.go:199-203`).
- `chatSessions` is a `map[TabID]*AgentSession` that already survives
  reconnects (`chat_ws.go:120-141`).
- Screenshots are not chat tools — only the external driver uses them.

The one consumer of an idle stream is the **external HTTP driver** (`POST
/agent`, `GET /agent?get=menu|screenshot`) used by Claude Code / Gemini.
`ResolveTab` (`ws.go:346`) needs an already-connected tab and an external CLI
cannot bootstrap one. That path is preserved behind an explicit opt-in.

## Design

### Transport

`POST /agent/turn?tab=<id>` returns `text/event-stream` framed exactly like
today (`data: <json>\n\n`), so `clientConn.push` and the browser-side frame
shape are unchanged. The browser reads it with `fetch` + `ReadableStream`
instead of `EventSource` (which cannot POST a body).

Message discrimination stays as-is — `ID > 0` is a command awaiting a reply,
`ID == 0` is a chat event (`sse.ts:165`).

Command replies keep going to `POST /agent/in?tab=` on a separate short
request. That asymmetry is required: the turn's response body is one-way.

```
browser                                   backend
   |-- POST /agent/turn {Message,...} ------->|  registers conn, starts loop
   |<-- data: {Type:"agentStatus"} -----------|
   |<-- data: {ID:7,Type:"navigate"} ---------|
   |--- POST /agent/in {ID:7,Type:"result"} ->|  resolves pendingReply
   |<-- data: {Type:"agentReply"} ------------|
   |<-- data: {Type:"turnEnd"} ---------------|  handler returns, body closes
```

### Backend

**`ws.go`**

- Keep `clientConn`, `registerClient`, `request`, `pending`, `/agent/in` and
  `HandleStream` as they are. `request()` resolves by tab, so it needs no
  change at all.
- `handleInbound`: drop the `ChatTypeUserMessage` branch (moves to
  `/agent/turn`). Keep `EventPageContent` (only logs) and `TypeReady` — the
  opt-in dev bridge still sends it on connect.

**`turn.go` (new)**

`HandleTurn(w, r)`:

1. `tab` from `?tab=`, 400 if empty. Require `http.Flusher`.
2. Decode `ChatUserMessage` + new `CompanyID` / `UserID` / `Path` fields from
   the body — these previously came from the stream's query params.
3. `ensureChatSession(tab)`, seeded from the body instead of `lookupClient`.
4. `inFlight.CompareAndSwap` → 409 + `agentError` if a turn is already running.
5. Build a `clientConn`, write SSE headers (including `X-Accel-Buffering: no`),
   flush.
6. **Register only if `lookupClient(tab) == nil`.** If a dev idle stream is
   open, leave it as the command channel and use the turn conn for events
   only — otherwise `registerClient` would push `replaced` at the dev stream
   and make it rotate its tab id.
7. Set the conn as the session's event sink (see below), run the turn in a
   goroutine, pump `cc.send` → response in the handler's select loop.
8. On turn completion: push `{Type:"turnEnd"}`, unregister (if we registered),
   `cc.close()`, return.
9. On `r.Context().Done()` (client gone): cancel the turn context and
   `failAllPending`.

Keepalive ticker stays — a slow LLM call can exceed a proxy idle timeout
mid-turn.

**`chat_ws.go`**

- `AgentSession` gains an event-sink field (`sinkMu sync.Mutex; sink
  *clientConn`) set by `HandleTurn` for the turn's duration.
- `sendJSON` writes to that sink instead of `lookupClient(s.TabID)`. This is
  what decouples chat events from the command channel.
- `onUserMessage` no longer spawns its own goroutine or owns `inFlight` —
  `HandleTurn` does both, and it needs the turn to run to completion before
  it closes the body. Signature becomes a blocking `RunUserMessage(ctx, msg)
  error`.

**Behaviour change — turns abort on disconnect.** Today the turn detaches with
`context.Background()` + 5 min so it finishes even if the user walks away
(`chat_ws.go:164-166`). With a turn-scoped stream that is pointless: the tools
run in the browser, so a dead client means every subsequent tool call blocks
until timeout anyway, and `sendJSON` would drop the reply regardless. The turn
context becomes the request context with the same 5 min cap. The user message
is still persisted before the loop starts, so history stays consistent.

**`main.go`** — mount `/agent/turn` through `corsMiddleware` next to the other
two (`main.go:409`). While there: the "Local-only" comments on lines 400 and
412 are stale — the mux is gated on `!core.Env.IS_SERVERLESS`, so these are
live on the VPS.

### Frontend

**`sse.ts`**

- `startAgentBridge` / `connectAgentStream` stay, but become the **dev-only**
  idle stream for the external driver. Exposed as `__agent.connect()` and
  auto-started only when `?agent=1` or a localStorage flag is set. Nothing
  calls it on boot.
- New `runAgentTurn(message, modelHash, timestamp, modeID, context)`:
  - POST to `/agent/turn?tab=`, body carries the message plus company/user/path.
  - Read `response.body` through a small SSE frame parser (buffer, split on
    `\n\n`, strip `data: `, ignore `: ping`).
  - Reuse the existing `handleStreamMessage` dispatch verbatim: `ID > 0` →
    `runCommand` → `postIn`; otherwise fan out to `chatListeners`.
  - Resolve when the body ends or `turnEnd` arrives; `releaseScreenStream()` in
    a `finally`.
- `isStreamConnected` is no longer meaningful for chat. Keep it exported for
  the dev stream, but the chat widget stops consulting it.

**`AgentChat.svelte`**

- `openPanel`: drop `startAgentBridge()`, keep `ensureHistoryLoaded()`.
- `sendMessage`: drop the 5-second `isStreamConnected` wait loop
  (`AgentChat.svelte:256-275`) — there is nothing to wait for. `await
  runAgentTurn(...)`, and clear `isBusy` in a `finally` so a transport failure
  can't wedge the widget.
- `subscribeAgentChat` and `handleChatEvent` are untouched — `runAgentTurn`
  feeds the same listener set, so all four event cases keep working.
- Failure path: on a rejected `runAgentTurn`, render the existing
  `⚠ No se pudo conectar con el agente.` row.

**`commands.ts`** — no change.

## Risks

- **`agentSections`** (page-builder mode) arrives as a normal event inside the
  turn body; the applier is synchronous in `handleChatEvent`, so nothing races
  with body close.
- **Proxy buffering** — if whatever fronts `genix-api-4.un.pe` buffers POST
  responses, status events batch up at the end. `X-Accel-Buffering: no` covers
  nginx; needs a live check after deploy.
- **HTTP/1.1 connection cap** — a turn holds one of the 6 per-origin slots for
  its duration. Acceptable; a page load doesn't overlap a turn.
- **Two-slot conn ambiguity** — the "register only if absent" rule means that
  with a dev stream open, commands go down the dev stream and events down the
  turn body. Both are dispatched identically by the browser, so this is
  correct, but it is the subtlest part of the change.

## Verification

1. `go build ./...` + `go vet ./...` in `backend/`.
2. `npx svelte-check` — must not add to the 12 pre-existing errors.
3. Manual: load the app, confirm **zero** `/agent/*` requests in the network
   tab while idle.
4. Manual: send a chat message; confirm one `/agent/turn` request that stays
   pending for the turn, N `/agent/in` replies, and that it closes on reply.
5. Manual: a multi-tool turn ("ve a productos y busca X") — status rows appear
   incrementally, navigation happens, reply lands.
6. Manual: close the tab mid-turn; backend logs the cancel, no goroutine leak.
7. Dev driver: `__agent.connect()`, then `curl 'localhost:PORT/agent?get=menu'`
   still resolves the tab.

## Out of scope

- Auth on `/agent/*` (still unauthenticated; pre-existing).
- Reworking the external driver to not need an idle tab.
- Chat history sync between tabs.
