# Plan: SvelteKit 3 migration + berryapps shell improvements

Source: berryapps commits `cc7d605..bee2604` and genix-ui `f1342fd..095291a` (6 commits genix's
checkout is behind). Status: **implemented, not committed.** Decisions: `RATIONALE.md`,
`domain-components/RATIONALE.md`.

## 1. Bump genix-ui to `origin/main` (6 commits)

| Commit | What genix gets |
|---|---|
| `d702fb6` | `SideMenu` optional `header(isMobile)` snippet (replaces logo + brand) |
| `2578df1` | `@genix/ui/notify` (toasts, loading overlay, `await confirmWarn()`), **host `notify` runtime option removed** |
| `3d4b853` | Mobile drawer slides on any `open` change; `footer` snippet in the drawer; `fa--trash`→`mdi--delete`, `fa--close`→`mdi--close-thick`; unlabelled field heights match labelled ones |
| `c8293e7` | Icon-only `Button` centered (gap moved to the label); selector/date-picker round buttons centered |
| `d338cd9` | Drawer closes **before** navigating (cold chunks no longer look like a dead tap) |
| `095291a` | Cache navigate uses Kit 3 `goto` options (`replace`, `reset`) — **requires Kit 3** |

The uncommitted `form/DateInput.svelte` change in genix's genix-ui checkout is kept (autostash).

## 2. Notiflix → `@genix/ui/notify` (forced by `2578df1`)

- `libs/helpers` `Notify` / `Loading` / `Confirm` (Notiflix wrappers) replaced by
  `notifySuccess/Failure/Warning/Info`, `showLoading/hideLoading`, `await confirmWarn()` — ~60 files.
- `libs/ui-runtime.svelte.ts` drops the `notify:` option; `<NotifyHost />` mounted in `routes/+layout.svelte`;
  notify z-indexes + Tailwind `@source` as in berryapps; `notiflix` dependency removed.

## 3. SvelteKit 3 (as berryapps `bee2604`)

- Kit `^3.0.0`, Svelte `^5.57.1`, Vite `^8.3.2`, adapter-static `^4`, vite-plugin-svelte `^7.3.1`,
  svelte-check `^4.7.6`, TypeScript `^6` (Kit 3 peers on 6).
- `svelte.config.js` → `sveltekit({ preprocess, compilerOptions.cssHash, adapter, files, prerender })` in
  `vite.config.ts`. `files.lib` (`pkg-core/lib`, does not exist) dropped. `tsconfig` extends `$app/tsconfig`.
- `$components/$core/$libs/$services/$domain` → `package.json` `imports` `#components/*` … with explicit
  extensions (~242 files / ~980 imports, scripted). `$routes`, `$ecommerce`, `$stores` checked for use.
- `$app/environment` → `$app/env` (17 files). `goto(..., { noScroll, replaceState })` in `core/store.svelte.ts`
  → `{ reset: false, replace: false }`.
- Service-worker esbuild alias plugin in `vite.config.ts` updated to the `#` names.
- **Storefront `webpage/`** migrated too, with the same `#` names (`../` targets + tsconfig `paths`).
- `$env/static/public` → `$app/env/public`, declared in a root `env.ts` (`files.src: '.'`).
- `CLAUDE.md`, `FRONTEND.md`, `.claude/skills/*` updated to `#…` imports.

## 4. Shell improvements from berryapps

- **Mobile header** (< 749px): only the info/notifications button stays; settings gear + reload move to the
  new `SideMenu` `footer` snippet (logo + gear `ButtonLayer` with its own open state + reload).
- `ReloadPageButton.svelte` extracted (shared by header and drawer footer).
- Hamburger `☰` text → `icon-[fa--bars]`.
- Mobile search: AgentChat kept as is. Module selector: skipped (genix has one module).
- Not ported: berryapps' `NotificationsButton` Tailwind class swap — it only replaces genix globals
  (`ff-bold`, `c-gray`) that berryapps lacks, and `ff-bold` is a font family, not a weight.

## 5. Verification

`bun install`, `svelte-kit sync`, `bun run check`, `bun run build` (main + webpage), then the
`agent-browser` skill on desktop and mobile widths: menu drawer, header buttons, a page with tabs, a toast.
