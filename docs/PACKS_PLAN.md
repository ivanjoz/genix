# Packs — bundle pricing by product and/or category

Status: **phase 1 done** (2026-11-14) — `product_pack` table (ID 126), `GET product-packs` /
`POST product-pack`, the Packs page. **Phase 2 partly and phase 3 done** (2026-11-16) — the Go
engine `sales.ApplySalePacks` (`backend/sales/types/sale_packs.go`), the shared case file,
`SaleOrderLine.PackID`, and the dining charge hook. Still pending in phase 2: the till and web
checkout hooks. Phases 4–6 pending.

## 1. Goal

A new page **Catálogo → Packs** (`production/packs`) where the user defines bundles priced as a
whole. A pack is a list of components, each `quantity × product` or `quantity × category`, freely
mixed:

```
Menú del día   = 1 × [cat] Entrada + 1 × [cat] Plato Fondo 1   → S/ 18.00 fixed
Combo burger   = 1 × [prod] Burger + 1 × [cat] Bebidas          → 15% off
2 cafés        = 2 × [prod] Café americano                      → S/ 2.00 off
```

Decisions already taken (2026-11-14):

| Topic | Decision |
|---|---|
| Channels | Dining table service, till sale (POS) **and** web store |
| Detection | **Automatic** — the engine finds the packs in the order; staff do nothing |
| Recording | **Prorated into line prices** — component lines are repriced so they sum to the pack price; each line keeps its `PackID` |
| Price types | Fixed pack price, percentage off, fixed amount off |
| Rounding | Up / nearest / down to the whole sol, applied to the pack total |

Assumed (say otherwise if not):

- A pack applies as many times as the order fills it (2 entradas + 2 fondos = 2 menús).
- Pricing only — no effect on stock, reservations or production.
- A product component matches the product in any presentation.
- No validity dates, weekdays, hours or per-channel switch on a pack (can be added later).
- A pack never raises the price: if its result is ≥ the components' catalog sum it is skipped.

## 2. Data model

New table in the catalog module, `backend/production/types/product_pack.go`
(skill `create-database-tables`):

```go
type ProductPack struct {
	db.TableStruct[ProductPackTable, ProductPack]
	CompanyID  int32                  `cb:"1"`
	ID         int32                  `cb:"2"`
	Name       string                 `cb:"3"`
	// Components is one colbin blob: a pack is always read whole.
	Components []ProductPackComponent `cb:"4"`
	// PriceType: 1 = fixed pack price, 2 = percentage off, 3 = amount off.
	PriceType  int8  `cb:"5"`
	// PriceValue: cents (PEN) for types 1 and 3; basis points (1500 = 15%) for type 2.
	PriceValue int32 `cb:"6"`
	// RoundingMode: 0 = none, 1 = up, 2 = nearest, 3 = down — to the whole sol (100 cents).
	RoundingMode int8 `cb:"7"`
	Status     int8  `json:"ss" cb:"ss,8"`
	Updated    int32 `json:"upd" cb:"upd,9"`
	UpdatedBy  int32 `cb:"10"`
}

// ProductPackComponent is one slot of the pack: exactly one of ProductID / CategoryID is set.
type ProductPackComponent struct {
	ProductID  int32 `cb:"1"`
	CategoryID int32 `cb:"2"` // ID in the shared list PRODUCT_SHARED_LIST_CATEGORIA_ID
	Quantity   int16 `cb:"3"`
}
```

`SaleOrderLine` gains `PackID int32 \`cb:"10"\`` — the pack that priced the line, 0 when none.
The sale's ticket/boleta can group by it and reports can tell pack revenue apart.

Backend validation on save (never trust the client): name required, ≥ 1 component, each component has
exactly one of product/category, quantity ≥ 1, the product/category exists, PriceType ∈ {1,2,3},
PriceValue > 0 (type 2: ≤ 10000), RoundingMode ∈ {0..3}. A pack must hold ≥ 2 units in total (a
single-unit "pack" is just a price change).

API: `GET production/packs` as a delta-cache endpoint (skill `delta-cache-api`), `POST production/packs`
to save, delete = Status 0.

## 3. The pricing engine (pure, deterministic)

One pure function, the same algorithm in Go and TypeScript:

```
ApplySalePacks(lines, productByID, packs) → lines   // repriced + split, PackID set
```

Input lines are already priced at catalog (`PriceSaleOrderLine`). Only **whole units** take part;
the sub-unit part of a line is never packed.

**Matching — greedy by relevance** (decided 2026-11-16). A pack's **relevance** is the sum of its
slots' weights: a product slot weighs **5**, a category slot **2** (`Ceviche + [cat] Entradas` = 7,
`[cat] Entrada + [cat] Fondo + [cat] Postre` = 6). A pack that names products is a deliberate offer
and wins the units it shares with a generic menu, even when the menu would save more. Units are
expanded one per whole unit. Then, repeatedly:

1. For every active pack, try to fill it from the units still free. Product slots are filled before
   category slots (a specific slot must not lose its unit to a generic one). Inside a slot, the
   **most expensive** eligible units are taken first (largest saving for the customer).
2. Compute each fillable pack's price (§3.1) and saving = catalog sum − pack price.
3. Apply the pack with the highest relevance; tie → larger saving; tie → lower pack ID. Mark its
   units used.
4. Stop when no pack can be filled with a positive saving.

Greedy is not a guaranteed optimum on adversarial overlaps, but it is deterministic, explainable to
staff, and exact for the real cases (menus with disjoint categories). The Go and TS versions must
take identical tie-breaks: units ordered by (price desc, line index asc).

### 3.1 Pack price

```
catalogSum = Σ unit prices of the filled units
type 1: packPrice = PriceValue                       (rounding ignored — it is already chosen)
type 2: packPrice = catalogSum × (10000 − PriceValue) / 10000, then RoundingMode
type 3: packPrice = catalogSum − PriceValue,                then RoundingMode
skip if packPrice ≥ catalogSum or packPrice ≤ 0
```

### 3.2 Proration

Each unit gets `floor(unitPrice × packPrice / catalogSum)`; the leftover cents go to the most
expensive unit, so the units sum to `packPrice` exactly.

### 3.3 Back to lines

A line whose units ended up at different prices is **split**: one output line per distinct
(product, presentation, PackID, unit price); its remaining unpacked units and sub-unit part stay on
a line at catalog price. Line order is kept (split lines follow their source line).

### 3.4 Currency

Packs are defined in PEN. A USD till sale converts the fixed amounts (types 1 and 3) with the sale's
exchange rate through `finance.ConvertUnitPrice`, the same way unit prices are converted. Dining is
PEN only.

## 4. Where the engine runs

**Backend (authoritative)** — `backend/sales/types/sale_packs.go` (+ `_test.go`), next to
`PriceSaleOrderLine`, so `sales` and `dining` both reach it through `sales/types`:

| Channel | Hook |
|---|---|
| Till sale | `validateSaleOrderLines` (`sales/sale_order_create.go`), after the per-line pricing, before the total |
| Web store | Same function — `web_checkout.go` already prices through it. The online order stores the packed lines, and the sale created from it keeps them |
| Dining | `PostDiningTableOrderCharge` (`dining/table_order_charge.go`), on the sale lines built from the table order. Packs are applied **at charge time**, not per round: an entrada in round 1 and a fondo in round 2 still form a menu. `TableOrderLine` keeps catalog prices |

The active packs are read once per request (one company-scoped query).

**Frontend (preview only)** — `frontend/core/sale-packs.ts` (+ test), so the till, the floor-service
bill and the web cart show the packed total before saving:

- `routes/sales/sale_order_create/sale_order.svelte.ts` — order total and line prices.
- `routes/dining/floor-service/table-charge.ts` — the bill before charging.
- `frontend/webpage` cart — needs the packs in the web catalog file (`ecommerce-catalog.ts`
  colbin file + delta gets a `Packs` list).

**Parity:** one shared JSON case file (`backend/sales/types/testdata/sale_packs_cases.json`: lines,
products, packs → expected lines) is run by both the Go and the TS tests. If the two engines
disagree, the shared cases catch it.

## 5. Packs page

`frontend/routes/production/packs/` (skill `create-page-layout`):

```
+page.svelte        Page shell: packs VTable (name, components summary, price rule) + side Layer form
ProductPackForm.svelte   Name, price type + value, rounding; editable VTable of components
                         (type product/category, item, quantity) — no Inputs inside rows
packs.svelte.ts     State + service calls
packs.ts            Pure rules: validation, "1× Entrada + 1× Plato Fondo 1" summary, price preview
packs.test.ts
DOCUMENTATION.md    (skill document-user-routes)
```

Menu: add `production/packs` to the Catálogo menu in `db/modules.toml`, after `production/products`.
Service: `frontend/services/production/packs.ts` (delta cache).

## 6. Phases

1. Table + save/get endpoints + Packs page (CRUD, no effect on sales yet).
2. Go engine + shared case file + tests; hook into till sale and web checkout; `PackID` on lines.
3. Dining charge hook.
4. TS engine (parity on the shared cases) + till and floor-service previews, each showing the
   applied packs with their saving.
5. Web catalog file carries packs + web cart preview (prices only, no pack advertising).
6. Ticket: show the pack name over its grouped lines.

## 7. Risks to verify before phase 2

- **Line splitting**: check that stock reservation, `PendingDelivery*`, `StockExitPending`, the daily
  production close and sale summaries tolerate two lines with the same product + presentation.
- SUNAT boleta: a prorated line is just its unit price (no `descuento` element needed) — confirm
  facturago's totals stay consistent with prorated prices.
- Dining charge currently rebuilds lines from the table order and compares `lockedOrder.Lines` —
  the pack repricing must happen on the sale lines only, not on the order lines.

## 8. Settled questions (2026-11-14)

1. **Two engines.** The Go engine is authoritative; the TS engine shows the discount before saving.
   Parity is held by the shared case file (§4).
2. **Show which pack applied.** The till and the floor-service bill show a label per applied pack
   under its lines (e.g. "Menú del día −S/ 4.00"); the ticket groups the lines by `PackID` (phase 6
   is no longer optional for the till and the bill).
3. **Web store: applied quietly in the cart for now** — no promo block on the storefront.
