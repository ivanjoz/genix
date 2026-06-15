#!/usr/bin/env bun
// Per-company storefront prerender for the Cloudflare storefront Worker.
//
// Sets VITE_COMPANY_ID (which flips the app to SSR+prerender, scopes API calls via
// Env.getCompanyID, and switches the base path to '' for a subdomain-root deploy),
// runs the SvelteKit build (prerenders the root page → HTML + CSS + JS baked into
// build/), then copies the output to a per-company dist folder.
//
// Usage:
//   bun scripts/prerender.mjs --company <id> [--asset-base <url>] [--out <dir>] [--page-base]
//
// --page-base: build & flatten ONLY the runtime /base shell instead of the root page.
//   This is a COMPANY-AGNOSTIC TEMPLATE (see routes/base/+page.ts): one HTML shell that
//   downstream tooling copies, rewrites the cdn-url/page-id <head> metas on, and serves
//   for any company/page. So it is NOT nested under a company id — it ships as
//   `index.html` inside `dist-prerender/base` (assets default to <FRONTEND_CDN>/websites/base).
//   --company is still required (it only flips SSR/prerender on); its value does not scope
//   the output. The page loads its content from the CDN snapshot at view time, so nothing
//   tenant-specific is baked in.
//
// The asset base defaults to `<FRONTEND_CDN>/websites/<company>` read from the repo's
// credentials.json (matching backend/exec's companyWebpageAssetBase); pass --asset-base
// only to override it (e.g. a manual/local run).
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
import { createHash } from 'node:crypto';

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
// --page-base builds & flattens ONLY the runtime /base shell into a `base/` folder.
const pageBase = args.includes('--page-base');
// This script lives in <repo>/scripts; the SvelteKit app is at <repo>/frontend/webpage.
const here = dirname(fileURLToPath(import.meta.url)); // <repo>/scripts
const repoRoot = resolve(here, '..');                 // <repo>
const appDir = resolve(repoRoot, 'frontend/webpage');
const buildDir = resolve(appDir, 'build');
// The live build lands in a dedicated `live/` subfolder so its flattened assets never
// collide with the root build's output for the same company.
// The template build is company-agnostic, so it lands in a shared `base/` folder — NOT
// nested under the company id (which only enables the SSR/prerender build mechanism).
const defaultOut = pageBase
  ? resolve(appDir, 'dist-prerender', 'base')
  : resolve(appDir, 'dist-prerender', String(company));
const outDir = resolve(getFlag('--out') || defaultOut);

// Asset base = the https origin the flattened JS/CSS get served from. By default it's
// derived from credentials.json the same way backend/exec's companyWebpageAssetBase
// does — <FRONTEND_CDN>/websites/<company> — so the deploy needs no --asset-base flag.
// Passing --asset-base overrides it (handy for a manual/local run).
const readFrontendCdn = () => {
  const credentialsPath = resolve(repoRoot, 'credentials.json');
  try {
    const cdn = JSON.parse(readFileSync(credentialsPath, 'utf8')).FRONTEND_CDN;
    return typeof cdn === 'string' ? cdn.trim().replace(/\/+$/, '') : '';
  } catch (error) {
    console.error(`[prerender] could not read FRONTEND_CDN from ${credentialsPath}: ${error.message}`);
    return '';
  }
};
const assetBaseOverride = getFlag('--asset-base');
const frontendCdn = assetBaseOverride ? '' : readFrontendCdn();
// Template assets are shared across companies → <FRONTEND_CDN>/websites/base (no company
// segment). The per-company root build keeps its <FRONTEND_CDN>/websites/<company> base.
const derivedAssetBase = pageBase
  ? `${frontendCdn}/websites/base`
  : `${frontendCdn}/websites/${company}`;
const assetBase = (
  assetBaseOverride ||
  (frontendCdn && derivedAssetBase) ||
  ''
).replace(/\/+$/, '');
if (!assetBase.startsWith('https://')) {
  console.error(
    'Error: asset base unresolved — set FRONTEND_CDN (https URL) in credentials.json or pass --asset-base.',
  );
  process.exit(1);
}

console.log(`[prerender] company=${company}`);
console.log(`[prerender] out=${outDir}`);
console.log(`[prerender] asset-base=${assetBase}`);
if (pageBase) console.log('[prerender] mode=page-base (only the /base shell)');

const env = { ...process.env, VITE_COMPANY_ID: String(company) };
// Gate the build (svelte.config entries + the routes' prerender flags) to /base only.
if (pageBase) env.VITE_PRERENDER_BASE = '1';
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

if (pageBase) promoteBasePage(outDir);

inlineBootstrap(outDir, assetBase);
flattenAssets(outDir);
rewriteAssetUrls(outDir, assetBase);
validateFlattenedAssetUrls(outDir, assetBase);
mergeCss(outDir, assetBase);
dropOrphanEnvChunk(outDir, assetBase);
inlineDompurifyStubChunk(outDir, assetBase);
inlineTrivialNodeChunks(outDir, assetBase);

// --- --page-base: promote the base shell to the folder root ---------------------
//
// The --page-base build prerenders only the /base route (→ base.html) plus the
// SvelteKit SPA fallback (404.html). Rename base.html → index.html so the flattened
// `base/` folder serves the page at its root, and drop the fallback — this deploy
// ships ONLY the base page. Runs before the rewrite/flatten steps so they see index.html.
function promoteBasePage(dir) {
  const basePage = resolve(dir, 'base.html');
  if (!existsSync(basePage)) {
    console.error('[prerender] --page-base: expected base.html in build output, not found');
    process.exit(1);
  }
  renameSync(basePage, resolve(dir, 'index.html'));
  rmSync(resolve(dir, '404.html'), { force: true });
  console.log('[prerender] --page-base: promoted base.html → index.html, dropped 404.html');
}

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

// --- Merge every stylesheet into a single file ---------------------------------
//
// SvelteKit (bundleStrategy 'split') emits one CSS file per route node and per
// component (vendor, ProductCard, node 0/2/3, …), each linked eagerly in the
// prerendered <head>. They all load on first paint anyway, so we concatenate them
// into one app.<hash>.css and collapse the multiple <link>s into a single request.
//
// Cascade order is taken verbatim from the <link> order SvelteKit emitted in
// index.html (Svelte scopes classes by hash, so cross-file order rarely matters, but
// global styles like tailwind/store.css do — preserving the emitted order is safest).
// The external fontello <link> (an https:// URL, not a local file) is left untouched.
//
// Beyond the <head> links, the only other CSS reference is SvelteKit's __vite_preload
// dependency list in a route chunk (e.g. ["…/vendor.css"]); those are repointed at the
// merged file so the preload hits the same cached bundle instead of a deleted file.
function mergeCss(dir, publicAssetBase) {
  const escapeRe = (value) => value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const localCss = readdirSync(dir).filter((name) => name.endsWith('.css'));
  if (localCss.length <= 1) {
    console.log('[prerender] mergeCss: <=1 stylesheet, nothing to merge');
    return;
  }

  // A <link> to one of our local CSS files: href points at publicAssetBase/<file>.css.
  // The external fontello link uses a different host, so it never matches.
  const localCssLinkRe = new RegExp(
    `<link[^>]*href="${escapeRe(publicAssetBase)}/([A-Za-z0-9._-]+\\.css)"[^>]*>`,
    'g',
  );

  // Cascade order = <link> order in index.html; any local CSS not linked there is
  // appended (sorted) so nothing is silently dropped.
  const indexPath = resolve(dir, 'index.html');
  const orderedFromHtml = [];
  if (existsSync(indexPath)) {
    for (const m of readFileSync(indexPath, 'utf8').matchAll(localCssLinkRe)) {
      if (localCss.includes(m[1]) && !orderedFromHtml.includes(m[1])) orderedFromHtml.push(m[1]);
    }
  }
  const ordered = [
    ...orderedFromHtml,
    ...localCss.filter((f) => !orderedFromHtml.includes(f)).sort(),
  ];

  const merged = ordered.map((f) => readFileSync(resolve(dir, f), 'utf8')).join('\n');
  const hash = createHash('sha256').update(merged).digest('base64url').slice(0, 8);
  const mergedName = `app.${hash}.css`;
  writeFileSync(resolve(dir, mergedName), merged);
  const mergedUrl = `${publicAssetBase}/${mergedName}`;
  const oldUrls = ordered.map((f) => `${publicAssetBase}/${f}`);

  let rewrittenFiles = 0;
  for (const name of readdirSync(dir)) {
    if (!/\.(html|js|css)$/.test(name) || name === mergedName) continue;
    const full = resolve(dir, name);
    const before = readFileSync(full, 'utf8');
    let content = before;
    if (name.endsWith('.html')) {
      // Collapse the <head> stylesheet links: the FIRST merged-file <link> becomes the
      // single merged link, the rest are dropped. Done BEFORE the URL rewrite below so
      // the old filenames are still present to match (and so the links don't all turn
      // into duplicate merged links).
      let replacedFirst = false;
      content = content.replace(localCssLinkRe, (tag, file) => {
        if (!ordered.includes(file)) return tag;
        if (replacedFirst) return '';
        replacedFirst = true;
        return `<link href="${mergedUrl}" rel="stylesheet">`;
      });
    }
    // Repoint every remaining reference to a merged file at the merged bundle, matching
    // the bare URL regardless of quoting. This covers SvelteKit's __vite_preload
    // dependency arrays — present both in route chunks (.js) and in the inlined
    // bootstrap script inside each HTML file, where the quotes are backslash-escaped
    // (\"…css\"). Without this those preloads would 404 on the now-deleted files.
    for (const oldUrl of oldUrls) {
      if (content.includes(oldUrl)) content = content.split(oldUrl).join(mergedUrl);
    }
    if (content !== before) {
      writeFileSync(full, content);
      rewrittenFiles++;
    }
  }

  for (const f of ordered) rmSync(resolve(dir, f), { force: true });

  console.log(
    `[prerender] merged ${ordered.length} stylesheet(s) into ${mergedName} ` +
      `(refs rewritten in ${rewrittenFiles} file(s))`,
  );
}

// --- Drop / inline tiny single-purpose chunks ----------------------------------
//
// bundleStrategy 'split' emits a few sub-100-byte chunks that each cost an HTTP
// request on the critical path but carry almost no code. The three helpers below
// remove them after the asset URLs have been flattened to absolute base/<file> form:
//   • env.js              — SvelteKit always emits $env/dynamic/public even when the
//                           storefront only uses inlined $env/static/public literals;
//                           here it is orphaned, so it's deleted.
//   • <dompurify-stub>.js — the DOMPurify stub (see vite.config.ts) reached only from
//                           a dynamic import in getPageContent(); inlined to a resolved
//                           module so the chunk + request vanish.
//   • node re-export      — a SvelteKit route node whose body is just
//                           `import{X}from"<chunk>";export{X as component}`; the loader
//                           is repointed straight at <chunk> and the shim is removed.

// Build-time helper: escape a string for use as a literal inside a RegExp.
// (A function declaration so it's hoisted above the pipeline calls at module top.)
function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

// Replace every dynamic `import("<url>")` (any quote style — ", ', backtick, or the
// backslash-escaped \" that appears in the inlined-bootstrap script inside HTML) with
// `replacement`. Used to swap a chunk import for an inline module expression.
function replaceDynamicImport(source, url, replacement) {
  // A quote is one of " ' ` optionally preceded by a backslash (HTML-escaped form).
  const quote = '(?:\\\\?["\'\\x60])';
  const re = new RegExp(`import\\(\\s*${quote}${escapeRegExp(url)}${quote}\\s*\\)`, 'g');
  return source.replace(re, replacement);
}

function dropOrphanEnvChunk(dir, publicAssetBase) {
  const envPath = resolve(dir, 'env.js');
  if (!existsSync(envPath)) return;
  const envUrl = `${publicAssetBase}/env.js`;
  const referenced = readdirSync(dir).some((name) => {
    if (name === 'env.js' || !/\.(html|js|css)$/.test(name)) return false;
    return readFileSync(resolve(dir, name), 'utf8').includes(envUrl);
  });
  if (referenced) {
    console.log('[prerender] env.js is referenced, keeping');
    return;
  }
  rmSync(envPath, { force: true });
  console.log('[prerender] removed orphan env.js');
}

function inlineDompurifyStubChunk(dir, publicAssetBase) {
  // The stub minifies to `var e={sanitize:e=>e};export{e as default};` (any minified
  // identifier). Discover it by shape, not name — the hash changes every build.
  const stubRe = /^var (\w+)=\{sanitize:\w+=>\w+\};export\{\1 as default\};?\s*$/;
  const stub = readdirSync(dir)
    .filter((name) => name.endsWith('.js'))
    .find((name) => stubRe.test(readFileSync(resolve(dir, name), 'utf8').trim()));
  if (!stub) {
    console.log('[prerender] inlineDompurifyStubChunk: no stub chunk found, skipping');
    return;
  }
  const stubUrl = `${publicAssetBase}/${stub}`;
  // The sole consumer destructures `{default:…}` and calls `.sanitize`; this path never
  // runs in the public storefront, so an identity sanitizer is behaviourally exact.
  const replacement = '(Promise.resolve({default:{sanitize:(s)=>s}}))';
  let rewritten = 0;
  for (const name of readdirSync(dir)) {
    if (!/\.(html|js)$/.test(name)) continue;
    const full = resolve(dir, name);
    const before = readFileSync(full, 'utf8');
    const after = replaceDynamicImport(before, stubUrl, replacement);
    if (after !== before) {
      writeFileSync(full, after);
      rewritten++;
    }
  }
  // Only delete once nothing references it (the stub has empty preload deps, so the
  // import call is its only reference — but verify before removing to avoid a 404).
  const stillReferenced = readdirSync(dir).some(
    (name) => name !== stub && /\.(html|js)$/.test(name) &&
      readFileSync(resolve(dir, name), 'utf8').includes(stubUrl),
  );
  if (stillReferenced) {
    console.warn(`[prerender] inlineDompurifyStubChunk: ${stub} still referenced, keeping`);
    return;
  }
  rmSync(resolve(dir, stub), { force: true });
  console.log(`[prerender] inlined DOMPurify stub (${stub}) into ${rewritten} file(s) and removed it`);
}

function inlineTrivialNodeChunks(dir, publicAssetBase) {
  // A SvelteKit route node that only re-exports a component:
  //   import{<imported> as <local>}from"<target-url>";export{<local> as component};
  // Its component is just <target>'s <imported> export, and <target> is already in the
  // node's preload deps — so the indirection chunk is pure overhead.
  const nodeRe = /^import\{(\w+) as (\w+)\}from"([^"]+)";export\{\2 as component\};?\s*$/;
  let removed = 0;
  for (const node of readdirSync(dir).filter((name) => name.endsWith('.js'))) {
    const match = nodeRe.exec(readFileSync(resolve(dir, node), 'utf8').trim());
    if (!match) continue;
    const [, importedName, , targetUrl] = match;
    const nodeUrl = `${publicAssetBase}/${node}`;
    if (targetUrl === nodeUrl) continue; // self-reference guard
    // Loader: pull `component` straight from the target chunk.
    const loader = `import(\`${targetUrl}\`).then((m)=>({component:m.${importedName}}))`;
    let rewritten = 0;
    for (const name of readdirSync(dir)) {
      if (!/\.(html|js)$/.test(name)) continue;
      const full = resolve(dir, name);
      const before = readFileSync(full, 'utf8');
      // 1) Rewrite the node loader's dynamic import (now points at <target>, not <node>).
      let content = replaceDynamicImport(before, nodeUrl, loader);
      // 2) Repoint any remaining bare <node> URL (the __vite_preload dep arrays) at
      //    <target>; it's already preloaded, so the duplicate modulepreload is deduped.
      if (content.includes(nodeUrl)) content = content.split(nodeUrl).join(targetUrl);
      if (content !== before) {
        writeFileSync(full, content);
        rewritten++;
      }
    }
    const stillReferenced = readdirSync(dir).some(
      (name) => name !== node && /\.(html|js)$/.test(name) &&
        readFileSync(resolve(dir, name), 'utf8').includes(nodeUrl),
    );
    if (stillReferenced) {
      console.warn(`[prerender] inlineTrivialNodeChunks: ${node} still referenced, keeping`);
      continue;
    }
    rmSync(resolve(dir, node), { force: true });
    removed++;
    console.log(`[prerender] inlined trivial node chunk (${node}) into ${rewritten} file(s) and removed it`);
  }
  if (removed === 0) console.log('[prerender] inlineTrivialNodeChunks: no trivial node chunks');
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
