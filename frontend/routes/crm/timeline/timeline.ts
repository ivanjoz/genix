// Pure logic of the CRM timeline (Gantt) report: one card per opportunity path in a pipeline, laid
// out in lanes per seller so cards that do not overlap share a lane. No Svelte, no fetch.
import { OUTCOME_LOST, OUTCOME_OPEN, OUTCOME_WON } from '#services/crm/crm-stage-log.ts'

/** Mirrors VISIT_OUTCOME_LEFT of the status service, kept here so this file has no runtime import. */
export const OUTCOME_LEFT = 3

export interface StatusRow {
  OpportunityID: number
  StartDate: number
  LastDate: number
  Visits: { StageID: number, Outcome: number, DaysPassed: number }[]
}

export interface OpportunityInfo {
  Title: string
  OwnerUserID: number
}

/** One state the card entered, on a day; `offset` (0..1) spreads several changes of one day. */
export interface TimelinePoint {
  day: number
  offset: number
  stageID: number
  outcome: number
}

export type TimelineEndMark = 'open' | 'won' | 'lost' | 'left'

export interface TimelineCard {
  opportunityID: number
  ownerID: number
  title: string
  startDay: number
  /** Last day drawn: the closing day, or today while it is open. */
  endDay: number
  lastDate: number
  endMark: TimelineEndMark
  points: TimelinePoint[]
}

const END_MARK_BY_OUTCOME: Record<number, TimelineEndMark> = {
  [OUTCOME_OPEN]: 'open', [OUTCOME_WON]: 'won', [OUTCOME_LOST]: 'lost', [OUTCOME_LEFT]: 'left',
}

/**
 * Cards from the status rows. The card belongs to the opportunity's current seller (the whole card
 * goes to the last seller), and an open one runs until today.
 */
export const buildTimelineCards = (
  rows: StatusRow[], opportunities: Map<number, OpportunityInfo>, today: number,
): TimelineCard[] => rows.filter((row) => row.Visits.length).map((row) => {
  const days = row.Visits.map((visit) => row.StartDate + visit.DaysPassed)
  const points = row.Visits.map((visit, index): TimelinePoint => {
    const sameDayIndexes = days.map((day, dayIndex) => day === days[index] ? dayIndex : -1).filter((dayIndex) => dayIndex >= 0)
    const positionInDay = sameDayIndexes.indexOf(index)
    return { day: days[index], offset: (positionInDay + 1) / (sameDayIndexes.length + 1), stageID: visit.StageID, outcome: visit.Outcome }
  })
  const endMark = END_MARK_BY_OUTCOME[points.at(-1)!.outcome] ?? 'open'
  const opportunity = opportunities.get(row.OpportunityID)
  return {
    opportunityID: row.OpportunityID,
    ownerID: opportunity?.OwnerUserID ?? 0,
    title: opportunity?.Title ?? `#${row.OpportunityID}`,
    startDay: row.StartDate,
    endDay: endMark === 'open' ? Math.max(today, days.at(-1)!) : days.at(-1)!,
    lastDate: row.LastDate,
    endMark,
    points,
  }
})

export type TimelineDateMode = 'start' | 'last'

/**
 * The cards the range selects: by StartDate or by LastDate inside [fromDay, toDay]. An open card
 * that started by the end of the range is always kept: it is still alive inside the range even if
 * nothing changed in it lately, and seeing those "left hanging" is part of the report.
 */
export const filterTimelineCards = (
  cards: TimelineCard[], mode: TimelineDateMode, fromDay: number, toDay: number, ownerIDs: number[] = [],
) => cards.filter((card) => {
  if (ownerIDs.length && !ownerIDs.includes(card.ownerID)) return false
  if (card.endMark === 'open' && card.startDay <= toDay) return true
  const date = mode === 'start' ? card.startDay : card.lastDate
  return date >= fromDay && date <= toDay
})

export interface SellerLanes {
  ownerID: number
  lanes: TimelineCard[][]
}

/**
 * Lanes per seller: each card goes in the first lane whose last card ended before it starts
 * (first fit, by start day), so a seller takes as few rows as their overlapping cards need.
 * Extents are clipped to the visible window, which is what decides whether two cards collide.
 */
export const packTimelineLanes = (cards: TimelineCard[], fromDay: number, toDay: number): SellerLanes[] => {
  const byOwner = new Map<number, TimelineCard[]>()
  for (const card of cards) byOwner.set(card.ownerID, [...(byOwner.get(card.ownerID) ?? []), card])
  return [...byOwner].map(([ownerID, ownerCards]) => {
    const sorted = [...ownerCards].sort((left, right) => left.startDay - right.startDay || left.endDay - right.endDay)
    const lanes: TimelineCard[][] = []
    const laneEnds: number[] = []
    for (const card of sorted) {
      const start = Math.max(card.startDay, fromDay)
      const end = Math.min(card.endDay, toDay)
      const laneIndex = laneEnds.findIndex((laneEnd) => laneEnd < start)
      if (laneIndex < 0) {
        lanes.push([card])
        laneEnds.push(end)
      } else {
        lanes[laneIndex].push(card)
        laneEnds[laneIndex] = end
      }
    }
    return { ownerID, lanes }
  })
}

/** Counts of a seller's cards by how they ended: the totals shown next to the name. */
export const endMarkCounts = (lanes: TimelineCard[][]) => {
  const counts: Record<TimelineEndMark, number> = { open: 0, won: 0, lost: 0, left: 0 }
  for (const card of lanes.flat()) counts[card.endMark]++
  return counts
}

// ---------------------------------------------------------------------------------------------
// Pixel snapping. At a zoom like 110% one CSS px is 1.1 screen px, so a 2px tick or a 3px line
// falls between screen pixels and the browser rounds each one differently: some ticks look 2px,
// some 3px, some faded. Sizes and positions are therefore computed in whole screen pixels.
// ---------------------------------------------------------------------------------------------

/** A CSS position moved to the nearest screen-pixel boundary. */
export const snapToScreenPixel = (cssPx: number, devicePixelRatio: number) =>
  Math.round(cssPx * devicePixelRatio) / devicePixelRatio

/** A CSS size turned into a whole number of screen pixels (at least one), back in CSS px. */
export const screenPixelSize = (cssPx: number, devicePixelRatio: number) =>
  Math.max(1, Math.round(cssPx * devicePixelRatio)) / devicePixelRatio
