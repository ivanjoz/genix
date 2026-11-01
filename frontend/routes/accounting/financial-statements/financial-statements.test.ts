import { describe, expect, it } from 'vitest'
import type { IAccountingOperation, IJournalLine } from '../accounts-consolidation/accounts-consolidation'
import {
  buildCashFlow, buildIncomeStatement, operationsOfView, restateCashBalances,
  STATEMENT_ROW_ACCOUNT, VIEW_OFFICIAL, type IStatementRow,
} from './financial-statements'

// line posts on the side its sign says: positive is Debe, negative is Haber.
const line = (accountCode: number, signedAmount: number): IJournalLine => ({
  AccountCode: accountCode, Debit: Math.max(0, signedAmount), Credit: Math.max(0, -signedAmount), IsNature: false,
})
const operation = (lines: IJournalLine[], inOfficialBook = true): IAccountingOperation => ({
  Key: "", Kind: 0, SourceID: 0, SourceType: 0, Date: 0, Description: "", DocumentLabel: "",
  InOfficialBook: inOfficialBook, Amount: 0, Lines: lines, Status: 1, Problem: "",
})
const amountOf = (rows: IStatementRow[], label: string) => rows.find(row => row.Label === label)?.Amount
const accountAmountOf = (rows: IStatementRow[], accountCode: number) =>
  rows.find(row => row.Kind === STATEMENT_ROW_ACCOUNT && row.AccountCode === accountCode)?.Amount

const sale = operation([line(1212, 11800), line(7012, -10000), line(40111, -1800)])
const costOfSale = operation([line(6912, 6000), line(2011, -6000)])
// A purchase received into stock nets out on 60 + 61.
const purchase = operation([line(6011, 6000), line(40111, 1080), line(4212, -7080)])
const reception = operation([line(2011, 6000), line(6111, -6000)])
const rent = operation([line(6352, 1500), line(4699, -1500)], false)
const collection = operation([line(101, 11800), line(1212, -11800)])
const supplierPayment = operation([line(1041, -7080), line(4212, 7080)])
const assetPayment = operation([line(1041, -20000), line(4654, 20000)])
const transfer = operation([line(1041, 5000), line(101, -5000)])
const unclassifiedWithdrawal = operation([line(101, -300), line(0, 300)])

describe('buildIncomeStatement', () => {
  const rows = buildIncomeStatement([sale, costOfSale, purchase, reception, rent, unclassifiedWithdrawal])

  it('runs revenue, cost and expenses down to the net result', () => {
    expect(amountOf(rows, "Net sales|Ventas netas")).toBe(10000)
    expect(amountOf(rows, "Cost of sales|Costo de ventas")).toBe(-6000)
    expect(amountOf(rows, "Gross profit|Utilidad bruta")).toBe(4000)
    expect(amountOf(rows, "Operating expenses|Gastos operativos")).toBe(-1500)
    expect(amountOf(rows, "Net profit|Resultado neto")).toBe(2500)
  })

  it('lists the result accounts under their sections in chart order, without the unclassified lines', () => {
    expect(rows.filter(row => row.Kind === STATEMENT_ROW_ACCOUNT).map(row => row.AccountCode))
      .toEqual([7012, 6011, 6111, 6912, 6352])
    // The received purchase still shows on 6011 and 6111, netting out within the cost of sales.
    expect(accountAmountOf(rows, 6011)).toBe(-6000)
    expect(accountAmountOf(rows, 6111)).toBe(6000)
  })
})

describe('buildCashFlow', () => {
  const operations = [sale, collection, supplierPayment, assetPayment, transfer, unclassifiedWithdrawal]

  it('sorts each cash line by its counter-account, and a transfer moves nothing', () => {
    const rows = buildCashFlow(operations, null)
    expect(amountOf(rows, "Operating activities|Actividades de operación")).toBe(11800 - 7080 - 300)
    expect(accountAmountOf(rows, 0)).toBe(-300)
    expect(amountOf(rows, "Investing activities|Actividades de inversión")).toBe(-20000)
    expect(amountOf(rows, "Financing activities|Actividades de financiamiento")).toBe(0)
    expect(amountOf(rows, "Net change in cash|Variación neta del efectivo")).toBe(11800 - 7080 - 20000 - 300)
    expect(rows.some(row => row.AccountCode === 101 || row.AccountCode === 1041)).toBe(false)
  })

  it('shows the gap between the real closing cash and opening + flows', () => {
    const netChange = 11800 - 7080 - 20000 - 300
    const rows = buildCashFlow(operations, { Opening: 50000, Closing: 50000 + netChange + 120, IsMissingRate: false })
    expect(rows[0].Amount).toBe(50000)
    expect(amountOf(rows, "Exchange difference and unreconciled|Diferencia de cambio y no conciliado")).toBe(120)
    expect(rows[rows.length - 1].Amount).toBe(50000 + netChange + 120)
  })
})

describe('restateCashBalances', () => {
  const cashBanks = [
    { ID: 1, Name: "Caja", Type: 1, CurrencyType: 1 },
    { ID: 2, Name: "Banco USD", Type: 2, CurrencyType: 2 },
  ]

  it('restates a dollar cash bank at the opening and the closing rate', () => {
    const restated = restateCashBalances([
      { CashBankID: 1, OpeningBalance: 1000, ClosingBalance: 2000 },
      { CashBankID: 2, OpeningBalance: 100, ClosingBalance: 100 },
    ], cashBanks, 3700, 3800)
    expect(restated).toEqual({ Opening: 1000 + 370, Closing: 2000 + 380, IsMissingRate: false })
  })

  it('flags a dollar balance with no rate', () => {
    expect(restateCashBalances([{ CashBankID: 2, OpeningBalance: 100, ClosingBalance: 0 }], cashBanks, 0, 3800)
      .IsMissingRate).toBe(true)
  })
})

describe('operationsOfView', () => {
  it('keeps only the official Diario entries in the official view', () => {
    expect(operationsOfView([sale, rent], VIEW_OFFICIAL)).toEqual([sale])
  })
})
