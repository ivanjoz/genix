# HTML → AST → Svelte Rendering Plan

Goal: let agents (and humans) author ecommerce sections as **raw HTML with custom component tags**, parse that HTML into a `ComponentAST` tree, and render the tree in Svelte. This coexists with the current Svelte-component section system (`SectionData` + `SectionRegistry`); it does not replace it.

```
HTML string  ──parseHTML()──▶  ComponentAST  ──<AstRenderer>──▶  DOM
(authored by agent)            (typed tree)     (recursive svelte)
```

---

## 1. Why this is needed (current state)

- **System A (active):** `SectionData[]` → `EcommerceRenderer.svelte` / `BuilderSectionRender.svelte` → `SectionRegistry[type].component`. Hand-written `.svelte` sections (`HeroStandard`, `ProductGridSimple`, `FeatureListSimple`), each exporting a `schema`. This keeps working untouched.
- **System B (defined but inert):** `SectionTemplate { HTML, ast: ComponentAST }` in `ecommerce-templates/`. `ComponentAST` exists in `renderer-types.ts` but **nothing renders it** (confirmed: only referenced in its own type file). `SectionTemplate.HTML` is the newly-added, currently-empty field.

This plan implements the missing pieces: the parser, the AST renderer, the component registry, and the style-attribute compiler — then wires an HTML section type into the existing builder.

---

## 2. Decisions (confirmed with user)

1. **Parser dependency: `htmlparser2`** (`bun add htmlparser2`). Reasons: actively maintained, zero-dep, fast (Cheerio's engine), runs isomorphically (browser builder + server/agent validation), and — unlike `parse5`/`DOMParser` — can **preserve tag case** so `<ProductGrid>` is not flattened to `productgrid`.
   - Parser options: `{ lowerCaseTags: false, lowerCaseAttributeNames: false, recognizeSelfClosing: true }`.
2. **Per-component prop schema** drives attribute type coercion and serves as the agent-facing spec.
3. **Coexist** with System A — add a new HTML/AST section type rendered by a recursive `AstRenderer`.
4. **HTML-native style attributes** (`background-color="9"`, `padding-y="80"`, …) compiled to inline `style`. Applies to **both native tags and custom components** (custom components receive them as wrapper styling in addition to their own typed props). `class="..."` raw-Tailwind escape hatch still accepted. Existing `resolveTokens()` stays available.

---

## 3. AST shape

Reuse the existing `ComponentAST` (`renderer/renderer-types.ts`). The parser will populate:

- `tagName` — native (`section`, `div`, `h1`–`h6`, `p`, `a`, `span`, `img`, `button`, `ul`, `li`, …) or custom (`ProductGrid`, `CategoryGrid`, `ProductsByCategory`, `Banner`, …).
- `css` — value of `class` (token syntax preserved for `resolveTokens`).
- `style` — compiled from style attributes (see §6).
- `text` — text content for leaf elements.
- `children` — child `ComponentAST[]`.
- `attributes` — raw passthrough attrs not otherwise consumed (e.g. `href`, `src`, `alt`, `id`, `aria-*`).
- typed component props (`productosIDs`, `categoriasIDs`, `limit`, …) — coerced per schema, stored as named fields (already present on `ComponentAST`/`ComponentProps`), with a catch-all for component-specific props.

No change to the `ComponentAST` interface is expected; if a custom prop has nowhere to live we add a `props?: Record<string, any>` field rather than widening every type.

---

## 4. Files to create / change

```
ecommerce/
├── html-ast/
│   ├── parse-html.ts          # parseHTML(html) -> ComponentAST | ComponentAST[]  (htmlparser2)
│   ├── style-attributes.ts    # whitelist + compileStyleAttributes() -> inline style + palette tokens
│   ├── component-registry.ts  # tagName -> { component, schema }  for AST custom components
│   ├── component-schemas.ts   # per-component prop schemas (types + coercion rules)
│   ├── coerce.ts              # string attr -> typed value (number/number[]/bool/enum) per schema
│   └── parse-html.test.ts     # unit tests for parser + coercion + style compiler
│
├── renderer/
│   └── AstRenderer.svelte     # NEW recursive renderer for ComponentAST (native + custom tags)
│
└── renderer-types.ts          # (maybe) add `props?: Record<string, any>` to ComponentAST
```

Integration touch-points (System A coexistence):
- `renderer/section-types.ts` — allow an HTML/AST section: e.g. `SectionData.content.html?: string` or a dedicated `type: 'HtmlSection'` whose `content.ast` holds the parsed tree.
- `templates/registry.ts` is auto-generated — do **not** hand-edit. Instead register the HTML section path via a small wrapper section component (`HtmlSection.svelte`) that runs `parseHTML(content.html)` (memoized) and renders `<AstRenderer>`, OR special-case it in `EcommerceRenderer`/`BuilderSectionRender`. Decide during impl; wrapper-component is cleaner and keeps the registry uniform.

---

## 5. Parser (`parse-html.ts`)

- Use `htmlparser2`'s `Parser`/handler (or `parseDocument` from `domhandler` + walk) with the options above.
- Walk the DOM-handler tree → `ComponentAST`:
  - element → node; `class` → `css`; style attrs → `compileStyleAttributes` → `style`; remaining attrs → coerce via schema (custom comps) or land in `attributes` (native).
  - text nodes → `text` (trimmed; collapse whitespace). Mixed text+element children handled (text becomes a child text node when siblings exist).
  - self-closing custom tags supported (`<ProductGrid ... />`).
- Returns a single root or an array (multiple top-level siblings).
- Pure, isomorphic, no Svelte imports — unit-testable in Bun.

## 6. Style attributes (`style-attributes.ts`) — color only

- Only **color** attributes are compiled to inline `style`, because a bare palette
  index must become a CSS variable. Whitelist: `background-color`, `color`, `border-color`.
- All other styling (spacing, sizing, alignment, etc.) uses **Tailwind classes** and
  is handled by the project's live micro-compiler (`stores/live-css.svelte.ts`) / Tailwind.
- Value resolution:
  - bare integer `1`–`10` → palette token `var(--color-N)` (works with existing `generatePaletteStyles`).
  - anything else (`#fff`, `red`, `rgb(...)`) → passthrough.
- Output: a CSS `style` string merged onto the node. Native tags and custom components both get it.

## 7. Component registry + schemas

- `component-registry.ts`: `Record<tagName, { component: Component, schema: ComponentPropSchema }>`. Seed with existing real components: `ProductsByCategory` (`ecommerce-components/ProductsByCategory.svelte`), `CategoryDescription`, `TextBlock`, and `ProductCard` (`components/ProductCard.svelte`). Add others from `ECOMMERCE_COMPONENTS.md` as they get built (most are still `⬜ Pending`).
- `component-schemas.ts`: per component, declare prop name → type (`number`, `number[]`, `boolean`, `string`, `enum[...]`) + optional default. Used by `coerce.ts`. Note prop-name mapping (e.g. doc uses `categoriasIDs`/`categoriaID`; `ProductsByCategory.svelte` expects `categoryID`, `limit`) — schema records the canonical attribute name and target prop.

## 8. AstRenderer.svelte (recursive)

- Props: `{ node: ComponentAST | ComponentAST[], palette?, values? }`.
- For each node: resolve `css` via existing `resolveTokens(css, variables, values, palette)`; merge compiled `style`.
- If `tagName` is a **custom component** (in registry) → render `<Config.component {...coercedProps} css={resolvedCss} style={style} />`.
- Else render the **native element**. Svelte needs static-ish element names for dynamic tags — use `<svelte:element this={tagName}>` for native tags, passing `class`, `style`, spread `attributes`, then `{node.text}` and recursive `{#each children}` via a `{#snippet}` (Svelte 5, matches the snippet pattern already used in `ui-components/misc/Renderer.svelte` and `BuilderSectionRender`).
- Guard against unknown/unsafe tags (whitelist native tags; unknown non-registered uppercase tag → visible error box, mirroring `EcommerceRenderer`'s unknown-type fallback).

## 9. Builder integration

- Add an `HtmlSection.svelte` template section (exports a `schema`; `content.html: string`) that memoizes `parseHTML(content.html)` and renders `<AstRenderer>`. It flows through the existing generated `SectionRegistry`, drag/drop, selection, and live-Tailwind effects with zero changes to `EcommerceBuilder`.
- Author HTML templates as `SectionTemplate.HTML` strings; the section stores the raw HTML and parses at render (cache the AST keyed by the HTML string to avoid re-parsing each keystroke).

---

## 10. Build order

1. `bun add htmlparser2`.
2. `style-attributes.ts` + `coerce.ts` + `component-schemas.ts` (pure, tested first).
3. `parse-html.ts` + `parse-html.test.ts` (parser correctness incl. custom tags, self-closing, mixed content, token preservation).
4. `component-registry.ts` (seed with the 4 existing components).
5. `AstRenderer.svelte` (recursive native + custom).
6. `HtmlSection.svelte` + section-type wiring; render one converted template end-to-end in `builder-store`.
7. Convert existing `ecommerce-templates/*` `ast` definitions' intent into `HTML` strings (incremental; `ast` stays as fallback).

## 11. Open items to confirm during impl

- Exact native-tag whitelist for `AstRenderer` (security: block `script`, `style`, event-handler attrs).
- Whether `ComponentAST` needs `props?: Record<string, any>` or existing named fields suffice.
- HTML section storage: raw `content.html` (parse at render, memoized) vs. pre-parsed `content.ast`. Leaning raw-HTML-stored for agent round-trips.
