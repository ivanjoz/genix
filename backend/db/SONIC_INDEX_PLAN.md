# Sonic Text Search Integration Plan

## Goal

Replace the current SQLite FTS5 backend (`backend/db/text_search/`) with
[Sonic](https://github.com/valeriansaliou/sonic), reached over its native
TCP protocol. The ORM keeps its existing call sites
(`syncTextSearchIndexAfterWrite`, `syncTextSearchStatusAfterWrite`); only
the bytes-leaving-the-process side changes.

Granularity matches the FTS5 plan: **one index per
`{table}_{partition}_s{0|1}`**, where the status group `s0` holds
records with `status == 0` and `s1` holds everything else.

### Why Sonic (vs SQLite FTS5 / SeekStorm / DuckDB)

- **Server-managed.** Sonic owns its own KV+FST storage, consolidation,
  and concurrency. No `WAL` / `-shm` files to back up, no shard-count
  decision to revisit, no per-process file lock fights.
- **Trivially networked.** TCP + a tiny line-oriented protocol. One
  daemon serves the whole fleet.
- **Schema-less.** Collections and buckets are created on first `PUSH`.
  No DDL, no migrations.
- **Shape fit.** Sonic's `(collection, bucket, object_id)` triple maps
  cleanly onto our `(table, partition+status_group, record_id)`.

Trade-offs accepted:

- A daemon now exists in the deploy. Mitigated by
  `cloud/text-searh/install_sonic.py` (single static binary under
  systemd).
- Search results capped at `LIMIT(100)` per query (Sonic default;
  configurable). Pagination uses `OFFSET`. Fine for our UI.
- No phrase search — Sonic does token-level matching. Already true of
  the FTS5 setup we're replacing.
- **Sonic's `PUSH` is additive**: re-pushing the same `object_id` with
  new text *adds* tokens rather than replacing them. The upsert path
  must `FLUSHO` first. See "Upsert flow" below.

---

## Scope

### In scope

1. **Delete** the SQLite FTS5 code under `backend/db/text_search/`
   (`driver.go`, `schema.go`, `writer.go`, `optimize.go`).
2. **Replace** with a hand-rolled Sonic TCP client in the same package
   path, so call sites in `text_search_index.go` and `insert-update.go`
   require zero signature changes.
3. **Two buckets per partition** (`s0` / `s1`); status moves are
   handled by **always flushing the other bucket** on every upsert
   (strategy "(a)" — stateless, correct, 3 commands/record).
4. **Two persistent connection pools** to the Sonic daemon:
    - **ingest pool** for `PUSH`, `FLUSHO`, `FLUSHB`
    - **search pool** for `QUERY`, `SUGGEST` (defined now, used by a
      future read PR)
   Both pools sized **min 2, max 8** per backend process.
5. **Same write entry points.** `UpsertBatch`, `UpsertRecord`,
   `DeleteRecord` keep their signatures; bodies talk Sonic.
   `Search()` / `Suggest()` ship as `ErrNotImplemented` stubs so the
   read PR is a one-file change.
6. **No DDL, no bootstrap.** Sonic creates collections on the first
   `PUSH`. The `_fts_meta` table and `EnsureIndex` machinery go away.
7. **Config from `credentials.json`.** New fields:
   - `SONIC_HOST` (default `127.0.0.1`)
   - `SONIC_PORT` (default `14446`)
   - `SONIC_PASSWORD` (no default — `install_sonic.py` writes it)
8. **Tests** at the unit level for protocol framing, name derivation,
   normalization, and the batch path against a mocked TCP server.

### Out of scope (deferred)

- A backend `Search()` handler routable from the frontend. The client
  stub exists; the handler lands later.
- Backfill of existing records into Sonic. First deploy only indexes
  new writes; a one-shot `fn-` exec will paginate existing tables and
  call `syncTextSearchIndexAfterWrite` after this PR lands.
- `TRIGGER consolidate` scheduling. Sonic auto-consolidates per its
  config (`consolidate_after`, `flush_after`); the client exposes a
  `Consolidate()` hook for the deferred maintenance PR.
- TLS / mTLS. The deploy runs Sonic on loopback or inside the same
  VPC; password-only auth is the project baseline.

---

## Sonic protocol primer (just what we need)

Sonic speaks a small text protocol on one TCP socket per connection.
Lines end with `\r\n`. Each connection is bound to one **channel** mode
(ingest / search / control) chosen at handshake.

Handshake:

```
S: CONNECTED <sonic-server v1.4.x>
C: START ingest <password>
S: STARTED ingest protocol(1) buffer(20000)
```

`buffer(N)` is the largest single line the server will accept (default
20000 bytes). We honor it via the truncation policy below.

Ingest commands we use:

```
PUSH   <collection> <bucket> <object_id> "<text>" [LANG(<iso>)]   -> OK
POP    <collection> <bucket> <object_id> "<text>"                 -> RESULT <n>
FLUSHO <collection> <bucket> <object_id>                          -> RESULT <n>
FLUSHB <collection> <bucket>                                      -> RESULT <n>
FLUSHC <collection>                                               -> RESULT <n>
PING                                                              -> PONG
QUIT                                                              -> ENDED quit
```

Search commands (defined now, used later):

```
QUERY   <collection> <bucket> "<terms>" [LIMIT(<n>)] [OFFSET(<n>)] -> PENDING <marker>
                                                                   <- EVENT QUERY <marker> id1 id2 ...
SUGGEST <collection> <bucket> "<word>"                             -> PENDING <marker>
                                                                   <- EVENT SUGGEST <marker> w1 w2 ...
```

Notes:

- **Quoting.** Text payloads are wrapped in double quotes; embedded `"`
  and `\` are backslash-escaped; newline becomes `\n`. We never quote
  identifiers (collection, bucket, object_id) — those must match
  `[A-Za-z0-9_:-]`.
- **Async results.** `QUERY` returns `PENDING <marker>` first, then a
  matching `EVENT QUERY <marker> ...` line. The client must read both.
  This is the reason search and ingest are separate pools.
- **Object IDs are strings.** We format `record_id` as decimal.
- **Additive PUSH.** A re-`PUSH` on the same `object_id` adds tokens to
  the existing set; it does *not* replace. The upsert flow `FLUSHO`s
  first.

---

## Name derivation

The current `IndexName(table, partition, statusGroup)` becomes
`CollectionAndBucket(table, partition, statusGroup)` returning two
strings:

```go
// CollectionAndBucket maps an ORM (table, partition, statusGroup) triple
// to the Sonic identifiers used on the wire.
//
//   collection = {tableName}
//   bucket     = p{partitionID}_s{statusGroup}
//
// Examples:
//   ("productos", 1,  0) -> ("productos",  "p1_s0")
//   ("productos", 1,  1) -> ("productos",  "p1_s1")
//   ("productos", 57, 1) -> ("productos",  "p57_s1")
//   ("clientes",  0,  0) -> ("clientes",   "p0_s0")
//
// The "p" prefix keeps the bucket identifier valid when partition_id
// is 0 (Sonic requires identifiers to start with a non-digit).
func CollectionAndBucket(tableName string, partitionID int32, statusGroup int8) (string, string)
```

`PickStatusGroup(status int8) int8` stays — same rule: `0` → `0`,
everything else → `1`.

Identifier safety mirrors the FTS5 check: collection and bucket are
validated against `^[A-Za-z_][A-Za-z0-9_]*$` before going on the wire.
Object IDs are integers, so no quoting / escaping is needed.

---

## Upsert flow (the key new bit)

Sonic's `PUSH` is additive: a re-push on the same `object_id` adds
tokens to the existing set. To get update semantics, we `FLUSHO` first
(empties the object's tokens), then `PUSH` the new text.

Status moves between `s0` and `s1` are handled **statelessly** —
strategy (a) from the design discussion: on every upsert, we also
`FLUSHO` the opposite bucket so a stale row from a previous status can
never linger. No Scylla read needed.

So per record, on an upsert with current status group `sg ∈ {0, 1}`:

```text
FLUSHO {collection} p{partition}_s{1-sg} {record_id}   # clear stale opposite-group
FLUSHO {collection} p{partition}_s{sg}   {record_id}   # clear current
PUSH   {collection} p{partition}_s{sg}   {record_id} "<text>"
```

Three commands per record, all sent on one ingest connection without
waiting between them (Sonic's per-connection serialization handles
ordering). `FLUSHO` on a missing object is not an error — it returns
`RESULT 0`. The cost on a warm connection is well under a millisecond.

`DeleteRecord` (explicit removal, e.g. record hard-deleted from
Scylla) becomes the same shape minus the `PUSH`:

```text
FLUSHO {collection} p{partition}_s0 {record_id}
FLUSHO {collection} p{partition}_s1 {record_id}
```

Both buckets, defensively — we don't need to know which one had it.

---

## File-level changes

### `backend/core/security.go`

```go
// SQLite FTS5 — embedded lexical search backend...
// SQLITE_FTS_DIR string                                         // REMOVE

// Sonic — lexical search backend reached over TCP. Installed by
// cloud/text-searh/install_sonic.py; password is written to
// credentials.json by that script.
SONIC_HOST     string
SONIC_PORT     int32
SONIC_PASSWORD string
```

### `backend/db/text_search/` (replace contents)

New layout:

- **`driver.go`** — package doc, `Configure(...)`, `Close()`,
  shared dialing helpers. No SQLite import. Pure `net.Conn` +
  `bufio.Reader/Writer`.
- **`protocol.go`** — line framing, response parsing, quoting helpers.
  All wire formatting lives here.
- **`pool.go`** — generic `connPool` backed by a buffered channel.
  Used by both ingest and search pools. Lazy fill, validate on
  checkout, recycle on protocol errors. Idle timeout 300 s.
- **`ingest.go`** — exported writer API (`UpsertBatch`,
  `UpsertRecord`, `DeleteRecord`, `FlushBucket`). Each call acquires
  one ingest connection, runs the commands, returns it.
- **`search.go`** — exported read API (`Search`, `Suggest`). Parsing
  helpers are unit-tested; exported functions return
  `ErrNotImplemented` in this PR.
- **`text.go`** — `NormalizeSearchText`, `PickStatusGroup`, `Record`
  (unchanged signatures), Sonic-quoting helper, `truncateForBuffer`.

Deleted from the current package: `schema.go`, `optimize.go`,
`writer.go`. The old SQLite `driver.go` content (shards, PRAGMAs,
`shardForPartition`, `_fts_meta`) is gone.

Dependencies dropped: `modernc.org/sqlite` + its transitive set. No new
modules — `net`, `bufio`, `sync`, `time` cover the client.

#### `driver.go` — public surface

```go
package text_search

import "context"

// Configure sets the Sonic endpoint and credentials. Safe to call
// once at process start. Subsequent calls before the first dial
// replace the previous values; afterwards they are ignored.
func Configure(host string, port int, password string)

// Close drains the ingest and search pools.
func Close() error

// Record: one row to push into a (table, partition, statusGroup) index.
type Record struct {
    ID         int64
    SearchText string
}

// PickStatusGroup centralizes the s0/s1 rule: status == 0 -> group 0,
// everything else -> group 1. Unchanged from the FTS5 plan.
func PickStatusGroup(status int8) int8

// UpsertBatch upserts every record into {table}/p{partition}_s{statusGroup}.
// For each record we additionally FLUSHO the opposite bucket so stale
// rows from a previous status never linger. Empty SearchText is skipped
// (we still FLUSHO so a record that lost all searchable text is removed).
// Empty slice: no-op.
func UpsertBatch(ctx context.Context, table string, partition int32,
    statusGroup int8, records []Record) error

// UpsertRecord — single-record convenience wrapper.
func UpsertRecord(ctx context.Context, table string, partition int32,
    statusGroup int8, recordID int64, searchText string) error

// DeleteRecord drops one document from both s0 and s1 buckets. Used
// when the ORM hard-deletes a row.
func DeleteRecord(ctx context.Context, table string, partition int32,
    statusGroup int8, recordID int64) error

// NormalizeSearchText: lowercase + strip to [a-z0-9 ] + collapse
// whitespace. Unchanged.
func NormalizeSearchText(s string) string

// Search / Suggest — defined for the deferred read PR.
type SearchResult struct{ IDs []int64 }

var ErrNotImplemented = errors.New("text_search: search path not implemented yet")

func Search(ctx context.Context, table string, partition int32,
    statusGroup int8, query string, limit, offset int) (*SearchResult, error)

func Suggest(ctx context.Context, table string, partition int32,
    statusGroup int8, word string) ([]string, error)
```

#### `pool.go` — connection pool

- Two pools: `ingestPool`, `searchPool`. Each sized `min=2, max=8`.
- Pool holds idle `*sonicConn` values in a buffered channel (size
  `max`). The `min` warm connections are dialed lazily on first use
  and kept around even when idle.
- Checkout: try the channel; if empty and total opened < `max`, dial
  a new one; if at max, block on the channel until one frees up.
- Return: if the last command on the connection errored at the
  protocol layer, `discard` it (close + decrement counter); otherwise
  put it back.
- Idle timeout: 300 s. Connections older than this are silently
  dropped at checkout and replaced. Matches Sonic's FST pool eviction
  (`inactive_after = 300` in the reference `config.cfg`).
- A `*sonicConn` carries:
  - `net.Conn`
  - `*bufio.Reader`, `*bufio.Writer`
  - `mode` (ingest / search)
  - `buffer int` (the `buffer(N)` advertised at handshake)
  - `usedAt time.Time`

#### `protocol.go` — wire framing

- `writeLine(w *bufio.Writer, parts ...string)` — joins with single
  spaces, appends `\r\n`, calls `Flush`.
- `readLine(r *bufio.Reader)` — reads up to `\r\n`, returns the
  trimmed line; errors on EOF mid-line.
- `quote(s string)` — builds the Sonic-quoted payload. Escapes `"`
  and `\` with a backslash; replaces `\r`, `\n`, `\t` with the
  literal `\r`, `\n`, `\t` escapes; rejects non-printable bytes (the
  normalizer already strips them; defensive belt).
- `parseResult(line string)` — recognizes:
    - `OK`             → success
    - `RESULT <n>`     → success with count
    - `PENDING <id>`   → return marker, expect EVENT line
    - `EVENT <kind> <id> <payload...>` → returns kind, id, payload
    - `ERR <reason>`   → typed error
    - `ENDED quit`     → clean server hangup
- `dialAndHandshake(host, port, mode, password)` — performs
  `CONNECTED` → `START` → `STARTED` and returns the negotiated buffer
  size.

#### `ingest.go` — write path

```go
func UpsertBatch(ctx context.Context, table string, partition int32,
    statusGroup int8, records []Record) error {

    if len(records) == 0 { return nil }
    collection, currentBucket := CollectionAndBucket(table, partition, statusGroup)
    _, otherBucket := CollectionAndBucket(table, partition, 1-statusGroup)
    if err := validateIDs(collection, currentBucket, otherBucket); err != nil {
        return err
    }

    conn, err := ingestPool.acquire(ctx)
    if err != nil { return err }
    defer ingestPool.release(conn)

    for _, r := range records {
        // 1. Clear stale row in the opposite-status bucket (status-move
        //    safety; no-op on RESULT 0).
        if err := conn.exec(ctx,
            fmt.Sprintf(`FLUSHO %s %s %d`, collection, otherBucket, r.ID),
        ); err != nil {
            ingestPool.discard(conn)
            return err
        }
        // 2. Clear current bucket so the PUSH replaces tokens rather
        //    than accumulating (Sonic's PUSH is additive).
        if err := conn.exec(ctx,
            fmt.Sprintf(`FLUSHO %s %s %d`, collection, currentBucket, r.ID),
        ); err != nil {
            ingestPool.discard(conn)
            return err
        }
        // 3. If SearchText is empty after normalization, the FLUSHOs
        //    above leave the record un-indexed — exactly what we want.
        if r.SearchText == "" { continue }
        text := truncateForBuffer(r.SearchText, conn.buffer, collection, currentBucket, r.ID)
        if err := conn.exec(ctx,
            fmt.Sprintf(`PUSH %s %s %d %s`, collection, currentBucket, r.ID, quote(text)),
        ); err != nil {
            ingestPool.discard(conn)
            return err
        }
    }
    return nil
}
```

- All three commands per record run on one connection; we don't wait
  for each `OK` before sending the next on the wire — but we do read
  each response line before moving on so a failure aborts the batch
  cleanly. (Pipelining is possible later as an optimization.)
- A middle-record failure leaves earlier records' updates landed.
  Same liveness behavior as the FTS5 path. The ORM's retry layer
  handles the rest.

#### `text.go` — truncation policy

```go
// truncateForBuffer keeps the formatted PUSH line under conn.buffer.
// Header overhead (`PUSH <coll> <bucket> <id> ""\r\n`) plus quoting
// expansion is computed exactly. Truncation chops on the last space
// inside the limit so we never split a token.
func truncateForBuffer(text string, bufferSize int,
    collection, bucket string, id int64) string
```

Silent truncate (confirmed default). No caller has ever hit 20 KB of
normalized text in practice; if one does, a token-boundary trim keeps
the index usable.

#### `search.go` — read path (parsing only)

`Search` and `Suggest` return `ErrNotImplemented`. Internal helpers
(`parseEventQuery`, `buildQueryLine`) are wired and unit-tested.

### `backend/db/text_search_index.go`

Minimal — the public Sonic API mirrors the old FTS5 one:

- `textSearchIndexInfo`: unchanged.
- `groupRecordsForTextSearch`: unchanged. Still groups by
  `(partition, statusGroup)` and builds `[]ftsBucket`.
- `syncTextSearchIndexAfterWrite`: unchanged body except for the
  comment — the inner `UpsertBatch` call now handles status-move
  cleanup via the FLUSHO-the-other-bucket trick, so the
  `TODO(status-move)` block deletes.
- `syncTextSearchStatusAfterWrite`: thin wrapper, as today.
- `buildSearchText`: unchanged.

### `backend/db/main.go`

- `textSearchIndexInfo` struct: unchanged.

### `backend/db/deploy.go`

- Already a no-op for text search (lines 884–887). Swap "FTS5" →
  "Sonic" in the wording.

### `backend/db/init.go`

- Comment: the `Configure` call now wires `(host, port, password)`
  instead of a directory.

### `backend/main.go` and `backend/exec/init.go`

Replace `configureTextSearchDir` with `configureTextSearchSonic`:

```go
func configureTextSearchSonic() {
    host := strings.TrimSpace(core.Env.SONIC_HOST)
    if host == "" { host = "127.0.0.1" }
    port := int(core.Env.SONIC_PORT)
    if port == 0 { port = 14446 }
    pw := strings.TrimSpace(core.Env.SONIC_PASSWORD)
    if pw == "" {
        if core.Env.IS_PROD {
            core.Log("text_search: SONIC_PASSWORD empty in prod; writes will fail")
        }
    }
    text_search.Configure(host, port, pw)
}
```

Called from `main.main` and `exec.ConfigInit`.

### Files deleted

- `backend/db/SEEKSTORM_INDEX_PLAN.md` — superseded by this file.
- `backend/db/text_search/driver.go` (current SQLite version)
- `backend/db/text_search/schema.go`
- `backend/db/text_search/writer.go`
- `backend/db/text_search/optimize.go`

Replaced by `driver.go` / `protocol.go` / `pool.go` / `ingest.go` /
`search.go` / `text.go` listed above.

---

## go.mod changes

```diff
- modernc.org/sqlite v1.50.1
```

And the transitive `modernc.org/libc`, `modernc.org/mathutil`,
`modernc.org/memory`, `github.com/ncruces/go-strftime`,
`github.com/remyoudompheng/bigfft`, `github.com/dustin/go-humanize`
entries that drop out with it.

No new modules added.

---

## Status semantics

Same as the FTS5 plan:

| `status` value  | Bucket  | Notes                                             |
|-----------------|---------|---------------------------------------------------|
| `0`             | `s0`    | "soft-deleted / inactive" search domain           |
| anything else   | `s1`    | "active" search domain                            |
| no status col   | `s1`    | tables without a `status` column always use `s1`  |

Move handling: every upsert FLUSHOs the **opposite** bucket too, so a
status flip leaves no stale tokens behind. No Scylla read on the write
path; no `TODO(status-move)`.

---

## Operational notes

- **Daemon.** One Sonic instance per environment, provisioned by
  `cloud/text-searh/install_sonic.py`. Password lives in
  `credentials.json` (written by the install script, read by the
  backend).
- **Bind address.** `install_sonic.py` defaults to `0.0.0.0:14446`.
  Prod should bind to loopback when colocated with the backend, or
  enforce a VPC ACL.
- **Backups.** Sonic persists `kv/` and `fst/` under `/var/lib/sonic`.
  Rebuildable from ScyllaDB (`syncTextSearchIndexAfterWrite` is
  idempotent), so we can either back them up or rely on the backfill
  exec to repopulate on a fresh daemon.
- **Connection limits.** Sonic's default is ~1024 concurrent clients;
  our two `min=2/max=8` pools per process are well under.
- **Monitoring.** `PING` on a control connection returns `PONG`; a
  follow-up PR can add a `/health/sonic` route.

---

## Tests

Replacements for the deleted FTS5 tests:

- `TestCollectionAndBucket` — `s0` / `s1` suffix, partition 0,
  identifier safety, large partition values.
- `TestPickStatusGroup` — unchanged rule.
- `TestNormalizeSearchText` — unchanged behavior.
- `TestQuoteHandlesQuotesAndBackslashes` — protocol quoting.
- `TestParseResultRecognizesAllFrameTypes` — `OK`, `RESULT n`,
  `PENDING id`, `EVENT QUERY id ...`, `ERR ...`, `ENDED quit`.
- `TestUpsertBatchSendsThreeCommandsPerRecord` — drives a
  `net.Pipe`-based fake server, asserts the exact bytes emitted:
  FLUSHO other, FLUSHO current, PUSH current.
- `TestUpsertBatchSkipsPushWhenTextEmpty` — both FLUSHOs still go,
  PUSH is omitted.
- `TestUpsertBatchRecoversFromTransientError` — server returns
  `ERR <reason>`; the call returns a typed error and the connection
  is discarded (not returned to the pool).
- `TestDeleteRecordFlushesBothBuckets` — verifies the `s0` + `s1`
  FLUSHO pair.
- `TestPoolRecyclesIdleConnections` — 300 s drop.
- `TestTruncateForBufferKeepsTokenBoundary` — text longer than
  Sonic's `buffer(N)` chops on the last space.
- `TestStatusFlipLeavesNoStaleRows` — integration over the
  `syncTextSearchIndexAfterWrite` path with a fake server; a record
  that moves `s0 → s1` triggers FLUSHO on `s0` and FLUSHO+PUSH on
  `s1`.

Test runner:

```bash
cd backend && go test ./db/text_search -count=1
cd backend && go test ./db -run TestSyncTextSearch -count=1
```

The `net.Pipe`-based fake server keeps tests hermetic — no Sonic
daemon required in CI.

---

## Migration / rollout

1. **Deploy Sonic.** Run `install_sonic.py` on the host. It writes
   `SONIC_PASSWORD` into `credentials.json` and starts the systemd
   unit.
2. **Add `SONIC_HOST` / `SONIC_PORT`** to `credentials.json` if the
   defaults (`127.0.0.1:14446`) don't match the deploy.
3. **Land this PR.** New writes index into Sonic from then on.
4. **Backfill (deferred PR).** A `fn-backfill-sonic` exec paginates
   every `TextSearchColumn` table and calls
   `syncTextSearchIndexAfterWrite` per page.
5. **Clean up the SQLite shards.** `rm -rf {SQLITE_FTS_DIR}/` once the
   backfill is verified.

---

## Decided defaults (confirmed in review)

- Status-move strategy **(a)**: stateless, FLUSHO the opposite bucket on
  every upsert. 3 commands per record.
- Pool size: **min 2, max 8** for both ingest and search pools.
- Truncation: **silent truncate on the last space inside Sonic's
  `buffer(N)`**.
- Read API: **stubs only** in this PR (`ErrNotImplemented`).
- Idle timeout: **300 s** client-side, matching Sonic's FST pool
  eviction.

Ready to start the code change on confirmation of this plan.
