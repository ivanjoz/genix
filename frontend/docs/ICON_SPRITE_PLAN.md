# Deduplicate picked icons into `SectionData.Svgs` (SVG sprite + `<use>`)

## Goal
Stop inlining each picked icon's SVG body on its `Icon` AST node. Instead store each
unique body **once** in the section's `Svgs` map and have every `Icon` node **reference**
it by id. Reference mechanism = SVG sprite: one `<symbol id="…">` per map entry, each
occurrence is `<svg viewBox=…><use href="#id"/></svg>`.

## Why `<use href="#id">`, not a CSS class
- A CSS class can only point at an SVG via `background-image: url(data-uri)` — loses
  `currentColor` inheritance (mdi tinting) and doesn't flow inline as a child.
- `<symbol>` + `<use>` is the native dedupe primitive. `<use>` resolves the id across the
  **whole document**, so an `Icon` node deep in the AST needs only the id — no need to thread
  the `Svgs` map down to it. `currentColor` resolves at the `<use>` site (mdi tint preserved).
- Per-icon `viewBox` stays on the referencing `<svg viewBox={vb}>`, so mixed sets
  (mdi 24, emojione 64, fc 48) render correctly. `<symbol>` is left bare (no own viewBox).

## Id scheme
`icon--<set>-<name>`, sanitized to id-safe chars, e.g. `icon--mdi-account`,
`icon--flat-color-icons-businessman`, `icon--emojione-grinning-face`. Set prefix avoids
cross-set collisions (mdi & flat-color-icons both have `home`).

## Persistence (REQUIRED — references break on reload otherwise)
The body lives only in `Svgs` now, so the map must persist.
- **Frontend** `section-types.ts`: rename `svgs` → `Svgs` (PascalCase, persisted), make it
  optional `Svgs?: { [id: string]: string }`. (Also fixes the pre-existing svelte-check
  errors from the required-but-unset `svgs`.)
- **Backend** `backend/webpage/types/page_content.go` `SectionContent`: add
  `Svgs map[string]string `json:",omitempty" cbor:"7,keyasint,omitempty"`` (next free cbor key).
- `sectionHash` already keeps it (it's in the `...persisted` spread), so dirty-tracking
  and save already cover it once the field is PascalCase.

## File changes
1. **`routes/webpage-builder/components/icon-sets.ts`** — add `id` to `PickerIcon`
   (`icon--<set>-<name>`); tag each icon with its source set when flattening so the id can
   be built (loaders currently lose which set an icon came from).
2. **`routes/webpage-builder/components/IconPicker.svelte`** — `onpick` passes
   `{ id, body, vb }` instead of `{ body, vb }`.
3. **`webpage/components/Icon.svelte`** — accept `svg` (id) + keep `body`/`vb`. Render
   `<svg viewBox={vb}>{#if svg}<use href={'#'+svg}/>{:else}{@html body}{/if}</svg>`.
   (`body` fallback keeps old inline nodes working + lets the editor chip render standalone.)
4. **New `webpage/renderer/IconSprite.svelte`** — `{#each Object.entries(svgs)}` →
   one hidden `<svg aria-hidden style="position:absolute;width:0;height:0"><symbol {id}>{@html body}</symbol></svg>`.
5. **`webpage/renderer/section-types.ts`** — `SectionProps` gains `svgs?`; rename field as above.
6. **`webpage/renderer/HtmlSection.svelte`** — render `<IconSprite svgs={svgs} />` once at the
   section root (before the AST). Pass `svgs` from `SectionProps`.
7. **`webpage/renderer/EcommerceRenderer.svelte`** — pass `svgs={element.Svgs}` into the section.
8. **`routes/webpage-builder/components/TextBlockEditor.svelte`** —
   - `insertIcon({ id, body, vb })`: `(editorStore.selectedSection.Svgs ??= {})[id] = body;`
     node = `{ tagName:'Icon', props:{ svg:id, vb } }`. Import `editorStore`.
   - Chip preview: render `<Icon svg={frag.props?.svg} body={frag.props?.body}
     vb={frag.props?.vb} />`. The live canvas sprite is in the same document, so `<use>`
     resolves; pass `body` too as a belt-and-suspenders fallback (look up from `Svgs[id]`).
   - Backspace/delete logic unchanged (operates on node identity, not body).

## Not doing (note as follow-ups)
- Page-level sprite dedupe across sections (identical ids across sections are harmless —
  `<use>` resolves to the first; duplicate-id HTML is technically invalid but renders fine).
- Garbage-collecting `Svgs` entries when the last referencing node is deleted (low value;
  a save-time sweep could prune unreferenced ids later).

## Status: implemented
- Frontend `Svgs?` field + `SectionProps.svgs`; backend `SectionContent.Svgs` (cbor key 7).
- `iconSpriteId()` + `PickerIcon.id`; IconPicker `onpick({ id, body, vb })`.
- `Icon.svelte` renders `<use href="#id">` (svg) with `body` inline fallback.
- New `IconSprite.svelte`; mounted once per section in `HtmlSection`; `svgs` threaded from
  `EcommerceRenderer`.
- `TextBlockEditor.insertIcon` writes body to `editorStore.selectedSection.Svgs[id]`, node
  stores `{ svg: id, vb }`; chip previews via `<use>` (+ legacy `body` fallback).
- `go build ./webpage/...` clean; `bun run check` clean on all touched files (the old
  required-`svgs` template errors are resolved by making the field optional).

## Validation
- `bun run check` (svelte-check) clean on touched files.
- Backend: `go build ./...` for the webpage types.
- Manual: pick an icon → appears in canvas via `<use>`; pick the SAME icon twice → `Svgs`
  has ONE entry, two nodes reference it; Save + reload → icons persist and render.
