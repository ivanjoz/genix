---
schema: 1
page_id: sales.sale_planning
route: /sales/sale_planning
title: Sales Planning (Proyección Ventas)
status: implemented
visibility: tenant
description_en: >-
  Manual sales plan per product for the 52 weeks of the year: a base weekly quantity, optional
  quantities for specific weeks, and a reusable seasonality curve of weekly percentages. Products
  tab to plan each product; Seasonality tab to create, edit and delete curves.
description_es: >-
  Plan manual de ventas por producto para las 52 semanas del año: cantidad base semanal, cantidades
  para semanas específicas y una curva de estacionalidad reutilizable con porcentajes por semana.
  Pestaña Productos para planificar cada producto; pestaña Estacionalidad para crear, editar y
  eliminar curvas.
---

# Sales Planning (Proyección Ventas)

<!-- DOC-ID: page-purpose -->
## Page purpose

Sales Planning (`Proyección Ventas`, `plan de ventas`) records, by hand, how many units of each
product the company expects to sell each week of a year. A plan combines a base weekly quantity
(`cantidad base semanal`), quantities typed for particular weeks, and a seasonality curve
(`curva de estacionalidad`) that raises or lowers chosen weeks by a percentage.

The plan is a repeating yearly pattern of 52 weeks, not tied to a specific year. It is entered by the
user; Genix does not calculate it from past sales. The automatic forecast from the sales history is
the **Sales Forecast (Pronóstico de Ventas)** page, which does not read these plans. The page tab
**Report (Reporte)** shows only "Próximamente" (coming soon): there is no planning report yet.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- **Week (Semana):** weeks 1 to 52 of the year. Week N starts on day (N−1)×7 of the year, and the
  editors group the weeks under the month their first day falls in (ENE, FEB, …).
- **Seasonality curve:** a named list of weekly percentages that can be assigned to many products.
  100 means no change and 150 means +50%. Only some weeks need a value: an empty week takes the value
  of the previous filled week, and the weeks before the first filled one count as 100.
- **Planned quantity of a week** = the curve's percentage of that week × (the quantity typed for that
  week, or the base quantity when the week is empty). A typed week replaces the base, but the curve
  still applies on top of it.

<!-- DOC-ID: capability.plan-product -->
## Plan a product's weekly sales (Planificar un producto)

### User intention (Intención del usuario)

Set the expected weekly sales of a product, for example to anticipate purchases or production for
seasonal products.

### Where to find it (Dónde encontrarlo)

**Commercial (Comercial) → Sales (Ventas) → Sales Planning (Proyección Ventas)** at
`/sales/sale_planning`, page tab **Planning (Planificación)**, view **Products (Productos)**. Use
**Filter products (Filtrar productos)** and click the product; its side panel opens with the product
name as title. Save with the panel's save button.

### Required information and prerequisites (Requisitos previos)

- **Base weekly quantity (Cantidad base semanal):** the usual units per week; it cannot be negative.
- **Seasonality curve (Curva de estacionalidad):** optional, **None (Ninguna)** by default. Create
  curves first in the **Seasonality (Estacionalidad)** view.
- The 52 week boxes, under the note **Weekly quantity (leave empty to use the base) (Cantidad por
  semana (vacío = usa la base))**: optional quantities for specific weeks.

### Business rules and rationale (Reglas y razón de negocio)

- A negative base is refused with "La cantidad base no puede ser negativa".
- Only weeks with a quantity above zero are kept; an empty or zero week uses the base.
- While editing, a colored bar under each week shows its resulting quantity: green below the base,
  yellow at the base, orange to red above it.
- The page keeps one plan per product; opening a product with a plan edits that plan.

### Result and side effects (Resultado y efectos)

The products table shows **Base Qty (Cant. Base)**, the **Seasonality (Estacionalidad)** curve name
and **Weekly quantity (Cantidad semanal)** as 52 small bars. Nothing else changes: no stock, purchase,
production or forecast is created or updated from the plan.

### Limitations (Limitaciones)

- A product plan has no delete button; only seasonality curves can be deleted.
- The plan has no year and no sites: one yearly pattern per product for the whole company.
- There is no report of the plans yet (**Report** tab: coming soon).

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo registro cuánto planeo vender por semana? ¿Cómo hago mi proyección de ventas?
- ¿Cómo pongo más ventas en diciembre? → type those weeks or assign a seasonality curve.
- Términos: proyección de ventas, plan de ventas, presupuesto de ventas, meta semanal,
  estacionalidad, temporada alta, temporada baja.

<!-- DOC-ID: capability.seasonality-curve -->
## Manage seasonality curves (Curvas de estacionalidad)

### User intention (Intención del usuario)

Describe once how a group of products varies over the year (campaigns, high and low season) and
reuse it in many product plans.

### Where to find it (Dónde encontrarlo)

Page tab **Planning (Planificación)**, view **Seasonality (Estacionalidad)**. **New (Nueva)** opens
**New curve (Nueva curva)**; clicking a curve opens it for editing. **Filter curves (Filtrar curvas)**
searches by name. The panel's delete button removes a saved curve after confirming **Delete curve
(Eliminar curva)**.

### Required information and prerequisites (Requisitos previos)

- **Name (Nombre):** at least 2 characters.
- At least one week with a percentage, under the note **Weekly percentage (100 = no change, 150 =
  +50%). Empty weeks inherit the previous filled week.** Each empty box shows, as placeholder, the
  percentage it inherits.

### Business rules and rationale (Reglas y razón de negocio)

- Refused without a name ("La curva necesita un nombre") or without any filled week ("Complete al
  menos una semana").
- The table column **Filled weeks (Semanas)** draws the resulting 52-week curve, labeling the weeks
  that have their own value; low weeks are green and peak weeks red.

### Result and side effects (Resultado y efectos)

A saved curve becomes selectable in every product plan. Editing a curve changes the planned
quantities of every product that uses it.

### Limitations (Limitaciones)

A curve is a percentage only; it holds no quantities by itself and has no effect until a product plan
selects it.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo creo una temporada alta para varios productos? ¿Qué significa 150 en la curva?
- ¿Cómo bajo las ventas esperadas en enero? → a percentage under 100 for those weeks.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"La cantidad base no puede ser negativa":** enter 0 or a positive base.
- **"La curva necesita un nombre" / "Complete al menos una semana":** name the curve and fill at
  least one week.
- **A product shows no bars:** it has no plan, or its base and weeks are all empty.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Sales Forecast (Pronóstico de Ventas):** the automatic demand forecast from recorded sales, stock
  and the commercial calendar; use it to see expected demand calculated by Genix.
- **Products (Productos):** the products listed here come from the product catalog.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/pages.toml
    role: user-interface
    hash: sha256:d435afce0db0eedf924345040be60e0f4d1a15672e0f1f2d9003687920fb3da7
    supports: [page-purpose, capability.plan-product, related-pages]
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.plan-product]
  - path: frontend/routes/sales/sale_planning/+page.svelte
    role: page
    hash: sha256:a9d44e7776a77d03c6511aaf2f9b4e3281b619ccde2c4d45957c8a736a07d6b7
    supports: [page-purpose]
  - path: frontend/routes/sales/sale_planning/SalesPlanning.svelte
    role: page
    hash: sha256:e40bdce62321088e90d2f43e1b65eab80ddcd7d261c5090e5dbdb7e3d6f522c9
    supports: [page-purpose, capability.plan-product]
  - path: frontend/routes/sales/sale_planning/SalesPlanningMantainer.svelte
    role: user-interface
    hash: sha256:ac954470858e2dd7c63f80297a2ae8d46b50f48f1faa61446c5d50bf4f2282b8
    supports: [concepts, capability.plan-product, capability.seasonality-curve, troubleshooting, related-pages]
  - path: frontend/routes/sales/sale_planning/sale_planning.svelte.ts
    role: frontend-service
    hash: sha256:6a333180f664441b21ca1c76dcd30e3849ff430492e5a94db65ec1a44caa1bb3
    supports: [concepts, capability.plan-product, capability.seasonality-curve]
  - path: backend/sales/sales_planning.go
    role: backend-handler
    hash: sha256:99c6b1fd2275b126224916e084933eb502f7e33860a2f90719136d8a16c80839
    supports: [capability.plan-product, capability.seasonality-curve]
  - path: backend/sales/types/sales_planning.go
    role: data-model
    hash: sha256:725910c8a7fc64b0a57b2862b776c8483e230f72c34c161b5e62f4cb089a5df7
    supports: [concepts, capability.plan-product, capability.seasonality-curve]
  - path: frontend/routes/sales/sales-forecast/sales-forecast.svelte.ts
    role: frontend-service
    hash: sha256:0cb127773179e6d1e0ec0bc580ee1c7b215a0f44151ef112a44c1b592c585e64
    supports: [page-purpose, related-pages]
```
