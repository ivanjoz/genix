---
name: by-ids-cache-api
description: Resolves records by ID from browser-cached dictionaries, so the frontend can join a report of IDs against them. Use when frontend needs a record by ID, building a report, or list or screen that references other records by ID.
version: 0.3.0
---

# By-IDs Cache API

**The pattern: joins in the frontend.** The backend answers a report with **IDs only** (the fact
table: `ClientID`, `ProductIDs`, `CreatedBy`…) instead of joining names, prices or labels into it.
The frontend resolves those IDs against **dictionaries** (users, products, clients) through this
cache — a star model. Each dictionary record travels once and stays in the browser; later reports
only re-download the records that changed.

```ts
// Fact rows from the report endpoint carry IDs only.
const saleOrders = await fetchSalesReport(...)
// Dimension lookups: one cached request per dictionary.
const clientIDs = [...new Set(saleOrders.map(e => e.ClientID))]
const productIDs = [...new Set(saleOrders.flatMap(e => e.ProductIDs))]
const [clients, products] = await Promise.all([
  clientIDs.length ? getRecordsByID<IClient>('client-provider-ids', clientIDs) : new Map(),
  productIDs.length ? getRecordsByID<IProduct>('p-products-ids', productIDs) : new Map(),
])
// Join in the frontend.
const rows = saleOrders.map(e => ({ ...e, clientName: clients.get(e.ClientID)?.Name }))
```

So: **keep report payloads to IDs and facts; never denormalize dictionary fields into them.**
Real example: `frontend/routes/accounting/accounts-consolidation/accounts-consolidation.svelte.ts`.

How it stays cheap: the browser sends the IDs with the version it holds, and the server returns
only the records whose version moved (versions are per slot, `ID % 256`). Details in
`by-ids-cache-internals`.

| Need | Skill |
|---|---|
| Join the IDs of a report or list against dictionaries; a label next to an ID | this one |
| A whole list kept in sync | `delta-cache-api` |
| Debug a stale record, change the protocol | `by-ids-cache-internals` |

A table can expose both a delta list and a by-IDs route (products, clients).

---

## 1. Schema

```go
// backend/core/types/users.go
func (t UserTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:       "users",
		Partition:  t.CompanyID,               // integer partition (required on Scylla)
		CacheByIDs: true,                      // the opt-in
		Keys:       db.Cols(t.ID.Autoincrement(0)), // exactly ONE integer key
	}
}
```

- Record struct: `Updated int32 "upd"` and `Status int8 "ss"`; the table struct declares
  `Updated` as `db.Col`. On this route `upd` is **the slot's version**, overwritten by the server —
  never use it as the record's write time.
- The ORM raises the slot on every write (Insert/Update/InsertUpdate). Nothing to do in your code.
- Deploy creates `<keyspace>.cache_by_ids_slots` for the first `CacheByIDs` table.
- Startup panics: missing `Updated` column, not exactly one integer key, non-integer partition.
  Calling `QueryCachedIDs` on a table without the flag returns an error.
- Soft-delete (`ss = 0`): the browser keeps it as a **tombstone** and never asks again.

## 2. Handler — versioned

```go
// backend/security/usuarios.go
func GetUsuariosByIDs(req *core.HandlerArgs) core.HandlerResponse {
	// Decodes ?cc=, refuses an empty request, puts the company in as the partition.
	cachedIDsRequest, err := req.GetCachedIDsRequest(req.User.CompanyID)
	if err != nil {
		return req.MakeErr(err)
	}
	// Only the records whose slot moved past the tick the client holds come back.
	partitions, err := db.QueryCachedIDs[coreTypes.User](cachedIDsRequest)
	if err != nil {
		return req.MakeErr("Error al obtener los usuarios.", err)
	}
	// Puts partition 0 back: the company never leaves the server.
	return req.MakeResponse(core.HideCachedPartitions(partitions))
}
```

Register: `"GET.users-ids": GetUsuariosByIDs` in the module's `ModuleHandlers`. Then regenerate
route IDs: `cd scripts && go run . generate_route_ids`.

- **Public route** (no token, name starts with `p-`): take the company from the query —
  `req.GetCachedIDsRequest(core.Coalesce(req.GetQueryInt("cid"), req.GetQueryInt("company-id")))`
  (`production/products.go`, `p-products-ids`).
- Naming in the repo is mixed (`users-ids`, `sale-order-by-ids`); follow the module's neighbours.
- The response is `[]db.CachedPartition[T]` (`{p, r}`); the frontend unpacks it.
- **Partition 0 only.** `GetCachedIDsRequest` refuses any other partition, so the frontend's
  `{[partition]: ids}` form does not work against these handlers. A table partitioned by something
  other than the company (stories by project) sends plain IDs in the company partition.

## 3. Handler — static (records that never change once created)

No `CacheByIDs`, no versions: the server returns every ID it is asked for, and the browser never
revalidates. Use only for truly immutable rows (lots, catalog rows).

```go
// backend/logistics/product-stock-movement.go
func GetProductStockLotsByIDs(req *core.HandlerArgs) core.HandlerResponse {
	requestedIDs, err := req.GetStaticRequestIDs() // []int64; refuses empty / partition ≠ 0
	if err != nil {
		return req.MakeErr(err)
	}
	lotIDs := core.Map(requestedIDs, func(id int64) int32 { return int32(id) })
	lots := []types.ProductStockLot{}
	query := db.Query(&lots)
	// The tenant filter is yours here: GetStaticRequestIDs takes no company.
	if err := query.CompanyID.Equals(req.User.CompanyID).ID.In(lotIDs...).Exec(); err != nil {
		return req.MakeErr("Error al obtener los lotes.", err)
	}
	return req.MakeResponse(core.StaticCachedPartition(lots))
}
```

---

## 4. Frontend

All from `@genix/ui/cache` (`frontend/packages/genix-ui/cache/cache-by-ids.svelte.ts`). Records
extend `IMinimalRecord { ID: number; ss: number; upd: number; _fch?: number }`.

| Call | Use it for |
|---|---|
| `getRecordsByID<T>(apiRoute, ids) → Map<ID, T>` | a batch of IDs (the main entry point) |
| `getRecordByID<T>(apiRoute, id, { cachedRecord? })` | one ID per component; calls within 80 ms merge into one request |
| `getRecordWithCache<T>(apiRoute, id) → { record, loading, refresh() }` | reactive `$state` ref: paints local, revalidates in background |
| `getStaticRecordsByID<T>(apiRoute, ids, { cacheNamespace?, databaseCompanyID? })` | static routes (section 3) |
| `getRecordByIDUpdated<T>(apiRoute, id, updated)` | local if `upd >= updated`, else forced fetch (no consumers yet) |
| `configureCacheByIDsRoute(apiRoute, { disableTombstone: true })` | routes whose records can be reactivated |
| `clearCacheByIDs()` | drop memory + the IndexedDB database |

```ts
// frontend/routes/accounting/accounts-consolidation/accounts-consolidation.svelte.ts
import { getRecordsByID } from '@genix/ui/cache'

const [saleOrders, products] = await Promise.all([
  saleOrderIDs.length ? getRecordsByID<ISaleOrder>('sale-order-by-ids', saleOrderIDs) : new Map(),
  productIDs.length ? getRecordsByID<IProduct>('p-products-ids', productIDs) : new Map(),
])
```

**Display a record's label** — `RecordByIDText` shows `User`, else `Name`, else
`FirstName LastName`, else the placeholder:

```svelte
<script lang="ts">
  import RecordByIDText from '#components/misc/RecordByIDText.svelte'
</script>

<!-- Inside a VTable: give the column an id, render it in cellRenderer -->
<VTable {data} {columns}>
  {#snippet cellRenderer(record, column)}
    {#if column.id === 'movimientoUsuario'}
      <RecordByIDText apiRoute="users-ids" recordID={record.CreatedBy} placeholder="" />
    {/if}
  {/snippet}
</VTable>
```

(`frontend/routes/finance/cash-banks-movements/+page.svelte`.)

**With a delta-list service**: set `routeByID` on the `GetHandler`; `service.syncIDs(ids)` fetches
only the IDs missing from `recordsMap` (it does not revalidate held ones).

**Browser behaviour you can rely on:** a record fetched < 5 s ago answers from memory; a fetch
failure answers from local data with a warning; held IDs the server left out are confirmed
unchanged; tombstones answer locally.

---

## 5. Checklist

**Backend**
- [ ] `CacheByIDs: true`, one integer key, integer partition
- [ ] Record has `upd` and `ss`; table struct has `Updated`
- [ ] Handler: `GetCachedIDsRequest(companyID)` → `db.QueryCachedIDs[T]` → `HideCachedPartitions`
- [ ] Static variant: `GetStaticRequestIDs()` + your own tenant filter + `StaticCachedPartition`
- [ ] Route registered; `generate_route_ids` run

**Frontend**
- [ ] `apiRoute` equals the backend route string
- [ ] Right entry point: batch → `getRecordsByID`; per-row component → `getRecordByID`;
  label → `RecordByIDText` / `getRecordWithCache`; immutable → `getStaticRecordsByID`
- [ ] Guard empty ID lists before calling
- [ ] `disableTombstone` if deleted records can come back

## Real examples

| Pattern | Backend | Frontend |
|---|---|---|
| Versioned | `backend/security/usuarios.go` | `frontend/routes/projects/project.svelte.ts` |
| Public versioned | `backend/production/products.go` (`p-products-ids`) | `frontend/webpage/components/ProductCard.svelte` |
| Static | `backend/logistics/product-stock-movement.go` | `frontend/routes/logistics/warehouse-movements/+page.svelte` |
| Per-ID in a detail panel | `backend/sales/sale_order_create.go` | `frontend/routes/accounting/invoicing/InvoiceDetailLayer.svelte` |
| Label in a table | — | `frontend/routes/finance/cash-banks-movements/+page.svelte` |
| Generic label endpoint (`records-by-ids?table=`) | `backend/config/generic_records.go` | — (no consumer yet) |
