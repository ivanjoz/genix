# How to Create a Section Template

A practical, verified guide for authoring webpage-builder section templates in this repo.
This reflects what the renderer **actually** supports today. Where it disagrees with the
older `ECOMMERCE_SECTIONS.md`, trust this file (that doc describes an aspirational API).

---

## 1. What a template is

A template is a single `SectionData` object exported from one `.ts` file. You author the
section as an **HTML string**; when a user adds it, the builder parses that HTML once into
an editable AST (`parseHTML` → `ComponentAST[]`) which the renderer/editor/CSS all read.
**Never hand-author the AST** — only write the `html` string.

```typescript
import type { SectionData } from '$ecommerce/renderer/section-types';

export const MySection: SectionData = {
    id: 'my-section-v1',          // unique, kebab-case, version suffix
    name: 'My Section',           // shown in the picker
    category: 'features',         // see categories below
    description: 'One line describing what it does.',
    thumbnail: `...`,             // optional miniature HTML for the picker card (220×100)
    html: `...`,                  // the real section markup (parsed to AST on add)
};
```

Categories: `hero | products | categories | testimonials | features | cta | footer | header | gallery | text`.

File location: drop it in the matching subfolder (`hero/`, `products/`, `features/`, …),
then register it in `templates/index.ts` (import + add to the `sectionTemplates` array and
the re-export block).

---

## 2. The two hard rules

### 2a. Only these custom component tags exist
Source of truth: `webpage/html-ast/component-registry.ts`. Anything else custom renders a
red **"Unknown tag"** error box.

| Tag | Purpose | Useful attributes |
|-----|---------|-------------------|
| `ProductGrid` | Self-fetching product grid | `categoryID`, `maxWidth`, `maxMargin`, `rows`, `rowsMobile` |
| `ProductsByCategory` | Products of a category (grid) | `categoryID`, `rows`, `rowsMobile`, `maxWidth`, `maxMargin` |
| `ProductCard` | Single product card | `productIDs` |
| `CategoryDescription` | Renders a category's description text | `categoryIDs="[22]"`, `class` |
| `ImageEffect` | Photo with effects/clip/overlay | see §4 |
| `Slider` | Carousel — **each direct child = one slide** | `autoplay`, `interval="6000"` |
| `TabbedLayer` | Tabs — **each direct child = one panel** | `options="Tab A\|Tab B\|Tab C"` |

> These self-fetch live catalog data; in the isolated `template-preview` route they may
> render empty/loading. That's expected — verify their **structure** (heading + layout).

### 2b. Only native HTML tags below are allowed
`div, section, article, aside, header, footer, main, nav, span, p, a, button, img, ul, ol,
li, figure, figcaption, h1–h6, strong, em, small, br, hr, table, thead, tbody, tr, td, th, label`.

**There is no `<svg>`.** For icons use: emoji glyphs as text (`<span>✓</span>`), CSS-drawn
shapes (rounded `div`s, borders), or the photo assets. `<input>` is a void tag but is **not**
in the native list — render form fields as styled `div`/`span` mockups, not real inputs.

---

## 3. Theming, colors, and styling

- **Palette colors:** `color="N"` and `background-color="N"` with `N` = 1..10 map to the
  store's palette CSS variables. Default palette runs **light→dark**: `1=#f8fafc` (lightest)
  … `10=#0f172a` (darkest). So dark section bg = `background-color="9"` with `color="1"` text.
- **Raw colors** also work: `color="#ffffff"`, `background-color="#0f172a"`.
- **Layout & typography:** Tailwind v4 utility classes in `class="..."`.
  - Gradients use the v4 names: `bg-linear-to-r from-... to-...` (`bg-gradient-to-r` also works via compat).
  - Tailwind `--spacing` is **1px** in this project, so `h-4` = 4px. Use larger steps than you'd expect (`py-16`, `gap-8`).
- **Editable nodes:** add `data-role="title" | "content" | "button" | "image"` so the
  builder exposes the right inline editor for that node. Put one `title`, one `content`,
  one `button` per logical block; `image` goes on the `ImageEffect`.
- Prefer semantic structure (`section` > `div` wrapper > content) and a centered
  `max-w-*` `mx-auto` container. Make it responsive mobile-first (`text-2xl md:text-4xl`,
  `grid-cols-1 md:grid-cols-3`).

---

## 4. ImageEffect cheatsheet

Photo component. Two modes:

**Fill mode** (full-bleed background, text overlaid as siblings):
```html
<section class="relative overflow-hidden min-h-[420px] flex items-center px-6 py-12">
    <ImageEffect data-role="image" fill fit="cover"
        src="https://ivanjoz.github.io/genix-assets/images/business-workspace/5.avif"
        effect="overlay" tint="#0f172a" intensity="0.9" />
    <div class="relative z-10 ...">…overlaid text…</div>
</section>
```

**Box mode** (photo in a clipped/aspect box, children render *inside* it):
```html
<ImageEffect data-role="image"
    src=".../6.avif" layout="curve-right" effect="duotone-matrix"
    tint="#4c2d82" tint2="#f59e0b" intensity="1" aspectRatio="21/9"
    class="rounded-2xl overflow-hidden min-h-[420px] flex items-center p-12">
    <div class="max-w-sm">…text on the tinted side…</div>
</ImageEffect>
```

- `layout`: `fade-{left,right,up,down}`, `slash-{left,right}`, `curve-{left,right}`, `curve-convex-{left,right}`
- `effect`: `duotone`, `duotone-matrix` (uses `tint`+`tint2`), `glass`, `vignette`, `overlay`
- `fit`: `cover` (default), `contain`, `contain-left`, `contain-right` (ignored by slash/curve)
- `intensity`: 0..1 (overlay/tint strength) · `aspectRatio`: e.g. `"21/9"` · `fill`: boolean flag

**Available photos (the only ones that exist):**
`https://ivanjoz.github.io/genix-assets/images/business-workspace/{4,5,6,7,8,9}.avif`

---

## 5. Slider & TabbedLayer

Each **direct child** of `<Slider>`/`<TabbedLayer>` is one slide/panel. A child can itself
be a full subtree (e.g. an `ImageEffect` fill hero). `TabbedLayer` needs an `options`
attribute with `|`-separated labels matching the child count.

```html
<Slider autoplay interval="6000">
    <div class="...slide 1...">…</div>
    <div class="...slide 2...">…</div>
</Slider>
```

---

## 6. Thumbnails

`thumbnail` is trusted miniature HTML shown in the picker, constrained to a **220×100** box
(`overflow:hidden`, flex column, centered). Hand-build a tiny static mock of the section —
use very small text sizes (`text-[7px]`), `bg-slate-*` blocks for images, and matching
colors. It does **not** parse to AST; it's rendered via `{@html}`, so plain Tailwind +
inline styles only (no custom component tags). Keep it lightweight and representative.

---

## 7. Verify with headless Chrome

The preview route renders a template both as its **live component** and its **thumbnail**
on a chrome-less page, with stable hooks:

- URL: `http://localhost:3570/webpage-builder/template-preview?id=<template-id>`
  (`bun run dev:main` serves the main app on port 3570).
- Wait for `main[data-render-state="ready"]` before screenshotting.
- Targets: `#template-live` (full render) and `#template-thumbnail` (picker mock).

Iterate: screenshot → eyeball → fix the `html`/`thumbnail` → reload → repeat until clean.

---

## 8. Checklist before done

- [ ] `id` unique + kebab-case + version suffix; file in the right subfolder.
- [ ] Registered in `templates/index.ts` (import, array, re-export).
- [ ] Only registered custom tags + allowed native tags (no `<svg>`, no `<input>`).
- [ ] Editable `data-role` attributes on title/content/button/image.
- [ ] Palette `color`/`background-color` numbers used for theme-able colors.
- [ ] Responsive (mobile-first breakpoints) and uses a centered `max-w-*` container.
- [ ] Thumbnail provided and representative.
- [ ] Screenshotted via the preview route — live + thumbnail both look professional.
