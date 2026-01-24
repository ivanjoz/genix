import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import { getCounter, getCounterFomFile } from './plugins.js';

const isBuild = process.argv.includes('build');
const componentMap = new Map();

if (isBuild) {
	getCounterFomFile();
}

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
				const key = filename || hash(css);
				if (!componentMap.has(key)) {
					componentMap.set(key, getCounter());
				}
				return componentMap.get(key);
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
			lib: 'lib',
			routes: 'routes',
			serviceWorker: 'workers/service-worker',
			appTemplate: 'app.html'
		},
		alias: {
			$lib: './lib',
			$components: './components',
			$core: './core',
			$shared: './shared',
			$ecommerce: './ecommerce',
			$services: './services'
		},
		prerender: {
			handleHttpError: 'warn'
		}
	}
};

export default config;
