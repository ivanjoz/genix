import { Notify } from "$lib/helpers"
import { GET, POST } from "$lib/http"

export interface IProductoStock {
  ID: string
  SKU?: string
  AlmacenID: number
  ProductoID: number
  PresentacionID: number
  Cantidad: number
  SubCantidad: number
  Lote?: string
  CostoUn?: number
  _cantidadPrev?: number
  _isVirtual?: boolean
  _hasUpdated?: boolean
  _search?: string
}

export const getProductosStock = async (almacenID: number): Promise<IProductoStock[]> => {
  let records = []
  try {
    records = await GET({ 
      route: `productos-stock?almacen-id=${almacenID}`,
      errorMessage: 'Hubo un error al obtener el stock.',
      useCache: { min: 0, ver: 6 },
    })
  } catch (error) {
    Notify.failure(error as string)
  }
  return records
}

export const postProductosStock = (data: IProductoStock[]) => {
  return POST({
    data,
    route: "productos-stock",
    refreshRoutes: ["productos-stock"]
  })
}