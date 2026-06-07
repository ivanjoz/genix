# Plan: Live Tailwind compilation via `@unocss/runtime`

## Goal
Agents generate HTML with arbitrary Tailwind classes that do **not** exist in
project source, and a WYSIWYG editor mutates classes (padding, color, border…)
at runtime. Build-time Tailwind (`@tailwindcss/vite`) can only ever see classes
present in source files, so it cannot cover these. Replace the hand-rolled
`stores/live-css.svelte.ts` micro-compiler with a real in-browser engine.

## Approach: on-demand `generate()`, NOT the MutationObserver runtime
Use **`@unocss/core`** (`createGenerator` → `await uno.generate(tokens)` →
`{ css }`), **not** `@unocss/runtime`. We already know exactly when content
changes (the editor store mutates, the agent regenerates HTML), so a global DOM
`MutationObserver` is wasteful and non-deterministic. Instead, keep the existing
`live-css.svelte.ts` shape — collect class strings reactively, inject one
`<style>` — and just swap the hand-rolled parser body for `uno.generate()`.
This is *less* code than the runtime wrapper and scoped to our state.

## Critical constraint: match the existing build `@theme`
`ecommerce/routes/tailwind.css` defines:
```css
@theme {
  --breakpoint-2xl: 1539px;
  --breakpoint-xl:  1379px;
  --breakpoint-lg:  1139px;
  --breakpoint-md:  749px;
  --spacing: 1px;          /* p-4 = 4px in THIS project, not 16px */
}
```
The UnoCSS runtime config **must reproduce this exactly**, or the same class
will render differently depending on whether it was in source (build engine) or
runtime-authored (UnoCSS engine):
- spacing base unit = `1px` (so `p-4` → `4px`)
- breakpoints md/lg/xl/2xl = 749/1139/1379/1539px
- palette colors `--color-1..10` exposed as theme colors so `bg-color-9`,
  `text-color-1`, `border-color-5` compile as real utilities

Open verification item: confirm whether `presetWind4`'s dynamic spacing honors a
`spacing` theme override to give `p-N = N*1px`. If its scale is fixed, configure
`theme.spacing` / preset options accordingly. This is the #1 thing to validate
before wiring it everywhere.

## Phase 1 — Dependency + generator config (no behavior change yet)
1. `bun add -D @unocss/core @unocss/preset-wind4` (preset name TBD after
   verifying spacing behavior; `preset-wind3` is the fallback if wind4 spacing
   can't be matched). NOTE: `@unocss/core`, not `@unocss/runtime`.
2. New module `ecommerce/stores/uno-generator.ts`:
   - `createGenerator({ presets, theme })` from a single source of truth shared
     with the palette (breakpoints, `spacing: '1px'`, `colors: { 'color-1': … }`).
   - **No preflight/reset** — emit utilities only; the page already ships
     Tailwind's preflight via `tailwind.css`, so a second reset would fight it.
   - Generator is created once (module singleton; `createGenerator` may be async).
3. Palette → theme colors: derive the `color-N` theme map from the active
   `ColorPalette`. Have utilities emit `var(--color-N)` so palette swaps stay
   live (via `generatePaletteStyles`) without regenerating CSS.

## Phase 2 — Rewrite `live-css.svelte.ts` around `generate()`
Keep the store + reactive-inject pattern; replace the body:
- `collectTokens(sections)` walks BOTH `section.css` slots AND each HTML
  section's parsed `content.ast` node `css` strings (agent classes live on AST
  nodes — today's `liveCSS` reads only `section.css`, so it would miss them).
- `async update()`: `const { css } = await uno.generate(collectTokens(...))`
  (uno caches matched tokens internally → cheap on repeat). `generate()` is
  async, so guard against out-of-order resolves (sequence counter, latest wins).
- Builder (`EcommerceBuilder.svelte`) keeps its `$effect` → `update()` and the
  `<svelte:head>` `<style>` injection; only `update()` becomes async. No
  MutationObserver, no global init.
- Verify WYSIWYG edits (EditorTab textareas → `editorStore.updateCss`) and AST
  text/class edits both re-trigger `update()`.

## Phase 3 — Wire into the published storefront (DEFERRED)
NOTE: `ecommerce/routes/+page.svelte` is currently a static demo — nothing
renders saved `SectionData` on the public storefront yet. So there is no path to
wire today. `collectTokens` + `generateCss` are exported and ready; drop them in
when the storefront actually renders sections.

Future wiring: the storefront renders saved sections but has no editor `$effect`. Generate once
on mount from the loaded sections' tokens and inject the `<style>`:
- In `ecommerce/routes/+layout.svelte` (or the renderer root), on mount call the
  same `collectTokens` + `uno.generate` and inject the result.
- **FOUC**: generation is client-side, so dynamic classes are unstyled until the
  first `generate()` resolves. Mitigations to evaluate:
  - Accept a brief flash for now (user wants live), OR
  - Precompile the stored HTML's tokens to a static `<style>` at publish/save
    time (best-of-both: live editor, static visitor CSS — same `generate()` run
    server-side or at save and persisted).
  - Document whichever we pick; do not silently ship FOUC.
- Two engines coexist on the storefront (build-time Tailwind for static UI like
  `FloatingCart`, UnoCSS for dynamic sections). No preflight clash since the
  generator emits utilities only (Phase 1).

## Phase 4 — Delete the micro-compiler
- Remove `ecommerce/stores/live-css.svelte.ts`.
- Update the stale comment in `ecommerce/html-ast/style-attributes.ts` (lines
  6-8) that references the micro-compiler.

## Phase 5 (optional, follow-up) — fold colors into theme utilities
Once `bg-color-N` / `text-color-N` compile via the runtime + build theme:
- Drop `compileStyleAttributes` (`style-attributes.ts`) and the inline-`style`
  color branch in `AstRenderer.svelte`, so color becomes plain Tailwind like
  every other utility — one render path instead of two.
- This is independent of the live-compiler swap; do it after Phases 1-4 verify.

## Verification checklist
- [x] `p-4` renders 4px (`calc(var(--spacing) * 4)`, `--spacing: 1px`). Verified.
- [x] `md:` breaks at 749px (theme breakpoints applied). Verified.
- [x] `bg-color-9` etc. emit `var(--color-N)` -> resolve to the active palette;
      palette swap updates live (no regen needed). Verified.
- [x] Arbitrary classes (flex, gap, rounded, hover:, `w-[200px]`) compile.
      Verified in spike.
- [x] Theme `:root` vars (`--text-4xl-fontSize`, `--container-Nxl`,
      `--fontWeight-bold`) emitted so text/width/weight utilities resolve;
      reset preflight NOT emitted (no clash). Verified.
- [x] Agent HTML classes on AST nodes (not just `section.css`) are collected
      (`collectTokens` walks `content.ast`). Verified.
- [x] `bunx svelte-check` clean for the new/changed files; `parse-html.test.ts`
      still green (17/17). Verified.
- [ ] WYSIWYG edits + AST edits re-trigger `update()` in the running app
      (manual smoke test in the builder — pending).
- [ ] Storefront FOUC behavior — N/A until the storefront renders sections
      (Phase 3 deferred).

## Risks / decisions to confirm before coding
1. **Preset spacing fidelity** — can `presetWind4` reproduce `--spacing: 1px`
   dynamic scale? If not, fall back to `presetWind3` with an explicit spacing
   map, or generate the spacing theme programmatically. (Blocking — verify first.)
2. **Utility parity** — UnoCSS ≈ Tailwind but not byte-identical; spot-check the
   utilities agents actually emit, plus anything from `@tailwindcss/typography`
   if agent content uses `prose`.
3. **Storefront FOUC** — pick and document a mitigation.
4. **Async ordering** — `generate()` is async; fast successive edits must not
   inject a stale result (latest-wins guard).
```
```
