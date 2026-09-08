# Sequence reservation in fareward

Move autoincrement/counter allocation out of the ORM's racy read-modify-write and into the
fareward daemon, which is already the project's single-active-process serialization point.

Status: **implemented and always on.** Every step below is built and tested. In Genix the ORM
always reserves through fareward — there is no configuration switch; the wiring lives in
`backend/db/autoincrement.go`. Approved decisions are recorded in "Decisions taken" below, and the
design rationale now also lives in `fareward/RATIONALE.md` and `backend/RATIONALE.md`.

---

## 1. The problem

`genix-orm/scylla/main.go:177` `GetCounter(keyspace, name, increment)` is the single allocator for
every counter in the system:

```go
SELECT current_value FROM <ks>.sequences WHERE name = ?   // -> stored
UPDATE <ks>.sequences SET current_value = current_value + ? WHERE name = ?
return stored + 1
```

Read, then increment, with nothing between them. Two callers that read the same `stored` both
return the same start value. Scylla `counter` columns make the *increment* atomic but give no way
to read the result of your own increment, so the read is always a guess about who else was in
flight.

Three call sites depend on it, all of them on hot write paths:

| Call site | File | What it mints | Cost of a duplicate |
|---|---|---|---|
| `fetchAutoincrementCounterStarts` | `scylla/insert-update.go:285` | Autoincrement PKs, counter `x{partition}_{table}_{part}` | **Silently overwritten records** |
| `fetchManagedCounterValues` | `scylla/insert-update.go:144` | `updated_version` write sequence, counter `x{partition}_{table}_updated`, increment 1 per write | Two writes share a version; delta-cache clients miss one |
| `GetAutoincrementID` | `scylla/main.go:211` | Arbitrary application keys (`images_<companyID>`, …), also exported as `db.GetAutoincrementID` | Depends on the caller |

`deploy.go:453` `ResetCounter` also read-modify-writes the same rows, but it is an admin path.

## 2. The approach

fareward becomes the **sole writer** of the `sequences` table and allocates in **hi-lo blocks**.

Per counter name the daemon holds an in-process mutex and an in-memory `(next, end)` range. A
request that fits in the live range is answered from RAM with no I/O at all. A request that does
not takes one durable block:

```text
lock(name)
UPDATE <ks>.sequences SET current_value = current_value + B WHERE name = ?
SELECT current_value FROM <ks>.sequences WHERE name = ?      -> V
own (V-B, V]      // sound only because nothing else writes this row
```

Because the durable counter is bumped *before* anything is handed out, it is always at or above
the high-water mark of what has been issued. A crash loses the unused tail of the live block — a
gap, never a reuse.

Re-deriving the range from the durable value on every block (rather than trusting a value cached
at startup) is what lets a manual repair be picked up without restarting the daemon. It is *not*
enough for `ResetCounter`, whose write can land while a block is still live — see §4a.

### What this costs

- **Exclusivity is mandatory.** The read-back is only sound because no other process writes that
  row. A backend still on the direct path would collide with the daemon's owned range. That is why
  the wiring is unconditional rather than a setting, and why the daemon must stay a single active
  instance — which it already must be for the limiter and the lock registry.
- **Restart burns IDs.** At most `block_size - 1` per counter that had a live block. This matters
  most for `updated_version`, which is bounded by the delta-view digit slot
  (`scyllaTable.maxDeltaVersionValue`, enforced at `insert-update.go:150`), so `block_size` stays
  modest and configurable rather than large.
- **Repairs must go through the daemon too.** `ResetCounter` moves a counter to an absolute value,
  and a daemon holding a block derived from the old value would keep issuing ids the reset just
  invalidated. Opcode `0x08` (§4a) moves the counter and drops the block under one lock.

## 3. Decisions taken

| Question | Decision |
|---|---|
| Atomicity model | Hi-lo blocks over the existing `counter` column, fareward exclusive. No schema change. |
| Counter name on the wire | Length-prefixed string, verbatim. `sequences` row keys stay byte-identical, so existing rows, `ResetCounter`, `deploy.go` and manual inspection are untouched. |
| fareward unreachable | **Fail the write.** No fallback to the direct path — a daemon blip must not silently start minting duplicate PKs. |
| Counters routed through fareward | Autoincrement PKs, the `updated_version` write sequence, and the `GetAutoincrementID` public API. `ResetCounter` was initially left direct; opcode `0x08` brought it in once the block-invalidation hazard was understood. |
| How Genix opts in | It does not opt in — the ORM is wired to the daemon unconditionally, in code, in `backend/db/autoincrement.go`. No config key, no environment variable. |

---

## 4. Wire protocol — opcode `0x07 ReserveSequence`

Second length-prefixed opcode after `LogRequest`, and the first that is both length-prefixed and
answered.

**Request** — `[0x07][len:u16][increment:u32][name bytes][tag:8]`

`increment` leads so the name is the tail and needs no length of its own; `len` already bounds it.
Name ceiling `SEQUENCE_NAME_MAX = 128` bytes (the widest ORM name,
`x{partition}_{table}_{part}`, is far under this).

**Reply** — the existing shape, with the value in the tail:
`[correlation:u16][status:u8][detail:u16][extra_len:u8=8][start:i64]`

`start` is the first reserved value, big-endian, matching what `GetCounter` returns today.
`status` 0 = reserved; nonzero = refused (`UNAVAILABLE_STATUS` for capacity/Scylla failure, a
distinct code for an invalid request). The Go client treats every nonzero status as an error.

**Sizing.** `REPLY_MAX_EXTRA_SIZE` is currently `2 * MAX_REQUIRED_ACCESS` = 8, which happens to be
exactly an `i64`. Redefine it as `max(2 * MAX_REQUIRED_ACCESS, SEQUENCE_REPLY_EXTRA_SIZE)` on both
sides so the fit is deliberate and survives `MAX_REQUIRED_ACCESS` changing. Same for the Go
mirror `farewardReplyMaxExtraSize` (`go/connection.go:40`).

## 4a. Wire protocol — opcode `0x08 SetSequence`

Added after the reserve path was working, once it became clear that `ResetCounter` could not stay on
the direct path. It moves a counter to an absolute value: the repair a restore needs.

**Request** — `[0x08][len:u16][value:i64][name bytes][tag:8]`

Same shape as a reservation, one field wider: an absolute `i64` where the `u32` increment was.
`value` must be ≥ 0 — zero is legitimate (an emptied partition resets to it, and the next id is 1),
negative is not, because ids are primary keys.

**Reply** — `[…][extra_len:u8=8][previous:i64]`, the value the counter held before. After a
destructive repair that figure exists nowhere else, which is also why the Go client sends this with
`requestOnce`: a retry would report the value the first attempt had already written.

**Why it is an opcode and not a direct write.** The daemon may be serving a block it derived from
the value being replaced. If the row moves underneath it, the durable counter stops bounding what
has been issued, and the next block claimed overlaps what the abandoned one already handed out. So
`set` takes the same per-name lock as `reserve`, applies the delta the counter column requires, and
marks the block spent — all three in one critical section, which is the only place they can be
atomic with respect to an in-flight reservation.

Waiting for the daemon to go idle is not an alternative: a block is re-derived only when exhausted,
so an idle counter is one holding a stale block indefinitely.

---

## 5. Work items

### 5a. fareward daemon (Rust)

**New `src/sequence/` module**, declared in `src/lib.rs` alongside `lock` and `limiter`.

- `protocol.rs` — `SEQUENCE_NAME_MAX`, `ReserveRequest { name: String, increment: u32 }`,
  `parse_reserve(&[u8]) -> Result<ReserveRequest, SequenceProtocolError>`, `SequenceReply` status
  enum. Rejects: zero increment, empty name, non-UTF8 name, name over the ceiling.
- `store.rs` — `SequenceStore` trait (`bump(name, by) -> Result<()>`, `read(name) -> Result<i64>`)
  plus `ScyllaSequenceStore` holding `Arc<Session>`, the keyspace, and both prepared statements.
  A trait, mirroring `LimiterStore`, so the allocator is unit-testable without a cluster.
- `allocator.rs` — `SequenceAllocator`. Sharded `HashMap<String, Arc<Mutex<SequenceState>>>` where
  `SequenceState { next: i64, end: i64 }`, bounded by `max_tracked_names`; evicting a name with a
  live block only burns its tail, so eviction is safe under pressure.
  `reserve(name, increment) -> Result<i64>` implements the block algorithm in §2.

  **Damaged-counter repair must mirror the ORM exactly.** `nextCounterRange`
  (`scylla/main.go:218`) already handles a counter driven non-positive by a repair/reset: it
  reserves from 1 and folds the correction into the same update. The allocator reproduces that
  rule and logs it, or a repaired counter would hand out non-positive PKs.

**`src/service/protocol.rs`** — add `Opcode::ReserveSequence = 0x07`, its `from_byte` arm, a
`PayloadWidth::LengthPrefixed { maximum: SEQUENCE_MAX_PAYLOAD_SIZE }` arm, `expects_reply() =
true`, and extend the existing exhaustive tests. The reader loop at `server.rs:244` already
handles a length-prefixed width generically and needs no change.

**`src/service/server.rs`** — thread `Arc<SequenceAllocator>` through `run` and
`handle_connection`, and add the dispatch arm. It must be **spawned with an in-flight permit**
like `ChargeCredits`, never inlined: a block miss does Scylla I/O and would head-of-line block
every other request on the connection. Answer with `send_reply_with_extra`.

**`src/main.rs`** — build the allocator from the already-shared `session.clone()` (`main.rs:36`)
and `config.sequences`, pass it into `server::run`.

**`src/config.rs`** — a `[sequences]` section: `enabled`, `block_size` (modest default, e.g. 64,
because of the `updated_version` delta-slot ceiling), `max_tracked_names`.

**Tests** (`src/sequence/*` unit tests + `tests/`): protocol round-trip and every rejection; the
allocator hands out strictly contiguous, non-overlapping ranges under concurrent reserves against
a fake store; a block miss issues exactly one bump and one read; the damaged-counter path restarts
at 1; eviction of a live block loses no soundness.

### 5b. fareward Go client

**New `go/sequences.go`** — `opcodeReserveSequence = byte(0x07)` and
`ReserveSequence(ctx, name string, increment int) (int64, error)`, which builds the payload,
sends, and decodes the 8-byte tail.

**`go/connection.go`** — `exchange` currently always calls `buildFarewardFrame`
(`connection.go:352`). Add `opcodeIsLengthPrefixed(opcode byte) bool` mirroring the Rust
`payload_width`, and pick `buildFarewardLengthPrefixedFrame` (which already exists at
`connection.go:496` but is only reachable from the fire-and-forget `send` path). Widen
`farewardReplyMaxExtraSize` per §4.

Uses `client.request` (retry-once), not `requestOnce`: an ambiguous disconnect that gets retried
reserves a second block and abandons the first, which is a gap — and gaps are already the accepted
cost of hi-lo. Availability is worth more here than a perfectly dense sequence.

**Tests** (`go/sequences_test.go`, following `request_log_test.go`): exact frame bytes including
the length header, tag coverage of that header, tail decode, nonzero status becomes an error.

### 5c. genix-orm — the hook

genix-orm is a standalone public repo and must not gain a fareward dependency, so the allocator is
something a consumer installs rather than something the ORM knows about. It already injects driver
functions through package vars (`db.GetAutoincrementID` at `scylla/init.go:20`,
`getWriteCounterValue` at `insert-update.go:18`), so this is one more of those.

The hook being optional is a property of the *library*, not of Genix: another consumer may leave it
nil and keep `GetCounter`. Genix always fills it in.

Add to `scylla`:

```go
// ReserveCounterRange, when set, replaces the local read-modify-write with an external allocator
// that serializes reservations. Nil means use GetCounter. There is no fallback once set: an
// external allocator owns the sequences row, so silently reverting to the local path would mint
// duplicates.
var ReserveCounterRange func(keyspace, name string, increment int) (int64, error)

func reserveCounter(keyspace, name string, increment int) (int64, error) {
    if ReserveCounterRange != nil {
        return ReserveCounterRange(keyspace, name, increment)
    }
    return GetCounter(keyspace, name, increment)
}
```

Route the three approved consumers through it:

- `insert-update.go:18` — `var getWriteCounterValue = reserveCounter` (the existing tests that
  swap this var keep working unchanged).
- `insert-update.go:285` — `GetCounter(...)` → `reserveCounter(...)`.
- `main.go:215` — `GetCounter(connParams.Keyspace, key, recordsSize)` → `reserveCounter(...)`.
- `deploy.go:505/516` — **unchanged**, per the decision above.

Errors propagate verbatim, so the write fails.

**Tests**: the hook is used for all three consumers when set; a hook error aborts the insert and no
row is written.

### 5d. backend wiring

`backend/db/autoincrement.go` sets `scylla.ReserveCounterRange` in package `init()`, next to
`db/driver.go`, which already declares the project's database choice. `db` and `fareward/go` are
both L0 in `docs/MODULE_BOUNDARIES.md`, so the import is within the layer; `db` cannot import
`core` (cycle), which is why it calls the fareward client directly.

No configuration switch. In Genix the ORM always reserves through the daemon: the alternative — a
setting — makes "wired" and "unwired" both representable, and the unwired half of a deployment is
precisely what mints duplicate primary keys. `init()` in the package every module already imports
is what makes the unwired state unreachable.

The keyspace argument is dropped: the daemon writes its own configured keyspace, and a backend
pointed at a different one than its daemon is a misconfiguration no per-call argument could repair.

### 5e. Docs

- `fareward/README.md` — the `sequences` table under "The tables it expects to already exist", the
  `0x07` frame under "TCP contract", and a "Sequence behavior" section covering hi-lo, the
  exclusivity requirement and the restart burn.
- `fareward/RATIONALE.md` and `backend/RATIONALE.md` — the decisions in §3.

---

## 6. Order of work

1. Rust protocol + allocator + store, with unit tests against the fake store. Self-contained.
2. Rust dispatch, config, `main.rs` wiring.
3. Go client + frame-builder change.
4. genix-orm hook.
5. Backend wiring in `db`'s package init.
6. Docs.

Steps 1–4 are independently testable; step 5 is what puts the daemon on the write path.

## 7. Resolved during implementation

- **There is no backend flag.** The ORM is wired to the daemon unconditionally in
  `backend/db/autoincrement.go`'s package `init()`. A switch would leave the direct path reachable,
  and reaching it is the duplicate-key bug.
- **`block_size` default is 64.** The `updated_version` ceiling turned out to be far less binding
  than §2 implied: at its narrowest the delta-view slot is 10⁸ writes per partition, so a restart
  burning ≤ 63 values would need on the order of a million restarts to consume 1% of it. The knob
  stays configurable, but it is not the tight constraint it looked like.
- **One install site covers every entry point.** Package `init()` fires before `main()` and before
  the `exec.ExecHandlers` dispatch, so the script paths — which do write records, and call
  `db.GetAutoincrementID` directly — cannot be left on the direct path. Verified that `backend` is
  the only binary writing through the ORM: `scripts/validation` names the ORM path only as a string
  in AST analysis, and `cloud`/`db-backup` do not link it.

## 8. Still deferred

- **Exclusivity holds only inside this codebase.** The `init()` covers every Genix entry point, but
  nothing stops a separate tool, a CQL session or a future service from advancing a `sequences` row
  directly, and doing so would collide with a block the daemon believes it owns. A daemon that could
  report "this counter is mine" would be the real fix.
- **No end-to-end test against a live daemon + ScyllaDB.** The allocator is tested against a fake
  store, the wire against a stub daemon, and the ORM hook against a stub allocator, but the three
  have not been run together on real infrastructure.
