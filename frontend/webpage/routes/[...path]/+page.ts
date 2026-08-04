import { Env } from '$core/env';
import { getStoreWebpageFromCDN } from '$services/ecommerce/page-content.svelte';

export async function load() {
  // El snapshot publicado (live/pages/<companyID>-<pageID>.json) lo escribe
  // PostPageContent en cada guardado: es el mismo payload que devuelve GET.p-webpage
  // (SEO + secciones activas) pero servido por el CDN, sin pasar por la API ni requerir
  // credenciales — que es lo que necesita el SSR dentro del Lambda.
  //
  // El pageID lo fija hooks.server.ts en el SSR y la meta page-id en el cliente.
  // Fallar no es fatal: el render se completa igual y el refresco de onMount rellena el
  // contenido al momento de la vista.
  try {
    return await getStoreWebpageFromCDN(Env.getPageID());
  } catch (webpageLoadError) {
    console.error("[StorePage] webpage load failed", webpageLoadError);
    return { sections: [], css: '', seo: {} };
  }
}
