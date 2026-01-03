import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

console.log('--- SVELTE CONFIG LOADED ---');

/** @type {import('@sveltejs/kit').Config} */
const config = {
	// Consult https://svelte.dev/docs/kit/integrations
	// for more information about preprocessors
	preprocess: vitePreprocess(),
	compilerOptions: {
		hmr: false,
		cssHash: ({ hash, css, name, filename }) => {
			const componentName = filename
				? filename.split(/[\\/]/).pop().split('.')[0]
				: 'comp';
			return `${componentName}_${hash(css).substring(0,4)}`;
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
		alias: {
			$components: './src/components',
			$core: './src/core',
			$shared: './src/shared'
		}
	}
};

export default config;
