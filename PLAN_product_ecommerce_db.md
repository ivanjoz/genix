# Plan — Deprecate product indexer → `.db` file + ecommerce delta endpoint

## Goal
Replace the binary `.idx` product index with a plain pipe-separated **`products-c<companyID>.db`** file on Cloudflare (30-min cache) holding 3 delta tables (products / brands / categories). The frontend fetches the `.db` first to seed its local delta caches, then does an incremental delta fetch from a new ecommerce endpoint. Remove all the old indexer machinery.

---

## 1. `.db` file format
Plain UTF-8 text, pipe (`|`) separated columns, no header row. Sections delimited by a line starting with `>>>`. All `|` characters stripped from any text (product/brand/category names).

```
>>>productos
<ID>|<Name>|<categoriesIDs csv>|<BrandID>|<NameUpdated>|<Status>
...
>>>marcas
<ID>|<Name>|<Updated>|<Status>
...
>>>categorias
<ID>|<Name>|<Updated>|<Status>
...
```

- `categoriesIDs` = comma-separated (e.g. `3,7,12`), empty if none.
- Initial file holds **active rows only** (`Status >= 1`); `Status=0` evictions come through the live delta endpoint, never the file.
- Watermark columns: products use **`NameUpdated`**, brands/categories use **`Updated`**.

---

## 2. Backend

### 2.1 Fix `GlobalCache` table — `backend/core/cache.go`
Current draft has mismatches. Correct it to the paired-struct convention:
- `GlobalCache` embeds `db.TableStruct[GlobalCacheTable, GlobalCache]`.
- `GroupID` type unified to `int16` in **both** struct and table (matches the col + the `int16` group ids 1/2/3).
- `GlobalCacheTable` embeds `db.TableStruct[GlobalCacheTable, GlobalCache]`.
- `GetSchema()` keeps `Name: "cache_global"`, `Partition: e.GroupID`, `Keys: []db.Coln{e.ID}` (GroupID is partition → lets the cron scan all companies in a group; ID = companyID is the clustering key).
- Run `static-project-validation` afterwards.

### 2.2 Cache helpers — `backend/core/cache.go`
Create the two helpers the user specified:
```go
func SaveCacheGlobal(groupID int16, companyID int32, content []byte, updated int32) error
func GetCacheGlobal(groupID int16, companyIDs ...int32) ([]GlobalCache, error)   // empty companyIDs => whole group (partition scan by GroupID)
```
`content` is stored but unused for now (reserved for future use, per user). Only `Updated` matters here.

### 2.3 Dirty tracking on write — `backend/business/productos.go`
In `PostProducts`, after the `db.Merge`, when a product is new or its `NameUpdated` advanced, upsert the company's row in **group 1** (productos):
`core.SaveCacheGlobal(1, companyID, nil, maxNameUpdated)`.
Likewise the brand/category save handler (`PostListasCompartidas`) upserts **group 2** (marcas) / **group 3** (categorias) keyed by the company's max `Updated`.
(Groups: `1`=productos, `2`=marcas, `3`=categorias.)

### 2.4 New file `backend/business/product-ecommerce.go`
Rewrite the file to own the whole ecommerce path:

**a) `buildProductsDbFile(companyID)`** — query active productos (`ID, Name, CategoryIDs, BrandID, NameUpdated, Status`) + active marcas + categorias (shared lists, `ListID IN (1,2)`), render the 3-section pipe file (stripping `|` from names, raw names — no cleaning), upload to R2 at `live/products-c<companyID>.db` with `Cache-Control: max-age=1800`. Returns the 3 max watermarks.

**b) `GetProductsEcommerce(req)`** — multi-table delta endpoint (route `p-productos-ecommerce`). Reads 3 watermark query params `productos`, `marcas`, `categorias`. Returns `{productos, marcas, categorias}`:
- productos: filter `NameUpdated > productos` (delta, all statuses) else `Status>=1` (first sync, though first sync normally comes from the file). Select `Name, CategoryIDs, BrandID, NameUpdated→upd, Status`.
- marcas/categorias: shared-list rows filtered by `Updated`, including `Status=0` for eviction.
- **Lazy rebuild:** at the top, compare group-1/2/3 `GetCacheGlobal` watermarks vs the last-built watermarks (stored in `core.Cache` key `products_db_built`, per company, packed 3 values). If any source watermark advanced → `buildProductsDbFile` + restamp the built cache. (Same lazy pattern the old `GetProductsIndex` used.)

### 2.5 Cron — single global 30-min tick
- New `backend/business/product-ecommerce-cron.go`:
  - `RegisterActionHandler(<newID>, "Rebuild Products DB", RebuildProductsDbHandler)` in `business` `init()` (currently no `init()` in `business/main.go` — add one). Pick an unused action id (sales uses `2`; use e.g. `3`).
  - `RebuildProductsDbHandler`: scan group 1 (`GetCacheGlobal(1)`) → for each company compare source vs `products_db_built` watermark → rebuild `.db` when advanced. Then **reschedule itself** `ScheduleCronAction(..., 30)` for the next 30-min frame.
- Seed the first schedule once at startup (near `core.StartCronWatcher()` in `backend/main.go`), owned by a system company id.

### 2.6 Register routes & delete old code — `backend/business/main.go`
- Add `"GET.p-productos-ecommerce": GetProductsEcommerce`.
- **Delete** routes `GET.p-productos-index` and `GET.p-productos-index-delta`.
- **Delete** `backend/business/productos-indexer.go` entirely (`GetProductsIndex`, `GetProductsIndexDelta`, `BuildProductosSearchIndex*`, source-cache helpers).
- **Delete** the whole `backend/libs/index_builder` package (verify no other importers first).

---

## 3. Frontend

### 3.1 Snapshot bootstrap pushed into the delta cache internals
Rather than a bespoke in-memory service, the CDN-snapshot bootstrap is a generic capability of the
delta cache, so persistence (IndexedDB rows + per-table watermarks) comes for free and any future
service can opt in:
- **`frontend/libs/cache/psv-parse.ts`** — `parsePsvResponse(content, schema)` parses a headerless,
  `|`-separated, `>>>`-section `.db` file into the multi-table `{ [section]: records[] }` shape the
  cache already consumes. Schema lists columns positionally as `"Field:TYPE"`; tokens: `T` text,
  `N` number, `O` JSON object, `A`/`AT` string array, `AN` number array, `AO` object array. Field
  names are literal record fields, so naming the watermark/status columns `upd`/`ss` wires them into
  the cache's `extractUpdated` / eviction logic directly.
- **`serviceHttpProps` + `GetHandler`** gain `fileRoute` (absolute CDN URL) and `fileSchema`.
- **`fetchDeltaCache`** (`delta-cache.fetch.ts`): when a route has no cache yet and `fileRoute`/
  `fileSchema` are set, `firstSyncFromSnapshotFile` fetches the file → `parsePsvResponse` →
  `saveInitialSnapshot` (persists rows + watermarks) → then immediately runs one server delta from
  those watermarks (closes the ≤30-min CDN staleness gap). File miss/parse error → falls back to a
  normal full server fetch (`updated=0`). Subsequent loads read IndexedDB and only delta-sync.

### 3.1b `productos-delta-service.ts`
Now a thin `GetHandler` subclass: `route = "p-productos-ecommerce"`, `useCache = { min: 30, ver: 1 }`,
`keysIDs = { productos:"ID", marcas:"ID", categorias:"ID" }`, `fileSchema` for the 3 sections, and
`fileRoute = Env.makeCDNRoute("live", \`products-c${cid}.db\`)`. `handler()` splits the merged
multi-table content into `productos`/`marcas`/`categorias` (active rows only); `load()` = `fetchOnline()`.

### 3.2 `product-search.ts`
- Delete `tryLoadBinaryIndex()`, `loadFromDecodedPayload()`, `decoder.ts`, and the `idx_plus_delta` branch.
- `bootstrap()` becomes: seed dictionary from fixed Spanish syllables (`rebuildFromDeltaOnly` path) + apply products/brands/categories from the new service. `source` is always delta/file-based now.
- Brand & category **names** now come from the marcas/categorias delta tables (not the old taxonomy payload in the `.idx`).

### 3.3 Cleanup
- Remove `frontend/core/product-search/decoder.ts` and any encoder bits used only by the binary index.

---

## 4. Open confirmations before coding
1. Action id `3` for the cron handler OK? System company id to own the global tick (e.g. `1`)?
2. Last-built watermark store: `core.Cache` row key `products_db_built` per company (packing the 3 ints in `Content`) — OK, vs adding groups 4/5/6 in `cache_global`?
3. CDN path `live/products-c<cid>.db` (reuse existing `live/` prefix the `.idx` used) — OK?

---

## 5. Step order
1. Fix `GlobalCache` + add `SaveCacheGlobal`/`GetCacheGlobal` → validate tables.
2. `buildProductsDbFile` + `GetProductsEcommerce` (lazy rebuild) + register route.
3. Add `Cache-Control` support to `SaveFileToR2`/`SaveFileArgs`.
4. Dirty-tracking writes in `PostProducts` / brand-category save.
5. Cron handler + startup seed.
6. Delete `productos-indexer.go`, old routes, `index_builder`.
7. Frontend: `parsePsvResponse` + `fileRoute`/`fileSchema` in cache internals & `GetHandler`; rewrite `productos-delta-service.ts` as a `GetHandler` subclass; `product-search.ts` simplification + delete `decoder.ts`.
8. Build backend, typecheck frontend, manual verify.
