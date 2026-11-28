---
schema: 1
page_id: production.templates
route: /production/templates
title: Production Templates (Plantillas de producción)
status: implemented
visibility: tenant
description_en: >-
  Save reusable lists of made products in their size runs (one series), with default series and
  warehouses, to create production orders quickly. Each template shows the estimated cost of one
  series and the supplies it needs.
description_es: >-
  Guardar listas reutilizables de productos fabricados con su serie de tallas (una serie), con
  series y almacenes por defecto, para crear órdenes de producción rápidamente. Cada plantilla
  muestra el costo estimado de una serie y los insumos que necesita.
---

# Production Templates (Plantillas de producción)

<!-- DOC-ID: page-purpose -->
## Page purpose

Production templates (`Plantillas de producción`) store a repeated production run, such as
"Zapatos Verano": which made products and how many units of each size make **one series** (`una
serie`, the size run or `curva de tallas`). A production order is then created from the template
for a number of series instead of typing the size grid again.

The page only keeps templates. It does not create, release or receive orders (use Production
orders / Órdenes de producción), does not reserve supplies, and does not edit recipes (use the
**Producción** tab of Products / Productos or of a semi-finished supply).

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A **series** (`serie`) is one run of sizes: 36×1, 37×2, 38×3, 39×3, 40×2, 41×1 is a series of 12
  pairs. An order made for 3 series plans three times each size.
- **Default series (Series por defecto)** prefills the number of series when an order is created
  from the template.
- An order created from a template is independent of it: later edits or deleting the template do
  not change the order.

<!-- DOC-ID: capability.create-edit-template -->
## Create or edit a template (Crear o editar una plantilla)

### User intention (Intención del usuario)

Save a product mix and size run that is produced repeatedly.

### Where to find it (Dónde encontrarlo)

**Operations (Operaciones) → Production (Producción) → Production templates (Plantillas de
producción)** at `/production/templates`. **New (Nueva)** opens **New template (Nueva plantilla)**;
clicking a template in the list opens it. Save with the panel's save button.

### Required information and prerequisites (Requisitos previos)

- **Name (Nombre):** at least 2 characters.
- **Default series (Series por defecto):** 1 by default; cannot be negative.
- Optional: **Receiving warehouse (Almacén de ingreso)**, **Supplies warehouse (Almacén de
  insumos)** and **Notes (Notas)**; they are copied to orders created from the template.
- **One series (Una serie):** the size grid. **Product (Producto)** adds a row; type the units under
  each size. Grey cells are sizes the product does not have; products without presentations use the
  **Units (Unidades)** column.
- Only made products (origin **Fabricado**) and semi-finished supplies are offered; to save, each
  needs a recipe or a standard cost.

### Business rules and rationale (Reglas y razón de negocio)

- At least one product; every row needs a product and units in some size.
- A product can appear only once, and each size must be an active presentation of the product —
  the same checks as a production order's lines.
- A chosen warehouse must be an active warehouse.

### Result and side effects (Resultado y efectos)

The template appears in the list with **Products (Productos)**, **Units / series (Unidades /
serie)**, **Default series (Series por defecto)** and **Warehouse (Almacén)**. It is offered in
**From template (Desde plantilla)** on Production orders and in **Template (optional) (Plantilla
(opcional))** on Purchase requirements. Nothing is reserved and no stock moves.

### Limitations (Limitaciones)

A template holds no supply adjustment, dates, stage or workshop; set those on the order.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo guardo una corrida de tallas para usarla siempre? ¿Cómo creo una plantilla de producción?
- Plantilla, modelo de orden, serie, curva de tallas, surtido.

<!-- DOC-ID: capability.series-estimate -->
## See the estimated cost and supplies of one series (Costo e insumos de una serie)

### User intention (Intención del usuario)

Know what one series costs and which supplies it needs before planning orders.

### Where to find it (Dónde encontrarlo)

In the template panel: **Estimated cost of one series (Costo estimado de una serie)** and the table
**Supplies of one series (Insumos de una serie)** with **Quantity (Cantidad)** and **Est. cost
(Costo est.)**.

### Business rules and rationale (Reglas y razón de negocio)

- The cost is units of one series × each product's estimated unit cost: its standard cost when set,
  otherwise its recipe at the supplies' current average cost and the labor types' cost per hour,
  plus its overhead per unit.
- The supply quantities include each supply's waste (`merma`), with no order adjustment.
- **Incomplete: some product or supply has no cost (Incompleto: algún producto o insumo no tiene
  costo)** means a product has no cost source or a supply has no stock value yet; that part is left
  out of the total. A supply without cost shows **no cost (sin costo)**.

### Limitations (Limitaciones)

It is an estimate with today's recipes and averages; the real cost is what an order's receipts
book.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cuánto cuesta producir una serie? ¿Cuánto cuero necesito por serie?

<!-- DOC-ID: capability.delete-template -->
## Delete a template (Eliminar una plantilla)

### Where to find it (Dónde encontrarlo)

Open the template → the panel's delete button, then confirm.

### Result and side effects (Resultado y efectos)

The template leaves the list. Orders created from it keep their lines.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Si borro la plantilla se borran las órdenes? → no.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"El nombre debe tener al menos 2 caracteres."** Use a longer name.
- **"Elija el producto de cada fila."** or **"Cada producto necesita unidades en alguna talla."**
  Complete or remove the empty rows.
- **"… no tiene receta ni costo estándar."** Define the product's recipe or standard cost on its
  Producción tab.
- **"El producto … está repetido en la plantilla."** Keep one row per product.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Production orders (Órdenes de producción):** **From template (Desde plantilla)** creates an
  order for a number of series; **Save as template (Guardar como plantilla)** creates a template
  from an order.
- **Purchase requirements (Necesidades de compra):** computes the supplies a template × series
  needs against a warehouse.
- **Products (Productos):** recipes, presentations (sizes) and standard costs.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.create-edit-template]
  - path: frontend/routes/production/templates/+page.svelte
    role: page
    hash: sha256:483907a8c351eef9d0ab02f33adb9456cfd697401be92646fa9c61203d5cd38f
    supports: [page-purpose, capability.create-edit-template, capability.series-estimate, capability.delete-template]
  - path: frontend/routes/production/templates/production-template.ts
    role: business-logic
    hash: sha256:333e01794d3491ee446a058c4f04a632acf13623ca203d00b80be5cb65f52584
    supports: [capability.create-edit-template, capability.series-estimate, troubleshooting]
  - path: frontend/routes/production/orders/ProductSizeGrid.svelte
    role: user-interface
    hash: sha256:6986b0297bcc078c6e95f9d17242fa934ed6e255eaf725219b7211056fe7cbf8
    supports: [capability.create-edit-template]
  - path: frontend/routes/production/orders/production-order.ts
    role: business-logic
    hash: sha256:ae1e83a71061c4619e8921f42a5364212fcc9008075189a77dc20619eb8d6874
    supports: [concepts, capability.series-estimate, related-pages]
  - path: frontend/routes/production/orders/production-order.svelte.ts
    role: frontend-service
    hash: sha256:af03f65459602f36a00094d11acb4c6a8791ee2502795ec8961d0c42b28d072a
    supports: [capability.create-edit-template, capability.series-estimate]
  - path: frontend/routes/production/products/product-production.ts
    role: business-logic
    hash: sha256:fbe2490b19f264171272ebdbb67b328e00503474ff313f35c17c343415923ef6
    supports: [capability.series-estimate]
  - path: frontend/routes/production/orders/+page.svelte
    role: user-interface
    hash: sha256:9889b9760fc5a6e3a368b911b2e56a29d8f287f9d3a5a99aa9a6e25ec13977ee
    supports: [concepts, capability.create-edit-template, related-pages]
  - path: frontend/routes/production/orders/OrderLayer.svelte
    role: user-interface
    hash: sha256:4a4bde050b0a7bbd17913252b2a69fefad3bc25285eca40e6ea66fd3ed29331c
    supports: [related-pages]
  - path: frontend/routes/production/requirements/+page.svelte
    role: user-interface
    hash: sha256:f9cef2e5361dcb7e33cc4af9ea3652c9d38d683b349df08529118f79c40b3815
    supports: [capability.create-edit-template, related-pages]
  - path: frontend/services/production/production-orders.svelte.ts
    role: frontend-service
    hash: sha256:df596ce61dd20414c561032eca941867ef89dd14b724eefd5d031d2a3b50c0b7
    supports: [capability.create-edit-template, capability.delete-template]
  - path: backend/production/production_orders.go
    role: backend-handler
    hash: sha256:d6e99162ca6371a0e3f00161ea2f0e47b0a554ae13d6d83e06d88bd5fef58883
    supports: [capability.create-edit-template, capability.delete-template]
  - path: backend/production/types/production_template.go
    role: data-model
    hash: sha256:44acdc64119c4ddb125583fe11db79aaee6cb98fa8bc04ee1e6abfef51b8d5bf
    supports: [concepts, capability.create-edit-template, troubleshooting]
  - path: backend/production/types/production_order_rules.go
    role: business-logic
    hash: sha256:17687af36766228e14a12cf1357c803a7a7e1968f8038bb057f0a68beeec2a4d
    supports: [capability.create-edit-template, troubleshooting]
```
