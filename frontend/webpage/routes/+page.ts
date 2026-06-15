import { getStoreWebpage } from '$services/ecommerce/page-content.svelte';

// Prerender the root page only for the per-company build (VITE_COMPANY_ID set). In
// dev this is false → the page is CSR and load() runs client-side. The --page-base
// build (VITE_PRERENDER_BASE) renders ONLY the /base shell, so the root opts out.
export const prerender =
  !!import.meta.env.VITE_COMPANY_ID && !import.meta.env.VITE_PRERENDER_BASE;

export async function load() {
  // ONE public call (GET.p-webpage) returns this page's SEO config + content. Runs at
  // build time for the prerender (bakes head + section markup into the HTML) and
  // client-side in dev. Failure is non-fatal: the build still completes and the
  // page's onMount refresh fills content in at view time.
  try {
    return await getStoreWebpage();
  } catch (webpageLoadError) {
    console.error("[StorePage] webpage load failed", webpageLoadError);
    return { sections: [], css: '', seo: {} };
  }
}
