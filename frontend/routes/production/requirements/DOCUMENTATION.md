---
schema: 1
page_id: production.requirements
route: /production/requirements
title: Purchase Requirements (Necesidades de compra)
status: implemented
visibility: tenant
description_en: >-
  Material requirements planning for production. Pick draft production orders and/or a template ×
  series and a supplies warehouse; the page explodes every recipe level, nets each supply against
  stock, reservations, blocked lots and confirmed purchase orders, and turns the shortfalls into
  pending purchase orders per supplier and draft orders for semi-finished supplies.
description_es: >-
  Cálculo de necesidades de materiales para la producción. Elegir órdenes en borrador y/o una
  plantilla por series y un almacén de insumos; la página explota cada nivel de receta, descuenta
  el stock, lo reservado, lo bloqueado y las órdenes de compra confirmadas, y convierte los
  faltantes en órdenes de compra pendientes por proveedor y en órdenes en borrador de semielaborados.
---

# Purchase Requirements (Necesidades de compra)

<!-- DOC-ID: page-purpose -->
## Page purpose

Purchase requirements (`Necesidades de compra`, an MRP or `cálculo de necesidades`) answers "what
do I have to buy, and what do I have to make first, to produce what I planned?". It takes draft
production orders and/or a template × series, explodes their recipes into supplies (`insumos`),
compares the result with what one supplies warehouse holds and expects, and proposes:

- purchase orders (`órdenes de compra`) for the bought supplies that are short, one per supplier;
- draft production orders for the semi-finished supplies (`semielaborados`) that are short.

The page saves no plan of its own: it calculates on screen and only writes when the user creates
the purchase orders or the draft orders. It does not release production orders (use Production
orders / Órdenes de producción) and does not confirm or receive purchases (use Purchase Orders /
Órdenes Compra).

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- **Need (Necesidad):** what the plan asks of a supply: units × recipe quantity ÷ yield × (1 +
  waste) × (1 + the draft's supply adjustment), rounded up. A template × series has no adjustment.
- **Level (Nivel):** 1 is asked directly by the planned products; a supply inside a semi-finished
  supply is at a deeper level. A supply used at several levels is settled at the deepest one.
- **Minimum (Mínimo):** the supply's minimum stock, counted only with **Replenish to the minimum
  (Reponer al mínimo)** ticked.
- **Stock**, **Reserved (Reservado)** by released production orders, **Blocked (Bloqueado)** lots
  (not released, such as quarantine), and **On order (Por llegar)** from confirmed purchase orders.
- **Shortfall (Faltante)** = need + minimum − (stock − reserved − blocked) − on order, never below
  0; shown in red when positive. Purchases and orders are proposed in whole units, rounded up.

<!-- DOC-ID: capability.calculate-requirements -->
## Calculate what the plan needs (Calcular insumos necesarios)

### User intention (Intención del usuario)

See, before releasing, which supplies are missing for one or several planned orders or for a
template run.

### Where to find it (Dónde encontrarlo)

**Operations (Operaciones) → Production (Producción) → Purchase requirements (Necesidades de
compra)** at `/production/requirements`. At the top: **Supplies warehouse (Almacén de insumos)**,
**Template (optional) (Plantilla (opcional))**, **Series** and **Replenish to the minimum (Reponer
al mínimo)**. The **Draft orders (Órdenes en borrador)** table lists drafts with **N°**, **Date
(Fecha)**, **Products (Productos)** and **Supplies wh. (Alm. insumos)**; select the rows to plan.
The result is **Supplies needed (Insumos necesarios)**.

### Required information and prerequisites (Requisitos previos)

- A supplies warehouse, then at least one selected draft or a template with series greater than 0.
- Recipes on the products' and semi-finished supplies' **Producción** tab.
- For **Reponer al mínimo**: the minimum stock set in Supplies & Materials (Insumos & Materiales) or
  Purchase Management (Gestión de Compras).

### Business rules and rationale (Reglas y razón de negocio)

- Every supply the plan reaches gets a row, short or not, so the user sees why nothing is
  proposed. Semi-finished supplies show a gear icon.
- A short semi-finished supply is planned to be made in whole units, and its own recipe adds to the
  need of the deeper supplies before they are netted.
- Only the selected warehouse's stock counts. **On order** counts confirmed purchase orders for that
  warehouse or without a warehouse; pending ones do not count.
- Opening the page from a refused release (`?order-id=…`) selects that draft and its supplies
  warehouse.

### Result and side effects (Resultado y efectos)

Nothing is saved by calculating.

### Limitations (Limitaciones)

- One warehouse at a time; stock in other warehouses is not considered.
- Released orders are not planned here: their supplies already appear as **Reserved**.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Qué insumos me faltan para la orden 15? ¿Cuánto cuero tengo que comprar para 3 series?
- ¿Por qué me pide comprar si tengo stock? → it may be reserved by another order, blocked, or in
  another warehouse.
- MRP, explosión de materiales, requerimiento de materiales, faltantes, reposición.

<!-- DOC-ID: capability.create-purchase-orders -->
## Create purchase orders for the shortfalls (Crear órdenes de compra)

### User intention (Intención del usuario)

Turn the missing bought supplies into purchase orders without typing them.

### Where to find it (Dónde encontrarlo)

Section **To buy (Por comprar)**: **Supply (Insumo)**, **Units (Unidades)**, **Provider
(Proveedor)**, **Price (Precio)** and **Total**, then **Create purchase orders (Crear órdenes de
compra)** and confirm.

### Required information and prerequisites (Requisitos previos)

- Every line with units needs a supplier.
- The supplier and price are prefilled with the cheapest supplier price on file for the supply (the
  **Proveedores** tab of Supplies & Materials or Purchase Management).

### Business rules and rationale (Reglas y razón de negocio)

- Units, supplier and price are editable. Choosing another supplier takes that supplier's price on
  file, when there is one.
- Semi-finished supplies are never bought; they appear under **To make (Por fabricar)**.

### Result and side effects (Resultado y efectos)

One purchase order per supplier is created in **Pending (Pendiente)** status, for the selected
warehouse, with whole units at the shown prices. The created numbers are shown. The button stays
disabled until the plan changes, so the same supplies are not ordered twice.

### Limitations (Limitaciones)

- The new purchase orders do not count as **On order (Por llegar)** until they are confirmed in
  Purchase Orders (Órdenes Compra).
- If one order fails, the ones already created remain and are listed in the error ("ya creadas").

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo genero las órdenes de compra de lo que falta? ¿Puedo cambiar el proveedor sugerido? → yes.
- Pedido a proveedor, requerimiento de compra, OC.

<!-- DOC-ID: capability.create-semi-finished-orders -->
## Create draft orders for semi-finished supplies (Crear órdenes de semielaborados)

### User intention (Intención del usuario)

Plan the in-house (or subcontracted) production of the semi-finished supplies the plan lacks.

### Where to find it (Dónde encontrarlo)

Section **To make (Por fabricar)**: **Semi-finished (Semielaborado)**, **Units (Unidades)** and
**Workshop (Taller)**, then **Create orders (Crear órdenes)** and confirm.

### Business rules and rationale (Reglas y razón de negocio)

- One draft per short semi-finished supply, for its shortfall in whole units.
- Its supplies come from, and the product enters, the selected warehouse.
- **Workshop** shows the workshop of the most recent subcontracted order of that supply, which the
  draft inherits with its price per unit; otherwise **In-house (Propio)**.

### Result and side effects (Resultado y efectos)

The drafts appear in Production orders and in this page's **Draft orders** table. Nothing is
reserved until each one is released.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo creo las órdenes de los semielaborados que faltan? ¿Puedo cambiar el taller? → edit the
  draft in Production orders.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"Seleccione el almacén de insumos y luego órdenes en borrador o una plantilla y sus series."**
  Pick the warehouse; **"Aún no hay nada planificado."** means no draft is selected and no template
  with series is set.
- **"No hay nada que comprar."** No bought supply is short, or every line has 0 units.
- **"Elija el proveedor de cada línea."** A supply has no supplier on file; pick one in the line or
  add it on the supply's **Proveedores** tab.
- **A draft is missing from the table:** only drafts are listed; released orders already reserve
  their supplies.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Production orders (Órdenes de producción):** the drafts planned here; a refused release points
  here.
- **Purchase Orders (Órdenes Compra):** confirm and receive the created purchase orders.
- **Supplies & Materials (Insumos & Materiales)** and **Purchase Management (Gestión de Compras):**
  minimum stock and suppliers with prices.
- **Production templates (Plantillas de producción):** the template × series option.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.calculate-requirements]
  - path: frontend/routes/production/requirements/+page.svelte
    role: page
    hash: sha256:f9cef2e5361dcb7e33cc4af9ea3652c9d38d683b349df08529118f79c40b3815
    supports: [page-purpose, concepts, capability.calculate-requirements, capability.create-purchase-orders, capability.create-semi-finished-orders, troubleshooting]
  - path: frontend/routes/production/requirements/requirements.svelte.ts
    role: frontend-service
    hash: sha256:2c7b810f478b9112c3aec45c12b60158c2ce361b5a91c421cc18c322a118f85a
    supports: [page-purpose, concepts, capability.calculate-requirements, capability.create-purchase-orders, capability.create-semi-finished-orders]
  - path: frontend/routes/production/requirements/requirements.ts
    role: business-logic
    hash: sha256:e405d3133047ea803f3ec06d9f850e106e2f57a557223723ef72eff43d2167b0
    supports: [concepts, capability.calculate-requirements, capability.create-purchase-orders, capability.create-semi-finished-orders, troubleshooting]
  - path: frontend/routes/production/orders/production-order.ts
    role: business-logic
    hash: sha256:ae1e83a71061c4619e8921f42a5364212fcc9008075189a77dc20619eb8d6874
    supports: [concepts]
  - path: frontend/routes/production/orders/production-order.svelte.ts
    role: frontend-service
    hash: sha256:af03f65459602f36a00094d11acb4c6a8791ee2502795ec8961d0c42b28d072a
    supports: [capability.calculate-requirements]
  - path: frontend/routes/production/orders/OrderLayer.svelte
    role: user-interface
    hash: sha256:4a4bde050b0a7bbd17913252b2a69fefad3bc25285eca40e6ea66fd3ed29331c
    supports: [capability.calculate-requirements, related-pages]
  - path: frontend/routes/logistics/purchase-management/supply-management.svelte.ts
    role: frontend-service
    hash: sha256:4a7b936c87a31a7bcef58b2bfc4de90cd17d3fb8c7a3bac850332140ea42e8af
    supports: [capability.calculate-requirements, capability.create-purchase-orders]
  - path: frontend/routes/production/supplies-materials/+page.svelte
    role: user-interface
    hash: sha256:a40d5e4f4bf55ffdf9db385897029e4b2c41671104724dc06d526c063cfdfb73
    supports: [capability.calculate-requirements, capability.create-purchase-orders, troubleshooting]
  - path: frontend/routes/logistics/purchase-management/ProductSupplyManagement.svelte
    role: user-interface
    hash: sha256:30c2fd2188588f40faf9ccf94399c0806e58d6dce7ca5189fb439c0e280c6cd2
    supports: [capability.calculate-requirements, capability.create-purchase-orders]
  - path: frontend/routes/logistics/purchase-orders/purchase_order.svelte.ts
    role: frontend-service
    hash: sha256:396bca1a4ee5bf35c14894a876e86223cf04618e98469f17ee40cbc36cffde9f
    supports: [concepts, capability.create-purchase-orders]
  - path: backend/logistics/purchase-order-management.go
    role: backend-handler
    hash: sha256:833a7dade609a4b174f3ffc3bb264a563437ff1dc37c5345efba6541e04ccf00
    supports: [capability.create-purchase-orders]
  - path: backend/production/production_orders.go
    role: backend-handler
    hash: sha256:d6e99162ca6371a0e3f00161ea2f0e47b0a554ae13d6d83e06d88bd5fef58883
    supports: [capability.create-semi-finished-orders]
```
