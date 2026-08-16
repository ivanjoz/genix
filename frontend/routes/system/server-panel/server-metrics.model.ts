export const SLOT_SECONDS = 5
export const SLOTS_PER_HOUR = 3600 / SLOT_SECONDS
export const WINDOW_HOURS = 4
export const WINDOW_SLOTS = WINDOW_HOURS * SLOTS_PER_HOUR
// One rendered point per eight slots, so four hours become 360 points. Above that the canvas is
// drawing several points per pixel and the extra detail is invisible.
export const SLOTS_PER_POINT = 8
export const CHART_POINTS = WINDOW_SLOTS / SLOTS_PER_POINT

// -1 is the daemon's "not measured": no cgroup for that unit, which is the normal state of the
// Backend* columns when the backend runs on Lambda. It must never reach a chart as a number.
const NOT_MEASURED = -1

export type ServerMetricField =
	| 'CpuPercent' | 'MemPercent' | 'DiskPercent' | 'NetRxRate' | 'NetTxRate'
	| 'BackendMemMb' | 'BackendCpuPercent'
	| 'ServerUtilsMemMb' | 'ServerUtilsCpuPercent'
	| 'SearchMemMb' | 'SearchCpuPercent'
	| 'ScyllaMemMb' | 'ScyllaCpuPercent'

export const SERVER_METRIC_FIELDS: ServerMetricField[] = [
	'CpuPercent', 'MemPercent', 'DiskPercent', 'NetRxRate', 'NetTxRate',
	'BackendMemMb', 'BackendCpuPercent', 'ServerUtilsMemMb', 'ServerUtilsCpuPercent',
	'SearchMemMb', 'SearchCpuPercent', 'ScyllaMemMb', 'ScyllaCpuPercent',
]

// How each stored int16 becomes the number a chart shows. Percentages are hundredths (2345 is
// 23.45%) and network rates are 5 KB/s units, both chosen by the daemon to fit an int16.
const METRIC_DISPLAY_SCALE: Record<ServerMetricField, number> = {
	CpuPercent: 0.01, MemPercent: 0.01, DiskPercent: 0.01,
	NetRxRate: 5, NetTxRate: 5,
	BackendMemMb: 1, BackendCpuPercent: 0.01,
	ServerUtilsMemMb: 1, ServerUtilsCpuPercent: 0.01,
	SearchMemMb: 1, SearchCpuPercent: 0.01,
	ScyllaMemMb: 1, ScyllaCpuPercent: 0.01,
}

// One hour of samples, the unit the backend records and the cache keys on. Every array is parallel
// to SlotsInHour, which is also the field the columnar delta merge aligns on.
export interface IServerMetricsHour extends Record<ServerMetricField, number[]> {
	ID: number
	SlotsInHour: number[]
	upd: number
	ss: number
}

export interface IServerMetricsResponse {
	Hours?: IServerMetricsHour[]
}

// What the charts read: one timestamp per point and one value array per metric, already in display
// units. `null` is a gap the chart must break the line on — either an unmeasured service or a slot
// the daemon never wrote.
export interface IServerMetricsSeries {
	timestamps: number[]
	values: Record<ServerMetricField, Array<number | null>>
	latest: Record<ServerMetricField, number | null>
	sampledSlots: number
}

// One entry per metric, built from the field list so a metric added there cannot be forgotten here.
const makeMetricRecord = <V>(makeInitialValue: () => V): Record<ServerMetricField, V> => {
	const metricRecord = {} as Record<ServerMetricField, V>
	for (const metricField of SERVER_METRIC_FIELDS) { metricRecord[metricField] = makeInitialValue() }
	return metricRecord
}

const makeEmptySeries = (): IServerMetricsSeries => ({
	timestamps: [],
	values: makeMetricRecord<Array<number | null>>(() => []),
	latest: makeMetricRecord<number | null>(() => null),
	sampledSlots: 0,
})

/**
 * Flattens the hour buckets into one dense, evenly-spaced series and reduces it to CHART_POINTS.
 *
 * The window ends at `nowUnixSeconds`, not at the newest sample received: a panel whose right edge
 * is always the last stored row would look healthy while the daemon was down, when the whole point
 * of the chart is to make that gap visible.
 *
 * Points are reduced by MAXIMUM. Every stored value is already the peak of its five seconds
 * (server_utils/PLAN_SERVER_METRICS.md), so max-of-peaks is still a peak, while an average would
 * invent a number that never happened and hide the spikes the table exists to record.
 */
export const buildServerMetricsSeries = (
	hourBuckets: IServerMetricsHour[],
	nowUnixSeconds: number,
): IServerMetricsSeries => {
	const series = makeEmptySeries()
	const newestSlot = Math.floor(nowUnixSeconds / SLOT_SECONDS)
	const firstSlot = newestSlot - WINDOW_SLOTS + 1

	const denseValues = makeMetricRecord<Array<number | null>>(
		() => new Array<number | null>(WINDOW_SLOTS).fill(null),
	)

	for (const hourBucket of hourBuckets) {
		const slotsInHour = hourBucket?.SlotsInHour || []
		for (let sampleIndex = 0; sampleIndex < slotsInHour.length; sampleIndex++) {
			const absoluteSlot = (hourBucket.ID * SLOTS_PER_HOUR) + slotsInHour[sampleIndex]
			const denseIndex = absoluteSlot - firstSlot
			if (denseIndex < 0 || denseIndex >= WINDOW_SLOTS) { continue }
			// Counted per slot, not per field: a Lambda deployment leaves every Backend* column
			// unmeasured, and that is a missing service rather than a missing sample.
			series.sampledSlots++

			for (const metricField of SERVER_METRIC_FIELDS) {
				const storedValue = hourBucket[metricField]?.[sampleIndex]
				if (storedValue === undefined || storedValue === null || storedValue === NOT_MEASURED) { continue }
				denseValues[metricField][denseIndex] = storedValue
			}
		}
	}

	for (let pointIndex = 0; pointIndex < CHART_POINTS; pointIndex++) {
		const groupFirstSlot = pointIndex * SLOTS_PER_POINT
		series.timestamps.push((firstSlot + groupFirstSlot) * SLOT_SECONDS)

		for (const metricField of SERVER_METRIC_FIELDS) {
			let groupPeak: number | null = null
			for (let slotOffset = 0; slotOffset < SLOTS_PER_POINT; slotOffset++) {
				const slotValue = denseValues[metricField][groupFirstSlot + slotOffset]
				if (slotValue === null) { continue }
				if (groupPeak === null || slotValue > groupPeak) { groupPeak = slotValue }
			}
			// Scaling after the reduction is safe because every scale is positive, so it cannot
			// reorder what the maximum picked.
			const displayValue = groupPeak === null ? null : groupPeak * METRIC_DISPLAY_SCALE[metricField]
			series.values[metricField].push(displayValue)
			if (displayValue !== null) { series.latest[metricField] = displayValue }
		}
	}

	return series
}
