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
import { svelteClassHasher } from './plugins.js';

// Custom plugin to build the service worker
const __dirname = process.cwd()
const publicDir = path.resolve(__dirname, 'static');

const serviceWorkerConfig: BuildOptions = {
  entryPoints: [path.resolve(__dirname, 'src/workers/service-worker.ts')],
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
    server.watcher.add(path.resolve(__dirname, 'src/workers/service-worker.ts'));
    server.watcher.on('change', async (filePath: string) => {
      if (filePath === path.resolve(__dirname, 'src/workers/service-worker.ts')) {
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
  server: {
    port: 3570, // Change this to your desired port
    headers: {
      'Cross-Origin-Embedder-Policy': 'require-corp',
      'Cross-Origin-Opener-Policy': 'same-origin',
    }
  },
  cacheDir: 'node_modules/.vite',
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
  plugins: [svelteClassHasher(), sveltekit(), tailwindcss(), serviceWorkerPlugin()]
});

