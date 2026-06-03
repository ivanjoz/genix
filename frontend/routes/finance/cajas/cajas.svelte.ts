import { GetHandler, POST, GET } from '$libs/http.svelte';
import { GETCached } from '$libs/cache/cache-query-by-id';
import { formatTime } from '$libs/helpers';
import { Notify } from '$libs/helpers';

export interface ICashBank {
  ID: number
  SiteID: number
  Name: string
  Description: string
  CurrencyType: number
  ReconciliationDate: number
  CurrentAmount: number
  ReconciliationAmount: number
  Type: number
  ss: number
  upd: number
}

export interface ICajaResult {
  Cajas: ICashBank[]
  CajasMap: Map<number, ICashBank>
}

export interface ICashBankMovement {
  ID: number
  CashBankID: number
  CajaRefID: number
  VentaID: number
  DocumentID: number
  ReferenceID: number
  Date: number          // UnixDay the movement occurred.
  Type: number
  Amount: number        // Outflows are negative.
  FinalAmount: number
  Created: number
  CreatedBy: number
}

export interface ICashReconciliation {
  ID: number
  Type: number
  CashBankID: number
  SaldoSistema: number
  DifferenceAmount: number
  ActualAmount: number
  Created: number
  CreatedBy: number
  _error?: string
}

export class CajasService extends GetHandler {
  route = "cajas"
  useCache = { min: 1, ver: 1 }

  Cajas: ICashBank[] = $state([])
  CajasMap: Map<number, ICashBank> = $state(new Map())

  handler(result: ICajaResult): void {
    console.log("result cajas::", result)
    this.Cajas = result.Cajas || []
    for (let e of this.Cajas) {
      e.CurrentAmount = e.CurrentAmount || 0
    }
    this.CajasMap = new Map(this.Cajas.map(x => [x.ID, x]))
  }

  constructor() {
    super()
    this.fetch()
  }
}

export const postCaja = (data: ICashBank) => {
  return POST({
    data,
    route: "cajas",
    refreshRoutes: ["cajas"]
  })
}

export interface IGetCajaMovimientos {
  CajaID: number
  dateInicio?: number
  dateFin?: number
  lastRegistros?: number
}

export interface ICajaMovimientosResult {
  movimientos: ICashBankMovement[]
}

export const getCajaMovimientos = async (args: IGetCajaMovimientos): Promise<ICashBankMovement[]> => {
  let route = `cash-banks-movements?caja-id=${args.CajaID}`

  if ((!args.dateInicio || !args.dateFin) && !args.lastRegistros) {
    throw ("No se encontró una date de inicio o fin.")
  }

  if (args.dateInicio && args.dateFin) {
    route += `&date-inicio=${args.dateInicio}`
    route += `&date-fin=${args.dateFin}`
  }
  if (args.lastRegistros) {
    route += `&last-registros=${args.lastRegistros}`
  }

  let result: ICajaMovimientosResult

  try {
    result = await GET({ route })
  } catch (error) {
    console.log("Error:", error)
    Notify.failure(error as string)
    throw error
  }

  return result.movimientos || []
}

// Fetch the cash-bank movements tied to a document (e.g. an Expense) or a reference (e.g. an
// ExpenseScheduled), via the DocumentID / ReferenceID local indexes (GET.cash-bank-movement-by-id).
export const getCashBankMovementByID = async (
  args: { documentID?: number, referenceID?: number, updated?: number },
): Promise<ICashBankMovement[]> => {
  let route = "cash-bank-movement-by-id?"
  if (args.documentID) route += `document-id=${args.documentID}`
  else if (args.referenceID) route += `reference-id=${args.referenceID}`
  else throw ("Debe enviar un documentID o un referenceID.")

  // When the caller provides the parent's `updated` watermark, serve from the route-keyed
  // cache: it returns the stored movements until `updated` changes, then re-fetches.
  if (typeof args.updated === 'number') {
    try {
      return await GETCached<ICashBankMovement>(route, args.updated, p => p?.movimientos || [])
    } catch (error) {
      Notify.failure(error as string)
      throw error
    }
  }

  let result: ICajaMovimientosResult
  try {
    result = await GET({ route })
  } catch (error) {
    Notify.failure(error as string)
    throw error
  }
  return result.movimientos || []
}

export const postCajaMovimiento = (data: ICashBankMovement) => {
  return POST({
    data,
    route: "cash-banks-movement",
    refreshRoutes: ["cajas"]
  })
}

export const postCajaCuadre = (data: ICashReconciliation) => {
  return POST({
    data,
    route: "cash-bank-reconciliation",
    refreshRoutes: ["cajas"]
  })
}

export interface ICajaCuadresResult {
  cuadres: ICashReconciliation[]
}

export const getCajaCuadres = async (args: IGetCajaMovimientos): Promise<ICashReconciliation[]> => {
  let route = `cash-bank-reconciliations?caja-id=${args.CajaID}`

  if ((!args.dateInicio || !args.dateFin) && !args.lastRegistros) {
    throw ("No se encontró una date de inicio o fin.")
  }

  if (args.dateInicio && args.dateFin) {
    route += `&date-hora-inicio=${args.dateInicio * 24 * 60 * 60}`
    route += `&date-hora-fin=${(args.dateFin + 1) * 24 * 60 * 60}`
  }
  if (args.lastRegistros) {
    route += `&last-registros=${args.lastRegistros}`
  }

  let result: ICajaCuadresResult

  try {
    result = await GET({ route })
  } catch (error) {
    console.log("Error:", error)
    Notify.failure(error as string)
    throw error
  }

  return result.cuadres || []
}

// Constantes compartidas
export const cajaTipos = [
  { id: 1, name: "Caja" },
  { id: 2, name: "Cuenta Bancaria" }
]

// Currency enum, must match CashBank.CurrencyType / Expense.CurrencyType on the backend.
export const cajaMonedaTipos = [
  { id: 1, name: "PEN" },
  { id: 2, name: "USD" }
]

export const cajaMovimientoTipos = [
  { id: 1, name: "-", group: 1 },
  { id: 2, name: "Cuadre Físico", group: 1 },
  { id: 3, name: "Transferencia", group: 2, isNegative: true },
  { id: 4, name: "Retiro", group: 2, isNegative: true },
  { id: 5, name: "Pérdida", group: 2, isNegative: true },
  { id: 6, name: "Pago Proveedor", group: 2, isNegative: true },
	{ id: 7, name: "Cobro", group: 2 },
  { id: 8, name: "Cobro (Venta)", group: 2 },
  { id: 9, name: "Pago Gasto", group: 2, isNegative: true }
]
