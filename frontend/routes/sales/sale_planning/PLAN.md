# Sales Planning — Frontend Plan

## Scope (this iteration)
Frontend module for **Planning** view only. **Report** view is a placeholder tab.

## Menu (modules.ts)
Add to the Commercial section (`id: 3`):
```ts
{ name: "Sales Planning|Proyección Ventas", route: "/comercial/sale_planning", icon: "icon-chart-bar" }
```

## Files (in `routes/comercial/sale_planning/`)

### 1. `+page.svelte` (SvelteKit route entry)
Thin — just renders `<SalesPlanning />`.

### 2. `SalesPlanning.svelte` (page shell)
- `<Page title="Proyección Ventas">` with top-tab `options`:
  - `{ id: 1, name: "Planning|Planificación" }`
  - `{ id: 2, name: "Report|Reporte" }`
- View 1 → `<SalesPlanningMantainer />`
- View 2 → placeholder ("Próximamente").

### 3. `SalesPlanningMantainer.svelte` (the working surface)
`OptionsStrip` with 2 sub-views:
- **(1) Products** — `VTable` of all products (from `ProductosService`). Each row shows product name + whether it has a plan (base qty / has weekly overrides). Row click → side `Layer id={1}`:
  - `BaseQuantity` input.
  - **52 weekly inputs** (`Quantity` per week) in a compact grid; empty = "use base". Saves `SalesPlanning` (ProductID, BaseQuantity, WeeklyQuantity[] only for filled weeks).
- **(2) Seasonality** — `VTable` of `SeasonalityCurve`s. "Nuevo" + row click → side `Layer id={2}`:
  - `Name` input.
  - 52 weekly **Percent** inputs (UI shows decimal e.g. `1.5`, stored as `*1000` int16).
  - **Carry-forward rule:** only filled weeks are stored. A week inherits the most recent earlier filled week's value (week 2 empty + week 1 = 1.5 → week 2 effectively 1.5). I'll add a small `resolveCurve()` helper that forward-fills for display/preview.

### 4. `sale_planning.svelte.ts` (services)
Two `GetHandler` delta-cache services:
- `SalesPlanningService` → route `sales-planning`
- `SeasonalityCurveService` → route `seasonality-curve`
Interfaces `ISalesPlanning`, `ISeasonalityCurve`, `ISalesPlanningWeek {Week,Quantity}`, `ISeasonalityCurveWeek {Week,Percent}`.

## Backend (REQUIRED for persistence — flag)
The frontend services need GET (delta-cache) + POST handlers for `sales_planning` and `seasonality_curve`. The tables already exist. I'll add handlers following the `delta-cache-api` skill in `backend/sales/`. **Confirm you want backend handlers in this iteration** (without them the UI renders but can't load/save).

## Open questions
1. **52 weekly inputs UX** — a 52-cell grid is dense. OK to render as a wrapped grid of small numeric inputs labeled W1..W52 (mobile: scrollable)? Or group by month?
2. **Percent UI unit** — show as multiplier (`1.5` = 150%) and store `1500`. Good?
3. **Backend handlers now or later?**
