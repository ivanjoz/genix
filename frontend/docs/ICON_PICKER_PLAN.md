# Icon / Emoji / Color-Icon Picker for TextBlockEditor

Add three picker tools to the WYSIWYG `TextBlockEditor` toolbar:

1. **Icons** — `@iconify-json/mdi` (7,638 monochrome icons, `currentColor`).
2. **Emojis** — `@iconify-json/emojione` (full-color emoji).
3. **Color icons** — `@iconify-json/flat-color-icons` (multicolor flat icons, ~330).

Picking an icon inserts it inline into the focused text line, rendered as **inline SVG** so it
works offline in both the builder preview and the storefront prerender (no Iconify API at runtime).

## Why inline SVG (not Tailwind `icon-[set--name]` classes)

The app renders icons via `@iconify/tailwind4`, which only emits CSS for icon classes it sees
**statically in source at build time**. A picker chooses icons **at runtime**, so those classes
would never be generated. We therefore store the icon's SVG **body** in the AST node and render it
with an `Icon` component. This keeps each saved page self-contained (no runtime dependency on the
icon JSON or the Iconify API).

## Data: how the JSON is obtained

- `@iconify-json/mdi` is already a devDependency. Add the two missing sets:
  - `bun add -d @iconify-json/emojione @iconify-json/flat-color-icons`
- The picker **lazy-loads** a set's JSON via dynamic import only when that tool is first opened:
  `await import('@iconify-json/mdi/icons.json')`. Vite splits each into its own async chunk, so the
  ~3 MB sets stay out of the eager bundle and are browser-cached after first open.
- No copying into `static/` and no prebuild step — dynamic import is enough.

## Files

### New — `frontend/webpage/components/Icon.svelte` (shared, in `$ecommerce`)
Renders an icon AST node. Props: `body` (SVG inner markup), `vb` (viewBox, default `0 0 24 24`),
plus the standard `css` / `style` the registry passes. Renders:
`<svg viewBox={vb} class={css} style={style} width="1em" height="1em">{@html body}</svg>`.
- `mdi` bodies use `currentColor` → inherit the fragment's text color (existing color tool works).
- `emojione` / `flat-color-icons` carry their own fills → render multicolor as-is.
- `{@html body}` is safe here: bodies come from the vendored Iconify packages, not user input.

### Edit — `frontend/webpage/html-ast/component-registry.ts`
Register `Icon` in `astComponentRegistry`. The existing `AstRenderer` custom-component path then
renders icon nodes automatically (passes `css`/`style`, spreads `props.body`/`props.vb`). **No
change to `AstRenderer.svelte`.**

### New — `frontend/routes/webpage-builder/components/IconPicker.svelte`
The picker panel shown inside the toolbar's `openTool` options row:
- Props: `set: 'mdi' | 'emojione' | 'flat-color-icons'`, `onpick: (icon) => void`.
- Lazy-loads the set JSON on mount; builds `{ name, body }[]` (+ resolves `aliases`).
- Search input (filters by name) + a virtualized grid (reuse `@humanspeak/svelte-virtual-list`,
  already a dependency) so 7k icons scroll smoothly.
- Each cell renders the icon (`<svg>…{@html body}`); click calls `onpick({ body, vb })` where
  `vb = '0 0 {set.width ?? 24} {set.height ?? 24}'`.

### Edit — `frontend/routes/webpage-builder/components/TextBlockEditor.svelte`
- Add three entries to the text-mode `TOOLS` array: `icons`, `emojis`, `colorIcons`, with compact
  inline-SVG glyphs in `ICONS` (consistent with the existing tool glyphs). These appear next to the
  color tools (the spot the screenshot arrows point at).
- When one is the `openTool`, render `<IconPicker>` in the options row instead of sliders/swatches.
- **Insertion** (`insertIcon`): insert an icon fragment into the focused line. An icon fragment is
  the AST node `{ tagName: 'Icon', props: { body, vb } }`, spliced into `line.children` right after
  the focused fragment (materializing a leaf line first, mirroring `splitFragment`). Re-read the
  proxied node back out of the array before any focus use (the `$state` proxy-identity gotcha noted
  in the file header).
- **Chip rendering** (`#each fragmentsOf(line)`): treat `frag.tagName === 'Icon'` as a read-only
  icon chip and render its SVG (`{@html frag.props.body}`) instead of a textarea.
- **`isIconFragment`**: return `true` when `frag.tagName === 'Icon'` (in addition to the existing
  emoji-text heuristic) so icons stay non-editable and the focus/delete logic skips them correctly.
- **Delete**: Backspace on an empty fragment already removes the previous fragment; confirm an `Icon`
  fragment is deletable that way (it has no editable textarea, so deletion targets it from the
  neighbor — verify and adjust `deleteFragment` if an icon chip needs its own remove affordance,
  e.g. a small × on the chip).

## Decisions (confirmed)

1. **mdi color** — mdi inherits the fragment text color (Text-color tool tints it); emoji and
   flat-color-icons keep their own fills.
2. **Icon size** — picker preview cells render at **18px**; the inserted `Icon` renders at **1em**,
   so the live render inherits the surrounding text size (Font-size tool controls it).
3. **Deletion** — **Backspace** at the start of the fragment following an icon removes the icon
   (`removePrevIcon`). `insertIcon` always leaves an editable span after the icon so there is a
   caret target to backspace from. No × affordance on the chip.

## Status: implemented

- Added deps `@iconify-json/emojione`, `@iconify-json/flat-color-icons` (mdi already present).
- `webpage/components/Icon.svelte` + registered `Icon` in `webpage/html-ast/component-registry.ts`.
- `routes/webpage-builder/components/icon-sets.ts` (lazy loaders) + `IconPicker.svelte`.
- `TextBlockEditor.svelte`: 3 toolbar tools, picker options-row, `insertIcon`/`removePrevIcon`,
  icon-chip rendering, Backspace deletion.
- `bun run check`: zero errors in the icon-feature files (pre-existing `svgs` template errors are
  unrelated). Sections persist as AST JSON, so `Icon` nodes survive save/reload.

## Validation

- `bun run check` (svelte-check) and `bun run build` (confirms the lazy chunks split and the
  storefront build still renders).
- Manual: open the builder text editor, open each of the 3 tools, search, pick an icon → it appears
  inline in the preview; Save and reload → icon persists and renders.
