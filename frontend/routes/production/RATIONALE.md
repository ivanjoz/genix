# RATIONALE — production (frontend)

Design decisions for the Producción module, newest first.

## `ProductsService` left the route folder for `services/production/`

**Context** — `ProductsService`, `IProduct` and the product image helpers lived in
`routes/business/products/products.svelte.ts` and were imported by 15 files across `sales`,
`logistics` and `accounting`. Routes are supposed to be leaves; those imports had routes reaching
sideways into another route's folder, with a mix of `$routes/...` and `../../` paths.

**Decision** — `services/production/products.svelte.ts`. Every consumer now imports
`$services/production/products.svelte`. `routes/production/products/` keeps only the page, its
sub-components and the Excel import/export.

**Rationale** — the service is a shared API connector, which is exactly what `services/` is for,
and the one-way import flow (`libs → services → domain-components → routes`) becomes true again.
Doing it during the module move cost nothing extra: the 15 import paths had to be rewritten either
way.

## Supplies & Materials sits under Producción, Gestión de Compras does not

**Context** — the two pages are adjacent in the UI and share `supply-management.svelte`, but they
write different tables: the supplies page writes the `Product` catalog, the purchase-management
page writes the replenishment policy.

**Decision** — `routes/production/supplies-materials/` moved; `routes/logistics/purchase-management/`
stayed. The supplies page imports the shared provider-row types from
`$routes/logistics/purchase-management/supply-management.svelte`.

**Rationale** — mirrors the backend split (see `backend/production/RATIONALE.md`). The remaining
cross-route import is the one thing left to clean up if `supply-management.svelte` grows: it would
become `services/logistics/supply-management.svelte.ts` by the same argument as `ProductsService`.
