import { Notify } from "notiflix"
import { GET, GetSignal, POST, makeGETFetchHandler } from "~/shared/http"
import { arrayToMapN } from "~/shared/main"

export interface IProductoPropiedad {
  ID: number, Nombre: string, Options: string[]
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
  Propiedades?: IProductoPropiedad[]
  Peso?: number
  Volumen?: number
  SbnCantidad?: number
  SbnUnidad?: string
  SbnPrecio?: number
  SbnDescuento?: number
  SbnPreciFinal?: number
  Images?: IProductoImage[]
  ss: number
  upd: number
}

export interface IProductoResult {
  Records?: IProducto[]
  productos: IProducto[]
  productosMap: Map<number,IProducto>
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
      result.productosMap = arrayToMapN(result.productos,'ID')
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

export interface IProductoStock {
  ID: string
  SKU?: string
  AlmacenID: number
  ProductoID: number
  Cantidad: number
  SubCantidad: number
  Lote?: string
  CostoUn?: number
  _cantidadPrev?: number
  _isVirtual?: boolean
  _hasUpdated?: boolean
}

export const getProductosStock = async (almacenID: number): Promise<IProductoStock[]> => {
  let records = []
  try {
    const result = await GET({ 
      route: "productos-stock", emptyValue: [],
      errorMessage: 'Hubo un error al obtener el stock.',
      cacheSyncTime: 1, mergeRequest: true,
      useIndexDBCache: 'productos_stock',
      partition: { key: 'AlmacenID', value: almacenID, param: 'almacen-id' }
    })
    records = result.Records || []
  } catch (error) {
    Notify.failure(error as string)
  }
  return records
}

export const postProductosStock = (data: IProductoStock[]) => {
  return POST({
    data,
    route: "productos-stock",
    refreshIndexDBCache: "productos_stock"
  })
}


interface IQueryAlmacenMovimientos {
  almacenID: number
  fechaInicio: number
  fechaFin: number
}

export const queryAlmacenMovimientos = async (args: IQueryAlmacenMovimientos): Promise<any[]> => {
  let records = []
  let route = `almacen-movimientos?almacen-id=${args.almacenID}`
  
  if(!args.fechaInicio || !args.fechaFin){
    throw("No se encontró una fecha de inicio o fin.")
  }
  
  route += `&fecha-hora-inicio=${args.fechaInicio*24*60*60 + window._zoneOffset}`
  route += `&fecha-hora-fin=${(args.fechaFin+1)*24*60*60 + window._zoneOffset}`

  try {
    const result = await GET({ 
      route, emptyValue: [],
      errorMessage: 'Hubo un error al obtener los movimientos del almacén',
    })
    records = result.Records || []
  } catch (error) {
    Notify.failure(error as string)
  }

  return records
}