// Mirrors StockMovementType in backend/logistics/types/stock_movement_type.go. The ids are
// persisted; 11 is reserved there for the warehouse-to-warehouse transfer.
export const StockMovementType = {
  MANUAL_ENTRY: 1,
  MANUAL_EXIT: 2,
  PURCHASE_RECEPTION: 3,
  SUPPLIES_PURCHASE: 4,
  ASSET_ACQUISITION: 5,
  ASSET_DISPOSAL: 6,
  ASSET_TRANSFER: 7,
  SALE_DELIVERY: 8,
  SALE_ANNULMENT_RETURN: 9,
  ASSET_SERIAL_CORRECTION: 10,
  OPENING_STOCK: 12,
  INTERNAL_CONSUMPTION: 13,
  SHRINKAGE: 14,
} as const

export const stockMovementTypes: { id: number, name: string }[] = [
  { id: StockMovementType.MANUAL_ENTRY, name: 'Manual entry|Entrada manual' },
  { id: StockMovementType.MANUAL_EXIT, name: 'Manual exit|Salida manual' },
  { id: StockMovementType.PURCHASE_RECEPTION, name: 'Purchase order reception|Ingreso por orden de compra' },
  { id: StockMovementType.SUPPLIES_PURCHASE, name: 'Supplies purchase|Compra de insumos' },
  { id: StockMovementType.ASSET_ACQUISITION, name: 'Asset acquisition|Alta de activo' },
  { id: StockMovementType.ASSET_DISPOSAL, name: 'Asset disposal|Baja de activo' },
  { id: StockMovementType.ASSET_TRANSFER, name: 'Asset transfer|Traslado de activo' },
  { id: StockMovementType.SALE_DELIVERY, name: 'Delivery (sale)|Entrega (venta)' },
  { id: StockMovementType.SALE_ANNULMENT_RETURN, name: 'Return (sale annulment)|Reingreso (anulación de venta)' },
  { id: StockMovementType.ASSET_SERIAL_CORRECTION, name: 'Asset serial correction|Corrección de serie de activo' },
  { id: StockMovementType.OPENING_STOCK, name: 'Opening stock|Saldo inicial' },
  { id: StockMovementType.INTERNAL_CONSUMPTION, name: 'Internal consumption|Consumo interno' },
  { id: StockMovementType.SHRINKAGE, name: 'Shrinkage|Merma' },
]

// The reasons a user may give a manual stock change, by direction — the backend's
// manualStockAdjustmentTypes. The first of each list is the default.
export const manualStockIncreaseTypes = stockMovementTypes.filter((movementType) =>
  [StockMovementType.MANUAL_ENTRY, StockMovementType.OPENING_STOCK].includes(movementType.id as 1 | 12))
export const manualStockDecreaseTypes = stockMovementTypes.filter((movementType) =>
  [StockMovementType.MANUAL_EXIT, StockMovementType.SHRINKAGE, StockMovementType.INTERNAL_CONSUMPTION].includes(movementType.id as 2 | 13 | 14))
