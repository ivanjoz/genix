import { GETWithGroupCache } from '$libs/http.svelte';
import { Notify } from '$libs/helpers';
import type { IProductStockLot } from '../products-stock/stock-movement';

interface IQueryAlmacenMovimientos {
  almacenID: number
  fechaInicio: number
  fechaFin: number
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
  Fecha?: number
  DocumentID?: number
  ProductoID?: number
  PresentacionID?: number
  Quantity?: number
  WarehouseQuantity?: number
  SubQuantity?: number
  MonetaryValue?: number
  Tipo?: number
  Created?: number
  CreatedBy?: number
  upc?: number
}

export interface IWarehouseProductMovementGroupRecord {
  id: number
  igVal: number[]
  records: IWarehouseProductMovement[]
  upc: number
}

export const movimientoTipos = [
  { id: 1, name: 'Entrada Manual' },
  { id: 2, name: 'Salida Manual' },
]

export const queryAlmacenMovimientos = async (args: IQueryAlmacenMovimientos): Promise<IWarehouseProductMovement[]> => {
  const isDirectLookup = !!(args.serialNumber?.trim() || args.lotCode?.trim() || (args.documentID || 0) > 0)
  if (!isDirectLookup && (!args.fechaInicio || !args.fechaFin)) {
    throw new Error('No se encontró una fecha de inicio o fin.')
  }

  const queryParams = new URLSearchParams()
  queryParams.set('fecha-inicio', String(args.fechaInicio))
  queryParams.set('fecha-fin', String(args.fechaFin))
  if (args.almacenID > 0) { queryParams.set('almacen-id', String(args.almacenID)) }
  if (args.productoID > 0) { queryParams.set('producto-id', String(args.productoID)) }
  if (args.tipo > 0) { queryParams.set('tipo', String(args.tipo)) }
  if (args.lotCode?.trim()) { queryParams.set('lot-code', args.lotCode.trim()) }
  if (args.documentID > 0) { queryParams.set('document-id', String(args.documentID)) }
  if (args.serialNumber?.trim()) { queryParams.set('serial-number', args.serialNumber.trim()) }

  const route = 'almacen-movimientos'
  const uriParams = Object.fromEntries(queryParams.entries())

  let result: IWarehouseProductMovementGroupRecord[]
  try {
    result = await GETWithGroupCache<IWarehouseProductMovement>(route, uriParams)
  } catch (error) {
    Notify.failure(String(error || 'Hubo un error al obtener los movimientos del almacén'))
    throw error
  }

  const movimientos = (result || [])
    .flatMap((groupRecord) => groupRecord.records || [])
    .filter((movement) => !!movement?.ID)

  // Keep the table stable across mixed grouped responses.
  movimientos.sort((a, b) => (b.Created || 0) - (a.Created || 0))

  console.log("movimientos::", movimientos)
  
  return movimientos
}
