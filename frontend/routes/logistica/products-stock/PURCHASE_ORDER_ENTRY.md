# Purchase Order Entry — Sub-page Spec

## 1. Goal

A new sub-page that lets the user receive merchandise from a previously created Purchase Order (OC). The user picks an OC, optionally types a lot code, and clicks products in a left-side card list to register them as incoming stock entries on the right-side panel — optionally grouped under a lot code, with optional expiration date and serial numbers per entry.

**Scope of this iteration: visual UI only.** Use existing services to read data (purchase orders, products, warehouses). DO NOT wire any save/post handler — the "Guardar" button is out of scope.

## 2. Page wiring (`+page.svelte`)

The Page already declares two tabs: `{ id: 1, name: "Movimiento" }` and `{ id: 2, name: "Ingreso OC" }`. Use `Core.pageOptionSelected` (from `$core/store.svelte`) to render conditionally:

- `Core.pageOptionSelected === 1` → `<ProductStockMovement />` (existing)
- `Core.pageOptionSelected === 2` → `<PurchaseOrderEntry />` (new file in this folder)

Both children mount lazily via `{#if}` so the inactive tab does not run its services.

## 3. Layout (`PurchaseOrderEntry.svelte`)

Two-column flex layout, mirroring `frontend/routes/logistica/purchase-orders/PurchaseOrderCreate.svelte:398–519`:

```svelte
<div class="flex h-full gap-20">
  <div class="flex-1 flex flex-col min-w-0 relative">
    <!-- Left column: order picker + lot input + product cards -->
  </div>
  <LayerStatic
    css="w-[50%] min-w-350 bg-white border-l border-gray-200 flex flex-col h-[calc(100vh-var(--header-height))] shadow-lg md:-m-10"
    mobileLayerTitle="Ingreso de OC"
    useMobileLayerVertical={124}
  >
    <!-- Right column: order header + warehouse selector + entries table -->
  </LayerStatic>
</div>
```

```
┌──────────────────────────────────────────┬──────────────────────────────┐
│ [Orden de Compra ▼] [LOTE…]    [Filtro…] │  Orden #1234   [Almacén ▼]   │
│ ┌──────────────────────────────────────┐ │  ─── SIN LOTE ───            │
│ │ Producto 1               │ │  ┌──────┬───────────┬──────┬──────┐      │
│ │ Producto 2               │ │  │ Prod │ Vencim.   │ Cant │ S/N  │      │
│ │ Producto 3               │ │  └──────┴───────────┴──────┴──────┘      │
│ │ ...                      │ │  ─── LOTE 5555 ───                       │
│ └──────────────────────────┘ │  (rows…)                                 │
└──────────────────────────────┴──────────────────────────────────────────┘
```

### 3.1 Left column

**Top row** (single horizontal row): `[Orden de Compra ▼] [LOTE…] ────flex spacer──── [FilterInput]`. Use `flex items-center gap-8` with the filter pushed to the right (`ml-auto`).

1. **Purchase order picker** — `SearchSelect` over `PurchaseOrdersService.records`, **filtered to status `CONFIRMED`** (exclude `PENDING`, `FULFILLED`, `CANCELED`). Show order number + date in the option label. `keyId="ID"`.
2. **Lot input** — single text input, label `LOTE…`, placeholder `Código de lote (opcional)`. Empty value means the next clicked product goes to the "SIN LOTE" section.
3. **Product filter** — `FilterInput` from `$components/micro/FilterInput.svelte`, bound to a `cardFilterText` `$state('')`. Placeholder: `Filtrar por nombre o SKU…`. Pushed to the right end of the top row (`ml-auto w-220`). Filters the product card list (not the right-side table) by case-insensitive substring match against:
   - `producto.Nombre`
   - `producto.SKU` (product-level SKU; field is uppercase `SKU` per `frontend/routes/negocio/productos/productos.svelte.ts:50`)
   - **All** `producto.Presentaciones[].sk` (presentation SKU; the field is `sk?: string`, defined at `productos.svelte.ts:19`) — match if any presentation SKU contains the term.

   The match is computed via a memoised lower-cased haystack per product (built once when the OC changes); the `FilterInput` already lowercases + trims its `value` binding.
4. **Empty state** — when no order is selected, replace the product card list with a single full-width banner:
   - Text: `Seleccione una Órden de Compra`
   - Style: red text on light-red background, centered, padded.
5. **Product cards** — once an order is selected, render one card per line in `DetailProductIDs[]` (filtered by `cardFilterText`). Each card shows:
   - Product name (from `ProductosService.recordsMap.get(productID).Nombre`).
   - Progress badge in muted text: `Ingresado: {addedSoFar} / {ordered}` where `ordered = DetailQuantities[i]` and `addedSoFar = sum of entries.quantity for that productID across all lots`.
   - Whole card is clickable.
6. **Click behavior** — clicking a card adds **+1** to the matching entry row, where "matching" is defined as same `productID` AND same current `lotCodeInput` value:
   - If a matching row already exists → `row.quantity += 1`.
   - Otherwise → push a new entry row with `quantity = 1`, `lotCode = lotCodeInput` (`''` → SIN LOTE group), `expirationDate = ''`, `serialNumbers = []`.
   - The user can still hand-edit the row's `quantity` cell on the right.

### 3.2 Right column (`LayerStatic`)

Use `LayerStatic` with the same css/props as `PurchaseOrderCreate.svelte:415–418` (see §3 layout snippet).

- **Header bar** (top of layer): same pattern as `PurchaseOrderCreate.svelte:426–457` — flex row with `border-b border-gray-100 bg-gray-50/50`:
  - Left: title `Orden #{orderNumber}` (or "—" before any selection), bold/h2.
  - Right: warehouse selector — `SearchSelect` over `AlmacenesService.Almacenes`, **editable**, default = `selectedOrder.WarehouseID`.
- **Body**: the entries `TableGrid` (see §4).

## 4. TableGrid enhancement: row renderer

The right-side body is a `TableGrid` of *entry rows*, but it must also render *lot section headers* in-between. Add support for custom row rendering.

### 4.1 New props on `TableGrid<TRecord>`

```ts
useRowRenderer?: (record: TRecord, rowIndex: number) => boolean
rowRenderer?: Snippet<[TRecord, number]>   // (record, rowIndex)
```

When `useRowRenderer?.(record, rowIndex)` returns `true`, render `{@render rowRenderer?.(record, rowIndex)}` spanning the full row width (single cell with `grid-column: 1 / -1`) instead of the per-column cells. Otherwise fall back to the existing column-based rendering. The row must still respect `rowHeight` and the virtual-list slot contract.

### 4.2 Edit points (file: `frontend/ui-components/vTable/TableGrid.svelte`)

- Add a `TableGridRowRendererSnippet<T>` type in `./types.ts` next to `TableGridCellRendererSnippet`.
- Add the two props to the `TableGridProps<TRecord>` interface (lines 17–39) and destructuring block (lines 41–63).
- Patch both row iterations (plain scroll ~line 343, virtual scroll ~line 429): wrap the existing per-column `{#each visibleColumns}` block in an `{#if useRowRenderer?.(rowRecord, rowIndex)}` branch that renders the snippet in a single full-span cell; else keep current behavior.
- The full-span cell needs `style="grid-column: 1 / -1"` and a class hook (e.g. `tg-row-custom`) so callers can style the section header.

### 4.3 Caller usage in this page

The `data` array passed to `TableGrid` is a *flat* list mixing two row shapes:

```ts
type EntryRow = {
  kind: 'entry'
  productID: number
  expirationDate?: string
  quantity: number
  serialNumbers: string[]
  lotCode: string  // '' for no-lot
}

type LotHeaderRow = {
  kind: 'lot-header'
  lotCode: string  // '' renders as "SIN LOTE"
}

type Row = LotHeaderRow | EntryRow
```

Build `data` by grouping entries by `lotCode` (preserve insertion order; "SIN LOTE" group always rendered if it has entries). For each group, push one `LotHeaderRow` followed by its `EntryRow`s.

`useRowRenderer` returns `row.kind === 'lot-header'`. The snippet renders `LOTE {lotCode}` (or `SIN LOTE` if empty) in red, bold, with a slight top margin.

### 4.4 Columns

| Column      | Width      | Render                                                                   |
|-------------|------------|--------------------------------------------------------------------------|
| Producto    | flex       | Product name from `ProductosService.recordsMap`                          |
| Vencimiento | 130px      | `<input type="date">` bound to `row.expirationDate`                      |
| Cantidad    | 80px       | Editable number cell (use existing `cellInputType: 'number'`)            |
| S/N         | 90px       | See §4.5                                                                 |
| (delete)    | 40px       | Icon button (`icon-trash` / `buttonDeleteHandler` on the column) — removes the entry row from `entries`. If it was the last row of its lot group, the group's `LotHeaderRow` disappears via the grouping `$derived`. |

### 4.5 S/N cell

Right-aligned, contents:
- If `row.serialNumbers.length === 0` → text `-` then a circular `+` button.
- Else → text `{row.serialNumbers.length}` then a circular pencil/edit button.

Clicking the button opens the **Serial Number modal** (see §6) for that row.

## 5. Mobile responsiveness

Out of scope for first pass. Use `LayerStatic` defaults (desktop always visible, mobile off-canvas via `show` prop). If the product card list and right panel can't fit side-by-side, document that as a follow-up.

## 6. Serial Number modal

Reuse the same visual pattern as the SerialNumber side-layer in `ProductStockMovement.svelte` (Layer `id={2}`, side, `sideLayerSize={740}`).

- Title: `{ProductName} — {n} Serial(es)`.
- Body: `TableGrid` with two columns:
  1. **Serial** — text input.
  2. **Cantidad** — number input.
- Auto-append a blank pending row when the user types into the last empty row's Serial cell (mirror `addPendingSerialNumberRow` logic).
- On close: write the resulting serial list back to `row.serialNumbers` (just the array; we don't post anywhere).

Use a unique `Layer` `id` not currently in use (e.g. `id={4}`).

## 7. State shape (top of `PurchaseOrderEntry.svelte`)

```ts
const purchaseOrders = new PurchaseOrdersService()
const productos = new ProductosService(true)
const almacenes = new AlmacenesService()

let selectedOrderID = $state(0)
let lotCodeInput = $state('')
let warehouseID = $state(0)
let cardFilterText = $state('')          // bound to FilterInput; pre-lowercased/trimmed
let entries = $state<EntryRow[]>([])    // flat list, grouping done in $derived

const selectedOrder = $derived(
  purchaseOrders.records.find(o => o.ID === selectedOrderID)
)
const orderProducts = $derived.by(() => {
  if (!selectedOrder) return []
  return selectedOrder.DetailProductIDs.map((pid, idx) => ({
    productID: pid,
    quantity: selectedOrder.DetailQuantities?.[idx] || 0,
  }))
})
const tableRows = $derived.by((): Row[] => {
  // group `entries` by lotCode, prefix each group with a LotHeaderRow
  ...
})
```

When the order changes, reset `entries = []` and default `warehouseID` to the order's `WarehouseID`.

## 8. Files to create / edit

| Path | Action |
|---|---|
| `frontend/routes/logistica/products-stock/PurchaseOrderEntry.svelte` | **Create** |
| `frontend/routes/logistica/products-stock/+page.svelte` | **Edit**: import + tab routing |
| `frontend/ui-components/vTable/TableGrid.svelte` | **Edit**: add `useRowRenderer` + `rowRenderer` props |
| `frontend/ui-components/vTable/types.ts` | **Edit**: export `TableGridRowRendererSnippet<T>` |

No backend or service changes.

## 9. Resolved decisions (from review round 1)

1. **Card click**: increment the existing matching row's `quantity` (match = same `productID` + same current lot input). New row only when no match.
2. **Default Quantity** when creating a new row: always `1`.
3. **Progress badge** on each product card: `Ingresado: {addedSoFar} / {ordered}` (compact text, muted).
4. **Delete button**: yes — final column on the right table (per row).
5. **Lot input vs existing rows**: lot is frozen on the row at click time. Changing the input later does not move existing rows.
6. **Warehouse selector**: editable, defaulted to the OC's `WarehouseID`.
7. **OC picker filter**: only `CONFIRMED` orders.
8. **Right-side container**: `LayerStatic` with the exact same css/props as `PurchaseOrderCreate.svelte:415–418`.
