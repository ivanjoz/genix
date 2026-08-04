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
			// Ningún build prerenderiza páginas: el renderer las emite bajo demanda en el
			// Lambda y la vista embebida del builder es un SPA puro. Así que la salida es
			// siempre el shell SPA en index.html.
			fallback: 'index.html',
			precompress: false,
			strict: true
		}),
		paths: {
			// El renderer sirve la tienda en la raíz del dominio de la company; dev/admin
			// conservan la base /webpage-app del proxy en :3572.
			base: isRendererBuild ? '' : '/webpage-app',
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
		output: {
			// 'split' enables code-splitting so vendor (node_modules) and app code land
			// in separate chunks (see manualChunks in vite.config.ts). 'single' would
			// reject manualChunks outright (codeSplitting:false).
			bundleStrategy: 'split'
		}
	}
};

export default config;
