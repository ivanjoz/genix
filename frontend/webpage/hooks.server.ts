import { building } from '$app/environment';
import { Env } from '$core/env';
import type { Handle } from '@sveltejs/kit';

// El Lambda de render pide cada página como '/<ruta>?cid=<companyID>&pid=<pageID>'.
//
// Un único bundle SSR sirve a TODAS las companies (antes había un build por company, con
// el id inlineado por Vite), así que el tenant se resuelve aquí por request y se deja
// escrito en el HTML como meta para que el cliente resuelva exactamente lo mismo tras la
// hidratación. Sin esto, el HTML servido y el bundle no coincidirían en de qué empresa es
// la página.
//
// IMPORTANTE: Env es un singleton de módulo, así que un proceso solo puede renderizar UNA
// página a la vez. El handler del Lambda renderiza en secuencia por ese motivo.
export const handle: Handle = async ({ event, resolve }) => {
	// Al generar el shell SPA del build, SvelteKit prohíbe leer searchParams: el HTML
	// emitido no puede depender del query. Ese shell no pertenece a ninguna company, así
	// que las metas quedan en 0 y el cliente resuelve el tenant por su cuenta.
	const companyID = building ? 0 : Number(event.url.searchParams.get('cid')) || 0;
	const pageID = building ? 0 : Number(event.url.searchParams.get('pid')) || 0;

	Env.companyID = companyID;
	Env.pageID = pageID;

	return resolve(event, {
		transformPageChunk: ({ html }) =>
			html
				.replaceAll('%genix.companyId%', String(companyID))
				.replaceAll('%genix.pageId%', String(pageID))
	});
};
