# Genix in the browser — backend compiled to WASM over IndexedDB

Goal: ship the Go backend as `GOOS=js GOARCH=wasm`, run it in a Web Worker next to the SvelteKit
frontend, and back the ORM with IndexedDB through the same `db.Executor` contract ScyllaDB uses.
No module code (`sales`, `logistics`, …) changes: the table declarations and queries stay as they are.

Status: **phase 1 done** (2026-10-31). Phases 2–4 are plans.

## Phase 1 result (measured)

`GOOS=js GOARCH=wasm go build .` builds with no overlay, and `go list -deps` under `js` holds no
AWS, gocql, scylla, dynamo or genixs3 package (guarded by `backend/wasm_deps_test.go`).

| Build | Raw | brotli -q11 | gzip -9 |
|---|---|---|---|
| Baseline (overlay, everything linked) | 90.7 MB | 8.9 MB | 14.2 MB |
| Phase 1 | 29.5 MB | 4.0 MB | 5.8 MB |

The phase 1 number is a **floor**: the idb skeleton's `CompileTable` panics, so the linker drops the
generic ORM code for every table (`genix-orm/db` code fell from 5.5 to 1.2 MB, `app/*` from 5 to
2.5 MB). Phase 2 brings part of it back. Part of the drop may also come from a concurrent change
that removed Qdrant/gRPC/msgpack from `backend/go.mod`; the baseline predates it.

---

## Baseline (measured 2026-10-31, before any change)

`GOOS=js GOARCH=wasm go build .` on `backend/` fails on two spots only:

- `libs/servermetrics/metrics_collector.go:143` — `syscall.Statfs` does not exist under `js`.
- `core/json_defaults.go:11` — `sonic.SetOmitEmptyDefault` is not in sonic's non-amd64 fallback.

With both stubbed through `-overlay` (repo untouched) the whole binary links:

| Build | Raw | brotli -q11 | gzip -9 |
|---|---|---|---|
| default | 90.7 MB | 8.9 MB | 14.2 MB |
| `-ldflags="-s -w"` | 85.5 MB | — | — |

Code section is 47 MB, data section 38 MB. Code attributed by function name:

| Package | Code |
|---|---|
| `github.com/aws/*` | 11.5 MB (S3 6.3, sts/signin/sso/ssooidc/config 2.5, dynamodb 0.6, lambda 0.5) |
| `genix-orm/scylla` + `gocql` | 4.4 + 1.1 MB |
| `genix-orm/db` (generic instantiations per table) | 5.5 MB — stays |
| `app/*` modules | ~5 MB — stays |

Expected after phase 1: roughly −30 MB raw (a third of the file). This is an estimate that scales the
data section with the code; phase 1 replaces it with a measured number.

---

## Phase 1 — Build-tag reorganization: compile without AWS, `genix-orm/scylla` and `gocql`

**Deliverable:** `GOOS=js GOARCH=wasm go build` succeeds, and its dependency list contains no
`github.com/aws/*`, no `github.com/gocql/gocql`, no `genix-orm/scylla`. The native build (Lambda +
local server) behaves exactly as today.

No IndexedDB code yet. Under `js`, the database is a skeleton driver whose methods return an error;
phase 2 fills it in.

### Convention

The repo already splits files this way for the AVIF encoder (`cloud/image_convert_avif.go` /
`cloud/image_convert_disabled.go`). Phase 1 uses the same pattern:

- `foo.go` gets `//go:build !js` — the server implementation, unchanged.
- `foo_js.go` gets `//go:build js` — the browser implementation. In phase 1 it either does the local
  equivalent or **fails loudly** with a descriptive error (as `image_convert_disabled.go` does).
- Both files expose the same exported surface, so callers in the modules compile unchanged.

The tag is the GOOS-derived `js`, not a custom tag: the WASM build needs no extra flags, and a native
build can never pick up the browser files by accident.

### 1.1 The database driver choice — `backend/db/`

| File | Change |
|---|---|
| `db/driver.go` | Renamed to `db/driver_scylla.go`, `//go:build !js`. The driver-neutral `Schema`/`Record` constraints moved to `db.go`. |
| `db/driver_js.go` (new) | `TableStruct` alias pointing at `idb.TableStruct` (skeleton below). |
| `db/autoincrement.go` | Imports scylla + fareward to install `scylla.ReserveCounterRange`/`SetCounterValue`. Rename to `db/autoincrement_scylla.go`, `//go:build !js`. `BypassFarewardSequences` goes with it: its only caller is `exec/init.go`, also excluded under `js`. |
| `db/data_frames.go` | Installs `scylla.AcquireFrameLock`. Merged into `db/driver_scylla.go`. No `js` counterpart: its only caller, `core.ConfigureFileStore`, is server-only. |

### 1.2 Skeleton driver — `genix-orm/idb/` (new package, git submodule)

- `idb/executor.go`: `type Exec[TableT, RecordT] struct{}` implementing every `db.Executor` method by
  returning `errNotImplemented`, and `type TableStruct[...] = db.TableStruct[Exec[...], TableT, RecordT]`
  — the exact shape of `scylla/executor.go:97`, so swapping the alias in `driver_js.go` is a one-line change.
- `CompileTable` panics with `errNotImplemented`: the contract has no error return, and a nil
  `db.Table` would only fail later, far from the cause. Nothing in the browser build runs yet, so
  phase 1 never reaches it; phase 2 replaces it with the real compile.
- `init()` assigns the `db` function variables the scylla driver assigns (`GetAutoincrementID`,
  `QueryCachedGenericByIDs`, `SetDebugLogging`) to stubs. `SetCodec` waits for phase 2.
- Lives in the root `genix-orm` module (`github.com/ivanjoz/genix-orm/idb`), next to `scylla`. Not
  committed or pushed in the submodule yet.
- No dependency besides `genix-orm/db`, `dataframe`, `utils`, `xunsafe` — no `syscall/js` yet.

Note: `genix-orm/dynamo` is **not** a `db.Executor` implementation — it is a separate ORM with its own
`dynamo.Model`. `backend/db/driver.go`'s comment ("`dynamo.TableStruct -> DynamoDB`") is stale. `idb`
is the first second implementation of the contract; dynamo is only a reference for its planner.

### 1.3 Entry point — `backend/main*.go`

`main.go` imports scylla, aws-lambda-go and `net/http` serving. `main-handlers.go` holds the router
and access checks, which the WASM build needs, but it also imports `app/exec`.

| File | Change |
|---|---|
| `main.go` | `//go:build !js`. Unchanged otherwise. |
| `main-handlers.go` | Stays shared. Remove `exec.ModuleHandlers` from `appHandlersModules` and move `ExecFuncHandler` (uses `exec.ExecHandlers`) out. |
| `main_exec.go` (new, `!js`) | Appends `exec.ModuleHandlers` (backups) to the router in an `init()`, holds `ExecFuncHandler`. |
| `main_js.go` (new, `js`) | WASM entry: builds the handler map and blocks with `select {}`. Exposing `genixHandle` to JS is phase 3; phase 1 only needs it to link. |

`app/exec` is then imported only by `!js` files, which drops `exec/*`'s scylla, gocql
(`exec/column_rename.go`) and `cloud.Init` usage from the WASM build in one move.

### 1.4 `core/` — the AWS entry points

| File | AWS use | Change |
|---|---|---|
| `core/security.go:800` | `GetAwsConfig()` → `aws-sdk-go-v2/config` (pulls sts/sso/signin) | Moved to `core/aws_config.go`, `!js`. The `aws` TOML section struct stays in `security.go` (plain data). |
| `core/responses.go` | `aws-lambda-go/events` in `MainResponse` (`LambdaResponse`, `LambdaStreamingResponse`) and in `MakeResponseFinal`, `MakeStreamingResponseFinal`, `MakeErrStreamingFinal` | The router returned Lambda shapes from `prepareResponse`. Per D5, `MainResponse` is removed and the `Make*Final` helpers, plus the new `MakeLambdaResponse` / `MakeLambdaStreamingResponse`, live in `core/responses_lambda.go`, `!js` (used by `main.go` and the agent). |
| `core/file_store.go` | `ConfigureFileStore` builds the S3 `filestore` | `!js`, no browser counterpart (only server entry points call it). The browser file store comes in phase 2. |
| `core/filestore/store.go` | S3 implementation of `utils.FileStore` | Imported only by `core/file_store.go`, so it drops out with no tag. |
| `core/text_search.go` | `genixs3` + `s3.NewFromConfig` | `!js`, no browser counterpart (only server entry points call it). |
| `core/json_defaults.go` | `sonic.SetOmitEmptyDefault` (fork-only, amd64) | `!js`. See decision D3 for the `js` side. |

### 1.5 `cloud/` — imported by 25 module files

What modules use from `cloud`: `SaveFile`, `GetFile`, `ListFiles`, `DeleteFile`, `SaveImage`,
`SaveFileArgs`, `ImageArgs`, `InvokeWebpageRenderer` + its types, `IsDataMirrorEnabled`, `Insert`,
`Select`, `GetByID`, `Init`, the company config blob functions, and the image conversion types.

| File | Change |
|---|---|
| `cloud/s3.go` | Split: `SaveFileArgs`, `FileInfo`, `ImageArgs` and `SaveImage` (now just `SaveFile` with the provider's default bucket) moved to `cloud/files.go` (shared). The S3/R2 functions stay in `s3.go`, `!js`. `cloud/files_js.go`: `SaveFile`/`GetFile`/`ListFiles`/`DeleteFile` fail loudly. See D2. |
| `cloud/webpage_renderer.go` | `!js`. `_js.go`: types shared via a small `webpage_renderer_types.go`, `InvokeWebpageRenderer` fails loudly. |
| `cloud/orm-dynamodb.go` | `!js`. |
| `cloud/orm-core.go` | `getProviderORM` names `NewDynamoORM`. Move the provider switch to `cloud/orm_provider.go` (`!js`); `cloud/orm_provider_js.go` returns "no cloud data mirror in local mode", and `IsDataMirrorEnabled` returns false under `js`. |
| `cloud/orm-sqlite.go` | `!js` (D4). |
| `cloud/lambda.go` | `ExecLambda` is used only by `image_convert_avif.go` (already behind `avif`). `!js`. |
| `cloud/cloudwatch.go` | `QueryLogsParsed`, `QueryLogsCloudWatch`, `GetDBMetrics` have **no caller**. `!js` (D1). |
| `cloud/company_config_blob.go` | No AWS itself; goes through `SaveFile`/`GetFile`, so it works once `files_js.go` does. |

### 1.6 `libs/servermetrics`

`metrics_collector.go` uses `syscall.Statfs`. Its only consumers are `config/system_metrics_sse.go`
and `config/system_memory_packages.go`. Move `collectDiskMetrics` to `disk_metrics.go` (`!js`) and
`disk_metrics_js.go` returning an error, which the collector already records as a warning. Server
metrics have no meaning in a tab, so phase 4 may drop those two routes under `js` instead.

### 1.7 Guard rails — keep it from regressing

`backend/wasm_deps_test.go` (`TestBrowserBuildLinksNoServerStack`, part of `go test .`) runs
`GOOS=js GOARCH=wasm go list -deps .` and fails on any AWS, gocql, scylla, dynamo or genixs3 package.
Size is measured by hand for now:

```bash
GOOS=js GOARCH=wasm go build -o /tmp/genix.wasm . && brotli -c -q 11 /tmp/genix.wasm | wc -c
```

### 1.8 Phase 1 acceptance

- [x] `GOOS=js GOARCH=wasm go build .` succeeds with no `-overlay`.
- [x] `go list -deps` under `js` has no AWS, gocql or scylla package (`wasm_deps_test.go`).
- [x] Native `go build ./...` and `go vet ./...` clean; `go test` passes for `app`, `core`, `db`,
      `cloud`; the `agent` module (which uses the Lambda helpers) builds.
- [ ] Lambda and local server boot and serve as before (smoke: login + one list route) — not run:
      a dev boot writes usage logs to the configured database.
- [x] Measured size table (raw / brotli / gzip) replaces the estimate above.

---

## Phase 2 — The `idb` driver (outline)

- **JS bridge with bulk operations** (`putMany`, `getMany`, `queryRange`, `deleteMany`, `counterAdd`):
  every `Executor` call is one bridge call and one IndexedDB transaction. An IndexedDB transaction
  auto-commits once the event loop has no pending request, so Go must never await between two
  requests of the same transaction.
- **Storage model:** one object store per table, value `{k: [...pk, ...ck], i1..iN: [...], d: Uint8Array}`
  — `d` is the colbin record (one `CopyBytesToJS`, no per-field `syscall/js`). Native compound array
  keys, `multiEntry` indexes for slice columns.
- **Planner:** pick the best index for the key range, filter the rest after decode (dynamo's
  `query.go`/`compile.go` as reference). Views become indexes, not copies.
- **Schema deploy:** schema hash → `version+1` → create stores/indexes in `onupgradeneeded`.
- **Order:** CRUD + Select → delta (`upd`) → `QueryCachedIDs` → counters → `SelectGrouped` → text
  search → DataFrames (OPFS candidate).
- **File store** behind `utils.FileStore` (IndexedDB or OPFS), which also backs `cloud/files_js.go`.
- **Tests:** the ORM's tests run under `GOOS=js` in Node (`go_js_wasm_exec` + `fake-indexeddb`).

## Phase 3 — Worker host and transport (outline)

- `main_js.go` exposes `genixHandle(request) → Promise<Uint8Array>`; each call runs in a goroutine.
- A dedicated Web Worker loads `genix.wasm`.
- `packages/genix-ui/http/index.ts:253` (the single `fetch`) routes to the worker in local mode;
  responses travel as transferable colbin buffers.

## Phase 4 — Local-mode behavior (outline, depends on the open questions below)

Auth, autoincrement without fareward, multi-tab, the frontend's own IndexedDB caches, and which
modules are disabled.

---

## Phase 1 decisions

- **D1 — tagged.** `cloud/cloudwatch.go` (no caller anywhere) is kept behind `!js`.
- **D2 — done (phase 2).** `cloud.ObjectStore` (`cloud/files.go`) is the public object store:
  `s3ObjectStore` (one S3 client per process, no longer one per call) and `r2ObjectStore` in
  `cloud/s3.go`, picked from providers.cdn on first use; `indexedDBObjectStore` in
  `cloud/files_js.go`, installed by the browser entry point with `cloud.SetObjectStore`. It stays
  separate from the ORM's `utils.FileStore` (no content type, cache headers or buckets there); in the
  browser both are IndexedDB object stores holding Blobs.
- **D3 — colbin only** in the browser build. `core/json_defaults.go` is `!js`.
- **D4 — excluded.** `cloud/orm-sqlite.go` is `!js`; the browser build has no cloud data mirror.
- **D5 — done.** `core.MainResponse` is gone. `mainHandler`/`prepareResponse` return the finished
  `*core.HandlerResponse` (nil when already written to the local server's ResponseWriter), and
  `main.go` renders it with `core.MakeLambdaResponse` / `core.MakeLambdaStreamingResponse`
  (`core/responses_lambda.go`, `!js`).
- `db.BypassFarewardSequences` has a single caller, `exec/init.go:129`, so it lives in
  `autoincrement_scylla.go` with no `js` counterpart.
- `core.ConfigureFileStore` / `ConfigureTextSearch` are called only from server entry points
  (`main.go`, `exec/init.go`), so they are `!js` with no browser counterpart, and `db.SetDataFrameLock`
  exists only in `driver_scylla.go`.

## Open questions for later phases

1. Standalone/demo (data never leaves the browser) or offline-first with sync to the server?
2. Multiple tabs: `SharedWorker` (one backend, weaker Safari support) or dedicated worker + Web Locks?
3. The frontend's delta cache and `cache-by-ids` in local mode: disable, or keep for one code path?
4. Which modules may be unavailable locally (invoicing/SUNAT, images, agent, webpage render)?
5. Auth in local mode: fixed local user and company, or the real login?
