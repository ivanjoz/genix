## Users and Profiles & Access merged into one two-tab route

**Context** — `/security/users` and `/security/access-profiles` were two menu options and two
routes that always got used together: you cannot assign a profile without creating it first, and
both pages read the same `access_list.yml` catalog and the same `perfiles` endpoint. Two separate
`PerfilesService` classes had drifted apart (`users.svelte.ts` had a stripped-down copy without
`perfilesMap`/`updatePerfil`), and each page fetched the catalog on its own `onMount`.

**Decision** — One route, `frontend/routes/security/users-profiles`, exposed as the single menu
option `Users & Profiles|Usuarios & Perfiles`. `+page.svelte` is the shell: it owns the one
`PerfilesService` and the one catalog load, and switches between `UsersTab.svelte` and
`ProfilesTab.svelte` through `Page`'s `options` strip. The duplicate `PerfilesService` is gone —
`users-profiles.svelte.ts` keeps the superset — as is the write-only `IProfile._open` flag and the
local `IProfile` copy that shadowed `$core/types/common`. `access-list-catalog.ts` stayed inside
the feature folder (not `core/`) so `backend/access_list.yml` keeps out of the storefront bundle,
which is why `+layout.svelte` still imports it from a route folder.

**Rationale** — Hoisting the service and the catalog into the shell is what makes the merge worth
doing: a profile created on the Profiles tab is immediately selectable on the Users tab, which was
impossible across two routes. The cost is two prop-drilled tabs instead of two self-contained
pages.

## Both access ids kept; the tab, not the route, is what gets gated

**Context** — `access_list.yml` id 2 (`Usuarios`, `POST.users`) and id 3 (`Perfiles & Accesos`,
`POST.perfiles`) were separate accesses on separate routes. `canAccessRoute` grants a route if
**any** access mapped to it passes, so pointing both ids at one route would have let a user holding
only `Usuarios` open the profiles editor.

**Decision** — Both ids now list `frontend_routes: "security/users-profiles"` and both are kept.
The page filters its own tab strip with `security.checkAcceso(USERS_ACCESS_ID, 1)` and
`security.checkAcceso(PROFILES_ACCESS_ID, 1)`, so a user is only offered the tabs they can read;
`Page` falls back to the first remaining option. `backend_apis` stay split, so the server-side
check is unchanged.

**Rationale** — Merging the two ids into one would have been simpler, but id 3 is persisted in
`profiles.accesos` as `id*10 + level`: retiring it would silently promote every profile holding
only `Usuarios` to full control of profiles and accesses. Per-tab gating keeps stored permissions
meaning exactly what they meant before. The cost is that route-level access is now coarser than
what the user actually sees — the tab strip is the real boundary in the UI, while the backend
still enforces `POST.users` and `POST.perfiles` separately.
