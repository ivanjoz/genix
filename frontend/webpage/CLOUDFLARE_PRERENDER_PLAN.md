# Cloudflare Multi-Tenant Storefront — Prerender for SEO + Client Re-render Plan

## Goal

Serve each SaaS client's storefront on its own subdomain (e.g. `tienda-x.un.pe`)
from Cloudflare, with:

1. **A prerendered static `.html`** so crawlers/social bots get fully-formed
   markup (SEO). Generated at *publish time* from the saved page AST.
2. **A client takeover** that, on load, fetches the current page content from the
   API and **re-renders the whole page from scratch**, replacing the prerendered
   markup. Layout shift is acceptable. This keeps real users always fresh.

## The key decision: NO hydration, ever

The realtime-staleness / hydration-mismatch problem is avoided **by construction**,
not patched:

- The prerendered HTML is **inert SEO markup only** — it ships zero component JS for
  the content sections.
- The client app boots with Svelte's **`mount()` (fresh client render)**, never
  `hydrate()`. `mount()` builds new DOM and does not reconcile against existing
  nodes, so there is no server/client diff to mismatch. We simply remove the
  prerendered block when the live render is ready.
- Therefore the prerendered content can be arbitrarily stale relative to the API —
  it does not matter, because the client never tries to match it.

This is effectively: **static SEO snapshot + SPA that re-renders into the same
page.** The current storefront is already a CSR SPA (`ssr=false`, `csr=true`); we
keep that and bolt a prerendered SEO block in front of it.

## Current state (verified)

- Storefront: CSR SPA. `routes/+layout.ts` → `ssr=false, csr=true, prerender=false`;
  `svelte.config.js` → `adapter-static`, `fallback: index.html`, `base: /store`.
  Deployed to GitHub Pages today.
- `routes/+page.svelte` already fetches `getPageContent()` in `onMount` and renders
  `EcommerceRenderer`. **This is exactly the "client re-render" we want** — it stays.
- Page content: ScyllaDB, `GET.ecommerce-page-content`, scoped to
  `req.User.CompanyID` (auth-only). Sections + one pre-generated whole-page CSS
  stylesheet (stored on section 1's `Css` column).
- Renderer + section templates: no top-level `window`/`document` access → SSR-safe.
  Registry (`ecommerce-templates/registry`) is statically importable.
- Tenant resolution (`Env.getCompanyID`): localStorage → `<meta name="loc">` → path.
  Products load from a per-company CDN snapshot `products-c<id>.db` (not the page API).

## Architecture

```
                    Cloudflare zone  *.un.pe  (wildcard DNS)
                                  │
                          ┌───────▼────────┐
   visitor / crawler ───► │  Serve Worker  │  reads Host → subdomain
                          └───────┬────────┘
                                  │ subdomain → companyID  (KV: tenant map)
                                  │ fetch published HTML    (R2: artifacts)
                                  ▼
                    returns prerendered index.html
                    (SEO block + inlined CSS + SPA bootstrap + companyID)
                                  │
                  browser runs SPA  ──► mount() fresh
                                  │
                  SPA fetches public page-content API ──► re-renders, removes SEO block
                                  │
                  SPA loads products-c<id>.db from CDN for product islands
```

### New pieces
1. **SSR render module** (`renderer/ssr-entry.ts`) — `renderPage({sections, css,
   palette, companyID, meta})` → `{ html, head }` using `import { render } from
   'svelte/server'` on a thin wrapper around `EcommerceRenderer`. Built as an SSR
   bundle (`vite build --ssr`) so it runs in Node or a CF Worker.
2. **Render-on-publish** — produces the static `index.html` per tenant and writes it
   to **R2** keyed by subdomain. (Variant: render on-demand in the Serve Worker and
   cache — see Alternatives.)
3. **Serve Worker** — maps `Host` → companyID via **KV**, serves the R2 artifact,
   injects companyID. Bound to the wildcard route.
4. **Public page-content API** — unauthenticated read of *published* content by
   companyID (the current GET is auth-scoped; public visitors have no token).
5. **Tenant registry** — subdomain ⇄ companyID, claimed in the builder, written to KV.

## Publish pipeline

1. Client edits in builder → clicks **Publish**.
2. Builder `savePageContent()` POSTs sections (unchanged) → backend stores in
   ScyllaDB as today, marking this revision **published**.
3. Backend (or a publish hook) invokes the **render step** with the published
   snapshot (sections + pre-generated CSS + companyID + SEO meta):
   - Run `renderPage()` → SEO HTML + `<head>` CSS.
   - Wrap in the `app.html` template: `<div id="seo-prerender">{html}</div>`,
     inlined `<style>{css}</style>`, SEO meta tags, `<script>window.__COMPANY_ID__=N</script>`,
     and the SPA bootstrap `<script>` tags.
   - Write to `R2: {subdomain}/index.html` (+ shared static assets once).
4. **Purge** the Cloudflare cache for that subdomain.

Between publishes the artifact is immutable → cacheable indefinitely at the edge.

## Serve pipeline (per request)

1. `tienda-x.un.pe` → Serve Worker.
2. Worker: `Host` → `subdomain` → KV lookup → `companyID`. (404 if unknown.)
3. Fetch `R2[{subdomain}/index.html]`, return it. Edge-cache with long TTL; the
   publish purge keeps it fresh.
4. Browser parses static SEO HTML (crawlers stop here — fully indexable).
5. SPA bootstraps: `mount()` fresh → `getPageContent()` (public API, by companyID
   from `window.__COMPANY_ID__`) → `EcommerceRenderer` renders live content into the
   app root → remove `#seo-prerender`. Layout shift is fine.
6. Product islands load `products-c<id>.db` from the CDN as today.

## Client takeover mechanism (the hydration-free swap)

- Served HTML body:
  ```html
  <div id="seo-prerender"> ...static section markup... </div>
  <div id="app"></div>            <!-- SPA mounts here, fresh -->
  ```
- SPA entry uses Svelte 5 **`mount(App, { target: document.getElementById('app') })`**
  (CSR, `ssr=false` stays) — never `hydrate()`.
- After the live render's first paint (e.g. in `onMount` once `sections` are set),
  remove `#seo-prerender`. No reconciliation, no mismatch — guaranteed.
- Confirm SvelteKit's adapter-static client entry uses `mount` (it does when
  `ssr=false`); if we move off SvelteKit's router for the storefront we control this
  directly.

## Backend changes

1. **Public page-content read**: new `GET.ecommerce-page-content-public` (no auth)
   taking `?company-id=` (or resolved from a signed/host param), returning only
   `Status>=1` published sections + the section-1 CSS. Reuses `GetPageContent`
   internals but scopes by an explicit companyID instead of `req.User.CompanyID`.
   *NEVER trust the client* — only return published content; validate companyID.
2. **Publish hook**: on publish, trigger the render step + R2 write + KV upsert +
   cache purge (via Cloudflare API token). Could be a Go call to a small render
   endpoint (the SSR bundle hosted as a Worker) to keep Go free of JS.
3. **Subdomain registry**: store subdomain ⇄ companyID (new column/table), validated
   for uniqueness; mirror into KV on change.

## Frontend / build changes

1. `renderer/ssr-entry.ts` + a `vite build --ssr` target producing the render module.
2. SPA entry confirmed to `mount` (not hydrate) and to remove `#seo-prerender` after
   first live paint.
3. `Env.getCompanyID`: add `window.__COMPANY_ID__` as the first source for the
   storefront (subdomain-driven), ahead of path/localStorage.
4. Drop the GitHub Pages `?p=` SPA hack in `app.html`; replace with the Cloudflare
   serve model. Likely move storefront off `base: /store` to root per-subdomain.

## Alternatives / variants

- **B-on-demand (render in Serve Worker + cache)**: skip the publish render step;
  the Serve Worker SSR-renders on cache-miss (reading the published snapshot from
  KV/R2 or the public API) and caches the HTML. Simpler publish (just purge),
  always correct, one render per cache-miss. Recommended if the publish hook is hard
  to wire from Go. Same client takeover.
- **B-Node-CI**: render via a Node step in CI instead of a Worker. Fine for low
  publish frequency; adds CI latency to "Publish".

## Open questions / assumptions to confirm

1. Is `un.pe` already on Cloudflare with **wildcard `*.un.pe` DNS** + SSL we can use?
2. **Render trigger**: publish-time render-to-R2 (literal prerender) vs
   on-demand-in-Worker-then-cache? (Both meet the SEO + fresh-client goals; the
   latter is operationally simpler.)
3. Subdomain assignment UX: auto from company slug, or user-chosen in the builder?
4. Any sections that must run JS at the SEO layer (none today — all content is
   re-rendered client-side), confirming we can ship the SEO block as pure markup.
5. Custom domains later (client brings `www.acme.com`)? Affects KV/SSL design now.

## Validation checklist (post-implementation)

- `renderPage()` produces identical visual output to the live SPA for each section type.
- Crawler (curl / Google Rich Results test) sees full content in the static HTML.
- Real browser shows live API content; `#seo-prerender` removed; no console
  hydration warnings (because we never hydrate).
- Publish → purge → next request reflects new content.
- Unknown subdomain → clean 404.
