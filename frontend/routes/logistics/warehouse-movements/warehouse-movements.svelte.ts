import { notifyFailure } from '@genix/ui/notify';
import { GETWithGroupCache } from '#libs/ui-runtime.svelte.ts';
import type { IProductStockLot } from '../products-stock/stock-movement';

interface IQueryAlmacenMovimientos {
  almacenID: number
  dateInicio: number
  dateFin: number
  productoID?: number
  lotCode?: string
  documentID?: number
  serialNumber?: string
  tipo?: number
}

export type { IProductStockLot }

export interface IWarehouseProductMovement {
  ID: number
  CompanyID?: number
  SerialNumber?: string
  LotID?: number
  lot?: IProductStockLot
  WarehouseID?: number
  WarehouseRefID?: number
  WarehouseRefQuantity?: number
  Date?: number
  DocumentID?: number
  ProductID?: number
  PresentationID?: number
  // The ledger keeps the pair in separate columns so it can accumulate with a plain + and
  // still be SUM()-ed. SubDivisor says what SubQuantity is counted in.
  Quantity?: number
  WarehouseQuantity?: number
  SubQuantity?: number
  SubDivisor?: number
  MonetaryValue?: number
  Type?: number
  Created?: number
  CreatedBy?: number
}

export interface IWarehouseProductMovementGroupRecord {
  id: number
  igVal: number[]
  records: IWarehouseProductMovement[]
  lw: number
}

export const queryAlmacenMovimientos = async (args: IQueryAlmacenMovimientos): Promise<IWarehouseProductMovement[]> => {
  const isDirectLookup = !!(args.serialNumber?.trim() || args.lotCode?.trim() || (args.documentID || 0) > 0)
  if (!isDirectLookup && (!args.dateInicio || !args.dateFin)) {
    throw new Error('No se encontró una date de inicio o fin.')
  }

  const queryParams = new URLSearchParams()
  queryParams.set('date-inicio', String(args.dateInicio))
  queryParams.set('date-fin', String(args.dateFin))
  // Names must match GetAlmacenMovimientos: each present name selects a different cache group.
  if (args.almacenID > 0) { queryParams.set('warehouse-id', String(args.almacenID)) }
  if (args.productoID > 0) { queryParams.set('product-id', String(args.productoID)) }
  if (args.tipo > 0) { queryParams.set('tipo', String(args.tipo)) }
  if (args.lotCode?.trim()) { queryParams.set('lot-code', args.lotCode.trim()) }
  if (args.documentID > 0) { queryParams.set('document-id', String(args.documentID)) }
  if (args.serialNumber?.trim()) { queryParams.set('serial-number', args.serialNumber.trim()) }

  const route = 'warehouse-movements'
  const uriParams = Object.fromEntries(queryParams.entries())

  let result: IWarehouseProductMovementGroupRecord[]
  try {
    result = await GETWithGroupCache<IWarehouseProductMovement>(route, uriParams)
  } catch (error) {
    notifyFailure(String(error || 'Hubo un error al obtener los movimientos del almacén'))
    throw error
  }

  // The server reads one cache group per request (warehouse wins over product), so the filters
  // that group did not apply are applied here.
  const [warehouseID, productID, movementType] = isDirectLookup
    ? [0, 0, 0] : [args.almacenID || 0, args.productoID || 0, args.tipo || 0]
  const movimientos = (result || [])
    .flatMap((groupRecord) => groupRecord.records || [])
    .filter((movement) => !!movement?.ID
      && (!warehouseID || movement.WarehouseID === warehouseID)
      && (!productID || movement.ProductID === productID)
      && (!movementType || movement.Type === movementType))

  // Keep the table stable across mixed grouped responses.
  movimientos.sort((a, b) => (b.Created || 0) - (a.Created || 0))

  console.log("movimientos::", movimientos)
  
  return movimientos
}
