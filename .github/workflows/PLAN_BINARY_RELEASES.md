# Plan: public Linux binary releases for Genix

Status: implemented on 2026-08-15; the first pushed `v*` tag and public Release are still pending.

## Decision

Add a dedicated `.github/workflows/release-binaries.yml` workflow. It will build and test:

- `backend/` for Linux `amd64` and `arm64` with Go and `CGO_ENABLED=0`.
- `server_utils/` for static-musl Linux `amd64` and `arm64` with Rust.
- A `SHA256SUMS` manifest covering every published binary.

The workflow will publish the four raw executables and the checksum manifest as assets of a
GitHub Release created from a `v*` tag. GitHub Releases, rather than Actions artifacts or the
Pages site, are the public distribution layer. Releases are designed to package binary files for
download, and their URLs can be versioned or point to the latest release. Actions artifacts remain
useful only for moving files between workflow jobs because they have retention and access
semantics intended for workflow runs.

Recommended Rust strategy: compile each architecture natively in a two-runner matrix using
`ubuntu-24.04` and `ubuntu-24.04-arm`. The repository is public, so both standard runners are free,
have four vCPUs, and can run concurrently. This avoids QEMU and Docker, allows tests to execute on
both architectures, and has the lowest release wall time. The existing musl targets and
`server_utils/.cargo/config.toml` continue to use `rust-lld`, so both results remain static.

## Current repository findings

- Before this implementation, `.github/workflows/ci-deploy.yml` was the only workflow; it builds
  the frontend and deploys GitHub Pages. Binary releases now remain isolated in
  `.github/workflows/release-binaries.yml` rather than sharing its `pages`, `id-token`, and
  frontend path filters.
- The repository is public and currently has no tags or GitHub Releases, so `v0.1.0` can establish
  the first release convention without conflicting with an existing scheme.
- `backend/go.mod` pins Go `1.26.0` with its `toolchain` directive and locally replaces two modules
  from the `backend/genix-orm` submodule.
- The backend dependency graph has no active Cgo files with `CGO_ENABLED=0`. Both direct
  cross-builds therefore work without a C cross-toolchain.
- `server_utils/Cargo.toml` requires Rust `1.95`, has a committed `Cargo.lock`, and already
  configures `rust-lld` for `x86_64-unknown-linux-musl` and
  `aarch64-unknown-linux-musl`.
- Local validation successfully produced all four outputs. `file` reported the Go outputs as
  statically linked and the Rust outputs as statically/static-PIE linked. The observed stripped
  sizes were approximately 59 MB/49 MB for the Go amd64/arm64 binaries and 7.8 MB/7.1 MB for the
  Rust amd64/arm64 binaries.
- `cargo test --locked --release --target x86_64-unknown-linux-musl` currently passes all 128
  `server_utils` unit and integration tests. The AArch64 suite still needs its first run on the
  native Actions runner.

### Resolved backend test prerequisites

`CGO_ENABLED=0 go test ./...` builds and runs without an external database and is now green. The
implementation resolved the two failures found during planning:

- `app/agent/ragdocs`: `frontend/routes/finance/cash-banks/DOCUMENTATION.md` contains stale source
  evidence for `backend/finance/cash_bank_movement.go`.
- `app/libs/text-search`: `TestLetterPairCodesAreStable` expects older codes for `('a', '$')` and
  `('b', 'o')`.

Neither package is excluded or marked as an allowed failure; `go test ./...` remains a mandatory
release gate.

## Public artifact contract

Keep the asset names stable between releases so both immutable version URLs and moving “latest”
URLs work. The names also match the prebuilt-file names already recognized by
`configure_server.py` and `configure_server_utils.py`.

| Component | Architecture | Release asset |
| --- | --- | --- |
| Backend | amd64 | `genix_app_linux_amd64` |
| Backend | arm64 | `genix_app_linux_arm64` |
| Server utils | amd64 | `genix-server-utils_linux_amd64` |
| Server utils | arm64 | `genix-server-utils_linux_arm64` |
| Integrity manifest | all | `SHA256SUMS` |

Example immutable URL:

```text
# A version URL must always return the bytes published for that tag.
https://github.com/ivanjoz/genix/releases/download/v0.1.0/genix_app_linux_amd64
```

Example moving URL:

```text
# The stable filename makes this URL follow the latest non-prerelease release.
https://github.com/ivanjoz/genix/releases/latest/download/genix_app_linux_amd64
```

Raw executables are preferable here because the existing deployment scripts can consume them
directly and set their installed mode themselves. A browser or `curl` download does not preserve
the executable bit, so user documentation must show `chmod +x` or `install -m 0755`.

## Workflow triggers and cache scope

Use one workflow with two trusted triggers:

1. A manual `workflow_dispatch` run builds/tests both release shapes but never publishes. Run it
   against `main` when validation or default-branch cache warming is useful.
2. A pushed tag matching `v*` builds the exact tagged commit and publishes a Release only after
   every required architecture succeeds. Ordinary commits do not start this workflow.

Default-branch warming is important. GitHub cache entries are scoped to a branch or tag: a cache
written by tag `v0.1.0` cannot be restored by `v0.1.1`, while a tag workflow can restore a cache
from the default branch. An occasional manual run on `main` therefore makes the next changed tag
fast without paying for every commit. GitHub currently evicts an unaccessed cache after seven days,
so a cold build remains possible and the workflow must never depend on a cache being present.

Before starting Rust runners for a tag, compare `server_utils/` and the binary-release workflow
against the nearest reachable previous `v*` tag. If both are unchanged and that tag's public
Release contains both server-utils assets, download, revalidate, and stage those immutable files
for the new Release. Otherwise, compile both architectures. Including the workflow in the diff is
conservative but necessary: a compiler, target, linker, or build-command change must invalidate
reuse even when the Rust source itself did not change.

Use a release-specific concurrency group such as `binaries-${{ github.ref }}` with
`cancel-in-progress: false`. Two workflows must never race to publish the same tag.

## Job design

### 1. `build-backend`

Run once on `ubuntu-24.04`, not as an architecture matrix. Go cross-compilation is inexpensive,
and one job lets both targets reuse the downloaded module cache and the same Go build cache.

1. Check out the release commit and initialize `backend/genix-orm`. Initializing every frontend
   submodule is unnecessary for this job.
2. Use `actions/setup-go` with `go-version-file: backend/go.mod`, `check-latest: false`, and its
   built-in cache.
3. Include `backend/go.sum`, `backend/genix-orm/go.sum`, and
   `backend/genix-orm/db/go.sum` in `cache-dependency-path`.
4. Run backend tests once on the amd64 runner with `CGO_ENABLED=0`.
5. Build both target architectures with `-trimpath` and the existing stripped linker flags.
6. Inspect each output with `file` and `readelf`; reject a wrong ELF machine or a dynamic
   interpreter.
7. Upload the two files as a one-day workflow artifact only on tag runs.

The implementation should use commands equivalent to:

```bash
# Build metadata identifies the immutable source tag and commit without using wall-clock time.
release_identity="${GITHUB_REF_NAME}-${GITHUB_SHA:0:12}"

# The backend is pure Go in this build, so the same compiler safely emits both Linux targets.
for target_architecture in amd64 arm64; do
  CGO_ENABLED=0 GOOS=linux GOARCH="$target_architecture" \
    go build -trimpath \
    -ldflags="-s -w -X app/core.BuildDate=$release_identity" \
    -o "../dist/genix_app_linux_${target_architecture}" .
done
```

The tag/commit build identity is preferable to the current local deployment timestamp for a
release: rerunning the same tag should use the same metadata, and the Release already records its
publication time.

### 2. `build-server-utils`

Use an explicit matrix rather than `cross` or QEMU:

| Runner | Rust target | Asset architecture |
| --- | --- | --- |
| `ubuntu-24.04` | `x86_64-unknown-linux-musl` | `amd64` |
| `ubuntu-24.04-arm` | `aarch64-unknown-linux-musl` | `arm64` |

For each matrix entry:

1. Check out only the repository content needed by `server_utils`; it has no submodule dependency.
2. Install the pinned Rust `1.95.0` toolchain with the minimal profile and only the current matrix
   target. Add a root or `server_utils/rust-toolchain.toml` containing the version/profile so local
   and CI release builds agree, but do not list both targets there because that would download an
   unused standard library in every matrix job.
3. Restore `Swatinem/rust-cache` for `server_utils -> target`. Make the target triple an explicit
   key component and save caches only for manual runs on `main`. Pin this third-party action to a
   reviewed full commit SHA in the implemented workflow.
4. Run `cargo test --locked --release --target "$RUST_TARGET"`. Because each job is native, both
   architectures can execute their tests without emulation.
5. Run `cargo build --locked --release --target "$RUST_TARGET"` and copy the executable to its
   stable asset name.
6. Inspect the ELF machine and assert that no `INTERP` program header exists.
7. Upload the file as a one-day workflow artifact only on tag runs.

The release build command remains deliberately simple:

```bash
# Cargo.lock and the pinned compiler make dependency selection repeatable.
cargo build --locked --release --target "$RUST_TARGET"
```

`Swatinem/rust-cache` is a better starting point than a hand-written whole-directory cache: it
keys on rustc and the Cargo manifests/lock/config, retains dependency artifacts, removes stale and
incremental output, and does not reuse the workspace crate incorrectly after source changes.
Target-specific dependencies must be compiled once for each architecture; no safe cache can turn
an x86_64 object into an AArch64 object. On later runs, each architecture restores its own compiled
dependency graph and normally recompiles only changed crates and the Genix crate.

When the planner proves that the Rust build inputs are unchanged, `reuse-server-utils` replaces the
matrix entirely: it downloads both files from the previous public Release, repeats the ELF machine
and static-link checks, and uploads one short-lived workflow artifact. This avoids compiling even
the workspace crate and avoids allocating the two Rust runners.

Do not introduce `sccache` or `cargo-chef` initially. `sccache` adds another service and is worth
measuring only if the normal Rust cache still misses frequently; `cargo-chef` mainly solves Docker
layer caching. Do not use `cross` here: its container image and Docker startup are useful for crates
with native target libraries, but this crate already builds both musl targets with the installed
Rust standard libraries and `rust-lld`.

### 3. `publish-release`

This job depends on the backend build plus either a successful Rust matrix or successful immutable
asset reuse, and runs only for a pushed `refs/tags/v*` ref.

1. Grant build jobs only `contents: read`. Grant this job `contents: write`; no PAT or repository
   secret is required because GitHub CLI can use the job’s `GITHUB_TOKEN` as `GH_TOKEN`.
2. Download all short-lived workflow artifacts into one `dist/` directory; a fresh Rust build
   contributes two artifacts, while the reuse path contributes one containing both files.
3. Assert that the directory contains exactly the four expected nonempty ELF files. Restore
   executable modes before inspection because Actions artifact transport does not preserve them.
4. Generate `SHA256SUMS` after the final filenames are in place.
5. Create the Release with the pre-existing tag, generated notes, and all assets in one
   `gh release create` command using `--verify-tag`.

Equivalent publishing command:

```bash
# Record bare asset names so verification works after users download them into any directory.
(
  cd dist
  sha256sum \
    genix_app_linux_amd64 \
    genix_app_linux_arm64 \
    genix-server-utils_linux_amd64 \
    genix-server-utils_linux_arm64 \
    > SHA256SUMS
)

# GitHub CLI creates a draft internally, uploads all assets, then publishes the release.
gh release create "$GITHUB_REF_NAME" dist/* \
  --verify-tag \
  --generate-notes \
  --title "Genix $GITHUB_REF_NAME"
```

Do not use `gh release upload --clobber`. Published assets should be immutable: if code or build
logic changes, create a new tag. If the repository enables GitHub’s immutable Releases setting,
tags and assets cannot be altered after publication. If an upload fails and leaves a draft,
inspect that draft and remove it deliberately before rerunning; never silently replace a public
binary.

## Cache policy summary

| Cache | Scope/key inputs | What it saves | Policy |
| --- | --- | --- | --- |
| Go setup cache | Runner, exact Go toolchain, all relevant `go.sum` files | Module downloads and Go build objects | Warm with a manual `main` run; tags restore it |
| Rust amd64 | Rust 1.95.0 host, musl target, Cargo lock/manifests/config | Registry downloads and dependency target artifacts | Separate from arm64; save on manual `main` runs |
| Rust arm64 | Rust 1.95.0 host, musl target, Cargo lock/manifests/config | Registry downloads and dependency target artifacts | Separate from amd64; save on manual `main` runs |
| Previous Release assets | Nearest reachable `v*` tag with unchanged Rust build inputs | Final server-utils amd64/arm64 files | Revalidate and republish; skip both Rust runners |
| Workflow artifacts | The current tag run only | Final binaries moving between jobs | One-day retention; never advertised publicly |

The build must be correct after a total cache miss. Cache hits are an optimization, never an input
to correctness or publication.

## Consumer documentation

After the workflow exists, update `backend/README.md`, `server_utils/README.md`,
`scripts/CONFIGURE_SERVER.md`, and `scripts/CONFIGURE_SERVER_UTILS.md` with both the immutable and
latest URLs. A minimal documented download should verify the checksum before installation:

```bash
# Download one public release asset and the manifest; no GitHub account or token is required.
curl --fail --location --remote-name \
  https://github.com/ivanjoz/genix/releases/latest/download/genix_app_linux_amd64
curl --fail --location --remote-name \
  https://github.com/ivanjoz/genix/releases/latest/download/SHA256SUMS

# Verify only the selected binary, then install it with an executable mode.
grep ' genix_app_linux_amd64$' SHA256SUMS | sha256sum --check --strict
sudo install -m 0755 genix_app_linux_amd64 /usr/local/bin/genix/genix_app
```

Automatic downloading inside the configure scripts should be a separate follow-up. It changes a
server from “use a local prebuilt file” to “trust and fetch a network release,” so it needs an
explicit version/latest choice, checksum enforcement, timeouts, and clear offline behavior.

## Implementation sequence

1. Add the Rust toolchain pin and `.github/workflows/release-binaries.yml`.
2. Manually run the workflow on `main` to populate default-branch caches and confirm all native
   tests and static-link checks without publishing.
3. Create and push the first annotated SemVer tag, recommended `v0.1.0` for this pre-alpha project.
4. Confirm the Release exposes all five assets, checksums match after a fresh download, and both
   `releases/download/v0.1.0/...` and `releases/latest/download/...` return HTTP 200 after redirects.
5. Update the four consumer/deployment documents with the tested URLs and commands.
6. Optionally enable immutable Releases and add GitHub build-provenance attestations after the
   basic release path is stable. Attestations require `id-token: write` and `attestations: write`;
   they complement rather than replace `SHA256SUMS`.

## Acceptance criteria

- A normal commit or push starts no binary-release work.
- A manual run builds/tests both components and creates no Release; a run on `main` warms caches.
- Pushing a `v*` tag creates exactly one public GitHub Release after every required build or reuse
  check succeeds.
- The Release contains the four stable binary names plus `SHA256SUMS` and generated notes.
- `file` and `readelf` verify the expected machine and absence of a dynamic loader for all four
  outputs.
- Rust tests run natively on both x86_64 and AArch64 without QEMU or Docker.
- A tag with unchanged `server_utils/` and unchanged release build logic reuses the preceding
  Release's two verified Rust assets and does not start either Rust matrix runner.
- A second manual `main` run records Go and Rust cache hits and does not rebuild the Rust dependency
  graph.
- Fresh unauthenticated `curl --fail --location` requests can download every versioned and latest
  asset, and `sha256sum --check` succeeds.
- Build jobs cannot write repository contents; only the final tag-only publish job has
  `contents: write`.

## Sources

- [GitHub: Linking to releases](https://docs.github.com/en/repositories/releasing-projects-on-github/linking-to-releases)
- [GitHub: Workflow artifacts](https://docs.github.com/en/actions/concepts/workflows-and-actions/workflow-artifacts)
- [GitHub: Workflow trigger and path-filter syntax](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax)
- [GitHub: Dependency caching and cache scope](https://docs.github.com/en/actions/reference/workflows-and-actions/dependency-caching)
- [GitHub: Hosted runner architectures and labels](https://docs.github.com/en/actions/reference/runners/github-hosted-runners)
- [GitHub CLI: `gh release create`](https://cli.github.com/manual/gh_release_create)
- [GitHub CLI: `gh release download`](https://cli.github.com/manual/gh_release_download)
- [GitHub: Authenticate with `GITHUB_TOKEN`](https://docs.github.com/en/actions/tutorials/authenticate-with-github_token)
- [Go setup action: version selection and built-in caching](https://github.com/actions/setup-go)
- [Rust: `aarch64-unknown-linux-musl` platform support](https://doc.rust-lang.org/rustc/platform-support/aarch64-unknown-linux-musl.html)
- [Cargo: Build-cache layout](https://doc.rust-lang.org/cargo/reference/build-cache.html)
- [`Swatinem/rust-cache`: cache behavior and key inputs](https://github.com/Swatinem/rust-cache)
- [`cross`: supported targets and container-based tradeoffs](https://github.com/cross-rs/cross)
