# Plan — `server_metrics`: one row every 5 seconds, written by server_utils

Status: **implemented**. `src/sysmetrics/` holds the collector and the writer, the schema is
`backend/core/types/server_metrics.go`, and 21 tests cover the two sides. Two things came out
differently than described below: the bound row is an array and not a tuple (Rust stops deriving
`Debug` and `PartialEq` for tuples at twelve elements, and this row has fifteen), and the previous
sub-sample's counters are `Option`s rather than plain numbers — the reason is a real defect the
first draft had, recorded under "Collection".

Still outstanding: `deploy.sh` → [5] Recrear Tablas, so the table exists before the daemon is
restarted against it. See "Deploying it".

Today the server panel at `/system/server-panel` shows live numbers and forgets them: the Go
collector in `backend/system/` samples on demand and streams the snapshot over SSE, so nothing
survives the tab being closed.

This adds the missing half — a daemon in `server_utils/` that samples the box every second and, every
five seconds, persists a fixed-width row holding the worst second of that window.

It follows the arrangement `user_logs` already uses, and for the same reason: **the Go ORM owns the
schema, the Rust daemon owns the writes**. `server_utils` is the only process that is always
present on the box (the backend may be a Lambda), so it is the only one that can promise a
continuous series.

Scope is the collector and the table. The live SSE panel is untouched; reading the history back is
a later job.

## Decisions taken

Answered before writing this:

1. **One box.** `scylla-server.service`, `genixsearch.service`, `genix-server-utils.service` and —
   when not on Lambda — `genix.service` all run on the same host, so per-service CPU and memory are
   local reads. No host dimension in the key.
2. **Network in 5 KB/s units.** int16 range becomes 0–163 MB/s with 5 KB/s of resolution. Idle
   traffic (the 1–2 KB/s the panel shows now) rounds to 0 or 1 rather than vanishing, and a
   saturated gigabit link clamps at 32767.
3. **CPU as percent of the whole machine.** Scylla pinning all 8 of 8 cores reads 100.00% = `10000`,
   not the top-style 800%. Every CPU column is then 0–10000, always fits an int16, and the
   per-service numbers are directly comparable to each other and to the host total.
4. **Every column is a peak, not an average.** The collector samples every **1 second** and each
   row carries the **maximum of the five sub-samples** in its window — for every metric, CPU and
   network included. See "What a row means" below, because this choice has one consequence worth
   stating up front.
5. **Collector and table only.** No read handler, no panel change, no removal of the SSE stream.

Two things I decided while writing this, both cheap to reverse — say the word:

- **Slot is 0-based, `0..17279`**, not `1..17280`. It is exactly `secondsIntoDay / 5`, with no `+1`
  to remember on either side, and it matches `FrameOfDay` in `user_logs.go`, which is already
  0-based. One character to change if you want the 1-based range.
- **`ServerUtilsCpuPercent` and `SearchCpuPercent` are included** even though you only asked for
  those two services' memory. The CPU number comes out of the same `cpu.stat` file the collector
  already opens for the memory read, so it is 4 bytes per row and two lines of code — and without
  them the host CPU total cannot be attributed. Drop them if you'd rather not carry them.

## What a row means

**The worst second of its five.** Not a point sample, not an average. The collector takes a full
sample every second, holds a running maximum per metric, and every fifth second writes that maximum
and resets. A service that pins the CPU for one second inside the window is visible in the row; with
a 5-second average it would have read as a fifth of that.

The consequence, stated so no reader of this table is misled later: **slots cannot be summed into
totals.** Adding up `NetRxRate` across a day overstates the bytes actually transferred, and adding
up `CpuPercent` overstates the CPU time actually burned, because each value is a peak standing in
for five seconds. This series answers "how bad did it get, and when" — for "how much was used in
total", the counters in `/proc` are the source, not this table.

The 1-second cadence also means rate metrics are computed over real elapsed time measured between
sub-samples, never an assumed 1.000 s, so tick jitter cannot inflate a rate.

## The table

`backend/core/types/server_metrics.go`, table ID **47** (46 is the highest in use). Every column
below the key is the peak of its window; the units describe a single sub-sample.

| Column                  | Type    | Meaning                                                 |
| ----------------------- | ------- | ------------------------------------------------------- |
| `Date`                  | `int16` | **Partition.** Unix day.                                |
| `Slot`                  | `int16` | **Clustering key.** `secondsIntoDay / 5`, `0..17279`.   |
| `CpuPercent`            | `int16` | Host CPU, hundredths of a percent.                      |
| `MemPercent`            | `int16` | Host memory used, hundredths of a percent.              |
| `DiskPercent`           | `int16` | Mount usage, hundredths of a percent.                   |
| `NetRxRate`             | `int16` | RX in 5 KB/s units.                                     |
| `NetTxRate`             | `int16` | TX in 5 KB/s units.                                     |
| `BackendMemMb`          | `int16` | `genix.service` anonymous memory, MB.                   |
| `BackendCpuPercent`     | `int16` | `genix.service` CPU, hundredths of a machine percent.   |
| `ServerUtilsMemMb`      | `int16` | `genix-server-utils.service`.                           |
| `ServerUtilsCpuPercent` | `int16` | ″                                                        |
| `SearchMemMb`           | `int16` | `genixsearch.service`.                                  |
| `SearchCpuPercent`      | `int16` | ″                                                        |
| `ScyllaMemMb`           | `int16` | `scylla-server.service`.                                |
| `ScyllaCpuPercent`      | `int16` | ″                                                        |

`DisableDefaultColumns: true` and no indexes — same as `user_logs`. There is no `created`/`updated`
to maintain on a row that is written once and never touched, and the only query this table will ever
serve is "give me one day's slots", which the partition already answers.

Cost: 17280 rows/day, ~30 bytes of values each — about 500 KB of values per day, ~15 MB at the
default 30-day TTL. The TTL is per-row and set from config, so a partition expires as a whole day.

### Encoding rules

- **Percent** → hundredths, clamped to `0..10000`. `23.45%` is `2345`.
- **Memory** → megabytes, saturating at `32767` (32 GB). A host bigger than that reports its ceiling
  rather than wrapping.
- **Network** → `bytesPerSecond / 5120`, saturating at `32767`.
- **`-1` means "not measured".** This is the whole answer to the Lambda case: when
  `genix.service` has no cgroup on the box, `BackendMemMb` and `BackendCpuPercent` are both `-1`,
  which a reader can tell apart from a genuine `0`. The same rule covers any service that is
  stopped, and any individual metric whose read failed — the rest of the row is still written.
  The running maximum is taken over *valid* sub-samples only, so a single failed read inside a
  window does not poison the column, and `-1` survives to the row only when not one of the five
  sub-samples produced a value. `max(-1, 0)` must never be allowed to yield `0`.

## Collection

All from cgroup v2 and `/proc`, no new process spawned and nothing shelled out. Roughly eleven small
virtual files per sub-sample — three host files plus two per service — so eleven reads a second,
which costs on the order of a millisecond and never touches a disk.

**Per service**, from the unit's cgroup directory — *searched for* under `/sys/fs/cgroup`, not
assumed. The first implementation hardcoded `system.slice/<unit>` and that was wrong on a stock
install: Scylla's packaging puts its unit at
`/sys/fs/cgroup/scylla.slice/scylla-server.slice/scylla-server.service`, so the one service most
worth watching was the only one that always reported absent. The path is resolved once and cached;
a failed search is retried on a 30-second interval, because an absent unit is the *normal* state for
the backend on Lambda and re-walking the tree every second costs 12 ms a pass for nothing.

- Memory: `memory.stat` → `anon + file_mapped`. Anonymous pages plus the file pages the service has
  actually mapped, which reconstructs `VmRSS` — measured against `/proc` on a real host it matched
  to the byte (40948 kB). `anon` alone was the first attempt and under-reports badly: it equals
  `RssAnon` but misses `RssFile`, 29 MB of the 41 MB a Go backend really holds. `anon + file` errs
  the other way, dragging in 88 MB of cold page cache for a quiet Scylla. `VmRSS` is also what the
  live SSE panel reports, so both views of the same second now agree.
- CPU: `cpu.stat` → `usage_usec`, a monotonic counter. The delta since the previous sub-sample
  divided by `elapsedMicros × cpuCount` gives the machine-percent for that second.

Two file reads per service, correct for multi-process and multi-threaded units alike, and a missing
directory is exactly the "service is absent" signal. `cpuCount` comes from counting the `cpuN` lines
in `/proc/stat`, so no `num_cpus` dependency.

**Host-wide**, mirroring the algorithms already in `backend/system/metrics_collector.go` so the
stored series and the live panel cannot disagree:

- CPU: `/proc/stat` aggregate line, `(totalDelta - idleDelta) / totalDelta`.
- Memory: `/proc/meminfo`, `1 - MemAvailable/MemTotal`.
- Network: `/proc/net/dev`, byte-counter deltas on the configured interface, or the first
  non-`lo` interface when it is left empty.
- Disk: `statvfs` on the configured mount. This one syscall is the reason for a new
  `libc = "0.2"` dependency — no file under `/proc` reports free space.

**Timing.** The loop ticks once a second, aligned at startup to the next whole second, and a row is
written on the tick that crosses a 5-second boundary of the wall clock — not on every fifth tick of
a counter. Day and slot come from the clock at that moment, so a restart lands back on the same
grid, and a skipped tick (`MissedTickBehavior::Skip`) leaves a genuine hole in the series instead of
shifting every later slot.

**A counter that could not be read must leave no baseline behind.** This is the one defect the
first draft of this plan had, found by running the collector against a real box: storing a zero for
a failed `/proc` read makes the *next* sub-sample subtract from zero and report the counter since
boot — a since-boot average for CPU, and for the network a rate that pins the column at its
ceiling. Because the row keeps the window's maximum, one such artefact wins its window outright and
looks exactly like a real spike. Every counter in `PreviousCounters` is therefore an `Option`, and
a delta with an absent side reports `NOT_MEASURED`.

Two more consequences of sub-sampling worth pinning down in code:

- Rate metrics need a predecessor, so the first sub-sample after startup only establishes the
  baseline and contributes no value. A collector started mid-window writes that window's row from
  however many valid sub-samples it managed, rather than waiting for a clean one.
- The running maximum is reset *after* the row is written, not before the first sub-sample of the
  next window, so no sub-sample can be counted into two rows or dropped between them.

**Failure policy: fails open, like `reqlog` and unlike the limiter.** A dropped metrics row costs a
gap in a chart; taking the process down would stop the rate limiter, the lock service and the SSE
bridge. Every error is a warning and a counter.

The `INSERT` is prepared **lazily and retried every 60 s** while it fails. The first version prepared
once at startup and disabled the collector when that failed, which is precisely how it broke on the
first real deployment: the daemon came up before `fn-homologate` had created the table, and the
collector stayed off for the life of the process while everything else ran normally. A table
arriving minutes after the daemon is ordinary deploy ordering, not an exceptional condition.

The collector therefore always spawns when enabled; there is no fallible startup step left.

## Wiring

New module `server_utils/src/sysmetrics/`:

- `mod.rs` — docs, the config-driven spawn, the `-1` sentinel and the saturating encoders.
- `collector.rs` — `/proc` and cgroup reads, delta state, one `MetricsSample` per tick.
- `writer.rs` — the prepared `INSERT ... USING TTL`, marked idempotent, one row per tick.

`main.rs` spawns it next to the existing `flush_task`, reusing the `session` already opened for the
limiter and the request log (a second pool to the same nodes buys nothing) and the existing
`shutdown_receiver`.

New config section, following the `[request_log]` pattern in `config.rs` — every key optional with a
default, every key overridable by env var:

```toml
[server_metrics]
enabled           = true
sample_seconds    = 1                             # cada cuánto se muestrea
row_seconds       = 5                             # cada cuántos segundos se escribe el máximo
ttl_days          = 30
disk_mount        = "/"
network_interface = ""                            # vacío = la primera interfaz que no sea lo
backend_unit      = "genix.service"               # ausente en Lambda: sus columnas quedan en -1
server_utils_unit = "genix-server-utils.service"
search_unit       = "genixsearch.service"
scylla_unit       = "scylla-server.service"
```

`row_seconds` decides what the clustering key means, so it is validated at load rather than trusted:
it must divide 86400 and be at least 3, otherwise the slot count overflows int16 or the last slot of
a day is a short one. `sample_seconds` must divide `row_seconds`.

## Files

| File                                          | Change                                              |
| --------------------------------------------- | --------------------------------------------------- |
| `backend/core/types/server_metrics.go`        | new — the two paired structs and `GetSchema()`       |
| `backend/core/types/server_metrics_test.go`   | new — schema and column-order pin                    |
| `server_utils/src/sysmetrics/{mod,collector,writer}.rs` | new                                       |
| `server_utils/src/lib.rs`                     | declare the module                                   |
| `server_utils/src/main.rs`                    | spawn the collector task                             |
| `server_utils/src/config.rs`                  | `ServerMetricsConfig` + loading                      |
| `server_utils/Cargo.toml`                     | `libc = "0.2"` for `statvfs`                         |
| `config.example.toml`                         | the `[server_metrics]` section, documented           |
| `server_utils/README.md`                      | one paragraph — the third thing the daemon now does  |

## Tests

- **Rust**: the encoders at their boundaries (clamp, saturate, the `-1` sentinel); day/slot
  derivation across a UTC midnight; `memory.stat`, `cpu.stat`, `/proc/stat`, `/proc/meminfo` and
  `/proc/net/dev` parsed from fixture strings; an absent unit directory producing `-1` for both of
  its columns rather than an error. Then the three the max-of-five arrangement introduces: a window
  of five sub-samples emitting the peak of each column and not the last or the mean; `-1` losing to
  any valid sub-sample and surviving only when all five are invalid; and the accumulator resetting
  exactly at the row boundary, so no sub-sample lands in two rows.
- **Go**: `server_metrics_test.go` asserting the schema compiles and that the column list matches
  the order the Rust `INSERT` binds — that assertion is the only thing tying the two sides together,
  the same role `user_logs_test.go` plays for `reqlog`.
- **Static**: `cd scripts && go run . check_tables`.

## Deploying it

1. `deploy.sh` → **[5] Recrear Tablas** — regenerates `controllers.generated.go` (which is what
   registers the new table) and runs `fn-homologate` to create it in Scylla.
2. `scripts/configure.py 37` — rebuilds and restarts the daemon.

## Not in this plan

Reading the series back: the Go handler for a day's slots, and the history view in the panel. The
live SSE stream and `backend/system/metrics_collector.go` stay exactly as they are.
