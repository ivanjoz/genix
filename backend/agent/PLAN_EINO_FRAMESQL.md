# PLAN — The agent on Eino, answering sales questions with FrameSQL

## Goal

1. **Rescue FrameSQL**: the SQL engine over DataFrames that genix-orm had in `dataframe/framesql`.
   Port it to the current `dataframe` module, the one that holds `sale-summary` and
   `sale-client-product` today.
2. **Replace the agent's harness with [Eino](https://github.com/cloudwego/eino)** (`v0.9.22`). It
   replaces the hand-written LLM loop, the LLM client, and the plumbing for the tools'
   ask-the-user flow.
3. **Make the agent answer sales questions**, from `backend/sales/types/sales.go`, with one
   FrameSQL tool over the two sale frames. It answers with a message, a table and/or a chart, and
   needs no second LLM call.
4. **Build the agent as its own Lambda** in `genix/agent/`, importing `app/...` from
   `../backend`. `backend/main.go` stops compiling it.
5. **Optional, later: give the agent a shell** over a per-run workspace, as `smartberry-agent`
   does. The harness is built so the shell can be added without reshaping it.

## 0. Where things are today (verified)

| Piece | Where it lives now |
|---|---|
| FrameSQL engine | Deleted from genix-orm in commit `edb2a3b`. The last version is at `edb2a3b^`: `dataframe/framesql/{framesql,parse,plan,engine}.go` + tests (~95 KB), and `DATA_FRAMES_SQL.md` (38 KB). |
| FrameSQL `db` aliases | berryapps `01be9538^:backend/db/frame_sql.go` (40 lines). Reference only. |
| FrameSQL agent tool | berryapps `01be9538^:backend/core/agent/agenttools/frame_sql.go` (655 lines): input schema, grammar for the model, view (table and chart), name resolution. Reference only. |
| Eino harness reference | `/home/ivanjoz/projects/smartberry-agent` (Eino `v0.9.19`), not imported, patterns only: `agents/runner.go` (Runner, timeout pause, event consumption, token usage, traces), `agents/model.go` (model aliases → `eino-ext` openai/claude), `aws/checkpoint.go` (CheckPointStore on S3), `projects/turn.go` (chat turn, streaming), `tools/shell/` (shell over a workspace), `AGENTS.md` ("Eino practices"). |
| Current dataframe read | `genix-orm/dataframe/read.go`: `dataframe.Read(..., isFresh, now) []Row{Keys, RowValues, Sums}`. It is reached through `db.QueryFrame[T](partition, frame).Fresh().Exec()` (`genix-orm/db/dataframe.go`). |
| Sale frames | `sales/types/sales.go`: `sale-summary` (Key `Date`; Rows `ProductID, SubDivisor`; 6 sums) and `sale-client-product` (Key `Date`; Rows `ClientID, ProductID, SubDivisor`; same sums). |
| Record names | `Product` has `TextSearch{Name, BrandName}` and `CacheByIDs`. `ClientProvider` (clients and providers, split by `Type`) has **no** TextSearch. |
| Agent loop today | `backend/agent/chat_loop.go` (hand-written tool loop, 12 iterations), `llm/client.go` (OpenAI-compatible HTTP for Meta and OpenRouter), `discovery/planner.go` (classifier call), `route_turn.go`. |

## 1. Decisions (settled with the user)

| # | Decision |
|---|---|
| D1 | Add a non-generic `db.ReadFrameRows(...)` to the ORM entry point for FrameSQL. |
| D2 | The sales FrameSQL tool is **always offered**, gated only by the user's access to `GET.sale-summary`. No tool index. berryapps is a reference, not a dependency. |
| D3 | Ask-the-user pauses use **Eino interrupt/resume with a stored checkpoint** (S3), not berryapps' replay. See section 4. |
| D4 | Add a `TextSearch` to `ClientProvider` (schema change) so names of clients resolve to IDs. |
| D5 | Charge inference by Eino's `TokenUsage`, not by request/response bytes. |
| D6 | `AgentMessage` moves to `backend/core/types`. |
| D7 | FrameSQL reads **fresh**: `isFresh = true` (change log included). The read compacts the partition when `CompactOn` is due, as any user query of the sales summary does today (`dataframe.Read` → `isCompactDue` → `compactDueFrame`). No new compaction path. |
| D8 | Both sale frames gain one quantity sum in a common unit, so the agent never handles sub-units (section 5, Phase 2). Frame `Version` 1 → 2 and a rebuild. |
| D9 | Checkpoints live in the S3 Express One Zone bucket the DataFrames already use (`[file_store].bucket`), through the same `utils.FileStore`. |
| D10 | `ClientProvider` TextSearch covers `Name` and `RegistryNumber`. |

## 2. What breaks when FrameSQL meets the current dataframe module

The engine was written against the earlier frame API. Three changes:

1. **Rows are now 1–3 columns** (`dataframe.MaxRowColumns = 3`). The old engine assumed a single
   `Frame.Rows Column` and a `file.RowIDs` column.
   - Fix: `Frame.Rows []Column`.
   - The row filter and the GROUP BY bind to a specific Rows column (`rowIndex`).
   - `foldFile` reads `row.RowValues[rowIndex]`.
2. **The read is no longer a streaming `Source.Scan` with a `selectFiles` hook.** Today it is
   `dataframe.Read`, which returns sorted, materialized `[]Row`.
   - `Source` becomes one read over a `dataframe.Selection`, and the engine folds the rows.
   - `MaxFiles` and `selectFiles` go away. The cost bound is the selection: at most
     `MaxFirstKeys = 400` values of Keys[0]. Both sale frames have one Key, so at most 400 files
     per read.
3. **Keys are `[]int64` slices**, not `[MaxKeys]int64` arrays. This is a mechanical change.

The parser and most of the planner (WHERE / GROUP BY / ORDER BY / LIMIT, PERIOD(), WEEK(),
MONTH(), named records) carry over unchanged. Their tests carry over too, with fixtures moved to
multi-column rows.

## 3. Target layout

```text
backend/genix-orm/dataframe/framesql/     engine, restored and ported (submodule: commit + push there)
backend/genix-orm/DATA_FRAMES_SQL.md      restored, updated to multi-column Rows
backend/db/frame_sql.go                   db.SQL* aliases + db.ReadFrameRows (D1)
backend/sales/types/sale_frames_sql.go    the two datasets: column names, kinds, labels, cards
backend/core/types/agent_messages.go      AgentMessage (D6)
agent/                                    own go.mod (module genix-agent), replace app => ../backend
  main.go                                 Lambda + local HTTP (InitPromptLog, /agent/* routes move here)
  harness/                                the only package that imports Eino's runner:
    model.go                              model aliases → eino-ext components (smartberry agents/model.go)
    runner.go                             Runner, timeout pause, event → SSE mapping, token usage
    checkpoint.go                         CheckPointStore on S3 (smartberry aws/checkpoint.go)
    middlewares.go                        tool errors as text, tool-round trimming, literal instruction
  agenttools/                             tool contract (Tool, Run, Answer, Chart, Table) → Eino tools
  agenttools/frame_sql.go                 the FrameSQL tool
  salestools/                             sales FrameSQL tool registration (datasets + record names)
  pagetools/                              get_page / navigate / invoke_batch / get_menu as Eino tools
  shell/                                  OPTIONAL (Phase 6): workspace + shell, from smartberry tools/shell
  bridge, channel, parse_html, page_snapshot, knowledge, ragdocs   moved as they are
```

`agenttools` imports no module. `salestools` imports only `app/sales/types`, `app/crm/types`,
`app/production/types`, `app/db` and `app/core`. This keeps to `MODULE_BOUNDARIES.md`.

## 4. Why the stored checkpoint (D3)

- **What it does:** a tool calls `tool.StatefulInterrupt(ctx, question, state)`. The runner writes
  the graph state to the `CheckPointStore` under the turn's id. The next request calls
  `Runner.ResumeWithParams(checkPointID, answers)`, and the same tool call continues with the
  user's pick (`tool.GetInterruptState`).
- **What it saves over berryapps' replay:**
  - No model call is repeated: the resume lands inside the tool, not at the start of the turn.
  - No bespoke `BoundAnswers` plumbing.
- **Same mechanism, two more uses:**
  - **The Lambda time limit:** smartberry's runner cancels at a safe point
    (`adk.CancelAfterToolCalls`) before the deadline, the checkpoint is written, and the next
    invocation resumes.
  - **Future shell runs (Phase 6)**, which are long.
- **Costs:**
  - S3 storage with a short lifecycle (1 day: an unanswered question is abandoned).
  - Checkpoints are gob-encoded Eino graph state, so a deploy between the question and the answer
    can make one unreadable. The turn must detect a failed resume and tell the user to ask again.
  - Every type in an interrupt state must be gob-registered.
- **Where (D9):** the S3 Express One Zone bucket of `core.ConfigureFileStore` (`[file_store].bucket`),
  under `agent-checkpoints/<companyID>/<turnID>.bin`, through the same `utils.FileStore`. No new
  S3 client and no new config.
  - Same contract as smartberry's `aws/checkpoint.go`: a missing object is "no checkpoint", any
    other error is an error.
  - Expiry is a 1-day lifecycle rule on that prefix. Directory buckets support expiration
    lifecycle rules.

## 5. Phases

### Phase 1 — Restore FrameSQL in genix-orm (no agent yet)

1. Restore `dataframe/framesql/*` and `DATA_FRAMES_SQL.md` from `edb2a3b^` in the genix-orm
   submodule.
2. Port it to the current module (section 2).
3. Add `db.ReadFrameRows(tableName, partition, frameName, selection, isFresh)` (D1). It returns
   `[]dataframe.Row` and `*dataframe.Frame` from the scylla executor's compiled
   `dataframe.Table`.
4. `backend/db/frame_sql.go`: the `db.SQL*` aliases plus `db.NewFrameSource(...)`, which always
   reads fresh (D7).
5. Tests:
   - The restored parse/plan/engine tests, ported.
   - One engine test with 3 Rows columns (client, product, divisor).
   - One `db` test against a local sale frame.

**Done when:** `go test ./dataframe/framesql/...` passes in genix-orm, and a scratch command runs
`SELECT product_id, SUM(amount) WHERE date IN PERIOD("M") GROUP BY product_id` against a real
company's `sale-summary`.

### Phase 2 — The sale datasets and client search

1. **Quantity in a common unit (D8).** Both frame structs gain two `float64` sums, filled in
   `eachSummaryLine`:
   - `SoldQuantity` = units + sub / divisor;
   - `PendingQuantity` = the same for pending.

   They are whole units with a fraction: 1 box + 1 candy at divisor 6 is 1.1667.
   - The frames keep floats as `int64 × FloatScale` (6 decimals), so adding and subtracting a
     sale's own line is exact.
   - This is the same sum the sales chart already computes client-side
     (`sale_orders_charts.ts`: `units + sub / subDivisor`).
   - It makes 8 sums, exactly `MaxMaskedSums`.

   The rollout:
   - `Version: 2` on `sale-summary` and `sale-client-product`.
   - Rebuild both frames for every company with `RebuildDataFramesPartition`, run once from a
     `backend/exec` command, so no user pays the first-read rebuild.
   - Update the chart's frame reader (`frontend/routes/sales/sale_orders_charts/sale_orders_charts.ts`)
     to the new sums order.

   FrameSQL needs one more kind, **`KindQuantity`**: a float sum read as `value / FloatScale` and
   shown with up to 3 decimals.

2. **Datasets.** `sales/types/sale_frames_sql.go`, as the agent sees them:

   | Dataset | Keys | Rows | Sums |
   |---|---|---|---|
   | `sale_summary` | `date` (day) | `product_id` (ref → products) | `amount` (cents, AmountPEN), `debt_amount` (cents), `quantity` (SoldQuantity), `pending_quantity` |
   | `sale_client_product` | `date` (day) | `client_id` (ref → clients; 0 = no client), `product_id` | same |

   - `sub_divisor` and the split `*_units` / `*_sub` sums stay in the frame for the chart, but they
     are **not exposed to the agent**. `framesql.Column` gains `Hidden`: the column is left out of
     the notation, the planner refuses it, and its rows fold into their product.
   - The agent cannot get sub-units wrong because it never sees them.
   - Each dataset gets a Title, a Description (what one row is and what its numbers count) and
     Spanish and English Examples.

3. **Client search (D4, D10).** `ClientProvider` gets
   `TextSearch: &db.TextSearch{Columns: db.Cols(e.Name, e.RegistryNumber), Status: e.Status}`.
   - Clients and providers share the table, so the client resolver keeps only `Type` = client
     hits.
   - Existing rows need the index built: run the ORM's text-index rebuild for the table. That
     fixes the data, with no fallback on read.

**Business traps the descriptions and tests must cover:**
- **Quantity** is in whole units with a fraction (D8). A test sells 1 box + 1 sixth and
  1 box + 1 twelfth of one product and expects `SUM(quantity) = 2.25`.
- **Annulled sales** add nothing, **debt** is unpaid amount, and **pending** is undelivered
  quantity (`eachSummaryLine`).
- **Money is in PEN cents**, already converted from USD with the sale's own rate. The model
  never divides by 100.

### Phase 3 — `agent/` as its own module and Lambda

1. Remove `agent.ModuleHandlers` from `main-handlers.go`.
2. Remove `InitPromptLog` and the `/agent/*` routes from `main.go`.
3. Move `AgentMessage` to `backend/core/types` (D6) and regenerate `controllers.generated.go`.
4. `git mv backend/agent agent`. Add `agent/go.mod` with `replace app => ../backend`, and repeat
   backend's `replace` lines with their paths rebased.
5. Rewrite `app/agent/...` imports to `genix-agent/...`.
6. Optional: add a root `go.work` for the editor.
7. Update `CLAUDE.md` (repo map) and `MODULE_BOUNDARIES.md`.

**Done when:** `go build ./...` passes in both `backend/` and `agent/`, `backend`'s binary no
longer contains `genix-agent` (`go list -deps`), and the chat works locally against the agent
process.

### Phase 4 — Eino harness

Follow `smartberry-agent/AGENTS.md` "Eino practices". Every point below has a working reference
there.

**Model** (`harness/model.go`)
- Aliases (`default`, `fast` for the classifier), mapped from `MODELS` / `DEFAULT_MODEL`.
- `eino-ext/components/model/openai` against OpenRouter's and Meta's OpenAI-compatible
  endpoints.
- **Verify first** that the reasoning and provider-routing fields `llm/client.go` sends today
  survive. If they don't, wrap the component.

**Agent**
- `adk.NewChatModelAgent` per turn, with only the turn's tools:
  - the page tools;
  - the sales FrameSQL tool when the user can call `GET.sale-summary` (D2);
  - documentation search.
- `ToolCallMiddlewares` returns tool errors to the model as text (smartberry
  `ErroresDeToolComoTexto`).
- `GenModelInput` gives a literal instruction, because Eino formats `{` in instructions once
  there are session values.
- `MaxIterations` replaces the loop's cap of 12.
- A reduction middleware replaces "keep the last 2 tool rounds".

**Runner** (`harness/runner.go`)
- The only place that creates `adk.NewRunner`, with `CheckPointStore` (D3) and streaming.
- Events map to the existing SSE events (`agentStatus`, `agentReply`, `agentError`, `turnEnd`)
  through the bridge.
- A timeout pause is scheduled `turnTimeout − margin` ahead, as smartberry's `programarCorte`
  does.

**Ask the user**
- `ResolveRecords` (an ambiguous product or client name) calls `tool.StatefulInterrupt` with the
  choice card.
- The answer comes back as a new `p-agent-turn` with `resume: {checkPointID, answers}` →
  `ResumeWithParams`.

**Turn result**
- The FrameSQL answer (message, table, chart) ends the turn with no second model call: a
  return-directly tool, or an ADK exit after the tool.

**Callbacks**
- An Eino `callbacks.Handler` writes the prompt log (`prompt_log.go`).
- It charges `TokenUsage` per model call (D5). `core.ChargeInferenceUsage` changes from bytes to
  tokens.
- It emits `agentStatus` progress labels.

**Discovery**
- The classifier/planner stays as one call through the `fast` model before the agent.

**Delete:** `llm/client.go`, `chat_loop.go`, and the parts of `route_turn.go` the runner replaces.

**Done when:**
- a page task ("go to sales and filter today") works end to end;
- "¿cuánto vendí este mes por producto?" returns a ranked table plus a bar chart;
- an ambiguous product name produces a choice card, the turn ends, and the answer resumes from the
  checkpoint without repeating a model call.

### Phase 5 — Sales answers end to end

1. `agent/agenttools/frame_sql.go`, using berryapps' as reference:
   - `db.SQL*` types;
   - multi-column Rows in the notation and in the examples for the model;
   - boot panics on a ref column without records.
2. `agent/agenttools/period.go`: today is `core.FechaUnix()`, the effective clock.
3. `salestools` registers `sales_frame_sql`:
   - products: names by ID through the cached-by-IDs read, name search through
     `db.SearchTextIDs[Product]`;
   - clients: through the new `ClientProvider` TextSearch, client hits only.
4. Protocol and frontend: `agentReply` gains `Tables` and `Charts`, and the chat widget renders
   them. Genix's `protocol.go` has neither today.
5. Golden questions (`agent/salestools/testdata/golden.jsonl`, 30–50, mostly Spanish): statement
   shape plus expected result on a seeded company.

### Phase 6 — OPTIONAL: shell for the agent (future)

Not built now. Phases 3–4 keep the room for it:

- **Port smartberry `tools/shell`:** `Workspace` implements both `filesystem.Backend` and
  `filesystem.Shell`, so `read_file` and `execute` see the same dir. Every command goes through
  `shell.Correr` with an environment built from zero (no AWS credentials, no config). Time,
  output and process-group limits are fixed in code.
- **Enabled per turn kind**, never in the default chat agent: a process opts in, as smartberry
  processes do.
- **Long runs** use the same checkpoint pause (D3). Workspace files that can't be rebuilt are
  uploaded in `Finish` and restored in `Build`, because a resume can land in another container.
- **CLI tools** come from a Lambda layer (smartberry `layers/cli`), unpacked and verified at turn
  start.
- **Whatever the shell produces is untrusted**: Go validates it before any DB write.

## 6. Order and dependencies

```text
Phase 1 (genix-orm FrameSQL) ──┐
Phase 2 (datasets, client TextSearch) ──┼──► Phase 5 (sales answers)
Phase 3 (agent/ module) ──► Phase 4 (Eino) ──┘          Phase 6 (shell): later, on top of Phase 4
```

- Phases 1–2 and 3–4 can run in parallel.
- Phase 1 is self-contained and testable without any agent. Do it first.

## 7. Risks

| Risk | Mitigation |
|---|---|
| Eino is pre-1.0 (`v0.9.x`) and its API moves | Pin the version. Only `harness/` and the tool adapters import Eino. |
| A checkpoint unreadable after a deploy | A failed resume answers "please ask again". Short S3 lifecycle. |
| The Eino model component drops OpenRouter reasoning/provider fields | Verify in Phase 4 before deleting `llm/`. A thin custom `ToolCallingChatModel` if needed. |
| The model writes SQL that is valid but wrong (sub-units, debt vs amount) | The descriptions name the traps. Golden questions. Engine errors are written for the model. |
| The agent Lambda pulls all of `app/core` (+ its `init()`) | Measure the binary and cold start after Phase 3. Trim later through `*/types`. |
| A frame mid-rebuild (`ErrFrameNotBuilt`) | Answer "try again in ~20 min". |

## 8. Open questions

None. Every decision is in section 1.
