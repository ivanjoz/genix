# Plan — migrate every delta-cache route from the `updated` timestamp pattern to `db.TypeDelta`

## 1. The two patterns

**Old (to be removed).** A hand-written branch in the handler over a `[Status]` + `[Updated]` (or
`[Status, Updated]`) pair of `TypeView` indexes, watermarked on the `updated` timestamp:

```go
updated := core.Coalesce(req.GetQueryInt("upd"), req.GetQueryInt("updated"))
query := db.Query(&records).CompanyID.Equals(req.User.CompanyID)
if updated > 0 {
    query.Updated.GreaterThan(updated)   // delta: every status
} else {
    query.Status.GreaterEqual(1)         // first sync: active only
}
```

Three defects: two writes in the same second are indistinguishable (records silently skipped), the
first-sync and delta halves need two separate views, and every handler re-implements the branch.

**New (reference: `business/types/client_provider.go` + `business/client_provider.go`).** One packed
`TypeDelta` view, watermarked on `updated_version` — the per-partition write sequence — and one
`Delta()` call:

```go
// types/…: schema
FixedValues: []db.FixedValues{
    {Col: t.Status, Values: []int64{0, 1}},
},
Indexes: []db.Index{
    {Type: db.TypeDelta, Keys: db.Cols(t.Status)},
},
```

```go
// handler
updatedSince := req.GetQueryInt("upv")
query.Select().CompanyID.Equals(req.User.CompanyID).Delta(updatedSince, 1)
```

`Delta()` pins the sync-filter column to the passed active value(s) on a first sync and fans out over
every declared `FixedValues` value on a delta sync, so rows flipped to `Status=0` still reach the
client for eviction. Which column that is comes from the query's shape — see §10.

## 2. What every converted table needs

1. **Record struct** — add `UpdatedVersion int32 \`json:"upv,omitempty"\`` (the ORM panics at compile
   time otherwise — see `bindManagedAuditColumns` in `genix-orm/scylla/reflect.go`).
2. **Table struct** — add `UpdatedVersion db.Col[XTable, int32]`.
3. **Schema** — add `FixedValues` for every declared delta key, add
   `{Type: db.TypeDelta, Keys: db.Cols(...)}`, and drop the `TypeView`s it replaces (audited
   per-table — see §9).
4. **Handler** — read `req.GetQueryInt("upv")` instead of `upd`/`updated`, replace the
   if/else with `Delta(upv, <active status values>)`, delete the per-status fan-out loops.
5. **Frontend** — bump `useCache.ver`. **Mandatory, not cosmetic:** a client that already holds a
   snapshot has an `upd` *timestamp* stored as its watermark (~3.9e8). Sent as `upv` it becomes a
   watermark far above any real sequence value, and the client would receive zero records forever.

## 3. Digit-budget check

An `updated_version` slot is 8 digits when the packed key fits an `int32`, 10 once it has already
spilled to `int64`, and **9 when one key is elastic** (`index_delta_view.go:13-24`).

Verified per candidate:

| Delta keys | slot digits | max packed | column |
|---|---|---|---|
| `Status{0,1}` | 1 + 8 | 199 999 999 | `int` |
| `Status{0..2}` | 1 + 8 | 299 999 999 | `int` |
| `Status{0..4}` | 1 + 8 | 499 999 999 | `int` |
| `Status{0,1}`, `Type{1,2}` (existing client_provider) | 1 + 1 + 8 | 1 299 999 999 | `int` |
| `Status{0,1}`, `ListID` (elastic) | 1 + 8 + 9 | 199 999 999 999 999 999 | `bigint` |
| `ListID` alone (elastic) | 9 + 9 | 999 999 999 999 999 999 | `bigint` |
| `WarehouseID.DecimalSize(5)`, `Status{0,1}` | 5 + 1 + 10 | 9 999 919 999 999 999 | `bigint` |

## 4. Work groups

### Group A — clean single-`Status` conversion (10 tables) — DONE

All ten compile to an `int` packed column (verified: `slotDigits=[1 8]`, `maxPacked` ≤ 299 999 999).

| Table | ID | Status values | Handler | Frontend service (`ver` bump) |
|---|---|---|---|---|
| `products` | 22 | 0,1 | `business/products.go` | `routes/business/products/products.svelte.ts` 10→11 |
| `warehouses` | 39 | 0,1 | `business/locations-warehouses.go` | `routes/business/branches-warehouses/…` 3→4 |
| `sites` | 31 | 0,1 | `business/locations-warehouses.go` | same service |
| `webpages` | 40 | 0,1,2 | `webpage/webpage_pages.go` | `services/webpage/pages.svelte.ts` 3→4 |
| `gallery_images` | 15 | 0,1 | `business/gallery-image.go` | `services/webpage/gallery.svelte.ts` 1→2 |
| `supply_material` | 32 | 0,1 | `logistics/supply-material-management.go` | `routes/logistics/supplies-materials/…` 2→3 |
| `cash_banks` | 6 | 0,1 | `finance/cash_banks.go` | `routes/finance/cash-banks/cajas.svelte.ts` 1→2 |
| `sales_planning` | 27 | 0,1 | `sales/sales_planning.go` | `routes/sales/sale_planning/…` 1→2 |
| `seasonality_curve` | 28 | 0,1 | `sales/sales_planning.go` | same service |
| `expenses_scheduled` | 14 | 0,1 | `finance/expenses.go` | `routes/finance/expenses/expenses.svelte.ts` 1→2 |

Notes on what the conversion actually needed:
- `webpages` first sync is `Status.GreaterEqual(1)` = active **and** published → `Delta(upv, 1, 2)`,
  with `FixedValues{Min: 0, Max: 2}`.
- `warehouses`/`sites` share one multi-table response, and each table advances its own
  `updated_version` sequence — so a single shared watermark would be wrong. The handler now reads one
  param per response key (`?Almacenes=…&Sedes=…`), which is exactly what the frontend already sends.
- `products` already carried `SaveUpdatedVersion: true`, so only the index + handler changed.
- `cash_banks` lost a `//TODO: Eliminar luego` block that forced `Updated = 1` on rows with no
  timestamp — it existed to stop a zero watermark, which `upv` (always ≥ 1 once written) makes moot.
- Four handlers (`sales_planning` ×2, `expenses_scheduled`, and the two `locations-warehouses` legs)
  dropped hand-rolled per-status loops entirely; `Delta()`'s fan-out replaces them.

### Group B — `Status` + a second filter column (1 table) — DONE

| Table | ID | Delta keys | Handler | Frontend service (`ver` bump) |
|---|---|---|---|---|
| `shared_list_records` | 29 | `Status`, `ListID` (elastic) | `business/shared-lists.go` | `services/business/shared-lists.svelte.ts` 6→7 |

`ListID` arrives from the client (`?ids=…`) and has no natural ceiling, so it is left **undeclared**
and absorbs the digit remainder — no `FixedValues` `Max` to guess. This required an ORM change; see
§7. Resolved layout: `slotDigits=[1 8 9]`, `bigint`, so `ListID` holds up to 99 999 999 and the
sequence up to 999 999 999 per partition.

Each list is its own response key (`id_<listID>`) with its own watermark, which the frontend already
sends per key — so nothing changed on that axis.

### Group C — multi-status bucket handlers — DONE

| Table | ID | Status values | Handler | Frontend (`ver`) |
|---|---|---|---|---|
| `expenses` | 13 | 0,1,2 | `finance/expenses.go` | `expenses.svelte.ts` 2→3 |
| `purchase_order` | 24 | 0,1,2,4 | `logistics/purchase-order-management.go` | `purchase_order.svelte.ts` 1→2 |

**No response-shape rework was needed.** A `TypeDelta` index is an abstraction over a packed
`TypeView`, so it also serves hand-written queries — see §12. That splits the two handlers by what
each actually wants:

- **`expenses` keeps its structure.** Each tab reads one specific status bucket, and the
  `_IDsToRemove` list is an `ExecScan` over the *complement* buckets — neither wants the fan-out. Both
  now pin `Status` themselves and call `Delta(upv)` **with no filter values**, which contributes only
  the exact watermark bound. `ExecScan` is unaffected: it shares `tableInfo` and the `Select` path
  with `Exec`, so predicates are orthogonal to execution.
- **`purchase_order` collapses into one query.** It already fanned out over all four statuses on a
  delta sync and pinned one on a first sync — precisely `Delta()`'s semantics. The whole errgroup and
  per-status merge is now `Delta(upv, int64(statusParam))`, −27 lines.

`FixedValues` lists `purchase_order`'s statuses explicitly (`{0,1,2,4}`) rather than `Min: 0, Max: 4`,
so the delta fan-out is four `IN` values instead of five.

**Watch the operator when hand-writing these queries.** A packed view builds its lower bound from the
statement value and ignores the operator, so `UpdatedVersion.GreaterThan(w)` behaves as `>= w` and
re-sends the boundary row on every poll. `Delta(w)` emits `>= w+1`, which is why the handlers call it
instead of writing the predicate themselves.

### Group D — stock tables — DONE (`product_sale_summary` deferred)

| Table | ID | Delta indexes | Handlers | Frontend (`ver`) |
|---|---|---|---|---|
| `warehouse_product_stock` | 37 | `[WarehouseID, Status]` **and** `[Status]` | `logistics/product-stock-movement.go` ×2, `logistics/product-supply-management.go` | `stock-movement.ts` 8→9 ×2, `supply-management.svelte.ts` 8→9 |
| `warehouse_product_stock_detail` | 38 | `[WarehouseID, Status]` | `logistics/product-stock-movement.go` | same |

Two delta indexes on the stock table, one per read shape, selected by what the query already pinned
— see §10. Resolved layouts: `[5 1 10]` bigint for the warehouse-scoped view, `[1 8]` int32 for the
company-wide one.

Both tables dropped `DisableUpdatedVersion` (§11) and gained `UpdatedVersion` on their record and
table structs. The three handlers lost their per-status loops.

**`product_sale_summary` (19) is still deferred.** Its delta is reconstructed from sentinel rows at
`Date=-1` and it has no `Status` column; that is a redesign, not a conversion.

**Tenant-scoping fix made in passing:** `GetAlmacenMovimientosGrouped`'s stock query ran
`AllowFilter()` with no `CompanyID` predicate, so it scanned every tenant's stock. Pinning the
partition is required for the delta view to be usable at all, so it now filters
`CompanyID.Equals(req.User.CompanyID)`. Flagging it because it is a behaviour change beyond the
mechanical conversion — the previous results were cross-tenant.

### Previously: Group D — tables with `DisableUpdatedVersion: true` (3 tables)

| Table | ID | Blocker |
|---|---|---|
| `warehouse_product_stock` | 37 | drop `DisableUpdatedVersion`; delta filter is `WarehouseID` + `Status`, and one route queries without `WarehouseID` → needs two `TypeDelta` views (`Status, WarehouseID.DecimalSize(5)` and `Status`) |
| `warehouse_product_stock_detail` | 38 | same |
| `product_sale_summary` | 19 | sentinel rows at `Date=-1` reconstruct changed dates; no `Status` column. Needs a redesign, not a mechanical conversion |

### Group E — no `Status` column, watermark-only sync (5 tables) — DONE

| Table | ID | Handler | Frontend service (`ver` bump) |
|---|---|---|---|
| `system_parameters` | 33 | `config/system_parameters.go` | `services/services/system-parameters.svelte.ts` 1→2 |
| `shipping_costs` | 30 | `sales/shipping_costs.go` | `routes/sales/shipping-costs/+page.svelte` 1→2 |
| `city_locations` | 8 | `business/locations-warehouses.go` (`GetCountryCities`) | `branches-warehouses.svelte.ts` `CountryCitiesService` 1→2 |
| `image_assets` | 16 | `business/image_assets.go` | `services/business/image-assets.svelte.ts` 3→4 |
| `image_assets_category` | 17 | same handler | same service |

All five filter **only** the partition plus the watermark — no status, no secondary column. So they
declare a keyless `{Type: db.TypeDelta}` and read with `Delta(upv)` (no filter values). This needed a
second ORM change; see §8.

Note on `image_assets`: the route returns projection structs (`ImageAssetSearchRecord`,
`ImageAssetCategoryRecord`), not table records, so both had to gain
`UpdatedVersion int32 \`json:"upv,omitempty"\`` and the queries had to `Select(… query.UpdatedVersion)`
— otherwise the client would have had no `upv` to watermark on. Its two response keys (`images`,
`categories`) already carried independent watermarks.

### Group F — special cases, excluded unless asked

- `business/product-ecommerce.go:79,132` — public ecommerce route; its watermark is also what drives
  the prerendered products `.db` snapshot (`core.SaveCacheGlobal(cacheGroupProducts, …)` keyed on
  `Updated`). Converting the read side without the snapshot side would desynchronize them.
- `config/cron-actions-scheduled.go:21` — no partition, `DisableUpdatedVersion`, `AllowFilter()`
  full scan. Not a delta view candidate.
- `logistics/product-supply-management.go:228` — reads `ProductStock`; follows Group D.

## 5. Decisions taken (confirmed)

| # | Decision |
|---|---|
| D1 | Groups **A, B and E are converted** (16 tables). Groups C, D and F stay untouched. |
| D2 | **The `TypeView`s the delta index replaces are deleted** once audited as unread. Only `products` and `shared_list_records` keep theirs, because `product-ecommerce.go` (Group F, not converted) still reads them. Superseded an earlier "keep everything alongside" decision — see §9. |
| D3 | **`Status` is not a prerequisite for `TypeDelta`.** `Delta(updatedSince)` takes zero filter values and then constrains nothing but the watermark, so the Group E tables do *not* need an invented `Status` column. |
| D4 | Every converted route gets a `useCache.ver` bump, for the reason in §2.5. |

## 7. ORM change: elastic delta key slots

Before this change, `compileSchemaDeltaView` panicked on a `TypeDelta` key that had neither a
`FixedValues` entry nor an explicit `DecimalSize()` — there was no int64 fallback, and the version
slot was only ever 8 or 10 digits. `shared_list_records.ListID` has no meaningful ceiling to declare,
so that rule forced a guess.

Now a key with neither declaration is **elastic**: it absorbs whatever digits the rest of the layout
leaves over. An elastic layout is always `bigint`, and the version slot pins to
`deltaVersionDigitsElastic = 9`, splitting the 18-digit `int64` budget between the sequence and
everything else.

- `index_delta_view.go` — `elasticKeyIndex` tracked while resolving keys; `planDeltaSlots` derives
  its width as `18 − 9 − (other declared digits)`.
- **At most one elastic key per index** — two would have no unambiguous split (panics naming both).
- `.Int32()` plus an elastic key panics: the layout cannot fit an `int32`.
- The "a different key order would have fit an int" hint is suppressed, since no order could.
- An elastic key may be `Keys[0]`, in which case only `Delta(upv)` — no filter values — can read it;
  `Delta(upv, x)` still panics in `declaredValuesOfColumn`, which is the pre-existing behaviour.

Tests: `TestDeltaViewRejectsKeyWithoutDeclaredRange` is replaced by
`TestDeltaViewLetsAnUndeclaredKeyAbsorbTheDigitRemainder` (asserts `[1 8 9]`, `bigint`, and the
999 999 999 version ceiling), plus `TestDeltaViewSizesALoneElasticKey` (`[9 9]`) and
`TestDeltaViewRejectsTwoKeysWithoutDeclaredRange`. Full `genix-orm` suite green.

**Carried-over hazard, not introduced here:** `computePackedInt64ValueNonNegative` trims a component
that overruns its slot from the right, silently aliasing it. The version slot has a write-time guard
(`maxDeltaVersionValue`, `insert-update.go:151`); key slots do not — an elastic `ListID` above
99 999 999 would collide rather than fail. That is already true of every `DecimalSize()`-declared
packed-view key, so it is a property of packed views, not of this change.

## 8. ORM change: keyless (watermark-only) delta indexes

Group E's five handlers filter only the partition plus the watermark. A packed view is the wrong
shape for that: a range on the trailing slot is only contiguous in packed space once every leading
key is pinned to a value, and here there is nothing to pin. What they want is a plain view on
`updated_version` — the exact analogue of the `[Updated]` `TypeView` they already had.

So `{Type: db.TypeDelta}` with **no** `Keys` is now legal and compiles to precisely that:
`compileSchemaDeltaView` returns early with `Keys = [updated_version]`, which
`compileSchemaView` recognises as `isSingleDeclaredSimpleView` and emits as an ordinary MV — not a
`Type 8` packed range view. No packing means no digit slot, so `maxDeltaVersionValue` stays 0 and
writes keep the column's full `int32` range rather than a trimmed budget.

Two guards moved:
- `index_delta_view.go` — the `len(indexCfg.Keys) == 0` panic is replaced by the early return; the
  version-key construction is factored into a `versionKeyColumn(digits)` closure shared by both paths.
- `reflect.go:440` — the generic `Indexes entry must not be empty` check now exempts `TypeDelta`,
  since that is the one index kind which supplies a key of its own.

The query side needed nothing: `Delta(updatedSince)` with no filter values already emitted only the
watermark predicate, and `resolveDeltaSyncFilterColumn` already had the
`"Delta() was given filter values, but its delta index declares no key to filter on"` path — dead
code until now, which is a good sign this was the intended design.

Tests added: `TestDeltaViewWithNoKeysIsAPlainUpdatedVersionView` (asserts a plain
`updated_version` view, no `Type 8` view, and no version ceiling) and
`TestDeltaWithFilterValuesRejectsAKeylessDeltaIndex`.

## 6. Validation after each group

```bash
cd backend && go build ./...
bash scripts/scripts.sh check-tables      # per scripts/CHECK_TABLES_SCRIPT.md
```

`check_tables.go` already understands `TypeDelta` (`declaresDeltaIndex`, line 285) and cross-checks
it against `SaveUpdatedVersion`. Schema compilation prints one
`Delta view registered: table=… slotDigits=… packedType=…` line per delta index — worth reading to
confirm each landed on `int` rather than `bigint`.

Results across A + B + E: `go build ./...` clean, `check_tables` clean (41 pairs), full `genix-orm`
suite green, and throwaway `db.MakeTable[…]()` compile passes over all 17 delta tables confirmed each
landed on its intended shape (`int32` packed for Group A, `bigint` elastic for `shared_list_records`,
watermark-only for Group E). `bun run check` on the frontend reports the same 12 pre-existing errors
as a clean tree — none introduced.

Not yet validated: the actual materialized-view creation and a live delta round-trip against
ScyllaDB. The database will be dropped and redeployed from scratch, so no migration path is needed.

## 9. Dead-view cleanup

Once a table read through `Delta()`, the `TypeView`s that served the old hand-rolled branch had no
readers left. Audited by grepping every `Status.*` / `Updated.*` predicate in production code
(excluding `exec/`, `tests/`, `genix-orm/`) and mapping each hit to its record type. Removed:

| Table | Removed |
|---|---|
| `warehouses`, `sites`, `webpages`, `gallery_images`, `cash_banks` | `[Status]`, `[Updated]` |
| `supply_material` | `[Status.DecimalSize(1)]`, `[Updated.DecimalSize(10)]` |
| `sales_planning`, `seasonality_curve`, `expenses_scheduled` | `[Status, Updated]` packed |
| `system_parameters`, `shipping_costs`, `city_locations`, `image_assets`, `image_assets_category` | `[Updated]` |

**Kept**, with a comment saying why: `products` (`[Status]` read at `product-ecommerce.go:86,157`,
`[Updated]` at `:84`) and `shared_list_records` (`[ListID, Status]` at `:134,168`,
`[ListID, Updated]` at `:132`). Both belong to Group F, which keeps a timestamp watermark because it
also drives the prerendered `.db` snapshot.

Also removed: the inert `db:"status,view"` / `db:"updated,view"` / `db:"updated,view.1"` tag suffixes
on `Warehouse`, `Site` and `GalleryImage`. `ParseDBTag` (`genix-orm/db/metacache.go:84`) only honours
`frozen`, `list` and `set`, so `view` was silently ignored — a stale second syntax for the same
declarations. Note `pk` is inert for the same reason and still appears on several structs; left alone
as it is outside this migration.

## 10. ORM change: several delta indexes per table, selected by query shape

A packed delta view is only reachable through a range on its trailing `updated_version` slot once
every leading key is pinned to a value — `TestPackedViewServesLeadingKeyAlone` shows what happens
otherwise: pinning `Status` on a `[Status, Type, upv]` view yields the range
`[1_0_00000000, 1_2_99999999+1)`, spanning **every** `Type` and **every** version. The watermark is
silently ignored.

That is what decides the new rule. `Delta()` no longer takes the filter column from `Keys[0]` of the
first delta index; it picks **the delta index whose every key is pinned by predicates already on the
query except exactly one**, and that remaining key is what `syncFilterValues` applies to. When
several indexes qualify the most specific wins.

`warehouse_product_stock` is the motivating case:

```go
Indexes: []db.Index{
    {Type: db.TypeDelta, Keys: db.Cols(e.WarehouseID.DecimalSize(5), e.Status)},
    {Type: db.TypeDelta, Keys: db.Cols(e.Status)},
},
```

```go
// pins WarehouseID → routes to [WarehouseID, Status], Status is the sync filter
query.CompanyID.Equals(companyID).WarehouseID.Equals(warehouseID).Delta(upv, 1)

// does not → routes to [Status]
query.CompanyID.Equals(companyID).Delta(upv, 1)
```

Implementation: `equalityPinnedColumns()` collects the `=` / `IN` predicates already on the query,
and `resolveDeltaSyncFilterColumn` scores each delta index against them. Panics name the declared
index shapes and the pinned columns, so a mismatch says what to fix.

**This narrows a previous guarantee.** `Delta()` used to be callable before the other predicates;
now it must come last, because with several candidate indexes the choice depends on predicates that
may not be set yet. The doc comment says so and the panic message repeats it. Restoring
order-independence means deferring the choice to `tryGetOrCompileSelectStatement`, which is a larger
refactor of the select path — worth doing if the constraint ever bites.

The old rule was also masking a bug: `deltaInferredSchema` declares `[Channel, Type]` and its test
called `Delta(0, 1)` with neither key pinned. That resolved to `Channel` and produced a query whose
watermark did nothing. The test now pins `Type` first, and
`TestDeltaRejectsBeingCalledBeforeTheKeyColumnsArePinned` covers the case that used to pass silently.

New tests: `TestDeltaPicksTheWiderIndexWhenTheQueryPinsItsKey`,
`TestDeltaFallsBackToTheNarrowIndexWhenItsKeyIsOpen`,
`TestDeltaRejectsBeingCalledBeforeTheKeyColumnsArePinned`.

## 11. ORM change: `DisableUpdatedVersion` deleted

The flag is gone from `TableSchema`. `updated_version` is now maintained exactly when something reads
it, which `consumesUpdatedVersion` (`genix-orm/scylla/reflect.go`) defines as:

1. the schema declares a `TypeDelta` index, or
2. `SaveUpdatedVersion: true` (the by-IDs cache), or
3. the table struct declares an `UpdatedVersion` column — so declaring the field is never silently
   ignored.

In cases 1 and 2 the record and table structs **must** declare the field, since the value has to
reach the client to come back as a watermark; the existing panic covers it.

This inverts the old default. Before, every table without the flag paid one counter read per write
for a DB-only column it never read. Now that cost is opt-in. Removing the flag from
`product_sale_summary`, `cron_actions` and `usage_log` is therefore a no-op — none of them has a
consumer, so none gets the column, exactly as before.

## 12. `TypeDelta` serves plain queries too

`TypeDelta` compiles to a packed `TypeView`, so nothing about it requires `Delta()`. A hand-written
query that pins the index's keys and ranges on `updated_version` routes to the same view:

```go
query.CompanyID.Equals(companyID).Status.Equals(0).UpdatedVersion.GreaterEqual(w)
// Capability selected: signature=company_id|=|status|=|updated_version|~
//   source=…__pk_status_updated_version_rng_view
```

Verified by `TestDeltaViewServesAPlainQueryWithoutDelta`. This is what makes Group C cheap: a handler
that wants one specific status bucket, or a scan over the buckets a row may have left, uses the delta
view directly instead of the fan-out.

Three ways to read a delta index, in increasing order of how much `Delta()` does for you:

| Form | Emits | Use when |
|---|---|---|
| `Status.Equals(s)` + `UpdatedVersion.GreaterEqual(w+1)` | nothing (fully manual) | never — the `+1` is easy to forget |
| `Status.Equals(s)` + `Delta(w)` | the exact watermark bound | the read wants one specific bucket |
| `Delta(w, s)` | watermark **and** the status filter, fanned out on a delta sync | the read wants "active now, plus whatever was evicted" |

`Delta(w)` with no filter values skips index selection entirely, so it composes with any predicates
and with `ExecScan`.
