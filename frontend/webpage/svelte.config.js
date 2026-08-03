import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import { getCounterForKey, makeClassKey } from '../plugins.js';

const isBuild = process.argv.includes('build');
// Build del renderer (scripts/build-renderer.mjs): emite el bundle SSR + los assets que
// el Lambda usa para renderizar CUALQUIER company. No prerenderiza nada: cada página se
// renderiza bajo demanda, así que aquí no hay tenant ni contenido que hornear.
const isRendererBuild = !!process.env.VITE_RENDERER_BUILD;

console.log('--- SVELTE CONFIG LOADED (pkg-store) ---');

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: vitePreprocess(),
	compilerOptions: {
		hmr: false,
		cssHash: ({ hash, css, name, filename }) => {
			// MUST be deterministic: SSR/prerender runs two separate build passes
			// (server + client). The persisted keyed counter (../plugins.js) resolves the
			// same key to the same name in both passes, so the prerendered HTML's scope
			// class matches the bundled CSS. Dev keeps readable component-name hashes.
			if (isBuild) {
				return getCounterForKey(makeClassKey('s', filename, filename ? undefined : '#' + hash(css)));
			}
			if (!filename) {
				return `svelte-${hash(css).substring(0, 8)}`;
			}
			const fileNamePart = filename.split(/[\\/]/).pop();
			if (!fileNamePart) {
				return `svelte-${hash(css).substring(0, 8)}`;
			}
			const componentName = fileNamePart
				.split('.')[0]
				.replace(/^\+/, '')
				.replace(/[^a-zA-Z0-9_-]/g, '_')
				.replace(/^[0-9]/, '_$&');
			const safeName = componentName || 'comp';
			return `${safeName}_${hash(css).substring(0, 8)}`;
		}
	},
	kit: {
		adapter: adapter({
			pages: 'build',
			assets: 'build',
			// In the prerender build the root is written to index.html (with content), so
			// the SPA fallback must NOT be index.html or it would overwrite that. 404.html
			// is what Cloudflare Pages serves for unmatched paths and still boots the SPA.
			fallback: process.env.VITE_COMPANY_ID ? '404.html' : 'index.html',
			precompress: false,
			strict: true
		}),
		paths: {
			// The per-company prerender build (VITE_COMPANY_ID set) deploys at the
			// subdomain root; dev/admin keep the /webpage-app base for the :3572 proxy.
			base: process.env.VITE_COMPANY_ID || isRendererBuild ? '' : '/webpage-app',
			// Con rutas relativas SvelteKit emite './_app/…' en la raíz y '../_app/…' en
			// una página anidada, así que el prefijo dependería de la profundidad de cada
			// página. El Lambda reescribe ese prefijo al CDN de la company con UNA regla,
			// así que necesita que sea siempre el mismo: '/_app/…'.
			relative: !isRendererBuild
		},
		files: {
			assets: 'static',
			// El proyecto no usa src/, así que el hook vive en la raíz de la app.
			hooks: { server: 'hooks.server' },
			lib: 'lib',
			routes: 'routes',
			appTemplate: 'app.html'
		},
		alias: {
			$domain: '../domain-components',
			$ecommerce: './',
			$stores: './stores',
			$routes: './routes',
			$components: '../packages/genix-ui',
			$core: '../core',
			$services: '../services',
			$libs: '../libs',
			$lib: './lib'
		},
		prerender: {
			handleHttpError: 'warn',
			// Sin crawl: [...path] hace match con CUALQUIER enlace interno, así que el
			// crawler prerenderizaría rutas como /shop con el contenido de la raíz. Cada
			// página se renderiza explícitamente (aquí por 'entries', y en producción una
			// invocación por página al Lambda).
			crawl: false,
			// '*' solo cubre rutas NO dinámicas, y la página de la tienda vive en el
			// catch-all [...path] (una ruta por página del builder), así que la raíz hay
			// que declararla a mano o no se prerenderiza nada.
			// El build --page-base (VITE_PRERENDER_BASE) renderiza SOLO el shell /base.
			// El build del renderer no prerenderiza nada: el Lambda renderiza bajo demanda.
			entries: isRendererBuild ? [] : process.env.VITE_PRERENDER_BASE ? ['/base'] : ['*', '/']
		},
		output: {
			// 'split' enables code-splitting so vendor (node_modules) and app code land
			// in separate chunks (see manualChunks in vite.config.ts). 'single' would
			// reject manualChunks outright (codeSplitting:false).
			bundleStrategy: 'split'
		}
	}
};

export default config;
