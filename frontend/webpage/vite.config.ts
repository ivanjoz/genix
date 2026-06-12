import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig, type AliasOptions, type ViteDevServer } from 'vite';
import * as esbuild from 'esbuild';
import type { BuildOptions } from 'esbuild';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import { createHash } from 'node:crypto';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
// The shared service-worker source and all $-aliased modules live in the parent
// frontend root, not in webpage/.
const frontendDir = path.resolve(__dirname, '..');

const isBuild = process.argv.includes('build');
// Keep store chunks minified whenever Vite is running a build.
const shouldMinifyBuildOutput = isBuild;

// Build the FULL RPC service worker (libs/workers/service-worker.ts) into
// static/sw.js — the same worker the admin app uses. The storefront's data layer
// (libs/sw-cache.ts) registers /sw.js at scope '/' and drives it over a
// MessageChannel RPC ("Action 3" = delta-cache fetch); a worker without that
// `message` handler makes every product fetch time out. SvelteKit's build copies
// static/ verbatim, so emitting here covers both `vite dev` and the prerender.
const publicDir = path.resolve(__dirname, 'static');
const serviceWorkerConfig: BuildOptions = {
  entryPoints: [path.resolve(frontendDir, 'libs/workers/service-worker.ts')],
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
  plugins: [
    {
      name: 'alias',
      setup(build) {
        build.onResolve({ filter: /^\$/ }, (args) => {
          const parts = args.path.split('/');
          const alias = parts[0];
          const rest = parts.slice(1).join('/');
          const baseDir = {
            '$core': 'core',
            '$ecommerce': 'webpage',
            '$routes': 'routes',
            '$domain': 'domain-components',
            '$components': 'ui-components',
            '$services': 'services',
            '$libs': 'libs',
          }[alias];
          if (!baseDir) return null;

          const possiblePaths: string[] = [];
          if (alias === '$core') {
            possiblePaths.push(path.join(baseDir, 'lib', rest));
            possiblePaths.push(path.join(baseDir, 'core', rest));
            possiblePaths.push(path.join(baseDir, 'assets', rest));
            possiblePaths.push(path.join(baseDir, rest));
          } else if (alias === '$components') {
            possiblePaths.push(path.join(baseDir, rest));
            possiblePaths.push(path.join('webpage', 'components', rest));
          } else {
            possiblePaths.push(path.join(baseDir, rest));
          }

          for (const possiblePath of possiblePaths) {
            const fullPath = path.resolve(frontendDir, possiblePath);
            if (fs.existsSync(fullPath)) return { path: fullPath };
            if (fs.existsSync(fullPath + '.ts')) return { path: fullPath + '.ts' };
            if (fs.existsSync(fullPath + '.js')) return { path: fullPath + '.js' };
            if (fs.existsSync(fullPath + '.svelte')) return { path: fullPath + '.svelte' };
            if (fs.existsSync(path.join(fullPath, 'index.ts'))) return { path: path.join(fullPath, 'index.ts') };
            if (fs.existsSync(path.join(fullPath, 'index.js'))) return { path: path.join(fullPath, 'index.js') };
          }
          return null;
        });
      },
    },
  ],
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
    const swSource = path.resolve(frontendDir, 'libs/workers/service-worker.ts');
    server.watcher.add(swSource);
    server.watcher.on('change', async (filePath: string) => {
      if (filePath === swSource) await buildSw();
    });
  },
});

const makeDevCssModuleClass = (name: string, filename: string) => {
  // Keep CSS module classes stable in dev so repeated selectors export one class name.
  const stableInput = path.relative(__dirname, filename) + ':' + name;
  const stableHash = createHash('sha256').update(stableInput).digest('base64url').slice(0, 6);
  return `m-${name}_${stableHash}`;
};

// The per-company storefront prerender (scripts/prerender.mjs) sets VITE_COMPANY_ID.
// In that build only, swap DOMPurify (~50 KB) for a tiny stub: it's reached solely from
// the UI agent's getPageContent() (ui-components/agent/registry.ts), a code path the
// public storefront never runs. Admin/dev builds keep the real library.
const isStorefrontPrerender = !!process.env.VITE_COMPANY_ID;

export default defineConfig({
  root: path.resolve(__dirname),
  publicDir: './static',
  resolve: {
    alias: isStorefrontPrerender
      ? { dompurify: path.resolve(__dirname, 'lib/dompurify-stub.js') }
      : {} as AliasOptions
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
      // Deterministic in every mode: SSR/prerender runs two build passes and a
      // stateful counter would assign different names per pass, breaking CSS-module
      // class matching between the prerendered HTML and the bundled CSS.
      generateScopedName: (name, filename, _css) => makeDevCssModuleClass(name, filename)
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
        // bundleStrategy 'split' (svelte.config.js) enables code-splitting, so we can
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
    // SvelteKit's `vite-plugin-sveltekit-guard` runs a `resolveId` hook on EVERY import
    // edge to build an import-map, used only to print a nice chain if a server-only
    // module ($lib/server, $env/*/private, *.server.*) leaks into client code. Under
    // rolldown each edge crosses the JS↔Rust boundary and it dominates plugin time
    // (PLUGIN_TIMINGS ~82%). This storefront prerender imports no server-only modules,
    // so we drop the guard. If server-only code is ever added, restore it to catch leaks.
    (async () => {
      const sk = await sveltekit();
      const arr = Array.isArray(sk) ? sk : [sk];
      return arr.filter((p) => p && (p as any).name !== 'vite-plugin-sveltekit-guard');
    })(),
    // NOTE: svelteClassHasher() is intentionally NOT used here. It rewrites style-block
    // class names via a stateful counter, which (a) diverges across the SSR + client
    // build passes prerender requires — breaking style matching — and (b) had an offset
    // bug that corrupted `.join(...)` calls. Svelte's deterministic cssHash already
    // scopes classes uniquely, so the extra minifier isn't needed for the storefront.
    tailwindcss()
  ].filter(x => x)
});
