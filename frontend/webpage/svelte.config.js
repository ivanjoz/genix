import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

console.log('--- SVELTE CONFIG LOADED (pkg-store) ---');

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: vitePreprocess(),
	compilerOptions: {
		hmr: false,
		cssHash: ({ hash, css, name, filename }) => {
			// MUST be deterministic: SSR/prerender runs two separate build passes
			// (server + client). A stateful counter (getCounter) diverges between them,
			// so the prerendered HTML's scope class wouldn't match the bundled CSS and
			// all scoped styles vanish. Hash the css content (stable across passes).
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
			base: process.env.VITE_COMPANY_ID ? '' : '/webpage-app'
		},
		files: {
			assets: 'static',
			lib: 'lib',
			routes: 'routes',
			appTemplate: 'app.html'
		},
		alias: {
			$domain: '../domain-components',
			$ecommerce: './',
			$stores: './stores',
			$routes: './routes',
			$components: '../ui-components',
			$core: '../core',
			$services: '../services',
			$libs: '../libs',
			$lib: './lib'
		},
		prerender: {
			handleHttpError: 'warn'
		},
		output: {
			bundleStrategy: 'single'
		}
	}
};

export default config;
