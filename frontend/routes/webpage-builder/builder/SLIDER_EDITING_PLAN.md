# Recursive AST Editor + Multi-Child (Slider) Editing — Plan

## Goal

Replace the flat, role-gated editor (`collectRoleNodes` → fixed list of inputs) with a
**recursive editor that mirrors `AstRenderer`**: one AST, two walks — `AstRenderer` paints
it, a new `AstEditor` edits it.

Two behaviours come out of this:

1. **Editable-by-default.** Any node carrying text is an editable input. No `data-role`
   required. `data-role` becomes an optional *label/marker*, not a gate.
2. **Multi-child containers (Slider).** A component whose children are "slides" renders an
   `OptionsStrip` (Slide 1…N); only the selected slide's subtree is shown in the editor,
   and the **live preview slider jumps to the same slide**.

## Confirmed decisions

1. **Editability:** editable by default; `data-role` is a label/marker only. An explicit
   opt-out marker (`data-noedit`) suppresses a node and its text from the editor.
2. **Active-slide state:** lives **local to each recursive `AstEditor` instance** (per Slider
   node) — never written onto the AST (would pollute/serialize the model). Nested sliders
   each get their own state for free.
3. **Preview sync:** picking "Slide N" in the strip moves the live `EcommerceSlider` to slide
   N (and pauses autoplay while editing). Shared via a small reactive store keyed by the
   slide-set identity (see below).
4. **Mixed content:** each `#text` run inside an element is its **own** input.

## Key facts the design relies on (verified in code)

- `parse-html.ts`: a **leaf element with only text** stores it in `node.text`; a
  **mixed-content element** stores text as `#text` child nodes (`TEXT_TAG`) among element
  children.
- `TextBlockEditor` already binds to a node and edits `.text` → works on *any* node with
  text (leaf or a `#text` child).
- `AstRenderer` already passes custom components `childNodes` (= `node.children`) and a
  `renderChild` snippet. `EcommerceSlider` receives `childNodes` (the slide-set array).
- The slide-ness of `<Slider>` is **intrinsic**: every direct child is one slide. The editor
  does not "search" for a container — it recurses and, on reaching a node whose component is
  flagged children-as-slides, treats `node.children` as slides.

## Editability rules (per node, in `AstEditor`)

| Node shape | Control |
|---|---|
| has `data-noedit` | skip node + subtree |
| `node.text` non-empty (leaf text) | `TextBlockEditor` bound to `node` |
| has `#text` children (mixed) | one `TextBlockEditor` per `#text` child (decision 4) |
| `ImageEffect` / `img` | `ImageBlockEditor` |
| `a` (anchor) | text input(s) + `href` input |
| component flagged `childrenAs: 'slides'` (Slider) | `OptionsStrip` over `node.children`; recurse **only** into the selected child |
| category-bound component (`ProductsByCategory`/`CategoryDescription`) | existing category `SearchSelect` (unchanged) |
| plain container (`div`/`section`/native) | no own input; recurse into children, optional nesting group/label |

Label for a control = `node.role` if present, else a humanized tag name (`h1` → "Heading",
`a` → "Link", etc.).

## The one declarative bit (Strategy B's single good idea)

Add to the Slider entry in `component-schemas.ts`:

```ts
Slider: {
  ...,
  childrenAs: 'slides', // editor renders children as navigable slides, not inline
}
```

`AstEditor` reads this from the schema/registry to decide slide vs. inline rendering. No
hardcoded tag set in the editor.

## Preview ↔ editor sync (decision 3)

New tiny store `frontend/ecommerce/stores/slide-sync.svelte.ts`:

```ts
import { SvelteMap } from 'svelte/reactivity';
// keyed by the slide-set array identity (node.children) — the SAME array reference is
// held by AstEditor (node.children) and passed to EcommerceSlider (childNodes), so both
// agree without threading props through AstRenderer.
const active = new SvelteMap<unknown, number>();
export const slideSync = {
  get: (key: unknown) => active.get(key) ?? 0,
  set: (key: unknown, i: number) => active.set(key, i),
  has: (key: unknown) => active.has(key),
};
```

- **`AstEditor`** (Slider node): `OptionsStrip` `onSelect` → `slideSync.set(node.children, i)`;
  its own `activeSlide` derives from `slideSync.get(node.children)`.
- **`EcommerceSlider`**: accept a `builderMode` flag (passed by `AstRenderer` only inside the
  builder). When set, `current` reads/writes `slideSync` keyed by `childNodes` and autoplay is
  disabled; in production it keeps today's internal `$state` (no store growth, no behaviour
  change).

## Files

**New**
- `frontend/ecommerce/builder-edit/AstEditor.svelte` — recursive editor (mirrors AstRenderer).
  Renders itself for children; handles text/image/link/slider/container cases above.
- `frontend/ecommerce/stores/slide-sync.svelte.ts` — shared active-slide index.

**Modified**
- `frontend/ecommerce/html-ast/editable.ts` — add helpers: `getEditableText(node)` (leaf vs
  `#text` runs), `isTextNode`, `isNoEdit`, `humanizeLabel`, `childrenAsSlides(node)` (schema
  lookup). Keep `collectCategoryNodes` etc.
- `frontend/ecommerce/html-ast/component-schemas.ts` — add `childrenAs: 'slides'` to `Slider`
  (and the optional `childrenAs` field to the schema type).
- `frontend/routes/tienda/builder-store/components/EditorTab.svelte` — replace the flat
  `roleNodes` block with `<AstEditor nodes={section.ast} {palette} />`. Keep the category
  selector and the styling group as-is.
- `frontend/ecommerce/components/EcommerceSlider.svelte` — add `builderMode` prop; when set,
  drive `current` from `slideSync` (keyed by `childNodes`) and disable autoplay.
- `frontend/ecommerce/renderer/AstRenderer.svelte` — pass `builderMode` down to custom
  components when rendering inside the builder (thread a `builder` prop / context).

## Open / lower-priority

- Nested sliders: handled by recursion + per-instance state; just verify keying by
  `node.children` stays unique.
- Plain-container labeling/grouping depth: start flat (recurse, no heading) unless the UI
  gets confusing.
- `data-noedit` parsing: add to `parse-html.ts` consumed attributes (like `data-role`) so it
  doesn't leak to the DOM, or read it from `attributes` in the editor only. Decide at impl.

## Verification

- `npm run check` filtered to touched files (baseline: only unrelated `logistica` errors).
- Manual: load `Slider (HTML)` template → editor shows Slide 1/2/3 strip; Slide 1 exposes
  image+title+content+button (now auto, no roles needed elsewhere); switching slides moves the
  live preview; edits mutate the live section; autoplay paused while editing.
```

