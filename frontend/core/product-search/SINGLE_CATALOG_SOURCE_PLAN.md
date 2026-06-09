# Single catalog source — refactor plan

## Goal
One catalog fetch for the whole storefront. `ProductEcommerceDataService` (delta cache +
CDN snapshot) becomes the single source. It loads once, at app start, behind a shared
promise; every other consumer (`ProductSearch`, category grids, category lookups) awaits
that same promise instead of fetching again.

## Problem today
- `ProductSearch` builds its own `new ProductEcommerceDataService()` internally and only
  loads it lazily, on the first search interaction.
- `productos.svelte.ts` fetches a *different* endpoint (`p-productos-cms`) via
  `getProductos()`, populating `productosServiceState` + the category map.
- Net: the catalog is fetched twice, from two endpoints, with no shared promise.

## Changes

### 1. `core/product-search/productos-delta-service.ts` — shared singleton
Add module-level accessors:
- `getProductEcommerceData(): Promise<ProductEcommerceDataService>` — lazily constructs a
  single instance, calls `load()` once, memoizes the promise (clears it on failure so a
  later call can retry), and returns the loaded instance.
- `getLoadedProductEcommerceData(): ProductEcommerceDataService | null` — sync accessor for
  the already-loaded instance (null before ready).

### 2. `core/product-search/product-search.ts` — consume the singleton
- Drop `private readonly data = new ProductEcommerceDataService()`.
- In `bootstrap()`: `this.data = await getProductEcommerceData()` (field becomes assigned,
  not constructed). `buildIndex()` reads `this.data` as before.

### 3. `services/services/productos.svelte.ts` — back lookups with the singleton
- Add `ensureProductosLoaded()`: awaits `getProductEcommerceData()`, then syncs
  `productosServiceState` (productos, productosMap, categorias, categoriasMap,
  productosByCategoryMap) from the loaded service **once**.
- Rewrite `getProductsByCategoryID(id)` and `getCategoryByID(id)` to await
  `ensureProductosLoaded()` and read from the synced maps (drop the `getProductos()` /
  `loadingPromise` path).
- Remove `getProductos()` and `productosPromiseMap` — fully dead after this refactor.
- Note: the delta `categorias` schema carries `ID`/`Name` only (no `Description`), so
  `getCategoryByID` returns name-only categories — `CategoryDescription.svelte` keeps
  rendering, description just won't be present from the snapshot.

### 4. `ecommerce/routes/+layout.ts` — stop the second fetch
- Remove `await getProductos(undefined)`; keep the `_isLocal` detection. Return `{}`.

### 5. `ecommerce/routes/+layout.svelte` — kick off the single load
- Remove the `data.productos.*` assignments.
- In `onMount`, call `ensureProductosLoaded()` alongside the existing
  `preloadProductSearch()`. Both resolve to the same singleton load — one fetch total.

### 6. `backend/business/productos.go` + `backend/business/main.go` — remove dead handler
- Delete `GetProductosCMS` and its `"GET.p-productos-cms"` route registration. The storefront
  now uses `p-productos-ecommerce` exclusively.

## Result
- First storefront mount triggers exactly one `getProductEcommerceData()` load.
- `ProductSearch`, `ProductsByCategory`, `ProductCards`, `getProductsByCategoryID`,
  `getCategoryByID` all await/read the same instance.
- No `p-productos-cms` fetch on the storefront; search no longer triggers a fresh fetch.
