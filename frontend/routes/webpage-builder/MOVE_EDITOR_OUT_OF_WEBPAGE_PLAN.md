# Plan: extract editor-only code from `webpage/` into `routes/webpage-builder/`

## Goal
`frontend/webpage/` is the standalone **runtime renderer** (storefront + prerender) and
must stay slim. `frontend/routes/webpage-builder/` is the **real editor**. Today some
editor-only modules live inside `webpage/` and the builder reaches into them via the
`$ecommerce` alias. Move every editor-only module into the builder so the prerender
bundle never sees authoring code.

## Verified dependency facts
- Render chain is clean: `renderer/AstRenderer.svelte` → `html-ast/parse-html` →
  `{coerce, component-schemas, style-attributes}`; none import `editor` or `live-css`.
  (Earlier "render imports editor" hits were comment text, not imports.)
- Editor coupling is one-directional: editor files import render modules, never vice-versa.

## Files to MOVE  (from `webpage/` → `routes/webpage-builder/`)
| Source | Destination | Consumers today |
|---|---|---|
| `webpage/templates/` (whole dir) | `routes/webpage-builder/templates/` | `editor.svelte.ts` (moving), builder `TemplatesTab.svelte`, `store-example.ts`, test |
| `webpage/stores/editor.svelte.ts` | `routes/webpage-builder/stores/editor.svelte.ts` | builder (`EcommerceBuilder`, `BuilderSectionRender`, `SectionEditorLayer`, `EditorTab`, `SectionStyleEditor`, `[pageID]/+page`), `live-css` |
| `webpage/stores/live-css.svelte.ts` | `routes/webpage-builder/stores/live-css.svelte.ts` | builder `EcommerceBuilder.svelte` only |
| `webpage/html-ast/editable.ts` | `routes/webpage-builder/html-ast/editable.ts` | builder `EditorTab.svelte`, `AstEditor.svelte`, test |

## Files that STAY in `webpage/` (render path — confirmed slim after move)
- `renderer/*` (all)
- `html-ast/{parse-html, coerce, component-schemas, style-attributes, component-registry}`
- `stores/{uno-generator, slide-sync, globals}`  ← still used by render components
- `components/`, `ecommerce-components/`, `services/`, `lib/`, `routes/`, `static/`, `cloudflare/`

## Import rewrites after the move
1. **Moved files importing render modules** — rewrite relative paths to the `$ecommerce` alias
   (works from the main app):
   - `editor.svelte.ts`: `../renderer/section-types` → `$ecommerce/renderer/section-types`;
     `../renderer/HtmlSection.svelte` → `$ecommerce/renderer/HtmlSection.svelte`;
     `../html-ast/parse-html` → `$ecommerce/html-ast/parse-html`;
     `../templates` → `../templates` (moves with it, stays relative).
   - `live-css.svelte.ts`: `./editor.svelte` → `./editor.svelte` (both move together);
     `./uno-generator` → `$ecommerce/stores/uno-generator`.
   - `editable.ts`: `./coerce`, `./component-schemas`, `./parse-html` → `$ecommerce/html-ast/*`.
2. **Builder files** currently using `$ecommerce/{templates,stores/editor,stores/live-css,html-ast/editable}`
   — repoint to the new in-builder location (relative or a builder-local `$lib` path).
   Affected: `TemplatesTab.svelte`, `store-example.ts`, `EcommerceBuilder.svelte`,
   `BuilderSectionRender.svelte`, `SectionEditorLayer.svelte`, `EditorTab.svelte`,
   `SectionStyleEditor.svelte`, `AstEditor.svelte`, `[pageID]/+page.svelte`.

## Open questions (need your call)
1. **`webpage/html-ast/parse-html.test.ts`** imports both render modules (`parse-html`,
   `coerce`, `component-schemas`, `style-attributes`) AND moving modules (`templates`,
   `editable`). If it stays in `webpage/` it would import *up* into the builder (bad
   direction). Options:
   - (a) Move the whole test into the builder and import render modules via `$ecommerce`.
   - (b) Split it: keep pure parse-html assertions in `webpage/`, move template/editable
     cases into the builder.
   **Which do you want?**
2. **`routes/webpage-builder/builder/store-example.ts`** — name suggests scratch/demo code.
   Is it live? If dead, delete it instead of rewriting its imports (pre-alpha, no back-compat).
3. **Destination layout** — I propose mirroring the source subfolders inside the builder
   (`templates/`, `stores/`, `html-ast/`). OK, or do you prefer flattening into the existing
   `builder/` / `components/` folders?

## Execution order (once approved)
1. `git mv` the four sources to their destinations.
2. Rewrite imports inside moved files (table §1).
3. Repoint builder consumers (table §2).
4. Handle the test + `store-example.ts` per answers above.
5. Typecheck / build both apps; confirm prerender bundle no longer references editor code.
