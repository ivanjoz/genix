# Sub-Accesses — Implementation Status

Companion to `docs/SUB_ACCESSES_PLAN.md`, which holds the design and the rationale. This file is the
handoff: what is finished, what is not, and what has to happen before any of it can deploy.

**Nothing is committed.** Everything below is in the working tree of the parent repo and of the
`fareward` submodule.

---

## Verification, as of the last run

| Check | Command | Result |
| --- | --- | --- |
| Backend build | `cd backend && go build ./...` | clean |
| Backend vet | `cd backend && go vet ./...` | clean |
| Backend tests | `cd backend && go test ./...` | 21 packages ok; **1 pre-existing failure**, see below |
| Table schema | `cd scripts && go run . check_tables` | 53 struct pairs, pass |
| Daemon + client | `cd fareward && cargo test` | **187 pass, 0 fail** |
| Go client | `cd fareward/go && go test ./...` | pass |
| Frontend types | `cd frontend && bun run check` | 10 errors, **all pre-existing**, see below |
| Catalog parser | `cd frontend && bun test routes/security/users-profiles/` | 6 pass |

**Pre-existing failures, not caused by this work and not fixed by it:**

- `app/agent/ragdocs` — `TestParseExamplesAndBuildStableChunks` fails on stale evidence for
  `frontend/core/modules.ts` in `frontend/routes/finance/cash-banks/DOCUMENTATION.md`. Confirmed by
  stashing this work and re-running: it fails identically on the untouched tree.
- `bun run check` — 10 errors across `SaleOrdersTable.svelte`,
  `warehouse-movements.svelte.ts`, `PurchaseOrderReport.svelte`, `ProductSupplyManagement.svelte`
  and `ONNXInference.svelte`. None of those files were touched here.

---

## DONE

### Step 1 — catalog migrated to TOML

`backend/access_list.yml` → `backend/access.toml`, sections renamed `access_list` → `access` and
`access_groups` → `groups`. The file was generated programmatically from the YAML and asserted
equal to it field by field (9 groups, 36 access entries) before the old one was deleted.

- `core/usuario-accesos.go` — `AccessListYaml` → `AccessCatalog`, `yaml` tags → `toml`,
  `gopkg.in/yaml.v3` → `github.com/pelletier/go-toml/v2` (**already a direct dependency**, no new
  one added).
- `main-handlers.go` — `//go:embed access.toml`.
- `routes/security/users-profiles/access-list-catalog.ts` — parser rewritten for TOML. It gets
  shorter: no indentation tracking, no `- ` prefix handling, records self-delimit on `[[name]]`, and
  single-line TOML arrays parse as JSON. Multi-line arrays and trailing commas are rejected with a
  message naming the problem rather than surfacing a `SyntaxError`.
- Every reference to the old filename updated across Go, Svelte, `RATIONALE.md` and
  `DOCUMENTATION.md` files.

### Step 2 — sub-accesses in the catalog and on the profile

- `core/usuario-accesos.go` parses `sub_accesses_ids` / `sub_accesses_names` into
  `AccessInfo.SubAccesosMask`. Load **refuses** — it does not repair — on a length mismatch between
  the two parallel arrays, an id outside `2..13`, a duplicate id, or a blank name. The catalog is
  embedded at build time, so a malformed declaration is a build-time mistake and must surface as
  one. New `AccessHelper.GetAccesoInfo(id)`.
- `security/types/perfiles.go` — new column `SubAccesos db.ColSlice[*ProfileTable, int32]`, holding
  `accesoID*100 + subID`. Readable on purpose: the profile is what a human edits.
- `security/perfiles.go::validateProfileSubAccesos` — rejects a sub-access the catalog never
  declared, one on an access that does not exist, and one granted without its parent access.
- `backend/access.toml` — acceso 10 (Punto de Venta) declares `[2, 3]` =
  `["Recibir Pago", "Despachar Producto"]`. **It is the only access with sub-accesses so far.**

### Step 3 — the two grant blobs, in Go and Rust

New `backend/core/accesos-blob.go` is the **only** writer of these bytes in Go:
`EncodeAccesosGrants`, `DecodeAccesosGrants`, `DecodeSubAccesoBytes`, `HasSubAcceso`,
`MakeAccesoNivelPacked` / `UnpackAccesoNivel`. The encoder sorts rather than requiring sorted input,
and refuses out-of-range ids, invalid levels, duplicates and over-ceiling masks.

- `core/types/users.go` — `AccesosComputed` is now `[]byte`; `AccesosSubComputed` added. Both are
  CQL `blob`, which `[]uint16` already was, so **no `ALTER TABLE` for the first column**.
- `security/usuarios.go` — `buildAccesosComputedFromPerfiles` merges in **two passes** over the
  profiles: every access first, then every sub-access. Order matters — a sub-access granted by one
  profile may qualify an access granted by another, and a single interleaved pass would drop those.
  Deleted the now-dead local `makeAccesoNivelUint16` / `makeAccesoNivelPacked`.
- `security/perfiles.go` — change detection compares **both** columns. Comparing only the first
  would skip the write for an edit that adds nothing but a sub-access, leaving users stale.
- `security/login.go` — sends both blobs; `buildBootstrapAdminAccesos` gives user 1 "Todos" on every
  access that declares sub-accesses.
- **Rust** `limiter/access.rs` — `UserAccessState` holds both blobs as `Box<[u8]>` verbatim; the
  `Vec<u16>` allocation, sort and dedup per cache load are gone. `verdict()` returns
  `Result<AccessVerdict, AccessDenial>`. `accesos_computed` keeps a real binary search on its fixed
  stride; `accesos_sub_computed` scans linearly with an early exit. Both are validated on load
  (ascending ids, whole words, terminated sub runs, no empty mask).
- **Rust** `limiter/storage.rs` — reads the second column; the doc comment claiming the blob is
  little-endian u16s is rewritten.

### Step 5 — the `:v9` reply frame

`[correlation:u16][status:u8][detail:u16][extra_len:u8][extra…]`.

- Locks, budgets and unauthorized charges keep their exact `status`/`detail` meaning and pay one
  byte of `extra_len = 0`. Locks still carry their generation in `detail` — which is why `detail`
  is a `u16` at all.
- `detail` on an authorized charge: bits 0..2 the code, bits 3..6 the granted-slot mask, bits 7..10
  the has-subs mask.
- The tail is the sub bytes **copied verbatim** out of the cached blob. The daemon re-encodes
  nothing and still knows nothing about what a sub-access means.
- `service/auth.rs` — `DOMAIN` bumped `fareward:v8` → `fareward:v9`. **SipHash-2-4 is unchanged**;
  only the domain-separator string moved, which is exactly what makes a mixed pair fail at the first
  frame instead of misparsing the new layout. All pinned vectors regenerated from the Go client and
  updated on both sides, plus the TCP integration harnesses that hardcoded `:v8` and a 5-byte reply.
- `go/connection.go` — reads the tail **before dispatching** (an unread tail desynchronizes every
  later reply), and refuses one over 8 bytes.
- `go/credits.go` — new `AccessGrant{GrantedSlots, SubAccesoBytes}`. `decodeAccessGrant` refuses
  every way the masks and the tail can disagree rather than half-reading them.
  `ChargeAPIUsage` / `ChargeAPIAccessOnly` now return `(*AccessGrant, error)`.

### The handler surface

- `core.UsuarioToken.SubAccesos map[int32]uint16`, tagged **`cb:"-"`** so it never rides in the
  browser-held session token.
- `main-handlers.go::mapGrantedSubAccesos` translates the daemon's slot indexes back to access ids
  on the side that embeds the catalog, and sets them on the **per-request** `args.User` — never the
  `IS_SERVERLESS` global, which is single-threaded-only by design.
- Handlers call `req.User.HasSubAcceso(accesoID, subAccesoID)`.

---

## PENDING

### 1. The recompute script — DEPLOY BLOCKING

Every stored `accesos_computed` is still little-endian `id<<2|nivel-1`. Under the new reader those
bytes decode to nonsense and **every user would be denied**. Nothing can ship until this runs.

Needs `scripts/recompute_user_accesos.go` following `scripts/SCRIPTS.md`: walk every company's
users, rebuild both columns through `buildAccesosComputedFromPerfiles`, write them back, and fire
one `InvalidateUserAccess(companyID, InvalidateAllCompanyUsers)` per company.

`users.accesos_sub_computed` and `profiles.sub_accesos` are new columns and need the DDL applied
(`check_tables` validates the structs; it does not create columns).

### 2. Frontend (step 6)

- `genix-ui` submodule, `security/accesos.ts` — `decodeStoredAccesosComputed` must return
  `Uint8Array`; add the second blob, the big-endian scan and `hasSubAcceso`. `create-security.ts`
  holds `Uint16Array` today and its `accesoResultCache` keys on a bare number, which must become
  `(accesoID, nivel, subID)`. **Submodule change — commit and push inside it.**
- `users-profiles.svelte.ts` — build `subAccesosMap`, send `SubAccesos` as `accesoID*100 + subID`,
  bump `useCache.ver`.
- `AccessCard.svelte` / `ProfilesTab.svelte` — sub-access chips, with "Todos" as a synthetic first
  option that clears the rest.
- `UserProfilesAccessSelector.svelte` — reflect sub-accesses in the read-only user view.

### 3. Tests not yet written

- `core.DecodeSubAccesoBytes` — the seam where the daemon's opacity ends. Has no test of its own.
- `mapGrantedSubAccesos` — slot-to-access-id translation, including the user-1 and unmapped-GET
  paths that produce no grant at all.
- `validateProfileSubAccesos` — the three rejection cases.
- An end-to-end check through the `agent-browser` skill, against a route with sub-accesses.

### 4. RATIONALE entries

Required by `CLAUDE.md` before commit — the plan's decisions have not been written into any
`RATIONALE.md` yet:

- `backend/RATIONALE.md` — the TOML catalog and parallel arrays; the two-column split and why the
  column name carries the "has sub-accesses" bit; big-endian; what replaced the defensive re-sort.
- `fareward/RATIONALE.md` — the `:v9` length-prefixed tail, and why the daemon copies sub bytes
  rather than re-encoding them.
- `frontend/routes/security/users-profiles/RATIONALE.md` — the editing model and the
  `accesoID*100 + subID` wire shape.
- `frontend/packages/genix-ui` — the `Uint16Array` → `Uint8Array` change and the cache-key change.

---

## Deploy order

1. Apply the DDL for `users.accesos_sub_computed` and `profiles.sub_accesos`.
2. Deploy backend **and** daemon together. The `:v9` domain makes a mixed pair fail at the first
   frame — loud, and intended, but it means the window is real.
3. Run `recompute_user_accesos`.
4. Deploy the frontend. Bump the stored-accesos storage key or its wrap checksum so a stale
   `Uint16Array` payload is discarded rather than misread.

---

## Known risks

1. **The two-array lookup.** An access lives in exactly one column, so any check that misses
   `accesos_computed` must also scan `accesos_sub_computed`. Missing that second lookup fails closed
   — a user is denied something they hold — but it has to be right in Go, Rust and TS. Rust has a
   test for it (`an_access_is_found_in_either_blob`); TS does not exist yet.
2. **Three hand-written parsers.** Go and Rust agree and are cross-pinned by a fixture captured from
   the Go encoder (`decodes_the_blobs_the_go_encoder_writes`). TS is the outstanding third.
   A concrete near-miss while writing these: a hand-written fixture for sub-accesses 1+13 expected
   `0x40` when the correct byte is `0x20` — sub 13 is bit 12, which lands at bit **5** of the second
   byte, not bit 6. Nothing about `0x40` looks wrong, and a reader making the same slip would grant
   sub-access 14 where 13 was meant.
3. **Scope limit, by design.** A handler can read sub-accesses only for the accesses that gated
   **its own route** — that is all the reply carries. Reading an unrelated access's sub-flags would
   need a different transport.
4. **Unrelated working-tree changes.** The `fareward` submodule and the backend carry pre-existing
   modifications that are **not part of this work** and should not be attributed to it: the SipHash
   migration (`src/siphash.rs`, `go/siphash/`, `vectors/`, `src/bridge/*`, `Cargo.*`, the walkthrough
   docs), the session-token move to BLAKE2s-128 (`usrToken:v3`), `backend/agent/bridge.go`, and
   `frontend/.../AccessCard.svelte`.
