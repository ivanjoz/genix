import type { FrameFileRows } from '@genix/ui/cache'
import type { SiteProductStockSeries } from '#core/stock-closing.ts'
import type { IProduct } from '#services/production/products.svelte.ts'
import { CURRENCY_PEN, convertUnitPrice, normalizeCurrency } from '#routes/finance/exchange-rate/exchange-rate.ts'

// The sales forecast page (plan/SALES_FORECAST_PLAN.md): the series the engine reads, assembled from
// the sale-quantity frame and the end-of-day stock, and what the page shows of its answer. The engine
// itself (forecast/, Rust → WASM) runs in sales-forecast.worker.ts.

/** What the user tunes on the left column. Persisted per company in the browser. */
export interface IForecastParams {
	/** Days of history the models are fitted on, the last one being yesterday. */
	historyDays: number
	/** Days forecast after yesterday. */
	horizonDays: number
	/** A closing stock at or below this marks a day as partially censored (the shelf ran low). */
	partialCensorThreshold: number
	/** Days of empty shelf the forecast holds through, then halves every as many days. */
	emptyShelfHalfLifeDays: number
	/** Upper bound of the Holt-Winters damping: how far a trend is carried. */
	maxDamping: number
	/** Weeks of history a product's own weekly shape needs to weigh as much as its category's. */
	shrinkWeeks: number
	/** Average days between sales from which a product is intermittent (TSB). */
	intermittentAdi: number
}

/** The engine's defaults (forecast/src/engine.rs EngineConfig), and a year of history. */
export const DEFAULT_FORECAST_PARAMS: IForecastParams = {
	historyDays: 365,
	horizonDays: 28,
	partialCensorThreshold: 0,
	emptyShelfHalfLifeDays: 28,
	maxDamping: 0.9,
	shrinkWeeks: 8,
	intermittentAdi: 1.32,
}

export interface IForecastParamSlider {
	key: keyof IForecastParams
	label: string
	help: string
	min: number
	max: number
	step: number
	suffix?: string
	/** Changing it reads another range of the frames, not only a new fit. */
	reloadsData?: boolean
}

// No group titles: each label must be understandable on its own.
export const FORECAST_PARAM_SLIDERS: IForecastParamSlider[] = [
	{
		key: 'historyDays', label: 'History days|Días de historia', suffix: 'd', min: 90, max: 730, step: 1, reloadsData: true,
		help: 'Days of sales the models learn from. A year or more learns the yearly season and the calendar; two years trusts it more.|Días de ventas de los que aprenden los modelos. Un año o más aprende la estacionalidad anual y el calendario; dos años la afianzan.',
	},
	{
		key: 'horizonDays', label: 'Forecast horizon|Horizonte de pronóstico', suffix: 'd', min: 7, max: 90, step: 1,
		help: 'Days forecast from today.|Días pronosticados desde hoy.',
	},
	{
		key: 'partialCensorThreshold', label: 'Stockout threshold|Umbral de quiebre', suffix: ' u', min: 0, max: 20, step: 1,
		help: 'A day that closes at or below this stock had its demand capped: its sales count as a floor, not as the demand.|Un día que cierra con este stock o menos tuvo la demanda limitada: sus ventas cuentan como un mínimo, no como la demanda.',
	},
	{
		key: 'emptyShelfHalfLifeDays', label: 'Empty shelf fade|Desvanecer sin stock', suffix: 'd', min: 7, max: 90, step: 1,
		help: 'Days of empty shelf the forecast holds; after them it halves every as many days, for a product no longer sold.|Días sin stock que el pronóstico se mantiene; luego se reduce a la mitad cada tantos días, para un producto que ya no se vende.',
	},
	{
		key: 'maxDamping', label: 'Trend reach|Alcance de tendencia', min: 0.8, max: 0.98, step: 0.01,
		help: 'How far a rising or falling trend is carried into the future. Higher follows products that grow or fade; lower is safer on noisy sales.|Cuánto se prolonga una tendencia al alza o a la baja. Más alto sigue a productos que crecen o se apagan; más bajo es más seguro con ventas ruidosas.',
	},
	{
		key: 'shrinkWeeks', label: 'Weekly shape from category|Forma semanal de la categoría', suffix: ' sem', min: 0, max: 26, step: 1,
		help: "Weeks of history a product needs before its own weekday pattern weighs as much as its category's. 0 uses only the product's.|Semanas de historia que un producto necesita para que su patrón por día de semana pese como el de su categoría. 0 usa solo el del producto.",
	},
	{
		key: 'intermittentAdi', label: 'Intermittency threshold|Umbral de intermitencia', min: 1, max: 3, step: 0.05,
		help: 'A product selling on average less often than once every this many days is forecast as intermittent (TSB: chance of a sale × usual size).|Un producto que vende en promedio menos de una vez cada tantos días se pronostica como intermitente (TSB: probabilidad de venta × tamaño usual).',
	},
]

/** Parameters read back from storage: unknown keys dropped, missing or invalid ones at their default. */
export const parseForecastParams = (stored: unknown): IForecastParams => {
	const params = { ...DEFAULT_FORECAST_PARAMS }
	if (!stored || typeof stored !== 'object') return params
	for (const slider of FORECAST_PARAM_SLIDERS) {
		const value = (stored as Record<string, unknown>)[slider.key]
		if (typeof value === 'number' && value >= slider.min && value <= slider.max) params[slider.key] = value
	}
	return params
}

/** The engine's EngineConfig fields the sliders set (camelCase, the rest at its defaults). */
export const engineConfigOf = (params: IForecastParams) => ({
	partialCensorThreshold: params.partialCensorThreshold,
	emptyShelfHalfLifeDays: params.emptyShelfHalfLifeDays,
	maxDamping: params.maxDamping,
	shrinkWeeks: params.shrinkWeeks,
	intermittentAdi: params.intermittentAdi,
})

/**
 * The last days of the history are held out of the fit and fed day by day, as the engine would see
 * them live: that scores each series' forecasts against sales it never learned from (the accuracy
 * the table and the warnings show).
 */
export const HOLDOUT_DAYS = 28

export interface IForecastWindow {
	/** First day of history. */
	startDay: number
	/** Last day of the fit; the holdout days follow it. */
	fitEndDay: number
	/** Yesterday: today's sales are still coming in. */
	lastDay: number
}

export const forecastWindowOf = (today: number, historyDays: number): IForecastWindow => {
	const lastDay = today - 1
	return { startDay: lastDay - historyDays + 1, fitEndDay: lastDay - HOLDOUT_DAYS, lastDay }
}

// The fit needs some weeks before the holdout (the engine refuses fewer than 21 days).
export const MIN_FIT_DAYS = 28

/** The first day with a sale at a site, or undefined without any. */
export const firstSaleDayOf = (saleFiles: FrameFileRows[]): number | undefined => {
	let firstSaleDay: number | undefined
	for (const file of saleFiles) {
		if ((file.keys[SALE_KEY_SITE_ID] as number) === 0 || (file.sums[SALE_SUM_QUANTITY] as number[]).every((quantity) => quantity === 0)) continue
		const day = file.keys[SALE_KEY_DATE] as number
		if (firstSaleDay === undefined || day < firstSaleDay) firstSaleDay = day
	}
	return firstSaleDay
}

/**
 * The window starting no earlier than the company's first recorded sale. Before it the business was
 * not selling through Genix, so a day with stock on hand and no sale says nothing about demand: read
 * as demand, months of such zeros make every product look intermittent and fill its yearly profile
 * with empty weeks. A product that does not sell while others do still reads its zeros as demand.
 */
export const clipWindowToFirstSale = (window: IForecastWindow, firstSaleDay: number): IForecastWindow =>
	({ ...window, startDay: Math.max(window.startDay, firstSaleDay) })

/** One site and product as the engine reads it: index i is the day startDay + i. */
export interface IForecastSeries {
	siteID: number
	productID: number
	/** The product's first category, 0 when it has none: the pool its weekly and yearly shapes share. */
	categoryID: number
	/** Closing stock of the day before startDay. */
	initialStock: number
	sales: Float64Array
	closing: Float64Array
	inflow: Float64Array
	saleDays: number
	/** Days with nothing on hand and no sale, while the product's stock is tracked: hidden demand. */
	emptyShelfDays: number
	negativeStockDays: number
}

// The "sale-quantity" frame (backend sales/types SaleQuantityFrame): Keys [Date, SiteID], Rows
// [ProductID], Sums [SoldQuantity] in whole units (an int sum, not scaled).
const SALE_KEY_DATE = 0
const SALE_KEY_SITE_ID = 1
const SALE_ROW_PRODUCT_ID = 0
const SALE_SUM_QUANTITY = 0

export interface IForecastSeriesBuild {
	series: IForecastSeries[]
	/** Units sold with no site (a sale without warehouse): left out of every series. */
	unsitedSaleUnits: number
}

/**
 * buildForecastSeries assembles one series per site and product that sold in the window, with its
 * stock. stockSeries starts on stockFromDay (at most startDay − 1, the opening stock) and covers at
 * least lastDay. A product that never sold in the window is left out; one that sold with no stock
 * movement goes in untracked (all zeros), which the engine never censors.
 */
export const buildForecastSeries = (
	saleFiles: FrameFileRows[], stockSeries: SiteProductStockSeries[], stockFromDay: number, window: IForecastWindow,
	categoryIDOfProduct: (productID: number) => number,
): IForecastSeriesBuild => {
	const dayCount = window.lastDay - window.startDay + 1
	const salesBySiteProduct = new Map<string, { siteID: number, productID: number, sales: Float64Array }>()
	let unsitedSaleUnits = 0
	for (const file of saleFiles) {
		const dayIndex = (file.keys[SALE_KEY_DATE] as number) - window.startDay
		if (dayIndex < 0 || dayIndex >= dayCount) continue
		const siteID = file.keys[SALE_KEY_SITE_ID] as number
		const productIDs = file.rows[SALE_ROW_PRODUCT_ID] as number[]
		const quantities = file.sums[SALE_SUM_QUANTITY] as number[]
		for (let rowIndex = 0; rowIndex < productIDs.length; rowIndex++) {
			if (siteID === 0) { unsitedSaleUnits += quantities[rowIndex]; continue }
			const key = `${siteID}_${productIDs[rowIndex]}`
			let entry = salesBySiteProduct.get(key)
			if (!entry) {
				entry = { siteID, productID: productIDs[rowIndex], sales: new Float64Array(dayCount) }
				salesBySiteProduct.set(key, entry)
			}
			entry.sales[dayIndex] += quantities[rowIndex]
		}
	}

	const stockBySiteProduct = new Map(stockSeries.map((stock) => [`${stock.siteID}_${stock.productID}`, stock]))
	const series: IForecastSeries[] = []
	for (const [key, { siteID, productID, sales }] of salesBySiteProduct) {
		const stock = stockBySiteProduct.get(key)
		// Day i of the window sits at stockOffset + i of the stock series.
		const stockOffset = window.startDay - stockFromDay
		const closing = stock ? stock.closing.slice(stockOffset, stockOffset + dayCount) : new Float64Array(dayCount)
		const inflow = stock ? stock.inflow.slice(stockOffset, stockOffset + dayCount) : new Float64Array(dayCount)
		const initialStock = stock ? stock.closing[stockOffset - 1] : 0
		const stockTracked = initialStock > 0 || closing.some((value) => value > 0) || inflow.some((value) => value > 0)
		let saleDays = 0, emptyShelfDays = 0, negativeStockDays = 0
		for (let dayIndex = 0; dayIndex < dayCount; dayIndex++) {
			if (sales[dayIndex] > 0) saleDays++
			if (closing[dayIndex] < 0) negativeStockDays++
			if (stockTracked && sales[dayIndex] === 0 && closing[dayIndex] <= 0) emptyShelfDays++
		}
		series.push({
			siteID, productID, categoryID: categoryIDOfProduct(productID),
			initialStock, sales, closing, inflow, saleDays, emptyShelfDays, negativeStockDays,
		})
	}
	series.sort((left, right) => left.siteID - right.siteID || left.productID - right.productID)
	return { series, unsitedSaleUnits }
}

/** What the worker receives: the series series-major over [startDay, lastDay]. */
export interface IForecastRunRequest {
	runID: number
	window: IForecastWindow
	horizonDays: number
	config: ReturnType<typeof engineConfigOf>
	calendarDays: Int32Array
	calendarTypes: Uint8Array
	siteIDs: Int32Array
	productIDs: Int32Array
	categoryIDs: Int32Array
	initialStock: Float64Array
	sales: Float64Array
	closingStock: Float64Array
	inflow: Float64Array
}

/** The engine's answers (forecast/src/wasm.rs): keys, forecast, metrics and pendingRefits. */
export interface IForecastRunResult {
	keys: {
		lastDay: number, siteIds: number[], productIds: number[], categoryIds: number[],
		models: ('HoltWinters' | 'Tsb')[], scenarios: ('A' | 'B' | 'C')[], emptyShelfDays: number[],
	}
	forecast: { firstDay: number, horizon: number, values: number[] }
	/** Series-major, horizons 1 to 7 per series. */
	metrics: { count: number[], wape: number[], bias: number[], mase: number[] }
	pendingRefits: { series: number[], reasons: string[] }
	elapsedMs: number
}

export type IForecastWorkerAnswer = { runID: number, result: IForecastRunResult } | { runID: number, error: string }

export const packForecastRequest = (
	runID: number, series: IForecastSeries[], window: IForecastWindow, params: IForecastParams,
	calendarEvents: { Date: number, EventType: number }[],
): IForecastRunRequest => {
	const dayCount = window.lastDay - window.startDay + 1
	const seriesMajor = (valuesOf: (oneSeries: IForecastSeries) => Float64Array) => {
		const packed = new Float64Array(series.length * dayCount)
		series.forEach((oneSeries, seriesIndex) => packed.set(valuesOf(oneSeries), seriesIndex * dayCount))
		return packed
	}
	return {
		runID, window, horizonDays: params.horizonDays, config: engineConfigOf(params),
		calendarDays: Int32Array.from(calendarEvents.map((event) => event.Date)),
		calendarTypes: Uint8Array.from(calendarEvents.map((event) => event.EventType)),
		siteIDs: Int32Array.from(series.map((oneSeries) => oneSeries.siteID)),
		productIDs: Int32Array.from(series.map((oneSeries) => oneSeries.productID)),
		categoryIDs: Int32Array.from(series.map((oneSeries) => oneSeries.categoryID)),
		initialStock: Float64Array.from(series.map((oneSeries) => oneSeries.initialStock)),
		sales: seriesMajor((oneSeries) => oneSeries.sales),
		closingStock: seriesMajor((oneSeries) => oneSeries.closing),
		inflow: seriesMajor((oneSeries) => oneSeries.inflow),
	}
}

/** Days [fromIndex, toIndex) of every series of a series-major array of dayCount days each. */
export const sliceSeriesDays = (values: Float64Array, dayCount: number, fromIndex: number, toIndex: number): Float64Array => {
	const seriesCount = values.length / dayCount
	const width = toIndex - fromIndex
	const sliced = new Float64Array(seriesCount * width)
	for (let seriesIndex = 0; seriesIndex < seriesCount; seriesIndex++) {
		sliced.set(values.subarray(seriesIndex * dayCount + fromIndex, seriesIndex * dayCount + toIndex), seriesIndex * width)
	}
	return sliced
}

/** Weeks the table projects, as one column of daily bars per week. */
export const PROJECTION_WEEKS = 4

export interface IProjectionWeek {
	/** Forecast per day of the week; fewer than 7 (or none) when the horizon ends inside it. */
	days: number[]
	total: number
}

/** The forecast's first PROJECTION_WEEKS weeks, counted from its first day. */
export const projectionWeeksOf = (forecast: number[]): IProjectionWeek[] =>
	Array.from({ length: PROJECTION_WEEKS }, (_, weekIndex) => {
		const days = forecast.slice(weekIndex * 7, weekIndex * 7 + 7)
		return { days, total: days.reduce((total, value) => total + value, 0) }
	})

/** The weeks with every day and total multiplied by a unit price: the projection in money. */
export const projectionWeeksInMoney = (weeks: IProjectionWeek[], unitPrice: number): IProjectionWeek[] =>
	weeks.map((week) => ({ days: week.days.map((units) => units * unitPrice), total: week.total * unitPrice }))

/**
 * A product's current list price per unit, in soles (FinalPrice is in cents). A dollar price converts at
 * the latest sell rate (× 1000); undefined when there is no rate to convert it with.
 */
export const unitPriceInSoles = (product: Pick<IProduct, 'FinalPrice' | 'CurrencyID'>, sellRate: number): number | undefined => {
	const priceCurrency = normalizeCurrency(product.CurrencyID)
	if (priceCurrency !== CURRENCY_PEN && !sellRate) return undefined
	return convertUnitPrice(product.FinalPrice, priceCurrency, CURRENCY_PEN, sellRate) / 100
}

/** One table row: a series with its forecast and holdout accuracy. */
export interface IForecastRow {
	key: string
	/** 0 on a row merging every site of the product. */
	siteID: number
	productID: number
	categoryID: number
	/** Mixed: a merged row whose sites run different models (or scenarios). */
	model: 'HoltWinters' | 'Tsb' | 'Mixed'
	scenario: 'A' | 'B' | 'C' | 'Mixed'
	/** Forecast per day of the horizon. */
	forecast: number[]
	projectionWeeks: IProjectionWeek[]
	/** Units sold over the holdout (the last HOLDOUT_DAYS days). */
	lastSales: number
	stockToday: number
	/** Consecutive days of empty shelf up to yesterday. */
	emptyShelfDays: number
	/** One day ahead, over the holdout: |error| / sales. NaN without sales. */
	wape: number
	/** One day ahead, over the holdout: |error| / |seasonal naive error|; above 1 the naive did better. */
	mase: number
	worseThanNaive: boolean
}

export const forecastRowsOf = (series: IForecastSeries[], result: IForecastRunResult, stockTodayOf: (siteID: number, productID: number) => number): IForecastRow[] => {
	const seriesByKey = new Map(series.map((oneSeries) => [`${oneSeries.siteID}_${oneSeries.productID}`, oneSeries]))
	const worseThanNaivePositions = new Set(result.pendingRefits.series.filter((_, index) => result.pendingRefits.reasons[index] === 'WorseThanNaive'))
	const horizon = result.forecast.horizon
	return result.keys.siteIds.map((siteID, position) => {
		const productID = result.keys.productIds[position]
		const key = `${siteID}_${productID}`
		const forecast = result.forecast.values.slice(position * horizon, (position + 1) * horizon)
		const sales = seriesByKey.get(key)?.sales
		const lastSales = sales ? sales.subarray(sales.length - HOLDOUT_DAYS).reduce((total, value) => total + value, 0) : 0
		const mase = result.metrics.mase[position * 7]
		return {
			key, siteID, productID, categoryID: result.keys.categoryIds[position],
			model: result.keys.models[position], scenario: result.keys.scenarios[position],
			forecast,
			projectionWeeks: projectionWeeksOf(forecast),
			lastSales,
			stockToday: stockTodayOf(siteID, productID),
			emptyShelfDays: result.keys.emptyShelfDays[position],
			wape: result.metrics.wape[position * 7],
			mase,
			worseThanNaive: worseThanNaivePositions.has(position) || mase > 1,
		}
	})
}

/** Sales-weighted mean of a per-site accuracy ratio, skipping the sites where it is undefined. */
const salesWeightedRatio = (siteRows: IForecastRow[], ratioOf: (row: IForecastRow) => number): number => {
	let weightedSum = 0, salesSum = 0
	for (const row of siteRows) {
		const ratio = ratioOf(row)
		if (!Number.isFinite(ratio)) continue
		weightedSum += ratio * row.lastSales
		salesSum += row.lastSales
	}
	return salesSum > 0 ? weightedSum / salesSum : NaN
}

/**
 * rowsMergedByProduct folds the per-site rows into one row per product, for the view of every site.
 * Quantities add up; WAPE weighted by sales is the sites' total absolute error over their total sales,
 * and MASE is weighted the same way. A product is worse than naive if any of its sites is.
 */
export const rowsMergedByProduct = (siteRows: IForecastRow[]): IForecastRow[] => {
	const siteRowsByProduct = new Map<number, IForecastRow[]>()
	for (const row of siteRows) {
		const productSiteRows = siteRowsByProduct.get(row.productID)
		if (productSiteRows) productSiteRows.push(row)
		else siteRowsByProduct.set(row.productID, [row])
	}
	return [...siteRowsByProduct.values()].map((productSiteRows) => {
		const first = productSiteRows[0]
		const sumOf = (valueOf: (row: IForecastRow) => number) => productSiteRows.reduce((total, row) => total + valueOf(row), 0)
		const forecast = first.forecast.map((_, dayIndex) => sumOf((row) => row.forecast[dayIndex]))
		return {
			key: `0_${first.productID}`, siteID: 0, productID: first.productID, categoryID: first.categoryID,
			model: productSiteRows.every((row) => row.model === first.model) ? first.model : 'Mixed',
			scenario: productSiteRows.every((row) => row.scenario === first.scenario) ? first.scenario : 'Mixed',
			forecast,
			projectionWeeks: projectionWeeksOf(forecast),
			lastSales: sumOf((row) => row.lastSales),
			stockToday: sumOf((row) => row.stockToday),
			emptyShelfDays: Math.max(...productSiteRows.map((row) => row.emptyShelfDays)),
			wape: salesWeightedRatio(productSiteRows, (row) => row.wape),
			mase: salesWeightedRatio(productSiteRows, (row) => row.mase),
			worseThanNaive: productSiteRows.some((row) => row.worseThanNaive),
		}
	})
}

/** Below this confidence a forecast is flagged to the user. */
export const LOW_CONFIDENCE = 0.5

/**
 * forecastConfidence condenses the holdout accuracy into a 0–1 score, 1 the best: the mean of the
 * share of sales the forecast got right (1 − WAPE) and its edge over repeating last week (1 − MASE/2,
 * 0.5 when it ties the naive). Each part is clamped to 0–1; an undefined one is left out, both NaN.
 */
export const forecastConfidence = (wape: number, mase: number): number => {
	const parts = [1 - wape, 1 - mase / 2].filter((part) => Number.isFinite(part)).map((part) => Math.min(1, Math.max(0, part)))
	return parts.length > 0 ? parts.reduce((total, part) => total + part, 0) / parts.length : NaN
}

export interface IForecastWarning {
	group: 'history' | 'data' | 'calendar' | 'stockout' | 'accuracy'
	severity: 'high' | 'medium' | 'low'
	title: string
	detail: string
	/** The products (and sites) concerned, already named. */
	items: string[]
}

export interface IForecastWarningsInput {
	params: IForecastParams
	window: IForecastWindow
	series: IForecastSeries[]
	rows: IForecastRow[]
	calendarEvents: { Date: number, EventType: number, IsGlobal?: boolean }[]
	unsitedSaleUnits: number
	/** False when the stock kept moving while it was read: the rebuilt daily stock may be off. */
	stockConsistent: boolean
	seriesName: (siteID: number, productID: number) => string
	productExists: (productID: number) => boolean
	productHasCategory: (productID: number) => boolean
}

// A series with fewer sale days than this in the window gives the models too little to learn from.
const MIN_SALE_DAYS = 8
// A share of empty-shelf days from which the sales understate the demand noticeably.
const HIDDEN_DEMAND_SHARE = 0.2
// Below a year of history, not every week of the year has been seen: no yearly profile.
const YEAR_DAYS = 365
// Scenario B (yearly and calendar factors) needs this much history per series (forecast/src/engine.rs).
const SCENARIO_B_DAYS = 180

const yearOfDay = (unixDay: number) => new Date(unixDay * 86_400_000).getUTCFullYear()

/** Everything about the data or the fit the user should know before trusting the forecast. */
export const forecastWarnings = (input: IForecastWarningsInput): IForecastWarning[] => {
	const { params, window, series, rows } = input
	const warnings: IForecastWarning[] = []
	// A warning about some products (or years) only when there is one; the others are pushed directly.
	const pushListed = (warning: IForecastWarning) => { if (warning.items.length > 0) warnings.push(warning) }
	const namesOf = (matching: { siteID: number, productID: number }[]) => matching.map((one) => input.seriesName(one.siteID, one.productID))

	// History: the window may start later than asked, at the company's first recorded sale.
	const historyDays = window.lastDay - window.startDay + 1
	if (historyDays < params.historyDays) {
		warnings.push({
			group: 'history', severity: 'medium', items: [],
			title: 'Sales recorded for a shorter time|Ventas registradas por menos tiempo',
			detail: `The first sale is ${historyDays} days old, so the forecast learns from ${historyDays} days instead of the ${params.historyDays} asked.|La primera venta tiene ${historyDays} días, así que el pronóstico aprende de ${historyDays} días en lugar de los ${params.historyDays} pedidos.`,
		})
	}
	if (historyDays < YEAR_DAYS) {
		warnings.push({
			group: 'history', severity: 'medium', items: [],
			title: 'History shorter than a year|Historia menor a un año',
			detail: `With ${historyDays} days not every week of the year has been seen: no product gets the yearly season.|Con ${historyDays} días no se ha visto cada semana del año: ningún producto recibe la estacionalidad anual.`,
		})
	}
	// On a window shorter than scenario B, every product is in A: the warnings above already say so.
	pushListed({
		group: 'history', severity: 'medium', items: historyDays < SCENARIO_B_DAYS ? [] : namesOf(rows.filter((row) => row.scenario === 'A')),
		title: 'Products with little history|Productos con poca historia',
		detail: `Less than ${SCENARIO_B_DAYS} days since their first sale or stock: forecast without yearly season or calendar effects.|Menos de ${SCENARIO_B_DAYS} días desde su primera venta o stock: pronóstico sin estacionalidad anual ni efectos de calendario.`,
	})
	pushListed({
		group: 'history', severity: 'low', items: namesOf(series.filter((oneSeries) => oneSeries.saleDays < MIN_SALE_DAYS)),
		title: 'Products with few sales|Productos con pocas ventas',
		detail: `Fewer than ${MIN_SALE_DAYS} days with sales in the history: their forecast is little more than an average.|Menos de ${MIN_SALE_DAYS} días con ventas en la historia: su pronóstico es poco más que un promedio.`,
	})

	// Data.
	if (input.unsitedSaleUnits > 0) {
		warnings.push({
			group: 'data', severity: 'high', items: [],
			title: 'Sales without a site|Ventas sin sede',
			detail: `${input.unsitedSaleUnits} units were sold without a warehouse, so they belong to no site and are left out of the forecast.|${input.unsitedSaleUnits} unidades se vendieron sin almacén: no pertenecen a ninguna sede y quedan fuera del pronóstico.`,
		})
	}
	if (!input.stockConsistent) {
		warnings.push({
			group: 'data', severity: 'high', items: [],
			title: 'Stock changed while it was read|El stock cambió mientras se leía',
			detail: 'Stock movements kept arriving while the page read the stock: the daily stock rebuilt from it may be off. Reload the page.|Siguieron llegando movimientos de stock mientras la página leía el stock: el stock diario reconstruido puede estar desfasado. Recargue la página.',
		})
	}
	pushListed({
		group: 'data', severity: 'medium', items: namesOf(series.filter((oneSeries) => oneSeries.negativeStockDays > 0)),
		title: 'Negative stock|Stock negativo',
		detail: 'Sold without stock on record: receipts are registered late or missing. Those days read as demand, never as stockouts.|Se vendió sin stock registrado: los ingresos se registran tarde o faltan. Esos días cuentan como demanda, nunca como quiebre.',
	})
	pushListed({
		group: 'data', severity: 'medium', items: namesOf(series.filter((oneSeries) => !input.productHasCategory(oneSeries.productID))),
		title: 'Products without a category|Productos sin categoría',
		detail: 'They share no weekly or yearly shape with similar products, and pool together as one category.|No comparten forma semanal ni anual con productos similares, y se agrupan todos como una sola categoría.',
	})
	pushListed({
		group: 'data', severity: 'low', items: namesOf(series.filter((oneSeries) => !input.productExists(oneSeries.productID))),
		title: 'Sold products not in the catalog|Productos vendidos fuera del catálogo',
		detail: 'They sold in the history but are deleted or not synced: they are forecast under their ID.|Se vendieron en la historia pero están eliminados o sin sincronizar: se pronostican con su ID.',
	})

	// Calendar.
	const activeEvents = input.calendarEvents
	const firstYear = yearOfDay(window.startDay)
	const lastYear = yearOfDay(window.lastDay + params.horizonDays)
	const yearsWithHolidays = new Set(activeEvents.filter((event) => event.EventType === 1).map((event) => yearOfDay(event.Date)))
	const yearsWithoutHolidays: string[] = []
	for (let year = firstYear; year <= lastYear; year++) if (!yearsWithHolidays.has(year)) yearsWithoutHolidays.push(String(year))
	pushListed({
		group: 'calendar', severity: 'high', items: yearsWithoutHolidays,
		title: 'Years without holidays|Años sin feriados',
		detail: 'The commercial calendar has no holidays for these years of the history or the horizon: their holidays read as ordinary days.|El calendario comercial no tiene feriados en estos años de la historia o del horizonte: sus feriados se leen como días normales.',
	})
	const paydaysInRange = activeEvents.some((event) => event.EventType === 2 && event.Date >= window.startDay && event.Date <= window.lastDay + params.horizonDays)
	if (!paydaysInRange) {
		warnings.push({
			group: 'calendar', severity: 'medium', items: [],
			title: 'No paydays|Sin días de pago',
			detail: 'No payday is loaded in the commercial calendar for the history or the horizon: the sales peaks around them are not anticipated.|No hay días de pago en el calendario comercial para la historia ni el horizonte: los picos de venta alrededor de ellos no se anticipan.',
		})
	}
	if (!activeEvents.some((event) => !event.IsGlobal)) {
		warnings.push({
			group: 'calendar', severity: 'low', items: [],
			title: 'No company events|Sin eventos de la empresa',
			detail: 'Only the national holidays are loaded. Add your paydays, campaigns and commercial dates in Catalog › Commercial Calendar.|Solo están los feriados nacionales. Agregue sus días de pago, campañas y fechas comerciales en Catálogo › Calendario Comercial.',
		})
	}

	// Stockouts.
	pushListed({
		group: 'stockout', severity: 'high', items: rows.filter((row) => row.emptyShelfDays > 0).map((row) => `${input.seriesName(row.siteID, row.productID)} · ${row.emptyShelfDays} d`),
		title: 'Out of stock now|Sin stock ahora',
		detail: `Empty shelf up to yesterday (days). Past ${params.emptyShelfHalfLifeDays} days their forecast starts to fade, as for a product no longer sold.|Sin stock hasta ayer (días). Pasados ${params.emptyShelfHalfLifeDays} días su pronóstico empieza a reducirse, como el de un producto que ya no se vende.`,
	})
	const dayCount = window.lastDay - window.startDay + 1
	pushListed({
		group: 'stockout', severity: 'medium',
		items: series.filter((oneSeries) => oneSeries.emptyShelfDays / dayCount >= HIDDEN_DEMAND_SHARE)
			.map((oneSeries) => `${input.seriesName(oneSeries.siteID, oneSeries.productID)} · ${Math.round(100 * oneSeries.emptyShelfDays / dayCount)}%`),
		title: 'Frequent stockouts|Quiebres frecuentes',
		detail: `Without stock on ${Math.round(HIDDEN_DEMAND_SHARE * 100)}% or more of the days: their sales understate the demand. The engine skips those days, but has fewer to learn from.|Sin stock en el ${Math.round(HIDDEN_DEMAND_SHARE * 100)}% o más de los días: sus ventas subestiman la demanda. El motor omite esos días, pero aprende de menos.`,
	})

	// Accuracy.
	pushListed({
		group: 'accuracy', severity: 'medium', items: namesOf(rows.filter((row) => row.worseThanNaive)),
		title: 'Worse than repeating last week|Peor que repetir la semana pasada',
		detail: `Over the last ${HOLDOUT_DAYS} days, held out of the fit, the forecast missed by more than repeating the same weekday of the previous week. Review their parameters or their data.|En los últimos ${HOLDOUT_DAYS} días, fuera del ajuste, el pronóstico erró más que repetir el mismo día de la semana anterior. Revise sus parámetros o sus datos.`,
	})

	const severityOrder = { high: 0, medium: 1, low: 2 }
	return warnings.sort((left, right) => severityOrder[left.severity] - severityOrder[right.severity])
}

/** The chart of a set of rows added up: their sales and stock over the last days, then their forecast. */
export interface IForecastChart {
	days: number[]
	sales: (number | null)[]
	stock: (number | null)[]
	forecast: (number | null)[]
}

export const CHART_HISTORY_DAYS = 56

export const forecastChartOf = (rows: IForecastRow[], seriesByKey: Map<string, IForecastSeries>, window: IForecastWindow, horizon: number): IForecastChart => {
	const historyDays = Math.min(CHART_HISTORY_DAYS, window.lastDay - window.startDay + 1)
	const totalDays = historyDays + horizon
	const chart: IForecastChart = {
		days: Array.from({ length: totalDays }, (_, index) => window.lastDay - historyDays + 1 + index),
		sales: new Array(totalDays).fill(null), stock: new Array(totalDays).fill(null), forecast: new Array(totalDays).fill(null),
	}
	for (let index = 0; index < historyDays; index++) { chart.sales[index] = 0; chart.stock[index] = 0 }
	for (let index = historyDays; index < totalDays; index++) chart.forecast[index] = 0
	for (const row of rows) {
		const series = seriesByKey.get(row.key)
		if (series) {
			const offset = series.sales.length - historyDays
			for (let index = 0; index < historyDays; index++) {
				(chart.sales[index] as number) += series.sales[offset + index];
				(chart.stock[index] as number) += Math.max(0, series.closing[offset + index])
			}
		}
		for (let index = 0; index < horizon; index++) (chart.forecast[historyDays + index] as number) += row.forecast[index] || 0
	}
	return chart
}
