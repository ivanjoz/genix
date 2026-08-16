import { GetHandler } from '$libs/ui-runtime.svelte'
import {
	SERVER_METRIC_FIELDS,
	WINDOW_HOURS,
	type IServerMetricsHour,
	type IServerMetricsResponse,
} from './server-metrics.model'

/**
 * Reads the four-hour window from `GET.server-metrics` as a delta.
 *
 * A sealed hour can never gain a sample, so once a bucket is cached it is never fetched again: the
 * cold load carries 2880 slots and every refresh after it carries only the handful recorded since.
 * That is what `columnarIDField`/`combineColumnarValuesOnFields` buy — the merge aligns incoming
 * values onto the cached arrays by slot offset instead of replacing the record.
 */
export class ServerMetricsService extends GetHandler {
	route = `server-metrics?hours=${WINDOW_HOURS}`
	// Under the 15 s refresh the view drives, so a scheduled refresh is never skipped by the TTL.
	useCache = { min: 0.2, ver: 1 }
	keyID = 'ID'
	columnarIDField = 'SlotsInHour'
	combineColumnarValuesOnFields = SERVER_METRIC_FIELDS

	hourBuckets: IServerMetricsHour[] = $state([])

	handler(response: IServerMetricsResponse): void {
		// Ascending by hour so the flattening walks time forwards; the cache returns whatever order
		// the delta happened to arrive in.
		this.hourBuckets = [...(response?.Hours || [])].sort(
			(leftBucket, rightBucket) => leftBucket.ID - rightBucket.ID,
		)
		console.debug('[ServerMetricsService] hour buckets:', this.hourBuckets.length)
	}

	constructor(init: boolean = false) {
		super()
		if (init) { this.fetch() }
	}
}
