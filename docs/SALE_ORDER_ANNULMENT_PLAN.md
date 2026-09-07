# Sale Order Annulment — Plan

Adds an **Anuladas** tab to `/sales/sale_orders_status` and a new endpoint that annuls a
sale order: refunds the collected money, returns delivered stock to the warehouse, rolls
back the day summary, and flips the order to `Status = 0`.

Status: **decisions resolved, ready to implement.** The answers to the four open questions
are folded into §3 and recorded in §8.

---

## 1. Scope

In scope:

- New backend endpoint `POST.sale-order-annul` (`backend/sales/sale_order_annul.go`).
- Cash reversal through the existing `finance/types.ApplyCashBankMovement`, withdrawn from
  an operator-chosen cash bank.
- Stock re-entry through the existing `logistics/types.ApplyMovimientos`.
- Sale-summary rollback through the existing `applyChangesToSaleSumary`.
- New **Anuladas** tab + a working "Anular" button on the order side panel, with a cash-bank
  selector and a mandatory reason.
- New `AnnulReason` column on `sale_order`.
- New "Anular Venta" sub-access on access 11, enforced on both sides.

Out of scope (see §9): credit notes for SUNAT, partial annulment, un-annulling.

---

## 2. What the code already does

### Sale order state

`backend/sales/types/sales.go:228` already defines the vocabulary, annulled included:

```go
OrderStatusAnnulled  = int8(0)   // 0 = Anulado
OrderStatusPending   = int8(1)   // 1 = Generado
OrderStatusPaid      = int8(2)
OrderStatusDelivered = int8(3)
OrderStatusCompleted = int8(4)
```

The delta index is already sized for it — `FixedValues{Col: e.Status, Min: 0, Max: 4}`
(`sales.go:123`) — so `Status.Equals(0).Delta(x)` is a valid query today. **No table
schema change is needed to read annulled orders.**

### Where the side effects live

| Effect | Function | Called from |
|---|---|---|
| Cash | `finance/types.ApplyCashBankMovement` (`cash_movement_apply.go:32`) | `sale_order_create.go:168` |
| Stock | `logistics/types.ApplyMovimientos` (`stock_movement_apply.go:152`) | `sale_order_create.go:221` |
| Summary | `applyChangesToSaleSumary` (`sale_summary.go:323`) | `updateSaleSummaryForChange` |

All three are leaf functions in `*/types` (or in the `sales` package itself) and take
signed amounts, so **a reversal is the same call with the sign flipped** — no new writer
code, which is what keeps this endpoint small.

### The two ledgers both index `DocumentID = sale.ID`

- `CashBankMovementTable` — `{Type: db.TypeLocalIndex, Keys: db.Cols(e.DocumentID)}`
  (`finance/types/cash_banks.go:131`).
- `WarehouseProductMovementTable` — same, `product-stock-movement.go:86`.

This is the pivot of the whole design: what to give back is **readable from the ledger**,
it does not have to be re-derived from `Status`.

### Frontend

- `SaleOrderGroup` (`sale_order_status.svelte.ts:53`) maps 1:1 to backend query params.
- `SaleOrdersService.handler` **drops every record with `ss <= 0`** (line 72) — annulled
  orders are filtered out client-side today.
- `onClickAnularSoloUI` (`+page.svelte:474`) is a placeholder that only shows a toast; the
  trash button in `titleSide` is already wired to it.
- `ConfirmWarn(title, message, ok, cancel, onOk)` from `$libs/helpers` is the project's
  destructive-action dialog (see `accounting/assets/+page.svelte:287`).

---

## 3. Design decisions

### D1 — Reverse from the ledger, not from `Status` **(recommended)**

Compute what to refund and what to return by reading the movement rows already written
for `DocumentID = sale.ID`, netting reversals that are already there:

```
refund_amount   = -( Σ Amount   where Type = SaleCollection(8)
                   + Σ Amount   where Type = SaleRefund(11) )
                   over every row with DocumentID = sale.ID, cash bank ignored (see D2)

stock_to_return = -( Σ Quantity where Type = 8   (sale delivery)
                   + Σ Quantity where Type = 9   (annulment re-entry) )
                   grouped by (warehouse, product, presentation, lot, serial, divisor)
```

Why:

- **Exact.** A sale delivered by lot/serial returns to *that* lot and *that* serial, with
  the divisor it was written under. Rebuilding from `DetailProductsIDs` would instead
  reconstruct an approximation, and would use the *current* `sale.WarehouseID` — which the
  update path is allowed to change (`sale_order_create.go:59`), so it can differ from the
  warehouse the goods actually left.
- **Idempotent.** Netting the reversal rows makes a retry after a partial failure a no-op
  instead of a second refund. There are no transactions here, so this is the only thing
  standing between a mid-flight error and money paid back twice.
- **Correct when nothing happened.** A sale that was never paid has no type-8 cash rows, a
  sale that was never delivered has no type-8 stock rows: both sums are zero and the branch
  costs one query that returns nothing. No `if delivered / if paid` ladder.

Cost: two extra queries (one per ledger) versus deriving from `sale.Status`.

Alternative rejected: mirror the `sale_order_create.go:186-215` loop with a positive
`lineQuantity`. Fewer queries, but it re-derives instead of reversing, and it inherits the
`Status`-vs-`DebtAmount` inconsistency noted in §9.

### D2 — The operator chooses the cash bank the refund is withdrawn from

The money does not have to leave the cash register or bank account that collected it: the
sale may have been collected in cash at a register that is now short, or days ago, and the
refund may be paid from a different register or by bank transfer. So `RefundCashBankID` is a
**required field of the request body**, and the panel offers the same active-cash-bank
selector the payment panel already uses.

The *amount* still comes from the ledger (the sum above), only the *destination* is chosen.
An operator cannot pick how much to give back — only where it comes from.

One `InternalCashMovement{CashBankID: RefundCashBankID, DocumentID: sale.ID,
Type: SaleRefund, Amount: -refund_amount}`. Idempotency survives this: the netting sums
every row with `DocumentID = sale.ID` regardless of which cash bank it sits on, so a
type-11 refund written to a different bank than the type-8 collection still nets to zero and
a retry is still a no-op.

Validation: reject a `RefundCashBankID` that is missing or inactive (`Status = 0`) whenever
`refund_amount > 0`. `GetCaja` (`cash_movement_apply.go:16`) already errors on a cash bank
that does not exist or belongs to another company; the `Status` check is ours to add, since
`ApplyCashBankMovement` does not make one. When the sale collected nothing, the field is
ignored rather than demanded.

Note there is no balance check anywhere in `ApplyCashBankMovement` — a refund can push
`CurrentAmount` negative and nothing prevents it. That is the existing behaviour for every
outflow type (withdrawals, supplier payments), so this plan does not change it.

### D3 — New enum values, no renumbering

- `finance/types.CashMovementTypeSaleRefund CashMovementType = 11` — the enum comment at
  `cash_banks.go:63` says the ids are fixed by the frontend list and must not be
  renumbered; 11 is the next free one. Mirror it in `cajaMovimientoTipos`
  (`frontend/routes/finance/cash-banks/cajas.svelte.ts:212`) as
  `{ id: 11, name: "Devolución (Anulación Venta)", group: 2, isNegative: true }`.
- Warehouse movement `Type = 9` — "Reingreso por anulación de venta", next after the `8`
  used for delivery (`sale_order_create.go:210`). This vocabulary has no Go enum today;
  the plan does **not** introduce one (out of scope), it only adds the literal with the
  same style of comment, plus the two labels to `movimientoTipos`
  (`logistics/warehouse-movements/warehouse-movements.svelte.ts:51`, which currently lists
  only 1 and 2).

### D4 — Lock the sale for the duration

`core.ActionAnnulSaleOrder LockAction = 4` in `core/enums.go`, acquired on `sale.ID` the
way `invoicing/invoice_api.go` does for `ActionInvoiceSaleOrder`. Two operators clicking
Anular at once would otherwise both read a zero refund-sum and both write one. D1's netting
makes a *sequential* retry safe; the lock is what makes a *concurrent* one safe.

### D5 — Order of operations: status flips last

1. Guards (annulled already? invoiced?).
2. Cash reversal and stock reversal, in parallel (`errgroup`, same shape as
   `PostSaleOrder`).
3. Summary rollback.
4. `db.Update` of the sale to `Status = 0`.

A failure anywhere before step 4 leaves the order un-annulled and retryable, and D1 makes
the retry safe. The reverse order would leave an order marked annulled with the money still
taken — the worse of the two failure modes.

### D6 — Refuse to annul an invoiced sale

`invoicing.FindBySaleOrder` (`emitter.go:168`) already answers "has this sale a live
electronic document?", skipping discarded and SUNAT-rejected ones. A sale with a live
invoice cannot simply vanish: SUNAT requires a credit note, which this project does not
emit yet (`DocTypeCreditNote` exists as a constant, nothing issues one).

`sales` may not import the `invoicing` body (`backend/docs/MODULE_BOUNDARIES.md`), so
**move `FindBySaleOrder` from `backend/invoicing/emitter.go` to
`backend/invoicing/types/`** and have both `invoicing` and `sales` call it there. It only
touches `core`, `db` and `InvoiceDocument`, and `invoicing/types` imports nothing from
`sales`, so no cycle. This mirrors why `ApplyCashBankMovement` lives in `finance/types`.

The reverse guard already exists: `invoice_api.go:163` refuses to invoice an annulled sale.

### D7 — One new column: `AnnulReason`, and nothing else

The reason is required (Q2), and it has nowhere to live today, so `sale_order` gains exactly
one column:

```go
AnnulReason string `json:",omitempty"`                        // SaleOrder
AnnulReason db.Col[*SaleOrderTable, string]                   // SaleOrderTable
```

It joins no index and no `FixedValues`, so it is an additive column change only.

No `AnnulledTime` / `AnnulledUser` alongside it: annulment is terminal — nothing writes to
the row afterwards — so the existing `Updated` / `UpdatedBy` *are* the "annulled at / by"
audit, and a second pair would only be a copy that can drift.

The handler must reject an empty or whitespace-only reason (`NEVER trust the client`), and
the confirm dialog must not let the operator submit without one. Cap it at a sane length
(200 chars) so the column cannot be used as free storage.

### D9 — "Anular Venta" as sub-access 2 of access 11

`access.toml:168` gains:

```toml
sub_accesses_ids = [2]
sub_accesses_names = ["Anular Venta"]
```

Id 2 because 1 is reserved for the implicit "Todos" (`core/usuario-accesos.go:65`).

Backend: `req.User.HasSubAcceso(11, 2)` at the top of the handler, right after the lock.
**This is the first `HasSubAcceso` call in the codebase**, so it also sets the pattern:
the check lives in the handler, reads the ids from named constants, and returns a plain
`req.MakeErr` — the daemon has already answered the question inside the frame that
authorized the route, so this is a map lookup, not a query.

Constants, for grepability, mirroring the frontend's `USERS_ACCESS_ID` convention
(`users-profiles.svelte.ts`):

```go
// backend/sales/types/sales.go
AccesoIDGestionVentas    = int32(11)   // access.toml [[access]] id 11
SubAccesoIDAnularVenta   = int32(2)
```

Frontend: the same two ids in `sale_order_status.svelte.ts`, and the trash button rendered
only when `security.checkSubAcceso(SALES_MANAGEMENT_ACCESS_ID, ANNUL_SALE_SUB_ACCESS_ID)`.
The UI check is convenience; the handler check is the enforcement.

**Migration consequence, needs to be said out loud:** an access lives in exactly one grant
payload (`genix-ui/security/SECURITY.md`), and grants without sub-accesses live in
`accesosComputed`, where `checkSubAcceso` cannot see them. So **every existing profile that
holds "Gestión Ventas" will not be able to annul until someone edits the profile and ticks
"Anular Venta" (or "Todos")**. Nothing breaks — annulment is new — but the box must be
ticked before the first operator can use it, and `recompute_accesos` must have re-run.

### D8 — `order-status=0` needs an explicit presence check

`GetSaleOrders` rejects `orderStatus <= 0` (`sale_orders_status.go:29`), so the annulled tab
cannot be expressed with the current parameter handling. Read the raw key instead:

```go
orderStatusValue, hasOrderStatus := req.Query["order-status"]
```

and accept `0` only when the key is present. Keeps one parameter for all three
status-pinned tabs rather than inventing an `annulled=1` flag.

---

## 4. Backend changes

**`backend/core/enums.go`**
- Add `ActionAnnulSaleOrder LockAction = 4`.

**`backend/finance/types/cash_banks.go`**
- Add `CashMovementTypeSaleRefund CashMovementType = 11`.

**`backend/invoicing/types/invoice_lookup.go`** (new, ~20 lines)
- Move `FindBySaleOrder` here verbatim from `emitter.go`; update the one caller in
  `emitter.go` and delete the original (pre-alpha: no shim).

**`backend/sales/types/sales.go`**
- Add the `AnnulReason` field to `SaleOrder` and its `db.Col` to `SaleOrderTable` (D7).
- Add `AccesoIDGestionVentas` / `SubAccesoIDAnularVenta` (D9).

**`backend/sales/sale_order_annul.go`** (new, ~180 lines)

```go
func PostSaleOrderAnnul(req *core.HandlerArgs) core.HandlerResponse
```

Body: `{ "ID": int64, "RefundCashBankID": int32, "Reason": string }`.

1. Parse; validate `ID > 0` and a non-blank `Reason` within the length cap.
2. `req.User.HasSubAcceso(AccesoIDGestionVentas, SubAccesoIDAnularVenta)` → else
   `"No tiene permiso para anular ventas."` (D9).
3. `core.AcquireLock(ctx, core.ActionAnnulSaleOrder, saleID, 2)`, deferred release.
4. Load the sale: `CompanyID.Equals(...).ID.Equals(saleID).Limit(1)`; error if missing.
5. Guard `sale.Status == types.OrderStatusAnnulled` → `"La venta ya está anulada."`
6. Guard `invoicing.FindBySaleOrder` → `"La venta tiene un comprobante electrónico
   emitido; requiere una nota de crédito."`
7. Read both ledgers by `DocumentID` and compute the net reversal (D1). This happens
   **before** the `errgroup` because the cash branch needs its sum to validate
   `RefundCashBankID`: demanding a cash bank on a sale that was never paid, or accepting an
   inactive one on a sale that was, are both decided by this number.
8. If `refundAmount > 0`: require `RefundCashBankID`, load it with `finance.GetCaja`, and
   reject `Status = 0` → `"La caja seleccionada está inactiva."` (D2).
9. `errgroup`:
   - `ApplyCashBankMovement` with the single negative movement — skipped when
     `refundAmount == 0`.
   - `ApplyMovimientos` with the sign-flipped `Type: 9` movements — skipped when empty.
   - `reverseSaleSummary(sale)` — see below.
10. `db.Update` the sale with `Status`, `AnnulReason`, `Updated`, `UpdatedBy` (mirroring the
    column list at `sale_order_create.go:248`; verify with the `static-project-validation`
    skill whether `Date` must accompany `Status` for the `Date+Status` index group).
11. Return the updated sale so the frontend can patch its row in place, exactly like
    `postSaleOrderUpdate` does today.

Split out as pure, row-taking functions so they unit-test without a query (§6):
`netCashRefundAmount([]CashBankMovement) int32` and
`netStockReturns([]WarehouseProductMovement) []InternalMovement`.

**`backend/sales/sale_summary.go`**
- Add `negateSummaryChanges([]ProductSummaryChange) []ProductSummaryChange` — flips
  `Quantity`, `SubQuantity`, `QuantityPendingDelivery`, `SubQuantityPendingDelivery`,
  `TotalAmount`, `TotalDebtAmount`, leaving `SubDivisor` alone.
- `reverseSaleSummary` = `MakeSummaryChangeFromOSaleOrder(sale, actionsOfStatus(sale.Status)...)`
  → negate → `applyChangesToSaleSumary(sale.CompanyID, sale.Date, changes, false)`.
  `actionsOfStatus` returns `[1]` plus `2` when `Status ∈ {2,4}` plus `3` when
  `Status ∈ {3,4}` — i.e. exactly the change set that was applied when the sale was
  created, so its negation cancels it. `applySummaryChangeToStats` already clamps at zero.

**`backend/sales/sale_orders_status.go`**
- Apply D8's presence check so `order-status=0` reaches
  `orderStatusToQuery = [OrderStatusAnnulled]`.
- Fix the removal list: it must exclude the status the tab is pinned to. Today the
  Finalizadas tab (`order-status=4`) builds `orderStatusToRemove = [4, 0]` and so tells the
  client to delete the very rows it just sent (§9).

**`backend/sales/main.go`**
- `"POST.sale-order-annul": PostSaleOrderAnnul`.

**`backend/access.toml`**
- Access 11 "Gestión Ventas" (`access.toml:168`):
  `backend_apis = "POST.sale-order,POST.sale-order-annul"`. Without this the route is
  unmapped and the gate rejects every caller — and `HasSubAcceso` would return `false`
  besides, since it only answers about accesses that authorized the route.
- Same entry: `sub_accesses_ids = [2]` / `sub_accesses_names = ["Anular Venta"]` (D9).

---

## 5. Frontend changes

**`sale_order_status.svelte.ts`**
- `SaleOrderGroup.ANULADO: 0`. The constructor compares with `===`, so `0` is safe there,
  but every `if (group)` style truthiness test must stay out.
- Constructor: `else if (group === SaleOrderGroup.ANULADO) route += '?order-status=0'`.
- `handler`: drop the `ss > 0` filter — it deletes exactly the records the new tab exists to
  show. Keep the `Created` sort.
- Bump `useCache.ver` 10 → 11 so no client mixes pre-filter cached payloads with post-filter
  ones.
- Add `AnnulReason` to `ISaleOrder`, and `ISaleOrderAnnulPayload
  { ID, RefundCashBankID, Reason }`.
- Add `postSaleOrderAnnul(payload)` next to `postSaleOrderUpdate`, `route:
  'sale-order-annul'`, same `refreshRoutes: ['sale-orders']`.
- Export `SALES_MANAGEMENT_ACCESS_ID = 11` and `ANNUL_SALE_SUB_ACCESS_ID = 2` (D9).

**`+page.svelte`**
- `options`: append `[SaleOrderGroup.ANULADO, 'Annulled|Anuladas']`.
- `getSaleOrderStatusName`: `case 0: return tr('Annulled|Anulada')`.
- The trash button renders only when
  `security.checkSubAcceso(SALES_MANAGEMENT_ACCESS_ID, ANNUL_SALE_SUB_ACCESS_ID)` and
  `selectedSaleOrder.ss !== 0`.
- Replace `onClickAnularSoloUI` with an annulment panel, because a reason and a cash bank
  cannot be collected from `ConfirmWarn` — it takes strings, not fields. The trash button
  switches the action area into an annul form (same slot the Pagar/Entregar panels occupy):
  a `SearchSelect` of active cash banks (`ss > 0`, defaulted to
  `selectedSaleOrder.LastPaymentCajaID`, shown only when the order was paid), a textarea for
  the reason, and a red confirm button that stays disabled until the reason is non-blank.
  `ConfirmWarn` then guards the final click, naming the order id and spelling out the three
  effects.
- Post behind the existing `isPostingSaleOrderAction` lock, then
  `applyUpdatedSaleOrderLocally`. Extend `saleOrderActionInProgress` and
  `getActionInProgressLabel` with `'anulacion'`.
- When `ss === 0`, replace the action area with an "Anulada el / por / motivo" block —
  `formatActionTime(Updated)`, `RecordByIDText` on `UpdatedBy`, and `AnnulReason` (D7).
  `canPaySaleOrder` / `canDeliverSaleOrder` already return `false` at `ss = 0`, so the
  Pagar/Entregar panels need no extra guard.

**`cajas.svelte.ts`** — add type 11 to `cajaMovimientoTipos`.
**`warehouse-movements.svelte.ts`** — add types 8 and 9 to `movimientoTipos`.

**Docs** — update `sale_orders_status/DOCUMENTATION.md` (the tab and the workflow) and
create `frontend/routes/sales/sale_orders_status/RATIONALE.md` (none exists) with D1, D5,
D6, D7 recorded per §1 of `CLAUDE.md`.

---

## 6. Tests

Pure functions, no DB:

- `negateSummaryChanges` + `actionsOfStatus`: for each status 1–4, creation change +
  annulment change sums to zero on every field.
- `netCashRefundAmount`: never paid → 0; paid once → the collected amount; paid then already
  refunded → 0 (idempotency); refunded to a different cash bank than the one that collected
  → still 0, since the sum ignores `CashBankID` (D2).
- `netStockReturns`: nothing delivered → empty; delivered once → exact reverse; delivered
  then already annulled → empty (idempotency); two lots on one product → two movements,
  each on its own lot.

Then `cd backend && go build ./... && go test ./sales/...`, and the
`static-project-validation` skill for both the new `AnnulReason` column and the `Status`
column-set question in §4 step 10.

---

## 7. Order of work

1. `core/enums.go`, `finance/types` enum, `sales/types` (column + access constants), move
   `FindBySaleOrder` → `invoicing/types`. Build + `static-project-validation`.
2. `sale_summary.go` helpers + their tests.
3. `sale_order_annul.go` + netting helpers + their tests.
4. `main.go`, `access.toml` (route + sub-access), `sale_orders_status.go` (tab query +
   removal-list fix).
5. Frontend service, then page, then the two label lists.
6. `RATIONALE.md`, `DOCUMENTATION.md`.
7. Tick "Anular Venta" on the profile being tested (D9) — nothing works before this.
8. Manual run against a delivered+paid order: check the chosen cash bank's balance drops by
   the collected amount, the warehouse row returns, the summary drops, and the order moves
   to the Anuladas tab. Then click Anular again on the same order and confirm it is refused
   at the `Status = 0` guard, and that a forged retry past the guard would net to zero.

---

## 8. Resolved decisions

**Q1 — Own sub-access for annulling? → Yes.** "Anular Venta", sub-access 2 of access 11.
See D9, including the first-caller pattern for `HasSubAcceso` and the consequence that
existing "Gestión Ventas" grants must be re-ticked.

**Q2 — Reason required? → Yes.** New `AnnulReason` column, mandatory and validated on the
backend. See D7.

**Q3 — May a delivered sale be annulled? → Yes.** The stock re-entry is the point. Noted
risk, accepted: when the goods physically left, the re-entry puts back stock that is not on
the shelf, and the correction is a physical count. The reason field (Q2) is what leaves a
trace of why.

**Q4 — "Closed" cash bank → the question is superseded by D2.** For the record: closed
means `Status = 0`, i.e. deactivated in `/finance/cash-banks` — there is no
"out of money" state, and no balance check exists anywhere in `ApplyCashBankMovement`.
Since the operator now picks the refund destination from the active list, the backend simply
rejects an inactive one.

**Q5 — Where does the annulled order's debt go?**
`DebtAmount` stays as it was on the annulled row. It is only ever read per-sale, so nothing
double-counts, but the Anuladas tab would display a debt on a void sale. Zero it on
annulment, or keep it as the historical record? Default if unanswered: keep it.

---

## 9. Found while reading — not fixed by this plan unless approved

- **`SaleOrderReprocessHandler` is a no-op.** `sale_summary_reprocess.go:99` returns an
  empty `FuncResponse` without calling `SaleOrderReprocess`. Every write schedules cron
  action 2 (`sale_order_create.go:268`) expecting a rebuild that never runs. This is why the
  incremental summary rollback in §4 is load-bearing rather than a nicety — there is no
  rebuild behind it to correct a drift. (`SaleOrderReprocess` itself already skips
  `Status < 1`, so it would handle annulled orders correctly if it were wired up.)
- **The Finalizadas tab deletes its own rows.** `sale_orders_status.go:37` builds
  `orderStatusToRemove = [OrderStatusCompleted, OrderStatusAnnulled]` unconditionally, so a
  delta sync of `order-status=4` returns the same ids in both `records` and
  `records_IDsToRemove`. §4 fixes this because the annulled tab hits the same path.
- **`Status` and `DebtAmount` can disagree.** `PostSaleOrder` calls `AddStatus(2)` whenever
  action 2 is included, but only appends summary action 2 when `DebtAmount == 0`
  (`sale_order_create.go:113` vs `:176`). A partially-paid sale is therefore `Status = 2`
  ("Pagado") with a non-zero debt. D1 sidesteps it for the refund (it reads the ledger, not
  the status), but `reverseSaleSummary` reads `Status`, so it will roll back a debt the
  creation path never recorded — clamped to zero by `applySummaryChangeToStats`, so it is
  contained, but it is a real inconsistency worth fixing separately.
- **Credit notes.** `types.DocTypeCreditNote` exists; nothing emits one. Until it does, D6's
  guard is the only correct answer for an invoiced sale.
- **`HasSubAcceso` has no callers yet.** D9 makes this handler the first. If a second
  destructive action needs the same treatment later, this is the shape to copy — and if the
  pattern turns out wrong, this is the only place to change.
