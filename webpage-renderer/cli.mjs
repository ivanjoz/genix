#!/usr/bin/env node
// Entrada CLI del renderer, para cuando no se está en Lambda.
//
// La usa backend/cloud/webpage_renderer.go: fuera de un entorno serverless el backend no
// puede hacer un Invoke de AWS, así que ejecuta este archivo con `node cli.mjs` y habla con
// él por los descriptores estándar. El handler no cambia entre los dos caminos.
//
//   stdin   el mismo JSON que recibe el Lambda: { companyID, hostname, pages, forceAssets? }
//   stdout  SOLO el resultado en JSON: { buildId, pages, assets, bytes }
//   stderr  los logs
//   exit    0 correcto, 1 error (el mensaje va por stderr)
//
// El desvío de console.log a stderr es la razón de que este wrapper exista: el handler loguea
// una línea por página, y si eso cayera en stdout el backend no podría deserializar la
// respuesta sin filtrar líneas.

console.log = (...args) => console.error(...args);
console.info = (...args) => console.error(...args);

const { render } = await import('./handler.mjs');

async function readStdin() {
	const chunks = [];
	for await (const chunk of process.stdin) chunks.push(chunk);
	return Buffer.concat(chunks).toString('utf8');
}

const rawEvent = (await readStdin()).trim();
if (!rawEvent) {
	console.error('[renderer-cli] no se recibió ningún evento por stdin');
	process.exit(1);
}

let event;
try {
	event = JSON.parse(rawEvent);
} catch (parseError) {
	console.error(`[renderer-cli] evento ilegible: ${parseError.message}`);
	process.exit(1);
}

try {
	const result = await render(event);
	process.stdout.write(JSON.stringify(result));
} catch (renderError) {
	// El stack va completo a stderr: es el único rastro que tendrá quien ejecute esto en local.
	console.error(`[renderer-cli] ${renderError?.stack || renderError}`);
	process.exit(1);
}
