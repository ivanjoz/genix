import { arrayToMapN } from '$libs/helpers';
import { getProductEcommerceData } from '$core/product-search/productos-delta-service';

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

export const productosServiceState = $state({
  productos: [],
  productosMap: new Map(),
  categorias: [],
  categoriasMap: new Map(),
  productosByCategoryMap: new Map(),
} as IProductsResult)

export type IFetch = (
  input: string | URL | Request, init?: RequestInit
) => Promise<Response>

export interface IProductsResult {
  productos: IProduct[]
  productosMap: Map<number, IProduct>
  categorias: IProductCategory[]
  categoriasMap: Map<number, IProductCategory>
  productosByCategoryMap: Map<number, IProduct[]>
}

// True once productosServiceState has been synced from the loaded catalog, so we only rebuild the
// maps once per shared load (subsequent calls just await the resolved singleton).
let serviceStateSynced = false;

// Populate productosServiceState (lists + lookup maps) from the loaded catalog. The category map is
// built here from each product's CategoryIDs.
const syncServiceStateFrom = (service: { productos: IProduct[]; categorias: IProductCategory[] }): void => {
	productosServiceState.productos = service.productos;
	productosServiceState.productosMap = arrayToMapN(service.productos || [], 'ID');
	productosServiceState.categorias = service.categorias;
	productosServiceState.categoriasMap = arrayToMapN(service.categorias || [], 'ID');

	const productosByCategoryMap = new Map<number, IProduct[]>();
	for (const producto of service.productos || []) {
		for (const categoriaId of producto.CategoryIDs || []) {
			let bucket = productosByCategoryMap.get(categoriaId);
			if (!bucket) {
				bucket = [];
				productosByCategoryMap.set(categoriaId, bucket);
			}
			bucket.push(producto);
		}
	}
	productosServiceState.productosByCategoryMap = productosByCategoryMap;
	serviceStateSynced = true;
};

// Ensure the single shared catalog is loaded and productosServiceState is populated from it. Every
// storefront catalog consumer goes through here, so the data is fetched exactly once.
export const ensureProductosLoaded = async (): Promise<void> => {
	const service = await getProductEcommerceData();
	if (!serviceStateSynced) {
		syncServiceStateFrom(service as unknown as { productos: IProduct[]; categorias: IProductCategory[] });
	}
};

export const getProductsByCategoryID = async (id: number): Promise<IProduct[]> => {
	//const products = productosServiceState.productosByCategoryMap.get(id);
	//if (products) return products;

	try {
		await ensureProductosLoaded();
	} catch (err) {
		console.error("Error cargando productos para getProductsByCategoryID:", err);
		return [];
	}

	console.log("productosServiceState", $state.snapshot(productosServiceState))

	return productosServiceState.productosByCategoryMap.get(id) || [];
}

export const getCategoryByID = async (id: number): Promise<IProductCategory | undefined> => {
	const category = productosServiceState.categoriasMap.get(id);
	if (category) return category;

	try {
		await ensureProductosLoaded();
	} catch (err) {
		console.error("Error cargando productos para getCategoryByID:", err);
		return undefined;
	}

	// Note: the delta snapshot carries category ID/Name only (no Description).
	return productosServiceState.categoriasMap.get(id);
}
