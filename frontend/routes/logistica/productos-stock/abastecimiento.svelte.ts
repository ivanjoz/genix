import { GetHandler, POST } from '$libs/http.svelte'

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

export interface IFechaProductoMovimientos {
  Fecha: number
  DetailProductsIDs: number[]
  DetailInflows: number[]
  DetailOutflows: number[]
  DetailMinimumStock: number[]
  upd: number
}

export class ProductStockMovementsDay {
  ProductID: number = 0
  v: Int32Array // Array of [maxFecha, outflows, inflows, minimumStock, ...] for each day down to minFecha.

  constructor(v: Int32Array) {
    this.v = v
  }

  addDataFrame(fecha: number, outflow: number, inflow: number, minimumStock: number) {
    const fechaIndex = this.getFechaIndex(fecha)
    this.v[fechaIndex + 1] = outflow
    this.v[fechaIndex + 2] = inflow
    this.v[fechaIndex + 3] = minimumStock
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

  getMinimumStock(fecha: number) {
    const fechaIndex = this.getFechaIndex(fecha)
    return fechaIndex >= 0 ? this.v[fechaIndex + 3] : 0
  }
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

export class AlmacenMovimientosGroupedService extends GetHandler {
  route = 'almacen-movimientos-grouped'
  keyID = 'Fecha'
  useCache = { min: 5, ver: 3 }

  records: IFechaProductoMovimientos[] = $state([])
  recordsMap: Map<number, IFechaProductoMovimientos> = $state(new Map())
  productMovements: ProductStockMovementsDay[] = $state([])
  productMovementsMap: Map<number, ProductStockMovementsDay> = $state(new Map())

  handler(result: IFechaProductoMovimientos[]): void {
		const productoFechaRangeMap: Map<number, [number, number]> = new Map()
    
    const groupedMovementRecords = (result || []).map((groupedMovementRecord) => ({
      Fecha: groupedMovementRecord.Fecha || 0,
      DetailProductsIDs: groupedMovementRecord.DetailProductsIDs || [],
      DetailInflows: groupedMovementRecord.DetailInflows || [],
      DetailOutflows: groupedMovementRecord.DetailOutflows || [],
      DetailMinimumStock: groupedMovementRecord.DetailMinimumStock || [],
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
          groupedMovementRecord.DetailMinimumStock[detailIndex] || 0,
        )
      }
    }

    this.records = groupedMovementRecords
    this.recordsMap = new Map(
      groupedMovementRecords.map((groupedMovementRecord) => [groupedMovementRecord.Fecha, groupedMovementRecord]),
    )
		this.productMovements = Array.from(productMovementsMap.values())
		console.log("productMovements", this.productMovements)
    this.productMovementsMap = productMovementsMap
  }

  constructor() {
    super()
    this.fetch()
  }
}
