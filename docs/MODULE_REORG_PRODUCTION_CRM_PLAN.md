# Module Reorganization Plan — `production` and `crm`

**Status:** executed. Backend and frontend build clean; module boundaries, tables, route IDs and
controllers all validate; the four moved pages and the POS were verified in the running app.

Three extractions were **forced** during execution and are not in the plan below — each is
recorded in the relevant `RATIONALE.md`:

1. `sanitizeProviderSupplyRows` / `validateProviderSupplyRows` had to leave the `logistics` body
   for `logistics/types/product_supply_providers.go`, because `PostSupplyMaterial` moved out of
   the module that declared them. The validator now takes `companyID` instead of `*HandlerArgs`.
2. `imageConfigDigitFull` and the `cacheGroup*` IDs crossed the new `business`/`production` line,
   so they became `business/types.ImageConfigDigitFull` and
   `business/types.CacheGroupProducts/Brands/Categories`.
3. `CustomersView.svelte` became `domain-components/ClientProviderMaintainer.svelte` (its two
   pages now sit in different modules), which in turn forced `CountryCitiesService` out of
   `routes/business/branches-warehouses/` into `services/business/country-cities.svelte.ts` —
   a domain component cannot import from `routes/`.

## Goal

Carve two new modules out of `business` (and one route out of `logistics`) so the roadmap
features on the public site have a place to land:

- **Producción** — "Productos, servicios e insumos": recipes, production orders, BOM explosion,
  per-unit cost, process stages, waste control, recipe versioning, combos, services without stock.
- **Clientes (CRM)** — client file with history, call/visit log, opportunity funnel, sales-rep
  portfolio, receivables and credit line, campaigns, loyalty, client portal.

**This plan moves existing code only. No new feature is implemented, no API route string is
renamed, no DB table or column changes.** Table IDs, column names, API route names and access
IDs all stay exactly as they are, so no data migration and no route-ID regeneration.

## Decisions taken (confirmed with the user)

| Question | Decision |
| --- | --- |
| What goes into `production` | Product catalog **and** supplies & materials. **Not** the product-supply/replenishment config (stays in `logistics`), **not** the ecommerce product feed (stays in `business`). |
| What `business` keeps | Infrastructure only: Sites & Warehouses, initial-data, image assets + gallery, shared-lists, city locations, ecommerce product feed. Package keeps the name `business`. |
| `ProductsService` | Moves out of the route folder into `services/production/`. Same for `ClientProviderService` → `services/crm/`. |
| Suppliers | Clients go to CRM; the **suppliers route** moves under Logística. The `ClientProvider` table itself is owned by `crm/types`, and `logistics` imports it. |

---

## 1. Backend

### 1.1 New package `app/production/types` (L2)

Created from a split of `business/types/productos.go`, which today holds two unrelated domains.

| Moves to `production/types/product.go` | Stays, in a new `business/types/sites_warehouses.go` |
| --- | --- |
| `Product`, `ProductTable`, `GetSchema` (table 22) | `Site`, `SiteTable`, `GetSchema` (table 31) |
| `ProductPresentation`, `ProductProperty`, `ProductProperties` | `Warehouse`, `WarehouseTable`, `GetSchema` (table 39) |
| `WarehouseStockMin` (it is `Product.Stock`'s element type) | `WarehouseLayout`, `WarehouseLayoutBlock` |
| `ProductStatusInactive/Active/Supply` | |
| `FillCategoriesWithStock`, `SelfParse`, `GetTextSearchIndex` | |

`business/types/productos.go` is deleted after the split.

### 1.2 New package `app/production` (L4)

| New file | From |
| --- | --- |
| `production/products.go` | `business/products.go` (537 lines, unchanged but for the package + import lines) |
| `production/supply_material.go` | `logistics/supply-material-management.go` |
| `production/main.go` | new `ModuleHandlers` map |
| `production/RATIONALE.md` | new |

`ModuleHandlers` takes these six route strings **verbatim** — three from `business`, two from
`logistics`, plus the two image posts that live in `products.go`:

```
GET.products                 GET.p-products-ids           GET.p-product-text-search
POST.products                POST.product-image           POST.product-category-image
GET.supply-material          POST.supply-material
```

`production/supply_material.go` keeps importing `logistics/types` for `ProductSupply` and
`ProductSupplyProviderRow` — L4 → L2, which the boundary rule allows.

### 1.3 New package `app/crm` + `app/crm/types`

| New file | From |
| --- | --- |
| `crm/types/client_provider.go` | `business/types/client_provider.go` |
| `crm/types/client_provider_save.go` | `business/types/client_provider_save.go` (`SaveClientProviders`, called by `sales`) |
| `crm/client_provider.go` | `business/client_provider.go` |
| `crm/main.go` | `ModuleHandlers`: `GET.client-provider`, `GET.client-provider-ids`, `POST.client-provider` |
| `crm/RATIONALE.md` | new |

### 1.4 `business` after the split

Keeps `locations-warehouses.go`, `initial-data.go`, `shared-lists.go`, `images.go`,
`image_assets*.go`, `gallery-image.go`, `product-ecommerce.go`, `product-ecommerce-cron.go`.
`product-ecommerce.go` now imports `production/types` for `Product` — a body importing another
module's `types`, which is legal.

`business/main.go` keeps: `GET.locations-warehouses`, `GET.country-cities`, `POST.sites`,
`POST.warehouses`, `POST.initial-data`, `GET.shared-lists`, `POST.shared-lists`,
`GET.image-id-counter`, `GET.image-assets`, `GET.image-asset-text-search`, `POST.gallery-image`,
`GET.gallery-images`, `GET.p-products-ecommerce`.

`logistics/main.go` loses `GET.supply-material` and `POST.supply-material`; everything else stays.

### 1.5 Import rewrites

25 files import `app/business/types`. Per the naming rule, the alias becomes the owning module's
name. Files that need **two** aliases are marked ⚠.

| File | New import(s) |
| --- | --- |
| `accounting/asset_api.go` | `production "app/production/types"` |
| `accounting/inventory_expense.go` | `production` |
| `invoicing/sale_order_to_cpe.go` ⚠ | `production` + `crm` |
| `invoicing/sale_order_to_cpe_test.go` | `production` |
| `invoicing/issuer.go` | `business` (uses `Site` only — unchanged) |
| `logistics/product-stock-movement.go` | `production` |
| `logistics/product-supply-management.go` | `crm` |
| `sales/sale_order_create.go` ⚠ | `production` + `crm` |
| `webpage/webpage_pages.go` | `business` (uses `NewIDToID` — unchanged) |
| `agent/pagebuilder/tools.go` | `business` (image assets — unchanged) |
| `business/product-ecommerce.go` ⚠ | `types` (own, for `SharedListRecord`) + `production` |
| `business/initial-data.go`, `locations-warehouses.go`, `shared-lists.go`, `images.go`, `image_assets*.go`, `gallery-image.go` | unchanged |
| `exec/init.go`, `exec/demo.go`, `exec/demo2.go`, `exec/test_selects.go` ⚠ | mixed — L5, may import anything |
| `tests/fixtures/products.go` | `production` |
| `tests/sample_records/generate_erp_history.go` ⚠ | `production` + `crm` + `business` + bodies |
| `tests/sample_records/generate_sale_orders.go` ⚠ | same |
| `exec/controllers.generated.go` | regenerated |

### 1.6 Wiring and validation

1. `main-handlers.go` — add `app/crm` and `app/production` to imports and to `appHandlersModules`.
2. `scripts/boundaries/check_module_imports.go` — add `"app/crm"` and `"app/production"` to
   `moduleBodies`. Without this the checker treats them as unknown and fails.
3. `exec/controllers.generated.go` — regenerate; `Product` moves to `production.Product`,
   `ClientProvider` to `crm.ClientProvider`.
4. `backend/docs/MODULE_BOUNDARIES.md` — add both to the L4 table, and move the
   `business/types/client_provider_save.go` row of the "types holds business logic" table to
   `crm/types/client_provider_save.go`.

### 1.7 Backend verification

```
cd backend && go build ./... && go vet ./... && go test ./...
cd scripts && go run . check_module_imports && go run . check_tables
cd scripts && go run . generate_controllers      # regenerates exec/controllers.generated.go
cd scripts && go run . generate_route_ids --check  # must stay green: no route string changed
```

---

## 2. Frontend

### 2.1 Route folder moves

| From | To |
| --- | --- |
| `routes/business/products/` | `routes/production/products/` |
| `routes/logistics/supplies-materials/` | `routes/production/supplies-materials/` |
| `routes/business/customers/` | `routes/crm/customers/` |
| `routes/business/suppliers/` | `routes/logistics/suppliers/` |
| `routes/business/branches-warehouses/` | unchanged |

Each folder carries its `DOCUMENTATION.md`; the route paths quoted **inside** those files are
updated in the same pass.

### 2.2 Services extracted from route folders

Both of these are imported by 10–15 routes across `sales`, `logistics` and `accounting` today —
routes reaching into another route's folder. The move fixes that while we are touching the paths
anyway.

**`services/production/products.svelte.ts`** ← `routes/business/products/products.svelte.ts`
exports `ProductsService`, `IProduct`, `IProductPresentation`, `IProductProperties`,
`IProductProperty`, `IProductoImage`, `productImageName`, `mainProductImage`.
The `//STRUCT:negocio.Product` annotation becomes `//STRUCT:production.Product` so
`sync_struct_interfaces` still resolves it.

**`services/crm/client-provider.svelte.ts`** ← `routes/business/customers/customers.svelte.ts`
exports `ClientProviderService`, `ClientProviderType`, `PersonType`, `IClientProvider`,
`postClientProviders`.

Import rewrites (all become `$services/...` absolute, replacing the current mix of `$routes/...`
and `../../` relative paths):

- Products — `routes/sales/{sale_order_create/+page.svelte, sale_order_create/sale_order.svelte.ts, sale_orders_status/+page.svelte, sale_orders_charts/+page.svelte, sale_orders_charts/SaleOrdersChartsByProduct.svelte, sale_orders_charts/SaleOrdersChartsDailySummary.svelte, sales-report/+page.svelte, sale_planning/SalesPlanningMantainer.svelte}`, `routes/logistics/{warehouse-movements/+page.svelte, products-stock/ProductStockMovement.svelte, products-stock/PurchaseOrderEntry.svelte, purchase-management/ProductSupplyManagement.svelte, purchase-orders/{PurchaseOrderCreate,PurchaseOrderReport,ProductCardSearch}.svelte}`
- Client/provider — `routes/accounting/assets/+page.svelte`, `routes/sales/{sale_order_create/+page.svelte, sale_orders_status/+page.svelte, sales-report/+page.svelte}`, `routes/logistics/{products-stock/PurchaseOrderEntry.svelte, purchase-management/ProductSupplyManagement.svelte, purchase-orders/{PurchaseOrderCreate,PurchaseOrderReport,PurchaseOrderForm,ProductCardSearch}.svelte, suppliers/+page.svelte}`, `routes/production/supplies-materials/+page.svelte`

`frontend/webpage/` (the storefront, separate build) is checked for references and updated if any
exist.

### 2.3 `core/modules.ts` menu

`NEGOCIO` keeps only Sedes & Almacenes. Two new menu groups, inserted so that Producción sits
next to Negocio and CRM next to Comercial:

```
CONFIGURACIÓN (1) · SYSTEM (9) · NEGOCIO (2) · PRODUCCIÓN (10) · COMERCIAL (3)
· CLIENTES CRM (11) · LOGÍSTICA (4) · FINANZAS (5) · TIENDA/WEB (7) · CONTABILIDAD (8)
```

| Menu | id | minName | Options |
| --- | --- | --- | --- |
| Business \| Negocio | 2 | NEG | Sedes & Almacenes |
| **Production \| Producción** | **10** | **PRD** | Productos → `/production/products`; Insumos & Materiales → `/production/supplies-materials` |
| **Clients (CRM) \| Clientes (CRM)** | **11** | **CRM** | Clientes → `/crm/customers` |
| Logistics \| Logística | 4 | LOG | existing four + **Proveedores → `/logistics/suppliers`** (loses Suministros) |

Menu ids 10 and 11 are unused today (1,2,3,4,5,7,8,9 are taken).

### 2.4 `backend/access_list.yml`

Access **ids are permanent and are not renumbered** — only their `group` and `frontend_routes`
change. Two new groups:

```yaml
  - id: 9
    name: "Producción"
  - id: 10
    name: "Clientes (CRM)"
```

| Access | id | `group` | `frontend_routes` |
| --- | --- | --- | --- |
| Productos | 8 | 2 → **9** | `business/products` → `production/products` |
| Suministros | 30 | 4 → **9** | `logistics/supplies-materials` → `production/supplies-materials` |
| Clientes | 9 | 2 → **10** | `business/customers` → `crm/customers` |
| Proveedores | 26 | 2 → **4** | `business/suppliers` → `logistics/suppliers` |

`backend_apis` lists are untouched — no route string changes.

### 2.5 Frontend verification

```
cd frontend && bun run check && bun run build
```

Plus an `agent-browser` pass over `/production/products`, `/production/supplies-materials`,
`/crm/customers`, `/logistics/suppliers` and the POS (`/sales/sale_order_create`, the heaviest
consumer of both moved services) to confirm the pages still render and load data.

---

## 3. RATIONALE.md updates

- `backend/production/RATIONALE.md` (new) — why the catalog left `business`; why supplies came
  from `logistics` (a supply *is* a `Product` row at `Status=2`, so the catalog owns it) while the
  replenishment config did not (that is purchasing).
- `backend/crm/RATIONALE.md` (new) — why one `ClientProvider` table serves two menus in two
  different modules, and why the suppliers *route* sits in Logística while the *table* is owned
  by `crm/types`.
- `backend/logistics/RATIONALE.md` — append the two departures.
- `frontend/routes/production/RATIONALE.md`, `frontend/routes/crm/RATIONALE.md` (new) — why
  `ProductsService` and `ClientProviderService` now live in `services/` rather than a route folder.

## 4. Explicit non-goals

- No new feature from either roadmap list is built.
- No DB table, column, table ID or API route string changes.
- `logistics/product-supply-management.go` and `routes/logistics/purchase-management/` stay put.
- `business/product-ecommerce*.go` stays in `business`.
- Old route URLs are **not** redirected. Pre-alpha; `/business/products` simply stops existing.

## 5. Risk notes

- The single riskiest step is the `business/types/productos.go` split: `Product` and
  `Site`/`Warehouse` share the file, and 25 files import the package. Mitigated by doing the
  split first and building before anything else moves.
- `check_module_imports` will fail loudly until step 1.6.2 registers the new module bodies —
  expected, not a regression.
- Frontend import rewrites are mechanical but wide (~25 files); `bun run check` is the gate.
