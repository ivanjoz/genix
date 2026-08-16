export const OBSERVABILITY_WINDOW_HOURS = 4
export const OBSERVABILITY_FRAME_COUNT = OBSERVABILITY_WINDOW_HOURS * 12

export interface IObservabilityRouteDetail {
	RouteID: number
	CPU: number
	Inference: number
	EstimatedRequests: number
	FailedRequests: number
	ErrorOccurrences: number
	ErrorIDs?: number[]
	ErrorIDCounts?: number[]
}

export interface IObservabilityFrame {
	ID: number
	TimeFrame: number
	Details?: IObservabilityRouteDetail[]
	upd: number
	ss: number
}

export interface IObservabilityRoute {
	ID: number
	Route: string
}

export interface IRequestErrorEntry {
	CodeLine: string
	Text: string
	upd?: number
}

export interface IRequestErrorByID {
	ID: number
	Entries: IRequestErrorEntry[]
}

export interface IObservabilityResponse {
	Frames?: IObservabilityFrame[]
	Routes?: IObservabilityRoute[]
}

export interface IObservabilityCard {
	RouteID: number
	Route: string
	Method: string
	Path: string
	FrameIDs: number[]
	EstimatedSuccessValues: number[]
	FailedRequestValues: number[]
	CPU: number
	Inference: number
	EstimatedRequests: number
	FailedRequests: number
	ErrorOccurrences: number
	ErrorCounts: Map<number, number>
	IsMetered: boolean
}

export interface IObservabilityErrorPreview {
	ID: number
	Count: number
	Entries: IRequestErrorEntry[]
}

const splitRoute = (route: string) => {
	const separatorIndex = route.indexOf('.')
	if (separatorIndex < 0) return { method: '', path: route }
	return {
		method: route.slice(0, separatorIndex),
		path: route.slice(separatorIndex + 1),
	}
}

/** Transposes absolute frame records into dense route-first chart cards. */
export const buildObservabilityCards = (
	frames: IObservabilityFrame[],
	routes: IObservabilityRoute[],
): IObservabilityCard[] => {
	const sortedFrames = [...frames].filter(frame => frame.ss > 0).sort((left, right) => left.ID - right.ID)
	const routeNameByID = new Map(routes.map(route => [route.ID, route.Route]))
	const cardsByRouteID = new Map<number, IObservabilityCard>()

	for (let frameIndex = 0; frameIndex < sortedFrames.length; frameIndex++) {
		const frame = sortedFrames[frameIndex]
		for (const detail of frame.Details || []) {
			let card = cardsByRouteID.get(detail.RouteID)
			if (!card) {
				const route = routeNameByID.get(detail.RouteID) || `ROUTE.${detail.RouteID}`
				const { method, path } = splitRoute(route)
				card = {
					RouteID: detail.RouteID, Route: route, Method: method, Path: path,
					FrameIDs: sortedFrames.map(currentFrame => currentFrame.ID),
					EstimatedSuccessValues: new Array(sortedFrames.length).fill(0),
					FailedRequestValues: new Array(sortedFrames.length).fill(0),
					CPU: 0, Inference: 0, EstimatedRequests: 0, FailedRequests: 0,
					ErrorOccurrences: 0, ErrorCounts: new Map(), IsMetered: false,
				}
				cardsByRouteID.set(detail.RouteID, card)
			}

			const estimatedSuccesses = Math.max(detail.EstimatedRequests - detail.FailedRequests, 0)
			card.EstimatedSuccessValues[frameIndex] = estimatedSuccesses
			card.FailedRequestValues[frameIndex] = detail.FailedRequests
			card.CPU += detail.CPU
			card.Inference += detail.Inference
			card.EstimatedRequests += detail.EstimatedRequests
			card.FailedRequests += detail.FailedRequests
			card.ErrorOccurrences += detail.ErrorOccurrences
			card.IsMetered ||= detail.CPU > 0 && (card.Method === 'GET' || card.Method === 'POST')

			for (let errorIndex = 0; errorIndex < (detail.ErrorIDs?.length || 0); errorIndex++) {
				const errorID = detail.ErrorIDs?.[errorIndex] || 0
				if (errorID <= 0) continue
				card.ErrorCounts.set(errorID, (card.ErrorCounts.get(errorID) || 0) + (detail.ErrorIDCounts?.[errorIndex] || 0))
			}
		}
	}

	return [...cardsByRouteID.values()].sort((left, right) =>
		right.CPU - left.CPU
		|| right.FailedRequests - left.FailedRequests
		|| left.RouteID - right.RouteID,
	)
}

export const collectObservabilityErrorIDs = (cards: IObservabilityCard[]): number[] => {
	const errorIDs = new Set<number>()
	for (const card of cards) {
		for (const errorID of card.ErrorCounts.keys()) errorIDs.add(errorID)
	}
	return [...errorIDs].sort((left, right) => left - right)
}

export const buildObservabilityErrorPreviews = (
	card: IObservabilityCard,
	errorRecords: Map<number, IRequestErrorByID>,
	limit = 3,
): IObservabilityErrorPreview[] => {
	return [...card.ErrorCounts.entries()]
		.sort(([leftID, leftCount], [rightID, rightCount]) => rightCount - leftCount || leftID - rightID)
		.slice(0, limit)
		.map(([ID, Count]) => ({ ID, Count, Entries: errorRecords.get(ID)?.Entries || [] }))
}

export const observabilityCardMatches = (
	card: IObservabilityCard,
	filterText: string,
	errorRecords: Map<number, IRequestErrorByID>,
): boolean => {
	if (!filterText) return true
	const searchableParts = [card.Route]
	for (const errorID of card.ErrorCounts.keys()) {
		for (const entry of errorRecords.get(errorID)?.Entries || []) {
			searchableParts.push(entry.Text, entry.CodeLine)
		}
	}
	return searchableParts.join(' ').toLowerCase().includes(filterText)
}
