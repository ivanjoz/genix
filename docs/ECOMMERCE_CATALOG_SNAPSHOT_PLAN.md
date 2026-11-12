# Ecommerce Catalog Snapshot Plan

Status: **implemented** (2026-11-12). Backend `backend/business/ecommerce-catalog.go`, client
`frontend/webpage/services/ecommerce-catalog{.ts,.svelte.ts}`. All decisions settled.

How the web store (`frontend/webpage/`, also rendered by `/webpage-builder/preview`) gets the
products it may sell, with their stock, close to real time. Heavy data is served by Cloudflare R2;
the backend answers only small deltas. It is a standalone module: **no delta cache and no service
worker**.

## Why

Today the store loads `live/products-c<companyID>.db` (`business/product-ecommerce.go`): every
active product of the company, with no stock. Then `GET.p-products-ecommerce` sends the same tables
whole, as a replace, through the genix-ui delta cache. Problems:

- **It ships products the store cannot sell** (no stock, oversold beyond `WebExtraStock`,
  `WebHidden`), and the storefront has no stock figure at all.
- **The "delta" is the whole catalog every time**: a file seed gives the delta cache no cursor, so
  its first request is a full sync.
- **The listing and checkout disagree**: checkout refuses what the listing shows.

## Overview

```
                 ┌───────────── IndexedDB (one value) ─────────────┐
page load ──►    │ catalog + since + generation                    │ ──► first paint
                 └─────────────────────────────────────────────────┘
   │ nothing saved?
   ├──► R2  live/ecommerce-c<cid>.cb  (colbin, max-age=15) ──► catalog, since = file.Updated
   │
   └──► API GET p-ecommerce-catalog?company-id&since&gen
           ├─ small delta  ──► merge, save, since = NextSince (readTime − 8 s)
           └─ UseFile      ──► download the file from R2, replace, ask again from it
```

One backend request per visit. A large catch-up is served by R2.

## Sellable rule

Pure functions in `backend/logistics/types/web_sellable.go` (`WebSellableStock`, `WebOrderFits`),
used by the file, the delta **and checkout** (`validateWebStock`), so the listing and checkout
cannot disagree.

For each presentation of a product with `Status = 1` and `WebHidden = false` (`PresentationID 0`
for a product without presentations), with the margin **shared per product** (D9):

```
net(presentation) = Σ over warehouses open to online sales (business.Warehouse.IsOpenToOnlineSales)
                        (Quantity + DetailQuantity − PendingDeliveryStore − PendingDeliveryOnline)
oversold          = Σ over presentations max(−net, 0)
marginLeft        = max(WebExtraStock − oversold, 0)
sellable(presentation) = max(net, 0) + marginLeft
```

An order fits when the product's `oversold` after it is at most `max(WebExtraStock, oversold
before)`. A product is listed when at least one of its presentations has `sellable > 0`. The stock
shown is the exact figure.

## Records

The file and the delta carry the same record shapes. The client keeps them by `ID`.

| Record | Fields |
|---|---|
| product | `ID, Name, CategoryIDs, BrandID, Price, FinalPrice, Image, Stock, ss`. `Stock` = `[{PresentationID, Units}, …]`, every sellable presentation, always complete. `Image` is the main image ID |
| brand / category | `ID, Name, ss` |

`ss: 0` means "remove it": the product is no longer sellable (sold out, oversold, hidden, inactive)
or the brand or category was deleted.

## The file

`live/ecommerce-c<companyID>.cb`: a **self-describing colbin message** (the schema travels inside
it, `colbin.MarshalSelfDescribing`), uploaded to R2 with `Content-Type: application/vnd.colbin` and
`Cache-Control: public, max-age=15`. A rebuilt file reaches every visitor within about 30 s: up to
15 s at the Cloudflare edge plus up to 15 s in the browser cache. The storefront decodes it with
the colbin wasm (`decodeColbinBody` from `@genix/ui/http`, loaded lazily).

```
{ Meta: {Created, Updated, Generation}, Productos: [product…], Marcas: [brand…], Categorias: [category…] }
```

- **Only sellable products**, with the same record structs the delta sends.
- **`Created`**: SUnixTime taken **before** the first read of the build.
- **`Updated`** = `Created − 4` ticks (**8 s**, D1). The client's first delta starts there. See
  "Concurrency margin".
- **`Generation`**: the company's catalog generation (see "Warehouse changes").

The company's current file is recorded in a `core.Cache` row (key `ecommerce_catalog_file`,
content `created,updated,generation`). **The backend never reads the file back from R2**: that row
is all it needs.

Built under a per-company lock (`core.ActionBuildEcommerceCatalog`, LockAction 11, 60 s lease). A
request that needs a file to answer queues behind a build under way and then reuses its file; a
request that only noticed the file aging uses a try-lock and skips.

## The delta

`GET p-ecommerce-catalog?company-id=<companyID>&since=<SUnixTime>&gen=<generation>`, public.
Answered in colbin to a client that announces it in `Accept`, JSON otherwise.

1. `readTime = SUnixTime()`, taken **before** any read.
2. No file recorded, or one built for another generation: build it now (waiting on the lock).
   If `since` is missing or `gen` differs from the file's generation: answer `UseFile`.
3. `ChangedSince(since)` (see "Prerequisite") on:
   - `products`: price, name, `WebHidden`, `WebExtraStock`, status;
   - `warehouse_product_stock` on the warehouses open to online sales: movements and
     pending-delivery counters, both of which bump `Updated`;
   - the brand and category shared lists (one read per `ListID`: it leads their delta index).
4. Take the union of the product IDs touched by product or stock changes, and recompute
   `sellable` for every presentation of each, over **all** of its online warehouse rows, not only
   the changed ones. Supplies (`Status 2`) are skipped.
5. Evaluate the rebuild rule (it may build inline, see below).
6. **More than 100 records** and the current file is newer than `since`: answer `UseFile`. The
   client downloads the file from R2 and asks again from its `Updated`; that second delta is small,
   because the file is recent.
7. Otherwise answer:

```
{
  UseFile: false, FileCreated, Generation,
  NextSince:  readTime − 4,
  Productos:  [ /* full product record, or {ID, ss: 0} */ ],
  Marcas:     [ ... ],
  Categorias: [ ... ]
}
```

`UseFile` answers carry `FileCreated` (the client appends `?v=<FileCreated>` to bust the CDN and
browser cache) and `Generation`.

A touched product is always sent **whole**. That covers a product restocked after the file was
built, which the client has never seen, with no record of which products each file contained.

`since` missing (a client with nothing saved whose file 404'd): build the file if it is missing,
then answer `UseFile`.

## Rebuild rule

The file is rebuilt when, since the current file:

- **more than 100 records** changed, or
- **at least 1** record changed and **10 minutes** have passed since the file's `Created`.

It is checked **in the delta request only**: no cron. A company nobody visits needs no fresh file.
The request uses what it already read, with no extra query:

- the records it read with `Updated ≥ file.Updated` are changes since the file;
- when the client's `since` is at or after the file's `Updated`, that is a **lower bound** (changes
  between the file and the client's `since` are not read). The rule may fire one visit late, which
  is harmless.

When the rule fires, the request **builds the file inline**, under the company's lock as a
try-lock (a build already under way means this one is skipped), and then answers. Not scheduled:
`core.ScheduleCronAction` lands on the next 5-minute frame, and a Lambda may stop background work
after the response. A failed rebuild is logged and the delta is answered anyway.

The file is also built when the delta request finds none recorded (first visit after deploy), or
one for an older generation.

## Warehouse changes

Switching a warehouse's online sales off, or deactivating it, changes every product's `sellable`
without touching any product or stock row: `ChangedSince` cannot see it. `PostWarehouse` compares
the stored warehouse's `IsOpenToOnlineSales` with the new one, and when it flips **bumps the
company's catalog generation** (`core.Cache` key `ecommerce_catalog_generation`, value = the bump's
SUnixTime). The next delta request finds the file on an older generation, builds a new one, and
answers `UseFile` to every client on the old generation. A new warehouse holds no stock yet, so it
bumps nothing: its first stock rows reach the catalog through the delta.

## Concurrency margin

**8 s** (D1). SUnixTime ticks are **2 s**, so 8 s = **4 ticks**. A write is stamped with
`Updated` when it is prepared and becomes visible at commit; a transaction that stamps at `t` and
commits after a read has passed it would be missed by a cursor starting after `t`. So every cursor
starts 8 s before the read that produced it:

- the file: `Updated = Created − 4`;
- the delta: next `since = readTime − 4`.

The overlap resends a few records, which the client overwrites with identical values. A
transaction slower than 8 s can still be missed, until the next rebuild or the next touch of the
same product. For reference, the ORM's own delta window allows 16 s (`DeltaWriteLagTicks = 8`).

## Client (`frontend/webpage/services/`)

No delta cache, no service worker, everything on the main thread.

- **`ecommerce-catalog.ts`**: pure logic and the record types, tested with `bun test`
  (`ecommerce-catalog.test.ts`):
  - seed a saved catalog from the file;
  - merge a delta (replace by `ID`, delete on `ss: 0`);
  - map a record to the store's `IProduct` and group products by category.
- **`ecommerce-catalog.svelte.ts`**: the reactive `EcommerceCatalog` (`$state`), the load sequence,
  the two fetches, `getEcommerceCatalog()`.
- **Storage**: raw IndexedDB (`frontend/libs/indexeddb-value.ts`), database `genix-ecommerce`,
  **one value** per company (`ecommerce-catalog-c<cid>`):
  `{schemaVersion, generation, since, productos, marcas, categorias}`. A storefront catalog is a
  few hundred KB, so whole-value read and write is simpler than per-record storage. No Dexie, to
  keep the storefront bundle small.
- **`schemaVersion`** is a code constant. A saved value with another version is discarded.

Page load:

1. Read IndexedDB. If a catalog is saved, publish it (first paint, no network).
2. If nothing is saved, fetch the file from R2, decode it, publish it, and take `since` and
   `generation` from its `Meta`. On a 404, call the delta without `since`.
3. Call the delta. On `UseFile`: fetch the file with `?v=<FileCreated>`, replace the catalog, and
   call the delta again from the file's `Updated` (at most once per load). Otherwise merge.
4. Save to IndexedDB with `since = NextSince`.

Two tabs loading at once each run this sequence; the last write wins, and both values are valid.

## Prerequisite: `ChangedSince` in genix-orm

The condition is `Updated ≥ T`, but `Updated` alone leads no index. On MariaDB a `TypeDelta` index
is a native index on `(CompanyID, packed)`, where `packed` = the pinned values followed by
`Updated` (`genix-orm/mariadbdriver/delta.go`). So `WHERE CompanyID = ? AND Updated >= ?` reads
every row of the company. The fast read is one contiguous span per pinned prefix:

```
CompanyID = ? AND packed >= PackedKey(prefix, T) AND packed <= PackedKey(prefix, max)
```

The driver already runs it (`readDeltaRecords`), but only inside the window protocol, which adds a
hash check and answers a full replace past 100 changes. **Decided (D5): expose it** as a query
option, e.g. `query.CompanyID.Equals(cid).ChangedSince(T)`:

- it picks the `TypeDelta` index like `Delta()` does;
- it fans out over every pinned prefix, so a row that moved to `Status 0` is still read;
- it returns plain records, with no hash, no `Metadata` and no replace threshold.

Both tables already have the index (`Status`; `WarehouseID, Status` on stock), so writes cost
nothing more. Implemented in the SQL drivers (MariaDB, PostgreSQL, OceanBase) with a conformance
test; the Scylla driver answers an error (Scylla is deprecated). genix-orm is its own repo:
committed and pushed there.

## Edge cases

- **Restock after the build**: the stock row changed, so the product is sent whole and the client
  adds it.
- **Sold out, or margin used up**: `{ID, ss: 0}`; the client removes it.
- **Hidden, or price change**: the product row changed, so it is recomputed and sent.
- **Client away for a week**: its delta is over 100 records and the file is newer, so the client
  reloads from R2.
- **Big restock**: the delta is over 100 records. The rule fires, the request builds the new file,
  and answers `UseFile`.
- **A file not rebuilt in time** (no visits): nothing is stale for visitors, because the delta
  always covers it.

## What goes away

Deleted: `live/products-c<cid>.db`, `GetProductsEcommerce` / `GET.p-products-ecommerce` (route ID
27 retired), `buildProductsDbFile`, its watermarks (`products_db_built`, the `cache_global` groups
1–3 marked by `PostProducts` and `PostSharedLists`, `business/types/ecommerce_cache.go`), the
30-minute rebuild cron (`product-ecommerce-cron.go`, cron action 3), and the storefront loader
`frontend/webpage/services/products.svelte.ts` with its delta cache usage.

Left behind as data, harmless: the old `products-c<cid>.db` objects on R2, the `cache_global` rows
of groups 1–3, and the last pending cron row of action 3 (it has no handler, is logged and ages out
of the 60-minute lookback).

## Decisions

- **D1. Margin**: 8 s (4 ticks).
- **D2. File → product IDs table**: not needed. Touched products are sent whole.
- **D3. Client storage**: IndexedDB, standalone (no delta cache, no service worker).
- **D4. Brands and categories**: in the same file and delta.
- **D5. `ChangedSince`**: added to genix-orm.
- **D6. Stock granularity**: per `(product, presentation)`, embedded in the product record.
- **D7. Stock figure**: the exact number.
- **D8. Large delta**: over 100 records, the client is sent to the file when it is newer.
- **D9. `WebExtraStock` with presentations**: one margin **shared per product**: what any
  presentation oversells uses it up for all of them (see "Sellable rule").
- **D10. File format**: colbin (`.cb`), the project's standard for stored files; the same record
  structs as the delta, no hand-written text format.
- **Rebuild rule**: over 100 changes, or at least 1 change and 10 min since the file. Checked in
  the delta request only, and built inline under a try-lock.
- **Cache**: `max-age=15` on the file.
