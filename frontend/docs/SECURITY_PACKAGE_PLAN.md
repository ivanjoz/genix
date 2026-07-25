# Plan: move the security mechanism into `@genix/ui`

Goal: make token lifecycle + access-control reusable by other projects, keeping Genix-specific
policy (route catalog, storefront public routes, Spanish messages, `Env` wiring) in the host app.

## 1. Current dependency audit (`frontend/core/security.ts`, 395 lines)

| Dependency | Nature | Decision |
|---|---|---|
| `Env.appId` | host namespace | inject as `storageNamespace` |
| `LocalStorage`, `IsClient` | SSR shim | package-owned (`esm-env` `BROWSER` + internal shim, same idiom as `http/get-handler.svelte.ts`) |
| `Notify` (notiflix) | host UI | inject via existing `UiNotificationAdapter` |
| `throttle` | pure | already in `@genix/ui/utilities` (`utilities/ui.ts`) |
| `decrypt` (AES-GCM) | pure crypto, **only consumer is security.ts** | **move** to `packages/genix-ui/utilities/crypto.ts` |
| `checksum`, `base64ToUInt16` | pure | already in `@genix/ui/utilities` |
| `IUser`, `ILoginResult` | host business types | generic `UserInfoType` param + minimal `SecurityLoginResult` |
| `getAccessEntriesForRoute` | reads `backend/access_list.yml` | inject as `resolveRouteAccessEntries` |
| `isPublicFrontendRoute` | Genix storefront rule | stays in host, injected as `isPublicRoute` |
| `Env.clearAccesos`, `Env.getToken`, `Env.canUserAccessRoute`, `Env.navigate` | host globals | host wires them to the instance; package takes `onLogout` |
| `registerReloadLogin` (breaks `services/login.ts` cycle) | host | becomes `security.setSessionRefresher(fn)` |
| Spanish literals ("La sesión ha expirado…") | host copy | inject `messages` |

### Dead code found — delete, do not port (pre-alpha, no back-compat)

- `getShowStore` — 0 consumers.
- `UserInfoParsed` interface — 0 consumers.
- `Params` bag — the only real consumer is `Params.getFechaUnix()` in
  `routes/logistics/purchase-management/supply-management.svelte.ts:279`.
  `domain-components/SystemParametersEditor.svelte:10` imports `Params` but never uses it (unused import).
  → drop `Params` entirely; add `getFechaUnix()` to `packages/genix-ui/utilities/date.ts`
  (next to the existing `zoneOffset` / `dateToFechaUnix`) and fix the two files.
  `setValue` / `getValue` / `getValueInt` / `toSunix` / `sunixTime` — 0 consumers, deleted.
- `debugger` statement in `isTokenValid()` — removed.
- `console.log` of the AES nonce inside `decrypt` — removed while moving (leaks crypto material to console).

## 2. New package module: `packages/genix-ui/security/`

```
security/
  accesos.ts             # pure bit-packing + stored-payload codec, zero config
  types.ts               # SecurityLoginResult, CreateSecurityOptions, SecurityRuntime
  create-security.ts     # createSecurity() factory — the whole stateful mechanism
  index.ts
  SECURITY.md            # module doc (package convention: CACHE_BY_IDS.md, DELTA_CACHE.md)
```

### `accesos.ts` (pure, exported for reuse/testing)

`normalizeAccesoNivel`, `makeAccesoNivelUint16`, `getAccesoNivelSearchRange`,
`hasPackedAccesoInRange`, `wrapAccesosComputed`, `decodeStoredAccesosComputed` — moved verbatim
(they mirror the backend packing, so behaviour must not change).

### `types.ts`

```ts
export interface SecurityLoginResult {
  UserToken: string; UserInfo: string; AccesosComputed?: string
  TokenExpTime: number; CompanyID: number
}
export interface SecurityRouteAccessEntry { id: number }

export interface SecurityMessages {
  sessionExpired: string
  sessionExpiresIn: (minutes: number) => string
}

export interface CreateSecurityOptions<UserInfoType> {
  storageNamespace: string                 // Env.appId
  onLogout: () => void                     // host navigates to /login
  messages: SecurityMessages
  notify?: Partial<UiNotificationAdapter>
  resolveRouteAccessEntries?: (route: string) => SecurityRouteAccessEntry[]
  isPublicRoute?: (route: string) => boolean
  getCompanyID?: () => number
  tokenRefreshThresholdSeconds?: number    // default 40*60
  tokenCheckIntervalSeconds?: number       // default 4*60
  refreshLockSeconds?: number              // default 30
  autoStartRefreshCheck?: boolean          // default false; Genix passes true
}

export interface SecurityRuntime<UserInfoType> {
  getToken(silent?: boolean): string
  checkIsLogin(): 0 | 2 | 3
  isLogged(): boolean
  isTokenValid(): boolean
  getUserInfo(): UserInfoType | null
  setUserInfo(userInfo: UserInfoType): void
  parseLogin(login: SecurityLoginResult, cipherKey: string): Promise<void>
  checkAcceso(accesoID: number, nivel?: number): boolean
  canAccessRoute(route?: string | null): boolean
  clearSession(): void
  setSessionRefresher(refresh: () => Promise<unknown>): void
  startRefreshCheck(): void
  stopRefreshCheck(): void
  initRefreshCheck(): void
}
```

### `create-security.ts`

One closure holding what `AccessHelper` + the module-level token code hold today:
`storedAccesos` string, decoded `Uint16Array`, `Map` result cache, `userInfo`, refresh interval id,
session refresher. Same storage keys (`<ns>UserToken`, `<ns>TokenExpTime`, `<ns>TokenCreated`,
`<ns>UserInfo`, `<ns>Accesos`, `<ns>CompanyID`, `<ns>TokenRefreshLock`) so existing sessions keep
working and `webpage/components/UsuarioMenu.svelte` (reads `genixUserInfo` straight from storage)
is unaffected. Same cross-tab refresh lock, same 15/5-minute throttled warnings, same
`AccesosComputed` checksum wrapping. `autoStartRefreshCheck` replaces the current
module-import `setTimeout(initTokenRefreshCheck, 1000)` side effect.

`canAccessRoute`: public route → `true`; no catalog entry for the route → `true` (unchanged
permissive default); otherwise `some(entry => checkAcceso(entry.id, 1))`.

### Package wiring

- `package.json`: add `"security"` to `files`, add the `./security` export block (mirrors `./http`).
- `index.ts`: `export * from './security/index.js'`.
- `utilities/index.ts`: export `decrypt` from the new `crypto.ts`, and `getFechaUnix` from `date.ts`.
- `README.md`: a "Security" section — options table + host-owned policy note.

## 3. Host app changes

**`frontend/core/security.ts`** shrinks to configuration (~55 lines): keeps
`isPublicFrontendRoute` (storefront rule), builds the instance, wires
`Env.getToken` / `Env.canUserAccessRoute` / `Env.clearAccesos`, and re-exports
`getToken`, `checkIsLogin`, `isLogged`, `isTokenValid`, `canUserAccessRoute`,
`registerReloadLogin`, plus the instance as `export const security`.

Call-site edits (rename `accessHelper` → `security`, method rename `parseAccesos` → `parseLogin`):

| File | Change |
|---|---|
| `services/login.ts` | `accessHelper.parseAccesos(...)` → `security.parseLogin(...)`, `accessHelper.isTokenValid()` → `security.isTokenValid()` |
| `domain-components/HeaderConfig.svelte` (3 sites) | `accessHelper.getUserInfo/setUserInfo` → `security.*` |
| `domain-components/SystemParametersEditor.svelte` | delete unused `Params` import |
| `routes/logistics/purchase-management/supply-management.svelte.ts` | `Params.getFechaUnix()` → `getFechaUnix()` from `@genix/ui/utilities` |
| `libs/helpers.ts` | delete `decrypt` (moved to the package) |

Unchanged (they import free functions that stay exported): `routes/+layout.svelte`,
`domain-components/Page.svelte`, `domain-components/AppHeader.svelte`,
`domain-components/SideMenu.svelte`, `routes/login/+page.svelte`, `core/agent/commands.ts`,
`routes/system/server-panel/{MemoryView,DashboardView}.svelte`, `libs/http.svelte.ts`.

The storefront constraint holds: `webpage/` still never imports `$core/security`, so
`access_list.yml` stays out of the public bundle.

## 4. Verification

1. `cd frontend/packages/genix-ui && bun run check`
2. `cd frontend && bun run check` (or the repo's svelte-check task)
3. `rg -n "accessHelper|parseAccesos|Params\." frontend --glob '!*.md'` → no stale references
4. Manual: login → `genixAccesos` written and `isTokenValid()` true; side menu hides
   unauthorized routes; reload page → refresh interval starts; logout → all keys cleared + `/login`.

## 5. Execution notes (deviations from the plan above)

- **Expiry-warning bug fixed while moving.** The original branch order was
  `if (secondsToExpire < 15min) warn(15) else if (< 5min) warn(5)` — the 5-minute branch was
  unreachable. The package checks the 5-minute window first.
- **`getUserInfo()` is now honestly typed `UserInfoType | null`** (the old `AccessHelper` cast
  `null as unknown as IUser`). That surfaced 12 real nullability errors in
  `domain-components/HeaderConfig.svelte`; fixed with an early return in `saveUsuario` and an
  `{#if userInfo}` guard around the profile inputs. Pre-existing bug left in place and *not*
  fixed (out of scope, worth a follow-up): `saveUsuario` notifies "Los password no coinciden"
  but does not `return`, so a mismatched password is still submitted.
- **Added `security/accesos.test.ts`** (6 tests) covering backend bit-packing parity,
  payload round-trip, checksum tamper rejection, and level/id isolation.
- `bun run check` in the package: 0 errors. `bun test`: 17 pass.
  `frontend`: the 12 remaining `svelte-check` errors are pre-existing, in 7 untouched files.
  `bun run build:main` succeeds, including static prerender (SSR-safe storage shim verified).

## 6. Follow-up: one configuration entry (`libs/ui-runtime.svelte.ts`)

Second pass, per request: the app must configure `@genix/ui` in exactly one call, and that
call must not live in `libs/http.svelte.ts`.

- **Package** — `createUiRuntime` now takes a `security` options block, creates the security
  instance itself, and exposes it as `runtime.security`. `getToken`, `canAccessRoute` and
  `onUnauthorized` became optional and default to that instance (`canAccessRoute` checks the
  current path through the new `getPathname` option, preserving the previous behaviour of
  blocking cached fetches on pages the user cannot open). `messages` and `onLogout` are
  optional with English/no-op defaults. `createSecurity` is still exported standalone.
- **`SecurityRuntime.setRouteAccessResolver`** — new registration hook. Needed because
  `webpage/` (the public storefront) shares `$libs` and imports the runtime file: a static
  import of the access catalog there would embed `backend/access_list.yml` in the public
  bundle. `routes/+layout.svelte` (the authenticated app) registers it instead.
- **`libs/ui-runtime.svelte.ts`** (new) — the single entry: `setFetchProgress`,
  `isPublicFrontendRoute`, the one `createUiRuntime<IUser>({ …, security: { … } })` call, and
  `export const security = genixUiRuntime.security`.
- **`libs/http.svelte.ts`** — reduced to importing that runtime and re-exporting
  `GET`/`POST`/`PUT`/`POST_XMLHR`/`buildHeaders`, the image converter, and `GetHandler`.
- **`core/security.ts` deleted.** All 10 consumers now use `security.*` from
  `$libs/ui-runtime.svelte` (`getToken`, `checkIsLogin`, `isLogged`, `canAccessRoute`).
  The `Env.getToken` / `Env.canUserAccessRoute` / `Env.clearAccesos` indirection hooks are
  removed from `core/env.ts`; `registerReloadLogin` is now `security.setSessionRefresher`.
- **Storefront logout fixed.** `webpage/components/UsuarioMenu.svelte` called
  `Env.clearAccesos?.()`, which was always `null` there (the hook was only assigned by the
  admin-only module) — logout silently did nothing. It now calls `security.clearSession()`,
  and `onLogout` returns to `/` on public routes because the storefront has no `/login`
  route. The component also reads the session through `security.getUserInfo()` instead of
  duplicating the storage-key parsing.
- **Bundle constraint verified after the change:** `rg 'frontend_routes|backend_apis'`
  → 0 hits in `webpage/build/`, 6 in the admin `build/`.

## 7. Notes / open points

- `packages/genix-ui` is a **git submodule** → two commits (package first, then host bump).
- Two moves are judgement calls, flagged rather than assumed: `decrypt` → `utilities/crypto.ts`
  and `getFechaUnix` → `utilities/date.ts`. Both are pure and single-consumer, so the package is
  the natural home; say the word if you'd rather they stay in `libs/helpers.ts`.
