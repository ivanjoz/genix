---
schema: 1
page_id: security.users
route: /security/users
title: Users (Usuarios)
status: implemented
visibility: tenant
description_en: >-
  System user management. Create and edit users with personal data, job title, and email;
  assign access profiles and individual permissions per user; set or change passwords.
description_es: >-
  Gestión de usuarios del sistema. Crear y editar usuarios con datos personales, cargo y correo;
  asignar perfiles de acceso y permisos individuales por usuario; establecer o cambiar contraseñas.
---

# Users (Usuarios)

<!-- DOC-ID: page-purpose -->
## Page purpose

Users (`Usuarios`) is the administrative page for the login accounts (`cuentas de usuario`)
that can sign in to a company (`empresa`). It creates and edits each user's personal data
(name, job title, email, document number), assigns the access profiles (`perfiles`) and
individual per-user permissions (`accesos individuales`) that decide what that user can see
and edit across Genix, and sets or changes the login password.

This page does not manage the profiles themselves — creating a profile and choosing which
accesses it grants happens in **Profiles & Access (Perfiles & Accesos)**. It also does not
handle a signed-in user editing their own profile from the account/header menu; that is a
separate self-service flow using the same backend endpoint family (`user-self`).

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A **user (usuario)** is one login account within a company: username (`Usuario`), first
  and last name, email, job title, document number, status, and the permissions below.
  Usernames are permanent once the user is created (this page's form disables the field);
  Genix does not offer renaming an existing username.
- A **profile (perfil)** is a reusable bundle of accesses (`accesos`) managed on
  **Profiles & Access**. Assigning one or more profiles to a user grants that user every
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

<!-- DOC-ID: capability.browse-users -->
## Find a user (Buscar un usuario)

Open **Configuration (Configuración) → Users (Usuarios)** at `/security/users`. The list
shows every active user of the company with ID, username plus full name, a summarized access
list (view vs. edit icons, truncated after 8 names per icon with a "... (N más)" suffix), the
email, the raw status value, and the last update date/time. Use the filter box to narrow the
list by username, first name, last name, or email; the filter matches locally against the
already-loaded list, not a server-side search. Selecting a row opens it in the edit layer
described below.

<!-- DOC-ID: capability.create-edit -->
## Create or edit a user (Crear o editar un usuario)

### User intention (Intención del usuario)

Create a login account for a new employee/collaborator, or update an existing user's
personal data, job title, email, document number, profiles, or individual accesses.

### Where to find it (Dónde encontrarlo)

On `/security/users`, use the green create button (top right of the toolbar) to open the side
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
made afterward on **Profiles & Access** is reflected the next time this user is saved or logs
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

<!-- DOC-ID: capability.assign-access -->
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

Profiles come from **Profiles & Access**; only profiles with an active status appear in the
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

- This selector cannot create or edit a profile itself; use **Profiles & Access** for that.
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

<!-- DOC-ID: capability.set-password -->
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

<!-- DOC-ID: capability.remove -->
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

<!-- DOC-ID: rules -->
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

<!-- DOC-ID: troubleshooting -->
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

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Profiles & Access (Perfiles & Accesos)** at `/security/access-profiles`: create and
  configure the profiles offered in this page's "PERFILES ::" selector, and see which
  accesses/levels each profile grants.
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
    hash: sha256:0839d4ae72db6d7a902b99dae286edd5be0a16d691543ee7c1643e61fb4bf014
    supports: [page-purpose, related-pages]
  - path: frontend/routes/security/users/+page.svelte
    role: page
    hash: sha256:df1de73a8a4f8bc635b5fbea1da9de2733a19e0fa7c981cbb8438bf61d7bb34e
    supports: [page-purpose, concepts, capability.browse-users, capability.create-edit, capability.assign-access, capability.set-password, capability.remove, rules, troubleshooting]
  - path: frontend/routes/security/users/UserProfilesAccessSelector.svelte
    role: user-interface
    hash: sha256:5c6c16dd5df1b3a01f0211fafe9150bf1227a06ab4427421cd57f70c63950b95
    supports: [concepts, capability.assign-access]
  - path: frontend/routes/security/users/users.svelte.ts
    role: frontend-service
    hash: sha256:bcce5c06748b895562d680a2e698f082a5fc484247e96259fe48b89f16cdd597
    supports: [concepts, capability.create-edit, capability.assign-access, capability.remove, rules]
  - path: frontend/services/services/users.svelte.ts
    role: frontend-service
    hash: sha256:791af77a675aa1ebf54a6a70872b6f543d71236ed2e7ab8ababb7e2486ed8114
    supports: [capability.create-edit, capability.remove, related-pages]
  - path: frontend/packages/genix-ui/cards/SearchDualCard.svelte
    role: user-interface
    hash: sha256:857498a1795dc294f30f3f57076ec39525045320686a3f77631b5fa2f5039a69
    supports: [capability.assign-access]
  - path: frontend/routes/security/access-profiles/access-list-catalog.ts
    role: shared-domain
    hash: sha256:0518ffb7303826a8017dddf235b6b3dd9eb83fb4748ffe8afc6588c3587bcb09
    supports: [concepts, capability.assign-access]
  - path: frontend/routes/security/access-profiles/access-profiles.svelte.ts
    role: shared-domain
    hash: sha256:34269127c34cd8b1964aa51143315a914bd52b1f98ab98469c9dff70e195c72f
    supports: [concepts, capability.assign-access]
  - path: frontend/domain-components/HeaderConfig.svelte
    role: user-interface
    hash: sha256:c8155a05c86726a2da17f798b85b2d401d5c4ef9ca970c3ec27ba73a9c154f82
    supports: [related-pages, rules]
  - path: backend/security/usuarios.go
    role: backend-handler
    hash: sha256:935f2eaa69e09aeec251d13599c9e5d4de36907e6e9d6bcece051cd542e5218d
    supports: [capability.create-edit, capability.assign-access, capability.set-password, capability.remove, rules, troubleshooting]
  - path: backend/core/types/users.go
    role: data-model
    hash: sha256:b6d7a7c08f228d3315d5c1d831dcec406094d4ae9c6218a3e8b51c19bca2ca9c
    supports: [concepts, capability.create-edit, capability.assign-access, rules]
  - path: backend/security/types/perfiles.go
    role: data-model
    hash: sha256:c95d19b22d23d9091c6b3c2b1e1464c335eb33714fe280448960e678a6d8db67
    supports: [concepts, capability.assign-access]
  - path: backend/access_list.yml
    role: permissions
    hash: sha256:0c00cfb3e7af9a918eb753846874ac7d213f1eacb3e46bded4049016e1c57951
    supports: [capability.assign-access, rules]
  - path: backend/main-handlers.go
    role: permissions
    hash: sha256:2474ede3472c063c1e28ea584cd6265dc4b7b8437231cfa94aa5671e66b62330
    supports: [capability.assign-access, rules]
```
