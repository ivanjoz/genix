# RATIONALE — agent

Design decisions for the agent (genix/agent, module `genix-agent`), newest first.

## Category names are matched against every active category
**Context** — `category_id` (the `sale-category` views) is a record column, so a name in quotes must
resolve. Categories are `shared_list_records` (ListID 1) and have no text search.
**Decision** — `searchCategories` reads every active category of the company and lets
`classifyMatches` pick, as with the text-search hits of products and clients. `SharedListCategoryID`
and `SharedListBrandID` moved from `business/product-ecommerce.go` to `business/types` so the agent can
use them.
**Rationale** — A company has tens of categories, so one read beats adding a text-search index. The
options of an ambiguous question come in the table's order, not ranked by a search.

## The golden questions check statements, not answers
**Context** — The plan asked for 30–50 golden questions with expected results on a seeded company.
No model or seeded data was available while building.
**Decision** — `salestools/testdata/golden.jsonl` has 12 questions, each with the reference call a
model should write. A test prepares each call on the real frames and plans its table and chart.
**Rationale** — It catches a frame or grammar change that breaks a known-good statement. Still
missing: expected numbers on seeded data, and a harness that runs the questions through a model.

## A chart's second scale is a second chart
**Context** — The tool can put a series on `y2` (units beside money); genix-ui's `Chart` has one
value axis.
**Decision** — `planAgentChart` returns one plan per scale. The `y2` series draw as a line chart
under the main one, in the next palette colors.
**Rationale** — No change to the shared chart component. Cost: the two scales don't share a plot.

## Tables and charts never reach the model again
**Context** — A sales answer carries up to 400 rows.
**Decision** — `agentReply` sends `Tables`/`Charts` and the browser stores them on the IndexedDB
row with the rest of the chat history. Later turns read only the answer's message and summary.
**Rationale** — The summary (the statement plus the first records) is enough for a follow-up, and
400 rows would cost thousands of prompt tokens on every turn.

## Name lookups ask one question per pause, and check the answer
**Context** — A statement may hold several ambiguous names, and the browser echoes the pick back.
**Decision** — `resolveNames` asks the first ambiguous name. The resumed call adds the pick to
the tool's interrupt state and runs again, which may ask the next one. An answer that isn't one of
the offered choices asks the same question again. A name with no match goes back to the model as an
error, not as the turn's reply (berryapps ended the turn).
**Rationale** — One question per pause fits `ChoiceQuestion` and the one-answer `Resume`. Checking
the pick against the stored choices means the client can't inject a record ID. Cost: one pause per
ambiguous name, and a model call for "not found".

## The sales tool is offered on every turn with tools
**Context** — D2 gates the tool on `GET.sale-summary` access. The agent has no access catalog,
since `access.toml` is embedded in the backend's `package main`.
**Decision** — `RunTurn` always adds `sales_frame_sql`; `RunReadOnlyTurn` doesn't. `GET.sale-summary`
is not mapped in `access.toml`, so the backend lets any signed-in session read it.
`TestSalesToolGateMatchesTheRoute` fails as soon as it gets mapped.
**Rationale** — The real gate today is "signed in", which the turn already checks. Building a
catalog the agent could read (moving `access.toml` or embedding a copy) was not worth it for an
unmapped route. Cost: about 2K schema tokens on every turn with tools, social turns included.

## A tool ends the turn by its result: `EndsTurn`, not `ReturnDirectly`
**Context** — Eino's `ReturnDirectly` picks by tool name before the tool runs, so a FrameSQL call
with a bad statement would end the turn with the error. The model has to see the error and fix
its SQL.
**Decision** — `harness.TurnRequest.EndsTurn(toolName, result)`. A `BeforeModelRewriteState`
middleware runs first among the handlers and checks the last round's tool results. When one ends
the turn, it stops the agent before the model call with `turnEndedError`, which `RunTurn` reports
as `EndToolName`/`EndToolResult`. `finish` always ends the turn; `sales_frame_sql` only when it
didn't return `{"error"…}`.
**Rationale** — The answer still costs no extra model call, and errors go back to the model. Cost:
it relies on a middleware error being surfaced as an event error, which the harness test covers.

## A paused turn stores its spec next to the checkpoint
**Context** — Eino's checkpoint holds the graph's state, not the agent's configuration. A resume
must rebuild the same agent: model, instruction, tools, mutation routes.
**Decision** — `pauseTurn` writes `executionSpec` as `<id>.turn.json` next to `<id>.bin`. The
`CheckPointID` (`^[a-z0-9]{6,20}$`) and the company/user prefix keep a resume inside the caller's own
checkpoints. A missing spec answers "the turn failed, ask again".
**Rationale** — The resume skips discovery: a second planner call could pick other tools, and the
checkpoint would not match them. Cost: two S3 objects per pause, and nothing expires them yet (an
S3 lifecycle rule on `agent-checkpoints/` is still to add).

## A turn pauses 40 s before its deadline
**Context** — A Lambda turn dies at `turnTimeout` (4 min); D3 wants a checkpoint instead.
**Decision** — `schedulePause` cancels with `CancelAfterToolCalls` 40 s before the context
deadline and waits up to 15 s for the tools in flight. The reply is "La consulta tomó más tiempo del
previsto. ¿Continúo?" with a "Continuar" button that resumes the checkpoint.
**Rationale** — 40 s covers one model call finishing plus the checkpoint upload. Cost: a long turn
uses about 3 min 20 s per invocation, not the full 4.

## The planner and the page builder stay on `llm.Client`
**Context** — The plan said to delete `llm/client.go` once the harness ran the turn.
**Decision** — Only the execution turn moved to Eino. The discovery planner (one structured call)
and the page-builder loop still call `llm.Client`. Its charge moved to tokens too.
**Rationale** — Neither needs checkpoints or tools through Eino, and porting the builder loop is
its own task. Cost: two model clients. `harness.ChatModel` reuses `llm.ProviderConnection`, so the
provider settings live in one place.

## Usage is charged from the event stream, and the model isn't streamed
**Context** — The plan mentioned an Eino `callbacks.Handler` for charging and the prompt log.
**Decision** — `consumeTurnEvents` charges each assistant message's `TokenUsage` as it arrives, and
cancels the turn at once when fareward refuses. The prompt log is the pruning middleware's last
step. Model calls use `Generate`.
**Rationale** — It's the one loop that already sees every message, and a refusal must stop the
turn, which a callback can't do. The chat shows progress labels, not tokens, so streaming would buy
nothing.

## The agent has its own minimal router, not the backend's gate
**Context** — The backend's request pipeline (`mainHandler`, `enforceAccessAndCredits`,
`prepareResponse`) lives in the backend's `package main`, which another module cannot import.
**Decision** — `cmd/server/main.go` has a ~30-line router over `agent.ModuleHandlers`: it strips
`/api/`, sets `RouteID`, validates the session for non-`p-` routes and answers through
`core.SendLocalResponse` / `core.MakeResponseFinal`. No access catalog, no base GET charge, no
`EmitRequestLog`.
**Rationale** — The agent has two routes: `p-agent-turn` already validates the session and charges
its own turn, and `agent-models` only needs a session. Moving the gate into a shared package would
have been a large refactor of the backend for two routes. Cost: `GET agent-models` is no longer
charged base credits and agent requests leave no row in the request log.

## The agent's Lambda is BUFFERED, 512 MB
**Context** — The backend's Function URL is RESPONSE_STREAM; the agent needs a URL of its own.
**Decision** — `AgentFunction` (`<app>-agent`, 512 MB, 300 s) with a BUFFERED URL; the handler is
the buffered one only.
**Rationale** — A turn answers `{Ok:true}` and everything else travels on the SSE bridge, so no
response needs streaming. 512 MB because the LLM round-trips and FrameSQL reads are heavier than a
backend CRUD request; tune after measuring.

## The frontend sends `agent-*` / `p-agent-*` routes to `[[endpoints]].agent`
**Context** — The agent now has its own host. `GetHandler` and `fetch` both build URLs through
`Env.makeRoute`, and the delta cache keys on the route string.
**Decision** — Each endpoint gains `agent` (config.toml `[[endpoints]].agent`, local:
`[agent].port`, 3590). `Env.makeRoute` sends any route named `agent-*` or `p-agent-*` to
`Env.API_ROUTES.AGENT`. The stream base is `bridge || agent`; a stream base other than the agent
host means the bridge. An endpoint without `agent` offers no assistant.
**Rationale** — One rule in one place, and no caller had to change. The alternative, absolute
URLs in each caller, would have leaked into the delta cache keys. Cost: the name prefix is now
load-bearing — a backend route named `agent-…` would be misrouted.

## The local agent server binds loopback only in dev
**Context** — The backend's dev run binds loopback plus the tailnet address.
**Decision** — `cmd/server` binds `127.0.0.1` when launched with `dev`, all interfaces otherwise.
**Rationale** — Less code than copying `resolveDevListenAddresses`. Cost: the chat does not work
from another tailnet machine in dev until that is ported.

## The agent's dev process runs under the nodemon CLI
**Context** — `start.js` drives the backend with the `nodemon` module, which is a singleton.
**Decision** — The agent runs as `nodemon --exec "go run ./cmd/server dev"` through `runScript`,
watching `agent/` plus `backend/core`, `backend/db` and `backend/sales/types`.
**Rationale** — A second `nodemon()` call would replace the backend's. The watch list is the
backend code the agent links today; a new `*/types` import has to be added to it.
