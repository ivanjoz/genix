import type { ICashFlowEntry, ICashFlowServerWarning } from '#services/finance/cash-flow.svelte.ts'
import { CURRENCY_USD, toPEN } from './cash-flow.ts'

// The warnings layer (plan/CASH_FLOW_FORECAST_PLAN.md §11): every gap that degrades the forecast,
// what the forecast assumed instead, and the page where it is fixed. The backend raises the checks
// that need its data (payroll, loans, tax setup); cash-flow-documents.ts the ones of the documents,
// from the cash-documents frame (one record per document day and site); the ones below need only
// what the page already holds (registers, rates, planned entries). All share this catalog.

export type CashFlowSeverity = 'high' | 'medium' | 'info'

export interface ICashFlowWarning extends ICashFlowServerWarning {
  severity: CashFlowSeverity
  message: string
  route: string
}

export const SEVERITY_ORDER: CashFlowSeverity[] = ['high', 'medium', 'info']

export const CASH_FLOW_WARNING_CATALOG: Record<string, { severity: CashFlowSeverity, message: string, route: string }> = {
  'tax-regime-missing': { severity: 'high', route: '/company/configuration',
    message: 'The company has no tax regime: no income tax prepayment is projected.|La empresa no tiene régimen tributario: no se proyecta el pago a cuenta de renta.' },
  'tax-calendar-missing': { severity: 'high', route: '/system/tax-calendar',
    message: 'No SUNAT calendar for these periods: their taxes and contributions fall due on the 15th of the next month.|Sin calendario SUNAT para estos periodos: sus impuestos y aportes vencen el 15 del mes siguiente.' },
  'tax-period-unpaid': { severity: 'high', route: '/finance/expenses',
    message: 'A closed period\'s tax is past due with no payment recorded: projected as due today.|El impuesto de un periodo cerrado venció sin pago registrado: se proyecta para hoy.' },
  'purchase-without-document': { severity: 'medium', route: '/finance/expenses',
    message: 'Purchases without a comprobante carry no IGV credit, so the IGV payable is overstated.|Compras sin comprobante no dan crédito fiscal: el IGV por pagar sale sobrestimado.' },
  'sale-without-due-date': { severity: 'medium', route: '/sales/sale_orders_status',
    message: 'Unpaid sales without a due date: collected on the sale date.|Ventas por cobrar sin fecha de vencimiento: se cobran en la fecha de la venta.' },
  'receivable-stale': { severity: 'high', route: '/sales/sale_orders_status',
    message: 'Debts overdue more than 90 days are left out: probably uncollectible or a payment not recorded.|Deudas vencidas hace más de 90 días quedan fuera: probablemente incobrables o un pago no registrado.' },
  'client-credit-days-missing': { severity: 'medium', route: '/crm/customers',
    message: 'Clients that owe money on cash terms: their sales fall due on the sale date.|Clientes con deuda en condición de contado: sus ventas vencen en la fecha de la venta.' },
  'purchase-order-without-payment-date': { severity: 'medium', route: '/logistics/purchase-orders',
    message: 'Purchase orders with debt and no payment date: paid on the order date.|Órdenes de compra con deuda y sin fecha de pago: se pagan en la fecha de la orden.' },
  'payable-overdue': { severity: 'medium', route: '/finance/expenses',
    message: 'Unpaid orders, expenses or assets past their due date: projected as paid today.|Órdenes, gastos o activos vencidos sin pagar: se proyectan como pagados hoy.' },
  'payables-before-window': { severity: 'high', route: '/finance/expenses',
    message: 'Unpaid orders, expenses or assets older than a year: projected as paid today.|Órdenes, gastos o activos por pagar de hace más de un año: se proyectan como pagados hoy.' },
  'sale-without-site': { severity: 'info', route: '/sales/sale_orders_status',
    message: 'Sales of the last months without a site: counted in the company lane.|Ventas de los últimos meses sin sede: cuentan en la columna de la empresa.' },
  'expense-without-site': { severity: 'info', route: '/finance/expenses',
    message: 'Expenses of the last months without a site: counted in the company lane.|Gastos de los últimos meses sin sede: cuentan en la columna de la empresa.' },
  'schedule-amount-zero': { severity: 'medium', route: '/finance/expenses',
    message: 'Recurring bills with no amount and no past periods: not projected.|Gastos programados sin monto ni periodos anteriores: no se proyectan.' },
  'loan-installment-overdue': { severity: 'high', route: '/finance/loans',
    message: 'Unpaid loan installments past their due date: projected as paid today.|Cuotas de préstamo vencidas sin pagar: se proyectan como pagadas hoy.' },
  'loan-received-as-cobro': { severity: 'high', route: '/finance/loans',
    message: 'Collections booked against loans received (451): a loan with no schedule, so its repayments are unknown. Register it in Loans.|Cobros registrados contra préstamos recibidos (451): un préstamo sin cronograma, sus cuotas son desconocidas. Regístrelo en Préstamos.' },
  'loan-rate-missing': { severity: 'info', route: '/finance/loans',
    message: 'Loans without a rate: their installments are taken as entered.|Préstamos sin tasa: sus cuotas se toman tal como se ingresaron.' },
  'payday-missing': { severity: 'medium', route: '/sales/calendar-events',
    message: 'No payday in the company calendar for these months: payroll is paid on the last day.|Sin día de pago en el calendario de la empresa para estos meses: la planilla se paga el último día.' },
  'payroll-contract-missing': { severity: 'high', route: '/hr/employees',
    message: 'Active employees without a contract in force: left out of the payroll projection.|Trabajadores activos sin contrato vigente: quedan fuera de la proyección de planilla.' },
  'payroll-run-open': { severity: 'medium', route: '/hr/payroll',
    message: 'Last month\'s payroll is not closed: projected from the payroll engine.|La planilla del mes pasado no está cerrada: se proyecta con el motor de planilla.' },
  'unspecified-cash-movements': { severity: 'info', route: '/finance/cash-banks-movements',
    message: 'Unspecified cash movements in the last 90 days: taken into the history as they are.|Movimientos de caja sin especificar en los últimos 90 días: entran al historial tal cual.' },
  'cash-bank-negative': { severity: 'high', route: '/finance/cash-banks-movements',
    message: 'Registers with a negative balance: taken as is.|Cajas o bancos con saldo negativo: se toman tal cual.' },
  'cash-bank-not-reconciled': { severity: 'medium', route: '/finance/cash-banks',
    message: 'Registers with no count in the last 30 days: their balance is taken as is.|Cajas o bancos sin arqueo en los últimos 30 días: su saldo se toma tal cual.' },
  'cash-bank-without-site': { severity: 'info', route: '/finance/cash-banks',
    message: 'Registers without a site: their cash is in the company lane.|Cajas o bancos sin sede: su saldo está en la columna de la empresa.' },
  'exchange-rate-stale': { severity: 'medium', route: '/finance/exchange-rate',
    message: 'Dollar amounts and no exchange rate in the last 7 days: dollars are taken at 1 to 1.|Hay montos en dólares y ningún tipo de cambio en los últimos 7 días: los dólares se toman 1 a 1.' },
  'planned-entry-past': { severity: 'medium', route: '/finance/cash-flow',
    message: 'Planned entries dated before today no longer count: move or delete them.|Entradas planificadas con fecha pasada ya no cuentan: muévalas o elimínelas.' },
  // The statistical layer's own (cash-flow-forecast.ts): nothing to fix on another page, only time.
  'history-short': { severity: 'medium', route: '',
    message: 'Sites selling for less than 90 days: their forecast sales have no weekly or yearly shape of their own yet.|Sedes que venden hace menos de 90 días: sus ventas pronosticadas aún no tienen forma semanal ni anual propia.' },
  'history-under-year': { severity: 'info', route: '',
    message: 'Less than a year of sales: the forecast has no yearly season, so the monthly view is flatter than the year will be.|Menos de un año de ventas: el pronóstico no tiene estacionalidad anual, así que la vista mensual es más plana de lo que será el año.' },
  'delay-curve-missing': { severity: 'medium', route: '',
    message: 'Sites with few document payments in 180 days: their forecast cash uses the company\'s collection or payment times, or the document day.|Sedes con pocos pagos de documentos en 180 días: su caja pronosticada usa los plazos de cobro o pago de la empresa, o el día del documento.' },
}

const RECORD_LIMIT = 50
const RECONCILIATION_MAX_AGE_DAYS = 30

/** Builds one warning from its offending records, capped like the backend's. */
const makeWarning = (code: string, records: { ID: number, Label: string, amountPEN: number }[]): ICashFlowServerWarning => ({
  Code: code,
  AmountAtStakePEN: records.reduce((total, record) => total + Math.abs(record.amountPEN), 0),
  Records: records.slice(0, RECORD_LIMIT).map(({ ID, Label }) => ({ ID, Label })),
  RecordCount: records.length,
})

/** The checks on what the page holds: registers, the dollar rate and the planned entries. */
export function pageWarnings(input: {
  cashBanks: { ID: number, Name: string, SiteID: number, CurrencyType: number, CurrentAmount: number, ReconciliationDate: number, ss: number }[]
  entries: ICashFlowEntry[]
  usesDollars: boolean
  usdSellRate: number
  today: number
}): ICashFlowServerWarning[] {
  const activeCashBanks = input.cashBanks.filter((cashBank) => cashBank.ss > 0)
  const asRecord = (cashBank: typeof activeCashBanks[number]) => ({ ID: cashBank.ID, Label: cashBank.Name,
    amountPEN: toPEN(cashBank.CurrentAmount, cashBank.CurrencyType, input.usdSellRate || 1000) })
  const checks: [string, { ID: number, Label: string, amountPEN: number }[]][] = [
    ['cash-bank-negative', activeCashBanks.filter((cashBank) => cashBank.CurrentAmount < 0).map(asRecord)],
    ['cash-bank-not-reconciled', activeCashBanks.filter((cashBank) => cashBank.ReconciliationDate < input.today - RECONCILIATION_MAX_AGE_DAYS).map(asRecord)],
    ['cash-bank-without-site', activeCashBanks.filter((cashBank) => !cashBank.SiteID).map(asRecord)],
    ['exchange-rate-stale', input.usesDollars && !input.usdSellRate ? [{ ID: 0, Label: 'USD', amountPEN: 0 }] : []],
    ['planned-entry-past', input.entries.filter((entry) => entry.ss > 0 && entry.Date < input.today)
      .map((entry) => ({ ID: entry.ID, Label: entry.Name, amountPEN: entry.Amount }))],
  ]
  return checks.filter(([, records]) => records.length > 0).map(([code, records]) => makeWarning(code, records))
}

/** Whether any money of the forecast is in dollars, which makes the rate matter. */
export const usesDollars = (currencyTypes: number[]) => currencyTypes.some((currencyType) => currencyType === CURRENCY_USD)

/** Server and page warnings with their catalog entry, by severity then amount at stake. */
export function rankWarnings(warnings: ICashFlowServerWarning[]): ICashFlowWarning[] {
  return warnings
    .map((warning) => {
      const entry = CASH_FLOW_WARNING_CATALOG[warning.Code] || { severity: 'info' as const, message: warning.Code, route: '' }
      return { ...warning, ...entry }
    })
    .sort((left, right) => SEVERITY_ORDER.indexOf(left.severity) - SEVERITY_ORDER.indexOf(right.severity)
      || right.AmountAtStakePEN - left.AmountAtStakePEN)
}
