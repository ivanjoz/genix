# auth-limiter → fareward rename — completed

The daemon has now been through three names. It was `server_utils` while its repository was
`github.com/ivanjoz/auth-limiter`; a first pass renamed everything to `auth-limiter` to match, and
held two contracts back. The repository has since moved to `github.com/ivanjoz/fareward`, so this
pass renames to `fareward` and **takes both held-back contracts with it**.

Design decisions live in `fareward/RATIONALE.md` and `backend/RATIONALE.md`.

## What the name became

| Surface | Before | After |
| --- | --- | --- |
| Submodule folder + `.gitmodules` name + url | `auth-limiter` | `fareward` |
| Rust crate + binary | `auth-limiter` | `fareward` |
| `RUST_LOG` target | `auth_limiter` | `fareward` |
| systemd units | `auth-limiter{,-restart}.{service,path}` | `fareward{,-restart}.{service,path}` |
| Release assets | `auth-limiter_linux_{amd64,arm64}` | `fareward_linux_{amd64,arm64}` |
| TOML section | `[auth_limiter]` | `[fareward]` |
| TOML key | `server_metrics.auth_limiter_unit` | `server_metrics.fareward_unit` |
| Env vars | `AUTH_LIMITER_{ADDRESS,PORT,PUBLIC,UNIT}` | `FAREWARD_*` |
| Go package | `core/auth_limiter/`, `core/auth_limiter_api.go` | `core/fareward/`, `core/fareward_api.go` |
| Go identifiers | `AuthLimiterClient`, `ErrAuthLimiterUnavailable`, … | `Fareward*` |
| Install scripts | `configure_auth_limiter.py`, `CONFIGURE_AUTH_LIMITER.md`, `test_configure_auth_limiter.py` | `*_fareward.*` / `CONFIGURE_FAREWARD.md` |
| Prose / UI label | "Auth Limiter" | "Fareward" |

## The two held-back contracts, now taken

1. **HMAC domain: `genix-server-utils:v6` → `fareward:v7`.** Held in
   `backend/core/fareward/connection.go` (`farewardAuthDomain`) and `fareward/src/service/auth.rs`
   (`DOMAIN`), which must stay byte-identical. Renaming it is not a frame-format change, but it
   invalidates every tag a peer on the old string produces — so it spends a version bump rather
   than leaving two mutually incompatible protocols both answering to `:v6`. The skew now surfaces
   as a failed HMAC on the first frame instead of a silent misread.

   The cross-language vectors were regenerated from the Go side and re-pinned in Rust, so the pair
   still proves the two implementations agree byte for byte:

   | Vector | Old tag | New tag |
   | --- | --- | --- |
   | charge, sequence 0 | `0F 13 FA B1 DE F3 CA A8` | `51 2A 79 02 61 0E A0 CE` |
   | charge, sequence 1 | `A9 87 64 7C D4 89 26 AD` | `36 7A F0 EA 23 BA 00 E8` |
   | lock acquire, sequence 0 | `63 FE 19 83 E2 84 E6 3E` | `10 9D 0A A7 58 22 CB D1` |
   | access invalidation, sequence 0 | `82 E1 EA 44 84 B5 90 74` | `04 EA 41 B9 79 38 55 50` |

2. **Scylla columns: `server_utils_{mem_mb,cpu_percent}` → `fareward_{mem_mb,cpu_percent}`.** The
   Go fields `ServerUtilsMemMb` / `ServerUtilsCpuPercent` in `core/types/server_metrics.go` carry
   no column tag, so the ORM derives the column name from the field name; they are also the JSON
   keys the Server Panel reads. Renamed to `FarewardMemMb` / `FarewardCpuPercent` in lockstep
   across the record struct, the table struct, `config/server_metrics.go`, the column-name test,
   the frontend's `ServerMetricField` union and `DashboardView.svelte`, and the Rust writer's
   INSERT.

## Deploy consequences (accepted)

- **The wire change is a lockstep deploy.** A backend on `:v6` and a daemon on `:v7` fail every
  frame in both directions. There is no compatibility window and no alias — the bump exists to make
  that loud.
- **The column change is a schema change on a live table.** `genix-orm` adds missing columns and
  never drops them, so a deployed `server_metrics` gains `fareward_*` and keeps `server_utils_*`
  until those rows expire under the table's TTL. Nothing back-fills: the Server Panel reads
  not-measured for windows sampled before the deploy. Order does not matter — the daemon's
  `ensure_prepared` retries on a one-minute interval, so a daemon that starts before the backend's
  schema deploy heals on its own.
- A host already running the daemon keeps `auth-limiter.service` (or, older still,
  `genix-server-utils.service`) until `configure_fareward.py` is re-run; the old unit is not removed
  automatically.
- Any `config.toml` with an `[auth_limiter]` or `[server_utils]` section, or `AUTH_LIMITER_*` /
  `SERVER_UTILS_*` in a systemd or Lambda environment, silently stops being read. The local
  `config.toml` was updated in place.
- The submodule url moved to `git@github.com:ivanjoz/fareward.git`. A clone that predates this
  commit needs `git submodule sync --recursive` before `git submodule update` resolves.

## Verification

| Check | Result |
| --- | --- |
| `backend`: `go build ./...`, `go vet ./...` | clean |
| `backend`: `gofmt -l` | clean, except a **pre-existing** `db/db.go` this change never touched |
| `backend`: `go test ./core/fareward/...` | pass, with the regenerated vectors |
| `fareward`: `cargo build --locked`, `cargo test` | 182 tests pass |
| `scripts`: `python -m unittest discover -s tests` | 81 tests pass |
| `scripts`: `go run . check_tables` | 53 table pairs, no errors |
| `scripts`: `go run . check_module_imports` | 48 packages, no violations |
