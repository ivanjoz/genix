# Sub-Accesses — Implementation Status

Companion to `docs/SUB_ACCESSES_PLAN.md`, which holds the design and the rationale. This file is the
handoff: what is finished, what is not, and what has to happen before any of it can deploy.

Steps 1–3 and 5 landed in `3d04ec13 feat(security)!: sub-accesses per access, and a TOML access
catalog`, together with the matching `fareward` submodule pointer. Everything since — the recompute
script, the frontend, the missing tests and the RATIONALE entries — is **uncommitted**, in the
working tree of the parent repo and of `frontend/packages/genix-ui`.

---

## Verification, as of the last run

| Check | Command | Result |
| --- | --- | --- |
| Backend build | `cd backend && go build ./...` | clean |
| Backend vet | `cd backend && go vet ./...` | clean |
| Backend tests | `cd backend && go test ./...` | all pass; **1 pre-existing failure**, see below |
| Module boundaries | `cd scripts && go run . check_module_imports` | 47 packages, no violations |
| Table schema | `cd scripts && go run . check_tables` | 53 struct pairs, pass |
| Daemon + client | `cd fareward && cargo test` | **187 pass, 0 fail** |
| Go client | `cd fareward/go && go test ./...` | pass |
| Frontend types | `cd frontend && bun run check` | 10 errors, **all pre-existing**, see below |
| Frontend build | `cd frontend && bun run build` | clean |
| Frontend tests | `bun test frontend/packages/genix-ui/security/ frontend/routes/security/users-profiles/` | 40 pass |
| Route documentation | `cd backend && go run ./agent/cmd/documentation-index -mode validate -document frontend/routes/security/users-profiles/DOCUMENTATION.md` | valid, 17 sections / 22 chunks |

**Pre-existing failures, not caused by this work and not fixed by it:**

- `app/agent/ragdocs` — `TestParseExamplesAndBuildStableChunks` fails on stale evidence for
  `frontend/core/modules.ts` in `frontend/routes/finance/cash-banks/DOCUMENTATION.md`. The test
  only checks `finance/cash-banks` and `logistics/purchase-orders`; neither is touched here.
- `bun run check` — 10 errors across `SaleOrdersTable.svelte`, `warehouse-movements.svelte.ts`,
  `PurchaseOrderReport.svelte`, `ProductSupplyManagement.svelte` and `ONNXInference.svelte`. None
  of those files were touched here.

---

## DONE

### Step 1 — catalog migrated to TOML

`backend/access_list.yml` → `backend/access.toml`, sections renamed `access_list` → `access` and
`access_groups` → `groups`. Generated programmatically from the YAML and asserted equal to it field
by field (9 groups, 36 access entries) before the old one was deleted.

`core/usuario-accesos.go` swapped `gopkg.in/yaml.v3` for `github.com/pelletier/go-toml/v2` (already
a direct dependency). `routes/security/users-profiles/access-list-catalog.ts` was rewritten and got
shorter: `[[access]]` self-delimits, so no indentation tracking and no `- ` prefixes. Multi-line
arrays and trailing commas are rejected by name.

### Step 2 — sub-accesses in the catalog and on the profile

- `core/usuario-accesos.go` parses `sub_accesses_ids` / `sub_accesses_names` into
  `AccessInfo.SubAccesosMask`. Load **refuses** on a length mismatch, an id outside `2..13`, a
  duplicate, or a blank name — the catalog is embedded at build time, so a malformed declaration is
  a build-time mistake. New `AccessHelper.GetAccesoInfo(id)`.
- `security/types/perfiles.go` — `SubAccesos db.ColSlice[*ProfileTable, int32]`, holding
  `accesoID*100 + subID`.
- `security/perfiles.go::validateProfileSubAccesos` — rejects an undeclared sub-access, one on an
  access that does not exist, and one granted without its parent access.
- `backend/access.toml` — acceso 10 (Punto de Venta) declares `[2, 3]` =
  `["Recibir Pago", "Despachar Producto"]`. **Still the only access with sub-accesses.**

### Step 3 — the two grant blobs, in Go and Rust

`backend/core/accesos-blob.go` is the **only** writer of these bytes in Go. `core/types/users.go`
made `AccesosComputed` a `[]byte` and added `AccesosSubComputed`; both are CQL `blob`, which
`[]uint16` already was. `security/usuarios.go::buildAccesosComputedFromPerfiles` merges in two
passes; `security/perfiles.go` change-detects on both columns; `security/login.go` sends both and
synthesizes "Todos" for user 1. Rust `limiter/access.rs` holds both blobs verbatim as `Box<[u8]>`
and validates them on load.

### Step 5 — the `:v9` reply frame

`[correlation:u16][status:u8][detail:u16][extra_len:u8][extra…]`, one shape for every opcode.
`DOMAIN` bumped `fareward:v8` → `fareward:v9` with SipHash-2-4 unchanged, so a mixed pair fails at
the first frame. `go/connection.go` reads the tail before dispatching; `go/credits.go` decodes
`AccessGrant{GrantedSlots, SubAccesoBytes}`.

### The handler surface

`core.UsuarioToken.SubAccesos map[int32]uint16`, tagged `cb:"-"`.
`main-handlers.go::mapGrantedSubAccesos` translates slot indexes back to access ids and sets them on
the **per-request** `args.User`. Handlers call `req.User.HasSubAcceso(accesoID, subAccesoID)`.

### Step 4 — the recompute script

- `backend/security/recompute_accesos.go` — `RecomputeUserAccesos`, registered as
  `fn-recompute-user-accesos` in `exec/main.go` (the `invoicing.EmitHandler` precedent: the merge it
  reproduces is unexported in `security`, and a second copy in a script is what would silently
  disagree with `PostUsuarios`).
- Dispatched as `cd scripts && go run . recompute_user_accesos`, and listed in the `./deploy.sh` TUI
  under **Base de Datos → Recomputar Accesos de Usuarios**.
- Walks every active company's users — **including inactive ones** — merges profiles *and* direct
  `AccessLevelIDs`, writes only the rows whose bytes changed naming only the two grant columns, and
  fires one `InvalidateUserAccess(companyID, InvalidateAllCompanyUsers)` per company that had
  writes. Idempotent: it recomputes from the profiles, so a second run reports `0 reescritos`.
- `scripts/RECOMPUTE_USER_ACCESOS.md` documents it. `exec/recompute_accesos_test.go` guards the
  registration.

### Step 6 — frontend

**`genix-ui` submodule** (commit and push inside it):

- `utilities/parsers.ts` — `base64ToUInt16` **replaced** by `base64ToBytes`; the export in
  `utilities/index.ts` follows.
- `security/accesos.ts` — rewritten as the third reader of the byte format:
  `decodeStoredAccesosComputed` returns `Uint8Array`, plus `makeAccesoNivelPacked`,
  `unpackAccesoNivel`, `findAccesoNivel` (binary search, fixed stride), `findAccesoSubGrant`
  (linear walk with early exit), `hasAcceso` (**the two-payload lookup**), `hasSubAcceso` (the
  "Todos" expansion) and `validateAccesosBlobs`.
- `security/create-security.ts` — holds both payloads, stores them under `AccesosV2` and
  `AccesosSub`, validates on load and discards a bad pair whole, keys `accesoResultCache` on
  `(accesoID, nivel, subAccesoID)`, and exposes `checkSubAcceso`. `isTokenValid` accepts either
  payload being non-empty.
- `security/types.ts` / `index.ts` / `SECURITY.md` updated.
- `form/Checkbox.svelte` and `form/CheckboxOptions.svelte` — box and label are now **one**
  `<button role="checkbox">`, so clicking the text toggles. `Checkbox` gained a controlled mode
  (`checked` + `onToggle`) for a value that is not a property of an object.

**App:**

- `core/types/common.ts` — `IProfile.SubAccesos` + `subAccesosMap`; `ILoginResult.AccesosSubComputed`.
- `routes/security/users-profiles/users-profiles.ts` — **new**, pure: `buildSubAccesoOptions`,
  `toggleSubAcceso` ("Todos" exclusive both ways), `packSubAccesos` (drops a sub-access whose parent
  access was cleared), `unpackSubAccesos`. Unit-tested in `users-profiles.test.ts`.
- `users-profiles.svelte.ts` — builds `subAccesosMap`, strips it before the POST, `useCache.ver` 2 → 3.
- `access-list-catalog.ts` — `sub_accesses_ids` / `sub_accesses_names` on the catalog entry type.
- `ProfilesTab.svelte` — `IAccess.subAccesos` from the catalog; packs `form.SubAccesos` on save,
  after `Accesos`.
- `AccessCard.svelte` — a checkbox row **under** the card, inside the same grid cell, shown only
  when the access declares sub-accesses **and** is granted.
- `UserProfilesAccessSelector.svelte` — a third row on the profile chip, `<access>: <sub-access>`.

### Tests written

- `core/accesos-blob_test.go::TestDecodeSubAccesoBytes` — the seam where the daemon's opacity ends,
  including the empty tail as a legitimate answer and the sub-13-is-bit-5-of-byte-2 near-miss.
- `sub_accesos_gate_test.go` (package `main`) — `mapGrantedSubAccesos`: slot→id translation, a
  granted access with no sub bytes (present with an empty mask), the no-grant paths (unmapped GET /
  `selfServiceRoutes` / user 1), and unreadable sub bytes keeping the entry.
- `security/perfiles_test.go` — `validateProfileSubAccesos` against the **real** `access.toml`,
  read off disk because the catalog is embedded into package `main`.
- `exec/recompute_accesos_test.go` — the deploy-blocking command is registered.
- `genix-ui/security/accesos.test.ts` — 23 cases: big-endian, the two-payload lookup, bit positions,
  the "Todos" expansion, and every rejection `validateAccesosBlobs` makes.
- `users-profiles.test.ts` — 17 cases over the editing rules.

### RATIONALE and documentation

- `backend/RATIONALE.md` — four entries: the two-column big-endian format (and what replaced the
  defensive re-sort), the TOML catalog and parallel arrays, the readable profile column, the gate's
  slot translation. Includes the **deferred scope**: sub-accesses are a profile concept and a direct
  `AccessLevelIDs` grant carries none.
- `scripts/RATIONALE.md` — why the format change is a data rebuild rather than a compatibility shim,
  and why the script lives in `security`.
- `fareward/RATIONALE.md` — the `:v9` length-prefixed tail and why the daemon copies sub bytes.
- `frontend/packages/genix-ui/RATIONALE.md` + `form/RATIONALE.md` — the `Uint8Array` change, the
  three-input cache key, the `AccesosV2` key bump, and the one-control checkbox.
- `frontend/routes/security/users-profiles/RATIONALE.md` — the editing model and the wire shape.
- `.../users-profiles/DOCUMENTATION.md` — sub-accesses documented across concepts, the grant
  capability, rules and troubleshooting; every source hash reviewed and refreshed, and
  `users-profiles.ts` added as evidence. Validator passes. **Not indexed** — indexing writes to
  Qdrant and was not requested.

---

## PENDING

### 1. The end-to-end `agent-browser` check — now unblocked, still not run

The migration was taken on the dev database (`150.136.42.240`, keyspace `genix`) on 2026-09-03.
Steps 1 and 3 of the deploy order below are **done**:

- `fn-homologate` added `users.accesos_sub_computed` and `profiles.sub_accesos`. Adding
  `sub_accesos` also rebuilt `profiles__pk_updated_view` and `profiles__pk_status_updated_rng_view`,
  which is a DROP + CREATE the homologator does on its own.
- `recompute_user_accesos` rewrote 1 of 3 users (company 1 user 2, `ZwCEAA==` → `AGcAhA==`, the same
  two grant words byte-swapped to big-endian). A second run reports `0 reescritos`.

Two things had to be fixed to get there, and both are worth knowing:

- Both companies in that database were `status = 0`, so the script's `Status.GreaterEqual(1)` filter
  matched nothing and it exited `no hay empresas activas`. They were legacy rows predating
  `PostSignUpCompany`, which writes `Status: 1`; both were set to `1`. Nothing gates *login* on
  company status, which is why they worked while being invisible to this script and to
  `observability_backfill`.
- The `db.Update` in `recompute_accesos.go` and in `PostPerfiles` named only the two blob columns,
  which the `(status, updated)` view on `users` rejects. Both now name `Status`. See
  `backend/RATIONALE.md` — the `PostPerfiles` one was a live bug, not only a migration-script bug.

What remains is the browser check itself: that the checkbox row renders and saves against a real
backend. Everything else it would cover is asserted by the test suites above.

### 2. Nothing reads a sub-access yet

`Punto de Venta` declares `Recibir Pago` and `Despachar Producto`, the profile stores them, the
daemon carries them and `req.User.HasSubAcceso` is available — but **no handler calls it** and no
frontend calls `security.checkSubAcceso`. The mechanism is complete and unused; the first consumer
is a separate piece of work. This is documented as a limitation on the route.

### 3. Deferred, by decision

Sub-accesses on a **direct** per-user access grant (`AccessLevelIDs`). Recorded in
`backend/RATIONALE.md` so it is a decision rather than a gap; the user asked to plan it separately.

---

## Deploy order

1. `fn-homologate` (deploy.sh → **Recrear Tablas**) — adds `users.accesos_sub_computed` and
   `profiles.sub_accesos`. `check_tables` validates the structs; it does not create columns.
2. Deploy backend **and** daemon together. The `:v9` domain makes a mixed pair fail at the first
   frame — loud, and intended, but the window is real.
3. `cd scripts && go run . recompute_user_accesos`.
4. Deploy the frontend. The stored-accesos key is already bumped to `AccesosV2`, so a stale
   `Uint16Array` payload is discarded rather than misread; no further action needed there.

Between 2 and 3 every authenticated request is denied. Keep it short.

---

## Known risks

1. **The two-payload lookup.** An access lives in exactly one column, so any check that misses
   `accesos_computed` must also scan `accesos_sub_computed`. Missing that second lookup fails closed
   — a user is denied something they hold. All three languages now have a test for exactly that
   lookup: Rust `an_access_is_found_in_either_blob`, Go `TestGrantsRoundTripThroughBothBlobs`, TS
   `an access held only in the sub payload`.
2. **Three hand-written parsers.** Go and Rust are cross-pinned by a fixture captured from the Go
   encoder (`decodes_the_blobs_the_go_encoder_writes`); TS is pinned by hand-written bytes matching
   the Go test's. The near-miss worth remembering: sub-access 13 is bit 12, which lands at bit **5**
   of the second byte (`0x20`), not bit 6 (`0x40`). Nothing about `0x40` looks wrong, and a reader
   making the same slip grants sub-access 14.
3. **Scope limit, by design.** A handler can read sub-accesses only for the accesses that gated
   **its own route** — that is all the reply carries.
4. **Unrelated working-tree changes.** `frontend/static/sw.js` is modified and `server_utils/` is
   untracked; neither is part of this work.
