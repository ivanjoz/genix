import { Params } from '$core/security'
import { decodeFromBase62 } from '$libs/helpers'
import { GetHandler, POST } from '$libs/http.svelte'
import type { IProductoStock } from './stock-movement'

export interface IProductSupplyProviderRow {
  ProviderID: number
  Capacity: number
  DeliveryTime: number
  Price: number
}

export interface IProductSupplyRow {
  ProductID: number
  MinimunStock: number
  SalesPerDayEstimated: number
  ProviderSupply: IProductSupplyProviderRow[]
  ss?: number
  upd?: number
  UpdatedBy?: number
}


export const createEmptyProviderSupplyRow = (): IProductSupplyProviderRow => ({
  ProviderID: 0,
  Capacity: 0,
  DeliveryTime: 0,
  Price: 0,
})

export const normalizeProviderSupplyRows = (providerSupplyRows: IProductSupplyProviderRow[] = []) => {
  return providerSupplyRows
    .filter((providerSupplyRow) => {
      return providerSupplyRow.ProviderID > 0 || providerSupplyRow.Capacity > 0 || providerSupplyRow.DeliveryTime > 0 || providerSupplyRow.Price > 0
    })
    .map((providerSupplyRow) => ({
      ProviderID: providerSupplyRow.ProviderID || 0,
      Capacity: providerSupplyRow.Capacity || 0,
      DeliveryTime: providerSupplyRow.DeliveryTime || 0,
      Price: providerSupplyRow.Price || 0,
    }))
}

export class ProductSupplyService extends GetHandler<any> {
  route = 'product-supply'
  keyID = 'ProductID'
  useCache = { min: 5, ver: 1 }

  records: IProductSupplyRow[] = $state([])
  recordsMap: Map<number, IProductSupplyRow> = $state(new Map())

  handler(result: IProductSupplyRow[]): void {
    const fetchedProductSupplyRecords = (result || [])
      .filter((productSupplyRecord) => (productSupplyRecord.ss || 0) > 0)

    this.records = fetchedProductSupplyRecords
    this.recordsMap = new Map(
      fetchedProductSupplyRecords.map((productSupplyRecord) => [productSupplyRecord.ProductID, productSupplyRecord]),
    )
  }

  constructor() {
    super()
    this.fetch()
  }
}

export const postProductSupply = (productSupplyRecord: IProductSupplyRow) => {
  const normalizedPayload: IProductSupplyRow = {
    ProductID: productSupplyRecord.ProductID,
    MinimunStock: productSupplyRecord.MinimunStock || 0,
    SalesPerDayEstimated: productSupplyRecord.SalesPerDayEstimated || 0,
    // Keep the payload sanitized so the backend only receives meaningful provider rows.
    ProviderSupply: normalizeProviderSupplyRows(productSupplyRecord.ProviderSupply),
  }

  return POST({
    data: normalizedPayload,
    route: 'product-supply',
    refreshRoutes: ['product-supply'],
  })
}


export interface IFechaProductoMovimientos {
  Fecha: number
  DetailProductsIDs: number[]
  DetailInflows: number[]
  DetailOutflows: number[]
  DetailFinalStock: number[]
  upd: number
}

export class ProductStockMovementsDay {
	ProductID: number = 0
  CurrentStock: number = 0
  v: Int32Array // Array of [maxFecha, outflows, inflows, finalStock, ...] for each day down to minFecha.

  constructor(v: Int32Array) {
    this.v = v
  }

  addDataFrame(fecha: number, outflow: number, inflow: number, finalStock: number = 0) {
    const fechaIndex = this.getFechaIndex(fecha)
    this.v[fechaIndex + 1] = outflow
    this.v[fechaIndex + 2] = inflow
    this.v[fechaIndex + 3] = finalStock
  }

  getFechaIndex(fecha: number) {
    const fechaBase = this.v[0]
    const totalDays = (this.v.length - 1) / 3
    const minimumFecha = fechaBase - totalDays + 1
    if (!fechaBase || fecha > fechaBase || fecha < minimumFecha) { return -1 }
    return (fechaBase - fecha) * 3
  }

  getOutflows(fecha: number) {
    const fechaIndex = this.getFechaIndex(fecha)
    return fechaIndex >= 0 ? this.v[fechaIndex + 1] : 0
  }

  getInflows(fecha: number) {
    const fechaIndex = this.getFechaIndex(fecha)
    return fechaIndex >= 0 ? this.v[fechaIndex + 2] : 0
  }

  getFinalStock(fecha: number) {
    const fechaIndex = this.getFechaIndex(fecha)
    return fechaIndex >= 0 ? this.v[fechaIndex + 3] : 0
  }

  calculateFinalStock(fechaCurrent: number) {
    const mostRecentFecha = this.v[0]
    if (!mostRecentFecha) { return }

    const minimumFecha = this.getMinimumFecha()
    const baseFecha = mostRecentFecha <= fechaCurrent ? mostRecentFecha : fechaCurrent
    const baseFechaIndex = this.getFechaIndex(baseFecha)
    const shouldDebugFinalStock = this.ProductID === 20003

    if (baseFechaIndex >= 0) {
      this.v[baseFechaIndex + 3] = this.CurrentStock
    }

    if (shouldDebugFinalStock) {
      console.group(`ProductStockMovementsDay::calculateFinalStock:${this.ProductID}`)
      console.log('initialState', {
        currentStock: this.CurrentStock,
        fechaCurrent,
        mostRecentFecha,
        minimumFecha,
        baseFecha,
        rawVector: Array.from(this.v),
      })
    }

    let moreRecentDayFinalStock = this.CurrentStock
    for (let fecha = mostRecentFecha; fecha >= minimumFecha; fecha--) {
      const fechaIndex = this.getFechaIndex(fecha)
      if (fechaIndex < 0) { continue }

      // The vector stores unix days from the most recent date down to the oldest one.
      // Keep future days pinned to the current snapshot because there is no future stock baseline.
      if (fecha > fechaCurrent) {
        this.v[fechaIndex + 3] = this.CurrentStock
        moreRecentDayFinalStock = this.CurrentStock
        if (shouldDebugFinalStock) {
          console.log('futureDayPinnedToCurrentStock', {
            fecha,
            fechaIndex,
            finalStock: this.v[fechaIndex + 3],
          })
        }
        continue
      }

      if (fecha === baseFecha) {
        moreRecentDayFinalStock = this.v[fechaIndex + 3]
        if (shouldDebugFinalStock) {
          console.log('baseDayAssigned', {
            fecha,
            fechaIndex,
            outflows: this.v[fechaIndex + 1] || 0,
            inflows: this.v[fechaIndex + 2] || 0,
            finalStock: this.v[fechaIndex + 3],
          })
        }
        continue
      }

      const moreRecentFechaIndex = this.getFechaIndex(fecha + 1)
      const outflows = moreRecentFechaIndex >= 0 ? this.v[moreRecentFechaIndex + 1] || 0 : 0
      const inflows = moreRecentFechaIndex >= 0 ? this.v[moreRecentFechaIndex + 2] || 0 : 0
      // Reverse the more recent day's net movement to reconstruct the previous day's ending stock.
      const previousDayFinalStock = moreRecentDayFinalStock - inflows + outflows
      this.v[fechaIndex + 3] = previousDayFinalStock
      if (shouldDebugFinalStock) {
        console.log('previousDayCalculated', {
          fecha,
          fechaIndex,
          moreRecentFecha: fecha + 1,
          moreRecentFechaIndex,
          outflows,
          inflows,
          moreRecentDayFinalStock,
          previousDayFinalStock,
        })
      }
      moreRecentDayFinalStock = previousDayFinalStock
    }

    if (shouldDebugFinalStock) {
      console.log('finalVector', Array.from(this.v))
      console.groupEnd()
    }
  }

  private getMinimumFecha() {
    const totalDays = (this.v.length - 1) / 3
    return this.v[0] - totalDays + 1
  }
}

interface IAlmacenMovimientosGroupedResponse {
	movimientos: IFechaProductoMovimientos[]
	productosStock: IProductoStock[]
}

const extractProductIDFromStockID = (productStockID: string) => {
	return decodeFromBase62(productStockID.split("_")[1])
}

export class AlmacenMovimientosGroupedService extends GetHandler {
  route = 'almacen-movimientos-grouped'
  keyID = 'Fecha'
  useCache = { min: 5, ver: 8 }

  records: IFechaProductoMovimientos[] = $state([])
  recordsMap: Map<number, IFechaProductoMovimientos> = $state(new Map())
  productMovements: ProductStockMovementsDay[] = $state([])
	productMovementsMap: Map<number, ProductStockMovementsDay> = $state(new Map())
  productoCurrentStock: Map<number,number> = new Map()

	handler(response: IAlmacenMovimientosGroupedResponse): void {
		// Extrae el current stock
		this.productoCurrentStock = new Map()
		
		for (const e of response.productosStock) {
			if(!e.Cantidad || e.Cantidad < 0){ continue }
			const productoID = extractProductIDFromStockID(e.ID)
			const currentStock = this.productoCurrentStock.get(productoID) || 0
			this.productoCurrentStock.set(productoID, (e.Cantidad||0) + currentStock)
		}
		
		console.log("productMovements", this.productoCurrentStock)
		console.log("movimientos response::", response)
		const fechaCurrent = Params.getFechaUnix()
		
		const productoFechaRangeMap: Map<number, [number, number]> = new Map()
    
    const groupedMovementRecords = (response.movimientos || []).map((groupedMovementRecord) => ({
      Fecha: groupedMovementRecord.Fecha || 0,
      DetailProductsIDs: groupedMovementRecord.DetailProductsIDs || [],
      DetailInflows: groupedMovementRecord.DetailInflows || [],
      DetailOutflows: groupedMovementRecord.DetailOutflows || [],
      DetailFinalStock: [],
      upd: groupedMovementRecord.upd || groupedMovementRecord.Fecha || 0,
    }))

    for (const groupedMovementRecord of groupedMovementRecords) {
      for (const productID of groupedMovementRecord.DetailProductsIDs) {
        const productoFechaRange = productoFechaRangeMap.get(productID)
        if (!productoFechaRange) {
          productoFechaRangeMap.set(productID, [groupedMovementRecord.Fecha, groupedMovementRecord.Fecha])
          continue
        }
        if (groupedMovementRecord.Fecha < productoFechaRange[0]) { productoFechaRange[0] = groupedMovementRecord.Fecha }
        if (groupedMovementRecord.Fecha > productoFechaRange[1]) { productoFechaRange[1] = groupedMovementRecord.Fecha }
      }
    }

    const productMovementsMap = new Map<number, ProductStockMovementsDay>()
    for (const [productID, [minimumFecha, maximumFecha]] of productoFechaRangeMap.entries()) {
      // Allocate only the stored day window for each product.
      const productMovements = new ProductStockMovementsDay(new Int32Array(1 + ((maximumFecha - minimumFecha + 1) * 3)))
      productMovements.ProductID = productID
      productMovements.v[0] = maximumFecha
      productMovementsMap.set(productID, productMovements)
    }

    for (const groupedMovementRecord of groupedMovementRecords) {
      for (const [detailIndex, productID] of groupedMovementRecord.DetailProductsIDs.entries()) {
        productMovementsMap.get(productID)?.addDataFrame(
          groupedMovementRecord.Fecha,
          groupedMovementRecord.DetailOutflows[detailIndex] || 0,
          groupedMovementRecord.DetailInflows[detailIndex] || 0,
        )
      }
    }

		for (const pg of productMovementsMap.values()) {
			pg.CurrentStock = this.productoCurrentStock.get(pg.ProductID) || 0
      pg.calculateFinalStock(fechaCurrent)
		}
    
		console.log("productMovementsMap", [...productMovementsMap.values()])
		
		this.records = groupedMovementRecords
    this.recordsMap = new Map(
      groupedMovementRecords.map((groupedMovementRecord) => [groupedMovementRecord.Fecha, groupedMovementRecord]),
    )
		this.productMovements = Array.from(productMovementsMap.values())
		this.productMovementsMap = productMovementsMap
	}

	constructor() {
    super()
    this.fetch()
  }
}
