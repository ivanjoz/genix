import { GetHandler, GETWithGroupCache } from '$libs/http.svelte'
import { Notify } from '$libs/helpers'

// Backend status codes for purchase orders.
export const PurchaseOrderStatus = {
  CANCELED: 0,
  PENDING: 1,
  FULFILLED: 2,
} as const

// Status selector options shared by the report filter and its summary strip.
// Using 0 as "all" keeps the URL query clean: the backend ignores status when it's 0.
export const purchaseOrderStatusOptions = [
  { ID: PurchaseOrderStatus.PENDING, Nombre: 'Pendiente' },
  { ID: PurchaseOrderStatus.FULFILLED, Nombre: 'Completada' },
  { ID: PurchaseOrderStatus.CANCELED, Nombre: 'Cancelada' },
]

// Minimal shape read by the report; full record lives on the backend.
export interface IPurchaseOrder {
  ID: number
  Date: number
  DateOfDelivery: number
  ProviderID: number
  TotalAmount: number
  Notes: string
  DetailProductIDs?: number[]
  DetailQuantities?: number[]
  DetailPrices?: number[]
  DetailPresentationIDs?: number[]
  ss: number
  upd: number
}

export interface IPurchaseOrderGroupRecord {
  id: number
  igVal: number[]
  records: IPurchaseOrder[]
  upc: number
}

export interface IPurchaseOrderReportForm {
  fechaInicio: number
  fechaFin: number
  status: number
  productID: number
  providerID: number
}

// Cached fetch of purchase orders filtered by status; ss>0 = active (soft-delete aware).
export class PurchaseOrdersService extends GetHandler<IPurchaseOrder> {
  route = ''
  useCache = { min: 0.2, ver: 1 }

  records: IPurchaseOrder[] = $state([])
  recordsMap: Map<number, IPurchaseOrder> = $state(new Map())

  constructor(status: number = PurchaseOrderStatus.PENDING, init: boolean = false) {
    super()
    this.route = `purchase-orders?status=${status}`
    if (init) { this.fetch() }
  }

  handler(result: IPurchaseOrder[]): void {
    const active = (result || []).filter((r) => (r.ss || 0) > 0)
    this.records = active
    this.recordsMap = new Map(active.map((r) => [r.ID, r]))
    // Newest first by ID (IDs are monotonically assigned on creation).
    this.records.sort((a, b) => b.ID - a.ID)
  }
}

// Hard client cap so extreme ranges never freeze the UI; sorted by most recent first.
const MAX_REPORT_RECORDS = 2000

// Report-mode query: backend indexes are grouped by Week, so the frontend still trims to
// the exact [fechaInicio, fechaFin] range (week boundaries may spill ±6 days outside it)
// and post-filters by status when the backend cannot (no Week+Status grouped index).
export const queryPurchaseOrders = async (filters: IPurchaseOrderReportForm): Promise<IPurchaseOrder[]> => {
  if (!filters.fechaInicio || !filters.fechaFin) {
    throw new Error('Debe especificar la fecha inicial y final.')
  }

  const queryParams = new URLSearchParams()
  queryParams.set('fecha-start', String(filters.fechaInicio))
  queryParams.set('fecha-end', String(filters.fechaFin))
  if ((filters.status || 0) > 0) { queryParams.set('status', String(filters.status)) }
  if ((filters.productID || 0) > 0) { queryParams.set('product-id', String(filters.productID)) }
  if ((filters.providerID || 0) > 0) { queryParams.set('provider-id', String(filters.providerID)) }

  const route = 'purchase-orders-query'
  const uriParams = Object.fromEntries(queryParams.entries())

  let result: IPurchaseOrderGroupRecord[]
  try {
    result = await GETWithGroupCache<IPurchaseOrder>(route, uriParams)
  } catch (error) {
    Notify.failure(String(error || 'No se pudo consultar el reporte de órdenes de compra.'))
    throw error
  }

  // ss holds the business status (0=Canceled, 1=Pending, 2=Fulfilled) — do not filter by ss>0 here
  // because status=0 (Canceled) is a legitimate filter target in report mode.
  const purchaseOrders = (result || [])
    .flatMap((groupRecord) => groupRecord.records || [])
    .filter((r) => (r?.ID || 0) > 0)
    // Trim to the exact day range because the backend uses Week-based grouped indexes.
    .filter((r) => (r.Date || 0) >= filters.fechaInicio && (r.Date || 0) <= filters.fechaFin)

  const statusFilter = filters.status || 0
  // Client-side status filter covers the status-only case where the backend cannot narrow
  // (no Week+Status grouped index exists).
  const finalRows = statusFilter > 0
    ? purchaseOrders.filter((r) => (r.ss || 0) === statusFilter)
    : purchaseOrders

  // Newest first by ID (IDs are monotonically assigned on creation).
  finalRows.sort((a, b) => b.ID - a.ID)

  return finalRows.slice(0, MAX_REPORT_RECORDS)
}
