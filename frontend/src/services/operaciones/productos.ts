import { GetSignal, POST, makeGETFetchHandler } from "~/shared/http"
import { arrayToMapN, arrayToMapS } from "~/shared/main"

export interface IProducto {
  ID: number,
  Nombre: string
  Descripcion: string
  Precio?: number
  Descuento?: number
  PrecioFinal?: number
  ss: number
  upd: number
}

export interface IProductoResult {
  productos: IProducto[]
}

export const useProductosAPI = (): GetSignal<IProductoResult> => {
  return  makeGETFetchHandler(
    { route: "productos", emptyValue: [],
      errorMessage: 'Hubo un error al obtener los productos.',
      cacheSyncTime: 1, mergeRequest: true,
      useIndexDBCache: 'productos',
    },
    (result_) => {
      const result = result_ as IProductoResult
      return result
    }
  )
}