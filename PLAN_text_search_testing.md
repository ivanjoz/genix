# Plan — Text Search testing (ORM read path + `GET.product-text-search` + Testing page)

## Goal
Expose the already-built GenixSearch **write** index for reads. Add ORM functions
`db.SearchTextIDs` (ids + weights only, no ScyllaDB fetch) and `db.SearchText`
(hydrated records), a `GET.product-text-search` API, and a new **TESTING** page
under the SYSTEM menu with one `OptionsStrip` sub-view **Text Indexes**
(`TestTextIndexes`) to query the `products` table by name.

## Current state (verified)
- `ProductTable.GetSchema()` declares `TextSearchColumn: e.Name` → products are
  already indexed into GenixSearch on every write (`backend/db/text_search_index.go`,
  `backend/db/text_search/ingest.go`).
- Index layout: collection = table name (`products`), bucket = `p{companyID}_s{group}`
  where group 0 = `status==0`, group 1 = everything else
  (`text_search/text.go: PickStatusGroup`, `CollectionAndBucket`).
- **Read path is a stub**: `text_search.Search(...)` returns `ErrNotImplemented`
  (`text_search/search.go:20`). The connection pool (`search` mode), `execPending`
  (PENDING→EVENT), `buildQueryLine`, and EVENT payload parsing all already exist.
- GenixSearch returns each match as a `<recordID>|<score>` token; `decodeIDs`
  currently drops the score. **Weights are available** — we just keep them.
- `text_search.Configure(...)` is already called at startup (`backend/main.go:160`,
  `backend/exec/init.go:32`).
- Handlers register in a `core.AppRouterType` map; product handlers live in
  `backend/business/productos.go`, registered in `backend/business/main.go`.
  `CompanyID` comes from `req.User.CompanyID`. GET routes are authorized by default
  (no `access_list.yml` change needed).
- SYSTEM menu lives in `frontend/core/modules.ts` (id 9); routes under
  `frontend/routes/system/...`.

---

## Backend

### 1. `text_search` package — implement the read path
File: `backend/db/text_search/search.go`

- Add a weighted result type and decoder:
  ```go
  type Match struct { ID int32; Weight float32 } // weight = GenixSearch score, 0 if absent
  func decodeMatches(payload []string) ([]Match, error) // split "<key>|<score>"
  ```
- Implement `Search` (and a weighted variant) for real:
  ```go
  func SearchMatches(ctx, table string, partition int32, statusGroup int8,
      query string, limit, offset int) ([]Match, error)
  ```
  Body: `initPools()` → `searchMgr.acquire(ctx)` → `execPending(buildQueryLine(...))`
  → `decodeMatches(res.payload)` → `searchMgr.release(conn)` (discard on broken conn).
  Keep `Search` returning `SearchResult{IDs}` as a thin wrapper over `SearchMatches`
  (or delete it if unused — pre-alpha, no back-compat). Remove `ErrNotImplemented`
  once nothing references it.

### 2. ORM public API — `db.SearchTextIDs` and `db.SearchText`
New file: `backend/db/text_search_query.go`

- Public weight type:
  ```go
  type IDWeight struct { ID int32 `json:"id"`; Weight float32 `json:"w"` }
  ```
- `SearchTextIDs` — ids + weights only, **no ScyllaDB round trip**:
  ```go
  func SearchTextIDs[T TableBaseInterface[E,T], E TableSchemaInterface[E]](
      partition int32, query string, statusGroup int8, limit int) ([]IDWeight, error)
  ```
  Resolve the compiled table via `MakeScyllaTable[T,E]()` to read `.name` and assert
  `.textSearchIndex != nil` (panic-free error if the table has no `TextSearchColumn`).
  Normalize the query with `text_search.NormalizeSearchText`, call
  `text_search.SearchMatches`, map `Match`→`IDWeight`.
- `SearchText` — hydrate full records by the returned ids, preserving weight order:
  ```go
  func SearchText[T TableBaseInterface[E,T], E TableSchemaInterface[E]](
      refSlice *[]T, partition int32, query string, statusGroup int8, limit int) ([]IDWeight, error)
  ```
  Calls `SearchTextIDs`, then fetches the records by ID through the existing
  by-ID query path (same mechanism `QueryCachedIDs`/the products `*-ids` handler use),
  reorders `*refSlice` to match descending weight, and returns the weights.

> **Open question A (status group):** default to group 1 (active, `status!=0`)?
> Products page only cares about active items. Alternative: query both groups and
> merge. Proposed default: **group 1**, with the param exposed for the test page.

### 3. API handler `GET.product-text-search`
File: `backend/business/productos.go` (register in `backend/business/main.go`)

```go
func GetProductTextSearch(req *core.HandlerArgs) core.HandlerResponse {
    q := req.GetQuery("q")
    if len(q) < 2 { return req.MakeErr("La búsqueda debe tener al menos 2 caracteres") }
    limit := int(req.GetQueryInt("limit")); if limit <= 0 { limit = 50 }
    // ids + weights only — no record bodies, as requested.
    matches, err := db.SearchTextIDs[businessTypes.ProductTable, businessTypes.ProductTable](
        req.User.CompanyID, q, 1 /*active*/, limit)
    if err != nil { return req.MakeErr("Error en la búsqueda de texto:", err) }
    return core.MakeResponse(req, &matches)
}
```
Register: `"GET.product-text-search": GetProductTextSearch`.
Response shape: `[{ "id": 123, "w": 4.5 }, ...]`.

> Note generic params: confirm the exact `TableBaseInterface`/`TableSchemaInterface`
> type args for `Product`/`ProductTable` against existing `db.Query(&products)` call
> sites in `productos.go` (will mirror those exactly).

---

## Frontend

### 4. Menu entry
File: `frontend/core/modules.ts` — append to SYSTEM (`id: 9`) options:
```ts
{ name: "Testing", route: "/system/testing", icon: "icon-shield", onlySaaS: true },
```

### 5. Page shell + OptionsStrip
File: `frontend/routes/system/testing/+page.svelte`
- `<Page title="Testing">` with one `OptionsStrip` (`useMobileGrid`) holding a single
  option `[1, "Text Indexes"]`. Local `let view = $state(1)`.
- `{#if view === 1}<TestTextIndexes />{/if}` — sibling component file.

### 6. `TestTextIndexes` sub-page
File: `frontend/routes/system/testing/TestTextIndexes.svelte`
- `Input`/`FilterInput` for the query string + a `Button` "Buscar" (debounced or
  on click) that calls the API.
- Service call to `GET.product-text-search?q=...&limit=...` returning
  `{ id, w }[]` (use the project's request helper / a tiny report service in
  `testing.svelte.ts`; **not** a delta-cache service since this is ad-hoc).
- `VTable` with columns: **Product ID**, **Weight** — matching the user's request
  to see only ids + weights (no record fetch). Show result count + elapsed ms.

> **Open question B:** the user said "only the product ids and the weight" — so the
> test page table shows ID + weight only (no product name lookup). Confirm, or should
> the page also resolve names for readability (would use the existing by-ids cache)?

---

## Out of scope / not doing
- No `access_list.yml` change (GET authorized by default).
- No new ScyllaDB table or schema migration.
- No backfill tooling (existing records already indexed on write).

## Verification
- `go build ./...` in `backend/`.
- Run app, open SYSTEM → Testing → Text Indexes, query a known product name,
  confirm ids+weights returned and ordered by weight.

## Questions before coding
- **A.** Status group default = active only (group 1)? (proposed)
- **B.** Test table shows ID + weight only, or also resolve product names?
- **C.** OK to delete/replace the stubbed `text_search.Search` + `ErrNotImplemented`
  rather than keep them (pre-alpha, no back-compat)?
