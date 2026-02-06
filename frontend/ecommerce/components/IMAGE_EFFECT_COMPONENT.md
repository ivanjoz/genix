# ImageEffect Component

The `ImageEffect` component is a powerful tool for the Genix ecommerce builder. It separates **Layout** (how the image is cropped to create space for text) from **Effect** (the visual treatment applied to the image).

## Properties

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `src` | `string` | **Required** | The URL of the image. |
| `layout` | `string` | `''` | Composition layout (Slash, Curve, Fade). Creates space for text. |
| `effect` | `string` | `''` | Visual treatment (Duotone, Glass, Vignette). |
| `color` | `string` | `'#ffffff'` | Primary color for background space and gradients. |
| `color2` | `string` | `'#000000'` | Secondary color (used for `duotone` effects). |
| `intensity`| `number` | `1` | Strength of the effect/layout opacity (0 to 1). |
| `blur` | `number` | `0` | Gaussian blur radius in pixels. |
| `aspectRatio`| `string`| `'auto'` | CSS aspect-ratio (e.g., `'16/9'`, `'21/9'`). |
| `css` | `string` | `''` | Tailwind/CSS classes for the main container. |
| `children` | `Snippet` | `undefined` | Content to render over the image background. |

---

## 1. Composition Layouts (`layout`)
These define the physical shape of the image and where the "empty space" for text is created.

| Layout | Description |
|--------|-------------|
| `fade-right` | **Image on Right**. Solid `color` on left with a soft gradient transition. |
| `fade-left` | **Image on Left**. Solid `color` on right with a soft gradient transition. |
| `slash-right`| **Image on Right**. Hard diagonal cut (50/50 split). Uses reflection trick. |
| `slash-left` | **Image on Left**. Hard diagonal cut (50/50 split). Uses reflection trick. |
| `curve-right`| **Image on Right**. Concave curve biting into the top corner. |
| `curve-left` | **Image on Left**. Concave curve biting into the top corner. |
| `curve-convex-right`| **Image on Right**. Pronounced outward sweeping curve. |
| `curve-convex-left` | **Image on Left**. Pronounced outward sweeping curve. |

> **Note:** `*-right` layouts place the image on the right side of the container.

---

## 2. Visual Effects (`effect`)
These are filters and treatments applied to the image itself. They can be combined with any layout.

| Effect | Description |
|--------|-------------|
| `duotone-matrix`| High-precision 2-color mapping using SVG filters. |
| `duotone` | Classic CSS blend-mode based 2-color mapping. |
| `glass` | Frosted glass effect (Blur + semi-transparent tint). |
| `vignette` | Radial fade towards the edges using `color`. |
| `overlay` | Solid color layer with adjustable opacity. |

---

## Usage Examples

### Hero Section with Curved Layout + Duotone
Combine a geometric composition with an artistic treatment.

```svelte
<ImageEffect 
  src="/hero.jpg" 
  layout="curve-right" 
  effect="duotone-matrix" 
  color="#4c2d82" 
  color2="#ffa500"
  css="min-h-[500px] flex items-center p-12"
>
  <div class="max-w-md">
    <h1 class="text-white text-5xl">Future of Fashion</h1>
    <p class="text-white">Discover our latest collection today.</p>
  </div>
</ImageEffect>
```

### Premium Product Card with Glass
Apply a soft fade and a frosted look.

```svelte
<ImageEffect 
  src="/watch.jpg" 
  layout="fade-right" 
  effect="glass" 
  blur={10}
  aspectRatio="16/9"
/>
```
