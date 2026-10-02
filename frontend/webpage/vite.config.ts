import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig, type AliasOptions, type ViteDevServer } from 'vite';
import * as esbuild from 'esbuild';
import type { BuildOptions } from 'esbuild';
import path from 'path';
import { fileURLToPath } from 'url';
import { createHash } from 'node:crypto';
import { svelteClassHasher, getCounterForKey, makeClassKey } from '../plugins.js';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
// The shared service-worker source and the shared layers (#core, #libs, ...) live in the
// parent frontend root, not in webpage/.
const frontendDir = path.resolve(__dirname, '..');

const isBuild = process.argv.includes('build');
// Keep store chunks minified whenever Vite is running a build.
const shouldMinifyBuildOutput = isBuild;

// Build the FULL package RPC service worker into
// static/sw.js — the same worker the admin app uses. The storefront's data layer
// registers /sw.js at scope '/' and drives it over a
// MessageChannel RPC ("Action 3" = delta-cache fetch); a worker without that
// `message` handler makes every product fetch time out. SvelteKit's build copies
// static/ verbatim, so emitting here covers both `vite dev` and the prerender.
const publicDir = path.resolve(__dirname, 'static');
const serviceWorkerConfig: BuildOptions = {
  entryPoints: [path.resolve(frontendDir, 'packages/genix-ui/service-worker/service-worker.ts')],
  format: 'esm',
  outfile: path.resolve(publicDir, 'sw.js'),
  bundle: true,
  // Sourcemaps cost build time and aren't shipped/used by the prerender deploy; keep
  // them only for `vite dev` (see configureServer, which omits this flag override).
  sourcemap: !isBuild,
  splitting: false,
  platform: 'browser',
  packages: 'bundle',
  target: 'esnext',
};

const serviceWorkerPlugin = () => ({
  name: 'build-store-service-worker',
  async buildStart() {
    await esbuild
      .build({ ...serviceWorkerConfig, minify: shouldMinifyBuildOutput })
      .catch(() => process.exit(1));
  },
  configureServer(server: ViteDevServer) {
    const buildSw = async () => {
      await esbuild
        .build({ ...serviceWorkerConfig })
        .catch((err) => console.error('[Store SW] Build failed:', err));
    };
    buildSw();
    const swSourceDir = path.resolve(frontendDir, 'packages/genix-ui/service-worker');
    const cacheSourceDir = path.resolve(frontendDir, 'packages/genix-ui/cache');
    server.watcher.add([swSourceDir, cacheSourceDir]);
    server.watcher.on('change', async (filePath: string) => {
      if (filePath.startsWith(swSourceDir) || filePath.startsWith(cacheSourceDir)) await buildSw();
    });
  },
});

const makeDevCssModuleClass = (name: string, filename: string) => {
  // Keep CSS module classes stable in dev so repeated selectors export one class name.
  const stableInput = path.relative(__dirname, filename) + ':' + name;
  const stableHash = createHash('sha256').update(stableInput).digest('base64url').slice(0, 6);
  return `m-${name}_${stableHash}`;
};

// El build del renderer (scripts/build-renderer.mjs) es el que se publica como tienda.
// Solo en él se cambia DOMPurify (~50 KB) por un stub mínimo: se alcanza únicamente desde
// getPageContent() del agente de UI (genix-ui/agent/registry.ts), un camino que la tienda
// pública nunca ejecuta. Los builds admin/dev conservan la librería real.
const isRendererBuild = !!process.env.VITE_RENDERER_BUILD;

export default defineConfig({
  root: path.resolve(__dirname),
  resolve: {
    alias: isRendererBuild
      ? {
          dompurify: path.resolve(__dirname, 'lib/dompurify-stub.js'),
          'supports-color': path.resolve(__dirname, 'lib/supports-color-stub.js')
        }
      : {} as AliasOptions
  },
  environments: {
    // The server chunks are only consumed by scripts/build-renderer.mjs, whose esbuild pass
    // minifies the final render.mjs. Minifying them here first is harmful: rolldown can name a
    // compiled snippet (a block-level `{function t(){}}`) like the component's props param,
    // and esbuild hoists that block function over the param, so the props read as undefined.
    ssr: { build: { minify: false } }
  },
  ssr: {
    // El build del renderer se empaqueta en un zip que corre sin node_modules (/tmp en el
    // VPS, /var/task en el Lambda), así que nada puede quedar como dependencia externa.
    //
    // Por defecto Vite externaliza las dependencias en SSR, y las CJS las deja como llamadas
    // a un createRequire propio. esbuild no puede seguir esas llamadas — para él son un
    // require() de usuario, no un import — así que sobreviven al bundle y el paquete se busca
    // en tiempo de ejecución, donde no existe. Así se coló mime-types (axios -> form-data).
    noExternal: isRendererBuild ? true : undefined
  },
  server: {
    port: 3571,
    fs: {
      strict: false,
      allow: [path.resolve(__dirname), path.resolve(__dirname, '..')]
    }
  },
  css: {
    modules: {
      // Build: deterministic, persisted, keyed counter (shared with the admin build via
      // ../plugins.js). Same file:name -> same class across BOTH prerender passes, so the
      // prerendered HTML and the bundled CSS agree. Dev keeps readable sha256 names.
      generateScopedName: (name, filename, _css) =>
        isBuild ? getCounterForKey(makeClassKey('m', filename, name)) : makeDevCssModuleClass(name, filename)
    }
  },
  build: {
    // The prerender ships hydration JS for modern browsers and is minified anyway, so
    // skip Vite 8's default 'baseline-widely-available' downlevel transpilation pass.
    target: 'esnext',
    minify: shouldMinifyBuildOutput,
    cssMinify: shouldMinifyBuildOutput,
    // Skip the gzip-size pass in the build reporter — it's pure CPU on every emitted
    // chunk and the prerender doesn't need it (our script prints raw sizes instead).
		reportCompressedSize: false,
    rollupOptions: {
      output: {
        hashCharacters: 'base64',
        // bundleStrategy 'split' (sveltekit() options above) enables code-splitting, so we can
        // separate dependencies from app code: everything under node_modules goes into
        // a single 'vendor' chunk, the rest stays in app/route chunks.
        manualChunks(id) {
          if (id.includes('node_modules')) return 'vendor';
        }
      }
    }
  },
  plugins: [
    serviceWorkerPlugin(),
    // svelteClassHasher now uses the deterministic, persisted keyed counter in
    // ../plugins.js, so it no longer diverges across the SSR + client build passes —
    // the reason it was previously disabled here. (The old commented-`.relative`
    // style-scan bug is fixed too: CSS comments are stripped before scanning.) Build only.
    isBuild && svelteClassHasher(),
    // SvelteKit's `vite-plugin-sveltekit-guard` runs a `resolveId` hook on EVERY import
    // edge to build an import-map, used only to print a nice chain if a server-only
    // module ($lib/server, $env/*/private, *.server.*) leaks into client code. Under
    // rolldown each edge crosses the JS↔Rust boundary and it dominates plugin time
    // (PLUGIN_TIMINGS ~82%). This storefront prerender imports no server-only modules,
    // so we drop the guard. If server-only code is ever added, restore it to catch leaks.
    (async () => {
      const sk = await sveltekit({
        preprocess: vitePreprocess(),
        compilerOptions: {
          cssHash: ({ hash, css, filename }) => {
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
            const componentName = (filename.split(/[\\/]/).pop() || '')
              .split('.')[0]
              .replace(/^\+/, '')
              .replace(/[^a-zA-Z0-9_-]/g, '_')
              .replace(/^[0-9]/, '_$&');
            return `${componentName || 'comp'}_${hash(css).substring(0, 8)}`;
          }
        },
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
        // El proyecto no usa src/, así que el hook y env.ts viven en la raíz de la app. Los
        // alias de ruta son los subpath imports "#…" de package.json.
        files: {
          src: '.',
          assets: 'static',
          hooks: { server: 'hooks.server' },
          routes: 'routes',
          appTemplate: 'app.html'
        },
        output: {
          // 'split' enables code-splitting so vendor (node_modules) and app code land
          // in separate chunks (see manualChunks below). 'single' would
          // reject manualChunks outright (codeSplitting:false).
          bundleStrategy: 'split'
        }
      });
      const arr = Array.isArray(sk) ? sk : [sk];
      return arr.filter((p) => p && (p as any).name !== 'vite-plugin-sveltekit-guard');
    })(),
    tailwindcss()
  ].filter(x => x)
});
