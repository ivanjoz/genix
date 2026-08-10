# Plan: Rewrite `sse_bridge/` (Go) into `server_utils/` (Rust)

Status: **implemented**. `server_utils/` hosts the bridge in `src/bridge/`; the `sse_bridge/` Go
module, `scripts/configure_sse_bridge.py`, and `scripts/CONFIGURE_SSE_BRIDGE.md` are deleted.
References to `sse_bridge/*.go` below are historical — they describe the source that was ported.

Still outstanding: a manual smoke test against a real browser tab (see §3). Everything else is
covered by the 50 automated tests in `cargo test` plus 14 in
`scripts/tests/test_configure_server_utils.py`.

Target: `server_utils/` absorbs the SSE bridge.

## 1. Decisions already made

- **One process.** `genix-server-utils` becomes a single binary that runs the raw-TCP credit rate
  limiter *and* the HTTP SSE bridge concurrently in the same tokio runtime, under one systemd
  unit. This trades the bridge's current ability to run on a separate host for one less moving
  part — acceptable because both are already meant to sit next to the backend.
- **Delete `sse_bridge/`** (Go module, README, tests) in the same change once the Rust side is
  verified. Pre-alpha, no compatibility shim.
- **Wire protocol is unchanged.** Same HTTP paths, JSON bodies, headers, status codes, and channel
  token format. `backend/agent/channel.go` and `frontend/core/agent/channel.ts` need **zero
  changes**.
- **Two secrets, split by purpose** (see §2.2): `internal_apikey` (new root config key, already
  present in the real `config.toml`) authenticates *service-to-service* calls — the rate
  limiter's TCP frames and the bridge's `X-Bridge-Auth` header. `secret_phrase` is reserved for
  *signing the browser's session token* only. This does touch `backend/agent/bridge.go` (which
  key signs `X-Bridge-Auth`) and `backend/main.go` (which key the rate-limiter client is
  configured with) — both are small, mechanical changes, not protocol changes.

## 2. What has to be built

### 2.1 Cargo layout

Stay a single `[package]` (not a workspace) — the two concerns already share config-loading and
the process lifetime. Each service owns a module tree, so the two can both have an `auth` and a
`server` module without colliding, and the crate root shows only the services plus their shared
config:

```text
server_utils/src/
├── main.rs              # spawns the TCP rate limiter AND the HTTP bridge, shared shutdown
├── lib.rs               # pub mod bridge; pub mod config; pub mod limiter;
├── config.rs            # shared: rate_limit policy, db, both secrets, sse_bridge port
├── limiter/             # unchanged behavior, moved wholesale under this module (see PLAN.md §14)
│   ├── mod.rs
│   ├── quota.rs         # was limiter.rs — RateLimiter + policy types
│   ├── auth.rs          # sequence-bound frame HMAC
│   ├── protocol.rs
│   ├── aggregation.rs
│   ├── credits_blob.rs
│   ├── time_frame.rs
│   ├── storage.rs
│   └── server.rs        # raw TCP listener
└── bridge/
    ├── mod.rs           # pub mod auth; pub mod channel; pub mod http; pub mod token;
    ├── token.rs         # colbin decode subset (see 2.3) + channel-token codec (see 2.4)
    ├── auth.rs          # session-token verification + X-Bridge-Auth verification
    ├── channel.rs       # ChannelRegistry / ClientChannel (async mirror of channel.go)
    └── http.rs          # axum router + the 5 handlers (mirror of handlers.go)
```

New dependencies in `Cargo.toml`:

- `axum` — HTTP server + native SSE support (`axum::response::sse`), keeps hand-rolled framing
  out of the bridge the way `hyper` alone would require.
- `base64` — the channel token and the session token both need URL-safe/standard base64.
- `tokio` gains the `"sync"` (already present), no new tokio features expected; axum needs
  `tokio`'s `"rt-multi-thread"` (already present).

No new dependency for HMAC — `hmac`, `sha2`, `subtle` are already in `Cargo.toml` and get reused
as-is for both the rate-limiter frame auth and the bridge's two auth schemes.

### 2.2 Config

Two root-level secrets, split by purpose, both already required at the project level:

| Key | Env override | Used for |
|---|---|---|
| `internal_apikey` | `INTERNAL_APIKEY` | Service-to-service authentication: the rate limiter's TCP frame HMAC (moved off `secret_phrase`, see below) *and* the bridge's `X-Bridge-Auth` header. |
| `secret_phrase` | `SECRET_PHRASE` | Signing/verifying the browser's session token only. The bridge uses it exclusively for the colbin `UserToken.Hash` check (§2.5) — nothing else in `server_utils` touches it. |

`internal_apikey` already exists as a root key in the real `config.toml` (`K1OzWIN0yarCc9ge`);
it is new to `config.example.toml` and to every Go/Rust struct that reads config. Extend
`AppConfig` (`config.rs`) with both fields plus the bridge's own settings:

```toml
[sse_bridge]
# Public URL is NOT read by this process — only by the Go backend and the frontend, to decide
# whether to use a bridge at all. Documented here for humans, not parsed.
url  = "https://genix-sse.example.com/"
port = 14012
```

- `sse_bridge.port` defaults to `14012` (unchanged from the Go default) if absent.
- Update `config.example.toml`: add root-level `internal_apikey` next to `secret_phrase`, keep
  the `[sse_bridge]` table (drop `apikey`), keep it visually next to `[rate_limit]` since both
  belong to `server_utils` now.

**This also changes already-implemented rate-limiter code and its Go counterpart** — not just
additive config:

- `server_utils/src/config.rs`: `AppConfig` currently loads one `secret_phrase: Vec<u8>` and
  `main.rs` hands it to `limiter::server::run` as the TCP frame HMAC key. Add
  `internal_apikey: Vec<u8>` as a second required root string and switch `main.rs` to pass
  *that* to `limiter::server::run`; keep `secret_phrase` on `AppConfig` (now used only inside
  `bridge::auth`).
- `backend/core/security.go`: add `INTERNAL_APIKEY string` to `EnvStruct`, `InternalApikey
  string \`toml:"internal_apikey"\`` to `fileConfig`, and `env.INTERNAL_APIKEY =
  file.InternalApikey` in `applyToEnv` (mirrors how `SECRET_PHRASE` is already plumbed, lines
  95/157/239).
- `backend/main.go`: the server-utils client is configured with `core.Env.INTERNAL_APIKEY`
  instead of `core.Env.SECRET_PHRASE`. The client package itself is untouched (it takes
  `secret string` as a parameter, agnostic to which config field fills it).
- `backend/agent/bridge.go`'s `makeBridgeServiceAuthHeader`: sign with `core.Env.INTERNAL_APIKEY`
  instead of `core.Env.SECRET_PHRASE`.
- Domain separation (`"genix-rate-limiter:v1"`, `"sse-bridge:v1|"`) already keeps these two uses
  of the same key from colliding with each other or with the session-token HMAC's own domain
  string (`"usrToken:v1"`) — switching keys is a config-clarity change, not a crypto fix.

### 2.3 colbin decode subset (`bridge/token.rs`)

The browser's session token is a **colbin value-mode encoding of a single `UserToken` struct**
(mirrors `backend/core/usuario-accesos.go`'s `UsuarioToken`, 4 encoded fields + 1 skipped):

```go
type UserToken struct {
    CompanyID int32  // no cb tag -> hashed field id from "CompanyID"
    ID        int32  // "ID"
    Created   int32  // "Created"
    Hash      uint64 // "Hash"
    User      string // "User"
    Error     string `cb:"-"` // skipped
}
```

A single (non-slice) struct target still goes through colbin's **records path** with
`recordCount = 1` (confirmed by reading `colbin/decode.go`: `topLevelIsRecords` returns `true` for
`reflect.Struct`). So the wire format to decode is:

```text
message  := [version:1=0x01] [recordCount:uvarint=1] subTable
subTable := [colCount:1] column*
column   := [field_id:1] [flags:1] payload      // field_id looked up, order-independent
```

We do **not** need a general colbin decoder — only:

- `int` columns (`ftInt`, flags bit layout: signed@3, precision@4-6, empty@7) for the three
  `int32` fields and the one `uint64` field. FOR/delta decode + LSB-first bit-unpacking
  (`colbin/bitstream.go`, `colbin/column_int.go` — port these two files' logic directly, they're
  ~100 lines total and self-contained).
- one `string` column (`ftString`) for `User`: an embedded `int` length column (n=1, width 32)
  followed by the raw UTF-8 bytes.
- field-id resolution: FNV-1a-32 over the field name, xor-folded to 8 bits, with linear probing
  over already-assigned ids — computed in **declaration order** (`CompanyID, ID, Created, Hash,
  User`), exactly mirroring `colbin/typeinfo.go`'s `fnv8` + `probeFieldID`. ~15 lines, no need to
  hardcode the resulting ids since the algorithm is small and this keeps it self-verifying against
  the Go side rather than a magic-number table.

This is a **fixed-shape decoder for this one struct**, not a general colbin implementation —
same scope discipline the existing Go `sse_bridge/auth.go` already uses ("mirrors
`core.UsuarioToken` byte for byte... a mismatch decodes silently to zero values").

Base64 layer: the token arrives through `Authorization: Bearer <token>`, base64-encoded with a
substituted alphabet (`_`→`/`, `-`→`+`, `~`→`=`, i.e. `core.MakeB64UrlEncode`'s inverse) *before*
standard base64 decoding — port `decodeBase64URLAlphabet` verbatim.

### 2.4 Channel token codec (`bridge/token.rs`)

Independent of colbin — this is the small custom varint format from `sse_bridge/channel.go`:

```text
bytes = uvarint(companyID) ‖ uvarint(userID) ‖ 6 random bytes (tab)
token = base64url(bytes), unpadded
```

Port `DecodeChannelToken`/`EncodeChannelToken` directly, including the canonical round-trip
rejection (decode, re-encode, compare). `sse_bridge/channel_vectors_test.go` already has shared
test vectors with the Go and TS implementations — reuse the same vectors in a Rust test so all
three copies keep agreeing byte-for-byte.

### 2.5 Auth (`bridge/auth.rs`)

Two schemes, each keyed by a different config secret (§2.2), both already partially covered by
existing crates:

- **Session token** (browser → bridge), keyed by `secret_phrase`: base64-decode (2.4's
  alphabet), colbin-decode into `UserToken` (2.3), then verify `Hash ==
  HMAC-SHA256(secret_phrase, "usrToken:v1" ‖ BE32(CompanyID) ‖ BE32(ID) ‖ BE32(Created) ‖
  User)[..8]` as big-endian `u64`. Port `computeUserTokenHash` exactly (byte layout, domain
  string, truncation). This is the **only** place in `server_utils` that reads `secret_phrase`.
- **Service auth** (`X-Bridge-Auth: <unix_ts>.<hex(hmac)>`, backend → bridge), keyed by
  `internal_apikey`: port `MakeServiceAuthHeader`/`verifyServiceAuthRequest` — domain string
  `"sse-bridge:v1|"`, ±300s skew window, `subtle`'s constant-time compare (already a dependency)
  instead of `hmac.Equal`. Same key as the rate limiter's TCP frame HMAC (§2.2), different domain
  string, so the two never produce comparable tags.

### 2.6 Channel registry (`bridge/channel.rs`)

Async port of `channel.go`/`channels.go`. Tokio equivalents of the Go primitives:

| Go | Rust |
|---|---|
| `chan []byte` (outbound queue, buffered) | `tokio::sync::mpsc::channel<Bytes>` |
| `chan struct{}` (`closed`) | `tokio::sync::Notify` or a `watch::channel<bool>` |
| `map[uint64]chan replyEnvelope` + mutex | `tokio::sync::Mutex<HashMap<u64, oneshot::Sender<ReplyEnvelope>>>` (one-shot fits `AwaitReply`/`DeliverReply` better than Go's buffered chan-of-1) |
| `sync.RWMutex` registry map | `tokio::sync::RwLock<HashMap<String, Arc<ClientChannel>>>` |
| arrival waiters (`AwaitChannel`) | `tokio::sync::Notify` per pending key, or a `broadcast`/waiter list matching the Go structure |

Same behavior contract as today: last-connection-wins on `OpenChannel` (evict + notify replaced),
no buffering (publish to a disconnected tab reports `Delivered:false`), `AwaitChannel` supports a
bounded wait for a reconnecting tab, `FailAllPendingReplies` on disconnect.

### 2.7 HTTP layer (`bridge/http.rs`)

`axum::Router` with the same five routes, same status codes, same CORS behavior:

| Route | Notes |
|---|---|
| `GET /sse?ch=` | `axum::response::sse::Sse` with a keepalive `Event::comment` every 20s (mirrors the Go ticker) matches axum's built-in `KeepAlive` helper — evaluate using it directly instead of hand-rolling the ticker. |
| `POST /in?ch=` | Same JSON envelope, same `Delivered` semantics. |
| `POST /publish` | Same `WaitMs`/`Delivered` contract. |
| `POST /rpc` | Same `TimeoutMs`/`WaitMs`, same status codes (409 no-client, 502 send-failed, 504 timeout). |
| `GET /health` | `{Ok, Channels, UptimeSeconds}`. |

CORS: reuse `tower-http`'s `CorsLayer` for `/sse` and `/in` only (mirrors `withClientCORS` being
applied selectively, not globally) — small additional dependency (`tower-http`, `cors` feature),
or hand-roll the 4-header response if pulling `tower-http` for one concern feels heavier than
warranted; decide during implementation, not up front.

### 2.8 `main.rs`

Both services share the config load and the shutdown signal, but draw on **different** secrets
(§2.2 — `internal_apikey` for the TCP frame HMAC, `secret_phrase` reserved for the bridge's
session-token check):

```text
load AppConfig (rate_limit policy + db + internal_apikey + secret_phrase + bridge port)
spawn: TCP rate-limiter listener (existing server::run, now keyed by internal_apikey)
spawn: axum HTTP bridge listener on sse_bridge.port (keyed by internal_apikey + secret_phrase)
spawn: 15s flush loop (existing)
select on ctrl_c -> broadcast shutdown -> join all -> final flush
```

No change to the rate limiter's own shutdown/flush behavior — only one more spawned task and one
more branch in the shutdown fan-out.

## 3. Testing

- **colbin vectors**: a tiny throwaway Go program (or a `_test.go` using the real `colbin` +
  `core.UsuarioToken`-shaped struct) prints hex bytes for a handful of representative
  `UserToken` values (small ids, large ids, empty `User`, multi-byte UTF-8 `User`). Hardcode those
  hex vectors as Rust unit tests, same pattern as `auth.rs`'s existing
  `matches_the_go_client_vectors`.
- **channel token vectors**: reuse the existing values from `channel_vectors_test.go` /
  `channel.ts`'s test suite as a shared Rust test table.
- **HMAC vectors**: `computeUserTokenHash` and the service-auth header, same style.
- **Channel registry**: port the intent of `channel_test.go` (last-connection-wins, reply
  correlation, `FailAllPendingReplies` on disconnect, `AwaitChannel` timeout) as async Rust tests.
- **HTTP integration**: axum handlers exercised with `tower::ServiceExt::oneshot` or a real
  `TcpListener` bound to an ephemeral port — cover the sequence diagram in `sse_bridge/README.md`
  (`/sse` handshake → `/publish` delivers → `/rpc` blocks → `/in` unblocks it).
- Manual smoke test against a real frontend tab before deleting `sse_bridge/`.

## 4. Deployment script changes

`scripts/configure_sse_bridge.py` (663 lines) currently does three things for a **single-purpose,
possibly-remote** Go binary: compile/install the binary, write its systemd units, write its Nginx
vhost. With the merge, `server_utils` needs deployment automation for the first time (its own
`PLAN.md` §15 Phase 6 flags this as not-yet-done) — this plan folds that gap in rather than leaving
two half-finished deploy stories.

Replace it with **`scripts/configure_server_utils.py`**:

- Compiles `server_utils` with `cargo build --release` (mirrors the Go `go build` step; reuses
  `detect_unprivileged_username`/`build_unprivileged_command` so Cargo's target dir is owned by the
  right user, same reasoning as the Go module cache today).
- Installs one binary (`genix-server-utils`) + one systemd unit (merging the rate-limiter's
  documented-but-not-yet-scripted unit and the bridge's existing unit into one `Type=simple`
  service) + the restart-on-binary-change `.path` unit.
- Still writes the Nginx vhost — **only for the bridge's HTTP port** (`sse_bridge.port`), reusing
  `build_bridge_nginx_configuration` verbatim (streaming settings, HTTP/3, CORS-left-to-the-app).
  The rate limiter's TCP port stays loopback-only and gets no Nginx entry, matching
  `server_utils/README.md`'s existing guidance ("should remain on loopback or a private network").
- Drops the `apikey`-prompting flow entirely (no more `sse_bridge.apikey`). `internal_apikey` is
  a root-level secret like `secret_phrase` and `admin_password` — neither of those is prompted for
  by any deploy script today (expected to already be in `config.toml` from initial setup), so
  `configure_server_utils.py` only validates `internal_apikey` and `secret_phrase` are both
  present and fails with the key name if not, rather than adding a new interactive prompt.
- Keeps `sse_bridge.url` resolution/validation (still needed to build the Nginx vhost's
  `server_name` and to warn if it still points at the Lambda URL).

Other references to update in the same change:

- `scripts/deployer/scripts.go:58` — rename the TUI entry `configure_sse_bridge` →
  `configure_server_utils`, repoint to the new script.
- `app.sh` — same rename (even though `app.sh` is deprecated per `scripts/DEPLOYER.md`, it still
  dispatches this today).
- `AGENTS.md` — replace the `sse_bridge/README.md` / `scripts/CONFIGURE_SSE_BRIDGE.md` bullets
  with `server_utils/README.md` / a new `scripts/CONFIGURE_SERVER_UTILS.md`.
- `DEPLOYMENT.md:96-99` — update the Lambda-mode paragraph to point at `server_utils/` instead of
  `sse_bridge/`.
- `scripts/CONFIGURE_SSE_BRIDGE.md` — replaced by a new `scripts/CONFIGURE_SERVER_UTILS.md`
  documenting the merged deployment (same structure, updated for one binary/unit).

## 5. Cleanup (after verification)

- Delete `sse_bridge/` entirely (Go module + README + all tests).
- Delete `scripts/configure_sse_bridge.py` and `scripts/CONFIGURE_SSE_BRIDGE.md`.
- Update `server_utils/README.md` and `server_utils/PLAN.md` to mention the bridge is now part of
  this crate (cross-link to this file), and update the "single active process" caveat in `PLAN.md`
  §3 — it already applies per-service; now it applies to the one merged process as a whole.
- Grep for any remaining `sse_bridge` string references project-wide before calling this done.

## 6. Suggested implementation order

1. `bridge/token.rs`: colbin int/string column decode + channel-token codec, with vectors ported
   first (this is the highest-uncertainty piece — get it green before building on top of it).
2. `bridge/auth.rs`: session-token verify + service-auth header verify, with HMAC vectors.
3. `bridge/channel.rs`: registry + channel, with async unit tests mirroring `channel_test.go`.
4. `bridge/http.rs` + axum wiring in `main.rs`, with integration tests.
5. Config + `config.example.toml` + README updates.
6. Deployment script rewrite.
7. Manual smoke test, then delete `sse_bridge/` and the old deploy script/doc.

## 7. Open items to confirm during review

- **CORS**: pull in `tower-http` or hand-roll the 4 headers — leaning hand-roll to match the
  project's "no unnecessary deps" instinct, final call during implementation.
- **SSE keepalive**: use axum's built-in `KeepAlive` vs. a manual ticker + `: ping` comment frame
  — behaviorally identical to the browser either way; axum's helper is less code.
- **`user_id = -1` company-aggregate convention** in the rate limiter is unrelated to this bridge
  work and stays untouched.
