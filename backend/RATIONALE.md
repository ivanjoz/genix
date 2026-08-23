## `*/types` packages are imported under the module's name, not a `<module>Types` alias

**Context** — every `<module>/types` folder declares `package types`, so every import of one
needs a local name. 147 such imports had accumulated in five competing styles:
`businessTypes`, `coreTypes`, `coretypes`, bare `types`, and `s` — the last of which made
`s.ClientProvider` in `business/client_provider.go` effectively ungrepable. This mattered
beyond tidiness: the module-boundary work (`MODULE_BOUNDARIES_PLAN.md`) moves cross-module
shared functions into `*/types`, and the call sites only stay unchanged if consumers already
refer to the types package by the domain's name.

**Decision** — two rules. Outside the owning module, alias to the module name:
`finance "app/finance/types"` → `finance.CashBank`. Inside the owning module, no alias — use
the package name: `import "app/finance/types"` → `types.CashBank`. Applied across 93 imports
in 61 files.

Three exceptions, all forced rather than chosen:
- **`core/types` is always `coreTypes`.** `app/core` is imported by 147 files as `core`, so
  `core/types` can never take that name. 23 occurrences.
- **A file importing both a module body and its types keeps `<module>Types`.** Only
  composition roots legitimately do this — `exec/demo2.go`, `exec/invoicing_beta.go` and the
  two `tests/sample_records` generators — since after the boundary work no module body imports
  another.
- **The five files carrying a cross-module violation were left untouched** —
  `accounting/asset_api.go`, `asset_payment.go`, `inventory_expense.go`,
  `sales/sale_order_create.go`, `logistics/purchase-order-management.go`. Moving the shared
  functions into `*/types` removes their module-body import, and with it the collision, so
  converting them now would mean editing them twice.

**Rationale** — reading `logistics.InternalMovement` at a call site names the domain; reading
`logisticsTypes.InternalMovement` names the plumbing. The split at the module border is what
makes both halves read well: inside `package finance`, `finance.CashBank` looks like a
self-reference and forces an alias line into every file for nothing, while outside it is
unambiguous. The cost is that the local name no longer matches the package clause, so a reader
must look at the import block to resolve `finance.` — accepted because the alternative is 147
call sites naming a folder instead of a domain. `sales/sale_order_create.go` already used the
inside-module half of this convention, so it was half-adopted by accident before being decided.
