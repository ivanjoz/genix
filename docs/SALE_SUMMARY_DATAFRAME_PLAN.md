# Sale summary on a DataFrame (plan)

Status: **plan, nothing implemented.** Goal: serve `/sales/sale_orders_charts` from a genix-orm
DataFrame computed straight from `sale_order`, and **deprecate `ProductSaleSummary`** (table id 19)
with its whole read-modify-write path.

The approach: a record can declare its frames with a **typed `ToFrames` method**. The frame engine
only ever sees the lines that method returns, never the record's columns. List columns
(`[]int32{...}`) are **not** supported by the engine: the method turns them into lines.

## 1. Where we are

| Piece | Today |
|---|---|
| Data | `product_sale_summary` rows `(CompanyID, Date, ProductID)` with a packed `Stats` blob (`SerializeInt30Struct`) |
| Write path | `sale_summary.go` (~480 lines): `MakeSummaryChangeFromOSaleOrder`, divisor alignment, clamp-to-zero, `negateSummaryChanges`, `summaryActionsOfStatus`; called from `sale_order_create.go` and `sale_order_annul.go` |
| Read path | `GET sale-summary` → `Date.In(...)`, rebuilds `types.SaleSummary` (columnar per day), `WithDeltaReplace` |
| Frontend | `SaleOrdersChartsService extends GetHandler` re-downloads the full range each time |
| Frames today | Keys/Rows/Sums are **scalar persisted columns** of the table; one record → one row; the changelog / `__dtf_log` stores the before-values of the tracked columns |

`sales-report` does **not** read `sale-summary`: it lists individual orders through
`sale-order-query` (`GETWithGroupCache`). It is not aggregatable and is **out of scope**.

## 2. The frame declaration (option B: one struct per frame)

```go
// sales/types/sales.go

// SaleSummaryFrame is one sale line as the "sale-summary" frame sums it.
type SaleSummaryFrame struct {
    Date          int16 `frame:"key"` // one file per day
    ProductID     int32 `frame:"row"`
    SubDivisor    int16 `frame:"row"` // its own row, so sixths never add to twelfths
    AmountPEN     int32 `frame:"sum"`
    DebtAmountPEN int32 `frame:"sum"`
    SoldUnits     int32 `frame:"sum"`
    SoldSub       int32 `frame:"sum"`
    PendingUnits  int32 `frame:"sum"`
    PendingSub    int32 `frame:"sum"`
}

func (e SaleOrderTable) GetSchema() db.TableSchema {
    return db.TableSchema{
        // ...
        DataFrames: []db.DataFrame{{
            Name:     "sale-summary",
            Version:  1,
            ToFrames: db.ToFrames((*SaleOrder).ToSaleSummaryFrames),
            // A day can be fully paid or fully delivered, which empties DebtAmountPEN or
            // PendingUnits for the whole file, and most products have no sub-units (section 3.5).
            UseZeroSumsMask: true,
        }},
    }
}

// ToSaleSummaryFrames is what one sale adds to the day summary, from its current state only.
func (e *SaleOrder) ToSaleSummaryFrames() []SaleSummaryFrame {
    if e.Status == OrderStatusAnnulled {
        return nil
    }
    isDeliveryPending := e.Status == OrderStatusPending || e.Status == OrderStatusPaid
    isPaymentPending := e.Status == OrderStatusPending || e.Status == OrderStatusDelivered
    lines := []SaleSummaryFrame{}
    for i, productID := range e.DetailProductsIDs {
        quantity := core.UnpackQuantityLine(e.DetailQuantities[i])
        line := SaleSummaryFrame{
            Date: e.Date, ProductID: productID, SubDivisor: subDivisorOfLine(e, i),
            AmountPEN: e.AmountInPEN(core.QuantityAmount(quantity, e.DetailPrices[i], e.DetailSubPrices[i])),
            SoldUnits: quantity.Units, SoldSub: quantity.Sub,
        }
        if isDeliveryPending {
            line.PendingUnits, line.PendingSub = quantity.Units, quantity.Sub
        }
        if isPaymentPending {
            line.DebtAmountPEN = line.AmountPEN
        }
        lines = append(lines, line)
    }
    return lines
}
```

### 2.0 Two ways to declare a frame, one declaration type

`db.DataFrame` stays the only declaration, with the same shape as dynamo's. A frame is declared
**either** with columns **or** with `ToFrames`, never both (panic when the schema compiles):

```go
type DataFrame struct {
    Name    string
    Version int16 // bump when the logic behind the frame changes (ToFrames' body)

    // Way 1: columns of the record, as today and as in dynamo. One record → one row.
    Keys  []Coln
    Rows  []Coln
    Sums  []Coln
    Count bool

    // Way 2: a typed method that turns the record into lines. One record → 0..N rows.
    ToFrames FrameProjection // built by db.ToFrames((*Record).ToXFrames)

    // Both ways, opt-in (section 3.5).
    UseZeroSumsMask bool
}
```

Way 1, unchanged for existing frames:

```go
DataFrames: []db.DataFrame{{
    Name:  "sales-by-client",
    Keys:  db.Cols(e.Date),
    Rows:  db.Cols(e.ClientID),
    Sums:  db.Cols(e.TotalAmount, e.DebtAmount),
    Count: true,
}},
```

- `db.ToFrames` is a generic helper (`func ToFrames[R, F any](method func(*R) []F) FrameProjection`)
  so the field itself is not generic and `db.DataFrame` stays one plain struct. It reflects `F`
  once and keeps the adapter that flattens `F` in field order.
- Inside the engine **way 1 compiles into a one-line projection** (keys, rows, sums read from the
  record's columns; no line when `status` = 0). There is one engine path; way 1 keeps its current
  behavior and its callers do not change.
- Reading: both ways use `db.QueryFrame[SaleOrder](...)`. A way 2 frame's `FrameRow.Key` stays
  empty (only `Sums`); the typed read was dropped (see below).

### 2.1 Rules of the frame struct (way 2)

Checked by `db.ToFrames` when the schema compiles (panic on a mistake):

- 1–3 `key` fields (the file), 1–3 `row` fields (the row inside it), 1–8 `sum` fields (more
  than 8 only without `UseZeroSumsMask`, section 3.5). Keys and
  rows are non-negative integers; sums are integers or floats. Declaration order is the frame's
  column order (the file and the browser decoder follow it).
- Two lines with the same keys and rows are summed by the engine, so the method never has to
  group by product itself.
- **The method must be pure**: only the record's own fields. No `core.Now()`, no catalog lookups.
  The engine calls it on past states of the record and subtracts.
- **`Version`** (the `1` above): a change to the method's logic does not change the struct, so it
  would not rebuild the frame. Bumping `Version` does. The struct layout and `Version` together
  are the frame's **shape**; a new shape rebuilds the frame on the next run.

*Implemented differently:* there is no typed read. The only reader is the browser through
`ExecChanged`, and a key is filtered by the record column of the same field name:

```go
body, err := db.QueryFrame[SaleOrder](companyID, "sale-summary").
    Between(saleOrderTable.Date, dateFrom, dateTo).ExecChanged(g, h)
```

### 2.2 What `Stats` becomes

| `Stats` field | `SaleSummaryFrame` field | Derived from the sale |
|---|---|---|
| `CompanyID` | partition | — |
| — (row key) | `Date` (key) | `sale.Date` |
| — (row key) | `ProductID` (row) | `DetailProductsIDs[i]` |
| `SubDivisor` | `SubDivisor` (**row**, not a sum) | `DetailSubDivisor[i]`, `QuantityDivisorNone` (1) when absent |
| `Quantity` / `SubQuantity` | `SoldUnits` / `SoldSub` | `UnpackQuantityLine(DetailQuantities[i])` |
| `QuantityPendingDelivery` / `Sub…` | `PendingUnits` / `PendingSub` | same, while `Status ∈ {Pending, Paid}` |
| `TotalAmount` | `AmountPEN` | `AmountInPEN(QuantityAmount(q, price, subPrice))` |
| `TotalDebtAmount` | `DebtAmountPEN` | same amount, while `Status ∈ {Pending, Delivered}` |
| `Updated` | dropped | the file's snapshot replaces it |

Deleted with it, because the frame is a function of the sale's **current** state and not an
accumulator: `actions`, divisor alignment (`alignSummaryChangeDivisor`, `alignStatsToChangeDivisor`),
clamp-to-zero, `negateSummaryChanges`, `summaryActionsOfStatus`, `annulSaleSummary`. An annulled
sale returns no lines, so it leaves every file by itself.

### 2.3 Sub-units cost nothing for products without them

Most products have no sub-unit: `SubDivisor = 1`, `SoldSub = PendingSub = 0` on every row. On
paid and delivered days `DebtAmountPEN` and `PendingUnits` are all zero too.

With format v3 (today, `dataframe/codec.go`) such a column still costs `varint min + uvarint
divisor` plus one head byte per 128 rows: ~40 bytes on a 1,000-row day file. **File format v4**
(section 3.5) brings that to **1 byte** (the zero sums mask, which this frame opts into) plus ~4 bytes for the constant
`SubDivisor` level.

When some products do sell loose, their `Sub` values are the heavy tail the codec already
handles with **patches**: only those rows pay a patch (position + high bits).

A product whose divisor was refined mid-range gets two rows that day (`SubDivisor` 6 and 12); the
frontend converts and combines them (section 5).

## 3. The engine change (genix-orm `dataframe/`, `scylla/`, `db/`)

Separate repo: commit and push inside the submodule.

### 3.1 From tracked columns to tracked lines

The engine stops knowing about columns. A record's state, for the engine, is **its lines per
frame**; everything that today asks "what did this record hold at X?" asks "which lines did it
produce at X?".

| Area | Today | With `ToFrames` |
|---|---|---|
| History | current tracked values + changelog of before-values per column | current lines (`ToFrames(stored row)`) + logged **before-lines** per frame |
| `valuesAt(X)` | undo column changes after X | `linesAt(X)`: undo line-set changes after X |
| A run / fresh read | subtract the row at w, add the row at T | subtract every line at w, add every line at T, per (file, row) |
| `filesOfHistory` | the file of the row at each moment | the union of the lines' files at each moment |
| Soft delete | the `status` column, built into the engine | the method returns no lines; the engine has no status rule |
| Versions file, `ReadChanged`, wire | — | **unchanged** |
| File codec | format v3 | **format v4** (section 3.5): the genix-ui decoder changes with it |

### 3.2 Write path (scylla)

1. A table with `ToFrames` frames **reads the whole stored row** of every record it writes (one
   select per partition, by ID, shared with the delta log's read). The ORM cannot know which fields
   the method reads, so it reads all of them.
2. Merge the written columns over the stored row (a payment writes only `Status` and audit fields).
3. For each frame, `before = ToFrames(stored)`, `after = ToFrames(merged)`. If they differ
   (or the record is new), log `before` (or the insert flag) in `<table>__dtf_log`, **before**
   the record rows, exactly as today.
4. A write that does not change any frame's lines logs nothing.

Log row `changes` blob, new format:

```text
u8       format (2)
u8       flags (insert)
per frame whose lines changed:
  u8       frame index
  u16      shape hash   (a log row from an older shape is ignored: that frame is rebuilt anyway)
  uvarint  line count
  line*    zigzag varint per key, row and sum field, in struct order
```

Size: a status change on a 30-line sale logs ~30 × 15 B ≈ 450 B, once. Today it logs one column.
Acceptable at sale-order volume; the log lives 48 h.

### 3.3 Rebuild and scans

`ScanHistories` / `ScanPartitionHistories` read full rows and call `ToFrames`. The first deploy
runs `RebuildDataFrames("sale-summary")`: a full `sale_order` scan.

### 3.3.1 dynamo (berryapps), in the same change

dynamo keeps the changelog on the item (`cgl`, `dynamo/data_frame.go`), not in a log table. It
moves to lines too, so both backends share one engine path:

- `cgl` entries hold the **before-lines** per frame, with the same entry layout as the scylla log
  blob (section 3.2): frame index, shape hash, lines. `NextChangelog` takes before/after lines
  instead of before/after tracked values; the 36 h window and the drop rule stay.
- The full stored record is already at hand: `putIfVersion` / `Modify` read it to compute the next
  `cgl` (`putChangelog`), so `ToFrames(stored)` costs no extra read.
- `trackedValuesOf` / `historyOf` become "lines of the item" through the frame's projection.
  `compileDataFrames` accepts `ToFrames`, `Version` and `UseZeroSumsMask` like scylla.
- berryapps' current frames are way 1 and keep their declarations. The `cgl` format and the file
  format v4 both change their shape, so **every berryapps frame is rebuilt and every stored `cgl`
  is stale**: reprocessing berryapps is done separately, after this change.
- Tests: the dynamo frame tests run on the new `cgl` with a way 1 and a way 2 frame.

### 3.4 `db/` layer

- `db.DataFrame` gains `Version`, `ToFrames` and `UseZeroSumsMask` (section 2.0).
- `db.ToFrames[R, F](method)`: reflects `F` once, returns a `FrameProjection` holding the
  `dataframe.Spec` built from the tags and the engine-side adapter `func(record) [][]int64` that
  flattens `F` in field order.
- Way 1 frames compile into a `FrameProjection` too: one line from the record's columns, none when
  `status` = 0.
- `db.QueryFrame[R, F]` returns `[]F`, filled back from the flattened row (floats ÷ `FloatScale`);
  `db.QueryFrame[R]` keeps returning `[]db.FrameRow` for way 1.

### 3.5 File format v4: empty and constant columns

Two rules, both in `codec.go`:

**1. The zero sums mask, opt-in per frame with `UseZeroSumsMask`.** Declared by a frame whose sum
columns are often empty for a whole file, with either way of declaring it. When set, after the
Rows and before the sum columns:

```text
u8       zero sums mask   bit s = 1 when sum column s has a non-zero value in this file
column*  only the sum columns whose bit is set, in struct order
```

- Fixed `u8`: a frame with `UseZeroSumsMask` has at most **8 sums** (checked when the schema
  compiles).
- A column whose bit is 0 costs **nothing**, wherever it sits in the struct: the order of the
  fields does not change the size. A file where 2 of 6 sums have data pays 1 byte for the other 4.
- The record count (`Count`) is never zero on a stored row, so it is outside the mask.
- The flag is part of the frame's shape: turning it on or off rebuilds the frame.

**2. Without the flag (the default)**, no mask byte: every sum column is written, with no limit on
their number. An all-zero column still costs only 2 bytes, as a constant column (rule 3). That is
the right choice for frames whose sums always carry data, such as
`enrollment-report` (`RecordedMinutes`, `Points`).

**3. Constant columns.** Any column the codec writes (a sum, a Rows level, a nested level's child
counts or first values) whose values are all equal is written as

```text
varint   the value   (the existing minimum)
u8       0           in the place of the divisor: "constant, no blocks follow"
```

2 bytes whatever the row count, instead of 2 bytes + 1 per 128 rows. The `SubDivisor` level of a
day with no sub-units (child counts all 1, first values all 1) drops from ~20 bytes to ~4 bytes on
1,000 rows. Divisor 0 is free: a real divisor is one of 1, 2, 5, 8, 10, 100, 1000. (Implemented
as divisor 0, not the `0xFF` first planned.)

`formatVersion` goes 3 → 4. It is in every frame's shape, so every frame rebuilds once on deploy.
The browser decoder `frame.wire.ts` (genix-ui, separate repo) and the shared vectors
`dataframe/testdata/frame_wire_vectors.json` move to v4 in the same change.

### 3.6 Tests

- `dataframe_test.go` brute force: records whose projection emits 0..N lines, lines that move
  between files and rows, duplicate (file, row) lines inside one record, a projection that empties
  on soft delete, crashes mid-run, rebuilds, partition rebuilds, the simulated `ReadChanged` client.
- Log blob round-trip, including a row of an older shape being ignored.
- Codec v4: masks with zero columns first/middle/last, all zero, none zero; a frame without `UseZeroSumsMask`
  with an all-zero column (still exact); constant Rows and nested levels; truncated files and the
  decode fuzzer; the size test of a 1,000-row day file without sub-units. The v4 vectors are
  pinned by both `TestFrameWireVectors` (Go) and `frame.wire.test.ts`.
- scylla: a partial update (only `Status`) produces the right before/after lines from the merged
  row.
- `sales/types`: `ToSaleSummaryFrames` against hand-computed lines: USD sale, packed quantities
  with sub-units, every status, annulled.

## 4. The sales change

1. Add `SaleSummaryFrame` + `ToSaleSummaryFrames` (`sales/types`, pure) and declare the frame on
   `SaleOrderTable`.
2. `GET sale-summary` becomes the frame route: `db.QueryFrame[SaleOrder]` with the frame cache's
   `from`/`to` as the `Date` range, `.ExecChanged(g, h)`. The client asks 8 weeks (cap 400 keys).
   The route name is unchanged, so `api_routes.generated.go` is too.
3. Delete: `sale_summary.go`, `sale_summary_status.go`'s builder, `ProductSaleSummary` + its
   table, `types.SaleSummary`, `SaleOrderProductStats`, the summary calls in
   `sale_order_create.go` and `sale_order_annul.go`, their tests, the entry in
   `controllers.generated.go`.
4. Drop table 19 (`product_sale_summary`) in the same deploy.

## 5. The frontend change

- `sale_orders_charts.svelte.ts`: stop extending `GetHandler`; call
  `GETFrameFiles('sale-summary', 'sale-summary', { fromKey, toKey })` (genix-ui frame cache).
- New `sale_orders_charts.ts` (pure): `file.keys[0]` = Date, `file.rows[0|1]` = ProductID /
  SubDivisor, `file.sums[0..5]` = the six sums in struct order. Builds what the two chart
  components consume, converting rows of the same product with different divisors to one.
  `sale_orders_charts.test.ts` covers it.
- `SaleOrdersChartsDailySummary.svelte` / `SaleOrdersChartsByProduct.svelte`: adapt only if the
  record shape they read changes.
- Update the route's `DOCUMENTATION.md` and `RATIONALE.md`.

## 6. Order of work

1. Engine: lines instead of columns, `ToFrames` + `db.ToFrames[R, F]`, way 1 compiled to a
   one-line projection, log format 2, file format v4, scylla and dynamo,
   tests. Push genix-orm; move `frame.wire.ts` to v4 and push genix-ui.
2. `ToSaleSummaryFrames` + tests; declare the frame; rebuild on a dev dataset and **compare the
   frame against the old `product_sale_summary` rows** before deleting anything.
3. Route + frontend switch.
4. Delete the old table and write path.

## 7. Open questions

1. **`WarehouseID`**: no current report filters the charts by warehouse, so it is left out. Add
   it only if a by-warehouse view of the charts is wanted; adding it later is one automatic
   rebuild (a `sale_order` scan).

Resolved: `sales-report` is out of scope; table 19 is dropped in the same deploy; dynamo moves to
lines in the same change and berryapps is reprocessed separately.
