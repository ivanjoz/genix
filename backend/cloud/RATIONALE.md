## The company config cache hands out a shared pointer, and its sweeper starts on the first write

**Context** — `LoadCompanyConfig` now keeps a 20-second in-memory map in front of the blob. Three
things about it were not part of the request: what a caller gets back, who starts the ticker, and
which clock the TTL reads.

**Decision** — `readCompanyConfigCache` returns the **same `*config.CompanyConfig`** every concurrent
caller holds, documented as read-only; there is no copy. The sweeper goroutine is started by a
`sync.Once` inside `writeCompanyConfigCache`, not from `main`. `CompactAndStoreCompanyConfig` caches
what it just built, so the write-through after a mutation is also the invalidation. The TTL uses
`time.Now`, not `core.Now`.

**Rationale** — Deep-copying would mean copying the private key and the certificate on every read,
which is the work the cache exists to avoid; `BuildIssuer` only reads, so the cost is a convention
someone could break — a caller that mutates the result corrupts every other holder. Starting from
`main` would put a ticker in the Lambda, in every script and in the test binaries, including
processes that never cache a company; starting it on the first write means it exists exactly where
there is something to sweep, and it is never stopped. `core.Now` is the wrong clock here because
`GENIX_HISTORICAL_UNIX` can freeze it, which would expire either nothing or everything.

**Cost** — an entry is held up to twice the TTL before its memory is released (expires at 20s, swept
by the following tick). It is never *served* in that window: the deadline is checked on the read.
The deadline is stored as unix nanoseconds, so it reads the wall clock and not the monotonic one a
`time.Time` carries: an NTP step or a VM suspend of more than 20 seconds expires the cache early,
which costs one rebuild.

## `WEBPAGE_RENDERER_URL` is validated before running the renderer locally

**Context** — `core.Env.WEBPAGE_RENDERER_URL` used to fall back to a hardcoded constant, so it was
never empty. It is now derived from `frontend.app_url`, which can be absent from config.toml.

**Decision** — `WEBPAGE_RENDERER_URL` joined the required-variable map in
`runWebpageRendererLocally`, so a missing key is reported as a config.toml problem instead of the
renderer failing on a hostless download URL.

**Rationale** — See the full entry in `backend/RATIONALE.md`.

## Backend AVIF/WebP conversion moves behind the `avif` build tag

**Context** — `github.com/ivanjoz/avif-webp-encoder` was **4,194,304 bytes of the production
Lambda — 7.7% of a 54 MB binary** — and none of it was code. Its `binaries` package is a single
`//go:embed` of a prebuilt Rust executable that `imageconv` extracts at runtime via `go-memexec`.
`imageconv` imports `binaries` unconditionally, so *any* import of the encoder package links the
whole blob, whether or not `Convert` is ever called.

The conversion it pays for is legacy. The frontend now converts images and uploads every resolution
itself: `business.SaveProductImage` takes `cloud.SaveImage` when `Content_x6` is present, and only
falls through to `cloud.SaveConvertImage` when a client sent a single unconverted image. Two files
imported the encoder — `cloud/s3.go` and `exec/image.go` — and `SaveConvertImage` had exactly one
caller.

**Decision** — the encoder is now reachable only from files tagged `avif`, and the default build
does not link the module at all (`go list -deps` confirms). Three pieces:

- `image_convert_types.go` (untagged) declares `ImageConvertInput` and `Image` locally, mirroring
  imageconv's structs field-for-field. They cross the conversion-Lambda boundary as JSON with
  default field-name marshalling, so the names and order are the wire format.
- `image_convert_avif.go` (`//go:build avif`) holds the real `SaveConvertImage`, the only
  `imageconv` import in the repo, and `USE_MULTILAMBDA`.
- `image_convert_disabled.go` (`//go:build !avif`) returns a descriptive error naming the missing
  tag. `exec/image_avif.go` and `exec/image_disabled.go` split the `compress-image` handler the
  same way, since `ExecHandlers` registers it unconditionally.

Result: 49,152,160 → 44,957,856 bytes. The `-tags avif` build measures 49,152,160 — byte-identical
to before the split, which is the control proving the refactor changed nothing but linkage.

**Rationale** — failing loudly beat the alternatives. Keeping the multilambda dispatch in the
default build would have worked (the main binary needs only the DTOs, not the encoder — only the
`_2` Lambda runs it), but it means `deploy.sh` building and shipping two different binaries, and
the fallback is dead anyway. Deleting the path outright would have saved the same bytes but thrown
away a working feature with no way back short of a revert. An error that names the build tag costs
nothing, and if the frontend ever regresses the upload fails visibly instead of silently writing no
CDN object.

The local DTOs are a real duplication and the cost of this decision: if upstream reorders a field,
nothing here breaks at compile time in the default build. That is mitigated in the tagged build,
where `convertImage` converts between the two with a direct struct conversion
(`imageconv.ImageConvertInput(input)`) — which only compiles while the mirrors are exact, so
`go build -tags avif` is the regression test for the wire format.

**Note for deploys.** With the default build, `SaveConvertImage` errors before reaching the
`_2` Lambda, so that Lambda is never invoked and need not be deployed. To restore backend
conversion, build with `-tags avif`; `deploy.sh` was not changed.
