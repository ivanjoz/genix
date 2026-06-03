import { GetHandler, POST, GET } from '$libs/http.svelte'
import { Notify } from '$libs/helpers'
import { tr } from '$core/store.svelte'

// IExpense / IExpenseScheduled mirror the Go structs in backend/finance/types/expenses.go.
export interface IExpense {
  ID: number
  ExpenseScheduledID: number
  PeriodDate: number
  Name: string
  Description: string
  CategoryID: number
  SupplierID: number
  CurrencyType: number
  Date: number
  DueDate: number
  Amount: number
  PaidAmount: number
  Created: number  // SUnixTime the expense was registered.
  ss: number  // Payment lifecycle: 0 removed · 1 created/pending · 2 fully paid.
  upd: number
}

export interface IExpenseScheduled {
  ID: number
  Name: string
  Description: string
  CategoryID: number
  SupplierID: number
  CurrencyType: number
  Amount: number
  Frequency: number
  StartDate: number
  EndDate: number
  ss: number
  upd: number
}

// --- Static, code-defined lists (mirror backend/finance/expenses.go §2.0) ----------

// Bilingual labels resolved through tr() at render time (see localizeOptions).
export const expenseCategories = [
  { id: 1, label: "Rent|Alquiler" },
  { id: 2, label: "Utilities|Servicios" },
  { id: 3, label: "Payroll|Planilla" },
  { id: 4, label: "Supplies|Insumos" },
  { id: 5, label: "Taxes|Impuestos" },
  { id: 6, label: "Maintenance|Mantenimiento" },
  { id: 7, label: "Transport|Transporte" },
  { id: 8, label: "Marketing|Marketing" },
  { id: 9, label: "Professional services|Servicios profesionales" },
  { id: 10, label: "Other|Otros" },
]

export const currencyTypes = [
  { id: 1, label: "PEN" },
  { id: 2, label: "USD" },
]

// Cadence dropdown (the `C` digit of the packed Frequency code).
export const expenseCadences = [
  { id: 1, label: "Weekly|Semanal" },
  { id: 2, label: "Monthly|Mensual" },
  { id: 3, label: "Every 2 months|Cada 2 meses" },
  { id: 4, label: "Every 3 months|Cada 3 meses" },
  { id: 5, label: "Every 4 months|Cada 4 meses" },
  { id: 6, label: "Yearly|Anual" },
]

// Status tabs for the Register list. Each tab is its own server-side delta query
// (?status=<code>); code 0 = all live, 1 = pending payment, 2 = fully paid.
export const ExpenseStatus = {
  TODOS: 0,
  PEND_PAGO: 1,
  PAGADOS: 2,
}

export const expenseStatusTabs = [
  { id: ExpenseStatus.TODOS, label: "All|Todos" },
  { id: ExpenseStatus.PEND_PAGO, label: "Pend. Payment|Pend. Pago" },
  { id: ExpenseStatus.PAGADOS, label: "Paid|Pagados" },
]

// localizeOptions resolves the bilingual `label` into a SearchSelect-ready `name`.
// Call inside a $derived so options re-localize when the language switches.
export const localizeOptions = (options: { id: number, label: string }[]) =>
  options.map(o => ({ id: o.id, name: tr(o.label) }))

// --- Frequency packing helpers (Frequency = cadence*100 + day) ----------------------

export const packFrequency = (cadence: number, day: number) => cadence * 100 + day
export const unpackFrequency = (frequency: number) => ({
  cadence: Math.floor(frequency / 100),
  day: frequency % 100,
})

// frequencySummary renders a human cadence label, e.g. "Monthly · day 12".
export const frequencySummary = (frequency: number): string => {
  const { cadence, day } = unpackFrequency(frequency)
  const cadenceLabel = tr(expenseCadences.find(c => c.id === cadence)?.label || "")
  if (!cadenceLabel) return "-"
  if (cadence === 1) {
    // Weekly: `day` is a weekday 1=Mon..7=Sun.
    const names = [tr("Mon|Lun"), tr("Tue|Mar"), tr("Wed|Mié"), tr("Thu|Jue"), tr("Fri|Vie"), tr("Sat|Sáb"), tr("Sun|Dom")]
    return `${cadenceLabel} · ${names[day - 1] || day}`
  }
  return `${cadenceLabel} · ${tr("day|día")} ${day}`
}

// --- Delta-cached list services -----------------------------------------------------

// One status tab per instance. The status code is embedded in `route` (?status=<code>)
// so each tab keeps a separate delta-cache key; rows that change status are evicted by
// the backend's `records_IDsToRemove` flag (see GetExpenses / GetSaleOrders).
export class ExpensesService extends GetHandler<IExpense> {
  route = "expenses"
  keyID = "ID"
  // 30s TTL: re-clicking a tab within the window reads cache instead of hitting the server.
  // Route depends on the status tab; bump ver so old (status-less) caches are dropped.
  useCache = { min: 0.5, ver: 2 }
  prependOnSave = true

  records: IExpense[] = $state([])
  recordsMap: Map<number, IExpense> = $state(new Map())

  constructor(status: number) {
    super()
    this.route += `?status=${status}`
  }

  handler(result: { records?: IExpense[] }): void {
    this.records = []
    this.recordsMap = new Map()
    this.addSavedRecords(...(result?.records || []))
  }
}

export class ExpensesScheduledService extends GetHandler<IExpenseScheduled> {
  route = "expenses-scheduled"
  keyID = "ID"
  useCache = { min: 5, ver: 1 }
  inferRemoveFromStatus = true
  prependOnSave = true

  records: IExpenseScheduled[] = $state([])
  recordsMap: Map<number, IExpenseScheduled> = $state(new Map())

  constructor(init: boolean = false) {
    super()
    if (init) this.fetch()
  }

  handler(result: IExpenseScheduled[]): void {
    this.records = []
    this.recordsMap = new Map()
    this.addSavedRecords(...(result || []))
  }
}

// --- Report-style POST / GET calls --------------------------------------------------

export const postExpense = (data: IExpense): Promise<IExpense> => {
  return POST({ data, route: "expenses", refreshRoutes: ["expenses"] })
}

export const postExpenseScheduled = (data: IExpenseScheduled): Promise<IExpenseScheduled> => {
  return POST({ data, route: "expenses-scheduled", refreshRoutes: ["expenses-scheduled"] })
}

export interface IExpensePayment {
  ExpenseID: number
  CashBankID: number
  Amount: number       // positive payment amount, in cents
  Date: number         // payment date (UnixDay)
  IsFullyPaid: boolean
}

// postExpensePayment returns the updated IExpense. The resulting cash-bank balance is computed
// server-side; the only balance rule is that it can't go negative (rejected as a plain error).
export const postExpensePayment = (
  data: IExpensePayment,
): Promise<IExpense> => {
  // Refresh expenses (paid amount/status) and cajas (balance) caches after the payment.
  return POST({ data, route: "expense-payment", refreshRoutes: ["expenses", "cajas"] })
}

// getSchedulePeriods triggers lazy period materialization on the backend and returns
// the schedule's Expense periods.
export const getSchedulePeriods = async (scheduleID: number): Promise<IExpense[]> => {
  try {
    const result = await GET({ route: `expense-schedule-periods?scheduleID=${scheduleID}` })
    return result.Periods || []
  } catch (error) {
    Notify.failure(error as string)
    throw error
  }
}
