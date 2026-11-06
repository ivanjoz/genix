# FoundationDB driver — implementation plan (full parity)

**Audience.** An implementing agent. Read §0 before writing any code. Every step in §11 has an
acceptance check; do not start a step before the previous one passes.

**Goal.** FoundationDB 8.0 replaces ScyllaDB behind **one build tag** (`-tags fdb`). Same
`db.Executor` contract, same `TableSchema` / `Index` declarations, same queries, same cache wire
protocols (genix-ui unchanged). No table declaration and no query in `backend/` changes. A shared
conformance suite proves both drivers return the same results.

**First milestone.** Benchmark FDB against Scylla on `backend/tests/bench`, in a local container.

**Status, 2026-11-05.** Plan only. Nothing implemented.

---

## 0. Read first

| File | Why |
|---|---|
| `backend/genix-orm/INTERNALS.md` | The driver contract (`Executor`, `Codec`, `TableCore`, hooks). §10 "Adding a driver" is the checklist this plan follows |
| `backend/genix-orm/scylla/ORM_INTERNALS.md`, `scylla/RATIONALE.md` | What every index type, view, cache protocol and write phase means. Parity = these semantics |
| `backend/genix-orm/db/cache/*.USER.md`, `db/cache/RATIONALE.md` | The three cache protocols the driver must serve unchanged |
| `backend/genix-orm/dynamo/README.md` | The closest existing driver: a key-value store with ORM-maintained indexes |
| `backend/db/driver_scylla.go`, `backend/db/db.go`, `backend/db/autoincrement_scylla.go` | The facade and how the driver is chosen per build tag |
| `backend/tests/bench/README.md`, `setup.go` | The benchmark stack this plan has to run on |
| `fareward/PROTOCOL_SHAPES.md` §1.1, `fareward/go/connection.go` | The framing style the daemon protocol copies |

**Repository rules that apply here** (from `CLAUDE.md`):
- `backend/genix-orm` and `fareward` are **git submodules**. Commit **and push inside the
  submodule**; a root commit does not publish them.
- Pre-alpha: no backwards compatibility, no runtime fallbacks for old data shapes. Wipe and reseed.
- English everywhere. Expressive names. Comment rationale on non-obvious blocks. No more code
  than necessary.
- Supervised mode: any choice not settled in §1 or §2 → **stop and ask**. Anything that deletes
  code not obviously dead → ask.
- Never `time.Now()` for a persisted value: `db.Now()` / the ORM's managed clock.

---

## 1. Settled decisions

| # | Decision | Choice |
|---|---|---|
| S1 | The switch | Build tag `fdb`. `backend/db/driver_fdb.go` is `//go:build fdb && !js`; `driver_scylla.go` becomes `//go:build !js && !fdb`. A runtime flag is impossible: the driver is a type argument of `TableStruct` |
| S2 | Declarations | Unchanged. Every `Index` type, `FixedValues`, `Size()`, `Int32()`, `KeyIntPacking`, `KeyConcatenated`, `Computed`, `DataFrames`, `TextSearch`, `CacheByIDs`, `GenericRecord` keeps its meaning. Storage hints with no FDB meaning (`set`/`list`/`frozen` tags, `Int32()` on a view) are accepted and ignored |
| S3 | Client linkage | A gateway daemon, `fdbgate`, links `libfdb_c`. The backend stays `CGO_ENABLED=0` and never imports the Apple binding |
| S4 | Daemon language | Go with cgo, official binding `github.com/apple/foundationdb/bindings/go`, `fdb.MustAPIVersion(800)` |
| S5 | Wire protocol | Length-prefixed binary frames over a unix socket, colbin bodies (§4) |
| S6 | Transaction model | One request = one FDB transaction, retried by the daemon (`db.Transact` / `db.ReadTransact`). The backend never holds an open transaction. Conditions are declarative |
| S7 | Where the daemon runs | Same container as `fdbserver` (§5). Later on a VPS: a systemd unit next to `fdbserver` |
| S8 | Planner | Its storage-independent part moves from `scylla/` to `genix-orm/db` (§7) |
| S9 | Cache protocols | Same wire protocols: delta `C.V` cursors, cache groups, by-IDs. Settle windows kept |
| S10 | Deployment scope | VPS mode only. Lambda is out of scope |
| S11 | Benchmark sizing | One `fdbserver` process, `--memory 2GiB`, `configure new single ssd-redwood-1` |

## 2. Decisions still open (ask the user before the step that needs them)

> **Taken 2026-11-06 in unsupervised mode (the user asked to finish and benchmark without
> stopping); review them:**
> - **D1 — column codec**, as recommended (`fdbdriver/rowcodec.go`).
> - **D2 — (b) for the benchmark**: fareward keeps its Scylla tables and does not run in the FDB
>   benchmark. The seeder skips the credit budgets with `GENIX_BENCH_NO_FAREWARD=1`; Layer B
>   (router) and `FarewardCharge` were not run on either driver.
> - **D3 — FoundationDB owns the sequences** under `-tags fdb` (`fdbdriver/admin.go`, `ReserveRange`).
> - **Driver location**: `genix-orm/fdbdriver` (root module, beside `scylla`), not inside the `fdb`
>   module: that module also builds the cgo daemon from its own folder (the container's build
>   context), which must not need the ORM's sources. `genix-orm/fdb` keeps the wire client.
> - **TypeDelta entries hold a copy of the row** (a change to §6.2): pointer entries made a sync one
>   point read per record in the daemon, 4.9x slower on a 10k first sync (§14).

**D1 — Row encoding (needed by Step 4). Recommended: an ORM column codec, not colbin of the struct.**
- *Column codec:* value = header + one entry per stored column, keyed by a 16-bit column ID
  (16 bits of CRC32 of the column name, collision panics at compile, as the DataFrame log does).
  It stores exactly `TableCore.Columns`, as Scylla does: managed columns (`created`, `updated`,
  `version`) that have **no struct field** survive, `db:"name"` tags keep their meaning, and a
  read can decode only the projected columns.
- *Colbin of the whole struct (the dynamo way):* less code, but managed columns without a struct
  field are lost, `cb:"-"` fields are dropped, and field identity follows colbin's rules instead
  of the `db` tags. Fails parity.

**D2 — fareward's own ScyllaDB tables (needed by Step 9, and by the benchmark if fareward must run).**
fareward (Rust, separate repo) writes to Scylla directly: `sequences` (`src/sequence/store.rs`),
`credit_usage_company`, `credit_usage_user`, `company_credit_budget` (`src/limiter/storage.rs`), the
access read `SELECT accesos_computed … FROM users` (`src/limiter/access.rs`), `user_logs`,
`request_errors` (`src/reqlog/writer.rs`), `server_metrics` (`src/sysmetrics/writer.rs`). Options:
- (a) Recommended: fareward gets a Rust client for `fdbgate` (same frames) and an FDB store behind a
  build feature. `users` is an ORM table, so fareward needs a decoder for the D1 row encoding.
- (b) fareward keeps a Scylla keyspace of its own. FDB then does not fully replace Scylla.
- For the **benchmark milestone only**: decide whether fareward runs (charges, access checks) or
  the bench disables it under `-tags fdb`.

**D3 — Autoincrement owner under `fdb`. Recommended: FDB itself.** Today fareward reserves every
ID because a Cassandra counter cannot report the result of *your* increment
(`backend/db/autoincrement_scylla.go`). FDB's serializable read+add (`ReserveRange`, §4) has no
such race, so `autoincrement_fdb.go` can wire `ReserveCounterRange` / `SetCounterValue` straight to
the driver. Confirm, since it moves sequence ownership off fareward under `fdb`.

**D4 — Legacy `exec/` code.** `exec/demo.go` (raw `DROP MATERIALIZED VIEW`, `scylla.Colx`) and
`exec/init.go:397-402` (`fn-recalc`, raw `DELETE FROM genix.almacen_producto…` on tables that look
renamed) look dead. Ask before deleting them; otherwise port them in Step 1.

---

## 3. Architecture

```
backend (CGO_ENABLED=0, -tags fdb)
  app tables ──► backend/db facade ──► genix-orm/fdb (driver, pure Go)
                                          │  frames over unix socket
                                          ▼
container genix-bench-fdb ──────────────────────────────────────────
  fdbgate (Go + cgo, libfdb_c 8.0)  ──►  fdbserver 8.0 (single, redwood)
  /run/genix-fdb/fdbgate.sock (bind-mounted to the host)
```

New modules and files:

```
backend/genix-orm/db/planner_*.go        moved planner (§7)
backend/genix-orm/db/write_prepare.go    moved storage-independent write preparation (§7)
backend/genix-orm/fdb/                   module github.com/ivanjoz/genix-orm/fdb  (pure Go)
  client.go            frame client: dial, multiplex, retry on broken connection
  protocol.go          request/reply structs (colbin), shape bytes, error codes
  keys.go              tuple encoding of every key kind (§6.2)
  rowcodec.go          D1 codec: encode, decode, decode-projection, chunking (§6.3)
  table.go             FdbTable{db.TableCore; …}, CompileTable
  executor.go          Exec[TableT, RecordT] implementing db.Executor
  write.go             the write algorithm (§8)
  read.go              the read algorithm (§9)
  index_entries.go     index entry computation and diff for every index type
  delta.go             TypeDelta reads/writes, moves
  cache_by_ids.go      by-IDs (slot files unchanged; record reads via GetMany)
  cache_groups.go      freshness rows, QueryIndexGroup, S3 snapshots
  data_frame_backend.go  dataframe.Backend + PartitionRecordScanner, the frame log
  text_search.go       sync hooks to db.TextSearchBackend
  controller.go        FdbController[T, E] implementing db.Controller
  admin.go             Connect, Deploy, WipePartition, ListTables, counters
  fdbgate/             module github.com/ivanjoz/genix-orm/fdb/fdbgate  (cgo; never imported)
    main.go  server.go  ops.go
  container/Containerfile, container/entrypoint.sh
backend/db/driver_fdb.go, backend/db/autoincrement_fdb.go
backend/go.mod: require + replace github.com/ivanjoz/genix-orm/fdb => ./genix-orm/fdb
```

---

## 4. `fdbgate` protocol

### 4.1 Framing

All integers big-endian.

```
request  [op:1] [correlation:u32] [length:u32] [payload: colbin]
reply    [status:1] [correlation:u32] [length:u32] [body: colbin]
```

- One connection is multiplexed; replies may arrive out of order and are matched by
  `correlation`. The client keeps a small pool (default 4 connections) and fails all pending
  requests of a connection that breaks.
- `length` is `u32`, unlike fareward's `u16`: a commit can carry up to 1 MB.
  `max_frame_bytes = 16 MiB` is enforced on both sides.
- No frame signature. The socket file is created `0660` and is the only access control; there is
  no TCP listener. Do not add one without asking.
- `status`: `0` ok, `1` condition failed (body says which), `2` FDB error (body: FDB error code +
  message), `3` bad request, `4` too large (a transaction limit was hit, body says which).

### 4.2 Operations

Every op runs in **one** FDB transaction. The daemon retries on retryable FDB errors (the binding's
`Transact` loop) and sets `TimeoutMillis = 4500` so a request never runs into the 5 s limit
silently: on timeout it answers `status 4`.

| op | name | request | reply |
|---|---|---|---|
| `0x01` | `GetMany` | `Keys [][]byte`, `Snapshot bool` | `Values [][]byte` (nil = absent), `ReadVersion i64` |
| `0x02` | `GetRanges` | `Ranges []Range` | `Results []RangeResult` |
| `0x03` | `IndexRangeGet` | `Ranges []Range`, `RecordPrefixes [][]byte` (one per range) | `Results []RangeResult`, `Records [][]byte` aligned with the index keys |
| `0x04` | `Commit` | `Conditions []Condition`, `Mutations []Mutation` | `CommittedVersion i64`, or status 1 with `FailedCondition u32` |
| `0x05` | `ReserveRange` | `CounterKey []byte`, `Count u32` | `First i64` |
| `0x06` | `SetCounter` | `CounterKey []byte`, `Value i64` | `Previous i64` |
| `0x07` | `ClearPrefix` | `Prefix []byte` | `Done bool`. Clears in chunks; the client repeats until `Done` |
| `0x08` | `Ping` | — | `ReadVersion i64` |

```
Range       { Begin, End []byte; Limit u32; Reverse bool; ByteLimit u32 }
RangeResult { Keys, Values [][]byte; More bool; NextBegin []byte }   NextBegin: continuation (the
                                                                     next Begin, or End when Reverse)
Condition   { Kind u8; Key []byte; Value []byte }
              Kind 1 KeyAbsent · 2 KeyPresent · 3 ValueEquals · 4 ValuePrefixEquals
Mutation    { Kind u8; Key []byte; Value []byte; EndKey []byte }
              Kind 1 Set · 2 Clear · 3 ClearRange · 4 AtomicAdd · 5 AtomicMax ·
                   6 SetVersionstampedKey · 7 SetVersionstampedValue
```

- Conditions are checked with reads **inside the commit's transaction**, so they add read
  conflict ranges: a concurrent writer of a checked key makes the commit conflict and retry inside
  the daemon. A retry re-checks the conditions; if one fails it answers status 1.
- `IndexRangeGet`: every index entry's **value** holds the record key's suffix (§6.2), so for
  each index key the daemon fetches `RecordPrefixes[i] + value` in the same transaction. The
  daemon parses no tuples. A global index's value carries the partition too, so its record prefix
  stops at `(ks, T)`.
- `ReserveRange`: `v = get(key) (0 if absent); set(key, v+count); return v+1`, in one transaction.
  Concurrent reservations conflict and retry in the daemon.
- **As built:** colbin has no `[][]byte`, so every `[][]byte` in the tables above travels as a
  `ByteList` (`Lengths []int32` + one flat `Data []byte`, length `-1` = absent). `GetMany` therefore
  has no separate "present" array: an absent key is an absent entry. `RangeResult.More` is computed
  by fetching `Limit+1` rows.
- Records over the value limit are chunked by the driver (§6.3). `GetMany` / `IndexRangeGet` replies
  return the head value only, and the driver issues a second `GetRanges` for the chunks of
  chunked rows.

### 4.3 Daemon configuration (env)

`FDBGATE_SOCKET` (default `/run/genix-fdb/fdbgate.sock`), `FDB_CLUSTER_FILE`
(default `/var/fdb/fdb.cluster`), `FDBGATE_MAX_FRAME_BYTES`, `FDBGATE_TX_TIMEOUT_MS` (4500),
`FDBGATE_LOG_LEVEL`.

---

## 5. Container (Step 0)

Verified facts (2026-11-05):
- `docker.io/foundationdb/foundationdb:8.0.0` exists. It ships `/usr/lib/libfdb_c.so` and starts
  with `tini -- /var/fdb/scripts/fdb.bash`, which passes extra args to `fdbserver`
  (so `--memory 2GiB` goes through CMD). The cluster file is `/var/fdb/fdb.cluster`; with no
  configuration it falls back to self-coordinated `docker:docker@<ip>:4500`.
- The release has `fdb-headers-8.0.0.tar.gz` (the C headers cgo needs).
- The Go binding has no tags. The `release-8.0` branch has `#define FDB_API_VERSION 800` and a
  committed `generated.go`. Pin commit `946e01da3a60d9aa2b22d2f60201e4e5df9daa43`
  (latest on `release-8.0` touching `bindings/go`, 2026-08-27):
  `go get github.com/apple/foundationdb/bindings/go@946e01da3a60`.

`fdb/container/Containerfile`:

```Dockerfile
FROM docker.io/foundationdb/foundationdb:8.0.0 AS fdb

FROM docker.io/library/golang:1.27-bookworm AS build
# C headers for cgo; libfdb_c from the runtime image so the versions match exactly
ADD https://github.com/apple/foundationdb/releases/download/8.0.0/fdb-headers-8.0.0.tar.gz /tmp/
RUN mkdir -p /usr/include/foundationdb && tar -xzf /tmp/fdb-headers-8.0.0.tar.gz -C /usr/include/foundationdb --strip-components=1
COPY --from=fdb /usr/lib/libfdb_c.so /usr/lib/libfdb_c.so
WORKDIR /src
COPY . .
RUN cd fdbgate && CGO_ENABLED=1 go build -o /out/fdbgate .

FROM fdb
COPY --from=build /out/fdbgate /usr/bin/fdbgate
COPY container/entrypoint.sh /var/fdb/scripts/entrypoint.sh
ENTRYPOINT ["/usr/bin/tini", "-g", "--", "/var/fdb/scripts/entrypoint.sh"]
```

**Corrections found while building (2026-11-05):**
- The header tarball is **flat** (no top-level directory), so no `--strip-components`, and it does
  **not** contain `fdb_c_types.h`, which `fdb_c.h` includes. The Containerfile fetches that one file
  from the `8.0.0` tag of the repository.
- `libfdb_c.so` is in `/usr/lib`, which is not on the image's loader path: set
  `LD_LIBRARY_PATH=/usr/lib` (the entrypoint does for `fdbgate`; do it for `fdbgate.test` too).
- `--memory 2GiB` alone fails (`--memory has to be larger than --cache-memory`, default cache 2 GiB):
  pass `--memory 2GiB --cache-memory 1GiB`.
- `fdbcli --exec status` blocks on an unconfigured database. The entrypoint probes with
  `configure new single ssd-redwood-1` instead (creates it, or answers "already exists").
- The container also ships `/usr/bin/fdbgate.test` (the in-process baseline benchmark).

`entrypoint.sh`: start `/var/fdb/scripts/fdb.bash "$@" &`, wait until `fdbcli --exec status`
answers, run `fdbcli --exec "configure new single ssd-redwood-1"` only if the database is
unconfigured, then `exec fdbgate`.

Run (benchmark):

```sh
podman build -t genix-fdb:8.0.0 -f backend/genix-orm/fdb/container/Containerfile backend/genix-orm/fdb
mkdir -p /run/user/$UID/genix-fdb
podman run -d --name genix-bench-fdb --network host \
  -e FDB_NETWORKING_MODE=host -e FDB_PORT=24500 \
  -v /run/user/$UID/genix-fdb:/run/genix-fdb:z \
  genix-fdb:8.0.0 --memory 2GiB
```

Port 24500 avoids colliding with a dev FDB on 4500. Socket ownership across the rootless podman
user namespace must let the host user connect; fix it with `--userns keep-id` or a `chmod` in the
entrypoint, and document whichever works.

---

## 6. Storage model

### 6.1 Namespaces

One FDB cluster, one top-level prefix per keyspace: the tuple `(keyspaceName,)`, keyspace =
`TableSchema.Namespace` or the configured default (`[db] name`). No directory layer: plain tuple
prefixes, to keep keys short and greppable.

### 6.2 Key kinds

`T` = `TableSchema.ID` (int16, already unique, `docs/TABLE_ID_ASSIGNMENT.md`). `P` = partition value
(int64; `0` for tables without `Partition`). Keys are FDB tuples (`fdb/tuple`); integers encode in
numeric order, signed included.

| Kind | Key | Value |
|---|---|---|
| Record | `(ks, T, P, 'r', pk…)` | row (§6.3) |
| Record chunk | `(ks, T, P, 'r', pk…, 'c', n)` | chunk n, n ≥ 1 |
| Local index entry | `(ks, T, P, 'i', indexID, indexValue…, pk…)` | packed `'r'` key suffix (for `IndexRangeGet`) |
| Global index entry | `(ks, T, 'G', indexID, indexValue…, P, pk…)` | full `'r'` key suffix including `P` |
| Delta entry | `(ks, T, P, 'd', deltaID, packedPinnedAndUpdated, pk)` | a copy of the row (was the `'r'` suffix; see §2 and §14) |
| Delta move | `(ks, T, P, 'm', deltaID, oldPrefix, pk)` | `updated` (SUnixTime) of the move |
| Cache-group freshness | `(ks, T, P, 'g', indexID, hash)` | `last_write` (int32 little-endian, for `AtomicMax`) |
| DataFrame log | `(ks, T, P, 'f', halfHour, …)` | the same `cgl` entry `data_frame_log.go` writes |
| Sequence | `(ks, '_seq', name)` | int64 little-endian (for `AtomicAdd`) |
| Table registry | `(ks, '_tables', T)` | table name (diagnostics only) |

Rules:
- `indexID` is the index's position in `TableSchema.Indexes` **plus a stable hash of its key
  column names**, so reordering two declarations does not silently swap their entries. A changed
  declaration means: run `RecalcVirtualColumns` on every partition (Step 1 makes that a facade call).
- Every index kind — local, packed, hash, range/radix view, `TypeView` with `Cols`,
  `TypeViewTable`, `TypeDelta` — is an **entry pointing to the record**. No projected copies: a
  view's payload is read from the record by `IndexRangeGet` in the same transaction. This
  replaces the Scylla materialized views and view tables with one mechanism.
- A slice column feeding an index (`TypeGlobalIndex` on `set<int>`, `TypeViewTable` fan-out,
  composite bucket sets, `zz_igs_` group hashes) writes **one entry per element**.
- Packed and radix values use the **unsigned** layout from `db.PackSlotValues` directly. The tuple
  layer orders `uint64` correctly, so `storeVirtualPacked`'s sign flip is not used.

### 6.3 Row codec (assumes D1 = column codec)

```
head value  [rowVersion:u64 BE] [flags:u8] [columnCount:u16] { [columnID:u16] [len:uvarint] [bytes] }*
flags bit 0 = chunked: then the head holds only rowVersion|flags|chunkCount:u16, and the
              column bytes are split across 'c' chunk keys of ≤ 90,000 bytes each
```

- `rowVersion` is the driver's optimistic-concurrency counter: every write sets it to
  `previous + 1`. It is **not** the ORM's managed `version` column (a TypeDelta table keeps
  stamping that one exactly as `stampDeltaWrites` does today).
- Column bytes: int8/16/32/64 zigzag varint; float32/64 fixed LE; bool 1 byte; string raw; `[]byte`
  raw; `[]int*` / `[]string` length-prefixed lists; anything else = what `Codec.EncodeStatementValue`
  produces today (colbin blob). Implement it as the FDB driver's `db.Codec` (`SetCodec` in the
  driver's `init`), as INTERNALS §4 describes.
- A column absent from the row decodes to its zero value, as a null Scylla column does.
- `columnID` = low 16 bits of `crc32.ChecksumIEEE(columnName)`. A collision inside one table
  panics in `CompileTable`, naming both columns.
- Decoding a projection skips the columns not requested.

---

## 7. Extraction into `genix-orm/db` (Step 2)

Scylla's planner and write preparation mix storage-independent logic with CQL rendering. Move the
independent part; Scylla keeps rendering. **Every existing Scylla test stays green**, the moved
code gets its own unit tests in `db`, and the integration tests (`*OnScylla`) pass against the
local Scylla container.

> **Finding (2026-11-06, before any code moved): §7.1's function list is not extractable on its
> own.** Every function in it takes a `ScyllaTable` and reads its private compiled state
> (`views`, `indexes`, `packedIndexes`, `deltaIndexes`, `fixedValueRanges`, `compositeBucketIndexes`,
> `indexGroups`, `keyConcatenated`, `keyIntPacking`, `capabilities`, `computedColumns`: ~56 uses
> across the planner files). That state is built by the compile side (`index_view_compile.go` 859
> lines, `view_tables.go`, `computed_columns.go`, `index_config.go`, plus the registration inside
> `packed_indexes.go`, `delta_view.go`, `index_groups.go`), and `viewInfo` carries CQL-shaped
> closures (`getStatementPrepared` returns `[]boundWhereClause`, `getCreateScript`). Moving the
> planner therefore means first moving the **compiled-table model** into `db` and splitting
> `viewInfo` into a neutral part and a CQL part. See the options asked of the user in the session.

### 7.1 Planner

Move (renaming the package-local types to exported ones in `db`):

| From `scylla/` | What it is |
|---|---|
| `select_compute.go`: `GetQuerySignature`, `ComputeCapabilities` (the signature part), `MatchQueryCapability`, `capabilityOpForStatement`, `capabilityDefaultOpForColumn` | capability signatures and matching |
| `fixed_values.go`: all | FixedValues ranges, enumeration, slot planning |
| `select_helpers.go`: `buildFixedValueFanoutStatements`, `countSignatureCoveredQueryColumns`, `applyFixedValueFanout`, `canRewriteKeyConcatenated`, `buildKeyConcatenatedStatements`, `canRewriteKeyIntPacking`, `buildKeyIntPackingStatements`, `narrowInt64ToColumnWidth`, `chunkStatementInValues` | statement rewriting |
| `select.go`: `planCompositeBuckets`, `getBetweenRange`, `recordMatchesPostFilter`, `deduplicateRecordsByPrimaryKey`, `makePrimaryKeyRecordKey` | bucketing, post-filter, dedupe |
| `packed_indexes.go`, `index_int_packing.go` (layout only; not `storeVirtualPacked`) | packed slot layout |
| `select_compute.go`: `buildPackedPrefixRangeClauses` (the range math, not the CQL text) | packed-prefix spans |
| `delta_view.go`: `planDeltaPinnedSlots`, `resolveDeltaPrefixes`, `selectDeltaIndex`, `pinnedValuesOf`, `deltaIndexInfo.packedKeyValue` | delta prefix planning |
| `select_grouped.go`: `buildIndexGroupSelectPlan`, `scoreIndexGroupCandidate`, `buildQueryIndexGroupHashes`, `resolveIndexGroupQueryValues`, `splitIndexGroupFetches` | cache-group planning |
| `index_groups.go`: `computeIndexGroupHashes`, `buildIndexGroupHashValues`, `allocateIndexGroupID`, `collectIndexGroupMarks` (minus the stored-row read) | group hashes and marks |

The planner's output becomes a driver-neutral **access plan** made of **typed access steps**
(decided with the user, 2026-11-06: option C of three — key spans only, a portable predicate
list, typed steps). Spans alone cannot express a hash `IN` list or a collection `CONTAINS`; a
predicate list would make every driver re-derive the range math. Today that logic lives inside the
`viewInfo.getStatementPrepared` closures, which return CQL text; the planner moves it into `db` and
the closures shrink to rendering.

Refined with the user the same day, while implementing:
- **A value list per prefix column.** `a IN (1,2) AND b IN (3,4) AND c >= ?` stays one step:
  `EqualPrefix` holds one value list per leading key column, and the cartesian product is implied.
  Scylla renders `IN` with today's chunking, so it sends the same number of queries. FDB and YDB
  expand the lists into ranges or a lookup join. A full-key `Points` step is therefore a `KeyRange`
  with no bound, and `Points` is gone.
- **A separate `Filter`.** These are predicates the source key cannot serve but the server can
  still apply (Scylla `ALLOW FILTERING`, a YQL `WHERE`). The driver may push them to the server or
  apply them in memory. `PostFilter` stays "re-check in memory" (hash collisions, packed spill).
- **Native GROUP BY stays Scylla-only**, outside `AccessPlan`. FDB and YDB refuse it with
  `errNotImplemented` until a later step decides how to support it.

```go
// db/access_plan.go
type AccessPlan struct {
    Source     *IndexSource      // nil = the table's primary key
    Steps      []AccessStep      // read independently, rows unioned (deduplicated when Source can repeat a row)
    Filter     []ColumnStatement // not servable by the key; pushed to the server or applied in memory
    PostFilter []ColumnStatement // must be re-checked in memory (RequiresPostFilter, hash overfetch)
    Limit      int32
    Descending bool
}

type AccessStep struct { // exactly one is set
    Range   *KeyRange
    Element *ElementLookup
}

type KeyRange struct {
    EqualPrefix                []KeyValues // partition first, then the leading key columns
    BoundColumn                string      // "" = prefix only (point / prefix lookups)
    From, To                   any         // nil = open on that side
    FromInclusive, ToInclusive bool
}

type KeyValues struct { Column string; Values []any } // one value = equality, several = IN

type ElementLookup struct { Column string; Value any } // a collection CONTAINS
```

Values in a step are the **neutral** layout (`IndexSource` describes the column: unsigned packed
values as `uint64`, `int32` hashes); each driver applies its own storage encoding while rendering
(Scylla: the top-bit flip of `storeVirtualPacked`, in `encodeAccessValue`). Rendering per driver:

| Step | Scylla (CQL) | FDB | YDB |
|---|---|---|---|
| `Range` | `=` / `IN (…)` per prefix column (chunked), then `>=`/`<=` on the bound | one tuple range per prefix combination | `WHERE` on the PK or `VIEW ix`; value lists via `IN` / `AS_TABLE($keys)` lookup join |
| `Element` | `CONTAINS ?` | index-entry prefix range | side-table range `element = $v`, joined back on the PK |
| `Filter` | ANDed into every step's `WHERE` | applied in memory | ANDed into the `WHERE` |

Keep the plan cache (`selectPlanCache`) in each driver, keyed by `computeSelectShapeHash`, which also
moves. A plan the planner refuses (a filter behind an unconstrained packed slot with no
FixedValues) is `nil, error`, as `getStatementPrepared` returns `nil` today.

### 7.2 Write preparation

Move from `insert-update.go` and neighbours: `applyWriteManagedColumns`, `resolveCreatedValue`,
`runSelfParseIfDefined`, `handlePreInsert` (with the counter fetch behind the existing
`ReserveCounterRange` hook), `collectAffectedColumnsFor{Insert,Include,Exclude}`,
`resolveUpdateColumnsForWrite`, `hasUsableIndexSourceValue`, `includeComputedColumns`,
`storedColumnsOf`, `splitInsertAndUpdateRecords`, `normalizeEmptyStringWriteValue`, and the
"columns updated together" check. Both drivers call them; only statement building stays in Scylla.

Do not move code that only renders CQL, or that reads Scylla. If a function mixes both, split it at
the read/render boundary and move the pure half.

---

## 8. FDB write algorithm

Applies to `Insert`, `Update`, `UpdateExclude`, `InsertUpdate*` and `Merge`. Records are processed
in **chunks**; a chunk is the unit of atomicity.

1. **Prepare** (shared code from §7.2): self-parse, managed columns, autoincrement / key packing,
   computed columns, the affected-column set.
2. **Pre-read**: one `GetMany` of the head row of every record in the chunk (plus its chunks when
   flagged). This single read replaces Scylla's `readStoredWrittenRows`, `readStoredRows` and
   `preloadExistingRecordsBySingleKey`.
3. **Build the new row**:
   - Insert: every stored column of the record (minus `columnsToExclude`).
   - Update: the stored row with the included columns overwritten. A record not stored yet gets a
     row with only the written columns plus keys (Scylla's `UPDATE` is an upsert; keep that).
   - `rowVersion = stored + 1` (or 1).
   - TypeDelta: stamp the managed `version` and compute the packed delta key from the **final**
     pinned values (stored value when the update does not write the column), as
     `stampDeltaWrites` does.
4. **Diff index entries**: compute the entry keys of the stored row and of the new row, for every
   index kind (§6.2). `Clear` the ones that disappear, `Set` the new ones. Unchanged entries are
   not rewritten.
5. **Derived rows, same commit**:
   - TypeDelta whose pinned prefix changed → `Set` a `'m'` move row (old prefix, ID).
   - Cache groups: for every group the record leaves or enters → `AtomicMax` on its `'g'` row
     with `last_write = now`.
   - DataFrames: the frame log entry (before values of the changed columns, or the insert flag),
     computed exactly as `writeDataFrameLog` does.
6. **Commit** with one condition per record: `ValuePrefixEquals(rowKey, storedHead[:8])` when it
   existed, `KeyAbsent(rowKey)` when it did not. Size the chunk so the commit stays under 1 MB and
   5,000 mutations; one record never spans chunks.
7. **Condition failed** → re-run steps 2–6 for that chunk only, up to 5 times, then return an error
   naming the table and key. For `Merge`, `onUpdate` / `onInsert` run again on the re-read rows.
   That is the documented semantics: the callback must be pure.
8. **After the commit** (marker phase, same order as Scylla): by-IDs `cache.RaiseWrittenSlots` on
   the file store, then the text-search sync (`syncTextSearchAfterWrite` logic through
   `db.TextSearchBackend`).

What gets stronger than Scylla: entries, moves, freshness marks and the frame log commit with the
row, so there is no view lag, no replica lag, and no "mark before data" window. Keep the cache
protocols' settle margins anyway (S9).

---

## 9. FDB read algorithm

1. Shared planner → `AccessPlan` (§7.1). A plan the planner refuses (a filter behind an
   unconstrained key with no FixedValues) fails exactly as it does on Scylla.
2. Render each `KeySpan` to an FDB range on the source's subspace. The limit per range is
   `Limit + overfetch` when a post-filter exists, else `Limit`.
3. Primary key source → `GetRanges`. Index / delta / global source → `IndexRangeGet`.
   All spans of one plan go in **one** request, so a fan-out reads at one read version.
4. Follow `More` / `NextBegin` continuations with further requests. Each page is its own
   transaction (no snapshot across pages, same as Scylla paging). Default page `ByteLimit` 1 MB.
5. Decode the projected columns only. Apply the post-filter, dedupe by primary key, merge spans in
   the requested order, cut at `Limit`.
6. `SelectGrouped` (native group by): the same read, aggregated in the driver with the moved
   `buildNativeGroupByPlan` semantics.

---

## 10. Feature parity checklist

Each row is done when its conformance test (§12) passes on both drivers.

| Feature | Scylla source of truth | FDB notes |
|---|---|---|
| Keys, partition, `KeyConcatenated`, `KeyIntPacking`, `AutoincrementPart`, `AutoincrementFrom` | `select_helpers.go`, `insert-update.go` | Shared code from §7; tuple keys |
| Autoincrement, `UseSequences`, `SequenceColumn`, `CounterColumn`, `GetAutoincrementID`, `ResetCounter` | `main.go`, `deploy.go`, `autoincrement_scylla.go` | `ReserveRange` / `SetCounter`; D3 |
| Managed columns, `DisableDefaultColumns` | `insert-update.go` | Shared code; stored by the row codec even without a struct field |
| `TypeLocalIndex`, packed local index | `index_config.go`, `packed_indexes.go` | `'i'` entries |
| `TypeGlobalIndex` (scalar and slice) | `index_config.go` | `'G'` entries, one per element |
| `TypeView` hash / range / with `Cols` | `index_view_compile.go` | `'i'` entries; payload read from the record |
| `TypeViewTable` | `view_tables.go` | `'i'` entries per fan-out element |
| FixedValues fan-out | `select_helpers.go` | Shared planner |
| Composite bucketing | `select.go` | Shared planner; one entry per bucket element |
| `Computed` columns | `computed_columns.go` | Same `From` rule; feeds indexes, never stored |
| `TypeDelta`: pinned keys, moves, threshold, `Limit`, projection, Metadata record | `delta_view.go` | `'d'` / `'m'`; `cache.PlanDeltaSync` unchanged |
| Delta snapshots | `delta_snapshot.go` | Unchanged (file store); port the read/serve/write calls |
| `CacheByIDs`, `QueryCachedIDs`, `QueryCachedGenericByIDs`, `GenericRecord` | `cache_by_ids*.go` | Slot files unchanged; records by `GetMany` |
| `TypeCacheGroup`, `TypeCacheGroupFromKey`, `TypeCacheGroupS3`, `QueryIndexGroup` | `index_groups.go`, `select_grouped.go`, `cache_group_snapshot.go` | `'g'` rows; `FromKey` reads a primary-key range |
| `SelectGrouped` | `select_compute.go` | Driver-side aggregation |
| `Merge`, `InsertUpdate*`, include/exclude updates | `merge.go`, `insert-update.go` | §8 |
| `TextSearch`, `SearchText`, `SearchTextIDs` | `text_search_index.go`, `text_search_query.go` | Same backend interface; FDB scan for `Rebuild` |
| `DataFrames`, frame log, `AcquireFrameLock`, compaction | `data_frame*.go` | `dataframe.Backend` + `PartitionRecordScanner`; state stays in the file store, lock stays on fareward |
| `db.Controller` (export/restore colbin, reload, recalc, reset counter, delete indexes, text-search maintenance) | `controller_generic.go`, `backup.go`, `scylla/main.go` | `FdbController[T, E]`; recalc = clear the partition's index subspaces and rewrite |
| Deploy / homologation | `deploy.go`, `init.go` | No DDL. Writes `'_tables'` rows; verifies `indexID` stability |
| Wipe company data, list tables, column rename | `exec/wipe_company_data.go`, `exec/column_rename.go` | `ClearPrefix`; table registry; rename = re-encode rows (column IDs change) |

---

## 11. Steps

Each step ends with its acceptance check, a commit inside each touched submodule (pushed), and a
root commit for `backend/` changes.

### Step 0 — Container and daemon skeleton
- §5 container; `fdbgate` with `Ping`, `GetMany`, `GetRanges`, `Commit`; `fdb/client.go` and
  `fdb/protocol.go`.
- Microbenchmark (`fdb/client_bench_test.go`, gated by `GENIX_FDB_BENCH=1`) against a second
  benchmark that uses the Apple binding in-process, inside `fdbgate` (its own `_test.go`):
  1 get, a 1,000-key range, a 1,000-mutation commit, at concurrency 1 and 64.
- **Accept:** the container starts from a clean volume and the database is configured; the
  numbers are written into §14. **Stop and report** if the socket hop adds more than ~2x to a
  single get or more than ~20% to the 1k range.

### Step 1 — Neutral facade, Scylla only (no behavior change)
Add to `backend/db` (Scylla implementation in `driver_scylla.go` or a sibling `*_scylla.go`, FDB in
`*_fdb.go`):

| Facade function | Replaces |
|---|---|
| `db.Connect(params ConnectParams)` | `scylla.SetScyllaConnection` (`main.go:369`, `tests/bench/setup.go:54`), `scylla.MakeScyllaConnection` (`exec/init.go`, `demo.go`, `wipe_company_data.go`, `column_rename.go`, `text_search.go`, `data_frames.go`, `observability_schema.go`), `exec/test.go` `makeConnParams` |
| `db.Deploy(controllers ...Controller)` | `scylla.Init` + `CreateKeyspaceIfNotExists` + `DeployScylla` (`exec/init.go:138,387,391`, `observability_schema.go:28`, `demo.go:549`) |
| `db.MakeController[RecordT, TableT]()` | `exec/migrate.go` `makeDBController` (`scylla.MakeScyllaTable`, `ScyllaController`, `RecordOf`) |
| `db.ListTables(keyspace)`, `db.WipePartition(table, partition)` | `wipe_company_data.go:85` `system_schema` scan, `:154` raw deletes |
| `db.SetCounter(name, value)` | `wipe_company_data.go:175` `scylla.SetCounterValue` |
| `db.RenameColumn(table, from, to)` | the raw CQL in `column_rename.go` |
| `SetDataFrameLock` | already in `driver_scylla.go:37`; add the FDB twin |

- `ConnectParams` holds what both drivers need: Scylla host/port/user/password/keyspace/max
  clustering key, FDB socket path, keyspace.
- `exec/generic_records_test.go` downcasts `scylla.ScyllaTable`: give it `//go:build !fdb`.
- D4 code: port or delete, per the user's answer.
- **Status (2026-11-05): code done, live acceptance pending.** As built, in `backend/db`:
  `connect.go` (`ConnectParams`, with `Scripting` choosing eager dial + long timeouts),
  `facade_scylla.go` (`Connect`, `Deploy` = Init + homologate, `MakeController` replacing
  `exec/migrate.go` and the generator's `makeDBController`, `ListCounters`, `WipePartition(TableDescription,
  partition)` which also clears the side tables and by-IDs slots), `SetCounter` in
  `autoincrement_scylla.go`, and `column_rename_scylla.go` (the whole Scylla-catalog migration moved
  out of `exec`, entry point `db.RenameColumns(apply)`). `db.ListTables` was **not** added: its only
  caller folded into `WipePartition`. D4 ported: `exec/demo.go` only had `DROP MATERIALIZED VIEW`
  inside comments; `fn-recalc` now calls `db.WipePartition` on `almacen_producto` / `almacen_movimiento`,
  **tables that no longer have a declaration** (the call finds nothing; delete it when convenient).
  Verified: `go build ./...`, `go vet ./...`, `go test ./db ./exec .` (incl. the GOOS=js dependency
  test), `go test ./...` in `backend/`, and a `GOOS=js GOARCH=wasm` build. Live, against the load
  stack's Scylla (127.0.0.2:9042, throwaway keyspace, dropped afterwards; `genix-bench-scylla` cannot
  start while `genix-load-scylla` holds the host-network supervisord port): `fn-init` twice (exit 0,
  idempotent, 123 tables), `fn-wipe-company-data` simulation (`ListCounters`) and `--apply`
  (`WipePartition` emptied the rows; it then stopped at `SetCounter` because no fareward was
  running, as before), `fn-rename-columns` report (`RenameColumns`).
  **Not run:** bench Layer A (needs the bench stack and fareward).
- **Accept:** `rg -n 'scylla\.' backend --type go -g '!genix-orm/**'` matches only
  `backend/db/*_scylla.go` (and comments). `go build ./...` and `go vet ./...` pass;
  `go run . fn-init` against the bench Scylla works; the bench Layer A runs.

### Step 2 — Extraction into `genix-orm/db` (§7)
Decided with the user (2026-11-06): **full extraction, staged**, every stage keeping all Scylla
tests green (unit + `GENIX_ORM_SCYLLA_TEST` integration):
1. **Leaf algorithms — done.** `db/planner_leaf.go` (+ `planner_leaf_test.go`): `ConvertToInt64`,
   `HashInt`/`HashInt64` (golden-value tests), week-code and composite-bucket math
   (`NormalizeCompositeRange`, `CompositeBucketID`, `PlanCompositeBuckets`, `BetweenRange`,
   `StatementValuesForHash`, `FlattenInt64Values`), `ColumnValueRange` / `EnumerateFixedValues`,
   `ChunkValues`, `NarrowInt64ToColumnWidth`, `PackRowValues` / `SumSlotBits` /
   `PackedVirtualBudgetBits`. The week-code memo became a `sync.Map` (it was a plain map written
   from request goroutines). `scylla/week_helpers.go` removed.
2. **Compiled-table model.**
   - **2a — neutral compiled state — done.** `db/compiled_state.go`: `CompositeBucketIndex`,
     `IndexGroupInfo`, `PackedIndexInfo`, `DeltaIndexInfo`, `FixedValueSlotPlan`, now fields of
     `db.TableCore` (`KeyConcatenated`, `KeyIntPacking`, `KeyIntPackingSlotBits`, `FixedValueRanges`,
     `PackedIndexes`, `DeltaIndexes`, `CompositeBucketIndexes`, `IndexGroups`, `IndexGroupIDs`), plus
     their neutral functions (`PlanFixedValueViewSlots`, `ResolveFixedValueRanges`,
     `PlanDeltaPinnedSlots`, `AllocateIndexGroupID`, group/composite hashing, `BasicHashInt`).
     `DeltaIndexInfo.PackedKey` is the shared **unsigned** layout; Scylla re-maps it with
     `deltaPackedStoredValue` (top-bit flip). Two copies of the same "flatten to int64s" function
     collapsed into `FlattenInt64Values`.
   - **2b — in progress.** Done: `viewInfo` embeds `db.IndexSource` (`db/index_source.go`: type,
     name, columns, packed layout, operators, post-filter flag, view-table fan-out) and keeps only
     its CQL closures plus view-table maintenance fields. `db.ComputedColumnInfo` +
     `ConfigureComputedColumns` / `IncludeComputedColumns` / `StoredColumnsOf` /
     `RejectComputedStatements` (`db/computed_columns.go`, moved with `git mv`) and
     `db.TextSearchIndexInfo` + `ConfigureTextSearchIndex` (`db/text_search_info.go`) are on
     `TableCore`. Pending: the compile code itself (`makeTable`, `index_view_compile.go`,
     `index_config.go`, `packed_indexes.go`, `view_tables.go`, the registration halves of
     `delta_view.go` / `index_groups.go`); its CQL closures capture compile-time locals at ~18
     sites, so the mechanism is a user decision. `QueryCapability` moves in stage 3.
   - **2b mechanism decided (2026-11-06, user): two-phase.** `db` describes (`Describe*` methods on
     `TableCore` fill neutral state and return an `IndexSource`); the driver materializes in a second
     pass that reads the `IndexSource`, never captured compile-time locals. A virtual column's value
     function returns the **neutral** layout (unsigned packed value, int32 hash) and the driver
     re-encodes it (Scylla: `storeVirtualPacked`, top-bit flip). This is the pattern
     `describe_index.go` already used for local, global and packed indexes.
   - **2b progress (2026-11-06).** The index half of the compile is **done**: `db.DescribeIndexes`
     (`db/describe_indexes.go`) is the schema-index dispatcher and returns the `IndexSource`s a driver
     must materialize. It calls `DescribeViewTable`, `DescribeView` (simple, hash, packed range),
     `DescribeDelta`, `DescribeIndexGroup` (+ the key-range group), the composite-bucket describe,
     and the existing local, global and packed ones. Every `IndexSource` carries a `Kind`
     (`SingleColumn`, `Packed`, `GroupHash`, `View`, `ViewTable`, `Delta`, `CompositeBucket`); Scylla's
     `makeTable` index loop is `DescribeIndexes` + `registerMaterializedIndex` per source, and every
     `compileSchema*` / `registerSchema*` wrapper is gone. `IndexSource` gained `DeclaredColumns`,
     `ProjectedColumns`, `SelectableColumns`, `PartitionColumn`, `KeepsBasePartition`,
     `MaintenanceIDColumn`, `RebuildColumnNames`, `IndexGroup`. Verification per stage: unit +
     `GENIX_ORM_SCYLLA_TEST` integration green, and a temporary probe dumping every genix table's view
     CREATE scripts, expected columns, virtual-column types, delta layouts, groups, composite buckets,
     index/view counts and flags from the old and the new code, normalized for map order and the LZ4
     work in progress: identical (68 views, 51 delta indexes, 24 groups).
     The table-level half is **done** too: `TableCore.CompileFromSchema` (`db/compile_table.go`:
     columns, keys, managed audit columns, autoincrement, `KeyIntPacking`, `KeyConcatenated`) and
     `TableCore.FinalizeColumns` are shared. Scylla's `makeTable` and `fdbdriver/compileTable` both call
     them (the FDB copy is gone) and keep only their own details inline: Scylla's `counter` DB type of
     a sequence column, the quoted-literal value of a concatenated key, and the default keyspace. The
     same probe showed the old and new compile give identical results for all 1,400 columns. The FDB
     live suite passes 5 of 5 runs. It had been failing because of a keyspace bug in the test, now
     fixed with `TestMain`.
     **Left in 2b (optional cleanup):** `ScyllaTable.views/indexes` should read `TableCore.IndexSources`
     instead of keeping their own maps. This does not block stage 3.
3. **Planner + `AccessPlan`** (§7.1) and write preparation (§7.2) on top of stage 2.
   - **3a — done (2026-11-06).** `db/access_plan.go` holds the types and `PlanSourceSteps`, the
     per-source planners that replaced the six `getStatementPrepared` closures: simple view (6) and
     view table (9) via `planKeyPrefixStep`, hash view (7), packed range and delta views (8),
     packed index, and cache-group hash. Also there: `FindIndexGroupStatement` and
     `ResolveIndexGroupQueryValues`, moved from Scylla. Scylla now only renders
     (`scylla/access_render.go`: `viewInfo.prepareWhereClauses`, `renderAccessSteps`,
     `encodeAccessValue`). A delta source now records its `PartitionColumn`.
     **Verification:**
     - A probe rendered every genix view (78 sources) and fixtures for packed indexes, view tables
       and projected views. Each ran over the same set of query shapes (all-eq, IN per column,
       skip per column, BETWEEN per column, one-sided bounds, CONTAINS), in the old and the new
       code. Output was normalized for predicate order and `IN (?)` vs `= ?`, and was identical.
     - Suites green: `db` unit (6 new planner tests), Scylla unit plus `GENIX_ORM_SCYLLA_TEST`
       integration, FDB live, YDB unit.
   - **3b, first slice — done (2026-11-06).** These moved to `db`:
     - `db/capabilities.go`: `QueryCapability` (its `Source` is now `*db.IndexSource`),
       `TableCore.ComputeCapabilities`, `MatchQueryCapability`, `CapabilityOpForStatement` and
       `TableCore.ApplyFixedValueFanout`.
     - `db/key_rewrites.go`: `TableCore.BuildKeyConcatenatedStatements` and
       `BuildKeyIntPackingStatements`.

     Scylla maps the chosen source back to its `viewInfo` (`viewOfSource`). Route selection and
     rendering stay in `compileSelectStatement` / `Compute` for now. Single-column and packed index
     capabilities are now listed in declaration order (before: Go map order), so ties between
     equal priorities no longer change between runs.
     **Verification:** a route probe built a query per capability signature, plus variants (drop
     the last predicate, add an extra column), for every genix table and 20 scylla fixtures. That
     covers plain, view, concatenated-key, int-packed and composite routes. It compiled and bound
     each one; old vs new output was identical. The 3a access probe was also identical, except
     where the schema itself changed. Suites green: `db`, Scylla unit plus integration, FDB live,
     YDB.
   - **3b, rest — done (2026-11-06).** Three slices, each verified old vs new with a probe:
     - **Routes** (`db/access_route.go`). `TableCore.ChooseAccessRoute` picks primary key /
       source / concatenated key / int-packed key / composite bucket / native index and returns
       an `AccessRoute`: the statements it serves, keeps and post-filters, its fixed-value
       fan-out, and whether results need deduplication. `TableCore.BindAccessPlan` turns a route
       plus the query values into an `AccessPlan`; composite buckets are planned there
       (`planCompositeBucketSteps`). Scylla's `compileSelectStatement` keeps only the CQL
       (`renderAccessPlan`). Native GROUP BY stays a Scylla route. Removed from Scylla:
       `tryBuildCompositeBucketPlan`, `canUseProjectedView`, `collectSelectStatements` (now
       `TableInfo.SelectStatements()`), `pickStatementsByIndexes`.
     - **Delta** (`db/delta_plan.go`). `TableCore.PlanDeltaRead` chooses the TypeDelta index and
       its pinned prefixes. Both Scylla `execDelta` and FDB `selectDelta` use it, and both
       drivers' own `resolveDeltaPrefixes` / `selectDeltaIndex` are gone. FDB's cap on fixed-value
       fan-out drops from 16 to `db.MaxFixedValueFanoutQueries` (8), the same as Scylla's;
       `maxDeltaPrefixReads` (16) now only bounds its read concurrency.
     - **Cache groups** (`db/index_group_plan.go`). `TableCore.PlanIndexGroupRead` does the
       validation, the group scoring and the hash fan-out. Also in `db`: `BaseGroupHashes`,
       `SplitIndexGroupFetches`, `GroupPostFilterStatements` and `TableCore.IndexGroupFetchStep`
       (one group's base-table read as an `AccessStep`). Scylla keeps the `__index_updated`
       freshness CQL, the S3 snapshots and the concurrent fetch. The `TypeCacheGroupFromKey`
       fetch is still bound as `[from, to)`, although `PackSlotPrefixBound(true)` is an inclusive
       maximum. That is a pre-existing quirk, kept as it was: a record whose key is exactly the
       block's last value would be missed.

     The select-shape cache (`computeSelectShapeHash`) stays per driver, because it caches
     rendered queries.
     **Verification:**
     - Route probe over every genix table plus the fixtures; group probe over every genix cache
       group (37) and 13 fixture reads, including no-index groups and collection groups. Old vs
       new output was identical for both.
     - Suites green: `db`, Scylla unit plus integration, FDB live, YDB unit, and the backend
       builds (no tag, `fdb`, `ydb`).
   - **§7.2 write preparation — done (2026-11-06).**
     - `db/record_slice.go`: `RecordSlice`, moved from Scylla. FDB's own copy is gone.
     - `db/write_prepare.go`: `ManagedUnixTime`, `StampManagedColumns` (returns `[]ManagedStamp`),
       `CoerceManagedInteger`, `RunSelfParse`, `AssignAutoincrementKeys` (the driver passes its
       counter allocator: Scylla `reserveCounter`, so the `ReserveCounterRange` hook still
       applies; FDB `ReserveRange`), `SplitInsertAndUpdateRecords`, `InsertColumns`,
       `AffectedColumnsFor{Insert,Include,Exclude}`, and `ResolveUpdateColumns`, which carries
       the partial-update refusals.

     Scylla keeps only statement building. `normalizeEmptyStringWriteValue` stays there too,
     because storing `""` as NULL is a CQL binding rule.

     FDB derives its written columns from `ResolveUpdateColumns` (stored columns only; a
     computed column becomes its From columns). Decided with the user: **same contract**. FDB
     now refuses the partial updates Scylla refuses, although its row merge would not need to,
     so a write accepted on one driver is accepted on all. FDB also refuses an Update with no
     columns now; before, it wrote every column.
     **Verification:**
     - Probe: old `resolveUpdateColumnsForWrite` vs `ResolveUpdateColumns` over every genix
       table, covering single-column and per-source includes and excludes, with and without a
       record: 4,324 cases, 290 of them refusals. Identical, including the panic messages.
     - Suites green: `db`, Scylla unit plus integration, FDB live (plus a new test of the
       written-column contract), YDB unit, and the backend builds and vet for the three tag sets.
   - **Step 2 is complete.** Next: YDB Step 2.
- **Accept:** `go test ./...` in `genix-orm` (`db`, `scylla`) green, including the integration tests
  against the local Scylla; new unit tests in `db` cover `AccessPlan` building for: primary key, IN
  fan-out, packed prefix + range, FixedValues fan-out, delta prefixes, cache-group candidates, the
  "filter behind unconstrained key" refusal.

### Step 3 — Conformance suite
- New package `genix-orm/conformance` (own module, imports `db`, `scylla`, `fdb`). One set of
  test table declarations covering every row of §10, written once and run per driver through
  `.Via(executor)` (the facade already supports per-query executors: `TestViaRoutesQueryToChosenDriverAtRuntime`).
- Each test writes the same records to both drivers, runs the same queries, and compares sorted
  results field by field. Cache protocols compare decoded responses (records + Metadata), with
  the clock frozen via `db.SetHistoricalUnix`.
- Start from the Scylla integration tests: `TestCacheProtocolsOnScylla`,
  `TestControllerPathsOnScylla`, `TestMergeOnScylla`, `TestQueryIndexGroupOnScylla`,
  `TestTextSearchOnScylla`, `TestInsertUpdateIncludeOnScylla`, `cache_integration_test.go`.
- Gate with `GENIX_CONFORMANCE=1`; needs both containers.
- **Accept:** green on Scylla vs Scylla (self-check), red on FDB with "not implemented" errors.

**Status (2026-11-06): built** (results in `YDB_PLAN.md` §18b, Step 3). The drivers cannot share a
binary, so the suite compares each one against golden files recorded on Scylla, not through
`.Via`. On `-tags fdb` the acceptance is **not met**: besides the "not implemented" failures it
finds four wrong answers:
- delta snapshots are never stored;
- `QueryRecordsJSON` ignores `OrderDesc` / `Limit`;
- an empty list reads back as `null`, not `[]`;
- writes never reach the text-search backend.

### Step 4 — Driver core (ask D1 first)
- `FdbTable`, `CompileTable` for every table genix declares (it must not panic on any of them),
  the row codec, `Codec`, keys, managed columns, sequences (D3), §8 writes, §9 reads, every index
  kind, computed columns, chunking.
- `backend/db/driver_fdb.go` + `autoincrement_fdb.go`; `backend/go.mod` require/replace.
- **Accept:** `GOFLAGS=-tags=fdb go build ./...` in `backend/`; conformance rows for keys,
  sequences, writes, merge, every index kind, computed columns and chunking are green;
  `go run -tags fdb . fn-init` and `GENIX_BENCH=1 go test -tags fdb ./tests/bench/ -run TestSeedBenchData`
  pass (D2 decides fareward's part).

**Status (2026-11-06): partial, enough for the benchmark.** `genix-orm/fdbdriver`:
- Done: `FdbTable` compile (columns, managed columns, computed columns, FixedValues, keys,
  autoincrement, `KeyIntPacking`, `KeyConcatenated`, TypeDelta), the D1 row codec (all column types,
  projection decode), keys (own order-preserving encoding, not FDB tuples: the tuple package lives in
  the cgo binding), §8 writes with pre-read, partial-update merge, conditional commit and conflict
  retry, sequences (D3), plain reads (point reads on a pinned key, partition range scans with exact
  in-memory filtering, first-key narrowing, `OrderDesc`, `Limit`, projection), a minimal
  `FdbController` (describe, JSON query/insert/update), `backend/db/driver_fdb.go` + `facade_fdb.go`,
  `[fdb] socket` → `core.Env.FDB_SOCKET`. `go build -tags fdb ./...` in `backend/` passes.
- Tests: `fdbdriver/driver_test.go` (codec round trip, key order, and a live test of writes, reads,
  partial updates, delta first/incremental sync and moves, gated by `GENIX_FDB_IT=1`).
- **Not done** (each fails loudly with "not implemented on FoundationDB yet", or is skipped on write):
  entries for every index kind other than TypeDelta (reads never use them, they scan the partition),
  global-index reads without a partition, `SelectGrouped` / cache groups, by-IDs cache and generic
  by-IDs, `Merge`, text search sync and queries, DataFrames and their log, delta snapshots, chunked
  rows (> 100 KB fail), `SequenceColumn` / `CounterColumn`, GroupBy, the controller's backup /
  restore / recalc / reindex, `ResetCounter`, `RenameColumns`. The Step 2 extraction is not finished,
  so the FDB driver carries its own copies of delta prefix resolution and the compile (marked in code).
- The conformance suite (Step 3) exists: `GENIX_CONFORMANCE=1 FDBGATE_SOCKET=… go test -tags fdb .`
  in `genix-orm/conformance`.

### Step 5 — Cache protocols
- Delta, delta snapshots, by-IDs, generic by-IDs, cache groups (three types), group snapshots.
- **Accept:** their conformance rows green; the three bench endpoints answer the same bytes on both
  drivers for a first sync and an incremental sync (compare decoded responses).

### Milestone — Benchmark
- `bench.config.toml` gains `[fdb] socket = "/run/user/<uid>/genix-fdb/fdbgate.sock"`, read into
  `core.Env.FDB_SOCKET` (add it to `config.example.toml` too).
- `tests/bench/setup.go`: `db.Connect(...)` from Step 1 already covers both drivers.
- README: the FDB container commands next to the Scylla ones.
- Run Layer A and Layer B exactly as `tests/bench/README.md` documents, with and without `-tags fdb`,
  same `-cpu`, same datasets. Fill §14.

### Step 6 — Remaining parity
- `SelectGrouped`, text search, DataFrames and their log, `FdbController` (backup/restore, recalc,
  reload, delete indexes), wipe, column rename, admin `exec/` scripts.
- **Accept:** every §10 row green; `go test -tags fdb ./...` in `backend/` green; `exec/` scripts run
  under `-tags fdb`.

### Step 7 — Documentation
- `genix-orm/fdb/README.md` (storage model, protocol, limits), `fdb/RATIONALE.md` (decisions with
  Context/Decision/Rationale, like `scylla/RATIONALE.md`), update `INTERNALS.md` (§3 planner in `db`,
  §10 conformance suite now exists), update `CLAUDE.md` repository map.

### Step 8 — fareward (per D2)
- Out of the benchmark milestone unless D2 says otherwise.

---

## 12. Testing rules

- Unit tests next to the code, no database: codec round-trips (every column type, chunked rows,
  projection decode), key ordering (tuple order = intended numeric order for every packed layout),
  index entry diff, `AccessPlan` rendering to ranges.
- The daemon has its own tests with a real FDB (gated by `GENIX_FDB_IT=1`): every op, every
  condition kind, conflict retry (two concurrent `ReserveRange` never return overlapping ranges),
  timeout → status 4, oversized commit → status 4.
- Integration and conformance tests are gated by env vars and skip otherwise, as the bench does.

---

## 13. Risks

| Risk | Mitigation |
|---|---|
| The socket hop dominates small reads | Step 0 measures it before anything else is built |
| The planner extraction breaks Scylla | Step 2 runs every Scylla test, integration included, before FDB code depends on it |
| Hot keys: a popular cache-group `'g'` row or sequence conflicts under load | `AtomicMax` / `AtomicAdd` never conflict on write; `ReserveRange` reserves in blocks (the facade asks for ≥ 1 ID per record; batch inserts ask once per chunk) |
| A large `Insert` is not atomic across chunks | Same as Scylla's unlogged batches today; documented in `fdb/RATIONALE.md` |
| Range reads over 5 s | Pages bounded by `ByteLimit` and the daemon's 4.5 s timeout |
| Index declaration changes leave stale entries | `indexID` includes a hash of the key columns; deploy reports a changed ID and the agent runs `RecalcVirtualColumns` |

---

## 14. Results

### Step 0 microbenchmark (2026-11-05, 12-thread laptop, one `fdbserver`, `ssd-redwood-1`, 100-byte values)

Same workloads, same server, same run. "binding" = Apple binding in-process inside the container
(`fdbgate.test`); "socket" = host client → unix socket → `fdbgate` → FDB.

| workload | concurrency | binding | socket | socket / binding |
|---|---|---|---|---|
| get 1 key | 1 | 139 µs | 214 µs | 1.54x (limit 2x: ok) |
| get 1 key | 64 | 28.6 µs/op | 42.8 µs/op | 1.50x |
| range 1,000 rows | 1 | 525 µs | 966 µs | **1.84x (limit 1.2x: exceeded)** |
| range 1,000 rows | 64 | 222 µs/op | 295 µs/op | 1.33x |
| commit 1,000 sets | 1 | 6.05 ms | 6.70 ms | 1.11x |
| commit 1,000 sets | 64 | 8.90 ms/op | 10.93 ms/op | 1.23x |

**Stop condition hit** (§11 Step 0: >20% on the 1k range). Not yet diagnosed. Suspects, in order:
colbin encode/decode of a 1,000-entry `ByteList` (both ends), the extra copy `readRange` makes into
the reply, and the per-frame allocation in `ReadFrame`/`WriteFrame`. The two sides also do different
work (the binding materializes `[]KeyValue`; the client decodes a colbin reply), so a fair
comparison may need the binding side to encode the same reply.

### Benchmark milestone — Layer A (2026-11-06)

Setup: i7-1355U laptop. **Scylla**: `genix-load-scylla` (127.0.0.2:9042, `--smp 2 --cpuset 0,1
--memory 1500M --overprovisioned 1`), keyspace `genix_bench`. **FDB**: `genix-bench-fdb`, one
`fdbserver` (`--memory 2GiB --cache-memory 1GiB`, ssd-redwood-1) + `fdbgate`, both pinned to cores
2,3 with `taskset`. Bench process on cores 4–11. `GENIX_BENCH_NO_SNAPSHOTS=1` (first syncs hit the
database), `GENIX_BENCH_NO_FAREWARD=1` (D2). Same seed on both. The colbin response bytes are
identical on both drivers for all nine endpoint × dataset pairs. Raw files: `/tmp/genix-bench/`.

Serial, one request at a time (`-benchtime 20x -count 3`, medians; ms):

| endpoint | dataset | scan Scylla | scan FDB | handler+zstd Scylla | handler+zstd FDB | incremental Scylla | incremental FDB |
|---|---|---|---|---|---|---|---|
| products | 1k | 10.4 | 3.4 | 12.0 | 4.8 | 2.48 | 0.48 |
| products | 10k | 93.5 | 29.8 | 105.9 | 42.2 | 2.48 | 0.43 |
| products | 50k | 478.8 | 143.4 | 531.6 | 221.6 | 2.47 | 0.42 |
| sale-orders | 1k | 6.0 | 2.8 | 7.5 | 4.1 | 2.58 | 0.66 |
| sale-orders | 10k (5000 cap) | 27.5 | 13.7 | 31.7 | 17.7 | 2.58 | 0.62 |
| sale-orders | 50k (5000 cap) | 28.0 | 14.1 | 31.4 | 17.2 | 2.58 | 0.58 |
| products-stock | 1k | 5.6 | 1.7 | 6.0 | 2.2 | 2.44 | 0.39 |
| products-stock | 10k | 46.4 | 14.0 | 49.4 | 18.9 | 2.47 | 0.39 |
| products-stock | 50k | 224.4 | 70.1 | 245.5 | 92.1 | 2.43 | 0.38 |

Geomean of every serial case: FDB −67% time. Allocations −45% on products, B/op +8–15% on first
syncs (the FDB rows are decoded from one buffer per page).

`HandlerConcurrent` (first sync, `-cpu 4 -benchtime 400x -count 2`; req/s, p99 ms):

| case | Scylla req/s | FDB req/s | Scylla p99 | FDB p99 |
|---|---|---|---|---|
| 1k products c16 / c128 | 252 / 246 | 664 / 920 | 81 / 532 | 31 / 205 |
| 1k sale-orders c16 / c128 | 518 / 558 | 826 / 1202 | 38 / 236 | 27 / 160 |
| 1k products-stock c16 / c128 | 552 / 561 | 1218 / 1825 | 40 / 230 | 23 / 128 |
| 10k products c16 / c128 | 23 / 34 | 86 / 95 | 871 / 3860 | 226 / 1947 |
| 10k sale-orders c16 / c128 | 111 / 111 | 208 / 264 | 219 / 2160 | 101 / 729 |
| 10k products-stock c16 / c128 | 54 / 54 | 170 / 216 | 346 / 2564 | 115 / 840 |
| 50k products c16 / c128 | 4.6 / 23 | 16 / **fail** | 5166 / 7343 | 1176 / — |
| 50k sale-orders c16 / c128 | 113 / 110 | 209 / 249 | 228 / 2103 | 100 / 739 |
| 50k products-stock c16 / c128 | 11 / 28 | 34 / 41 | 2068 / 4573 | 538 / 3554 |

FDB 50k products at c128 stopped on the bench's heap guard (live heap > 6144 MB, peak 7.5 GB):
128 concurrent 11.5 MB responses decoded at once. It is a memory ceiling of the handler under that
load, not a database error; Scylla stayed under it only because it served fewer requests at a time.

**Delta entries: pointer vs row copy.** The first FDB run kept TypeDelta entries as pointers (value =
record key suffix, read with `IndexRangeGet`): 10k products first sync **166 ms**, against **37 ms**
for a plain range scan of the same records and **34 ms** once the entries hold the row. Pointer
entries cost one point read per record inside the daemon (~14 µs each on one `fdbserver`); the run
with pointer entries is in `/tmp/genix-bench/fdb-pointer-entries/`. Every number above uses row copies.

**Caveats.** Scylla runs `--overprovisioned` (no busy polling), which inflates its small-read latency
(2.4 ms per incremental sync); the README's own bench container uses the same flag. The FDB driver
does less per write than Scylla's (no by-IDs slots, cache-group marks, DataFrame log or text
search), which affects seeding, not these reads. Layer B (router) and the fareward benchmarks were
not run (D2). Single runs on a laptop: treat differences under ~15% as noise.

## 15. Out of scope

Lambda; a versionstamp redesign of the cache protocols; atomic writes across commit chunks; a
multi-node FDB cluster; migrating existing Scylla data (pre-alpha: wipe and reseed).
