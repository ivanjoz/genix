---
name: delta-cache-api
description: Builds an API that keeps one or more tables or pre-filtered subsets per endpoint in sync in the browser, each downloading only the records that changed since its last sync. Used for data a page shows on load, without the user running a query — any list with fixed filters.
version: 0.2.0
---

# Delta Cache API

The browser keeps a list in IndexedDB and sends a cursor `C.V` ("I hold up to C, my last sync
started at V"). The server answers what changed since C, the IDs that left the list, and a hash
of the unsettled slice. The first sync, or a sync behind by more than 100 changes, is a **replace**
(the whole list). You never handle cursors, hashes or metadata yourself: `Delta()` and `GetHandler`
do. The protocol is explained for people in `backend/genix-orm/db/cache/DELTA_CACHE.USER.md`.

| Need | Skill |
|---|---|
| Data a page shows on load: a **list with fixed filters** kept in sync | this one |
| Resolve **records by ID** (a label next to an ID) | `by-ids-cache-api` |
| A **report with user-chosen filters**, read as groups of records | `cache-groups-api` |
| Debug a sync mismatch, change the protocol | `delta-cache-index-internals` |

Scylla only implements this protocol (Dynamo has an older delta-log variant, the browser `idb` build none).

---

## 1. Schema

```go
// backend/production/types/product.go
type Product struct {
	db.TableStruct[ProductTable, Product]
	CompanyID int32
	ID        int32
	// ...
	Status   int8    `json:"ss" cb:"ss,34"`
	Updated  int32   `json:"upd" cb:"upd,35"`
	Version  int16   `json:"ver" cb:"ver,40"`
	// Metadata is set only on the metadata record a Delta() read appends.
	Metadata *string `json:"mdt" cb:"mdt,41"`
}

func (e ProductTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "products",
		Partition: e.CompanyID,
		Keys:      db.Cols(e.ID.Autoincrement(0)),
		// Delta() enumerates the values of each pinned Key: declare every Status value.
		FixedValues: []db.FixedValues{{Col: e.Status, Values: []int64{0, 1, 2}}},
		Indexes: []db.Index{
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
		},
	}
}
```

- **Record fields:** `Status int8 "ss"`, `Updated int32 "upd"`, `Version int16 "ver"`,
  `Metadata *string "mdt"` (record struct only). The table struct declares `Updated` and `Version`
  as `db.Col`. The ORM stamps `Updated`/`Version` on every write — never set them by hand.
- **Exactly one integer key** (the ID) and an **integer partition**. A composite-key table cannot
  have a `TypeDelta`: send it whole with `core.WithDeltaReplace(records)` (section 3).
- **Keys = prefixes.** Each combination of Key values is one slice the sync covers. The **last**
  Key takes the values passed to `Delta(cursor, values...)`; every other Key must be pinned by a
  query equality or a `FixedValues` list (fan-out capped at 8).
  - `Keys: db.Cols(e.Status)` → "active products" = `Delta(cursor, 1)`.
  - `Keys: db.Cols(t.Type, t.Status)` → `Type.Equals(…)` in the query + `Delta(cursor, int8(1))`
    (`crm/types/client_provider.go`).
  - `Keys: db.Cols(e.WarehouseID.Size(17), e.Status)` → width from `.Size(n)` or the
    `FixedValues` max; at most one Key may have neither (`logistics/types/product-stock.go`).
  - `{Type: db.TypeDelta}` with no Keys → the whole partition, `Delta(cursor)`.
- Several `TypeDelta` on one table are fine if their Keys differ; the query picks the one whose
  Keys all resolve.
- `Index.DeltaReplaceThreshold` overrides the 100-change replace threshold (no table sets it).
- **Never hard-delete** a TypeDelta record: set `Status = 0`. A soft-delete reaches the browser as a
  removed ID; a hard delete is never heard of.

Startup panics name the missing piece (no `Updated`/`Version`, non-integer key or partition, two
indexes with the same Keys, Keys that don't fit beside `updated`). See `create-database-tables`
for the general schema rules.

---

## 2. Handler — single table

```go
// backend/crm/client_provider.go
func GetClientProviders(req *core.HandlerArgs) core.HandlerResponse {
	requestedType := req.GetQueryInt("type")
	if requestedType != int32(types.ClientProviderTypeClient) && requestedType != int32(types.ClientProviderTypeProvider) {
		return req.MakeErr("Debe enviar type=1 (cliente) o type=2 (proveedor).")
	}

	clientProviders := []types.ClientProvider{}
	query := db.Query(&clientProviders)
	query.Select().
		CompanyID.Equals(req.User.CompanyID). // partition equality is required
		Type.Equals(int8(requestedType)).     // pins the first Key
		Delta(req.GetDeltaCursor(), int8(1))  // last Key (Status) = 1. Call Delta() LAST.

	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener clientes/proveedores:", err)
	}
	return req.MakeResponse(clientProviders)
}
```

- `req.GetDeltaCursor()` reads `?up=`; an absent or malformed value is the zero cursor = first sync.
  Never parse the cursor yourself.
- `Exec()` **appends one metadata record** (ID −1, `Metadata` set) to the slice. Return the slice
  as is. If you post-filter or map the records, keep that record and its `ID`, `Version`,
  `Metadata` (`production/products.go` keeps `product.Metadata == nil &&` in its filter).
- Inactive records never need a post-filter: a record that leaves the prefix (status change,
  soft delete) arrives as a removed ID.
- `Select()` with no columns is a full read. `Select(cols…)`/`Exclude(…)` is a projection: the ORM
  force-adds partition, ID, `updated`, `version` and pinned Keys, but **disables the S3 snapshot**,
  so every replace reads Scylla.

Register as usual: `"GET.client-provider": GetClientProviders` in the module's `ModuleHandlers`.

## 3. Handler — several tables in one response

One cursor **per response key**, read with `req.GetDeltaCursor("<key>")`. The key name must equal
the response field / map key — the frontend sends one query param per key.

```go
// backend/business/locations-warehouses.go
almacenes, sedes := []types.Warehouse{}, []types.Site{}
errGroup := errgroup.Group{}
errGroup.Go(func() error {
	query := db.Query(&almacenes)
	query.Select().CompanyID.Equals(req.User.CompanyID).Delta(req.GetDeltaCursor("Almacenes"), 1)
	return query.Exec()
})
errGroup.Go(func() error {
	query := db.Query(&sedes)
	query.Select().CompanyID.Equals(req.User.CompanyID).Delta(req.GetDeltaCursor("Sedes"), 1)
	return query.Exec()
})
if err := errGroup.Wait(); err != nil {
	return req.MakeErr(err)
}
return core.MakeResponse(req, &map[string]any{"Sedes": sedes, "Almacenes": almacenes})
```

- A struct response works the same (`logistics/product-stock-movement.go`, `GetProductsStockResult`).
- A table with **no** TypeDelta in the same response: `core.WithDeltaReplace(records)` — the browser
  replaces that list on every sync.
- **Projected DTOs** (`accounting/accounting_operations_api.go`): the DTO must carry `upd`, `ver`,
  `mdt` and the ID under the **same JSON names the table record uses**, or the browser's slice hash
  never matches and every sync near a write costs an extra `.r` round trip.

---

## 4. Frontend — `GetHandler` service

```ts
// frontend/services/production/products.svelte.ts
import { GetHandler } from '#libs/ui-runtime.svelte.ts'

export class ProductsService extends GetHandler<IProduct> {
  route = "products"
  routeByID = "p-products-ids"     // optional: by-IDs route used by syncIDs()
  useCache = { min: 1, ver: 13 }   // min = TTL in minutes; bump ver to wipe the route's cache
  inferRemoveFromStatus = true     // hide ss=0 in memory (removed IDs already evict)
  prependOnSave = true

  // products answers { records, records_IDsToRemove }; a bare-array route receives the array.
  handler(result: { records?: IProduct[] }): void {
    const products = result?.records || []
    this.records = []
    this.recordsMap = new Map()
    this.addSavedRecords(...products)
  }

  constructor(init = false) {
    super()
    if (init) this.fetch()   // IndexedDB first (instant), then the delta sync
  }
}
```

- `fetch()` paints from IndexedDB, then syncs. `fetchOnline()` forces the network.
- **Parameterized routes**: the route string (with its query) is the cache key, so
  `client-provider?type=1` and `?type=2` are independent lists. Build it in the constructor
  (`frontend/services/crm/client-provider.svelte.ts`).
- **Multi-table**: `handler(result)` receives `{ Sedes, Almacenes }`; cursors are stored per key
  automatically (`routes/business/branches-warehouses/branches-warehouses.svelte.ts`).
- After a save that affects other cached routes, pass `refreshRoutes: ["locations-warehouses"]`
  to `POST`.

## 5. Frontend — one-off `GET`

```ts
// frontend/routes/logistics/products-stock/stock-movement.ts
import { GET } from '#libs/ui-runtime.svelte.ts'

const response = await GET({
  route: `productos-stock?almacen-id=${almacenID}`,
  useCache: { min: 0.2, ver: 8 },
  keysIDs: { ProductStockDetail: ["ProductStockID", "LotID", "SerialNumber"] },
})
```

`GET` forwards `route, useCache, keyID, keysIDs, cacheMode, columnarIDField,
combineColumnarValuesOnFields` — not `conversion`, `fileRoute` or `partition`.

## 6. Record keys

| Option | Meaning |
|---|---|
| default | keyed by `record.ID` |
| `keyID: "ProductID"` | another single field |
| `keyID: ["A", "B"]` | composite: stored as `cmp:` + `JSON.stringify([A, B])` |
| `keysIDs: { Detail: [...] }` | per response key (multi-table) |

The **last** field of a composite key is the server ID used for removed IDs and the slice hash.
Columnar merge (parallel arrays merged element-wise): `columnarIDField` +
`combineColumnarValuesOnFields`.

---

## 7. Checklist

**Backend**
- [ ] Record has `ss`, `upd`, `ver`, `mdt`; table struct has `Updated` and `Version` cols
- [ ] One integer key, integer partition; `FixedValues` for every pinned Key without `.Size(n)`
- [ ] `{Type: db.TypeDelta, Keys: …}` whose last Key is what `Delta(cursor, values…)` receives
- [ ] Handler: partition equality, other Keys pinned, `Delta(req.GetDeltaCursor(…), …)` called last
- [ ] Return the slice untouched (or keep the ID −1 metadata record when filtering)
- [ ] Multi-table: `GetDeltaCursor("<Key>")` names = response keys
- [ ] Soft-delete only (`ss = 0`)

**Frontend**
- [ ] `GetHandler` subclass (or `GET`) with `useCache = { min, ver }`; bump `ver` after a shape change
- [ ] `keyID` / `keysIDs` when the record key isn't `ID`
- [ ] `refreshRoutes` on saves that touch other cached routes

## Real examples

| Pattern | Backend | Frontend |
|---|---|---|
| Single table, `Status` key | `backend/production/products.go` | `frontend/services/production/products.svelte.ts` |
| Two Keys (Type pinned by equality) | `backend/crm/client_provider.go` | `frontend/services/crm/client-provider.svelte.ts` |
| Multi-table map response | `backend/business/locations-warehouses.go` | `frontend/routes/business/branches-warehouses/branches-warehouses.svelte.ts` |
| Multi-table struct + `WithDeltaReplace` | `backend/logistics/product-stock-movement.go` | `frontend/routes/logistics/products-stock/stock-movement.ts` |
| Six tables, projected DTOs | `backend/accounting/accounting_operations_api.go` | `frontend/routes/accounting/accounts-consolidation/accounts-consolidation.svelte.ts` |
| Dynamic response keys | `backend/business/shared-lists.go` | — |
