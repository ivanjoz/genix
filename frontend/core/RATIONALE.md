## Menu group "Negocio" folded into "Mi Empresa"

**Context** — The side menu had a group `Configuración` whose first option was `Mi Empresa`, and a
separate one-option group `Negocio` holding `Sedes & Almacenes`. The naming was inverted (the group
read as the page, the page as the group) and a group with a single option wasted a whole level.

**Decision** — Group id 1 is now `My Company|Mi Empresa` (minName `EMP`) and its first option is
`Configuration|Configuración`. `Sedes & Almacenes` moved into it as the second option, and the
`Negocio` group (id 2) was deleted. `backend/access.toml` mirrors the change: access group 1 is
renamed to `Mi Empresa`, access id 1 to `Configuración`, access id 7 moves to group 1, and access
group 2 (`Negocio`) is removed.

**Rationale** — Menu group ids are presentation-only: they are re-derived at runtime in
`routes/security/access-profiles` and never persisted, so dropping id 2 breaks no stored profile.
What *is* persisted is `access_list.id` (as `id*10 + level` in `profiles.accesos`), and those ids
are untouched — `Sedes & Almacenes` stays id 7, so existing profiles keep the same permission.
Cost: the access-profiles screen groups accesses by these same ids, so the two files must be kept
in sync by hand; the alternative (generating the menu from the YAML) is a bigger change than this
rename warrants.
