---
schema: 1
page_id: production.settings
route: /production/settings
title: Production Settings (Configuración de producción)
status: implemented
visibility: tenant
description_en: >-
  Define the labor types recipes and production orders plan in minutes, each with an estimated
  cost per hour, and the stages that make the columns of the production orders board.
description_es: >-
  Definir los tipos de trabajo que las recetas y las órdenes de producción planifican en minutos,
  cada uno con su costo por hora estimado, y las etapas que forman las columnas del tablero de
  órdenes de producción.
---

# Production Settings (Configuración de producción)

<!-- DOC-ID: page-purpose -->
## Page purpose

Production settings (`Configuración de producción`) holds two small catalogs the production pages
use:

- **Labor types (Tipos de trabajo)**, such as `Zapatero` or `Maquinista`: the kinds of work a
  recipe plans in minutes, with an estimated cost per hour (`costo por hora`).
- **Stages (Etapas)**, such as `Corte`, `Aparado`, `Armado`: the columns of the production orders
  board (`tablero`).

The page does not log the hours actually worked (that is the **Labor (Mano de obra)** tab of a
production order) and does not set what each worker costs (that comes from the worker's contract in
Employees & Contracts / Empleados & Contratos).

<!-- DOC-ID: capability.labor-types -->
## Manage labor types (Gestionar tipos de trabajo)

### User intention (Intención del usuario)

List the kinds of work production uses so recipes can plan minutes of each and estimate their
labor cost.

### Where to find it (Dónde encontrarlo)

**Operations (Operaciones) → Production (Producción) → Production settings (Configuración de
producción)** at `/production/settings`, tab **Labor types (Tipos de trabajo)**. The list shows
**ID**, **Labor type (Tipo de trabajo)** and **Cost / hour (Costo / hora)** and can be filtered by
name. **New (Nuevo)** opens **New labor type (Nuevo tipo de trabajo)**; click a row to edit it.

### Required information and prerequisites (Requisitos previos)

- **Name (Nombre):** at least 2 characters.
- **Cost per hour (estimate) (Costo por hora (estimado)):** cannot be negative; 0 is allowed (the
  type then plans hours without estimating their cost).

### Business rules and rationale (Reglas y razón de negocio)

As the form says, the cost per hour is used to estimate the cost of recipes and orders; the labor
booked to products uses each worker's real cost per hour. Specifically it is used for:

- the estimated unit cost on a recipe's **Producción** tab and a template's estimated series cost;
- the planned labor cost frozen on a production order when it is released;
- the recipe labor the Daily close (Cierre del día) adds at standard, since that close logs no
  hours;
- the rate of a worker off payroll who has no agreement, when hours are logged on an order.

### Result and side effects (Resultado y efectos)

The labor type is offered in recipes and in the hours of production orders. Changing its cost per
hour does not change orders already released.

### Limitations (Limitaciones)

Deleting a labor type (the panel's delete button) keeps the recipe lines that name it, but those
recipes cannot be saved again until the line is removed, and no new hours can be logged with it.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Dónde creo los tipos de mano de obra? ¿Dónde pongo cuánto cuesta la hora de un zapatero?
- Puesto, oficio, operación, centro de trabajo, tarifa por hora, costo hora-hombre.

<!-- DOC-ID: capability.stages -->
## Manage board stages (Gestionar etapas)

### User intention (Intención del usuario)

Define the steps of the shop floor so released orders can be followed on the board.

### Where to find it (Dónde encontrarlo)

Tab **Stages (Etapas)**: the list shows **Position (Posición)** and **Stage (Etapa)**. **New
(Nuevo)** opens **New stage (Nueva etapa)** with **Name (Nombre)** and **Position (Posición)**.

### Business rules and rationale (Reglas y razón de negocio)

- The board shows one column per stage, ordered by position, left to right.
- A new stage is proposed 10 positions after the last one, so another can be inserted in between
  later.
- The name needs at least 2 characters and the position cannot be negative.
- A stage is information only: moving an order between stages moves no stock and changes no cost.

### Result and side effects (Resultado y efectos)

The stage becomes a column of the **Board (Tablero)** view in Production orders and an option of
the order's **Stage (Etapa)** field.

### Limitations (Limitaciones)

Deleting a stage leaves its orders under **No stage (Sin etapa)** on the board until they are moved.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo agrego columnas al tablero de producción? ¿Cómo cambio el orden de las etapas? → edit the
  position.
- Etapa, fase, proceso, estación, columna del kanban.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"El nombre del tipo de trabajo debe tener al menos 2 caracteres."** / **"El nombre de la etapa
  debe tener al menos 2 caracteres."** Use a longer name.
- **A recipe no longer saves after deleting a labor type** ("el tipo de trabajo … de la receta no
  existe o no está activo"): remove that labor line from the recipe.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Production orders (Órdenes de producción):** the board by stage and the hours logged per labor
  type.
- **Products (Productos)** and **Supplies & Materials (Insumos & Materiales):** recipes that plan
  minutes per labor type.
- **Hours vs payroll (Horas vs planilla):** hours logged on orders compared with payroll.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.labor-types]
  - path: db/pages.toml
    role: user-interface
    hash: sha256:d435afce0db0eedf924345040be60e0f4d1a15672e0f1f2d9003687920fb3da7
    supports: [page-purpose, related-pages]
  - path: frontend/routes/production/settings/+page.svelte
    role: page
    hash: sha256:099df0f6e0ab94e5006c23f904d14a748016c0d8a24841231e65ff8ee194f97e
    supports: [page-purpose, capability.labor-types, capability.stages]
  - path: frontend/routes/production/settings/LaborTypesTab.svelte
    role: user-interface
    hash: sha256:0222d7a9d5ad9e0b77ea0564b04fef2948a2ee7837e659f24da77358987c34e1
    supports: [capability.labor-types]
  - path: frontend/routes/production/settings/StagesTab.svelte
    role: user-interface
    hash: sha256:b05c9192aa80249bfc2ab8538a8ecf232a5b269291f5dd9601b6f6fcaec85b41
    supports: [capability.stages]
  - path: frontend/services/production/production-settings.svelte.ts
    role: frontend-service
    hash: sha256:6cf6bcc03800775f82c7d7181622709b2c69dc6996d70cb794ba431b88653214
    supports: [capability.labor-types, capability.stages]
  - path: frontend/routes/production/orders/OrderBoard.svelte
    role: user-interface
    hash: sha256:917e76d173bc929837dd9a7880483fe1826b1c8acd289e784d24df59ba2612b6
    supports: [capability.stages, related-pages]
  - path: frontend/routes/production/orders/OrderLayer.svelte
    role: user-interface
    hash: sha256:4a4bde050b0a7bbd17913252b2a69fefad3bc25285eca40e6ea66fd3ed29331c
    supports: [capability.stages]
  - path: frontend/routes/production/products/ProductProductionTab.svelte
    role: user-interface
    hash: sha256:396cea9f2101672b7e390f136ef6cfc6e9f2594c3bb7cf7db7c0b6831f467035
    supports: [capability.labor-types, related-pages]
  - path: frontend/routes/production/templates/production-template.ts
    role: business-logic
    hash: sha256:333e01794d3491ee446a058c4f04a632acf13623ca203d00b80be5cb65f52584
    supports: [capability.labor-types]
  - path: backend/production/production_settings.go
    role: backend-handler
    hash: sha256:d7041a2c9ffb4538868a4fa2eee6bf25d2683dd264826df9096cec8a40367327
    supports: [capability.labor-types, capability.stages, troubleshooting]
  - path: backend/production/types/production_settings.go
    role: data-model
    hash: sha256:59e0240726b80da6d0ce79d8799afd9c78cef72a6c47aacbf46e3e086cc4b50f
    supports: [capability.labor-types, capability.stages, troubleshooting]
  - path: backend/production/production_orders.go
    role: backend-handler
    hash: sha256:d6e99162ca6371a0e3f00161ea2f0e47b0a554ae13d6d83e06d88bd5fef58883
    supports: [capability.labor-types, capability.stages]
  - path: backend/production/types/production_order_rules.go
    role: business-logic
    hash: sha256:17687af36766228e14a12cf1357c803a7a7e1968f8038bb057f0a68beeec2a4d
    supports: [capability.labor-types]
  - path: backend/production/types/daily_close.go
    role: business-logic
    hash: sha256:066fccc9aa5e9d7bb834c32bb26d5d2b1e7c5960f5e34c56f0a1e6c415e83a7b
    supports: [capability.labor-types]
  - path: backend/production/production_labor.go
    role: backend-handler
    hash: sha256:8e1de7d581d752f00c10882fd64198bfad95b25e5175486eadb78a1b7732173b
    supports: [capability.labor-types]
  - path: backend/production/types/product_production.go
    role: business-logic
    hash: sha256:9264a245d159a927e3b7e6716daff8597dd0079357149792612d24f249ab223f
    supports: [capability.labor-types, troubleshooting]
  - path: backend/hr/types/labor_cost.go
    role: business-logic
    hash: sha256:65feaa94e6f2db9b0aa6225194be57dd5df28ff655d2da28cfbaf7bf257af1b0
    supports: [page-purpose, capability.labor-types]
```
