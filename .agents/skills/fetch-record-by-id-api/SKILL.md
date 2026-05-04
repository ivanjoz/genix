---
name: fetch-record-by-id-api
description: Backend `*-ids` handlers + frontend callers that resolve records by ID through a 3-layer cache (memory → IndexedDB → server delta), using `db.QueryCachedIDs` and `cache-by-ids.svelte.ts`. Use for "give me records [12, 87, 412]". For list/watermark sync, use `delta-cache-api`.
version: 0.1.0
---

# Fetch Record By ID API

Resolves specific records by `ID` without re-downloading unchanged ones. Backend keeps a per-record `ccv` (`uint8`); frontend stores it next to each cached row and sends it back so the server returns only changed/new rows.

| Need | Skill |
|---|---|
| "Full list, plus what changed since X" | `delta-cache-api` (uses `Updated` watermark) |
| "Records `[12, 87, 412]` — skip unchanged" | this skill (`ccv` per ID) |

A table can expose both endpoints independently.

---

## 1. Backend schema (versioned)

ORM enforces these at startup (`backend/db/cache_version.go:67`):

```go
func (t ClientProviderTable) GetSchema() db.TableSchema {
    return db.TableSchema{
        Name:             "client_provider",
        Partition:        t.EmpresaID,                       // int32 or int64
        SaveCacheVersion: true,
        Keys:             []db.Coln{t.ID.Autoincrement(0)},  // exactly ONE, int16/int32/int64
    }
}
```

Record struct must expose a `uint8 ccv` field (only on the record, not the Table):

```go
type ClientProvider struct {
    db.TableStruct[ClientProviderTable, ClientProvider]
    EmpresaID    int32 `json:",omitempty"`
    ID           int32 `json:",omitempty"`
    Status       int8  `json:"ss,omitempty"`
    Updated      int32 `json:"upd,omitempty"`
    CacheVersion uint8 `json:"ccv,omitempty"`  // required: name "CacheVersion" OR json tag "ccv"
}
```

## 2. Backend handler (versioned)

```go
func GetClientProvidersByIDs(req *core.HandlerArgs) core.HandlerResponse {
    cachedIDs := req.ExtractCacheVersionValues()
    if len(cachedIDs) == 0 {
        return req.MakeErr("No se enviaron ids a buscar.")
    }
    clientProviders := []s.ClientProvider{}
    if err := db.QueryCachedIDs(&clientProviders, cachedIDs); err != nil {
        return req.MakeErr("Error al obtener clientes/proveedores.", err)
    }
    return req.MakeResponse(clientProviders)
}
```

Register in package `main.go`: `"GET.client-provider-ids": GetClientProvidersByIDs`.

`QueryCachedIDs` reads `cache_version` first, returns only IDs whose client `ccv` mismatches (plus IDs without local cache). Unchanged cached rows → omitted from response.

## 3. Backend handler (static / immutable-per-ID)

For catalog-like rows that never change after creation. Schema does **not** set `SaveCacheVersion`. Example: `backend/logistica/product-stock-movement.go:74`.

```go
func GetProductStockLotsByIDs(req *core.HandlerArgs) core.HandlerResponse {
    lotIDRecords := req.ExtractCacheVersionValues()
    if len(lotIDRecords) == 0 {
        return req.MakeErr("No se enviaron ids de lotes.")
    }
    // ExtractCacheVersionValues returns int64; convert to the table's key type.
    lotIDs := core.Map(lotIDRecords, func(e db.IDCacheVersion) int32 { return int32(e.ID) })
    lots := []logisticaTypes.ProductStockLot{}
    query := db.Query(&lots)
    if err := query.CompanyID.Equals(req.Usuario.EmpresaID).ID.In(lotIDs...).Exec(); err != nil {
        return req.MakeErr("Error al obtener los lotes.", err)
    }
    return req.MakeResponse(req, &lots)
}
```

## 4. Frontend record contract

Versioned consumers extend `IMinimalRecord` (`frontend/libs/cache/cache-by-ids.svelte.ts:15`):

```ts
export interface IMinimalRecord {
    ID: number      // unique
    ccv?: number    // server cache-version 0..255
    ss: number      // 1 active, 0 tombstone
    _fch?: number   // internal: last-fetched seconds
    upd: number     // only required if you call getRecordByIDUpdated
}
```

Static consumers only need `{ ID: number }`. `ss === 0` is held in cache (so it's never re-requested) but filtered out of results.

## 5. Frontend entry points

All in `frontend/libs/cache/cache-by-ids.svelte.ts`. `apiRoute` = backend route string (e.g. `"client-provider-ids"`).

**`getRecordsByID(apiRoute, ids) → Map<ID, T>`** — explicit batch, in-flight dedup.

**`getRecordByID(apiRoute, id, options?) → T | undefined`** — single-ID with 80 ms buffer window: many components asking different IDs in the same tick → one server request. Use inside per-record components.

```ts
await getRecordByID<IProducto>('p-productos-ids', 42, { cachedRecord: knownRow })
// or { cachedRecord: false } to force "no local copy"
```

**`getRecordByIDUpdated(apiRoute, id, updated) → T | undefined`** — local-first when an authoritative `upd` is known (e.g. from a parent list, websocket, or save response). If `localRecord.upd >= updated`, returns local without network. Otherwise fetches by ID directly (no `cc-ver` revalidation). **`updated` is required** — pass a real authoritative value or use `getRecordByID` instead.

**`getRecordWithCache(apiRoute, id) → IRecordRef<T>`** — Svelte `$state` ref: paints local immediately, revalidates in background. Designed for label components (`frontend/ui-components/micro/RecordByIDText.svelte`).

```svelte
<script lang="ts">
  const ref = getRecordWithCache<IProducto>('p-productos-ids', recordID)
</script>
{#if ref.loading}…{:else}{ref.record?.Nombre ?? '-'}{/if}
```

**`<RecordByIDText apiRoute recordID placeholder?/>`** — drop-in `<span>` wrapper around `getRecordWithCache`. Picks the first non-empty of `Usuario` / `Nombre` / `Name` / `Nombres + Apellidos` from the resolved record. Use this instead of writing your own `getRecordWithCache` plumbing whenever you just need to render a record's display label.

```svelte
<RecordByIDText apiRoute="usuarios-ids" recordID={row.CreatedBy} placeholder="-" />
```

Inside a `VTable` cell, mount it via the `cellRenderer` snippet and gate on `column.id` (the snippet only fires for columns that declare an `id` — columns without `id` keep the default `getValue`/`render` path):

```svelte
<VTable {data} ...>
  {#snippet cellRenderer(record, column)}
    {#if column.id === 'usuario'}
      <RecordByIDText apiRoute="usuarios-ids" recordID={record.CreatedBy} placeholder="" />
    {/if}
  {/snippet}
</VTable>
```

See `frontend/routes/finanzas/cajas/+page.svelte` for a live example.

**`getStaticRecordsByID(apiRoute, ids) → Map<ID, T>`** — for static endpoints. Cache-first by ID, server only asked for IDs missing in both memory and IDB. No staleness, no `ccv`, no tombstones.

**`clearCacheByIDs()`** — drops memory tables, in-flight buffers, and the IndexedDB database. Call on logout/company switch.

**`GetHandler.routeByID`** — when a table already has a delta-cache `GetHandler` service, set `routeByID` to the by-IDs endpoint and the service will use it to refetch specific records (missing/deleted/explicitly invalidated) without a full delta sync. Convenient when the same table needs both list-style watermark sync and per-ID lookups:

```ts
export class ProductosService extends GetHandler<IProducto> {
  route = "productos"
  routeByID = "p-productos-ids"   // backend `*-ids` handler — same one used by getRecordByID
  useCache = { min: 5, ver: 9 }
  inferRemoveFromStatus = true
}
```

See `frontend/routes/negocio/productos/productos.svelte.ts:90`.

## 6. Wire protocol

```
GET <apiRoute>?ids=<concat>&cc-ids=<concat>&cc-ver=<concat>&cmp=<empresaID>
```

- `ids` = no local cache. `cc-ids` + `cc-ver` = positional pairs for cached rows.
- All three use `concatenateInts` (base64-url chunks, `u8`/`u16`/`u32` buckets joined by `.`). Don't compose by hand.
- `cc-ver` must be `0..255`. Encoder throws otherwise (corrupt cache).
- Response shape: `T[]` or `{ records: T[] }`.

## 7. Cache lifecycle

```
caller → memory map → IDB (compound key [storeName+ID]) → server (only if missing>0 OR stale>0)
```

- `CACHE_TIME = 5s`. Fresh memory hits never round-trip.
- Stale hits go through buffered batch.
- Returned `ss=0` → written to cache, filtered from results.
- Cached rows the server *didn't* return → treated as validated unchanged, `_fch` bumped to now.
- One shared IDB store per company+env, partitioned by `apiRoute`.

## 8. Checklist

**Backend versioned**
- [ ] `SaveCacheVersion: true`, single int key, int partition
- [ ] Record has `CacheVersion uint8 \`json:"ccv,omitempty"\``
- [ ] Handler: `ExtractCacheVersionValues()` + `QueryCachedIDs(&records, cachedIDs)`
- [ ] Empty-payload guard
- [ ] Route: `GET.<name>-ids`

**Backend static**
- [ ] No `SaveCacheVersion`
- [ ] Convert `IDCacheVersion.ID` (int64) to the table's key type before `In(...)`
- [ ] Route: `GET.<name>-by-ids` (convention)

**Frontend**
- [ ] Record extends `IMinimalRecord` (versioned) or `{ ID: number }` (static)
- [ ] Right entry point: explicit batch → `getRecordsByID`; per-record component → `getRecordByID`; reactive label → `getRecordWithCache`; immutable → `getStaticRecordsByID`; known authoritative `upd` → `getRecordByIDUpdated`
- [ ] `apiRoute` matches backend route string

## 9. Failure modes

| Symptom | Cause |
|---|---|
| Server keeps returning the same rows | Response struct missing `CacheVersion uint8` / `ccv` tag |
| `Error: invalid cc-ver for <id>` | Corrupt local `ccv > 255` → `clearCacheByIDs()` |
| ORM panic on startup | Schema violates single-key / key-type / partition-type rule (`backend/db/cache_version.go:67`) |
| First response is full table | Cold start: no `cc-ids` to send → all requested `ids` fetched |

## Real examples

| Pattern | Backend | Frontend |
|---|---|---|
| Versioned by ID | `backend/negocio/productos.go:47` | `frontend/demo/records-by-id.svelte.ts` |
| Versioned + parameterized list sibling | `backend/negocio/client_provider.go:48` | `frontend/routes/negocio/clientes/clientes-proveedores.svelte.ts` |
| Static by ID | `backend/logistica/product-stock-movement.go:74` | `frontend/routes/logistica/products-stock/ProductStockMovement.svelte:95` |
| Reactive label | — | `frontend/ui-components/micro/RecordByIDText.svelte` |
