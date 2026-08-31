# Module Boundaries

**The rule: a module body may never import another module body.**

Shared code crosses module lines through `<module>/types` only. Enforced by
`cd scripts && go run . check_module_imports` (also a "Validar Límites de Módulos" entry in
`./deploy.sh`). Go cannot catch this at compile time — it is perfectly happy with
`accounting` importing `finance` — so the check is the only thing keeping it true.

## Layers

Imports flow one way. A package may import its own layer and anything below it.

| Layer | Packages | May import |
| --- | --- | --- |
| **L0 Foundation** | `db`, `libs/*`, `genix-orm`, `facturago`, `fareward/go` | nothing from `app/` |
| **L1 Core** | `core`, `core/types` | L0 |
| **L2 Contracts** | `<module>/types` | L0, L1, other `<module>/types` — **and nothing else, ever** |
| **L3 Infra** | `cloud` | L0–L2 |
| **L4 Module bodies** | `accounting`, `agent`, `business`, `config`, `crm`, `finance`, `invoicing`, `logistics`, `production`, `sales`, `security`, `webpage` | L0–L3, plus their own subpackages |
| **L5 Composition roots** | root `main`, `exec`, `tests/*`, `scripts` | anything |

A module's own subpackages (`agent/llm`, `agent/pagebuilder`) are internal to it and may be
imported freely from within the same module.

## Why L2 is the load-bearing rule

`<module>/types` is imported by everyone, so it is the one place shared code can live without
creating a module-to-module edge. That only works while it stays a **leaf**: if
`finance/types` could import `logistics`, every module could route around the rule through it.
The checker enforces the leaf property, and there is deliberately no escape hatch — when a
function in `finance/types` needs `logistics` proper, the design is wrong, not the rule.

## Consequence: `types` holds business logic

Some `*/types` packages hold real logic, not just declarations:

| Package | Holds | Called by |
| --- | --- | --- |
| `finance/types/cash_movement_apply.go` | `ApplyCashBankMovement`, `GetCaja` — the cash ledger writer | `sales`, `logistics`, `accounting`, `finance` |
| `logistics/types/stock_movement_apply.go` | `ApplyMovimientos`, `RecalcProductStockByMovements` — the stock engine | `sales`, `accounting`, `logistics`, `exec` |
| `crm/types/client_provider_save.go` | `SaveClientProviders` | `sales`, `crm` |
| `logistics/types/product_supply_providers.go` | `SanitizeProviderSupplyRows`, `ValidateProviderSupplyRows` | `logistics`, `production` |
| `business/types/image_assets_agent.go` | `FindImageCandidates` | `agent/pagebuilder` |

The folder name undersells these. Each carries a file-level comment saying why it lives there,
and the filenames are specific enough to grep (`fd stock_movement`). This is the accepted cost
of not having a fourth `shared` layer.

## Import naming

Every `<module>/types` folder declares `package types`, so every import needs a local name:

- **Outside the owning module** — alias to the module name:
  `finance "app/finance/types"` → `finance.CashBank`, `logistics.ApplyMovimientos(...)`.
  The call site names the domain, not the plumbing. A file that needs two of them takes two
  aliases — `production.Product` next to `crm.ClientProvider` in `sales/sale_order_create.go`.
- **Inside the owning module** — no alias, use the package name:
  `import "app/finance/types"` → `types.CashBank`.

Three exceptions, all forced:

- `core/types` is always `coreTypes`. `app/core` is imported by ~150 files as `core`, so
  `core/types` can never take that name.
- A file importing both a module body **and** its types keeps `<module>Types`. Only
  composition roots legitimately do this.
- Generated files follow whatever the generator emits.

## Where shared code goes

1. **A type, constant or enum** → `<module>/types`, next to the table it describes. A value
   vocabulary belongs to whoever owns the column: `CashMovementType` lives in `finance/types`
   because `CashBankMovement` does, even though `logistics`, `sales` and `accounting` all write
   it.
2. **A function two modules need** → `<module>/types`, in a file named for the job, with a
   file-level comment explaining why it is not in the module body. It must depend on nothing
   but `core`, `db` and its own package.
3. **Neither fits** → the boundary is telling you the ownership is wrong. Work out which module
   owns the behaviour before reaching for a new package.
