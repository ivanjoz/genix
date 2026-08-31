# server_utils → auth-limiter rename — completed

The submodule folder, the Rust crate, the Go client package, the TOML section, the env vars and the
systemd units all carried the name `server_utils` while the repository behind them was
`github.com/ivanjoz/auth-limiter`. Scope chosen: **rename everything**, with two contracts held back.

Design decisions live in `auth-limiter/RATIONALE.md` and `backend/RATIONALE.md`.

## What the name became

| Surface | Before | After |
| --- | --- | --- |
| Submodule folder + `.gitmodules` name | `server_utils` | `auth-limiter` |
| Rust crate + binary | `genix-server-utils` | `auth-limiter` |
| `RUST_LOG` target | `genix_server_utils` | `auth_limiter` |
| systemd units | `genix-server-utils{,-restart}.{service,path}` | `auth-limiter{,-restart}.{service,path}` |
| Release assets | `genix-server-utils_linux_{amd64,arm64}` | `auth-limiter_linux_{amd64,arm64}` |
| TOML section | `[server_utils]` | `[auth_limiter]` |
| TOML key | `server_metrics.server_utils_unit` | `server_metrics.auth_limiter_unit` |
| Env vars | `SERVER_UTILS_{ADDRESS,PORT,PUBLIC,UNIT}` | `AUTH_LIMITER_*` |
| Go package | `core/server_utils/`, `core/server_utils_api.go` | `core/auth_limiter/`, `core/auth_limiter_api.go` |
| Go identifiers | `ServerUtilsClient`, `ErrServerUtilsUnavailable`, … | `AuthLimiter*` |
| Install scripts | `configure_server_utils.py`, `CONFIGURE_SERVER_UTILS.md`, `test_configure_server_utils.py` | `*_auth_limiter.*` / `CONFIGURE_AUTH_LIMITER.md` |
| Prose / UI label | "Server Utils" | "Auth Limiter" |

## Deliberately NOT renamed — two contracts

1. **`genix-server-utils:v6`** — the HMAC domain separator, mirrored in
   `backend/core/auth_limiter/connection.go` and `auth-limiter/src/service/auth.rs`. The
   *identifiers* holding it were renamed; the *value* was not. It identifies the protocol, and its
   `:v6` suffix is the wire-version channel — a rename is not a wire change and must not spend a
   bump or invalidate frames a deployed backend still signs.
2. **`server_utils_mem_mb` / `server_utils_cpu_percent`** — live Scylla columns in `server_metrics`.
   The Go fields `ServerUtilsMemMb` / `ServerUtilsCpuPercent` carry no column tag, so the ORM
   derives the column name from the field name; they are also the JSON keys the Server Panel reads.
   Renaming them is a schema migration on a table that already holds rows. Both declarations now
   carry a comment saying so.

## Verification

| Check | Result |
| --- | --- |
| `backend`: `go build ./...`, `go vet ./...`, `gofmt -l` | clean |
| `backend`: `go test ./...` | pass, except one **pre-existing** `agent/ragdocs` failure (stale hash for `frontend/core/modules.ts`, a file this change never touched) |
| `auth-limiter`: `cargo build`, `cargo test` | 182 tests pass |
| `scripts`: `python -m unittest discover -s tests` | 80 tests pass |
| `scripts`: `go run . check_tables` | 53 table pairs, no errors |
| `scripts`: `go run . check_module_imports` | 48 packages, no violations |
| `frontend`: `bun run check` | 10 errors, all **pre-existing** in 5 files this change never touched |
| `node --check start.js` | clean |

`DOCUMENTATION.md` provenance hashes: 57 entries staled by this change were refreshed. **189 were
already stale before it** and were left alone — that system is broadly out of date repo-wide and
fixing it is not this change's job.

## Deploy consequences (accepted)

- A host already running the daemon keeps `genix-server-utils.service` until
  `configure_auth_limiter.py` is re-run; the old unit is not removed automatically.
- Any `config.toml` with a `[server_utils]` section, or `SERVER_UTILS_*` in a systemd/Lambda
  environment, silently stops being read. The local `config.toml` was updated in place.
- The first tagged release after this commit cannot reuse prior binaries: the CI reuse check looks
  for the new asset names and will not find them, so it rebuilds once.
