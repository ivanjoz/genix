// Pure logic of the CRM timeline (Gantt) report: one card per opportunity path in a pipeline,
// grouped per seller. Cards are GanttTimeline items, so the component draws them as they are; it
// also packs them in lanes. No Svelte, no fetch.
import type { GanttTimelineGroup, GanttTimelineItem, GanttTimelinePoint } from '#components/gantt/gantt-timeline.ts'
import { OUTCOME_LOST, OUTCOME_OPEN, OUTCOME_WON } from '#services/crm/crm-stage-log.ts'

/** Mirrors VISIT_OUTCOME_LEFT of the status service, kept here so this file has no runtime import. */
export const OUTCOME_LEFT = 3

/** Line colour of a stage without one, and of the time a card spent closed or away. */
export const CLOSED_COLOR = '#94a3b8'

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
export interface TimelinePoint extends GanttTimelinePoint {
  stageID: number
  outcome: number
}

export type TimelineEndMark = 'open' | 'won' | 'lost' | 'left'

/** A card: `id` is the opportunity ID, `endDay` the closing day, or today while it is open. */
export interface TimelineCard extends GanttTimelineItem {
  id: number
  ownerID: number
  lastDate: number
  endMark: TimelineEndMark
  points: TimelinePoint[]
}

/** Icon and label of each card ending: drawn on the line, in the tooltip and in the legend. */
export const END_MARKS: Record<TimelineEndMark, { icon: string, label: string }> = {
  won: { icon: 'icon-[fa--star] text-yellow-500', label: 'won|ganada' },
  lost: { icon: 'icon-[fa--times] text-red-500', label: 'lost|perdida' },
  left: { icon: 'icon-[fa--sign-out] text-gray-500', label: 'moved to another pipeline|pasó a otro pipeline' },
  open: { icon: 'icon-[fa--angle-double-right] text-blue-500', label: 'still open|sigue abierta' },
}

export const END_MARK_BY_OUTCOME:Record<number, TimelineEndMark> = {
  [OUTCOME_OPEN]: 'open', [OUTCOME_WON]: 'won', [OUTCOME_LOST]: 'lost', [OUTCOME_LEFT]: 'left',
}

/**
 * Cards from the status rows. The card belongs to the opportunity's current seller (the whole card
 * goes to the last seller), and an open one runs until today. A point takes its stage's colour; a
 * closing point (won, lost, left) shows its icon and the time until a reopening is dashed grey.
 */
export const buildTimelineCards = (
  rows: StatusRow[], opportunities: Map<number, OpportunityInfo>, stageColors: Map<number, string>, today: number,
): TimelineCard[] => rows.filter((row) => row.Visits.length).map((row) => {
  const days = row.Visits.map((visit) => row.StartDate + visit.DaysPassed)
  const points = row.Visits.map((visit, index): TimelinePoint => {
    const sameDayIndexes = days.map((day, dayIndex) => day === days[index] ? dayIndex : -1).filter((dayIndex) => dayIndex >= 0)
    const positionInDay = sameDayIndexes.indexOf(index)
    const isClosing = visit.Outcome !== OUTCOME_OPEN
    return {
      day: days[index],
      offset: (positionInDay + 1) / (sameDayIndexes.length + 1),
      color: isClosing ? CLOSED_COLOR : stageColors.get(visit.StageID) || CLOSED_COLOR,
      icon: isClosing ? END_MARKS[END_MARK_BY_OUTCOME[visit.Outcome]]?.icon : undefined,
      dashed: isClosing,
      stageID: visit.StageID,
      outcome: visit.Outcome,
    }
  })
  const endMark = END_MARK_BY_OUTCOME[points.at(-1)!.outcome] ?? 'open'
  const opportunity = opportunities.get(row.OpportunityID)
  return {
    id: row.OpportunityID,
    ownerID: opportunity?.OwnerUserID ?? 0,
    label: opportunity?.Title ?? `#${row.OpportunityID}`,
    labelCss: endMark === 'won' ? 'text-green-600 ff-bold' : undefined,
    startDay: row.StartDate,
    endDay: endMark === 'open' ? Math.max(today, days.at(-1)!) : days.at(-1)!,
    endIcon: endMark === 'open' ? END_MARKS.open.icon : undefined,
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

/** One timeline group per seller, sorted by the seller's name. */
export const groupCardsBySeller = (cards: TimelineCard[], ownerName: (userID: number) => string): GanttTimelineGroup<TimelineCard>[] => {
  const cardsByOwner = new Map<number, TimelineCard[]>()
  for (const card of cards) cardsByOwner.set(card.ownerID, [...(cardsByOwner.get(card.ownerID) ?? []), card])
  return [...cardsByOwner].map(([ownerID, items]) => ({ id: ownerID, label: ownerName(ownerID), items }))
    .sort((left, right) => left.label.localeCompare(right.label))
}

/** Counts of a seller's cards by how they ended: the totals shown next to the name. */
export const endMarkCounts = (cards: TimelineCard[]) => {
  const counts: Record<TimelineEndMark, number> = { open: 0, won: 0, lost: 0, left: 0 }
  for (const card of cards) counts[card.endMark]++
  return counts
}
