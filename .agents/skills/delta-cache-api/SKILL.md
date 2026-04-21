---
name: delta-cache-api
description: Build backend Go handlers and frontend services for delta/incremental cache APIs — so the frontend only fetches records changed since the last sync, using GetHandler or GET.
version: 0.1.0
---

# Delta Cache API

The delta cache minimizes traffic by sending only records changed since the client's last sync. The frontend stores a watermark (max `Updated`) per response key and resends it on the next request. The backend returns records newer than the watermark, plus rows flipped to `Status=0` so the client can evict them.

## Core concepts

- **`Updated`** — integer Unix timestamp on every record. Drives the index key and the client watermark. JSON tag: `json:"upd,omitempty"`.
- **`Status`** — `int8`. Active (`>= 1`) or inactive (`0`). Initial fetch returns only active; delta fetch returns all statuses so the client can evict `ss=0` rows. JSON tag: `json:"ss,omitempty"`.
- **Watermark** — the client sends `?updated=<max>` (single table) or `?<ResponseKey>=<max>` (multi-table). Named after the response struct field.
- **First-sync vs delta** — branch on whether the watermark query param is present and `> 0`.

---

## 1. Backend — single-table handler

```go
func GetProductos(req *core.HandlerArgs) core.HandlerResponse {
    updated := core.Coalesce(req.GetQueryInt("upd"), req.GetQueryInt("updated"))

    records := []negocioTypes.Producto{}
    query := db.Query(&records)
    query.EmpresaID.Equals(req.Usuario.EmpresaID)

    if updated > 0 {
        query.Updated.GreaterThan(updated) // delta: include Status=0 rows
    } else {
        query.Status.GreaterEqual(1)        // initial: active only
    }

    if err := query.Exec(); err != nil {
        return req.MakeErr("error al obtener productos:", err)
    }
    return core.MakeResponse(req, &records)
}
```

Variations seen in the codebase:
- Parameterized route (`?type=X`) — apply the extra filter before the delta branch (see `backend/negocio/client_provider.go`).
- No status filter at all on initial — e.g. `system_parameters.go` returns every row.
- Sentinel rows — `sale_summary_status.go` stores metadata at `Fecha=-1` and reconstructs dates from the delta. Use only when the natural row count would be too high to stream.

---

## 2. Backend — multi-table handler

Return a struct with one slice per table. **Query-param name = struct field name** (the frontend reads the key from the response shape).

```go
type GetProductosStockResult struct {
    ProductStock       []ProductStockV2
    ProductStockDetail []ProductStockDetail
}

func GetProductosStock(req *core.HandlerArgs) core.HandlerResponse {
    productStockUpdated       := req.GetQueryInt("ProductStock")
    productStockDetailUpdated := req.GetQueryInt("ProductStockDetail")

    statuses := []int8{1}
    if productStockUpdated > 0 || productStockDetailUpdated > 0 {
        statuses = []int8{0, 1}
    }
    // ... errgroup: query each table per status bucket, ---
    name: delta-cache-api
    description: Design and implement backend Go handlers and frontend TypeScript services (or one-off `GET` calls) for delta/incremental cache APIs. Covers watermark semantics, `Updated`/`Status` field conventions, `TypeView` index strategies, composite `keysIDs`, columnar-merge, and the multi-table response shape — so the frontend only fetches records changed since the last sync.
    version: 0.1.0
    ---filter Updated.GreaterEqual(<updated>)
    return req.MakeResponse(result)
}
```

Query each status bucket separately so the `[Partition, Status, Updated]` view is used (see index strategies below). Real example: `backend/logistica/product-stock-movement.go:174`.

---

## 3. DB index strategies

`Updated` queries must hit a `TypeView` index. Several shapes are valid — pick the one that matches your filter:

| Query shape | Index keys |
|---|---|
| `Updated > X` only | `[Updated]` |
| `Status=S AND Updated > X` | `[Status, Updated]` or two separate views `[Status]` + `[Updated]` |
| `Filter=F AND Updated > X` | `[Filter, Updated]` (e.g. `[Type, Updated]`, `[ListaID, Updated]`) |
| `Partition AND Status AND Updated > X` | `[PartitionCol, Status, Updated]` |

```go
Indexes: []db.Index{
    {Type: db.TypeView, Keys: []db.Coln{e.Updated}, KeepPart: true},
    {Type: db.TypeView, Keys: []db.Coln{e.Type.Int32(), e.Updated.DecimalSize(8)}, KeepPart: true},
    {Type: db.TypeView, Keys: []db.Coln{e.WarehouseID, e.Status.DecimalSize(1), e.Updated.DecimalSize(10)}, KeepPart: true},
},
```

- `Updated` should be the **last** key when combined with filter columns.
- `DecimalSize(N)` packs numeric columns into fewer bytes — apply to `int8` status (`1`), mid-range ints (`8–9`), or Unix seconds (`10`). It's an optimization, not a requirement.
- `KeepPart: true` keeps partition scoping in the view.
- Two narrow views (`[Status]` + `[Updated]`) are fine if the access pattern never ANDs them (e.g. `productos.go`).

---

## 4. Frontend — `GetHandler` subclass (typical case)

Use `GetHandler` for list-style data owned by a screen/service. `fetch()` loads from IndexedDB first (instant), then delta-syncs from the server and merges.

```typescript
export class ProductosService extends GetHandler<IProducto> {
    route = "productos"
    routeByID = "p-productos-ids"           // optional: refetch specific IDs
    useCache = { min: 5, ver: 9 }           // TTL min; bump ver to invalidate
    inferRemoveFromStatus = true            // drop records where ss===0
    prependOnSave = true

    constructor(init: boolean = false) {
        super()
        if (init) this.fetch()
    }

    handler(result: IProducto[]): void {
        this.records = []; this.recordsMap = new Map()
        this.addSavedRecords(...result)
    }
}
```

Use in a view:

```svelte
<script lang="ts">
  const productos = new ProductosService(true) // true → fetch on mount
</script>
{#each productos.records as producto (producto.ID)}
  <div>{producto.Nombre}</div>
{/each}
```

### Parameterized routes — one cache collection per query-param set

Delta snapshots are keyed by **route + query string**, so `productos?type=1` and `productos?type=2` are two independent collections (distinct IndexedDB snapshots, independent watermarks). Build the route in the constructor to scope the instance:

```typescript
export class ClientProviderService extends GetHandler<IClientProvider> {
    route = ""
    routeByID = "client-provider-ids"
    useCache = { min: 5, ver: 1 }

    constructor(clientProviderType: number = 0, init: boolean = false) {
        super()
        this.route = `client-provider?type=${clientProviderType}`
        if (init) this.fetch()
    }

    handler(result: IClientProvider[]): void {
        this.records = (result || []).filter(r => (r.ss || 0) > 0)
        this.recordsMap = new Map(this.records.map(r => [r.ID, r]))
    }
}

// View:
const clients   = new ClientProviderService(ClientProviderType.CLIENT,   true)
const providers = new ClientProviderService(ClientProviderType.PROVIDER, true)
```

When a mutation can affect multiple scoped caches (e.g. saving records that straddle `type=1` and `type=2`), pass `refreshRoutes` to `POST` so both snapshots revalidate — see `clientes-proveedores.svelte.ts`.

---

## 5. Frontend — direct `GET` helper (one-off / multi-table)

When you don't need a persistent service class (e.g. the screen just needs the rows once, or the shape is a multi-table struct consumed immediately), call `GET` with cache options directly:

```typescript
// frontend/routes/logistica/products-stock/stock-movement.ts
const response = await GET({
    route: `productos-stock?almacen-id=${almacenID}`,
    useCache: { min: 0.2, ver: 8 },
    keysIDs: { ProductStockDetail: ["ProductStockID", "LotID", "SerialNumber"] },
})
// response.ProductStock, response.ProductStockDetail ...
```

The cache layer still stores per-response-key watermarks and sends them back as query params on the next request — same behavior as `GetHandler`.

---

## 6. `keyID` vs `keysIDs` — records without a single `ID` field

The cache needs a stable key per record to merge deltas. Configure it by shape:

| Field | Value | Meaning |
|---|---|---|
| `keyID: "ID"` | default | Each record keyed by `record.ID` |
| `keyID: "ProductID"` | single name | Keyed by another single field |
| `keyID: ["A", "B"]` | array | Composite key — cache stores `"cmp:" + JSON.stringify([record.A, record.B])` |
| `keysIDs: { Detail: ["StockID","LotID","SerialNumber"] }` | per response key | Multi-table: override the key-building per response array |

The composite form is a **list of field names to pack**, not a single field holding an array. `getRequiredRecordID` in `frontend/libs/cache/delta-cache.fetch.ts` reads each field from the record and JSON-stringifies the tuple.

Real examples:
- `keyID: "Fecha"` — `sale_orders_charts.svelte.ts`
- `keyID: "ProductID"` — `supply-management.svelte.ts`
- `keysIDs: { actionsScheduled: ["UnixMinutesFrame","ID"] }` — `cron-actions.svelte.ts`
- `keysIDs: { ProductStockDetail: ["ProductStockID","LotID","SerialNumber"] }` — `stock-movement.ts`

---

## 7. Columnar merge (parallel-array fields)

If a record holds parallel arrays (e.g. `ProductIDs: [1,2]`, `Quantity: [10,20]`) and deltas should merge element-wise rather than replace the whole record, set:

```typescript
columnarIDField = "ProductIDs"
combineColumnarValuesOnFields = ["Quantity", "TotalAmount", "TotalDebtAmount"]
```

On merge the cache aligns by `ProductIDs` position, updating matching indices and appending new ones. See `sale_orders_charts.svelte.ts`.

---

## 8. Configuration reference

| Property | Type | Purpose |
|---|---|---|
| `route` | `string` | API endpoint (may include fixed query params) |
| `routeByID` | `string` | Endpoint for fetching a specific set of IDs |
| `useCache` | `{ min, ver }` | TTL (minutes) and cache schema version |
| `keyID` | `string \| string[]` | Record key field(s); default `"ID"` |
| `keysIDs` | `Record<string, string \| string[]>` | Per-response-key overrides (multi-table) |
| `inferRemoveFromStatus` | `boolean` | Auto-evict records where `ss === 0` |
| `prependOnSave` | `boolean` | New saves at the top of `records` |
| `columnarIDField` | `string` | Parallel-array ID field for columnar merge |
| `combineColumnarValuesOnFields` | `string[]` | Array fields to merge element-wise |
| `headers` | `Record<string,string>` | Extra HTTP headers |

---

## 9. Checklist

**Backend**
- [ ] `Updated int32 \`json:"upd,omitempty"\`` and `Status int8 \`json:"ss,omitempty"\`` on the record struct
- [ ] `TypeView` index that matches the query shape (Updated last when composite)
- [ ] Handler reads `upd`/`updated` (single-table) or per-response-key param (multi-table)
- [ ] `updated > 0` → filter by `Updated`, include inactive rows; else filter by `Status >= 1`
- [ ] Multi-table: response struct field names match the expected query-param names

**Frontend**
- [ ] `useCache = { min, ver }` set; bump `ver` to force a full refresh after schema changes
- [ ] `keyID`/`keysIDs` set when the record has no `ID` or uses composite keys
- [ ] `inferRemoveFromStatus = true` if the list view should drop inactive rows
- [ ] For one-off fetches, use `GET({ route, useCache, keysIDs })` instead of a service class (multi-table responses work with either — `GetHandler` subclasses can consume multi-table shapes in `handler()`)

---

## Real examples

| Pattern | Backend | Frontend |
|---|---|---|
| Single table, `ID` key | `negocio/productos.go` | `productos.svelte.ts` |
| Parameterized filter | `negocio/client_provider.go` | `clientes-proveedores.svelte.ts` |
| Multi-table via `GetHandler` | `negocio/sedes-almacenes.go` | `sedes-almacenes.svelte.ts` |
| Multi-table via `GET` + composite key | `logistica/product-stock-movement.go` | `products-stock/stock-movement.ts` |
| Columnar merge | `comercial/sale_summary_status.go` | `sale_orders_charts.svelte.ts` |
| Sentinel-row delta | `comercial/sale_summary_status.go` | — |
