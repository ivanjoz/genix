import { Notify } from '$libs/helpers';
import { GET, POST } from '$libs/http.svelte';

export const makeStockID = (e: IProductoStock): string =>
  [e.WarehouseID, e.ProductID, e.PresentationID || 0, e.Lote || '', e.SKU || ''].join('_')

export interface IProductoStock {
  ID: string
  SKU?: string
  WarehouseID: number
  ProductID: number
  PresentationID: number
  Quantity: number
  SubQuantity: number
  Lote?: string
  CostoUn?: number
  _cantidadPrev?: number
  _isVirtual?: boolean
  _isNew?: boolean
  _hasUpdated?: boolean
  _search?: string
}

export const getProductosStock = async (almacenID: number): Promise<IProductoStock[]> => {
  let records = []
  try {
    records = await GET({ 
      route: `productos-stock?almacen-id=${almacenID}`,
      errorMessage: 'Hubo un error al obtener el stock.',
      useCache: { min: 0.2, ver: 7 },
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
