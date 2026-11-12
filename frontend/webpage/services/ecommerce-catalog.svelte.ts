// The web store catalog, one reactive instance for the whole storefront. No delta cache and no
// service worker: everything runs on the main thread. docs/ECOMMERCE_CATALOG_SNAPSHOT_PLAN.md.
//
// Page load:
//   1. The catalog saved in IndexedDB is published (first paint, no network). With nothing saved the
//      R2 file (live/ecommerce-c<cid>.cb, colbin) is fetched and published instead.
//   2. In the background, one delta request brings the catalog from its cursor to now. When the
//      backend answers UseFile (stale generation, large catch-up) the file is fetched again, busting
//      the CDN cache with its Created, and the delta is asked once more from the file's Updated.
//   3. The result is published and saved.
// Two tabs loading at once each run this; the last write wins, and both values are valid.
import { browser } from '$app/env';
import { arrayToMapN } from '#libs/helpers.ts';
import { readIndexedDBValue, writeIndexedDBValue } from '#libs/indexeddb-value.ts';
import { Env } from '#core/env.ts';
import { COLBIN_ACCEPT, decodeColbinBody, isColbinContentType } from '@genix/ui/http';
import {
	groupProductsByCategory, isUsableSavedCatalog, mergeCatalogDelta, savedCatalogFromFile, toStoreProduct,
	type ICatalogDelta, type ICatalogFile, type ICatalogListRecord, type IProduct, type IProductCategory,
	type ISavedCatalog,
} from './ecommerce-catalog.ts';

const CATALOG_DATABASE_NAME = 'genix-ecommerce';
const CATALOG_DELTA_ROUTE = 'p-ecommerce-catalog';

const savedCatalogKey = (): string => `ecommerce-catalog-c${Env.getCompanyID()}`;

// Each publish reassigns the $state fields, so anything reading them re-renders when the delta lands.
export class EcommerceCatalog {
	productos = $state<IProduct[]>([]);
	marcas = $state<ICatalogListRecord[]>([]);
	categorias = $state<IProductCategory[]>([]);
	productosMap = $state<Map<number, IProduct>>(new Map());
	categoriasMap = $state<Map<number, IProductCategory>>(new Map());
	productosByCategoryMap = $state<Map<number, IProduct[]>>(new Map());
	// Bumped on every publish; watch it from a $effect when a non-template consumer (the search index)
	// must recompute.
	version = $state(0);

	getProduct(id: number): IProduct | undefined {
		return this.productosMap.get(id);
	}

	getCategory(id: number): IProductCategory | undefined {
		return this.categoriasMap.get(id);
	}

	getProductsByCategory(id: number): IProduct[] {
		return this.productosByCategoryMap.get(id) ?? [];
	}

	publish(saved: ISavedCatalog): void {
		this.productos = saved.productos.map(toStoreProduct);
		this.marcas = saved.marcas;
		this.categorias = saved.categorias;
		this.productosMap = arrayToMapN(this.productos, 'ID');
		this.categoriasMap = arrayToMapN(this.categorias, 'ID');
		this.productosByCategoryMap = groupProductsByCategory(this.productos);
		this.version++;
	}
}

const decodeResponse = async <T>(response: Response): Promise<T> =>
	isColbinContentType(response.headers.get('Content-Type'))
		? decodeColbinBody(new Uint8Array(await response.arrayBuffer()))
		: response.json();

// The R2 file, or null when there is none yet (first visit after deploy: the delta builds it).
// fileCreated busts the CDN and browser cache when the backend names a file newer than a cached one.
const fetchCatalogFile = async (fileCreated?: number): Promise<ICatalogFile | null> => {
	const fileRoute = Env.makeCDNRoute('live', `ecommerce-c${Env.getCompanyID()}.cb`);
	const response = await fetch(fileCreated ? `${fileRoute}?v=${fileCreated}` : fileRoute);
	if (!response.ok) {
		console.warn('[EcommerceCatalog] file unavailable:', response.status);
		return null;
	}
	return decodeColbinBody(new Uint8Array(await response.arrayBuffer()));
};

// With nothing saved, since 0 tells the backend to answer UseFile (building the file if missing).
const fetchCatalogDelta = async (saved: ISavedCatalog | null): Promise<ICatalogDelta> => {
	const route = `${CATALOG_DELTA_ROUTE}?since=${saved?.since ?? 0}&gen=${saved?.generation ?? 0}`;
	const response = await fetch(Env.makeRoute(route), { headers: { Accept: COLBIN_ACCEPT } });
	if (!response.ok) {
		let message = `Error ${response.status}`;
		try { message = (await response.json())?.error || message; } catch { /* not JSON */ }
		throw new Error(message);
	}
	return decodeResponse<ICatalogDelta>(response);
};

// Step 2 and 3 of the load: brings the published catalog to now and saves it.
const syncCatalog = async (catalog: EcommerceCatalog, saved: ISavedCatalog | null): Promise<void> => {
	let delta = await fetchCatalogDelta(saved);
	if (delta.UseFile) {
		const catalogFile = await fetchCatalogFile(delta.FileCreated);
		if (!catalogFile) throw new Error('The catalog file named by the backend is not on the CDN');
		saved = savedCatalogFromFile(catalogFile);
		catalog.publish(saved);
		delta = await fetchCatalogDelta(saved);
	}
	// A second UseFile right after a fresh file means it was rebuilt meanwhile: the next load takes it.
	if (saved && !delta.UseFile) {
		const hasChanges = !!(delta.Productos?.length || delta.Marcas?.length || delta.Categorias?.length);
		saved = mergeCatalogDelta(saved, delta);
		if (hasChanges) catalog.publish(saved);
	}
	if (saved) await writeIndexedDBValue(CATALOG_DATABASE_NAME, savedCatalogKey(), saved);
};

// Step 1 of the load: what can be shown before any API call.
const readInitialCatalog = async (): Promise<ISavedCatalog | null> => {
	try {
		const saved = await readIndexedDBValue<ISavedCatalog>(CATALOG_DATABASE_NAME, savedCatalogKey());
		if (isUsableSavedCatalog(saved)) return saved;
	} catch (indexedDBError) {
		console.warn('[EcommerceCatalog] IndexedDB read failed:', indexedDBError);
	}
	const catalogFile = await fetchCatalogFile().catch((fileError) => {
		console.warn('[EcommerceCatalog] file fetch failed:', fileError);
		return null;
	});
	return catalogFile ? savedCatalogFromFile(catalogFile) : null;
};

const ecommerceCatalog = new EcommerceCatalog();
let catalogLoadPromise: Promise<EcommerceCatalog> | null = null;

/** The shared catalog; resolves once the first list is published, the delta lands afterwards. */
export const getEcommerceCatalog = (): Promise<EcommerceCatalog> => {
	catalogLoadPromise ??= (async () => {
		if (!browser) return ecommerceCatalog;
		const initialCatalog = await readInitialCatalog();
		if (initialCatalog) ecommerceCatalog.publish(initialCatalog);
		void syncCatalog(ecommerceCatalog, initialCatalog).catch((syncError) => {
			console.error('[EcommerceCatalog] delta sync failed', syncError);
		});
		return ecommerceCatalog;
	})();
	return catalogLoadPromise;
};
