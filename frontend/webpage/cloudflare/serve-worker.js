// Worker de las tiendas: resuelve hostname + ruta a un objeto de R2 y lo sirve.
//
// El HTML lo publica la Lambda de render (webpage-renderer/handler.mjs), un PUT por
// página, así que publicar una company NO toca a las demás. Antes esto eran Workers Static
// Assets, cuyo manifest reemplaza el namespace completo: republicar una sola tienda exigía
// tener en disco el HTML de todas.
//
// Archivo único a propósito: es el artefacto que scripts/cloudflare_deploy.go sube
// verbatim (sin esbuild ni wrangler) y a la vez el `main` de wrangler.jsonc para
// `wrangler dev`. Env: { SITE_HTML: R2Bucket } — el binding lo inyecta el deploy Go.

// Prefijo en R2 del HTML de las tiendas. Debe coincidir con HTML_KEY_ROOT del handler.
const HTML_KEY_ROOT = 'websites-html';

export default {
  async fetch(request, env, ctx) {
    // Los artefactos de la tienda son de solo lectura.
    if (request.method !== 'GET' && request.method !== 'HEAD') {
      return new Response('Method Not Allowed', {
        status: 405,
        headers: { Allow: 'GET, HEAD' },
      });
    }

    // La Cache API del POP absorbe las visitas siguientes: solo la primera de cada POP
    // llega a R2. La URL de la request ya incluye el hostname, así que la clave de caché
    // separa tenants sin trabajo extra. Solo GET: cache.put rechaza cualquier otro método.
    const isCacheable = request.method === 'GET';
    const cache = caches.default;
    if (isCacheable) {
      const cachedResponse = await cache.match(request);
      if (cachedResponse) return cachedResponse;
    }

    const objectKey = buildObjectKey(new URL(request.url));
    const storedObject = await env.SITE_HTML.get(objectKey);
    if (!storedObject) {
      console.warn('[serve-worker] objeto no encontrado', objectKey);
      return new Response('Not Found', { status: 404 });
    }

    // El content-type y el cache-control los fijó el PUT del renderer (HTML corto,
    // assets inmutables, sw.js sin caché), así que se reenvían tal cual.
    const headers = new Headers();
    storedObject.writeHttpMetadata(headers);
    headers.set('etag', storedObject.httpEtag);

    const response = new Response(isCacheable ? storedObject.body : null, { headers });
    if (isCacheable) ctx.waitUntil(cache.put(request, response.clone()));
    return response;
  },
};

// Una navegación se sirve como <ruta>/index.html —así lo escribe el renderer, un documento
// por página del builder—. Los archivos del origen (sw.js, favicon.ico) se piden por su
// nombre exacto, y lo que los distingue es que el último segmento lleva extensión.
// Cualquier ruta sin HTML publicado es un 404: el storefront no tiene fallback de SPA,
// cada página navegable se prerenderiza (ver getCompanyWebpagePages en el backend).
function buildObjectKey(requestUrl) {
  const hostname = requestUrl.hostname.toLowerCase();
  const requestedPath = requestUrl.pathname.replace(/\/+$/, '');
  const isFileRequest = (requestedPath.split('/').at(-1) || '').includes('.');
  return `${HTML_KEY_ROOT}/${hostname}${isFileRequest ? requestedPath : `${requestedPath}/index.html`}`;
}
