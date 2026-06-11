#!/usr/bin/env bun
// Per-company storefront prerender for the Cloudflare storefront Worker.
//
// Sets VITE_COMPANY_ID (which flips the app to SSR+prerender, scopes API calls via
// Env.getCompanyID, and switches the base path to '' for a subdomain-root deploy),
// runs the SvelteKit build (prerenders the root page → HTML + CSS + JS baked into
// build/), then copies the output to a per-company dist folder.
//
// Usage:
//   bun scripts/prerender.mjs --company <id> --asset-base <url> [--out <dir>]
//
// The build-time content/SEO fetch targets the configured API (PUBLIC_ENDPOINTS in
// .env → the dev API). The public GET.p-webpage endpoint must be deployed there for
// real content to bake in; if the fetch fails the build still succeeds and the
// client fills content in at view time.

import { spawnSync } from 'node:child_process';
import {
  cpSync,
  rmSync,
  existsSync,
  readdirSync,
  statSync,
  readFileSync,
  writeFileSync,
  renameSync,
} from 'node:fs';
import { resolve, dirname, relative, basename, extname } from 'node:path';
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
const assetBase = (getFlag('--asset-base') || '').replace(/\/+$/, '');
if (!assetBase.startsWith('https://')) {
  console.error('Error: --asset-base <https-url> is required.');
  process.exit(1);
}

const here = dirname(fileURLToPath(import.meta.url));
const appDir = resolve(here, '..');           // frontend/webpage
const buildDir = resolve(appDir, 'build');
const outDir = resolve(getFlag('--out') || resolve(appDir, 'dist-prerender', String(company)));

console.log(`[prerender] company=${company}`);
console.log(`[prerender] out=${outDir}`);
console.log(`[prerender] asset-base=${assetBase}`);

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

inlineBootstrap(outDir, assetBase);
flattenAssets(outDir);
rewriteAssetUrls(outDir, assetBase);
validateFlattenedAssetUrls(outDir, assetBase);

// --- Inline the SvelteKit bootstrap entries into the prerendered HTML -----------
//
// With bundleStrategy 'split', SvelteKit boots via two tiny external modules:
//   entry/start.<hash>.js (~0.1 KB, re-exports start/load_css from the runtime chunk)
//   entry/app.<hash>.js   (~3 KB, the route table + the __vite_preload glue)
// Each is a separate HTTP request on the critical path. We inline both into the HTML
// and drop their files, eliminating two requests, while leaving the big vendor/route
// chunks external (they stay code-split + cacheable).
//
// Mechanism: the bootstrap is a CLASSIC script using dynamic import() + currentScript,
// so we can't paste the module bodies in directly (module scripts have no
// currentScript, and start/app share colliding minified import names). Instead we swap
//   import("./_app/immutable/entry/app.<hash>.js")
// for an import of a Blob built from the (rewritten) source:
//   import(URL.createObjectURL(new Blob([<src>], { type: "text/javascript" })))
// The entries reference siblings relatively (../chunks, ../nodes, ../assets). A blob:
// URL base is NOT hierarchical, so it can resolve neither those relative specifiers nor
// a root-absolute /_app/... one. Only a fully-qualified URL works from a blob module,
// so the prerender replaces those specifiers with the company's public R2 prefix.
//
// CSP note: importing a blob module needs `script-src blob:`. This app already ships
// inline classic scripts without a nonce, so no strict CSP is in force; if one is ever
// added, it must allow blob: (and keep allowing inline) or this step must be reverted.
function inlineBootstrap(dir, publicAssetBase) {
  const entryDir = resolve(dir, '_app/immutable/entry');
  if (!existsSync(entryDir)) {
    console.warn('[prerender] inlineBootstrap: no entry dir, skipping');
    return;
  }
  const escapeRe = (s) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  // Make embedded source safe to sit inside <script>…</script> and a JS string literal.
  const toJsString = (src) =>
    JSON.stringify(src).replace(/<\//g, '<\\/').replace(/<!--/g, '<\\!--');
  // Blob modules need absolute imports because blob: URLs have no hierarchical base.
  const rewriteSpecifiers = (src) =>
    src
      .replaceAll('../chunks/', `${publicAssetBase}/`)
      .replaceAll('../nodes/', `${publicAssetBase}/`)
      .replaceAll('../assets/', `${publicAssetBase}/`);

  const entryFiles = readdirSync(entryDir).filter((f) => /^(app|start)\..*\.js$/.test(f));
  if (entryFiles.length === 0) {
    console.warn('[prerender] inlineBootstrap: no app/start entries found, skipping');
    return;
  }
  const sources = Object.fromEntries(
    entryFiles.map((f) => [f, rewriteSpecifiers(readFileSync(resolve(entryDir, f), 'utf8'))]),
  );

  const htmlFiles = readdirSync(dir).filter((f) => f.endsWith('.html'));
  let inlinedAny = false;
  for (const name of htmlFiles) {
    const htmlPath = resolve(dir, name);
    let html = readFileSync(htmlPath, 'utf8');
    let changed = false;
    for (const [file, src] of Object.entries(sources)) {
      const esc = escapeRe(file);
      // Drop the modulepreload <link> for this entry (the file is about to vanish).
      const linkRe = new RegExp(`[ \\t]*<link[^>]*entry/${esc}"[^>]*>\\n?`, 'g');
      // Swap the dynamic import of the entry file for a blob import of its source.
      const importRe = new RegExp(`import\\("[^"]*entry/${esc}"\\)`, 'g');
      if (importRe.test(html)) {
        const blob = `import(URL.createObjectURL(new Blob([${toJsString(src)}],{type:"text/javascript"})))`;
        // Replacement MUST be a function: a string replacement would interpret `$$`,
        // `$&`, `$1`… in the inlined source as special patterns (e.g. corrupting
        // `$$slots` → `$slots`). A function's return value is inserted verbatim.
        html = html.replace(importRe, () => blob).replace(linkRe, '');
        changed = true;
      }
    }
    if (changed) {
      writeFileSync(htmlPath, html);
      inlinedAny = true;
    }
  }

  if (!inlinedAny) {
    console.warn('[prerender] inlineBootstrap: no bootstrap imports matched in HTML, skipping delete');
    return;
  }
  for (const f of entryFiles) rmSync(resolve(entryDir, f), { force: true });
  console.log(`[prerender] inlined bootstrap entries into HTML: ${entryFiles.join(', ')}`);
}

// Point every generated JS/CSS dependency at the company's public R2 prefix.
// sw.js remains same-origin because browsers reject cross-origin service workers.
function rewriteAssetUrls(dir, publicAssetBase) {
  const assetNames = readdirSync(dir).filter((name) => /\.(js|css)$/.test(name) && name !== 'sw.js');
  const escapeRe = (value) => value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  let rewrittenFiles = 0;

  for (const fileName of readdirSync(dir)) {
    if (!/\.(html|js|css)$/.test(fileName) || fileName === 'sw.js') continue;

    const filePath = resolve(dir, fileName);
    let content = readFileSync(filePath, 'utf8');
    const originalContent = content;
    for (const assetName of assetNames) {
      const escapedAssetName = escapeRe(assetName);
      const absoluteAssetUrl = `${publicAssetBase}/${assetName}`;
      content = content.replace(
        new RegExp(`(["'\`])(?:\\./|/)?${escapedAssetName}\\1`, 'g'),
        (_match, quote) => `${quote}${absoluteAssetUrl}${quote}`,
      );
    }
    if (content !== originalContent) {
      writeFileSync(filePath, content);
      rewrittenFiles++;
    }
  }

  console.log(
    `[prerender] rewrote ${assetNames.length} asset URL(s) to ${publicAssetBase} ` +
      `in ${rewrittenFiles} file(s)`,
  );
}

function validateFlattenedAssetUrls(dir, publicAssetBase) {
  const invalidAssetPrefix = `${publicAssetBase}/_app/immutable/`;
  for (const fileName of readdirSync(dir)) {
    if (!/\.(html|js|css)$/.test(fileName)) continue;
    const fileContent = readFileSync(resolve(dir, fileName), 'utf8');
    if (fileContent.includes(invalidAssetPrefix)) {
      throw new Error(
        `[prerender] unflattened CDN asset URL remains in ${fileName}: ${invalidAssetPrefix}`,
      );
    }
  }
  console.log('[prerender] validated flat CDN asset URLs');
}

// --- Flatten the deploy to a single folder of code -----------------------------
//
// Goal: ship ONLY .js / .css / .html, all in one flat directory, and discard
// everything else (images, fonts, favicon, sourcemaps, version.json …). Non-code
// assets are expected to be served from an external https:// host — the app already
// references its images that way (e.g. https://ivanjoz.github.io/genix-assets/...), so
// nothing here needs to point at them.
//
// SvelteKit emits code under _app/immutable/{chunks,nodes,assets}/ plus libs/, and the
// files cross-reference each other by those relative dirs. After moving every code file
// to the root we rewrite those references to be same-folder:
//   HTML:  ./_app/immutable/assets/X.css  → ./X.css
//          /_app/immutable/chunks/X.js     → /X.js          (404.html uses absolute)
//          libs/fontello-embedded.css      → fontello-embedded.css
//   JS:    import "../chunks/X.js"          → import "./X.js" (chunk→chunk "./X" already ok)
// CSS url() (fonts via /libs/…, fontello's legacy ../font/…) is left untouched: those
// are the discarded "other assets" the deploy expects to serve from https elsewhere.
function flattenAssets(dir) {
  const keepExt = new Set(['.js', '.css', '.html']);
  const isKeep = (name) => keepExt.has(extname(name).toLowerCase());

  // 1. Move every code file up to the root (collision-guarded).
  const seen = new Map(); // basename -> origin path
  const walk = (d) => {
    for (const e of readdirSync(d, { withFileTypes: true })) {
      const full = resolve(d, e.name);
      if (e.isDirectory()) {
        walk(full);
      } else if (isKeep(e.name)) {
        const target = resolve(dir, e.name);
        if (full === target) continue; // already at root
        if (seen.has(e.name)) {
          throw new Error(
            `[prerender] flatten collision: "${e.name}" from ${full} and ${seen.get(e.name)}`,
          );
        }
        seen.set(e.name, full);
        renameSync(full, target);
      }
    }
  };
  walk(dir);

  // 2. Drop everything else: subdirectories and stray non-code files at the root.
  let discarded = 0;
  for (const e of readdirSync(dir, { withFileTypes: true })) {
    const full = resolve(dir, e.name);
    if (e.isDirectory()) {
      rmSync(full, { recursive: true, force: true });
      discarded++;
    } else if (!isKeep(e.name)) {
      rmSync(full, { force: true });
      discarded++;
    }
  }

  // 3. Rewrite cross-references to the flat layout. Both HTML and JS reference the
  // moved files; CSS only references discarded assets (left for https).
  //
  // dirCollapse drops a `_app/immutable/<dir>/` or `libs/` directory prefix while
  // preserving any leading ./  ../  or / token. The lookbehind anchors
  // the match to a URL delimiter (quote/backtick/paren/space/=) so the short `libs`
  // token can't match mid-identifier (e.g. `mylibs/`). Covers HTML href/src and JS
  // string refs such as the layout's `libs/fontello-embedded.css`.
  const dirCollapse = (s) =>
    s.replace(
      /(?<=["'`(\s=])((?:\.\.?\/|\/)?)(?:_app\/immutable\/(?:chunks|nodes|assets|entry)|libs)\//g,
      (_m, lead) => lead || '',
    );
  // JS sibling imports use ../<dir>/ (e.g. a node importing ../chunks/X.js); flatten to
  // ./ . (chunk→chunk imports are already ./X.js and need no change.)
  const jsRelCollapse = (s) => s.replace(/\.\.\/(?:chunks|nodes|assets|entry)\//g, './');

  let htmlCount = 0;
  let jsCount = 0;
  for (const name of readdirSync(dir)) {
    const ext = extname(name).toLowerCase();
    const full = resolve(dir, name);
    if (ext === '.html') {
      writeFileSync(full, dirCollapse(readFileSync(full, 'utf8')));
      htmlCount++;
    } else if (ext === '.js') {
      writeFileSync(full, dirCollapse(jsRelCollapse(readFileSync(full, 'utf8'))));
      jsCount++;
    }
  }

  console.log(
    `[prerender] flattened to single folder: ${seen.size} code file(s) moved, ` +
      `${discarded} dir(s)/asset(s) discarded, refs rewritten in ${htmlCount} html + ${jsCount} js`,
  );
}

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
console.log(`\n[prerender] upload assets to ${assetBase} and deploy HTML with the storefront Worker`);
