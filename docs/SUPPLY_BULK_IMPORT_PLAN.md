# Supply data: bulk API, sample generator, and Excel import/export

Three deliverables for `logistics/purchase-management` (Gestión de Compras):

1. `POST.product-supply` becomes bulk (array).
2. A new sample generator: 100 providers, 2000 products with supply config.
3. Excel export/import on the page, with a changed-rows-only preview modal and one bulk save.

Decisions already taken by the human: array on the same route; unknown provider name = row
error (compared with `normalizeStringN`); `Stock Actual` is export-only; the generator also fills
Capacity / Entrega / Precio and `SalesPerDayEstimated`; products are taken in ID order, not at
random.

---

## 1. Backend — `POST.product-supply` takes an array

`backend/logistics/product-supply-management.go`

- `PostProductSupply` unmarshals `[]types.ProductSupply` instead of one record. No compatibility
  branch for the object form — pre-alpha.
- Per record: sanitize provider rows, `ProductID > 0`, `MinimunStock >= 0`,
  `SalesPerDayEstimated >= 0`.
- Reject a payload that repeats the same `ProductID` — two rows for one product would race inside
  the same `db.Merge`.
- **Validate that every `ProductID` exists** and belongs to the company, in one query for the whole
  payload. Today the single-record handler does not check this at all; a bulk import of a sheet
  someone typed product names into makes it necessary.
- Cap the payload at 1000 records per call and return a descriptive error above it.
- `db.Merge` receives the whole slice at once, keeping the current merge/insert callbacks.
- Response: the saved records array.

`backend/logistics/types/product_supply_providers.go`

- `ValidateProviderSupplyRows` currently runs one provider query **per record**. Kept as-is for its
  one-record callers, but reimplemented on top of a new
  `ValidateProviderSupplyRowsBatch(companyID, rowsPerRecord [][]ProductSupplyProviderRow) error`
  that does the structural checks per record and **one** `ID.In(...)` query for the union of
  provider IDs. 2000 records must not mean 2000 round trips.
- Caller to check: `backend/production/supply_material.go` (unchanged behaviour).

Frontend counterpart in `supply-management.svelte.ts`: `postProductSupply` takes
`IProductSupplyRow[]`; the side Layer save sends a one-element array.

## 2. Generator — `generate_supply_data`

New `backend/tests/sample_records/generate_supply_data.go` + `supply_providers.json`
(100 static providers, same shape as `erp_history_providers.json`: name, personType 2,
registryNumber, countryId 604, cityId 150101).

Flow:

1. Seed the 100 providers through `POST.client-provider` (dedups), then read them back and resolve
   the 100 IDs by normalized name.
2. Take the first 2000 active products by ID.
3. Per product: 2 distinct providers drawn at random, `MinimunStock` 2–50,
   `SalesPerDayEstimated` 0–20, and per provider row `Capacity` 50–500, `DeliveryTime` 1–15 days,
   `Price` = 55–75 % of the product's sale price, varied ±10 % per provider so the two quotes
   differ.
4. Write in batches of 500 through the new bulk `POST.product-supply`.

Notes:

- **No clock manipulation.** Supply config is current state, not history — unlike
  `generate_erp_history`, this generator writes at the real clock.
- **Idempotent**, because `product_supply` is keyed by `ProductID` and written with `db.Merge`.
  Re-running overwrites the same 2000 rows instead of duplicating, so there is no resume file.

Wiring: `fn-generate-supply-data` in `backend/exec/`, a `generate_supply_data` case in
`scripts/main.go`, an `app.sh` branch, and `scripts/GENERATE_SUPPLY_DATA.md`.

Flags: `--products=2000`, `--providers-per-product=2`, `--min-stock=2,50`, `--sales-per-day=0,20`,
`--dry-run`.

## 3. Frontend — Excel export / import

New `frontend/routes/logistics/purchase-management/supply-management.excel.ts`, modelled on
`routes/production/products/products.excel.ts`. `@genix/ui/excel` already supports merged group
headers (`subcols`) and 2 header rows, so the 3×4 provider layout needs no library change.

Sheet layout (header rows 1–2):

| Producto | Stock Actual | Stock Mínimo | Ventas/Día Est. | Proveedor 1 (Proveedor, Capacidad, Entrega, Precio) | Proveedor 2 (…) | Proveedor 3 (…) |
|---|---|---|---|---|---|---|

- **Producto** is the row key: resolved on import by `normalizeStringN(name)` against the products
  service. Not found → row error. Two products normalizing to the same name → row error naming
  both, since picking one silently would write the config onto the wrong product.
- **Stock Actual** exports from the reconstructed movement stock and maps to no field, so the
  importer reads it and assigns nothing.
- **Precio** exports as a decimal (cents ÷ 100) and re-multiplies on import, matching what the side
  panel shows.
- **Proveedor** cells resolve by `normalizeStringN` too; an unknown name is a row error and the row
  is excluded from the bulk payload.

Import pipeline (`processSupplyImportFile`):

1. `loadFile` → `extractRecords(handler)`; the handler resolves product and provider names, builds
   `ProviderSupply[]`, and collects per-row errors.
2. Diff each resolved row against `productSupplyService.recordsMap`: `MinimunStock`,
   `SalesPerDayEstimated`, and the provider rows compared field by field in order.
3. **Return only rows that actually changed**, each carrying `_updatedFields` so the preview
   highlights the changed cells (`bg-purple-100`, as the Productos import does). A sheet round-
   tripped without edits produces an empty preview and a "no changes" notice.

UI in `ProductSupplyManagement.svelte`:

- Export and Import buttons in the toolbar, beside the record count.
- A `Modal` with `useFileImportWithErrors`, a `TableGrid` preview of the changed rows using the same
  column definitions, and the file's row errors listed underneath.
- **Guardar** sends every previewed row to the bulk route in batches of 500, then refreshes the
  service. All labels bilingual (`EN|ES`).

## 4. Decisions to confirm

- **More than 3 providers.** The spec says 3 groups. A product configured with 4 would silently
  lose one on export and have it deleted on re-import. Plan: the export emits 3 groups **or as many
  as the widest product needs**, whichever is larger; the import reads whatever groups the sheet
  has. The sheet is still the 3-group shape in practice.
- **Import replaces the provider list.** An empty provider slot in the sheet removes that provider
  from the product — the sheet is authoritative for the row, not additive.

## 5. Out of scope

- No changes to the warehouse stock ledger: `Stock Actual` is never written back.
- `GET.product-supply` is untouched; the delta cache keeps working off `upd`.
- Route `DOCUMENTATION.md` gets updated after the feature works (skill `document-user-routes`).
