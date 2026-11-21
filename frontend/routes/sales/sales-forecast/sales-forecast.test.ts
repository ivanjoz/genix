import { describe, expect, it } from 'vitest'
import type { FrameFileRows } from '@genix/ui/cache'
import type { SiteProductStockSeries } from '#core/stock-closing.ts'
import {
	DEFAULT_FORECAST_PARAMS, buildForecastSeries, clipWindowToFirstSale, firstSaleDayOf, forecastRowsOf, forecastWarnings, forecastWindowOf, parseForecastParams,
	sliceSeriesDays, type IForecastRunResult,
} from './sales-forecast.ts'

const SITE = 1
const PRODUCT_A = 10
const PRODUCT_B = 20
const TODAY = 20_000

const saleFile = (date: number, siteID: number, productIDs: number[], quantities: number[]) =>
	({ keys: [date, siteID], rows: [productIDs], sums: [quantities], snapshot: 0 }) as unknown as FrameFileRows

describe('forecastWindowOf', () => {
	it('ends yesterday and holds the last 28 days out of the fit', () => {
		expect(forecastWindowOf(TODAY, 90)).toEqual({ startDay: TODAY - 90, fitEndDay: TODAY - 29, lastDay: TODAY - 1 })
	})
})

describe('buildForecastSeries', () => {
	const window = forecastWindowOf(TODAY, 90)

	it('takes the series that sold, with their stock shifted past the opening day', () => {
		const stockDays = 92 // startDay − 1 to today
		const closing = new Float64Array(stockDays).fill(5)
		closing[0] = 7
		const stockSeries: SiteProductStockSeries[] = [
			{ siteID: SITE, productID: PRODUCT_A, closing, inflow: new Float64Array(stockDays), outflow: new Float64Array(stockDays) },
			// Stock but no sale: not forecast.
			{ siteID: SITE, productID: PRODUCT_B, closing: new Float64Array(stockDays).fill(3), inflow: new Float64Array(stockDays), outflow: new Float64Array(stockDays) },
		]
		const saleFiles = [
			saleFile(window.startDay, SITE, [PRODUCT_A], [2]),
			saleFile(window.lastDay, SITE, [PRODUCT_A], [4]),
			// Outside the window, and with no site.
			saleFile(TODAY, SITE, [PRODUCT_A], [9]),
			saleFile(window.lastDay, 0, [PRODUCT_A], [3]),
		]
		const { series, unsitedSaleUnits } = buildForecastSeries(saleFiles, stockSeries, window.startDay - 1, window, () => 7)
		expect(unsitedSaleUnits).toBe(3)
		expect(series).toHaveLength(1)
		const [productSeries] = series
		expect(productSeries.categoryID).toBe(7)
		expect(productSeries.initialStock).toBe(7)
		expect(productSeries.sales).toHaveLength(90)
		expect(productSeries.sales[0]).toBe(2)
		expect(productSeries.sales[89]).toBe(4)
		expect(productSeries.closing).toHaveLength(90)
		expect(productSeries.saleDays).toBe(2)
	})

	it('counts empty-shelf and negative-stock days, and leaves untracked stock uncensored', () => {
		const stockDays = 92
		const closing = new Float64Array(stockDays).fill(4)
		closing.fill(0, 11, 21) // window days 10–19: empty shelf, no sale
		closing[30] = -2 // window day 29: sold below zero
		const stockSeries = [{ siteID: SITE, productID: PRODUCT_A, closing, inflow: new Float64Array(stockDays), outflow: new Float64Array(stockDays) }]
		const saleFiles = [saleFile(window.startDay + 29, SITE, [PRODUCT_A, PRODUCT_B], [1, 1])]
		const { series } = buildForecastSeries(saleFiles, stockSeries, window.startDay - 1, window, () => 0)
		const tracked = series.find((one) => one.productID === PRODUCT_A)!
		expect(tracked.emptyShelfDays).toBe(10)
		expect(tracked.negativeStockDays).toBe(1)
		// No stock rows at all: zeros, which the engine reads as untracked, never as empty shelf.
		const untracked = series.find((one) => one.productID === PRODUCT_B)!
		expect(untracked.emptyShelfDays).toBe(0)
		expect(untracked.closing.every((value) => value === 0)).toBe(true)
	})
})

describe('clipWindowToFirstSale', () => {
	it('starts the history at the first sale with a site, reading the stock from its own start', () => {
		const window = forecastWindowOf(TODAY, 90)
		const firstSaleDay = window.startDay + 30
		const saleFiles = [
			saleFile(window.startDay + 10, 0, [PRODUCT_A], [5]), // no site: does not count
			saleFile(window.startDay + 20, SITE, [PRODUCT_A], [0]), // nothing sold
			saleFile(firstSaleDay, SITE, [PRODUCT_A], [2]),
		]
		expect(firstSaleDayOf(saleFiles)).toBe(firstSaleDay)
		const clipped = clipWindowToFirstSale(window, firstSaleDayOf(saleFiles)!)
		expect(clipped).toEqual({ ...window, startDay: firstSaleDay })
		expect(clipWindowToFirstSale(window, window.startDay - 50)).toEqual(window)

		// Stock read over the whole asked window: day i of the clipped one sits 31 + i into it.
		const stockDays = 92
		const closing = Float64Array.from({ length: stockDays }, (_, index) => index)
		const stockSeries = [{ siteID: SITE, productID: PRODUCT_A, closing, inflow: new Float64Array(stockDays), outflow: new Float64Array(stockDays) }]
		const [series] = buildForecastSeries(saleFiles, stockSeries, window.startDay - 1, clipped, () => 0).series
		expect(series.sales).toHaveLength(60)
		expect(series.sales[0]).toBe(2)
		expect(series.initialStock).toBe(30)
		expect(series.closing[0]).toBe(31)
	})
})

describe('sliceSeriesDays', () => {
	it('cuts the same days out of every series', () => {
		const values = Float64Array.from([1, 2, 3, 4, 10, 20, 30, 40])
		expect([...sliceSeriesDays(values, 4, 1, 3)]).toEqual([2, 3, 20, 30])
	})
})

describe('parseForecastParams', () => {
	it('keeps valid values and defaults the rest', () => {
		expect(parseForecastParams({ horizonDays: 14, maxDamping: 5, junk: 1 })).toEqual({ ...DEFAULT_FORECAST_PARAMS, horizonDays: 14 })
		expect(parseForecastParams(null)).toEqual(DEFAULT_FORECAST_PARAMS)
	})
})

describe('forecastRowsOf and forecastWarnings', () => {
	const window = forecastWindowOf(TODAY, 90)
	const series = [{
		siteID: SITE, productID: PRODUCT_A, categoryID: 0, initialStock: 0,
		sales: new Float64Array(90).fill(1), closing: new Float64Array(90), inflow: new Float64Array(90),
		saleDays: 90, emptyShelfDays: 0, negativeStockDays: 3,
	}]
	const metricsOf = (mase: number) => Array.from({ length: 7 }, () => mase)
	const result: IForecastRunResult = {
		keys: { lastDay: window.lastDay, siteIds: [SITE], productIds: [PRODUCT_A], categoryIds: [0], models: ['HoltWinters'], scenarios: ['A'], emptyShelfDays: [3] },
		forecast: { firstDay: TODAY, horizon: 10, values: Array.from({ length: 10 }, () => 2) },
		metrics: { count: metricsOf(28), wape: metricsOf(0.3), bias: metricsOf(0), mase: metricsOf(1.4) },
		pendingRefits: { series: [], reasons: [] },
		elapsedMs: 1,
	}

	it('sums the forecast and flags a series worse than the seasonal naive', () => {
		const [row] = forecastRowsOf(series, result, () => 5)
		expect(row.next7).toBe(14)
		expect(row.horizonTotal).toBe(20)
		expect(row.lastSales).toBe(28)
		expect(row.stockToday).toBe(5)
		expect(row.worseThanNaive).toBe(true)
	})

	it('warns about short history, data gaps, missing calendar years and stockouts', () => {
		const rows = forecastRowsOf(series, result, () => 0)
		const warnings = forecastWarnings({
			params: { ...DEFAULT_FORECAST_PARAMS, historyDays: 90 },
			window, series, rows,
			// Holidays only in the history's first year; no paydays, no company events.
			calendarEvents: [{ Date: window.startDay, EventType: 1, IsGlobal: true }],
			unsitedSaleUnits: 4,
			stockConsistent: true,
			seriesName: () => 'Product A — Site',
			productExists: () => true,
			productHasCategory: () => false,
		})
		const titles = warnings.map((warning) => warning.title.split('|')[0])
		expect(titles).toEqual(expect.arrayContaining([
			'History shorter than a year', 'Sales without a site', 'Negative stock',
			'Products without a category', 'No paydays', 'No company events', 'Out of stock now', 'Worse than repeating last week',
		]))
		expect(titles).not.toContain('Products with few sales')
		// A 90-day window puts every product in scenario A; the history warnings already say so.
		expect(titles).not.toContain('Products with little history')
		expect(titles).not.toContain('Stock changed while it was read')
		// High severity first.
		expect(warnings[0].severity).toBe('high')
	})
})
