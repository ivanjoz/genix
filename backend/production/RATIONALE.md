# RATIONALE — production

Design decisions for the product catalog and its supplies, newest first.

## `GET.products` evicts a supply by id instead of shipping it with `ss=2`

**Context** — `Delta(updatedSince, 1)` pinned `Status=1` only on a first sync; every later delta
fanned out over all three declared statuses so the client could evict deletions. That also handed
it every supply (`Status=2`), and the frontend `GetHandler` evicts on `ss === 0` alone, so supplies
piled up in the products cache. `sale_order_create` joins warehouse stock against that cache and
rendered a supply as a sellable row — with `NaN` for a price, because a supply is saved with a
purchase `Price` and no `FinalPrice`.

**Decision** — the handler now pins `Status.Equals(ProductStatusActive)` and calls `Delta()` with
no filter values, so the record bodies are Status=1 on every sync. Two extra ID-only delta scans
(status 0 and status 2) feed a `records_IDsToRemove` key, which is the delta cache's existing
eviction channel. The response shape goes from a bare array to
`{records, records_IDsToRemove}`, matching `GetExpenses` and `GetSaleOrdersStatus`; the frontend
service reads `result.records` and its cache `ver` went 12 → 13.

**Rationale** — the cheaper-looking fix was to leave the row in the response and rewrite its
`Status` to 0 so `inferRemoveFromStatus` would drop it. That does not work: `addSavedRecords` calls
`recordsMap.set` *before* the tombstone check, so an `ss=0` row stays reachable through
`recordsMap.get(id)` — which is exactly how `sale_order_create` looks products up. Only the
`_IDsToRemove` channel deletes the row from the IndexedDB snapshot, so `handler()` never sees it at
all. The cost is a second and third delta query per sync (ID-only, and skipped entirely on a first
sync) and a response shape that is no longer a bare array.

## The catalog leaves `business` and becomes its own module

**Context** — `business` had grown into two unrelated jobs: company infrastructure (sites,
warehouses, image assets, shared lists) and the product catalog. The roadmap adds production
recipes, production orders, per-unit costing, process stages, waste control and recipe versioning —
all of which key off `Product`. Piling them into `business` would have made a module that owns
everything and explains nothing.

**Decision** — a new L4 module `app/production` owns `Product` and everything hanging off it:
`production/types/product.go` holds the row, its table, its presentations, properties and status
vocabulary; `production/products.go` holds the catalog handlers. `business/types/productos.go`
was split, with `Site` and `Warehouse` staying behind in `business/types/sites_warehouses.go`.

**Rationale** — the alternative was a `business/product/` subpackage, which keeps the ownership
ambiguous and leaves every consumer importing `app/business/types` for two different domains. A
real module boundary is what makes the upcoming recipe/order/costing tables have an obvious home,
and what lets `check_module_imports` police the edges. The cost is a wide but mechanical import
rewrite across 25 files, and two aliases (`production` and `business`) in the handful of files
that genuinely touch both.

## Supplies came from `logistics`, the replenishment config did not

**Context** — `GET/POST.supply-material` lived in `logistics`, but a supply *is* a `Product` row
at `Status = 2`: same table, same stock engine, same movement ledger. Meanwhile
`ProductSupply` — minimum stock and provider rows per product — also lived in `logistics`.

**Decision** — `supply-material-management.go` moved here as `production/supply_material.go`.
`product-supply-management.go` stayed in `logistics`.

**Rationale** — the split follows who owns the row. The supply catalog writes `Product`, so it
belongs with the catalog; `ProductSupply` is a purchasing policy (when to reorder, from whom), so
it belongs with purchase orders. `PostSupplyMaterial` still writes both, because the user sees one
record — it imports `logistics/types` for `ProductSupply`, an L4 → L2 edge the boundary rule
allows.

## Provider-row validation moved into `logistics/types`

**Context** — `sanitizeProviderSupplyRows` and `validateProviderSupplyRows` were unexported
helpers in the `logistics` body, and the supply-material handler that needed them left the module.

**Decision** — they are now `logistics/types.SanitizeProviderSupplyRows` and
`.ValidateProviderSupplyRows`, in `logistics/types/product_supply_providers.go`. The validator
takes a `companyID int32` instead of `*core.HandlerArgs`.

**Rationale** — a module body may not import another module body, so the only shared home is
`<module>/types`, and the rows they validate (`ProductSupplyProviderRow`) already live there.
Dropping `HandlerArgs` for the one field it actually used keeps the HTTP layer out of `types`.

## Two constants were exported so `business` and `production` could share them

**Context** — `imageConfigDigitFull` was declared in `products.go` and read by
`business/images.go` (the image ID counter); `cacheGroupProducts` was declared in
`business/product-ecommerce.go` and written by `PostProducts`. Both crossed the new boundary.

**Decision** — `business/types.ImageConfigDigitFull` and
`business/types.CacheGroupProducts/Brands/Categories` (in `business/types/ecommerce_cache.go`).

**Rationale** — `business` owns the image ID encoding and the ecommerce snapshot, so the constants
stay on its side of the line and `production` imports them. Duplicating the literals in both
modules would have been smaller but would let the two drift silently, and both encode a value that
is already persisted in image IDs and cache rows.
