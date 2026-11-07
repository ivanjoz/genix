# OceanBase driver — implementation plan (full parity with ScyllaDB)

**Audience.** An implementing agent with no prior context. Read §0–§2 before writing any code. Work
the steps of §16 in order; every step has an acceptance check, and a step does not start before the
previous one passes. When this plan and the code disagree, the code wins: stop, report the
difference, and ask.

**Goal.** OceanBase CE 5.0 (MySQL mode) replaces ScyllaDB behind **one build tag**
(`-tags oceanbase`), plug and play like the FoundationDB and YDB drivers: same `db.Executor`
contract, same `TableSchema` / `Index` declarations, same queries, same cache wire protocols
(genix-ui unchanged). **No table declaration and no query in `backend/` changes.** The shared
conformance suite (`genix-orm/conformance`) proves that Scylla and OceanBase return the same results.

**Status, 2026-11-06.** Plan written. Design decisions O1–O18 taken with the user; the Step 0
probes are in §18b. D1 decided. Step 0 done (§18b); D2–D5 decided after it (§2), with the TTL purge
verified (D4). Steps 1, 2, 3 and 4 done (§18b). Next: Step 5 (index reads: shared planner, hints,
paging).

---

## 0. Read first

| File | Why |
|---|---|
| `CLAUDE.md` (repo root) | Operating mode (supervised), code rules, submodule rules, conventions |
| `docs/YDB_PLAN.md` | **The sister plan.** Same structure, same shared code (Step P is done), same conformance suite. This plan only states where OceanBase differs; when a section here says "as YDB", read that section there |
| `backend/genix-orm/INTERNALS.md` §2–§4, §7, §10 | Driver contract: `Executor`, `Codec`, `DBTypeResolver`, the init hooks, `TableCore`, "adding a driver" |
| `backend/genix-orm/scylla/ORM_INTERNALS.md`, `scylla/RATIONALE.md` | What every index type, view, cache protocol and write phase means. **Parity = these semantics** |
| `backend/genix-orm/db/schema.go`, `db/index_source.go`, `db/describe_index.go`, `db/compiled_state.go` | `Index`, index types, `TableCore`, `IndexSource` and the neutral compiled state every driver materializes |
| `backend/genix-orm/db/access_plan.go`, `db/write_prepare.go` | The shared planner (`AccessPlan`, route choice, delta and cache-group planning) and the shared write preparation. The driver only renders and executes them |
| `backend/genix-orm/db/coltype.go` | Stable ORM type IDs. Never renumber |
| `backend/genix-orm/db/cache/*.USER.md`, `db/cache/RATIONALE.md` | The three cache protocols |
| `backend/genix-orm/ydbdriver/*.go` | The closest driver. Copy its file split, `tableSpec` → DDL → deploy shape, `sequences.go` and its tests, not its YQL |
| `backend/genix-orm/conformance/` | The shared suite; this driver adds `driver_oceanbase_test.go` |
| `backend/db/driver_ydb.go`, `facade_ydb.go` | How a build tag swaps the driver in the backend facade |

**Rules that apply** (from `CLAUDE.md`): `backend/genix-orm` is a git submodule (commit **and
push** inside it); pre-alpha, no backwards compatibility, wipe and reseed; English everywhere;
expressive grepable names; no more code than necessary; supervised mode (anything not settled in
§1–§2 → stop and ask, all questions at once); ORM managed clock, never `time.Now()` for persisted
values; `Read` / `Edit` tools, `rg`, `fd`; report honestly.

---

## 1. Settled decisions

| # | Decision | Choice |
|---|---|---|
| O1 | The switch | Build tag `oceanbase`. `backend/db/{driver,facade}_oceanbase.go` are `//go:build oceanbase && !js`. Every file tagged `!js && !fdb && !ydb` today gets `&& !oceanbase`. A runtime switch is impossible: the driver is a type argument of `TableStruct` |
| O2 | Declarations | Unchanged. Hints with no OceanBase meaning (`set`/`list`/`frozen` db tags, `Int32()` on a view, `UseListAsDefault`) are accepted and ignored |
| O3 | Server | **OceanBase CE 5.0.1.0**, MySQL-compatible tenant `test`, native install (no container) on the `homelab` node, reached over Tailscale at `100.64.0.4:2881`, user `root@test`. Mini mode, 8 GB |
| O4 | Go client | **`github.com/go-sql-driver/mysql` v1.10.1** through `database/sql`. Pure Go: the backend stays `CGO_ENABLED=0`. Server-side prepared statements (no `interpolateParams`) |
| O5 | Module | `genix-orm/obdriver`, **its own Go module** like `ydbdriver`, so the default build does not carry the MySQL client |
| O6 | Storage model | **Native relational tables**: one OceanBase table per ORM table, typed columns, **synchronous** secondary indexes. Every index kind maps to a native index; the only ORM-maintained side tables are the three Scylla already has (`__delta_moves`, `__index_updated`, `__dtf_log`) |
| O7 | Company first | **`company_id` (the table's partition column) is the first column of the primary key and of every index**, except indexes that are global by declaration (`TypeGlobalIndex`, an `Index.Partition` override). A company's rows are one contiguous key range in the table and in each index, so a company-scoped read is a bounded range scan (§18b) |
| O8 | Physical partitioning | **Configurable**, `[oceanbase] partitions`, **default 1 = unpartitioned**. With N > 1: `PARTITION BY KEY(company_id) PARTITIONS N`. An OceanBase partition is a physical tablet (like a Scylla tablet), not a free logical key like a Scylla partition, so it is not the unit of tenant isolation (O7 is). Step 0 measures tenant memory and deploy time for N = 1 and N = 8 over every genix table; the user then fixes the default |
| O9 | Local vs global index | An index whose first column is the table's partition column → `LOCAL`. Any other index → `GLOBAL`. Same rule for N = 1 and N > 1, so changing N never changes the DDL of indexes |
| O10 | Covering | Views (`TypeView`, its `Cols`) and TypeDelta indexes carry their projection with **`STORING (…)`**, so the read never goes back to the main table, **only when every projected column is storable** (D2: no LOB / JSON). Otherwise they are plain indexes |
| O11 | Slice columns | **JSON only for slices that have an index** (user decision). A slice column that any `IndexSource` keys on (including the `zz_igs_*` group-hash sets of cache groups over a slice) is a `JSON` array with a **multi-value index** `(company_id, CAST(col->'$' AS <type> ARRAY), <order columns…>)`. Every other slice is colbin bytes in a `LONGBLOB` (same bytes as the Scylla codec). Indexed **string** slices are refused (D3) |
| O12 | Slice index reads | `<v> MEMBER OF (col->'$')` for one element, `JSON_OVERLAPS(col->'$', ?)` for several. The engine dedupes a record that matches several elements (§18b). No side tables for slices, no Go-side dedupe |
| O13 | Index hints | Every statement that reads through an index carries **`/*+ INDEX(t ix_<name>) */`** for the index the shared planner chose. The optimizer never picks the access path on its own: on small tables it chose a full primary-key scan over a matching multi-value index (§18b) |
| O14 | Sequences | **OceanBase owns the sequences** under `-tags oceanbase`: table `_sequences(name, value)`, one range reservation per transaction (`SELECT … FOR UPDATE` + `UPDATE`), as YDB's `_sequences`. Native `CREATE SEQUENCE` and `AUTO_INCREMENT` are not used; fareward is not asked for IDs |
| O15 | Compression | `COMPRESSION = 'zstd_1.3.8'` on every table, written explicitly (it is also the server default). Closest to Scylla's `ZstdCompressor` |
| O16 | Collation | **`utf8mb4` / `utf8mb4_bin`** on every table and string column, written explicitly. The server default `utf8mb4_general_ci` is case- and accent-insensitive; Scylla compares bytes, so any other collation breaks key equality, ordering and uniqueness parity |
| O17 | Text search, scope | Text search: the existing `db.TextSearchBackend` (search daemon), not OceanBase's full-text index. VPS mode only. Out of scope: §19 |
| O18 | One multi-value index per JSON column | OceanBase 5.0.1 uses **no** multi-value index of a JSON column that has two of them (§18b). `CompileTable` **panics** when two `IndexSource`s key on the same slice column, naming the table, the column and both sources. Step 2 reports whether any genix table hits it (the `zz_igs_*` sets are one column per cache group, so they do not collide with each other); if one does, stop and ask |
---

## 2. Decisions taken late

**D1 — String column types (decided 2026-11-06, user).** MySQL needs a length on any string that is
part of a key or index; `TEXT` columns cannot be indexed without a prefix. Strings in the primary key
or in any index key → `VARCHAR(512)` (a longer write fails with an error, strict SQL mode, never
truncates); every other string → `LONGTEXT` (OceanBase keeps LOBs inline up to
`lob_inrow_threshold`, 4 KB by default). Step 0 confirmed it: wide `VARCHAR`/`VARBINARY` hit the
~1.5 MB declared row cap (§18b).

**Decided after Step 0 (2026-11-06, user):**
- **D2 — Covering with LOBs: cover only if every column fits.** `STORING` refuses LOB and JSON
  columns (§18b). A view or delta index gets `STORING (<projection>)` only when every column of its
  projection is storable (not `LONGTEXT` / `LONGBLOB` / `JSON`). Otherwise it is a plain index and
  reads look the row up in the table. So a TypeDelta index covers only on tables with no LOB or
  JSON column. Measured: with a LOB in the projection, covering and plain cost the same.
- **D3 — String slice indexes: refused.** The server defect on case variants inside one record
  (§18b) gives wrong answers for an indexed `[]string`. `CompileTable` panics on any index whose key
  includes a string slice, naming the table and column. Today's genix slice indexes are integer
  cache-group hashes (YDB plan §18b Step 2).
- **D4 — DataFrame log expiry: tenant TTL.** The purge needs **two** tenant parameters, both set on
  the `test` tenant on 2026-11-06 (`ALTER SYSTEM SET`): `enable_ttl = true` (alone it only runs a
  `COMPACTION` task that deletes nothing) and `enable_kv_ttl = true` (the row-deleting task of
  MySQL-mode `TTL (…)` tables). With both, a row 3 days past its TTL was deleted within 20 s of
  `ALTER SYSTEM TRIGGER TTL`, and a fresh row was kept. The scheduled run is daily at
  `ttl_duty_time` (01:00), so expired rows can stay up to a day past 48 h; readers already filter by
  `half_hour`. Deploy checks both parameters and fails with a clear message when either is off.
- **D5 — Default partition count: 1.** O8 stands.

---

## 3. Architecture

```
backend (CGO_ENABLED=0, -tags oceanbase)
  app tables ──► backend/db facade ──► genix-orm/obdriver (pure Go)
                                          │  database/sql + go-sql-driver/mysql, MySQL protocol
                                          ▼
homelab 100.64.0.4:2881 (Tailscale) ── OceanBase CE 5.0.1 mini, tenant `test` (MySQL mode)
```

### 3.1 Module and files

```
backend/genix-orm/obdriver/                 module github.com/ivanjoz/genix-orm/obdriver
  go.mod            require genix-orm/db, genix-orm/dataframe (replace => ../db, ../dataframe),
                    github.com/go-sql-driver/mysql v1.10.1, github.com/ivanjoz/colbin
  driver.go         Exec[TableT, RecordT] (zero-size), TableStruct alias, init() hooks,
                    errNotImplemented, connection (Connect, Ping, the shared *sql.DB)
  codec.go          db.Codec + DBTypeResolver (§7)
  values.go         record ⇄ SQL args (placeholder rows), row scanning into records
  table.go          ObTable{db.TableCore; …}, compile cache, CompileTable
  index_mapping.go  IndexSource → native index (§8.2)
  ddl.go            CREATE TABLE / ADD INDEX / side tables text; pure, golden-tested
  deploy.go         Deploy + homologation against information_schema (§8.5)
  write.go          §9 write algorithm
  side_tables.go    delta moves, cache-group marks, frame log rows
  read.go           §10 AccessPlan → SQL, execution, paging, decoding
  delta.go          TypeDelta reads (window rows, moves)
  cache_by_ids.go   QueryCachedIDs, QueryCachedGenericByIDs
  cache_groups.go   SelectGrouped, QueryIndexGroup, group snapshots
  data_frame_backend.go, text_search.go
  sequences.go      _sequences: reserve, set, list
  controller.go     ObController[RecordT, TableT] implementing db.Controller
  admin.go          WipePartition, RenameColumns, ListCounters, WipeDatabase (tests)
  server/README.md  how the homelab instance runs, connection, verified facts (§5)
  *_test.go         unit tests (no DB); *_it_test.go gated by GENIX_OCEANBASE_IT=1
backend/db/driver_oceanbase.go, facade_oceanbase.go
backend/go.mod      require + replace github.com/ivanjoz/genix-orm/obdriver => ./genix-orm/obdriver
backend/core/security.go   [oceanbase] section → core.Env.OCEANBASE_*
config.example.toml        [oceanbase] section
genix-orm/conformance/driver_oceanbase_test.go
```

### 3.2 Naming

| Object | OceanBase name |
|---|---|
| Database | the keyspace name, e.g. `genix` (`CREATE DATABASE` at deploy) |
| Main table | `<database>.<table>`, e.g. `genix.products` |
| Index | `ix_<IndexSource.Name>` |
| Delta moves / cache-group marks / DataFrame log | `<table>__delta_moves`, `<table>__index_updated`, `<table>__dtf_log` (as Scylla and YDB) |
| Sequences | `<database>._sequences` |

---

## 4. Prerequisites

Step P of the YDB plan (neutral compile, `AccessPlan` planner, shared write preparation) is done.
Nothing to extract. If a step here needs logic that lives only in `ydbdriver` (e.g. the delta-move
or frame-log row computation) and is not storage-specific, move it to `genix-orm/db` and use it
from both drivers; do not copy it.

---

## 5. Server (Step 0)

- Native OceanBase CE 5.0.1.0 on `homelab`, managed as a service there (installed outside this repo).
  `obdriver/server/README.md` records how it runs and is restarted, the connection string and the
  verified facts, like `ydbdriver/container/README.md`.
- DSN: `root@test:<password>@tcp(100.64.0.4:2881)/<database>?parseTime=false&multiStatements=false`.
- The tenant has **no password and listens on the LAN** today (reported 2026-11-06). Setting a password
  and restricting `ob_tcp_invited_nodes` to loopback + Tailscale is the user's call, on the homelab;
  the driver reads the password from config either way.
- The benchmark Milestone cannot pin the same cores as the Scylla/YDB containers (different host).
  Its numbers are recorded with the host spec and are not compared head to head without that caveat.

---

## 6. Client conventions (`database/sql` + `go-sql-driver/mysql` v1.10.1)

- **Connection.** One process-wide `*sql.DB`, opened lazily on first use (server) or eagerly when
  `ConnectParams.Scripting`. `SetMaxOpenConns` from config (default 64), `SetMaxIdleConns` equal.
- **Transactions.** `db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})` (the
  tenant default). Writes take row locks through `SELECT … FOR UPDATE` pre-reads, so concurrent
  writers of the same rows serialize instead of aborting. Retry the whole chunk on a deadlock or
  lock-wait timeout (MySQL errors 1213 / 1205 and their OceanBase equivalents, verified in Step 0)
  with bounded attempts; the chunk body is rebuilt from the re-read state.
- **Statements.** Always parameterized. Statement text is cached per shape
  (`computeSelectShapeHash` for reads, written-column set for writes) and prepared once per
  connection by `database/sql`. Multi-row statements are bounded by **65,535 placeholders** and
  `max_allowed_packet` (16 MB on the tenant): chunking (§9) respects both.
- **Results.** `rows.Next()` streaming; never `SELECT` a whole partition without keyset paging.
- **Errors.** `fmt.Errorf("obdriver: %s %s: %w", op, table, err)`.
- **DDL.** Outside transactions. Schema reads through `information_schema.COLUMNS`,
  `STATISTICS`, `TABLES`, `PARTITIONS`.

---

## 7. Types and codec

### 7.1 Type mapping (`DBTypeResolver`, keyed by ORM type ID from `db/coltype.go`)

| ORM type ID | Go | OceanBase column type |
|---|---|---|
| 1 | string | D1: `VARCHAR(512)` in a key or index, `LONGTEXT` otherwise; always `utf8mb4_bin` (O16) |
| 2 / 3 / 4 / 5 | int64 / int32 / int16 / int8 | `BIGINT` / `INT` / `SMALLINT` / `TINYINT` |
| 6 / 7 | float32 / float64 | `FLOAT` / `DOUBLE` |
| 8 | bool | `BOOL` (`TINYINT(1)`) |
| 9 (`TypeBlob`) | `[]byte` and anything without a native mapping | `LONGBLOB` |
| 11–17 | `[]string`, `[]int64` … `[]float64` | **not indexed**: `LONGBLOB` (colbin). **Indexed**: `JSON` (O11) |
| 21–27 | pointers to scalars | same as the scalar (nullable) |
| 31–35 | pointers to slices | as 11–17 |
| unsigned Go kinds | uint8 … uint64 | `TINYINT UNSIGNED` … `BIGINT UNSIGNED` (native, full range verified) |
| packed / hash / delta virtual columns | computed by `db` | `BIGINT UNSIGNED` (or `INT UNSIGNED` when `PackedIsInt32`): the unsigned layout, as YDB. Scylla's sign flip is **not** used |
| `zz_igs_*` group-hash set | `[]int32` | `JSON` with a multi-value index (O11) |

- Primary-key and partition columns are `NOT NULL`; every other column is nullable and NULL decodes
  to the zero value, as on Scylla.
- Managed columns (`created`, `updated`, `version`) and every virtual column an index keys on are
  real columns, same names as Scylla.
- Multi-value array cast per element type: integer slices → `SIGNED ARRAY`, string slices →
  `CHAR(512) ARRAY` (follows D1), unsigned → `UNSIGNED ARRAY`. A float slice in an index key panics
  in `CompileTable` (no genix table has one; ask if one appears).
- An indexed JSON slice is written as a JSON array literal of its elements; an empty or nil slice
  is `[]`, never NULL (the multi-value index skips NULL, and parity needs `[]`, see the FDB finding
  in YDB plan §18b Step 3).

### 7.2 Codec (`codec.go`)

Implements `db.Codec` (INTERNALS §4), installed with `db.SetCodec` in `init()`, as YDB §7.2:
`RenderValue` / `EncodeStatementValue` return the `database/sql` argument (ints as-is, unindexed
slices → colbin `[]byte`, indexed slices → JSON `[]byte`); `AssignValue` decodes the scanned value
(colbin or JSON for slices); `CompileDriverAccessors` nothing; collection options resolve `DBType`
by §7.1.

---

## 8. Schema and DDL

### 8.1 Main table

```sql
CREATE TABLE `genix`.`products` (
    company_id INT          NOT NULL,
    id         BIGINT       NOT NULL,
    name       LONGTEXT,                               -- D1
    category_id INT,
    price      DOUBLE,
    photos     LONGBLOB,                               -- []string, not indexed: colbin
    tags       JSON,                                   -- []int32, indexed: O11
    created    INT,
    updated    INT,
    version    SMALLINT,
    zz_dlt_ss  BIGINT UNSIGNED,                        -- delta virtual column
    PRIMARY KEY (company_id, id),
    INDEX ix_<local>  (company_id, category_id) STORING (name, price) LOCAL,
    INDEX ix_<delta>  (company_id, zz_dlt_ss) STORING (<every other stored column>) LOCAL,
    INDEX ix_<slice>  (company_id, (CAST(tags->'$' AS SIGNED ARRAY))) LOCAL,
    INDEX ix_<global> (category_id) GLOBAL
) DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_bin
  COMPRESSION = 'zstd_1.3.8'
  [PARTITION BY KEY(company_id) PARTITIONS <N>]        -- only when N > 1 (O8)
;
```

- Primary key = partition column first, then the clustering keys in declaration order
  (`TableCore.PartKey`, `TableCore.Keys`). Without a partition, the keys alone, no `PARTITION BY`.
- `STORING` lists only columns that are neither in the index key nor in the primary key.
- A table without `Partition` that Scylla stores under a fixed partition keeps that behaviour
  (read how `scylla` and `ydbdriver` handle it; ask if unclear).

### 8.2 `IndexSource` → OceanBase

| Type | Meaning | OceanBase |
|---|---|---|
| 1 | global index on one column | `(col) GLOBAL` |
| 2 | local index | `(company_id, col) LOCAL` |
| 3 | hash / cache-group index | `(company_id, hashCol) LOCAL` |
| 6 | simple view | `(company_id, keys…) STORING (Cols, or all stored columns) LOCAL` |
| 7 | hash view | `(company_id, hashCol) STORING (…) LOCAL` |
| 8 | range / radix (packed) view, TypeDelta included | `(company_id, packedCol) STORING (…) LOCAL`; TypeDelta stores every stored column |
| 9 | view table (fan-out) | panic as not implemented, as YDB (no genix table declares it; ask) |
| key includes a slice | global slice index, cache group over a slice, `zz_igs_*` | multi-value index `(company_id, CAST(col->'$' AS … ARRAY), <order columns…>) LOCAL`; a global slice index is `(CAST(…)) LOCAL` (multi-value indexes cannot be `GLOBAL`, §18b Step 2) |
| `Index.Partition` set | partition override | that column first, `GLOBAL` (O9) |

`TypeCacheGroupFromKey` needs no index (primary-key range). `TextSearch` creates no index.
Composite bucketing: panic as not implemented, as YDB; ask before implementing.
OceanBase allows 128 indexes per table (YDB plan D5 found at most 6 per genix table).

### 8.3 Index identity

`ix_<IndexSource.Name>`. A changed source name, key or `STORING` list is a **drop + add** at deploy.
`ADD INDEX` builds online; deploy waits for the DDL to return (it is synchronous in MySQL mode) so
`fn-init` leaves a usable schema.

### 8.4 Side tables

Same columns and keys as YDB plan §8.4 for `__delta_moves`, `__index_updated` and `__dtf_log`, with
§7.1 types, O15/O16 table options and, when N > 1, the same `PARTITION BY KEY(<partition column>)`.
There is **no** slice-index side table (O11).

`__dtf_log` uses table TTL (verified, §18b):

```sql
CREATE TABLE `genix`.`products__dtf_log` (
    half_hour INT NOT NULL, partition_id INT NOT NULL, ts INT NOT NULL, record_id BIGINT NOT NULL,
    changes LONGBLOB, written_at TIMESTAMP NOT NULL,
    PRIMARY KEY (half_hour, partition_id, ts, record_id)
) … TTL (written_at + INTERVAL 48 HOUR);
```

Generated names over 64 characters (index and side-table names) are cut to 55 characters plus `_`
and an 8-hex hash (§18b Step 2).

`written_at` comes from the ORM clock (`db.Now()`), never the server clock. Step 0 checks how
promptly the TTL task deletes expired rows; readers must not rely on it (the Scylla reader already
filters by `half_hour`).

`_sequences`: `(name VARCHAR(512) NOT NULL PRIMARY KEY, value BIGINT NOT NULL)`, O15/O16 options.

### 8.5 Deploy and homologation (`deploy.go`)

For every controller passed to `db.Deploy`, as YDB §8.6 with these sources and actions:
1. Compile the table (panics surface at boot). `CREATE DATABASE IF NOT EXISTS`.
2. Read `information_schema` (`COLUMNS`, `STATISTICS`, `TABLES` for compression/collation,
   `PARTITIONS` for the partition count). Absent → `CREATE TABLE` (with indexes) + side tables.
   Present → diff:
   - missing columns → `ALTER TABLE … ADD COLUMN`;
   - extra columns and extra side tables → **report only**, never drop;
   - column type or primary key changed → report and fail (pre-alpha: wipe and reseed);
   - compression not `zstd_1.3.8` → `ALTER TABLE … COMPRESSION = 'zstd_1.3.8'`;
   - collation not `utf8mb4_bin` → report and fail (a conversion rewrites every row; reseed);
   - **partition count ≠ configured N → report and fail** (repartitioning is a rebuild; reseed);
   - indexes: drop undeclared, add missing, drop + add changed (§8.3).
3. Idempotent: a second run changes nothing and prints only a summary.

Golden tests for `ddl.go`: plain table, every index kind, delta table, indexed and unindexed slices,
a slice index with order columns, a table without partition, N = 1 and N = 8.

---

## 9. Write algorithm (`write.go`, `side_tables.go`)

Serves every write method and `Merge`, through the shared write preparation (`db/write_prepare.go`)
and the `writePhase{records, writtenColumns, isInsert}` shape of the other drivers.

**Chunking.** A chunk is one transaction, sized by **row writes** (main rows + side-table rows),
starting at 1,000, and by placeholders (`rows × columns ≤ 65,535`) and bytes (≤ 8 MB, half of
`max_allowed_packet`). One record never spans chunks. A large `Insert` is not atomic across chunks
(as Scylla).

**Per chunk, inside one transaction:**

1. **Prepare** (shared): managed columns, autoincrement (reserved **once per chunk, before** the
   transaction, so a retry does not burn IDs), key packing, computed and virtual columns.
2. **Pre-read** with `SELECT <needed columns> FROM t WHERE (company_id, id) IN ((?,?), …) FOR UPDATE`,
   only when the table needs stored values: updates of tables with TypeDelta, cache groups or
   DataFrames, partial updates touching pinned columns, `Merge`. The pre-read locks the rows, so
   steps 3–7 see a stable state. A pure insert into a table with none of these skips it. (Indexed
   slices need **no** pre-read: the multi-value index follows the JSON column by itself.)
3. **Main rows.** `INSERT INTO t (<written columns>) VALUES (…), (…) ON DUPLICATE KEY UPDATE
   c = VALUES(c), …` for the non-key written columns: unlisted columns stay untouched and a missing
   row is created, exactly Scylla's upsert semantics (verified, §18b). One statement shape per
   written-column set. Never `REPLACE` (it deletes the row first).
4. **TypeDelta.** Stamp `version` and the packed delta column from the final pinned values, as YDB
   §9.4; a changed pinned prefix → one row in `__delta_moves`.
5. **Cache-group marks.** Groups entered or left → `__index_updated` upserts, deduplicated per chunk.
6. **DataFrame log.** `__dtf_log` rows computed as Scylla's `writeDataFrameLog`.
7. **Commit.** Indexes are synchronous: once committed, every index reflects the row.

**After the commit** (as Scylla): `cache.RaiseWrittenSlots` for by-IDs tables, then the text-search
sync.

**Merge.** `onUpdate` / `onInsert` run on the locked pre-read rows; on a retry (deadlock) they run
again on the re-read rows (callbacks are pure, the documented contract).

**Ordering note.** Marks, delta moves and the row commit together and every index is synchronous, so
the reader never sees a mark before the index reflects the row. The 16 s settle windows stay
unchanged (shared protocol constants), they are simply never needed for index lag here.

---

## 10. Read algorithm (`read.go`)

1. Shared planner → `AccessPlan`. A query the planner refuses fails exactly as on Scylla.
2. Render by source; statement text cached per shape hash, values as placeholders.

| Source | SQL |
|---|---|
| Primary key, range | `SELECT <proj> FROM t WHERE company_id = ? AND id >= ? AND id < ? ORDER BY company_id, id [DESC] LIMIT ?` |
| Primary key, points | `… WHERE (company_id, id) IN ((?,?), …)` |
| Native index | `SELECT /*+ INDEX(t ix_<name>) */ <proj> FROM t WHERE <span on index columns> ORDER BY <index columns> [DESC] LIMIT ?` |
| Slice index, one element | `SELECT /*+ INDEX(t ix_<name>) */ <proj> FROM t WHERE company_id = ? AND ? MEMBER OF (col->'$') …` |
| Slice index, several elements | `… AND JSON_OVERLAPS(col->'$', CAST(? AS JSON)) …` (engine dedupes, O12) |

3. **Spans.** Equality fan-out over one column → `col IN (…)`. Several range spans → one statement
   per span, run concurrently on the pool, merged in the requested order.
4. **Projection.** Index key + primary key + `STORING` covers it → no table lookup; otherwise the
   engine looks up the main table itself. No join in our SQL.
5. **Ordering on a slice index.** The multi-value index narrows by the order column's range but
   still sorts (`TOP-N SORT` in the plan, §18b), since one record has several entries. Correct, and
   bounded by the company + element range; the benchmark measures it.
6. **Paging.** Keyset on the source's key order, never `OFFSET`.
7. **Post-filter, merge, `Limit`**: shared planner functions. **Decode** only projected columns.

---

## 11. Cache protocols

As YDB §11, with these renderings: delta window rows read through `ix_<delta>` (covering, hinted),
moves from `__delta_moves`, marks from `__index_updated`, group reads through the hash index or the
multi-value index, `FromKey` as a primary-key range, by-IDs as primary-key points. Every read is a
plain read-committed statement: no stale mode exists or is needed. `cache.PlanDeltaSync`, snapshots,
slot files and group snapshots are reused unchanged.

---

## 12. Controller and admin (`controller.go`, `admin.go`, `sequences.go`)

As YDB §12, with these differences:

| Operation | OceanBase |
|---|---|
| `WipePartition(table, partition)` | `DELETE FROM t WHERE company_id = ? LIMIT 10000` repeated until 0 rows, on the table and each `<table>__*` side table (bounded transactions); by-IDs slot cleanup as `facade_scylla.go` |
| `RenameColumns(apply)` | Native `ALTER TABLE … RENAME COLUMN old TO new` (verified), per pair from the catalog `column_rename_scylla.go` uses; report-only when `apply` is false. Indexes and `STORING` lists that name the column follow it |
| `DeleteViewsAndIndexes` | Drop every `ix_*` index |
| `GetAutoincrementID(key, n)` | One transaction: `INSERT IGNORE INTO _sequences (name, value) VALUES (?, 0)`; `SELECT value … FOR UPDATE`; `UPDATE … value = value + ?`; return `previous + 1` |
| `SetCounter`, `ListCounters` | Same table |
| `WipeDatabase` (conformance only) | `DROP DATABASE IF EXISTS` + `CREATE DATABASE` |

---

## 13. Backend facade (`backend/db`, Step 1)

As YDB §13:
- `driver_oceanbase.go`, `facade_oceanbase.go` (`Connect`, `Deploy`, `MakeController`,
  `ListCounters`, `WipePartition`, `SetCounter`, `BypassFarewardSequences` no-op, `RenameColumns`);
  `obdriver`'s `init()` sets `db.GetAutoincrementID`.
- `connect.go`: `ConnectParams` gains `OceanBase OceanBaseParams` (host, port, user, password,
  partitions, max conns). The OceanBase database is `Keyspace`, so there is no separate database
  setting.
- Build tags: every `!js && !fdb && !ydb` file gets `&& !oceanbase`; re-run
  `rg -n "go:build" backend --type go -g '!genix-orm/**'`. `-tags oceanbase` combined with another
  driver tag fails to compile on duplicate declarations; acceptable.
- `core/security.go`: `[oceanbase] host, port, user, password, partitions, max_conns` →
  `core.Env.OCEANBASE_*`, next to `[ydb]`. `config.example.toml`: a `[oceanbase]` section in the same
  style (comments in Spanish like the rest of that file, values in English).
- `backend/go.mod`: require + replace `github.com/ivanjoz/genix-orm/obdriver => ./genix-orm/obdriver`.

---

## 14. fareward

Out of this plan for now. Under `-tags oceanbase` the ORM owns sequences (O14), so fareward's
sequence store is not used. fareward's limiter, request log and system metrics read/write ORM tables
through its Scylla store; an OceanBase store (Rust `mysql_async` or `sqlx`, behind a Cargo feature)
follows the rules of YDB plan §14 and is planned after the Milestone. Ask before starting it.

---

## 15. Feature parity checklist

| Feature | OceanBase |
|---|---|
| Keys, partition, `KeyConcatenated`, `KeyIntPacking`, `AutoincrementPart/From` | Composite PK, company first; shared prepare code |
| Autoincrement, `UseSequences`, `SequenceColumn`, `CounterColumn`, `GetAutoincrementID`, `ResetCounter` | `_sequences` |
| Managed columns, `DisableDefaultColumns` | Real columns |
| `TypeLocalIndex`, packed index | `LOCAL (company_id, col / packed)` |
| `TypeGlobalIndex` scalar | `GLOBAL (col)` |
| Any index on a slice column | JSON column + multi-value index |
| `TypeView` hash / range / `Cols` | `LOCAL … STORING` |
| `TypeViewTable` | Not implemented (ask) |
| FixedValues fan-out, composite buckets | Shared planner; composite buckets not implemented (ask) |
| `Computed` | Real column when an index keys on it; otherwise not stored |
| `TypeDelta` (+ moves, snapshots) | Covering `STORING` index + `__delta_moves` |
| `CacheByIDs`, generic by-IDs | PK points; slot files unchanged |
| `TypeCacheGroup*`, `QueryIndexGroup`, snapshots | `__index_updated` + hash / multi-value index |
| `SelectGrouped` | Driver-side aggregation with the shared group-by plan |
| `Merge`, `InsertUpdate*`, include/exclude | §9, one transaction per chunk |
| `TextSearch` | Existing backend interface |
| `DataFrames`, frame log, `AcquireFrameLock` | `__dtf_log` with TTL; lock on fareward |
| `db.Controller` | `ObController` |
| Deploy / homologation | Real DDL, §8.5 |
| Wipe, counters, column rename | Bounded `DELETE` loop; `_sequences`; native rename |

---

## 16. Steps

Each step ends with its acceptance check, `go vet`, a commit **and push** inside `genix-orm`, a root
commit for `backend/` / `docs/` changes, and an update of this plan's status line and §18b.

### Step 0 — Server and client spike (no ORM code)
1. `obdriver/go.mod`, `driver.go` with `Connect` / `Ping`; `server/README.md`.
2. `obdriver/spike_it_test.go` (gated `GENIX_OCEANBASE_IT=1`, throwaway database dropped at the end),
   each item its own test, results in §18b. Items already probed by hand (§18b) are re-run in Go:
   - every §8.1 DDL feature in one statement, N = 1 and N = 8;
   - `EXPLAIN` of each §10 statement shape: hinted index used, range bounded by company, reverse
     scan for `DESC LIMIT`, no table lookup when `STORING` covers;
   - multi-value index: one/several elements, dedupe, order column, empty array `[]`;
   - `ON DUPLICATE KEY UPDATE` subset semantics, placeholder limit, `max_allowed_packet` behaviour;
   - `SELECT … FOR UPDATE` + concurrent writer: blocking, deadlock error codes, retry;
   - one transaction with 1k / 5k / 20k row writes: time, success, error text;
   - TTL deletion latency on `__dtf_log`;
   - `ALTER TABLE … ADD INDEX` on a populated table (online, duration), `RENAME COLUMN` with an index;
   - **partition overhead (O8):** deploy every genix table with N = 1 and N = 8 (reuse the Step 2
     compile once it exists, or a script generating equivalent DDL), record tenant memory
     (`GV$OB_MEMORY`, `GV$OB_TABLETS`), tablet count and deploy time. The user sets the default.
3. Microbenchmark (`GENIX_OCEANBASE_BENCH=1`), same workloads as YDB Step 0: point get, 1,000-row
   range, 1,000-row commit, concurrency 1 and 64, parameterized statements.
- **Accept:** every item recorded in §18b.

### Step 1 — Facade and build tags (§13)
- **Accept:** in `backend/`: `go build ./...` with no tag, `-tags fdb`, `-tags ydb`,
  `-tags oceanbase`, `GOOS=js GOARCH=wasm go build .`, and `go vet` for each. Under
  `-tags oceanbase` every driver call may answer `errNotImplemented`.

### Step 2 — Compile, DDL, deploy (answer D1 first)
- `table.go`, `index_mapping.go`, `ddl.go` (+ golden tests), `deploy.go`, `codec.go`.
- `CompileTable` compiles every genix table without panicking, except deliberate not-implemented
  panics; if one fires on a genix table, stop and ask.
- **Accept:** `fn-homologate` twice against the server through a config copy with an
  `[oceanbase]` section (`GENIX_CONFIG_FILE`): first run creates everything, second reports
  `0 created, 0 altered`. A drift test (drop an index, change a `STORING`, change compression, add a
  stray index) is restored by one run.

### Step 3 — Conformance
- `conformance/driver_oceanbase_test.go`: `TableStruct` alias, `connectDriver`, `resetAndDeploy`
  (`WipeDatabase` + `Deploy`), `makeController`. Runs against the Scylla golden files.
- **Accept:** all tests fail only with `not implemented:`.

### Step 4 — Writes, sequences, primary-key reads
- **Accept:** conformance rows for keys, sequences, writes, merge, computed columns green.

### Step 5 — Index reads
- Native indexes, multi-value indexes, FixedValues fan-out, post-filter, paging.
- **Accept:** conformance rows for every index kind green;
  `GENIX_BENCH=1 go test -tags oceanbase ./tests/bench/ -run TestSeedBenchData` passes.

### Step 6 — Cache protocols (§11)
- **Accept:** their conformance rows green; bench endpoints return the same decoded bytes as Scylla
  for a first and an incremental sync.

### Milestone — Benchmark
- `bench.config.toml` gains `[oceanbase]`. Layer A under `-tags oceanbase`, same datasets,
  `GENIX_BENCH_NO_FAREWARD=1`, `GENIX_BENCH_NO_SNAPSHOTS=1`. Record the host spec (§5 caveat),
  lock-wait / deadlock rate, storage per table against Scylla's zstd size, the slice-index
  `TOP-N SORT` cost. Fill §18b in the FDB plan §14 format.

### Step 7 — Remaining parity
- `SelectGrouped`, text search, DataFrames + frame log, the full `ObController`, `WipePartition`,
  `RenameColumns`, every `exec/` script under `-tags oceanbase`.
- **Accept:** every §15 row green; `go test -tags oceanbase ./...` in `backend/` green.

### Step 8 — Documentation
- `obdriver/README.md` (storage model, mapping, limits, server), `obdriver/RATIONALE.md`
  (Context / Decision / Rationale per O-decision), `INTERNALS.md` §10, root `CLAUDE.md` repository
  map, `config.example.toml`.

---

## 17. Testing rules

- Unit tests next to the code, no database: type mapping, codec round trip for every ORM type ID
  (colbin and JSON slices), DDL golden strings, index mapping per `IndexSource.Type`, chunk sizing
  (row writes, placeholders, bytes), `AccessPlan` → SQL rendering including hints.
- Integration tests gated by `GENIX_OCEANBASE_IT=1` against the homelab server, each on a throwaway
  database dropped at the end. Conformance gated by `GENIX_CONFORMANCE=1`; benchmarks by
  `GENIX_OCEANBASE_BENCH=1` / `GENIX_BENCH=1`. Tests skip, never fail, when their env var is absent.

---

## 18. Risks

| Risk | Mitigation |
|---|---|
| Optimizer picks another path than the planner's | O13 index hints on every index read; Step 0 `EXPLAIN` checks each statement shape |
| A wrong collation silently breaks key parity | O16 explicit `utf8mb4_bin`; deploy fails on any other collation |
| Per-tablet overhead with N > 1 on the 8 GB mini node | O8 default 1; Step 0 measures N = 1 vs N = 8 |
| Row-lock contention on hot rows (`_sequences`, `__index_updated` marks, one company's rising IDs) | Reserve IDs once per chunk outside the write tx; dedupe marks per chunk; bounded deadlock retry; measure lock-wait rate |
| Slice-index ordered reads sort (`TOP-N SORT`) | Bounded by company + element range; benchmark measures; no genix slice index has order columns today (YDB plan §18b Step 2) |
| String length limits (D1) reject a long key value | Error, never truncation (strict mode); D1 sizes chosen with the user |
| Remote server over Tailscale adds network latency to every statement | Benchmark records RTT; run the backend on the homelab for the Milestone if RTT dominates |
| Server has no password and is reachable from the LAN | User's call on the homelab (§5); driver supports a password from day one |
| 5.0 patch releases change behaviour | Record the exact build in §18b; rerun conformance on upgrades |

---

## 18b. Results

### Manual probes (2026-11-06) — `mariadb` client against `100.64.0.4:2881`, `root@test`

| Item | Result |
|---|---|
| Server | `5.7.25-OceanBase_CE-v5.0.1.0`, build r100000042026072912, `ob_compatibility_mode = MYSQL`, `transaction_isolation = READ-COMMITTED`, `max_allowed_packet = 16777216`, `collation_server = utf8mb4_general_ci` (→ O16) |
| Multi-value index `(cid, CAST(tags->'$' AS UNSIGNED ARRAY))`, unpartitioned and `PARTITION BY KEY(cid) PARTITIONS 16` | Created. `5 MEMBER OF (tags->'$')` with the index hint: range scan `range(1,5,MIN,MIN ; 1,5,MAX,MAX)`, bounded by company and element |
| Same, `JSON_OVERLAPS(tags->'$', '[3,7]')` | Two ranges `(1,3,…)`, `(1,7,…)`; a record holding both 3 and 7 returned **once** |
| Same with an order column `(cid, CAST(…), upd)`, `upd > 150 ORDER BY upd DESC LIMIT 1` | Range narrowed to `(1,5,150 ; 1,5,MAX)`, then `TOP-N SORT` |
| Without the hint (3 rows) | Optimizer chose a primary-key scan with a filter (→ O13) |
| Two multi-value indexes over the **same** JSON column (`(cid, CAST(tags…))` and `(cid, CAST(tags…), upd)`) | **Neither is used**, even hinted, partitioned or not: plain primary-key scan with a filter (→ O18). Two multi-value indexes over **different** JSON columns: each is used. The `LOCAL` keyword makes no difference |
| Multi-value index without the `->'$'` path, queried as `MEMBER OF (tags)` | Not used: the query expression must match the index expression exactly. The renderer always emits `col->'$'` |
| `SIGNED ARRAY` with negative values, `CHAR(64) ARRAY` with strings | Both work |
| `(cid, cat) STORING (name, price) LOCAL` | `is_index_back=false` for a projection inside the index |
| `ORDER BY cat DESC, id DESC LIMIT 5` on that index | `TABLE RANGE SCAN s(ix_cat,Reverse)` with `limit(5)`, no sort |
| `(cat) GLOBAL` on partitioned and unpartitioned tables | Accepted; `is_global_index=true` on reads |
| `BIGINT UNSIGNED` = 18446744073709551615 | Stored and read back exactly |
| `TTL (expire_at + INTERVAL 48 HOUR)` | Accepted; `SHOW CREATE TABLE` reports `TTL = … BY DELETING`. Deletion latency not measured yet |
| `INSERT … ON DUPLICATE KEY UPDATE cat = VALUES(cat)` with a column subset | Existing row: only `cat` changed, `name` kept. Missing row: created, unlisted columns NULL |
| `ALTER TABLE … RENAME COLUMN` | Works |
| `CREATE SEQUENCE` / `nextval` | Works in MySQL mode (not used, O14) |
| Default table options | `COMPRESSION = 'zstd_1.3.8'`, `ROW_FORMAT = DYNAMIC`, `BLOCK_SIZE = 16384` |

### Step 0 (2026-11-06) — `go-sql-driver/mysql` v1.10.1, server on `homelab` (AMD Ryzen 7 5700U, 16 threads, 14 GB, OceanBase mini 8 GB)

Raw log lines are `SPIKE-RESULT` in `obdriver/spike_*_it_test.go` output. Timings were taken with the
test binary running **on the homelab** against `127.0.0.1` (`GENIX_OCEANBASE_HOST`), so Wi-Fi does
not distort them. From the laptop over Tailscale: `SELECT 1` p50 4.1 ms, p99 181 ms.
On the server itself: p50 0.12 ms, p99 0.20 ms.

**Rows read** is `total_memstore_read_row_count + total_ssstore_read_row_count` from
`GV$OB_SQL_AUDIT` for the prepared execution itself (found by `LAST_TRACE_ID()`), and the plan is the
plan-cache plan of that execution, not an `EXPLAIN` of literals. 4 companies × 5,000 rows, target
company 2, after a minor freeze. Same results with 1 and 8 partitions:

| Read shape | Returned | Rows read | Plan |
|---|---|---|---|
| PK range 100 | 100 | 100 | `items` |
| PK points `(company_id, id) IN (…)` × 3 | 3 | 3 | `items` |
| Local index equality | 250 | 250 | `items(ix_cat)` |
| Local index `DESC LIMIT 10` | 10 | 10 | `items(ix_cat)` |
| Local index `IN` fan-out × 3 | 750 | 750 | `items(ix_cat)` |
| Delta range `DESC LIMIT 50` | 50 | 50 | `items(ix_delta)` |
| Multi-value, one element (`? MEMBER OF`) | 500 | 500 | `items(ix_tags)` |
| Multi-value, two elements (`JSON_OVERLAPS(…, CAST(? AS JSON))`) | 1,142 | 1,214 | `items(ix_tags)`; the 72 extra are records holding both elements, deduped by the engine (result count exact) |
| Global index (cross-company by design) | 1,000 | 1,000 | `items(ix_gcat)` |

**A company-scoped read touches no other company's rows.** Prepared statements with bound parameters
use the hinted index, multi-value included.

| Item | Result |
|---|---|
| Session settings from the DSN | `collation_connection = utf8mb4_bin`, `ob_query_timeout = 600000000` applied on every pooled connection |
| `STORING` a `LONGTEXT`, `TEXT`, `MEDIUMTEXT` or `LONGBLOB` column | **Refused**: `1167 The used storage engine can't index column`. `JSON`: `3152 JSON column cannot be used in key specification` (→ D2) |
| `STORING` `VARCHAR(262144)` / `VARBINARY(1048576)` | Accepted. `VARCHAR` max is 262,144 characters, `VARBINARY` max 1,048,576 bytes |
| Declared row width | Capped near **1.5 MB**, for the table **and** each index: two `VARBINARY(1048576)` columns fail even in a plain table (`1118 Row size too large`). LOB columns do not count. So long strings and blobs stay `LONGTEXT` / `LONGBLOB` (D1 stands) and are never coverable |
| Full-company read, 5,000 rows, all 8 columns incl. LOBs | Via the covering `ix_delta` (non-LOB columns stored): 13.6 ms p50. Via a lean `(company_id, zz_dlt, id)` index with no `STORING`: 13.5 ms. PK scan: 10.4 ms. Projection of only the stored non-LOB columns via `ix_delta`: 3.3 ms. **When a LOB is projected, `STORING` buys nothing** |
| Multi-value: `[]`, `NULL`, negative ints, `[7,7]` | `[]` stored and read back; negative matches; a duplicated element matches its record once |
| **Multi-value on strings, case variants in one record** | **Server defect.** Record `["Ab","ab"]`: `'ab' MEMBER OF` through the index misses it (found without the index). The index keeps one entry for elements of one record that differ only by case, but looks up case-sensitively. Same with `CAST(… AS BINARY(512) ARRAY)`; no `COLLATE` syntax is accepted inside the cast. Distinct strings in one record (`["ab","cd"]`) work (→ D3) |
| `INSERT … ON DUPLICATE KEY UPDATE` subset (Go, prepared) | Existing row: only listed columns change. Missing row: created, unlisted NULL |
| Placeholders | 65,534 fine; 65,536 → `1390 Prepared statement contains too many placeholders` |
| One statement with a large value | 8 MB and 15 MB fine; 17 MB (above `max_allowed_packet` 16 MB) → the server drops the connection (`invalid connection`). The §9 8 MB chunk budget holds |
| `SELECT … FOR UPDATE` on a row another transaction locked | Waits, proceeds once the holder commits |
| Lock-wait timeout (`SET ob_trx_lock_timeout`) | `1205 Lock wait timeout exceeded` after the set time. Tenant default `-1` = wait until `ob_query_timeout` |
| Deadlock | Detected in ~0.4 s; one side gets `1213 Deadlock found when trying to get lock`, the other proceeds. §6 retries on 1205 / 1213 |
| One transaction, 1k / 5k / 20k rows (+1 index entry each), 1,000-row statements | 80 ms / 74 ms / 225 ms, all committed |
| TTL table | Created, but the tenant had **`enable_ttl = False`**: `ALTER SYSTEM TRIGGER TTL` → `10501 TTL feature is not enabled`. After enabling it (D4): the trigger is accepted and its tasks finish `OB_SUCCESS`, but an expired row was still there 2 min later (open, see D4) |
| `ADD INDEX` on 200k rows with 4 concurrent writers | 1.3 s; writers never failed (157 inserts during the build); index count matches table count afterwards |
| `RENAME COLUMN` of an indexed column | 260 ms; the index follows the new name |
| `ALTER TABLE … COMPRESSION` on 200k rows | 83 ms (metadata only; applies to new data) |
| **Partition overhead (O8)**: 132 tables × (PK + 4 local + 1 global index), empty | N = 1: **1,048 tablets**, deploy 19.6 s, drop 17.5 s. N = 8: **7,524 tablets**, deploy 28.9 s, drop 22.1 s. Tenant held memory (`GV$OB_TENANT_MEMORY.HOLD`, ~1.44 GB) moved by under ±40 MB in both cases: **no measurable memory cost for empty tablets**; the per-tablet cost under data (memstores, compaction) is not measured here (→ D5) |
| Other | Tenant `query_sql` in `GV$OB_SQL_AUDIT` is always empty (identify executions by trace id); the recycle bin is off (`DROP` frees tablets immediately) |

Microbenchmark (prepared statements, binary on the server, 5,000-row table with one local index):

| Workload | Concurrency 1 | Concurrency 64 |
|---|---|---|
| Point get | 5,948 ops/s, p50 0.14 ms, p99 0.38 ms | 46,854 ops/s, p50 1.05 ms, p99 4.8 ms |
| Range 1,000 rows | 1,655 ops/s, p50 0.55 ms, p99 0.92 ms | 9,362 ops/s, p50 5.1 ms, p99 23 ms |
| Commit 1,000 rows (one tx, one upsert statement) | 16 ops/s, p50 47 ms, p99 253 ms | 115 ops/s (115k rows/s), p50 320 ms, p99 5.5 s |

Not comparable head to head with the YDB numbers (different host, §5).

### Corrections found
- The tenant's default `ob_query_timeout` is **10 s**: `driver.go` sets 600 s per connection through
  the DSN, or partition scans and index builds would fail.
- `obdriver/go.mod` needs only the MySQL client for Step 0; `genix-orm/db` and `dataframe` are
  added in Step 1/2 with the first code that imports them.
- A multi-value index is used only when the query expression matches the index expression
  exactly (`col->'$'`); `MEMBER OF (col)` without the path skips it.

### Step 1 (2026-11-06) — facade and build tags
- `backend/db/driver_oceanbase.go`, `facade_oceanbase.go` (`oceanbase && !js`); `connect.go` gains
  `OceanBaseParams`; `core/security.go` reads `[oceanbase]`; `main.go`, `exec/test.go` and
  `tests/bench/setup.go` pass it; `config.example.toml` has the section; `backend/go.mod` requires
  and replaces `obdriver`.
- Every `!js && !fdb && !ydb` file, `exec/generic_records_test.go` and the Scylla / FDB conformance
  driver files gained `&& !oceanbase`.
- `obdriver`: `Exec` (every operation `errNotImplemented`), `TableStruct`, `ObController`, and
  `Deploy` / `WipePartition` / `SetCounter` / `ListCounters` stubs. `CompileTable` and
  `GetTable` panic until Step 2. `init()` sets `GetAutoincrementID`, `QueryCachedGenericByIDs` and
  `SetDebugLogging`; the type resolver and codec come with `codec.go` in Step 2.
- `go mod tidy` in `backend/` also moved `genix-orm/fdb` to indirect and dropped the unused
  `github.com/google/go-cmp` (drift that predated this step).
- **Accept:** `go build ./...` with no tag, `-tags fdb`, `-tags ydb`, `-tags oceanbase`, and
  `GOOS=js GOARCH=wasm go build .`: all OK. `go vet` of `db`, `core`, `exec` for the four tag sets:
  OK. `go vet` of `genix-orm/conformance` (no tag, fdb, ydb) and of `obdriver`: OK.

### Step 2 (2026-11-06) — compile, DDL, deploy
- `codec.go` (type map §7.1, colbin codec; the JSON form of indexed slices is bound by the write and
  read paths in Step 4), `table.go` (`ObTable`, compiled through `CompileFromSchema`,
  `DescribeIndexes`, `FinalizeColumns`), `index_mapping.go`, `ddl.go`, `deploy.go`, and
  `WipeDatabase` in `admin.go`.
- **Rules the server forced** (each verified on it):
  - **Multi-value indexes are `LOCAL` only**: `GLOBAL` → `1235 Not supported feature`. A global
    slice index (`TypeGlobalIndex` on a slice) is a `LOCAL` multi-value index on the element alone;
    with N > 1 partitions its lookup visits every partition. genix declares none today.
  - **Identifiers are capped at 64 characters.** `ix_<source>` reached 78
    (`ix_warehouse_product_movement__pk_date_product_id_type_sub_divisor_rng_view`). A generated name
    (index or side table) over 64 keeps its first 55 characters plus `_` and the 8-hex FNV-1a hash of
    the full name (`boundedIdentifier`). One genix index needs it.
  - `__dtf_log.written_at` is a `TIMESTAMP` (time-zone independent); TTL on it purges (D4).
- Homologation reads `information_schema` (`TABLES` collation, `COLUMNS`, `STATISTICS` keys,
  `PARTITIONS` count) in four statements per database, and `SHOW CREATE TABLE` per table for what only
  it reports: `STORING`, `GLOBAL`, compression and TTL. A multi-value index reports its element as
  `json_query(`col`,'$' RETURNING <type> …)` plus hidden trailing parts (primary-key columns and a
  `__doc_id_*`), which the comparison ignores. Deploy fails with the two `ALTER SYSTEM SET` lines
  when a table needs TTL and `enable_ttl` / `enable_kv_ttl` is off.
- The shared compile already refuses a local index on a slice and collapses duplicate index
  declarations, so O18 cannot be declared today; its guard stays for future index kinds.
- Tests: `TestMappingOfEveryIndexKind` (the YDB plan's every-index-kind table),
  `TestMappingStoresAProjectionWithoutLobs` (D2 covering), `TestMappingRefusesStringSliceIndexes`
  (D3), DDL golden strings (unpartitioned, 8 partitions, TTL frame log, `_sequences`, add / drop
  index, add column, compression), metadata normalization, `TestBoundedIdentifier`; integration:
  `TestDeployHomologatesADriftedTable` (drop an index, change a `STORING`, change compression, add a
  stray index, add a column → one run restores everything; type, primary-key and partition-count
  changes refused) and `TestDeployReadsBackPartitionedAndTTLTables`.
- **Accept:** `fn-homologate` with a config copy holding `[oceanbase]` (`GENIX_CONFIG_FILE`), from an
  empty `genix` database: first run `129 tables, 129 created` (20 s), second run
  `0 created, 0 altered, 129 unchanged` (1 s). 129 = YDB's 132 minus its 3 slice side tables.
- Deployed: 113 secondary indexes on 61 tables, at most 7 per table (`purchase_order`,
  `sale_order`); 3 multi-value indexes (the cache groups over a slice); `GLOBAL` indexes on 4
  tables; `STORING` on 9 tables. D2 in numbers: of 44 tables with a delta index, **7** have a
  covering one; the rest hold a `LONGTEXT` / `LONGBLOB` / `JSON` column and read back from the table.
  2 TTL frame-log tables. The `genix` database stays on the server for the next steps.

### Step 3 (2026-11-06) — conformance
- `conformance/driver_oceanbase_test.go` (`oceanbase && !fdb && !ydb`): `TableStruct` alias,
  `connectDriver` (`GENIX_OCEANBASE_HOST` / `_PORT` / `_USER` / `_PASSWORD`, default the homelab
  server, 1 partition), `resetAndDeploy` (`obdriver.WipeDatabase` + `obdriver.Deploy` on the
  `genix_orm_conformance` database), `makeController`. `driver_ydb_test.go` gained `&& !oceanbase`;
  `conformance/go.mod` requires and replaces `obdriver`. `go vet` green for no tag, fdb, ydb and
  oceanbase.
- **Accept:**
  - Scylla (`GENIX_ORM_SCYLLA_TEST=127.0.0.2:9042`) matches its golden files: 11/11 (14 s).
  - `-tags oceanbase`: all 11 fail with `not implemented` only (10 at the first `Insert`, 1 at
    `Controller.InsertRecordsJSON`), after wipe + deploy succeeded in each (28 tables created over
    the run, no deploy error). 9 s.
- The §15 rows not covered yet are the ones YDB plan §18b Step 3 lists; each gets its test with the
  step that implements it.

### Step 4 (2026-11-06) — writes, sequences, primary-key reads
- `sequences.go`: `GetAutoincrementID` / `SetCounter` / `ListCounters` on `_sequences`, as §12:
  `INSERT IGNORE` at 0, `SELECT … FOR UPDATE`, `UPDATE`, one transaction. A write reserves its IDs
  in the table's own database, before its chunk transaction. `sequences_it_test.go`: 16 writers ×
  20 reservations of 5 IDs on one counter, all 1,600 IDs distinct and gapless (4.5 s).
- `driver.go` `runTransaction`: read committed; a `1213` deadlock or `1205` lock-wait timeout rolls
  back and reruns the body (at most 5 attempts), which re-reads what it locks.
- `values.go`: binding and scanning per stored column. Scalars as themselves; an indexed slice as
  JSON array text, `[]` when nil or empty (Scylla reads an indexed list back as `[]` too); any other
  slice or complex value as colbin, NULL when empty (Scylla reads `null`); managed DB-only columns
  from the write's stamps; generated columns from `db` (unsigned layout). Scan targets are typed by
  field kind, so the text and binary protocols both decode.
- `write.go`: the FDB `writePhase` shape over `db/write_prepare.go`. An insert writes every column
  but the excluded ones, plus every generated column; an update writes `db.ResolveUpdateColumns`
  (computed columns as their From columns). Chunks of ≤ 1,000 records and ≤ 65,535 placeholders,
  one transaction each; statements split at 8 MB. `INSERT … ON DUPLICATE KEY UPDATE c = VALUES(c)`
  over the written non-key columns. A table with TypeDelta indexes locks and reads the chunk's
  stored version and pinned columns (`SELECT … FOR UPDATE` on the primary-key tuples), stamps
  `cache.NextVersion`, writes the packed delta column from the final pinned values, and writes
  `__delta_moves` rows for changed prefixes in the same transaction (Scylla writes them after the
  rows; here they commit together).
- **Refused for now, loudly:** a write to a table with cache groups (marks: Step 6), DataFrames (frame
  log: Step 7) or text search (sync: Step 7) answers `not implemented`; so do by-IDs slot raises.
- `merge.go`: Scylla's Merge. The stored records are read by primary-key tuples **before** the write
  transaction, unlocked, as Scylla does (not under `FOR UPDATE` as §9 proposed): the callbacks run
  once, and the plan's locked variant would rerun them on a deadlock retry for no parity gain.
- `read.go`: a plain `Select` renders every predicate as a SQL condition (`=`, `!=`, ranges, `IN`,
  tuple `BETWEEN`, `CONTAINS` on an indexed slice as `? MEMBER OF (col->'$')`), the projection,
  primary-key order (`DESC` on `OrderDesc`) and `LIMIT`; the server picks the access path. No
  planner, hints or keyset paging yet (Step 5); a predicate on a generated or computed column is
  refused. Delta and cache-group reads answer `not implemented`.
- **Accept:** `-tags oceanbase` conformance: `TestAutoincrementKeys`, `TestMerge` and
  `TestInsertUpdateInclude` match the Scylla golden files. The rest fail with `not implemented`,
  except `TestIndexReads`: its first five reads match (global, local, slice and view reads), then
  step 5 "an open range on the view is refused" is accepted, since the shared planner's refusal
  comes with Step 5. The TypeDelta write path ran in the cache tests' inserts, but its reads (and
  so its verification) are Step 6.

---

## 19. Out of scope

Lambda; OceanBase Oracle mode; columnar (`WITH COLUMN GROUP`) tables; native full-text, vector and
spatial indexes; multi-node cluster topology and replica placement; automatic repartitioning;
fareward OceanBase store (§14, after the Milestone); migrating Scylla data (pre-alpha: wipe and
reseed).
