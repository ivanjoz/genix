import { GetSignal, POST, makeGETFetchHandler } from "~/shared/http"
import { arrayToMapN, arrayToMapS } from "~/shared/main"

export interface IProductoPropiedad {
  ID: number
  Nombre: string
  Options: string[]
}

export interface IProducto {
  ID: number,
  Nombre: string
  Descripcion: string
  Precio?: number
  Descuento?: number
  PrecioFinal?: number
  Propiedades?: IProductoPropiedad[]
  Peso?: number
  Volumen?: number
  SbnCantidad?: number
  SbnUnidad?: string
  SbnPrecio?: number
  SbnDescuento?: number
  SbnPreciFinal?: number
  ss: number
  upd: number
}

export interface IProductoResult {
  Records?: IProducto[]
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
      for(let e of result.Records){
        e.Propiedades = e.Propiedades || []
      }
      console.log("result productos:: ", result)
      result.productos = (result.Records || []).filter(x => x.ss > 0)
      return result
    }
  )
}

export const postProducto = (data: IProducto[]) => {
  return POST({
    data,
    route: "productos",
    refreshIndexDBCache: "productos"
  })
}
