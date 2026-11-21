// Runs the forecast engine (forecast/, built by forecast/build-wasm.sh) off the main thread: a fit
// optimizes every series' parameters and takes seconds on a large catalog.
import initForecastEngine, {
	create, forecast, keys, metrics, pendingRefits, refit, setCalendar, update,
} from '#libs/genix-forecast/genix_forecast.js'
import { sliceSeriesDays, type IForecastRunRequest, type IForecastWorkerAnswer } from './sales-forecast.ts'

const engineReady = initForecastEngine()

/**
 * Fits on the history up to fitEndDay, then feeds the held-out days one batch at a time as the live
 * page would: the engine scores its forecasts on them (metrics) and ends on yesterday, from where it
 * forecasts the horizon.
 */
const runForecast = (request: IForecastRunRequest) => {
	const { window } = request
	const dayCount = window.lastDay - window.startDay + 1
	const fitDayCount = window.fitEndDay - window.startDay + 1

	let state = create(window.fitEndDay, request.config)
	state = setCalendar(state, request.calendarDays, request.calendarTypes)
	state = refit(state, window.startDay, request.siteIDs, request.productIDs, request.categoryIDs, request.initialStock,
		sliceSeriesDays(request.sales, dayCount, 0, fitDayCount),
		sliceSeriesDays(request.closingStock, dayCount, 0, fitDayCount),
		sliceSeriesDays(request.inflow, dayCount, 0, fitDayCount))
	// A new state appends the refit's series in batch order, the order update reads.
	state = update(state, window.fitEndDay + 1,
		sliceSeriesDays(request.sales, dayCount, fitDayCount, dayCount),
		sliceSeriesDays(request.closingStock, dayCount, fitDayCount, dayCount),
		sliceSeriesDays(request.inflow, dayCount, fitDayCount, dayCount))
	return {
		keys: keys(state),
		forecast: forecast(state, request.horizonDays),
		metrics: metrics(state),
		pendingRefits: pendingRefits(state),
	}
}

self.onmessage = async (event: MessageEvent<IForecastRunRequest>) => {
	const request = event.data
	let answer: IForecastWorkerAnswer
	try {
		await engineReady
		const startedAt = performance.now()
		const result = runForecast(request)
		answer = { runID: request.runID, result: { ...result, elapsedMs: performance.now() - startedAt } }
	} catch (runError) {
		console.error('[sales-forecast.worker] run failed', { runID: request.runID, runError })
		answer = { runID: request.runID, error: runError instanceof Error ? runError.message : String(runError) }
	}
	self.postMessage(answer)
}
