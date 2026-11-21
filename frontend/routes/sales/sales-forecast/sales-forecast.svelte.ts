import { getFechaUnix } from '@genix/ui/utilities'
import type { FrameFileRows } from '@genix/ui/cache'
import { Env } from '#core/env.ts'
import { closingStockSeries, isStockNewerThanFrame, type SiteProductStockSeries } from '#core/stock-closing.ts'
import { fetchDailyProductStockFiles } from '#services/logistics/daily-product-stock.ts'
import { fetchSaleQuantityFiles } from '#services/sales/sale-quantity.ts'
import { CalendarEventsService } from '#services/sales/calendar-events.svelte.ts'
import { ProductsService } from '#services/production/products.svelte.ts'
import { SharedListsService } from '#services/business/shared-lists.svelte.ts'
import { WarehousesService } from '#routes/business/branches-warehouses/branches-warehouses.svelte.ts'
import { ProductStockSimpleService } from '#routes/logistics/products-stock/stock-movement.ts'
import {
	MIN_FIT_DAYS, buildForecastSeries, clipWindowToFirstSale, firstSaleDayOf, forecastRowsOf, forecastWarnings, forecastWindowOf, packForecastRequest, parseForecastParams,
	type IForecastParams, type IForecastRow, type IForecastSeries, type IForecastWarning, type IForecastWindow,
	type IForecastWorkerAnswer,
} from './sales-forecast.ts'

// The stock is read after its daily frame; a movement landing between both reads offsets every
// rebuilt day, so the pair is read again (plan §10.3). After this many tries the page goes on and warns.
const STOCK_READ_ATTEMPTS = 3
// Shared list 1 holds the product categories.
const CATEGORY_LIST_ID = 1

const paramsStorageKey = () => `sales-forecast-params:c${Env.getCompanyID()}`

// short: sales recorded for too few days to fit and hold out the last weeks.
type TForecastStatus = 'loading' | 'fitting' | 'ready' | 'empty' | 'short' | 'error'

export class SalesForecastState {
	params: IForecastParams = $state(parseForecastParams(JSON.parse(localStorage.getItem(paramsStorageKey()) || 'null')))
	status: TForecastStatus = $state('loading')
	errorMessage = $state('')
	elapsedMs = $state(0)
	// Raw, not a deep $state proxy: it goes to the worker, and postMessage cannot clone a Proxy.
	window: IForecastWindow = $state.raw(forecastWindowOf(getFechaUnix(), 365))
	series: IForecastSeries[] = $state.raw([])
	rows: IForecastRow[] = $state.raw([])
	warnings: IForecastWarning[] = $state.raw([])

	products = new ProductsService()
	categories = new SharedListsService([CATEGORY_LIST_ID])
	warehouses = new WarehousesService(false)
	calendarEvents = new CalendarEventsService(false)
	stock = new ProductStockSimpleService()

	private worker: Worker | undefined
	private runID = 0
	private unsitedSaleUnits = 0
	private stockConsistent = true
	private loadedHistoryDays = 0
	private stockTodayBySiteProduct = new Map<string, number>()

	seriesName = (siteID: number, productID: number) => {
		const productName = this.products.recordsMap.get(productID)?.Name || `#${productID}`
		const siteName = this.warehouses.SedesMap.get(siteID)?.Name || `#${siteID}`
		return `${productName} — ${siteName}`
	}

	/** Saves the parameters and runs the forecast again; a new history reads another range first. */
	async applyParams() {
		localStorage.setItem(paramsStorageKey(), JSON.stringify(this.params))
		if (this.params.historyDays !== this.loadedHistoryDays) await this.loadInputs()
		else this.runForecast()
	}

	/** Reads every input (cached in the browser, only changes travel) and runs the forecast. */
	async loadInputs() {
		this.status = 'loading'
		try {
			const today = getFechaUnix()
			const window = forecastWindowOf(today, this.params.historyDays)
			await Promise.all([this.products.fetch(), this.categories.fetch(), this.warehouses.fetch(), this.calendarEvents.fetch()])
			const [saleFiles, stockSeries] = await Promise.all([
				fetchSaleQuantityFiles(window.startDay, window.lastDay),
				this.readStockSeries(window.startDay - 1, today),
			])
			const firstSaleDay = firstSaleDayOf(saleFiles) ?? window.lastDay + 1
			const salesWindow = clipWindowToFirstSale(window, firstSaleDay)
			this.loadedHistoryDays = this.params.historyDays
			if (salesWindow.fitEndDay - salesWindow.startDay + 1 < MIN_FIT_DAYS) {
				this.window = salesWindow
				this.series = []
				this.rows = []
				this.status = firstSaleDay > window.lastDay ? 'empty' : 'short'
				this.refreshWarnings()
				return
			}
			this.buildSeries(saleFiles, stockSeries, window.startDay - 1, salesWindow)
			this.runForecast()
		} catch (loadError) {
			console.error('[sales-forecast] load failed', loadError)
			this.status = 'error'
			this.errorMessage = loadError instanceof Error ? loadError.message : String(loadError)
		}
	}

	/** The end-of-day stock of every site and product over [fromDay, today], read frame first. */
	private async readStockSeries(fromDay: number, today: number): Promise<SiteProductStockSeries[]> {
		let stockFrame = await fetchDailyProductStockFiles(fromDay, today)
		await this.stock.fetch()
		this.stockConsistent = !isStockNewerThanFrame(this.stock.records, stockFrame.snapshot, stockFrame.files)
		for (let attempt = 1; attempt < STOCK_READ_ATTEMPTS && !this.stockConsistent; attempt++) {
			stockFrame = await fetchDailyProductStockFiles(fromDay, today)
			await this.stock.fetchOnline()
			this.stockConsistent = !isStockNewerThanFrame(this.stock.records, stockFrame.snapshot, stockFrame.files)
		}
		const siteIDByWarehouseID = new Map(this.warehouses.Almacenes.map((warehouse) => [warehouse.ID, warehouse.SiteID]))
		const stockSeries = closingStockSeries(this.stock.records, siteIDByWarehouseID, stockFrame.files, fromDay, today)
		this.stockTodayBySiteProduct = new Map(stockSeries.map((series) => [`${series.siteID}_${series.productID}`, series.closing[series.closing.length - 1]]))
		return stockSeries
	}

	private buildSeries(saleFiles: FrameFileRows[], stockSeries: SiteProductStockSeries[], stockFromDay: number, window: IForecastWindow) {
		const categoryIDOfProduct = (productID: number) => this.products.recordsMap.get(productID)?.CategoryIDs?.[0] || 0
		const { series, unsitedSaleUnits } = buildForecastSeries(saleFiles, stockSeries, stockFromDay, window, categoryIDOfProduct)
		this.window = window
		this.series = series
		this.unsitedSaleUnits = unsitedSaleUnits
	}

	/** Fits and forecasts in the worker; an answer to an older run (the user moved a slider since) is dropped. */
	runForecast() {
		const calendarEvents = this.calendarEvents.records.filter((event) => event.ss > 0)
		if (this.series.length === 0) {
			this.rows = []
			this.status = 'empty'
			this.refreshWarnings()
			return
		}
		this.worker ??= this.createWorker()
		this.status = 'fitting'
		const request = packForecastRequest(++this.runID, this.series, this.window, this.params, calendarEvents)
		this.worker.postMessage(request, [
			request.calendarDays.buffer, request.calendarTypes.buffer, request.siteIDs.buffer, request.productIDs.buffer,
			request.categoryIDs.buffer, request.initialStock.buffer, request.sales.buffer, request.closingStock.buffer, request.inflow.buffer,
		])
	}

	private createWorker() {
		const worker = new Worker(new URL('./sales-forecast.worker.ts', import.meta.url), { type: 'module', name: 'sales-forecast' })
		worker.onmessage = (event: MessageEvent<IForecastWorkerAnswer>) => {
			const answer = event.data
			if (answer.runID !== this.runID) return
			if ('error' in answer) {
				this.status = 'error'
				this.errorMessage = answer.error
				return
			}
			this.rows = forecastRowsOf(this.series, answer.result,
				(siteID, productID) => this.stockTodayBySiteProduct.get(`${siteID}_${productID}`) || 0)
			this.elapsedMs = answer.result.elapsedMs
			this.status = 'ready'
			this.refreshWarnings()
		}
		return worker
	}

	private refreshWarnings() {
		this.warnings = forecastWarnings({
			params: this.params,
			window: this.window,
			series: this.series,
			rows: this.rows,
			calendarEvents: this.calendarEvents.records.filter((event) => event.ss > 0),
			unsitedSaleUnits: this.unsitedSaleUnits,
			stockConsistent: this.stockConsistent,
			seriesName: this.seriesName,
			productExists: (productID) => this.products.recordsMap.has(productID),
			productHasCategory: (productID) => (this.products.recordsMap.get(productID)?.CategoryIDs?.length || 0) > 0,
		})
	}

	destroy() {
		this.worker?.terminate()
	}
}
