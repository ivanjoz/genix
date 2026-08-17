---
schema: 1
page_id: sales.shipping-costs
route: /sales/shipping-costs
title: Shipping Costs (Costos de Envío)
status: implemented
visibility: tenant
description_en: >-
  Shipping rate configuration per geographic zone. Set a flat rate and a per-kilogram rate for
  each department, province, and district; filter zones by name and save every edited zone in
  one batch.
description_es: >-
  Configuración de tarifas de envío por zona geográfica. Establecer una tarifa fija y una tarifa
  por kilogramo para cada departamento, provincia y distrito; filtrar zonas por nombre y guardar
  todas las zonas editadas en un solo lote.
---

# Shipping Costs (Costos de Envío)

<!-- DOC-ID: page-purpose -->
## Page purpose

Shipping Costs (`Costos de Envío`, shown inside the page itself as `Delivery Costs` /
`Costos de Delivery`) is where a company configures how much it charges to deliver to each
geographic zone (`zona`): department (`departamento`), province (`provincia`), and district
(`distrito`). For every zone it stores two independent numbers, a flat rate (`Fijo`) and a
per-kilogram rate (`Por Kg`), and lets the user filter zones and save several edited zones in
one operation.

This page only stores the configured rates. As of this review, no other confirmed workflow in
Genix (sale orders, point of sale, or storefront checkout) reads this table to calculate a
sale's shipping charge automatically — the page is a rate catalog, not a live pricing engine.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A **shipping zone (`zona de envío`)** is one entry from Peru's national geography catalog
  (the same `departamento` → `provincia` → `distrito` list used to pick a branch's location on
  **Sites & Warehouses**). The frontend requests this catalog for a fixed country (Peru); this
  page has no country selector.
- A **flat rate (`Fijo`)** is a fixed shipping price for a zone regardless of weight. A
  **per-kilogram rate (`Por Kg`)** is meant to be multiplied by an order's weight elsewhere;
  this page stores the number but performs no such multiplication itself.
- Each zone keeps its own **independent** flat/per-kg pair, keyed by that zone's own numeric ID.
  Genix applies no cascade or inheritance between levels: configuring a department's rate does
  not pre-fill, override, or fall back to its provinces' or districts' rates, and vice versa —
  every province and every district needs its own value if it is to have one.
- A rate of `0` (or an emptied cell) is treated as **not configured**: it renders as `-` and is
  excluded from the min–max range summaries described below.

<!-- DOC-ID: capability.browse-zones -->
## Browse and filter zones (Buscar y filtrar zonas)

### User intention (Intención del usuario)

Quickly locate a department, province, or district to check or edit its configured shipping
rate.

### Where to find it (Dónde encontrarlo)

Open **Commercial (Comercial) → Shipping Costs (Costos de Envio)** at `/sales/shipping-costs`
(the page's own header reads `Delivery Costs` / `Costos de Delivery`). The left panel shows one
card per department; use the **Filter (Filtrar)** box above it to narrow the list.

### Business rules and rationale (Reglas y razón de negocio)

Filtering matches department, province, and district names against the already-loaded
geography list — it is not a server search — and ignores accents/diacritics, so typing
`junin` matches `Junín`. A department card stays visible if its own name matches, or if any of
its provinces or districts match. Each department card shows a badge with its own flat rate,
plus the min–max range of the configured (non-zero) flat rates among its direct provinces and
among all its districts (a single configured value is shown as that value on both sides of the
range).

When a filter narrows the visible departments down to 1–6 matches, Genix automatically opens
(selects) the first match in the right-hand panel instead of requiring an extra click.

Selecting a department opens the right panel: department-level rate fields plus a
province/district table. Clicking a province row expands its districts inline in the same
table; clicking it again collapses them.

### Limitations (Limitaciones)

Only Peru's geography catalog is available; there is no way on this page to switch to another
country's departments/provinces/districts.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo busco la tarifa de envío de un departamento, provincia o distrito?`
- `¿Por qué al buscar se abrió automáticamente un departamento?` Porque el filtro dejó pocas
  coincidencias (1 a 6) y Genix abre la primera automáticamente.
- Search terms: `zona de envío`, `departamento`, `provincia`, `distrito`, `filtrar`, `buscar`.

<!-- DOC-ID: capability.set-shipping-rates -->
## Set flat and per-kilogram rates (Establecer tarifa fija y tarifa por kilo)

### User intention (Intención del usuario)

Configure how much Genix should charge to ship to a specific department, province, or district,
as a flat fee, a per-kilogram fee, or both.

### Where to find it (Dónde encontrarlo)

- **Department level:** with a department selected, use the **Fijo** and **Por Kg** input
  fields at the top of the right panel (disabled until a department is chosen).
- **Province/district level:** edit the **Fijo** and **Por Kg** columns directly in the
  province/district table rows (inline numeric cell editing); expand a province to reach its
  districts.
- Use the **Save (Guardar)** button in the left panel's toolbar to send every pending edit.

### Required information and prerequisites (Requisitos previos)

An existing zone from the geography catalog (departments, provinces, and districts already
exist as reference data; nothing needs to be created first). Values are plain numbers; a
cleared or non-numeric cell normalizes to `0`.

### Business rules and rationale (Reglas y razón de negocio)

Edits are kept locally in the browser first and only reach the server when **Guardar** is
pressed; typing into a cell or the department fields does not save immediately. Guardar sends
only the zones that changed since the last successful save, in a single request.

The server independently rejects the whole batch if any row has an invalid/missing `CityID`, a
negative `Fijo` or `Por Kg`, or more than one row for the same city in the same request. Saving
values that are identical to what is already stored does not create a new update: Genix keeps
the previous save timestamp when a zone's `Fijo`/`Por Kg` did not actually change, avoiding
write churn.

### Result and side effects (Resultado y efectos)

Saving creates or updates one shipping-cost record per edited zone (rate values, saving user,
and update time); zones that were not edited keep their previous configuration untouched. After
a successful save, the page also reloads its zone-cost list from the server.

### Limitations (Limitaciones)

- Unsaved edits exist only in the current browser tab; reloading the page or navigating away
  before pressing **Guardar** loses them. A background refresh of the underlying data (for
  example while the tab stays open) does not overwrite a cell that has already been edited but
  not yet saved.
- There is no delete action; the only way to "remove" a configured rate is to set it back to
  `0`, which then displays as `-` like an unconfigured zone.
- Setting a department's rate does not propagate to its provinces or districts, and configuring
  a province/district does not derive from its department; see **Business concepts** above.
- As noted in **Page purpose**, no other confirmed Genix workflow currently reads these values
  to compute an order's shipping charge automatically.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo configuro el costo de envío por departamento, provincia o distrito?`
- `¿Por qué el costo del departamento no se aplica a sus provincias o distritos?` Porque cada
  zona guarda su propia tarifa; Genix no hereda ni distribuye el costo entre niveles.
- `¿Este costo se aplica automáticamente a mis ventas?` Actualmente no hay una integración
  confirmada que use esta tabla para calcular el flete de un pedido.
- `¿Cómo elimino una tarifa?` No existe eliminar; poner la tarifa en 0 la deja como no
  configurada (`-`).
- Search terms: `costo de envío`, `flete`, `tarifa fija`, `tarifa por kilo`, `costos de
  delivery`, `guardar cambios`.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

Viewing this page's zone list requires only being a logged-in user of the company: the read
endpoint (`GET.shipping-costs`) has no access mapped in the permission catalog, so Genix allows
any authenticated company user to open the page and see every zone's configured rate. Saving is
the restricted operation: the write endpoint (`POST.shipping-costs`) is gated behind the
"Costos de Envío" catalog access (access id 13, module group `Comercial`), and that access only
offers two levels — **View (Visualizar)** and **Full (Todo)** — with no separate Create/Edit
tier. Because the server requires at least the second level to save, only a user whose profile
or individual access grants **Full (Todo)** on "Costos de Envío" can actually save changes here;
View-only access lets someone open the page but not edit rates.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"El costo de envío en posición N debe tener CityID válido.":** one of the edited rows lost
  its zone reference; reload the page and reapply the edit.
- **"El costo de envío en posición N no puede tener costos negativos.":** a negative number was
  typed into `Fijo` or `Por Kg`; correct it before saving.
- **"Se envió más de un costo de envío para la misma ciudad.":** the same zone appears twice in
  the pending batch; reload the page and re-enter the change.
- **"El user no posee alguno de los accesos: Costos de Envío":** the acting user's profile only
  has (at most) the View level for this access; ask an administrator to grant the Full (Todo)
  level for "Costos de Envío" on **Profiles & Access**.
- **An edited cell reverted after leaving the page:** the edit was never saved with **Guardar**;
  unsaved edits are not kept once the page is left or reloaded.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Sites & Warehouses (Sedes & Almacenes)** at `/business/branches-warehouses` uses the same
  Peru department/province/district geography catalog to place a branch, though it does not
  read or write shipping rates.
- **Profiles & Access (Perfiles & Accesos)** grants the "Costos de Envío" access (View or Full)
  that controls who can save changes on this page.

### FILES

```yaml
# Exact source hashes captured after claim-by-claim review.
schema: 1
hash_algorithm: sha256
files:
  - path: frontend/core/modules.ts
    role: user-interface
    hash: sha256:0839d4ae72db6d7a902b99dae286edd5be0a16d691543ee7c1643e61fb4bf014
    supports: [page-purpose, capability.browse-zones, related-pages]
  - path: frontend/routes/sales/shipping-costs/+page.svelte
    role: page
    hash: sha256:db3925655205f745c94ca53ac2cc434d4002338d88923f5bc032ca0ef6b4bf3d
    supports: [page-purpose, concepts, capability.browse-zones, capability.set-shipping-rates, rules, troubleshooting]
  - path: frontend/routes/business/branches-warehouses/branches-warehouses.svelte.ts
    role: frontend-service
    hash: sha256:8f3a3fdc8dc47344ada0fc5f56361026effe8099d44c68c9a6ef54a68e488302
    supports: [concepts, capability.browse-zones, related-pages]
  - path: backend/sales/shipping_costs.go
    role: backend-handler
    hash: sha256:a68c0cc9312a1def76bb21205d1594c04e59c39fd48d911c7b403dbecaa008a7
    supports: [capability.set-shipping-rates, rules, troubleshooting]
  - path: backend/sales/types/shipping_costs.go
    role: data-model
    hash: sha256:09d63aa2f7699d50cf332f8607fff248b1596e813b51c4d22e7f06c2ea0e14c3
    supports: [concepts, capability.set-shipping-rates]
  - path: backend/business/locations-warehouses.go
    role: backend-handler
    hash: sha256:591db493e761ac7d062bca3889e7ff4b3f6e85ced92017621658957d8742af39
    supports: [concepts, capability.browse-zones]
  - path: backend/business/types/generales.go
    role: data-model
    hash: sha256:9d96bb2ae3bc6a5d8626bdc4a6a6348e63ddb97072434bde3403ec70b3178789
    supports: [concepts]
  - path: backend/access_list.yml
    role: permissions
    hash: sha256:0c00cfb3e7af9a918eb753846874ac7d213f1eacb3e46bded4049016e1c57951
    supports: [rules]
  - path: backend/main-handlers.go
    role: permissions
    hash: sha256:2474ede3472c063c1e28ea584cd6265dc4b7b8437231cfa94aa5671e66b62330
    supports: [rules]
  - path: frontend/routes/security/access-profiles/access-profiles.svelte.ts
    role: shared-domain
    hash: sha256:34269127c34cd8b1964aa51143315a914bd52b1f98ab98469c9dff70e195c72f
    supports: [rules]
  - path: frontend/routes/security/access-profiles/+page.svelte
    role: shared-domain
    hash: sha256:f113cbaee8d9ad9f180f07993ce42054dde14523c91e38075b9178b1723f7a55
    supports: [rules]
```
