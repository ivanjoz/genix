import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';
import path from 'path';
import { svelteClassHasher, getCounter, getCounterFomFile } from '../plugins.js';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const isBuild = process.argv.includes('build');
const cssModuleMap = new Map();

if (isBuild) {
  getCounterFomFile();
}

export default defineConfig({
  root: path.resolve(__dirname),
  publicDir: './static',
  server: {
    port: 3571,
    fs: {
      strict: false,
      allow: [path.resolve(__dirname), path.resolve(__dirname, '..')]
    }
  },
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
    minify: false,
    cssMinify: false,
    rollupOptions: {
      output: {
        hashCharacters: 'base64',
        manualChunks: (id) => {
          if (id.includes('/ui-components/') || id.includes('/domain-components/') || id.includes('/core/') || id.includes('/libs/') || id.includes('/services/')) {
            return 'shared';
          }
          return 'vendor';
        }
      }
    }
  },
  plugins: [
    sveltekit(),
    isBuild && svelteClassHasher(),
    tailwindcss()
  ].filter(x => x)
});
