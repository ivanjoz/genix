# ACCESO GRANTS STORAGE PLAN

One editable grant shape for profiles and users, stored as a colbin blob.

> Status: **implemented**. Backend, frontend and the schema are done and verified end to end; the
> old columns (`profiles.accesos`, `profiles.sub_accesos`, `users.access_level_ids`) are still in
> the database, unmapped, waiting to be dropped by hand.
>
> Two things the plan did not foresee, both recorded below: §8 (the base64 trap) and §9 (what was
> verified and what was not).
>
> Original status: **approved**. The three open questions in §7 are answered:
>
> 1. **One level per access, the highest.** The multi-level list goes away; the Profiles tab's level
>    buttons become a one-of choice.
> 2. **Full JSON keys** — `{AccesoID, Nivel, SubAccesos}`. Only the `cb` tags are numbered.
> 3. **No migration.** The existing rows are being deleted by hand, so §4 is dropped: the new column
>    starts empty and the old ones are left unmapped until they are dropped manually.

## 1. What changes

Today the two records that grant access store it in three different flat integer arrays, each with
its own hand-rolled packing:

| Record | Column | Go type | Encoding |
|---|---|---|---|
| Profile | `accesos` | `[]int32` | `accesoID*10 + nivel`, one entry per level |
| Profile | `sub_accesos` | `[]int32` | `accesoID*100 + subID`, one entry per sub-access |
| User | `access_level_ids` | `[]int32` | `accesoID*10 + nivel` — **no sub-accesses at all** |

After:

```go
// core/types (shared: a profile and a user grant access the same way).
//
// The cb tags are positional field ids, not names: that is what keeps the blob compact and lets a
// field be renamed without rewriting every stored row. Same convention as
// sales/types.SaleOrderProductStats. Ids are permanent — a removed field's number is never reused.
type AccesoGrantRecord struct {
    AccesoID   uint16  `cb:"1"`
    Nivel      uint8   `cb:"2"`
    SubAccesos []uint8 `cb:"3" json:",omitempty"`  // sub-access ids, 1 = "Todos"
}
```

| Record | Column | Go type |
|---|---|---|
| Profile | `accesos_grants` | `db.Col[*ProfileTable, []coreTypes.AccesoGrantRecord]` |
| User | `accesos_grants` | `db.Col[*UserTable, []coreTypes.AccesoGrantRecord]` |

`db.Col` with a struct-slice element is already a supported ORM shape — it lands as a `blob` and the
converter runs `colbin.Marshal` on it (`genix-orm/scylla/converter.go:784`). Precedent:
`ProductTable.Presentations`, `WarehouseTable.Layout`, `SalesPlanningTable.WeeklyQuantity`.

**Not changed:** `users.accesos_computed` / `accesos_sub_computed`. Those are the *derived* runtime
blobs, read byte for byte by `fareward/src/limiter/access.rs` and `genix-ui/security/accesos.ts`.
They stay exactly as they are; this plan only replaces the **editable input**, which is what
`PostUsuarios` reads to build them. **fareward needs no change.**

## 2. Why one `nivel`, not a list

`profiles.accesos` holds one entry *per level*, so an access granted VER+TODO is two entries. The new
shape has a single `Nivel` per access. That is not a loss: `addAccesoNivelToGrants` already collapses
levels to the highest one when it merges, and `core.AccesoGrant` — the merged shape the blobs are
built from — has carried a single `Nivel uint8` all along. The multi-level list only ever existed in
the editor.

**Consequence for the UI:** the access card's level buttons become a single choice (radio-like)
rather than independent toggles. This is a visible behaviour change on the Profiles tab and needs
confirming — see §7.

## 3. Column type cannot be altered in place

The ORM adds missing columns automatically (`deploy.go:969` `ALTER TABLE … ADD`) but for a column
whose type changed it only **logs a warning** (`deploy.go:951`) and moves on. Scylla will not convert
`list<int>` to `blob` either.

So the new column takes a new name, `accesos_grants`, and the old ones are left in the database
unmapped (the deployer logs them as "exists in the DB, not mapped in the struct"). Dropping them is a
separate manual `ALTER TABLE … DROP` once the data is confirmed migrated.

## 4. Data migration

Live rows exist: `profiles.accesos` / `sub_accesos` are populated, `users.access_level_ids` is
populated for at least one user. Options:

- **A. One-off migration function** (recommended) — a `fn-migrate-accesos` entry in the script
  dispatcher that reads every profile and user, converts the flat arrays to `[]AccesoGrantRecord`,
  and writes the new column. Dry-run first, `--apply` to commit. ~60 lines, deleted after it runs.
- **B. Re-enter by hand** — pre-alpha, the dataset is one company and a handful of profiles.

A is cheap and removes the risk of a profile silently losing grants.

## 5. Work items

**Backend**

1. `core/types/accesos.go` (new) — `AccesoGrantRecord` + `ToMergedGrants` helper that converts a
   `[]AccesoGrantRecord` into the `[]core.AccesoGrant` (SubMask) shape the encoder wants.
2. `core/types/users.go` — drop `AccessLevelIDs`, add `AccesosGrants []AccesoGrantRecord`; same in
   `UserTable` as `db.Col[..., []AccesoGrantRecord]` `db:"accesos_grants"`.
3. `security/types/perfiles.go` — drop `Accesos` + `SubAccesos`, add `AccesosGrants`.
4. `security/perfiles.go` — `validateProfileSubAccesos` becomes `validateAccesoGrants(grants)`:
   same catalog checks (access exists, declares that sub, sub id ≤ `MaxSubAccesoID`, "Todos"
   exclusivity), now driven by the nested list instead of the `accesoID*100` refs. Used by **both**
   the profile and the user handler — a user's direct grants are validated for the first time.
5. `security/usuarios.go` — `buildAccesosComputedFromPerfiles` and the direct-grant merge both walk
   `AccesosGrants`; `addSubAccesoToGrants` now also runs for direct user grants, which is what makes
   a user-level sub-access reach `accesos_sub_computed`.
6. `security/recompute_accesos.go` — same field rename.
7. Tests: `perfiles_test.go` cases rewritten against the new shape; add a `core/types` round-trip
   test (grants → merged → `EncodeAccesosGrants` → `DecodeAccesosGrants`).

**Frontend**

8. `core/types/common.ts` — `IAccesoGrant { a: number; n: number; s?: number[] }`;
   `IProfile.AccesosGrants`, `IUser.AccesosGrants`; drop `Accesos`, `SubAccesos`, `AccessLevelIDs`.
9. `users-profiles.ts` — `packSubAccesos` / `unpackSubAccesos` / `decodeAccessLevelIDs` /
   `encodeAccessLevelIDs` all collapse into **one pair**: `toAccesoGrants(accesosMap, subAccesosMap)`
   and `fromAccesoGrants(grants)`. Their tests fold in the same way.
10. `users-profiles.svelte.ts` — the profile cache hydrator reads `AccesosGrants`; cache `ver` bumps
    to 4, since a v3 record would hydrate as "grants nothing".
11. `ProfilesTab.svelte` — `savePerfil` writes `AccesosGrants` instead of the two arrays.
12. `UserProfilesAccessSelector.svelte` — the two sync effects use the new pair; the edit form gains
    a `subAccesosMap`, which is all `AccessCard` needs to start showing the sub-access row.
13. `UsersTab.svelte` — form init/clone.

**Docs** — `SUB_ACCESSES_PLAN.md` and `SUB_ACCESSES_STATUS.md` describe the `accesoID*100` input
format; both get a note. `RATIONALE.md` in `security/users-profiles` and a new entry in
`backend/security`.

## 6. Order of execution

1. `AccesoGrantRecord` + conversion helpers + their tests (no callers yet).
2. Backend structs, handlers, tests. `go build ./...` + `go test ./security/... ./core/...`.
3. Deploy the schema (adds the column), run the migration dry-run, then `--apply`.
4. Frontend types, helpers, tests, then the three components.
5. Verify in the browser: profile grants round-trip, user grants round-trip, a user-level
   sub-access reaches `accesos_sub_computed` (decode it back with `DecodeAccesosGrants`).
6. Drop the old columns.

## 7. Open questions

1. **Single level per access** (§2) — confirm the Profiles tab's level buttons become one-of, not
   many-of. Everything downstream already assumes the highest level wins, so this only formalises it,
   but it changes how the card behaves.
2. **JSON key spelling** — the `cb` tags are numbered, but the `json` tags are what the frontend
   sees: `{a: 12, n: 4, s: [2,3]}` as written above, versus `{AccesoID, Nivel, SubAccesos}`. Short
   keys keep the payload small at the cost of being unreadable in a network trace.
3. **Migration**: option A or B (§4).

## 8. The base64 trap

`[]uint8` **is** `[]byte`, and `encoding/json` writes any byte slice as a base64 string. So the
first working save stored the grant correctly and handed the browser back
`"SubAccesos":"Ag=="` where it expects `[2]`.

It is asymmetric, which is what makes it dangerous: the *decoder* accepts both a base64 string and
an array of numbers, so the write path worked from day one and only the read path was broken —
a sub-access that comes back as four characters of base64 instead of a ticked checkbox.

Fixed by giving the field a named type, `SubAccesoIDs []uint8`, with a `MarshalJSON` that writes an
array. The storage type is unchanged, so colbin is unaffected. `core/types/accesos_test.go` asserts
both round trips, JSON and colbin, because neither failure announces itself.

## 9. What was verified

Verified end to end in the running app: granting an access to a user, ticking one of its
sub-accesses, saving, and reading the row back — `AccesosGrants` holds
`[{AccesoID: 10, Nivel: 4, SubAccesos: [2]}]` and `accesos_sub_computed` holds `00 2b 02`, which
decodes to access 10 at nivel 4 with sub-access 2. The users table renders the new shape.

**Not verified:** reopening a saved user in the edit layer and seeing the sub-access checkbox
already ticked. The layer only opens from the table's `onRowClick`, which the agent browser's
`selectRow` does not invoke — the same gap that blocks the profiles editor and the assets detail
panel. The data reaching that layer is correct (§8's test plus the row read above); what is
unchecked is the component rendering it.

Also unverified for the same reason: the **Profiles tab** editing path. Its code changed with
everything else and typechecks, but a profile can only be selected by clicking its table row.
