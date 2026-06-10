#!/usr/bin/env bun
// Per-company storefront prerender for Cloudflare Pages.
//
// Sets VITE_COMPANY_ID (which flips the app to SSR+prerender, scopes API calls via
// Env.getCompanyID, and switches the base path to '' for a subdomain-root deploy),
// runs the SvelteKit build (prerenders the root page → HTML + CSS + JS baked into
// build/), then copies the output to a per-company dist folder.
//
// Usage:
//   bun scripts/prerender.mjs --company <id> [--out <dir>]
//
// The build-time content/SEO fetch targets the configured API (PUBLIC_ENDPOINTS in
// .env → the dev API). The public GET.p-webpage endpoint must be deployed there for
// real content to bake in; if the fetch fails the build still succeeds and the
// client fills content in at view time.

import { spawnSync } from 'node:child_process';
import { cpSync, rmSync, existsSync, readdirSync, statSync } from 'node:fs';
import { resolve, dirname, relative } from 'node:path';
import { fileURLToPath } from 'node:url';

const args = process.argv.slice(2);
const getFlag = (name) => {
  const index = args.indexOf(name);
  return index >= 0 ? args[index + 1] : undefined;
};

const company = getFlag('--company');
if (!company || Number.isNaN(Number(company)) || Number(company) <= 0) {
  console.error('Error: --company <id> is required (a positive number).');
  process.exit(1);
}

const here = dirname(fileURLToPath(import.meta.url));
const appDir = resolve(here, '..');           // frontend/webpage
const buildDir = resolve(appDir, 'build');
const outDir = resolve(getFlag('--out') || resolve(appDir, 'dist-prerender', String(company)));

console.log(`[prerender] company=${company}`);
console.log(`[prerender] out=${outDir}`);

const env = { ...process.env, VITE_COMPANY_ID: String(company) };
const result = spawnSync('bun', ['run', 'build'], { cwd: appDir, env, stdio: 'inherit' });
if (result.status !== 0) {
  console.error('[prerender] build failed');
  process.exit(result.status || 1);
}
if (!existsSync(buildDir)) {
  console.error(`[prerender] expected build output missing: ${buildDir}`);
  process.exit(1);
}

rmSync(outDir, { recursive: true, force: true });
cpSync(buildDir, outDir, { recursive: true });

// Size summary — raw byte sizes of the emitted .js / .css (no gzip; fast stat-only
// walk). Helps spot bundle bloat per build.
const collectAssets = (dir) => {
  const assets = [];
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = resolve(dir, entry.name);
    if (entry.isDirectory()) {
      assets.push(...collectAssets(full));
    } else if (/\.(js|css)$/.test(entry.name)) {
      assets.push({ path: relative(outDir, full), size: statSync(full).size });
    }
  }
  return assets;
};

const formatKB = (bytes) => `${(bytes / 1024).toFixed(1)} KB`;
const assets = collectAssets(outDir).sort((a, b) => b.size - a.size);
const totals = assets.reduce(
  (acc, a) => {
    const kind = a.path.endsWith('.css') ? 'css' : 'js';
    acc[kind] += a.size;
    acc.all += a.size;
    return acc;
  },
  { js: 0, css: 0, all: 0 },
);
const pathWidth = assets.reduce((w, a) => Math.max(w, a.path.length), 0);

console.log('\n[prerender] asset sizes (.js / .css):');
for (const a of assets) {
  console.log(`  ${a.path.padEnd(pathWidth)}  ${formatKB(a.size).padStart(10)}`);
}
console.log(`  ${'-'.repeat(pathWidth)}  ${'-'.repeat(10)}`);
const jsCount = assets.filter((a) => a.path.endsWith('.js')).length;
const cssCount = assets.length - jsCount;
console.log(`  JS  (${jsCount} files): ${formatKB(totals.js)}`);
console.log(`  CSS (${cssCount} files): ${formatKB(totals.css)}`);
console.log(`  Total          : ${formatKB(totals.all)}`);

console.log('\n[prerender] done.');
console.log(`\nDeploy with:\n  wrangler pages deploy ${outDir}`);
