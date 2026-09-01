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
