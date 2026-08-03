#!/usr/bin/env bun
// Construye el artefacto que el Lambda de render descarga: webpage-renderer.zip.
//
// Un solo build, agnóstico de company, que produce:
//   render.mjs      el servidor SSR de SvelteKit bundleado con esbuild (sin node_modules)
//   assets/_app/**  los js/css que el Lambda sube tal cual a websites/<companyID>/
//   site/           sw.js + favicon.ico (van al origen del sitio, no al CDN)
//   manifest.json   las reescrituras de HTML que el Lambda aplica a cada render
//
// Por qué hay reescrituras: el HTML que emite el SSR apunta a '/_app/…' (raíz del sitio),
// pero los assets viven en el CDN bajo websites/<companyID>/. Ese prefijo es lo único que
// depende del tenant, así que se resuelve en el Lambda y los assets se suben SIN TOCAR.
// Las demás reescrituras (unir las hojas de estilo en una sola) son deterministas por
// build, así que se calculan aquí una vez en vez de en cada render.
//
// Uso:  bun scripts/build-renderer.mjs [--skip-build]

import { spawnSync } from 'node:child_process';
import {
  cpSync,
  existsSync,
  mkdirSync,
  readdirSync,
  readFileSync,
  rmSync,
  statSync,
  writeFileSync,
} from 'node:fs';
import { createRequire } from 'node:module';
import { dirname, extname, relative, resolve } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { createHash } from 'node:crypto';

const here = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(here, '..');
const appDir = resolve(repoRoot, 'frontend/webpage');
const clientDir = resolve(appDir, 'build');
const serverDir = resolve(appDir, '.svelte-kit/output/server');
const outDir = resolve(appDir, 'build-renderer');
const stageDir = resolve(outDir, 'package');
const zipPath = resolve(outDir, 'webpage-renderer.zip');

// El SSR emite las URLs de assets contra la raíz del sitio (paths.relative=false en
// svelte.config.js); el Lambda cambia este prefijo por el del CDN de cada company.
const ASSET_PATH_PREFIX = '/_app/';

const fail = (message) => {
  console.error(`[build-renderer] ${message}`);
  process.exit(1);
};

// --- 1. Build de SvelteKit en modo renderer ------------------------------------
if (!process.argv.includes('--skip-build')) {
  console.log('[build-renderer] building SvelteKit (VITE_RENDERER_BUILD=1)');
  const build = spawnSync('bun', ['run', 'build'], {
    cwd: appDir,
    env: { ...process.env, VITE_RENDERER_BUILD: '1' },
    stdio: 'inherit',
  });
  if (build.status !== 0) fail('el build de SvelteKit falló');
}

if (!existsSync(resolve(serverDir, 'index.js'))) fail(`falta el servidor SSR: ${serverDir}`);
if (!existsSync(resolve(clientDir, '_app'))) fail(`faltan los assets cliente: ${clientDir}/_app`);

rmSync(outDir, { recursive: true, force: true });
mkdirSync(stageDir, { recursive: true });

// --- 2. Bundle del servidor SSR ------------------------------------------------
//
// El output de SvelteKit son decenas de chunks que importan a @sveltejs/kit desde
// node_modules. esbuild los colapsa en un archivo para que el zip no lleve dependencias.
const esbuild = createRequire(resolve(appDir, 'package.json'))('esbuild');
const renderPath = resolve(stageDir, 'render.mjs');
await esbuild.build({
  entryPoints: [resolve(appDir, 'lambda/renderer-entry.js')],
  outfile: renderPath,
  bundle: true,
  format: 'esm',
  platform: 'node',
  target: 'node22',
  minify: true,
  // Parte del runtime de SvelteKit sigue siendo CJS; en un bundle ESM `require` no existe,
  // así que se reconstruye desde node:module.
  banner: {
    js: "import{createRequire as __genixCreateRequire}from'node:module';const require=__genixCreateRequire(import.meta.url);",
  },
});
console.log(`[build-renderer] render.mjs ${formatKB(statSync(renderPath).size)}`);

// --- 3. Assets cliente ---------------------------------------------------------
//
// Se copian tal cual: los chunks se referencian entre sí con rutas relativas, así que
// funcionan desde cualquier prefijo del CDN sin reescribir un solo byte.
//
// Solo js y css. El grafo arrastra binarios de funciones que la tienda pública nunca
// ejecuta (el wasm de excelize para exportar a Excel, los encoders avif del builder):
// son megabytes que ni el zip ni el CDN necesitan. El pipeline anterior también los
// descartaba, así que el sitio publicado se comporta igual.
const assetsDir = resolve(stageDir, 'assets');
const discarded = [];
cpSync(resolve(clientDir, '_app'), resolve(assetsDir, '_app'), {
  recursive: true,
  filter: (source) => {
    if (statSync(source).isDirectory()) return true;
    const extension = extname(source).toLowerCase();
    if (extension === '.js' || extension === '.css') return true;
    discarded.push({ name: relative(clientDir, source), size: statSync(source).size });
    return false;
  },
});
const discardedBytes = discarded.reduce((total, file) => total + file.size, 0);
console.log(
  `[build-renderer] descartados ${discarded.length} archivo(s) no-código (${formatKB(discardedBytes)}): ` +
    discarded
      .sort((a, b) => b.size - a.size)
      .slice(0, 5)
      .map((file) => file.name)
      .join(', '),
);

// --- 4. Una sola hoja de estilo ------------------------------------------------
//
// SvelteKit (bundleStrategy 'split') emite un css por nodo de ruta y por componente, y
// los enlaza todos en el <head>: se cargan siempre en la primera pintura, así que se
// concatenan en uno y se ahorran N-1 peticiones. Las referencias desde los chunks JS
// (los arrays de dependencias de __vite_preload) se reapuntan aquí porque el nombre del
// archivo unido ya se conoce en tiempo de build.
const cssDir = resolve(assetsDir, '_app/immutable/assets');
const stylesheets = existsSync(cssDir)
  ? readdirSync(cssDir).filter((name) => name.endsWith('.css')).sort()
  : [];

// El orden de cascada real es el de los <link> del HTML renderizado, no el alfabético,
// así que la unión ocurre en el paso 5, cuando ya hay un render de prueba del que leerlo.
const shouldMergeStylesheets = stylesheets.length > 1;

// --- 5. Render de prueba: valida el bundle y fija el orden de la cascada --------
const { renderPage } = await import(pathToFileURL(renderPath).href);
const smoke = await renderPage({
  origin: 'https://renderer.invalid',
  path: '/',
  companyID: 0,
  pageID: 0,
});
if (smoke.status !== 200) fail(`el render de prueba devolvió HTTP ${smoke.status}`);
if (!smoke.html.includes('<meta name="company-id"')) {
  fail('el render de prueba no emitió la meta company-id (¿hooks.server.ts no corrió?)');
}
console.log(`[build-renderer] render de prueba OK (${formatKB(smoke.html.length)} de HTML)`);

// Los <link> de css tal cual los emite el SSR, en orden de cascada.
const stylesheetTags = [
  ...smoke.html.matchAll(/<link[^>]+href="(\/_app\/immutable\/assets\/[^"]+\.css)"[^>]*>/g),
].map((match) => ({ tag: match[0], file: match[1].split('/').pop() }));

const htmlRewrites = [];

if (shouldMergeStylesheets) {
  if (stylesheetTags.length < 2) {
    fail(`hay ${stylesheets.length} hojas de estilo pero el HTML enlaza ${stylesheetTags.length}`);
  }
  // Solo las que la página de tienda enlaza. Las demás pertenecen a otras rutas
  // (/components, /base) y fundirlas aquí engordaría el css de todas las páginas.
  const ordered = stylesheetTags.map((link) => link.file);

  const merged = ordered.map((file) => readFileSync(resolve(cssDir, file), 'utf8')).join('\n');
  const mergedName = `app.${createHash('sha256').update(merged).digest('base64url').slice(0, 8)}.css`;
  writeFileSync(resolve(cssDir, mergedName), merged);
  for (const file of ordered) rmSync(resolve(cssDir, file), { force: true });

  // Los chunks JS listan sus css como dependencias de preload; se reapuntan al unido.
  let patchedChunks = 0;
  for (const jsFile of walkFiles(assetsDir).filter((file) => file.endsWith('.js'))) {
    const before = readFileSync(jsFile, 'utf8');
    let content = before;
    for (const file of ordered) content = content.split(file).join(mergedName);
    if (content !== before) {
      writeFileSync(jsFile, content);
      patchedChunks++;
    }
  }

  // En el HTML: el primer <link> pasa a ser el unido y el resto desaparecen.
  const mergedHref = `/_app/immutable/assets/${mergedName}`;
  htmlRewrites.push({
    find: stylesheetTags[0].tag,
    replace: `<link href="${mergedHref}" rel="stylesheet">`,
    required: true,
  });
  for (const link of stylesheetTags.slice(1)) {
    htmlRewrites.push({ find: link.tag, replace: '', required: true });
  }

  console.log(
    `[build-renderer] ${ordered.length} hojas unidas en ${mergedName} ` +
      `(${formatKB(merged.length)}, refs reapuntadas en ${patchedChunks} chunk(s))`,
  );
}

// --- 6. Validación: aplicar las reglas al HTML de prueba -----------------------
//
// Es exactamente lo que hará el Lambda en cada render. Si una regla dejara de casar (un
// cambio de formato en los <link> que emite SvelteKit, por ejemplo), la página saldría
// con css muerto y 404s silenciosos: mejor romper aquí, en CI.
const rehearsalBase = 'https://cdn.invalid/websites/1';
let rehearsal = smoke.html;
for (const rule of htmlRewrites) {
  if (rule.required && !rehearsal.includes(rule.find)) {
    fail(`la regla de reescritura no casa con el HTML renderizado: ${rule.find}`);
  }
  rehearsal = rehearsal.split(rule.find).join(rule.replace);
}
rehearsal = rehearsal.split(ASSET_PATH_PREFIX).join(`${rehearsalBase}${ASSET_PATH_PREFIX}`);

for (const referenced of [...rehearsal.matchAll(/["'(]([^"'()\s]*\/_app\/[^"'()\s]+)/g)]) {
  const url = referenced[1];
  if (!url.startsWith(rehearsalBase)) fail(`quedó una URL de asset sin prefijar: ${url}`);
  const assetPath = url.slice(`${rehearsalBase}/`.length).split('?')[0];
  if (!existsSync(resolve(assetsDir, assetPath))) {
    fail(`el HTML referencia un asset que no está en el paquete: ${assetPath}`);
  }
}
console.log('[build-renderer] reglas validadas contra el render de prueba');

// --- 7. Archivos del origen del sitio ------------------------------------------
//
// El service worker DEBE ser same-origin (el navegador rechaza uno cross-origin) y el
// favicon lo pide el <head> como /favicon.ico, así que estos dos no van al CDN.
const siteDir = resolve(stageDir, 'site');
mkdirSync(siteDir, { recursive: true });
const siteFiles = ['sw.js', 'favicon.ico'].filter((name) => existsSync(resolve(clientDir, name)));
for (const name of siteFiles) cpSync(resolve(clientDir, name), resolve(siteDir, name));

// --- 8. Manifest ---------------------------------------------------------------
const assetFiles = walkFiles(assetsDir).map((file) => relative(assetsDir, file));
const buildId = createHash('sha256')
  .update(readFileSync(renderPath))
  .update(assetFiles.sort().join('|'))
  .digest('base64url')
  .slice(0, 12);

writeFileSync(
  resolve(stageDir, 'manifest.json'),
  JSON.stringify(
    { buildId, assetPathPrefix: ASSET_PATH_PREFIX, htmlRewrites, assets: assetFiles, siteFiles },
    null,
    2,
  ),
);

// --- 9. Zip --------------------------------------------------------------------
const zip = spawnSync('zip', ['-r', '-q', '-X', zipPath, '.'], { cwd: stageDir, stdio: 'inherit' });
if (zip.status !== 0) fail('no se pudo comprimir el artefacto');

console.log(`\n[build-renderer] buildId=${buildId}`);
console.log(`[build-renderer] assets=${assetFiles.length} site=${siteFiles.length} rewrites=${htmlRewrites.length}`);
console.log(`[build-renderer] ${zipPath} ${formatKB(statSync(zipPath).size)}`);

function walkFiles(directory) {
  if (!existsSync(directory)) return [];
  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const full = resolve(directory, entry.name);
    return entry.isDirectory() ? walkFiles(full) : [full];
  });
}

function formatKB(bytes) {
  return `${(bytes / 1024).toFixed(1)} KB`;
}
