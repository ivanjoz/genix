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

  constructor(init: boolean = false) {
    super()
    if (init) {
      this.fetch()
    }
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
  DetailFecha: number[] = []
  DetailInflows: number[] = []
  DetailOutflows: number[] = []
  DetailFinalStock: number[] = []

  setDateRange(minimumFecha: number, maximumFecha: number) {
    // Keep the date axis explicit so each index maps to one calendar day without sparse gaps.
    for (let fecha = minimumFecha; fecha <= maximumFecha; fecha++) {
      this.DetailFecha.push(fecha)
      this.DetailInflows.push(0)
      this.DetailOutflows.push(0)
      this.DetailFinalStock.push(0)
    }
  }

  addDataFrame(fecha: number, outflow: number, inflow: number) {
    const fechaIndex = this.getFechaIndex(fecha)
    if (fechaIndex < 0) { return }

    this.DetailOutflows[fechaIndex] = outflow
    this.DetailInflows[fechaIndex] = inflow
  }

  getFechaIndex(fecha: number) {
    const minimumFecha = this.DetailFecha[0]
    if (minimumFecha === undefined) { return -1 }

    const fechaIndex = fecha - minimumFecha
    return fechaIndex >= 0 && fechaIndex < this.DetailFecha.length ? fechaIndex : -1
  }

  getOutflows(fecha: number) {
    const fechaIndex = this.getFechaIndex(fecha)
    return fechaIndex >= 0 ? this.DetailOutflows[fechaIndex] || 0 : 0
  }

  getInflows(fecha: number) {
    const fechaIndex = this.getFechaIndex(fecha)
    return fechaIndex >= 0 ? this.DetailInflows[fechaIndex] || 0 : 0
  }

  getFinalStock(fecha: number) {
    if (!this.DetailFecha.length) { return 0 }

    const minimumFecha = this.DetailFecha[0]
    const maximumFecha = this.DetailFecha[this.DetailFecha.length - 1]
    if (fecha < minimumFecha) { return 0 }
    if (fecha > maximumFecha) {
      // Stock stays unchanged after the last movement until a newer movement exists.
      return this.DetailFinalStock[this.DetailFinalStock.length - 1] || 0
    }

    const fechaIndex = this.getFechaIndex(fecha)
    return fechaIndex >= 0 ? this.DetailFinalStock[fechaIndex] || 0 : 0
  }

  calculateFinalStock(fechaCurrent: number) {
    if (!this.DetailFecha.length) { return }

    const minimumFecha = this.DetailFecha[0]
    const maximumFecha = this.DetailFecha[this.DetailFecha.length - 1]
    const baseFecha = maximumFecha <= fechaCurrent ? maximumFecha : fechaCurrent
    const baseFechaIndex = this.getFechaIndex(baseFecha)
    const shouldDebugFinalStock = this.ProductID === 20003

    if (baseFechaIndex >= 0) {
      // Anchor the reconstruction on the latest known stock snapshot for this product.
      this.DetailFinalStock[baseFechaIndex] = this.CurrentStock
    }

    if (shouldDebugFinalStock) {
      console.group(`ProductStockMovementsDay::calculateFinalStock:${this.ProductID}`)
      console.log('initialState', {
        currentStock: this.CurrentStock,
        fechaCurrent,
        maximumFecha,
        minimumFecha,
        baseFecha,
        detailFecha: this.DetailFecha,
        detailOutflows: this.DetailOutflows,
        detailInflows: this.DetailInflows,
        detailFinalStock: this.DetailFinalStock,
      })
    }

    let moreRecentDayFinalStock = this.CurrentStock
    for (let fechaIndex = this.DetailFecha.length - 1; fechaIndex >= 0; fechaIndex--) {
      const fecha = this.DetailFecha[fechaIndex]

      // Future days keep the current snapshot because there is no later baseline to reverse from.
      if (fecha > fechaCurrent) {
        this.DetailFinalStock[fechaIndex] = this.CurrentStock
        moreRecentDayFinalStock = this.CurrentStock
        if (shouldDebugFinalStock) {
          console.log('futureDayPinnedToCurrentStock', {
            fecha,
            fechaIndex,
            finalStock: this.DetailFinalStock[fechaIndex],
          })
        }
        continue
      }

      if (fecha === baseFecha) {
        moreRecentDayFinalStock = this.DetailFinalStock[fechaIndex]
        if (shouldDebugFinalStock) {
          console.log('baseDayAssigned', {
            fecha,
            fechaIndex,
            outflows: this.DetailOutflows[fechaIndex] || 0,
            inflows: this.DetailInflows[fechaIndex] || 0,
            finalStock: this.DetailFinalStock[fechaIndex],
          })
        }
        continue
      }

      const moreRecentFechaIndex = fechaIndex + 1
      const outflows = moreRecentFechaIndex < this.DetailFecha.length ? this.DetailOutflows[moreRecentFechaIndex] || 0 : 0
      const inflows = moreRecentFechaIndex < this.DetailFecha.length ? this.DetailInflows[moreRecentFechaIndex] || 0 : 0
      // Reverse the next day's net movement to reconstruct this day's ending stock.
      const previousDayFinalStock = moreRecentDayFinalStock - inflows + outflows
      this.DetailFinalStock[fechaIndex] = previousDayFinalStock
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
      console.log('finalColumnarData', {
        detailFecha: this.DetailFecha,
        detailOutflows: this.DetailOutflows,
        detailInflows: this.DetailInflows,
        detailFinalStock: this.DetailFinalStock,
      })
      console.groupEnd()
    }
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
			if(!e.Quantity || e.Quantity < 0){ continue }
			const productoID = extractProductIDFromStockID(e.ID)
			const currentStock = this.productoCurrentStock.get(productoID) || 0
			this.productoCurrentStock.set(productoID, (e.Quantity||0) + currentStock)
		}
		
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
      const productMovements = new ProductStockMovementsDay()
      productMovements.ProductID = productID
      // Prefill the full day window so stock reconstruction and charts stay continuous on days without movements.
      productMovements.setDateRange(minimumFecha, maximumFecha)
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
		
		console.log("productMovements", this.productoCurrentStock)
		console.log("movimientos response::", response)
	}

	constructor() {
    super()
    this.fetch()
  }
}
