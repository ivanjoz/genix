import { Env } from '$core/lib/env';
import { arrayToMapN } from '$core/lib/helpers';
import { GET } from '$core/lib/http';

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
} as IProductosResult)

export type IFetch = (
  input: string | URL | Request, init?: RequestInit
) => Promise<Response>

export interface IProductosResult {
  productos: IProducto[]
  productosMap: Map<number, IProducto>
  categorias: IProductoCategoria[]
  categoriasMap: Map<number, IProductoCategoria>
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
        resolve(res)
      })
      .catch(err => {
        reject(err)
      })
    }))
  }

  return await productosPromiseMap.get(apiRoute)
}

// Simple test version - productos.svelte.js
export function useProductosService(categoriasIDs?: number[]) {

  return productosServiceState
}
