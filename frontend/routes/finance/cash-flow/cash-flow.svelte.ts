import type { FrameFileRows } from '@genix/ui/cache'
import { getSunatRateRange } from '@ivanjoz/public-business-data'
import { Env } from '#core/env.ts'
import { todayUnixDay } from '#libs/helpers.ts'
import { WarehousesService } from '#routes/business/branches-warehouses/branches-warehouses.svelte.ts'
import { CajasService } from '#routes/finance/cash-banks/cajas.svelte.ts'
import { EXCHANGE_RATE_MAX_AGE_DAYS, calendarMonthKeys, latestSellRate } from '#routes/finance/exchange-rate/exchange-rate.ts'
import { BcrpDefaultRates, ExchangeRatesService } from '#routes/finance/exchange-rate/exchange-rate.svelte.ts'
import { formatTime } from '#libs/helpers.ts'
import { ClientProviderService, ClientProviderType } from '#services/crm/client-provider.svelte.ts'
import {
  CashFlowEntriesService, fetchClientDebtFiles, fetchHistoryFrameFiles, getCashDocumentsDay, getCashFlowCommitments,
  type ICashDocumentDetail, type ICashFlowCommitment, type ICashFlowOldPayables, type ICashFlowServerWarning, type ICashFlowTaxOutlook,
} from '#services/finance/cash-flow.svelte.ts'
import { CalendarEventsService } from '#services/sales/calendar-events.svelte.ts'
import { CASH_FLOW_LAYER_KNOWN, CASH_FLOW_YEAR_DAYS, toPEN, type ICashFlowCell } from './cash-flow.ts'
import {
  cashDocumentRowsOf, clientCreditDaysWarning, closedPeriodTaxes, closedTaxCommitments, documentsOfItems, documentWarnings,
  DOCUMENT_WINDOW_DAYS, drilldownDatesByKind, firstScanPeriod, KIND_SALE_OPEN, knownDocuments, oldPayableCommitments, salesTaxRatioOf,
  scanWarningRecords, sellRateLookup, splitTaxPeriods, taxPeriodBases, type ICashDocumentRow, type ITaxPeriodBase,
} from './cash-flow-documents.ts'
import {
  buildCashFlowSeries, buildDelayCurves, cashFlowWindowOf, clipSeries, clipWindowToFirstSale, DELAY_CURVE_DAYS,
  forecastFlows, forecastTaxes, forecastWarnings, MIN_ENGINE_HISTORY_DAYS, packEngineRequest, shiftOverdueReceivables,
  type ICashFlowSeriesBuild, type IDelayCurves, type IEngineForecast, type IEngineWorkerAnswer,
} from './cash-flow-forecast.ts'

const isoDateOf = (unixDay: number) => new Date(unixDay * 86400000).toISOString().slice(0, 10)

// short: too little sales history for the engine; the Forecast layer then holds only the taxes.
type TForecastStatus = 'loading' | 'fitting' | 'ready' | 'short' | 'error'

// The cash flow page's state: the backend's commitments (payroll, loans, schedules; read once per
// visit, a year ahead), the open documents and their taxes from the cash-documents frame, the planned
// entries, the registers, the sites, the dollar rate the forecast restates with, and the Forecast
// layer computed in the browser from the history frames.
export class CashFlowState {
  today = todayUnixDay()
  cashBanksService = new CajasService()
  sitesService = new WarehousesService()
  entriesService = new CashFlowEntriesService(true)
  exchangeRatesService = new ExchangeRatesService(true)
  bcrpDefaultRates = new BcrpDefaultRates()
  calendarEvents = new CalendarEventsService(false)
  clientsService = new ClientProviderService(ClientProviderType.CLIENT)

  // Raw: rebuilt whole, never edited in place.
  backendCommitments = $state.raw<ICashFlowCommitment[]>([])
  serverWarnings = $state.raw<ICashFlowServerWarning[]>([])
  taxOutlook = $state.raw<ICashFlowTaxOutlook | null>(null)
  oldPayables = $state.raw<ICashFlowOldPayables>({ Local: 0, Dollars: 0 })
  // From the cash-documents frame: the open documents, the closed periods' taxes, their warnings.
  documentCommitments = $state.raw<ICashFlowCommitment[]>([])
  documentWarnings = $state.raw<ICashFlowServerWarning[]>([])
  private openTaxBases: ITaxPeriodBase[] = []
  private salesTaxRatio = 0
  private creditCarried = 0
  isLoading = $state(false)
  loadError = $state('')

  forecastStatus = $state<TForecastStatus>('loading')
  forecastError = $state('')
  forecastCommitments = $state.raw<ICashFlowCommitment[]>([])
  forecastWarnings = $state.raw<ICashFlowServerWarning[]>([])
  private delayCurves = $state.raw<IDelayCurves | null>(null)

  // The last sell rate (× 1000), own or BCRP, at most a week old; 0 when there is none.
  usdSellRate = $derived(latestSellRate(this.exchangeRatesService.records, this.bcrpDefaultRates.ratesByMonthKey,
    this.today, EXCHANGE_RATE_MAX_AGE_DAYS))
  /** The Known and Scheduled layers, an overdue receivable moved by its site's collection curve. */
  commitments = $derived.by(() => {
    const commitments = [...this.backendCommitments, ...this.documentCommitments]
    return this.delayCurves ? shiftOverdueReceivables(commitments, this.delayCurves, this.today) : commitments
  })

  private worker: Worker | undefined
  private runID = 0

  constructor() {
    this.bcrpDefaultRates.load(calendarMonthKeys(new Date(), 2))
    this.load()
  }

  /** Commitments first: their tax periods and old payables complete the frame's documents. */
  async load() {
    await this.loadCommitments()
    await this.loadForecast()
  }

  async loadCommitments() {
    this.isLoading = true
    this.loadError = ''
    try {
      const response = await getCashFlowCommitments(CASH_FLOW_YEAR_DAYS)
      this.backendCommitments = response.Commitments || []
      this.serverWarnings = response.Warnings || []
      this.taxOutlook = response.TaxOutlook
      this.oldPayables = response.OldPayables
    } catch (error) {
      this.loadError = String(error)
    }
    this.isLoading = false
  }

  /**
   * Reads the frames (cached in the browser, only changes travel): the open documents and the taxes
   * come from cash-documents at once, then the engine forecasts from the same rows.
   */
  async loadForecast() {
    this.forecastStatus = 'loading'
    try {
      const window = cashFlowWindowOf(this.today)
      const firstDocumentDay = this.today - DOCUMENT_WINDOW_DAYS
      await Promise.all([this.calendarEvents.fetch(), this.cashBanksService.fetch(), this.sitesService.fetch(), this.clientsService.fetch()])
      const [documentFiles, delayFiles, debtFiles, sunatRates] = await Promise.all([
        fetchHistoryFrameFiles('cash-documents', 'cash-documents', firstDocumentDay, this.today),
        fetchHistoryFrameFiles('cash-delay', 'cash-delay', this.today - DELAY_CURVE_DAYS, window.lastDay),
        fetchClientDebtFiles(),
        // Dollar documents are restated at SUNAT's sell rate of their comprobante's day, as the taxes are.
        getSunatRateRange(isoDateOf(firstDocumentDay - 130), isoDateOf(this.today)).catch((error) => {
          console.warn('[cash-flow] SUNAT exchange rates unavailable:', error)
          return []
        }),
      ])
      const sellRateOfDay = sellRateLookup(sunatRates, this.usdSellRate || 1000)
      const rows = cashDocumentRowsOf(documentFiles)
      this.applyDocuments(rows, debtFiles, sellRateOfDay)
      const seriesBuild = buildCashFlowSeries(rows, window, sellRateOfDay)
      this.delayCurves = buildDelayCurves(delayFiles, this.cashBanksService.Cajas, this.usdSellRate || 1000, this.today - DELAY_CURVE_DAYS)
      const salesWindow = clipWindowToFirstSale(window, seriesBuild.firstSaleDay ?? window.lastDay + 1)
      this.forecastWarnings = forecastWarnings({
        window: salesWindow, firstSaleDayBySite: seriesBuild.firstSaleDayBySite, series: seriesBuild.series, curves: this.delayCurves,
        siteName: (siteID) => this.sitesService.SedesMap.get(siteID)?.Name || (siteID ? `#${siteID}` : 'Empresa'),
      })
      if (salesWindow.lastDay - salesWindow.startDay + 1 < MIN_ENGINE_HISTORY_DAYS) {
        this.assembleForecast(null, seriesBuild)
        this.forecastStatus = 'short'
        return
      }
      this.runEngine(clipSeries(seriesBuild.series, window, salesWindow), salesWindow, seriesBuild)
    } catch (loadError) {
      console.error('[cash-flow] forecast load failed', loadError)
      this.forecastStatus = 'error'
      this.forecastError = loadError instanceof Error ? loadError.message : String(loadError)
    }
  }

  private runEngine(series: ICashFlowSeriesBuild['series'], window: ReturnType<typeof cashFlowWindowOf>, seriesBuild: ICashFlowSeriesBuild) {
    const runID = ++this.runID
    this.worker ??= new Worker(new URL('./cash-flow.worker.ts', import.meta.url), { type: 'module', name: 'cash-flow' })
    this.worker.onmessage = (event: MessageEvent<IEngineWorkerAnswer>) => {
      const answer = event.data
      if (answer.runID !== runID) return
      if ('error' in answer) {
        this.forecastStatus = 'error'
        this.forecastError = answer.error
        return
      }
      this.assembleForecast(answer.forecast, seriesBuild)
      this.forecastStatus = 'ready'
    }
    this.forecastStatus = 'fitting'
    const calendarEvents = this.calendarEvents.records.filter((event) => event.ss > 0)
    const request = packEngineRequest(runID, `c${Env.getCompanyID()}`, series, window, CASH_FLOW_YEAR_DAYS + 1, calendarEvents)
    this.worker.postMessage(request, [
      request.calendarDays.buffer, request.calendarTypes.buffer, request.siteIDs.buffer, request.productIDs.buffer,
      request.categoryIDs.buffer, request.values.buffer,
    ])
  }

  /**
   * The frame's open documents as the Known layer, the closed periods' taxes, and the document
   * warnings (counts and amounts per document day and site: the frame keeps no documents).
   */
  private applyDocuments(rows: ICashDocumentRow[], debtFiles: FrameFileRows[], sellRateOfDay: (day: number) => number) {
    const outlook = this.taxOutlook || { Rules: { ChargesIgv: false, IncomeTaxRate: 0, NrusQuota: 0 }, Periods: [] }
    const known = knownDocuments(rows, this.today, sellRateOfDay)
    const { closed, open } = splitTaxPeriods(taxPeriodBases(rows, outlook, sellRateOfDay), this.today)
    const { dues, creditCarried } = closedPeriodTaxes(closed, outlook.Rules)
    const closedTaxes = closedTaxCommitments(dues, this.today)
    this.openTaxBases = open
    this.salesTaxRatio = salesTaxRatioOf(closed)
    this.creditCarried = creditCarried
    this.documentCommitments = [...known.commitments, ...oldPayableCommitments(this.oldPayables, this.today), ...closedTaxes.commitments]

    const warningRecords = known.warningRecords
    for (const [code, records] of scanWarningRecords(rows, firstScanPeriod(outlook, this.today), sellRateOfDay)) warningRecords.set(code, records)
    const siteName = (siteID: number) => this.sitesService.SedesMap.get(siteID)?.Name || (siteID ? `#${siteID}` : 'Empresa')
    const warnings = documentWarnings(warningRecords, (date, siteID) => `${formatTime(date, 'd-m-Y')} · ${siteName(siteID)}`)
    if (closedTaxes.unpaid.length) {
      warnings.push({ Code: 'tax-period-unpaid', AmountAtStakePEN: closedTaxes.unpaidAmount, Records: closedTaxes.unpaid, RecordCount: closedTaxes.unpaid.length })
    }
    const oldPayablesLocal = this.oldPayables.Local + Math.round(this.oldPayables.Dollars * (this.usdSellRate || 1000) / 1000)
    if (oldPayablesLocal > 0) warnings.push({ Code: 'payables-before-window', AmountAtStakePEN: oldPayablesLocal, Records: [], RecordCount: 1 })
    this.documentWarnings = [...warnings, ...clientCreditDaysWarning(debtFiles, this.clientsService.recordsMap)]
  }

  /**
   * The documents behind a cell's frame items (GET.cash-documents-day, one call per Kind), each in
   * place of the items it stands for; the cell's other items stay as they are.
   */
  async drilldownItems(items: ICashFlowCell['items'], siteID: number | undefined): Promise<ICashFlowCell['items']> {
    const datesByKind = drilldownDatesByKind(items)
    if (datesByKind.size === 0) return items
    const documentItems: ICashFlowCell['items'] = []
    for (const [kind, dates] of datesByKind) {
      const documents = documentsOfItems(items, kind, (await getCashDocumentsDay(kind, dates, siteID)) || [])
      const sign = kind === KIND_SALE_OPEN ? 1 : -1
      documentItems.push(...documents.map((document: ICashDocumentDetail) => ({
        label: document.Label, date: document.Date, dueDate: document.DueDate, layer: CASH_FLOW_LAYER_KNOWN,
        amountPEN: sign * toPEN(document.Amount, document.CurrencyType, this.usdSellRate || 1000),
      })))
    }
    return [...items.filter((item) => !item.kind), ...documentItems]
  }

  /** The forecast documents as cash, then the taxes of the open and future periods on top. */
  private assembleForecast(engineForecast: IEngineForecast | null, seriesBuild: ICashFlowSeriesBuild) {
    const endDay = this.today + CASH_FLOW_YEAR_DAYS - 1
    const flows = engineForecast && this.delayCurves
      ? forecastFlows(engineForecast, this.delayCurves, this.today, endDay)
      : { commitments: [], salesByDay: new Map<number, number>(), spendingByDay: new Map<number, number>() }
    const taxes = this.taxOutlook
      ? forecastTaxes({
        rules: this.taxOutlook.Rules, openBases: this.openTaxBases, salesTaxRatio: this.salesTaxRatio, creditCarried: this.creditCarried,
        salesByDay: flows.salesByDay, spendingByDay: flows.spendingByDay, purchaseTaxRatio: seriesBuild.purchaseTaxRatio,
      })
      : []
    this.forecastCommitments = [...flows.commitments, ...taxes]
  }

  destroy() {
    this.worker?.terminate()
  }
}
