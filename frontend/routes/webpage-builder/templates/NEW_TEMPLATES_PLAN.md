# Plan: 20 New Section Templates

> Rules, components, theming, ImageEffect API and the verify loop are documented in
> **`HOW_TO_CREATE_TEMPLATE.md`**. This file only plans the 20 specific templates.

## Conventions for this batch
- **Copy:** English placeholder text (matches existing templates).
- **Icons:** emoji glyphs + CSS shapes (no `<svg>` exists).
- **Photos:** only `business-workspace/{4..9}.avif` exist — assigned below; non-photo
  sections use palette colors / Tailwind gradients.
- **Theming:** palette numbers (`background-color="9"`, `color="1"`, …) so they re-skin.
- Every template ships a `thumbnail` and `data-role` editable nodes.
- Each goes in its category subfolder and is registered in `index.ts`.

## The 20 templates

### Headers / nav
1. **site-header-nav-v1** — `header/` — Logo (text) left, nav links center, CTA button right;
   sticky-looking bar on light bg. data-role: title (logo), button.

### Hero
2. **hero-split-image-v1** — `hero/` — 2-col grid: text block left (eyebrow + title + content
   + button), `ImageEffect` box (photo 4, `aspectRatio 4/3`, rounded) right. Stacks on mobile.
3. **hero-gradient-stats-v1** — `hero/` — Centered title/content/button on a `bg-linear-to-br`
   dark gradient, with a 3-up stats row (`12k+ / 98% / 24h`) below.
4. **hero-minimal-eyebrow-v1** — `hero/` — Generous whitespace, small uppercase eyebrow span,
   large title, one-line content, dual buttons (primary + ghost).

### Features
5. **feature-trio-icons-v1** — `features/` — Heading + 3 cards, each: emoji-in-rounded-square,
   sub-title, blurb. `grid md:grid-cols-3`.
6. **feature-alternating-v1** — `features/` — Two alternating rows of `ImageEffect` (photos 6, 7)
   + text; image on opposite side each row.
7. **feature-checklist-v1** — `features/` — Left: title + ✓ benefits list (`ul`); right:
   `ImageEffect` (photo 8) with `curve-left`.
8. **stats-band-v1** — `features/` — Dark band (`background-color="9"`) with 4 large metric
   numbers + labels in a responsive grid.

### CTA
9. **cta-banner-v1** — `cta/` — Full-width gradient banner, centered title + content + button.
10. **cta-split-image-v1** — `cta/` — `ImageEffect` `curve-right` duotone (photo 5) with text
    panel + button beside it.
11. **cta-newsletter-v1** — `cta/` — Title + subtitle, and an email "input" mock
    (styled `div` + `span` placeholder) next to a Subscribe button. (No real `<input>`.)

### Testimonials / social proof
12. **testimonials-grid-v1** — `testimonials/` — Heading + 3 quote cards (quote, name, role,
    ★★★★★ as text). `grid md:grid-cols-3`.
13. **testimonial-spotlight-v1** — `testimonials/` — One large centered quote, big quotation
    mark, author + role, on a tinted bg.
14. **logos-trusted-by-v1** — `testimonials/` — "Trusted by" eyebrow + a row of text "logos"
    (styled brand wordmarks) in muted color.

### Products (ecommerce)
15. **products-featured-grid-v1** — `products/` — Heading + `ProductGrid` + centered
    "View all →" link.
16. **products-category-v1** — `products/` — Heading + `CategoryDescription` + `ProductsByCategory`.

### Text / content
17. **about-story-v1** — `text/` — 2-col: story paragraphs left (title + content), `ImageEffect`
    (photo 9) right; small signature line.
18. **faq-list-v1** — `text/` — Heading + stacked Q/A items (`div`s with bold question + answer
    paragraph, divider `hr`s).

### Gallery
19. **gallery-grid-v1** — `gallery/` — Heading + responsive grid of the 6 photos (4–9) as
    `img` tiles with rounded corners / hover-ish styling.

### Footer
20. **site-footer-v1** — `footer/` — Dark footer: brand blurb column + 2–3 link columns +
    a small newsletter mock, bottom copyright row with `hr`.

## New subfolders to create
`features/`, `cta/`, `testimonials/`, `text/`, `gallery/`, `header/`, `footer/`.

## Execution
1. Build in batches of ~5, registering each in `index.ts` as I go.
2. `bun run dev:main` (:3570) running in background.
3. For each: screenshot `template-preview?id=<id>` (live + thumbnail) with headless Chrome,
   eyeball, fix, repeat until professional.
4. Final pass: re-screenshot all 20.
