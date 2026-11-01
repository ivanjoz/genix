import { findPCGEAccount } from '#core/pcge.ts'
import type { IAccountingOperation, IOperationCashBank } from '../accounts-consolidation/accounts-consolidation'
import { convertUSD } from '../accounts-consolidation/accounts-consolidation.utils'

// The Estados Financieros of one period, built from the same entries as Consolidado & Cuentas.
// Every amount is PEN cents, signed as it moves the result (or the cash): + adds, − takes away.

export const STATEMENT_ROW_SECTION = 1   // a section header, carrying the section's total
export const STATEMENT_ROW_ACCOUNT = 2   // one account inside a section
export const STATEMENT_ROW_SUBTOTAL = 3  // a running result: Utilidad bruta, Variación neta…
export const STATEMENT_ROW_BALANCE = 4   // a cash balance: Saldo inicial, Saldo final

export interface IStatementRow {
  Kind: number          // STATEMENT_ROW_*
  Label: string         // bilingual; an account row's is "" and reads its PCGE name
  AccountCode: number   // only on an account row; 0 = the lines still unclassified
  Amount: number
}

export const VIEW_MANAGEMENT = 1
export const VIEW_OFFICIAL = 2
export const statementViewOptions = [
  { ID: VIEW_MANAGEMENT, Name: "Management|Gestión" },
  { ID: VIEW_OFFICIAL, Name: "Official|Oficial" },
]

// The official view reads only what the official Diario holds (the entries with a comprobante,
// and the internal ones), so its statements agree with the books sent to SUNAT.
export const operationsOfView = (operations: IAccountingOperation[], view: number): IAccountingOperation[] =>
  view === VIEW_OFFICIAL ? operations.filter(operation => operation.InOfficialBook) : operations

// A section groups the accounts whose code starts with one of its prefixes; a subtotal is the
// running sum of every section above it.
interface IStatementSection { Label: string, AccountPrefixes: string[] }
interface IStatementSubtotal { Label: string, IsSubtotal: true }

// The hybrid layout: revenue and cost of sales first, for the gross margin, then the expenses by
// nature. The entries carry no class 9 / 79 allocation, so a by-function layout is not possible.
// Cost of sales adds 60 + 61: a purchase received into stock nets out there, and only what never
// entered stock (or left it through 69) is cost. 69 is the cost of what was sold.
const INCOME_STATEMENT_LAYOUT: (IStatementSection | IStatementSubtotal)[] = [
  { Label: "Net sales|Ventas netas", AccountPrefixes: ["70", "74"] },
  { Label: "Cost of sales|Costo de ventas", AccountPrefixes: ["69", "60", "61"] },
  { Label: "Gross profit|Utilidad bruta", IsSubtotal: true },
  { Label: "Operating expenses|Gastos operativos", AccountPrefixes: ["62", "63", "64", "65", "66", "68"] },
  { Label: "Other operating income|Otros ingresos de gestión", AccountPrefixes: ["71", "72", "73", "75", "76", "78"] },
  { Label: "Operating profit|Utilidad operativa", IsSubtotal: true },
  { Label: "Financial income|Ingresos financieros", AccountPrefixes: ["77"] },
  { Label: "Financial expenses|Gastos financieros", AccountPrefixes: ["67"] },
  { Label: "Profit before tax|Resultado antes de impuestos", IsSubtotal: true },
  { Label: "Income tax|Impuesto a la renta", AccountPrefixes: ["88"] },
  { Label: "Net profit|Resultado neto", IsSubtotal: true },
]

// The direct method sorts each cash line by the account on the other side of the cash: what
// buys or sells fixed assets (3x, and the payables for them, 465) is investing; shareholders,
// financial debt and equity (44, 45, 5x) are financing; everything else is operating —
// unclassified counter-accounts included, so the flows always add up to the cash that moved.
const CASH_FLOW_LAYOUT: IStatementSection[] = [
  { Label: "Operating activities|Actividades de operación", AccountPrefixes: [] },
  { Label: "Investing activities|Actividades de inversión", AccountPrefixes: ["3", "465"] },
  { Label: "Financing activities|Actividades de financiamiento", AccountPrefixes: ["44", "45", "5"] },
]

const CASH_ACCOUNT_PREFIX = "10"
const isCashAccount = (accountCode: number) => String(accountCode).startsWith(CASH_ACCOUNT_PREFIX)

// sectionOf is the section an account falls in: the one naming a prefix of its code, else the
// catch-all section (no prefixes), if the layout has one. Otherwise it is not in the statement.
function sectionOf(accountCode: number, sections: IStatementSection[]): IStatementSection | undefined {
  return sections.find(section => section.AccountPrefixes.some(prefix => String(accountCode).startsWith(prefix)))
    || sections.find(section => !section.AccountPrefixes.length)
}

// sectionRows lays the accounts out under their sections, in chart order, and adds the subtotals.
function sectionRows(
  amountByAccount: Map<number, number>, layout: (IStatementSection | IStatementSubtotal)[],
): IStatementRow[] {
  const sections = layout.filter((step): step is IStatementSection => !("IsSubtotal" in step))
  const rows: IStatementRow[] = []
  let runningResult = 0
  for (const step of layout) {
    if ("IsSubtotal" in step) {
      rows.push({ Kind: STATEMENT_ROW_SUBTOTAL, Label: step.Label, AccountCode: 0, Amount: runningResult })
      continue
    }
    const accountRows = [...amountByAccount]
      .filter(([accountCode, amount]) => amount && sectionOf(accountCode, sections) === step)
      .sort(([firstCode], [secondCode]) => String(firstCode).localeCompare(String(secondCode)))
      .map(([accountCode, amount]): IStatementRow => ({ Kind: STATEMENT_ROW_ACCOUNT, Label: "", AccountCode: accountCode, Amount: amount }))
    const sectionTotal = accountRows.reduce((total, row) => total + row.Amount, 0)
    runningResult += sectionTotal
    rows.push({ Kind: STATEMENT_ROW_SECTION, Label: step.Label, AccountCode: 0, Amount: sectionTotal }, ...accountRows)
  }
  return rows
}

// buildIncomeStatement sums each result account's Haber − Debe over the period: income is
// positive, expense negative. The unclassified lines (account 0) are left out; the page warns.
export function buildIncomeStatement(operations: IAccountingOperation[]): IStatementRow[] {
  const amountByAccount = new Map<number, number>()
  for (const operation of operations) {
    for (const line of operation.Lines) {
      if (!line.AccountCode) { continue }
      amountByAccount.set(line.AccountCode, (amountByAccount.get(line.AccountCode) || 0) + line.Credit - line.Debit)
    }
  }
  return sectionRows(amountByAccount, INCOME_STATEMENT_LAYOUT)
}

// One cash bank's balance before the period and at its end, in the cash bank's currency.
// Mirrors CashBankPeriodBalance in backend/accounting/accounting_cash_balances_api.go.
export interface ICashBankPeriodBalance {
  CashBankID: number
  OpeningBalance: number
  ClosingBalance: number
}

export interface IRestatedCashBalances {
  Opening: number
  Closing: number
  IsMissingRate: boolean   // a USD cash bank with a balance has no rate: its balance counts as 0
}

const CURRENCY_USD = 2

// restateCashBalances adds the cash banks' balances in PEN: a USD cash bank at the sell rate of
// the day before the period (opening) and of its last day (closing).
export function restateCashBalances(
  balances: ICashBankPeriodBalance[], cashBanks: IOperationCashBank[], openingRate: number, closingRate: number,
): IRestatedCashBalances {
  const currencyByCashBankID = new Map(cashBanks.map(cashBank => [cashBank.ID, cashBank.CurrencyType]))
  const restated: IRestatedCashBalances = { Opening: 0, Closing: 0, IsMissingRate: false }
  for (const balance of balances) {
    if (currencyByCashBankID.get(balance.CashBankID) !== CURRENCY_USD) {
      restated.Opening += balance.OpeningBalance
      restated.Closing += balance.ClosingBalance
      continue
    }
    restated.Opening += convertUSD(balance.OpeningBalance, openingRate)
    restated.Closing += convertUSD(balance.ClosingBalance, closingRate)
    if ((balance.OpeningBalance && !openingRate) || (balance.ClosingBalance && !closingRate)) {
      restated.IsMissingRate = true
    }
  }
  return restated
}

// buildCashFlow is the direct cash flow. An operation that moves cash posts every other line of
// its entry as a flow under that line's account (Haber − Debe: what came in through it). A
// transfer between two cash banks has no other line, so it moves nothing. With the balances,
// the gap between the real closing cash and opening + flows is shown on its own line: the
// exchange difference of the USD cash banks, plus whatever cash the view's entries do not hold
// (an operation missing its rate; in the official view, the undocumented sales and purchases).
export function buildCashFlow(
  operations: IAccountingOperation[], balances: IRestatedCashBalances | null,
): IStatementRow[] {
  const amountByAccount = new Map<number, number>()
  for (const operation of operations) {
    if (!operation.Lines.some(line => isCashAccount(line.AccountCode))) { continue }
    for (const line of operation.Lines) {
      if (isCashAccount(line.AccountCode)) { continue }
      amountByAccount.set(line.AccountCode, (amountByAccount.get(line.AccountCode) || 0) + line.Credit - line.Debit)
    }
  }

  const flowRows = sectionRows(amountByAccount, [
    ...CASH_FLOW_LAYOUT, { Label: "Net change in cash|Variación neta del efectivo", IsSubtotal: true },
  ])
  if (!balances) { return flowRows }

  const netChange = flowRows[flowRows.length - 1].Amount
  return [
    { Kind: STATEMENT_ROW_BALANCE, Label: "Opening cash|Saldo inicial de efectivo", AccountCode: 0, Amount: balances.Opening },
    ...flowRows,
    {
      Kind: STATEMENT_ROW_SUBTOTAL, Label: "Exchange difference and unreconciled|Diferencia de cambio y no conciliado",
      AccountCode: 0, Amount: balances.Closing - balances.Opening - netChange,
    },
    { Kind: STATEMENT_ROW_BALANCE, Label: "Closing cash|Saldo final de efectivo", AccountCode: 0, Amount: balances.Closing },
  ]
}

// The label an account row shows.
export const accountRowName = (accountCode: number): string =>
  accountCode ? findPCGEAccount(accountCode)?.name || "" : "Unclassified|Sin clasificar"
