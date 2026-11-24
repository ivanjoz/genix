// Runs the forecast engine (forecast/, built by forecast/build-wasm.sh) off the main thread for the
// cash flow's Forecast layer, and keeps its state in IndexedDB per company (plan §6): a visit updates
// the stored state with the days since its last one and fits again only the series that need it,
// instead of fitting a year of every series again.
import initForecastEngine, {
  create, forecast, keys, pendingRefits, refit, setCalendar, update,
} from '#libs/genix-forecast/genix_forecast.js'
import { readIndexedDBValue, writeIndexedDBValue } from '#libs/indexeddb-value.ts'
import {
  changeCheckSpan, ENGINE_STATE_VERSION, hashSeriesDays, planEngineRun, seriesKeyOf,
  type ICashFlowSeries, type IEngineRunRequest, type IEngineWorkerAnswer, type IStoredEngineState,
} from './cash-flow-forecast.ts'

const ENGINE_DATABASE = 'genix-cash-flow-engine'
const engineReady = initForecastEngine()

const runEngine = async (request: IEngineRunRequest) => {
  const { window } = request
  const dayCount = window.lastDay - window.startDay + 1
  const series: ICashFlowSeries[] = request.keys.map((key, index) => ({
    key, siteID: request.siteIDs[index], productID: request.productIDs[index], categoryID: request.categoryIDs[index],
    values: request.values.subarray(index * dayCount, (index + 1) * dayCount),
  }))
  const seriesByKey = new Map(series.map((oneSeries) => [oneSeries.key, oneSeries]))

  let stored: IStoredEngineState | undefined
  try {
    stored = await readIndexedDBValue<IStoredEngineState>(ENGINE_DATABASE, request.storageKey)
  } catch (readError) {
    console.warn('[cash-flow.worker] stored state unreadable, fitting from scratch', readError)
  }
  const plan = planEngineRun(stored, series, window)

  // Fits the series again over [startDay, toDay], toDay being the state's last day; new ones are appended.
  const refitSeries = (state: Uint8Array, chosen: ICashFlowSeries[], toDay: number) => {
    if (chosen.length === 0) return state
    const width = toDay - window.startDay + 1
    const values = new Float64Array(chosen.length * width)
    chosen.forEach((oneSeries, index) => values.set(oneSeries.values.subarray(0, width), index * width))
    const zeros = new Float64Array(chosen.length * width)
    return refit(state, window.startDay, Int32Array.from(chosen.map((oneSeries) => oneSeries.siteID)),
      Int32Array.from(chosen.map((oneSeries) => oneSeries.productID)), Int32Array.from(chosen.map((oneSeries) => oneSeries.categoryID)),
      new Float64Array(chosen.length), values, zeros, zeros)
  }
  // The state's series in its order; one no longer in the data (a category nobody spends on now) reads zeros.
  const seriesOfState = (state: Uint8Array): ICashFlowSeries[] => {
    const stateKeys = keys(state)
    return (stateKeys.siteIds as number[]).map((siteID, position) => {
      const productID = stateKeys.productIds[position]
      return seriesByKey.get(seriesKeyOf(siteID, productID))
        ?? { key: seriesKeyOf(siteID, productID), siteID, productID, categoryID: stateKeys.categoryIds[position], values: new Float64Array(dayCount) }
    })
  }
  const pendingSeriesOf = (state: Uint8Array) => {
    const stateSeries = seriesOfState(state)
    return [...new Set(pendingRefits(state).series as number[])].map((position) => stateSeries[position])
  }

  let state: Uint8Array
  if (plan.mode === 'full' || !stored) {
    state = create(window.lastDay, {})
    state = setCalendar(state, request.calendarDays, request.calendarTypes)
    state = refitSeries(state, series, window.lastDay)
  } else {
    // Calendar first: a changed event on a learned day asks its series for a refit.
    state = setCalendar(stored.state, request.calendarDays, request.calendarTypes)
    const refitKeys = new Set([...plan.refitKeys, ...pendingSeriesOf(state).map((oneSeries) => oneSeries.key)])
    state = refitSeries(state, [...refitKeys].flatMap((key) => seriesByKey.get(key) || []), stored.lastDay)
    if (stored.lastDay < window.lastDay) {
      const fromIndex = stored.lastDay + 1 - window.startDay
      const stateSeries = seriesOfState(state)
      const width = dayCount - fromIndex
      const values = new Float64Array(stateSeries.length * width)
      stateSeries.forEach((oneSeries, index) => values.set(oneSeries.values.subarray(fromIndex, dayCount), index * width))
      const zeros = new Float64Array(stateSeries.length * width)
      state = update(state, stored.lastDay + 1, values, zeros, zeros)
      // The new days may fire a refit trigger (a model switch, a new period).
      state = refitSeries(state, pendingSeriesOf(state), window.lastDay)
    }
  }

  const engineForecast = forecast(state, request.horizonDays)
  const finalSeries = seriesOfState(state)
  const { fromIndex, toIndex } = changeCheckSpan(window.lastDay, window.startDay)
  const toStore: IStoredEngineState = {
    version: ENGINE_STATE_VERSION, lastDay: window.lastDay, state,
    keys: finalSeries.map((oneSeries) => oneSeries.key),
    recentHashes: finalSeries.map((oneSeries) => hashSeriesDays(oneSeries.values, fromIndex, toIndex)),
  }
  await writeIndexedDBValue(ENGINE_DATABASE, request.storageKey, toStore)
    .catch((writeError) => console.warn('[cash-flow.worker] state not saved', writeError))
  return {
    mode: plan.mode,
    forecast: {
      siteIDs: finalSeries.map((oneSeries) => oneSeries.siteID), productIDs: finalSeries.map((oneSeries) => oneSeries.productID),
      firstDay: engineForecast.firstDay, horizon: engineForecast.horizon, values: Array.from(engineForecast.values as ArrayLike<number>),
    },
  }
}

self.onmessage = async (event: MessageEvent<IEngineRunRequest>) => {
  const request = event.data
  let answer: IEngineWorkerAnswer
  try {
    await engineReady
    const startedAt = performance.now()
    const result = await runEngine(request)
    console.debug('[cash-flow.worker] forecast ready', { mode: result.mode, series: result.forecast.siteIDs.length, ms: performance.now() - startedAt })
    answer = { runID: request.runID, ...result }
  } catch (runError) {
    console.error('[cash-flow.worker] run failed', { runID: request.runID, runError })
    answer = { runID: request.runID, error: runError instanceof Error ? runError.message : String(runError) }
  }
  self.postMessage(answer)
}
