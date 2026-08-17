import { describe, expect, it } from 'bun:test'
import {
	buildObservabilityCards,
	buildObservabilityErrorPreviews,
	mergeObservabilityRoutes,
	observabilityCardMatches,
	type IObservabilityFrame,
	type IRequestErrorByID,
} from './observability.model'

const routes = [
	{ ID: 34, Route: 'GET.products' },
	{ ID: 90, Route: 'POST.shared-lists' },
]

describe('observability model', () => {
	it('transposes absolute frames and zero-fills missing route slots', () => {
		const frames: IObservabilityFrame[] = [
			{
				ID: 400, TimeFrame: 106_000_000, upd: 400, ss: 1,
				Details: [{
					RouteID: 34, CPU: 10, Inference: 2, EstimatedRequests: 5,
					FailedRequests: 2, ErrorOccurrences: 3, ErrorIDs: [7], ErrorIDCounts: [2],
				}],
			},
			{ ID: 550, TimeFrame: 106_000_001, upd: 550, ss: 1, Details: [] },
		]

		const [card] = buildObservabilityCards(frames, routes)
		expect(card.FrameIDs).toEqual([400, 550])
		expect(card.EstimatedSuccessValues).toEqual([3, 0])
		expect(card.FailedRequestValues).toEqual([2, 0])
		expect(card.EstimatedRequests).toBe(5)
		expect(card.FailedRequests).toBe(2)
		expect(card.ErrorOccurrences).toBe(3)
		expect(card.IsMetered).toBe(true)
	})

	it('retains an error-only route and sorts it after credit usage', () => {
		const frames: IObservabilityFrame[] = [{
			ID: 400, TimeFrame: 106_000_000, upd: 400, ss: 1,
			Details: [
				{ RouteID: 90, CPU: 0, Inference: 0, EstimatedRequests: 0, FailedRequests: 4, ErrorOccurrences: 4 },
				{ RouteID: 34, CPU: 2, Inference: 0, EstimatedRequests: 1, FailedRequests: 0, ErrorOccurrences: 0 },
			],
		}]

		const cards = buildObservabilityCards(frames, routes)
		expect(cards.map(card => card.RouteID)).toEqual([34, 90])
		expect(cards[1].IsMetered).toBe(false)
		expect(cards[1].FailedRequestValues).toEqual([4])
	})

	it('ranks error hashes and searches resolved preview text', () => {
		const [card] = buildObservabilityCards([{
			ID: 400, TimeFrame: 106_000_000, upd: 400, ss: 1,
			Details: [{
				RouteID: 34, CPU: 2, Inference: 0, EstimatedRequests: 1,
				FailedRequests: 1, ErrorOccurrences: 1, ErrorIDs: [7, 8], ErrorIDCounts: [2, 5],
			}],
		}], routes)
		const errorRecords = new Map<number, IRequestErrorByID>([[8, {
			ID: 8,
			Entries: [{ CodeLine: 'products.go:42', Text: 'database timeout' }],
		}]])

		expect(buildObservabilityErrorPreviews(card, errorRecords)[0].ID).toBe(8)
		expect(observabilityCardMatches(card, 'timeout', errorRecords)).toBe(true)
		expect(observabilityCardMatches(card, 'unrelated', errorRecords)).toBe(false)
	})

	it('preserves route names when a frame-only delta omits cold metadata', () => {
		expect(mergeObservabilityRoutes(routes, undefined)).toBe(routes)
		expect(mergeObservabilityRoutes(routes, [])).toBe(routes)
	})

	it('normalizes sparse cached metrics before computing chart values', () => {
		const [card] = buildObservabilityCards([{
			ID: 400, TimeFrame: 106_000_000, upd: 400, ss: 1,
			Details: [{ RouteID: 34, CPU: 2, EstimatedRequests: 1 }],
		}], routes)

		expect(card.Route).toBe('GET.products')
		expect(card.FailedRequests).toBe(0)
		expect(card.ErrorOccurrences).toBe(0)
		expect(card.EstimatedSuccessValues).toEqual([1])
		expect(card.EstimatedRequests).toBe(1)
		expect(card.IsMetered).toBe(true)
	})

	it('labels a log without a route ID instead of rendering undefined', () => {
		const [card] = buildObservabilityCards([{
			ID: 400, TimeFrame: 106_000_000, upd: 400, ss: 1,
			Details: [{ FailedRequests: 1, ErrorOccurrences: 1 }],
		}], routes)

		expect(card.RouteID).toBe(0)
		expect(card.Method).toBe('API')
		expect(card.Path).toBe('UNKNOWN')
		expect(card.FailedRequestValues).toEqual([1])
	})
})
