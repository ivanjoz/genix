---
schema: 1
page_id: security.users-profiles
route: /security/users-profiles
title: Users & Profiles (Usuarios & Perfiles)
status: implemented
visibility: tenant
description_en: >-
  User and access-profile management on one page with two tabs. The Users tab creates and edits
  login accounts with their personal data, assigned profiles, individual permissions and password;
  the Profiles tab creates reusable profiles and grants each one View or Full control over the
  accesses in the system catalog.
description_es: >-
  Gestión de usuarios y perfiles de acceso en una sola página con dos pestañas. La pestaña
  Usuarios crea y edita cuentas de acceso con sus datos personales, perfiles asignados, permisos
  individuales y contraseña; la pestaña Perfiles crea perfiles reutilizables y les otorga el nivel
  Visualizar o Todo sobre los accesos del catálogo del sistema.
---

# Users & Profiles (Usuarios & Perfiles)

<!-- DOC-ID: page-purpose -->
## Page purpose

Users & Profiles (`Usuarios & Perfiles`) is the administrative page for who can sign in to a
company (`empresa`) and what they are allowed to do. It holds two tabs:

- **Users (Usuarios)** — the login accounts (`cuentas de usuario`): personal data (name, job
  title, email, document number), the access profiles (`perfiles`) and individual per-user
  permissions (`accesos individuales`) assigned to each one, and the login password.
- **Profiles (Perfiles)** — the reusable profiles themselves: their name and description, and
  exactly which accesses (`accesos`) from the system catalog each profile grants and at which
  level (`Visualizar` or `Todo`).

The page does not define the catalog of accesses itself (their names, which pages/APIs they gate,
which levels they offer); that catalog lives in `backend/access_list.yml` and is shared,
read-only input to both tabs. It also does not handle a signed-in user editing their own profile
from the account/header menu; that is a separate self-service flow using the same backend
endpoint family (`user-self`).

<!-- DOC-ID: navigation -->
## Tabs and who sees them (Pestañas y quién las ve)

The two tabs are gated by two separate accesses in the catalog: **Usuarios** (access id 2) and
**Perfiles & Accesos** (access id 3). Both point at this single route, so holding either one is
enough to open the page — the tab whose access the user does not hold is simply not offered in
the top strip, and the page opens on the first tab the user can actually see. Granting someone
the Users tab therefore does not grant them the Profiles tab, and vice versa.

The last tab a user selected is remembered per route in the browser, so returning to the page
reopens the tab they were last working in.

<!-- DOC-ID: users.concepts -->
## Business concepts (Conceptos del negocio)

- A **user (usuario)** is one login account within a company: username (`Usuario`), first
  and last name, email, job title, document number, status, and the permissions below.
  Usernames are permanent once the user is created (this page's form disables the field);
  Genix does not offer renaming an existing username.
- A **profile (perfil)** is a reusable bundle of accesses (`accesos`) managed on
  the **Profiles (Perfiles)** tab. Assigning one or more profiles to a user grants that user every
  access the profiles contain.
- An **access (acceso)** identifies one restricted page or capability in the access catalog
  (for example "Usuarios", "Perfiles & Accesos", or accesses belonging to other modules such
  as Finance or Logistics). Each access can be granted at one or more levels; the levels
  configured for that specific access decide which of **View (Visualizar)**, **Create
  (Crear)**, **Edit (Editar)**, or **Full (Todo)** are actually offered for it.
- **Individual access (`Accesos` on this page, labeled "ACCESOS ::")** grants or overrides a
  specific access/level directly on one user, independent of their profiles. Genix computes
  the user's effective access as the union of every level coming from assigned profiles plus
  every individually granted level.
- On this page, the table's Access (`Accesos`) column and the form's profile chips summarize
  effective access into two buckets: a view-only eye icon for accesses granted at level 1
  (Visualizar) and a pencil icon for accesses granted at any higher level (Crear, Editar, or
  Todo), listing the access name under each icon.

<!-- DOC-ID: users.capability.browse-users -->
## Find a user (Buscar un usuario)

Open **My Company (Mi Empresa) → Users & Profiles (Usuarios & Perfiles)** at
`/security/users-profiles` and select the **Users (Usuarios)** tab. The list
shows every active user of the company with ID, username plus full name, a summarized access
list (view vs. edit icons, truncated after 8 names per icon with a "... (N más)" suffix), the
email, the raw status value, and the last update date/time. Use the filter box to narrow the
list by username, first name, last name, or email; the filter matches locally against the
already-loaded list, not a server-side search. Selecting a row opens it in the edit layer
described below.

<!-- DOC-ID: users.capability.create-edit -->
## Create or edit a user (Crear o editar un usuario)

### User intention (Intención del usuario)

Create a login account for a new employee/collaborator, or update an existing user's
personal data, job title, email, document number, profiles, or individual accesses.

### Where to find it (Dónde encontrarlo)

On the **Users (Usuarios)** tab, use the green create button (top right of the toolbar) to open the side
layer for a new user, or click an existing row to open the same layer pre-filled for editing.
Save with the layer's **Save (Guardar)**/**Update (Actualizar)** action.

### Required information and prerequisites (Requisitos previos)

- **Username (Usuario)** and **First Name (Nombres)** are required and must each be at least
  4 characters; the check runs on the frontend and is enforced again by the server.
- **Last Name (Apellidos)**, **Document # (Nº Documento)**, **Job Title (Cargo)**, and
  **Email** are optional operational fields.
- At least one **profile (perfil)** must be selected unless the user being saved is the
  fixed company administrator (internal ID 1). Having only individual accesses (`Accesos`)
  without any profile does **not** satisfy this requirement — the server counts profiles
  only when deciding whether the user has "al menos 1 permiso".
- Username cannot be changed once the user exists: the input is disabled for any user with
  an ID, so editing only affects the other fields.

### Business rules and rationale (Reglas y razón de negocio)

The company's fixed administrator account (internal ID 1) always keeps the username `admin`;
the server overwrites whatever username value it receives for that ID. That user is also the
only one exempt from the "must have at least one profile" rule.

Editing preserves the user's creation date and creator (`Created`/`CreatedBy`) and, unless a
new password is supplied, the existing password hash — an edit that leaves the password field
empty does not reset or clear the login password.

### Result and side effects (Resultado y efectos)

Saving creates or updates the user record and recomputes its effective access
(`AccesosComputed`) from the current profiles plus individual accesses, so a profile change
made afterward on the **Profiles (Perfiles)** tab is reflected the next time this user is saved or logs
in through the normal access-refresh path. A new user is created with **Status = 1**
(active); this page exposes no field to change status afterward.

### Limitations (Limitaciones)

- There is no visible active/inactive toggle; the Status column simply shows the raw stored
  number instead of a label, and nothing on this page can change it.
- The document number, job title, and email fields are free text; the page performs no format
  validation on them (for example no email-format check is enforced here).

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo creo un usuario nuevo?`
- `¿Por qué no puedo cambiar el nombre de usuario (login) de alguien ya creado?`
- `¿Por qué me pide asignar al menos un perfil?` Un usuario (salvo el administrador con ID 1)
  necesita como mínimo un perfil; los accesos individuales solos no bastan.
- Search terms: `usuario`, `crear usuario`, `editar usuario`, `nombre de usuario`, `cargo`,
  `documento`, `perfil obligatorio`.

<!-- DOC-ID: users.capability.assign-access -->
## Assign profiles and individual access (Asignar perfiles y accesos individuales)

### User intention (Intención del usuario)

Decide exactly what a user can see and edit: attach one or more reusable profiles, and/or
grant or restrict specific accesses directly on that user without changing a shared profile.

### Where to find it (Dónde encontrarlo)

Inside the create/edit layer, use the dual selector below the personal-data fields:
**PERFILES ::** on the left to search and add profiles, **ACCESOS ::** on the right to search
and add individual access/level entries. Click a search result to add it as a chip; hover a
selected chip and use its trash icon to remove it.

### Required information and prerequisites (Requisitos previos)

Profiles come from the **Profiles (Perfiles)** tab; only profiles with an active status appear in the
selector. Individual accesses come from the same access catalog (`access_list.yml`) used
across Genix, covering accesses for every module (Finance, Logistics, Security, System, etc.),
not only this page's own access. Each catalog access lists the levels it makes available (for
example some accesses offer View, Create, Edit, and Full; the Users/Perfiles & Accesos
accesses on this catalog only offer **View (Visualizar)** and **Full (Todo)** — there is no
separate Create/Edit granularity for managing users and profiles). Only levels declared for
that access appear as selectable options for it.

### Business rules and rationale (Reglas y razón de negocio)

Effective access for a user is the union of levels coming from every assigned profile plus
every individually granted access/level; the server keeps only the highest level per access
when profiles overlap, and adds the individually granted ones on top. This lets an admin grant
a user broader or narrower access on a single page without editing (or forking) a shared
profile used by other users.

Saving a user requires server-side permission on the "Usuarios" access at the Full (Todo)
level; because that access only offers View or Full, granting someone the ability to edit
users on this page always means granting them full control of it, not a partial edit level.
Viewing the user list itself is currently open to any authenticated user of the company,
since no read-only access is mapped for it in the catalog.

### Result and side effects (Resultado y efectos)

The user's `ProfileIDs` and `AccessLevelIDs` are saved as provided, and the server recomputes
`AccesosComputed` (the flattened effective access list actually enforced on API calls) from
the current profile assignments plus the individual accesses.

### Limitations (Limitaciones)

- This selector cannot create or edit a profile itself; use the **Profiles (Perfiles)** tab for that.
- Removing every profile from a user without granting equivalent individual accesses can
  leave them without access to pages they used to reach through that profile.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo le doy a un usuario acceso a un módulo sin crear un perfil nuevo?` Usa "ACCESOS ::"
  para otorgárselo individualmente.
- `¿Qué pasa si el usuario tiene un perfil y también un acceso individual para lo mismo?`
  Genix usa el nivel más alto entre ambos.
- `¿Por qué sólo veo "Visualizar" y "Todo" para Usuarios/Perfiles & Accesos?` Esos dos
  accesos del catálogo no ofrecen niveles intermedios de Crear/Editar.
- Search terms: `perfiles`, `accesos individuales`, `permisos`, `acceso total`, `sólo lectura`,
  `visualizar`, `todo`.

<!-- DOC-ID: users.capability.set-password -->
## Set or change the password (Establecer o cambiar la contraseña)

### User intention (Intención del usuario)

Set the initial login password when creating a user, or change an existing user's password
without touching any other field.

### Where to find it (Dónde encontrarlo)

**Password** and **Confirm Password (Password (Repetir))** fields at the bottom of the
create/edit layer.

### Required information and prerequisites (Requisitos previos)

For a new user, both password fields are required, must match, and the password must be at
least 6 characters. For an existing user, the fields are optional; leaving them empty keeps
the current password (the placeholder shows `UNCHANGED|SIN CAMBIAR`). Entering a password on
an edit still requires at least 6 characters and both fields to match.

### Business rules and rationale (Reglas y razón de negocio)

Both password values are trimmed before validation. The server independently re-checks the
minimum length for a brand-new user and re-hashes the password only when a value of at least
6 characters was actually sent; the plaintext password is never stored or returned by the
save response.

### Limitations (Limitaciones)

This page has no "send password reset" or "forgot password" flow; a password change always
requires an admin (or the user, through the separate self-service profile editor) to type the
new password directly into this form.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo cambio la contraseña de un usuario sin tocar sus otros datos?`
- `¿Por qué dice "SIN CAMBIAR" en el campo de password?` Porque dejarlo vacío conserva la
  contraseña actual.
- Search terms: `contraseña`, `password`, `clave`, `cambiar clave`, `restablecer contraseña`.

<!-- DOC-ID: users.capability.remove -->
## Remove a user from the list (Quitar un usuario de la lista)

Selecting an existing user and using the layer's **Delete (Eliminar)** action re-submits the
currently loaded form data unchanged and then removes that user only from the browser's local
list; there is no confirmation prompt before this happens. This action does **not** send any
deletion or deactivation signal to the server: the user's stored `Status` is not changed, so
the account remains active and still able to log in. Because saving also marks the `users`
data as needing a refresh, the removed row can reappear once the list refreshes from the
server (for example after reopening the page), since the server still returns it as an active
user.

### Limitations (Limitaciones)

Treat **Eliminar** on this page as removing the row from view only, not as deactivating or
deleting the account. There is currently no verified way on this page to deactivate
(`desactivar`) or permanently delete (`eliminar`) a user's ability to log in.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo desactivo o elimino un usuario?` El botón "Eliminar" actual sólo lo quita de la
  lista visible; no bloquea su acceso ni cambia su estado en el servidor.
- `¿Por qué el usuario que "eliminé" sigue apareciendo después de recargar?` Porque su
  registro sigue activo en el servidor.
- Search terms: `eliminar usuario`, `borrar usuario`, `desactivar usuario`, `dar de baja`.

<!-- DOC-ID: users.rules -->
## Cross-capability business rules (Reglas generales)

- A user's effective permissions always come from the same two sources — assigned profiles
  and individually granted accesses — combined by taking the highest level per access; every
  capability on this page (view, create, edit) reads and writes that same combined model.
- Saving through this admin page (as opposed to the separate self-service `user-self` path)
  always requires the "Usuarios" access at Full (Todo) level, and always allows the acting
  admin to set any user's profiles, individual accesses, and status-affecting fields; the
  self-service path restores those fields from the stored record instead of trusting the
  submitted body, so a user editing their own profile elsewhere cannot grant themselves
  access this way.

<!-- DOC-ID: users.troubleshooting -->
## Common problems (Problemas comunes)

- **“El usuario y el nombre deben tener al menos 4 caracteres”:** lengthen the Username or
  First Name field; both need at least 4 characters.
- **“El password tiene menos de 6 caracteres” / “Los password no coinciden”:** occurs when
  creating a user, or when entering a new password on an edit, without a valid 6+ character
  match in both password fields.
- **“El user debe tener al menos 1 permiso”:** assign at least one profile in "PERFILES ::";
  individual accesses in "ACCESOS ::" alone do not satisfy this requirement.
- **A "deleted" user reappears:** the current Delete action only hides the row locally; it
  does not deactivate the account on the server.
- **Can't rename an existing username:** the Username field is intentionally disabled once
  the user has been saved.

<!-- DOC-ID: profiles.concepts -->
## Business concepts (Conceptos del negocio)

- A **profile (perfil)** is a named, reusable bundle of accesses: `Name (Nombre)`,
  `Description (Descripción)`, and the list of accesses/levels it grants. Assigning a profile to a
  user (on the Users (Usuarios) tab) grants that user every access the profile contains.
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
- Each catalog entry belongs to a **group (grupo)** (`Mi Empresa`, `Producción`, `Comercial`,
  `Clientes (CRM)`, `Logística`, `Finanzas`, `Tienda`, `Contabilidad`, `System`) used purely to
  organize the access grid into labeled sections on this page; it does not affect what the access unlocks.
- Saving a profile encodes its granted accesses as one packed number per access/level
  (`accesoID * 10 + nivel`) in the profile's `Accesos` list. Access IDs are permanent in the
  catalog and are never renumbered or reused, so a profile's stored grants keep meaning even as
  the catalog grows.

<!-- DOC-ID: profiles.capability.browse-profiles -->
## Find a profile (Buscar un perfil)

Open **My Company (Mi Empresa) → Users & Profiles (Usuarios & Perfiles)** at
`/security/users-profiles` and select the **Profiles (Perfiles)** tab. The left table lists every
active profile (ID, Name) for the company; only profiles with an active status are ever loaded or
shown, matching the same active-only listing used by the profile selector on the Users (Usuarios)
tab. Use the filter box above the table to narrow the list by name; the filter matches locally against the already-loaded list, not a
server-side search. Selecting a row loads that profile into the Access panel on the right.

<!-- DOC-ID: profiles.capability.create-edit-profile -->
## Create or edit a profile's name and description (Crear o editar nombre y descripción)

### User intention (Intención del usuario)

Create a new reusable profile before assigning users to it (on the Users (Usuarios) tab), or rename/update
the description of an existing profile without touching what it grants.

### Where to find it (Dónde encontrarlo)

On the **Profiles (Perfiles)** tab, use the green create button (top of the left table's toolbar) to
open a form modal for a new profile. To edit an existing profile's name/description, click its
row's "..." action to open the same modal pre-filled (**Editing|Editando**). Save with the
modal's own save action.

### Required information and prerequisites (Requisitos previos)

- **Name (Nombre)** is required; the only frontend check is that it is non-empty (there is no
  minimum-length rule here, unlike the 4-character rule on the Users (Usuarios) tab). This check runs on the
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

<!-- DOC-ID: profiles.capability.assign-access -->
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

<!-- DOC-ID: profiles.rules -->
## Cross-capability business rules (Reglas generales)

- The name/description modal and the Access panel are two independent save actions on the same
  profile: saving one never implicitly changes the other (accesses survive a name-only edit; a
  brand-new profile has no accesses until the Access panel is saved once).
- Every capability that changes what a profile grants ultimately writes the same encoded
  `Accesos` list, which is the single source `buildAccesosComputedFromPerfiles` reads when
  recomputing what a user assigned to this profile can actually do.

<!-- DOC-ID: profiles.troubleshooting -->
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
  Users (Usuarios) tab to confirm that profile is assigned to the user.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- The two tabs are each other's counterpart: use **Profiles (Perfiles)** to decide what a profile
  grants, and **Users (Usuarios)** to decide who holds it and to add individual accesses on top.
- `backend/access_list.yml` is the shared source of every access name, group, and level shown on
  both tabs; changing what an access offers or unlocks requires a backend/catalog change, not an
  action on this page.
- The account/header profile editor (self-service) lets a signed-in user update their own
  personal data and password through the same backend user-save logic (`user-self`), but it
  restores profiles, individual accesses, username, and status from the stored record instead
  of accepting them from the request, so it cannot be used to change permissions.

### FILES

```yaml
# Exact source hashes captured after claim-by-claim review.
schema: 1
hash_algorithm: sha256
files:
  - path: frontend/core/modules.ts
    role: user-interface
    hash: sha256:9c8c3f964115419c52460054da95dcebb0712239278102487705331d1973d39b
    supports: [page-purpose, users.capability.browse-users, profiles.capability.browse-profiles, related-pages]
  - path: frontend/routes/security/users-profiles/+page.svelte
    role: page
    hash: sha256:4d5d4d24228c8636b9ec9a351c9a0cee8bbbb7182e6f9b735e63232c59e5f4fb
    supports: [page-purpose, navigation]
  - path: frontend/routes/security/users-profiles/UsersTab.svelte
    role: page
    hash: sha256:dff12d538f854fcb10dbe75d9896a57c1929f9ec6efe85628041a2a3496c2738
    supports: [users.concepts, users.capability.browse-users, users.capability.create-edit, users.capability.assign-access, users.capability.set-password, users.capability.remove, users.rules, users.troubleshooting]
  - path: frontend/routes/security/users-profiles/ProfilesTab.svelte
    role: page
    hash: sha256:926539cd0f6bae44d117d266e24c3862d3b86248f608bffdc6e1d906a2580a06
    supports: [profiles.concepts, profiles.capability.browse-profiles, profiles.capability.create-edit-profile, profiles.capability.assign-access, profiles.rules, profiles.troubleshooting]
  - path: frontend/routes/security/users-profiles/UserProfilesAccessSelector.svelte
    role: user-interface
    hash: sha256:85190ef3ed8f569bd18ff4943a9d112c4de2012afb23a2bd9bfba0f0d57b8427
    supports: [users.concepts, users.capability.assign-access]
  - path: frontend/routes/security/users-profiles/AccessCard.svelte
    role: user-interface
    hash: sha256:81712f82226d6bc6dc663d38ac29792a16b5403eca57b106b212afbf1b9283c3
    supports: [profiles.concepts, profiles.capability.assign-access]
  - path: frontend/routes/security/users-profiles/users-profiles.svelte.ts
    role: frontend-service
    hash: sha256:ca4c0b2f8b4246a970aa9192df9e6b4479f4aa104a69a226c55f03ccfcc05322
    supports: [navigation, users.concepts, users.capability.create-edit, users.capability.assign-access, users.capability.remove, profiles.concepts, profiles.capability.create-edit-profile, profiles.capability.assign-access, profiles.rules]
  - path: frontend/routes/security/users-profiles/access-list-catalog.ts
    role: shared-domain
    hash: sha256:0518ffb7303826a8017dddf235b6b3dd9eb83fb4748ffe8afc6588c3587bcb09
    supports: [users.concepts, users.capability.assign-access, profiles.concepts, profiles.capability.assign-access, profiles.troubleshooting]
  - path: frontend/services/services/users.svelte.ts
    role: frontend-service
    hash: sha256:791af77a675aa1ebf54a6a70872b6f543d71236ed2e7ab8ababb7e2486ed8114
    supports: [users.capability.create-edit, users.capability.remove, related-pages]
  - path: frontend/packages/genix-ui/cards/SearchDualCard.svelte
    role: user-interface
    hash: sha256:857498a1795dc294f30f3f57076ec39525045320686a3f77631b5fa2f5039a69
    supports: [users.capability.assign-access]
  - path: frontend/packages/genix-ui/layers/Modal.svelte
    role: user-interface
    hash: sha256:9655dee89b61466bdd1966a6bd3b9a959b96bc65b13ae647f48257e009293db2
    supports: [profiles.capability.create-edit-profile]
  - path: frontend/domain-components/Page.svelte
    role: user-interface
    hash: sha256:fd6faa8be32a768a0255013dc069e2f3fe0814acfe6232494084458aa8d1aa98
    supports: [navigation]
  - path: frontend/domain-components/HeaderConfig.svelte
    role: user-interface
    hash: sha256:e64f69c280c150204c11dc57d1bff1908703b15bf88f05c8d3e2fd77506665c9
    supports: [related-pages, users.rules]
  - path: backend/security/usuarios.go
    role: backend-handler
    hash: sha256:7b302ab72bc3ebe8e5e06a4b12cf0cf95e55534e49733087ad078ebca69dfaa8
    supports: [users.capability.create-edit, users.capability.assign-access, users.capability.set-password, users.capability.remove, users.rules, users.troubleshooting, profiles.capability.assign-access]
  - path: backend/security/perfiles.go
    role: backend-handler
    hash: sha256:113302558b3786f75888ce60a4fa77b986407b4c329a0517a6faa3f6772a5765
    supports: [profiles.capability.create-edit-profile, profiles.capability.assign-access, profiles.rules]
  - path: backend/core/types/users.go
    role: data-model
    hash: sha256:4a69d1976058bea987c991c59e97a5b16f0a1c91e12a120f702e929bbe53aa22
    supports: [users.concepts, users.capability.create-edit, users.capability.assign-access, users.rules]
  - path: backend/security/types/perfiles.go
    role: data-model
    hash: sha256:ebe1a63290b8e313dc31dfa6113a1fa228a1437279e4286c86717d7416aa7aa3
    supports: [users.concepts, profiles.concepts, profiles.capability.assign-access]
  - path: backend/access_list.yml
    role: permissions
    hash: sha256:491c43a25f0837ef39fc1cb43a82ac890610cf3fa74c9dee4757744eb72f396b
    supports: [navigation, users.capability.assign-access, users.rules, profiles.concepts, profiles.capability.assign-access, related-pages]
  - path: backend/main-handlers.go
    role: permissions
    hash: sha256:0e4a825ccd2fe08e6a586cfcd24406b715f5e548c9e71d93091421a2d39e8956
    supports: [users.capability.assign-access, users.rules, profiles.capability.assign-access]
```
