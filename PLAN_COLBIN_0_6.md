# Plan: genix on colbin v0.6.2 and the current genix-orm / genix-ui

## Why

genix-orm `main` (pushed from berryapps, `a3ec5a3`) requires **colbin v0.6.2**. genix and fareward
are still on **v0.3.x**, so genix can't take genix-orm `main`, and with it the text-search work
rebased there and the shared `utils` module. Between the two versions colbin changed:

- **Its wire format**, in three steps that old readers can't follow:
  - 0.4: narrow tables key their columns at eight bits;
  - 0.5: a map of structs carries a struct index;
  - 0.6: a narrow nibble sizes every field.

  Nothing written by 0.3 is guaranteed to read under 0.6, in Go or in Rust.
- **The Go API:** `Codec.Append` and `Codec.Encode` return `([]byte, error)`, and `FieldIDs`
  reports `uint16` ids.

genix-orm `main` also changes three things genix uses (berryapps commit `c7cb040e`):
- integers and `[]byte` stored by scylla (CSV values and `KeyConcatenated` keys);
- base62 is removed;
- one sortable base64, `utils.Base64` (`-0-9A-Z_a-z`, unpadded), for everything genix-orm and
  genix-ui encode themselves.

**Data:** genix has no production data. Everything stored is **wiped**, not migrated: the scylla
keyspace, the genix-s3 text indexes, sessions and browser caches.

## What was verified (2026-10-17)

genix's backend was built against genix-orm `main` and colbin v0.6.2 using a temporary modfile and
build overlay, without touching the repository. `go build ./...` and `go vet ./...` (test code
included) pass once these three things are fixed:

1. `fareward/go`: the `Append` calls that now return an error (section 2).
2. `backend/genix-orm/search/genixs3/backend.go`: the uncommitted change in the submodule
   (`ids` → `[]genix.Ref` for genix-s3's `Apply`).
3. `backend/db/db.go:233-234`: the `orm.EncodeToBase62` / `orm.DecodeFromBase62` aliases, removed
   from genix-orm.

Not verified: fareward's Rust build on colbin v0.6.2, genix's frontend on genix-ui `main`, and the
runtime of any of it.

## Current state of the submodules in genix

- `backend/genix-orm`:
  - It is checked out on the **`backup/text-search-before-rebase`** branch, at `e6f0b23`: the
    5 text-search commits before the rebase, plus the uncommitted `search/genixs3/backend.go`.
  - Its local `main` is genix-orm `main`, `a3ec5a3`, which holds those commits rebased:
    `80058cb`, `9f016eb`, `c44c734`, `76ff614`, and the dynamo text search rewritten as `a3ec5a3`.
  - genix's superproject still records `e6f0b23`.
- `frontend/packages/genix-ui`: `c62ac0b`, 25 commits behind `main` (`f7525df`).
- `fareward`: `8746d7c` (`fareward:v11`), colbin **v0.3.0** (Go, vectors and Rust).

## Phases

### 1. genix-orm on `main`

1. In `backend/genix-orm`: `git stash` the `search/genixs3/backend.go` change, then
   `git checkout main` and `git stash pop`.
2. Commit that change on `main` and push it. It adapts `genixs3` to genix-s3's current `Apply`, the
   `[]genix.Ref` version.
3. Delete `backup/text-search-before-rebase` once genix builds on `main`.
4. In `backend/go.mod`:
   - add `replace github.com/ivanjoz/genix-orm/utils => ./genix-orm/utils`;
   - require `github.com/ivanjoz/genix-orm/utils v0.0.0`;
   - move `github.com/ivanjoz/colbin` to `v0.6.2`.

   `go mod tidy` fails in genix because of the protobuf fork (`thirdparty/protobuf`), so edit with
   `go mod edit -require=...` and build with `GOFLAGS=-mod=mod` once to refresh `go.sum`.

### 2. fareward on colbin v0.6.2 (Go, Rust, vectors)

The six colbin request frames change bytes, so **the backend's client and the daemon must ship
together**.

1. **Go client** (`fareward/go`):
   - `go.mod`: colbin `v0.6.2`.
   - Handle the error from each `Append` call:
     - `access_invalidation.go:65`
     - `budgets.go:91`
     - `credits.go:405`
     - `locks.go:216`, `locks.go:284`
     - `request_log.go:101`

     **Decided: the error goes up.** A function that returns `[]byte` alone today returns
     `([]byte, error)`, and every caller up to the request handler returns that error.
2. **Vectors** (`fareward/vectors`): colbin `v0.6.2`; regenerate the vectors the Rust tests pin,
   for example the frames in `src/service/auth.rs:116`.
3. **Rust** (`fareward/Cargo.toml`):
   - `colbin = { git = ..., tag = "v0.6.2", features = ["derive"] }`.
   - Rebuild and fix whatever the crate's API changed. It is untested here: the crate's version
     history is in colbin's `INTERNALS.md` / `README.md`.
   - The colbin users are:
     - `src/lock/protocol.rs`, `src/service/protocol.rs`, `src/limiter/protocol.rs`,
       `src/reqlog/protocol.rs`;
     - `src/limiter/budget.rs`, `src/limiter/access.rs`;
     - `src/bridge/auth.rs`, `src/bridge/token.rs`. The browser session token is a colbin message
       of `core.UsuarioToken`, and its test vectors were printed by `colbin.Marshal` from Go.
       Regenerate them.
     - `src/service/auth.rs`.
4. **Protocol version:** the frame bytes change, so bump `DOMAIN` `fareward:v11` → `fareward:v12`:
   - `src/service/auth.rs:38`, and the Go client's copy of it;
   - `PROTOCOL_SHAPES.md`, `README.md`.

   A mismatched backend and daemon then fail the handshake instead of misreading frames.
5. Run `cargo test`, the Go tests, and the vectors on both sides. Cut a fareward release, then point
   genix's `fareward` submodule at it.

### 3. genix backend

1. `db/db.go`: delete the `EncodeToBase62` / `DecodeFromBase62` aliases.
2. `core/helpers.go:1564-1590`: genix's own base62 (`EncodeToBase62`, `DecodeFromBase62` and the
   key builder that uses it). Move the key builder to `utils.EncodeUintBase64` /
   `utils.DecodeUintBase64` and delete the base62 copy.
3. **Base64 for what genix-ui decodes**, all through `utils.Base64`:
   - `security/login.go:160-161`: `AccesosComputed` / `AccesosSubComputed` (today
     `core.BytesToBase64(..., true)`, std base64 with `- _ ~`).
   - `core/cache.go:153, 177`: `parseConcatenatedUint16s` / `parseConcatenatedInts`, used by
     `cc-gh`, `cc-upc`, `ids`, `cc-ids`, `cc-ver` (today `base64.RawURLEncoding`).
   - After that, `core.BytesToBase64` / `MakeB64UrlEncode` / `MakeB64UrlDecode` keep only callers
     whose other side is external. Move the internal ones to `utils.Base64` and delete what's left
     unused.
4. Stay on standard base64 where the other side is external: JWTs, data URLs, the S3/Lambda body,
   certificate uploads (`invoicing/company_secrets_api.go`).
5. `go build ./...`, `go vet ./...`, `go test ./...`.

### 4. genix-search / genix-s3

`genix-search/genix-s3/go.mod` requires colbin **v0.3.1**. genix's binary already links it at
v0.6.2 (minimal version selection), and it builds. But the index files it writes to S3 change
format.

1. Move `genix-s3` to colbin `v0.6.2` in its own repository and run its tests.
2. Wipe the genix-s3 index bucket, then rebuild every text index with `ReindexTextSearchAll` on
   each scylla table that declares `TextSearch` (today `business/types/image_assets.go`).

### 5. genix-ui to `main` (25 commits), with the ScyllaDB driver on the new cache protocols

These commits change **contracts with the backend**, not only components:

- `7593690` **cache-by-ids**: the request is now berryapps' year-tick `cc` parameter (dynamo's
  `DecodeCachedIDsRequest`). genix's backend reads `cc-ids` / `cc-ver`, and genix-ui `main` no
  longer sends them.
- `c217fe8` **delta cache**: sync from an S3 change log by byte-offset cursor and record versions,
  which is dynamo's delta index.
- `70fcf9c` caches keyed by millisecond `upd`, then SUnixTime `upd` (berryapps).
- `f7525df` the sortable base64 (phase 3 covers the backend side).
- `975e136` `HTMLEditor` is tiptap (RoosterJS removed); plus prop changes to `Button` (`mode`,
  `height`, `type="button"`), the labels and `Layer`.

**Decided:** genix-orm's **scylla** driver serves the same two cache protocols as dynamo, so one
genix-ui serves both apps. The protocols are file based, not DynamoDB based: their state lives in
the ORM's `utils.FileStore` (S3), so the scylla driver reuses them rather than reimplementing them.

What scylla has today, and what replaces it:

| Today in scylla (genix) | Replaced by (dynamo's, shared) |
|---|---|
| `SaveUpdatedVersion`: a `cache_updated_version` table with an int16 version per (partition, slot); the `cc-ids` / `cc-ver` request; `QueryCachedIDs(refSlice, []IDUpdatedVersion)` | `CacheByIDs`: the 768-byte `ids-cache/<cycle year>/<base pk>` year-tick slot files; the `cc` request (`DecodeCachedIDsRequest`); `QueryCachedIDs(request)` |
| Delta views (`index_delta_view.go`): rows keyed by a per-partition `updated_version` sequence; the client sends its last `upv` as `?up=` | `TypeDelta`: the 10-minute change-log files `delta-log/<base pk>/<index tag>/<pinned values>/<bucket>`, read past a byte cursor `?up=<bucket>.<bytes>...`; the response ends with the `Metadata` (`mdt`) records |

Steps:

1. **Shared package.** Move what is driver independent out of `dynamo/` into one genix-orm module,
   for example `genix-orm/cache` (it depends only on `utils`):
   - the slot files: the year cycle, the ticks, the read/raise of a file with `If-Match`;
   - `DecodeCachedIDsRequest` and the "which IDs to read" plan;
   - the delta log: `delta_log_codec.go`, the bucket/seal/replace rules, the appends, the cursor
     reading and the Metadata records, with `testdata/delta_log_vectors.json`.

   dynamo keeps only its side: which keys to read (BatchGetItem) and its write hooks. Its tests have
   to pass unchanged. berryapps must not notice the move.
2. **Record fields in genix.** Both protocols need, per record:
   - `Updated int32 json:"upd"`, a **SUnixTime** stamped by the ORM on every write. genix's records
     already have `Updated int32`: confirm its unit and that the scylla ORM stamps it;
   - a managed `Version int16 json:"ver"` (the delta log's entry version, dynamo's `ver`). Scylla
     has no per-record version today: add it as a column the write path raises.
   - `Metadata *string json:"mdt,omitempty"` on every table with a delta index.

   Remove `UpdatedVersion` (`upv`), the `updated_version` sequence and its counters.
3. **Scylla write path.** After the records of a write land (insert, update, merge, delete),
   raise their slot files and append to their delta logs, through the shared package. A failed
   append fails the write, so the caller retries it, as in dynamo.
4. **Scylla read path:**
   - `QueryCachedIDs(request)`: read the slot files, then select the changed IDs.
   - `.Delta(cursor, statuses...)`: list the view on a first sync or a replace, otherwise read the
     logs and select the named records.

   Both read through the existing select builder.
5. **Schema.** Translate each of the ~20 `DeltaView` declarations into a `TypeDelta` index and each
   `SaveUpdatedVersion` (6 tables) into `CacheByIDs`. Then delete `index_delta_view.go`,
   `cache_updated_version*.go`, the `cache_updated_version` table and the `upv` plumbing.
6. **genix backend:**
   - install the `FileStore` at boot (berryapps' `core/filestore` is the model; genix already has an
     S3 client for genix-s3);
   - by-IDs routes decode `cc` and call `QueryCachedIDs(request)`;
   - delta routes take the cursor from `req` and return the records as the ORM gives them (the
     `Metadata` records included);
   - delete the `cc-ids` / `cc-ver` parsing in `core/cache.go`. `cc-gh` / `cc-upc` (group cache)
     stay, on `utils.Base64` (phase 3).
7. **Tests:** the shared package's vectors, a ScyllaDB integration test of both protocols
   (write → by-IDs read → delta read past a cursor → replace), and berryapps' dynamo tests.

Then `bun run check`, and a pass over the screens that use `HTMLEditor` and `Button`.

### 6. Wipe and start clean

Nothing is migrated:

1. Drop and recreate the scylla keyspace (tables recreated by the ORM deploy). That includes
   fareward's sequences table.
2. Empty the genix-s3 index bucket (phase 4).
3. Existing session tokens are colbin 0.3 messages, and their access blobs use the old alphabet:
   everybody logs in again. Clear the browsers' localStorage and IndexedDB caches on the dev
   machines.
4. Recreate the bootstrap admin, seed data and companies with genix's usual scripts.

### 7. Verification

1. `go build`, `go vet` and `go test` on the backend, plus the tests in every genix-orm module.
2. fareward: `cargo test`, the Go tests and the vectors, with the backend and daemon both at
   `fareward:v12`.
3. Run genix locally:
   - log in: fareward decodes the colbin session token, and the access blob opens in the browser;
   - load a cached list through both protocols: a by-IDs read with `cc`, and a delta sync past a
     `?up=` cursor;
   - run a text search on `image_assets` after `ReindexTextSearchAll`;
   - check the credit/budget/lock frames against the daemon.
4. `bun run check` in the frontend.

## Order

Phases 1 → 2 → 3 → 4 are needed to build and run genix on colbin v0.6.2. Phase 5 comes after
them. Its step 1 (the shared package) is a genix-orm change that berryapps also takes, so it ships
and is verified in berryapps first. Phase 6 runs once, after phase 5: the backend, the daemon and
genix-s3 are deployed together on a wiped keyspace.

## Decisions

1. fareward's `Append` errors are passed up to the caller (phase 2.1).
2. The scylla driver serves dynamo's cache protocols, through a package both drivers share
   (phase 5).
