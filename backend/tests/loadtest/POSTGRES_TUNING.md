# PostgreSQL tuning for throughput (2026-11-07)

This is the V2 sales-only benchmark on the 2 vCPU budget, run with `run_postgres.sh` and
`measure_postgres.sh` (`DB_KIND=postgres`) on PostgreSQL 18. It follows the MariaDB tuning
(`MARIADB_TUNING.md`) and ports that tuning's driver changes to pgdriver, so the two compare on equal
terms.

## Setup

- 10 companies, `--sales-v2 --sales-only --settle-later=30`, with a 90 s window. Single runs.
- "Saturation" is `--think-ms=0`. "Think 200" is capped near 190 sales/s by 40 cashiers.
- The raw files are in `backend/measure/<run>`.

The server was already tuned for 1.5 GB:

- `shared_buffers=384MB`, `synchronous_commit=off`;
- `checkpoint_timeout=15min`, `max_wal_size=2GB`;
- `wal_compression=lz4`, `jit=off`.

## Steps

Each step keeps the previous ones.

| step | change | run | sales/s | DB CPU per sale | backend CPU per sale | p99 (stage) |
|---|---|---|---|---|---|---|
| 0 | baseline: 32 connections, one statement per round trip | pg-b-sat-1 | 250.5 | 3.66 ms | 3.46 ms | 245 ms |
| 0 | same, think 200 (2026-11-06) | pg-v2-{1,2} | 191.2 / 191.7 | 2.71 / 2.71 ms | 3.0 ms | 43 / 39 ms |
| 1 | backend pool of 16 connections | pg-pool16-sat-1 | 289.8 | 3.11 ms | 3.17 ms | 209 ms |
| 1 | pool of 8 (about the same; 16 kept) | pg-pool8-sat-1 | 285.8 | 3.06 ms | 3.28 ms | 205 ms |
| 1 | pool 16, think 200 | pg-pool16-v2-1 | 190.0 | 3.08 ms | 3.04 ms | 40 ms |
| 2 | pipelined write transactions; inserts of minted keys skip the locking pre-read | pg-pipe-sat-1 / -2 / -3 | **350.4 / 353.1 / 338.7** | **2.60 / 2.55 / 2.76 ms** | **2.76 / 2.72 / 2.79 ms** | **173 / 171 / 179 ms** |
| 2 | same, think 200 | pg-pipe-v2-1 | 193.6 | 2.31 ms | 2.43 ms | 23 ms |
| 3a | `wal_compression=off` (fresh container): no gain, not kept | pg-walc-off-sat-1 | 338.8 | 2.78 ms | 2.77 ms | 179 ms |
| 3b | + `shared_buffers=768MB` (fresh container): no gain, not kept | pg-sb768-sat-1b | 337.6 | 2.79 ms | 2.77 ms | 181 ms |
| 3c | step 2 + `wal_level=minimal`, `max_wal_senders=0` (fresh container): no gain, not kept | pg-walmin-sat-1 | 318.5 | 2.99 ms | 2.93 ms | 194 ms |

`wal_level=minimal` skips WAL only for bulk loads into a table created or truncated in the same
transaction, and for index builds. A sale's INSERTs and UPDATEs are logged in full at every level,
and the run still wrote about 1 GB of WAL. The setting also rules out replicas and point-in-time
recovery, so even a zero gain makes it a bad trade.

`pg-pipe-sat-2` and `-3` were meant as steps 3a and 3b. They ran unchanged because the container's
command-line `-c` settings override `ALTER SYSTEM`, so they are repeats of step 2: three runs of
338.7–353.1 sales/s. The two that started on a freshly started server (`-3`, and steps 3a and 3b)
sit at about 338, against about 350 on a warm one. That difference is noise or warm-up, not a
setting. `run_postgres.sh stack` now takes `POSTGRES_FLAGS` for such settings.

## Result

Step 2 is the tuned configuration: a pool of 16 connections, the pipelined pgdriver, and the server
as it was.

| | PostgreSQL, before | PostgreSQL, tuned | MariaDB InnoDB, tuned |
|---|---|---|---|
| sales/s, saturation | 250.5 (249–253 earlier) | **338.7–353.1** (+35–40%) | 277.8 (276.2 with pool 16) |
| DB CPU per sale, saturation | 3.66 ms | **2.55–2.79 ms** | 3.74 ms |
| backend CPU per sale, saturation | 3.46 ms | **2.72–2.79 ms** | 2.90 ms |
| sale p99, saturation | 245 ms | **171–179 ms** | 198 ms |
| sales/s, think 200 | 191 (cap) | 193.6 (cap) | 189.0 (cap) |
| DB CPU per sale, think 200 | 2.71 ms | **2.31 ms** | 3.51 ms |
| sale p99, think 200 | 39–43 ms | **23 ms** | 43 ms |

- **With the same driver changes, PostgreSQL leads MariaDB by about 25% at saturation.** It needs
  about 30% less database CPU per sale, and its p99 is lower at both loads. Section 1b of
  `DATABASE_COMPARISON.md`, where tuned MariaDB beat stock PostgreSQL, compared a tuned driver with
  an untuned one.
- **MariaDB gains nothing from the smaller pool** (276.2 against 277.8 sales/s): its thread pool
  already bounds the concurrency on the server.
- **The gains are on the client side.** The data fits in memory and commits do not wait for the
  disk, so the server settings left (WAL compression, a larger buffer pool) change nothing
  measurable. What helps is fewer processes competing for the 2 vCPUs and fewer round trips per
  transaction.
- **To apply it outside the benchmark:**
  - Set `[postgres].max_conns` to about 8 × vCPUs. The scripts keep 32 by default so the earlier
    runs stay reproducible; pass `POSTGRES_MAX_CONNS=16`.
  - The driver change ships with pgdriver itself.

### Step 1: fewer backends

PostgreSQL runs one process per connection. With 32 of them on 2 vCPUs, the kernel switches between
busy backends, and their lock waits overlap. At 16 the same work goes through fewer processes, which
gives 16% more sales/s and 15% less database CPU per sale. Pool 8 is the same within noise. This is
PostgreSQL's equivalent of MariaDB's thread pool. The setting is `[postgres].max_conns`
(`POSTGRES_MAX_CONNS` in the scripts).

### Step 2: pipelined write transactions (pgdriver)

The same change as MariaDB's step 3, through pgx batches:

- `runTransaction` runs on the native pgx connection under a pooled database/sql connection.
- BEGIN and every statement without a result are queued in a `pgx.Batch`. The batch is pipelined:
  - with the next read, which rides in the same batch and sees the queued writes;
  - or with the COMMIT.
- A sale's write transaction is then one or two round trips: the locking read together with BEGIN,
  then the writes together with COMMIT.
- An error stops the batch there; the attempt sends ROLLBACK, and a deadlock retries.

In the same step, an insert whose every record took a new autoincrement key skips the `SELECT … FOR
UPDATE` of its stored rows: nothing can be stored under a key the sequence just minted.

Both changes cut client work as much as server work: backend CPU per sale fell from 3.17 ms to
2.76 ms.
