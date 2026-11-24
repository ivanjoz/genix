import type { FrameFileRows } from '@genix/ui/cache'
import { periodOfDay } from '#routes/hr/payroll/payroll.ts'
import type { ICashFlowCommitment, ICashFlowServerWarning } from '#services/finance/cash-flow.svelte.ts'
import { igvOfGross, isMemoTaxGroup } from '#core/tax-groups.ts'
import {
  isSaleKind, isScheduledExpenseKind, rowAmountLocal, spendingGroupOf, type ICashDocumentRow, type ITaxPeriodBase,
  type ITaxRules,
} from './cash-flow-documents.ts'
import {
  CASH_FLOW_LAYER_FORECAST, CASH_FLOW_STREAM_FORECAST_COLLECTIONS, CASH_FLOW_STREAM_FORECAST_OUTFLOWS,
  CASH_FLOW_STREAM_FORECAST_TAXES, toPEN,
} from './cash-flow.ts'

// The Forecast layer of the cash flow (plan/CASH_FLOW_FORECAST_PLAN.md §4, §6, §7.3): the documents
// not recorded yet. The forecast engine (Rust → WASM, run in cash-flow.worker.ts) learns each site's
// daily sales and unscheduled spending; the delay curves turn those documents into cash; the taxes of
// the open and future periods follow from them. Pure: no Svelte, no fetch, no engine.

/** Days of history the engine learns from: a year sees the yearly season. */
export const HISTORY_DAYS = 365
/** Days of payments the delay curves are measured over. */
export const DELAY_CURVE_DAYS = 180
/** Below this many (day, delay) entries a site's curve is too thin: it uses the company's. */
export const DELAY_CURVE_MIN_ENTRIES = 30
/** The frame clips every delay to this (backend CashDelayMaxDays). */
export const DELAY_MAX_DAYS = 120
/** The engine refuses fewer than 21 days of history; a few weeks more give it a shape to learn. */
export const MIN_ENGINE_HISTORY_DAYS = 28
/** A site that sells for less than this is forecast without a weekly or yearly shape of its own. */
export const HISTORY_SHORT_DAYS = 90
/**
 * The last days of history an incremental run compares with what the stored state learned: a series
 * that changed there (a late expense, an annulled sale) is fitted again. Older corrections are rare.
 */
export const CHANGE_CHECK_DAYS = 120
/** Bump when the series' meaning changes: a stored state of another version is fitted from scratch. */
export const ENGINE_STATE_VERSION = 2

// Series ids inside the engine: the sales of a site are product 0; its unscheduled spending of a
// kind is product 1000 + its Kind group (10 expenses, 20 purchase orders, 30 assets). Sales of every
// site share one category pool; each spending kind is its own pool across sites.
export const SALES_PRODUCT_ID = 0
const SALES_CATEGORY_ID = 1
const OUTFLOW_PRODUCT_BASE = 1000

// The cash-delay frame (backend finance/types CashDelayFrame): Keys [Date, CashBankID], then rows
// and sums in declaration order.
const KEY_DATE = 0
const KEY_CASH_BANK = 1
const DELAY_ROW_TYPE = 0
const DELAY_ROW_DAYS = 1
const DELAY_SUM_AMOUNT = 0
const CASH_MOVEMENT_SALE_COLLECTION = 8

export interface ICashFlowWindow {
  /** First day of history. */
  startDay: number
  /** Yesterday: today's documents are still coming in, and they are Known anyway. */
  lastDay: number
}

export const cashFlowWindowOf = (today: number): ICashFlowWindow => ({ startDay: today - HISTORY_DAYS, lastDay: today - 1 })

/** One site's sales or spending of a category as the engine reads it: index i is the day startDay + i, in soles. */
export interface ICashFlowSeries {
  key: string
  siteID: number
  productID: number
  categoryID: number
  values: Float64Array
}

export const seriesKeyOf = (siteID: number, productID: number) => `${siteID}_${productID}`
export const isSalesSeries = (productID: number) => productID === SALES_PRODUCT_ID

export interface ICashFlowSeriesBuild {
  series: ICashFlowSeries[]
  /** The company's first day with a sale in the window, or undefined without any. */
  firstSaleDay: number | undefined
  firstSaleDayBySite: Map<number, number>
  /** Crédito fiscal over gross amount of the spending in the window: the IGV a forecast purchase recovers. */
  purchaseTaxRatio: number
}

/**
 * buildCashFlowSeries sums the cash-documents rows into one sales series per site (open and settled
 * sales) and one spending series per site and kind (expenses, purchase orders, assets), over the
 * window, in soles. A schedule's periods are left out: the schedules project them. Sales of no site
 * (site 0) are a series of their own: their cash is the company lane's, as the Known receivables are.
 */
export function buildCashFlowSeries(rows: ICashDocumentRow[], window: ICashFlowWindow, sellRateOfDay: (day: number) => number): ICashFlowSeriesBuild {
  const dayCount = window.lastDay - window.startDay + 1
  const seriesByKey = new Map<string, ICashFlowSeries>()
  const valuesOf = (siteID: number, productID: number, categoryID: number) => {
    const key = seriesKeyOf(siteID, productID)
    let series = seriesByKey.get(key)
    if (!series) {
      series = { key, siteID, productID, categoryID, values: new Float64Array(dayCount) }
      seriesByKey.set(key, series)
    }
    return series.values
  }

  let firstSaleDay: number | undefined
  const firstSaleDayBySite = new Map<number, number>()
  let spendingCents = 0, spendingTaxCents = 0
  for (const row of rows) {
    const dayIndex = row.date - window.startDay
    // A memo is beside its document's total (a free line's value, a detracción, a retención): the
    // series count the totals only.
    if (dayIndex < 0 || dayIndex >= dayCount || isScheduledExpenseKind(row.kind) || isMemoTaxGroup(row.taxGroup)) continue
    const amountCents = rowAmountLocal(row, sellRateOfDay)
    if (isSaleKind(row.kind)) {
      if (amountCents <= 0) continue
      valuesOf(row.siteID, SALES_PRODUCT_ID, SALES_CATEGORY_ID)[dayIndex] += amountCents / 100
      if (firstSaleDay === undefined || row.date < firstSaleDay) firstSaleDay = row.date
      if (row.date < (firstSaleDayBySite.get(row.siteID) ?? Infinity)) firstSaleDayBySite.set(row.siteID, row.date)
      continue
    }
    const productID = OUTFLOW_PRODUCT_BASE + spendingGroupOf(row.kind)
    valuesOf(row.siteID, productID, productID)[dayIndex] += amountCents / 100
    spendingCents += amountCents
    spendingTaxCents += igvOfGross(row.taxGroup, amountCents)
  }

  const series = [...seriesByKey.values()].sort((left, right) => left.siteID - right.siteID || left.productID - right.productID)
  return { series, firstSaleDay, firstSaleDayBySite, purchaseTaxRatio: spendingCents > 0 ? spendingTaxCents / spendingCents : 0 }
}

/**
 * The window from the company's first sale: before it the business was not selling through Genix,
 * and months of zeros read as demand would flatten every shape (as in the sales forecast).
 */
export const clipWindowToFirstSale = (window: ICashFlowWindow, firstSaleDay: number): ICashFlowWindow =>
  ({ ...window, startDay: Math.max(window.startDay, firstSaleDay) })

/** The series cut to a later window start. */
export const clipSeries = (series: ICashFlowSeries[], window: ICashFlowWindow, clipped: ICashFlowWindow): ICashFlowSeries[] =>
  series.map((oneSeries) => ({ ...oneSeries, values: oneSeries.values.slice(clipped.startDay - window.startDay) }))

/** FNV-1a over a span of days (in cents), to tell whether the history a state learned has changed since. */
export function hashSeriesDays(values: Float64Array, fromIndex: number, toIndex: number): number {
  let hash = 2166136261
  for (let index = Math.max(0, fromIndex); index <= toIndex && index < values.length; index++) {
    hash = Math.imul(hash ^ Math.round(values[index] * 100), 16777619) >>> 0
  }
  return hash
}

/** The day span a stored state's change check covers, as indexes of a series starting on startDay. */
export const changeCheckSpan = (stateLastDay: number, startDay: number) =>
  ({ fromIndex: stateLastDay - CHANGE_CHECK_DAYS + 1 - startDay, toIndex: stateLastDay - startDay })

/** What the worker keeps in IndexedDB per company: the engine state and what it learned. */
export interface IStoredEngineState {
  version: number
  lastDay: number
  /** The state's series in its order (keys()), and the hash of each one's change-check span. */
  keys: string[]
  recentHashes: number[]
  state: Uint8Array
}

export type TEnginePlan = { mode: 'full' } | { mode: 'incremental', refitKeys: string[] }

/**
 * planEngineRun decides how much of the stored state is reused. It is fitted from scratch when there
 * is none, it is of another version, or it is too old to be updated (its last day before the
 * window's history, or after yesterday). Otherwise it is updated with the new days, after fitting
 * again the new series and those whose recent history changed since it learned it.
 */
export function planEngineRun(stored: IStoredEngineState | undefined, series: ICashFlowSeries[], window: ICashFlowWindow): TEnginePlan {
  if (!stored || stored.version !== ENGINE_STATE_VERSION || stored.lastDay > window.lastDay
    || stored.lastDay - window.startDay + 1 < MIN_ENGINE_HISTORY_DAYS) {
    return { mode: 'full' }
  }
  const storedPositionByKey = new Map(stored.keys.map((key, position) => [key, position]))
  const { fromIndex, toIndex } = changeCheckSpan(stored.lastDay, window.startDay)
  const refitKeys = series
    .filter((oneSeries) => {
      const position = storedPositionByKey.get(oneSeries.key)
      return position === undefined || hashSeriesDays(oneSeries.values, fromIndex, toIndex) !== stored.recentHashes[position]
    })
    .map((oneSeries) => oneSeries.key)
  return { mode: 'incremental', refitKeys }
}

/** A site's share of the paid amount at each delay (index = days after the document), and where it came from. */
export interface IDelayCurve {
  shares: number[]
  source: 'site' | 'company' | 'none'
}

export type TCashDirection = 'collection' | 'payment'

export interface IDelayCurves {
  curveOf: (direction: TCashDirection, siteID: number) => IDelayCurve
}

/**
 * buildDelayCurves measures, per site and direction, how the paid amount spreads over the days after
 * its document (§4), from the cash-delay files of the last DELAY_CURVE_DAYS. A register gives its site
 * and currency. A site with fewer than DELAY_CURVE_MIN_ENTRIES (day, delay) entries uses the company
 * curve; without any payment at all, everything is paid on the document's day.
 */
export function buildDelayCurves(
  delayFiles: FrameFileRows[], cashBanks: { ID: number, SiteID: number, CurrencyType: number }[], usdSellRate: number, fromDay: number,
): IDelayCurves {
  const cashBankByID = new Map(cashBanks.map((cashBank) => [cashBank.ID, cashBank]))
  type ICurveSums = { amounts: number[], entries: number }
  const newSums = (): ICurveSums => ({ amounts: new Array(DELAY_MAX_DAYS + 1).fill(0), entries: 0 })
  const sumsBySiteDirection = new Map<string, ICurveSums>()
  const companySums: Record<TCashDirection, ICurveSums> = { collection: newSums(), payment: newSums() }

  for (const file of delayFiles) {
    if ((file.keys[KEY_DATE] as number) < fromDay) continue
    const cashBank = cashBankByID.get(file.keys[KEY_CASH_BANK] as number)
    const types = file.rows[DELAY_ROW_TYPE] as number[]
    const delays = file.rows[DELAY_ROW_DAYS] as number[]
    const amounts = file.sums[DELAY_SUM_AMOUNT] as number[]
    for (let rowIndex = 0; rowIndex < types.length; rowIndex++) {
      const direction: TCashDirection = types[rowIndex] === CASH_MOVEMENT_SALE_COLLECTION ? 'collection' : 'payment'
      const amountPEN = Math.abs(toPEN(amounts[rowIndex], cashBank?.CurrencyType || 1, usdSellRate))
      const delay = Math.min(Math.max(delays[rowIndex], 0), DELAY_MAX_DAYS)
      const siteKey = `${cashBank?.SiteID || 0}_${direction}`
      let siteSums = sumsBySiteDirection.get(siteKey)
      if (!siteSums) { siteSums = newSums(); sumsBySiteDirection.set(siteKey, siteSums) }
      for (const sums of [siteSums, companySums[direction]]) {
        sums.amounts[delay] += amountPEN
        sums.entries++
      }
    }
  }

  const sharesOf = (sums: ICurveSums) => {
    const total = sums.amounts.reduce((sum, amount) => sum + amount, 0)
    return sums.amounts.map((amount) => amount / total)
  }
  const companyCurves: Record<TCashDirection, IDelayCurve> = {
    collection: { shares: [], source: 'company' }, payment: { shares: [], source: 'company' },
  }
  for (const direction of ['collection', 'payment'] as TCashDirection[]) {
    companyCurves[direction] = companySums[direction].entries > 0
      ? { shares: sharesOf(companySums[direction]), source: 'company' }
      : { shares: [1], source: 'none' }
  }
  return {
    curveOf: (direction, siteID) => {
      const siteSums = sumsBySiteDirection.get(`${siteID}_${direction}`)
      return siteSums && siteSums.entries >= DELAY_CURVE_MIN_ENTRIES
        ? { shares: sharesOf(siteSums), source: 'site' }
        : companyCurves[direction]
    },
  }
}

/**
 * shiftOverdueReceivables moves each overdue receivable (the backend dates it today) to when its
 * site's collections usually arrive at that age: the median of the curve past the sale's age, so a
 * sale already late waits for the late part of the curve. Without collections that late, it stays today.
 */
export function shiftOverdueReceivables(commitments: ICashFlowCommitment[], curves: IDelayCurves, today: number): ICashFlowCommitment[] {
  return commitments.map((commitment) => {
    const documentDate = commitment.DocumentDate || 0
    const isOverdueReceivable = commitment.Stream === 1 && documentDate > 0 && commitment.Date === today
      && (commitment.DueDate || documentDate) < today
    if (!isOverdueReceivable) return commitment
    const age = today - documentDate
    const { shares } = curves.curveOf('collection', commitment.SiteID)
    const lateShare = shares.slice(age + 1).reduce((total, share) => total + share, 0)
    if (lateShare <= 0) return commitment
    let cumulative = 0
    for (let delay = age + 1; delay < shares.length; delay++) {
      cumulative += shares[delay]
      if (cumulative >= lateShare / 2) return { ...commitment, Date: Math.max(today, documentDate + delay) }
    }
    return commitment
  })
}

/** The engine's answer the page needs: its series in state order and their forecast from firstDay. */
export interface IEngineForecast {
  siteIDs: number[]
  productIDs: number[]
  firstDay: number
  horizon: number
  /** Series-major, soles. */
  values: number[]
}

export interface IForecastFlows {
  commitments: ICashFlowCommitment[]
  /** Forecast gross sales and spending of the whole company per day (PEN cents), for the taxes. */
  salesByDay: Map<number, number>
  spendingByDay: Map<number, number>
}

/**
 * forecastFlows turns the engine's forecast documents into cash (§4): each day's forecast is spread
 * over the following days by its site's curve, cash(d) = Σ_k forecast(d − k) · share(k). Only
 * documents dated tomorrow or later are forecast: today's and the past are Known. One commitment per
 * stream, site and day, up to endDay.
 */
export function forecastFlows(forecast: IEngineForecast, curves: IDelayCurves, today: number, endDay: number): IForecastFlows {
  const amountByStreamSiteDay = new Map<string, { stream: number, siteID: number, day: number, amount: number }>()
  const salesByDay = new Map<number, number>()
  const spendingByDay = new Map<number, number>()
  forecast.siteIDs.forEach((siteID, position) => {
    const isSales = isSalesSeries(forecast.productIDs[position])
    const stream = isSales ? CASH_FLOW_STREAM_FORECAST_COLLECTIONS : CASH_FLOW_STREAM_FORECAST_OUTFLOWS
    const { shares } = curves.curveOf(isSales ? 'collection' : 'payment', siteID)
    const totals = isSales ? salesByDay : spendingByDay
    for (let offset = 0; offset < forecast.horizon; offset++) {
      const documentDay = forecast.firstDay + offset
      // A series the engine could not forecast answers NaN: it adds nothing.
      const value = forecast.values[position * forecast.horizon + offset]
      const documentCents = Number.isFinite(value) ? Math.max(0, value) * 100 : 0
      if (documentDay <= today || documentDay > endDay || documentCents === 0) continue
      totals.set(documentDay, (totals.get(documentDay) || 0) + documentCents)
      for (let delay = 0; delay < shares.length && documentDay + delay <= endDay; delay++) {
        if (!shares[delay]) continue
        const day = documentDay + delay
        const key = `${stream}_${siteID}_${day}`
        const entry = amountByStreamSiteDay.get(key) || { stream, siteID, day, amount: 0 }
        entry.amount += documentCents * shares[delay] * (isSales ? 1 : -1)
        amountByStreamSiteDay.set(key, entry)
      }
    }
  })
  const commitments = [...amountByStreamSiteDay.values()]
    .map((entry): ICashFlowCommitment => ({
      Stream: entry.stream, Layer: CASH_FLOW_LAYER_FORECAST, SiteID: entry.siteID, SourceID: 0,
      Label: entry.stream === CASH_FLOW_STREAM_FORECAST_COLLECTIONS
        ? 'Collection of forecast sales|Cobro de ventas pronosticadas' : 'Payment of forecast purchases and expenses|Pago de compras y gastos pronosticados',
      Date: entry.day, DueDate: 0, Amount: Math.round(entry.amount), CurrencyType: 1,
    }))
    .filter((commitment) => commitment.Amount !== 0)
  return { commitments, salesByDay, spendingByDay }
}

/**
 * forecastTaxes completes each open or future period with the forecast sales and spending (§7.3):
 * IGV = sales IGV (the closed periods' ratio) − crédito fiscal (the spending's ratio) − the saldo a
 * favor carried, a negative result carrying forward; renta = rate × net sales; a Nuevo RUS company
 * pays its cuota instead. The bases hold what the open period recorded so far (the cash-documents
 * frame). Each tax falls due on its period's SUNAT date, in the company lane.
 */
export function forecastTaxes(input: {
  rules: ITaxRules
  openBases: ITaxPeriodBase[]
  salesTaxRatio: number
  creditCarried: number
  salesByDay: Map<number, number>
  spendingByDay: Map<number, number>
  purchaseTaxRatio: number
}): ICashFlowCommitment[] {
  const { rules, salesTaxRatio } = input
  const sumOfPeriod = (amountsByDay: Map<number, number>, period: number) => {
    let total = 0
    for (const [day, amount] of amountsByDay) if (periodOfDay(day) === period) total += amount
    return total
  }
  const commitments: ICashFlowCommitment[] = []
  const addTax = (period: number, dueDate: number, name: string, amount: number) => {
    if (amount <= 0) return
    commitments.push({
      Stream: CASH_FLOW_STREAM_FORECAST_TAXES, Layer: CASH_FLOW_LAYER_FORECAST, SiteID: 0, SourceID: period,
      Label: `${name} ${period}`, Date: dueDate, DueDate: dueDate, Amount: -Math.round(amount), CurrencyType: 1,
    })
  }
  let creditCarried = input.creditCarried
  for (const base of input.openBases) {
    if (rules.NrusQuota > 0) {
      addTax(base.period, base.dueDate, 'Nuevo RUS', rules.NrusQuota)
      continue
    }
    const salesGross = sumOfPeriod(input.salesByDay, base.period)
    const salesTax = base.salesTax + salesGross * salesTaxRatio
    const salesNet = base.salesNet + salesGross - salesGross * salesTaxRatio
    if (rules.ChargesIgv) {
      const igv = salesTax - base.purchaseTaxCredit - base.retentionCredit - sumOfPeriod(input.spendingByDay, base.period) * input.purchaseTaxRatio - creditCarried
      creditCarried = Math.max(0, -igv)
      addTax(base.period, base.dueDate, 'IGV', igv)
    }
    addTax(base.period, base.dueDate, 'Renta', salesNet * rules.IncomeTaxRate / 10000)
  }
  return commitments
}

/**
 * The statistical warnings (§11): a history shorter than a year (no yearly season, so the monthly
 * view is flatter than it will be), sites selling for less than HISTORY_SHORT_DAYS, and sites whose
 * delay curve fell back to the company's or to day 0.
 */
export function forecastWarnings(input: {
  window: ICashFlowWindow
  firstSaleDayBySite: Map<number, number>
  series: ICashFlowSeries[]
  curves: IDelayCurves
  siteName: (siteID: number) => string
}): ICashFlowServerWarning[] {
  const warnings: ICashFlowServerWarning[] = []
  const push = (code: string, records: { ID: number, Label: string }[]) => {
    if (records.length > 0) warnings.push({ Code: code, AmountAtStakePEN: 0, Records: records, RecordCount: records.length })
  }
  const historyDays = input.window.lastDay - input.window.startDay + 1
  push('history-under-year', historyDays < HISTORY_DAYS ? [{ ID: 0, Label: `${historyDays} d` }] : [])
  push('history-short', [...input.firstSaleDayBySite]
    .filter(([, firstSaleDay]) => input.window.lastDay - firstSaleDay + 1 < HISTORY_SHORT_DAYS)
    .map(([siteID, firstSaleDay]) => ({ ID: siteID, Label: `${input.siteName(siteID)} · ${input.window.lastDay - firstSaleDay + 1} d` })))
  const fallbackCurves: { ID: number, Label: string }[] = []
  for (const oneSeries of input.series) {
    const direction: TCashDirection = isSalesSeries(oneSeries.productID) ? 'collection' : 'payment'
    const label = `${input.siteName(oneSeries.siteID)} · ${direction === 'collection' ? 'cobros' : 'pagos'}`
    if (input.curves.curveOf(direction, oneSeries.siteID).source !== 'site' && !fallbackCurves.some((record) => record.Label === label)) {
      fallbackCurves.push({ ID: oneSeries.siteID, Label: label })
    }
  }
  push('delay-curve-missing', fallbackCurves)
  return warnings
}

/** The series packed series-major for the worker, with the change-check hashes the stored state keeps. */
export interface IEngineRunRequest {
  runID: number
  storageKey: string
  window: ICashFlowWindow
  horizonDays: number
  calendarDays: Int32Array
  calendarTypes: Uint8Array
  keys: string[]
  siteIDs: Int32Array
  productIDs: Int32Array
  categoryIDs: Int32Array
  /** Series-major over the window. */
  values: Float64Array
}

export type IEngineWorkerAnswer = { runID: number, forecast: IEngineForecast, mode: TEnginePlan['mode'] } | { runID: number, error: string }

export const packEngineRequest = (
  runID: number, storageKey: string, series: ICashFlowSeries[], window: ICashFlowWindow, horizonDays: number,
  calendarEvents: { Date: number, EventType: number }[],
): IEngineRunRequest => {
  const dayCount = window.lastDay - window.startDay + 1
  const values = new Float64Array(series.length * dayCount)
  series.forEach((oneSeries, seriesIndex) => values.set(oneSeries.values, seriesIndex * dayCount))
  return {
    runID, storageKey, window, horizonDays,
    calendarDays: Int32Array.from(calendarEvents.map((event) => event.Date)),
    calendarTypes: Uint8Array.from(calendarEvents.map((event) => event.EventType)),
    keys: series.map((oneSeries) => oneSeries.key),
    siteIDs: Int32Array.from(series.map((oneSeries) => oneSeries.siteID)),
    productIDs: Int32Array.from(series.map((oneSeries) => oneSeries.productID)),
    categoryIDs: Int32Array.from(series.map((oneSeries) => oneSeries.categoryID)),
    values,
  }
}
