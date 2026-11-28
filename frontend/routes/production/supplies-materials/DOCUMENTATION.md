---
schema: 1
page_id: production.supplies-materials
route: /production/supplies-materials
title: Supplies & Materials (Insumos & Materiales)
status: implemented
visibility: tenant
description_en: >-
  Register the supplies and materials production consumes or the company uses: name, SKU, price,
  sub-unit, lot policy, default accounting account and depreciation for assets, minimum stock and
  suppliers with their prices. Mark a supply as semi-finished to give it its own recipe.
description_es: >-
  Registrar los insumos y materiales que consume la producción o usa la empresa: nombre, SKU,
  precio, sub-unidad, control de lotes, cuenta contable por defecto y depreciación para activos,
  stock mínimo y proveedores con sus precios. Marcar un insumo como semielaborado para darle su
  propia receta.
---

# Supplies & Materials (Insumos & Materiales)

<!-- DOC-ID: page-purpose -->
## Page purpose

Supplies & Materials (`Insumos & Materiales`) is the catalog of everything the company buys to use
rather than to resell: raw materials (`materia prima`) consumed by recipes, consumables
(`suministros`) and equipment that depreciates (`activos`). Each supply carries its unit and
sub-unit, lot policy, default accounting account, minimum stock (`stock mínimo`) and the suppliers
(`proveedores`) that sell it with their prices.

A supply can also be **semi-finished (`semielaborado`)**: something the company makes itself with
its own recipe (for example stitched uppers) and then consumes in other recipes.

The page does not move stock (purchases enter it through Purchase Orders / Órdenes Compra) and
does not register products for sale (use Products / Productos or Merchandise / Mercaderías).

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

The list column **Type (Tipo)** shows what a supply is:

- **Consumable (Consumible):** no depreciation; used up by production or by the company.
- **Asset (Activo):** has a **Depreciation (Depreciación)** term; the Assets (Activos) page can then
  register units of it as fixed assets.
- **Semi-finished (Semielaborado):** made by the company with the recipe on its **Producción** tab;
  never depreciated, and its stock account follows from being made (231 Productos en proceso).

A **sub-unit (`sub-unidad`)** lets a recipe take a fraction of a supply: a supply in kg with
**Sub-units per unit (Sub-unidades por unidad)** 1000 and **Sub-unit (Sub-unidad)** `g`.

<!-- DOC-ID: capability.create-edit-supply -->
## Create or edit a supply (Crear o editar un insumo)

### User intention (Intención del usuario)

Register a raw material, consumable or piece of equipment so it can be bought, stocked and used in
recipes.

### Where to find it (Dónde encontrarlo)

**Operations (Operaciones) → Catalogs (Catálogos) → Supplies & Materials (Insumos & Materiales)** at
`/production/supplies-materials`. **New (Nuevo)** opens **New Supply or Material (Nuevo Insumo o
Material)**; clicking a row opens it. The list shows **ID**, **Supply (Insumo)**, **SKU**, **Brand
(Marca)**, **Type (Tipo)**, **Account (Cuenta)** and **Price (Precio)**, and **Filter supplies
(Filtrar insumos)** searches by name or SKU. Fields are on the **Info (Información)** tab; save with
the panel's save button.

### Required information and prerequisites (Requisitos previos)

- **Name (Nombre):** at least 2 characters.
- Optional: **SKU**, **Base Price (Precio Base)**, **Currency (Moneda)**, **Minimum Stock (Stock
  Mínimo)**, **Brand (Marca)**, **Sub-unit (Sub-unidad)**, **Sub-units per unit (Sub-unidades por
  unidad)**, **Shelf life (Vida útil)** in days and **Description (Descripción)**.
- Checkboxes: **Tracks lots (Controla lotes)**, **New lots enter in quarantine (Lotes nuevos en
  cuarentena)** and **Semi-finished: made by the company (Semielaborado: lo fabrica la empresa)**.
- Only for a supply that is not semi-finished: **Depreciation (Depreciación)** — **No
  depreciable**, **4 años · equipos de cómputo**, **5 años · vehículos**, **10 años · maquinaria**,
  **20 años · muebles** or **33 años · edificios** — and **Default account (Cuenta contable por
  defecto)**.

### Business rules and rationale (Reglas y razón de negocio)

- Price, minimum stock and shelf life cannot be negative.
- Left empty, the default account is 3369 (other equipment) when the supply depreciates, otherwise
  6032 (supplies / `suministros`). Purchase lines of the supply propose this account.
- A sub-unit with more than one part needs a name. Once set, the sub-unit can only be refined to a
  multiple (for example from 6 to 12) and cannot be removed; refining converts the supply's stock.
- A supply cannot switch between semi-finished and bought while it has stock.
- A semi-finished supply cannot depreciate and takes no default account.
- A supply used in a recipe must have an inventory (`existencias`) account; an expense or asset
  account cannot be consumed in production, and the recipe that uses it is refused.

### Result and side effects (Resultado y efectos)

The supply is offered in purchase orders, in recipes (the **Producción** tab of Products and of
semi-finished supplies), and, when it depreciates, on the Assets page. The minimum stock is the same
value Purchase Management (Gestión de Compras) shows, and is used by **Replenish to the minimum
(Reponer al mínimo)** in Purchase requirements (Necesidades de compra).

### Limitations (Limitaciones)

This page does not enter stock or an opening balance (`saldo inicial`); use Purchase Orders
(Órdenes Compra) or Stock Changes (Cambios Stock).

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Dónde registro la materia prima? ¿Cómo creo un insumo en kilos que se use en gramos?
- ¿Cómo registro una máquina o una computadora como activo? → choose a depreciation term.
- Insumo, materia prima, material, suministro, envase, empaque, activo fijo.

<!-- DOC-ID: capability.supply-providers -->
## Register the suppliers of a supply (Proveedores del insumo)

### User intention (Intención del usuario)

Record who sells the supply, at what price, how much they can deliver and how long they take.

### Where to find it (Dónde encontrarlo)

Supply panel → **Providers (Proveedores)** tab. The add button creates a row with **Proveedor**,
**Capacidad**, **Entrega** and **Precio**; each row can be deleted.

### Business rules and rationale (Reglas y razón de negocio)

- Each row needs a valid supplier; the same supplier cannot repeat.
- Capacity, delivery time and price cannot be negative. Empty rows are dropped on save.

### Result and side effects (Resultado y efectos)

Purchase requirements proposes, for each short supply, the supplier with the lowest price on this
list, and takes the supplier's price when another one is chosen.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Dónde pongo el proveedor y el precio de compra del insumo? ¿Cuál proveedor es el más barato?

<!-- DOC-ID: capability.semi-finished -->
## Define a semi-finished supply and its recipe (Semielaborado y su receta)

### User intention (Intención del usuario)

Model an intermediate item the company makes and later consumes (a dough, a sauce, a cut piece),
so production orders can make it and other recipes can use it.

### Where to find it (Dónde encontrarlo)

Tick **Semi-finished: made by the company (Semielaborado: lo fabrica la empresa)**; the **Production
(Producción)** tab appears with **Yield (units) (Rinde (unidades))**, **Overhead per unit (Costos
indirectos por unidad)**, **Standard cost (Costo estándar)**, the **Recipe (Receta)** of supplies
with **Waste % (Merma %)**, the **Labor (Mano de obra)** minutes per labor type, and the **Estimated
cost per unit (Costo estimado por unidad)**.

### Business rules and rationale (Reglas y razón de negocio)

- The supply cannot be part of its own recipe, directly or through other semi-finished supplies,
  and recipes can nest at most 5 levels of semi-finished supplies.
- The recipe needs a yield when it has lines; waste must be between 0% and 100%; a supply or labor
  type cannot repeat.
- Unticking the box clears the recipe on save.

### Result and side effects (Resultado y efectos)

The supply can be planned in Production orders and Production templates, and Purchase requirements
proposes draft orders for it when it is short instead of buying it.

### Limitations (Limitaciones)

A semi-finished supply is never bought through Purchase requirements.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo registro una masa o salsa que preparo y uso en varios platos? ¿Qué es un semielaborado?
- Intermedio, subensamble, preparación, mise en place, producto en proceso.

<!-- DOC-ID: capability.delete-supply -->
## Delete a supply (Eliminar un insumo)

### Where to find it (Dónde encontrarlo)

Open the supply → the panel's delete button → confirm **Eliminar Insumo**.

### Result and side effects (Resultado y efectos)

The supply leaves the list, together with its minimum stock and suppliers.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"El nombre debe tener al menos 2 caracteres."** Use a longer name.
- **"… tiene stock: no puede cambiar entre semielaborado y comprado mientras lo tenga."** Consume or
  move out the stock first.
- **"… el divisor de sub-unidad sólo puede refinarse a un múltiplo de …"** / **"no se puede quitar
  la sub-unidad"**: choose a multiple of the current sub-units per unit.
- **"No se puede repetir el mismo proveedor…"** Keep one row per supplier.
- **"… tiene la cuenta …, que no es de existencias; no puede consumirse en producción."** Change the
  supply's default account to an inventory account before using it in a recipe.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Products (Productos):** recipes of made products use these supplies.
- **Purchase requirements (Necesidades de compra):** what to buy for planned production, with the
  suppliers and minimum stock set here.
- **Purchase Management (Gestión de Compras):** the same minimum stock and suppliers, for products
  too.
- **Purchase Orders (Órdenes Compra):** buy and receive supplies.
- **Assets (Activos):** register depreciable supplies as fixed assets.
- **Production orders (Órdenes de producción):** make semi-finished supplies.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.create-edit-supply]
  - path: frontend/routes/production/supplies-materials/+page.svelte
    role: page
    hash: sha256:a40d5e4f4bf55ffdf9db385897029e4b2c41671104724dc06d526c063cfdfb73
    supports: [page-purpose, concepts, capability.create-edit-supply, capability.supply-providers, capability.semi-finished, capability.delete-supply, troubleshooting]
  - path: frontend/routes/production/supplies-materials/supply-material.svelte.ts
    role: frontend-service
    hash: sha256:6b66976ab4f051b7d911deb14be91121845f48ad5727183cd01475e63d120681
    supports: [concepts, capability.create-edit-supply]
  - path: frontend/routes/production/products/ProductProductionTab.svelte
    role: user-interface
    hash: sha256:396cea9f2101672b7e390f136ef6cfc6e9f2594c3bb7cf7db7c0b6831f467035
    supports: [capability.semi-finished]
  - path: frontend/routes/production/products/product-production.ts
    role: business-logic
    hash: sha256:fbe2490b19f264171272ebdbb67b328e00503474ff313f35c17c343415923ef6
    supports: [capability.semi-finished]
  - path: frontend/routes/logistics/purchase-management/supply-management.svelte.ts
    role: frontend-service
    hash: sha256:4a7b936c87a31a7bcef58b2bfc4de90cd17d3fb8c7a3bac850332140ea42e8af
    supports: [capability.supply-providers, capability.create-edit-supply]
  - path: frontend/routes/logistics/purchase-management/ProductSupplyManagement.svelte
    role: user-interface
    hash: sha256:30c2fd2188588f40faf9ccf94399c0806e58d6dce7ca5189fb439c0e280c6cd2
    supports: [capability.create-edit-supply, related-pages]
  - path: frontend/routes/production/requirements/requirements.ts
    role: business-logic
    hash: sha256:e405d3133047ea803f3ec06d9f850e106e2f57a557223723ef72eff43d2167b0
    supports: [capability.supply-providers, capability.semi-finished]
  - path: frontend/routes/production/requirements/requirements.svelte.ts
    role: frontend-service
    hash: sha256:2c7b810f478b9112c3aec45c12b60158c2ce361b5a91c421cc18c322a118f85a
    supports: [capability.create-edit-supply]
  - path: frontend/routes/production/orders/production-order.svelte.ts
    role: frontend-service
    hash: sha256:af03f65459602f36a00094d11acb4c6a8791ee2502795ec8961d0c42b28d072a
    supports: [capability.semi-finished]
  - path: frontend/routes/accounting/assets/+page.svelte
    role: user-interface
    hash: sha256:784e3dece6c26a6413cb91727f757abc5b1870d06106d375f16bebc1ce8289ca
    supports: [concepts, capability.create-edit-supply, related-pages]
  - path: backend/production/supply_material.go
    role: backend-handler
    hash: sha256:b58e88f9c3a539814e9d40985a695fc6760e2f5339a017e9170fb0a0d8f228cb
    supports: [concepts, capability.create-edit-supply, capability.semi-finished, capability.delete-supply, troubleshooting]
  - path: backend/production/products.go
    role: business-logic
    hash: sha256:548268ab8b8b2e9867692ed231e48fc83c05f967e7d34b7d6bdd561ae61c135e
    supports: [capability.create-edit-supply, troubleshooting]
  - path: backend/production/types/product_production.go
    role: business-logic
    hash: sha256:9264a245d159a927e3b7e6716daff8597dd0079357149792612d24f249ab223f
    supports: [capability.create-edit-supply, capability.semi-finished, troubleshooting]
  - path: backend/accounting/types/pcge.go
    role: business-logic
    hash: sha256:b1eb5ab7555176b1940ee05020c241940c3e0a82dc6c33b06276a0209e6af753
    supports: [capability.create-edit-supply]
  - path: frontend/core/stock-movement-type.ts
    role: shared-domain
    hash: sha256:cfcee015c4d6fadb03f9a9e5d66064e93d734cfa5216e2d36996c47140eecc9c
    supports: [capability.create-edit-supply]
  - path: frontend/routes/logistics/products-stock/ProductStockMovement.svelte
    role: user-interface
    hash: sha256:6345adce9c479b03c02ce6c06cd5f1e37708df2e940db19b61d767a6cdad85ff
    supports: [capability.create-edit-supply]
  - path: backend/logistics/types/product_supply_providers.go
    role: business-logic
    hash: sha256:1bdba99849003fb73d341c4fbde6281040e2f1b09a8bb7ebc193abbeafa1ebe1
    supports: [capability.supply-providers, troubleshooting]
```
