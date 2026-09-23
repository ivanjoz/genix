## sendUserNotification: popup window, color vs. icon, and the v2 row shape
**Context** — The header layer needed a message API and a "recent only" popup, which left some
behavior open: whether icon and color are one choice, what "last 5 seconds" is measured against
while the popup stays open, whether processes pop it too, and what to do with v1 rows.
**Decision** — `color` (blue|green|yellow|red) and `icon` (info|success|warning|error) are separate
options, and `icon` defaults to the one that matches the color. The 5-second window is fixed when the
popup opens (`notificationsLayer.recentSinceTime`), so messages that arrive while it's open are added
and nothing drops out. A layer the user already opened on the full list stays on the full list.
Only `sendUserNotification` pops the layer; `addProcess`/`updateProcess` do not. A process has no
color of its own: its status decides it (running blue + spinner, done green, canceled red). Rows now
carry `title/subtitle/message/color/icon` for both kinds (no `name/text/type`). The IndexedDB version
went to 2, and its upgrade clears the v1 rows.
**Rationale** — Keeping the icon separate costs one optional field and lets a green card carry, say,
an info icon. Fixing the window at open time matches "they don't disappear until you click outside".
Processes already announce themselves through the spinning ring and badge, so popping the layer on
every upload would only interrupt the user. Clearing
v1 rows loses only local UI history and avoids a read-time shape fallback.

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
