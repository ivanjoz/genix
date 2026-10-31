# PLAN — The cost of every sale (perpetual moving average)

Status: **phases 1–4 implemented** (D6 approved; phase 4 follows the D3 and D4 recommendations:
one entry per sale and day, only types 8/9 posted). The backfill has not been run yet. D1, D2 and
D5 are still open (§7).

Feeds the cost-of-sales line that `frontend/routes/accounting/accounts-consolidation/PLAN.md` §3
defers. It implements the perpetual branch of `docs/ACCOUNTING_MODEL_PLAN.md` §4.3, with that
plan's recommended moving weighted average per warehouse (D3). It reuses the moving average the
stock engine already keeps.

## 1. Goal

- Every stock outflow carries its cost at the moment it happens.
- A sale's cost is the sum of its delivery movements, found through `DocumentID`.
- A period's cost of sales is the sum of the period's delivery costs minus its annulment returns.
- No valuation replay at read time; the cost is a stored fact.

## 2. What already exists

| Fact | Where | State |
| --- | --- | --- |
| Moving average per stock row, PEN cents per whole unit, net of IGV | `ProductStock.AverageCost`, moved by `NextMovingAverageCost` on every costed inflow | ✅ |
| Every inflow has a value, or the row's average stands in for it | `ApplyMovimientos`: inflows need a `Price` or an existing `AverageCost` (the check being added now) | ✅ (WIP) |
| A movement's value | `WarehouseProductMovement.MonetaryValue` — written **on inflows only** today | ⚠️ |
| Sale delivery / annulment return point to the sale | types 8 / 9, `DocumentID = SaleOrder.ID`, `DocumentID` local index | ✅ |
| Period read of the ledger | `(Date, Type)` index group | ✅ |
| Replay of the average from the ledger | `RecalcProductStockByMovements` (`fn-recalc-stock`) | ✅ |
| The total value of a stock row | — (only the rounded average is stored) | ❌ |

The missing pieces are **one new column and one column filled on one more side**:
`ProductStock.StockValue`, and `MonetaryValue` on outflows.

## 3. The rule

`ProductStock` gains `StockValue int64` — the value of everything on the row (free bucket plus
lots), PEN cents, net of IGV. `AverageCost` stays `int32`. `StockValue` is the source of truth for
costing; `AverageCost` is its rounded per-unit view (§3.3).

Inside `ApplyMovimientos`, inside the company lock, after the delta is resolved
(`balance` = the row's sub-units before the movement):

```
inflow  (delta > 0):  MonetaryValue = line value at Price (or at AverageCost when it has no Price)
                      StockValue   += MonetaryValue
                      AverageCost   = round(StockValue × divisor ÷ new balance)
outflow (delta < 0):  MonetaryValue = |delta| == balance ? StockValue
                                                         : round(StockValue × |delta| ÷ balance)
                      StockValue   -= MonetaryValue
                      AverageCost does not move
```

- **Sign.** `MonetaryValue` stays positive on both sides; `Quantity`'s sign says the direction.
  Sub-units are valued pro rata, exactly like whole units.
- **Every outflow type is stamped:** sale delivery (8), manual exit (2), consumption (13),
  shrinkage (14), the transfer's outgoing leg, and the asset types (6, 7, 10). The accounting reads
  the types it posts; the others become a valued Kardex 13.1 for free.
- **The emptying outflow takes the remainder**, so the rounding of every earlier outflow is
  absorbed and the row reaches value 0 with its units (§3.3). Negative balances are already
  rejected, so an outflow can never take more value than the row holds.
- **`AverageCost` survives an empty row.** At 0 units `StockValue` is 0 and no average can be
  derived; `AverageCost` keeps the last known cost, which values a priceless inflow into that row.
- **No value (`StockValue = 0` with units on hand)** can only be stock that entered before the WIP
  check. The outflow is stamped 0 and the period shows it as uncosted (§5).

### 3.1 Annulment return (type 9) re-enters at the cost it left with

`netStockReturns` (`sales/sale_order_annul.go`) already groups the sale's type 8/9 movements per
bucket. It also sums their `MonetaryValue` per bucket (Σ type 8 − Σ type 9), and the return carries
that sum as `InflowValue`, the inflow's exact total. A unit `Price` would be re-multiplied by the
quantity and round. So the goods come back at the cost that left, to the cent — not re-priced at
today's average — and the sale's cost nets to 0.

### 3.2 Transfers

The outgoing leg is stamped from the origin's `StockValue` (rule above). The incoming leg enters
with the outgoing leg's `MonetaryValue` — not re-priced at the average, as `expandStockTransfers`
does today — so the value moves between warehouses to the cent.

### 3.3 Why the stock value ends at exactly zero

The method rests on one invariant: **stock value = units on hand × average**. Inflows change
both sides and re-compute the average; outflows remove units *at* the average, so the average
does not move and the invariant still holds afterwards.

| Step | Units | Movement value | Stock value | Average |
| --- | ---: | ---: | ---: | ---: |
| Buy 10 @ 20.00 | 10 | +200.00 | 200.00 | 20.00 |
| Buy 10 @ 15.00 | 20 | +150.00 | 350.00 | (200 + 150) ÷ 20 = **17.50** |
| Sell 12 | 8 | −12 × 17.50 = **−210.00** | 140.00 | 17.50 (unchanged) |
| Sell 8 | 0 | −8 × 17.50 = **−140.00** | **0.00** | — |

The system "knows" the cost of the 12 because it never tracks which cookie came from which
purchase: after the second purchase every unit costs 17.50, and that is the price each one leaves
at. 210 + 140 = 350, exactly what was bought.

**Why the average alone is not enough: rounding.** Today `ProductStock` stores only
`AverageCost`, rounded to whole cents per unit. When the true average is not a whole number of
cents, valuing each outflow at the rounded average accumulates the remainders:

| Step | Units | Movement value | True stock value | Stored average |
| --- | ---: | ---: | ---: | ---: |
| Buy 3 for 10.00 | 3 | +10.00 | 10.00 | 3.333… → **3.33** |
| Sell 1 | 2 | −3.33 | 6.67 | 3.33 |
| Sell 2 | 0 | −6.66 | **0.01** left over | — |

Cost of sales is 9.99 against 10.00 bought; the cent never leaves the inventory account (2011),
and the next inflow on an empty row starts a fresh average (`NextMovingAverageCost`,
`balanceSubUnits <= 0`), so the residual is lost from the stock but stays in the books. Sub-units
(`divisor`) make the remainders larger and more frequent.

**Hence `StockValue` (D6): carry the value, derive the average.** Outflows are valued from the
row's total value (rule in §3), so the same example closes exactly:

| Step | Units | Movement value | `StockValue` | `AverageCost` |
| --- | ---: | ---: | ---: | ---: |
| Buy 3 for 10.00 | 3 | +10.00 | 10.00 | 3.33 |
| Sell 1 | 2 | −round(10.00 × 1 ÷ 3) = −3.33 | 6.67 | 3.33 |
| Sell 2 (empties the row) | 0 | −6.67 (the remainder) | **0.00** | 3.33 (kept) |

Each outflow is still the average cost to the cent, but the last one absorbs the accumulated
rounding, so Σ outflows = Σ inflows exactly and 2011 clears when the warehouse is empty. This is
the usual "cost pool" implementation of moving average.

`StockValue` is `int64` because a row's total value is unbounded by the unit price: `int32` cents
caps at about 21.4 million PEN, and overflowing it would corrupt the books silently.
`AverageCost` stays `int32` — it is a per-unit price.

## 4. Reading it

| Question | Read |
| --- | --- |
| Cost of one sale | movements by `DocumentID = saleID`, types 8 and 9: Σ value(8) − Σ value(9) |
| Period cost of sales | `(Date, Type)` index group, types 8 and 9, the period's days |
| Per product / margin | the same rows grouped by `ProductID` against the sale's line amounts |

The accounting operations endpoint (phase 3 of the consolidation plan) sends these rows as
numbers. The builder posts **one cost entry per sale, dated at the delivery**:

- Dr 6912 · Cr 2011 for products with `Origin = 1` (bought)
- Dr 6922 · Cr 2111 for `Origin = 2` (made)
- A supply delivered in a sale posts Dr 6912 · Cr the supply's stock account

The entry is internal (document type `00`). It **follows its sale** in or out of the official
Diario, like the sale's collections: the cost of a sale without comprobante stays out, so the
official books never deduct a cost against revenue they do not hold.

A sale delivered in a later month than it was sold books its revenue in the sale's month and its
cost in the delivery's month. That is how perpetual inventory behaves, and it matches the stock.

## 5. Gaps, made visible

| Case | Shown as |
| --- | --- |
| An outflow stamped 0 (no average when it left) | "uncosted delivery" on the consolidation page, with the product and units |
| Reception costed before its comprobante (gross, `logistics/RATIONALE.md` "AverageCost is written at movement time") | not detected — see D2 |

## 6. Backfill (fix the data)

Existing outflows have `MonetaryValue = 0` and no row has a `StockValue`.
`RecalcProductStockByMovements` already replays every stock row in ledger order to rebuild the
average. It is extended to **run the §3 rule over the ledger**: rebuild each row's `StockValue`,
stamp each outflow's `MonetaryValue` from it, and write the changed movements and stock rows, once
per company: `fn-recalc-stock` (Param1 = company ID).

- The ledger is sorted by `Created` (inflows first within one second), because its key is not
  chronological within a day.
- A recorded `MonetaryValue > 0` is kept; only rows at 0 are valued. A return (9) at 0 takes what
  its sale still has out of that row; any other inflow at 0 takes the running average. History that entered without a cost stays uncosted (§5) — there is no
value to invent.

## 7. Open decisions

| # | Decision | Recommendation |
| --- | --- | --- |
| D1 | Store the sale's cost also on `SaleOrder` (a `CostAmount` column) | **No.** The movements are the fact and `DocumentID` is indexed; a copy on the sale can drift (annulments, partial deliveries) |
| D2 | A reception costed gross before its comprobante: restate the average when the comprobante is registered? | **No restatement** (keeps the current RATIONALE). Make the comprobante required on receptions instead, or accept the gap and flag receptions whose order had no comprobante when received |
| D3 | Cost entry granularity: one per sale, or one per period | **One per sale** — it links to the sale, and the 13.1 Kardex needs the per-movement cost anyway |
| D4 | Where shrinkage (14) and consumption (13) post | Defer to the consolidation page: 659 / 656 as fixed lines, using the same stamped value |
| D5 | Company-level switch periodic / perpetual (`ACCOUNTING_MODEL_PLAN` §3.2) | Not now. Perpetual is the only method; the switch comes with that plan if it is approved |
| D6 | Rounding residue: outflows at a rounded `AverageCost` leave cents in stock when it reaches 0 (§3.3) | **Decided:** `ProductStock.StockValue int64` carries the row's value and outflows are valued from it, the emptying outflow taking the remainder (§3, §3.3). `AverageCost int32` stays as the rounded per-unit view and the last known cost of an empty row. The backfill (§6) fills `StockValue` in the same replay |

## 8. Phases

| # | Scope |
| --- | --- |
| 1 | `ProductStock.StockValue int64` column; `ApplyMovimientos` keeps it and stamps outflows from it (§3) + tests (sub-units, divisor refinement, no value, the emptying outflow closing to 0, the 3-for-10.00 case) |
| 2 | Annulment return at the delivered cost (§3.1) and the transfer's incoming leg at the outgoing value (§3.2) + tests |
| 3 | Backfill through `RecalcProductStockByMovements` (rebuilds `StockValue`, stamps outflows) + `fn-recalc-stock apply` |
| 4 | ✅ `GET.accounting-operations` sends the period's type 8/9 movements and their products; `accounts-consolidation.ts` posts one cost entry per sale and day (`CV-<sale>-<day>`); a delivery stamped 0 makes it incomplete |
