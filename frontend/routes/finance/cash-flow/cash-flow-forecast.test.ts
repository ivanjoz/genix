import { describe, expect, it } from 'vitest'
import type { FrameFileRows } from '@genix/ui/cache'
import type { ICashFlowCommitment } from '#services/finance/cash-flow.svelte.ts'
import type { ICashDocumentRow, ITaxPeriodBase, ITaxRules } from './cash-flow-documents.ts'
import { CASH_FLOW_STREAM_FORECAST_COLLECTIONS, CASH_FLOW_STREAM_FORECAST_OUTFLOWS } from './cash-flow.ts'
import {
  buildCashFlowSeries, buildDelayCurves, changeCheckSpan, DELAY_CURVE_MIN_ENTRIES, ENGINE_STATE_VERSION, forecastFlows,
  forecastTaxes, forecastWarnings, hashSeriesDays, planEngineRun, shiftOverdueReceivables, type IDelayCurves,
} from './cash-flow-forecast.ts'

// UnixDay 20026 is 2024-10-30.
const TODAY = 20026

const frameFile = (keys: number[], rows: number[][], sums: number[][]) => ({ keys, rows, sums }) as unknown as FrameFileRows

describe('buildCashFlowSeries', () => {
  const window = { startDay: TODAY - 3, lastDay: TODAY - 1 }
  const row = (date: number, siteID: number, kind: number, taxGroup: number, amount: number, documentOffset = 0): ICashDocumentRow =>
    ({ date, siteID, kind, taxGroup, documentOffset, dueOffset: 0, amount })
  it('sums sales per site and spending per site and kind, in soles', () => {
    const rows = [
      row(TODAY - 3, 1, 1, 1, 1000), row(TODAY - 3, 1, 2, 1, 500), row(TODAY - 1, 2, 2, 1, 2000),
      row(TODAY - 2, 1, 10, 1, 11800), row(TODAY - 2, 1, 21, 4, 5000),
      // A schedule's period is projected by the schedules, not learned.
      row(TODAY - 2, 1, 12, 1, 99900),
      // 10 dollars at 3.5.
      row(TODAY - 2, 1, 20, 4, 1000, 128),
    ]
    const build = buildCashFlowSeries(rows, window, () => 3500)
    expect(build.series.map((series) => series.key)).toEqual(['1_0', '1_1010', '1_1020', '2_0'])
    expect(Array.from(build.series[0].values)).toEqual([15, 0, 0])
    expect(Array.from(build.series[2].values)).toEqual([0, 85, 0])
    expect(build.firstSaleDay).toBe(TODAY - 3)
    expect(build.firstSaleDayBySite.get(2)).toBe(TODAY - 1)
    expect(build.purchaseTaxRatio).toBeCloseTo(1800 / (11800 + 5000 + 3500))
  })
})

describe('planEngineRun', () => {
  const window = { startDay: TODAY - 200, lastDay: TODAY - 1 }
  const values = new Float64Array(200).fill(3)
  const series = [{ key: '1_0', siteID: 1, productID: 0, categoryID: 1, values }]
  const storedLastDay = TODAY - 2
  const { fromIndex, toIndex } = changeCheckSpan(storedLastDay, window.startDay)
  const stored = {
    version: ENGINE_STATE_VERSION, lastDay: storedLastDay, keys: ['1_0'], recentHashes: [hashSeriesDays(values, fromIndex, toIndex)],
    state: new Uint8Array(),
  }

  it('fits from scratch without a usable state', () => {
    expect(planEngineRun(undefined, series, window)).toEqual({ mode: 'full' })
    expect(planEngineRun({ ...stored, version: ENGINE_STATE_VERSION + 1 }, series, window)).toEqual({ mode: 'full' })
    expect(planEngineRun({ ...stored, lastDay: TODAY }, series, window)).toEqual({ mode: 'full' })
  })

  it('updates the state, fitting again only new or corrected series', () => {
    expect(planEngineRun(stored, series, window)).toEqual({ mode: 'incremental', refitKeys: [] })
    const corrected = new Float64Array(values)
    corrected[toIndex] = 4
    const newSeries = { key: '2_0', siteID: 2, productID: 0, categoryID: 1, values }
    expect(planEngineRun(stored, [{ ...series[0], values: corrected }, newSeries], window))
      .toEqual({ mode: 'incremental', refitKeys: ['1_0', '2_0'] })
  })
})

describe('delay curves', () => {
  const cashBanks = [{ ID: 1, SiteID: 1, CurrencyType: 1 }, { ID: 2, SiteID: 2, CurrencyType: 1 }]
  // Site 1: enough collections, half on the day, half 10 days later. Site 2: a single one.
  const delayFiles = [
    ...Array.from({ length: DELAY_CURVE_MIN_ENTRIES / 2 }, (_, index) => frameFile([TODAY - 1 - index, 1], [[8, 8], [0, 10]], [[100, 100]])),
    frameFile([TODAY - 1, 2], [[8], [5]], [[300]]),
  ]
  const curves = buildDelayCurves(delayFiles, cashBanks, 1000, TODAY - 180)

  it('uses the site curve when it has enough payments, the company one otherwise', () => {
    const siteCurve = curves.curveOf('collection', 1)
    expect(siteCurve.source).toBe('site')
    expect(siteCurve.shares[0]).toBeCloseTo(0.5)
    expect(siteCurve.shares[10]).toBeCloseTo(0.5)
    expect(curves.curveOf('collection', 2).source).toBe('company')
    expect(curves.curveOf('payment', 1)).toEqual({ shares: [1], source: 'none' })
  })

  it('moves an overdue receivable to the median of the curve past its age', () => {
    const lateCurves: IDelayCurves = { curveOf: () => ({ shares: [0.4, 0, 0, 0, 0, 0.2, 0, 0, 0.4], source: 'site' }) }
    const overdue: ICashFlowCommitment = {
      Stream: 1, Layer: 1, SiteID: 1, SourceID: 9, Label: 'Venta 9', Date: TODAY, DueDate: TODAY - 1, DocumentDate: TODAY - 3,
      Amount: 500, CurrencyType: 1,
    }
    // Age 3: of what is left (0.2 at 5, 0.4 at 8), the median falls on day 8.
    expect(shiftOverdueReceivables([overdue], lateCurves, TODAY)[0].Date).toBe(TODAY - 3 + 8)
    // Past the end of the curve: it stays today.
    expect(shiftOverdueReceivables([{ ...overdue, DocumentDate: TODAY - 20 }], lateCurves, TODAY)[0].Date).toBe(TODAY)
  })
})

describe('forecastFlows', () => {
  const curves: IDelayCurves = {
    curveOf: (direction) => direction === 'collection' ? { shares: [0.5, 0.5], source: 'site' } : { shares: [1], source: 'site' },
  }
  it('spreads tomorrow\'s forecast documents by the curve, never today\'s', () => {
    const flows = forecastFlows({ siteIDs: [1, 1], productIDs: [0, 1003], firstDay: TODAY, horizon: 3, values: [9, 10, 20, 7, 0, 5] },
      curves, TODAY, TODAY + 2)
    const collections = flows.commitments.filter((commitment) => commitment.Stream === CASH_FLOW_STREAM_FORECAST_COLLECTIONS)
    // Day 1: 1000 cents → 500 on day 1 and 500 on day 2. Day 2: 2000 → 1000 on day 2 (day 3 is past the end).
    expect(collections.map((commitment) => [commitment.Date - TODAY, commitment.Amount])).toEqual([[1, 500], [2, 1500]])
    const outflows = flows.commitments.filter((commitment) => commitment.Stream === CASH_FLOW_STREAM_FORECAST_OUTFLOWS)
    expect(outflows.map((commitment) => [commitment.Date - TODAY, commitment.Amount])).toEqual([[2, -500]])
    expect(flows.salesByDay.get(TODAY + 1)).toBe(1000)
  })
})

describe('forecastTaxes', () => {
  const base = (period: number, dueDate: number, salesTax: number, salesNet: number, purchaseTaxCredit: number): ITaxPeriodBase =>
    ({ period, dueDate, salesTax, salesNet, purchaseTaxCredit, retentionCredit: 0, paidIgv: 0, paidIncome: 0 })
  const openBases = [base(2410, TODAY + 20, 5000, 30000, 1000), base(2411, TODAY + 50, 0, 0, 0)]
  // A forecast sale of 11800 on 2024-10-31 and one on 2024-11-01; spending of 5900 in November.
  const rules: ITaxRules = { ChargesIgv: true, IncomeTaxRate: 150, NrusQuota: 0 }
  const salesByDay = new Map([[TODAY + 1, 11800], [TODAY + 2, 11800]])
  const spendingByDay = new Map([[TODAY + 2, 5900]])

  it('adds the forecast to what the open period recorded, carrying the saldo a favor', () => {
    const taxes = forecastTaxes({ rules, openBases, salesTaxRatio: 0.1525, creditCarried: 1000, salesByDay, spendingByDay, purchaseTaxRatio: 0.18 })
    const amountOf = (label: string) => -(taxes.find((tax) => tax.Label === label)?.Amount || 0)
    // October: 5000 + 1799.5 − 1000 − 1000 carried. November: 1799.5 − 1062.
    expect(amountOf('IGV 2410')).toBe(Math.round(5000 + 11800 * 0.1525 - 1000 - 1000))
    expect(amountOf('IGV 2411')).toBe(Math.round(11800 * 0.1525 - 5900 * 0.18))
    expect(amountOf('Renta 2410')).toBe(Math.round((30000 + 11800 * (1 - 0.1525)) * 0.015))
    expect(taxes.every((tax) => tax.SiteID === 0)).toBe(true)
  })

  it('charges only the cuota under Nuevo RUS', () => {
    const taxes = forecastTaxes({ rules: { ChargesIgv: false, IncomeTaxRate: 0, NrusQuota: 2000 }, openBases, salesTaxRatio: 0.1525,
      creditCarried: 1000, salesByDay, spendingByDay, purchaseTaxRatio: 0.18 })
    expect(taxes.map((tax) => [tax.Label, tax.Amount])).toEqual([['Nuevo RUS 2410', -2000], ['Nuevo RUS 2411', -2000]])
  })
})

describe('forecastWarnings', () => {
  it('warns about short histories and borrowed delay curves', () => {
    const curves: IDelayCurves = { curveOf: (_, siteID) => ({ shares: [1], source: siteID === 1 ? 'site' : 'company' }) }
    const warnings = forecastWarnings({
      window: { startDay: TODAY - 100, lastDay: TODAY - 1 },
      firstSaleDayBySite: new Map([[1, TODAY - 100], [2, TODAY - 30]]),
      series: [{ key: '1_0', siteID: 1, productID: 0, categoryID: 1, values: new Float64Array() },
        { key: '2_0', siteID: 2, productID: 0, categoryID: 1, values: new Float64Array() }],
      curves, siteName: (siteID) => `Sede ${siteID}`,
    })
    expect(warnings.map((warning) => [warning.Code, warning.Records?.map((record) => record.Label)])).toEqual([
      ['history-under-year', ['100 d']],
      ['history-short', ['Sede 2 · 30 d']],
      ['delay-curve-missing', ['Sede 2 · cobros']],
    ])
  })
})
