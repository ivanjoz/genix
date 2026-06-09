# Webpage: Cloudflare Deploy + SEO/Domain Config Maintainer — Plan

Authoritative plan. Deep architecture notes live in
`frontend/webpage/CLOUDFLARE_PRERENDER_PLAN.md`.

Confirmed decisions:
- Cloudflare zone `un.pe` exists; subdomains registered manually today, **no
  wildcard** — provisioning is **per-claim via the Cloudflare DNS API**.
- **Render-to-R2 at publish** only (render-on-demand is a future feature — NOT now).
- Subdomains are **user-chosen** in the builder.
- Support **custom domains** (`www.acme.com`) with **Cloudflare auto-SSL**.
- New **SEO + domain config maintainer** at `frontend/routes/webpage-builder/config/`.
- Domain change is **throttled to once per 2 hours per company** (single domain field).

> Folder rename context: the standalone storefront app is now `frontend/webpage/`
> (was `frontend/ecommerce/`); the builder route is now
> `frontend/routes/webpage-builder/` (was `routes/webpage/`). The `$ecommerce`
> alias name is unchanged and now points at `frontend/webpage/`.

---

## Part 1 — Rendering & hydration model (core decision)

**Two-step: prerendered SEO markup + fresh client re-render. NO hydration.**

- Prerendered `.html` is inert SEO markup only (zero content-section JS).
- Client boots with Svelte 5 **`mount()`** (fresh CSR render), never `hydrate()`,
  so there is no server/client diff to mismatch. The prerendered block may be
  arbitrarily stale; the client fetches the API, re-renders everything, then
  removes the SEO block. Layout shift accepted.
- The current storefront (`frontend/webpage/routes/+page.svelte`) already fetches
  `getPageContent()` in `onMount` and renders — that is step 2, kept as-is.

Verified SSR-safe: renderer + section templates have no top-level
`window`/`document`; the registry is statically importable.

---

## Part 2 — Publish → R2 pipeline (render-to-R2 at publish)

1. **SSR render module** `frontend/webpage/renderer/ssr-entry.ts` exporting
   `renderPage({ sections, css, palette, companyID, seo })` → `{ html, head }`
   via `import { render } from 'svelte/server'` around `EcommerceRenderer`.
   Built as an SSR bundle (`vite build --ssr`) → loadable in Node / a CF Worker.
2. On **Publish** (builder → `savePageContent()` stores sections + marks the
   revision published), the backend invokes a **render step** (a small CF Worker
   hosting the SSR bundle, since Go can't run Svelte) with the published snapshot +
   the store's SEO config:
   - `renderPage()` → SEO `<body>` markup + inlined pre-generated CSS.
   - Wrap in `app.html`: `<div id="seo-prerender">{html}</div>`, `<style>{css}</style>`,
     all SEO `<head>` tags from the maintainer,
     `<script>window.__COMPANY_ID__=N</script>`, and the SPA bootstrap.
   - Write to **R2** `{companyID}/index.html` (shared static assets uploaded once).
3. **Purge** the Cloudflare cache for that store's hostname(s).

Between publishes the artifact is immutable → cache indefinitely at the edge.

---

## Part 3 — Domains: per-claim provisioning (no wildcard)

On domain save the backend provisions via the Cloudflare API; there is no wildcard.

- **Subdomains on `un.pe`** (e.g. `tienda-x.un.pe`):
  - **Validate (NEVER trust client):** lowercase/dns-safe format; not in the
    **reserved list** (see below); and **the DNS record must not already exist** —
    query the Cloudflare DNS API for the hostname and reject if found.
  - Create a proxied DNS record (CNAME/A) for the hostname pointing at the Worker,
    plus a Worker route for that hostname.
- **Custom domains** (`www.acme.com`): register via **Cloudflare for SaaS — Custom
  Hostnames API**; client adds the CNAME; **SSL is issued automatically**. Surface
  the validation/SSL status in the maintainer.

**Reserved subdomains** (block, case-insensitive): `www`, `api`, `app`, `admin`,
`store`, `mail`, `cdn`, `static`, `assets`, `dev`, `staging`, `test`, `blog`,
`shop`, `pay`, `dashboard`, `support`, `status`, `docs`, `ns1`, `ns2`. Keep the
list in one shared place (backend constant) and validate against it server-side.

**Serve Worker** (bound to provisioned routes): `Host` → KV lookup
(`hostname → companyID`) → fetch `R2[{companyID}/index.html]` → return, injecting
`window.__COMPANY_ID__`. Unknown host → 404. Long edge-cache TTL, purged on publish.

**KV tenant map** `hostname → companyID`, upserted on each successful domain save
(old hostname removed from KV + DNS when replaced).

---

## Part 4 — SEO + Domain Config Maintainer

Dedicated route **`frontend/routes/webpage-builder/config/+page.svelte`**, built with
the project `Page` shell + standard form components (see `create-page-layout` skill
+ `UI_COMPONENTS.md`).

### Fields
- **Domain group** (single domain field)
  - `subdomain` input (`tienda-x` → `tienda-x.un.pe`) OR `customDomain`
    (`www.acme.com`) — one active domain per store. Read-only SSL/validation status.
  - **Disabled within the 2-hour window**: show a countdown ("puedes cambiar el
    dominio nuevamente en HH:MM") derived from the server's `DomainUpdated`.
- **SEO `<head>` group** (baked into the prerendered head on publish):
  `title`, `metaDescription`, `metaKeywords`, `ogTitle`, `ogDescription`, `ogImage`,
  `twitterCard`, `faviconUrl`, `themeColor`, `lang`, `canonicalUrl`, `robots`.
- **Save** posts the config; **Publish** (existing builder action) re-renders with
  the latest SEO tags.

### Frontend service
`frontend/services/ecommerce/store-config.svelte.ts` — `getStoreConfig()` /
`saveStoreConfig()` over `GET/POST.ecommerce-store-config`. Maintainer reads on
mount, binds fields, disables the domain input based on `DomainUpdated`.

---

## Part 5 — Backend changes

### New table `ecommerce_store_config` (one row per company)
Use the `create-database-tables` skill (paired `XRecord`/`XRecordTable`,
`GetSchema()`, static validation). Partition by `CompanyID`.

Fields (SUnixTime int32 for datetimes):
- `CompanyID int32` (partition key)
- `Domain string` — the single active hostname (subdomain `…un.pe` or custom).
- `IsCustomDomain int8`
- `Title, MetaDescription, MetaKeywords string`
- `OgTitle, OgDescription, OgImage string`
- `TwitterCard, FaviconUrl, ThemeColor, Lang, CanonicalUrl, Robots string`
- `DomainUpdated int32` — SUnixTime of the last domain change (throttle source)
- `PublishedRevision int32`, `Updated int32`, `UpdatedBy int32`, `Status int8`

### Handlers
- `GET.ecommerce-store-config` (auth) → the company's config.
- `POST.ecommerce-store-config` (auth) → upsert. **Validation:**
  - If `Domain` changed: enforce the **2-hour throttle per company** — reject if
    `core.SUnixTime() - cfg.DomainUpdated < 3600` (2h = 7200s; SUnixTime is
    `(unix-1e9)/2`, so 7200s = **3600 SUnixTime units**). Error includes remaining
    time. On success set `DomainUpdated = now`.
  - Validate format + reserved list; for subdomains, **confirm the DNS record does
    not already exist** (Cloudflare API) before creating it.
  - On success: create DNS record / custom hostname, upsert KV `hostname →
    companyID`, remove the previous hostname (DNS + KV).
  - SEO fields: length/format validation; no throttle.
- **Public read** `GET.ecommerce-page-content-public` (no auth, by `?company-id=`)
  — returns only published (`Status>=1`) sections + section-1 CSS, for the client
  re-render on the public storefront (the existing GET is `req.User.CompanyID`-scoped).
- **Publish hook**: render-to-R2 + cache purge (Part 2); ensure hostname provisioned.

### Config / secrets
Cloudflare **API token**, **Zone ID**, **R2 bucket**, **KV namespace**, **Custom
Hostnames fallback origin** → backend config/credentials.

---

## Part 6 — Frontend / build changes
1. `frontend/webpage/renderer/ssr-entry.ts` + `vite build --ssr` target.
2. Storefront entry confirmed to `mount` (not hydrate) and to remove
   `#seo-prerender` after first live paint.
3. `Env.getCompanyID`: add `window.__COMPANY_ID__` as the first source.
4. Drop the GitHub Pages `?p=` SPA hack in `frontend/webpage/app.html`; storefront
   moves off `base:/store` to root per-hostname on Cloudflare.
5. New maintainer route + `store-config.svelte.ts` service (Part 4).

---

## Implementation order
1. Backend: `ecommerce_store_config` table + GET/POST handlers + 2h throttle +
   reserved list + Cloudflare DNS provisioning (existence check).
2. Frontend: config maintainer route + `store-config.svelte.ts`.
3. SSR render module + `vite build --ssr`.
4. Cloudflare: R2 bucket, KV map, Serve Worker, render Worker.
5. Public page-content endpoint + storefront `mount`/SEO-block swap.
6. Wire Publish → render-to-R2 + purge. Validate (crawler sees content; live render
   replaces it; no hydration warnings; domain throttle rejects < 2h; reserved/
   existing subdomains rejected).
