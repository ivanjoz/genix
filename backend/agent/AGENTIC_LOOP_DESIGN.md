# Genix Agent — Design (Backend-Hosted, OpenRouter-Powered)

Status: **draft for review** — open questions at the bottom. Do not implement
until the user signs off on each section.

> **Transport note (superseded):** this doc describes the original WebSocket
> design (`/ws/agent`, `/ws/agent-chat`). The transport has since been replaced
> by SSE + POST — a single per-tab SSE stream (`GET /agent/stream`) carries all
> server→browser traffic (page commands *and* chat events), and `POST /agent/in`
> carries all browser→backend traffic (command replies, `ready`/`pageContent`,
> and chat `userMessage`). See [`SSE_REFACTOR_PLAN.md`](./SSE_REFACTOR_PLAN.md).
> The `AgentSession` / agentic-loop logic below is unchanged; only the wire
> transport differs.

## 1. Goal

Today `backend/agent/` is a transport bridge: external tools (Claude Code,
Gemini CLI) call `POST /agent` and the backend forwards the request to a
single browser tab over WS. We want to flip this around: the backend itself
becomes the agent. The user types a request in a frontend widget ("help me
navigate to products", "where can I see stock?"), the backend calls
OpenRouter, and the LLM drives the page through the same WS bridge until the
task is done.

Out of scope for this doc: the widget UI itself. We only build the channel
and the loop.

## 2. Architecture overview

```
 frontend widget          backend (the agent)          OpenRouter (LLM)
 ───────────────          ─────────────────            ────────────────
   user msg ──┐
              │  WS /ws/agent-chat
              ├────────────────▶  AgentSession
              │                    ├── persist user msg ─▶ ScyllaDB (agent_messages)
              │                    ├── build context (last 5 + page snapshot + menu)
              │                    └── loop:
              │                         ├── OpenRouter chat.completions  ◀──▶ LLM
              │                         ├── if tool_calls: run via existing
              │                         │   InvokeBatch / Navigate / GetPageContent
              │                         │   → loop back with tool results
              │                         └── if final text: persist + push to widget
              │                    persist agent msg ──▶ ScyllaDB
   reply  ◀───┤
              │
```

Two distinct WS connections per browser tab:

- `/ws/agent` — **existing** page-driving channel (`commands.ts` ↔ `ws.go`).
  Keep as-is.
- `/ws/agent-chat` — **new** user↔agent chat channel for the widget.

The new channel never talks to the LLM directly. It only carries chat
messages between the widget and `AgentSession`, which uses the existing
`/ws/agent` channel under the hood when it needs to act on the page.

Rationale: the existing channel is single-tab, local-only, and assumes
"the backend drives the browser" — the new agent fits that model exactly,
so we reuse `InvokeBatch`, `Navigate`, `GetPageContent`, `GetMenu` instead
of re-implementing them.

## 3. Connection information

**Decided** (Q1): Reuse the existing session cookie. The widget opens
`wss://…/ws/agent-chat` with credentials; the WS upgrade handler resolves
the user from the cookie (same path the rest of the backend HTTP API
uses) and stamps `CompanyID`/`UserID` on the session. Connections without
a valid session are rejected at upgrade time.

Per-tab session state (in-memory, not persisted):

```go
type AgentSession struct {
    CompanyID  int32
    UserID     int32
    SessionID  int64        // SUnixTime() of session open
    ChatConn   *websocket.Conn
    PageConn   *clientConn  // existing /ws/agent client for this tab — see Q2
    InFlight   atomic.Bool  // one task at a time per session
}
```

**Decided** (Q2): Extend `ws.go` so the page connection is keyed per
user/tab. This rewires the existing single-client model.

### Plan for the multi-tab refactor of `ws.go`

Today:

```go
var currentClient *clientConn   // one global
var pending = map[uint64]*pendingReply{}  // global request id space
```

After:

```go
type tabKey struct { EmpresaID, UserID int32; TabID string }

type clientConn struct {
    key      tabKey
    conn     *websocket.Conn
    writeM   sync.Mutex
    pending  map[uint64]*pendingReply   // per-tab — request ids no longer collide across tabs
    pendMu   sync.Mutex
    idCount  atomic.Uint64
}

var (
    clientsMu sync.RWMutex
    clients   = map[tabKey]*clientConn{}
)
```

Lookup paths:

- The page WS (`/ws/agent`) authenticates via the same session cookie
  and identifies itself with a `TabID` (generated client-side, sent in
  the initial `ready` message — or as a query param on the WS URL).
- `AgentSession` (chat side) is created with the same `(EmpresaID,
  UserID, TabID)` triple, so it can call `requestFor(tabKey, ...)`
  directly.
- External `/agent` HTTP callers (Claude Code etc.) need to pick a tab
  too. Simplest: add a `?tab=...` query param; if omitted and there's
  exactly one client for the authenticated user, use it; otherwise 400.

**Decided** (Q2b): Same-tab pairing. The widget lives in the SPA, so
page WS and chat WS open from the same tab and share a `TabID` (created
once per page-load, scoped to the tab). Both connections carry it on
upgrade.

Backward compatibility for `POST /agent` (Claude Code path): since
this project is pre-alpha and AGENTS.md says "DO NOT implement
backwards compatibility", I'll just update the HTTP endpoint to require
`?tab=` (or pick the unique tab) and update `HTTP_API.md`.

## 4. Database table

Name: `agent_messages`. Partition by a synthetic `CompanyUserID`
(`CompanyID*1_000_000 + UserID`) so every user's chat history lives in
its own partition — load-last-N for the chat widget is a single
intra-partition slice. Clustering on `SessionID, Timestamp` lets us
order messages within a session naturally.

> **Convention deviation**: the rest of the project uses `EmpresaID`
> (see `types/users.go`). Per user request, this table uses `CompanyID`
> + `CompanyUserID`. Flagging once; intentional.

| Column           | Type   | Notes                                                                       |
| ---------------- | ------ | --------------------------------------------------------------------------- |
| `CompanyUserID`  | int64  | **Partition key.** `CompanyID*1_000_000 + UserID`. Filled by `PrepareCloudSync` so callers never compute it. int64 (not int32) — `CompanyID*1e6` can blow int32. |
| `CompanyID`      | int32  | Original company id — kept as a regular column so we can join/filter.       |
| `UserID`         | int32  | Original user id — regular column.                                          |
| `SessionID`      | int64  | **Clustering key #1.** Groups messages of one chat session.                 |
| `Timestamp`      | int64  | **Clustering key #2.** Unix ms — also the message ID.                       |
| `Message`        | string | Raw user text or final agent text.                                          |
| `AttachedContent`| string | Page HTML / screenshot ref at send-time. **Column exists but written empty for now** — reserved for future use. |
| `Role`           | int8   | `1` = user, `2` = agent.                                                    |
| `Summary`        | string | Short summary of actions performed or long-message digest.                  |
| `TokensUsed`     | int32  | Cumulative tokens for this turn (prompt + completion). Only set on `Role=2` (agent) rows. Used for per-user budget tracking (Q6). |

**Decided** (Q3): Create the column, write empty string for now, define
the population logic later. The page snapshot used by the loop lives in
RAM only and is **not** stored in `AttachedContent`.

Primary key shape (ScyllaDB): `((CompanyUserID), SessionID, Timestamp)`.
Load-last-N query becomes:
`WHERE CompanyUserID = ? AND SessionID = ? ORDER BY Timestamp DESC LIMIT 5`.

Plus `Status int8` and `Updated int32` for project consistency.
`CreatedBy`/`UpdatedBy` are intentionally omitted: rows are insert-only
and `UserID` already identifies the author. `Created` is redundant with
`Timestamp` (the message ID is unix-ms — already a creation time).

**Note**: the table will NOT be generated by `./app.sh create` — that
script hard-codes `EmpresaID` as the partition key. We'll write
`backend/types/agent_messages.go` by hand modeled on `types/users.go:Perfil`,
then run `./app.sh check_tables` to validate.

## 5. The agentic loop

A single iteration of the loop is one OpenRouter call. The loop terminates
when the LLM returns a final assistant message with **no tool calls**, or
when it hits the iteration cap.

### Inputs to each LLM call

1. **System prompt** (static): describes who the agent is, the tools, and
   the response format. Includes the `methods="..."` cheat-sheet from
   `HTTP_API.md` so the model knows what `setValue`/`select`/`navigate`/…
   accept.
2. **Page state** (refreshed every iteration via `GetPageContent`):
   - Sanitized HTML (already produced by `parse_html.go`).
   - Components registry (the compact `[id Type label methods]` list).
   - Current route + accessible menu (`GetMenu`, fetched once per session
     unless the menu changes).
3. **Conversation tail**: the last **5** messages for `(EmpresaID, UserID,
   SessionID)` from `agent_messages`, ordered oldest→newest.
4. **Current user turn**: the message the widget just sent.

### Tools exposed to the LLM

Mirror the existing HTTP API. Each tool is one-to-one with a Go function:

| Tool name       | Go function          | Notes                                  |
| --------------- | -------------------- | -------------------------------------- |
| `invoke_batch`  | `InvokeBatch`        | Args: `[{HandleID, Method, Args}]`     |
| `navigate`      | `Navigate`           | Args: `{Route}`                        |
| `get_page`      | `GetPageContent`     | No args — refresh snapshot mid-loop    |
| `get_menu`      | `GetMenu`            | No args                                |
| `finish`        | (sentinel)           | Final answer to user — ends the loop   |

Why a `finish` tool instead of "LLM emits plain text → done"? Mixing
free-form text with tool calls is the most common source of broken agent
loops. Forcing the model to call `finish({message, summary})` makes the
exit condition deterministic and gives us a clean `Summary` field for
free.

### Loop pseudocode

```go
func (s *AgentSession) RunTurn(ctx context.Context, userMsg string) error {
    // 1. Persist user message immediately (so a crash mid-loop still keeps history).
    saveMessage(s, RoleUser, userMsg, "")

    history := loadLastN(s, 5)              // includes the user msg we just saved
    page    := GetPageContent(ctx)          // initial snapshot
    menu    := GetMenu(ctx)                 // optional — cache per session

    const maxIters = 8
    for i := 0; i < maxIters; i++ {
        resp, err := openrouter.Chat(ctx, buildRequest(history, page, menu))
        if err != nil { return err }

        if call := resp.ToolCall(); call != nil {
            result := dispatchTool(ctx, call)   // invoke_batch / navigate / get_page / …
            history = append(history, resp.ToToolMessage(), result.ToToolResultMessage())
            if call.Name == "get_page" { page = result.(PageContent) }
            continue
        }

        if final := resp.FinishCall(); final != nil {
            saveMessage(s, RoleAgent, final.Message, final.Summary)
            s.ChatConn.WriteJSON(WireReply{Message: final.Message, Summary: final.Summary})
            return nil
        }

        // Defensive: model emitted plain assistant text instead of finish().
        // Treat as final answer.
        saveMessage(s, RoleAgent, resp.Text(), summarize(resp.Text()))
        return nil
    }
    return errors.New("agent exceeded max iterations")
}
```

Key points:
- `InFlight` guards concurrent turns per session — second user message
  while the agent is still running is rejected with a "busy" reply.
- The 5-message context is **persisted history**. The loop's *internal*
  tool-call/result messages are NOT saved to `agent_messages` — they live
  only in the in-memory `history` slice for that turn. Otherwise the
  context window fills up with raw HTML snapshots within a few turns.
- `Summary` is the LLM-authored summary. For user messages we either copy
  `Message` if short, or have the LLM produce a digest at the start of the
  next turn — I'd skip that initially and just leave `Summary` empty for
  user messages.

**Decided** (Q4): Blocking responses. Agent calls
`chat.completions` normally and emits one `agentReply` event when the
loop finishes. Streaming can be added later if latency feels bad.

## 6. OpenRouter integration

New package `backend/agent/llm/` (so the loop logic stays in
`backend/agent/`):

- `openrouter.go` — thin HTTP client around
  `https://openrouter.ai/api/v1/chat/completions`. Supports tool calling
  (`tools`/`tool_choice`) in the OpenAI-compatible shape OpenRouter
  uses.
- API key from env: `OPENROUTER_API_KEY`.
- **Decided** (Q5): Default model `tencent/hy3-preview`. Resolved from
  `core.Env.OPENROUTER_KEY` + `OPENROUTER_MODEL` (loaded from
  `credentials.json` — same path as the rest of the project).

  **Constraint discovered in step 2 validation**: this model's provider
  routing **does not support `tool_choice: "required"`** (returns
  HTTP 404). Use `"auto"` or omit `tool_choice`. The system prompt has
  to do the work of telling the model "always call `finish` to end the
  turn" instead of relying on a forced tool choice.
- **Decided** (Q6): Per-message `TokensUsed` column on `agent_messages`
  (see schema above). Records `prompt_tokens + completion_tokens` summed
  across the loop's iterations for one user turn. The actual quota /
  enforcement policy is out of scope here — we just collect the data so
  a budget check can be added later by summing recent rows.

## 7. File layout (proposal)

```
backend/agent/
├── agent.go            (unchanged)
├── http.go             (unchanged)
├── protocol.go         (unchanged)
├── ws.go               (unchanged — see Q2)
├── commands.ts ...
├── chat_ws.go          NEW — /ws/agent-chat handler, AgentSession
├── chat_loop.go        NEW — RunTurn, tool dispatch
├── chat_store.go       NEW — saveMessage, loadLastN
├── llm/
│   ├── openrouter.go   NEW — HTTP client, tool-call schema
│   └── prompts.go      NEW — system prompt, tool definitions
└── AGENTIC_LOOP_DESIGN.md (this file)

backend/types/
└── agent_messages.go   NEW — generated via scripts/create_edit_table
```

Frontend (widget — out of scope for this doc, but the channel shape):

```
frontend/core/agent-chat/
├── ws.ts               NEW — connects to /ws/agent-chat
└── session.ts          NEW — sends user msg, receives WireReply
```

## 8. Wire format for `/ws/agent-chat`

Mirrors the existing `protocol.go` style: capitalized field names, no json
tags.

**Client → server**:
```json
{ "Type": "userMessage", "Payload": { "Message": "help me find stock", "Timestamp": 1736300000123 } }
```

**Server → client**:
```json
{ "Type": "agentReply",  "Payload": { "Message": "...", "Summary": "...", "Timestamp": 1736300003456 } }
{ "Type": "agentStatus", "Payload": { "State": "thinking" | "acting" | "idle", "Step": 2, "MaxSteps": 8 } }
{ "Type": "agentError",  "Payload": { "Message": "openrouter rate limit" } }
```

`agentStatus` is optional but cheap to add now and useful for the widget
to show "Agent is acting on step 2/8…".

## 9. Implementation order (proposed PRs)

Build in small, reviewable slices:

1. **Table only** — generate `agent_messages` via `scripts/create_edit_table`,
   add `chat_store.go` with `saveMessage` + `loadLastN`. No WS, no LLM.
   Validates the schema.
2. **OpenRouter client** — `llm/openrouter.go` + a small CLI/test that
   calls it with a hard-coded prompt and no tools. Validates the API
   integration.
3. **WS channel** — `/ws/agent-chat` handler, in-memory `AgentSession`,
   echoes user messages back. No LLM yet.
4. **Loop, no tools** — wire LLM in, but only `finish` tool is registered.
   Agent can chat but can't act on the page.
5. **Tools** — register `invoke_batch`, `navigate`, `get_page`, `get_menu`
   and dispatch them through the existing `agent.go` API.
6. **Polish** — `agentStatus` events, iteration cap UX, error mapping.

Stop and re-align after each step.

---

## Open questions — all resolved

- **Q1 Auth** → reuse session cookie. ✓
- **Q2 Multi-tab** → key page connections by `(CompanyID, UserID, TabID)`. ✓
- **Q2b Tab pairing** → same-tab. ✓
- **Q3 AttachedContent** → column exists, written empty, future use. ✓
- **Q4 Streaming** → blocking responses first cut. ✓
- **Q5 OpenRouter model** → `tencent/hy3-preview` default, env-configurable. ✓
- **Q6 Token budget** → `TokensUsed int32` column on `agent_messages`. ✓

Once these are settled I'll start with step 1 (table generation only) and
ask for a review before moving on.
