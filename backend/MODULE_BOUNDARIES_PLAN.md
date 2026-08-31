# Backend Module Boundaries — Analysis & Refactor Plan

Status: **DONE.** All eight steps executed across seven commits (`ca28b4db` … `25aa19de`), plus
`4613a2c` in `genix-orm`. All six violations closed and enforced by
`cd scripts && go run . check_module_imports`. The rule now lives in
`backend/docs/MODULE_BOUNDARIES.md`; decisions in `backend/RATIONALE.md`. This file is the
historical plan — the doc is the reference.

Goal: a module body may never import another module body. Only `core`, `db`, generic
layers, and `*/types` cross module lines. Anything genuinely shared moves into
`<module>/types`.

**Decisions taken (§8 resolved):** shared code goes in `types`, no `shared` packages (Q1);
the ledger types become a named enum (Q2 — **the ORM blocker is fixed, §3.4**); the stock recalc moves
with the engine (Q3); `agent/webpage` → `agent/pagebuilder` (Q4); the alias sweep is one
commit (Q5).

---

## 1. The layering rule

Imports flow one way. A package may import its own layer's siblings only where noted.

| Layer | Packages | May import |
| --- | --- | --- |
| **L0 Foundation** | `db`, `libs/*`, `genix-orm`, `facturago` | nothing from `app/` |
| **L1 Core** | `core`, `core/types`, `core/auth-limiter` | L0 |
| **L2 Contracts** | `<module>/types` | L0, L1, other `<module>/types` |
| **L3 Infra** | `cloud` (`system` relocates to L0, §5.3) | L0–L2 |
| **L4 Module bodies** | `accounting`, `agent`, `business`, `config`, `finance`, `invoicing`, `logistics`, `sales`, `security`, `webpage` | L0–L3 + own subpackages. **Never** another module body |
| **L5 Composition roots** | root `main`, `exec`, `tests/*`, `scripts` | anything |

**There is no `shared` layer.** Everything cross-module lives in L2 — see §3.

L2 is the load-bearing rule: a `<module>/types` package may import `db`, `core`, and
other `<module>/types`, and **nothing else, ever**. It is already true today (every
`*/types` imports only `app/db`, sometimes `app/core`) and the §7 check enforces it. The
moment a function in `finance/types` needs `logistics` proper, the scheme has no escape
hatch — that is the point, and it is why L2 stays a strict leaf-plus-siblings layer.

---

## 2. Current violations (complete list)

Six import edges break the rule. There are no import *cycles* today — Go would have
rejected them — so every fix is a move, not an inversion.

| # | From | To | Site | Symbols used |
| --- | --- | --- | --- | --- |
| V1 | `accounting` | `finance` | `accounting/asset_payment.go:7` | `finance.ApplyCashBankMovement`, `finance.GetCaja` |
| V2 | `accounting` | `logistics` | `accounting/asset_api.go:8`, `accounting/inventory_expense.go:8` | `logistics.ApplyMovimientos`, `logistics.SupplyProductStatus` |
| V3 | `logistics` | `finance` | `logistics/purchase-order-management.go:6` | `finance.ApplyCashBankMovement` |
| V4 | `sales` | `business`, `finance`, `logistics` | `sales/sale_order_create.go:4,8,10` | `business.SaveClientProviders`, `finance.ApplyCashBankMovement`, `logistics.ApplyMovimientos` |
| V5 | `agent/webpage` | `business` | `agent/webpage/tools.go:12` | `business.FindImageCandidates`, `business.AgentImageCandidate` |
| V6 | `config` | `system` | `config/system_memory_packages.go:5`, `config/system_metrics_sse.go:5` | `servermetrics.CollectGoHeapPackageReport`, `servermetrics.NewServerMetricsCollector` |

Legal today and staying legal: `invoicing → sales/types`, `security → finance/types`,
`business → finance/types`, `logistics → business/types`, `accounting → business/types`,
`cloud → security/types` + `config/types`. All types-only.

`exec`, `tests/sample_records` and root `main`/`main-handlers.go` import module bodies
freely. That is correct — they are the composition roots and are exempt.

### The boundary is already costing us

Two places paid for the missing seam by **copying code**:

- `sales/sale_order_create.go:412` — `packProductStockIDForSale`, a hand copy of
  `logistics.packProductStockID` (`logistics/product-stock-movement.go:282`). Its comment
  claims a circular dependency: *"logistica already depends on comercial via some
  paths"*. **That is stale** — `logistics` imports only `business/types`, never
  `business`. Two copies of an ORM key-packing formula that must agree with the schema.
- `accounting/asset_api.go:15` — `const supplyProductStatus int8 = 2`, a copy of
  `logistics.SupplyProductStatus` (`logistics/supply-material-management.go:17`), in a
  file that *also* imports `logistics`. Both the copy and the import, in the same file.

Fixing the seam deletes both duplicates.

---

## 3. Where each shared symbol goes

Everything lands in the owning module's existing `types` package. No new packages.

**Why this works with no call-site churn.** Combined with the §4 alias convention,
consumers import `app/logistics/types` **as** `logistics`, so `accounting` keeps writing
`logistics.ApplyMovimientos(req, movements)` — byte-for-byte the same call as today. Only
the import line changes. Same for `finance.ApplyCashBankMovement` and
`business.SaveClientProviders`: ~15 call sites across 4 modules, zero edits below the
import block.

**Tooling verified safe.** `types/` is a codegen input, so this needed checking:
- `scripts/generators/sync_struct_interfaces.go:98` walks every `.go` file whose parent
  folder is named `types` — but `parseBackendStructFile` only reacts to `*ast.TypeSpec`
  wrapping an `*ast.StructType`. Functions, constants and named non-struct types are
  skipped silently. Adding logic files does not feed the frontend interface generator.
- `scripts/validation/check_tables.go` loads `./...` and only reacts to structs embedding
  `db.TableStruct` plus their `GetSchema` methods. Functions are ignored.

**The cost, stated plainly.** Two things get worse and we are accepting them:
1. `logistics/types` goes from 542 lines of struct definitions to ~1000, half of it the
   heaviest business logic in the module, under a folder named `types`. Mitigation:
   dedicated, self-describing filenames (below) so `fd stock_movement` still finds it.
2. `business/types` is imported by 8 packages that only want `Product`. They will now
   compile the client-provider upsert and the text-search image lookup. Harmless in a
   single binary, but it is real coupling growth.

### 3.1 `finance/types` — the cash ledger writer

New file `finance/types/cash_movement_apply.go`:

| Symbol | From |
| --- | --- |
| `ApplyCashBankMovement` | `finance/cash_bank_movement.go` (whole file, 100 lines — delete it) |
| `GetCaja` | `finance/shared.go` (whole file, 23 lines — delete it) |

Verified deps: `core`, `db`, `finance/types` (becomes local). Zero references back into
`finance`. `finance/types` gains an `app/core` import, which `business/types`,
`logistics/types` and `sales/types` already have.

Fixes V1 (cash half), V3, V4 (cash third).
Consumers: `finance` (`expenses.go`, `cash_banks.go` — now `types.ApplyCashBankMovement`),
`accounting`, `logistics`, `sales`, `tests/sample_records`.

### 3.2 `logistics/types` — the stock movement engine

New file `logistics/types/stock_movement_apply.go` — `logistics/product-stock-movement.go:262`–end
moves as one block, plus all of `logistics/stock-reprocess.go` (Q3: yes):

| Symbol | Line | Stays private? |
| --- | --- | --- |
| `applyMovimientosLockByCompany`, `applyMovimientosLockMapMu` | 262–263 | yes |
| `getApplyMovimientosCompanyLock` | 265 | **yes** — Q3 keeps it private |
| `packProductStockID` | 282 | **yes** — and the `sales` copy dies |
| `maxProductStockLastPrices`, `appendProductStockLastPrice` | 286–288 | yes |
| `ApplyMovimientos` | 310 | exported |
| `resolveLotIDsForMovements` | 609 | yes |
| `RecalcProductStockByMovements` | `stock-reprocess.go:14` | exported (Q3 — moves so the lock stays private) |

Deleting `sales/sale_order_create.go:412` `packProductStockIDForSale` is part of this step,
and `validateSaleStock` switches to the (now shared, still-private-to-the-package) helper
via a thin exported accessor or by calling `ApplyMovimientos`' own validation — decide when
the code is in front of us; it is one call site.

Fixes V2 (movement half), V4 (stock third).

### 3.3 `business/types` — client upsert + agent image lookup

Two new files:

| File | Contents | From |
| --- | --- | --- |
| `business/types/client_provider_save.go` | `SaveClientProviders` + private `companyRegistryNumberPattern`, `isValidEmailAddress` | `business/client_provider.go:87`, `:13`, `:214` — none used elsewhere |
| `business/types/image_assets_agent.go` | `AgentImageCandidate`, `FindImageCandidates` + private `imageAssetsPublicBaseURL`, `imageCategoryNames`, `imageAssetURL` | `business/image_assets_agent.go` (whole file, 91 lines) |

`imageAssetCategoryGroupID` (`business/image_assets_sync.go:20`) is a schema constant — the
group partition image assets live in — and moves into `business/types` too. Its four other
readers in `business` (`image_assets.go:51,73,113`, `image_assets_sync.go:143,165,311`)
switch to the `types` reference.

Caveat: `AgentImageCandidate` **is** a struct in a `types` folder, so
`sync_struct_interfaces` will now emit a TS interface for it if a frontend route declares
one by that name. Nothing does today, and it is inert unless matched — but it is the one
new behaviour this step introduces.

Fixes V5, V4 (client third).

### 3.4 The ledger type enum (Q2) — and its blocker

`CashBankMovement.Type` is currently **four private copies in four modules**, each with a
comment saying it mirrors the frontend's `cajaMovimientoTipos`:

| Value | Meaning | Declared at |
| --- | --- | --- |
| 6 | supplier payment | `logistics/purchase-order-management.go:21` (`cajaMovimientoTipoPagoProveedor`) |
| 8 | sale collection | `sales/sale_order_create.go:158` — **a bare literal `Type: 8`**, no constant at all |
| 9 | expense payment | `finance/expenses.go:26` (`movementTypeExpensePayment`) |
| 10 | asset payment | `accounting/asset_payment.go:16` (`movementTypeAssetPayment`) |

`accounting/asset_payment.go:12` documents why the vocabulary must be global: the type is
the discriminator that keeps asset payments and expense payments apart when they collide
on `DocumentID`. Split four ways, nothing stops the next module reusing a taken number.

Decision (Q2): one named type in `finance/types`.

```go
type CashMovementType int8

const (
    CashMovementTypeSupplierPayment CashMovementType = 6
    CashMovementTypeSaleCollection  CashMovementType = 8
    CashMovementTypeExpensePayment  CashMovementType = 9
    CashMovementTypeAssetPayment    CashMovementType = 10
)
```

**The ORM blocker is fixed** (`genix-orm`, uncommitted). `db.Col[T, E any]` has no constraint
so a named type always compiled, but six paths dispatched on a type switch over `any`, which
a named type never matches. Confirmed empirically by running the new tests against the
unpatched code:

| Site | Did |
| --- | --- |
| `db/convert.go` `ToInt64`/`ToFloat64`/`ToFloat32` | returned **0** — the write path for every accessor in `db/accessors.go` and the partition key in `db/table.go` |
| `scylla/helpers.go` `convertToInt64`/`convertToInt32` | printed "not an integer", returned **0** |
| `scylla/converter.go` `valueToCSVBase64` | emitted decimal text `"10"` where readers expect base64 `"a"` — **a corrupt value, not a missing one** |
| `scylla/converter.go` `makeNumericSlice` | its `~int8` constraint accepts `[]CashMovementType`, then the inner switch appended nothing: 4 elements encoded to `",,,"` |
| `scylla/helpers.go` `HashInt` | hashed the decimal text, so a named type hashed differently from the type it wraps |
| `scylla/merge.go` `isNonPositiveNumericValue` | returned false for a named zero, flipping the insert-vs-update decision |

Fix: one `db.NormalizeNamedNumeric(value any) (any, bool)` unwraps a named numeric into its
plain underlying type, called from each switch's **default branch only** — so exact-type fast
paths stay reflection-free, and the branches it touches were only ever reached by values the
code already handled wrongly. Normalizing to the plain type rather than widening to int64 is
what keeps `HashInt` width-stable, so no persisted hash changes. Termination comes from a
`PkgPath() == ""` guard: a predeclared type never normalizes, so recursion runs at most once.
Rationale recorded in `genix-orm/scylla/RATIONALE.md`.

Verified: `genix-orm` all three modules build + test green, `backend` builds + vets, and
`check_tables` passes (53 table pairs). Step 3 can now type the column
`db.Col[CashBankMovementTable, CashMovementType]` directly, with `check_tables`'
`types.Identical` satisfied because both sides are the named type.

### 3.5 `SupplyProductStatus` → `business/types`

V2's other half. It is `Product.Status == 2`, and `Product` lives in `business/types`.
Move it there as `ProductStatusSupply`; delete the `accounting` copy
(`asset_api.go:15` `supplyProductStatus`, plus its "mirrors" comment at :13) and the
`logistics` declaration (`supply-material-management.go:17`), repointing
`supply-material-management.go:37,132` and `accounting/asset_api.go:91`,
`inventory_expense.go:74`.

## 4. The `*/types` import-alias convention

Today 132 aliased imports of a `*/types` package, in five different styles:

```
26  businessTypes "app/business/types"      7  coretypes "app/core/types"
17  coreTypes "app/core/types"              5  s "app/webpage/types"
15  financeTypes "app/finance/types"        2  s "app/sales/types"
14  logisticsTypes "app/logistics/types"    1  s "app/business/types"
12  configTypes "app/config/types"          1  sales "app/sales/types"
10  invoicingTypes "app/invoicing/types"    1  types "app/core/types"
 9  salesTypes "app/sales/types"            1  webpageTypes "app/webpage/types"
 7  accountingTypes "app/accounting/types"  1  agentTypes "app/agent/types"
 4  securityTypes "app/security/types"
```

`s` is the worst offender — `business/client_provider.go` reads `s.ClientProvider`,
which is ungrepable.

Every `*/types` folder declares `package types`, so **some** name is always needed.
Proposed rule, two halves:

- **Outside the owning module** — alias to the module name:
  `finance "app/finance/types"` → `finance.CashBank`, `business.Product`,
  `logistics.InternalMovement`. This is what you asked for, and it reads as the domain
  noun rather than as plumbing.
- **Inside the owning module** — no alias; use the package name `types`:
  `import "app/finance/types"` → `types.CashBank`. `sales/sale_order_create.go` already
  does exactly this (`types.SaleOrder`), so the convention is half-adopted.

Why split it: inside `package finance`, writing `finance.CashBank` is legal Go but reads
as a self-reference, and it forces an alias line into every file in the module for no
gain. Outside, `finance.CashBank` is unambiguous and grepable.

This convention is what makes §3 free: because consumers alias `app/logistics/types` to
`logistics`, moving `ApplyMovimientos` into the types package leaves every call site
reading `logistics.ApplyMovimientos(...)` exactly as it does today. §4 and §3 are one
design, not two — which is why step 6 should land *before* the moves, not after (see §6).

Churn: ~132 import lines and every qualified reference under them. Entirely mechanical
(`gofmt -r` or sed per package, then `go build`), but it touches almost every file, so it
should land as its own commit, separate from the moves.

---

## 5. Secondary findings

### 5.1 `sales` imports `app/sales/types` unaliased, everything else aliases it
Not a violation, just the inconsistency §4 resolves. Keep the `sales` style, spread it.

### 5.2 `agent/webpage` collides with the top-level `webpage` module
Two unrelated packages named `webpage`: `app/agent/webpage` (the page-builder agent loop)
and `app/webpage` (the storefront module). Nothing imports both today, so it compiles —
but the names are one file away from needing an alias, and grep for "webpage" spans two
domains. **Decided (Q4): rename `agent/webpage` → `agent/pagebuilder`.** Importers:
`agent/main.go` and the package's own files; `agent/webpage/lint_test.go` and
`prompts.go` are already modified in the working tree, so this rename should land before
further edits pile up there.

### 5.3 `system/` is not a module
`system/` has **zero** `app/` imports, no `ModuleHandlers`, and holds only OS metric
collection (`ServerMetricsCollector`, `CollectGoHeapPackageReport`, snapshot structs). It
is a generic utility that happens to sit at module level, which is why its only consumer
imports it as `servermetrics "app/system"` (V6) — an alias that exists purely to correct
the folder name.

→ Move to `libs/servermetrics/` (L0). V6 stops being a violation by definition, the alias
disappears, and `system` stops looking like a domain module. `libs/` over `core/` because
`libs/` is for non-business generics, which this is — it reads the OS, not the domain.

### 5.4 `config` is a module body, `cloud` is infra
`config` declares `ModuleHandlers` and is imported only by `main-handlers.go` → L5.
`cloud` declares no handlers and is imported by `business`, `invoicing`, `security`,
`webpage`, `config`, `exec` → L3 infra. Both classifications already hold; recording them
so the enforcement check encodes the right layer.

---

## 6. Execution order

Each step ends with `cd backend && go build ./... && go vet ./...`, plus
`go test ./<pkg>/...` where tests exist (`accounting`, `invoicing`, `business`, `finance`
have them) and `cd scripts && go run . check_tables` for steps 2 and 4. Every step is
independently committable; **one commit each, straight to `main`.**

The alias sweep moves to **step 1**. It has to: §3 depends on consumers referencing
`app/logistics/types` as `logistics`, so doing the sweep first means steps 4–6 are pure
import-line swaps with no call-site edits. Doing it last would mean touching every call
site twice.

| Step | Change | Scope |
| --- | --- | --- |
| **1** | Apply the §4 alias convention repo-wide: outside the owning module alias to the module name, inside it use bare `types`. Kills `s`, `coretypes`, `financeTypes`, et al. | ~132 import blocks + their qualified references. One mechanical commit, no behaviour change |
| **2** | `system/` → `libs/servermetrics/`. **Fixes V6.** | 3 files moved, 2 importers |
| **3** | Ledger enum → `finance/types` (§3.4) + `ProductStatusSupply` → `business/types` (§3.5). Delete the 2 duplicate constants and the bare `Type: 8`. Includes the ORM fix if you pick (a). | `finance/types`, `business/types`, 8 call sites, possibly a `genix-orm` commit |
| **4** | `ApplyCashBankMovement` + `GetCaja` → `finance/types`. **Fixes V1 (cash), V3, V4 (cash).** | 2 files deleted/moved, 6 importers |
| **5** | Stock engine + recalc → `logistics/types`. Delete `packProductStockIDForSale`. **Fixes V2 (movement), V4 (stock).** | `product-stock-movement.go` split, `stock-reprocess.go` moved, 3 importers |
| **6** | `SaveClientProviders` + agent image lookup → `business/types`; `imageAssetCategoryGroupID` with them. **Fixes V5, V4 (client).** | 2 files, 5 importers |
| **7** | `agent/webpage` → `agent/pagebuilder` (Q4). | 1 package + `agent/main.go` |
| **8** | Add the `check_module_imports` script (§7) + `backend/docs/MODULE_BOUNDARIES.md`. | new script + doc |

After step 6 all six violations are closed. Steps 7–8 are independent of the rest.
Each step also appends its entry to the relevant `RATIONALE.md` (`finance/`, `logistics/`,
`business/`, `accounting/`) per CLAUDE.md §1 — the boundary rule itself goes in the new
`backend/docs/MODULE_BOUNDARIES.md`.

## 7. Enforcement

A plan that is not checked decays. Add `scripts` dispatcher command
`check_module_imports` (per `scripts/SCRIPTS.md`):

1. `go list -json ./...` over `backend/` — gives each package's import path and `Imports`.
2. Classify every package into L0–L6 by path (table in §1, as a literal map — grepable,
   no clever inference).
3. Fail with the offending `package → package` edge and the layer rule it broke.

Wire it into `deploy.sh` alongside `check_tables`, and document the rule in
`backend/docs/MODULE_BOUNDARIES.md` so it is readable without reading the checker.

---

## 8. Decisions taken

| # | Question | Decision |
| --- | --- | --- |
| Q1 | `shared` packages vs. `types` | **`types`.** No `shared` packages anywhere. Verified feasible — the two codegen tools that read `types/` folders only react to struct TypeSpecs and `TableStruct` embeds, so logic files are inert to them (§3). Accepted cost: `logistics/types` roughly doubles in size and `business/types` grows transitive weight for its 8 importers. |
| Q2 | Ledger constants: plain `int8` or a named type | **Named type `CashMovementType`**, option (a) — the ORM is fixed (§3.4). `db.NormalizeNamedNumeric` + 6 fallback call sites + 2 test files in `genix-orm`, uncommitted. |
| Q3 | Does `RecalcProductStockByMovements` move with the engine | **Yes.** It moves into `logistics/types`, which keeps the company lock and `packProductStockID` private. |
| Q4 | Rename `agent/webpage` | **Yes** → `agent/pagebuilder`. |
| Q5 | Alias sweep: one commit or incremental | **One commit** — and promoted to step 1, since §3's zero-churn property depends on it landing first. |

### Status: complete

| Step | Commit |
| --- | --- |
| ORM fix (Q2 prerequisite) | `4613a2c` in `genix-orm` |
| 1 — alias sweep, 93 imports / 61 files | `ca28b4db` |
| 2 — `system/` → `libs/servermetrics/` (V6) | `1a7ffef6` |
| 3 — ledger enum + product status | `c1d5e4e5` |
| 4 — cash ledger → `finance/types` (V1, V3, V4) | `b37d3776` |
| 5 — stock engine → `logistics/types` (V2, V4) | `77260796` |
| 6+7 — `business/types`, `agent/pagebuilder` (V4, V5) | `ea44c600` |
| 8 — `check_module_imports` + docs | `25aa19de` |

Verified: `go build ./...`, `go vet ./...`, `gofmt` clean, full backend test suite, `check_tables`
(53 pairs), `check_module_imports` (43 packages), `scripts` tests, `genix-orm` all three modules.

One pre-existing test failure is untouched: `agent/ragdocs`
`TestParseExamplesAndBuildStableChunks` reports stale evidence for
`frontend/core/modules.ts` in `finance/cash-banks/DOCUMENTATION.md`. It fails identically at
`b9dbe1cf` — `modules.ts` last changed in `7a044df8`, after the doc recorded its hash in
`287d1005`. 127 further evidence entries were already stale and were deliberately left alone;
the 46 that this refactor invalidated were repointed and refreshed.
