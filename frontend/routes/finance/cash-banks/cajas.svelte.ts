import { notifyFailure } from '@genix/ui/notify';
import { GetHandler, GETWithGroupCache, POST } from '#libs/ui-runtime.svelte.ts';

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
  // The single account the web store's Culqi card/Yape charges are collected into.
  ReceivesCulqiPayments: boolean
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
  CounterAccountCode: number // PCGE counter-account, only for the types in CASH_MOVEMENT_ACCOUNT_OPTIONS.
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
  CounterAccountCode: number // Sent with the request; the backend stores it on the movement.
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

// A cash bank's movements of a date range, read as its (cash bank, day) cache groups so only the
// days that moved are downloaded again. With lastRegistros the server sends the latest N movements
// uncached (a pass-through group), newest first.
export const getCajaMovimientos = async (args: IGetCajaMovimientos): Promise<ICashBankMovement[]> => {
  const uriParams: Record<string, string> = { 'caja-id': String(args.CajaID) }
  if (args.lastRegistros) {
    uriParams['last-registros'] = String(args.lastRegistros)
  } else if (args.dateInicio && args.dateFin) {
    uriParams['date-inicio'] = String(args.dateInicio)
    uriParams['date-fin'] = String(args.dateFin)
  } else {
    throw ("No se encontró una fecha de inicio o fin.")
  }

  try {
    const groups = await GETWithGroupCache<ICashBankMovement>('cash-banks-movements', uriParams)
    return groups.flatMap((group) => group.records || [])
      .filter((movement) => (movement?.ID || 0) > 0)
      .sort((a, b) => b.ID - a.ID)
  } catch (error) {
    notifyFailure(error as string)
    throw error
  }
}

// The cash-bank movements of one document (e.g. an Expense), as that document's cache group: it is
// downloaded again only after a new movement of the document (GET.cash-bank-movement-by-id).
// DocumentID is shared by every document kind, so only the movements of `movementType` are kept.
export const getDocumentCashMovements = async (documentID: number, movementType: number): Promise<ICashBankMovement[]> => {
  try {
    const groups = await GETWithGroupCache<ICashBankMovement>('cash-bank-movement-by-id', { 'document-id': String(documentID) })
    return groups.flatMap((group) => group.records || [])
      .filter((movement) => (movement?.ID || 0) > 0 && movement.Type === movementType)
  } catch (error) {
    notifyFailure(error as string)
    throw error
  }
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

// One cash bank's physical counts, synced through its delta log (the route carries the cash bank,
// so each one is its own cached list). The caller calls fetch().
export class CashReconciliationsService extends GetHandler<ICashReconciliation> {
  route = "cash-banks-reconciliations"
  useCache = { min: 0.2, ver: 1 }

  records: ICashReconciliation[] = $state([])
  recordsMap: Map<number, ICashReconciliation> = $state(new Map())

  constructor(cashBankID: number) {
    super()
    this.route += `?caja-id=${cashBankID}`
  }

  handler(result: ICashReconciliation[]): void {
    this.records = []
    this.recordsMap = new Map()
    this.addSavedRecords(...(result || []))
    this.records.sort((a, b) => b.Created - a.Created)
  }
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
  { id: 11, name: "Devolución (Anulación Venta)", group: 3, isNegative: true },
  { id: 12, name: "Pago Planilla", group: 3, isNegative: true },
  { id: 13, name: "Pago Aportes Planilla", group: 3, isNegative: true },
]

// The movement an expense payment writes (backend CashMovementTypeExpensePayment).
export const CASH_MOVEMENT_TYPE_EXPENSE_PAYMENT = 9

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
  { movementType: 7, accountCode: 501, name: 'Capital contribution|Aporte de capital' },
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
