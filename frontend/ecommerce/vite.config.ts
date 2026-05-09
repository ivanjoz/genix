import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';
import path from 'path';
import { svelteClassHasher, getCounter, getCounterFomFile } from '../plugins.js';
import { fileURLToPath } from 'url';
import { createHash } from 'node:crypto';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const isBuild = process.argv.includes('build');
// Keep store chunks minified whenever Vite is running a build.
const shouldMinifyBuildOutput = isBuild;
const cssModuleMap = new Map();

if (isBuild) {
  getCounterFomFile();
}

const makeDevCssModuleClass = (name: string, filename: string) => {
  // Keep CSS module classes stable in dev so repeated selectors export one class name.
  const stableInput = path.relative(__dirname, filename) + ':' + name;
  const stableHash = createHash('sha256').update(stableInput).digest('base64url').slice(0, 6);
  return `m-${name}_${stableHash}`;
};

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
        return makeDevCssModuleClass(name, filename);
      }
    }
  },
  build: {
    minify: shouldMinifyBuildOutput,
    cssMinify: shouldMinifyBuildOutput,
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
