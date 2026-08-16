import { describe, expect, it } from 'bun:test'
import {
	buildServerMetricsSeries,
	CHART_POINTS,
	SERVER_METRIC_FIELDS,
	SLOTS_PER_HOUR,
	SLOTS_PER_POINT,
	SLOT_SECONDS,
	WINDOW_SLOTS,
	type IServerMetricsHour,
} from './server-metrics.model'

// An hour bucket with every array parallel to the given slot offsets. Metrics not named are left
// as -1, the daemon's "not measured", which is what an absent service actually writes.
const makeHourBucket = (
	bucketID: number,
	slotsInHour: number[],
	measuredValues: Partial<Record<(typeof SERVER_METRIC_FIELDS)[number], number[]>>,
): IServerMetricsHour => {
	const hourBucket = { ID: bucketID, SlotsInHour: slotsInHour, upd: 0, ss: 1 } as IServerMetricsHour
	for (const metricField of SERVER_METRIC_FIELDS) {
		hourBucket[metricField] = measuredValues[metricField] ?? slotsInHour.map(() => -1)
	}
	return hourBucket
}

// The unix second at which `bucketID` is the newest complete hour, so a sample at slot offset N of
// that bucket lands inside the window.
const makeNowForBucket = (bucketID: number) => (bucketID + 1) * 3600 - SLOT_SECONDS

describe('buildServerMetricsSeries', () => {
	it('reduces the window to a fixed number of points regardless of what arrived', () => {
		// The chart's x-axis is the window, not the data: a daemon that wrote nothing must still
		// produce a full-width series of gaps rather than a short chart of a different scale.
		const series = buildServerMetricsSeries([], makeNowForBucket(492_000))

		expect(series.timestamps.length).toBe(CHART_POINTS)
		expect(series.values.CpuPercent.length).toBe(CHART_POINTS)
		expect(series.values.CpuPercent.every((pointValue) => pointValue === null)).toBe(true)
		expect(series.sampledSlots).toBe(0)
	})

	it('converts stored units into what the chart shows', () => {
		const bucketID = 492_010
		const lastSlotOffset = SLOTS_PER_HOUR - 1
		const series = buildServerMetricsSeries(
			[makeHourBucket(bucketID, [lastSlotOffset], {
				CpuPercent: [2345], ScyllaMemMb: [620], NetRxRate: [3],
			})],
			makeNowForBucket(bucketID),
		)

		// Hundredths of a percent become percent, and 5 KB/s units become KB/s.
		expect(series.latest.CpuPercent).toBeCloseTo(23.45, 5)
		expect(series.latest.ScyllaMemMb).toBe(620)
		expect(series.latest.NetRxRate).toBe(15)
	})

	it('keeps the peak of each group rather than averaging it away', () => {
		// Every stored value is already the peak of its five seconds, so a spike in one slot has to
		// survive the reduction; an average would dilute it by the seven quiet slots beside it.
		const bucketID = 492_020
		const firstSlotOfLastGroup = SLOTS_PER_HOUR - SLOTS_PER_POINT
		const groupSlots = Array.from({ length: SLOTS_PER_POINT }, (_, index) => firstSlotOfLastGroup + index)
		const cpuValues = groupSlots.map(() => 100)
		cpuValues[3] = 9_000

		const series = buildServerMetricsSeries(
			[makeHourBucket(bucketID, groupSlots, { CpuPercent: cpuValues })],
			makeNowForBucket(bucketID),
		)

		expect(series.values.CpuPercent[CHART_POINTS - 1]).toBeCloseTo(90, 5)
	})

	it('renders an unmeasured service as a gap and never as zero', () => {
		// The Backend* columns are -1 on every Lambda deployment. Charting that as 0% would show a
		// backend sitting idle when there is no backend on the box at all.
		const bucketID = 492_030
		const lastSlotOffset = SLOTS_PER_HOUR - 1
		const series = buildServerMetricsSeries(
			[makeHourBucket(bucketID, [lastSlotOffset], { CpuPercent: [1000] })],
			makeNowForBucket(bucketID),
		)

		expect(series.values.BackendCpuPercent[CHART_POINTS - 1]).toBe(null)
		expect(series.latest.BackendCpuPercent).toBe(null)
		expect(series.latest.CpuPercent).toBeCloseTo(10, 5)
	})

	it('drops samples older than the window instead of shifting the axis', () => {
		// A stale bucket the cache has not evicted yet must not push newer points off the chart.
		const currentBucket = 492_040
		const nowUnixSeconds = makeNowForBucket(currentBucket)
		const staleBucket = currentBucket - 10

		const series = buildServerMetricsSeries(
			[
				makeHourBucket(staleBucket, [0], { CpuPercent: [9_999] }),
				makeHourBucket(currentBucket, [SLOTS_PER_HOUR - 1], { CpuPercent: [1_500] }),
			],
			nowUnixSeconds,
		)

		expect(series.sampledSlots).toBe(1)
		expect(series.latest.CpuPercent).toBeCloseTo(15, 5)
	})

	it('anchors the right edge on now so a stopped daemon shows a growing gap', () => {
		// The newest sample is a full hour old. The series must still end at now, leaving the tail
		// empty, because a chart that ends at the last row would look healthy while nothing ran.
		const currentBucket = 492_050
		const nowUnixSeconds = makeNowForBucket(currentBucket)
		const newestSlot = Math.floor(nowUnixSeconds / SLOT_SECONDS)
		const oneHourAgoBucket = currentBucket - 1

		const series = buildServerMetricsSeries(
			[makeHourBucket(oneHourAgoBucket, [SLOTS_PER_HOUR - 1], { CpuPercent: [1_000] })],
			nowUnixSeconds,
		)

		expect(series.timestamps[CHART_POINTS - 1]).toBe((newestSlot - SLOTS_PER_POINT + 1) * SLOT_SECONDS)
		expect(series.values.CpuPercent[CHART_POINTS - 1]).toBe(null)
		expect(series.values.CpuPercent.filter((pointValue) => pointValue !== null).length).toBe(1)
	})

	it('spans exactly the configured window', () => {
		const bucketID = 492_060
		const nowUnixSeconds = makeNowForBucket(bucketID)
		const series = buildServerMetricsSeries([], nowUnixSeconds)

		const spannedSeconds = series.timestamps[CHART_POINTS - 1] - series.timestamps[0]
		expect(spannedSeconds).toBe((WINDOW_SLOTS - SLOTS_PER_POINT) * SLOT_SECONDS)
	})
})
