---
name: cache-groups-api
description: Builds an API for reports or queries answered as groups of records (buckets by day, period, category, any integer step), cached per group in the browser so only changed groups are downloaded again.
version: 0.2.0
---

# Cache Groups API

Records are split into **groups** by an index's Keys: every record belongs to the group of its Key
values. Each group has a **last write** (`lw`). The browser stores each group with its `lw` and
sends back the ones it holds; the server returns **only the groups whose `lw` changed**, and an
empty "keep yours" entry for the rest. `TypeCacheGroupS3` also keeps a server-side copy of each
group in S3, shared by every browser. Explained for people in
`backend/genix-orm/db/cache/CACHE_GROUPS.USER.md`.

**Any query that reads whole groups of records fits — not only date ranges.** A group is just a
combination of integer Key values, and a query names the groups it wants:

- `Between(a, b)` on an integer Key → **one group per integer step** from a to b: days
  (`Date`), weeks (`Week`), months, fiscal periods, a price or quantity **bucket column**, any
  sequence you can number.
- `Equals` / `In` / `Contains` on the other Keys → narrows (or multiplies) the groups: one client,
  a set of warehouses, every group containing a product.
- A query with no natural range still works: `Between(v, v)` on one Key is a single step.

Examples: sales per (day, client); purchases per (week, provider); stock movements per (day,
warehouse); records per (period, category); per (bucket, status).

| Need | Skill |
|---|---|
| A report or query read as **groups of records** (buckets, steps, ranges) | this one |
| A whole list kept in sync | `delta-cache-api` |
| Debug a group that is stale/empty, change the protocol | `cache-groups-index-internals` |

Scylla only (Dynamo and the browser `idb` build don't support cache groups).

---

## 1. Schema

```go
// backend/sales/types/sales.go (abridged)
Indexes: []db.Index{
	{Type: db.TypeCacheGroupS3, Keys: db.Cols(e.Date)},
	{Type: db.TypeCacheGroupS3, Keys: db.Cols(e.Date, e.Status)},
	{Type: db.TypeCacheGroupS3, Keys: db.Cols(e.Date, e.ClientID)},
	{Type: db.TypeCacheGroup,   Keys: db.Cols(e.Date, e.DetailProductIDs)}, // slice Key
	{Type: db.TypeCacheGroup,   Keys: db.Cols(e.Date, e.ClientID, e.DetailProductIDs)},
},
```

| Declaration | What the ORM creates |
|---|---|
| One Key | a local index on that column (a slice column panics: no local index on a collection) |
| Several scalar Keys | hidden `zz_ig_<a>_<b>` int32 hash column + local index |
| Several Keys, one a slice | hidden `zz_igs_…` `set<int>` + **global** index; a record is in one group per element |
| `TypeCacheGroupFromKey` | nothing: Keys must be a prefix of the table's `KeyIntPacking`, so a group is a key range |
| `TypeCacheGroupS3` | same as `TypeCacheGroup` + S3 group files |

Every type also writes marks to `<table>__index_updated` on each write (created at deploy).

- Entries take **`Keys` only** (no `Cols`, no `UseHash`).
- **Keys are integer columns**, and one of them is the **step column** the query ranges over with
  exactly one `Between` (a date, a week code, a period, a bucket you compute and store). The other
  Keys are filters.
- **Steps are plain integers**: `Between(a, b)` probes every integer in between. Non-contiguous codes
  (YYWW weeks: 2652 → 2701) probe the gaps too — harmless (no group there) but each costs a probe.
- **Keep groups small**: a group changes as a whole — one edited record resends its whole group.
  Day over month; a narrow bucket over a wide one.
- **S3**: every stored column must round-trip through colbin. A stored `cb:"-"` column panics at
  startup. Choose S3 for groups read by many and written rarely (past days of a report). No S3
  variant of FromKey (its read is already a cheap key range).
- A slice Key can be a computed column (`db.ComputedCol`, e.g. `DetailProductIDs`) — only inside a
  2+ Key cache group.
- **Partial updates**: writing some of a group's source columns while the struct has the others
  zero panics. Include every source column of the touched groups.
- Data that existed before a group was declared has no marks and is **invisible** until
  `RecalcGroupIndexHashes(partition)` runs (`scylla/deploy.go`).

---

## 2. Handler

```go
// backend/sales/sale_order_query.go
func SaleOrderQuery(req *core.HandlerArgs) core.HandlerResponse {
	dateStart := req.GetQueryInt16("date-start")
	dateEnd := req.GetQueryInt16("date-end")
	clientID := req.GetQueryInt("client-id")
	productID := req.GetQueryInt("product-id")
	saleOrderStatus := req.GetQueryInt("status")
	if dateStart <= 0 || dateEnd <= 0 {
		return req.MakeErr("Debe especificar el rango de fechas.")
	}
	if dateEnd < dateStart {
		return req.MakeErr("La fecha final no puede ser menor a la fecha inicial.")
	}
	// Each day is one group to probe and maybe fetch: cap the range.
	if dateEnd-dateStart > 120 {
		return req.MakeErr("Sólo se pueden consultar hasta 120 días a la vez.")
	}

	// cc-gh / cc-lw: the groups the browser holds and the lw it holds each at.
	heldGroups, err := core.ExtractGroupIndexCacheValues(req)
	if err != nil {
		return req.MakeErr(err)
	}

	saleOrders := []db.RecordGroup[types.SaleOrder]{}
	query := db.QueryIndexGroup(&saleOrders).CompanyID.Equals(req.User.CompanyID)
	for _, heldGroup := range heldGroups {
		query.IncludeCachedGroup(heldGroup.GroupHash, heldGroup.LastWrite)
	}
	query.Date.Between(dateStart, dateEnd)

	// Pick the group from WHICH params are present, never from their values.
	switch {
	case clientID > 0 && productID > 0:
		query.ClientID.Equals(clientID).DetailProductIDs.Contains(productID) // Date+ClientID+DetailProductIDs
	case productID > 0:
		query.DetailProductIDs.Contains(productID) // Date+DetailProductIDs
	case clientID > 0:
		query.ClientID.Equals(clientID) // Date+ClientID
	case saleOrderStatus > 0:
		query.Status.Equals(int8(saleOrderStatus)) // Date+Status
	}

	if err := query.Exec(); err != nil {
		core.Log("Error querying sale order groups:", err)
		return req.MakeErr("Error al obtener las órdenes de venta agrupadas.")
	}
	return core.MakeResponse(req, &saleOrders)
}
```

Register as usual (`"GET.sale-order-query": SaleOrderQuery`).

**How the index is chosen** (`scylla/select_grouped.go`):
- Partition `Equals` required; **exactly one `Between`** in the whole query.
- Every Key of a candidate group needs exactly one statement: `Equals`, `Contains`, non-empty `In`,
  or the `Between`. The group with **most Keys** wins; ties → lower fan-out (`In` sizes × range
  width), then declaration order.
- `Between` expands to one group per integer step. `In` multiplies groups.
- No match → `QueryIndexGroup did not find a compatible TypeCacheGroup for table "…"`.
- **Not allowed**: `Select()`, `Exclude()`, `GroupBy()`, `OrderDesc()`, `Limit()`, `AllowFilter()`.

**Filters outside the chosen group's Keys** are applied in memory after reading the group. Avoid
them on cached routes: the group's `lw` and the browser's group key ignore that filter, so a result
filtered by `status=1` could be served for `status=2`. **Send the server only the filters that
select a group; filter the rest on the client** (`logistics/purchase_order.go` does this for status).

**Response**: `[]db.RecordGroup[T]` → JSON `{ ig, id, igVal, records, lw }` per group. A group with
no records ever has no mark and is not returned. A non-grouped lookup in the same route can be
wrapped as `db.RecordGroup[T]{IndexID: -1, Records: flat}` — the browser passes it through
uncached (`logistics/product-stock-movement.go`, lookup by lot/serial/document).

**Server-side use with nothing cached** (books, exports): call `QueryIndexGroup` without
`IncludeCachedGroup` and flatten `group.Records` (`accounting/sales_book_api.go`).

---

## 3. Frontend

```ts
// frontend/routes/sales/sales-report/sale_order_report.svelte.ts
import { GETWithGroupCache } from '#libs/ui-runtime.svelte.ts'

const uriParams: Record<string, string> = {
  'date-start': String(dateStart),
  'date-end': String(dateEnd),
}
// Set optional params only when used: each set of param NAMES is its own cache shape.
if (clientID > 0) uriParams['client-id'] = String(clientID)

const groups = await GETWithGroupCache<ISaleOrder>('sale-order-query', uriParams)
const saleOrders = groups
  .flatMap((group) => group.records || [])
  .filter((saleOrder) => (saleOrder?.ID || 0) > 0)
```

- `GETWithGroupCache(route, uriParams)` sends the held groups as `cc-gh`/`cc-lw`, merges "keep
  yours" entries with IndexedDB, stores changed groups, and returns every group.
- Cache shape = route + **sorted param names** (values excluded); within a shape a group is keyed by
  its `igVal`. So a param that changes the chosen group must change the set of names.
- The caller flattens, sorts and trims (e.g. exact dates inside a week group, client-side status).
- Param names must match what the handler reads.
- Admin/debug: `clearGroupCache`, `listGroupCacheStats` from `@genix/ui/cache`.

---

## 4. Checklist

**Backend**
- [ ] Cache group Keys = the step column + the columns the report filters by; small groups
- [ ] `TypeCacheGroupS3` only for many-readers / rare-writes tables, no stored `cb:"-"` columns
- [ ] Handler validates and caps the range (each step is a probe and maybe a fetch)
- [ ] `ExtractGroupIndexCacheValues` → `IncludeCachedGroup` for each → one `Between` → `Exec`
- [ ] Group chosen from which params are present; non-group filters done on the client
- [ ] No `Select`/`Limit`/`OrderDesc`/`GroupBy` on the grouped query

**Frontend**
- [ ] `GETWithGroupCache(route, uriParams)`; optional params set only when active
- [ ] Param names identical to the handler's `GetQuery*` names
- [ ] Flatten `group.records`, filter `ID > 0`, sort/trim on the client

## Real examples

| Pattern | Backend | Frontend |
|---|---|---|
| Day groups, S3, optional client/product/status | `backend/sales/sale_order_query.go` + `sales/types/sales.go` | `frontend/routes/sales/sales-report/sale_order_report.svelte.ts` |
| Week groups, status filtered client-side | `backend/logistics/purchase_order.go` + `logistics/types/purchase_order.go` | `frontend/routes/logistics/purchase-orders/purchase_order.svelte.ts` |
| FromKey groups + `IndexID: -1` direct lookup | `backend/logistics/product-stock-movement.go` (`GetAlmacenMovimientos`) | `frontend/routes/logistics/warehouse-movements/warehouse-movements.svelte.ts` |
| Server-side flatten, no browser cache | `backend/accounting/sales_book_api.go`, `purchases_book_api.go` | — |
