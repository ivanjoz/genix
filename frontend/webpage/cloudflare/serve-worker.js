// Pre-built deploy artifact: the hand-stripped JS of src/serve-worker.ts (no
// imports, only TypeScript type annotations were removed, so behavior is
// identical). Shipped as-is to Cloudflare by scripts/cloudflare_deploy.go — the
// Go deployer reads this file verbatim, so no esbuild/wrangler/node step is
// needed. Keep this in sync with src/serve-worker.ts (used for `wrangler dev`).
const navigationAssetPath = '/index.html';

export default {
  async fetch(request, env) {
    // Storefront artifacts are read-only.
    if (request.method !== 'GET' && request.method !== 'HEAD') {
      return new Response('Method Not Allowed', {
        status: 405,
        headers: { Allow: 'GET, HEAD' },
      });
    }

    const requestUrl = new URL(request.url);
    const hostname = requestUrl.hostname.toLowerCase();
    const requestedFileName = requestUrl.pathname.split('/').at(-1) ?? '';
    const requestedPath =
      requestUrl.pathname === '/' || !requestedFileName.includes('.')
        ? navigationAssetPath
        : requestUrl.pathname;

    // The hostname is the tenant key, so no runtime company lookup is needed.
    const assetUrl = new URL(`/${hostname}${requestedPath}`, requestUrl.origin);
    const response = await env.ASSETS.fetch(new Request(assetUrl, request));

    if (response.status === 404) {
      console.warn('[serve-worker] asset not found', { hostname, requestedPath });
    }

    return response;
  },
};
