# Plan — Per-company storefront prerender via SvelteKit (Cloudflare Pages)

## Goal
A script that takes a **companyID** and produces a **static artifact** (HTML + CSS +
JS, content + SEO baked into the HTML) ready for **Cloudflare Pages**, using
**SvelteKit's built-in prerender** (not a headless browser, not a separate SSR
module). One build per tenant.

## Why this shape
- Prerender = SvelteKit runs the app through **SSR at build time** and bakes the
  HTML. The render tree is already SSR-safe (browser globals are guarded), so this
  is low-risk on the component side.
- The only real refactor: content/SEO must load in a **`load` function** (runs at
  build), not `onMount` (browser-only — prerender never runs it, would bake an
  empty shell).
- Tenant is pinned at build time via **`VITE_COMPANY_ID`** (Vite inlines it into
  both the Node build and the client bundle). This single env var drives: SSR/
  prerender on, API scoping (`Env.getCompanyID`), and the root base path.

## The blocker it solves
`GET.ecommerce-page-content` / `GET.website-config` are auth-scoped
(`req.User.CompanyID`). A token-less build — and any deployed anonymous visitor —
401s. So we add ONE public `p-` read (the `p-` prefix bypasses auth, `main-handlers.go:123`)
that returns both the page's SEO config and its content in a single call.

---

## Changes

### 1. Backend — one public read (`webpage_public.go`)
- **`GET.p-webpage?company-id=<id>&id=<pageID>`** → `{ Config, Sections }`: the
  page's SEO metatags (only the known SEO keys — never the domain) plus its active
  content sections (+ section-1 CSS). Validates `company-id > 0`; `id` defaults to
  the root/Inicio page (`defaultPageID`). Scoped by the query param, never a
  session. *Never trust the client.* (Section rows have no per-section publish flag
  and the root is a system page with no `webpages` publish gate, so this is the
  live active content; real user pages can add a gate later.)
- The SEO-key filter is shared via `publicSeoMetatags()` in `webpage_config.go`.
- Register in `backend/webpage/main.go`.

### 2. Frontend — one store loader
- `services/ecommerce/page-content.svelte.ts`: `getStoreWebpage(pageID = 0)` — one
  `GET.p-webpage` call → `{ sections, css, seo }`. No auth; `makeRoute` appends
  `company-id` from `Env.getCompanyID()`. (The authed `getPageContent`/
  `getWebsiteConfig` stay for the admin builder.)

### 3. Frontend — pin tenant
`core/env.ts` `getCompanyID()`: first source = `import.meta.env.VITE_COMPANY_ID`
(undefined elsewhere → no effect on the admin app). Gives the right company at
build (Node) and on the deployed client (inlined), so products/API scope correctly.

### 4. Frontend — data via `load`, gated SSR/prerender
- `routes/+page.ts` (new): `load()` → `getStoreWebpage()` → `{sections, css, seo}`
  (one call). `export const prerender = !!import.meta.env.VITE_COMPANY_ID`.
- `routes/+layout.ts`: `export const ssr = !!import.meta.env.VITE_COMPANY_ID`
  (dev stays CSR); no data fetch (page owns SEO + content).
- `routes/+page.svelte`: render content + SEO `<head>` from `data` (so markup exists
  at SSR) **plus** an `onMount` refresh to the latest content (recovers if the build
  fetch was empty; no hydration mismatch — initial render uses the same `data` SSR used).
- `routes/+layout.svelte`: product loads stay in `onMount` (client islands).

### 5. Build — root base for subdomain deploy
`svelte.config.js`: `base = process.env.VITE_COMPANY_ID ? '' : '/webpage-app'`.
Dev keeps `/webpage-app` (proxy on :3572); the prerender build serves at `/`.

### 6. Script — `scripts/prerender.mjs` (bun)
`--company <id>` (required), `--out <dir>` (default `dist-prerender/<id>`):
sets `VITE_COMPANY_ID` → `bun run build` (SvelteKit prerenders `/`, bakes HTML+CSS+JS
into `build/`) → copies `build/` → `out/`, then flattens to a single folder, rewrites
JS/CSS asset URLs to the asset base (`<FRONTEND_CDN>/websites/<id>`, read from
`credentials.json`; `--asset-base <url>` overrides), merges the stylesheets into one,
and inlines a few tiny single-purpose chunks. Prints the upload + Worker-deploy steps.
> The deployed **dev API** (`genix-dev-api-2.un.pe`, from `PUBLIC_ENDPOINTS`) is the
> build-time fetch target — the public `p-` endpoints must be deployed there for the
> build to bake real content. If the fetch fails the build still succeeds (empty
> content) and the client `onMount` refresh fills it in at view time.

## Decisions (confirmed)
1. Public endpoints — **yes**.
2. Published-only — **yes**, always the **root** page.
3. Deploy at **root `/`**.
4. **No browser dependency** — use SvelteKit's own prerender.

## Validation
- `go build ./...` + `check_tables`.
- `bun run build` (no env) → existing CSR build still works.
- `VITE_COMPANY_ID=<id> bun run build` → build succeeds, `build/index.html` at root
  base, content baked once the public endpoints are live on the dev API.
- `curl build/index.html` shows section markup + SEO tags (crawler test).
- `wrangler pages dev build/` (or the out dir) serves it; logged-out visitor sees
  content, no 401.
