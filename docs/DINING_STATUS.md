# Dining — Implementation Status

Handoff for `docs/DINING_PLAN.md`, which holds the design. This file records what is finished,
which decisions have been taken, and what the next agent has to do. Updated 2026-11-13.

---

## Steps 1–3: done

The tables, the menu and floor pages, and charging and annulling a table are implemented and were
tested in the browser and checked in the database.

### Step 3: charging a table goes through the shared create-sale core

- **The shared core** lives in `backend/sales/saleissue/types/` (`saleissue "app/sales/saleissue/types"`).
  - `new_sale.go`:
    - `PrepareNewSale(req, NewSaleRequest)` → `PreparedSale`
    - `WriteNewSaleIn(transaction, req, &prepared)`, called inside the caller's transaction
    - `ScheduleNewSaleDocument(&prepared)`, called after the commit
  - `sale_issue_series.go`: `ResolveSaleIssueSeries`.
  - `sale_client.go`: `ResolveSaleClientID` (a walk-in client is created in its own transaction).
  - It cannot live in `sales/types`: `invoicing/types`, `config/types` and `crm/types` all import
    `sales/types`, so that would be an import cycle.
- **Three callers:**
  - The till: `createSaleOrder` in `backend/sales/sale_order_create.go`. A till sale now forces
    `OnlineSaleOrderID = 0`.
  - The web-order settlement: `settleOnlineSaleOrder` in `backend/sales/online_sale_order.go`.
  - The table charge: `PostDiningTableOrderCharge` in `backend/dining/table_order_charge.go`.
- **Dining endpoints:**
  - `POST.dining-table-order-charge`: the request names `IssueSeriesID`, the client (an existing
    one or `ClientInfo` for a new one) and `CashBankID`. A `CashBankID` of 0 records the sale
    unpaid, as debt.
  - `POST.dining-table-order-annul`: needs a `Reason` of at most 200 characters, which is stored in
    `AnnulReason` (cb 17).
  - Both require sub-access 2 "Cobrar Mesa" of access 61.
- **Frontend:**
  - `frontend/routes/dining/floor-service/TableChargeForm.svelte`, with the rules in
    `table-charge.ts` and its tests in `table-charge.test.ts`.
  - The "Cobrar mesa" and "Anular pedido" buttons are in `TableOrderPanel.svelte`.
- **Fixed along the way:**
  - Annulling a web order with a reason over 200 characters crashed the handler: `core.If`
    evaluates both branches, so `reason[:200]` ran even for short reasons. Over-long reasons are
    now refused, there and in the dining annul.
  - The dining pages showed a second error toast reading `[object Object]`. `POST` already shows
    the error toast, so the callers' own `notifyFailure(String(err))` calls were removed.

### Verified

- `go build ./...` passes in the backend and the agent.
- `go test ./sales/... ./dining/... ./invoicing/... ./production/...` passes.
- `scripts$ go run . check_module_imports` and `check_tables` pass.
- The frontend vitest suite passes (74 tests; how to run it is in the memory note "running frontend
  tests").
- `svelte-check` shows the 13 errors that were already there; none are in dining.
- In the browser, with the database checked afterwards:
  - A paid table charge with a boleta.
  - An unpaid charge with a made product and a newly registered client.
  - The "unserved lines" blocker.
  - Annulling an order, which released its reservations.
  - Till sales through the API: paid, on credit, and an update that delivers and takes a partial
    payment.
- **Not tested in the browser:** settling a web order. There was no pending web order, and creating
  one needs a storefront session. It runs the same core code as the other two callers.

---

## Step 4: live updates over the SSE bridge (DINING_PLAN §8), partly done

### Done

1. **fareward (Rust)**
   - `src/bridge/topic.rs` validates topic names (`<name>.<companyID>[.…]`).
   - `src/bridge/channel.rs` keeps a topic registry that is updated when a channel is replaced or
     closed, and provides `publish_topic`.
   - `src/bridge/http.rs`:
     - `GET /sse?ch=…&topics=a,b` answers 403 to a topic of another company.
     - `POST /publish-topic {Topic, Message}` answers `{Delivered: n}` and never waits on a slow tab.
   - Tests are in `channel.rs`, `topic.rs` and `tests/bridge_http.rs`, and `cargo test` passes.
     Clippy is not installed for this toolchain, so it was not run.
   - The contract is documented in `fareward/README.md` under "Topics".
2. **Go client** `fareward/go/bridge.go`
   - `ConfigureBridge(url, internalAPIKey)`, `BridgeConfigured`, `BridgePublish`, `BridgeRPC` and
     `BridgePublishTopic`. `ConfigureBridge` adds the scheme to a bare host.
   - Tests in `bridge_test.go`, including the auth-header value the Rust tests also check.
   - `agent/bridge.go` now uses this client.
   - `core.ConfigureBridge`, `core.BridgeConfigured` and `core.BridgePublishTopic` are re-exported
     in `backend/core/fareward_api.go`.
   - Both startups configure the client: `backend/main.go` and `agent/cmd/server/main.go`.
3. **Backend publish:** `backend/dining/live_updates.go`.
   - `publishDiningLayoutChange` sends `{Type:"dining", LayoutID, Updated}` to topic
     `dining.<companyID>.<layoutID>`.
   - It runs after the commit of the round, line-status, charge and annul writes.
   - It has a 2 s timeout, and a failure is only logged; the write still succeeds.
   - Tested end to end: a stream opened from the browser received the round and annul nudges, and a
     topic of another company got a 403.

### Decisions taken (by the user, 2026-11-13)

- **Q1: how the browser finds the bridge URL.** Add a **new `[[endpoints]]` field**.
  - The existing `bridge` field stays as it is: it decides the agent's connection and falls back to
    `agent`.
  - Suggested name: `live`.
  - The Local endpoint uses `http://<host>:<sse_bridge.port>`, because `start.js` already runs
    fareward.
- **Q2: which bridge the local backend publishes to.** In `dev` mode the backend always publishes
  to `127.0.0.1:<sse_bridge.port>`.
  - Today it publishes to `sse_bridge.url`, which in `config.toml` is the remote
    `genix-bridge-5.un.pe`.
  - That bridge does not have `/publish-topic` yet, so every dining write in local dev logs an error.
- **Q3: how a VPS exposes the bridge.** Use **a path on the backend's own domain** (for example
  `/live/*`) routed by HAProxy to the bridge port.
  - It needs no new DNS name and no new certificate.
  - Do not add a separate bridge site.

### To do

1. **Q2**
   - Where `core.ConfigureBridge` is called in `backend/main.go`, use
     `http://127.0.0.1:<sse_bridge.port>` when the process runs in dev mode (`core.ReadDevArgument`
     or the existing dev flag).
   - That needs the port in `core.Env`: add `SSE_BRIDGE_PORT` from `file.SSEBridge.Port`.
2. **Q1: frontend endpoint field**
   - In `frontend/core/env.ts`, add `live` to the endpoint options. Keep it separate from
     `bridge`/`agent`, and give the Local endpoint `http://<host>:<sse_bridge.port>`.
   - Document the field in `config.example.toml` next to `bridge`.
   - Find where PUBLIC_ENDPOINTS is generated from `config.toml` (`rg PUBLIC_ENDPOINTS`) and pass
     the new field through.
3. **Frontend subscription** on the floor-service page and, later, the kitchen page.
   - Use `frontend/libs/sse-client.ts`. It is fetch-based, so it can send
     `Authorization: Bearer security.getToken(true)`, and it reconnects with backoff.
   - Build the channel token with `encodeChannelToken(companyID, userID, mintTabID())` from
     `frontend/core/agent/channel.ts`.
   - Use a tab id of its own. If it reused the agent's channel, the bridge (last connection wins)
     would close one of the two streams.
   - Request `topics=dining.<companyID>.<layoutID>`, and open a new stream when the layout changes.
   - On a `{Type:"dining"}` frame for the current layout, run `ordersService.fetchOnline()`.
   - Raise the fallback poll from 15 s to 30 s.
   - Keep the new code in its own file under `frontend/routes/dining/`, for example
     `dining-live.svelte.ts`.
4. **Q3: deployer**
   - In `scripts/configure.py` (around line 323), a VPS backend install (modes 1–2) currently runs
     `configure_fareward.py --service-only`, which installs no bridge site.
   - Change it so that the backend's HAProxy site sends `/live/*` to `127.0.0.1:<sse_bridge.port>`
     with the prefix stripped.
   - The bridge answers CORS preflights itself.
   - Keep the stream unbuffered with a long-enough timeout: the bridge pings every 20 s and
     HAProxy's timeouts are 60 s.
   - Then that endpoint's `live` is `https://<backend domain>/live`.
   - Update `config.example.toml`: the `[sse_bridge]` comment still says the bridge is only for
     Lambda.
   - Update `scripts/tests/test_configure_fareward.py` if the behaviour it tests changes.
5. **Docs**
   - In `docs/DINING_PLAN.md` §8, mark "What exists" as done and record Q1–Q3.
   - Update the floor-service `DOCUMENTATION.md` once the page refreshes live (skill
     `document-user-routes`, then validate it).
6. Steps after 4 in `DINING_PLAN.md` §12: the kitchen display, the QR page and the floor-map colours.

---

## Known issues not fixed

- **Settings that leave no bridge:** `sse_bridge.url` set to the Lambda URL means "no bridge" to the
  configure script, but `ConfigureBridge` treats it as a bridge. Dining would then publish to Lambda
  and log 404s.
- **Online orders:** that page still shows the second `[object Object]` error toast.
- **Warehouse Movements:** asset transfers show origin and destination swapped.
- **Unused flags:** commercial company flags 2–5 and 8 are read by nothing.
- **Old till APIs:** `backend/access.toml` still lists `POST.sale-order-v2` and `v3`.
- **Walk-in client:** if building the invoice fails after a new walk-in client was registered, that
  client stays registered.
- **Stale docs:** about 25 route docs were out of date before this work, because
  `frontend/core/modules.ts` was removed and the menu moved to `db/*.toml`. `credit-consumption`
  has no FILES section.
- **Carried over from earlier steps:**
  - The grid-size field rescales the layout.
  - "Cambios sin guardar" stays on after saving.
  - 763 rows of pending-delivery drift.
  - Access 30 lacks `GET.supply-material`.

## Local state left behind

- **Test data in the database:**
  - Sales 1008902, 32100, 32200 and 32300.
  - Client 193.
  - Table orders 65, 129 (charged) and 193 (annulled).
  - Product 10111, reactivated.
- **Running processes:** the last session left the backend running with
  `GENIX_CONFIG_FILE=/tmp/genix-live/config.toml`, a copy of `config.toml` whose `sse_bridge.url` is
  `127.0.0.1:14012`. Restart it normally once Q2 is implemented.
