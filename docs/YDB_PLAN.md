# YDB driver — implementation plan (full parity with ScyllaDB)

**Audience.** An implementing agent with no prior context. Read §0–§2 before writing any code. Work
the steps of §16 in order; every step has an acceptance check, and a step does not start before the
previous one passes. When this plan and the code disagree, the code wins: stop, report the
difference, and ask.

**Goal.** YDB 26.3 replaces ScyllaDB behind **one build tag** (`-tags ydb`), plug and play like the
FoundationDB driver (`docs/FOUNDATIONDB_PLAN.md`): same `db.Executor` contract, same `TableSchema` /
`Index` declarations, same queries, same cache wire protocols (genix-ui unchanged). **No table
declaration and no query in `backend/` changes.** The shared conformance suite proves that Scylla
and YDB return the same results.

**Status, 2026-11-06.** Steps 0 and 1 done (results in §18b). All design decisions are taken,
D5 included. Step 2 waits on Step P (§4).

---

## 0. Read first

| File | Why |
|---|---|
| `CLAUDE.md` (repo root) | Operating mode (supervised), code rules, submodule rules, conventions |
| `docs/FOUNDATIONDB_PLAN.md` | The sister plan. Its Step 1 facade and Step 2 extraction are **prerequisites** here (§4). Its §14 has the benchmark format to copy |
| `backend/genix-orm/INTERNALS.md` §2–§4, §7, §10 | Driver contract: `Executor`, `Codec`, `DBTypeResolver`, the three init hooks, `TableCore`, "adding a driver" |
| `backend/genix-orm/scylla/ORM_INTERNALS.md`, `scylla/RATIONALE.md` | What every index type, view, cache protocol and write phase means. **Parity = these semantics** |
| `backend/genix-orm/db/schema.go` | `Index`, index type constants (`TypeGlobalIndex`=1 … `TypeCacheGroupS3`=12), `TableSchema`, `ColumnNameUpdated`/`ColumnNameVersion` |
| `backend/genix-orm/db/table.go`, `db/index_source.go`, `db/compiled_state.go`, `db/describe_index.go` | `TableCore` and the neutral compiled state every driver materializes (`IndexSources`, `DeltaIndexes`, `PackedIndexes`, `IndexGroups`, `CompositeBucketIndexes`, `ComputedColumns`, `TextSearchIndex`) |
| `backend/genix-orm/db/coltype.go` | Stable ORM type IDs (1 string, 2 int64 … 13 `[]int32` …). Never renumber |
| `backend/genix-orm/db/registry.go` | `db.Controller` interface |
| `backend/genix-orm/db/cache/*.USER.md`, `db/cache/RATIONALE.md`, `db/cache/delta_window.go` | The three cache protocols. `DeltaWriteLagTicks = 8` (16 s) is the settle window this plan relies on |
| `backend/genix-orm/scylla/delta_view.go` (header comment), `index_groups.go`, `data_frame_log.go` | The Scylla side tables YDB reproduces: `<table>__delta_moves`, `<table>__index_updated`, `<table>__dtf_log` |
| `backend/genix-orm/fdbdriver/*.go` | The second driver. Copy its file split and its shapes (`writePhase`, `recordSink`, `FdbController`), not its storage code |
| `backend/db/driver_fdb.go`, `facade_fdb.go`, `facade_scylla.go`, `connect.go`, `autoincrement_scylla.go` | How a build tag swaps the driver in the backend facade |
| `backend/tests/bench/README.md`, `setup.go` | The benchmark stack |
| `fareward/src/limiter/storage.rs`, `limiter/access.rs`, `reqlog/writer.rs`, `sysmetrics/writer.rs` | Step 8 |

**Rules that apply** (from `CLAUDE.md`):
- `backend/genix-orm` and `fareward` are **git submodules**: commit **and push inside the
  submodule**. A root commit does not publish them.
- Pre-alpha: no backwards compatibility, no runtime fallbacks for old data shapes. Wipe and reseed.
- English everywhere. Expressive, grepable names. Comment rationale on non-obvious blocks. No more
  code than necessary. Standalone functions over controller objects.
- Supervised mode: anything not settled in §1–§2 → **stop and ask**, all questions at once.
  Never delete code that is not obviously dead without asking.
- Never `time.Now()` for a persisted value: the ORM's managed clock (`db.Now()` / `core.SUnixTime()`).
- Reading/editing files: `Read` / `Edit` tools, not `sed`/`cat`. Search: `rg`, `fd`.
- Report honestly: failing builds and skipped scope are reported with their output.

---

## 1. Settled decisions

| # | Decision | Choice |
|---|---|---|
| Y1 | The switch | Build tag `ydb`. `backend/db/{driver,facade,autoincrement}_ydb.go` are `//go:build ydb && !js`. Every file tagged `!js && !fdb` today gets `&& !ydb`. A runtime switch is impossible: the driver is a type argument of `TableStruct` |
| Y2 | Declarations | Unchanged. Hints with no YDB meaning (`set`/`list`/`frozen` db tags, `Int32()` on a view, `UseListAsDefault`) are accepted and ignored |
| Y3 | Server | **YDB 26.3**, image `docker.io/ydbplatform/local-ydb:26.3.1.21` (newest 26.3 tag, 2026-11-05). Always pin the full tag |
| Y4 | Go client | **`github.com/ydb-platform/ydb-go-sdk/v3` v3.156.1**, pure Go over gRPC. No gateway daemon, no cgo: the backend stays `CGO_ENABLED=0`. Use the **Query service** (`db.Query()`) for every DML and DDL. Never the legacy `ExecuteDataQuery` (1,000-row cap) |
| Y5 | Storage model (D1) | **Native relational tables**: one YDB row table per ORM table, typed columns, primary key `(partition, keys…)`, YDB secondary indexes for scalar index kinds, ORM-maintained **side tables** for what YDB cannot index |
| Y6 | Index consistency | **Every YDB secondary index is `GLOBAL ASYNC`**, the TypeDelta covering index included (user decision). Same contract as Scylla's materialized views. Reads through an async index run in **stale read-only** mode (the only mode YDB allows on them) |
| Y7 | Side tables | Written by the ORM **in the record's transaction** (consistent with the row). They hold **keys only, never copied columns** (user decision: copies multiply write pressure). Reads join back to the main table on its full primary key |
| Y8 | Slice indexes | Any index whose key includes a slice column is a side table, one row per element. Remaining index keys after the slice column go into the side-table key after `element`. `CompileTable` **rejects** such an order column if it is `updated` or `version` (user decision) |
| Y9 | Dedupe | Side-table reads over several elements are merged and deduplicated in Go while merging; one element needs no dedupe. No `DISTINCT` in YQL |
| Y10 | Slice columns (D4) | YDB row tables have no container column types. A slice column is a YDB `String` column holding the **colbin** encoding (same bytes the Scylla codec produces for blobs) |
| Y11 | Sequences (D3) | **YDB owns the sequences** under `-tags ydb`: table `_sequences(name Utf8, value Int64)`, range reservation in one serializable transaction. YDB `Serial` columns are not used |
| Y12 | fareward (D2) | fareward gets a **YDB store** (Rust `ydb` crate 0.18.2, `ydb-platform/ydb-rs-sdk`) behind a Cargo feature. Step 8 |
| Y13 | Step order | **FDB plan Step 2 (planner + compile extraction into `genix-orm/db`) is finished first.** No copies of planner or compile code inside `ydbdriver` |
| Y14 | Planner | The shared `AccessPlan` (FDB plan §7.1) is rendered to YQL by `ydbdriver` |
| Y15 | Cache protocols | Same wire protocols and settle windows. `DeltaWindowCursor.ReadTime` keeps coming from the ORM clock |
| Y16 | Text search | Existing `db.TextSearchBackend` (search daemon). YDB's native full-text index is not used |
| Y17 | Scope | VPS mode. Lambda, column (OLAP) tables, changefeeds, vector/full-text/JSON indexes: out of scope (§19) |
| Y18 | Posting lists | Not the default for slice indexes. Only if the benchmark shows a hot element; ask first |
| Y19 | Compression | **lz4 on every table** (main tables, side tables, `_sequences`): `FAMILY default (COMPRESSION = "lz4")` in each `CREATE TABLE` (user decision, 2026-11-06). Scylla uses `ZstdCompressor` level 3, but YDB 26.3 row tables **reject zstd**: schemeshard answers `Unsupported ColumnCodec` (`ydb/core/tx/schemeshard/schemeshard_info_types.cpp:1456`, branch `stable-26-3`), the datashard maps it to LZ4 (`ydb/core/tx/datashard/datashard_user_table.h:218`, `// FIXME: not supported`), and the docs list only `"off"` / `"lz4"`. zstd exists only for column (OLAP) tables. No driver-side zstd on values. Revisit when a YDB release accepts zstd on row tables |
| Y20 | Partitioning (D6) | User decision, 2026-11-06. YDB partitions a row table by **ranges of its primary key** (no hashing); each partition is one single-threaded DataShard tablet. **Every main table, side table and `_sequences`**: `AUTO_PARTITIONING_BY_SIZE = ENABLED`, `AUTO_PARTITIONING_BY_LOAD = ENABLED`, `AUTO_PARTITIONING_PARTITION_SIZE_MB = 2000`, `AUTO_PARTITIONING_MIN_PARTITIONS_COUNT = 1`, `AUTO_PARTITIONING_MAX_PARTITIONS_COUNT = 50`, written explicitly even where they equal YDB's defaults (load splitting is **off** by default). **Every async index** gets the same five settings through `ALTER TABLE t ALTER INDEX ix SET (…)` right after it is created: an index table does not inherit the table's settings and YDB does not accept them at index creation. **No pre-split** (`UNIFORM_PARTITIONS` needs a `Uint32`/`Uint64` first key column, ours is `company_id Int32`; `PARTITION_AT_KEYS` on company ids would be a static guess). The benchmark runs a **warm-up** so load splits happen before measuring (Milestone) |

---

## 2. Decisions taken late

**D5 — Tables with more than 20 YDB indexes (decided 2026-11-06, user): it is an error.** YDB allows
20 secondary indexes per table (side tables do not count). `CompileTable` panics naming the table and
its indexes; there is no automatic spill to side tables. Step 2 still prints the real count per genix
table first; if a genix table exceeds 20, stop and report it (do not work around it).

D6 (partitioning) was decided: Y20.

---

## 3. Architecture

```
backend (CGO_ENABLED=0, -tags ydb)
  app tables ──► backend/db facade ──► genix-orm/ydbdriver (pure Go)
                                          │  gRPC, ydb-go-sdk/v3 Query service
                                          ▼
container genix-bench-ydb ─────────────────────────────────────────
  ydbplatform/local-ydb:26.3.1.21   single node, gRPC :2136, UI :8765, database /local
```

### 3.1 Module and files

`ydbdriver` is its **own Go module** (like `dynamo/`) so the default Scylla build does not carry
gRPC/protobuf in `genix-orm/go.mod` (see `genix-orm/BINARY_SIZE_FINDINGS.md`).

```
backend/genix-orm/ydbdriver/                 module github.com/ivanjoz/genix-orm/ydbdriver
  go.mod            require github.com/ivanjoz/genix-orm (replace => ../),
                    github.com/ydb-platform/ydb-go-sdk/v3 v3.156.1
  driver.go         Exec[TableT, RecordT] (zero-size), TableStruct alias, init() hooks,
                    errNotImplemented, connection (Connect, Ping, the shared *ydb.Driver)
  codec.go          db.Codec implementation + DBTypeResolver (§7)
  values.go         record ⇄ YDB value conversion: build `List<Struct<…>>` params, scan rows
  table.go          YdbTable{db.TableCore; …}, compile cache, CompileTable, D5/Y8 checks
  index_mapping.go  IndexSource → native async index or side table (§8.3, §8.4)
  ddl.go            CREATE TABLE / ADD INDEX / side tables text; pure, golden-tested
  deploy.go         Deploy + homologation against DescribeTable (§8.6)
  write.go          §9 write algorithm (phases, chunks, transaction)
  side_tables.go    entry computation and diff for slice-index side tables, delta moves,
                    cache-group marks, frame log
  read.go           §10 AccessPlan → YQL, execution, paging, decoding
  delta.go          TypeDelta reads (window rows, moves)
  cache_by_ids.go   QueryCachedIDs, QueryCachedGenericByIDs
  cache_groups.go   SelectGrouped, QueryIndexGroup, group snapshots
  data_frame_backend.go  dataframe.Backend + PartitionRecordScanner, frame log reads
  text_search.go    sync hooks to db.TextSearchBackend
  sequences.go      _sequences: reserve, set, list
  controller.go     YdbController[RecordT, TableT] implementing db.Controller
  admin.go          WipePartition, RenameColumns, ListCounters
  container/README.md   run commands, verified facts (§5)
  *_test.go         unit tests (no DB); *_it_test.go gated by GENIX_YDB_IT=1
backend/db/driver_ydb.go, facade_ydb.go, autoincrement_ydb.go
backend/go.mod      require + replace github.com/ivanjoz/genix-orm/ydbdriver => ./genix-orm/ydbdriver
backend/core/security.go   [ydb] section → core.Env.YDB_ENDPOINT, core.Env.YDB_DATABASE
config.example.toml        [ydb] section
```

### 3.2 Naming (grepable, mirrors Scylla)

| Object | YDB path / name |
|---|---|
| Main table | `<database>/<keyspace>/<table>` e.g. `/local/genix/products` |
| Native index | `ix_<IndexSource.Name>` (IndexSource names are already stable and unique per table) |
| Slice-index side table | `<table>__sx_<IndexSource.Name>` |
| Delta moves | `<table>__delta_moves` (same name as Scylla) |
| Cache-group marks | `<table>__index_updated` (same name as Scylla) |
| DataFrame log | `<table>__dtf_log` (same name as Scylla) |
| Sequences | `<keyspace>/_sequences` |

`WipePartition` and `DeleteViewsAndIndexes` find side tables by the `<table>__` prefix.

---

## 4. Prerequisites (from the FDB plan) — Step P

YDB needs, in `genix-orm/db`, everything the Scylla compile and planner compute today. As of
2026-11-06 the FDB plan's Step 2 is at **stage 2b, in progress**: the neutral state types exist on
`TableCore`, but the **compile code still lives in `scylla/`** (`makeTable`, `index_view_compile.go`,
`index_config.go`, `packed_indexes.go`, `view_tables.go`, the registration halves of `delta_view.go`
and `index_groups.go`), and the `AccessPlan` planner (stage 3) does not exist.

Before YDB Step 2, these must exist in `genix-orm/db` and be used by Scylla with all Scylla tests
green (unit + `GENIX_ORM_SCYLLA_TEST` integration):

1. A neutral compile entry point that fills **every** `TableCore` field from a `TableSchema`,
   including `IndexSources` (with the virtual columns they need: packed, hash, `zz_dlt_*`, `zz_igs_*`,
   composite buckets, concatenated keys) — callable by any driver.
2. `AccessPlan` / `KeySpan` and the planner (FDB plan §7.1), with the plan cache keyed by
   `computeSelectShapeHash`.
3. Write preparation (FDB plan §7.2): managed columns, self-parse, autoincrement, computed columns,
   affected-column resolution, "columns updated together" check.

This is FDB plan work and follows that plan's rules. Its open mechanism question (how to split the
CQL closures from compile-time state, ~18 sites) is asked there, not here. **Accept:** FDB plan
Step 2 acceptance passes. Only then start YDB Step 2.

YDB Steps 0 and 1 do not depend on Step P and can run before or during it.

---

## 5. Container (Step 0)

```sh
podman volume create genix-ydb-data
podman run -d --name genix-bench-ydb --hostname localhost \
  -p 2136:2136 -p 8765:8765 \
  -e GRPC_PORT=2136 -e MON_PORT=8765 \
  -e YDB_USE_IN_MEMORY_PDISKS=false \
  -v genix-ydb-data:/ydb_data \
  --cpuset-cpus 2,3 --memory 3g \
  docker.io/ydbplatform/local-ydb:26.3.1.21
```

- Endpoint `grpc://localhost:2136`, database `/local`, UI `http://localhost:8765`.
- CPU/memory must match the Scylla and FDB benchmark containers (FDB plan §14: cores 2,3). Record the
  final flags in `ydbdriver/container/README.md`.
- Verify and record (corrections go into the README, like FDB plan §5 "Corrections found"): the
  ready probe (container health or `ydb -e grpc://localhost:2136 -d /local scheme ls`), persistence
  across `podman restart`, and port collisions with the load stack (Scylla on 127.0.0.2).
- `/local` with no auth. Credentials (`[ydb] user/password` or token) are added only if the user asks.

---

## 6. SDK conventions (`ydb-go-sdk/v3` v3.156.1)

Verify every API name below against the v3.156.1 godoc before using it; the SDK renames options
between minor versions. Record any difference in this section.

- **Connection.** One process-wide `*ydb.Driver` from `ydb.Open(ctx, endpoint+database)`, opened
  lazily on first use (server) or eagerly when `ConnectParams.Scripting` (exec scripts), like the
  FDB facade. Session pool limit from config (default 64).
- **Read-write transaction.** `driver.Query().DoTx(ctx, func(ctx, tx query.TxActor) error {…},
  query.WithTxSettings(query.TxSettings(query.WithSerializableReadWrite())))`. The SDK retries
  the whole callback on `ABORTED` (optimistic lock broken) and transport errors, so the callback
  must be idempotent: build every mutation inside it from the re-read state.
- **Read-only.** `driver.Query().Do(ctx, …)` with tx control: `SnapshotReadOnly` for primary-key and
  side-table reads; `StaleReadOnly` for any statement that reads `VIEW ix_*`; `OnlineReadOnly` is
  allowed for single point reads if the Step 0 numbers favour it.
- **Statements.** Always parameterized. Query text is limited to **10 KB**, total parameters to
  **50 MB**: never inline values, never build `UNION ALL` chains proportional to the input. Bulk
  inputs travel as one `List<Struct<…>>` parameter consumed with `AS_TABLE($param)`.
- **Results.** Stream with the Query service result iterator (result sets → rows); never
  materialize a whole partition in one response.
- **Errors.** Wrap with the table name and operation: `fmt.Errorf("ydbdriver: %s %s: %w", op, table, err)`.
- **DDL.** Through `driver.Query().Exec` with no transaction. Schema listing through
  `driver.Scheme().ListDirectory`; table description through `DescribeTable`.

---

## 7. Types and codec

### 7.1 Type mapping (`DBTypeResolver`, keyed by ORM type ID from `db/coltype.go`)

| ORM type ID | Go | YDB column type |
|---|---|---|
| 1 | string | `Utf8` |
| 2 / 3 / 4 / 5 | int64 / int32 / int16 / int8 | `Int64` / `Int32` / `Int16` / `Int8` |
| 6 / 7 | float32 / float64 | `Float` / `Double` |
| 8 | bool | `Bool` |
| 9 (`TypeBlob`) | `[]byte` and anything without a native mapping | `String` |
| 11–17 | `[]string`, `[]int64` … `[]float64` | `String` (colbin, Y10) |
| 21–27 | pointers to scalars | same as the scalar (nullable) |
| 31–35 | pointers to slices | `String` (colbin) |
| unsigned Go kinds | uint8 … uint64 | `Uint8` … `Uint64` (native; no blob needed, unlike Scylla) |
| packed / hash / delta virtual columns | computed by `db` | `Uint64` (or `Uint32` when `PackedIsInt32`): the **unsigned** layout from `db.PackSlotValues` / `DeltaIndexInfo.PackedKey`. Scylla's sign flip (`deltaPackedStoredValue`) is **not** used |
| `zz_igs_*` group hash set (slice) | `[]int32` | never a column: it is a side table (§8.4) |

- Primary-key and partition columns are `NOT NULL`. Every other column is nullable; NULL decodes to
  the zero value, as a null Scylla column does.
- Managed columns (`created`, `updated`, `version`) are real columns even when the record struct has
  no field for them (as Scylla stores them).
- Virtual columns that a native index keys on (packed, hash, delta, concatenated, computed) are real
  columns, written by the ORM on every write, same names as Scylla.

### 7.2 Codec (`codec.go`)

Implements `db.Codec` (INTERNALS §4), installed with `db.SetCodec` in `init()`:
- `RenderValue` / `EncodeStatementValue`: return the Go value to put in a `Struct` param field
  (int types as-is, slices → colbin bytes, unsigned as-is).
- `AssignValue`: decode a scanned YDB value into the field (colbin for slices).
- `CompileDriverAccessors`: nothing (no collection literals).
- `ApplyCollectionOptions` / `ApplyCollectionDefaults`: return the type unchanged with `DBType`
  resolved by §7.1.

`values.go` builds, per table, a cached **struct type descriptor** for `List<Struct<…>>` params
(column name → YDB type), so a write chunk is one param, and a per-column scanner for reads that
writes through `IColInfo.SetValue(ptr, value)`.

---

## 8. Schema and DDL

### 8.1 Main table

```sql
CREATE TABLE `genix/products` (
    company_id Int32 NOT NULL,
    id         Int64 NOT NULL,
    name       Utf8,
    price      Double,
    tags       String,            -- []int32 as colbin
    created    Int32,
    updated    Int32,
    version    Int16,
    zz_dlt_ss  Uint64,            -- delta virtual column (pinned | updated)
    PRIMARY KEY (company_id, id),
    INDEX ix_<name> GLOBAL ASYNC ON (company_id, category_id) COVER (name, price),
    INDEX ix_<deltaName> GLOBAL ASYNC ON (company_id, zz_dlt_ss) COVER (<every other stored column>),
    FAMILY default (COMPRESSION = "lz4")          -- Y19
)
WITH (
    AUTO_PARTITIONING_BY_SIZE = ENABLED,
    AUTO_PARTITIONING_BY_LOAD = ENABLED,
    AUTO_PARTITIONING_PARTITION_SIZE_MB = 2000,
    AUTO_PARTITIONING_MIN_PARTITIONS_COUNT = 1,
    AUTO_PARTITIONING_MAX_PARTITIONS_COUNT = 50     -- Y20
);
-- then, for every index (an index table does not inherit the settings above):
ALTER TABLE `genix/products` ALTER INDEX ix_<name> SET (
    AUTO_PARTITIONING_BY_SIZE = ENABLED, AUTO_PARTITIONING_BY_LOAD = ENABLED,
    AUTO_PARTITIONING_PARTITION_SIZE_MB = 2000,
    AUTO_PARTITIONING_MIN_PARTITIONS_COUNT = 1, AUTO_PARTITIONING_MAX_PARTITIONS_COUNT = 50
);
```

- Primary key = Scylla partition column first, then the clustering keys in declaration order
  (`TableCore.PartKey`, `TableCore.Keys`). Without a partition, the keys alone.
- A table without `Partition` that Scylla stores under a fixed partition keeps that behaviour (read
  how `scylla` handles it before deciding; ask if unclear).
- `COVER` lists only columns that are neither in the index key nor in the primary key (YDB adds
  those implicitly).

### 8.2 `IndexSource` → YDB (scalar keys)

`IndexSource.Type` codes (see `db/index_source.go`):

| Type | Meaning | YDB |
|---|---|---|
| 1 | global index on one column | `GLOBAL ASYNC ON (col)` (no partition) |
| 2 | local index | `GLOBAL ASYNC ON (partition, col)` |
| 3 | hash / cache-group index (virtual hash column) | `GLOBAL ASYNC ON (partition, hashCol)` |
| 6 | simple view | `GLOBAL ASYNC ON (partition, keys…) COVER (Cols, or all stored columns when Cols is empty)` |
| 7 | hash view | `GLOBAL ASYNC ON (partition, hashCol) COVER (…)` |
| 8 | range / radix (packed) view, TypeDelta included | `GLOBAL ASYNC ON (partition, packedCol) COVER (…)`; a TypeDelta view covers **every** stored column |
| 9 | view table (fan-out) | side table (§8.4) |
| `Index.Partition` set | partition override | that column replaces `partition` as the first index key column |

Any `IndexSource` whose key includes a **slice** column (global index on `set<int>`, cache group over
a slice, composite buckets, `zz_igs_*`) is not a native index: §8.4. `TypeCacheGroupFromKey` needs
no index (it is a primary-key range). `TextSearch` creates no index here.

Count of native indexes > 20 → D5.

### 8.3 Index identity

Native indexes are named `ix_<IndexSource.Name>`. A declaration change that changes the source name
or its columns is a **drop + add** at deploy (YDB cannot alter an index in place, nor switch it
between sync and async). Deploy prints each change; an index build runs in the background in YDB:
deploy waits for the build operation to finish before returning (`fn-init` must leave a usable
schema).

### 8.4 Side tables

Key-only. Created with the main table, the Y20 partitioning settings, and the same
`FAMILY default (COMPRESSION = "lz4")` (Y19) as every table; the templates below omit both for
brevity, `ddl.go` always emits them.

**Slice-index side table** `<table>__sx_<source>`:

```sql
-- local (partitioned) index
CREATE TABLE `genix/products__sx_<source>` (
    company_id Int32 NOT NULL,
    element    Int32 NOT NULL,        -- slice element type (or Int32 group hash)
    <order columns…>,                 -- remaining index keys after the slice column, if any (Y8)
    id         Int64 NOT NULL,        -- main-table key columns
    PRIMARY KEY (company_id, element, <order columns…>, id)
);
-- global index (no partition): PRIMARY KEY (element, <order columns…>, company_id, id)
```

`CompileTable` panics if an order column is `updated` or `version`, naming table, index and column
(Y8). If a genix table hits it: stop and ask; do not change the declaration.

**Delta moves** `<table>__delta_moves` (same columns as Scylla's, `delta_view.go:deltaMovesCreateScript`):

```sql
CREATE TABLE `genix/products__delta_moves` (
    partition_id Int32 NOT NULL, index_tag Utf8 NOT NULL, pinned_prefix Uint64 NOT NULL,
    updated Int32 NOT NULL, id Int64 NOT NULL,
    PRIMARY KEY (partition_id, index_tag, pinned_prefix, updated, id)
);
```

**Cache-group marks** `<table>__index_updated` (Scylla: `index_groups.go:getIndexUpdatedTableCreateScript`):

```sql
CREATE TABLE `genix/products__index_updated` (
    partition_id Int32 NOT NULL, index_id Int16 NOT NULL, index_hash Int32 NOT NULL,
    last_write Int32,
    PRIMARY KEY (partition_id, index_id, index_hash)
);
```

**DataFrame log** `<table>__dtf_log` (Scylla: `data_frame_log.go`, TTL 48 h):

```sql
CREATE TABLE `genix/products__dtf_log` (
    half_hour Int32 NOT NULL, partition_id Int32 NOT NULL, ts Int32 NOT NULL, record_id Int64 NOT NULL,
    changes String, expire_at Uint32,
    PRIMARY KEY (half_hour, partition_id, ts, record_id)
)
WITH (TTL = Interval("PT0S") ON expire_at AS SECONDS);   -- expire_at = write time + 48 h
```

Key column types (`partition_id`, `id`, `record_id`) follow the table's real key types, as the
Scylla scripts do.

### 8.5 Sequences

```sql
CREATE TABLE `genix/_sequences` (name Utf8 NOT NULL, value Int64, PRIMARY KEY (name));
```

### 8.6 Deploy and homologation (`deploy.go`)

For every controller passed to `db.Deploy`:
1. Compile the table (panics surface here, at boot).
2. `DescribeTable`. Absent → `CREATE TABLE` (with indexes) + side tables. Present → diff:
   - missing columns → `ALTER TABLE … ADD COLUMN`;
   - default family compression not `lz4` → `ALTER TABLE … ALTER FAMILY default SET COMPRESSION "lz4"`;
   - extra columns → **report only**, never drop (Scylla deploy does the same);
   - column type changed → report and fail (pre-alpha: the user wipes and reseeds);
   - indexes: drop the ones not declared, add the missing ones (§8.3), then apply the Y20
     settings to each new index with `ALTER INDEX … SET`;
   - partitioning of the table, of each side table and of each index differs from Y20 →
     `ALTER TABLE … SET (…)` / `ALTER INDEX … SET (…)` (both cheap, no data rewrite);
   - side tables: create missing, report extra.
3. Idempotent: a second run changes nothing and prints nothing but a summary.

Unit-test `ddl.go` with golden strings for: a plain table, a table with every index type, a delta
table, a slice index with and without order columns, a table without partition.

---

## 9. Write algorithm (`write.go`, `side_tables.go`)

Serves `Insert`, `Update`, `UpdateExclude`, `InsertUpdate`, `InsertUpdateInclude`,
`InsertUpdateExclude`, `Merge`. Reuse the FDB driver's shapes: a call becomes a list of
`writePhase{records, writtenColumns, isInsert}`.

**Chunking.** A chunk is one transaction. Size it by **row writes**, not records: main rows +
side-table rows (a record with a 50-element slice index counts 51+). Start at 1,000 row writes and
≤ 8 MB of params; tune in Step 0. One record never spans chunks. A large `Insert` is not atomic
across chunks (same as Scylla's unlogged batches).

**Per chunk, inside `DoTx` (serializable):**

1. **Prepare** — shared code from Step P: self-parse, managed columns (`created`/`updated` from the
   ORM clock, `version` on TypeDelta tables), autoincrement (`_sequences`, reserved **once per
   chunk**, before `DoTx`, so a retry does not burn IDs), `KeyIntPacking` / `KeyConcatenated`,
   computed columns, virtual columns, the written-column set.
2. **Pre-read** the stored rows of the chunk, only if the table needs them: updates (partial
   update must know pinned/side-table values it does not write), TypeDelta (stored version and
   pinned values), any side table (entry diff), DataFrames (before values), cache groups (groups the
   record leaves). One statement:
   ```sql
   SELECT t.<needed columns> FROM AS_TABLE($keys) AS k
   INNER JOIN `genix/products` AS t ON t.company_id = k.company_id AND t.id = k.id;
   ```
   A pure insert into a table with none of the above skips it. These reads take optimistic locks:
   a concurrent writer of the same rows aborts this transaction and the SDK retries it.
3. **Main rows.** `UPSERT INTO \`table\` SELECT * FROM AS_TABLE($rows);` where `$rows` has **only
   the written columns** (plus keys and managed/virtual columns). YDB `UPSERT` leaves unlisted
   columns untouched and creates a missing row — exactly Scylla's `UPDATE` upsert semantics. Use
   one `$rows` param per distinct written-column set (insert phase vs update phase). Never `REPLACE`.
4. **TypeDelta.** Stamp `version` (stored + 1, wrap per `cache.MaxVersion`), compute the packed
   delta column from the **final** pinned values (stored value when the update does not write the
   column), as Scylla's `stampDeltaWrites` does. A record whose pinned prefix changed → one row in
   `__delta_moves` under the old prefix.
5. **Slice-index side tables.** For each record, old element set (from the pre-read) vs new:
   `DELETE FROM \`side\` ON SELECT * FROM AS_TABLE($removed);` and
   `UPSERT INTO \`side\` SELECT * FROM AS_TABLE($added);`. Unchanged elements are not written. An
   order-column change moves every entry of that record (delete + insert).
6. **Cache-group marks.** For every group the record enters or leaves: `UPSERT` into
   `__index_updated` with `last_write = now` (ORM clock), one row per `(index_id, index_hash)`,
   deduplicated per chunk.
7. **DataFrame log.** The `__dtf_log` entry computed exactly as Scylla's `writeDataFrameLog`
   (before values of changed columns, or the insert flag), `expire_at = now + 48 h`.
8. **Commit.** Last statement with commit. YDB then updates the async indexes in the background.

**After the commit** (same order as Scylla): `cache.RaiseWrittenSlots` for by-IDs tables, then the
text-search sync through `db.TextSearchBackend`.

**Merge.** Steps 1–8 with `onUpdate(previous, current)` / `onInsert` called on the pre-read rows
inside the callback. On a retry they run again on the re-read rows: the callbacks must be pure
(already the documented contract).

**Bulk restore / seeding.** `BulkUpsert` (non-transactional, fast) only for tables with **no**
side tables, delta, cache groups or DataFrames; everything else uses the transactional path.
Step 0 checks that BulkUpsert maintains async indexes in 26.3; if it does not, never use it.

**Ordering note.** Side tables and marks commit with the row; async indexes lag behind it. A reader
can therefore see a cache-group mark or a delta move before the index reflects the row. That is the
same window Scylla has (marks published before the write, views lagging), and the 16 s settle
windows of both protocols cover it, provided the measured lag fits (§16 Step 0, §18).

---

## 10. Read algorithm (`read.go`)

1. Shared planner (Step P) → `AccessPlan{Source, Spans, PostFilter, Limit, Descending}`. A query
   the planner refuses fails exactly as on Scylla.
2. Render by source. Statement text is cached per `computeSelectShapeHash`; values are params.

| Source | YQL | Tx mode |
|---|---|---|
| Primary key | `SELECT <proj> FROM \`t\` WHERE company_id = $p AND id >= $from AND id < $to ORDER BY company_id, id [DESC] LIMIT $n` | Snapshot RO |
| Primary key, IN on keys | `SELECT <proj> FROM AS_TABLE($keys) AS k INNER JOIN \`t\` AS t ON <full pk>` | Snapshot RO |
| Native async index | `SELECT <proj> FROM \`t\` VIEW ix_<name> WHERE <span on index columns> ORDER BY <index columns> [DESC] LIMIT $n` | **Stale RO** |
| Slice-index side table | §10.1 | Snapshot RO |

3. **Spans.** Equality fan-out over one column → one statement with `col IN $values` (YDB turns it
   into point lookups). Several range spans → one statement per span inside **one** read-only
   transaction (Snapshot RO) or one stale call each (Stale RO), executed concurrently, merged in the
   requested order. Never concatenate spans into the query text (10 KB limit).
4. **Projection.** If every projected column is in the index key, primary key or `COVER`, YDB
   answers from the index; otherwise it looks up the main table itself. No join in our SQL.
5. **Paging.** Keyset pagination on the source's key order (`> last key`), never `OFFSET`. Stream
   result sets.
6. **Post-filter, dedupe, merge, `Limit`**: shared planner functions.
7. **Decode** only the projected columns through §7.2.

### 10.1 Slice-index side table reads

Default — one statement, one snapshot, lookup join on the full primary key:

```sql
$ids = SELECT company_id, id FROM `genix/products__sx_<source>`
       WHERE company_id = $p AND element IN $elements
       [ORDER BY <order columns> LIMIT $n];
SELECT t.<proj> FROM $ids AS i
INNER JOIN `genix/products` AS t ON t.company_id = i.company_id AND t.id = i.id;
```

- One element: no dedupe needed.
- Several elements with `Limit`/order: read the entries per element (each already sorted by the
  side-table key), merge them in Go, drop repeated primary keys while merging (Y9), cut to `Limit`,
  then fetch the records with the `AS_TABLE($keys)` join, in the same snapshot transaction.
- Step 0 measures both forms; keep the default unless the numbers say otherwise.

---

## 11. Cache protocols

| Protocol | YDB |
|---|---|
| **Delta** (`TypeDelta`, `db/cache/delta_window.go`) | Per pinned prefix: `SELECT … FROM \`t\` VIEW ix_<delta> WHERE company_id = $p AND zz_dlt_x >= $from AND zz_dlt_x <= $to ORDER BY zz_dlt_x DESC LIMIT $n` (Stale RO). The window hash reads `(id, version)` from the same view. Moves: `SELECT id FROM \`t__delta_moves\` WHERE partition_id = $p AND index_tag = $tag AND pinned_prefix = $prefix AND updated >= $from` (Snapshot RO). `cache.PlanDeltaSync`, replace threshold, `Limit`, projection and the Metadata record are reused unchanged. Prefix resolution and FixedValues fan-out come from the shared planner (`db.PlanDeltaPinnedSlots`, Step P) |
| **Delta snapshots** | Unchanged file-store logic (`db/cache/delta_snapshot.go`); port the Scylla call sites (`scylla/delta_snapshot.go`) |
| **By-IDs** (`CacheByIDs`, `QueryCachedIDs`) | Slot files unchanged. Records by primary key (`AS_TABLE($keys)` join, Snapshot RO). `cache.RaiseWrittenSlots` after each commit |
| **Generic by-IDs** (`QueryCachedGenericByIDs`) | Resolve the table through the registry, assert `YdbTable` (report a table compiled by another driver clearly, as Scylla does), read `GenericRecordSchema` columns by key |
| **Cache groups** (`TypeCacheGroup`, `FromKey`, `S3`), `QueryIndexGroup`, `SelectGrouped` | Marks from `__index_updated` (Snapshot RO). Group reads: hash index (Stale RO) or slice side table (§10.1); `FromKey` = primary-key range. Aggregation in the driver with the shared group-by plan. Group snapshots in S3 / file store unchanged |

---

## 12. Controller and admin (`controller.go`, `admin.go`, `sequences.go`)

| Operation | YDB |
|---|---|
| `GetRecords`, `GetRecordsGob` | Primary-key paging per partition |
| `ExportRecordsColbin` | Stream the partition by primary key in `batchSize` pages, colbin-encode each batch |
| `RestoreRecordsColbin` | `deletePartitionFirst` → `WipePartition`; then the write path (or BulkUpsert per §9) |
| `ReloadRecords` | Read the partition, rewrite every row through the write path (recomputes virtual columns and side tables) |
| `RecalcVirtualColumns` | Same as reload, writing only virtual columns + side tables |
| `RecalcGroupIndexHashes` | Rewrite hash columns, slice side tables of cache groups, and marks |
| `ResetCounter` | `_sequences` |
| `DeleteViewsAndIndexes` | Drop every `ix_*` index and every `<table>__sx_*` side table (mirror what the Scylla controller drops; read `scylla/controller_generic.go` first) |
| Text-search maintenance (`ReindexTextSearch*`, `CompactTextSearch`, `DropTextSearch`) | Same `db.TextSearchBackend` calls as Scylla; the rebuild scans the partition by primary key |
| `CompactDataFrames`, `RebuildDataFramesPartition` | `dataframe.Backend` + `PartitionRecordScanner` over primary-key scans; state stays in the file store; frame lock stays on fareward (`SetDataFrameLock`) |
| `DescribeTable`, `QueryRecordsJSON`, `InsertRecordsJSON`, `UpdateRecordsJSON` | As `FdbController` does, on the YDB read/write paths |
| `WipePartition(table, partition)` | `BATCH DELETE FROM \`t\` WHERE company_id = $p` (26.x non-transactional batch delete) on the table and each `<table>__*` side table; by-IDs slot cleanup as `facade_scylla.go` does. Verify `BATCH DELETE` on tables with async indexes in Step 0; otherwise delete in keyed chunks |
| `RenameColumns(apply)` | YDB has no column rename: `ADD COLUMN new`, `BATCH UPDATE \`t\` SET new = old`, `DROP COLUMN old`, per pair from the same catalog `column_rename_scylla.go` uses; report-only when `apply` is false |
| Sequences: `GetAutoincrementID(key, n)` | One serializable tx: `SELECT value FROM _sequences WHERE name = $n`; `UPSERT … value + $count`; return `previous + 1` |
| `SetCounter(name, value)` | Same tx, returns the previous value |
| `ListCounters` | `SELECT name FROM _sequences` |

---

## 13. Backend facade (`backend/db`, Step 1)

- `driver_ydb.go`: `type TableStruct[…] = ydbdriver.TableStruct[TableT, RecordT]`; `SetDataFrameLock`
  forwarding to the driver.
- `facade_ydb.go`: same signatures as `facade_fdb.go`: `Connect`, `Deploy`, `MakeController`,
  `ListCounters`, `WipePartition`, `SetCounter`, `BypassFarewardSequences` (no-op: YDB owns
  sequences), `RenameColumns`.
- `autoincrement_ydb.go`: wires `db.GetAutoincrementID` to `ydbdriver`.
- `connect.go`: `ConnectParams` gains `YDBEndpoint`, `YDBDatabase`.
- Build tags: `autoincrement_scylla.go`, `facade_scylla.go`, `driver_scylla.go`,
  `column_rename_scylla.go`, `column_rename_scylla_test.go` → `//go:build !js && !fdb && !ydb`;
  `exec/generic_records_test.go` → `//go:build !fdb && !ydb`. Re-run
  `rg -n "go:build" backend --type go -g '!genix-orm/**'` to catch any file added since.
  `-tags fdb,ydb` together fails to compile on duplicate declarations; that is acceptable.
- `core/security.go`: `[ydb] endpoint`, `database` → `core.Env.YDB_ENDPOINT`, `core.Env.YDB_DATABASE`,
  next to the `[fdb]` section. `config.example.toml`: a `[ydb]` section in the same style
  (comment in Spanish like the rest of that file, values in English).
- `tests/bench/setup.go`: pass the two fields in `db.Connect`.
- `backend/go.mod`: `require github.com/ivanjoz/genix-orm/ydbdriver v0.0.0` +
  `replace … => ./genix-orm/ydbdriver`.

---

## 14. fareward (Step 8)

fareward (Rust, `fareward/` submodule) writes and reads **ORM tables** that the backend declares
(`core/types/credit_usage.go`, `server_metrics.go`, `exec/observability_schema.go`, and `users`).
Under `-tags ydb` those tables are created by the ORM's YDB deploy, so fareward must read and write
them **in exactly the shape §7–§9 define**.

- Cargo feature `ydb` with `ydb = "0.18.2"`; the Scylla store stays the default build.
- The storage traits already exist (`limiter/storage.rs`: `load_exact`, `load_range`,
  `load_budget`, `load_user_access`; `reqlog/writer.rs` `flush`; `sysmetrics/writer.rs`
  `write_window`). Add a YDB implementation of each behind the feature.
- `users.accesos_computed` / `accesos_sub_computed` are slices → colbin `String` columns (Y10).
  fareward already depends on the `colbin` crate (`bridge/token.rs`): decode with it.
- `sequence/store.rs` is **not** ported (Y11): under YDB the ORM never asks fareward for IDs.
- Before writing: check whether any table fareward writes has managed columns, virtual columns,
  side tables, by-IDs or cache groups. If one does, fareward would have to maintain them: **stop and
  ask** rather than reimplementing ORM write logic in Rust.
- **Accept:** `cargo test --features ydb` green; bench Layer B and the `FarewardCharge` benchmarks
  run under `-tags ydb`.

---

## 15. Feature parity checklist

Each row is done when its conformance test (Step 3) passes on YDB.

| Feature | YDB |
|---|---|
| Keys, partition, `KeyConcatenated`, `KeyIntPacking`, `AutoincrementPart/From` | Composite PK; shared prepare code |
| Autoincrement, `UseSequences`, `SequenceColumn`, `CounterColumn`, `GetAutoincrementID`, `ResetCounter` | `_sequences` |
| Managed columns, `DisableDefaultColumns` | Real columns |
| `TypeLocalIndex`, packed index | `GLOBAL ASYNC` on (partition, col / packed) |
| `TypeGlobalIndex` scalar | `GLOBAL ASYNC` on (col) |
| Any index on a slice column | Key-only side table, lookup join |
| `TypeView` hash / range / `Cols` | `GLOBAL ASYNC … COVER` |
| `TypeViewTable` | Key-only side table (not used by any genix table today; still implement) |
| FixedValues fan-out, composite buckets | Shared planner; side table for slice buckets |
| `Computed` | Real column when an index keys on it; otherwise not stored (as Scylla) |
| `TypeDelta` (+ moves, snapshots) | Covering async index + `__delta_moves` |
| `CacheByIDs`, generic by-IDs | PK lookups; slot files unchanged |
| `TypeCacheGroup*`, `QueryIndexGroup`, snapshots | `__index_updated` + hash index / side table |
| `SelectGrouped` | Driver-side aggregation |
| `Merge`, `InsertUpdate*`, include/exclude | §9, serializable tx per chunk |
| `TextSearch` | Existing backend interface |
| `DataFrames`, frame log, `AcquireFrameLock` | `__dtf_log` with TTL; lock on fareward |
| `db.Controller` | `YdbController` |
| Deploy / homologation | Real DDL, §8.6 |
| Wipe, counters, column rename | `BATCH DELETE`; `_sequences`; add + copy + drop |

---

## 16. Steps

Each step ends with: its acceptance check, `go vet`, a commit **and push** inside `genix-orm` (and
`fareward` in Step 8), and a root commit for `backend/` / `docs/` changes. Update this plan's
status lines and §18b as you go.

### Step 0 — Container and SDK spike (no ORM code)
1. §5 container; `ydbdriver/go.mod`; `driver.go` with `Connect` / `Ping`.
2. `ydbdriver/spike_it_test.go` (gated `GENIX_YDB_IT=1`) verifying, each as its own test, and
   recording the result in §18b:
   - `CREATE TABLE … INDEX … GLOBAL ASYNC ON (…) COVER (…)` is accepted. **If not: stop and ask**
     (TypeDelta and covering views depend on it).
   - `SELECT … VIEW ix` under Stale RO returns covered columns; fails or not under Snapshot RO.
   - `ORDER BY … DESC LIMIT` on an index is a reverse read (check the plan via `EXPLAIN`), not a sort.
   - `UPSERT` of a subset of columns leaves the others untouched and creates missing rows.
   - `AS_TABLE($keys) INNER JOIN` on the full PK is a lookup join (`EXPLAIN`).
   - `BulkUpsert` on a table with an async index: is the index updated?
   - `BATCH DELETE` / `BATCH UPDATE` on a table with async indexes.
   - Transaction size: 1k / 5k / 20k row writes in one tx — time, success, error text.
   - TTL on `Uint32 … AS SECONDS`.
   - `FAMILY default (COMPRESSION = "lz4")` is accepted at create, `ALTER FAMILY … SET COMPRESSION`
     at homologation, and `DescribeTable` reports the family as `COMPRESSION_LZ4`.
   - Y20: the five partitioning settings at `CREATE TABLE`; `ALTER INDEX … SET (…)` on an async
     index; `DescribeTable` reports them back for the table and for the index. Write a single
     company's rows with increasing IDs at full speed and record how long load splitting takes to
     split the hot partition.
   - **Async index lag**: write at fixed rates (concurrency 1 and 64), poll the index until each row
     appears; record p50 / p99 / p99.9 / max. **If p99.9 is not comfortably under 16 s: stop and
     ask** (the window is a shared protocol constant).
3. Microbenchmark (`GENIX_YDB_BENCH=1`), same workloads as FDB plan §14: 1 point get, a 1,000-row
   range, a 1,000-row commit, concurrency 1 and 64.
- **Accept:** container starts clean from an empty volume; every item above recorded in §18b.

### Step 1 — Facade and build tags (§13)
- **Accept:** in `backend/`: `go build ./...`, `go build -tags fdb ./...`, `go build -tags ydb ./...`,
  `GOOS=js GOARCH=wasm go build .`, and `go vet` for the three tag sets. Under `-tags ydb` every
  driver call may still answer `errNotImplemented`.

### Step P — Prerequisite extraction (§4)
- Owned by the FDB plan. **Accept:** FDB plan Step 2 acceptance.

### Step 2 — Compile, DDL, deploy (ask D5 first)
1. Print the native index count per genix table (a temporary test or a `fn-` script), ask D5.
2. `table.go`, `index_mapping.go`, `ddl.go` (+ golden tests), `deploy.go`, `codec.go`, `values.go`.
3. `CompileTable` must compile **every** table genix declares without panicking, except the
   deliberate D5 / Y8 panics. If one of those fires on a genix table, stop and ask.
- **Accept:** `go run -tags ydb . fn-init` twice against the container: first creates everything,
  second reports no change. Tables, indexes and side tables visible in the UI (`:8765`).

### Step 3 — Conformance suite
- Shared with FDB plan Step 3: package `genix-orm/conformance` (own module), one set of test table
  declarations covering every row of §15, run per driver through `.Via(executor)`, results compared
  sorted and field by field, cache responses decoded and compared with the clock frozen via
  `db.SetHistoricalUnix`. Gated by `GENIX_CONFORMANCE=1`. If it does not exist yet, build it here.
- Seed it from the Scylla integration tests (`TestCacheProtocolsOnScylla`,
  `TestControllerPathsOnScylla`, `TestMergeOnScylla`, `TestQueryIndexGroupOnScylla`,
  `TestTextSearchOnScylla`, `TestInsertUpdateIncludeOnScylla`, `cache_integration_test.go`).
- Async indexes: a conformance read through an index after a write must wait (poll with a timeout)
  rather than sleep a fixed time.
- **Accept:** Scylla vs Scylla green; YDB fails only with `errNotImplemented`.

### Step 4 — Writes, sequences, primary-key reads
- §9 (all write methods, `Merge`), `_sequences`, §10 primary-key source, managed and virtual
  columns, computed columns, key packing/concatenation.
- **Accept:** conformance rows for keys, sequences, writes, merge, computed columns green.

### Step 5 — Index reads
- Native async index reads, slice side tables (§8.4 writes, §10.1 reads), FixedValues fan-out,
  composite buckets, post-filter, paging.
- **Accept:** conformance rows for every index kind green;
  `GENIX_BENCH=1 go test -tags ydb ./tests/bench/ -run TestSeedBenchData` passes.

### Step 6 — Cache protocols (§11)
- **Accept:** their conformance rows green; the bench endpoints return the same decoded bytes as
  Scylla for a first and an incremental sync (after the settle window).

### Milestone — Benchmark
- `bench.config.toml` gains `[ydb]`; README gains the YDB container commands next to Scylla and FDB.
- Layer A with no tag, `-tags fdb` and `-tags ydb`: same CPU pinning, same datasets,
  `GENIX_BENCH_NO_FAREWARD=1`, `GENIX_BENCH_NO_SNAPSHOTS=1`. Layer B after Step 8.
- **YDB warm-up (Y20):** after seeding, run the `HandlerConcurrent` load for a few minutes (load
  splits need a partition above 50% CPU for several tens of seconds), then poll partition counts
  (`DescribeTable` with partition stats, or the UI) until they stop changing for one minute. Only
  then measure. Record the partition count of each benchmarked table and index before measuring.
- Also record: async index lag under the bench load, transaction abort rate, storage size per table
  against Scylla's zstd-3 size for the same dataset (Y19: lz4 vs zstd; covering delta indexes double
  delta tables).
- Fill §18b in the FDB plan §14 format.

### Step 7 — Remaining parity
- `SelectGrouped`, text search, DataFrames + frame log, the full `YdbController`, `WipePartition`,
  `RenameColumns`, every `exec/` script under `-tags ydb`.
- **Accept:** every §15 row green; `go test -tags ydb ./...` in `backend/` green.

### Step 8 — fareward YDB store (§14)

### Step 9 — Documentation
- `ydbdriver/README.md` (storage model, mapping, limits, container), `ydbdriver/RATIONALE.md`
  (Context / Decision / Rationale per Y-decision, like `scylla/RATIONALE.md`), `INTERNALS.md` §10
  (third driver, conformance suite), root `CLAUDE.md` repository map, `config.example.toml`.

---

## 17. Testing rules

- Unit tests next to the code, no database: type mapping, codec round trip for every ORM type ID,
  DDL golden strings, index mapping per `IndexSource.Type`, side-table entry diff (added / removed /
  order-column move), chunk sizing by row writes, AccessPlan → YQL rendering, Y8 / D5 panics.
- Integration tests gated by `GENIX_YDB_IT=1` against the §5 container, each on a throwaway
  keyspace directory that the test drops at the end.
- Conformance gated by `GENIX_CONFORMANCE=1`; benchmarks by `GENIX_YDB_BENCH=1` / `GENIX_BENCH=1`.
- Tests skip, never fail, when their env var is absent.

---

## 18. Risks

| Risk | Mitigation |
|---|---|
| Async index lag exceeds the 16 s settle window under load | Step 0 and the benchmark measure it; stop and ask before touching the shared window |
| `GLOBAL ASYNC … COVER` unsupported in 26.3 | Step 0 first item; stop and ask |
| YQL compile + gRPC per statement dominates small reads | Parameterized statements (server query cache), Step 0 numbers; `OnlineReadOnly` for point reads if faster |
| Optimistic-lock aborts on hot rows (`_sequences`, `__index_updated` marks) | Reserve IDs once per chunk outside the tx; dedupe marks per chunk; measure abort rate |
| 20 indexes per table | D5 |
| Inserts with increasing IDs always hit the last partition of a company (same as Scylla's one partition per company); the delta index (keyed by `updated`) likewise | Load splitting (Y20) splits the hot range; Step 0 measures how fast; the benchmark warms up first |
| An index table keeps YDB's defaults (load splitting off) if `ALTER INDEX … SET` is skipped | Deploy applies and homologates it on every run (§8.6); Step 0 verifies it reads back |
| Covering delta indexes double the storage of delta tables | Accepted (FDB plan §14 evidence on row copies); measured |
| Large slices write many side-table rows | Chunk by row writes; element diff on update |
| A large insert is not atomic across chunks | Same as Scylla and FDB; documented in RATIONALE |
| Dependence on Step P | Y13: YDB Step 2 waits; Steps 0–1 run meanwhile |
| fareward writing ORM tables with derived state | §14 check; stop and ask |
| 26.3 patch releases change behaviour | Pin `26.3.1.21`; bump deliberately and rerun conformance |

---

## 18b. Results

### Step 0 (2026-11-06) — container `local-ydb:26.3.1.21`, SDK v3.156.1, laptop i7-1355U

Raw log lines are `SPIKE-RESULT` in `ydbdriver/spike_*_it_test.go` output. Container pinned to cores 2,3
(client on 4–11).

| Item | Result |
|---|---|
| `INDEX … GLOBAL ASYNC ON (…) COVER (…)` | Accepted |
| `VIEW ix` read modes | StaleRO works. SnapshotRO and the default serializable mode fail: `Read operation can be performed on async index table … only with StaleRO isolation level` (Y6 confirmed) |
| `ORDER BY updated DESC LIMIT 2` on the index | Reverse range read with `ReadLimit`, no sort node (EXPLAIN: `TableRangeScan … Reverse:true`) |
| `UPSERT` of a column subset | Other columns untouched; missing row created with NULLs |
| `AS_TABLE($keys) INNER JOIN` on the full PK | `TableLookupJoin` / `InnerJoin (Lookup)`, `lookup_by [company_id,id]` |
| `BulkUpsert` on a table with an async index | Accepted; index caught up in 74 ms |
| `BATCH UPDATE` / `BATCH DELETE` with an async index | Both accepted; row counts correct |
| One transaction, 1k / 5k / 20k rows | 26 ms / 51 ms / 197 ms, all succeeded |
| TTL `Interval("PT0S") ON col AS SECONDS` on `Uint32` | Accepted; DescribeTable returns it |
| `FAMILY default (COMPRESSION = "lz4")` | Accepted at CREATE; `ALTER FAMILY default SET COMPRESSION "lz4"` accepted; DescribeTable reports `lz4`. Default is `none` |
| Y20 settings | Accepted at CREATE and read back (`BySize=on, 2000 MB, ByLoad=on, min 1, max 50`). An index table starts with `ByLoad=off, 2048 MB, max 0`; `ALTER TABLE … ALTER INDEX ix SET (…)` is accepted and reads back the five values (describe the index at `<table>/<ix>/indexImplTable`) |
| Hot-partition split (1 company, rising IDs, 32 writers × 200-row batches, 20.8 M rows, 6 min) | 1 → 2 partitions at ≈2 min, 3 at ≈3m50, 4 at ≈5m40: load splitting works but is slow, so the benchmark warm-up must run several minutes |
| Async index lag (write ack → row visible by index poll, 20 ms poll) | conc. 1 (600 rows): p50 23 ms, p99 37 ms, p99.9 42 ms, max 981 ms. conc. 64 (19,200 rows): p50 680 ms, p99 1.03 s, p99.9 1.47 s, max 2.99 s. Far under the 16 s window; no stop condition |
| Restart | `podman restart` back in ≈5 s; a table and its row survive it |

Microbenchmark (table with one async covering index; **statements inline their literals, so every
statement is compiled; the plan's parameterized form is not measured here**, and the container was
still compacting the 20 M-row test just before): point get c1 154 ops/s p50 6.4 ms, c64 249 ops/s p50
256 ms; 1,000-row range c1 84 ops/s p50 11.6 ms, c64 916 ops/s p50 54 ms; 1,000-row commit c1 37 ops/s
p50 24 ms, c64 25 ops/s p50 1.09 s p99 14.4 s (64 writers on one company's key range = one hot
partition). Redo with parameterized statements before drawing conclusions (Milestone).

### Corrections found
- `podman run --cpuset-cpus` fails here (cpuset controller not delegated to the rootless user
  cgroup). Start without it and pin with `taskset -acp 2,3 <host pids from podman top … hpid>`, as the
  FDB container is. Re-pin after every restart.
- `ydbdriver/go.mod` requires `genix-orm/db` and `genix-orm/dataframe` (with replaces), not the root
  `genix-orm` module (§3.1).
- No `autoincrement_ydb.go`: `ydbdriver`'s `init()` sets `db.GetAutoincrementID`, as `fdbdriver` does (§13).
- `COUNT(*)` is `Uint64`; cast it for an `Int64` scan.

### Done ahead of Step P (2026-11-06)
These do not depend on the neutral compile entry point, so they landed while Step 2 waits:
- `ddl.go` (+ golden tests, + `ddl_it_test.go` executing every generated statement on the container):
  main table, side tables, `_sequences`, the Y19/Y20 settings, `ALTER INDEX … SET`, add column, and the
  D5 / Y8 panics. Input is driver-owned `tableSpec`; Step 2's `index_mapping.go` fills it from `IndexSource`.
- `sequences.go` (§12): `GetAutoincrementID`, `SetCounter`, `ListCounters` on `_sequences`, serializable
  `DoTx`. Integration test: 32 writers × 10 reservations, 320 disjoint ranges.

### Step P progress (2026-11-06)
Owned by the FDB plan (its Step 2, stage 2b; mechanism decided there: two-phase describe/materialize).
`db` now describes view tables, TypeView (simple, hash, packed range) and TypeDelta. Still open:
cache-group and composite-bucket registration, the table-level compile in `makeTable`, then the
`AccessPlan` planner (stage 3) and write preparation. YDB Step 2 starts when those land.

### Step 1 (2026-11-06)
`go build` with no tag, `-tags fdb`, `-tags ydb` and `GOOS=js GOARCH=wasm`, and `go vet` for the three
tag sets (`db`, `core`, `exec` for ydb): all green. `ydbdriver` answers `errNotImplemented`; its
`CompileTable` panics until Step P.

## 19. Out of scope

Lambda; YDB column (OLAP) tables; native full-text, vector and JSON indexes (the JSON index needs a
single-integer primary key and is always synchronous); `GLOBAL SYNC` indexes (an opt-in per index
would need a new `Index` option — ask first); changefeeds (a later option to replace delta moves and
cache-group marks); posting-list side tables (Y18); multi-node cluster; migrating Scylla data
(pre-alpha: wipe and reseed).
