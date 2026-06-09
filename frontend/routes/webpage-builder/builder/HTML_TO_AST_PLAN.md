# Plan: HTML templates → AST at add/load time (AST as the single source of truth)

## Goal

Make the **AST the canonical, persisted model** for HTML sections. HTML is only an
*authoring source* for templates: it is parsed to a `ComponentAST[]` **once**, at the
moment a plantilla is added (or a page is loaded), stored as `content.ast`, and from then
on every consumer — rendering, the WYSIWYG editor, runtime CSS collection, and save —
operates purely on the AST. The raw `content.html` is dropped after conversion.

This also wires the real HTML plantillas (`ecommerce-templates/templates/*`) into the
`TemplatesTab`, which today only lists the 4 registry *component* types and never exposes
the HTML banners.

## Current state (why save printed `html`)

- HTML sections are seeded as `content: { html: <string> }` — no `ast`
  (`store-example.ts`).
- The AST is produced **lazily and transiently** in two places:
  1. `HtmlSection.svelte:27` — `content.ast ?? parseHTML(content.html)` at render, never
     persisted.
  2. `EditorTab.svelte:23-27` — an `$effect` parses `html → ast` and persists it, but only
     when the section is *selected for editing*, and keeps `content.html` around.
- `collectTokens` (uno-generator) only reads `content.ast`, so HTML sections that were
  never selected contribute no tokens to runtime CSS.
- Result: `editorStore.sections` (what save dumps) holds `html`, not `ast`.

## Target flow

```
template HTML string ──parseHTML()──▶ content.ast  (stored)   ──▶ render / edit / CSS / save
        (authoring source, discarded after parse)
```

`parseHTML` runs at exactly two boundaries:
- **add** — when a plantilla is dropped/clicked in `TemplatesTab`.
- **load** — when sections enter the store (seed / future backend load).

Everywhere downstream assumes `content.ast` is present.

## Changes

### 1. `ecommerce/stores/editor.svelte.ts` — `addSection` handles HTML templates
- Import `sectionTemplates` (`$ecommerce/ecommerce-templates/templates`) and `parseHTML`.
- Build an id→template lookup once at module load.
- In `addSection(id, index)`: if `id` matches an HTML plantilla, create an `HtmlSection`:
  ```ts
  {
    id: crypto.randomUUID(),
    type: 'HtmlSection',
    category: template.category,
    content: { ast: parseHTML(template.HTML) },   // no `html`
    css: {}
  }
  ```
  Otherwise fall back to the existing registry-component path (unchanged).

### 2. `routes/tienda/builder-store/components/TemplatesTab.svelte` — list HTML plantillas
- Replace the `SectionList` (registry component types) source with `sectionTemplates`
  (`id`, `name`, `category`, `description`).
- `onSelect` and the drag payload still carry `{ id }`; `addSection` resolves it.
- Note / decision: pure component-only sections (HeroStandard, FeatureListSimple,
  ProductGridSimple) become unreachable from the tab. They remain registered for rendering;
  product/category functionality is reached through HTML templates that embed custom
  component tags (e.g. `<ProductsByCategory>`). Per the pre-alpha "no backwards-compat"
  rule we list the HTML plantillas only.

### 3. Load normalization — single boundary in `EcommerceBuilder.svelte`
- In the seed `$effect` that does `editorStore.sections = [...elements]`, normalize first:
  for any `HtmlSection` with `content.html` and no `content.ast`, set
  `content.ast = parseHTML(content.html)` and delete `content.html`.
- This guarantees the AST invariant for both `store-example` and any future backend load,
  while keeping `store-example.ts` authored in readable HTML.

### 4. `ecommerce/templates/html/HtmlSection.svelte` — render AST directly
- Drop the `parseHTML` fallback; render `content.ast` (guaranteed present):
  ```ts
  const ast = $derived(content.ast ?? []);
  ```
  (Empty guard only — no parsing in the render path.)
- Update `schema.content` from `['html']` → `[]` (editing is role-driven via the AST, not a
  flat field; `EditorTab` already branches on `isHtmlSection`).

### 5. `routes/tienda/builder-store/components/EditorTab.svelte` — remove lazy parse
- Delete the `$effect` (lines 22-27) and the now-unused `parseHTML` import. AST is
  guaranteed by the add/load boundaries.

### 6. `SectionEditorLayer.svelte` — save
- `handleSave` already prints `editorStore.sections` + generated CSS; with the above it now
  emits real AST instead of `html`. No change needed (keep the console output).

## Out of scope
- Persisting to the backend (save still just prints).
- Any change to `parse-html.ts` itself.

## Verification
- Add each plantilla from the Plantillas tab → renders correctly.
- Click GUARDAR → console `=== PAGE AST ===` shows `content.ast` (no `html`) for every
  HTML section, and `=== TAILWIND CSS ===` includes classes from never-selected sections.
- Editing role nodes (title/content/button/image) still works.
- `npm run check` (svelte-check) passes.
