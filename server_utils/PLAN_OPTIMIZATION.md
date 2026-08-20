# Memory and hot-path optimization

An ordered plan to bound the daemon's resident memory and cut the per-request cost of `admit_at`.
Read [CREDIT_LIMITER_WALKTHROUGH.md](CREDIT_LIMITER_WALKTHROUGH.md) first — this document assumes
its vocabulary (shard, subject, budget, usage record, the extra pool) and changes nothing about the
wire contract or the decision it describes.

**Nothing here is a behaviour change.** Every stage either moves the same numbers through a cheaper
container, or reclaims state the daemon can already rebuild from ScyllaDB. Where a stage touches a
contract the paired Go/Rust tests pin, the invariant and the test that holds it are named.

---

## 0. Baseline

Struct sizes measured with `size_of` on the current tree (x86-64, `1.95.0`):

| Type | Size | Notes |
|---|---|---|
| `TokenBucket` | 32 B | `u128` scaled tokens + `Instant` |
| `SubjectState` | 112 B | two `TokenBucket`s + two periods + two `Credits` |
| `CompanyBudgetState` | 128 B | |
| `UserAccessState` | 32 B | plus a heap `Box<[u16]>` for the grants |
| `UsageRecord` | 40 B | plus a `BTreeMap` node, see stage 2 |
| `Credits` | 16 B | |
| `UsageKey` | 12 B | |
| `MAX_FRAME_SIZE` | 1118 B | per live connection, inside the task future |

The number that matters is not any single one of these — it is that three of the four maps holding
them are never emptied.

### What grows without bound

`ShardState` (`src/limiter/quota.rs:376`) holds four maps. Only `usage` is pruned
(`prune_clean_usage`, `quota.rs:1098`). `subjects`, `budgets` and `access` are insert-only: the only
removal anywhere is `invalidate_access` dropping `access` entries on an explicit `0x06` frame.

Resident memory therefore tracks **distinct `(company, user)` pairs seen since boot**, not
concurrent load:

```
  subjects   112 B value + 8 B key + control byte    ≈ 138 B per user
  access      32 B value + 8 B key + grants heap     ≈  80 B per user
  budgets    128 B value + 4 B key                   ≈ 140 B per company
                                                     ─────────────────
                                                     ≈ 220 B per user, permanently
```

100k users ≈ 22 MB and never falls; 1M ≈ 220 MB. A restart is currently the only way memory is
returned.

### What costs per request

An authorized-and-charged frame performs ~23 `HashMap` lookups through the default SipHash-1-3, on
keys the code already holds: 2 on `access`, 3 `contains_key` in the `ensure_*` calls, 2 `get_mut` in
the period refresh, 4 across the two-window violation loop, 3 more `get`, 3 `get_mut` to charge, 4
in `increment_usage`, 1 in `increment_platform_usage`. At ~15-20 ns per hash that is roughly 400 ns
per request spent hashing.

Every admitted frame also takes one **process-global** mutex (`platform_usage`, `quota.rs:579`),
and a cold subject performs **five sequential ScyllaDB round trips while holding the shard mutex**.

---

## Stage 1 — Evict stale access entries

**Change.** In the flush pass, drop `access` entries that fail `is_fresh(now, access_cache_seconds)`.

**Why first.** It is the largest fast-growing map, and the eviction is provably invisible: an entry
past the TTL is re-read by `ensure_access` on its next use anyway, so dropping it early only changes
*when* the row is re-read, never the verdict. Today a company whose 50k users each logged in once
holds 50k dead entries until the process restarts.

**Invariant.** A dropped entry must be indistinguishable from an expired one.

**Tests that hold it.** `grants_are_read_once_and_reread_past_the_ttl`, `a_missing_user_is_cached_too`
(a cached miss must still be a cached miss inside the TTL), and
`invalidation_forces_a_reread_for_one_user_or_a_whole_company`.

**New test.** After a flush past the TTL, the map is empty and the next request re-reads exactly once
— asserted through `MemoryStore::user_reads`, which already counts point reads for this purpose.

**Saving.** Caps `access` at the users active within one TTL window (default 600 s) instead of all
users ever.

**Risk.** None identified. Self-contained.

---

## Stage 2 — `RoutedCredits` as a sorted `Vec`, and encode under the lock

**Change.** `pub type RoutedCredits = BTreeMap<u16, Credits>` (`credits_blob.rs:39`) becomes a sorted
`Vec<(u16, Credits)>` behind a small wrapper exposing `get`/`insert`/`iter`. Then
`UsageRecord::snapshot` (`aggregation.rs:65`) encodes to `Vec<u8>` while the shard lock is held, and
`UsageSnapshot` carries bytes instead of a map.

**Why.** `BTreeMap` allocates a whole `LeafNode` on first insert. For `K=u16, V=Credits` with std's
`CAPACITY = 11`, that node is 8 (parent) + 2 + 2 + pad + `[u16;11]` 22 + pad + `[Credits;11]` 176 =
**216 bytes**, whether the user touched one route or eleven. After pruning there are ~2 live records
per user active today plus 2 per company, so most of ~500 B/user/day is empty tree node.

A sorted `Vec` is 24 B of header plus 24 B per route — 96 B for a three-route user against 240 B —
and brings three secondary wins:

- `snapshot` currently **clones the entire map on every flush**. A `Vec` clone is one `memcpy`; a
  `BTreeMap` clone is a tree walk plus a fresh node allocation, per dirty record, every 15 s.
  Encoding under the lock removes even that: `encode` over a few dozen bytes is far cheaper than the
  clone it replaces, and `flush_snapshot` (`quota.rs:1077`) then just hands the driver its bytes.
- `encode` walks contiguous memory instead of chasing node pointers.
- Ascending route order — which `encode`'s canonical-form contract depends on — is preserved
  structurally by binary-search insert, exactly as the `BTreeMap` preserved it.

**Invariant.** The blob is canonical: entries ascend by route, an all-zero route is omitted, every
value uses the narrowest width that holds it. One set of totals has exactly one representation.

**Tests that hold it.** `requested_example_matches`, `the_full_route_range_round_trips`,
`width_boundaries_round_trip`, `rejects_noncanonical_or_truncated_data` — all in `credits_blob.rs`,
all byte-exact, none of which should need editing. If any of them requires a change, the container
swap has leaked into the format and the change is wrong.
`mutation_during_flush_remains_dirty` (`aggregation.rs`) holds the version protocol across the
`snapshot` signature change.

**Saving.** Roughly 60 % off the usage records, plus one fewer allocation *and* one fewer tree walk
per dirty record per flush.

**Risk.** Low, and entirely contained by the four codec tests. The signature change to
`UsageSnapshot` touches `flush_dirty`, `flush_snapshot` and `mark_flushed`.

### 2b (optional, separate commit) — `u32` per-route counters

`width_for` errors above `u32::MAX`, so a per-route `u64` pair is provably wider than anything that
can ever be persisted. A route-local `{cpu: u32, inference: u32}` halves the entry again, to 12 B.
`Credits` stays `u64` wherever it is a genuine sum (`hour_used`, `day_used`, `month_used`), so this
introduces a second type and a conversion at the `increment`/`sum` boundary — worth it only after
stage 2 is landed and measured, and it needs its own overflow test at the `u32` ceiling.

---

## Stage 3 — Integer hasher and collapsed lookups

**Change, two independent parts.**

1. A ~15-line multiply-xor `BuildHasher` for the four `ShardState` maps and `platform_usage`. No new
   dependency.
2. Fetch both `SubjectState`s in one pass with `HashMap::get_disjoint_mut` (stable since 1.86; the
   toolchain is pinned at 1.95), and hoist the budget `get_mut` to a single binding for the whole of
   `admit_at`. ~23 lookups becomes ~12 with no restructuring.

**Why the custom hasher is safe here.** These keys arrive inside an HMAC-authenticated frame that is
domain-separated by protocol version, so hash-flooding by an untrusted caller is not in the threat
model — an attacker who can choose `company_id` can already charge credits. This must be stated in
the code comment, because the reason it is safe is not local to the hasher.

**Invariant.** Iteration order of these maps must not be load-bearing. `flush_dirty` and
`prune_clean_usage` both iterate `usage`; neither depends on order, and this is worth re-verifying
while making the change rather than assuming it.

**Tests that hold it.** The whole `quota.rs` suite passes unchanged, in particular
`accepted_request_dirties_company_user_and_platform_rows` and
`platform_aggregate_is_shared_across_company_shards` (sharding is `company_id % shard_count`, not a
hash, so it is unaffected — the test confirms that).

**Saving.** ~2× on the hot path's hashing component.

**Risk.** Low. Mechanical.

---

## Stage 4 — `TokenBucket` to `u64`, one shared `last_refill`

**Change.** `scaled_tokens: u128` (`quota.rs:66`) becomes `u64` scaled to **microseconds** instead of
nanoseconds, and the two `last_refill: Instant` fields collapse into one on `SubjectState`.

**Why.** Nanosecond scaling forces multi-word `u128` mul/add/min on every refill. Microsecond scaling
keeps headroom to `limit ≈ 1.8e12`, far above any credit figure the config can express, and makes
refill single-word. The two buckets are always refilled at the same instant — `refresh_periods`
refills both unconditionally, and `SubjectState::recovered` builds both from the same `now` — so the
second `Instant` stores the same information twice.

`SubjectState` goes 112 B → **64 B**, and the surviving `last_refill` becomes exactly the
last-touch stamp stage 5 needs.

**Invariant.** The bucket must still admit exactly `limit` credits per 10 s window and refuse the
next one. Rounding moves from nanosecond to microsecond granularity, which is below the resolution
any test or caller observes — but a limit large enough to overflow the `u64` scale must saturate
rather than wrap, and that needs an explicit guard plus a test.

**Tests that hold it.** `exact_limit_is_allowed_and_next_credit_is_rejected` is the boundary test and
must pass untouched. `the_extra_pool_never_relaxes_a_burst_gate` and
`extra_spending_still_consumes_burst_and_hourly_credits` hold the bucket's interaction with the pool.

**New test.** A limit at the `u64` scale ceiling saturates instead of wrapping.

**Saving.** 48 B per subject, and cheaper arithmetic on every request.

**Risk.** Low-medium — it is the only stage that changes an arithmetic behaviour, if only in
rounding. Land it alone.

---

## Stage 5 — Idle eviction for `subjects` and `budgets`

**Change.** In the flush pass, after `mark_flushed` and the existing prune, drop:

- a `SubjectState` whose `last_refill` is older than a new `idle_ttl` **and** whose usage records are
  clean;
- a `CompanyBudgetState` with `version == flushed_version` and no live subject left for the company.

**Why this is safe, and why the gating conditions are not optional.** Both are already reconstructed
from ScyllaDB on a cold miss, by code that runs today:

- `ensure_subject` reseeds `hour_used` by summing the flushed five-minute rows and `day_used` from
  the flushed daily row.
- `ensure_budget` reseeds `month_used` by summing the month's daily rows and subtracting
  `month_extra_cpu_used`, and recovers `day_extra_used` / `month_extra_used` from the flushed budget
  row — including the period checks that read an untouched window as zero.

So eviction is a *cold start for one tenant*, which is a path with tests already on it. The gating is
what makes it that and not credit given away: **dropping a dirty subject would lose the counters the
daily gate is judged on, which is free credit.** Clean-only is the whole safety argument.

**Invariant.** Evicting and re-loading a subject or budget must produce the same admission decision
as never having evicted it. The daily and monthly counters must survive the round trip; the extra
pool must not reset mid-day.

**Tests that hold it.** `a_restart_recovers_the_pool_and_does_not_lose_entitlement_to_it` is the
closest existing analogue and the model for the new tests.
`extra_spending_is_counted_apart_from_the_quota_it_bypassed`, `the_pool_resets_on_the_local_business_day`,
`daily_usage_resets_on_the_local_day_boundary`, `user_daily_budget_is_half_the_company_budget` and
`monthly_budget_is_shared_by_every_company_user` all constrain the counters an eviction must not
disturb.

**New tests.**
1. Charge, flush, evict, charge again — the daily gate refuses at the same credit it would have.
2. A dirty subject is **not** evicted (assert it survives a flush whose write failed).
3. Evicting a company mid-day does not hand it a fresh extra pool.

**Saving.** Turns the two remaining unbounded maps into working sets. This is the stage that makes
resident memory a function of *active* tenants.

**Risk.** Medium — highest of any stage here, because a wrong gating condition is free credit rather
than a crash. Needs the three new tests before it lands. `idle_ttl` should default conservatively
(an hour or more): eviction trades memory for a cold-start read, and reads under the shard lock are
what stage 7 exists to make cheaper.

**Note.** Evicting the company-aggregate subject makes `flush_dirty_budget_usage` fall back to
`Credits::default()` for `day_used` (`quota.rs:1024`). That only happens when the budget is dirty and
the subject is not, which the gating conditions make impossible — worth an assertion rather than a
comment.

---

## Stage 6 — Flush concurrency, per-shard marking, folded prune

**Change.**

- Pipeline the per-record upserts with bounded concurrency (`JoinSet` + `Semaphore`; the store is
  already `Arc<dyn LimiterStore>`).
- Group `mark_flushed` by shard: one lock acquisition per shard instead of one per record
  (`quota.rs:1082`), and fold `prune_clean_usage` — which currently re-locks and re-scans every shard
  immediately afterwards — into that same pass. `flush_dirty_budget_usage` has the identical
  per-company re-lock at `quota.rs:1061`.
- Give the three `snapshots` vectors a capacity.

**Why.** `flush_dirty` awaits one ScyllaDB upsert at a time. At a 2 ms round trip, ~7,500 dirty
records take longer than the 15 s interval, and `MissedTickBehavior::Skip` in `main.rs` means the
daemon then *silently drops flush cycles* rather than visibly falling behind. Concurrency here is
what keeps the flush interval meaning what it says.

**Invariant.** The flush stays lossless: a mutation landing during a write leaves the record dirty
rather than being hidden by an older completed write. Ordering between records is already
irrelevant — each row is an absolute replacement, and every statement is marked idempotent — but the
version compare-and-set in `mark_flushed` is what makes concurrency safe and must not be weakened.
Prune must still run strictly after marking, or it prunes nothing (it keeps `!is_clean()`).

**Tests that hold it.** `mutation_during_flush_remains_dirty` (`aggregation.rs`) is the direct one.
`flush_publishes_the_counters_admission_is_decided_on` holds the budget-usage half, including
`MemoryStore`'s write *counter* — a flush that rewrites an unchanged row is as wrong as one that
skips a changed one, and that assertion is what catches an over-eager concurrent flush.

**New test.** A flush with one failing write leaves exactly that record dirty and marks the rest,
independent of completion order.

**Saving.** Wall-clock, not memory — but it bounds how large the dirty set can grow between flushes,
which is memory.

**Risk.** Medium. Concurrency around the version protocol. The compare-and-set already makes it
correct; the risk is in the refactor, not the design.

---

## Stage 7 — Cold-start round trips

**Change, two parts.**

1. `tokio::try_join!` the independent reads: (daily point read ‖ hour range) in `ensure_subject`, and
   (`load_budget` ‖ month `load_range`) in `ensure_budget`. The month range start is derived from the
   clock (`time_frame::month_start_day`), **not** from the budget row, so it genuinely does not
   depend on it. Five sequential round trips become three.
2. Two-phase load: check under the lock, release it, load, re-lock and `entry().or_insert_with()`.

**Why.** The walkthrough is candid that `ensure_access` awaits ScyllaDB under the shard mutex, but
the cost compounds: a cold user's first request does `load_user_access` → `load_exact`(daily) →
`load_range`(hour, 12 rows) → `load_budget` → `load_range`(month) — five sequential round trips with
the shard mutex held, blocking every other company in that shard. At the default
`available_parallelism()` shards, one cold company stalls 1/8 of all traffic for ~10 ms. Stage 5
makes cold starts more frequent by design, which is why this stage follows it.

**Invariant.** A duplicate concurrent load must discard the loser, never clobber a record that has
already been charged. `merge_loaded` (`aggregation.rs:84`) is written for exactly this and documents
it; `or_insert_with` gives the same property for `subjects` and `budgets`.

**Tests that hold it.** `platform_aggregate_extends_the_absolute_row_after_restart` and
`a_restart_recovers_the_pool_and_does_not_lose_entitlement_to_it` hold the load paths.

**New test.** Two concurrent cold requests for the same `(company, user)` charge twice in total —
neither loses a charge to the other's load.

**Risk.** Medium. Dropping a lock mid-operation is where races get introduced. Part 1 (`try_join!`)
is strictly safe and can land on its own; part 2 should be a separate commit.

---

## Stage 8 — De-globalise `platform_usage`

**Change.** Replace the single `platform_usage` map with per-shard partials, summed at flush time
into the one absolute row.

**Why.** Every admitted charge takes `self.platform_usage.lock()` (`quota.rs:579`) — one mutex shared
by all shards, serialising them at the tail of `admit_at`. The sharding buys much less than it looks
like it does.

Worse, the lock is taken *before* `ensure_platform_usage`, which on a cold five-minute frame performs
a ScyllaDB read **while holding the process-global lock**. That happens every 300 s by construction:
a periodic spike where one request's round trip blocks every charge in the process.

**Two fixes, different sizes.** The spike alone is fixed cheaply — have the flush task pre-warm the
next frame's key, or load outside the lock and `entry().or_insert()`. The contention needs the
per-shard split.

**Invariant.** The comment being defended at `quota.rs:365` is that competing absolute snapshots must
not overwrite one another. Per-shard partials preserve it as long as the flush is the sole writer of
that row and sums every shard before writing — which it is, and must remain.

**Tests that hold it.** `platform_aggregate_is_shared_across_company_shards` is precisely this
contract; `platform_aggregate_extends_the_absolute_row_after_restart` holds the load side, and
`accepted_request_dirties_company_user_and_platform_rows` the write side. All three must pass
unchanged — if the partials leak into what the row contains, one of them fails.

**Risk.** Medium-high, and lowest value per unit of risk of anything here — it is a throughput fix
for a load the project does not have yet. The cheap pre-warm is worth doing now; the split can wait
for evidence.

---

## Stage 9 — Smaller items, independent of the above

- **`reqlog/errors.rs:40`** — `should_write` does `code_line.to_string()` on **every** call, including
  the suppression path, which is the gate's entire purpose. Re-key as
  `HashMap<i32, Vec<(Box<str>, Instant)>>`: the id is already a hash of the code line, so the inner
  list is almost always length 1, the hot path becomes allocation-free, and
  `a_hash_collision_does_not_suppress_the_other_line` keeps holding the collision behaviour that
  rules out keying by hash alone. `evict_oldest` is also an O(20,000) scan per new distinct line once
  at capacity — its comment justifies the linear scan, and it stays justified, but it is worth noting
  that it is the same scan every time rather than once.
- **`Cargo.toml` release profile** — add `codegen-units = 1` beside `lto = "thin"` (or move to
  `lto = "fat"`) for cross-crate inlining on the hot path. Measure the build-time cost; it is real.

---

## Deliberately not doing

**`panic = "abort"`.** It would shrink the binary and remove landing pads, and it is the wrong call
here. `server.rs` spawns a handler task per frame; a panicking handler currently costs one
connection, and `JoinSet` reports it. Under abort it kills a daemon the walkthrough states must be
the single active process — with the Go side failing closed behind it, that converts one bad frame
into a platform-wide outage.

**Replacing the musl allocator.** This is the largest single unknown. The daemon ships as a static
musl binary, and mallocng is weak under multithreaded churn in both throughput and RSS retention —
which is exactly this workload (per-flush clones, per-record encode vectors, per-request task
futures). `mimalloc` typically cuts RSS materially, but its build script needs a cross C compiler,
which breaks the "install a prebuilt binary, no toolchain on the host" property documented at
`Cargo.toml:33` and relied on by `configure_server_utils.py`. There is no production-grade pure-Rust
replacement.

That is a deliberate trade, not a drive-by: it costs a documented deploy property. Stages 2 and 6
reduce allocation churn directly, which reduces how much the allocator choice matters — so the honest
sequencing is to land them, measure RSS, and only then decide whether the remaining gap is worth the
build change. If it is, the alternative worth pricing first is building against `-gnu` where hosts
allow it.

**`MAX_FRAME_SIZE` buffers.** 1118 B held as a stack array inside each connection's future
(`server.rs:189`) is ~1.1 MB at the 1,024-connection default. Fine as it is; recorded here so a
future reader does not mistake it for a leak.

---

## Order, and why

| # | Stage | Value | Risk | Lands alone |
|---|---|---|---|---|
| 1 | Evict stale `access` | High | None | yes |
| 2 | `RoutedCredits` → `Vec`, encode under lock | High | Low | yes |
| 3 | Integer hasher + `get_disjoint_mut` | Medium | Low | yes |
| 4 | `TokenBucket` → `u64`, shared `last_refill` | Medium | Low-med | **must** |
| 5 | Idle eviction for `subjects` / `budgets` | High | Medium | **must** |
| 6 | Flush concurrency + per-shard marking | Medium | Medium | yes |
| 7 | `try_join!` cold reads; then load outside lock | Medium | Medium | two commits |
| 8 | `platform_usage` pre-warm; then per-shard split | Low | Med-high | two commits |
| 9 | `errors.rs` allocation, `codegen-units` | Low | None | yes |

Stages 1-3 are self-contained and can go in together if that reads better as a single commit. 4 and
5 each want their own commit: 4 is the only arithmetic change, and 5 is the only one where a mistake
is free credit rather than a crash. 4 precedes 5 because 5 uses the `last_refill` field 4 leaves
behind. 7 follows 5 because 5 makes cold starts deliberately more frequent.

---

## Verification

Per stage:

1. `cargo test` green with **no edits to the tests named above**. An edit to one of those means the
   change reached further than intended — that is the signal, and it should be treated as a failure
   rather than reconciled.
2. `cargo test --test lock_tcp --test request_log_tcp --test bridge_http` for the integration
   contracts.
3. The Go side untouched: `TestChargeFrameMatchesTheRustAuthVector` and
   `TestChargePayloadMatchesTheWireOffsets` still pass, since no stage changes the wire format. Worth
   running anyway, because "no stage changes the wire format" is a claim, not a fact.

For the memory claims specifically, `size_of` assertions on `SubjectState` and the route-map entry
are worth adding as real tests rather than a throwaway probe: they are the only thing that will
notice when a future field silently doubles a per-user cost.

RSS before and after, measured the same way both times — the daemon is long-lived and its interesting
memory behaviour is at the hour scale, not the minute scale, so a short run proves nothing about
stages 1 and 5.
