import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig, type ViteDevServer } from 'vite';
import * as esbuild from 'esbuild'
import type { BuildOptions } from 'esbuild'
import fs from 'fs'
import mime from 'mime-types'
import path from 'path'
import { fileURLToPath } from 'url';
import type { IncomingMessage, ServerResponse } from 'http'
import { createHash } from 'node:crypto';
import { type RolldownOptions } from 'rolldown'
import { svelteClassHasher, getCounterForKey, makeClassKey } from './plugins.js';

// Get __dirname equivalent for ES modules
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Get project directory
const projectDir = process.cwd()

const isBuild = process.argv.includes('build');
// Keep all production build artifacts under the same minification rule.
const shouldMinifyBuildOutput = isBuild;

const makeDevCssModuleClass = (name: string, filename: string) => {
  // Keep CSS module classes stable in dev so repeated selectors export one class name.
  const stableInput = path.relative(projectDir, filename) + ':' + name;
  const stableHash = createHash('sha256').update(stableInput).digest('base64url').slice(0, 6);
  return `m-${name}_${stableHash}`;
};

// Custom plugin to build the service worker
const publicDir = path.resolve(projectDir, 'static');

const serviceWorkerConfig: BuildOptions = {
  entryPoints: [path.resolve(projectDir, 'packages/genix-ui/service-worker/service-worker.ts')],
  format: 'esm', // Service workers typically use ES modules
  outfile: path.resolve(publicDir, 'sw.js'),
  bundle: true,
  sourcemap: true,
  splitting: false,
  platform: 'browser',
  packages: 'bundle' as const,
  target: 'esnext',
}

const serviceWorkerPlugin = () => ({
  name: 'build-service-worker',
  async buildStart() {
    // En dev lo construye configureServer (que además lo deja en watch), así que
    // hacerlo aquí también compilaba el SW dos veces por arranque.
    if (!isBuild) { return }
    console.log("build start: service worker::")
    // Ensure the output directory exists

    await esbuild.build({
      ...serviceWorkerConfig,
      minify: shouldMinifyBuildOutput,
    }).catch(() => process.exit(1));
  },
  // In dev mode, we can also watch the service worker file for changes
  configureServer(server: ViteDevServer) {

    const buildSw = async () => {
      console.log(`[Service Worker] Rebuilding service-worker.ts to public/sw.js...`);
      await esbuild.build({ ...serviceWorkerConfig })
        .catch((err) => console.error('[Service Worker] Build failed:', err));
    };

    // Initial build of SW when dev server starts
    buildSw();

    // Rebuild when the package worker shell or cache engine changes.
    const serviceWorkerSourceDir = path.resolve(projectDir, 'packages/genix-ui/service-worker');
    const cacheSourceDir = path.resolve(projectDir, 'packages/genix-ui/cache');
    server.watcher.add([serviceWorkerSourceDir, cacheSourceDir]);
    server.watcher.on('change', async (filePath: string) => {
      if (filePath.startsWith(serviceWorkerSourceDir) || filePath.startsWith(cacheSourceDir)) {
        await buildSw();
        // server.hot.send({ type: 'full-reload' });
      }
    });

    // Custom middleware to serve files from the public directory directly
    server.middlewares.use((req: IncomingMessage, res: ServerResponse, next: () => void) => {
      // Serve WASM files with correct MIME type
      if (req.url && req.url.endsWith('.wasm')) {
        res.setHeader('Content-Type', 'application/wasm');
        res.setHeader('Cross-Origin-Embedder-Policy', 'require-corp');
        res.setHeader('Cross-Origin-Opener-Policy', 'same-origin');
      }

      // This helps prevent intercepting routes that should be handled by the SPA
      if (req.url && req.url.startsWith('/sw.js')) {
        const filePath = path.join(publicDir, req.url);
        // Check if the file exists in the public directory
        if (fs.existsSync(filePath) && fs.lstatSync(filePath).isFile()) {
          const contentType = mime.lookup(filePath) || 'application/octet-stream';
          res.setHeader('Content-Type', contentType);
          fs.createReadStream(filePath).pipe(res);
          return; // Stop processing this request
        }
      }
      next(); // Pass to the next middleware if not handled
    });
  },
});

export default defineConfig({
  root: path.resolve(__dirname),
  define: {
    'global': 'globalThis'
  },
  server: {
    port: 3570, // Change this to your desired port
    // No HMR: a save must leave the open page untouched. Vite pushes nothing, so neither the
    // component subtree nor its services are rebuilt — pick up the change with a manual refresh.
    // vite-plugin-svelte reads this and turns compilerOptions.hmr off on its own.
    hmr: false,
    fs: {
      strict: false,
      allow: [
        path.resolve(__dirname),
        path.resolve(__dirname, '..'),
      ],
    },
    headers: {
      'Cross-Origin-Embedder-Policy': 'require-corp',
      'Cross-Origin-Opener-Policy': 'same-origin',
    }
  },
  cacheDir: 'node_modules/.vite',
  css: {
    modules: {
      generateScopedName: (name, filename, _css) => {
        // Deterministic, persisted, keyed minified name (see plugins.js). Same
        // file:name -> same class across runs, processes and the prerender's two passes.
        if (isBuild) {
          return getCounterForKey(makeClassKey('m', filename, name));
        }
        return makeDevCssModuleClass(name, filename);
      }
    }
  },
  build: {
    minify: shouldMinifyBuildOutput,
    cssMinify: shouldMinifyBuildOutput,
    rollupOptions: {
      cache: true,
      output: {
        // This tries to keep the IDs based on content rather than index
        hashCharacters: 'base64',
      }
    } as RolldownOptions
  },
  optimizeDeps: {
    exclude: ['@jsquash/avif'],
  },
  worker: {
    format: 'es',
    plugins: () => [tailwindcss()],
  },
  plugins: [
    sveltekit({
      preprocess: vitePreprocess(),
      compilerOptions: {
        cssHash: ({ hash, css, filename }) => {
          if (isBuild) {
            // Deterministic keyed name; keyed by file (or css hash when filename
            // is absent) so both prerender passes resolve the same scope class.
            return getCounterForKey(makeClassKey('s', filename, filename ? undefined : '#' + hash(css)));
          }
          if (!filename) {
            return `svelte-${hash(css).substring(0, 8)}`;
          }
          // Readable dev scope class: the sanitized component name plus the css hash.
          const componentName = (filename.split(/[\\/]/).pop() || '')
            .split('.')[0]
            .replace(/^\+/, '')
            .replace(/[^a-zA-Z0-9_-]/g, '_')
            .replace(/^[0-9]/, '_$&');
          return `${componentName || 'comp'}_${hash(css).substring(0, 8)}`;
        }
      },
      // Static adapter in SPA mode: every route falls back to index.html.
      adapter: adapter({
        pages: 'build',
        assets: 'build',
        fallback: 'index.html',
        precompress: false,
        strict: true
      }),
      // No src/ folder: the app's folders (and the env.ts entry) sit at the frontend root.
      // Path aliases are the "#…" subpath imports in package.json.
      files: { src: '.', assets: 'static', routes: 'routes', appTemplate: 'app.html' },
      prerender: { handleHttpError: 'warn' }
    }),
    isBuild && svelteClassHasher(),
    tailwindcss(),
    serviceWorkerPlugin()
  ].filter(x => x)
});
