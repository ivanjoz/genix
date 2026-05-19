# Refactor Plan: Rename Spanish Fields to English

## Overview

All Go struct fields and sub-struct names across the registered DB controllers must be renamed from Spanish to English. The corresponding TypeScript interfaces and Svelte components must be updated in lockstep.

**Pre-alpha rule applies:** no backwards-compatibility shims. DB column names will change where no explicit `db:` tag overrides the ORM default (snake_case of field name). Accept the schema drift.

---

## Open Questions (resolve before executing)

All questions resolved:

1. ~~`Sbn*` prefix~~ — `Sbn` = SubUnidad. New prefix `Sbu`, English suffixes.
2. ~~`Parametros` struct name~~ — rename to `Parameters` / `ParametersTable`. Table name string stays `"parametros"` (hardcoded in `GetSchema`). Update controller registration in `exec/init.go`.
3. ~~`MovimientoInterno` struct name~~ — rename to `InternalMovement`.
4. ~~`InnerStruct`~~ — keep name as-is; only rename Spanish fields inside (`Hola`→`Greeting`, `Hola2`→`Greeting2`).
5. ~~`CityLocation` computed fields~~ — `Departamento`→`Department`, `Provincia`→`Province`, add `Distrito`→`District` (hierarchy level 3, missing from struct).

---

## Field Rename Mappings

### `backend/config/types/empresas.go`

| Old | New | DB column impact |
|-----|-----|-----------------|
| `Nombre` | `Name` | `nombre` → `name` |
| `RazonSocial` | `LegalName` | `razon_social` → `legal_name` |
| `Telefono` | `Phone` | `telefono` → `phone` |
| `Representante` | `Representative` | no change (same snake_case) |
| `Direccion` | `Address` | `direccion` → `address` |
| `Ciudad` | `City` | `ciudad` → `city` |
| `EmailVerificado` | `EmailVerified` | `email_verificado` → `email_verified` |
| `TelefonoVerificado` | `PhoneVerified` | `telefono_verificado` → `phone_verified` |
| `NotificacionEmail` | `NotificationEmail` | `notificacion_email` → `notification_email` |
| `CulqiConfig.LlaveLive` | `CulqiConfig.KeyLive` | (nested, no direct col) |
| `CulqiConfig.LlavePubLive` | `CulqiConfig.PubKeyLive` | (nested) |
| `CulqiConfig.LlaveDev` | `CulqiConfig.KeyDev` | (nested) |
| `CulqiConfig.LlavePubDev` | `CulqiConfig.PubKeyDev` | (nested) |
| `CompanyPub.Nombre` | `CompanyPub.Name` | (not a DB col) |

### `backend/config/types/parametros.go`

| Old | New | DB column impact |
|-----|-----|-----------------|
| `Grupo` | `Group` | `grupo` → `group` |
| `Valor` | `Value` | `valor` → `value` |
| `ValorInt` | `ValueInt` | `valor_int` → `value_int` |
| `Valores` | `Values` | `valores` → `values` |
| _(struct)_ `Parametros` | `Parameters` | table name stays `"parametros"` (hardcoded in `GetSchema`) |
| _(struct)_ `ParametrosTable` | `ParametersTable` | — |

Also update `exec/init.go`: `configTypes.Parametros` → `configTypes.Parameters`.

### `backend/core/types/users.go`

| Old | New | DB column impact |
|-----|-----|-----------------|
| `Nombres` | `FirstName` | `nombres` → `first_name` |
| `Apellidos` | `LastName` | `apellidos` → `last_name` |
| `PerfilesIDs` | `ProfileIDs` | `perfiles_ids` → `profile_i_ds` ⚠️ add explicit `db:"profile_ids"` tag |
| `AccesosNivelIDs` | `AccessLevelIDs` | `accesos_nivel_ids` → needs explicit `db:"access_level_ids"` tag |
| `Cargo` | `JobTitle` | `cargo` → `job_title` |
| `DocumentoNro` | `DocumentNumber` | `documento_nro` → `document_number` |

### `backend/finance/types/cajas.go`

#### `CashBank`
| Old | New | DB column impact |
|-----|-----|-----------------|
| `Tipo` | `Type` | `tipo` → `type` |
| `SedeID` | `SiteID` | `sede_id` → `site_id` |
| `Nombre` | `Name` | `nombre` → `name` |
| `Descripcion` | `Description` | `descripcion` → `description` |
| `MonedaTipo` | `CurrencyType` | `moneda_tipo` → `currency_type` |
| `CuadreFecha` | `ReconciliationDate` | `cuadre_fecha` → `reconciliation_date` |
| `CuadreSaldo` | `ReconciliationAmount` | `cuadre_saldo` → `reconciliation_amount` |
| `SaldoCurrent` | `CurrentAmount` | `saldo_current` → `current_amount` |

#### `CashBankMovement`
| Old | New | DB column impact |
|-----|-----|-----------------|
| `CajaID` | `CashBankID` | `caja_id` → `cash_bank_id` — ⚠️ also used in `KeyIntPacking` |
| `CajaRefID` | `CashBankRefID` | `caja_ref_id` → `cash_bank_ref_id` |
| `DocumentoID` | `DocumentID` | `documento_id` → `document_id` — also in Indexes |
| `Tipo` | `Type` | `tipo` → `type` |
| `SaldoFinal` | `FinalAmount` | `saldo_final` → `final_amount` |
| `Monto` | `Amount` | `monto` → `amount` |

#### `CashReconciliation`
| Old | New | DB column impact |
|-----|-----|-----------------|
| `CajaID` | `CashBankID` | `caja_id` → `cash_bank_id` |
| `Tipo` | `Type` | `tipo` → `type` |
| `MovimientoID` | `MovementID` | `movimiento_id` → `movement_id` |
| `SaldoSistema` | `SystemAmount` | `saldo_sistema` → `system_amount` |
| `SaldoReal` | `ActualAmount` | `saldo_real` → `actual_amount` |
| `SaldoDiferencia` | `DifferenceAmount` | `saldo_diferencia` → `difference_amount` |

#### `InternalCashMovement` (non-DB helper struct)
| Old | New |
|-----|-----|
| `CajaID` | `CashBankID` |
| `CajaRefID` | `CashBankRefID` |
| `Tipo` | `Type` |
| `Monto` | `Amount` |
| `SaldoFinal` | `FinalAmount` |

#### `SaleProduct` (non-DB helper struct in cajas.go)
| Old | New |
|-----|-----|
| `Cantidad` | `Quantity` |
| `Monto` | `Amount` |

### `backend/logistics/types/product-stock-movement.go`

#### `WarehouseProductMovement`
| Old | New | DB column impact |
|-----|-----|-----------------|
| `PresentacionID` | `PresentationID` | `presentacion_id` → `presentation_id` — also in Indexes |
| `Tipo` | `Type` | `tipo` → `type` — also in multiple Indexes |

#### `MovimientoInterno` → `InternalMovement`
| Old | New |
|-----|-----|
| _(struct)_ `MovimientoInterno` | `InternalMovement` |
| `PresentacionID` | `PresentationID` |
| `ReemplazarCantidad` | `ReplaceQuantity` |
| `Tipo` | `Type` |
| `AlmacenDestinoID` | `DestWarehouseID` |
| `Cantidad` | `Quantity` |
| `SubCantidad` | `SubQuantity` |

### `backend/business/types/generales.go`

#### `CityLocation`
| Old | New | DB column impact |
|-----|-----|-----------------|
| `PaisID` | `CountryID` | `pais_id` → `country_id` — also in Partition |
| `Nombre` | `Name` | `nombre` → `name` |
| `PadreID` | `ParentID` | `padre_id` → `parent_id` |
| `Jerarquia` | `Hierarchy` | `jerarquia` → `hierarchy` |
| `Departamento` | `Department` | `json:"-"`, no DB col |
| `Provincia` | `Province` | `json:"-"`, no DB col |
| _(add)_ `Distrito` | `District` | `json:"-"`, no DB col — hierarchy level 3, missing from struct |

#### `SharedListRecord`
| Old | New | DB column impact |
|-----|-----|-----------------|
| `ListaID` | `ListID` | `lista_id` → `list_id` — also in Views/Indexes |
| `Nombre` | `Name` | `nombre` → `name` |
| `Descripcion` | `Description` | `descripcion` → `description` |
| `NombreHash` | `NameHash` | `nombre_hash` → `name_hash` — also in LocalIndex |

Also update `SelfParse()`: `e.NombreHash` → `e.NameHash`, `e.Nombre` → `e.Name`, `e.ListaID` → `e.ListID`.

### `backend/business/types/productos.go`

#### `Product`
| Old | New | DB column impact |
|-----|-----|-----------------|
| `Nombre` | `Name` | `nombre` → `name` — also in LocalIndex |
| `Descripcion` | `Description` | `descripcion` → `description` |
| `CategoriasIDs` | `CategoryIDs` | `categorias_ids` → `category_i_ds` ⚠️ add `db:"category_ids"` tag |
| `MarcaID` | `BrandID` | `marca_id` → `brand_id` |
| `Precio` | `Price` | `precio` → `price` |
| `MonedaID` | `CurrencyID` | `moneda_id` → `currency_id` |
| `UnidadID` | `UnitID` | `unidad_id` → `unit_id` |
| `Descuento` | `Discount` | `descuento` → `discount` |
| `PrecioFinal` | `FinalPrice` | `precio_final` → `final_price` |
| `Peso` | `Weight` | `peso` → `weight` |
| `Volumen` | `Volume` | `volumen` → `volume` |
| `SbnCantidad` | `SbuQuantity` | `sbn_cantidad` → `sbu_quantity` |
| `SbnUnidad` | `SbuUnit` | `sbn_unidad` → `sbu_unit` |
| `SbnPrecio` | `SbuPrice` | `sbn_precio` → `sbu_price` |
| `SbnDescuento` | `SbuDiscount` | `sbn_descuento` → `sbu_discount` |
| `SbnPrecioFinal` | `SbuFinalPrice` | `sbn_precio_final` → `sbu_final_price` |
| `NombreHash` | `NameHash` | `nombre_hash` → `name_hash` — also in LocalIndex |
| `Propiedades` | `Properties` | `propiedades` → `properties` |
| `Presentaciones` | `Presentations` | `presentaciones` → `presentations` |
| `StockReservado` | `ReservedStock` | `stock_reservado` → `reserved_stock` |
| `CategoriasConStock` | `CategoriesWithStock` | `categorias_con_stock` → `categories_with_stock` — also GlobalIndex |

Also update `SelfParse()` and `FillCategoriasConStock()` body.

#### `ProductPresentation`
| Old | New |
|-----|-----|
| `Precio` | `Price` |
| `DiferenciaPrecio` | `PriceDifference` |

#### `ProductImage`
| Old | New |
|-----|-----|
| `Descripcion` | `Description` |

#### `ProductProperty` / `ProductProperties`
| Old | New |
|-----|-----|
| `Nombre` | `Name` |

#### `WarehouseStockMin`
| Old | New |
|-----|-----|
| `Cantidad` | `Quantity` |

#### `Warehouse`
| Old | New | DB column impact |
|-----|-----|-----------------|
| `SedeID` | `SiteID` | `sede_id` → `site_id` |
| `Nombre` | `Name` | `nombre` → `name` |
| `Descripcion` | `Description` | `descripcion` → `description` |
| `Ciudad` | `City` | `ciudad` → `city` — `json:",omitempty"` only (not a DB col) |

#### `Site`
| Old | New | DB column impact |
|-----|-----|-----------------|
| `Nombre` | `Name` | `nombre` → `name` |
| `Descripcion` | `Description` | `descripcion` → `description` |
| `Direccion` | `Address` | `direccion` → `address` |
| `CiudadID` | `CityID` | keeps `db:"pais_ciudad_id"` tag — no column change |
| `Ciudad` | `City` | `db:"-"`, no DB col |

### `backend/security/types/perfiles.go`

| Old | New | DB column impact |
|-----|-----|-----------------|
| `Nombre` | `Name` | `nombre` → `name` |
| `Descripcion` | `Description` | `descripcion` → `description` |
| `Modulos` | `Modules` | `modulos_ids` (explicit `db:` tag) — no column change |

### `backend/exec/demo2.go`

| Old | New |
|-----|-----|
| `ListaID` | `ListID` |
| `Nombre` | `Name` |
| `Descripcion` | `Description` |
| `InnerStruct.Hola` | `InnerStruct.Greeting` |
| `InnerStruct.Hola2` | `InnerStruct.Greeting2` |

Also update the commented-out `DemoStruct` literal inside `Test38`.

---

## Affected Backend Go Files (non-struct usages to update)

After renaming struct fields, the compiler will flag every broken reference. Files known to use these fields:

| File | Fields used |
|------|------------|
| `backend/business/sedes-almacenes.go` | `Site`, `Warehouse`, `CityLocation` fields |
| `backend/business/productos.go` | `Product`, `ProductPresentation`, etc. |
| `backend/finance/cajas.go` | `CashBank`, `CashBankMovement`, `CashReconciliation`, `InternalCashMovement` fields |
| `backend/logistics/` (movement handlers) | `WarehouseProductMovement`, `MovimientoInterno` fields |
| `backend/security/perfiles.go` | `Profile` fields |
| `backend/config/empresas.go` | `Company`, `CulqiConfig` fields |
| `backend/config/parametros.go` | `Parametros` / `Parameters` fields |
| `backend/core/users*.go` | `User` fields |
| `backend/exec/demo2.go` | `DemoStruct`, `InnerStruct` |
| `backend/exec/init.go` | controller registration (if `Parametros` struct is renamed) |

Strategy: **compile after each struct file change** to let the compiler produce the full list of broken call sites — this is faster than grepping.

---

## Affected Frontend Files

### TypeScript Interface Definitions (must be updated first)

| File | Interfaces |
|------|-----------|
| `frontend/core/types/common.ts` | `IUser`, `IProfile` |
| `frontend/routes/negocio/sedes-almacenes/sedes-almacenes.svelte.ts` | `ISite`, `IWarehouse`, `ICityLocation` |
| `frontend/services/services/ciudades.svelte.ts` | `ICiudad` |
| `frontend/routes/negocio/productos/productos.svelte.ts` | `IProduct`, `IProductPresentation` |
| `frontend/services/services/productos.svelte.ts` | `IProduct` |
| `frontend/routes/finanzas/cajas/cajas.svelte.ts` | `ICashBank`, `ICashBankMovement`, `ICashReconciliation` |
| `frontend/routes/configuracion/empresas/empresas.svelte.ts` | `ICompany`, `ICompanyCulqui` |
| `frontend/routes/configuracion/parametros/empresas.svelte.ts` | same |
| `frontend/services/negocio/listas-compartidas.svelte.ts` | `ISharedListRecord` |
| `frontend/routes/seguridad/usuarios/usuarios.svelte.ts` | `IUser` usage |
| `frontend/routes/seguridad/perfiles-accesos/perfiles-accesos.svelte.ts` | `IProfile` |
| `frontend/routes/logistica/almacen-movimientos/almacen-movimientos.svelte.ts` | `IWarehouseProductMovement` |
| `frontend/core/env.ts` | `ICompanyParams` |
| `frontend/core/modules.ts` | module config using `Nombre` |

### Svelte Components (field access in templates/scripts)

**Business / Products:**
- `frontend/routes/negocio/productos/+page.svelte`
- `frontend/routes/negocio/productos/CategoriasMarcas.svelte`
- `frontend/routes/negocio/productos/Atributos.svelte`
- `frontend/routes/negocio/clientes/ClientesProveedoresView.svelte`
- `frontend/routes/negocio/sedes-almacenes/+page.svelte`

**Commercial / Sales:**
- `frontend/routes/comercial/SaleOrdersTable.svelte`
- `frontend/routes/comercial/sale_order_create/+page.svelte`
- `frontend/routes/comercial/sale_order_create/ProductoVentaCard.svelte`
- `frontend/routes/comercial/sale_order_create/sale_order.svelte.ts` (`PrecioFinal`, `Precio`)
- `frontend/routes/comercial/sale_orders_status/+page.svelte`
- `frontend/routes/comercial/reporte-ventas/+page.svelte`
- `frontend/routes/comercial/shipping-costs/+page.svelte`
- `frontend/routes/comercial/sale_orders_charts/SaleOrdersChartsByProduct.svelte`
- `frontend/routes/comercial/sale_orders_charts/SaleOrdersChartsDailySummary.svelte`

**Logistics:**
- `frontend/routes/logistica/almacen-movimientos/+page.svelte`
- `frontend/routes/logistica/purchase-orders/+page.svelte`
- `frontend/routes/logistica/purchase-orders/PurchaseOrderForm.svelte`
- `frontend/routes/logistica/purchase-orders/PurchaseOrderCreate.svelte`
- `frontend/routes/logistica/purchase-orders/PurchaseOrderReport.svelte`
- `frontend/routes/logistica/purchase-orders/ProductCardSearch.svelte`
- `frontend/routes/logistica/gestion-compras/ProductSupplyManagement.svelte`
- `frontend/routes/logistica/products-stock/PurchaseOrderEntry.svelte`
- `frontend/routes/logistica/products-stock/ProductStockMovement.svelte`
- `frontend/routes/logistica/supplies-materials/+page.svelte`

**Finance:**
- `frontend/routes/finanzas/cajas/+page.svelte`
- `frontend/routes/finanzas/cajas/CajaForm.svelte`
- `frontend/routes/finanzas/cajas-movimientos/+page.svelte`

**Security:**
- `frontend/routes/seguridad/usuarios/+page.svelte`
- `frontend/routes/seguridad/usuarios/UserProfilesAccessSelector.svelte`
- `frontend/routes/seguridad/perfiles-accesos/+page.svelte`

**Configuration:**
- `frontend/routes/configuracion/empresas/+page.svelte`
- `frontend/routes/configuracion/parametros/+page.svelte`
- `frontend/routes/configuracion/backups/+page.svelte`

**Ecommerce:**
- `frontend/ecommerce/components/CiudadesSelector.svelte`
- `frontend/ecommerce/components/ProductCard.svelte`
- `frontend/ecommerce/components/CartMenu.svelte`
- `frontend/ecommerce/components/FloatingCart.svelte`
- `frontend/ecommerce/components/UsuarioMenu.svelte`
- `frontend/ecommerce/ecommerce-components/ecommerce-attributes/CategoryDescription.svelte`

**Shared / Domain:**
- `frontend/ui-components/misc/RecordByIDText.svelte`
- `frontend/domain-components/HeaderConfig.svelte`
- `frontend/core/product-search/productos-delta-service.ts`

---

## Execution Phases

### Phase 0 — All questions resolved. Proceed directly to Phase 1.

### Phase 1 — Backend struct definitions
For each file below, rename fields in **both** the record struct and the matching `*Table` struct simultaneously. Then run `go build ./...` to get the full compiler error list.

Order (least dependencies first):
1. `config/types/parametros.go`
2. `config/types/empresas.go`
3. `core/types/users.go`
4. `security/types/perfiles.go`
5. `finance/types/cajas.go`
6. `business/types/generales.go`
7. `business/types/productos.go`
8. `logistics/types/product-stock-movement.go`
9. `exec/demo2.go`

### Phase 2 — Backend call sites
Fix every compiler error from Phase 1. Pay special attention to:
- Handler files that construct or query these structs
- `SelfParse()` / `FillCategoriasConStock()` bodies
- `GetSchema()` bodies that reference field names (they use the table struct columns, not string names, so they update automatically — but verify)

### Phase 3 — Frontend TypeScript interfaces
Update all interface definitions listed in the "Interface Definitions" section. These are the source of truth for the frontend — update them first so TypeScript surfaces all downstream errors.

### Phase 4 — Frontend components
Fix every TypeScript/Svelte error from Phase 3. Go file-by-file through the Svelte component list.

### Phase 5 — Verify
- `go build ./...` returns zero errors
- `cd frontend && npm run check` (or equivalent type-check command) returns zero errors
- Manually test: products page, sites/warehouses page, users page, cash banks page

---

## DB Column Name Notes

Fields **without** an explicit `db:` tag will have their column name change to snake_case of the new English field name. Since the project is pre-alpha, the DB schema can be recreated. No migration scripts needed.

Fields that already have an explicit `db:` tag (e.g. `CiudadID db:"pais_ciudad_id"`) will **not** change their column name — only the Go field name changes.

For `*IDs` fields like `CategoriasIDs` → `CategoryIDs`, the ORM's snake_case of `CategoryIDs` is `category_i_ds` which looks wrong. Add explicit `db:"category_ids"` tags for these. Same for `ProfileIDs` → `db:"profile_ids"` and `AccessLevelIDs` → `db:"access_level_ids"`.
