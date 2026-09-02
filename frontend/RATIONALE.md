## Svelte HMR is enabled in dev (no more full page reloads)

**Context** — `compilerOptions.hmr: false` was set in both `svelte.config.js` and
`webpage/svelte.config.js` since the initial Svelte frontend commit. With Svelte HMR
off, `@sveltejs/vite-plugin-svelte` has no way to swap a component in place, so every
edit to any `.svelte` file made the dev server issue a `full-reload`. That wiped the
in-memory delta caches and any open Layer/Modal state on each save.

**Decision** — Removed the `hmr: false` flag from both configs. The plugin's default
(HMR on in dev, ignored in build) now applies, so editing a component patches it in
place and the page keeps its state.

**Rationale** — The flag predates Svelte 5; the old `svelte-hmr` for Svelte 4 was
flaky enough that disabling it was a common workaround. Svelte 5 ships HMR in the
compiler itself and vite-plugin-svelte 7 relies on it. The cost is the usual HMR
caveat: module-level `$state` in a `*.svelte.ts` re-initializes when that module is
hot-replaced, so a stale-looking store after an edit is a reason to hard-refresh, not
a bug. Full reload is still the fallback for non-component modules Vite cannot patch.
