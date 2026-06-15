// The base route is a runtime CSR storefront shell: it fetches its published CDN
// snapshot at view time (see +page.svelte), so there is NO build-time load(). We only
// prerender it (SSR-bakes the spinner shell, SEO defaults, and the early-fetch <head>
// script) when prerender.mjs --page-base sets VITE_PRERENDER_BASE.
//
// IMPORTANT: this prerender is a COMPANY-AGNOSTIC TEMPLATE, not a per-company page.
// It is built ONCE and reused for many companies/pages: downstream tooling copies this
// HTML, rewrites the `cdn-url` + `page-id` <head> metas (+page.svelte:60-61) to point at
// a different tenant/page's CDN snapshot, and serves the copy from another domain. The
// content (sections/SEO) is never baked in — everything is driven by those two metas at
// view time. VITE_COMPANY_ID only flips SSR/prerender on; its value is just a placeholder
// default in the emitted metas (the copy step overwrites it) and does NOT scope output.
export const prerender = !!import.meta.env.VITE_COMPANY_ID && !!import.meta.env.VITE_PRERENDER_BASE;
