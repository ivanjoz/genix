# PLAN — One set of facts, three levels of accounting

Status: **proposal — not approved, nothing implemented.** Open decisions are in §10.

Supersedes, once approved, the "Periodic inventory … valuing the closing stock at stock × last
purchase price" decision in `frontend/routes/accounting/books/PLAN.md` §6 (see §4.1 for why).

---

## 0. Summary

The ERP has to serve two very different companies with **the same data model**:

- **The informal shop** buys, sells, and wants to know three things: what is in the warehouse,
  how much money is in each till and bank account, and how much it sold. It will never register
  a supplier invoice on time and will never do a physical count at month end.
- **The formal company** wants a valued permanent inventory (Kardex), an automatic Libro Diario,
  and automatic financial statements (Estado de Resultados, Estado de Situación Financiera).

The design that serves both:

1. **Facts are recorded once, in the same shape, whatever the company's level.** A sale, a
   reception, a payment, a stock adjustment — each is stored with everything any level could
   need, and none of them depend on the valuation method or on whether accounting is on.
2. **Every accounting output is a projection** — a pure function over the facts, parameterized
   by the company's chosen level and inventory method. Switching from periodic to perpetual does
   not change a single stored row; it changes which function reads them.
3. **A gap in the facts is never silent.** When a projection needs something a fact lacks — a
   purchase with no cost, a cash movement with no counter-account — it emits a **diagnostic
   line**: what is missing, which record, how much money is at risk, and how to fix it. Reports
   still add up; the unresolved amount is shown as its own explicit line.
4. **Projections are live while a period is open and frozen when it closes.** Closing a period
   is what turns "the system can show a Diario" into "the company can file a Diario".

### Does the requested idea make sense?

Yes, with three refinements that the rest of this document builds on:

- **Both methods need a cost on every inflow, not only periodic.** They differ in *what else*
  they need and in *how far a missing cost spreads* (§4.4). Periodic additionally needs an
  opening value and a closing quantity; perpetual needs the cost to be known before the goods
  leave, or it has to accept provisional costs that are recomputed later.
- **A failure line must carry an amount and a fix link**, not only a message, and the report must
  still reconcile (§5). "12 units received without a cost on OC 45, ~S/ 120 estimated, register
  the supplier's invoice" is actionable; "missing cost" is not.
- **The method reported to SUNAT is fixed for the fiscal year.** The engine can compute both at
  any time (useful as a what-if), but the numbers that feed the books are computed with the
  method the company has locked for that year (§3.4).

---

## 1. Where the system stands today

Verified against the code in September 2026. File references are given so each claim can be
checked.

### 1.1 Facts that already exist and are good enough

| Fact | Where | Accounting use |
| --- | --- | --- |
| Electronic comprobante per sale: base, IGV, exonerado/inafecto, total, currency, state, frozen buyer | `invoicing/types/invoice_document.go` | Revenue and IGV (70 / 40111 / 12). Feeds the RVIE today |
| Sale order: lines, prices, currency, commercial exchange rate, debt, due date | `sales/types/sales.go` | Receivable (12) and revenue for sales without comprobante |
| Supplier comprobante on purchase orders, expenses and assets: type, series, number, issue date, base, IGV, untaxed, other, currency, rate, frozen supplier | `finance/types/purchase_document.go` and the three owning tables | Purchases (60 / 40111 / 42). Feeds the RCE today |
| Expense with accrual state: amount owed, paid, due date, type (simple / inventory / depreciation) | `finance/types/expenses.go` | Expense and payable (6x / 42) |
| Fixed asset: acquisition value, payment state, straight-line depreciation, disposal | `accounting/types/asset.go`, `accounting/depreciation.go` | 33 / 465 / 68 / 39 |
| Cash movement with a type, a document, and a PCGE account for user-classified types | `finance/types/cash_banks.go`, `cash_movement_account.go` | The 10 side of almost every entry |
| Append-only stock ledger with date, quantity, document, running warehouse balance, and a value on inflows | `logistics/types/product-stock-movement.go` | Kardex in physical units; the base of any valuation |
| Official exchange rates from BCRP | `finance/types/bcrp_rates.go`, `exchange_rate.go` | PEN restatement of USD operations |

### 1.2 What is missing

- **No cost on outflows and no valuation.** `MonetaryValue` is only filled when the movement has
  a `Price` (`stock_movement_apply.go:420`), which only inflows have. No average cost, no valued
  balance, no cost of sales anywhere.
- **Inflow values are not a cost.** Purchase-order line prices include IGV
  (`PurchaseOrderCreate.svelte:139`, `invoicing/types/sale_order_to_cpe.go:24` — the whole ERP
  prices gross), and they are in the order's currency with no conversion to PEN.
- **Movement `Type` is too coarse to say which document a movement came from.** Only four
  values exist: 1 = any inflow, 2 = any outflow, 8 = sale, 9 = sale annulment
  (`warehouse-movements.svelte.ts:51`). Purchase-order receptions, inventory expenses, asset
  acquisitions and manual entries all write type 1, with a `DocumentID` from their own
  sequence — movement `DocumentID = 15` may be OC 15, expense 15, asset 15 or nothing. The
  movements page labels all of them "Entrada Manual". Asset disposals, transfers and serial
  corrections fall back to 1/2 as well. There is no opening-stock type and no
  internal-consumption type, so supplies bought through an inventory expense only ever go up.
- **No account mapping for expenses.** The ten expense categories (`finance/expenses.go:19`,
  `expenses.svelte.ts:68`) name no account. Two of them cannot be plain expenses at all: paying
  IGV settles 40111, and payroll needs 41/40 with its withholdings.
- **No opening balance**, no manual journal entries, no period close, no period lock. Any
  source record can be edited after its month was reported.
- **No place for** loans, income tax, payroll provisions, exchange-rate differences, or the sale
  price of a disposed asset.
- **No company-level setting** for the tax regime, the accounting level or the inventory method.

### 1.3 Bugs that corrupt facts — prerequisites

These make the facts wrong, so no projection can be right on top of them. They are fixed first,
whatever level is being built.

**Status: all six fixed (phase 0).**
1. `expandStockTransfers` writes a transfer as two rows, each naming the other warehouse in
   `WarehouseRefID`. The inflow is priced at the origin's average.
2. The type comes from the delta, and a replace-to-zero applies.
3. A PO can be paid while Confirmed or Fulfilled, from a register in the order's currency.
4. The server owns the sale debt. An unpaid sale owes its total, a payment collects the drop in
   debt, and only a zero debt marks it Pagado.
5. Only 3, 4, 5 (negative) and 7 (positive, with a PCGE account) are manual cash types.
6. `TaxAmount` is computed per line by `invoicing.SaleOrderTaxAmount`.

The descriptions below are kept as the record of what was wrong.

1. **An asset transfer adds stock at the origin warehouse.** `PutAssetTransfer` sets
   `DestWarehouseID` (`accounting/asset_api.go:308`) and a positive quantity, but
   `ApplyMovimientos` never reads `DestWarehouseID` and never writes `WarehouseRefID`. Net
   effect: +N units at the origin, nothing at the destination.
2. **A stock adjustment gets the wrong type, and adjusting to zero does nothing.** The type is
   derived from the requested *target* quantity, not from the delta
   (`stock_movement_apply.go:364`), so lowering stock from 10 to 5 is recorded as an inflow. And
   a target of 0 is filtered out as a no-op before `ReplaceQuantity` is considered (`:165`).
3. **A purchase order can only be paid while Confirmed** (`purchase-order-management.go:425`).
   Once the goods are received it is Fulfilled, and its remaining `DebtAmount` can never be
   paid — the ordinary credit purchase (receive now, pay in 30 days) is broken.
4. **A later sale payment trusts the client.** The cash collected is computed as
   `TotalAmount − DebtAmount` (`sale_order_create.go:217`) with `DebtAmount` taken from the
   request (`:71`). It works only because the frontend always pays in full; a partial payment
   would count the first part twice.
5. **Manual cash movements can claim a document type without a document.** `PostCashBankMovement`
   accepts any type (`finance/cash_banks.go:277`), so a "Pago Proveedor" or "Cobro (Venta)" can
   be created by hand with no `DocumentID`: the cash moves and no payable or receivable is
   settled.
6. **`SaleOrder.TaxAmount` is whatever the client sent.** The server never computes it. It only
   matters for sales without a comprobante, which are exactly the ones that need it.

---

## 2. The principle: facts once, projections many

```
               ┌───────────────── FACTS (stored, method-agnostic) ─────────────────┐
               │ sales · comprobantes · purchase orders · receptions · expenses    │
               │ assets · cash movements · stock movements · opening balances      │
               └──────────────┬───────────────────────────────┬────────────────────┘
                              │                               │
                  company settings: level, method, regime     │
                              │                               │
      ┌───────────────────────▼──────────┐      ┌─────────────▼──────────────────┐
      │ Level 0  quantities · cash ·     │      │ Level 1  valuation (periodic   │
      │          sales volume            │      │          or perpetual) → COGS, │
      │          (never fails)           │      │          Kardex 13.1, margin   │
      └──────────────────────────────────┘      └─────────────┬──────────────────┘
                                                              │
                                                ┌─────────────▼──────────────────┐
                                                │ Level 2  posting rules →       │
                                                │          journal → ledger →    │
                                                │          trial balance → EEFF  │
                                                └─────────────┬──────────────────┘
                                                              │ period close
                                                ┌─────────────▼──────────────────┐
                                                │ FROZEN: journal entries,       │
                                                │ inventory snapshot, lock       │
                                                └────────────────────────────────┘
          every projection emits DIAGNOSTICS for the facts it could not use
```

Five rules, in order of importance:

1. **Stored facts never depend on the method or the level.** A company that turns accounting on
   in March gets its March books from the same rows it has been writing since January.
2. **Projections are pure functions** — no DB, no clock, no Svelte — so the valuation, the
   posting rules and the statements are unit-testable in isolation (CLAUDE.md §4).
3. **Gaps are diagnostic lines, never silent omissions and never guesses written back into the
   data.** A projection may *estimate* the amount at risk for display; it never stores the
   estimate as if it were a fact (CLAUDE.md §3, "fix the data, never patch the flow").
4. **Reports reconcile.** A statement with gaps still adds up: the unresolved amount is its own
   line, not a hidden difference.
5. **A closed period is immutable.** Its journal and inventory snapshot are frozen, and the
   source records dated inside it are locked. Corrections go into the open period as
   adjustments.

---

## 3. Accounting levels

### 3.1 The three levels

| | **Level 0 — Operational** | **Level 1 — Valued inventory** | **Level 2 — Accounting** |
| --- | --- | --- | --- |
| For | The informal shop | A shop that wants its real margin | A company that files books |
| Requires | Nothing beyond selling and buying | A cost on every inflow (§4) | Level 1 + account mapping + opening balance + period close |
| Stock | Quantities per warehouse (exists) | + value per product, cost of sales | same |
| Cash | Balance per till and bank (exists) | same | + reconciled against the 10 accounts |
| Sales | Volume, by day, product, client (exists) | + gross margin per sale, product, day | + revenue per the books |
| Result | **Cash result**: collected − paid, labelled "not profit" | **Management P&L**: sales − COGS − expenses − depreciation | **Estado de Resultados** + **Estado de Situación Financiera** |
| Kardex | Formato 12.1 (physical units) | + Formato 13.1 (valued) | same |
| Books | RVIE + RCE (exist) | same | + Libro Diario (5.2 or 5.1), Mayor, plan contable 5.4 |
| Diagnostics shown | none about cost | cost diagnostics (`INV-*`) | all (`INV-*`, `ACC-*`) |
| What a gap does | nothing | makes figures provisional | blocks the period close |

Level 0 is not "less correct" — it just asks fewer questions. Everything it shows is computed
from facts that are always present.

### 3.2 Company settings

| Setting | Values | Notes |
| --- | --- | --- |
| `AccountingLevel` | 0 · 1 · 2 | Can be raised at any period start (§3.4) |
| `InventoryMethod` | 1 periodic · 2 perpetual | Only meaningful at level ≥ 1. Locked per fiscal year |
| `CostFormula` | 1 weighted average (first) · 2 FIFO (later) | NIC 2 and art. 62 of the Reglamento de la LIR accept both; **last purchase price is not an accepted formula** |
| `TaxRegime` | NRUS · RER · RMT · General | Decides which IGV rules and which books apply |
| `IgvCreditEligible` | derived from the regime | NRUS gets no IGV credit, so IGV becomes part of cost (§4.5) |

Where these live (company flags vs. the company config blob) is open decision D1.

### 3.3 Levels vs. what SUNAT requires

Confirm with an accountant; thresholds are in UIT of the fiscal year. Sources: art. 65 LIR,
art. 11 D.L. 1269 (RMT), art. 35 of the Reglamento de la LIR, and
`frontend/routes/accounting/docs/SUNAT-Libro-Diario-Formato-Simplificado.md` §2.

| Company | Books required | Kardex required | Minimum level |
| --- | --- | --- | --- |
| NRUS | none | none | 0 |
| RER | Ventas + Compras | per income, see below | 0 (1 if > 300 UIT) |
| RMT/General ≤ 300 UIT | Ventas + Compras + Diario Simplificado (5.2) | none | 2 |
| 300 – 500 UIT | + Diario (5.1) + Mayor | Inventario permanente **en unidades físicas** (12.1) | 2 |
| 500 – 1 500 UIT | + Inventarios y Balances | 12.1 | 2 |
| > 1 500 UIT | same | Inventario permanente **valorizado** (13.1) → perpetual | 2 + perpetual |
| > 1 700 UIT | contabilidad completa | 13.1 | 2 + perpetual |

The system does not enforce these — it cannot know the company's income before the year ends —
but the settings page shows the recommendation for the declared regime and the current year's
sales.

### 3.4 Moving between levels and methods

- **Raising the level** happens at a period start. Going to level 1 needs an **opening inventory
  valuation** (quantity × unit cost per item, §4.6 type 12); going to level 2 also needs an
  **opening entry** (cash, receivables, payables, assets, capital — §6.6). Until they exist,
  the projections emit `INV-004` / `ACC-007`.
- **Lowering the level** never deletes anything; the higher projections simply stop being shown.
  Closed periods stay closed.
- **Changing the inventory method** takes effect from the start of a fiscal year. The engine can
  still compute the other method at any time as a what-if report, clearly labelled as such.

---

## 4. Inventory valuation — method-agnostic

### 4.1 What both methods need

Both periodic and perpetual answer the same equation, per item and per period:

```
opening value + inflows value  =  cost of what left  +  closing value
```

They differ only in **which term is measured and which is solved for**:

- **Periodic** measures opening, inflows and the closing *quantity*, values the closing stock with
  the cost formula, and **solves for the cost of what left** (COGS = opening + purchases − closing).
- **Perpetual** measures every outflow at the running cost when it happens, and **the closing
  value falls out** of the running balance.

So **both need the value of every inflow**. That is the refinement to "periodic needs every
purchase to have a cost": perpetual needs it too, and needs it more urgently.

Why not the current plan (periodic at "stock × last purchase price"): last purchase price is a
replacement-cost approximation, not a cost formula accepted by NIC 2 or by art. 62 of the
Reglamento de la LIR, and it would make the P&L swing with the latest invoice rather than with
what was actually sold.

### 4.2 Periodic

Per item, per period:

```
opening    = closing snapshot of the previous period (qty, value)        — or the opening valuation
inflows    = Σ costed inflows of the period (purchases, opening stock, surpluses, returns)
closing qty= ledger balance at period end        (default)
           | physical count at period end        (when the company records one, §10 D5)
unit cost  = (opening value + inflows value) / (opening qty + inflows qty)   — monthly weighted average
           | FIFO over the period's inflow layers                          — when CostFormula = FIFO
closing    = closing qty × unit cost
COGS       = opening value + inflows value − closing value
```

What periodic **tolerates**, which is why it suits the informal shop at level 1:

- outflows with no cost and no document detail;
- sales, consumption of supplies, shrinkage and theft all mixed together — they all land in COGS;
- a purchase whose cost arrives late, as long as it arrives before the period is closed;
- no per-sale margin (it cannot give one, by construction).

What periodic **cannot tolerate**:

- an inflow with no cost → that item's unit cost for the period is undefined (`INV-001`);
- no opening value for an item that had stock before the first recorded period (`INV-004`);
- a closing quantity that is not the real one. The ledger balance is used by default; when it
  diverges from reality, only a count (a stock adjustment) fixes it.

### 4.3 Perpetual

The ledger is replayed in chronological order, per item (and warehouse, §4.7):

```
on inflow  (qty q, cost c): value += q·c ; qty += q ; avg = value / qty     — moving weighted average
on outflow (qty q)        : cost = q·avg ; value -= cost ; qty -= q         — stamped on the outflow
on transfer               : outflow at origin avg, inflow at destination with that same cost
on return of a sale       : inflow at the cost the original outflow left with (found by DocumentID)
```

What perpetual **gives** that periodic cannot: cost of sales per sale and per day, gross margin
per product, a valued Kardex 13.1, and shrinkage valued separately from the cost of sales.

What perpetual **needs**:

- the cost of an inflow known before the goods leave — or it accepts a **provisional** cost that is
  recomputed when the final one arrives (§4.5);
- no negative stock. The stock engine already rejects negative balances
  (`stock_movement_apply.go:474`), which is what makes a moving average well defined;
- correct movement types and documents, because each outflow's cost goes to a different account
  (sale → 69, shrinkage → 659, consumption → 656…).

### 4.4 What a missing cost does, side by side

| Situation | Periodic | Perpetual |
| --- | --- | --- |
| One reception of item X on day 3 has no cost | X's unit cost for the **whole month** is undefined → COGS of X is provisional | Every outflow of X **from day 3 on** is provisional, and so is every later period until fixed |
| The cost arrives on day 20 | Nothing to recompute; the month uses it | Outflows of X on days 3–20 are recomputed on replay (free while the period is open) |
| The cost arrives after the period closed | Not allowed: the close was blocked by `INV-001` | Same |
| Opening stock of X has no value | COGS of X undefined for the first period | Every outflow of X undefined until the opening lot is consumed |
| Supplies are never marked as consumed | Correct: consumption falls into COGS via the closing quantity | Stock of supplies only grows → `INV-006` |
| Shrinkage never recorded | Silently absorbed in COGS (correct for periodic) | The ledger overstates stock until a count; the count adjustment is then valued |

Because projections are computed while the period is open, recomputing is only a replay — no
stored cost ever has to be rewritten.

### 4.5 The cost of an inflow

**The cost of an inflow is a property of its source document, not of the movement.** The movement
keeps the provisional value it was written with (`MonetaryValue`, as today); the projection reads
the document to find the final one. That keeps the ledger append-only and lets a late invoice
correct the cost without touching the ledger.

Rules, applied by one pure function `ResolveInflowCost(movement, document) → (unitCostPEN, status)`:

1. **Currency:** restate in PEN at the official SBS/BCRP selling rate of the document's issue date
   (the rate RCE field 27 already stores on the document). The commercial spread never enters the
   cost.
2. **IGV:** subtract it when the company can take it as credit **and** the document gives the
   right to it (factura `01`, with `TaxAmount > 0`). Otherwise — boleta, ticket, NRUS, a
   purchase with no comprobante — the IGV is part of the cost.
3. **Allocation:** the comprobante is order-level (`TaxableAmount` on the order); lines carry
   gross order prices. Each received line's cost is its gross line value × (comprobante net ÷
   comprobante gross). Any difference between what was ordered and what was invoiced lands
   proportionally on the lines received.
4. **Status:**
   - `final` — the document has a registered comprobante (or is an expense, asset or opening lot
     whose own value is final);
   - `provisional` — a reception with no comprobante yet: order price, converted and net of IGV at
     the standard rate, emitting `INV-002`;
   - `missing` — no price anywhere (a manual upward adjustment with no cost, a stock entry with no
     document): `INV-001`.

Inflow sources and where each gets its cost:

| Source | Today | Cost source |
| --- | --- | --- |
| Purchase-order reception | type 1, OC id, price = order line gross | the OC's comprobante, allocated (rule 3); provisional until registered |
| Inventory expense (supplies) | type 1, expense id, price = unit price | the expense's comprobante |
| Opening stock | **does not exist** — users adjust stock instead | the unit cost typed on the opening movement (new, §4.6 type 12) |
| Upward adjustment (surplus) | type 1 (or wrongly 1 on a decrease, bug 2), no price | optional typed cost; else current average (perpetual) or the period unit cost (periodic) |
| Sale return / annulment | type 9, sale id, no price | perpetual: cost of the original outflow; periodic: period unit cost |
| Transfer in | broken (bug 1) | cost of the matching transfer out |
| Asset acquisition | type 1, asset id | **excluded**: fixed assets are account 33, not inventory |

### 4.6 Movement catalog

**Status: implemented (phase 1)**, in `backend/logistics/types/stock_movement_type.go` and mirrored
in `frontend/core/stock-movement-type.ts`.

- **Engine contract.** The engine enforces the DocumentID rule and each type's direction. The
  grouped view is widened.
- **Deliberately not yet built:**
  - **Type 11:** reserved until a stock transfer screen exists; asset transfers use 7.
  - **`StockMovementSunatOperation`:** built together with the Kardex report, when the Tabla 12
    codes are verified.
- **Manual reasons.** Types 12, 13 and 14 are chosen on the stock page's save summary.
  - The user picks one reason for the save's increases and one for its decreases.
  - Opening stock gets past company flag 6, and always needs its cost.
- **Tabla 5 and Tabla 6 are derived, not stored** (settled).
  - Tabla 5: a product is `01`, a supply `06`.
  - Tabla 6: from `UnitID` (Kg → `KGM`, g → `GRM`, Libras → `LBR`, none → `NIU`).
  - Both are written with the Kardex header.
- **To check against Tabla 12:** code `28 Ajuste por diferencia de inventario` may fit types 1/2
  better than `99`.

**The movement type is the only discriminator.** `WarehouseProductMovement.Type` becomes an owned
enum, `StockMovementType` in `logistics/types`, the same pattern `CashMovementType` already uses
(`finance/types/cash_banks.go:63`). Each value is written by exactly one code path and names
exactly one source table, so two things are **pure functions of the type** and are not stored:

- `StockMovementDocumentTable(type)` — which table `DocumentID` points into (or none);
- `StockMovementSunatOperation(type)` — the SUNAT **Tabla 12** code for the Kardex.

**The type is a logistics fact, not an accounting classification.** It answers "what happened to
the stock, and which document did it" — which every level needs, level 0 included (a loss report
needs shrinkage apart from a manual exit; an initial load needs opening stock apart from a
purchase). Ownership follows that split:

| Lives in | What | Why there |
| --- | --- | --- |
| `logistics/types` | `StockMovementType`, `StockMovementDocumentTable(type)` | which document caused a movement is stock knowledge; every writer already imports this package |
| `accounting/types` | `StockMovementSunatOperation(type)`, and the posting rules that read the type (§6.2) | SUNAT codes and accounts are accounting knowledge; `accounting` may import `logistics/types`, never the reverse |

A new type is only added when there is a new writer, a new source document, or a new
operational meaning. Never for an accounting reason alone: whether a merma posts to 6911 or 659
(D6) is decided in `accounting/types`, not by splitting the type.

The internal type has to be finer than Tabla 12, because Tabla 12 is many-to-one: an OC
reception and an inventory expense are both `02 Compra`, but their `DocumentID`s point into
different tables. Storing the Tabla-12 code would lose the document again.

`ApplyMovimientos` enforces the contract: a type that names a table requires `DocumentID > 0`,
and a type that names none requires `DocumentID = 0`. The type is always set by the writer; the
current fallback that derives 1/2 from the quantity's sign (`stock_movement_apply.go:364`, bug 2)
is deleted.

Ids 1, 2, 8 and 9 are persisted and keep their meaning; the rest are proposed numbering. Verify
Tabla 12 codes against the current table before implementing.

| Type | Movement | Written by | `DocumentID` → | Tabla 12 | Sign | Cost |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | Manual entry | stock adjustment, upward | — | 99 Otros | + | typed, or running average |
| 2 | Manual exit | stock adjustment, downward | — | 99 Otros | − | computed |
| 3 | Purchase-order reception | `receivePurchaseOrder` | purchase_order | 02 Compra | + | document (§4.5) |
| 4 | Supplies purchase | `PostInventoryExpense` | expense | 02 Compra | + | document (§4.5) |
| 5 | Asset acquisition | `PostAsset` | asset | — | + | excluded from inventory valuation |
| 6 | Asset disposal | `PutAssetDisposal` | asset | — | − | excluded |
| 7 | Asset transfer | `PutAssetTransfer` | asset | — | −/+ | excluded |
| 8 | Sale delivery | `PostSaleOrder` | sale_order | 01 Venta | − | computed (perpetual) |
| 9 | Sale annulment return | `PostSaleOrderAnnul` | sale_order | 05 Devolución recibida | + | cost of the original outflow |
| 10 | Asset serial correction | `PutAssetEdit` (out old serial, in new) | asset | — | −/+ | excluded |
| 11 | Stock transfer | **new** warehouse-to-warehouse transfer | — | 11 Transferencia | −/+ | computed at origin |
| 12 | Opening stock | **new** | — | 16 Saldo inicial | + | typed |
| 13 | Internal consumption | **new**, for supplies | — | 10 Salida a producción / consumo | − | computed |
| 14 | Shrinkage (merma) | stock adjustment with reason merma | — | 13 Mermas | − | computed |

Adjustment reasons (count, merma, surplus, correction) are therefore types, not a separate
column. A future supplier return would be one more type pointing at purchase_order (`06`).

Products also need two SUNAT attributes for the Kardex headers: **tipo de existencia** (Tabla 5 —
`01` mercaderías for `Status = 1`, `05`/`06` materiales auxiliares/suministros for `Status = 2`)
and the **SUNAT unit of measure** (Tabla 6, e.g. `NIU`, `KGM`) mapped from `Product.UnitID`.

### 4.7 Granularity

- **The item** is `(ProductID, PresentationID)` — the same key the stock rows use.
- **The average is kept per warehouse** — per stock row, `(Warehouse, Product, Presentation)`.
  Settled and implemented: `ProductStock.AverageCost` is the moving weighted average, moved by
  `ApplyMovimientos` on every costed inflow and replayed by `RecalcProductStockByMovements`. It is
  an operational figure written at movement time; the projection here still resolves costs from
  the documents (see `backend/logistics/RATIONALE.md`).
- **Fact-quality switches (implemented).** Company flag 6 blocks manual stock increases, so stock
  only enters through a purchase order; decreases stay allowed. Flag 9 requires a unit cost (PEN)
  on every manual increase. Both are enforced by the stock engine on the delta. A company with
  flag 6 or 9 on cannot produce `INV-001` from the stock page.
- **Sub-units** are valued at a pro-rata of the unit cost, exactly as
  `core.QuantityValueAtUnitPrice` already does on inflows.

### 4.8 Kardex outputs

**Formato 12.1 — Registro de Inventario Permanente en Unidades Físicas** (level 0). Per item:
date, document type/series/number (Tabla 10), operation type (Tabla 12), inflow qty, outflow
qty, balance. Everything comes from the ledger once the §4.6 types exist; the movement type
names the source table, and the document's SUNAT type/series/number are joined from it (the sale's comprobante,
the OC's supplier comprobante, or `00` + internal number when there is none).

**Formato 13.1 — Registro de Inventario Permanente Valorizado** (level 1, perpetual). Header:
period, RUC, name, establecimiento, item code, tipo de existencia (Tabla 5), description, unit
(Tabla 6), valuation method. Rows: the 12.1 columns plus unit cost and total cost for inflows,
outflows and balance. Every row whose cost is provisional or missing is marked, and the file
cannot be exported while any is.

### 4.9 Snapshots and cost of computing

Replaying the whole ledger on every request does not scale indefinitely. A **closing snapshot**
per period and item — `(qty, value)`, plus the FIFO layers when `CostFormula = FIFO` — is
written when the period is closed. A projection then replays only from the last snapshot to the
requested date. For a company that never closes periods (level 1 without closes), see D10.

---

## 5. Diagnostics — the failure lines

### 5.1 Shape

```go
type Diagnostic struct {
	Code          string // "INV-001"
	Severity      int8   // 1 info · 2 provisional (figures may change) · 3 blocking (prevents close)
	SourceKind    int8   // the source table of the offending record (for stock, derived from its movement type)
	SourceID      int64
	ProductID     int32  // when it is an inventory diagnostic
	Date          int16
	AmountAtRisk  int64  // cents, PEN — an ESTIMATE, never stored as a fact
	EstimateBasis int8   // how AmountAtRisk was estimated (last known cost, order price, …)
}
```

The message and the fix action are resolved from `Code` on the frontend (bilingual, `tr()`), and
each row links to the record that fixes it (the OC's "register comprobante" action, the expense
form, the adjustment…).

### 5.2 Catalog

Severity shown as *level 1 / level 2*: `–` not shown, `P` provisional, `B` blocks the close.

| Code | Condition | L1 | L2 | Fix |
| --- | --- | --- | --- | --- |
| INV-001 | Inflow with no cost (§4.5 status `missing`) | P | B | Type the cost, or register the document |
| INV-002 | Reception whose OC has no comprobante (provisional cost) | P | B | Register the supplier's comprobante on the OC |
| INV-003 | Foreign-currency cost with no exchange rate for the date | P | B | Load the rate in `/finance/exchange-rate` |
| INV-004 | Item has stock before its first costed movement (no opening value) | P | B | Record an opening-stock movement (type 12) |
| INV-005 | Upward adjustment with no cost, valued at the running average | info | P | Type the cost if the average is not right |
| INV-006 | Perpetual: supplies with inflows and no consumption in the period | info | info | Record consumption (type 13), or use periodic |
| INV-007 | Return with no matching original outflow | P | B | Link the return to its sale |
| INV-009 | Periodic: closing quantity from the ledger, no count recorded this period | info | info | Record a physical count |
| INV-010 | Negative running quantity on replay (should be impossible, see the engine guard) | B | B | Bug — investigate |
| ACC-001 | Expense category with no account, or "Taxes" with no tax chosen | – | B | Choose the account / tax |
| ACC-002 | Cash movement of a document-bearing type with no document (bug 5) | – | P | Link it to the document, or reclassify |
| ACC-003 | Cash movement of type 1 "unspecified": no counter-account | – | B | Reclassify the movement |
| ACC-004 | USD cash movement, receivable or payable with no official rate for the date | – | B | Load the rate |
| ACC-005 | Sale without comprobante: IGV split computed at 18 % of the gross | – | info | Issue the comprobante, if one is due |
| ACC-006 | Purchase or expense without comprobante: cost with no tax support (reparo) | – | info | Get the comprobante |
| ACC-007 | No opening entry for the first accounting period | – | B | Enter the opening balance (§6.6) |
| ACC-008 | Posting that does not balance | B | B | Bug — the posting rules are unit-tested to prevent it |
| ACC-009 | Collections exceed the receivable, or payments exceed the payable | – | B | Fix the payment or the document |
| ACC-010 | Annulled sale whose comprobante has no credit note or baja | – | B | Issue the credit note |
| ACC-011 | Disposed asset with no disposal value | – | P | Enter the sale value (0 if scrapped) |
| ACC-012 | Payroll paid as a plain expense (no withholdings, no EsSalud) | – | info | Use a manual payroll entry |

### 5.3 How reports show them

Reports still reconcile: the unresolved part is its own line, never folded into a total.

```
MANAGEMENT P&L — SEP 2026 (periodic, weighted average)                         S/
  Net sales                                                              48,210.00
  Cost of sales                                                         −31,480.00
    ⚠ Unresolved: 3 items with receptions without cost  (INV-001 ×4)     ~−1,120.00  estimated
  Gross margin                                                           16,730.00   provisional
  Operating expenses                                                    −9,300.00
  Depreciation                                                            −420.00
  Result                                                                  7,010.00   provisional
  ─────────────────────────────────────────────────────────────────────────────
  4 diagnostics · 1 blocks the close · [review]
```

The Kardex 13.1 marks each provisional row; the Diario lists the entries it could not post, with
their amount, below the balanced entries; the trial balance shows the unresolved amount as a
"pending classification" line outside the accounts, so debits still equal credits.

### 5.4 Close rules

- A period can be closed at level 2 only when it has **no blocking diagnostic**.
- At level 1 an inventory-only close is allowed with provisional diagnostics, if the user
  confirms; the snapshot records that it closed provisional (D10).
- Level 0 has no close.

---

## 6. The accounting layer (level 2)

### 6.1 Chart of accounts

The PCGE is fixed in code, as already decided (books `PLAN.md` §6), in one file that also feeds
PLE 5.4. Only the accounts the posting rules use are listed; sub-account digits are to be
confirmed with an accountant (D8).

| Account | Name | Used by |
| --- | --- | --- |
| 101 · 1041 | Caja · Cuentas corrientes operativas | every cash movement (`CashBank.Type` 1/2) |
| 1212 | Facturas, boletas y otros comprobantes por cobrar — emitidas | credit sales |
| 142 · 1419 | Accionistas · Otras cuentas por cobrar al personal | withdrawals, shortages (exists) |
| 2011 | Mercaderías | inventory |
| 25 | Materiales auxiliares, suministros y repuestos | supplies (`Product.Status = 2`) |
| 33 · 39 | Propiedad, planta y equipo · Depreciación acumulada | assets |
| 40111 | IGV — cuenta propia | sales, purchases, monthly settlement |
| 4017 | Impuesto a la renta | income-tax payments |
| 4211 · 4212 | Facturas por pagar — no emitidas · emitidas | receptions without / with comprobante |
| 441 | Accionistas — dividendos | dividend payment (exists) |
| 465 | Pasivos por compra de activo inmovilizado | asset purchases |
| 50 · 59 | Capital · Resultados acumulados | opening, annual close |
| 6011 · 603 | Compras de mercaderías · de materiales y suministros | purchases |
| 6111 · 613 | Variación de mercaderías · de materiales y suministros | reception destination |
| 621 · 627 | Remuneraciones · Seguridad social | payroll |
| 631 · 632 · 634 · 635 · 636 · 637 | Transporte · Asesoría · Mantenimiento · Alquileres · Servicios básicos · Publicidad | expense categories |
| 641 | Tributos — gobierno central | taxes that are an expense |
| 655 · 656 · 659 | Costo neto de enajenación de activos · Suministros · Otros gastos de gestión | disposal, supplies, other |
| 676 · 776 | Diferencia de cambio (gasto · ingreso) | FX |
| 6814 | Depreciación de PPE | depreciation |
| 6911 | Costo de ventas — mercaderías | COGS |
| 7011 · 704 | Ventas de mercaderías · Prestación de servicios | revenue |
| 756 · 759 | Enajenación de activos · Otros ingresos de gestión | disposal, surplus |
| 89 | Determinación del resultado del ejercicio | annual close |

Class 9 (cost by function) and 79 are left out until a company over 1 700 UIT needs them.

### 6.2 Posting rules

One pure function per source kind: `Post<Source>(fact, settings, rates) → ([]JournalLine,
[]Diagnostic)`. Every function is tested against the invariant Σ debit = Σ credit.

| # | Event | Date | Debit | Credit | Notes |
| --- | --- | --- | --- | --- | --- |
| 1 | Comprobante issued (factura/boleta) | `IssueDate` | 1212 total | 7011/704 net · 40111 IGV | State Rechazado/Baja → no entry, like RVIE nota 3 |
| 2 | Credit note (07) | its `IssueDate` | 7011 net · 40111 IGV | 1212 total | reverse of 1 |
| 3 | Sale without comprobante | sale `Date` | 1212 total | 7011 net · 40111 IGV | split at 18 %, `ACC-005` |
| 4 | Sale collection (cash type 8/7) | movement date | 101/1041 | 1212 | |
| 5 | Sale refund (type 11) | movement date | 1212 | 101/1041 | the revenue reversal is the credit note |
| 6 | Sale delivery — **perpetual** | movement date | 6911 | 2011 | at the computed cost |
| 7 | OC reception, comprobante registered | reception date | 2011 | 6111 | at final cost |
|   | … and its comprobante | `DocIssueDate` | 6011 net · 40111 IGV | 4212 total | IGV into cost when not creditable (§4.5) |
| 8 | OC reception, no comprobante yet | reception date | 6011 · 2011 | 4211 · 6111 | provisional cost, `INV-002` |
|   | … comprobante arrives | `DocIssueDate` | 4211 · 40111 · (6011 difference) | 4212 | settles 4211 |
| 9 | Supplier payment (type 6) | movement date | 4212 | 101/1041 | needs bug 3 fixed |
| 10 | Simple expense | `Date` | 6x (category → account) · 40111 | 4212 | Taxes → 40111/4017/641 by chosen tax; Payroll → `ACC-012` |
| 11 | Expense payment (type 9) | movement date | 4212 | 101/1041 | |
| 12 | Inventory expense (supplies) | `Date` | 603 net · 40111 · 25 | 4212 · 613 | same shape as 7 |
| 13 | Supplies consumption — perpetual (stock type 13) | movement date | 656 | 25 | |
| 14 | Asset acquisition | `AcquisitionDate` | 33 net · 40111 | 465 | donated (price 0): counter-account D9 |
| 15 | Asset payment (type 10) | movement date | 465 | 101/1041 | |
| 16 | Depreciation (expense type 4) | period date | 6814 | 39 | exists as data |
| 17 | Asset disposal | `DisposalDate` | 39 accumulated · 655 net book value | 33 cost | sale value: 1212 / 756 — `ACC-011` |
| 18 | Cash transfer (type 3) | movement date | 10 destination | 10 origin | posted once, from the outflow row (exists) |
| 19 | Withdrawal · loss · count (types 4/5/2) | movement date | per `AccountCode` | 10 | or reverse for a surplus (exists) |
| 20 | Downward adjustment (merma) | movement date | 6911 or 659 (D6) | 2011 | perpetual; periodic: nothing, it is in COGS |
| 21 | Upward adjustment (surplus) | movement date | 2011 | 759 | perpetual |
| 22 | **Periodic close** | last day | 6911 | 2011 | COGS = opening + inflows − closing (§4.2) |
| 23 | FX revaluation at month end | last day | 676 or 10/12/42 | 10/12/42 or 776 | USD cash, receivables, payables at the closing official rate |
| 24 | Opening entry | first day | balances | balances | §6.6 |
| 25 | Manual entry | typed | typed | typed | §6.6 |
| 26 | Annual close | 31/12 | 7x · 89 | 6x · 89 → 59 | |

Periodic and perpetual share every rule except 6, 13, 20, 21 (perpetual only) and 22 (periodic
only) — which is the method-agnosticism made concrete.

### 6.3 Currency

- The books are in PEN. Every USD fact is restated at the **official SBS/BCRP selling rate of its
  date**, from the existing `exchange_rates` table. Sales keep their commercial `ExchangeRate`
  (spread included) for pricing, but the books use the official rate — the RVIE's field 28
  requires it too (D11).
- Month-end revaluation (rule 23) turns the difference between booking rate and closing rate into
  676/776.

### 6.4 The journal: derived while open, frozen at close

- **Open period:** the journal is computed on demand from the facts, as the books `PLAN.md`
  already decided. Nothing is stored; fixing a fact fixes the journal.
- **Close:** the projection is run one last time and written to `journal_entry`, each entry with
  its CUO, `M`-prefixed correlativo, document fields and CAR (books `PLAN.md` §1.4). From then on,
  PLE 5.2/5.1, the Mayor and the statements for that period read the frozen rows.
- **Why freeze:** a filed book cannot change when someone edits an old expense, and PLE needs a
  stable CUO/correlativo. The frozen rows are also the audit trail.

### 6.5 Period lifecycle

```
Open ──close (no blocking diagnostics)──▶ Closed ──reopen (admin, logged)──▶ Open
```

- Closing writes the frozen journal, the inventory snapshot (§4.9) and `AccountingPeriod.Status`.
- **The lock:** any create or edit of a source fact whose accounting date falls in a closed period
  is rejected by the backend, with the period named in the error. One helper,
  `accounting.EnsurePeriodOpen(companyID, date)`, is called by every handler that writes facts.
- Reopening deletes that period's frozen rows and every later closed period's (they depend on
  its closing snapshot), and logs who reopened it and why.

### 6.6 Opening balance and manual entries

- **Opening entry:** a guided form for the first period — cash per till/bank (prefilled from
  `CashBank.CurrentAmount`), receivables and payables (prefilled from open debts), valued opening
  stock (prefilled quantities, typed unit cost → writes type-16 movements), assets (prefilled from
  `accounting.Asset`), and capital as the balancing figure. Stored as a manual entry with the `A`
  correlativo prefix.
- **Manual entries** are the flexibility valve for what the ERP does not model — payroll with its
  withholdings, loans and interest, income-tax provisions, CTS/gratificaciones accruals,
  corrections. Account, debit, credit, gloss, document reference; validated as balanced; only in
  open periods. The ERP automates the routine and the accountant completes the rest.

### 6.7 Outputs

| Output | From | Notes |
| --- | --- | --- |
| Libro Diario Simplificado — PLE 5.2 + 5.4 | frozen journal | ≤ 300 UIT |
| Libro Diario — PLE 5.1 · Libro Mayor — PLE 6.1 | frozen journal | > 300 UIT |
| Balance de comprobación | ledger balances | also the working view of an open period |
| Estado de Resultados (by nature) | classes 6 and 7 | the official version of the level-1 management P&L |
| Estado de Situación Financiera | classes 1–5 + the period's result | a fixed PCGE → statement-line map in code |

Consistency check, run on every statement: the period's result from the Estado de Resultados
equals the change in equity in the Estado de Situación Financiera.

---

## 7. Data model changes

All table edits go through the `create-database-tables` skill and `check_tables`.

**Facts (all levels):**

| Table | Change |
| --- | --- |
| `warehouse_product_movement` | **no new column.** `Type` becomes the `StockMovementType` enum (§4.6), from which the source table and the Tabla-12 code are derived; transfers write `WarehouseRefID`. No backfill: existing data is wiped (pre-alpha). **The grouped-movement view is widened** to `ProductID.DecimalSize(8)` + `Type.DecimalSize(2)` (`product-stock-movement.go:104`): it packed `Type` into one digit, which capped the catalog at 9 types. Settled: the catalog is never limited by a view. 99 types; 99,999,999 products per company. `SubDivisor` keeps its 4 digits (`MaxQuantityDivisor` = 1000) |
| `InternalMovement` / stock adjustment handler | optional `UnitCost`; the adjustment reason is chosen as a type (1, 2, 14); opening-stock (12) and consumption (13) entry points |
| `products` | `+ ExistenceType int8` (Tabla 5), SUNAT unit code mapped from `UnitID` (Tabla 6) |
| `expenses` / `expenses_scheduled` | category → account in code; the Taxes category gets a `TaxKind` (IGV · renta · other) |
| `cash_bank_movements` | document-bearing types require a `DocumentID` (bug 5) |
| `accounting.Asset` | `+ DisposalValue int32` |
| company settings | `AccountingLevel`, `InventoryMethod`, `CostFormula`, `TaxRegime` (D1) |

**Frozen projections (level ≥ 1):**

| Table | Columns |
| --- | --- |
| `accounting_period` | CompanyID, Period `YYYYMM`, Status, Method, ClosedTime, ClosedBy, ReopenReason |
| `inventory_snapshot` | CompanyID, Period, ProductID, PresentationID, WarehouseID (per D2), Quantity, SubQuantity, Value, FIFO layers when needed |
| `journal_entry` | CompanyID, Period, ID, CUO, Correlativo, Date, SourceKind, SourceID, DocType, DocSeries, DocNumber, CAR, Gloss, `DetailAccountCode []int32`, `DetailDebit []int64`, `DetailCredit []int64`, Manual bool |

The `Detail*` names follow the house rule: singular, parallel arrays.

---

## 8. Code layout

```
backend/accounting/types/          pure — importable by any module, no DB, no clock
  pcge.go                          the chart (feeds 5.4) and the expense-category → account map
  inventory_cost.go                ResolveInflowCost (§4.5)
  inventory_valuation.go           ValuePeriodic / ValuePerpetual → valued rows + snapshot + diagnostics
  posting.go                       Post<Source> functions (§6.2)
  statements.go                    trial balance, Estado de Resultados, Estado de Situación
  diagnostics.go                   Diagnostic + codes
  *_test.go                        conservation (§4.1), Σ debit = Σ credit, periodic = perpetual
                                   when there is a single inflow price, provisional → final replay
backend/accounting/
  valuation_api.go                 GET inventory valuation / Kardex 12.1 / 13.1 / management P&L
  journal_api.go                   GET journal (derived or frozen), trial balance, statements
  period_api.go                    POST close / reopen; EnsurePeriodOpen helper
  manual_entry_api.go              opening and manual entries
frontend/routes/accounting/
  inventory-valuation/             Kardex and valued stock, with the diagnostics panel
  statements/                      management P&L (L1), EEFF (L2)
  periods/                         close/reopen, blocking diagnostics list
  journal-entries/                 opening and manual entries
  books/                           gains the Diario (5.2/5.1) from the frozen journal
```

Computation happens on the **backend**, unlike the sales and purchase books that join in the
browser: the close must be server-authoritative, and the valuation replays the whole ledger
(D12).

---

## 9. Phases

Each phase ships on its own and leaves the system consistent.

| # | Scope | Unlocks |
| --- | --- | --- |
| 0 | Fix the six fact bugs (§1.3) | correct quantities, cash, debts for everyone |
| 1 ✓ | Complete facts: `StockMovementType` enum + engine contract + widened grouped view; opening-stock, consumption and shrinkage types; adjustment cost; product existence type and SUNAT unit (derived — §4.6) | Level 0: Kardex 12.1 |
| 2 | Company settings (level, method, regime) + level-0 cash result report | the settings every projection reads |
| 3 | `ResolveInflowCost` + valuation engine (periodic and perpetual) + `INV-*` diagnostics | Level 1: valued stock, COGS, margin, Kardex 13.1, management P&L |
| 4 | Posting rules + derived journal + trial balance + `ACC-*` diagnostics | Level 2 preview |
| 5 | Periods: close, freeze, lock, reopen; inventory snapshot; opening and manual entries | a filable Diario |
| 6 | Statements + PLE 5.2/5.4 (and 5.1/6.1) export | automatic EEFF |
| 7 | FX revaluation, annual close, FIFO | full year cycle |

---

## 10. Open decisions

| # | Decision | Recommendation |
| --- | --- | --- |
| D1 | Where the accounting settings live: company flags (`company_flags.toml`, numeric `type`) or the company config blob | config blob: these are settings, not rule switches, and the regime is not a checkbox |
| D3 | Perpetual cost formula at launch | moving weighted average only; FIFO in phase 7 |
| D4 | Periodic cost formula at launch | monthly weighted average |
| D5 | Periodic closing quantity: ledger balance, or a mandatory count | ledger balance by default; a count, when recorded, is an adjustment movement that wins |
| D6 | Shrinkage account: 6911 (normal, part of cost) or 659 | 659 for type 14; normal merma to 6911 when the accountant says so |
| D7 | Receptions without comprobante through 4211 (no emitidas) | yes — it keeps the payable and the stock right before the invoice arrives |
| D8 | PCGE sub-account digits (2011 vs 20111, 6814, 603x…) | confirm with an accountant before phase 4 |
| D9 | Counter-account of a donated asset (`PurchaseAmount = 0`, value > 0) | an owner's contribution to capital (50/52), confirmed by the accountant |
| D10 | Level 1 without closes: replay from the start, or automatic monthly soft snapshots | replay from the start at first (small companies); soft snapshots when a tenant gets slow |
| D11 | Books at the official rate while sales price at the commercial rate + spread | yes; the spread is a pricing rule, not an accounting one |
| D12 | Projections on the backend (unlike the browser-joined books) | backend, because the close must be authoritative |
| D13 | Sales without comprobante in the books' revenue | yes, with `ACC-005`: income is income even when no comprobante was issued |
| D14 | Supplier credit notes (07) — already open in books `PLAN.md` §6 | settle there; the posting rule is the reverse of rule 7 |

---

## 11. Out of scope

- A payroll module (planilla, AFP/ONP, EsSalud, CTS, gratificaciones) — manual entries cover it.
- Loans and financial instruments — manual entries.
- Class 9 cost-by-function accounting (needed only above 1 700 UIT).
- Landed cost (freight and customs added to inventory cost).
- Manufacturing and cost roll-ups (the ERP has no production orders).
- Deferred and current income tax computation (NIC 12).
- Multi-company consolidation.
