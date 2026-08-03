export const csr = true;
// SSR en los dos builds de tienda: el antiguo prerender por company (VITE_COMPANY_ID) y
// el bundle del renderer (VITE_RENDERER_BUILD), que el Lambda ejecuta bajo demanda.
// En dev / la vista embebida del builder sigue siendo un SPA puro en CSR.
export const ssr = !!import.meta.env.VITE_COMPANY_ID || !!import.meta.env.VITE_RENDERER_BUILD;
// El renderer NO prerenderiza: no hay company ni contenido en tiempo de build.
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
