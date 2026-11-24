import { describe, expect, it } from 'vitest'
import type { ICashFlowCommitment, ICashFlowEntry } from '#services/finance/cash-flow.svelte.ts'
import {
  buildCashFlowGrid, CASH_FLOW_LAYER_PLANNED, CASH_FLOW_LAYER_SCHEDULED, cashBanksByLane, cashFlowLanes, firstNegativeDay,
  LANE_COMPANY, LANE_DETRACTIONS, LANE_TOTAL, monthlyGrid, payTaxesFromDetractions, PLANNED_STREAM_ID, toPEN,
} from './cash-flow.ts'
import { pageWarnings, rankWarnings } from './cash-flow-warnings.ts'

const TODAY = 20000

const commitment = (overrides: Partial<ICashFlowCommitment>): ICashFlowCommitment => ({
  Stream: 1, Layer: 1, SiteID: 1, SourceID: 1, Label: 'x', Date: TODAY, DueDate: TODAY, DocumentDate: 0, Amount: 0, CurrencyType: 1,
  ...overrides,
})

const entry = (overrides: Partial<ICashFlowEntry>): ICashFlowEntry => ({
  ID: 1, Date: TODAY, SiteID: 0, Amount: 0, CurrencyType: 1, Kind: 5, Name: 'Plan', ss: 1, upd: 0, ...overrides,
})

describe('buildCashFlowGrid', () => {
  const commitments = [
    commitment({ Stream: 1, SiteID: 1, Date: TODAY, Amount: 10000 }),
    commitment({ Stream: 3, SiteID: 2, Date: TODAY + 1, Amount: -4000 }),
    commitment({ Stream: 9, SiteID: 0, Date: TODAY + 2, Amount: -100, CurrencyType: 2, Layer: 2 }),
    commitment({ Stream: 1, SiteID: 1, Date: TODAY + 10, Amount: 999 }), // past the horizon
  ]
  const cashBanks = [
    { SiteID: 1, CurrencyType: 1, CurrentAmount: 5000 },
    { SiteID: 0, CurrencyType: 2, CurrentAmount: 1000 },
  ]
  const entries = [entry({ Date: TODAY + 1, Amount: -2000 }), entry({ ID: 2, Date: TODAY - 1, Amount: -7000 })]

  it('rolls the total lane from today\'s cash, dollars at the sell rate', () => {
    const grid = buildCashFlowGrid({ commitments, entries, cashBanks, lane: LANE_TOTAL, today: TODAY, horizonDays: 3, usdSellRate: 3700 })
    // Opening: 5000 + 1000 USD cents × 3.7.
    expect(grid.opening).toBe(5000 + 3700)
    expect(grid.rows.map((row) => row.streamID)).toEqual([1, 3, 9, PLANNED_STREAM_ID])
    // Day 0 +10000; day 1 −4000 −2000 (the past entry does not count); day 2 −370.
    expect(grid.closing).toEqual([18700, 12700, 12330])
    expect(grid.rows[2].cells[2].items[0].layer).toBe(CASH_FLOW_LAYER_SCHEDULED)
    expect(grid.rows[3].cells[1].items[0].layer).toBe(CASH_FLOW_LAYER_PLANNED)
  })

  it('folds the days by month, closing at each month\'s last day', () => {
    // UnixDay 20026 is 2024-10-30: two days of October, then November.
    const daily = buildCashFlowGrid({ commitments: [commitment({ Date: 20026, Amount: 100 }), commitment({ Date: 20027, Amount: 50 }),
      commitment({ Date: 20028, Amount: -30 })], entries: [], cashBanks: [], lane: LANE_TOTAL, today: 20026, horizonDays: 4, usdSellRate: 1000 })
    const monthly = monthlyGrid(daily)
    expect(monthly.days).toEqual([20026, 20028])
    expect(monthly.rows[0].cells.map((cell) => cell.amount)).toEqual([150, -30])
    expect(monthly.closing).toEqual([150, 120])
  })

  it('keeps one site, or the company lane, apart', () => {
    const siteGrid = buildCashFlowGrid({ commitments, entries, cashBanks, lane: 1, today: TODAY, horizonDays: 3, usdSellRate: 3700 })
    expect(siteGrid.opening).toBe(5000)
    expect(siteGrid.closing).toEqual([15000, 15000, 15000])
    const companyGrid = buildCashFlowGrid({ commitments, entries, cashBanks, lane: LANE_COMPANY, today: TODAY, horizonDays: 3, usdSellRate: 3700 })
    expect(companyGrid.closing).toEqual([3700, 1700, 1330])
  })

  it('finds the first day below zero', () => {
    const grid = buildCashFlowGrid({ commitments, entries: [entry({ Date: TODAY + 2, Amount: -20000 })], cashBanks,
      lane: LANE_TOTAL, today: TODAY, horizonDays: 3, usdSellRate: 3700 })
    expect(firstNegativeDay(grid)).toBe(TODAY + 2)
  })
})

describe('helpers', () => {
  it('restates dollars and lists the lanes', () => {
    expect(toPEN(1000, 2, 3750)).toBe(3750)
    expect(toPEN(1000, 1, 3750)).toBe(1000)
    const sites = [{ ID: 1, Name: 'Centro', ss: 1 }, { ID: 2, Name: 'Cerrada', ss: 0 }]
    expect(cashFlowLanes(sites, false).map((lane) => lane.id)).toEqual([LANE_TOTAL, 1, LANE_COMPANY])
    expect(cashFlowLanes(sites, true).map((lane) => lane.id)).toEqual([LANE_TOTAL, 1, LANE_COMPANY, LANE_DETRACTIONS])
  })
})

describe('detracciones account', () => {
  it('keeps its balance out of the company total', () => {
    const cashBanks = cashBanksByLane([
      { Type: 2, SiteID: 0, CurrencyType: 1, CurrentAmount: 5000 },
      { Type: 3, SiteID: 0, CurrencyType: 1, CurrentAmount: 800 },
    ])
    const commitments = [commitment({ SiteID: LANE_DETRACTIONS, Amount: 300 })]
    const total = buildCashFlowGrid({ commitments, entries: [], cashBanks, lane: LANE_TOTAL, today: TODAY, horizonDays: 1, usdSellRate: 1 })
    expect(total.closing).toEqual([5000])
    const detractions = buildCashFlowGrid({ commitments, entries: [], cashBanks, lane: LANE_DETRACTIONS, today: TODAY, horizonDays: 1, usdSellRate: 1 })
    expect(detractions.closing).toEqual([1100])
  })

  it('pays the taxes first, as far as its balance goes', () => {
    const igv = commitment({ Stream: 10, SiteID: 0, Date: TODAY + 2, Amount: -1000, Label: 'IGV' })
    const renta = commitment({ Stream: 11, SiteID: 0, Date: TODAY + 3, Amount: -500, Label: 'Renta' })
    const supplier = commitment({ Stream: 3, SiteID: 0, Date: TODAY + 2, Amount: -700 })
    const deposit = commitment({ SiteID: LANE_DETRACTIONS, Date: TODAY + 1, Amount: 400 })
    const paid = payTaxesFromDetractions([igv, renta, supplier, deposit], 800)
    // The IGV takes the whole balance (800 + 400): 1000 of it; the renta the 200 left.
    const amountsOf = (siteID: number, stream: number) =>
      paid.filter((row) => row.SiteID === siteID && row.Stream === stream).map((row) => row.Amount)
    expect(amountsOf(LANE_DETRACTIONS, 10)).toEqual([-1000])
    expect(amountsOf(0, 10)).toEqual([])
    expect(amountsOf(0, 11)).toEqual([-300])
    expect(amountsOf(LANE_DETRACTIONS, 11)).toEqual([-200])
    expect(amountsOf(0, 3)).toEqual([-700])
  })
})

describe('warnings', () => {
  it('raises the page checks and ranks them with the server\'s', () => {
    const page = pageWarnings({
      cashBanks: [
        { ID: 1, Name: 'Caja', SiteID: 1, CurrencyType: 1, CurrentAmount: -500, ReconciliationDate: TODAY - 2, ss: 1 },
        { ID: 2, Name: 'Banco', SiteID: 0, CurrencyType: 1, CurrentAmount: 9000, ReconciliationDate: TODAY - 31, ss: 1 },
      ],
      entries: [entry({ Date: TODAY - 3, Amount: -100 })],
      usesDollars: true, usdSellRate: 0, today: TODAY,
    })
    expect(page.map((warning) => warning.Code)).toEqual(['cash-bank-negative', 'cash-bank-not-reconciled', 'cash-bank-without-site', 'exchange-rate-stale', 'planned-entry-past'])
    const ranked = rankWarnings([...page, { Code: 'loan-installment-overdue', AmountAtStakePEN: 9000, Records: [], RecordCount: 1 }])
    expect(ranked.map((warning) => warning.severity)).toEqual(['high', 'high', 'medium', 'medium', 'medium', 'info'])
    // Among the high ones, the larger amount at stake first.
    expect(ranked[0].Code).toBe('loan-installment-overdue')
  })
})
