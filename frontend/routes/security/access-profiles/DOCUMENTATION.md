---
schema: 1
page_id: security.access-profiles
route: /security/access-profiles
title: Profiles & Access (Perfiles & Accesos)
status: implemented
visibility: tenant
description_en: >-
  Access profile (perfil) management. Create and edit profiles with a name and description, then
  grant each profile View (Visualizar) or Full (Todo) control over the accesses in the system
  catalog by selecting access cards grouped by catalog group.
description_es: >-
  Gestión de perfiles de acceso. Crear y editar perfiles con nombre y descripción, y otorgarle a
  cada uno el nivel Visualizar o Todo sobre los accesos del catálogo del sistema seleccionando
  tarjetas de acceso agrupadas por grupo del catálogo.
---

# Profiles & Access (Perfiles & Accesos)

<!-- DOC-ID: page-purpose -->
## Page purpose

Profiles & Access (`Perfiles & Accesos`) is the administrative page for the reusable access
profiles (`perfiles`) assigned to users elsewhere in Genix. It creates and edits a profile's name
and description, and — separately — decides exactly which accesses (`accesos`) from the system
catalog that profile grants, and at which level (`Visualizar` or `Todo`).

This page does not manage individual users or their per-user overrides; that happens on
**Users (Usuarios)** at `/security/users`, which assigns one or more profiles created here (plus
any individual accesses) to a specific user. This page also does not define the catalog of
accesses itself (their names, which pages/APIs they gate, which levels they offer); that catalog
lives in `backend/access_list.yml` and is shared, read-only input to both this page and Users.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A **profile (perfil)** is a named, reusable bundle of accesses: `Name (Nombre)`,
  `Description (Descripción)`, and the list of accesses/levels it grants. Assigning a profile to a
  user (on the Users page) grants that user every access the profile contains.
- An **access (acceso)** is one entry in the system-wide catalog (`access_list.yml`), such as
  "Usuarios", "Perfiles & Accesos", "Cajas & Bancos", or "Órdenes Compra". Each catalog entry
  declares which levels it offers via a `levels` field (concatenated digits, e.g. `14` = level 1 +
  level 4).
- A **level (nivel)** is one of four defined grant levels: **Visualizar (View)**, **Crear
  (Create)**, **Editar (Edit)**, or **Todo (Full)**. The interface can display all four, but which
  ones are actually selectable for a given access depends only on that access's own `levels`
  value in the catalog — as of the current catalog, every one of its entries is configured with
  `levels: 14`, so in practice **every access on this page currently offers only Visualizar or
  Todo**; no access presently exposes an intermediate Crear/Editar grant.
- Each catalog entry belongs to a **group (grupo)** (`Configuración`, `Negocio`, `Comercial`,
  `Logística`, `Finanzas`, `Tienda`, `Contabilidad`, `System`) used purely to organize the access
  grid into labeled sections on this page; it does not affect what the access unlocks.
- Saving a profile encodes its granted accesses as one packed number per access/level
  (`accesoID * 10 + nivel`) in the profile's `Accesos` list. Access IDs are permanent in the
  catalog and are never renumbered or reused, so a profile's stored grants keep meaning even as
  the catalog grows.

<!-- DOC-ID: capability.browse-profiles -->
## Find a profile (Buscar un perfil)

Open **Configuration (Configuración) → Profiles & Access (Perfiles & Accesos)** at
`/security/access-profiles`. The left table lists every active profile (ID, Name) for the
company; only profiles with an active status are ever loaded or shown, matching the same active-
only listing used by the profile selector on the Users page. Use the filter box above the table to
narrow the list by name; the filter matches locally against the already-loaded list, not a
server-side search. Selecting a row loads that profile into the Access panel on the right.

<!-- DOC-ID: capability.create-edit-profile -->
## Create or edit a profile's name and description (Crear o editar nombre y descripción)

### User intention (Intención del usuario)

Create a new reusable profile before assigning users to it (on the Users page), or rename/update
the description of an existing profile without touching what it grants.

### Where to find it (Dónde encontrarlo)

On `/security/access-profiles`, use the green create button (top of the left table's toolbar) to
open a form modal for a new profile. To edit an existing profile's name/description, click its
row's "..." action to open the same modal pre-filled (**Editing|Editando**). Save with the
modal's own save action.

### Required information and prerequisites (Requisitos previos)

- **Name (Nombre)** is required; the only frontend check is that it is non-empty (there is no
  minimum-length rule here, unlike the 4-character rule on the Users page). This check runs on the
  frontend only — the server does not independently re-validate that a name was supplied.
- **Description (Descripción)** is optional free text.

### Business rules and rationale (Reglas y razón de negocio)

Saving through this modal alone does not touch the profile's granted accesses: when editing an
existing profile, the accesses already assigned are carried over unchanged in the request. A new
profile is created with no accesses at all (`Accesos` empty) until the Access panel (below) is
used and saved separately.

### Result and side effects (Resultado y efectos)

Saving creates or updates the profile row. After a successful save, the modal closes and the
right-hand Access panel returns to its unselected state ("Select a profile to edit its access
permissions"), even for a profile just created — the user must click its row again to open it and
assign accesses. If the profile already has users, saving here has no effect on those users'
computed access, since only the accesses/levels list drives that recomputation (see below), not
the name or description.

### Limitations (Limitaciones)

There is no delete action for a profile on this page (the modal component this page uses supports
one, but it is not enabled here). A profile can only be created or edited, never removed from
this page.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo creo un perfil nuevo?`
- `¿Por qué después de crear el perfil ya no veo sus accesos seleccionados?` Porque el panel de
  Accesos se cierra tras guardar; hay que volver a hacer clic en la fila del perfil.
- `¿Cómo elimino un perfil?` Actualmente esta página no ofrece una acción de eliminar perfiles.
- Search terms: `perfil`, `crear perfil`, `editar perfil`, `nombre de perfil`, `descripción`.

<!-- DOC-ID: capability.assign-access -->
## Grant accesses to a profile (Otorgar accesos a un perfil)

### User intention (Intención del usuario)

Decide exactly which pages/capabilities a profile unlocks, and at which level, so every user later
assigned that profile inherits the same grants.

### Where to find it (Dónde encontrarlo)

Select a profile in the left table so its name appears at the top of the right panel
("Access (Accesos) of/de <name>"). The panel shows the full catalog as a grid of access cards
grouped under section headers (the catalog group name, e.g. `Configuración`, `Finanzas`), sorted
by group. Toggle cards/level buttons as described below, then use the panel's own
**Save (Guardar)** button (next to the profile name) — this is a separate action from the
name/description modal's save.

### Required information and prerequisites (Requisitos previos)

A profile must already be selected; if none is selected, the panel shows a red notice asking the
user to select one instead of the access grid. The access catalog itself is bundled with the
frontend build and parsed on page load; if that parsing fails, an amber error banner appears and
no access cards render, though the profile's name/description can still be edited via the modal.

### Business rules and rationale (Reglas y razón de negocio)

Each access card shows only the level buttons that access's catalog entry actually offers (today,
Visualizar and Todo for every access, per the concept above). Two interaction styles both change
the same underlying selection:

- **On desktop**, clicking a specific level icon on a card toggles only that level for that
  access; the whole-card body does not respond to clicks. Clicking a card's border color reflects
  the currently selected level (Todo's color takes priority over Visualizar's when both happen to
  be selected).
- **On mobile/tablet**, tapping anywhere on the card toggles the access on/off using its highest
  available level (Todo when the access offers it, otherwise Visualizar); the individual level
  buttons remain available for a more precise choice.

Selecting more than one level for the same access (for example both Visualizar and Todo) is
allowed by the interface, but is functionally redundant: Genix's access checks only require the
level to be at or above what a route demands, so granting the higher level already covers the
lower one.

Saving this panel requires the acting user to hold the "Perfiles & Accesos" access itself at the
**Todo (Full)** level — because that access only offers Visualizar or Todo, granting someone the
ability to edit profiles on this page always means granting them full control of it. Viewing the
profile list is currently open to any authenticated user of the company, since no read-only access
is mapped for the underlying read endpoint in the catalog.

### Result and side effects (Resultado y efectos)

Saving replaces the profile's stored `Accesos` (encoded access/level pairs) and recomputed
`Modules` list with exactly what the grid currently shows selected — clearing every card and
saving leaves the profile with zero accesses (it still exists, it simply grants nothing).

If the profile already has users, saving also recomputes each affected user's effective access
(`AccesosComputed`, the union of every profile assigned to that user plus their individual
accesses, keeping the highest level per access) and persists it immediately for every user whose
result actually changed; the user does not need to re-save their own record for the new grants (or
removals) to take effect on their next access-checked call.

### Limitations (Limitaciones)

- No access on this page's catalog currently offers a Crear/Editar level distinct from Visualizar
  or Todo; do not expect a partial "can create but not edit" grant to be configurable today.
- The Access panel has no per-module filter or tab in the current interface: state exists
  internally to scope the grid to one module, but nothing on the page sets it, so every group
  from every catalog entry always renders together, ordered by group.
- A handful of catalog entries (for example "Facturación", "Estados Financieros", "Balance",
  "Activos") currently have no linked frontend route or backend API; toggling them on a profile
  changes only the stored grant and has no visible effect anywhere else in Genix yet.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo le doy a un perfil acceso a una página del sistema?` Selecciona el perfil y activa la
  tarjeta de ese acceso en el panel de la derecha, luego guarda con el botón del panel.
- `¿Por qué sólo veo "Visualizar" y "Todo" en las tarjetas?` Porque el catálogo actual no define
  niveles intermedios de Crear/Editar para ningún acceso.
- `¿Puedo filtrar los accesos por módulo?` No hay un selector de módulo visible actualmente; se
  muestran todos los grupos juntos.
- Search terms: `accesos`, `perfil`, `nivel`, `visualizar`, `todo`, `permisos`, `catálogo de
  accesos`, `grupo`.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

- The name/description modal and the Access panel are two independent save actions on the same
  profile: saving one never implicitly changes the other (accesses survive a name-only edit; a
  brand-new profile has no accesses until the Access panel is saved once).
- Every capability that changes what a profile grants ultimately writes the same encoded
  `Accesos` list, which is the single source `buildAccesosComputedFromPerfiles` reads when
  recomputing what a user assigned to this profile can actually do.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **“Missing required properties to add the profile” / “Faltan propiedades para agregar el
  perfil”:** the Name field was empty when saving the create/edit modal; enter a name.
- **A newly created profile shows no accesses selected when reopened:** this is expected — the
  Access panel resets after every save from the name/description modal; reselect the profile row
  and choose its accesses, then use the panel's own Save button.
- **An access card shows no level options, or the whole grid is missing:** the access catalog
  failed to parse (an amber banner explains it); this points to a build/catalog defect rather than
  something the user can fix from this page.
- **A user still can't reach a page after being granted the right profile:** confirm the access
  was actually saved from the panel's own Save button, not only typed into the card, and check the
  Users page to confirm that profile is assigned to the user.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Users (Usuarios)** at `/security/users`: assigns the profiles created and configured here to
  specific users, and can additionally grant or override individual accesses per user on top of
  their profiles. Use that page to decide who has a given profile; use this page to decide what
  the profile itself grants.
- `backend/access_list.yml` is the shared source of every access name, group, and level shown on
  this page's cards; changing what an access offers or unlocks requires a backend/catalog change,
  not an action on this page.

### FILES

```yaml
# Exact source hashes captured after claim-by-claim review.
schema: 1
hash_algorithm: sha256
files:
  - path: frontend/core/modules.ts
    role: user-interface
    hash: sha256:0839d4ae72db6d7a902b99dae286edd5be0a16d691543ee7c1643e61fb4bf014
    supports: [page-purpose, capability.browse-profiles, related-pages]
  - path: frontend/routes/security/access-profiles/+page.svelte
    role: page
    hash: sha256:f113cbaee8d9ad9f180f07993ce42054dde14523c91e38075b9178b1723f7a55
    supports: [page-purpose, concepts, capability.browse-profiles, capability.create-edit-profile, capability.assign-access, rules, troubleshooting]
  - path: frontend/routes/security/access-profiles/AccessCard.svelte
    role: user-interface
    hash: sha256:fd4806c087300d27ca6063d2347365be8368d00ea08f9973e495204f7a14fc27
    supports: [concepts, capability.assign-access]
  - path: frontend/routes/security/access-profiles/access-profiles.svelte.ts
    role: frontend-service
    hash: sha256:34269127c34cd8b1964aa51143315a914bd52b1f98ab98469c9dff70e195c72f
    supports: [concepts, capability.create-edit-profile, capability.assign-access, rules]
  - path: frontend/routes/security/access-profiles/access-list-catalog.ts
    role: shared-domain
    hash: sha256:0518ffb7303826a8017dddf235b6b3dd9eb83fb4748ffe8afc6588c3587bcb09
    supports: [concepts, capability.assign-access, troubleshooting]
  - path: frontend/packages/genix-ui/layers/Modal.svelte
    role: user-interface
    hash: sha256:93683aa4acfc28fce24750436038c2fd916e6f8ddb18d746c1ee284d7dcca688
    supports: [capability.create-edit-profile]
  - path: backend/security/perfiles.go
    role: backend-handler
    hash: sha256:861f732beabf364927b86442008150b031b2713d81fe32d2fd5111e41cd79680
    supports: [capability.create-edit-profile, capability.assign-access, rules]
  - path: backend/security/types/perfiles.go
    role: data-model
    hash: sha256:c95d19b22d23d9091c6b3c2b1e1464c335eb33714fe280448960e678a6d8db67
    supports: [concepts, capability.assign-access]
  - path: backend/security/usuarios.go
    role: business-logic
    hash: sha256:935f2eaa69e09aeec251d13599c9e5d4de36907e6e9d6bcece051cd542e5218d
    supports: [concepts, capability.assign-access, rules]
  - path: backend/access_list.yml
    role: permissions
    hash: sha256:0c00cfb3e7af9a918eb753846874ac7d213f1eacb3e46bded4049016e1c57951
    supports: [concepts, capability.assign-access, related-pages]
  - path: backend/main-handlers.go
    role: permissions
    hash: sha256:2474ede3472c063c1e28ea584cd6265dc4b7b8437231cfa94aa5671e66b62330
    supports: [capability.assign-access, rules]
```
