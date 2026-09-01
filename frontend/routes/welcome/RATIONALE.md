## Github link in the header
**Context** — The landing page had no route to the source, which is one of its three selling points ("Código abierto").
**Decision** — A "Github" link sits after "Ingresar" in the desktop nav and as its own full-width row under the account actions in the mobile menu, pointing at `https://github.com/ivanjoz/genix` in a new tab. The Octicons mark is inlined as a `{#snippet githubMark}` on its native 16-unit grid, filled with `currentColor`.
**Rationale** — Inlining as a snippet keeps one copy of the path for both breakpoints and lets the mark inherit the link's colour and hover state, which an `<img>` could not. The label is left untranslated: "Github" is a proper noun. In the mobile menu it sits outside the Sign in / Register grid so developer mode's two-column layout stays intact.

## Raleway SemiBold on the large headlines
**Context** — The big titles needed a display face distinct from the Open Sans body text, matching the treatment used on the unicore_website landing page.
**Decision** — Added a `display` family (Raleway 600, `frontend/static/fonts/raleway-v37-latin-600.woff2`) plus a `.ff-display` alias to the shared `styles/fonts.css`, and applied it to the hero `h1` and the three section `h2`s. Smaller headings such as the login card title keep Open Sans. Its `@font-face` lists the relative `/fonts/...` path before the `genix.un.pe` copy, and the `.ff-display` rule is re-asserted inside the ≤749px media query.
**Rationale** — Keeping the face in the shared stylesheet avoids a second typography source of truth, and the alias means the scope of the change is one class in the markup. The relative-first `src` makes the font resolve in local dev before the static folder is deployed, while the absolute fallback serves the storefront, which ships no `static/fonts` of its own. The mobile re-assert is needed because the Open Sans → Inter swap remaps `.font-bold` at equal specificity and later in the file, which would otherwise pull the headlines back to Inter; Raleway is a display face and is deliberately not part of that swap.

## Simplified primary navigation
**Context** — The welcome header should not direct visitors to the roadmap section.
**Decision** — Removed the Roadmap item from the shared desktop and mobile navigation list.
**Rationale** — Both menus derive from this list, so the change remains consistent without removing the page content.

## Highlighted availability notice
**Context** — The release date must read as an intentional notice rather than body copy over the dark hero image.
**Decision** — The public hero uses a dark translucent, violet-bordered notice card with `#ffeed6` text and a full-height rectangular accent.
**Rationale** — It preserves contrast and visual separation while belonging to the hero's dark palette, without a top-aligned bullet.

## Single public endpoint and developer-only CTAs
**Context** — The hosted application has a single intended backend, while public registration is not yet open.
**Decision** — The welcome page selects the first configured endpoint without showing a picker. It shows the February 8, 2027 availability notice and hides every registration button by default; visiting with `?dev=x` persists developer mode in local storage and exposes registration plus product-discovery CTAs.
**Rationale** — This removes server-level and registration choices from the public flow while retaining an explicit, persistent development entry point for testing the registration workflow.
