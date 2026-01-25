import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig, type ViteDevServer } from 'vite';
import * as esbuild from 'esbuild'
import type { BuildOptions } from 'esbuild'
import fs from 'fs'
import mime from 'mime-types'
import path from 'path'
import type { IncomingMessage, ServerResponse } from 'http'
import { type RolldownOptions } from 'rolldown'
import { svelteClassHasher, getCounter, getCounterFomFile } from './plugins.js';

const isBuild = process.argv.includes('build');
const cssModuleMap = new Map<string, string>();

if (isBuild) {
  getCounterFomFile();
}

// Custom plugin to build the service worker
const __dirname = process.cwd()
const publicDir = path.resolve(__dirname, 'static');

const serviceWorkerConfig: BuildOptions = {
  entryPoints: [path.resolve(__dirname, 'pkg-core/workers/service-worker.ts')],
  format: 'esm', // Service workers typically use ES modules
  outfile: path.resolve(publicDir, 'sw.js'),
  bundle: true,
  sourcemap: true,
  splitting: false,
  platform: 'browser',
  packages: 'bundle' as const,
  target: 'esnext',
  plugins: [
    {
      name: 'alias',
      setup(build) {
        build.onResolve({ filter: /^\$/ }, args => {
          const parts = args.path.split('/');
          const alias = parts[0];
          const rest = parts.slice(1).join('/');
          const baseDir = {
    '$core': 'pkg-core',
    '$store': 'pkg-store',
    '$routes': 'routes',
    '$ui': 'pkg-ui',
    '$components': 'pkg-components',
    '$services': 'pkg-services'
  }[alias];

          if (!baseDir) return null;

          // For $core and $components, try multiple subdirectories
          let possiblePaths: string[] = [];
          if (alias === '$core') {
            possiblePaths.push(path.join(baseDir, 'lib', rest));
            possiblePaths.push(path.join(baseDir, 'core', rest));
            possiblePaths.push(path.join(baseDir, 'assets', rest));
            possiblePaths.push(path.join(baseDir, rest));
          } else if (alias === '$components') {
            possiblePaths.push(path.join(baseDir, rest));
            possiblePaths.push(path.join('pkg-ui/components', rest));
            possiblePaths.push(path.join('pkg-components/ecommerce', rest));
          } else {
            possiblePaths.push(path.join(baseDir, rest));
          }

          // Try each possible path until we find one that exists
          for (const possiblePath of possiblePaths) {
            let fullPath = path.resolve(__dirname, possiblePath);
            if (fs.existsSync(fullPath)) {
              return { path: fullPath };
            }
            if (fs.existsSync(fullPath + '.ts')) {
              return { path: fullPath + '.ts' };
            }
            if (fs.existsSync(fullPath + '.js')) {
              return { path: fullPath + '.js' };
            }
            if (fs.existsSync(fullPath + '.svelte')) {
              return { path: fullPath + '.svelte' };
            }
            if (fs.existsSync(path.join(fullPath, 'index.ts'))) {
              return { path: path.join(fullPath, 'index.ts') };
            }
            if (fs.existsSync(path.join(fullPath, 'index.js'))) {
              return { path: path.join(fullPath, 'index.js') };
            }
          }

          return null;
        });
      },
    },
  ],
}

const serviceWorkerPlugin = () => ({
  name: 'build-service-worker',
  async buildStart() {
    console.log("build start: service worker::")
    // Ensure the output directory exists

    await esbuild.build({
      ...serviceWorkerConfig,
      minify: process.env.NODE_ENV === 'production',
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

    // Watch for changes in the service worker source file
    server.watcher.add(path.resolve(__dirname, 'pkg-core/workers/service-worker.ts'));
    server.watcher.on('change', async (filePath: string) => {
      if (filePath === path.resolve(__dirname, 'pkg-core/workers/service-worker.ts')) {
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
  publicDir: 'static',
  server: {
    port: 3570, // Change this to your desired port
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
        if (isBuild) {
          const key = `${filename}:${name}`;
          if (!cssModuleMap.has(key)) {
            cssModuleMap.set(key, getCounter());
          }
          return cssModuleMap.get(key)!;
        }
        return `m-${name}_${Math.random().toString(36).substring(2, 6)}`;
      }
    }
  },
  build: {
    // This disables minification for the entire build
    minify: false,
    // You can also disable CSS minification specifically if needed
    cssMinify: false,
    rollupOptions: {
      cache: true,
      output: {
        // This tries to keep the IDs based on content rather than index
        hashCharacters: 'base64',
        manualChunks: (id) => {
          // Separate admin and store chunks to prevent loading admin code on store routes
          if (id.includes('/admin/') || id.includes('/cms/') || id.includes('/operaciones/')) {
            return 'admin';
          }
          if (id.includes('/ecommerce/') || id.includes('/store/')) {
            return 'store';
          }
          // Shared code goes to common chunk
          if (id.includes('/components/') || id.includes('/lib/') || id.includes('/core/')) {
            return 'shared';
          }
          // Everything else goes to vendor
          return 'vendor';
        },
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
    sveltekit(),
    isBuild && svelteClassHasher(),
    tailwindcss(),
    serviceWorkerPlugin()
  ].filter(x => x)
});
