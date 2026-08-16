import { getStaticRecordsByID } from '@genix/ui/cache'
import { GetHandler } from '$libs/ui-runtime.svelte'
import {
	buildObservabilityCards,
	collectObservabilityErrorIDs,
	OBSERVABILITY_WINDOW_HOURS,
	type IObservabilityFrame,
	type IObservabilityResponse,
	type IObservabilityRoute,
	type IRequestErrorByID,
} from './observability.model'

export class ObservabilityService extends GetHandler {
	route = `observability?hours=${OBSERVABILITY_WINDOW_HOURS}`
	useCache = { min: 0.2, ver: 1 }
	keyID = 'ID'

	frames: IObservabilityFrame[] = $state([])
	routes: IObservabilityRoute[] = $state([])
	errorRecords: Map<number, IRequestErrorByID> = $state(new Map())
	private pendingErrorIDs = new Set<number>()

	handler(response: IObservabilityResponse): void {
		this.frames = [...(response?.Frames || [])].sort((left, right) => left.ID - right.ID)
		this.routes = [...(response?.Routes || [])]
		void this.resolveErrorRecords()
		console.debug('[ObservabilityService] merged:', {
			frames: this.frames.length,
			routes: this.routes.length,
		})
	}

	private async resolveErrorRecords(): Promise<void> {
		const errorIDs = collectObservabilityErrorIDs(buildObservabilityCards(this.frames, this.routes))
		const missingIDs = errorIDs.filter(errorID => !this.errorRecords.has(errorID) && !this.pendingErrorIDs.has(errorID))
		if (missingIDs.length === 0) return
		for (const errorID of missingIDs) this.pendingErrorIDs.add(errorID)

		try {
			const fetchedRecords = await getStaticRecordsByID<IRequestErrorByID>(
				'request-errors-by-ids',
				missingIDs,
				{ cacheNamespace: 'request-errors-v1' },
			)
			this.errorRecords = new Map([...this.errorRecords, ...fetchedRecords])
			console.debug('[ObservabilityService] resolved error previews:', fetchedRecords.size)
		} finally {
			for (const errorID of missingIDs) this.pendingErrorIDs.delete(errorID)
		}
	}

	constructor(init = false) {
		super()
		if (init) void this.fetch()
	}
}
