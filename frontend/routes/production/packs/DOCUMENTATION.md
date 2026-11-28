---
schema: 1
page_id: production.packs
route: /production/packs
title: Packs (Packs)
status: implemented
visibility: tenant
description_en: >-
  Define packs of products or categories sold together, such as a lunch menu (starter + main
  course), priced at a fixed total, a percentage off or an amount off with optional rounding to the
  sol. Genix finds the packs in a restaurant table's bill when it is charged and prorates the pack
  price over the lines.
description_es: >-
  Definir packs de productos o categorías vendidos juntos, como un menú (entrada + plato de fondo),
  con precio fijo, descuento en porcentaje o descuento en monto y redondeo opcional al sol. Genix
  encuentra los packs en la cuenta de una mesa al cobrarla y reparte el precio del pack entre las
  líneas.
---

# Packs (Packs)

<!-- DOC-ID: page-purpose -->
## Page purpose

Packs (`packs`, `combos`, `menús`) are bundles priced as a whole: "1 × Entrada + 1 × Plato de fondo
= S/ 18", or "2 gaseosas with 15% off". Each slot (`componente`) is a specific product or any
product of a category (`categoría`). The page defines the packs; Genix applies them by itself when
a restaurant table's bill is charged — the cashier does not pick the pack.

The page does not change catalog prices (use Products / Productos) and does not sell. In the
current flows packs are applied when a table is charged in Tables (Mesas); the Point of Sale (Punto
de Venta) does not apply packs.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A **product slot** takes units of that product; a **category slot** (`Categoría: …`) takes any
  product of that category, as the form says: "Un componente de categoría acepta cualquier producto
  de esa categoría."
- **Price rule (Regla de precio):**
  - **Fixed pack price (Precio fijo del pack):** the pack total, in soles.
  - **Percentage off (Descuento en porcentaje):** a percentage off the sum of the units' catalog
    prices.
  - **Amount off (Descuento en monto):** soles off that sum.
- **Rounding of the pack total (Redondeo del total del pack):** only for the two discounts — **No
  rounding (Sin redondeo)**, **Round up to the sol (Al sol superior)**, **Round to the nearest sol
  (Al sol más cercano)** or **Round down to the sol (Al sol inferior)**. A fixed price is never
  rounded.

<!-- DOC-ID: capability.create-edit-pack -->
## Create or edit a pack (Crear o editar un pack)

### User intention (Intención del usuario)

Offer a menu or combo at a special price that is applied automatically when its items are ordered
together.

### Where to find it (Dónde encontrarlo)

**Commercial (Comercial) → Catalog (Catálogo) → Packs** at `/production/packs`. **New (Nuevo)**
opens **New pack (Nuevo pack)**; click a pack in the list to edit it. The list shows **Pack**,
**Components (Componentes)** (for example `1× Entrada + 1× Plato Fondo 1`) and **Price (Precio)** (`S/
18.00`, `−15.00%` or `−S/ 2.00`, with the rounding).

### Required information and prerequisites (Requisitos previos)

- **Name (Nombre):** at least 2 characters.
- **Price rule (Regla de precio)** and its value: **Pack price (S/) (Precio del pack (S/))**,
  **Discount (%) (Descuento (%))** or **Discount (S/) (Descuento (S/))**.
- **Components (Componentes):** **Component (Componente)** adds a row; pick **Product or category
  (Producto o categoría)** and type **Quantity (Cantidad)**. Products and categories must be active.

### Business rules and rationale (Reglas y razón de negocio)

- At least one component; every row needs a product or category and a quantity of 1 or more.
- A product or category cannot repeat: add up its quantity in one row.
- The pack must total at least 2 units (a one-unit "pack" would only be a price change).
- The value must be greater than 0, and a percentage must be below 100%.

### Result and side effects (Resultado y efectos)

The pack is saved and starts applying to the next table charges. Prices already charged do not
change.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo creo un menú del día con entrada y segundo a precio fijo? ¿Cómo hago un combo con
  descuento?
- Pack, combo, menú, promoción, oferta 2x, precio especial, paquete.

<!-- DOC-ID: capability.how-packs-apply -->
## How packs are applied to a bill (Cómo se aplican los packs)

### User intention (Intención del usuario)

Understand which pack the bill takes and why a line shows a different price than the catalog.

### Where to find it (Dónde encontrarlo)

In Tables (Mesas): a round being composed shows the packs that round fills as a header row over
its dishes, and the charge form shows the packs the whole bill fills, priced as the charge will.

### Business rules and rationale (Reglas y razón de negocio)

- Packs apply to the whole bill, not per round: a starter of the first round and a main course of
  the second still form a menu.
- A pack only applies when it is cheaper than the units at catalog price.
- When several packs compete for the same units, the one with more relevance wins: a product slot
  weighs more than a category slot. A tie goes to the larger saving, then to the pack created first.
- A pack applies as many times as the bill fills it.
- Product slots are filled before category slots, and each slot takes the most expensive eligible
  unit first, giving the customer the largest saving.
- The pack price is spread over its units in proportion to their catalog prices; the leftover
  cents go to the most expensive unit, so the lines add up exactly to the pack price.
- Only whole units take part; fractions (sub-units) stay at catalog price.

### Result and side effects (Resultado y efectos)

On the sale, a line whose units end at different prices is split into one line per pack price,
plus the rest at catalog price. Each repriced line remembers the pack that priced it. The table's
order itself keeps the catalog prices.

### Limitations (Limitaciones)

- Packs are applied only when charging a table in Tables (Mesas); the Point of Sale (Punto de
  Venta) does not apply them.
- There is no manual choice of pack, no date range and no activation switch: every saved pack
  applies until it is deleted.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Por qué el plato salió más barato en la cuenta? → a pack priced it.
- ¿Por qué no se aplicó el menú? → the units cost less than the pack at catalog price, a more
  relevant pack took them, or the sale was not a table charge.

<!-- DOC-ID: capability.delete-pack -->
## Delete a pack (Eliminar un pack)

### Where to find it (Dónde encontrarlo)

Open the pack → the panel's delete button, then confirm.

### Result and side effects (Resultado y efectos)

The pack stops applying. Sales already made keep their prices.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo quito una promoción? ¿Se recalculan las ventas anteriores? → no.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"Un pack necesita al menos 2 unidades en total."** Raise a quantity or add a component.
- **"Un producto o categoría está repetido: sume su cantidad en una fila."** Keep one row per item.
- **"El descuento debe ser menor a 100%."** Use a smaller percentage.
- **"El producto … del pack no existe o no está activo."** / **"La categoría … del pack no existe o
  no está activa."** Remove the slot or reactivate the item in Products.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Tables (Mesas):** where packs are applied when a table's bill is charged.
- **Products (Productos):** catalog prices and categories the packs are built from.
- **Point of Sale (Punto de Venta):** sells at catalog prices; packs do not apply there.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.create-edit-pack]
  - path: db/pages.toml
    role: user-interface
    hash: sha256:d435afce0db0eedf924345040be60e0f4d1a15672e0f1f2d9003687920fb3da7
    supports: [related-pages]
  - path: frontend/routes/production/packs/+page.svelte
    role: page
    hash: sha256:5cc3ff32126c41a2ceaaa13cf9835f31b07b5c4f031f1843ec6f722c4c6db7fe
    supports: [page-purpose, concepts, capability.create-edit-pack, capability.delete-pack]
  - path: frontend/routes/production/packs/packs.ts
    role: business-logic
    hash: sha256:f9ec69c4a367e7413593f2620f429728ec5a9a797b917f5a4bef6c29ef6d93d6
    supports: [capability.create-edit-pack, troubleshooting]
  - path: frontend/services/production/product-packs.svelte.ts
    role: frontend-service
    hash: sha256:32686fe1c7ee2a64f58e387c174299f609f1692c09622c52e7d5813df0286577
    supports: [capability.create-edit-pack, capability.delete-pack]
  - path: backend/production/product_packs.go
    role: backend-handler
    hash: sha256:a4e2efe105bb7cfd84dee65e46956260ec59cb3f0bada8088839e2ef5dc9cbb8
    supports: [concepts, capability.create-edit-pack, capability.delete-pack, troubleshooting]
  - path: backend/production/types/product_pack.go
    role: data-model
    hash: sha256:df86b869dc7901fa0319b4f7c5a75e74739e154aa5c75bb19a49c6341e63639a
    supports: [concepts, capability.create-edit-pack, capability.how-packs-apply]
  - path: backend/sales/types/sale_packs.go
    role: business-logic
    hash: sha256:feb9aa9f0c400de5594bab3e539756ab7885dd5099e7967bc4570de116e3cbc9
    supports: [capability.how-packs-apply]
  - path: backend/sales/types/sale_products.go
    role: business-logic
    hash: sha256:87c99ebfbf74a92b58d29f3cac0cf18f0050912dc4eeea8b306d2707a12d3e2e
    supports: [capability.how-packs-apply]
  - path: backend/dining/table_order_charge.go
    role: backend-handler
    hash: sha256:386bfea6d92d26936e10004ec53b5d3b118a119c2bd3cc31e90a1f75b23e8605
    supports: [page-purpose, capability.how-packs-apply]
  - path: backend/sales/sale_order_create.go
    role: backend-handler
    hash: sha256:bdce9e105581050ae6830b9f2ec31479584401003ab78837155bc37ca6e7f4ee
    supports: [page-purpose, capability.how-packs-apply, related-pages]
  - path: frontend/routes/dining/floor-service/RoundComposer.svelte
    role: user-interface
    hash: sha256:25ef0b2042bc0e26711b9d09a6ad275cd5f2213562d9c2bb84efc49690c7917a
    supports: [capability.how-packs-apply]
  - path: frontend/routes/dining/floor-service/TableChargeForm.svelte
    role: user-interface
    hash: sha256:adf6b6ec71a70429c55954e1f7e6e99439db6e1024665d8de343d907d8018b94
    supports: [capability.how-packs-apply]
```
