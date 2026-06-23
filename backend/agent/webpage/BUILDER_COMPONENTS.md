# Builder Agent — Custom Components Reference

> The complete set of custom HTML component tags the webpage builder
> understands. Companion to **`BUILDER_INSTRUCTIONS.md`**. Source of truth:
> `webpage/html-ast/component-schemas.ts` (props/coercion) and
> `webpage/html-ast/component-registry.ts` (the tags that exist).

**Rules that apply to every component below:**

- These are the **only** custom tags. Any other capitalized tag renders a red
  "Unknown tag" error box.
- They are written as normal HTML tags with attributes:
  `<ProductGrid categoryID="22" rows="2" />`.
- Each is **self-fetching** — it pulls live catalog/category data on its own. In
  isolated previews they may render empty/loading; that is expected. Verify
  *structure*, not data.
- A `class="..."` attribute styles the component's outer wrapper (margins,
  padding, radius); internal styling is owned by the component.
- Attribute value coercion: `number` (`rows="3"`), `boolean` (presence = true,
  e.g. `autoplay`; or `="false"`), `number[]` (`categoryIDs="[22, 7]"`),
  `string`. Unknown attributes pass through untouched.

---

## ProductGrid

Self-fetching product grid. With `categoryID` it shows one category; without it,
the full catalog. Card count = columns × rows.

| Attribute | Type | Default | Notes |
|-----------|------|---------|-------|
| `categoryID` | number | — | Limit to one category; omit for all products |
| `rows` | number | 3 | Number of rows |
| `rowsMobile` | number | — | Rows on mobile |
| `maxWidth` | number | — | Max grid width |
| `maxMargin` | number | — | Max horizontal margin |

```html
<section class="py-24 px-6">
  <div class="max-w-6xl mx-auto">
    <h2 color="9" class="text-3xl font-bold mb-12 text-center">Our Products</h2>
    <ProductGrid rows="2" rowsMobile="1" />
  </div>
</section>
```

---

## ProductsByCategory

Grid of products belonging to a single category. Same layout knobs as
`ProductGrid` (forwarded to an inner grid).

| Attribute | Type | Default | Notes |
|-----------|------|---------|-------|
| `categoryID` | number | — | The category to show |
| `rows` | number | 3 | Number of rows |
| `rowsMobile` | number | — | Rows on mobile |
| `maxWidth` | number | — | Max grid width |
| `maxMargin` | number | — | Max horizontal margin |

```html
<ProductsByCategory categoryID="22" rows="2" />
```

---

## ProductCard

A single product card.

| Attribute | Type | Notes |
|-----------|------|-------|
| `productoID` | number | The product to render |
| `mode` | string | `vertical` \| `horizontal` |
| `useQuantityControls` | boolean | Show quantity stepper |
| `hideCloseButton` | boolean | Hide the card's close button |

```html
<ProductCard productoID="104" mode="vertical" useQuantityControls />
```

---

## CategoryDescription

Renders the description text of one or more categories.

| Attribute | Type | Notes |
|-----------|------|-------|
| `categoryIDs` | number[] | e.g. `categoryIDs="[22]"` |

```html
<CategoryDescription categoryIDs="[22]" class="max-w-3xl mx-auto text-center" />
```

---

## ImageEffect

A photo with an optional clip/composition **layout** and a visual **effect**.
Two ways to use it:

**Fill mode** — full-bleed background; overlay text as *sibling* nodes after it
(the parent must be `relative`):

```html
<section class="relative overflow-hidden min-h-[420px] flex items-center px-6 py-12">
  <ImageEffect fill fit="cover"
    src="https://ivanjoz.github.io/genix-assets/images/business-workspace/5.avif"
    effect="overlay" tint="#0f172a" intensity="0.9" />
  <div class="relative z-10 max-w-6xl mx-auto">…overlaid text…</div>
</section>
```

Use `fill` only when the immediate parent has real dimensions (`min-h-*`,
`h-*`, or equivalent). A `fill` ImageEffect inside a plain right/left flex or
grid column with no height collapses visually. For a side image, use box mode.

**Box mode** — photo in a clipped/aspect box; children render *inside* it:

```html
<ImageEffect
  src="https://ivanjoz.github.io/genix-assets/images/business-workspace/6.avif"
  layout="curve-right" effect="duotone-matrix" tint="#4c2d82" tint2="#f59e0b"
  intensity="1" aspectRatio="21/9"
  class="rounded-2xl overflow-hidden min-h-[420px] flex items-center p-12">
  <div class="max-w-sm">…text on the tinted side…</div>
</ImageEffect>
```

Side-column image example:

```html
<div class="flex-1">
  <ImageEffect
    data-role="image"
    src="https://ivanjoz.github.io/genix-assets/images/business-workspace/5.avif"
    fit="cover" aspectRatio="4/3"
    class="w-full min-h-[360px] rounded-2xl overflow-hidden" />
</div>
```

| Attribute | Type | Notes |
|-----------|------|-------|
| `src` | string | Image URL |
| `layout` | string | `fade-{left,right,up,down}`, `slash-{left,right}`, `curve-{left,right}`, `curve-convex-{left,right}` |
| `effect` | string | `duotone`, `duotone-matrix` (uses `tint`+`tint2`), `glass`, `vignette`, `overlay` |
| `tint` | string | Primary tint/overlay color (maps to the `color` prop) |
| `tint2` | string | Second tint for `duotone-matrix` (maps to `color2`) |
| `intensity` | number | 0..1 overlay/tint strength |
| `blur` | number | Blur amount |
| `aspectRatio` | string | e.g. `"21/9"` |
| `fill` | boolean | Absolute background layer of the positioned parent |
| `fit` | string | `cover` (default), `contain`, `contain-left`, `contain-right` (ignored by slash/curve layouts) |

**Available photos (the only ones that exist):**
`https://ivanjoz.github.io/genix-assets/images/business-workspace/{4,5,6,7,8,9}.avif`

---

## Slider

Carousel. **Each direct child node becomes one slide** — children are the
slides, not an attribute. A slide can be a full subtree (e.g. an `ImageEffect`
fill hero with overlaid text).

| Attribute | Type | Default | Notes |
|-----------|------|---------|-------|
| `autoplay` | boolean | — | Auto-advance |
| `interval` | number | 5000 | ms between slides |
| `loop` | boolean | true | Wrap around |
| `arrows` | boolean | true | Show prev/next arrows |
| `dots` | boolean | true | Show pagination dots |

```html
<section class="relative overflow-hidden min-h-[420px] flex items-center">
  <Slider autoplay interval="6000">
    <div class="relative min-h-[420px] flex items-center px-6 py-12">…slide 1…</div>
    <div class="min-h-[420px] w-full flex items-center justify-center bg-slate-800">…slide 2…</div>
  </Slider>
</section>
```

---

## TabbedLayer

Tabbed container: a strip of options selects which single child is shown.
**Each direct child is one tab panel.** Provide `options` with `|`-separated
labels matching the child count (falls back to "Tab N").

| Attribute | Type | Notes |
|-----------|------|-------|
| `options` | string | Pipe-separated labels, e.g. `"My Style\|Option 2\|Option 3"` |

```html
<TabbedLayer options="Overview|Specs|Reviews">
  <div class="p-12">…overview panel…</div>
  <div class="p-12">…specs panel…</div>
  <div class="p-12">…reviews panel…</div>
</TabbedLayer>
```
