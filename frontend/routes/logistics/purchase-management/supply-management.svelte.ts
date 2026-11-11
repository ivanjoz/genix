import { getFechaUnix } from '@genix/ui/utilities'
import { decodeFromBase62 } from '#libs/helpers.ts'
import { GetHandler } from '#libs/ui-runtime.svelte.ts'
import { ProductStockSimpleService, type IProductStock } from '../products-stock/stock-movement'
import { queryAlmacenMovimientos } from '../warehouse-movements/warehouse-movements.svelte'
import { aggregateDailyProductMovements, type IDateProductMovements } from './supply-management'

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
  useCache = { min: 5, ver: 2 }

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

// The route is bulk: a single-record save from the side panel sends a one-element array, and the
// Excel import sends batches of the same shape. Each write brings back the delta of `product-supply`
// (GetHandler.writeAndSync), so the table shows the saved rows without a GET after it.
export const postProductSupply = (productSupplyService: ProductSupplyService, productSupplyRecords: IProductSupplyRow[]) => {
  const normalizedPayload: IProductSupplyRow[] = productSupplyRecords.map((productSupplyRecord) => ({
    ProductID: productSupplyRecord.ProductID,
    MinimunStock: productSupplyRecord.MinimunStock || 0,
    SalesPerDayEstimated: productSupplyRecord.SalesPerDayEstimated || 0,
    // Keep the payload sanitized so the backend only receives meaningful provider rows.
    ProviderSupply: normalizeProviderSupplyRows(productSupplyRecord.ProviderSupply),
  }))

  return productSupplyService.writeAndSync('product-supply', normalizedPayload)
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
    for (let date = minimumFecha; date <= maximumFecha; date++) {
      this.DetailFecha.push(date)
      this.DetailInflows.push(0)
      this.DetailOutflows.push(0)
      this.DetailFinalStock.push(0)
    }
  }

  addDataFrame(date: number, outflow: number, inflow: number) {
    const dateIndex = this.getFechaIndex(date)
    if (dateIndex < 0) { return }

    this.DetailOutflows[dateIndex] = outflow
    this.DetailInflows[dateIndex] = inflow
  }

  getFechaIndex(date: number) {
    const minimumFecha = this.DetailFecha[0]
    if (minimumFecha === undefined) { return -1 }

    const dateIndex = date - minimumFecha
    return dateIndex >= 0 && dateIndex < this.DetailFecha.length ? dateIndex : -1
  }

  getOutflows(date: number) {
    const dateIndex = this.getFechaIndex(date)
    return dateIndex >= 0 ? this.DetailOutflows[dateIndex] || 0 : 0
  }

  getInflows(date: number) {
    const dateIndex = this.getFechaIndex(date)
    return dateIndex >= 0 ? this.DetailInflows[dateIndex] || 0 : 0
  }

  getFinalStock(date: number) {
    if (!this.DetailFecha.length) { return 0 }

    const minimumFecha = this.DetailFecha[0]
    const maximumFecha = this.DetailFecha[this.DetailFecha.length - 1]
    if (date < minimumFecha) { return 0 }
    if (date > maximumFecha) {
      // Stock stays unchanged after the last movement until a newer movement exists.
      return this.DetailFinalStock[this.DetailFinalStock.length - 1] || 0
    }

    const dateIndex = this.getFechaIndex(date)
    return dateIndex >= 0 ? this.DetailFinalStock[dateIndex] || 0 : 0
  }

  calculateFinalStock(dateCurrent: number) {
    if (!this.DetailFecha.length) { return }

    const minimumFecha = this.DetailFecha[0]
    const maximumFecha = this.DetailFecha[this.DetailFecha.length - 1]
    const baseFecha = maximumFecha <= dateCurrent ? maximumFecha : dateCurrent
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
        dateCurrent,
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
    for (let dateIndex = this.DetailFecha.length - 1; dateIndex >= 0; dateIndex--) {
      const date = this.DetailFecha[dateIndex]

      // Future days keep the current snapshot because there is no later baseline to reverse from.
      if (date > dateCurrent) {
        this.DetailFinalStock[dateIndex] = this.CurrentStock
        moreRecentDayFinalStock = this.CurrentStock
        if (shouldDebugFinalStock) {
          console.log('futureDayPinnedToCurrentStock', {
            date,
            dateIndex,
            finalStock: this.DetailFinalStock[dateIndex],
          })
        }
        continue
      }

      if (date === baseFecha) {
        moreRecentDayFinalStock = this.DetailFinalStock[dateIndex]
        if (shouldDebugFinalStock) {
          console.log('baseDayAssigned', {
            date,
            dateIndex,
            outflows: this.DetailOutflows[dateIndex] || 0,
            inflows: this.DetailInflows[dateIndex] || 0,
            finalStock: this.DetailFinalStock[dateIndex],
          })
        }
        continue
      }

      const moreRecentFechaIndex = dateIndex + 1
      const outflows = moreRecentFechaIndex < this.DetailFecha.length ? this.DetailOutflows[moreRecentFechaIndex] || 0 : 0
      const inflows = moreRecentFechaIndex < this.DetailFecha.length ? this.DetailInflows[moreRecentFechaIndex] || 0 : 0
      // Reverse the next day's net movement to reconstruct this day's ending stock.
      const previousDayFinalStock = moreRecentDayFinalStock - inflows + outflows
      this.DetailFinalStock[dateIndex] = previousDayFinalStock
      if (shouldDebugFinalStock) {
        console.log('previousDayCalculated', {
          date,
          dateIndex,
          moreRecentFecha: date + 1,
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

/** Free-bucket stock per product, summed over warehouses (negative rows count as none). */
const sumCurrentStockByProduct = (productStockRecords: IProductStock[]) => {
  const currentStockByProductID = new Map<number, number>()
  for (const productStock of productStockRecords) {
    if (!productStock.Quantity || productStock.Quantity < 0) { continue }
    currentStockByProductID.set(productStock.ProductID,
      (currentStockByProductID.get(productStock.ProductID) || 0) + productStock.Quantity)
  }
  return currentStockByProductID
}

const buildProductMovementsMap = (
  groupedMovementRecords: IDateProductMovements[], currentStockByProductID: Map<number, number>, windowDays: number,
) => {
    const dateCurrent = getFechaUnix()
    // Every product with stock or a movement in the window spans the whole window: the final stock
    // is reconstructed backwards from today's stock, so a day before a product's first movement in
    // the window still holds its real stock instead of 0.
    const productIDs = new Set(currentStockByProductID.keys())
    for (const groupedMovementRecord of groupedMovementRecords) {
      for (const productID of groupedMovementRecord.DetailProductsIDs) { productIDs.add(productID) }
    }

    const productMovementsMap = new Map<number, ProductStockMovementsDay>()
    for (const productID of productIDs) {
      const productMovements = new ProductStockMovementsDay()
      productMovements.ProductID = productID
      productMovements.setDateRange(dateCurrent - windowDays + 1, dateCurrent)
      productMovementsMap.set(productID, productMovements)
    }

    for (const groupedMovementRecord of groupedMovementRecords) {
      for (const [detailIndex, productID] of groupedMovementRecord.DetailProductsIDs.entries()) {
        productMovementsMap.get(productID)?.addDataFrame(
          groupedMovementRecord.Date,
          groupedMovementRecord.DetailOutflows[detailIndex] || 0,
          groupedMovementRecord.DetailInflows[detailIndex] || 0,
        )
      }
    }

    for (const productMovements of productMovementsMap.values()) {
      productMovements.CurrentStock = currentStockByProductID.get(productMovements.ProductID) || 0
      productMovements.calculateFinalStock(dateCurrent)
    }
    return productMovementsMap
}

/**
 * Daily inflows/outflows of the last `windowDays` days plus each day's reconstructed final stock.
 * Both sources are browser caches: the current stock is the delta-synced `products-stock` list, and
 * the movements are the per-day cache groups of `warehouse-movements` (only changed days download),
 * folded per product on the client.
 */
export class SupplyMovementsService {
  productStockService = new ProductStockSimpleService(true)
  windowDays: number
  records: IDateProductMovements[] = $state([])
  isReady = $state(false)
  productoCurrentStock = $derived(sumCurrentStockByProduct(this.productStockService.records))
  productMovementsMap = $derived.by(() => buildProductMovementsMap(this.records, this.productoCurrentStock, this.windowDays))

  constructor(windowDays: number) {
    this.windowDays = windowDays
    this.loadMovements()
  }

  async loadMovements() {
    const dateFin = getFechaUnix()
    const movements = await queryAlmacenMovimientos({ almacenID: 0, dateInicio: dateFin - this.windowDays + 1, dateFin })
    this.records = aggregateDailyProductMovements(movements)
    this.isReady = true
  }
}
