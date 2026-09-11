# PLAN — Local sale history + thermal ticket printing (`/sales/sale_order_create`)

> **Status: implemented.** The row shape below was superseded during the build — the history
> stores references only (`saleID` as the sole primary key, `clientID` / `warehouseID` /
> `cashierID` / `cashBankID` / `status`, no names, no `seriesCode`, no `docType`), and everything
> else is resolved through a `TicketContext` of the record maps the page already loads. The
> reasoning that survives is in `frontend/routes/sales/sale_order_create/RATIONALE.md`.

Feature folder: `frontend/routes/sales/sale_order_create/`

## Confirmed by the user

| Decision | Answer |
|---|---|
| Ticket kind | Depends on the series: sale id tail `00` → internal receipt; any other series → fiscal document (boleta/factura) |
| History scope | Per company + env, cap 200 rows (same pattern as `core/notifications.idb.ts`) |
| Print method | Print the modal with `@media print` rules |
| Ticket widths | 58 mm (32 cols) and 80 mm (48 cols) |

## What already exists (no new endpoint needed)

`POST sale-order` returns the created `SaleOrder`, and the document number is packed into its id
(`backend/sales/types/sale_order_id.go`):

- `seriesID  = saleID % 100` — `0` means the till named no comprobante
- `correlativo = Math.floor(saleID / 10000)`
- the series **code** (`F001`) comes from `EmpresaParametrosService.empresa.InvoiceSeries`, matched by `SeriesID`

So the ticket can print `F001-00000130` without a round trip. Only the SUNAT verdict is async, and
it is not printed.

## Files

### New — `sale_history.idb.ts`
Dexie store, DB name `${companyID}_sale_history_${env}`, table `sales: '++id, savedAt'`, pruned to
the newest 200 on hydrate. Row is a **self-contained snapshot** so a ticket reprints identically
even after the catalog changes:

```ts
export interface SaleHistoryLine {
  productName: string
  quantity: Quantity          // { units, sub }
  subDivisor: number
  subUnitName: string
  unitPrice: number           // cents
  subUnitPrice: number        // cents
  amount: number              // cents, line total
  serialNumbers: string[]
}

export interface SaleHistoryRow {
  id?: number
  saleID: number
  savedAt: number             // ms, local clock — this is a local log, not a persisted date
  seriesCode: string          // "" when seriesID is 0
  docTypeName: string         // "Boleta" / "Factura" / ""
  clientName: string
  clientRegistryNumber: string
  warehouseName: string
  cashierName: string
  cashBankName: string
  isPaid: boolean
  isDelivered: boolean
  totalAmount: number         // cents
  taxAmount: number           // cents
  lines: SaleHistoryLine[]
}
```

Exports: `loadSaleHistory()`, `appendSaleHistory(row)`, `clearSaleHistory()`.

### New — `sale_ticket.ts` (pure, unit-testable — no Svelte, no fetch)
- `TICKET_58MM = 32`, `TICKET_80MM = 48`, `TICKET_WIDTH_OPTIONS` for `CheckboxOptions`
- `saleDocumentNumber(row)` → `"B001-00000130"`, or `"VENTA #1301100"` when there is no series
- `renderSaleTicket(row, company, columns): string` — the full monospace ticket text
- helpers `padColumns(left, right, columns)`, `centerText`, `wrapText`, `dividerLine`

Ticket layout (company header, doc number, date/cashier/client, lines, totals, footer) is built
here only — the Svelte side just renders the returned string in a `<pre>`.

### New — `sale_ticket.test.ts`
Vitest, mirroring `routes/accounting/invoicing/invoicing.test.ts`: no line exceeds the column
width, amounts stay right-aligned, long product names wrap, series 0 vs series N numbering.
⚠ `vitest` is **not installed** in `frontend/node_modules` — the file follows the repo convention
but I will not be able to execute it.

### New — `SaleHistoryCards.svelte`
Cards for the history view: doc number / sale id, time, client, item count, total, paid + delivered
chips, and a printer icon button on the right that opens the ticket modal.

### New — `SaleTicketModal.svelte`
- `CheckboxOptions` (`useButtonsSlim`, `type="single"`) at the top to switch 58 mm / 80 mm
- `<pre class="ticket-print-root">` with a monospace stack, sized in **mm** so the on-screen preview
  is the physical ticket: `font-size: 2.5mm`, char width `1.5mm` → 32 cols = 48 mm, 48 cols = 72 mm
- a Print button calling `window.print()`
- print rules scoped in the component:
  ```css
  @media print {
    :global(body *) { visibility: hidden !important }
    .ticket-print-root, .ticket-print-root * { visibility: visible !important }
    .ticket-print-root { position: fixed; inset: 0 auto auto 0; padding: 5mm }
  }
  ```
  plus a `<svelte:head>` `@page { size: 58mm auto; margin: 0 }` rebuilt from the selected width.

### Changed — `sale_order.svelte.ts`
`postSaleOrder` returns `{ sale, soldProducts } | undefined` instead of `boolean`, capturing the
cart **before** it resets, so the page can build the history snapshot. Pre-alpha: the old boolean
return is removed, not kept alongside.

### Changed — `+page.svelte`
- `OptionsStrip` replaces the `Detalle de Venta` title inside `LayerStatic`:
  `[[1, "New Sale|NUEVA VENTA"], [2, "History|HISTORIAL"]]`, bound to a `layerView` state
- `layerView === 1` → the current totals bar, form rows and cart table (unchanged)
- `layerView === 2` → `SaleHistoryCards`
- on a successful `handlePostSaleOrder`: build the row from the returned sale + company series +
  selected client/warehouse/caja + `security.getUserInfo()`, `appendSaleHistory`, then
  `layerView = 2` so the new sale is on screen with its print button

### Changed — `RATIONALE.md`
Records only my own calls: the raw-values snapshot (vs. pre-formatted text), zero-padding the
correlativo to 8 digits (the invoicing panel does not pad), the `postSaleOrder` return-type change,
and the mm-based ticket sizing.

## Out of scope
No backend change, no new route, no reprint of sales made on another browser — the history is local
by design.
