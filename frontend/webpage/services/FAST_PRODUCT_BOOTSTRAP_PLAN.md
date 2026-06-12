# Fast product bootstrap — plan

## Goal
Render the storefront catalog as early as possible by doing the whole load **manually on the
main thread** — no service worker, no `GetHandler`, no message-channel round-trip. Start the
`products-c<companyID>.db` download the instant the storefront mounts, publish the product
list to the UI from memory immediately, and only then refine it with the server delta.

IndexedDB persistence + server deltas are still used (so the catalog stays cached and
incrementally synced in the existing format) — but they run **inline on the main thread**, not
inside the SW.

## Why the current path is slow
`getProductEcommerceData()` → `ProductEcommerceDataService.load()` → `fetchOnline()` →
`fetchCacheParsed` → `sw-cache.n()` posts to the **service worker** and awaits it. First load
pays SW install/activate (`waitForServiceWorkerEndpoint`, up to ~2s), a MessageChannel
round-trip, then the SW's IDB seed+delta — *all before the first product paints*.

## What we can reuse directly on the main thread (verified)
- `parsePsvResponse(text, schema)` (`libs/cache/psv-parse.ts:43`) — pure parser, `.db` text →
  `{ productos, marcas, categorias }`.
- `fetchDeltaCache(args)` (`libs/cache/delta-cache.fetch.ts:821`) — the full delta-cache core
  (seed + delta + IDB persistence in the standard row format). It uses **only `self.fetch`**
  (≡ `window.fetch` on the main thread) and Dexie IDB — no `clients`/`caches`/`skipWaiting`/
  `registration`. So it runs fine called directly, bypassing the SW message channel entirely.
- `readDeltaCacheSubObject(args)` (`delta-cache.fetch.ts:947`) — main-thread IDB read of one
  cached table for a route.
- `PRODUCT_ECOMMERCE_FILE_SCHEMA` + the image/`ss` normalization currently in `handler()`.

## Design

### One reactive entry point
There is exactly **one** public function: `getProductEcommerceData()`. It returns a memoized
singleton **reactive catalog object** (backed by `$state`) and resolves as soon as the first
list is in memory (then mutates its own `$state` in place when the delta lands). Every
component does:

```ts
const catalog = await getProductEcommerceData();
// then read reactive state / call accessors:
catalog.productos              // IProduct[]
catalog.categorias             // IProductCategory[]
catalog.getProduct(id)         // IProduct | undefined
catalog.getCategory(id)        // IProductCategory | undefined
catalog.getProductsByCategory(id) // IProduct[]
```

The object holds its data in a single `$state` (`productos`, `productosMap`, `categorias`,
`categoriasMap`, `productosByCategoryMap`) plus a `version` counter. Because it is the same
singleton object, any component reading `catalog.productos` / calling `catalog.getX()` inside a
`$derived`/template re-runs automatically when Phase 1 → Phase 2 mutates the state. The
accessors are **synchronous** (the caller already awaited the load) — no per-call `await`.

**Removed entirely:** `ensureProductosLoaded`, the standalone `productosServiceState`,
`getProductsByCategoryID`, `getCategoryByID`, `getLoadedProductEcommerceData`, and the
`ProductEcommerceDataService` / `GetHandler` subclass. Their behavior moves onto the catalog
object.

### File consolidation
`$state` requires a `.svelte.ts` module, so the loader + state + accessors all live in one
reactive service file (fold `productos-delta-service.ts` into `productos.svelte.ts`, or a new
`product-ecommerce.svelte.ts`). The `IProduct` / `IProductCategory` / image interfaces stay
exported from it for the type-only importers.

### localStorage watermark
Key `pe-fetch-c<companyID>` → unix seconds of the last successful load.
- absent or `now - ts >= 3600` → **cold** (IDB not reliably populated)
- present and `now - ts < 3600` → **warm** (IDB holds a fresh product table)

### Phase 1 — first paint (main thread, no IDB wait)
- **Cold:** `fetch(fileRoute)` once → keep the response **text in memory** → `parsePsvResponse`
  → normalize → publish `productos/marcas/categorias`. Resolve `getProductEcommerceData()`.
- **Warm:** `readDeltaCacheSubObject` for `productos`/`marcas`/`categorias` straight from IDB →
  normalize → publish → resolve. No network. (If the read comes back empty — IDB cleared but
  localStorage survived — fall back to the cold file fetch.)

### Phase 2 — delta refine (main thread, background, not awaited by the resolve)
Call `fetchDeltaCache(args)` **directly** (imported, run on the main thread — *not* via
`sw-cache.n()`):
- **Cold:** pass the `.db` text we already downloaded so the seed does **not** re-fetch the
  file (single fetch total — see the one-line change below). It seeds IDB from that text, then
  deltas.
- **Warm:** IDB routeRow already exists → delta only.
- When it returns, if the data changed (compare max `upd` watermark / record count), publish
  again. Then write `now` to the `pe-fetch-c<companyID>` localStorage key.

### One small change to `delta-cache.fetch.ts` (single-fetch)
`firstSyncFromSnapshotFile` (line 783) currently always does `self.fetch(args.fileRoute)`. Add
an optional `args.fileContent`: when present, parse that text instead of fetching. The cold
Phase-2 call passes the bytes Phase 1 already downloaded → the `.db` is fetched exactly once.

### Reactivity — plain Svelte 5 `$state`, no pub/sub
The list arrives in two waves (Phase 1, then Phase 2). Both waves **mutate the catalog
object's own `$state` in place** (`catalog.productos = ...`, rebuild the maps, `version++`).
Any component that read the singleton via `await getProductEcommerceData()` and references
`catalog.productos` / `catalog.getX()` in a `$derived` or template re-renders automatically.
No subscriber callbacks, no separate flag module — the `version` field on the object is the
"flag" a `$effect` can watch when a component needs an explicit recompute (e.g. the search
index rebuild in `ProductSearchLayer.svelte`).

### Files touched
1. **The reactive service** (consolidated `productos.svelte.ts`, absorbing
   `productos-delta-service.ts`) — defines the `$state` catalog singleton, the two-phase loader
   (localStorage watermark, main-thread file fetch + parse, warm IDB read via
   `readDeltaCacheSubObject`, Phase-2 `fetchDeltaCache` with `fileContent`, change-detection
   re-publish, `version++`), the accessors (`getProduct`, `getCategory`,
   `getProductsByCategory`), `PRODUCT_ECOMMERCE_FILE_SCHEMA`, normalization, and the exported
   interfaces. Exposes only `getProductEcommerceData()`.
2. **`core/product-search/product-search.ts`** — `bootstrap()` does
   `const catalog = await getProductEcommerceData()`, reads `catalog.productos/.marcas`, and
   `buildIndex()` is re-run by a `$effect` on `catalog.version` in `ProductSearchLayer.svelte`
   (expose a `rebuild()` the effect calls).
3. **`libs/cache/delta-cache.fetch.ts`** — optional `fileContent` shortcut in
   `firstSyncFromSnapshotFile` (+ `fileContent?: string` on the props type).
4. **Consumer migration** (drop `ensureProductosLoaded` / `productosServiceState` /
   `getProductsByCategoryID` / `getCategoryByID`):
   - `webpage/routes/+layout.svelte` — `onMount`: `void getProductEcommerceData()` (kick the load).
   - `webpage/ecommerce-components/ProductsByCategory.svelte` —
     `(await getProductEcommerceData()).getProductsByCategory(id)`.
   - `webpage/ecommerce-components/ecommerce-attributes/CategoryDescription.svelte` —
     `(await getProductEcommerceData()).getCategory(id)`.
   - `webpage/components/ProductCards.svelte` — read `catalog.productos` from the singleton.
   - `routes/webpage-builder/components/EditorTab.svelte` — `await getProductEcommerceData()`
     then read `catalog.categorias` / products.
   - Type-only importers (`ProductCard.svelte`, `ProductSearchLayer.svelte`,
     `store.svelte.ts`) — just repoint the `IProduct` import to the consolidated module.

No backend, route, schema, or `.db`-format changes. No new dependencies. The service worker is
not involved in the catalog load at all.

## Tradeoffs / notes
- `getProductEcommerceData()`'s contract changes from "fully synced" to "first list ready";
  the `$state`-driven re-render keeps late consumers correct.
- Running `fetchDeltaCache` on the main thread does its network + IDB work off the SW. This is
  intentional (that's the latency we're removing); it's already main-thread-safe.
- Warm read trusts the 1h window but self-heals (empty IDB → cold fetch).

## Result
- Mount → `.db` downloads (cold) or IDB read (warm) immediately on the main thread; first list
  paints with zero service-worker involvement.
- The delta runs inline afterward and re-publishes only on real changes.
- The `.db` file is fetched at most once per cold load; IDB + delta persistence stay in the
  existing format.
