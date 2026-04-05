# Generate Historical Sale Orders Plan

## Goal

Create a Go operational script that:

- Reads current product stock from `GET.productos-stock`.
- Picks 100 products and assigns stock between 100 and 500 units each.
- Adds SKUs to 15 of those 100 products, with random 10-character alphanumeric SKU codes and stock between 1 and 10 units per SKU.
- Creates sale orders for the last 30 days using `POST.sale-order`.
- Keeps the final sale distribution at:
  - 10% unpaid + undelivered
  - 10% paid only
  - 10% delivered only
  - 70% completed
- Uses 1 to 8 products per order and 1 to 5 units per line.
- Generates a random number of sale orders per day between 300 and 600.

## Important Findings

1. The backdate problem is not in `GetSaleOrders`.
   The user mentioned `backend/comercial/sale_orders_status.go`, but the order date is assigned inside `PostSaleOrder` in `backend/comercial/sale_order_create.go`:
   `sale.Fecha = core.TimeToFechaUnix(time.Now())`

2. Changing global time is not the safe approach.
   Payment and warehouse side effects also stamp `time.Now()` / `core.SUnixTime()`:
   - `backend/comercial/sale_order_create.go`
   - `backend/finanzas/caja-movimientos-apply.go`
   - `backend/logistica/product-stock-movement.go`

3. If only `sale.Fecha` is backdated, the database will contain mixed historical/current timestamps.
   That would leave sale date in the past, but payment movement date, delivery movement date, and `Updated`/`Created` timestamps in the present.

## Recommended Approach

Implement a scoped historical-time override for scripts instead of changing process-global time.

### Phase 1: Add a minimal historical timestamp override

Add a small helper in `backend/core` that resolves an effective date/time for internal script-driven writes.

Proposed shape:

- Add a helper that reads an optional request-scoped override from `HandlerArgs`.
- Return:
  - effective `time.Time`
  - effective `fecha unix`
  - effective `sunix`

Rationale:

- Keeps the override explicit.
- Avoids mutating the global clock.
- Lets the script generate historical records consistently.

### Phase 2: Thread the override through the write paths

Update these write paths to use the effective time helper:

- `backend/comercial/sale_order_create.go`
  - order `Fecha`
  - `Created`
  - `Updated`
  - payment/delivery audit fields
- `backend/finanzas/caja-movimientos-apply.go`
  - movement `Fecha`
  - movement `Created`
  - caja `Updated`
- `backend/logistica/product-stock-movement.go`
  - warehouse movement `Fecha`
  - warehouse movement `Created`
  - stock `Updated`

Rationale:

- Sale order, cash movement, and warehouse movement must share the same historical date.
- This keeps summaries and operational traces coherent.

### Phase 3: Create the data generation script

Create a new script in `scripts/` and register it in:

- `scripts/main.go`
- `app.sh`
- a new companion doc under `scripts/`

Proposed name:

- `scripts/generate_sale_orders.go`
- command: `./app.sh generate_sale_orders`

### Phase 4: Script flow

1. Resolve script context
   - Use fixed execution context:
     - company `1`
     - user `1`
     - warehouse `1`
     - caja `1`

2. Build the 30-day date range
   - Use `core.TimeHelper`.
   - Generate an ordered slice from 30 days ago to today.

3. Load stock candidates
   - Call `GetProductosStock` for the target warehouse.
   - Build a distinct product list from current stock rows.

4. Select the 100 products
   - Prefer products already represented in stock for the target warehouse.
   - If there are fewer than 100 candidates, stop with a clear error instead of inventing data.

5. Seed stock for the 100 products
   - Use `POST.productos-stock`.
   - For each selected product, set base stock between 100 and 500.
   - For 15 selected products, add extra SKU-specific stock rows with random 10-char alphanumeric codes and quantity between 1 and 10.

6. Build an in-memory stock ledger
   - Track available stock per stock key:
     `warehouse + product + presentation + sku + lote`
   - Deduct only from this in-memory ledger while generating orders.
   - Never ask `POST.sale-order` to consume more than available stock.

7. Generate sale orders day by day
   - For each date in the 30-day range, create a random number of orders between 300 and 600.
   - Each order includes 1 to 8 products.
   - Each line uses quantity 1 to 5.
   - Use price data from the product records if required by the sale payload.

8. Apply status distribution
   - Apply the status mix across the full generated dataset.
   - Then update each order according to the required mix:
     - no actions
     - payment only
     - delivery only
     - payment + delivery
   - When calling the handlers, inject the historical date override for the specific day.
   - Because daily totals are random, compute exact target counts after generation so the final percentages stay as close as possible to:
     - 10% unpaid + undelivered
     - 10% paid only
     - 10% delivered only
     - 70% completed

9. Add extensive debug logging
   - Chosen warehouse/caja
   - selected products count
   - SKU products count
   - generated orders per day
   - status distribution counters
   - stock remaining summary

## File Changes Planned

- `backend/core/...`
  - add a small request-scoped effective-time helper
- `backend/comercial/sale_order_create.go`
  - stop using raw `time.Now()` for sale-order writes
- `backend/finanzas/caja-movimientos-apply.go`
  - stop using raw current time for historical script writes
- `backend/logistica/product-stock-movement.go`
  - stop using raw current time for historical script writes
- `scripts/generate_sale_orders.go`
  - new operational script
- `scripts/main.go`
  - register command
- `app.sh`
  - expose command
- `scripts/GENERATE_SALE_ORDERS.md`
  - usage and parameters

## Open Questions To Confirm Before Implementation

1. The script context is fixed to company `1`, user `1`, warehouse `1`, caja `1`.

2. The script will use one warehouse only.

3. Output does not need to be deterministic.

4. Daily volume will be random between 300 and 600 orders, so the total 30-day volume will vary run to run.

## Recommendation

Do not edit `backend/comercial/sale_orders_status.go` for the historical-write feature.

The minimal correct path is:

- add a request-scoped effective-time helper
- update the sale/cash/warehouse write paths to use it
- implement the generator script on top of those handlers

This keeps the system behavior explicit and avoids hidden global time hacks.
