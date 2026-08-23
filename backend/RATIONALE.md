## The cash ledger writer lives in `finance/types`, not in `finance`

**Context** — `ApplyCashBankMovement` is the only way money moves: `sales` writes a collection,
`logistics` a supplier payment, `finance` an expense payment, `accounting` an asset payment. All
four reached it by importing `app/finance`, which is exactly the module-body-to-module-body
import the boundary rule forbids (V1, V3, V4 in `MODULE_BOUNDARIES_PLAN.md`).

**Decision** — `finance/cash_bank_movement.go` and `finance/shared.go` (`GetCaja`) are gone; both
functions now live in `finance/types/cash_movement_apply.go`. Consumers import only
`app/finance/types`.

**Rationale** — the alternative was a `finance/shared` package, which is one more layer and one
more name to learn. Putting it in `types` works because of the import convention: consumers
already alias `app/finance/types` to `finance`, so
`finance.ApplyCashBankMovement(req, []finance.InternalCashMovement{...})` is **byte-for-byte the
call site that existed before** — the two-line import block collapsed to one line and nothing
below it changed. The moved code depends on nothing but `core`, `db` and its own tables, so
`types` stays a leaf and the rule has no escape hatch, which is the point.

The cost is honest: `types` now holds the heaviest logic in the module, so the folder name
undersells it. A file-level comment in `cash_movement_apply.go` says why it is there, and the
filename is specific enough that `fd cash_movement` still finds it.

Fallout worth recording: four `DOCUMENTATION.md` files cited
`backend/finance/cash_bank_movement.go` as evidence, and the ragdocs parser validates evidence
by path **and** content hash. Deleting the file broke them, and editing 60+ backend files in the
earlier steps invalidated 42 more hashes. All 46 were repointed and refreshed. 127 evidence
entries were **already** stale before this work started and were deliberately left alone — that
is pre-existing documentation drift, not ours to absorb into a refactor commit.

