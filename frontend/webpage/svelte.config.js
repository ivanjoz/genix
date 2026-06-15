import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import { getCounterForKey, makeClassKey } from '../plugins.js';

const isBuild = process.argv.includes('build');

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
			handleHttpError: 'warn',
			// Default ['*'] crawls every non-dynamic route. The --page-base build
			// (VITE_PRERENDER_BASE) renders ONLY the /base shell, so restrict the crawl.
			entries: process.env.VITE_PRERENDER_BASE ? ['/base'] : ['*']
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
