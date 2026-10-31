# PLAN — PCGE chart of accounts + "Consolidado & Cuentas" (`/accounting/accounts-consolidation`)

Status: **approved. Phases 1–5 built (chart, account columns + forms, operations endpoint + builder, the page, the journal and chart tabs) — see §9.**

## 0. Purpose — two ledgers from the same entries

This work exists to produce the **monthly Balance Sheet and the monthly Income Statement**. Those
are management reports and include **every** movement in the system, with or without a
comprobante. The SUNAT books are a subset of the same entries:

| | Management ledger | Official Libro Diario (PLE 5.2) |
| --- | --- | --- |
| What it holds | every movement — with and without comprobante | only operations backed by a numbered document |
| Valid for | the company's own financial accounting | SUNAT |
| Where it is shown | **this page** (and a "Diario de gestión" view of it) | `/accounting/books` → Diario Simplificado + TXT |
| Feeds | Balance + Estado de Resultados (monthly) | 5.2 / 5.4 files |

One builder produces the entries; each entry carries `InOfficialBook`. The official Diario is
**everything relevant to SUNAT — every entry except the sales and purchases (orders, expenses,
assets) without a comprobante, and the payments/collections of those**. Internal entries
(depreciation, transfers, withdrawals, capital, cost of sales, disposals) are in, with document
type `00`. A movement without a
comprobante is not valid for SUNAT but is valid — and required — for the financial statements.

## 1. Core idea: the user classifies the nature line, the system derives the rest

A purchase factura of S/ 118:

| Line | Account | Debe | Haber | Who decides |
| --- | --- | ---: | ---: | --- |
| Nature | **60x / 63x / 33x …** | 100.00 | | the user (defaulted) — "Clasf. Contable" |
| Tax | 40111 IGV | 18.00 | | the document's `TaxAmount` |
| Counterpart | 4212 Proveedores | | 118.00 | the operation kind — **locked** |

The entry balances by construction once the nature account(s) are set and the record's amounts
are consistent. IGV, payable, receivable and cash lines are never user-editable.

**Nature lines are per document, except purchases of supplies, which are per line** (§4).

## 2. The PCGE chart (`backend/accounting/types/pcge.go`)

✅ **Built.** 556 accounts (440 postable) in `pcge_accounts.go`: every 2- and 3-digit account of
PCGE 2019, plus 4/5-digit subaccounts under the families the rules and an SME post to.
Transcribed from the official catalog (Resolución 002-2019-EF/30). Synced to
`frontend/core/pcge.ts` through the `//CATALOG:` generator.

```go
type PCGEAccount struct {
    Code     int32  // numeric, as PLE 5.4 field 2 requires
    Name     string // official Spanish PCGE name
    Postable bool   // a leaf (3+ digits, no children in the list); the others only group
    // Pairs the rules derive from, set only where they apply:
    StockCode                   int32 // 60x → the 20x–26x stock its reception debits
    VariationCode               int32 // 60x → the 61x its reception credits
    AccumulatedDepreciationCode int32 // 33x → its 3952x
    DepreciationExpenseCode     int32 // 33x → its 6841x
}
```

- **Codes are `int32`.** `CashBankMovement.AccountCode` (`int16`) became
  `CounterAccountCode int32` (see RATIONALE). The `PCGE*` constants in
  `finance/types/cash_movement_account.go` stay there, because of an import cycle, and a test keeps
  them postable in the chart.
- **Applicable accounts per kind** (`PCGEKindPrefixes`, checked by `IsPCGEAccountAllowed` /
  `ValidatePCGEAccount`, filtered in the frontend by `pcgeAccountsForKind`):

| Kind | Allowed prefixes |
| --- | --- |
| Supply (catalog default and purchase line) | 602, 603, 604, 656, 33 |
| Simple expense | 62, 63, 64, 651–654, 656, 658, 659, 67, 40 |
| Asset | 32, 33, 34 |

Sales and the derived lines (IGV, payables, receivables, cash) have no kind: the system picks
them. Manual cash movements keep `CashMovementAccountOptions`.

## 3. Operations and their entries

Every source record of the period is one operation = one entry. CUO = kind prefix + source id.

| Operation | Source | Nature line | Derived lines | Official Diario? |
| --- | --- | --- | --- | --- |
| Sale | `SaleOrder` (not annulled) + its `InvoiceDocument` | Cr 7012 / 7022 per line, from the product's origin (§4) | Dr 1212 · Cr 40111 | if issued |
| Sale credit note | `InvoiceDocument` 07 | Dr 7012 / 7022 | inverse of the sale | yes |
| Sale collection / refund | `CashBankMovement` 8 / 11 | — | Dr cash · Cr 1212 (or inverse) | follows the sale |
| Purchase | `PurchaseOrder` | **per line**: product → 6011; supply → the line's account | Dr 40111 · Cr 4212 | if comprobante |
| Purchase reception | stock movement of the order | — | Dr 20x/25x · Cr 61x per line (from the 60x pair) | follows the order |
| Supplier payment | `CashBankMovement` 6 | — | Dr 4212 · Cr cash | follows the order |
| Simple expense | `Expense` type 1 | Dr `AccountCode` | Dr 40111 if taxed · Cr 4212 (comprobante) / 4699 (none) | if comprobante |
| Inventory expense | `Expense` type 2 | Dr the supply's account (one `ProductID`) | as purchase + reception | if comprobante |
| Expense payment | `CashBankMovement` 9 | — | Dr 4212/4699 · Cr cash | follows the expense |
| Asset acquisition | `Asset` | Dr `AccountCode` (default generic 33x) | Dr 40111 · Cr 4654; **donated → Cr 501 Capital social** | if comprobante |
| Asset payment | `CashBankMovement` 10 | — | Dr 4654 · Cr cash | follows the asset |
| Depreciation | `Expense` type 4 | — | Dr 6841x · Cr 3952x (from the asset's 33x) | internal |
| Asset disposal | `Asset.DisposalDate` | — | Dr 3952x · Dr 6551 · Cr 33x | internal |
| Manual cash movement | `CashBankMovement` 2/4/5/7 | existing `AccountCode` | cash side | internal |
| Unspecified cash movement | `CashBankMovement` 1 | **must be classified** | cash side | internal |
| Transfer | `CashBankMovement` 3, negative row | — | 101 ↔ 1041 | internal |
| Cost of sales | sale delivery / annulment return (`WarehouseProductMovement` 8 / 9), at their stamped cost — `docs/SALE_COST_PLAN.md` | — | Dr 6912 · Cr 2011 (bought) · Dr 6922 · Cr 2111 (made) · Dr 6912 · Cr the supply's stock account; a return reversed | follows the sale |

Cash side: `CashBank.Type` → 101 (till) or 1041 (bank). A purchase line's nature amount is the
order's taxable base split by line value, with the rounding remainder on the last line so the
lines sum exactly to the base.

Not covered yet: payroll (no module), manual stock adjustments / shrinkage, apertura/cierre,
manual journal adjustments.

## 4. Products vs supplies

- **A product (`Product.Status = 1`) is bought or made** — a new `Product.Origin int8` column,
  set on the product form. Its accounts follow from it; no account selector for products:

  | `Origin` | Meaning | Stock | Sold | Cost of sales | Bought? |
  | --- | --- | --- | --- | --- | --- |
  | 1 `ProductOriginBought` | merchandise (mercaderías) | 2011 | 7012 | 6912 | Dr 6011 on purchase |
  | 2 `ProductOriginMade` | finished good (productos terminados) | 2111 | 7022 | 6922 | never — a made product on a purchase order is rejected |

  Every existing product is backfilled to `1`. How a made product's cost enters stock (Dr 2111 /
  Cr 7111 variación de la producción almacenada) belongs to the production orders, which do not
  exist yet. Until then a made product's period cost of sales is valued at its stock × last cost,
  the same as merchandise.
- **A supply (`Status = 2`) has its own default account**, a new `Product.AccountCode int32`
  column, picked from the supply-applicable list when the supply is created or edited
  (`routes/production/supplies-materials`). Typical values: 6032 materiales auxiliares, 6041
  envases y embalajes, 656 suministros, 3361 equipo de cómputo.
- **A purchase order line carries its own account**: `PurchaseOrderLine.AccountCode int32`
  (inside the `Lines` blob). Stamped by the server from the supply's default when the client sends
  0; `0` for product lines. (The per-line selector waits until supplies can be added to the PO cart.) So two lines of the same supply can go to
  different accounts, and editing the supply's default never rewrites past purchases.
- An inventory expense (`Expense` type 2) has one supply, so its `AccountCode` is prefilled the
  same way.

## 5. Defaults

| Source | Default `AccountCode` |
| --- | --- |
| Expense category 1 Rent | 6352 Alquileres — edificaciones |
| 2 Utilities | 6361 Energía eléctrica |
| 3 Payroll | 6211 Sueldos y salarios |
| 4 Supplies | 656 Suministros |
| 5 Taxes | **0** — IGV paid (40111) and an expense tax (64x) differ |
| 6 Maintenance | 6343 Mantenimiento — propiedad, planta y equipo |
| 7 Transport | 6311 Transporte |
| 8 Marketing | 6371 Publicidad |
| 9 Professional services | 6329 Otros servicios de asesoría |
| 10 Other | **0** |
| Supply line / inventory expense | the supply's `Product.AccountCode` |
| Asset | generic 33x (3369 Otros equipos diversos) unless the user picks one |

Defaults are **stored when the record is written** (the form preselects them), and a backfill
stamps them on existing rows. `AccountCode = 0` therefore always means "unclassified".

## 6. Status

| Status | Meaning | Balances? |
| --- | --- | --- |
| ✅ Classified | every nature line has an account, amounts consistent | yes |
| 🔴 Unclassified | a nature line with `AccountCode = 0` | no |
| ⚠️ Incomplete data | totals don't add up, USD without PEN rate, received order without comprobante… | no — fixed on the source record, linked from the row |

Page header, per period, by count **and** by amount:

```
Operations 412 · Classified 389 (94% · 98% of amount) · Unclassified 17 · Incomplete 6
Debe S/ 1 245 300.00 = Haber S/ 1 245 300.00 ✔        Sin comprobante: 38 ops · S/ 12 400.00
```

The period is closed for the financial statements when unclassified = incomplete = 0.

## 7. The page

```
frontend/routes/accounting/accounts-consolidation/
  +page.svelte                      Shell: period selector, tabs, wiring
  OperationsTable.svelte            Tab Movimientos
  OperationEntryLayer.svelte        Side Layer: the entry of one operation (+ per-line accounts)
  ChartOfAccounts.svelte            Tab Plan de cuentas + period balance per account
  accounts-consolidation.svelte.ts  $state, GET/PUT calls
  accounts-consolidation.ts         Pure: operations from raw sources, entry rules, status, totals
  accounts-consolidation.test.ts    Every rule in §3 balances; status and split edge cases
  RATIONALE.md
```

**Tab Movimientos** — one `VTable`, every operation of the period:

| Fecha | Operación | Descripción | Nº Doc | Monto (S/) | Clasf. Contable | Estado |
| --- | --- | --- | --- | ---: | --- | --- |
| 03/09 | Gasto | Alquiler local | F001-000123 | 1 180.00 | `[6352 Alquileres ▾]` | ✅ |
| 05/09 | Compra | Proveedor X | F002-004410 | 590.00 | `3 líneas · 2 cuentas` → Layer | ✅ |
| 06/09 | Venta | — | *sin comprobante* | 45.00 | 7012 (fixed) | ✅ |
| 07/09 | Mov. caja | Sin especificar | — | -200.00 | `[ — ▾ ]` | 🔴 |

- Nº Doc shows *sin comprobante* (chip) when there is none; a filter splits con / sin comprobante.
- Single-account rows edit inline. A purchase with supply lines shows a summary and is edited in
  the Layer, one selector per line. Fixed-account rows (payments, transfers, sales, depreciation)
  are read-only but listed, so the table is really every movement.
- Multi-select + "apply account" for bulk classification.
- Amounts in soles, USD converted with the exchange-rate calendar (`finance/types/exchange_rate.go`).

**Tab Diario de gestión** — the same entries, Diario layout (CUO, fecha, glosa, cuenta, Debe,
Haber), all movements. The official Diario lives in `/accounting/books` and filters
`InOfficialBook`.

**Tab Plan de cuentas** — the chart as a tree with the period's Debe / Haber / saldo per account
(balance de comprobación). Read-only.

## 8. Backend

```
GET.accounting-operations?period=YYYYMM  raw period slices of every §3 source (numbers, not rows)
PUT.accounting-classification            [{ Kind, ID, Line, AccountCode }] → validates postable +
                                         allowed for the kind; writes the source record
```

Entries are built in the browser (`accounts-consolidation.ts`), like the books, so the screen,
the official 5.2 TXT and the statements read the same entries. Writes cross modules through the
`*/types` tables, which `MODULE_BOUNDARIES.md` allows.

Schema (via `create-database-tables`, then `check_tables`):

- `+ AccountCode int32` on `Expense`, `Asset`, `Product` (supply default).
- `+ Origin int8` on `Product` (bought / made, §4).
- `+ AccountCode int32` on `PurchaseOrderLine` (blob field, no column).
- ✅ `CashBankMovement.AccountCode int16` → `CounterAccountCode int32`.
- Backfill script stamping the §5 defaults and `Origin = 1`.

`SaleOrder` gets no account column: its nature lines follow from each product's `Origin`.

## 9. Phases

| # | Scope |
| --- | --- |
| 1 | ✅ PCGE chart + catalog sync + allowed prefixes + `int32` widening. **Data migration pending**: `fn-rename-columns apply` → `fn-homologate` → `fn-backfill-cash-movement-accounts apply` |
| 2 | ✅ `AccountCode` columns (+ `ExpenseScheduled`), `Product.Origin` on the product form, supply default on the supplies page, selectors on the Expense / Expense schedule / Inventory expense / Asset forms (applicable accounts only), server-side defaults + `fn-backfill-account-codes`. The PO line selector is deferred: supplies cannot reach the PO cart yet; the server stamps the line (see `backend/logistics/RATIONALE.md`). **Data backfill pending**: `fn-backfill-account-codes apply` |
| 3 | ✅ `GET.accounting-operations` (`backend/accounting/accounting_operations_api.go`) + pure builder `accounts-consolidation.ts` + 16 tests. Not yet run against real data: it reads `counter_account_code`, so it waits for the phase 1 migration. Cost of sales left out (`docs/SALE_COST_PLAN.md`); depreciation read through the assets' `AssetID` index; builder in the browser |
| 4 | ✅ Page: operations table with status filter, status header, entry Layer (`OperationEntryLayer.svelte`), `PUT.accounting-classification` (`backend/accounting/accounting_classification_api.go`), bulk apply through `VTable` row selection (genix-ui), menu + `access.toml` id 38. An unspecified cash movement is classified by retyping it (withdrawal / loss / collection) |
| 5 | ✅ Page tabs: Movimientos (`OperationsTable.svelte`), Diario de gestión (`JournalTable.svelte`, `entrySides`), Plan de cuentas (`ChartOfAccounts.svelte`, `buildTrialBalance`: the period's balance de comprobación as a tree). `+page.svelte` is now only the shell |
| 6 | Official Diario 5.2 + 5.4 in `/accounting/books` (filter `InOfficialBook`) |
| 7 | Monthly Balance Sheet + Income Statement from the same entries (own plan) |

## 10. Decisions taken

1. `AccountCode` column on each source table.
2. Defaults stored at write time + backfill; `0` = unclassified.
3. Selector also on the forms, offering only the applicable accounts.
4. Products carry their accounts through `Product.Origin` (bought / made, added now). Supplies
   are per purchase line, defaulted from a new account on the supply.
5. Chart: every 2/3-digit account + SME subaccounts.
6. Movements without comprobante are **not** in the official Diario; they are in the management
   ledger and the financial statements.
7. Donated asset → Cr 501 Capital social.
8. Asset with no account → generic 33x.
9. Derived lines are locked.
10. The official Diario holds everything relevant to SUNAT: every entry except sales and
    purchases without comprobante. Internal entries go in with document type `00`. In the
    official Diario, cash and inventory totals therefore differ from the management ledger; each
    entry still balances.
