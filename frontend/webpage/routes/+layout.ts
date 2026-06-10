export const csr = true;
// SSR is enabled only for the per-company prerender build (VITE_COMPANY_ID set).
// In dev / the admin-embedded view it stays a pure CSR SPA, as before.
export const ssr = !!import.meta.env.VITE_COMPANY_ID;
export const prerender = !!import.meta.env.VITE_COMPANY_ID;
// This prevents automatic data serialization
export const trailingSlash = 'ignore';

const localHosts = ["localhost", "127.0.0.1", "sveltekit-prerender"];

export async function load({ url }) {
  (globalThis as any)._isLocal = localHosts.some(x => url.host.includes(x));

  // Page SEO + content are loaded per-page in +page.ts (one public p-webpage call).
  // The catalog is loaded client-side from the single shared source (see +layout.svelte onMount).
  return {};
}
