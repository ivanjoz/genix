# USD sales at the till — plan

Goal: when the sale is collected into a USD cash-bank, the whole sale is in dollars — line
prices, subtotal, total, the cash movement and the SUNAT document. A product priced in USD
keeps its price; a product priced in PEN is converted with the company's latest sell rate
plus the "Spread de tipo de cambio" flag (flag 7).

Settled with the user: backend + frontend + USD CPE · latest day's **sell** rate · the spread
always favours the store · product `CurrencyID` 0 is read as PEN (no migration) · unpaid
sales pick their currency explicitly · reports convert USD to PEN · a rate older than 7 days
blocks a cross-currency sale.

## Rules (backend is the authority, the till only previews them)

- **Sale currency** (`CurrencyType`, 1 PEN, 2 USD): paid now → the payment cash-bank's
  currency; unpaid → the currency the till picked. The server rejects a paid sale whose
  cash-bank currency differs from the sent `CurrencyType`.
- **Rate** = sell rate of the most recent day with a non-zero sell rate in `exchange_rates`
  for the company, ×1000 like the rate table. Only loaded when a line crosses currencies.
  No rate, or the latest one older than 7 days → the sale is rejected with a clear message.
- **Spread** (flag 7) always favours the store, so it moves the rate in opposite directions:
  - PEN product in a USD sale: `round(price × 1000 / (sell − spread))` — more dollars.
  - USD product in a PEN sale: `round(price × (sell + spread) / 1000)` — more soles.
  - same currency: unchanged.
  Per unit, rounded to cents, before multiplying by quantity — so the printed lines always
  add up to the total. Applies to both `FinalPrice` and `SbuFinalPrice`. The sale stores the
  effective rate of its direction in `ExchangeRate`.
- **700-sol boleta rule** compares the PEN equivalent: `total × rate / 1000` for a USD sale.

## Backend

1. `sales/types/sales.go` — two new SaleOrder columns (schema change):
   `CurrencyType int8` and `ExchangeRate int32` (×1000, spread included; 0 on a PEN sale
   with no USD product). `DetailPrices`, `DetailSubPrices`, `TotalAmount`, `TaxAmount` are
   stored **in the sale currency**.
2. `finance/types` — `LatestSellRate(companyID) (rate int32, day int16, err)`: reads the
   company's months newest first and returns the last filled sell day. Lives in `types`
   because `sales` may not import the `finance` body.
3. `sales/sale_order_create.go` — `validateSaleOrderLines` also selects `CurrencyID`; resolves
   the sale currency from the cash-bank, loads the rate + company flag 7 only when some line
   crosses currencies, converts the unit prices, recomputes the total. Rejects a cash-bank
   that does not exist.
4. Annulment reverses the cash movement by its own amount, which is already in the
   cash-bank's currency — no change. (There is no "pay a pending sale later" endpoint yet;
   when one appears it must reject a cash-bank in another currency.)
5. `invoicing/types/sale_order_to_cpe.go` + `document_reserve.go` — `Currency` from the sale
   (`"USD"` / `CurrencyUSD`) instead of the PEN constant. facturago already supports USD
   (currency code in the XML, "DOLARES AMERICANOS" in the legend) — no submodule change.
   `invoicing.RequiresCustomerIdentity` gets the PEN equivalent.
6. `sales/sale_summary*.go` — stats accumulate PEN: a USD line amount is converted back with
   the sale's stored `ExchangeRate`.
7. Tests: conversion both directions, spread direction, rounding, missing/stale rate, caja
   currency mismatch.

## Frontend

1. `services/production/products.svelte.ts` + products page/Excel — rename `MonedaID` →
   `CurrencyID` (8 references), so the product's currency actually round-trips.
2. `routes/finance/exchange-rate/exchange-rate.ts` — pure `latestSellRate(records)`, same rule
   as the backend.
3. `routes/sales/sale_order_create/sale_order.ts` — pure `convertUnitPrice(price,
   productCurrency, saleCurrency, rate)` + tests.
4. `sale_order.svelte.ts` — `recalcTotales` and the cart's PRECIO column use converted prices;
   `form.CurrencyType` follows the picked cash-bank when paid.
5. `+page.svelte` — loads `ExchangeRatesService`; total/subtotal shown with `US$` / `S/`.
   Paid: the existing "Moneda: Dólares" box, plus the rate used and its date. Unpaid: that
   box becomes a PEN/USD selector beside the due date. A missing or stale rate shows as a
   warning before submit. Recalculates when the cash-bank or currency changes.
6. Ticket + local history: `TOTAL US$` for a USD sale (`sale_ticket.ts`, `sale_history.ts`).
7. `SaleOrdersTable.svelte` (sales lists): currency marker on the amount.
8. Mirror `CurrencyType` / `ExchangeRate` in `ISaleOrder`.

