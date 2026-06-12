import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import { getCounterForKey, makeClassKey } from './plugins.js';
import path from 'path';

const isBuild = process.argv.includes('build');

console.log('--- SVELTE CONFIG LOADED ---');

/** @type {import('@sveltejs/kit').Config} */
const config = {
	// Consult https://svelte.dev/docs/kit/integrations
	// for more information about preprocessors
	preprocess: vitePreprocess(),
	compilerOptions: {
		hmr: false,
		cssHash: ({ hash, css, name, filename }) => {
			if (isBuild) {
				// Deterministic keyed name; keyed by file (or css hash when filename
				// is absent) so both prerender passes resolve the same scope class.
				return getCounterForKey(makeClassKey('s', filename, filename ? undefined : '#' + hash(css)));
			}

			if (!filename) {
				return `svelte-${hash(css).substring(0, 8)}`;
			}
			// Extract component name and sanitize it
			const componentName = filename
				.split(/[\\/]/)
				.pop()
				.split('.')[0]
				.replace(/^\+/, '') // Remove leading + from route files
				.replace(/[^a-zA-Z0-9_-]/g, '_') // Replace invalid chars with underscore
				.replace(/^[0-9]/, '_$&'); // Ensure it doesn't start with a number

			// Fallback if name is empty after sanitization
			const safeName = componentName || 'comp';
			return `${safeName}_${hash(css).substring(0, 8)}`;
		}
	},
	kit: {
		// Static adapter configured for SPA mode
		adapter: adapter({
			// default options are shown
			pages: 'build',
			assets: 'build',
			fallback: 'index.html', // This enables SPA mode - all routes fall back to index.html
			precompress: false,
			strict: true
		}),
		files: {
			assets: 'static',
			lib: 'pkg-core/lib',
			routes: 'routes',
			appTemplate: 'app.html'
		},
		alias: {
			$domain: path.resolve('./domain-components'),
			$ecommerce: path.resolve('./webpage'),
			$stores: path.resolve('./webpage/stores'),
			$routes: path.resolve('./routes'),
			$components: path.resolve('./ui-components'),
			$core: path.resolve('./core'),
			$services: path.resolve('./services'),
			$libs: path.resolve('./libs')
		},
		prerender: {
			handleHttpError: 'warn'
		}
	}
};

export default config;
