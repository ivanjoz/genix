import { describe, expect, it } from 'vitest'
import {
  buildTimelineCards, endMarkCounts, filterTimelineCards, packTimelineLanes, screenPixelSize, snapToScreenPixel, type StatusRow,
} from './timeline.ts'

const row = (OpportunityID: number, StartDate: number, visits: [number, number, number][]): StatusRow => ({
  OpportunityID, StartDate, LastDate: StartDate + visits.at(-1)![2],
  Visits: visits.map(([StageID, Outcome, DaysPassed]) => ({ StageID, Outcome, DaysPassed })),
})
const titles = new Map([[1, { Title: 'Kitchen', OwnerUserID: 7 }], [2, { Title: 'Other', OwnerUserID: 7 }], [3, { Title: 'Third', OwnerUserID: 8 }]])

describe('timeline', () => {
  it('builds a card with dated points, spreading changes of the same day', () => {
    const [card] = buildTimelineCards([row(1, 100, [[1, 0, 0], [2, 0, 3], [1, 0, 3], [1, 1, 5]])], titles, 200)
    expect(card).toMatchObject({ title: 'Kitchen', ownerID: 7, startDay: 100, endDay: 105, endMark: 'won' })
    expect(card.points.map((point) => point.day)).toEqual([100, 103, 103, 105])
    expect(card.points[1].offset).toBeLessThan(card.points[2].offset)
  })

  it('runs an open card until today, ends a transferred one with left', () => {
    const [open, left] = buildTimelineCards([row(1, 100, [[1, 0, 0]]), row(2, 100, [[1, 0, 0], [1, 3, 4]])], titles, 150)
    expect(open).toMatchObject({ endMark: 'open', endDay: 150 })
    expect(left).toMatchObject({ endMark: 'left', endDay: 104 })
  })

  it('filters by start or last date, always keeping open cards that started', () => {
    const cards = buildTimelineCards([row(1, 100, [[1, 0, 0]]), row(2, 50, [[1, 0, 0], [1, 2, 60]]), row(3, 300, [[1, 0, 0]])], titles, 400)
    expect(filterTimelineCards(cards, 'start', 90, 120).map((card) => card.opportunityID)).toEqual([1])
    expect(filterTimelineCards(cards, 'last', 105, 120).map((card) => card.opportunityID)).toEqual([1, 2])
    expect(filterTimelineCards(cards, 'last', 105, 120, [8])).toEqual([])
  })

  it('packs cards that do not overlap into the same lane', () => {
    const cards = buildTimelineCards([
      row(1, 100, [[1, 0, 0], [1, 1, 5]]),   // 100..105
      row(2, 103, [[1, 0, 0], [1, 2, 4]]),   // 103..107 overlaps the first: second lane
    ], titles, 400)
    const later = { ...cards[0], opportunityID: 9, startDay: 106, endDay: 110 } // fits after the first
    const [seller] = packTimelineLanes([...cards, later], 0, 1000)
    expect(seller.lanes.map((lane) => lane.map((card) => card.opportunityID))).toEqual([[1, 9], [2]])
    expect(endMarkCounts(seller.lanes)).toEqual({ open: 0, won: 2, lost: 1, left: 0 })
  })
})

describe('pixel snapping', () => {
  it('keeps every tick the same whole number of screen pixels at 110%', () => {
    const ratio = 1.1
    const widths = [0, 32, 64, 96].map(() => screenPixelSize(2, ratio) * ratio)
    for (const width of widths) expect(width).toBeCloseTo(2)
    // Positions land on screen-pixel boundaries: multiplied back they are whole numbers.
    for (const x of [15.5, 47.3, 79.9]) expect(Number.isInteger(Math.round(snapToScreenPixel(x, ratio) * ratio * 1e6) / 1e6)).toBe(true)
    expect(screenPixelSize(0.2, ratio)).toBeCloseTo(1 / ratio)
    expect(snapToScreenPixel(16, 1)).toBe(16)
  })
})
