## The access card is compact; its action buttons live on hover

**Context** — the access grid renders 36 cards, and each one reserved a third row for the
`VER`/`TODO` buttons even when nothing was granted. At 68px+ per card the six groups never fit on
one screen, so granting a permission always meant scrolling to find the card first.

**Decision** — on desktop (`deviceType === 1`) the card is only as tall as its two text lines
(~48px): the action buttons are absolutely positioned at centre-right and appear on hover with no
transition, over a background-coloured fade so the route text stays readable underneath. A
selected card shows its granted levels as bare coloured icons in the top-right corner, and that
badge hides while the buttons are showing — the corner is one slot, used by whichever of the two
is relevant. Corners are square and the selected card carries no outline or accent border: the
5px left bar in the level's colour plus the corner icon are the entire selected state. That bar is
an `::before` reading a `--access-color` custom property, not a `border-left`: a border miters
diagonally where it meets the 1px top and bottom borders, which visibly bevels the bar's ends. Touch (`deviceType > 1`) is untouched: the buttons stay in flow and the badge is
hidden, since there is no hover there to reveal anything.

**Rationale** — hover-revealed controls cost discoverability, which is affordable here because the
whole card is already a click target on touch and the badge keeps the *state* permanently visible
— only the *controls* hide, and the 5px left border carries the granted level's colour whether or
not the card is hovered. Reserving the button row instead would have kept the grid at half the
density. The gate is `perfilForm?.accesosMap`, not `perfilForm`: the unselected form is `{}`, which
is truthy, so gating on the object alone rendered buttons that threw on click once they became
hover-reachable.

## Users and Profiles & Access merged into one two-tab route

**Context** — `/security/users` and `/security/access-profiles` were two menu options and two
routes that always got used together: you cannot assign a profile without creating it first, and
both pages read the same `access.toml` catalog and the same `perfiles` endpoint. Two separate
`PerfilesService` classes had drifted apart (`users.svelte.ts` had a stripped-down copy without
`perfilesMap`/`updatePerfil`), and each page fetched the catalog on its own `onMount`.

**Decision** — One route, `frontend/routes/security/users-profiles`, exposed as the single menu
option `Users & Profiles|Usuarios & Perfiles`. `+page.svelte` is the shell: it owns the one
`PerfilesService` and the one catalog load, and switches between `UsersTab.svelte` and
`ProfilesTab.svelte` through `Page`'s `options` strip. The duplicate `PerfilesService` is gone —
`users-profiles.svelte.ts` keeps the superset — as is the write-only `IProfile._open` flag and the
local `IProfile` copy that shadowed `$core/types/common`. `access-list-catalog.ts` stayed inside
the feature folder (not `core/`) so `backend/access.toml` keeps out of the storefront bundle,
which is why `+layout.svelte` still imports it from a route folder.

**Rationale** — Hoisting the service and the catalog into the shell is what makes the merge worth
doing: a profile created on the Profiles tab is immediately selectable on the Users tab, which was
impossible across two routes. The cost is two prop-drilled tabs instead of two self-contained
pages.

## Both access ids kept; the tab, not the route, is what gets gated

**Context** — `access.toml` id 2 (`Usuarios`, `POST.users`) and id 3 (`Perfiles & Accesos`,
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
