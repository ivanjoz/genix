---
schema: 1
page_id: sales.calendar-events
route: /sales/calendar-events
title: Commercial Calendar (Calendario Comercial)
status: implemented
visibility: tenant
description_en: >-
  Commercial calendar of dates that move sales: holidays, paydays, gratificación and commercial
  dates. Shows the national holidays shared by every company next to the company's own events.
  Create, edit and delete company events. The Sales Forecast learns from these dates and Cash Flow
  uses the paydays and holidays.
description_es: >-
  Calendario comercial con las fechas que mueven las ventas: feriados, días de pago, gratificación y
  fechas comerciales. Muestra los feriados nacionales compartidos por todas las empresas junto con
  los eventos propios de la empresa. Crear, editar y eliminar eventos de la empresa. El Pronóstico de
  Ventas aprende de estas fechas y el Flujo de Caja usa los días de pago y los feriados.
---

# Commercial Calendar (Calendario Comercial)

<!-- DOC-ID: page-purpose -->
## Page purpose

The Commercial Calendar (`Calendario Comercial`) is the list of dates that move a company's sales:
holidays (`feriados`), paydays (`días de pago`, `quincena`, `fin de mes`), the gratificación and
commercial dates (`fechas comerciales`, such as campaigns). One table shows two kinds of events
together, ordered by date: the global holidays (`feriados globales`) every company shares, and the
company's own events.

The page only records dates. It does not compute a forecast or a cash projection itself: the
**Sales Forecast (Pronóstico de Ventas)** page reads these dates to learn how sales change around
them, and the **Cash Flow (Flujo de Caja)** page reads the paydays and holidays to date payroll
payments.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- An **event** (`evento`) is one date with a **Type (Tipo)** and a **Name (Nombre)**. Each event is a
  single day; there is no yearly repetition, so each year's dates are entered as their own events.
- The event types are **Holiday (Feriado)**, **Payday (Día de pago)**, **Gratificación** and
  **Commercial date (Fecha comercial)**.
- The **Scope (Alcance)** column says whose event it is: **Global** for the national holidays shared
  by every company, **Company (Empresa)** for events this company created.
- Global holidays can be read by every company but only the system super-administrator can create
  or change them.

<!-- DOC-ID: capability.create-event -->
## Add a company event (Agregar un evento)

### User intention (Intención del usuario)

Record the company's paydays, gratificación dates, campaigns and other commercial dates, or a
holiday the national calendar does not have, so the forecast and the cash flow take them into
account.

### Where to find it (Dónde encontrarlo)

**Commercial (Comercial) → Catalog (Catálogo) → Commercial Calendar (Calendario Comercial)** at
`/sales/calendar-events`. Press **New (Nuevo)**; the side panel **New event (Nuevo evento)** opens.
Save with the panel's save button.

### Required information and prerequisites (Requisitos previos)

- **Date (Fecha):** required.
- **Type (Tipo):** required; a new event starts as **Payday (Día de pago)**.
- **Name (Nombre):** required, at most 120 characters. Spaces at the start and end are removed.

### Business rules and rationale (Reglas y razón de negocio)

- Missing values are refused with **"Seleccione una fecha."**, **"Seleccione el tipo de evento."** or
  **"Ingrese un nombre."**.
- An event the company creates always belongs to the company (scope **Empresa**). Only the
  super-administrator sees the checkbox **Global holiday (every company) (Feriado global (todas las
  empresas))**, and only while creating a new event.

### Result and side effects (Resultado y efectos)

The event appears in the table at its date. From then on the Sales Forecast and Cash Flow read it
the next time they are calculated (see **How other pages use the calendar**).

### Limitations (Limitaciones)

- There is no recurring event: a payday every 15th or every month end must be entered once per date.
- A company user cannot add a global holiday; it is added to the company only.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Dónde registro los días de pago de mi empresa? ¿Cómo agrego la quincena o el fin de mes?
- ¿Cómo registro una campaña o fecha comercial (Día de la Madre, Navidad, Cyber)?
- ¿Cómo agrego un feriado regional o un día no laborable que no aparece?
- Términos: calendario comercial, feriado, día de pago, gratificación, fecha comercial, campaña,
  evento.

<!-- DOC-ID: capability.edit-delete-event -->
## Edit or delete an event (Editar o eliminar un evento)

### User intention (Intención del usuario)

Correct the date, type or name of an event, or remove an event entered by mistake.

### Where to find it (Dónde encontrarlo)

Click the event's row in the table; the side panel opens with the event's name as title. Change the
fields and save, or use the panel's delete button and confirm **Delete event (Eliminar evento)**. The
search box above the table filters events by name.

### Business rules and rationale (Reglas y razón de negocio)

- Date, type and name can be changed on a company event. Its scope cannot change after it is saved.
- A **Global** holiday opens read-only for a company user, with the note that global holidays are
  shared by every company and only the administrator can edit them. Neither save nor delete is
  offered.

### Result and side effects (Resultado y efectos)

A deleted event leaves the table and stops counting for the Sales Forecast and Cash Flow.

### Limitations (Limitaciones)

A company cannot hide or remove a global holiday for itself.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Por qué no puedo editar un feriado? → it is a global holiday; only the administrator edits it.
- ¿Cómo borro un evento del calendario?

<!-- DOC-ID: capability.calendar-effects -->
## How other pages use the calendar (Cómo se usa el calendario)

### User intention (Intención del usuario)

Understand what the calendar changes, and which dates must be loaded for the forecast and the cash
flow to be right.

### Where to find it (Dónde encontrarlo)

The effects show on **Sales Forecast (Pronóstico de Ventas)** and **Cash Flow (Flujo de Caja)**; this
page only holds the dates.

### Business rules and rationale (Reglas y razón de negocio)

- **Sales Forecast:** the forecast learns, per product category, how much sales change on each event
  type and applies it to the coming dates. That learning applies only to products with at least 180
  days of history. When one day has several events, only the strongest counts, in this order:
  Feriado, Gratificación, Fecha comercial, Día de pago.
- The Sales Forecast warns when the calendar is incomplete: **Years without holidays (Años sin
  feriados)**, **No paydays (Sin días de pago)** in the history or the horizon, and **No company
  events (Sin eventos de la empresa)** when only the national holidays exist.
- **Cash Flow:** payroll net pay is projected on the company's **Día de pago** of each month. Only the
  company's own paydays count, not global events. A month without a payday is paid on its last day
  and Cash Flow shows the alert "Sin día de pago en el calendario de la empresa". Holidays (global and
  company) are skipped when counting business days, for example for the AFP due date on the fifth
  business day of the next month.

### Limitations (Limitaciones)

The **Gratificación** and **Commercial date** types only feed the Sales Forecast; Cash Flow reads
only paydays and holidays from this calendar.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Para qué sirve el calendario comercial? ¿Por qué el pronóstico dice "Sin días de pago"?
- ¿Por qué el flujo de caja paga la planilla el último día del mes? → load the company's paydays.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"Solo el super-administrador puede editar los feriados globales.":** a global holiday cannot be
  changed by the company. Add a company event instead if the company needs another date.
- **"El nombre del evento no puede superar 120 caracteres.":** shorten the name.
- **The forecast still warns "Años sin feriados":** the holidays of that year are missing from the
  calendar; the forecast reads that year's holidays as ordinary days.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Sales Forecast (Pronóstico de Ventas):** uses every event type to forecast demand and lists the
  calendar warnings.
- **Cash Flow (Flujo de Caja):** uses the company's paydays for the payroll dates and the holidays for
  business-day deadlines.
- **Sales Planning (Proyección Ventas):** a manual weekly plan per product; it does not read this
  calendar.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/pages.toml
    role: user-interface
    hash: sha256:d435afce0db0eedf924345040be60e0f4d1a15672e0f1f2d9003687920fb3da7
    supports: [page-purpose, capability.create-event, related-pages]
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.create-event]
  - path: frontend/routes/sales/calendar-events/+page.svelte
    role: page
    hash: sha256:4ea2c2a7c35ea6be03b6b1ef05299081afa1d57b953b3816d71ac3d8ae71edf7
    supports: [page-purpose, concepts, capability.create-event, capability.edit-delete-event]
  - path: frontend/services/sales/calendar-events.svelte.ts
    role: frontend-service
    hash: sha256:30ef9695b92696527fe97d7c21a3cbfef46c8dc5658b51389297f782a756cfc8
    supports: [concepts, capability.create-event, capability.edit-delete-event]
  - path: backend/sales/calendar_events.go
    role: backend-handler
    hash: sha256:35d0a75682750ed3fb548495311dcb3f4ab098086cfd0f3972a19855efed4726
    supports: [concepts, capability.create-event, capability.edit-delete-event, troubleshooting]
  - path: backend/sales/types/calendar_event.go
    role: data-model
    hash: sha256:85831ab6921806e88a4ba02b566b7b7073aef6b494438c3592bcd7e856bb8d52
    supports: [concepts, capability.create-event, troubleshooting]
  - path: frontend/routes/sales/sales-forecast/sales-forecast.ts
    role: business-logic
    hash: sha256:358eebaf98bc71d874b22b11257982dc874353e0b403997f824d603e94d436ba
    supports: [capability.calendar-effects, troubleshooting, related-pages]
  - path: forecast/src/calendar.rs
    role: business-logic
    hash: sha256:4bbfec3487b7164c15c1f485d05944963cf15bbad6ca6525a41ebfeed11b715b
    supports: [capability.calendar-effects]
  - path: forecast/src/engine.rs
    role: business-logic
    hash: sha256:f98c9e5b881df939a9dc15bf96966cf838c0097085e33b921ac3475fc49718df
    supports: [capability.calendar-effects]
  - path: backend/finance/cash_flow_commitments.go
    role: business-logic
    hash: sha256:642f494dd1491665739525d2db42e59621e75966dc902a9652b58346da0aa7f0
    supports: [capability.calendar-effects, related-pages]
  - path: backend/finance/types/cash_flow.go
    role: business-logic
    hash: sha256:c6bdf483ab369509e76f19d56bf0b79a903a530ae37edbd26c8db92adf7f8be3
    supports: [capability.calendar-effects]
  - path: frontend/routes/finance/cash-flow/cash-flow-warnings.ts
    role: user-interface
    hash: sha256:ca990a0d244a892e677b9519509169482a4fe63cb1567cd5983289eb65bdd615
    supports: [capability.calendar-effects]
  - path: frontend/routes/sales/sale_planning/SalesPlanningMantainer.svelte
    role: user-interface
    hash: sha256:ac954470858e2dd7c63f80297a2ae8d46b50f48f1faa61446c5d50bf4f2282b8
    supports: [related-pages]
```
