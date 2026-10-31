## Virtual modules from db/pages.toml + db/modules.toml: open points settled while building it
**Context** — The menu moved from `core/modules.ts` to two TOML files, with five modules, the module
selector, shared pages that stay in the module the user came from, pending pages shown disabled, and
the profile's `Modules` field deleted. Building it left these points open.
**Decision** —
- The four placeholders got English kebab-case routes for their future pages: `finance/cash-flow`,
  `finance/accounts-management`, `accounting/financial-statements`, `accounting/balance` (the old
  `/finance/gestion-cuentas` and `/finance/flujo-de-caja` were never built). The test fails if a
  pending page's folder exists, or if a built page's folder does not.
- Pages without an icon got one; `develop-ui/showroom` (in no menu before) is in System › Desarrollo.
- SaaS-only means any route listed in a `saas_only` module. `isSaaSRouteDenied` replaces the
  per-option `onlySaaS` flag in both the layout gate and the menu filter.
- The agent's `getMenu` and the search bar's menu lookup cover every module, not only the one the
  side menu shows. A shared page is listed once. `getMenu` now also sends the page `description`
  (the Go side already had the field).
- The profile editor groups accesses by their `access.toml` group only; the module filter
  (`moduleSelected`, never set by any control) and `routeCatalogIndex` are gone.
- The TOML parser moved to `libs/toml-tables.utils.ts`, now with booleans and dotted child sections
  (`[[modules.menus]]`); `access-list-catalog.ts` uses it too.
- `logout` moved from `HeaderConfig.svelte` to `core/env.ts`, so the module selector's "Salir" uses
  the same one. It also wipes the module memory (`genix_module_navigation`), on purpose: the next
  user may be someone else.
- On desktop, the selector takes the logo's place at the top of the side menu, as in Berryapps; the
  Genix logo moved to the mobile drawer's footer.
- `Profile.Modules` was removed from the Go struct and from the `fn-rename-columns` plan
  (`modulos_ids → modules`). The ORM's homologation does not drop columns, so the Scylla column
  stays in the table, unused.
**Rationale** — Each one is the smallest change that keeps a single source of truth (the TOML files)
with no runtime fallback. Cost: the orphan Scylla column needs a manual `ALTER TABLE profiles DROP`
if it should go. The agent now sees pages from modules the user is not in, which is what
"navigate anywhere" needs.

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
