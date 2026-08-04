// Lambda de render de las webpages de company (Node, invocada por la Lambda de Go).
//
// Descarga el artefacto que publica CI (webpage-renderer.zip: el servidor SSR de la
// tienda + sus assets), renderiza las páginas de UNA company y publica el resultado:
//   assets js/css  →  R2  websites/<companyID>/_app/…        (inmutables, copia propia
//                                                             de la company)
//   HTML por página →  R2  websites-html/<hostname>/…/index.html
//   sw.js/favicon  →  R2  websites-html/<hostname>/…         (deben ser same-origin)
//
// Evento: { companyID, hostname, pages: [{ id, path }], forceAssets? }
// Respuesta: { buildId, pages, assets, bytes }
//
// Sin dependencias npm: el runtime de Lambda solo trae Node, y añadir una librería
// obligaría a empaquetar node_modules en el zip de la función. Por lo mismo el zip de la
// función lleva SOLO este archivo (cloud/webpage-renderer.go): cli.mjs y los tests que lo
// acompañan nunca viajan a AWS.
//
// Fuera de Lambda el backend lo ejecuta con `node cli.mjs` (backend/cloud/webpage_renderer.go),
// así que este módulo no puede asumir el entorno de AWS en ningún punto.

import { inflateRawSync } from 'node:zlib';
import { mkdirSync, rmSync, writeFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { pathToFileURL } from 'node:url';

const RENDERER_ZIP_URL = process.env.RENDERER_ZIP_URL || '';
const FRONTEND_CDN = (process.env.FRONTEND_CDN || '').replace(/\/+$/, '');
const CLOUDFLARE_ACCOUNT = process.env.CLOUDFLARE_ACCOUNT || '';
const CLOUDFLARE_TOKEN = process.env.CLOUDFLARE_TOKEN || '';
const CLOUDFLARE_BUCKET = process.env.CLOUDFLARE_BUCKET || '';
// Redirigible para poder ejercitar el handler completo contra un R2 simulado.
const CLOUDFLARE_API_BASE = process.env.CLOUDFLARE_API_BASE || 'https://api.cloudflare.com/client/v4';

// Prefijo en R2 del HTML servido por el Worker de la tienda. Debe coincidir con el que
// lee frontend/webpage/cloudflare/serve-worker.js.
const HTML_KEY_ROOT = 'websites-html';
const UPLOAD_CONCURRENCY = 4;

// El artefacto descomprimido sobrevive entre invocaciones calientes; una nueva versión
// solo se descarga cuando cambia el ETag.
let loadedRenderer = null;

export async function render(event) {
	const companyID = Number(event?.companyID) || 0;
	const hostname = String(event?.hostname || '').trim().toLowerCase();
	const pages = Array.isArray(event?.pages) ? event.pages : [];

	if (companyID <= 0) throw new Error('companyID inválido');
	if (!/^[a-z0-9.-]+\.[a-z]{2,}$/.test(hostname)) throw new Error(`hostname inválido: ${hostname}`);
	if (pages.length === 0) throw new Error('no se recibió ninguna página para renderizar');
	for (const missing of ['RENDERER_ZIP_URL', 'FRONTEND_CDN', 'CLOUDFLARE_ACCOUNT', 'CLOUDFLARE_TOKEN', 'CLOUDFLARE_BUCKET']) {
		if (!process.env[missing]) throw new Error(`falta la variable de entorno ${missing}`);
	}

	const renderer = await loadRenderer();
	const assetKeyPrefix = `websites/${companyID}`;
	const assetBase = `${FRONTEND_CDN}/${assetKeyPrefix}`;
	const htmlKeyPrefix = `${HTML_KEY_ROOT}/${hostname}`;

	const uploadedAssets = await publishAssets(renderer, assetKeyPrefix, htmlKeyPrefix, !!event?.forceAssets);

	// SECUENCIAL a propósito: hooks.server.ts fija el tenant sobre el singleton Env del
	// bundle SSR, así que dos renders simultáneos en el mismo proceso se pisarían.
	let bytes = 0;
	for (const page of pages) {
		const pageID = Number(page?.id) || 0;
		const path = normalizePagePath(page?.path);
		const rendered = await renderer.renderPage({
			origin: `https://${hostname}`,
			path,
			companyID,
			pageID
		});
		if (rendered.status !== 200) {
			throw new Error(`la página ${path} (id ${pageID}) devolvió HTTP ${rendered.status}`);
		}

		const html = applyHtmlRewrites(rendered.html, renderer.manifest, assetBase);
		const key = `${htmlKeyPrefix}${path === '/' ? '' : path}/index.html`;
		await putObject(key, Buffer.from(html, 'utf8'), 'text/html; charset=utf-8', 'public, max-age=60');
		bytes += Buffer.byteLength(html);
		console.log(`[renderer] page=${path} id=${pageID} → ${key} (${html.length} bytes)`);
	}

	return { buildId: renderer.manifest.buildId, pages: pages.length, assets: uploadedAssets, bytes };
}

// El Worker sirve una navegación como <host>/<ruta>/index.html, así que la ruta se
// normaliza a la misma forma con la que se construye la clave.
export function normalizePagePath(rawPath) {
	const path = String(rawPath || '/').trim();
	if (path === '' || path === '/') return '/';
	const normalized = `/${path.replace(/^\/+|\/+$/g, '')}`;
	if (!/^\/[a-z0-9\-/]+$/i.test(normalized)) throw new Error(`ruta de página inválida: ${rawPath}`);
	return normalized;
}

// El HTML del SSR apunta a '/_app/…' (raíz del sitio). Las reglas del manifest son las
// transformaciones deterministas que CI ya calculó (unir las hojas de estilo); lo único
// que depende del tenant es anteponer la base del CDN de la company.
export function applyHtmlRewrites(html, manifest, assetBase) {
	let output = html;
	for (const rule of manifest.htmlRewrites || []) {
		if (rule.required && !output.includes(rule.find)) {
			throw new Error(`la regla de reescritura no casa con el HTML renderizado: ${rule.find}`);
		}
		output = output.split(rule.find).join(rule.replace);
	}
	return output.split(manifest.assetPathPrefix).join(`${assetBase}${manifest.assetPathPrefix}`);
}

// --- Artefacto del renderer -----------------------------------------------------

async function loadRenderer() {
	const response = await fetch(RENDERER_ZIP_URL, {
		headers: loadedRenderer ? { 'if-none-match': loadedRenderer.etag } : {}
	});
	if (response.status === 304 && loadedRenderer) return loadedRenderer;
	if (!response.ok) throw new Error(`no se pudo descargar ${RENDERER_ZIP_URL}: HTTP ${response.status}`);

	const etag = response.headers.get('etag') || '';
	// Algunos orígenes no responden 304; comparar el ETag evita reextraer lo mismo.
	if (loadedRenderer && etag && etag === loadedRenderer.etag) return loadedRenderer;

	const started = Date.now();
	const entries = readZipEntries(Buffer.from(await response.arrayBuffer()));
	const directory = `/tmp/renderer-${etag.replace(/[^a-zA-Z0-9]/g, '') || Date.now()}`;

	rmSync(directory, { recursive: true, force: true });
	const assets = [];
	const siteFiles = [];
	let manifest = null;
	for (const entry of entries) {
		const target = resolve(directory, entry.name);
		mkdirSync(dirname(target), { recursive: true });
		writeFileSync(target, entry.data);

		if (entry.name === 'manifest.json') manifest = JSON.parse(entry.data.toString('utf8'));
		else if (entry.name.startsWith('assets/')) assets.push({ name: entry.name.slice(7), data: entry.data });
		else if (entry.name.startsWith('site/')) siteFiles.push({ name: entry.name.slice(5), data: entry.data });
	}
	if (!manifest) throw new Error('el artefacto no trae manifest.json');

	const { renderPage } = await import(pathToFileURL(resolve(directory, 'render.mjs')).href);

	// El artefacto anterior ya no se usa: /tmp son 512 MB y las invocaciones calientes
	// acumularían una copia por versión publicada.
	if (loadedRenderer) rmSync(loadedRenderer.directory, { recursive: true, force: true });

	loadedRenderer = { etag, directory, renderPage, manifest, assets, siteFiles };
	console.log(
		`[renderer] artefacto cargado buildId=${manifest.buildId} assets=${assets.length} ` +
			`site=${siteFiles.length} en ${Date.now() - started}ms`
	);
	return loadedRenderer;
}

// Lector mínimo de ZIP: se recorre el directorio central (la fuente autoritativa de los
// tamaños) y se infla cada entrada. Ver el comentario de cabecera sobre por qué no hay
// una librería aquí.
export function readZipEntries(buffer) {
	let endOfCentralDirectory = -1;
	for (let offset = buffer.length - 22; offset >= 0; offset--) {
		if (buffer.readUInt32LE(offset) === 0x06054b50) {
			endOfCentralDirectory = offset;
			break;
		}
	}
	if (endOfCentralDirectory < 0) throw new Error('zip inválido: falta el End of Central Directory');

	const entryCount = buffer.readUInt16LE(endOfCentralDirectory + 10);
	let cursor = buffer.readUInt32LE(endOfCentralDirectory + 16);
	const entries = [];

	for (let index = 0; index < entryCount; index++) {
		if (buffer.readUInt32LE(cursor) !== 0x02014b50) throw new Error('zip inválido: cabecera central corrupta');
		const compressionMethod = buffer.readUInt16LE(cursor + 10);
		const compressedSize = buffer.readUInt32LE(cursor + 20);
		const nameLength = buffer.readUInt16LE(cursor + 28);
		const extraLength = buffer.readUInt16LE(cursor + 30);
		const commentLength = buffer.readUInt16LE(cursor + 32);
		const localHeaderOffset = buffer.readUInt32LE(cursor + 42);
		const name = buffer.toString('utf8', cursor + 46, cursor + 46 + nameLength);

		// La cabecera local repite nombre y campo extra con longitudes propias, así que
		// el inicio de los datos solo se puede calcular leyéndola.
		const localNameLength = buffer.readUInt16LE(localHeaderOffset + 26);
		const localExtraLength = buffer.readUInt16LE(localHeaderOffset + 28);
		const dataStart = localHeaderOffset + 30 + localNameLength + localExtraLength;
		const raw = buffer.subarray(dataStart, dataStart + compressedSize);

		if (!name.endsWith('/')) {
			entries.push({ name, data: compressionMethod === 0 ? raw : inflateRawSync(raw) });
		}
		cursor += 46 + nameLength + extraLength + commentLength;
	}
	return entries;
}

// --- Publicación ----------------------------------------------------------------

// Los assets son idénticos para toda company que use el mismo artefacto, así que se
// marca la versión publicada y se salta la subida cuando no cambió: editar el contenido
// de una página pasa a costar solo los PUT del HTML.
async function publishAssets(renderer, assetKeyPrefix, htmlKeyPrefix, force) {
	const markerKey = `${assetKeyPrefix}/.renderer-build`;
	if (!force && (await getObjectText(markerKey)) === renderer.manifest.buildId) {
		console.log(`[renderer] assets al día (buildId=${renderer.manifest.buildId}), no se resuben`);
		return 0;
	}

	const uploads = [
		...renderer.assets.map((asset) => ({
			key: `${assetKeyPrefix}/${asset.name}`,
			data: asset.data,
			// Los nombres llevan hash de contenido: nunca cambian bajo la misma URL.
			cacheControl: 'public, max-age=31536000, immutable'
		})),
		...renderer.siteFiles.map((file) => ({
			key: `${htmlKeyPrefix}/${file.name}`,
			data: file.data,
			// El service worker NO puede cachearse: es lo que gobierna las actualizaciones.
			cacheControl: file.name === 'sw.js' ? 'public, max-age=0, must-revalidate' : 'public, max-age=86400'
		}))
	];

	for (let index = 0; index < uploads.length; index += UPLOAD_CONCURRENCY) {
		await Promise.all(
			uploads
				.slice(index, index + UPLOAD_CONCURRENCY)
				.map((upload) => putObject(upload.key, upload.data, contentTypeFor(upload.key), upload.cacheControl))
		);
	}

	await putObject(markerKey, Buffer.from(renderer.manifest.buildId, 'utf8'), 'text/plain', 'no-store');
	console.log(`[renderer] subidos ${uploads.length} archivo(s) para buildId=${renderer.manifest.buildId}`);
	return uploads.length;
}

export function contentTypeFor(key) {
	if (key.endsWith('.js')) return 'text/javascript; charset=utf-8';
	if (key.endsWith('.css')) return 'text/css; charset=utf-8';
	if (key.endsWith('.ico')) return 'image/x-icon';
	if (key.endsWith('.json')) return 'application/json';
	return 'application/octet-stream';
}

// Misma API REST de R2 que usa backend/cloud/s3.go:SaveFileToR2.
function objectUrl(key) {
	return `${CLOUDFLARE_API_BASE}/accounts/${CLOUDFLARE_ACCOUNT}/r2/buckets/${CLOUDFLARE_BUCKET}/objects/${key}`;
}

async function putObject(key, data, contentType, cacheControl) {
	const response = await fetch(objectUrl(key), {
		method: 'PUT',
		headers: {
			authorization: `Bearer ${CLOUDFLARE_TOKEN}`,
			'content-type': contentType,
			'cache-control': cacheControl
		},
		body: data
	});
	if (!response.ok) {
		throw new Error(`R2 PUT ${key} → HTTP ${response.status}: ${(await response.text()).slice(0, 300)}`);
	}
}

async function getObjectText(key) {
	const response = await fetch(objectUrl(key), {
		headers: { authorization: `Bearer ${CLOUDFLARE_TOKEN}` }
	});
	if (response.status === 404) return '';
	if (!response.ok) {
		throw new Error(`R2 GET ${key} → HTTP ${response.status}: ${(await response.text()).slice(0, 300)}`);
	}
	return (await response.text()).trim();
}
