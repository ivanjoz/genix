## Installing fareward retires the `genix-server-utils` units

**Context** — The rename gave the daemon new unit names and the installer simply started writing
them. On a host configured before the rename the old unit stayed enabled and running, so the new
`fareward.service` could never bind 14013 — 530 restarts on the production VPS — while the August
binary kept the port and answered the backend it no longer speaks the protocol of (`:v8`, no
sequence opcode). The backend saw a connection closed with no reply, which only the sequence
allocator reports, because locks and charges tolerate an unreachable daemon.

**Decision** — `remove_legacy_units` stops, disables and deletes `genix-server-utils.service` and
its two helper units before the new units are written, and counts as a systemd change so the
reload happens. The superseded binary is left on disk. The failure table also matches `Address in
use`, the daemon's own wording, next to the `Address already in use` that systemd prints.

**Rationale** — Only the installer touches units, so only it can retire one; leaving the removal to
the operator is what produced a port conflict that nothing diagnosed. It is not a compatibility
shim — nothing reads the old unit, it is deleted — and the diagnosis bug is the reason to do both at
once: the installer had the perfect explanation for this failure and printed "no known cause",
because it was matching a phrase this daemon never writes.

## The deploy TUI keeps a separate env-only action

**Context** — `cloud accion=1` now pushes the Lambda `CONFIG` as well as the code, which makes the
TUI's action 13 (`updateLambdaEnvironmentVariables`, AWS CLI) look redundant.

**Decision** — Action 13 stays, and it grew the same comment-stripping and 4 KB environment check
as `cloud` — copied, not shared, because `scripts` and `cloud` are separate Go modules.

**Rationale** — Pushing configuration without recompiling and re-uploading a binary is worth an
action of its own, and it is the only path that also updates the renderer Lambda. Copying twenty
lines beats a module dependency between two deploy tools; the two are named as each other's twin in
comments so a change to one is findable from the other.

## The grant-column change is a data rebuild, not a compatibility shim

**Context** — `users.accesos_computed` kept its grant word but changed container and byte order:
little-endian `[]uint16` became big-endian `[]byte`. Read the new way, every stored blob decodes into
accesses nobody granted, so every user is denied. The reader could have sniffed the old shape — a
`[]uint16` blob and a `[]byte` blob are both just bytes — and translated on the fly.

**Decision** — No shim. `recompute_user_accesos` (`fn-recompute-user-accesos`, backed by
`security.RecomputeUserAccesos`) rebuilds both grant columns for every user of every company from
their profiles and direct grants, and invalidates the daemon's cache per company. It runs after
`fn-homologate` and after the backend/daemon deploy; see `RECOMPUTE_USER_ACCESOS.md`.

**Rationale** — Pre-alpha, so there is nothing to stay compatible with, and a sniffing reader is
exactly the wrong thing to build here: it would have to guess, in three separate processes, and a
wrong guess authorizes rather than fails. The rebuild is also cheap and correct by construction — the
profiles are the source of truth, so it recomputes rather than converts, which makes it idempotent
and reusable as a repair tool for any user whose blobs drift later.

It lives in `security` rather than in `scripts/` because the merge it has to reproduce
(`buildAccesosComputedFromPerfiles`, highest nivel per access, union of sub-masks, then the direct
`AccessLevelIDs`) is unexported there, and a second copy of that merge in a script is precisely the
thing that would silently disagree with `PostUsuarios`. The cost is the same one `fn-emit-cpe`
already pays: the entry point is registered in `exec/main.go` and dispatched through
`runBackendExec`, so it is a backend function with a script name rather than a script.

## The renderer zip URL comes from `frontend.app_url`

**Context** — This module carried its own hardcoded copy of the webpage-renderer artifact URL,
duplicated across three Go modules that cannot import each other.

**Decision** — The literal is gone; the URL is `frontend.webpage_renderer_url`, or
`<frontend.app_url>/webpage-renderer.zip` when that key is empty.

**Rationale** — See the full entry in `backend/RATIONALE.md`.

## The deployer resolves one release per component, not one release for all of them

**Context** — `configure.py` had a single `LATEST_RELEASE_URL` pointing at `ivanjoz/genix` and one
`SHA256SUMS` covering all four assets, because one workflow published all of them. `fareward`
now releases from its own repository, so its binary and its manifest live somewhere else.

**Decision** — `GENIX_RELEASE_URL` and `FAREWARD_RELEASE_URL`, and
`download_selected_binaries` groups the requested assets by publisher: each group downloads that
release's `SHA256SUMS` and verifies only its own assets against it. Both releases name their
manifest `SHA256SUMS`, so `download_latest_release_file` takes a `local_name` and they land in
`tmp/` as `SHA256SUMS.genix` and `SHA256SUMS.fareward`.

**Rationale** — The grouping is what makes the verification honest: a flat asset list with one
manifest would have to either skip entries it cannot find or verify a binary against checksums
that never covered it. Writing both manifests to the same `tmp/SHA256SUMS` was the concrete bug
this avoids — the second download would overwrite the first, and whichever asset was checked
afterwards would be compared against the wrong release. `test_each_component_is_verified_against_
its_own_release_manifest` pins it by giving each manifest only its own asset, so a regression to a
shared manifest fails rather than silently passing.
