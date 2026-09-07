# RATIONALE — security

Design decisions for the security module, newest first.

## One grant shape for profiles and users, stored as a colbin blob

**Context** — Three flat integer arrays with three hand-rolled packings expressed the same idea:
`profiles.accesos` (`accesoID*10 + nivel`), `profiles.sub_accesos` (`accesoID*100 + subID`) and
`users.access_level_ids` (`accesoID*10 + nivel`). A user could not hold a sub-access at all — the
column did not exist, and `PostUsuarios` merged direct grants with `addAccesoNivelToGrants`, which
has no sub-bytes to give. Two records granting the same thing, in two shapes, one of them crippled.

**Decision** — `coreTypes.AccesoGrantRecord{AccesoID uint16, Nivel uint8, SubAccesos SubAccesoIDs}`,
stored by both records in an `accesos_grants` blob column that colbin serializes. One merge function
(`addAccesoGrantToMerge`), one validator (`ValidateAccesoGrants`), both used by the profile handler
and the user handler. The derived runtime blobs — `accesos_computed` / `accesos_sub_computed`, read
byte for byte by fareward and by genix-ui — are untouched. Full plan and the answered design
questions: `docs/ACCESO_GRANTS_STORAGE_PLAN.md`.

**Rationale** — Nesting the sub-accesses inside the grant makes an orphan *unrepresentable*: the two
checks that used to guard "a sub-access whose access nobody granted" — one in the validator, one in
the merge — are simply gone, because a sub-access now lives inside the access it qualifies. One
`Nivel` and not a list for the same reason: the merge kept only the highest anyway, so the list only
ever let the editor express something the backend discarded on the way in. What it costs: an opaque
blob where there used to be integers a human could read in a `SELECT`, and one permanent obligation
— the `cb` field ids are positional, so they are never renumbered and a removed field's number is
never reused.

## `[]uint8` reaches the browser as base64 unless you stop it

**Context** — `AccesoGrantRecord.SubAccesos` was `[]uint8`, which in Go **is** `[]byte`.
`encoding/json` writes any byte slice as a base64 string, so a grant saved correctly came back to
the client as `"SubAccesos":"Ag=="` where it expects `[2]`.

**Decision** — A named type, `SubAccesoIDs []uint8`, with a `MarshalJSON` that writes a number array.
`core/types/accesos_test.go` asserts the exact JSON, and separately the colbin round trip.

**Rationale** — The failure is asymmetric and therefore quiet: the *decoder* takes both spellings, so
writing worked from the first save and only reading was broken. It would have surfaced as a
sub-access rendering as four characters of base64, far from its cause. A named type keeps the
storage type — and so colbin's encoding — exactly as designed; the alternative, widening the field
to `[]uint16`, would have changed the blob to work around a JSON quirk.
