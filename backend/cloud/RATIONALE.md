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
