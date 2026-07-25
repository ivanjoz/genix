# Plan: Extract a reusable UI + utilities package (`genix-ui`)

**Goal:** move everything that is **not business logic** into an independent, reusable
repo consumable by other projects. Distribution: **git submodule**. Coupling into the
host app is broken by **injecting the seam via Svelte context/props** (no global
singletons imported across the boundary).

**Status:** reusable UI extraction implemented. The former `ui-components/**` tree,
`SideMenu`, `MobileMenu`, `HTMLEditor`, page/layout UI state, initial utilities, charts,
Excel, the generic HTTP transport, cache engines, and service-worker transport
now live in `genix-ui`. Genix business infrastructure is connected through thin host
adapters. Authentication policy and agent transport remain host concerns.

---

## 1. Ownership boundary

### Moves into `genix-ui`

| Area | Scope | Notes |
|------|-------|-------|
| UI components | former `ui-components/**` | Complete: now under root-level package folders |
| Pure utilities | `libs/funcs/**`, `date.ts`, `sharedHelpers.ts`, pure parts of `helpers.ts`, assets | No host configuration |
| UI runtime | page metadata, device/layout state, header settings, i18n adapter | Created per component tree |
| Generic agent UI | `genix-ui/agent/registry.ts` | Complete |
| Domain renderers | refactored `SideMenu`, `MobileMenu`, `HTMLEditor` | See §4 |
| Generic stores | host-owned in-memory images and field persistence | Notifications remain an injected visual adapter |
| Generic caches | delta, records-by-ID, group, route, and IndexedDB caches | Tenant GET/navigation are injected |
| Excel | workbook import/export and fluent builder | WASM URL and translation are injected |
| HTTP transport | GET, POST, PUT, and upload client | Auth, routes, cache IO, and reporting are injected |

### Stays in the Genix host

- `domain-components/Page.svelte`. It remains the business-aware page coordinator.
- `Header.svelte`, `AppHeader.svelte`, `HeaderConfig.svelte`, and request-log UI.
- Authentication, access policy, login forms, route declarations, and router bindings.
- `core/security.ts`, `core/modules.ts`, `services/**`, product search, and concrete menus.
- Company/API/environment configuration in `core/env.ts`.
- `Core.module`, `Core.ecommerce`, and storefront `mainMenuOptions`.
- The Genix agent bridge: `core/agent/commands.ts`, `sse.ts`, models, backend routes, and
  security-aware menu commands. Reusable agent UI can be extracted later through props.
- HTTP authentication/access policy and DOM progress rendering.

## 2. Corrected dependency findings

The reusable seam is wider than three imports:

```text
genix-ui components ─▶ injected UI runtime capabilities + internal agent registry
genix-ui host ───▶ one injected auth + tenant + route + worker/reporting configuration
genix-ui cache ─▶ host-derived tenant identity + GET/navigation adapters
genix-ui HTTP ───▶ host-derived auth + API routes + cache IO + request reporting
core/agent ─────▶ Genix security + menus + API routes + backend wire protocol
SideMenu ───────▶ menu declarations + access policy + SvelteKit router + UI renderer
MobileMenu ─────▶ storefront actions + LoginForm + drawer/grid UI
Page ───────────▶ auth + navigation + route persistence + UI page metadata
HTMLEditor ─────▶ Rooster editor + browser detection + generic SVG helper
```

Consequences:

- Svelte context is suitable for components initialized under a provider.
- Context is **not** a replacement for dependency injection in module-level HTTP/cache
  services. Those modules require factories or explicit constructor parameters.
- `$app/*` imports should be removed from package code so the package can run in plain
  Svelte/Vite as well as SvelteKit.
- `helpers.ts` must be split: pure functions move; Notiflix, DOM progress, storage scope,
  and image-worker orchestration move only after receiving explicit adapters.

## 3. UI runtime and Core migration

The package owns UI state. The host creates one runtime per rendered component tree and
business components set UI values through its typed API.

```text
genix-ui/runtime/
  UiProvider.svelte        # Creates/provides one runtime for its descendant tree
  context.svelte.ts        # Typed useUI()/provideUi()
  create-ui-state.svelte.ts
  types.ts                 # Small capability contracts
  i18n.ts
```

State moved out of `Core` in the first slice:

- Page UI: `pageTitle`, `pageOptions`, `pageOptionSelected`, `useTopMinimalMenu`.
- Navigation UI: `mobileMenuOpen`, `deviceType`.
- Header UI: settings-open state.

State that stays host-owned:

- `Core.module`, `Core.ecommerce`, `mainMenuOptions`.
- Request activity and business initialization state.
- Authentication/access readiness and application initialization.
- Tenant, endpoint, token, CDN, and service-worker configuration.

Root layouts provide the application runtime with `provideUi(ui)`. Fresh `mount()` trees
must provide that runtime again because Svelte context does not cross mount roots.
`UiProvider` is also exported for consumers that prefer component-based composition.
Stateful components require a runtime context; there is no implicit fallback.

`Page.svelte` stays local. It continues to own authentication, redirect, route-scoped tab
persistence, and cleanup. It sets `ui.state.pageTitle/pageOptions/useTopMinimalMenu` and
reads/writes `ui.state.pageOptionSelected`. Existing pages that read
`Core.pageOptionSelected` are migrated directly to the UI runtime.

Non-component libraries use explicit factories:

```ts
// Transport configuration stays explicit because these calls can run outside components.
const http = createHttpClient({
  makeRoute,
  getToken,
  reportActivity,
  notify
})
```

## 4. Component splits

### 4.1 SideMenu

Package renderer:

```ts
export interface MenuItem {
  id?: number | string
  label: string
  route?: string
  icon?: string
  shortLabel?: string
  children?: MenuItem[]
  meta?: Record<string, unknown> // Host-only flags remain opaque to the renderer.
}

export interface SideMenuProps {
  model: MenuItem[]
  activePath: string
  open: boolean
  canAccess?: (item: MenuItem) => boolean
  translate?: (label: string) => string
  onNavigate?: (route: string) => void | Promise<void>
}
```

The package keeps rendering, collapse/expand, active-group detection, responsive drawer,
and view transitions. The host maps `core/modules.ts` into `MenuItem[]`, supplies access
policy and navigation, and binds the open state. `AppHeader` toggles that bound state
instead of receiving a function through `Core.toggleMobileMenu`.

### 4.2 MobileMenu

The existing `MobileMenu.svelte` is a storefront action menu, not another `SideMenu`.

- Package `MobileMenu`: drawer/backdrop, transitions, generic item grid, close behavior,
  bindable open state, item-selection callback, and optional content snippet.
- Host wrapper: `mainMenuOptions`, login/register/order/store actions, `LoginForm`, URL
  navigation flags, and any business-specific layers.

### 4.3 HTMLEditor

The editor is already mostly generic. The package version:

- Uses a platform-neutral browser check instead of `$app/environment`.
- Keeps Rooster setup, formatting, tables, sanitization, and toolbar UI.
- Supports direct `value` binding plus `onChange`; the existing `saveOn/save` pattern may
  remain as a convenience adapter if it does not complicate the public API.
- Receives future upload/link/media behavior as callbacks rather than importing services.
- Moves its palette, icons, and editor types with the component.

### 4.4 Page

`domain-components/Page.svelte` is **not extracted**. It remains the host adapter between
business page lifecycle and package-owned UI state. Its visual container may remain in the
same file; creating a package `PageShell` is not required.

## 5. Package structure and consumption

```text
genix-ui/
  package.json
  svelte.config.js
  tsconfig.json
  README.md
  LICENSE
  index.ts
  charts/
  runtime/
  menu/
  editor/
  utilities/
```

- Git-submodule/workspace exports point directly to root-level source so editor navigation
  reaches the implementation. A future registry build needs a dedicated staging/source
  directory because `svelte-package` cannot safely use the package root as its input.
- Add as `frontend/packages/genix-ui` and register it in the Bun workspace.
- Consume through package exports such as `@genix/ui`, `@genix/ui/menu`, and
  `@genix/ui/editor`. Do not repoint host `$components`/`$libs` aliases across the boundary.
- `svelte` is a peer dependency. Runtime libraries are dependencies of the entry points
  that use them; `@sveltejs/kit` is not a required peer.
- Tailwind v4 consumers add explicit `@source` coverage for the root component folders and adopt the documented
  `--spacing: 1px` theme contract.
- `plugins.js` remains a Genix build optimization, not a consumer requirement.

## 6. Reusable data utilities

The generic cache engines ship from `@genix/ui/cache`. Excel ships from `@genix/ui/excel`,
and the reusable request transport ships from `@genix/ui/http`. Genix injects tenant
identity, authenticated routing, notifications, request reporting, and the WASM asset URL.
The worker request contract also lives beside the cache implementation.

These utilities remain valid extraction candidates:

- Remaining service-worker lifecycle policy.
- Worker modules with application-specific build/output contracts.
- Notifications and chat history after database names receive an application namespace.
- Agent chat UI after transport, model loading, and command execution become injected.

They may become subpath exports or separate packages such as `@genix/data` and
`@genix/agent`. They must not receive one replacement `UiEnv` god object.

## 7. Execution phases

1. **Package foundation — complete** — independent flat source-first repository,
   exports, workspace/submodule wiring, and package validation.
2. **Initial UI runtime — complete** — create UI state per tree, wire both root layouts
   and fresh `mount()` trees, then migrate page/device/menu/header fields out of `Core`.
3. **Page host migration — complete** — keep `Page.svelte` local while moving its page
   metadata to the UI runtime; migrate consumers of `Core.pageOptionSelected`.
4. **Leaf extraction — in progress** — parsers, unmarshalling, date/week conversion, and
   compact object mapping now ship from `@genix/ui/utilities`; canvas and compact cell
   charts ship from `@genix/ui/charts`; the data-driven AST `Renderer` ships from the root
   export. The unused billboard wrapper and the never-imported vendored Typed-IDB adapter
   were removed because their APIs were obsolete and superseded by Dexie. All former
   `ui-components` groups, assets, and the agent registry now live in the package.
5. **Initial component splits — complete** — refactor/extract `SideMenu`, `MobileMenu`,
   and `HTMLEditor`; retain thin Genix wrappers for policies and business content.
6. **Stateful UI extraction — complete** — overlays, media, tables, forms, cards,
   navigation, files, and agent registration use package-owned state plus explicit host
   adapters from `core/ui-runtime.ts`.
7. **Cache/SW extraction — complete** — cache engines, request contracts, worker event
   shell, RPC handlers, and browser client live in the package. Tenant-aware routing and
   UI reporting are injected.
8. **Excel/HTTP transport extraction — complete** — Excel receives an explicit WASM and
   translation runtime; the HTTP client receives auth, routing, cache, notification, and
   activity adapters.
9. **Cached service extraction — complete** — the Svelte 5 `GetHandler`, grouped-cache
   orchestration, and stable name normalization live in the package. Genix injects access
   policy, authenticated transport, record-by-ID resolution, and notifications through a
   thin zero-argument adapter. DOM progress UI remains host-owned.
10. **Unified runtime composition — complete** — `createUiRuntime` accepts routing, auth,
   tenant identity, service-worker URL, reporting, and process notifications once, then
   returns UI state, HTTP, cache, worker, `GetHandler`, uploads, in-memory images, field
   persistence, and record resolution as one flat runtime.
11. **Consumer documentation and smoke test** — install the submodule into a second minimal
   Svelte app and verify package exports, styling, and runtime isolation.

Verification after every phase:

- Package: `bun run check` and export/type resolution validation.
- Admin host: `bun run check` and `bun run build:main`.
- Storefront: `bun --cwd webpage run check` and `bun run build:store`.
- Add a real forbidden-import check; the currently documented `scripts/analyze-dag.ts`
  does not exist.
- Manual smoke: auth redirect, page tab restore, desktop/mobile navigation, access
  filtering, overlays, editor formatting, image upload, tables, and cache inspector.

## 8. Open risks and required decisions

- The package repository is `git@github.com:ivanjoz/genix-ui.git` under the MIT license.
- The custom Tailwind spacing scale is part of the component design contract and must be
  explicit for every consumer.
- UI runtime migration touches many page imports; complete it in one coherent phase because
  this pre-alpha project does not require compatibility shims.
- Fresh Svelte `mount()` trees do not inherit parent context and require their own provider.
- IndexedDB access uses Dexie everywhere; the vendored `typed-idb` adapter was never imported
  and was dropped rather than carried into the package.
