import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import { getCounter, getCounterFomFile } from '../plugins.js';

const isBuild = process.argv.includes('build');
const componentMap = new Map();

if (isBuild) {
	getCounterFomFile();
}

console.log('--- SVELTE CONFIG LOADED (pkg-store) ---');

/** @type {import('@sveltejs/kit').Config} */
const config = {
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
			fallback: 'index.html',
			precompress: false,
			strict: true
		}),
		paths: {
			base: isBuild ? '/store' : ''
		},
		files: {
			assets: 'static',
			lib: 'lib',
			routes: 'routes',
			appTemplate: 'app.html'
		},
		alias: {
			$ui: '../pkg-ui',
			$store: './',
			$stores: './stores',
			$routes: './routes',
			$components: '../pkg-components',
			$core: '../pkg-core',
			$services: '../pkg-services',
			$lib: './lib'
		},
		prerender: {
			handleHttpError: 'warn'
		}
	}
};

export default config;
