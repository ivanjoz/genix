import { notifyFailure } from '@genix/ui/notify';
import { GetHandler, POST, GET } from '#libs/ui-runtime.svelte.ts';
import { GETCached } from '@genix/ui/cache';
import { formatTime } from '#libs/helpers.ts';

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
  CashBankRefID: number
  VentaID: number
  DocumentID: number
  ReferenceID: number
  Date: number          // UnixDay the movement occurred.
  Type: number
  AccountCode: number   // PCGE counter-account, only for the types in CASH_MOVEMENT_ACCOUNT_OPTIONS.
  Amount: number        // Outflows are negative.
  FinalAmount: number
  Created: number
  CreatedBy: number
}

export interface ICashReconciliation {
  ID: number
  Type: number
  CashBankID: number
  SystemAmount: number
  DifferenceAmount: number
  ActualAmount: number
  AccountCode: number   // Sent with the request; the backend stores it on the movement.
  Created: number
  CreatedBy: number
  _error?: string
}

export class CajasService extends GetHandler {
  route = "cash-banks"
  useCache = { min: 1, ver: 2 }

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
    route: "cash-banks",
    refreshRoutes: ["cash-banks"]
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
    notifyFailure(error as string)
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
      notifyFailure(error as string)
      throw error
    }
  }

  let result: ICajaMovimientosResult
  try {
    result = await GET({ route })
  } catch (error) {
    notifyFailure(error as string)
    throw error
  }
  return result.movimientos || []
}

export const postCajaMovimiento = (data: ICashBankMovement) => {
  return POST({
    data,
    route: "cash-banks-movement",
    refreshRoutes: ["cash-banks"]
  })
}

export const postCajaCuadre = (data: ICashReconciliation) => {
  return POST({
    data,
    route: "cash-banks-reconciliation",
    refreshRoutes: ["cash-banks"]
  })
}

export interface ICajaCuadresResult {
  cuadres: ICashReconciliation[]
}

export const getCajaCuadres = async (args: IGetCajaMovimientos): Promise<ICashReconciliation[]> => {
  let route = `cash-banks-reconciliations?caja-id=${args.CajaID}`

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
    notifyFailure(error as string)
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

// group 2 = registered by hand from the cash register (backend ManualCashMovementTypes);
// group 3 = written by its own document (order, sale, expense, asset) and never typed here.
export const cajaMovimientoTipos = [
  { id: 1, name: "-", group: 1 },
  { id: 2, name: "Cuadre Físico", group: 1 },
  { id: 3, name: "Transferencia", group: 2, isNegative: true },
  { id: 4, name: "Retiro", group: 2, isNegative: true },
  { id: 5, name: "Pérdida", group: 2, isNegative: true },
  { id: 6, name: "Pago Proveedor", group: 3, isNegative: true },
  { id: 7, name: "Cobro", group: 2 },
  { id: 8, name: "Cobro (Venta)", group: 3 },
  { id: 9, name: "Pago Gasto", group: 3, isNegative: true },
  { id: 10, name: "Pago Activo", group: 3, isNegative: true },
  { id: 11, name: "Devolución (Anulación Venta)", group: 3, isNegative: true }
]

export interface ICashMovementAccountOption {
  movementType: number
  /** -1 outflow, 1 inflow, absent = either. Only a physical count needs it. */
  direction?: number
  accountCode: number
  name: string
}

/** Generated from `CashMovementAccountOptions` in backend/finance/types/cash_movement_account.go —
 *  run `go run ./scripts sync_struct_interfaces` after changing it there. */
//CATALOG:finance.CashMovementAccountOptions
export const CASH_MOVEMENT_ACCOUNT_OPTIONS: ICashMovementAccountOption[] = [
  { movementType: 4, accountCode: 142, name: 'Owner or shareholder withdrawal|Retiro del socio o accionista' },
  { movementType: 4, accountCode: 441, name: 'Dividend payment|Pago de dividendos' },
  { movementType: 5, accountCode: 659, name: 'Other expense|Otro gasto' },
  { movementType: 5, accountCode: 1419, name: 'Charged to an employee|Cargo a un trabajador' },
  { movementType: 2, direction: -1, accountCode: 659, name: 'Other expense|Otro gasto' },
  { movementType: 2, direction: -1, accountCode: 1419, name: 'Charged to the cashier|Cargo al cajero' },
  { movementType: 2, direction: 1, accountCode: 759, name: 'Other income|Otro ingreso' },
  { movementType: 7, accountCode: 759, name: 'Other income|Otro ingreso' },
  { movementType: 7, accountCode: 142, name: 'Owner or shareholder repayment|Devolución del socio o accionista' },
  { movementType: 7, accountCode: 1419, name: 'Repayment by an employee|Devolución de un trabajador' },
  { movementType: 7, accountCode: 50, name: 'Capital contribution|Aporte de capital' },
  { movementType: 7, accountCode: 451, name: 'Loan received|Préstamo recibido' },
]

/** The accounts a movement of this type and amount may post against, mirroring
 *  CashMovementAccountOptionsFor in the backend. The form asks only when there is more than
 *  one; a single option is assigned by the backend. */
export const cashMovementAccountOptions = (movementType: number, amount: number): ICashMovementAccountOption[] => {
  if (!amount) return []
  const direction = amount < 0 ? -1 : 1
  return CASH_MOVEMENT_ACCOUNT_OPTIONS.filter((option) =>
    option.movementType === movementType && (!option.direction || option.direction === direction))
}
