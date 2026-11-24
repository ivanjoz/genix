import { describe, expect, it } from 'vitest'
import type { FrameFileRows } from '@genix/ui/cache'
import { igvOfGross, TAX_GROUP_DETRACTION, TAX_GROUP_FREE, TAX_GROUP_RETENTION } from '#core/tax-groups.ts'
import { LANE_DETRACTIONS } from './cash-flow.ts'
import {
  cashDocumentRowsOf, clientCreditDaysWarning, closedPeriodTaxes, closedTaxCommitments, documentsOfItems, documentWarnings,
  drilldownDatesByKind, dueOffsetOf, knownDocuments, NO_DUE_DATE, oldPayableCommitments, rowAmountLocal,
  salesTaxRatioOf, scanWarningRecords, sellRateLookup, taxPeriodBases, type ICashDocumentRow, type ITaxPeriodBase,
} from './cash-flow-documents.ts'

// UnixDay 20026 is 2024-10-30.
const TODAY = 20026
const ONE_TO_ONE = () => 1000

const row = (overrides: Partial<ICashDocumentRow>): ICashDocumentRow =>
  ({ date: TODAY - 5, siteID: 1, kind: 1, taxGroup: 1, documentOffset: 0, dueOffset: 0, amount: 1000, ...overrides })

describe('cashDocumentRowsOf', () => {
  it('flattens the files: Keys [Date, SiteID], Rows [Kind, TaxGroup, DocumentOffset, DueOffset], Sums [Amount]', () => {
    const file = { keys: [TODAY, 2], rows: [[1, 10], [1, 6], [0, 130], [30, NO_DUE_DATE]], sums: [[500, 700]] } as unknown as FrameFileRows
    expect(cashDocumentRowsOf([file])).toEqual([
      { date: TODAY, siteID: 2, kind: 1, taxGroup: 1, documentOffset: 0, dueOffset: 30, amount: 500 },
      { date: TODAY, siteID: 2, kind: 10, taxGroup: 6, documentOffset: 130, dueOffset: NO_DUE_DATE, amount: 700 },
    ])
  })
})

describe('rates and amounts', () => {
  it('takes the last SUNAT rate on or before the day, else the fallback', () => {
    const rateOf = sellRateLookup([{ unixDay: TODAY - 2, sell: 3.75 }, { unixDay: TODAY - 10, sell: 3.7 }], 3600)
    expect([rateOf(TODAY), rateOf(TODAY - 5), rateOf(TODAY - 20)]).toEqual([3750, 3700, 3600])
  })
  it('restates dollars at the rate of the comprobante\'s day, Date + DocumentOffset − 128', () => {
    const days: number[] = []
    const amount = rowAmountLocal(row({ documentOffset: 131, amount: 1000 }), (day) => { days.push(day); return 3500 })
    expect([amount, days[0]]).toEqual([3500, TODAY - 5 + 3])
    expect(rowAmountLocal(row({ amount: 1000 }), () => 3500)).toBe(1000)
  })
  it('derives the IGV from the tax group', () => {
    expect([igvOfGross(1, 11800), igvOfGross(2, 11000), igvOfGross(4, 11800)]).toEqual([1800, 1000, 0])
  })
})

describe('knownDocuments', () => {
  it('dates open rows on their due date, today when past, and leaves stale receivables out', () => {
    const known = knownDocuments([
      row({ dueOffset: 10 }), row({ taxGroup: 4, dueOffset: 10, amount: 500 }), // one commitment: due in 5 days
      row({ kind: 2, amount: 9000 }), // settled: nothing to move
      row({ dueOffset: NO_DUE_DATE, amount: 300 }), // no due date: its own date, past → today
      row({ date: TODAY - 200, dueOffset: 0, amount: 70 }), // stale
      row({ kind: 20, dueOffset: 2, amount: 400 }), // a purchase order overdue
      row({ kind: 30, dueOffset: 30, amount: 100, documentOffset: 128 }), // an asset in dollars
    ], TODAY, ONE_TO_ONE)
    const summary = known.commitments.map((commitment) => [commitment.Stream, commitment.Date - TODAY, commitment.Amount, commitment.CurrencyType])
    expect(summary).toEqual([[1, 5, 1500, 1], [1, 0, 300, 1], [2, 0, -400, 1], [4, 25, -100, 2]])
    expect([...known.warningRecords.keys()]).toEqual(['sale-without-due-date', 'receivable-stale', 'payable-overdue'])
  })
  it('collects an open detracción into the detracciones lane, and moves no cash for the other memos', () => {
    const known = knownDocuments([
      row({ dueOffset: 10, amount: 11800 }),
      row({ dueOffset: 10, taxGroup: TAX_GROUP_DETRACTION, amount: 1416 }),
      row({ kind: 2, taxGroup: TAX_GROUP_RETENTION, amount: 300 }),
    ], TODAY, ONE_TO_ONE)
    expect(known.commitments.map((commitment) => [commitment.SiteID, commitment.Amount])).toEqual([[1, 10384], [LANE_DETRACTIONS, 1416]])
  })
  it('owes the old payables today', () => {
    expect(oldPayableCommitments({ Local: 500, Dollars: 0 }, TODAY).map((commitment) => [commitment.Date, commitment.Amount]))
      .toEqual([[TODAY, -500]])
  })
})

describe('taxes', () => {
  // The closed-period cases the backend's ClosedPeriodTaxes was tested with.
  const base = (period: number, values: Partial<ITaxPeriodBase>): ITaxPeriodBase =>
    ({ period, dueDate: TODAY - 10, salesTax: 0, salesNet: 0, purchaseTaxCredit: 0, retentionCredit: 0, paidIgv: 0, paidIncome: 0, ...values })
  const bases = [
    base(2608, { salesTax: 1000, salesNet: 100000, purchaseTaxCredit: 1500 }),
    base(2609, { salesTax: 2000, salesNet: 200000, purchaseTaxCredit: 800, paidIncome: 1000 }),
    base(2610, { salesTax: 3000, paidIgv: 3000 }),
  ]
  it('computes the closed periods with the saldo a favor carried', () => {
    const { dues, creditCarried } = closedPeriodTaxes(bases, { ChargesIgv: true, IncomeTaxRate: 150, NrusQuota: 0 })
    expect(dues.map((due) => due.igv)).toEqual([0, 700, 0])
    expect(creditCarried).toBe(0)
    expect(closedPeriodTaxes(bases.slice(0, 1), { ChargesIgv: true, IncomeTaxRate: 0, NrusQuota: 0 }).creditCarried).toBe(500)
    expect(dues.map((due) => due.incomeTax).slice(0, 2)).toEqual([1500, 2000])
    expect(Math.round(salesTaxRatioOf(bases) * 10000)).toBe(196)
    const nrus = closedPeriodTaxes(bases.slice(0, 1), { ChargesIgv: false, IncomeTaxRate: 0, NrusQuota: 2000 }).dues[0]
    expect([nrus.nrus, nrus.igv, nrus.incomeTax]).toEqual([2000, 0, 0])
  })
  it('owes a past due period today and flags it unpaid', () => {
    const closed = closedTaxCommitments([{ period: 2609, dueDate: TODAY - 3, igv: 700, incomeTax: 0, nrus: 0 }], TODAY)
    expect(closed.commitments.map((commitment) => [commitment.Stream, commitment.Date, commitment.Amount])).toEqual([[10, TODAY, -700]])
    expect([closed.unpaid.length, closed.unpaidAmount]).toEqual([1, 700])
  })
  it('sums each period\'s sales IGV and crédito fiscal from the rows dated in it', () => {
    const outlook = { Rules: { ChargesIgv: true, IncomeTaxRate: 0, NrusQuota: 0 }, Periods: [{ Period: 2410, DueDate: TODAY + 20, PaidIgvLocal: 5, PaidIncomeLocal: 0 }] }
    const [october] = taxPeriodBases([
      row({ amount: 11800 }), row({ kind: 2, taxGroup: 4, amount: 1000 }), row({ kind: 21, amount: 2360 }),
      row({ date: TODAY - 60, amount: 99999 }), // September: not in the outlook
    ], outlook, ONE_TO_ONE)
    expect([october.salesTax, october.salesNet, october.purchaseTaxCredit, october.paidIgv]).toEqual([1800, 11000, 360, 5])
  })
  it('owes the IGV of a free line without counting it as a sale', () => {
    const outlook = { Rules: { ChargesIgv: true, IncomeTaxRate: 0, NrusQuota: 0 }, Periods: [{ Period: 2410, DueDate: TODAY + 20, PaidIgvLocal: 0, PaidIncomeLocal: 0 }] }
    const [october] = taxPeriodBases([row({ kind: 2, taxGroup: TAX_GROUP_FREE, amount: 2360 })], outlook, ONE_TO_ONE)
    expect([october.salesTax, october.salesNet]).toEqual([360, 0])
  })
  it('credits the retenciones against the IGV and leaves the detracción out of the sales', () => {
    const outlook = { Rules: { ChargesIgv: true, IncomeTaxRate: 0, NrusQuota: 0 }, Periods: [{ Period: 2410, DueDate: TODAY + 20, PaidIgvLocal: 0, PaidIncomeLocal: 0 }] }
    const [october] = taxPeriodBases([
      row({ amount: 11800 }), row({ kind: 2, taxGroup: TAX_GROUP_RETENTION, amount: 354 }), row({ taxGroup: TAX_GROUP_DETRACTION, amount: 1416 }),
    ], outlook, ONE_TO_ONE)
    expect([october.salesTax, october.salesNet, october.retentionCredit]).toEqual([1800, 10000, 354])
    const { dues } = closedPeriodTaxes([october], { ChargesIgv: true, IncomeTaxRate: 0, NrusQuota: 0 })
    expect(dues[0].igv).toBe(1800 - 354)
  })
})

describe('warnings', () => {
  it('scans the recent periods for documents without a site or a comprobante', () => {
    const records = scanWarningRecords([
      row({ siteID: 0 }), row({ kind: 11, siteID: 0, taxGroup: 6 }), row({ date: TODAY - 200, siteID: 0 }),
    ], 2410, ONE_TO_ONE)
    expect([...records.keys()]).toEqual(['sale-without-site', 'expense-without-site', 'purchase-without-document'])
  })
  it('counts one record per document day and site, the amounts summed', () => {
    const warnings = documentWarnings(new Map([['payable-overdue', [
      { date: TODAY - 2, siteID: 1, amountLocal: -300 }, { date: TODAY - 2, siteID: 1, amountLocal: -200 }, { date: TODAY - 1, siteID: 2, amountLocal: -50 },
    ]]]), (date, siteID) => `${date}·${siteID}`)
    expect(warnings).toEqual([{ Code: 'payable-overdue', AmountAtStakePEN: 550, RecordCount: 2,
      Records: [{ ID: TODAY - 2, Label: `${TODAY - 2}·1` }, { ID: TODAY - 1, Label: `${TODAY - 1}·2` }] }])
  })
  it('flags the clients that owe on cash terms', () => {
    const debtFiles = [{ keys: [0], rows: [[5, 6, 7]], sums: [[1000, 2000, 0]] }] as unknown as FrameFileRows[]
    const clients = new Map([[5, { Name: 'Ana', CreditDays: 0 }], [6, { Name: 'Luis', CreditDays: 30 }], [7, { Name: 'Eva', CreditDays: 0 }]])
    expect(clientCreditDaysWarning(debtFiles, clients)).toEqual([
      { Code: 'client-credit-days-missing', AmountAtStakePEN: 1000, Records: [{ ID: 5, Label: 'Ana' }], RecordCount: 1 },
    ])
  })
})

describe('drill-down', () => {
  const items = [
    { kind: 1, documentDate: TODAY - 5, dueDate: TODAY + 5 },
    { kind: 1, documentDate: TODAY - 3, dueDate: 0 },
    { kind: 20, documentDate: TODAY - 3, dueDate: TODAY },
    { dueDate: 0 }, // a payroll item: not a document
  ]
  it('asks for each Kind\'s document dates', () => {
    expect([...drilldownDatesByKind(items)]).toEqual([[1, [TODAY - 5, TODAY - 3]], [20, [TODAY - 3]]])
  })
  it('keeps the documents whose day and due offset match an item', () => {
    const documents = [
      { ID: 1, Label: 'Venta 1', Date: TODAY - 5, DueDate: TODAY + 5, Amount: 100, CurrencyType: 1 },
      { ID: 2, Label: 'Venta 2', Date: TODAY - 5, DueDate: TODAY + 40, Amount: 100, CurrencyType: 1 }, // another cell
      { ID: 3, Label: 'Venta 3', Date: TODAY - 3, DueDate: 0, Amount: 100, CurrencyType: 1 },
    ]
    expect(documentsOfItems(items, 1, documents).map((document) => document.ID)).toEqual([1, 3])
    expect(dueOffsetOf(TODAY, TODAY - 4)).toBe(0)
  })
})
