// Tests del handler del renderer. Se ejecutan con `bun test` desde esta carpeta.
//
// El handler entero se puede ejercitar en proceso porque tiene dos costuras: CLOUDFLARE_API_BASE
// se puede apuntar a un servidor local que hace de R2, y RENDERER_ZIP_URL es un fetch normal.
// Así que aquí se levanta un node:http que sirve las dos cosas —el artefacto y el bucket— y se
// comprueba qué claves se escriben y con qué cabeceras.
//
// Sobre las importaciones dinámicas: el módulo lee process.env en constantes de nivel superior al
// cargarse, y guarda el artefacto extraído en `loadedRenderer`. Los dos son estado de módulo, así
// que cada caso que necesite un entorno distinto pide una instancia nueva con loadHandler().

import { afterEach, expect, test } from 'bun:test';
import { createServer } from 'node:http';
import { deflateRawSync } from 'node:zlib';

// --- Constructor de ZIP -----------------------------------------------------------
//
// Escrito a mano por la misma razón que el lector del handler: no arrastrar una dependencia.
// Produce el formato que readZipEntries espera, con el directorio central como fuente de los
// tamaños. El CRC va en cero a propósito: el lector no lo verifica.

function buildZip(entries, { deflate = false } = {}) {
	const localParts = [];
	const centralParts = [];
	let offset = 0;

	for (const entry of entries) {
		const nameBytes = Buffer.from(entry.name, 'utf8');
		const rawData = Buffer.isBuffer(entry.data) ? entry.data : Buffer.from(entry.data, 'utf8');
		const storedData = deflate ? deflateRawSync(rawData) : rawData;
		const compressionMethod = deflate ? 8 : 0;

		const localHeader = Buffer.alloc(30);
		localHeader.writeUInt32LE(0x04034b50, 0);
		localHeader.writeUInt16LE(20, 4);
		localHeader.writeUInt16LE(compressionMethod, 8);
		localHeader.writeUInt32LE(storedData.length, 18);
		localHeader.writeUInt32LE(rawData.length, 22);
		localHeader.writeUInt16LE(nameBytes.length, 26);
		localParts.push(localHeader, nameBytes, storedData);

		const centralHeader = Buffer.alloc(46);
		centralHeader.writeUInt32LE(0x02014b50, 0);
		centralHeader.writeUInt16LE(20, 4);
		centralHeader.writeUInt16LE(20, 6);
		centralHeader.writeUInt16LE(compressionMethod, 10);
		centralHeader.writeUInt32LE(storedData.length, 20);
		centralHeader.writeUInt32LE(rawData.length, 24);
		centralHeader.writeUInt16LE(nameBytes.length, 28);
		centralHeader.writeUInt32LE(offset, 42);
		centralParts.push(centralHeader, nameBytes);

		offset += localHeader.length + nameBytes.length + storedData.length;
	}

	const localBuffer = Buffer.concat(localParts);
	const centralBuffer = Buffer.concat(centralParts);

	const endOfCentralDirectory = Buffer.alloc(22);
	endOfCentralDirectory.writeUInt32LE(0x06054b50, 0);
	endOfCentralDirectory.writeUInt16LE(entries.length, 8);
	endOfCentralDirectory.writeUInt16LE(entries.length, 10);
	endOfCentralDirectory.writeUInt32LE(centralBuffer.length, 12);
	endOfCentralDirectory.writeUInt32LE(localBuffer.length, 16);

	return Buffer.concat([localBuffer, centralBuffer, endOfCentralDirectory]);
}

const RENDER_MODULE_OK = `
export async function renderPage({ path, companyID, pageID }) {
	return {
		status: 200,
		html: '<html><head></head><body>' + path + '|' + companyID + '|' + pageID +
			'<script src="/_app/immutable/entry.js"></script></body></html>'
	};
}
`;

const RENDER_MODULE_FAILING = `
export async function renderPage() {
	return { status: 500, html: '' };
}
`;

function buildArtifact({ buildId = 'build-1', renderModule = RENDER_MODULE_OK } = {}) {
	return buildZip([
		{ name: 'manifest.json', data: JSON.stringify({ buildId, assetPathPrefix: '/_app/', htmlRewrites: [] }) },
		{ name: 'render.mjs', data: renderModule },
		{ name: 'assets/_app/immutable/entry.js', data: 'console.log(1)' },
		// Las dos formas en que un .js lleva dentro sus propias rutas de assets, que hay que
		// prefijar igual que las del HTML, y el helper de Vite que compone las relativas en runtime.
		// Ver applyAssetRewrites.
		{
			name: 'assets/_app/immutable/entry/app.js',
			data:
				'const P=function(e){return`/`+e};' +
				'const n=["_app/immutable/nodes/1.abc.js"];const w=new URL(`/_app/immutable/workers/w.js`,``+import.meta.url);'
		},
		{ name: 'assets/_app/immutable/style.css', data: 'body{color:red}' },
		{ name: 'site/sw.js', data: '// service worker' },
		{ name: 'site/favicon.ico', data: Buffer.from([0, 0, 1, 0]) }
	]);
}

// --- R2 y origen del artefacto simulados ------------------------------------------

const ACCOUNT = 'test-account';
const BUCKET = 'test-bucket';
const S3_BUCKET = 'test-storefront';

// Un solo mapa para R2 y S3, indexado por clave: las claves no se pisan (websites/ vs
// websites-html/) y cada objeto recuerda en qué almacén cayó (`store`).
async function startFakeCloudflare() {
	const objects = new Map();
	let zipBuffer = buildArtifact();
	let zipEtag = '"etag-1"';
	let zipRequests = 0;
	let respond304 = false;
	const failingStores = new Set();

	const server = createServer((request, response) => {
		const url = new URL(request.url, 'http://127.0.0.1');

		if (url.pathname === '/renderer.zip') {
			zipRequests++;
			// El handler manda if-none-match a partir de la segunda vez; el flag permite probar
			// los dos comportamientos del origen (304 real, y un origen que nunca lo emite).
			if (respond304 && request.headers['if-none-match'] === zipEtag) {
				response.writeHead(304).end();
				return;
			}
			response.writeHead(200, { etag: zipEtag, 'content-type': 'application/zip' });
			response.end(zipBuffer);
			return;
		}

		const r2Prefix = `/accounts/${ACCOUNT}/r2/buckets/${BUCKET}/objects/`;
		const s3Prefix = `/${S3_BUCKET}/`;
		const store = url.pathname.startsWith(r2Prefix) ? 'r2' : url.pathname.startsWith(s3Prefix) ? 's3' : null;
		if (!store) {
			response.writeHead(404).end('ruta inesperada');
			return;
		}
		const key = decodeURIComponent(url.pathname.slice((store === 'r2' ? r2Prefix : s3Prefix).length));

		if (request.method === 'PUT') {
			const chunks = [];
			request.on('data', (chunk) => chunks.push(chunk));
			request.on('end', () => {
				if (failingStores.has(store)) {
					response.writeHead(403).end('{"success":false}');
					return;
				}
				objects.set(key, {
					store,
					body: Buffer.concat(chunks),
					contentType: request.headers['content-type'],
					cacheControl: request.headers['cache-control'],
					authorization: request.headers.authorization || ''
				});
				response.writeHead(200, { 'content-type': 'application/json' }).end('{"success":true}');
			});
			return;
		}

		const stored = objects.get(key);
		if (!stored) {
			response.writeHead(404).end('Not Found');
			return;
		}
		response.writeHead(200).end(stored.body);
	});

	await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
	const baseUrl = `http://127.0.0.1:${server.address().port}`;

	return {
		objects,
		baseUrl,
		zipUrl: `${baseUrl}/renderer.zip`,
		get zipRequests() {
			return zipRequests;
		},
		setArtifact(buffer, etag) {
			zipBuffer = buffer;
			zipEtag = etag;
		},
		enable304() {
			respond304 = true;
		},
		rejectPuts(store) {
			failingStores.add(store);
		},
		async close() {
			await new Promise((resolve) => server.close(resolve));
		}
	};
}

let moduleCounter = 0;
let openServers = [];

// Instancia fresca del handler: las constantes de entorno se fijan al cargar el módulo, así que
// cambiarlas después no tendría efecto sin volver a importarlo.
async function loadHandler(cloudflare) {
	process.env.RENDERER_ZIP_URL = cloudflare.zipUrl;
	process.env.FRONTEND_CDN = 'https://cdn.example.com/';
	process.env.CLOUDFLARE_ACCOUNT = ACCOUNT;
	process.env.CLOUDFLARE_TOKEN = 'token-de-prueba';
	process.env.CLOUDFLARE_BUCKET = BUCKET;
	process.env.CLOUDFLARE_API_BASE = cloudflare.baseUrl;
	process.env.STOREFRONT_BUCKET = S3_BUCKET;
	process.env.STOREFRONT_S3_ENDPOINT = cloudflare.baseUrl;
	process.env.AWS_REGION = 'us-east-1';
	process.env.AWS_ACCESS_KEY_ID = 'AKIDPRUEBA';
	process.env.AWS_SECRET_ACCESS_KEY = 'secreto-de-prueba';
	delete process.env.AWS_SESSION_TOKEN;
	return import(`./handler.mjs?instance=${++moduleCounter}`);
}

async function setup() {
	const cloudflare = await startFakeCloudflare();
	openServers.push(cloudflare);
	const handler = await loadHandler(cloudflare);
	return { cloudflare, handler };
}

afterEach(async () => {
	for (const server of openServers) await server.close();
	openServers = [];
});

const BASE_EVENT = {
	companyID: 7,
	hostname: 'tienda.un.pe',
	pages: [
		{ id: 10, path: '/' },
		{ id: 11, path: '/about' }
	]
};

// --- Render completo --------------------------------------------------------------

test('publica un index.html por página bajo el hostname', async () => {
	const { cloudflare, handler } = await setup();

	const result = await handler.render(BASE_EVENT);

	expect(result.pages).toBe(2);
	expect(result.buildId).toBe('build-1');
	expect(cloudflare.objects.has('websites-html/tienda.un.pe/index.html')).toBe(true);
	expect(cloudflare.objects.has('websites-html/tienda.un.pe/about/index.html')).toBe(true);

	const home = cloudflare.objects.get('websites-html/tienda.un.pe/index.html');
	expect(home.contentType).toBe('text/html; charset=utf-8');
	expect(home.cacheControl).toBe('public, max-age=60');
	// La ruta y los identificadores del tenant llegaron al render.
	expect(home.body.toString('utf8')).toContain('/|7|10');
});

test('reescribe el prefijo de assets al CDN de la company', async () => {
	const { cloudflare, handler } = await setup();

	await handler.render(BASE_EVENT);

	const html = cloudflare.objects.get('websites-html/tienda.un.pe/index.html').body.toString('utf8');
	expect(html).toContain('https://cdn.example.com/websites/7/_app/immutable/entry.js');
	// El prefijo desnudo no debe sobrevivir: sería una petición al origen del sitio.
	expect(html).not.toContain('src="/_app/');
});

test('separa los assets del CDN de los archivos que van al origen del sitio', async () => {
	const { cloudflare, handler } = await setup();

	await handler.render(BASE_EVENT);

	const asset = cloudflare.objects.get('websites/7/_app/immutable/entry.js');
	expect(asset.cacheControl).toBe('public, max-age=31536000, immutable');
	expect(asset.contentType).toBe('text/javascript; charset=utf-8');

	// sw.js gobierna las actualizaciones del sitio, así que no puede cachearse, y tiene que ser
	// same-origin: va bajo el hostname, no bajo el CDN.
	const serviceWorker = cloudflare.objects.get('websites-html/tienda.un.pe/sw.js');
	expect(serviceWorker.cacheControl).toBe('public, max-age=0, must-revalidate');

	expect(cloudflare.objects.has('websites-html/tienda.un.pe/favicon.ico')).toBe(true);
	expect(cloudflare.objects.get('websites/7/.renderer-build').body.toString('utf8')).toBe('build-1');
});

// --- Republicación ----------------------------------------------------------------

test('no resube los assets cuando el marcador ya tiene ese buildId', async () => {
	const { cloudflare, handler } = await setup();

	const first = await handler.render(BASE_EVENT);
	expect(first.assets).toBeGreaterThan(0);

	cloudflare.objects.delete('websites/7/_app/immutable/entry.js');
	const second = await handler.render(BASE_EVENT);

	// Los site files se suben siempre, así que la cuenta no baja a cero: son ellos y nada más.
	expect(second.assets).toBe(2);
	// Sigue borrado: la segunda pasada no tocó los assets.
	expect(cloudflare.objects.has('websites/7/_app/immutable/entry.js')).toBe(false);
	// El HTML sí se reescribe siempre.
	expect(cloudflare.objects.has('websites-html/tienda.un.pe/index.html')).toBe(true);
});

test('forceAssets resube los assets aunque el marcador coincida', async () => {
	const { cloudflare, handler } = await setup();

	await handler.render(BASE_EVENT);
	cloudflare.objects.delete('websites/7/_app/immutable/entry.js');
	const forced = await handler.render({ ...BASE_EVENT, forceAssets: true });

	expect(forced.assets).toBeGreaterThan(0);
	expect(cloudflare.objects.has('websites/7/_app/immutable/entry.js')).toBe(true);
});

// --- Artefacto --------------------------------------------------------------------

test('reusa el artefacto cuando el origen responde 304', async () => {
	const { cloudflare, handler } = await setup();
	cloudflare.enable304();

	await handler.render(BASE_EVENT);
	await handler.render(BASE_EVENT);

	// Dos peticiones al origen, pero la segunda fue un 304 y no volvió a extraer nada.
	expect(cloudflare.zipRequests).toBe(2);
});

test('reusa el artefacto cuando el origen repite el ETag sin emitir 304', async () => {
	const { cloudflare, handler } = await setup();

	const first = await handler.render(BASE_EVENT);
	const second = await handler.render(BASE_EVENT);

	expect(second.buildId).toBe(first.buildId);
	expect(cloudflare.zipRequests).toBe(2);
});

test('adopta un artefacto nuevo cuando cambia el ETag', async () => {
	const { cloudflare, handler } = await setup();

	await handler.render(BASE_EVENT);
	cloudflare.setArtifact(buildArtifact({ buildId: 'build-2' }), '"etag-2"');
	const republished = await handler.render(BASE_EVENT);

	expect(republished.buildId).toBe('build-2');
	expect(republished.assets).toBeGreaterThan(0);
	expect(cloudflare.objects.get('websites/7/.renderer-build').body.toString('utf8')).toBe('build-2');
});

test('acepta entradas comprimidas con deflate', async () => {
	const cloudflare = await startFakeCloudflare();
	openServers.push(cloudflare);
	cloudflare.setArtifact(
		buildZip(
			[
				{ name: 'manifest.json', data: JSON.stringify({ buildId: 'deflate-1', assetPathPrefix: '/_app/', htmlRewrites: [] }) },
				{ name: 'render.mjs', data: RENDER_MODULE_OK },
				{ name: 'assets/_app/immutable/entry.js', data: 'console.log(1)'.repeat(50) }
			],
			{ deflate: true }
		),
		'"etag-deflate"'
	);
	const handler = await loadHandler(cloudflare);

	const result = await handler.render(BASE_EVENT);

	expect(result.buildId).toBe('deflate-1');
	expect(cloudflare.objects.get('websites/7/_app/immutable/entry.js').body.toString('utf8')).toBe(
		'console.log(1)'.repeat(50)
	);
});

test('falla cuando el artefacto no trae manifest.json', async () => {
	const cloudflare = await startFakeCloudflare();
	openServers.push(cloudflare);
	cloudflare.setArtifact(buildZip([{ name: 'render.mjs', data: RENDER_MODULE_OK }]), '"etag-sin-manifest"');
	const handler = await loadHandler(cloudflare);

	await expect(handler.render(BASE_EVENT)).rejects.toThrow(/manifest\.json/);
});

// --- Validación del evento --------------------------------------------------------

test('rechaza eventos inválidos antes de tocar la red', async () => {
	const { cloudflare, handler } = await setup();

	await expect(handler.render({ ...BASE_EVENT, companyID: 0 })).rejects.toThrow(/companyID/);
	await expect(handler.render({ ...BASE_EVENT, hostname: 'sin-punto' })).rejects.toThrow(/hostname/);
	await expect(handler.render({ ...BASE_EVENT, pages: [] })).rejects.toThrow(/ninguna página/);
	expect(cloudflare.zipRequests).toBe(0);
});

test('exige las variables de entorno de R2 y de S3', async () => {
	const { handler } = await setup();

	for (const variableName of ['CLOUDFLARE_BUCKET', 'STOREFRONT_BUCKET', 'AWS_SECRET_ACCESS_KEY']) {
		const saved = process.env[variableName];
		delete process.env[variableName];
		try {
			await expect(handler.render(BASE_EVENT)).rejects.toThrow(new RegExp(variableName));
		} finally {
			process.env[variableName] = saved;
		}
	}
});

test('aborta cuando una página no devuelve 200', async () => {
	const cloudflare = await startFakeCloudflare();
	openServers.push(cloudflare);
	cloudflare.setArtifact(buildArtifact({ buildId: 'falla-1', renderModule: RENDER_MODULE_FAILING }), '"etag-falla"');
	const handler = await loadHandler(cloudflare);

	await expect(handler.render(BASE_EVENT)).rejects.toThrow(/HTTP 500/);
	expect(cloudflare.objects.has('websites-html/tienda.un.pe/index.html')).toBe(false);
});

test('propaga un fallo de R2 en vez de darlo por bueno', async () => {
	const { cloudflare, handler } = await setup();
	// Desde el servidor y no cambiando process.env: el bucket y el token se capturan en
	// constantes al cargar el módulo, así que tocarlos en caliente no tendría ningún efecto.
	cloudflare.rejectPuts('r2');

	await expect(handler.render(BASE_EVENT)).rejects.toThrow(/R2 PUT/);
	expect([...cloudflare.objects.values()].some((object) => object.store === 'r2')).toBe(false);
});

test('propaga un fallo de S3 en vez de darlo por bueno', async () => {
	const { cloudflare, handler } = await setup();
	cloudflare.rejectPuts('s3');

	await expect(handler.render(BASE_EVENT)).rejects.toThrow(/S3 PUT/);
	expect(cloudflare.objects.has('websites-html/tienda.un.pe/index.html')).toBe(false);
});

// --- Destinos: HTML a S3, assets a R2 ---------------------------------------------

test('el HTML y los site files van a S3 firmados; los assets y el marcador a R2', async () => {
	const { cloudflare, handler } = await setup();
	await handler.render(BASE_EVENT);

	for (const key of [
		'websites-html/tienda.un.pe/index.html',
		'websites-html/tienda.un.pe/about/index.html',
		'websites-html/tienda.un.pe/sw.js',
		'websites-html/tienda.un.pe/favicon.ico'
	]) {
		const object = cloudflare.objects.get(key);
		expect(object.store).toBe('s3');
		expect(object.authorization).toStartWith('AWS4-HMAC-SHA256 Credential=AKIDPRUEBA/');
	}
	expect(cloudflare.objects.get('websites/7/_app/immutable/entry.js').store).toBe('r2');
	expect(cloudflare.objects.get('websites/7/.renderer-build').store).toBe('r2');
});

// Vector oficial de AWS ("PUT Object", Signature Version 4, header-based auth): si la firma
// cambia, S3 rechaza todos los PUT con 403.
test('signS3Request reproduce el vector de ejemplo de AWS', async () => {
	const { handler } = await setup();
	const payload = Buffer.from('Welcome to Amazon S3.', 'utf8');

	const headers = handler.signS3Request({
		method: 'PUT',
		url: new URL(`https://examplebucket.s3.amazonaws.com/${handler.encodeS3Key('test$file.text')}`),
		payload,
		headers: { date: 'Fri, 24 May 2013 00:00:00 GMT', 'x-amz-storage-class': 'REDUCED_REDUNDANCY' },
		credentials: { accessKeyId: 'AKIAIOSFODNN7EXAMPLE', secretAccessKey: 'wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY' },
		region: 'us-east-1',
		now: new Date('2013-05-24T00:00:00Z')
	});

	expect(headers['x-amz-content-sha256']).toBe('44ce7dd67c959e0d3524ffac1771dfbba87d2b6b4b4e99e42034a8b803f8b072');
	expect(headers.authorization).toBe(
		'AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20130524/us-east-1/s3/aws4_request, ' +
			'SignedHeaders=date;host;x-amz-content-sha256;x-amz-date;x-amz-storage-class, ' +
			'Signature=98ad721746da40c64f1a55b78f14c238d841ea1380cd77a1b5971af0ece108bd'
	);
});

// --- Funciones puras --------------------------------------------------------------

test('normalizePagePath deja las rutas en la forma con la que se construye la clave', async () => {
	const { handler } = await setup();

	expect(handler.normalizePagePath('/')).toBe('/');
	expect(handler.normalizePagePath('')).toBe('/');
	expect(handler.normalizePagePath(undefined)).toBe('/');
	expect(handler.normalizePagePath('about')).toBe('/about');
	expect(handler.normalizePagePath('/about/')).toBe('/about');
	expect(handler.normalizePagePath('//nosotros//')).toBe('/nosotros');
	expect(handler.normalizePagePath('/tienda/zapatos')).toBe('/tienda/zapatos');

	expect(() => handler.normalizePagePath('/a b')).toThrow(/ruta de página inválida/);
	expect(() => handler.normalizePagePath('/a?b=1')).toThrow(/ruta de página inválida/);
	expect(() => handler.normalizePagePath('/../secreto')).toThrow(/ruta de página inválida/);
});

test('applyHtmlRewrites aplica el manifest y antepone la base del CDN', async () => {
	const { handler } = await setup();
	const manifest = {
		assetPathPrefix: '/_app/',
		htmlRewrites: [{ find: '<!--marcador-->', replace: '<link rel="stylesheet" href="/_app/merged.css">', required: true }]
	};

	const output = handler.applyHtmlRewrites('<head><!--marcador--></head>', manifest, 'https://cdn/websites/7');

	expect(output).toBe('<head><link rel="stylesheet" href="https://cdn/websites/7/_app/merged.css"></head>');
});

test('applyHtmlRewrites falla cuando una regla required no casa', async () => {
	const { handler } = await setup();
	const manifest = {
		assetPathPrefix: '/_app/',
		htmlRewrites: [{ find: '<!--ausente-->', replace: '', required: true }]
	};

	expect(() => handler.applyHtmlRewrites('<head></head>', manifest, 'https://cdn')).toThrow(/no casa/);
	// Sin required, una regla que no casa es un no-op.
	manifest.htmlRewrites[0].required = false;
	expect(handler.applyHtmlRewrites('<head></head>', manifest, 'https://cdn')).toBe('<head></head>');
});

test('readZipEntries lee entradas stored y deflate, y salta directorios', async () => {
	const { handler } = await setup();
	const stored = handler.readZipEntries(buildZip([{ name: 'a/', data: '' }, { name: 'a/b.txt', data: 'hola' }]));

	expect(stored.map((entry) => entry.name)).toEqual(['a/b.txt']);
	expect(stored[0].data.toString('utf8')).toBe('hola');

	const deflated = handler.readZipEntries(buildZip([{ name: 'c.txt', data: 'x'.repeat(200) }], { deflate: true }));
	expect(deflated[0].data.toString('utf8')).toBe('x'.repeat(200));
});

test('readZipEntries falla sin End of Central Directory', async () => {
	const { handler } = await setup();

	expect(() => handler.readZipEntries(Buffer.alloc(64))).toThrow(/End of Central Directory/);
});

test('contentTypeFor cubre los tipos que se publican', async () => {
	const { handler } = await setup();

	expect(handler.contentTypeFor('a/b.js')).toBe('text/javascript; charset=utf-8');
	expect(handler.contentTypeFor('a/b.css')).toBe('text/css; charset=utf-8');
	expect(handler.contentTypeFor('favicon.ico')).toBe('image/x-icon');
	expect(handler.contentTypeFor('x.json')).toBe('application/json');
	expect(handler.contentTypeFor('x.bin')).toBe('application/octet-stream');
});

// --- Rutas de assets dentro de los .js --------------------------------------------

test('prefija las rutas que los .js llevan dentro, en sus dos formas', async () => {
	const { cloudflare, handler } = await setup();
	await handler.render(BASE_EVENT);

	const bundle = cloudflare.objects.get('websites/7/_app/immutable/entry/app.js').body.toString('utf8');

	// Relativa: el runtime la concatenaba con una base vacía y acababa pidiéndola al hostname.
	expect(bundle).toContain('"https://cdn.example.com/websites/7/_app/immutable/nodes/1.abc.js"');
	// Absoluta: se resolvía contra la raíz del CDN, sin el websites/<companyID>.
	expect(bundle).toContain('`https://cdn.example.com/websites/7/_app/immutable/workers/w.js`');
	// Y no queda ninguna sin prefijar.
	expect(bundle).not.toContain('"_app/');
	expect(bundle).not.toContain('`/_app/');
});

test('deja el helper assetsURL de Vite en identidad', async () => {
	const { cloudflare, handler } = await setup();
	await handler.render(BASE_EVENT);

	const bundle = cloudflare.objects.get('websites/7/_app/immutable/entry/app.js').body.toString('utf8');

	// Con los deps ya absolutos, anteponer la base del build los mandaría al hostname de la tienda
	// como '/https://cdn…'.
	expect(bundle).toContain('const P=function(e){return e};');
	expect(bundle).not.toContain('return`/`+e');
});

test('aborta si hay deps relativos pero no aparece el helper assetsURL', async () => {
	const cloudflare = await startFakeCloudflare();
	openServers.push(cloudflare);
	// Es lo que pasaría si Vite cambiara la forma del helper: la regex dejaría de casar y el sitio
	// se publicaría pidiendo las rutas al origen equivocado.
	cloudflare.setArtifact(
		buildZip([
			{ name: 'manifest.json', data: JSON.stringify({ buildId: 'sin-helper', assetPathPrefix: '/_app/', htmlRewrites: [] }) },
			{ name: 'render.mjs', data: RENDER_MODULE_OK },
			{ name: 'assets/_app/immutable/entry/app.js', data: 'const n=["_app/immutable/nodes/1.abc.js"];' }
		]),
		'"etag-sin-helper"'
	);
	const handler = await loadHandler(cloudflare);

	await expect(handler.render(BASE_EVENT)).rejects.toThrow(/assetsURL/);
});

test('no reescribe los assets que no son .js', async () => {
	const { cloudflare, handler } = await setup();
	await handler.render(BASE_EVENT);

	expect(cloudflare.objects.get('websites/7/_app/immutable/style.css').body.toString('utf8')).toBe(
		'body{color:red}'
	);
});

test('applyAssetRewrites no toca el buffer original', async () => {
	const handler = await loadHandler(await startFakeCloudflare().then((c) => (openServers.push(c), c)));
	const manifest = { assetPathPrefix: '/_app/' };
	const original = Buffer.from('const n=["_app/immutable/nodes/1.abc.js"];', 'utf8');
	const copy = Buffer.from(original);

	const rewritten = handler.applyAssetRewrites(original, 'entry/app.js', manifest, 'https://cdn/websites/7');

	// El artefacto se cachea en memoria y lo comparten todas las companies del proceso: mutarlo
	// haría que la segunda se publicara con la base de la primera.
	expect(original.equals(copy)).toBe(true);
	expect(rewritten.data.toString('utf8')).toContain('"https://cdn/websites/7/_app/immutable/nodes/1.abc.js"');
});

// --- Site files por hostname ------------------------------------------------------

test('un dominio nuevo recibe sw.js y favicon aunque la company ya tenga los assets al día', async () => {
	const { cloudflare, handler } = await setup();

	await handler.render(BASE_EVENT);
	// Mismo company, otro hostname: el marcador sigue al día, pero los site files cuelgan del
	// hostname y ahí no hay nada todavía.
	const second = await handler.render({ ...BASE_EVENT, hostname: 'otra.un.pe' });

	expect(second.assets).toBe(2);
	expect(cloudflare.objects.has('websites-html/otra.un.pe/sw.js')).toBe(true);
	expect(cloudflare.objects.has('websites-html/otra.un.pe/favicon.ico')).toBe(true);
});
