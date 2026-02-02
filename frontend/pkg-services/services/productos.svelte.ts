import { Env } from '$core/env';
import { arrayToMapN } from '$core/helpers';
import { GET } from '$core/http';

const maxCacheTime = 60 * 5 // 2 segundos
const productosPromiseMap: Map<string, Promise<any>> = new Map()

export interface IProductoPropiedad {
  id: number, nm: string, ss: number
}

export interface IProductoPropiedades {
  ID: number, Nombre: string, Options: IProductoPropiedad[], Status: number
}

export interface IProductoImage {
  n: string /* Nombre del imagen */
  d: string /* descripcion de la imagen */
}

export interface IProducto {
  ID: number,
  Nombre: string
  Descripcion: string
  Precio?: number
  Descuento?: number
  PrecioFinal?: number
  ContentHTML?: string
  Propiedades?: IProductoPropiedades[]
  Peso?: number
  Volumen?: number
  SbnCantidad?: number
  SbnUnidad?: string
  SbnPrecio?: number
  SbnDescuento?: number
  SbnPreciFinal?: number
  Images?: IProductoImage[]
  Image?: IProductoImage
  CategoriasIDs?: number[]
  Stock?: { a /* almacen */: number, c /* cantidad */: number }[]
  ss: number
  upd: number
  _stock?: number
  _moneda?: string
}

export interface IProductoCategoria {
  ID: number,
  Descripcion: string,
  Nombre: string,
}

export const productosServiceState = $state({
  productos: [],
  productosMap: new Map(),
  categorias: [],
  categoriasMap: new Map(),
  productosByCategoryMap: new Map(),
} as IProductosResult)

export type IFetch = (
  input: string | URL | Request, init?: RequestInit
) => Promise<Response>

export interface IProductosResult {
  productos: IProducto[]
  productosMap: Map<number, IProducto>
  categorias: IProductoCategoria[]
  categoriasMap: Map<number, IProductoCategoria>
  productosByCategoryMap: Map<number, IProducto[]>
}

export const getProductos = async (categoriasIDs?: number[]): Promise<IProductosResult> => {
  const apiRoute = `p-productos-cms?categorias=${(categoriasIDs || [0]).join(".")}`

  if(!productosPromiseMap.has(apiRoute)) {
    const headers = new Headers()
    headers.append('Authorization', `Bearer 1`)
    console.log("Consultando Productos | API:", apiRoute)

		productosPromiseMap.set(apiRoute, new Promise((resolve, reject) => {
			GET({ route: apiRoute })
			.then(res => {
				console.log("Productos response:", Object.keys(res))
				console.log(res)

        for(const e of (res.productos||[]) as IProducto[]){
          e.Image = (e.Images||[])[0] || { n: "" } as IProductoImage
        }

        res.productosMap = arrayToMapN(res.productos || [], 'ID')
        res.categoriasMap = arrayToMapN(res.categorias || [], 'ID')
        res.updated = Math.floor(Date.now() / 1000)

        // Actualizamos el estado global
        productosServiceState.productos = res.productos
        productosServiceState.productosMap = res.productosMap
        productosServiceState.categorias = res.categorias
        productosServiceState.categoriasMap = res.categoriasMap

        // Construir el mapa de productos por categoría
        const productosByCategoryMap = new Map<number, IProducto[]>()
        for (const producto of res.productos || []) {
          if (producto.CategoriasIDs) {
            for (const categoriaId of producto.CategoriasIDs) {
              if (!productosByCategoryMap.has(categoriaId)) {
                productosByCategoryMap.set(categoriaId, [])
              }
              productosByCategoryMap.get(categoriaId)!.push(producto)
            }
          }
        }
        productosServiceState.productosByCategoryMap = productosByCategoryMap

        resolve(res)
      })
      .catch(err => {
        reject(err)
      })
    }))
  }

  return await productosPromiseMap.get(apiRoute)
}

let loadingPromise: Promise<IProductosResult> | null = null;

export const getProductoByID = async (id: number): Promise<IProducto | undefined> => {
	// 1. Si ya lo tenemos en el mapa, lo retornamos inmediatamente
	const p = productosServiceState.productosMap.get(id);
	if (p) return p;

	// 2. Si no hay productos cargados todavía, iniciamos o esperamos la carga inicial
	if (productosServiceState.productos.length === 0) {
		if (!loadingPromise) {
			loadingPromise = getProductos();
		}

		try {
			await loadingPromise;
		} catch (err) {
			console.error("Error cargando productos para getProductoByID:", err);
			return undefined;
		}
	}

	// 3. Después de cargar (o si ya había productos pero no el que buscamos), 
	// buscamos en el mapa actualizado. Si no está aquí, es que no existe.
	return productosServiceState.productosMap.get(id);
}

export const getProductsByCategoryID = async (id: number): Promise<IProducto[]> => {
	// 1. Si ya lo tenemos en el mapa, lo retornamos inmediatamente
	const products = productosServiceState.productosByCategoryMap.get(id);
	if (products) return products;

	// 2. Si no hay productos cargados todavía, iniciamos o esperamos la carga inicial
	if (productosServiceState.productos.length === 0) {
		if (!loadingPromise) {
			loadingPromise = getProductos();
		}

		try {
			await loadingPromise;
		} catch (err) {
			console.error("Error cargando productos para getProductsByCategoryID:", err);
			return [];
		}
	}

	// 3. Después de cargar (o si ya había productos pero no el que buscamos), 
	// buscamos en el mapa actualizado. Si no está aquí, es que no existe.
	return productosServiceState.productosByCategoryMap.get(id) || [];
}

export const getCategoryByID = async (id: number): Promise<IProductoCategoria | undefined> => {
	// 1. Si ya lo tenemos en el mapa, lo retornamos inmediatamente
	const category = productosServiceState.categoriasMap.get(id);
	if (category) return category;

	// 2. Si no hay categorías cargadas todavía, iniciamos o esperamos la carga inicial
	if (productosServiceState.categorias.length === 0) {
		if (!loadingPromise) {
			loadingPromise = getProductos();
		}

		try {
			await loadingPromise;
		} catch (err) {
			console.error("Error cargando productos para getCategoryByID:", err);
			return undefined;
		}
	}

	// 3. Después de cargar (o si ya había categorías pero no la que buscamos), 
	// buscamos en el mapa actualizado. Si no está aquí, es que no existe.
	return productosServiceState.categoriasMap.get(id);
}
