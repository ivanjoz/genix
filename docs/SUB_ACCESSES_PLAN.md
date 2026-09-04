# Sub-Accesses Plan

> **Status.** Every step is implemented. Steps 1, 2, 3 and 5 landed in `3d04ec13`; steps 4 and 6 —
> the recompute script and the frontend — plus the remaining tests and the RATIONALE entries are in
> the working tree. Backend build/vet/test, `check_tables`, `check_module_imports`, `cargo test`
> (187), the frontend typecheck and build, and the route-documentation validator all pass; the two
> failures that remain (`agent/ragdocs` stale evidence, 10 `svelte-check` errors) are pre-existing
> and unrelated. **The one thing left is the end-to-end `agent-browser` check**, which is blocked on
> the live migration in §6 because it rewrites authorization data on the shared database. Read
> `docs/SUB_ACCESSES_STATUS.md` for the handoff and the deploy order.
>
> Two things the plan did not decide, now decided: nothing in Genix **reads** a sub-access yet
> (`req.User.HasSubAcceso` and `security.checkSubAcceso` exist and have no callers — the first
> consumer is separate work), and sub-accesses on a **direct** per-user access grant are deferred by
> request, recorded in `backend/RATIONALE.md`.

Adds a second, flag-shaped permission level under each access: an access may declare up to 13
sub-accesses (id + name), and a profile may grant any subset of them. Sub-accesses are **not** a
hierarchy and carry **no level** — they are flags a handler reads to decide what it may do inside a
route it is already authorized for.

Spans three repos: `backend` (Go), `fareward` (Rust daemon + its Go client), `frontend`
(`genix-ui` submodule + the users-profiles route).

---

## 1. Decisions

Settled before writing this. Each one becomes a `RATIONALE.md` entry in step 8.

| # | Decision |
| --- | --- |
| D1 | The catalog moves from YAML to TOML, and its sections are renamed `access_list` → `access`, `access_groups` → `groups`. File becomes `backend/access.toml`. |
| D2 | Sub-accesses are declared as two parallel single-line arrays: `sub_accesses_ids` and `sub_accesses_names`. |
| D3 | Sub-access id `1` is reserved for "Todos" and is never declared in the catalog. Ids run `2..13`. |
| D4 | Grants are split across **two `[]byte` columns**: `accesos_computed` for accesses with no granted sub-access, `accesos_sub_computed` for those with at least one. An access lives in exactly one of them. Both are blobs the ORM copies verbatim and Rust reads without conversion. |
| D5 | Both columns are **big-endian** and share the same 16-bit grant word `[14 bits accesoID][2 bits nivel-1]`. In `accesos_sub_computed` that word is always followed by sub bytes of `[1 bit MORE][7 bits flags]` — **the column itself is the "has sub-accesses" flag**, which is why no bit is spent on one and both keep 14-bit ids. Entries sorted ascending by accesoID. |
| D6 | The profile keeps a readable list. The frontend sends `SubAccesos []int32` as `accesoID*100 + subID`; Go validates it against the catalog and packs the blob. Binary encoding exists in exactly one place. |
| D7 | The fareward reply frame gains a length-prefixed tail. `detail` reports which required slots were granted and which contributed sub bytes; the tail is those sub bytes copied verbatim out of the cached blob. |
| D8 | The daemon stays ignorant of the catalog. It returns opaque masks; "id 1 means all" is expanded in Go and TS. |

### Why `[]byte` over widening to `[]uint32`

`genix-orm/scylla/converter.go:161` and `:232` take a `reflect.Copy` fast path for `[]uint8` — a
memcpy in both directions — while `[]uint16` pays a per-element `binary.LittleEndian` loop at `:170`
and `:244`. On the Rust side `StoredUserAccess.grants_blob` is *already* `Vec<u8>` straight out of
Scylla (`storage.rs:27`), and `decode_grants` is the only thing that converts it. With a `[]byte`
blob both processes hold the identical bytes and the conversion disappears.

`accesos_computed` keeps its fixed 2-byte stride, so binary search survives there unchanged.
`accesos_sub_computed` is variable-width and is scanned linearly — which costs nothing at its size,
and is why the variable half was pushed into the array that stays small.

### Why big-endian

`access.rs::decode_grants` currently carries a standing warning: *"The blob is little-endian. Every
integer in this daemon's wire protocol is big-endian and this column is not… Getting this backwards
would not fail — it would silently authorize the wrong things."* Since the bytes are now ours to
choose and the ORM only copies them, big-endian deletes that exception, and it makes sorting by raw
`u16` identical to sorting by accesoID.

### What is lost, and the replacement

`decode_grants` sorts and dedups defensively so that an out-of-order blob degrades into a wrong
answer for one user rather than a broken binary search. That defense cannot survive on
`accesos_sub_computed`, where entries are variable-width and position is load-bearing.
**Replacement, applied to both columns for symmetry:** validate that `accesoID` strictly increases
while walking — free, since the parser is already walking — and reject the blob loudly otherwise.
Same intent, better failure mode, and it also catches the sub array's own corruptions (a truncated
entry, a dangling `MORE`).

The recurring cost is **three hand-written parsers** (Go, Rust, TS) that must agree exactly, where
today they share a one-line pack/unpack. Mitigated by fixture tests on all three sides, following
the pattern `access.rs` already uses: assert against bytes written by hand *and* against a real blob
read out of the dev database.

### Why two columns rather than one self-delimiting blob

The alternative was a single column where the grant word spends one bit on a `SUB_FOLLOWS` flag:
`[13 bits accesoID][2 bits nivel-1][1 bit SUB_FOLLOWS]`. One column, one lookup.

Two columns wins because **the column name carries that bit for free**. In `accesos_sub_computed`
a sub byte always follows — that is the definition of the column — so nothing has to be encoded to
say so, and the id field keeps its full 14 bits in both arrays instead of dropping to 13.

It also keeps `accesos_computed` at a **fixed 2-byte stride**, so the common array stays binary
searchable and its length check is a single modulo. Only `accesos_sub_computed` is variable-width,
and it is the small one: an access appears there only when a profile actually granted it a
sub-access.

The cost is real and accepted: the authorization check consults both arrays, and the "not in the
first, now look in the second" branch has to be right in all three languages. It fails closed — a
missed second lookup denies a user who should be allowed — but it is the thing to keep an eye on.

The performance question that prompted this is empty either way, and is worth writing down so it is
not re-opened: a user's grants are ~110 bytes total, and the check sits behind a
`tokio::sync::Mutex` on the shard (`quota.rs:455`) and a `HashMap` lookup on `(company_id, user_id)`
(`quota.rs:474`), each of which costs more than scanning every byte of both arrays. Binary search
versus linear scan is not the layer where anything is won here.

### Rejected: a fixed 2-byte sub-mask instead of the continuation bit

Also considered: drop `MORE` and always spend 2 bytes on the sub-mask when `SUB_FOLLOWS` is set, so
an entry is 2 or 4 bytes and never anything else. Simpler parser, 16 sub-accesses instead of 13.

Not taken. The continuation bit keeps the common case at one byte, and "Todos" (sub id 1, bit 0) is
expected to be the most frequently granted sub-access — so most sub-bearing accesses spend one byte,
not two. The saving is small in absolute terms; the decision is to keep the encoding the user
specified rather than trade it for parser convenience.

---

## 2. Binary formats

### 2.1 Grant blobs — `users.accesos_computed` and `users.accesos_sub_computed`

Both columns are `[]byte`, big-endian, sorted ascending by accesoID. They share one grant word:

```
grant : u16 BIG-endian
        bits 15..2   accesoID        14 bits, 1..16383
        bits  1..0   nivel - 1       1..4
```

`accesos_computed` — accesses with **no** granted sub-access. Nothing but grant words, so the
stride is a fixed 2 bytes and the whole array is binary searchable.

`accesos_sub_computed` — accesses with **at least one** granted sub-access. Every grant word is
followed by one or more sub bytes; no flag says so because the column is the flag.

```
sub   : u8
        bit      7   MORE      another sub byte follows
        bits  6..0   flags     byte 0 = sub 1..7, byte 1 = sub 8..14
                               bit 0 of byte 0 = sub-access 1 = "Todos"
```

Sub-accesses cap at 13, so an entry here is 3 or 4 bytes. Scan it linearly: read 2 bytes BE, then
consume `u8`s until one without `MORE`; early-exit once accesoID passes the target.

An access is in **exactly one** array, so a lookup that misses the first must try the second. Both
are written from a single in-memory merge in one row update, so they cannot disagree.

Sizes: a user holding all 36 catalog accesses with no sub-accesses is 72 bytes in
`accesos_computed` and empty in the other — identical to today.

**Neither CQL column type is new.** `converter.go:118`/`:122` map both `[]uint8` and `[]uint16` to
the same `blob`, so `accesos_computed` needs no `ALTER TABLE`, only a data recompute (step 4).
`accesos_sub_computed` is a new column and does need one.

### 2.2 Reply frame — all opcodes

```
[correlation:u16][status:u8][detail:u16][extra_len:u8][extra: extra_len bytes]
```

`extra_len == 0` for lock acquire/release, budget mutate, and any charge that requested no
authorization. Their `status`/`detail` meanings are untouched — locks keep their generation in
`detail` (`locks.go:182`), which is why `detail` is `u16` in the first place.

`detail` for opcode `0x01` (charge):

| bits | meaning |
| --- | --- |
| 0..2 | verdict: `0` not requested, `1` granted, `2` no-access, `3` unknown-user, `4` inactive-user |
| 3..6 | **granted mask** — bit N set = `required_access[N]` is held |
| 7..10 | **has-subs mask** — bit N set = slot N contributed sub bytes to the tail |
| 11..15 | free |

`extra` = the sub-byte runs for the slots in the has-subs mask, ascending slot order, **copied
verbatim from the cached blob** — the daemon re-encodes nothing. Each run self-terminates on its
`MORE` bit. Maximum tail is 8 bytes (4 slots × 2); the client rejects anything larger for `0x01`.

Existing denial precedence is unchanged: `UnknownUser` / `InactiveUser` outrank `NoAccess`, because
Go turns the first two into 401s and the last into a 403.

This is a wire change, so `farewardAuthDomain` goes `:v8` → `:v9` (`connection.go:44`) and `DOMAIN`
in `fareward/src/service/auth.rs` with it. **Backend and daemon must deploy together.**

### 2.3 Catalog — `backend/access.toml`

```toml
# System access catalog. Single source of truth: the backend embeds it (main-handlers.go) to
# authorize every POST/PUT route, and the frontend imports it as text (access-list-catalog.ts)
# to decide which pages and menu options are shown.
#
# RULES
#   - `id`s are permanent: persisted in `profiles.accesos` as (id*10 + nivel). Never renumbered,
#     never reused; a retired access is deleted and its id retired with it.
#   - `frontend_routes`: comma-separated, NO leading "/". Matching is by segment prefix, longest
#     wins, so "webpage-builder" also covers "/webpage-builder/<pageID>".
#   - `backend_apis`: comma-separated "<METHOD>.<route>". EVERY non-public POST/PUT route must
#     appear here — the backend denies by default what is not mapped. GETs are the reverse:
#     unmapped they are open to any session, and mapping one is what closes it.
#   - `levels`: concatenated digits of the offered levels. 14 = 1 (VER) + 4 (TODO).
#   - `sub_accesses_ids` / `sub_accesses_names`: parallel arrays, equal length, single line.
#     Ids are permanent, run 2..13. Id 1 is reserved for "Todos" and is never declared.

[[groups]]
id = 1
name = "Mi Empresa"

[[access]]
id = 10
name = "Punto de Venta"
group = 3
levels = 14
frontend_routes = "sales/sale_order_create"
backend_apis = "GET.company-parametros,POST.sale-order,POST.system-parameters"
sub_accesses_ids = [2, 3, 4]
sub_accesses_names = ["Anular venta", "Aplicar descuento", "Reimprimir ticket"]
```

---

## 3. Step 1 — YAML → TOML migration (own commit, no behaviour change)

Landed separately and mechanically so the sub-access diff is not tangled with a format migration.
**No sub-access fields yet.**

- `backend/access_list.yml` → `backend/access.toml`, sections renamed per D1, header translated to
  English (project rule; the current header is legacy Spanish).
- `backend/main-handlers.go:64` — `//go:embed access.toml`.
- `backend/core/usuario-accesos.go` — `AccessListYaml` → `AccessCatalog`; `yaml:"…"` tags become
  `toml:"…"`; `yaml.Unmarshal` → `toml.Unmarshal`. `github.com/pelletier/go-toml/v2` is **already a
  direct dependency** (`backend/go.mod:75`), so no new dep. Drop `gopkg.in/yaml.v3` only if nothing
  else uses it.
- `frontend/routes/security/users-profiles/access-list-catalog.ts` — rewrite
  `parseAccessListCatalog` for TOML. It gets shorter: no indentation tracking, no `- ` prefix
  handling, records self-delimit on `[[name]]`.

  ```ts
  for (const sourceLine of tomlContent.split(/\r?\n/)) {
    const line = sourceLine.trim()
    if (!line || line.startsWith('#')) { continue }

    const sectionMatch = /^\[\[([a-z_]+)\]\]$/.exec(line)
    if (sectionMatch) { /* push a fresh record into parsedCatalog[section] */ ; continue }

    const [, fieldName, rawValue] = /^([a-z_]+)\s*=\s*(.*)$/.exec(line)
    activeRecord[fieldName] = rawValue.startsWith('[') || rawValue.startsWith('"')
      ? JSON.parse(rawValue)   // single-line TOML arrays and strings are JSON-shaped
      : Number(rawValue)
  }
  ```

  Reject multi-line arrays and trailing commas explicitly with a clear message — both are legal TOML
  and neither survives `JSON.parse`. This matches how the parser already throws on unsupported
  shapes rather than letting a `SyntaxError` surface.
- `IAccessListCatalogPayload` → `{ groups, access }`; update every consumer of `.access_list` /
  `.access_groups` (`ProfilesTab.svelte`, `access-list-catalog.test.ts`, and the `?raw` import path).
- Grep for the old filename across `.md` docs, `frontend/webpage/components/UsuarioMenu.svelte`,
  `backend/agent/turn.go`, `fareward/README.md` and `fareward/go/credits.go` comments.

**Verify:** `cd backend && go build ./... && go vet ./... && go test ./...` ·
`cd frontend && bun run check` · the existing `access-list-catalog.test.ts` must pass with only its
fixture rewritten.

---

## 4. Step 2 — catalog and profile gain sub-accesses

- `access.toml`: add `sub_accesses_ids` / `sub_accesses_names` to the accesses that need them.
- `core/usuario-accesos.go`: parse them into `AccessInfo`. Two hard validations at load, both
  naming the offending access id — parallel arrays invite exactly these:
  1. length mismatch between the two arrays;
  2. an id outside `2..13`, or a duplicate.
- `security/types/perfiles.go`: new column `SubAccesos db.ColSlice[*ProfileTable, int32]` holding
  `accesoID*100 + subID`. Schema change → use the `create-database-tables` skill and
  `scripts/CREATE_EDIT_TABLE.md`. `users.accesos_sub_computed` is a new column on the same pass.
- `security/perfiles.go` (`PostPerfiles`): validate each entry against the catalog — access must
  exist, sub id must be declared on *that* access, and the profile must actually grant the parent
  access. Drop + log anything else. **NEVER trust the client.**

**Verify:** `cd scripts && go run . check_tables` · `cd backend && go test ./security/...`

---

## 5. Step 3 — the blob, in three languages

### Go (writer + reader)

- `core/responses.go`: `MakeAccesoNivelPacked` keeps its `accesoID<<2 | (nivel-1)` packing — the
  grant word is unchanged, only its endianness and container are. Add the matching unpack, plus the
  two blob encoders. Single shared pair, exported, so `security` and the tests use one
  implementation.
- `security/usuarios.go::buildAccesosComputedFromPerfiles`: merge across the user's profiles by
  taking the **max nivel** per access (as today) and **OR-ing the sub-masks**; then split the merged
  map into the two sorted blobs by whether the mask is zero. Returns both.
- `core/types/users.go`: `AccesosComputed` becomes `[]byte`; add `AccesosSubComputed []byte` on both
  `User` and `UserTable`.
- `security/perfiles.go:99-114`: the change detection must compare **both** columns before deciding
  a user is unchanged, and `db.Update` must name both.

### Rust (reader)

- `access.rs`: `UserAccessState` holds `grants: Box<[u8]>` and `sub_grants: Box<[u8]>` **verbatim** —
  no `Vec<u16>`, no sort, no dedup, one fewer allocation per cache load.
- `decode_grants` → `validate_grants`: `grants` is checked with a length modulo and a
  strictly-increasing walk; `sub_grants` is walked for truncated entries, dangling `MORE`, and the
  same ordering rule. Both return the bytes as-is.
- `holds()` → one pass over `grants` and one over `sub_grants`, resolving **all**
  `MAX_REQUIRED_ACCESS` slots and returning `(granted_mask, has_subs_mask, tail_bytes)`. The
  `required | 0b11` bucket-ceiling trick goes away; it becomes a plain
  `granted_nivel >= required_nivel` after unpacking, which reads better.
- `storage.rs`: `StoredUserAccess` gains `sub_grants_blob`; the `SELECT` at `:217` reads the second
  column; the doc comment at `:23-24` currently says little-endian u16s and must be rewritten.

### TypeScript (reader) — `genix-ui` submodule

- `security/accesos.ts`: `decodeStoredAccesosComputed` returns `Uint8Array`; `base64ToUInt16` is
  **replaced** by `base64ToBytes`. As built, the range helpers are gone rather than adapted:
  `findAccesoNivel` binary-searches the fixed stride and returns the level, `findAccesoSubGrant`
  walks the variable-width bytes, `hasAcceso` does the two-payload lookup, and
  `hasSubAcceso(subBlob, accesoID, subID)` applies "Todos". `validateAccesosBlobs` replaces the
  defensive re-sort.
- `security/create-security.ts`: `accesosComputed: Uint16Array` → two `Uint8Array`s, and
  `accesoResultCache` must key on `(accesoID, nivel, subID)` rather than a bare number.
- `security/index.ts` + `types.ts`: update the exports and `checkAcceso`'s signature; add
  `checkSubAcceso`.
- This is a **submodule change** — commit and push inside `frontend/packages/genix-ui`.

### "Todos" expansion (D8)

One helper per language, and only in Go and TS: a sub-access is held when bit 0 (id 1, "Todos") is
set **or** bit `subID-1` is set. The daemon never applies this rule — it returns the raw mask.

**Verify:** `cd backend && go build ./... && go test ./...` ·
`cd fareward && cargo build && cargo test` · `cd frontend && bun run check`

---

## 6. Step 4 — data recompute

Existing `accesos_computed` values are little-endian `id<<2|nivel-1`; the grant word survives the
change but its **endianness does not**, so every stored blob decodes to nonsense and every user
would be denied. Pre-alpha means no compatibility shim, so:

- New script `scripts/recompute_user_accesos.go` (dispatcher convention: `scripts/SCRIPTS.md`).
  Walks every company's users, rebuilds both grant columns from their profiles through the same
  `buildAccesosComputedFromPerfiles` path, writes them back, and fires one
  `InvalidateUserAccess(companyID, InvalidateAllCompanyUsers)` per company.
- Run it **after** the backend deploy and **before** or immediately alongside the daemon deploy.

**Deploy order for this whole change:**
1. Deploy backend + daemon together (the `:v9` domain bump makes a mixed pair fail at the first
   frame, which is the intended loud failure).
2. Run `recompute_user_accesos`.
3. Deploy the frontend (its stored `Accesos` payload is refreshed on next login anyway; bump the
   storage key or the wrap checksum so a stale `Uint16Array` payload is discarded rather than
   misread).

---

## 7. Step 5 — the reply frame and the Go handler surface

### fareward

- `service/protocol.rs`: `REPLY_SIZE` 5 → 6 fixed head, `encode_reply(sequence, status, detail,
  extra: &[u8])`. Bump `DOMAIN` to `:v9` in `service/auth.rs`.
- `service/server.rs:328-330`: `Decision::Allowed` now carries the granted/has-subs masks and the
  tail; `send_reply` and its `mpsc::Sender<[u8; REPLY_SIZE]>` channel type must take a variable
  buffer.
- `limiter/quota.rs:460-482`: `Decision::Allowed` gains the payload; `admit_at` returns it.

### fareward/go client

- `connection.go:33,44`: read the 6-byte head then `extra_len` bytes; `muxReply` gains `extra []byte`.
  Reject `extra_len > 8` for `opcodeChargeCredits` and `!= 0` for the opcodes that never carry one.
- `credits.go`: `ChargeAPIUsage` / `ChargeAPIAccessOnly` return the decoded per-slot sub-masks
  alongside their error. Keep it opaque — `doc.go:44` already states this package carries a verdict,
  not a name, and that stays true.
- **Submodule change** — commit and push inside `fareward`.

### backend

- `core/usuario-accesos.go`: `UsuarioToken` gains `SubAccesos map[int32]uint16` tagged **`cb:"-"`**
  — like `Error`, it must never be serialized into the token.
- `main-handlers.go::enforceAccessAndCredits`: map each returned slot mask back onto
  `accessInfos[i].ID` and set it on `args.User`. Set it on the **per-request** `args.User`, never the
  `core.User` global — that global is only assigned under `IS_SERVERLESS` precisely because local/VPS
  mode is concurrent.
- New `core` helper: `func (u *UsuarioToken) HasSubAcceso(accesoID int32, subID uint8) bool`,
  applying the "Todos" rule.

### Three edge cases that produce no sub-accesses

`resolveRouteAccess` returns an empty `requiredAccess` — and therefore sends no frame — in exactly
three cases, all deliberate. Each needs a decided answer:

| Case | Behaviour |
| --- | --- |
| Unmapped GET | No gating access exists, so there are no sub-accesses to read. A handler on such a route must not depend on them. |
| `selfServiceRoutes` | Same. |
| **User 1** | Bypasses the check because `login.go` synthesizes its accesses and never persists them, so its blob is empty and the daemon would deny it. Go must synthesize **all sub-accesses of every access** for user 1, mirroring what `MakeUsuarioResponse` already does for the accesses themselves. |

**Scope limit, stated plainly:** a handler can read the sub-accesses of the accesses that **gated
its own route**, and nothing else. Reading an unrelated access's sub-flags would need the full blob,
which is not what the reply carries.

**Verify:** `cd fareward && cargo test` · `cd backend && go test ./...` ·
skill `agent-browser` against a route with sub-accesses, as a real end-to-end check.

---

## 8. Step 6 — frontend surface

- `security/login.go::MakeUsuarioResponse`: `encodeAccesosComputedBase64` emits the raw blob; the
  user-1 synthesis path must also emit sub-masks.
- `users-profiles.svelte.ts`: `PerfilesService.handler` builds `subAccesosMap` alongside
  `accesosMap`; `postPerfil` strips it and sends `SubAccesos: number[]` (`accesoID*100 + subID`).
  Bump `useCache.ver`.
- `AccessCard.svelte` / `ProfilesTab.svelte`: sub-access chips under the level chips, only for
  accesses that declare any. "Todos" (id 1) is offered as a synthetic first chip that clears the
  rest.
- `UserProfilesAccessSelector.svelte`: reflect sub-accesses in the read-only user view.

**Verify:** `cd frontend && bun run check && bun run build` · skill `agent-browser` on
`security/users-profiles`.

---

## 9. Step 7 — tests to update

| File | What changes |
| --- | --- |
| `backend/access_list_slots_test.go` | Catalog path/format; add a sub-access parse + parallel-array-mismatch case; keep the "every route fits `MaxRequiredAccess`" guard. |
| `fareward/src/limiter/access.rs` (tests) | `decodes_a_real_blob_from_the_database` fixture must be **regenerated** from a recomputed dev-DB row, now covering both columns. Keep `the_blob_is_little_endian` in spirit, inverted to big-endian — it is the one thing that can be wrong without failing. Add a case for an access held only in `accesos_sub_computed`, which is the lookup that must not stop at the first array. |
| `fareward/src/limiter/protocol.rs` (tests) | Reply head/tail offsets by hand. |
| `fareward/go/credits_test.go` | `detail` bit decoding, `extra_len` bounds. |
| `genix-ui/security/accesos.test.ts` | Blob scan, "Todos" expansion, malformed-blob rejection. |
| `frontend/.../access-list-catalog.test.ts` | TOML parsing, arrays, rejected multi-line/trailing-comma. |

---

## 10. RATIONALE entries to write

- `backend/RATIONALE.md` — D1/D2 (catalog format + parallel arrays), D4/D5 (blob format, and what
  the lost defensive sort is replaced with), D6 (profile stays readable, Go packs), D8.
- `fareward/` — the `:v9` reply frame: why the tail is length-prefixed and shared by every opcode,
  and why the daemon copies sub bytes rather than re-encoding them.
- `frontend/routes/security/users-profiles/RATIONALE.md` — the sub-access editing model and the
  `accesoID*100 + subID` wire shape.
- `frontend/packages/genix-ui` — the `Uint16Array` → `Uint8Array` scan and the cache-key change.

---

## 11. Open risks

1. **Coordinated deploy.** Backend, daemon and the recompute script are one atomic change. A
   backend on `:v9` against a daemon on `:v8` fails at the first frame — loud, which is right, but
   it means the window is real. `scripts/DEPLOYER.md` fixes the execution order; confirm the
   recompute slots in correctly.
2. **Three parsers.** The format is small and each side has fixture tests, but this is the durable
   maintenance cost of D4 and it should be stated in the RATIONALE rather than discovered later.
3. **The two-array lookup.** An access lives in exactly one column, so every check that misses
   `accesos_computed` must also scan `accesos_sub_computed`. Missing that second lookup fails closed
   (a user is denied something they hold) rather than open, but it has to be right in Go, Rust and
   TS. This is the price paid for keeping 14-bit ids and a fixed-stride common array.
4. **`extra_len` on unanswered opcodes.** `opcodeLogRequest` and `opcodeInvalidateUserAccess` are
   never answered at all, so they are unaffected — but the reader must still be explicit that a
   non-charge reply carrying a tail is a protocol error, not something to skip.
