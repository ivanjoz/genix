// Storefront product catalog — a single reactive source for the whole ecommerce module.
//
// getProductEcommerceData() returns one memoized, reactive ProductCatalog (Svelte 5 $state).
// Every consumer does `const catalog = await getProductEcommerceData()` and then reads
// `catalog.productos` / `catalog.getProduct(id)` / `catalog.getProductsByCategory(id)` etc.
// directly — no helper functions, no separate state object.
//
// The load is done MANUALLY on the main thread (no GetHandler, no service worker round-trip) so
// the first product list paints as early as possible:
//   Phase 1 (first paint): cold → fetch products-c<companyID>.db once and parse it in memory;
//                           warm (<1h, tracked in localStorage) → read the cached tables from
//                           IndexedDB. The promise resolves here.
//   Phase 2 (background):  run the standard delta cache (fetchDeltaCache) directly on the main
//                          thread to persist into IndexedDB and apply the server delta; if the
//                          records changed, the catalog state is re-published and components
//                          re-render. Cold passes the already-downloaded bytes so the .db file is
//                          never fetched twice.
import { browser } from '$app/environment';
import { arrayToMapN } from '$libs/helpers';
import { buildHeaders } from '$libs/http.svelte';
import { Env } from '$core/env';
import { parsePsvResponse } from '$libs/cache/psv-parse';
import { fetchDeltaCache, readDeltaCacheSubObject } from '$libs/cache/delta-cache.fetch';
import type { serviceHttpProps } from '$libs/workers/service-worker';
import type { ISharedListRecord } from '$services/negocio/listas-compartidas.svelte';

export interface IProductProperty {
  id: number, nm: string, ss: number
}

export interface IProductProperties {
  ID: number, Name: string, Options: IProductProperty[], Status: number
}

export interface IProductoImage {
  n: string /* Nombre del imagen */
  d: string /* descripcion de la imagen */
}

export interface IProduct {
  ID: number,
  Name: string
  Description: string
  Price?: number
  Discount?: number
  FinalPrice?: number
  ContentHTML?: string
  Properties?: IProductProperties[]
  Peso?: number
  Volumen?: number
  SbuQuantity?: number
  SbuUnit?: string
  SbuPrice?: number
  SbuDiscount?: number
  SbuFinalPrice?: number
  Images?: IProductoImage[]
	Image?: IProductoImage
  BrandID?: number
  CategoryIDs?: number[]
  Stock?: { a /* almacen */: number, c /* cantidad */: number }[]
  ss: number
  upd: number
  _stock?: number
  _moneda?: string
}

export interface IProductCategory {
  ID: number,
  Description: string,
  Name: string,
}

// Column order MUST match what the backend writes into the .db snapshot (no header row).
const PRODUCT_ECOMMERCE_FILE_SCHEMA = {
	productos: ["ID:N", "Name:T", "CategoryIDs:AN", "BrandID:N", "Price:N", "FinalPrice:N", "Image:T", "upd:N", "ss:N"],
	marcas: ["ID:N", "Name:T", "upd:N", "ss:N"],
	categorias: ["ID:N", "Name:T", "upd:N", "ss:N"]
};

const CATALOG_ROUTE = "p-productos-ecommerce";
const CATALOG_MODULE = "a";
const CATALOG_CACHE = { min: 30, ver: 1 };
// Window during which a localStorage watermark means "the product table is fresh enough in
// IndexedDB", so we can skip the file fetch and read straight from IDB.
const FRESH_WINDOW_SECONDS = 3600;

interface ICatalogTables {
	productos?: IProduct[];
	marcas?: ISharedListRecord[];
	categorias?: IProductCategory[];
}

// The reactive catalog. Each publish reassigns the $state fields (new arrays/maps), so any
// component reading them in a $derived/template re-renders on the Phase 1 → Phase 2 transition.
export class ProductCatalog {
	productos = $state<IProduct[]>([]);
	marcas = $state<ISharedListRecord[]>([]);
	categorias = $state<IProductCategory[]>([]);
	productosMap = $state<Map<number, IProduct>>(new Map());
	categoriasMap = $state<Map<number, IProductCategory>>(new Map());
	productosByCategoryMap = $state<Map<number, IProduct[]>>(new Map());
	// Bumped on every publish; watch it from a $effect when a non-template consumer (e.g. the
	// search index) must recompute after the delta lands.
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
}

// Keep only active products and normalize the image into the {n,d} shape ProductCard expects.
// The .db snapshot carries a flat image-name string (Image); the server delta carries the full
// Images array — handle both.
const normalizeProductos = (rows: IProduct[]): IProduct[] => {
	const active = (rows ?? []).filter((record) => (record.ss ?? 0) > 0);
	for (const product of active) {
		const rawImage = product.Image as unknown;
		const imageName = (typeof rawImage === "string" ? rawImage : product.Image?.n) || product.Images?.[0]?.n || "";
		product.Image = { n: imageName, d: "" };
	}
	return active;
};

// A cheap fingerprint (active count + max watermark) to decide whether the Phase 2 delta changed
// anything worth re-rendering for.
const productosSignature = (productos: IProduct[]): string => {
	let maxUpd = 0;
	for (const product of productos) {
		if ((product.upd ?? 0) > maxUpd) maxUpd = product.upd ?? 0;
	}
	return `${productos.length}:${maxUpd}`;
};

// Reassign the catalog state from a set of (possibly raw) tables.
const publish = (catalog: ProductCatalog, tables: ICatalogTables): void => {
	const productos = normalizeProductos(tables.productos ?? []);
	const marcas = (tables.marcas ?? []).filter((record) => (record.ss ?? 0) > 0);
	const categorias = (tables.categorias ?? []).filter((record) => ((record as ISharedListRecord).ss ?? 0) > 0);

	const productosByCategoryMap = new Map<number, IProduct[]>();
	for (const producto of productos) {
		for (const categoriaId of producto.CategoryIDs ?? []) {
			let bucket = productosByCategoryMap.get(categoriaId);
			if (!bucket) {
				bucket = [];
				productosByCategoryMap.set(categoriaId, bucket);
			}
			bucket.push(producto);
		}
	}

	catalog.productos = productos;
	catalog.marcas = marcas;
	catalog.categorias = categorias;
	catalog.productosMap = arrayToMapN(productos, 'ID');
	catalog.categoriasMap = arrayToMapN(categorias as unknown as IProductCategory[], 'ID');
	catalog.productosByCategoryMap = productosByCategoryMap;
	catalog.version++;
};

const watermarkKey = (): string => `pe-fetch-c${Env.getCompanyID()}`;

const readLastFetchUnix = (): number => {
	try {
		return parseInt(localStorage.getItem(watermarkKey()) || "0", 10) || 0;
	} catch {
		return 0;
	}
};

const writeLastFetchUnix = (): void => {
	try {
		localStorage.setItem(watermarkKey(), String(Math.floor(Date.now() / 1000)));
	} catch {
		/* storage unavailable — warm path just falls back to the file fetch next time */
	}
};

const snapshotFileRoute = (): string =>
	Env.makeCDNRoute("live", `products-c${Env.getCompanyID()}.db`);

// Props for the main-thread delta-cache call. fetchDeltaCache reads/writes the same IndexedDB the
// service worker uses, so persistence + delta stay in the standard format.
// When `fileMissing` is set (the .db snapshot 404s — a valid scenario), we drop fileRoute so the
// cache skips the snapshot seed and goes straight to a full API fetch (no watermark → updated=0,
// missingFile=1), which downloads the whole catalog and lets the backend rebuild the snapshot.
const makeDeltaArgs = (opts: { fileContent?: string; fileMissing?: boolean } = {}): serviceHttpProps => ({
	route: CATALOG_ROUTE,
	routeParsed: Env.makeRoute(CATALOG_ROUTE),
	module: CATALOG_MODULE,
	__enviroment__: Env.enviroment,
	__companyID__: Env.getCompanyID(),
	__version__: CATALOG_CACHE.ver,
	__accion__: 3,
	__client__: 0,
	useCache: CATALOG_CACHE,
	keysIDs: { productos: "ID", marcas: "ID", categorias: "ID" },
	headers: buildHeaders('json'),
	fileRoute: opts.fileMissing ? undefined : snapshotFileRoute(),
	fileSchema: PRODUCT_ECOMMERCE_FILE_SCHEMA,
	fileContent: opts.fileContent,
	fileMissing: opts.fileMissing,
	cacheMode: 'refresh',
	status: { code: 200, message: "" },
});

// readDeltaCacheSubObject returns either [] (no cached route) or { [responseKey]: records }.
const extractCachedRows = <T>(result: unknown, responseKey: string): T[] => {
	if (Array.isArray(result)) return [];
	return ((result as Record<string, T[]>)?.[responseKey]) ?? [];
};

// Phase 1 (cold): fetch the .db snapshot once, parse + publish in memory, return the raw text so
// Phase 2 can seed IndexedDB without a second download. If the file is missing/unreachable the
// catalog stays empty for now and `fileMissing` tells Phase 2 to full-download via the delta.
const seedFromFile = async (catalog: ProductCatalog): Promise<{ fileText?: string; fileMissing: boolean }> => {
	try {
		const response = await fetch(snapshotFileRoute());
		if (!response.ok) {
			console.warn("[ProductCatalog] snapshot file unavailable, will full-download via delta:", response.status);
			return { fileMissing: true };
		}
		const fileText = await response.text();
		publish(catalog, parsePsvResponse(fileText, PRODUCT_ECOMMERCE_FILE_SCHEMA) as ICatalogTables);
		return { fileText, fileMissing: false };
	} catch (fileError) {
		console.warn("[ProductCatalog] snapshot file fetch failed, will full-download via delta:", fileError);
		return { fileMissing: true };
	}
};

// Phase 1 (warm): read the cached tables straight from IndexedDB on the main thread. Returns false
// if the product table is empty (IDB cleared but localStorage survived) so we can fall back.
const seedFromIndexedDB = async (catalog: ProductCatalog): Promise<boolean> => {
	try {
		const base = {
			route: CATALOG_ROUTE,
			module: CATALOG_MODULE,
			__enviroment__: Env.enviroment,
			__companyID__: Env.getCompanyID(),
		};
		const [productosResult, marcasResult, categoriasResult] = await Promise.all([
			readDeltaCacheSubObject({ ...base, propInResponse: "productos" }),
			readDeltaCacheSubObject({ ...base, propInResponse: "marcas" }),
			readDeltaCacheSubObject({ ...base, propInResponse: "categorias" }),
		]);
		const productos = extractCachedRows<IProduct>(productosResult, "productos");
		if (productos.length === 0) return false;
		publish(catalog, {
			productos,
			marcas: extractCachedRows<ISharedListRecord>(marcasResult, "marcas"),
			categorias: extractCachedRows<IProductCategory>(categoriasResult, "categorias"),
		});
		return true;
	} catch (idbError) {
		console.warn("[ProductCatalog] IndexedDB seed failed:", idbError);
		return false;
	}
};

// Phase 2: persist + delta through the standard delta cache, then re-publish if records changed.
// On a missing snapshot this is the path that actually downloads the full catalog (updated=0).
const runDeltaSync = async (
	catalog: ProductCatalog,
	opts: { fileContent?: string; fileMissing?: boolean },
): Promise<void> => {
	const beforeSignature = productosSignature(catalog.productos);
	const result = await fetchDeltaCache(makeDeltaArgs(opts));
	const content = result?.content as ICatalogTables | undefined;
	if (content) {
		const nextProductos = normalizeProductos(content.productos ?? []);
		if (productosSignature(nextProductos) !== beforeSignature) {
			publish(catalog, content);
		}
	}
	writeLastFetchUnix();
};

// Single shared, memoized reactive catalog. resolves as soon as the first list is in memory
// (Phase 1); Phase 2 mutates the same instance in place afterward.
let catalogSingleton: ProductCatalog | null = null;
let catalogPromise: Promise<ProductCatalog> | null = null;

export const getProductEcommerceData = async (): Promise<ProductCatalog> => {
	if (catalogPromise) return catalogPromise;

	const catalog = catalogSingleton ?? new ProductCatalog();
	catalogSingleton = catalog;

	// Assign the in-flight promise synchronously (before any await) so concurrent callers share the
	// same single load.
	catalogPromise = (async () => {
		if (!browser) return catalog;

		// TODO: TESTING ONLY — artificial 5s delay to exercise the loading skeleton. Remove.
		// await new Promise((resolve) => setTimeout(resolve, 5000));

		const now = Math.floor(Date.now() / 1000);
		const lastFetch = readLastFetchUnix();
		const isWarm = lastFetch > 0 && now - lastFetch < FRESH_WINDOW_SECONDS;

		let fileText: string | undefined;
		let fileMissing = false;
		if (isWarm) {
			// Warm: the table should already be in IndexedDB. Only touch the file if IDB came back empty.
			const seeded = await seedFromIndexedDB(catalog);
			if (!seeded) ({ fileText, fileMissing } = await seedFromFile(catalog));
		} else {
			({ fileText, fileMissing } = await seedFromFile(catalog));
		}

		// Background: run the delta + IndexedDB persistence without blocking the first paint. When the
		// snapshot was missing, this full-downloads the catalog.
		void runDeltaSync(catalog, { fileContent: fileText, fileMissing }).catch((deltaError) => {
			console.error("[ProductCatalog] delta sync failed", deltaError);
		});

		return catalog;
	})();

	try {
		return await catalogPromise;
	} catch (bootstrapError) {
		// Clear so a later call can retry the load.
		catalogPromise = null;
		throw bootstrapError;
	}
};
