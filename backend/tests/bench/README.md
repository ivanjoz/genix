# Backend benchmarks

Benchmarks that measure what a GET costs on a VPS-shaped server: ScyllaDB and fareward on the same
machine, the backend in a long-running process, colbin responses compressed with zstd. They skip
unless `GENIX_BENCH=1` is set.

| Layer | Where | Measures |
|---|---|---|
| A — stages | `tests/bench/stages_test.go` | One request at a time, `-benchmem`: ORM scan → + colbin → + zstd, token check, fareward charge at 1/64/512/2048 in flight, handler alone under concurrency |
| B — router | `router_bench_test.go` (package `main`) | `LocalHandler` end to end at a fixed number of requests in flight: req/s, p50/p99, peak heap, error %, log bytes per request |

Endpoints: `GET.products`, `GET.sale-orders` (pending tab), `GET.products-stock`, each as a
**first sync** (no cursor, the whole view) and an **incremental sync** (a cursor with nothing new,
the steady state of every page load). Datasets: companies 101/102/103 with 1k/10k/50k products,
stock rows and sale orders (sale orders are capped at 5000 per tab by the handler).

## The stack

Everything is named by `bench.config.toml` (throwaway secrets, keyspace `genix_bench`), which both
the benchmarks and fareward read through `GENIX_CONFIG_FILE`. Nothing touches the dev `config.toml`,
the remote database or S3.

```sh
# 1. ScyllaDB, sized like a 4 GB server's share. Host networking because fareward's Rust driver
#    dials the address the node advertises; 29042 because 19042 is Scylla's own shard-aware port.
podman run -d --name genix-bench-scylla --network host docker.io/scylladb/scylla:latest \
  --smp 2 --memory 2G --overprovisioned 1 --developer-mode 1 \
  --listen-address 127.0.0.1 --rpc-address 127.0.0.1 --broadcast-rpc-address 127.0.0.1 \
  --native-transport-port 29042 --seeds 127.0.0.1

# 2. Schema + base rows (once per container). Run from backend/.
GENIX_CONFIG_FILE=$PWD/tests/bench/bench.config.toml go run . fn-init

# 3. fareward on 24013/24012, beside the dev daemon on 14013/14012. Run from fareward/.
cargo build --release
GENIX_CONFIG_FILE=$PWD/../backend/tests/bench/bench.config.toml ./target/release/fareward

# 4. Seed the three datasets and their credit budgets (idempotent). Run from backend/.
GENIX_BENCH=1 go test ./tests/bench/ -run TestSeedBenchData -v
```

Why the S3 pieces are absent: on the Scylla driver the file store only holds snapshots, which a VPS
keeps in its local snapshot cache, so `Setup()` installs that cache over an in-memory store (memory
tier only: every run starts cold). `GENIX_BENCH_NO_SNAPSHOTS=1` turns it off, to measure first syncs
read from ScyllaDB. The genix-s3 text search index is replaced by a no-op, because product writes
would otherwise go to S3 — only the seeder writes.

## The FoundationDB stack (`-tags fdb`)

```sh
# Image and container (fdbserver + the fdbgate daemon); the socket appears in /run/user/<uid>/genix-fdb
podman build -t genix-fdb:8.0.0 -f genix-orm/fdb/container/Containerfile genix-orm/fdb
podman run -d --name genix-bench-fdb --network host -e FDB_NETWORKING_MODE=host -e FDB_PORT=24500 \
  -v /run/user/$(id -u)/genix-fdb:/run/genix-fdb:z genix-fdb:8.0.0 --memory 2GiB --cache-memory 1GiB
```

Point a copy of `bench.config.toml` at it with `[fdb] socket = "/run/user/<uid>/genix-fdb/fdbgate.sock"`
(select it with `GENIX_CONFIG_FILE`) and add `-tags fdb` to every `go test` below. No `fn-init` is
needed. fareward does not run against FDB yet: set `GENIX_BENCH_NO_FAREWARD=1`, which skips the
credit budgets when seeding and limits you to the database-only benchmarks (stages, incremental,
HandlerConcurrent). Results and setup of the first comparison: `docs/FOUNDATIONDB_PLAN.md` §14.

## Running

```sh
# Layer A
GENIX_BENCH=1 go test ./tests/bench/ -run '^$' -bench 'Stage|CheckUser' -benchmem -benchtime 20x
GENIX_BENCH=1 go test ./tests/bench/ -run '^$' -bench FarewardCharge -benchtime 20000x
GENIX_BENCH=1 go test ./tests/bench/ -run '^$' -bench HandlerConcurrent -benchtime 400x -cpu 4

# Layer B — -cpu 4 approximates a 4 vCPU server whose other cores belong to Scylla
GENIX_BENCH=1 go test . -run '^$' -bench 'Router/incremental' -benchtime 10000x -cpu 4
GENIX_BENCH=1 GENIX_BENCH_MAX_HEAP_MB=4096 go test . -run '^$' -bench 'Router/first' -benchtime 600x -cpu 4 -timeout 60m

# Profiles: add -cpuprofile /tmp/cpu.out -memprofile /tmp/mem.out -o /tmp/bench.test, then
go tool pprof -top /tmp/bench.test /tmp/cpu.out
```

- `-benchtime Nx` should be well above the concurrency, or most workers never get a request.
- `GENIX_BENCH_CONCURRENCY=64,512` overrides the concurrency sweep (router and HandlerConcurrent).
- Router scenarios: `1k`/`10k`/`50k` send every request to one company, so one partition on one
  Scylla shard. `multi8x10k` rotates requests over eight 10k companies (201–204 on shard 1, 210–214 on shard 0), which spreads the
  load across shards the way a real multi-tenant server does. `TestSeedBenchData` seeds all of them.
- The router benchmark runs with the server's admission control (`ConfigureServerAdmission`:
  request slots + GOMEMLIMIT), so requests over the slots queue and, after 2 s, answer 503 — counted
  in `err-%`. `GENIX_BENCH_NO_ADMISSION=1` turns it off for a comparison.
- `GENIX_BENCH_MAX_HEAP_MB` (default 6144) stops a run whose live heap crosses it, instead of
  letting the machine that also hosts Scylla reach the OOM killer.
- Scylla-side load shedding: `podman exec genix-bench-scylla curl -s localhost:9180/metrics | rg reads_shed`.

## Reading the metrics

| Metric | Meaning |
|---|---|
| `req/s`, `p50-ms`, `p99-ms` | Throughput and latency at that number of requests in flight |
| `peak-heap-MB` | Highest live heap seen (runtime/metrics, sampled every 5 ms) |
| `peak-go-MB` | Everything the Go runtime had mapped — the closest in-process figure to RSS |
| `err-%` | Requests that did not answer 200; the first error is logged under the result |
| `log-B/op` | Bytes `core.Log` and the ORM printed to stdout per request (on a VPS every line prints) |
| `resp-B/op` | Bytes written to the client, after compression |

The net/http server is not in the loop (requests go straight to `LocalHandler`), so per-connection
costs — a goroutine and about 8 KB of buffers each — come on top of the router figures.
