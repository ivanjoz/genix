## Clicking a user in the access layer opens them on their Accesos tab

**Context** — "Take them to the by-user view with the user selected" leaves open what *selected*
means, and which of the user layer's two tabs it opens on.

**Decision** — It switches the view, opens the edit layer for that user, and sets it to Accesos, not
Información. Clicking a user card in the table does nothing on its own — the whole row opens the
access layer, which is where the users are clickable.

**Rationale** — Whoever clicks a name inside an access review is asking about that person's grants,
and Información would put a form between them and the answer. `usuarioLayerView` is deliberately not
reset anywhere else, so this is the one path that forces a tab.

## "Por Acceso" files a user under one level only, and the row does not grow

**Context** — The by-access view (`AccessUsersTable.svelte`, `buildAccessUserRows`) was specified as
one row per access level with the users holding it. Two things the spec did not settle.

**Decision** — A user appears on the row of the level they *effectively* hold and no other: TODO
does not also list them under VER. Ties in the ranking break on total users, then access id.

`TableGrid`'s virtualizer is fixed-height, so a row's user cards sit in one line that scrolls
sideways when there are more than fit, rather than the row growing.

**Rationale** — Exact-level filing keeps the two views consistent — a TODO user counts once, in the
TODO segment of their bar, and once, on the TODO row here — so the same person is never counted
twice in the same table. The sideways scroll is what fixed row heights cost; clipping the extra
users out of the cell was the alternative, and the point of the view is that the list is complete.

## The users table shows accesses as one filled bar per group, not a list of names

**Context** — The Accesos column printed two comma-separated name lists (eye = readable, pencil =
editable), truncated at eight names. It answered "which accesses" and nothing about how much of the
product the user can reach, and every row was a paragraph of text.

**Decision** — One bar per access group the user holds something in. The bar's width is the group's
whole catalog and each segment is the share of it granted at one level: 3 of a group's 8 accesses at
TODO is 3/8 of the bar, painted in that level's own colour with its own icon (`accesoAcciones`),
widest level first. The empty tail is what the user does not hold, so the track is an outlined box
rather than a filled one, and a segment's `title` reads "Todo: 3/8".

VER's icon is `mdi--eye`; every other icon on the page stays on `fa`. MDI's solid eye reads at 12px
where FontAwesome's outline one does not, but its glyph fills only 62% of its 24×24 box where the FA
ones fill theirs edge to edge — so on its own it renders a size smaller than the shield beside it.
The correction is `transform: scale(1.25)` on `[class*="mdi--eye"]`, in the three components that
draw a level icon. A transform rather than a bigger box: it fixes the optical size without changing
the layout around it, which is what moving the whole set to MDI would have cost (the FA shield is
drawn 1280 wide on a 1536 grid and is the shape that was wanted).

`AccessGroupBars.svelte` owns the whole thing — the merge, the markup and the styles — so the cell
renderer is one tag and `UsersTab` carries no bar code at all. Two small pure functions in
`users-profiles.ts` remain because they are the only part worth a test:
`buildEffectiveNivelByAcceso` (profiles + own grants, highest wins) and `buildAccessGroupBars`
(a `filter`/`map` per group, no intermediate index — the catalog is 36 rows).
`summarizeUsuarioAccesses` and `formatAccessSummary` are deleted.

**Rationale** — Groups are the unit an operator thinks in ("can they touch Finanzas?"), and a
proportion is readable at a glance where a name list is not. It reads as intended because every
access in `access.toml` declares exactly levels 1 and 4, so a bar is at most two segments: green eye
(VER) and purple shield (TODO). An access catalog that started offering CREAR/EDITAR would draw four
segments per bar with no code change, but the bar would get busy — that is the pressure point.
Groups the user holds nothing in are left out; nine empty tracks per row would be most of the table.

## "Todos" is implicit — ticking every declared sub-access is what stores it

**Context** — The editor offered a synthetic "Todos" checkbox alongside the declared sub-accesses,
with an exclusivity rule in both directions: ticking Todos cleared the rest, ticking anything else
cleared Todos. Two ways to express "all of them" and a rule to keep them apart.

**Decision** — "Todos" is no longer a checkbox. `buildSubAccesoOptions` returns only what the catalog
declares; `toggleSubAcceso` takes the declared ids and runs expand → toggle → collapse
(`expandSubAccesos` / `collapseSubAccesos`), so a selection covering every declared sub-access is
stored as `[SUB_ACCESO_TODOS_ID]` and a stored Todos expands back into every box before one is
unticked. `isSubAccesoChecked` is what ticks the boxes for a stored Todos. Nothing changed on the
wire or in the backend: `security/perfiles.go` already accepts id 1 on any access declaring
sub-accesses, and `hasSubAcceso` already treats it as satisfying every check.

**Rationale** — One control per thing that can be granted, and the "all" state is a fact about the
selection rather than a separate option to keep consistent with it. **Consequence worth knowing:**
storing Todos means "everything this access ever declares", so a sub-access added to `access.toml`
later is granted retroactively to whoever holds Todos today. Storing the enumeration instead would
not do that — this is the behaviour that was asked for, and it is what makes Todos worth storing.

## The sub-accesses section is a 2-column grid, cards and checkboxes alike

**Context** — The section was a `flex flex-wrap` row of cards with a 240px floor, and each card's
checkboxes were a wrapping flex row. A single flex line stretches to `align-content: stretch`, which
is what gave the cards their empty vertical run, and ragged wrapping left the labels unaligned.

**Decision** — The cards are `grid grid-cols-2 gap-8 items-start` and the card's `min-width` floor is
gone; inside each card the checkboxes are `grid-template-columns: repeat(2, minmax(0, 1fr))`.

**Rationale** — `items-start` is what stops a card from growing to its neighbour's height, and the
inner grid puts the checkboxes on a fixed pair of columns so they line up regardless of label length.

## Bold card titles cap at 15px across the accesses view, and the card's left gutter is 9px

**Context** — The bold titles of the four card kinds in the accesses view (dropdown option, granted
chip, profile chip, sub-access card) were not one size: the profile chip had no explicit size at all
and inherited a larger one. `.acceso-card` also reserved 13px of left padding for a 5px colour bar
that is transparent on an ungranted access, which read as an empty strip on every dropdown card.

**Decision** — Every bold card title in the view is `text-[15px]` and none goes above it — 16px was
tried first and looked oversized against the 13px route line under it. The card's left padding drops
to 9px (4px of visible bar + a 5px gutter).

**Rationale** — One size for the titles is what makes the card kinds read as the same object at
different stages. The gutter still clears the bar, so a granted card's colour never touches its text.

## The accesses dropdown is 75% wide, and the user layer's tab survives a row click

**Context** — The `ACCESOS ::` options panel was widened to `calc(200% + 10px)` of its own input,
which is exactly the full width of the two-column `SearchDualCard` — it covered the `PERFILES ::`
input entirely. Separately, `openEditUsuarioLayer` forced `usuarioLayerView` back to
`USUARIO_VIEW_INFO`, so clicking a second user while reviewing accesses snapped the layer back to
Información.

**Decision** — The panel is `w-[calc(150%+7.5px)] md:-ml-[calc(50%+7.5px)]`: percentages resolve
against the half-width input, so `1.5·half + 7.5px` is 75% of the container, still right-aligned with
its own input. `openEditUsuarioLayer` no longer touches `usuarioLayerView`;
`openCreateUsuarioLayer` still resets it to Información.

**Rationale** — Keeping the right edge pinned and growing leftwards leaves the left input's label
readable while the two option columns still fit. The sticky tab treats "review accesses across
several users" as one task; creating a user still starts on Información because the required
identity fields live there.

## Profiles and users now save the same grant list, so the editor became one shape

**Context** — A profile saved two arrays (`Accesos`, `SubAccesos`) and a user saved a third
(`AccessLevelIDs`) that could not express a sub-access at all. Four pure helpers existed only to
convert between those spellings and the editor's two maps: `packSubAccesos`, `unpackSubAccesos`,
`decodeAccessLevelIDs`, `encodeAccessLevelIDs`.

**Decision** — Both records save `AccesosGrants: IAccesoGrant[]` — one record per access, carrying
its `Nivel` and its `SubAccesos`. The four helpers collapse into `toAccesoGrants(accesosMap,
subAccesosMap)` and `fromAccesoGrants(grants)`. `accesosMap` becomes `Map<number, number>`: one level
per access, not a list. `AccessCard`'s level buttons are now a one-of choice — clicking the selected
one revokes the access, which is the row's only "off" state. Both cache versions bump (`perfiles` to
4, `users` to 2) because a cached record in the old shape hydrates as granting nothing, and saving it
would silently revoke everything. Backend side and the storage format:
`docs/ACCESO_GRANTS_STORAGE_PLAN.md`.

**Rationale** — The user layer needed sub-accesses, and the honest way to get them was to stop having
two shapes rather than to add a fourth array. The cost is the one-of level buttons — a visible
behaviour change on the Profiles tab, confirmed before it was built, and not a data loss: the merge
always kept only the highest level.

## Sub-accesses get their own section, below the granted accesses

**Context** — `AccessCard` renders its sub-access checkboxes under the card, which is right on the
Profiles tab where the cards are the page. Reusing that card as a dropdown option put the checkboxes
inside the search dropdown: a list that scrolls, filters, and closes.

**Decision** — `AccessCard` takes `hideSubAccesos`, which the user layer sets on its dropdown
options. The checkboxes move to a **Sub-accesos** section under the chips: one card per granted
access that declares sub-accesses, its name over its checkbox row. Ticking uses the same
`toggleSubAcceso` rule, so "Todos" stays exclusive in both directions.

**Rationale** — A checkbox inside a dropdown is a target you have to keep the dropdown open to hit,
and it hides an access's real state behind a search term. Below the chips, the section reads as what
it is: a footnote on the accesses already granted, and it is absent entirely when none of them
declares a sub-access — which today is every access but one.

## The user layer is two tabs, and titles itself with the user

**Context** — With the access selector in it, the user layer was one long column: identity, then a
grid of access cards, then the password fields — and its title said "Actualizar Usuario", which is
the operation, not the record. Reaching the password meant scrolling past everything.

**Decision** — `Layer` already renders an `OptionsStrip` from its `options` + bindable `selected`, so
the layer declares two: **Information** (identity + password) and **Access** (the dual selector).
The title is now the user's name — full name, else the login, else "New User|Nuevo Usuario" — with
`Layer`'s new `titleIcon` prop drawing a user glyph ahead of it. The view resets to Information every
time the layer opens.

**Rationale** — The inactive tab unmounts rather than being hidden, which is fine here and checked in
the browser: the selector rebuilds its editable map from `AccessLevelIDs` on mount, and that field is
written on every grant, so a round trip through Information loses nothing. Keeping both mounted would
mean guarding the selector's two sync effects against a view that is not on screen. `titleIcon` went
on `Layer` rather than being faked into the title string because `title` is rendered as text through
`ui.translate`, so an icon class in it would print as characters.

## The user layer grants access through the same AccessCard the profiles tab uses

**Context** — The user layer offered accesses as a flat list of `"<access> (<LEVEL>)"` strings, one
option per access × level, and echoed every grant back as its own chip. Reading a user's grants meant
reading "Activos (VER)", "Activos (TODO)" as unrelated lines, and the profiles tab right next to it
already had a card that says the same thing in one row. The two shapes did not agree: a user stores
grants flat, as `accesoID * 10 + nivel`, while `AccessCard` edits a profile's `accesosMap`.

**Decision** — One access per option **in the dropdown**, rendered by `AccessCard` itself, two
columns wide, with its level buttons doing the granting. The layer edits through a profile-shaped
`accessEditForm` holding only an `accesosMap`, and two effects keep it in step with the stored field:
one decodes `AccessLevelIDs` whenever the layer swaps user, the other re-encodes on every map change.
Below, a granted access gets a **simpler, read-only card**: a rectangular strip whose left edge is
the level column — the widest granted level's icon on top, its colour bar filling the rest — and one
line each for the access name and its route, both cut with an ellipsis. Its sole action is the chip's
trash button. Fixed rows and no wrapping is the whole point: a card that grew to fit its longest
access name left a row of them ragged, and the list can hold dozens. That card
is the chip element itself: SearchDualCard's own wrapper span is collapsed with `display: contents`
and the snippet emits contents only, so what renders is one div with one shadow rather than three
nested boxes fighting each other's padding. It carries **no border in any state** — `._chip:hover`
ties on specificity with a plain `._chip-right` override and wins on source order, so the chip's own
frame kept reappearing around the card on hover; the override now hangs off `._container` to break
the tie, and the hover cue is an `outline`, which never takes part in layout.
`buildAccesosCatalog` / `buildRouteCatalogIndex` moved out of `ProfilesTab` into `users-profiles.ts`
so both tabs read one catalog, and `decodeAccessLevelIDs` / `encodeAccessLevelIDs` are pure and
unit-tested. Sub-accesses stay out: `AccessCard` now hides that row when there is no `subAccesosMap`,
which is exactly the user case, because a user record has nowhere to store them.

**Rationale** — Homologating the *editor* on `AccessCard` costs the conversion pair but buys one
visual and one interaction model wherever levels are chosen; a second editing card that merely looks
like the first is the version that drifts. The summary below is deliberately **not** that card: two
editors for the same grant on one screen is the ambiguity, not the convenience — the list grants, the
cards report. The adapter is the source of truth while the layer is open and
`AccessLevelIDs` is derived from it, so there is one direction of truth and no ping-pong between the
two effects. Cost: a grant now takes a hover plus a click on the desktop (the level buttons only
appear on hover, inherited from the profiles tab), and picking an option by keyboard or through the
product's agent has no level attached, so it grants the widest level the access declares.

## Sub-accesses are a checkbox row under the card, and "Todos" is exclusive

**Context** — An access may declare up to 13 sub-accesses: flags a handler reads to decide what it
may do inside a route it is already authorized for. They had to be editable on the profile, and the
access card is deliberately compact — as tall as its two text lines, with its level buttons floating
over it on hover, which is the decision the entry below this one records. Putting anything else
inside the card undoes that.

**Decision** — A row of checkboxes **below** the card, inside the same grid cell, shown only while
the access is granted and only for the accesses that declare sub-accesses. The card keeps its
height; the cell grows. `AccessCard` gained a wrapper element to hold both. Sub-access id 1 is
offered as a synthetic first option, "Todos": selecting it clears the rest, and selecting anything
else clears it.

**Rationale** — The row is visible rather than hover-revealed because a permission the operator
cannot see is one they will not grant, and because a hover strip has no equivalent on touch. It goes
under the card rather than in it so the 35 accesses that declare no sub-access are unaffected — one
access declares them today.

"Todos" is exclusive in both directions because it *satisfies every check on its access*, so holding
it alongside individual sub-accesses is contradictory: the individual ticks would be stored, read,
and then have no effect. Making it clear the others says that in the UI instead of leaving the
operator to discover it. It is never declared in the catalog, so it is offered on any access that
declares sub-accesses at all and refused on one that declares none.

The checkbox itself is `@genix/ui`'s `Checkbox` in controlled mode, because the value lives in a
`Map` rather than on an object — see that package's `form/RATIONALE.md`.

## The wire shape is `accesoID * 100 + subAccesoID`, and the rules are a pure module

**Context** — The profile posts its sub-accesses to the backend, which validates them against the
embedded catalog and packs them into a user's binary grant blob. The frontend needed a shape to send
and somewhere to put the editing rules.

**Decision** — `SubAccesos: number[]` of `accesoID * 100 + subAccesoID`, mirroring how `Accesos`
already encodes `accesoID * 10 + nivel`. `subAccesosMap: Map<accesoID, subAccesoID[]>` is the
editable form shape and is stripped before the POST, exactly like `accesosMap`.
`buildSubAccesoOptions`, `toggleSubAcceso`, `packSubAccesos` and `unpackSubAccesos` live in
`users-profiles.ts` — no Svelte, no fetch — and are unit-tested there.

**Rationale** — Readable on purpose: the profile is what a human edits, so `1002` is legible in a
database console and in a log line. Binary encoding happens exactly once, in
`backend/core/accesos-blob.go`, when a *user's* grants are computed. Three hand-written parsers of
that binary format already exist; a fourth, in the browser's editing path, would be one too many.

Two rules earn their place in the pure module rather than the component. `packSubAccesos` **drops**
any sub-access whose parent access is no longer granted, because the backend rejects the whole
profile over one — dropping is right here and rejecting is right there: this is clearing a leftover
of the operator's own editing (tick sub-accesses, then clear the access), not accepting an
unauthorized grant. And `buildSubAccesoOptions` returns **nothing** when the catalog's two parallel
arrays disagree in length, rather than a shifted list: naming the wrong sub-access is how an
operator grants the wrong permission. The backend refuses to load a mismatched pair at all, so
reaching that branch means the text parser dropped something.

`PerfilesService.useCache.ver` went to 3. A cached v2 record has no `SubAccesos`, so it would edit
as if the profile granted none — and saving it would silently revoke every sub-access it holds.

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
