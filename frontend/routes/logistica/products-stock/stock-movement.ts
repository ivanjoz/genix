import { Notify } from '$libs/helpers';
import { GET, GetHandler, POST } from '$libs/http.svelte';

export const makeStockID = (e: Pick<IProductoStock, 'WarehouseID' | 'ProductID' | 'PresentationID'>): number =>
  // Mirrors backend/logistica/product-stock-movement.go::packProductStockID.
  e.WarehouseID * 1e14 + e.ProductID * 1e5 + (e.PresentationID || 0) * 10

export interface IProductStockDetail {
  ProductStockID: number
  LotID: number
  SerialNumber?: string
  WarehouseID: number
  ProductID: number
  Quantity: number
  SubQuantity: number
  ExpirationDate?: number
  upd?: number
  UpdatedBy?: number
  Created?: number
  CreatedBy?: number
	ss?: number
	LotCode?: string
}

export interface IProductStockLot {
  ID: number
  Date?: number
  Name?: string
  SupplierID?: number
  DeliveryNoteID?: number
  DeliveryNoteCode?: string
  Hash?: string
  Created?: number
  CreatedBy?: number
  ss?: number
}

export interface IProductoStock {
  ID: number
  WarehouseID: number
  ProductID: number
  PresentationID: number
  Quantity: number
  SubQuantity: number
  DetailQuantity?: number
  DetailSubQuantity?: number
  DetailComputedDate?: number
  DetailComputedQuantity?: number
  DetailComputedSubQuantity?: number
  StockStatus?: number
  Created?: number
  CreatedBy?: number
  upd?: number
  UpdatedBy?: number
  ss?: number
  StockDetails: IProductStockDetail[]
  _cantidadPrev?: number
  _isVirtual?: boolean
  _isNew?: boolean
  _hasUpdated?: boolean
  _search?: string
}

interface IGetProductosStockResponse {
  ProductStock?: IProductoStock[]
  ProductStockDetail?: IProductStockDetail[]
}

export interface IPostProductoStockItem {
  WarehouseID: number
  ProductID: number
  PresentationID: number
  Quantity: number
  SubQuantity?: number
  SerialNumber?: string
  LotID?: number
  LotCode?: string
}

export const getWarehouseProductStock = async (almacenID: number): Promise<IProductoStock[]> => {
  let records: IProductoStock[] = []
  try {
    const response = await GET({ 
      route: `warehouse-product-stock?almacen-id=${almacenID}`,
      errorMessage: 'Hubo un error al obtener el stock.',
			useCache: { min: 0.2, ver: 8 },
			keysIDs: { ProductStockDetail: ["ProductStockID","LotID","SerialNumber"] }
    })
		const normalizedResponse = response as IGetProductosStockResponse | null | undefined
    console.log("getProductosStock::", normalizedResponse)
		
		if (!normalizedResponse) {
      records = []
    } else {
      const stockDetailsByStockID = new Map<number, IProductStockDetail[]>()
      for (const stockDetail of normalizedResponse.ProductStockDetail || []) {
        const stockDetails = stockDetailsByStockID.get(stockDetail.ProductStockID)
        if (stockDetails) {
          stockDetails.push(stockDetail)
        } else {
          stockDetailsByStockID.set(stockDetail.ProductStockID, [stockDetail])
        }
      }

      records = normalizedResponse.ProductStock || []
      for (const productStockRecord of normalizedResponse.ProductStock || []) {
        // Attach the matching detail rows directly to each backend stock row.
        productStockRecord.StockDetails = stockDetailsByStockID.get(productStockRecord.ID) || []
      }
    }
    console.debug('getProductosStock::normalizedResponse', {
      almacenID,
      productStockCount: records.length,
      productStockDetailCount: records.reduce((detailCount, productStockRecord) => detailCount + productStockRecord.StockDetails.length, 0),
    })
  } catch (error) {
    Notify.failure(error as string)
  }
  return records
}

export class ProductStockSimpleService extends GetHandler<IProductoStock> {
  route = "products-stock"
  useCache = { min: 0.2, ver: 8 }
  inferRemoveFromStatus = true

  constructor(init: boolean = false) {
    super()
    if (init) this.fetch()
  }

	handler(result: IProductoStock[]): void {
		console.log("ProductStockSimpleService", result)
		
    this.records = []
    this.recordsMap = new Map()
    this.addSavedRecords(...result)
  }
}


export const postProductosStock = (data: IPostProductoStockItem[]) => {
  return POST({
    data,
    route: "productos-stock",
    refreshRoutes: ["productos-stock"]
  })
}
