// Punto de entrada del bundle SSR que corre dentro del Lambda de render.
//
// scripts/build-renderer.mjs empaqueta este archivo con esbuild en un único `render.mjs`
// sin node_modules: SvelteKit deja su servidor en .svelte-kit/output/server (es lo mismo
// que consume cualquier adapter), pero repartido en decenas de chunks que importan al
// paquete @sveltejs/kit. Bundlearlo aquí es lo que permite meterlo en un zip de pocos MB.
import { Server } from '../.svelte-kit/output/server/index.js';
import { manifest } from '../.svelte-kit/output/server/manifest.js';

const server = new Server(manifest);
let serverInitialized = false;

// Renderiza UNA página de UNA company. El tenant y la página viajan en el query porque
// hooks.server.ts los lee de ahí (y el propio SvelteKit no tiene otro canal por request
// hacia los load() universales).
//
// IMPORTANTE: llamadas SECUENCIALES. hooks.server.ts fija el tenant sobre el singleton
// Env, así que dos renders simultáneos en el mismo proceso se pisarían.
export async function renderPage({ origin, path, companyID, pageID }) {
	if (!serverInitialized) {
		await server.init({ env: process.env });
		serverInitialized = true;
	}

	const url = new URL(path || '/', origin);
	url.searchParams.set('cid', String(companyID));
	url.searchParams.set('pid', String(pageID));

	const response = await server.respond(new Request(url), {
		getClientAddress: () => '127.0.0.1'
	});

	const html = await response.text();
	return {
		status: response.status,
		html: withPreloadTags(html, response.headers.get('link'))
	};
}

// SvelteKit solo inserta las pistas de precarga como <link> cuando prerenderiza; en una
// respuesta SSR las manda en la cabecera Link, contando con que un servidor las emita.
// Aquí el HTML acaba como archivo estático en R2, así que esa cabecera se perdería y el
// navegador descubriría los chunks en cascada (parsear → ejecutar el bootstrap →
// importar → importar…). Se reescriben como <link> en el <head>, que es exactamente lo
// que emitía el prerender anterior.
function withPreloadTags(html, linkHeader) {
	const tags = (linkHeader || '')
		.split(/,(?=\s*<)/)
		.map((entry) => entry.match(/^\s*<([^>]+)>[^,]*rel="modulepreload"/))
		.filter(Boolean)
		.map((match) => `<link rel="modulepreload" href="${match[1]}">`)
		.join('');

	return tags ? html.replace('</head>', `${tags}</head>`) : html;
}
