import { describe, expect, it } from 'vitest'
import { buildTimelineCards, CLOSED_COLOR, endMarkCounts, filterTimelineCards, groupCardsBySeller, type StatusRow } from './timeline.ts'

const row = (OpportunityID: number, StartDate: number, visits: [number, number, number][]): StatusRow => ({
  OpportunityID, StartDate, LastDate: StartDate + visits.at(-1)![2],
  Visits: visits.map(([StageID, Outcome, DaysPassed]) => ({ StageID, Outcome, DaysPassed })),
})
const titles = new Map([[1, { Title: 'Kitchen', OwnerUserID: 7 }], [2, { Title: 'Other', OwnerUserID: 7 }], [3, { Title: 'Third', OwnerUserID: 8 }]])
const stageColors = new Map([[1, 'blue'], [2, 'amber']])

describe('timeline', () => {
  it('builds a card with dated, coloured points, spreading changes of the same day', () => {
    const [card] = buildTimelineCards([row(1, 100, [[1, 0, 0], [2, 0, 3], [1, 0, 3], [1, 1, 5]])], titles, stageColors, 200)
    expect(card).toMatchObject({ label: 'Kitchen', ownerID: 7, startDay: 100, endDay: 105, endMark: 'won', endIcon: undefined })
    expect(card.points.map((point) => point.day)).toEqual([100, 103, 103, 105])
    expect(card.points[1].offset).toBeLessThan(card.points[2].offset)
    expect(card.points.map((point) => point.color)).toEqual(['blue', 'amber', 'blue', CLOSED_COLOR])
    expect(card.points[3]).toMatchObject({ dashed: true, icon: expect.stringContaining('star') })
  })

  it('runs an open card until today, ends a transferred one with left', () => {
    const [open, left] = buildTimelineCards([row(1, 100, [[1, 0, 0]]), row(2, 100, [[1, 0, 0], [1, 3, 4]])], titles, stageColors, 150)
    expect(open).toMatchObject({ endMark: 'open', endDay: 150, endIcon: expect.any(String) })
    expect(left).toMatchObject({ endMark: 'left', endDay: 104 })
  })

  it('filters by start or last date, always keeping open cards that started', () => {
    const cards = buildTimelineCards([row(1, 100, [[1, 0, 0]]), row(2, 50, [[1, 0, 0], [1, 2, 60]]), row(3, 300, [[1, 0, 0]])], titles, stageColors, 400)
    expect(filterTimelineCards(cards, 'start', 90, 120).map((card) => card.id)).toEqual([1])
    expect(filterTimelineCards(cards, 'last', 105, 120).map((card) => card.id)).toEqual([1, 2])
    expect(filterTimelineCards(cards, 'last', 105, 120, [8])).toEqual([])
  })

  it('groups cards per seller, sorted by name, and counts how they ended', () => {
    const cards = buildTimelineCards([row(1, 100, [[1, 0, 0], [1, 1, 5]]), row(2, 103, [[1, 0, 0], [1, 2, 4]]), row(3, 100, [[1, 0, 0]])], titles, stageColors, 400)
    const sellers = groupCardsBySeller(cards, (userID) => (userID === 7 ? 'Zoe' : 'Ana'))
    expect(sellers.map((seller) => [seller.id, seller.items.length])).toEqual([[8, 1], [7, 2]])
    expect(endMarkCounts(sellers[1].items)).toEqual({ open: 0, won: 1, lost: 1, left: 0 })
  })
})
