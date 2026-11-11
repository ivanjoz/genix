// Pure logic of the CRM reports (docs/CRM_REPORTS_QUOTES_PLAN.md §8). No Svelte, no fetch: every
// report is computed here from stage log rows, which carry the opportunity's state before and
// after each change, so any period can be aggregated from its own rows alone.
import type { ICrmOpportunityStageLog } from '#services/crm/crm-opportunity.svelte.ts'
import type { ICrmPipeline } from '#services/crm/crm-pipeline.svelte.ts'
import { memberLevel } from '../pipeline/pipeline.ts'
import { OUTCOME_LOST, OUTCOME_OPEN, OUTCOME_WON, STAGE_LOG_KIND_CREATED } from '#services/crm/crm-stage-log.ts'

/** One side of a log row: where the opportunity was, whose it was and how much it was worth. */
export interface OpportunityState {
  pipelineID: number
  stageID: number
  position: number
  outcome: number
  ownerID: number
  amount: number
}

export const beforeStateOf = (row: ICrmOpportunityStageLog): OpportunityState | null =>
  row.Kind === STAGE_LOG_KIND_CREATED ? null : {
    pipelineID: row.FromPipelineID, stageID: row.FromStageID, position: row.FromPosition,
    outcome: row.FromOutcome, ownerID: row.FromOwnerUserID, amount: row.FromAmount,
  }

export const afterStateOf = (row: ICrmOpportunityStageLog): OpportunityState => ({
  pipelineID: row.ToPipelineID, stageID: row.ToStageID, position: row.ToPosition,
  outcome: row.ToOutcome, ownerID: row.ToOwnerUserID, amount: row.ToAmount,
})

/** UnixDay (UTC) of an SUnixTime, as backend core.SUnixTimeToUnixDay. */
export const unixDayOfSUnixTime = (sUnixTime: number) => Math.floor((sUnixTime * 2 + 1e9) / 86400)

/** YYMM of a UnixDay, as backend core.PeriodOfDay. */
export const periodOfUnixDay = (unixDay: number) => {
  const date = new Date(unixDay * 86400 * 1000)
  return (date.getUTCFullYear() % 100) * 100 + date.getUTCMonth() + 1
}

/** First UnixDay of a YYMM period, as backend core.PeriodFirstDay. */
export const periodFirstDay = (period: number) => Date.UTC(2000 + Math.floor(period / 100), (period % 100) - 1, 1) / 86400000

/** Last UnixDay of a YYMM period. */
export const periodLastDay = (period: number) => Date.UTC(2000 + Math.floor(period / 100), period % 100, 1) / 86400000 - 1

/** The YYMM period `months` months after (or before, if negative) the given one. */
export const addMonthsToPeriod = (period: number, months: number) => {
  const monthIndex = Math.floor(period / 100) * 12 + (period % 100) - 1 + months
  return Math.floor(monthIndex / 12) * 100 + (monthIndex % 12) + 1
}

/** Every YYMM period from `fromPeriod` to `toPeriod`, both included, ascending. */
export const periodsBetween = (fromPeriod: number, toPeriod: number): number[] => {
  const periods: number[] = []
  for (let period = fromPeriod; period <= toPeriod; period = addMonthsToPeriod(period, 1)) periods.push(period)
  return periods
}

// ---------------------------------------------------------------------------------------------
// R1 — net progress. The signed "reach" rule (§8.1): a state reaches every position up to its own;
// a won state also reaches the Won column ("won from" its position, E17); a lost state reaches
// only the Lost column (E4). A row adds Reach(after) to the after-owner and subtracts
// Reach(before) from the before-owner, so summed over any period an opportunity that goes back
// and forth counts only its net progress, and transfers / owner changes move the count.
// ---------------------------------------------------------------------------------------------

/** Report column keys: `p<position>` for a stage depth, plus the closing columns. */
export const positionColumn = (position: number) => `p${position}`
export const WON_COLUMN = 'won'
export const LOST_COLUMN = 'lost'
export const wonFromColumn = (position: number) => `won@${position}`

export interface ReachCell {
  ownerID: number
  column: string
  count: number
  amount: number
}

/** The columns one state reaches, each counted once with the state's amount. */
export const reachColumnsOf = (state: OpportunityState): string[] => {
  if (state.outcome === OUTCOME_LOST) return [LOST_COLUMN]
  const columns = Array.from({ length: Math.max(0, state.position) }, (_, index) => positionColumn(index + 1))
  if (state.outcome === OUTCOME_WON) columns.push(WON_COLUMN, wonFromColumn(state.position))
  return columns
}

/** The signed reach cells a log row contributes to the report of one pipeline. */
export const stageReachDeltas = (row: ICrmOpportunityStageLog, pipelineID: number): ReachCell[] => {
  const cells: ReachCell[] = []
  const before = beforeStateOf(row)
  if (before && before.pipelineID === pipelineID) {
    for (const column of reachColumnsOf(before)) cells.push({ ownerID: before.ownerID, column, count: -1, amount: -before.amount })
  }
  const after = afterStateOf(row)
  if (after.pipelineID === pipelineID) {
    for (const column of reachColumnsOf(after)) cells.push({ ownerID: after.ownerID, column, count: 1, amount: after.amount })
  }
  return cells
}

export interface NetProgressCell extends ReachCell {
  /** The time bucket: a UnixDay or a YYMM period, as asked. */
  bucket: number
}

/**
 * Sums the reach deltas of the rows by owner, time bucket and column. Cells that net to zero in
 * both count and amount are dropped: a deal that went back and forth inside the bucket leaves
 * nothing behind.
 */
export const aggregateNetProgress = (rows: ICrmOpportunityStageLog[], pipelineID: number, bucketBy: 'day' | 'period'): NetProgressCell[] => {
  const cellsByKey = new Map<string, NetProgressCell>()
  for (const row of rows) {
    const unixDay = unixDayOfSUnixTime(row.Time)
    const bucket = bucketBy === 'day' ? unixDay : periodOfUnixDay(unixDay)
    for (const delta of stageReachDeltas(row, pipelineID)) {
      const key = `${delta.ownerID}|${bucket}|${delta.column}`
      const cell = cellsByKey.get(key) ?? { ...delta, bucket, count: 0, amount: 0 }
      cell.count += delta.count
      cell.amount += delta.amount
      cellsByKey.set(key, cell)
    }
  }
  return [...cellsByKey.values()].filter((cell) => cell.count !== 0 || cell.amount !== 0)
}

/** Totals of each column per owner, over every bucket of the cells. */
export const netProgressByOwner = (cells: NetProgressCell[]): Map<number, Map<string, { count: number, amount: number }>> => {
  const byOwner = new Map<number, Map<string, { count: number, amount: number }>>()
  for (const cell of cells) {
    const columns = byOwner.get(cell.ownerID) ?? new Map()
    const total = columns.get(cell.column) ?? { count: 0, amount: 0 }
    total.count += cell.count
    total.amount += cell.amount
    columns.set(cell.column, total)
    byOwner.set(cell.ownerID, columns)
  }
  return byOwner
}

/**
 * The report's columns in display order: every stage depth up to the deepest one seen (or the
 * pipeline's current stage count), then Won, the "won from" splits present, and Lost.
 */
export const netProgressColumns = (cells: NetProgressCell[], stageCount: number): string[] => {
  const deepestPosition = Math.max(stageCount, ...cells.filter((cell) => cell.column.startsWith('p')).map((cell) => Number(cell.column.slice(1))))
  return [
    ...Array.from({ length: deepestPosition }, (_, index) => positionColumn(index + 1)),
    WON_COLUMN, ...wonFromPositionsOf(cells).map(wonFromColumn), LOST_COLUMN,
  ]
}

/** The "won from" positions that appear in the cells, ascending: the report shows only those. */
export const wonFromPositionsOf = (cells: NetProgressCell[]): number[] =>
  [...new Set(cells.filter((cell) => cell.column.startsWith('won@')).map((cell) => Number(cell.column.slice(4))))].sort((a, b) => a - b)

// ---------------------------------------------------------------------------------------------
// R2 — daily state. Occupancy is where open opportunities sit, by stage ID (not position: a
// reorder does not change who sits where). A row removes the before-state from its stage and adds
// the after-state to its own, counting only open states of the report's pipeline.
// ---------------------------------------------------------------------------------------------

export interface OccupancyCell {
  ownerID: number
  stageID: number
  count: number
  amount: number
}

export const stageOccupancyDeltas = (row: ICrmOpportunityStageLog, pipelineID: number): OccupancyCell[] => {
  const cells: OccupancyCell[] = []
  const before = beforeStateOf(row)
  if (before && before.pipelineID === pipelineID && before.outcome === OUTCOME_OPEN) {
    cells.push({ ownerID: before.ownerID, stageID: before.stageID, count: -1, amount: -before.amount })
  }
  const after = afterStateOf(row)
  if (after.pipelineID === pipelineID && after.outcome === OUTCOME_OPEN) {
    cells.push({ ownerID: after.ownerID, stageID: after.stageID, count: 1, amount: after.amount })
  }
  return cells
}

/** The CRM access "CRM Reporte completo" (backend/access.toml): it opens every pipeline's reports. */
export const ACCESO_ID_CRM_FULL_REPORT = 57

/**
 * The pipelines a user may report on, as backend crm/types/report_rules.go ReportablePipelineIDs:
 * the active ones, all of them with the full-report access, otherwise those the user is a member
 * of. Computed from the cached pipelines list, so the reports need no request of their own for it.
 */
export const reportablePipelineIDs = (pipelines: ICrmPipeline[], userID: number, holdsFullReport: boolean): number[] =>
  pipelines
    .filter((pipeline) => pipeline.ss > 0 && (holdsFullReport || memberLevel(pipeline, userID) > 0))
    .map((pipeline) => pipeline.ID)
    .sort((left, right) => left - right)

/** An open opportunity as it stands now: the anchor the daily state is rebuilt backwards from. */
export interface OpenOpportunityNow {
  OwnerUserID: number
  StageID: number
  Amount: number
  ExpectedCloseDate?: number
}

export interface DailyStageState {
  day: number
  byStage: Map<number, { count: number, amount: number }>
}

/**
 * The open opportunities per stage at the END of each day from fromDay to toDay, for the chosen
 * owners (all when empty). It starts from today's open opportunities and undoes, day by day going
 * back, the occupancy deltas of the log, so rows must cover every day from fromDay to today.
 */
export const dailyStageStates = (
  rows: ICrmOpportunityStageLog[], pipelineID: number, openNow: OpenOpportunityNow[],
  fromDay: number, toDay: number, today: number, ownerIDs: number[] = [],
): DailyStageState[] => {
  const includesOwner = (ownerID: number) => !ownerIDs.length || ownerIDs.includes(ownerID)
  const current = new Map<number, { count: number, amount: number }>()
  const addTo = (stageID: number, count: number, amount: number) => {
    const total = current.get(stageID) ?? { count: 0, amount: 0 }
    total.count += count
    total.amount += amount
    current.set(stageID, total)
  }
  for (const opportunity of openNow) {
    if (includesOwner(opportunity.OwnerUserID)) addTo(opportunity.StageID, 1, opportunity.Amount)
  }

  const deltasByDay = new Map<number, OccupancyCell[]>()
  for (const row of rows) {
    const unixDay = unixDayOfSUnixTime(row.Time)
    if (unixDay < fromDay) continue
    const deltas = stageOccupancyDeltas(row, pipelineID).filter((delta) => includesOwner(delta.ownerID))
    deltasByDay.set(unixDay, [...(deltasByDay.get(unixDay) ?? []), ...deltas])
  }

  const states: DailyStageState[] = []
  for (let day = today; day >= fromDay; day--) {
    if (day <= toDay) states.push({ day, byStage: new Map([...current].map(([stageID, total]) => [stageID, { ...total }])) })
    // Undo this day's changes: what remains is the state at the end of the day before.
    for (const delta of deltasByDay.get(day) ?? []) addTo(delta.stageID, -delta.count, -delta.amount)
  }
  return states.reverse()
}

/** Open opportunities of one pipeline and owner set: whether a state counts as open in it. */
const isOpenIn = (state: OpportunityState | null, pipelineID: number): boolean =>
  !!state && state.pipelineID === pipelineID && state.outcome === OUTCOME_OPEN

/** Days between two SUnixTimes (each unit is 2 seconds). */
export const daysBetweenSUnixTimes = (fromTime: number, toTime: number) => (toTime - fromTime) * 2 / 86400

/** Rows of each opportunity in time order (ID breaks ties: rows of one write share the second). */
export const rowsByOpportunity = (rows: ICrmOpportunityStageLog[]): Map<number, ICrmOpportunityStageLog[]> => {
  const byOpportunity = new Map<number, ICrmOpportunityStageLog[]>()
  for (const row of [...rows].sort((left, right) => left.Time - right.Time || left.ID - right.ID)) {
    byOpportunity.set(row.OpportunityID, [...(byOpportunity.get(row.OpportunityID) ?? []), row])
  }
  return byOpportunity
}

// ---------------------------------------------------------------------------------------------
// R3 — pipeline waterfall. The open amount of a seller moves only through these steps:
// opening + new + increased − decreased − won − lost + reopened ± transferred = closing.
// Every row is split into steps whose sum is exactly its occupancy delta (R2), so the waterfall
// always closes, and the opening of a month is its closing minus the month's steps.
// ---------------------------------------------------------------------------------------------

export const WATERFALL_STEPS = ['new', 'increased', 'decreased', 'won', 'lost', 'reopened', 'transferredIn', 'transferredOut'] as const
export type WaterfallStep = typeof WATERFALL_STEPS[number]

export interface WaterfallDelta {
  ownerID: number
  step: WaterfallStep
  count: number
  amount: number
}

/** The waterfall steps one log row contributes to the report of one pipeline. */
export const waterfallDeltas = (row: ICrmOpportunityStageLog, pipelineID: number): WaterfallDelta[] => {
  const before = beforeStateOf(row)
  const after = afterStateOf(row)
  const beforeOpen = isOpenIn(before, pipelineID)
  const afterOpen = isOpenIn(after, pipelineID)
  const leaving = (step: WaterfallStep): WaterfallDelta => ({ ownerID: before!.ownerID, step, count: -1, amount: -before!.amount })
  const entering = (step: WaterfallStep): WaterfallDelta => ({ ownerID: after.ownerID, step, count: 1, amount: after.amount })

  if (beforeOpen && afterOpen) {
    // A hand-over moves the deal between sellers; otherwise only an amount change moves the total.
    if (before!.ownerID !== after.ownerID) return [leaving('transferredOut'), entering('transferredIn')]
    const amountChange = after.amount - before!.amount
    if (!amountChange) return []
    return [{ ownerID: after.ownerID, step: amountChange > 0 ? 'increased' : 'decreased', count: 0, amount: amountChange }]
  }
  if (beforeOpen) {
    if (after.pipelineID !== pipelineID) return [leaving('transferredOut')]
    return [leaving(after.outcome === OUTCOME_WON ? 'won' : 'lost')]
  }
  if (afterOpen) {
    if (!before) return [entering('new')]
    if (before.pipelineID !== pipelineID) return [entering('transferredIn')]
    return [entering('reopened')]
  }
  return []
}

export interface WaterfallTotals {
  opening: { count: number, amount: number }
  steps: Map<WaterfallStep, { count: number, amount: number }>
  closing: { count: number, amount: number }
}

/**
 * The waterfall of each period from fromPeriod to toPeriod (never past today's period), for the
 * chosen owners (all when empty). The closing of a period is today's open opportunities with every
 * later row undone, so rows must cover every day from the first period to today.
 */
export const waterfallByPeriod = (
  rows: ICrmOpportunityStageLog[], pipelineID: number, openNow: OpenOpportunityNow[],
  fromPeriod: number, toPeriod: number, today: number, ownerIDs: number[] = [],
): Map<number, WaterfallTotals> => {
  const includesOwner = (ownerID: number) => !ownerIDs.length || ownerIDs.includes(ownerID)
  const lastPeriod = Math.min(toPeriod, periodOfUnixDay(today))
  const stepsByPeriod = new Map<number, Map<WaterfallStep, { count: number, amount: number }>>()
  // Open total at the end of the last shown period: today's, minus what happened after it.
  const closingOfLast = { count: 0, amount: 0 }
  for (const opportunity of openNow) {
    if (!includesOwner(opportunity.OwnerUserID)) continue
    closingOfLast.count++
    closingOfLast.amount += opportunity.Amount
  }
  for (const row of rows) {
    const period = periodOfUnixDay(unixDayOfSUnixTime(row.Time))
    for (const delta of waterfallDeltas(row, pipelineID)) {
      if (!includesOwner(delta.ownerID)) continue
      if (period > lastPeriod) {
        closingOfLast.count -= delta.count
        closingOfLast.amount -= delta.amount
        continue
      }
      const steps = stepsByPeriod.get(period) ?? new Map()
      const total = steps.get(delta.step) ?? { count: 0, amount: 0 }
      total.count += delta.count
      total.amount += delta.amount
      steps.set(delta.step, total)
      stepsByPeriod.set(period, steps)
    }
  }

  // Walk back from the last period: each opening is its closing minus its own steps.
  const result = new Map<number, WaterfallTotals>()
  let closing = closingOfLast
  for (const period of periodsBetween(fromPeriod, lastPeriod).reverse()) {
    const steps = stepsByPeriod.get(period) ?? new Map()
    const opening = { ...closing }
    for (const total of steps.values()) {
      opening.count -= total.count
      opening.amount -= total.amount
    }
    result.set(period, { opening, steps, closing })
    closing = opening
  }
  return result
}

/** The whole range as one waterfall: the first opening, every period's steps summed, the last closing. */
export const waterfallSummary = (byPeriod: Map<number, WaterfallTotals>): WaterfallTotals | null => {
  const periods = [...byPeriod.keys()].sort((left, right) => left - right)
  if (!periods.length) return null
  const steps = new Map<WaterfallStep, { count: number, amount: number }>()
  for (const totals of byPeriod.values()) {
    for (const [step, total] of totals.steps) {
      const summed = steps.get(step) ?? { count: 0, amount: 0 }
      summed.count += total.count
      summed.amount += total.amount
      steps.set(step, summed)
    }
  }
  return { opening: byPeriod.get(periods[0])!.opening, steps, closing: byPeriod.get(periods.at(-1)!)!.closing }
}

// ---------------------------------------------------------------------------------------------
// R4 — funnel and time in stage. The cohort is every opportunity that ENTERED the pipeline in
// the range (created in it or transferred in), so its whole history from entry is loaded. A deal
// reached a position if it ever sat there or deeper; lost deals count where they died (E4 only
// applies to the net-progress report). Time in stage counts finished stays only: a deal still
// sitting in a stage today is left out, so a young cohort does not skew the durations.
// ---------------------------------------------------------------------------------------------

export interface FunnelReport {
  /** Opportunities that entered in the range. */
  entered: number
  /** reachedByPosition[k - 1] = how many reached position k or deeper. */
  reachedByPosition: number[]
  won: number
  lost: number
  /** Finished stays per stage ID, in days. */
  stayDaysByStage: Map<number, number[]>
}

const entersPipeline = (row: ICrmOpportunityStageLog, pipelineID: number) =>
  row.ToPipelineID === pipelineID && (row.Kind === STAGE_LOG_KIND_CREATED || row.FromPipelineID !== pipelineID)

export const funnelReport = (
  rows: ICrmOpportunityStageLog[], pipelineID: number, fromDay: number, toDay: number, ownerIDs: number[] = [],
): FunnelReport => {
  const report: FunnelReport = { entered: 0, reachedByPosition: [], won: 0, lost: 0, stayDaysByStage: new Map() }
  for (const opportunityRows of rowsByOpportunity(rows).values()) {
    const entryIndex = opportunityRows.findIndex((row) => {
      const unixDay = unixDayOfSUnixTime(row.Time)
      return entersPipeline(row, pipelineID) && unixDay >= fromDay && unixDay <= toDay
    })
    if (entryIndex < 0) continue
    // The seller filter applies to whoever owned the deal when it entered.
    if (ownerIDs.length && !ownerIDs.includes(opportunityRows[entryIndex].ToOwnerUserID)) continue
    report.entered++

    let deepestPosition = 0
    let lastState: OpportunityState | null = null
    let stay: { stageID: number, since: number } | null = null
    for (const row of opportunityRows.slice(entryIndex)) {
      const after = afterStateOf(row)
      // A stay ends when the deal leaves its stage open in this pipeline: moved, closed or transferred.
      if (stay && !(isOpenIn(after, pipelineID) && after.stageID === stay.stageID)) {
        const stays = report.stayDaysByStage.get(stay.stageID) ?? []
        stays.push(daysBetweenSUnixTimes(stay.since, row.Time))
        report.stayDaysByStage.set(stay.stageID, stays)
        stay = null
      }
      if (isOpenIn(after, pipelineID) && !stay) stay = { stageID: after.stageID, since: row.Time }
      if (after.pipelineID === pipelineID) deepestPosition = Math.max(deepestPosition, after.position)
      lastState = after
    }
    for (let position = 1; position <= deepestPosition; position++) {
      report.reachedByPosition[position - 1] = (report.reachedByPosition[position - 1] ?? 0) + 1
    }
    if (lastState?.pipelineID === pipelineID && lastState.outcome === OUTCOME_WON) report.won++
    if (lastState?.pipelineID === pipelineID && lastState.outcome === OUTCOME_LOST) report.lost++
  }
  return report
}

export const medianOf = (values: number[]) => {
  if (!values.length) return 0
  const sorted = [...values].sort((left, right) => left - right)
  const middle = Math.floor(sorted.length / 2)
  return sorted.length % 2 ? sorted[middle] : (sorted[middle - 1] + sorted[middle]) / 2
}

export const averageOf = (values: number[]) => values.length ? values.reduce((total, value) => total + value, 0) / values.length : 0

// ---------------------------------------------------------------------------------------------
// R5 — seller scorecard. Raw sums per seller are kept apart from the rates derived from them, so a
// team subtotal adds its members' sums and derives its own rates (never averages of rates).
// ---------------------------------------------------------------------------------------------

export interface ScorecardSums {
  wonCount: number
  wonAmount: number
  lostCount: number
  /** Won deals whose creation row is loaded, and the sum of their days from creation to won. */
  cycleCount: number
  cycleDaysSum: number
  openCount: number
  activities: number
  /** Won-amount goal of the range, in the amount scale. */
  goalAmount: number
}

export const emptyScorecardSums = (): ScorecardSums =>
  ({ wonCount: 0, wonAmount: 0, lostCount: 0, cycleCount: 0, cycleDaysSum: 0, openCount: 0, activities: 0, goalAmount: 0 })

export interface ScorecardInput {
  /** Rows inside the range. */
  rowsInRange: ICrmOpportunityStageLog[]
  /** Rows from the range's start year to today: where the creation rows are looked up. */
  rowsToToday: ICrmOpportunityStageLog[]
  pipelineID: number
  openNow: OpenOpportunityNow[]
  /** Author of every activity written in the pipeline inside the range. */
  activityAuthorIDs: number[]
  /** Won-amount goal of each seller for the range. */
  goalAmountByOwner: Map<number, number>
}

export const scorecardSumsByOwner = (input: ScorecardInput): Map<number, ScorecardSums> => {
  const byOwner = new Map<number, ScorecardSums>()
  const sumsOf = (ownerID: number) => {
    const sums = byOwner.get(ownerID) ?? emptyScorecardSums()
    byOwner.set(ownerID, sums)
    return sums
  }

  // Won and lost are net of reopenings, as in R1: the reach rule already does that.
  for (const cell of aggregateNetProgress(input.rowsInRange, input.pipelineID, 'period')) {
    if (cell.column === WON_COLUMN) {
      sumsOf(cell.ownerID).wonCount += cell.count
      sumsOf(cell.ownerID).wonAmount += cell.amount
    } else if (cell.column === LOST_COLUMN) {
      sumsOf(cell.ownerID).lostCount += cell.count
    }
  }

  const createdTimeByOpportunity = new Map<number, number>()
  for (const row of input.rowsToToday) {
    if (row.Kind === STAGE_LOG_KIND_CREATED) createdTimeByOpportunity.set(row.OpportunityID, row.Time)
  }
  for (const row of input.rowsInRange) {
    const isWin = row.ToPipelineID === input.pipelineID && row.ToOutcome === OUTCOME_WON && row.FromOutcome !== OUTCOME_WON
    const createdTime = createdTimeByOpportunity.get(row.OpportunityID)
    if (!isWin || createdTime === undefined) continue
    sumsOf(row.ToOwnerUserID).cycleCount++
    sumsOf(row.ToOwnerUserID).cycleDaysSum += daysBetweenSUnixTimes(createdTime, row.Time)
  }

  for (const opportunity of input.openNow) sumsOf(opportunity.OwnerUserID).openCount++
  for (const authorID of input.activityAuthorIDs) sumsOf(authorID).activities++
  for (const [ownerID, goalAmount] of input.goalAmountByOwner) sumsOf(ownerID).goalAmount += goalAmount
  return byOwner
}

export const addScorecardSums = (left: ScorecardSums, right: ScorecardSums): ScorecardSums => ({
  wonCount: left.wonCount + right.wonCount, wonAmount: left.wonAmount + right.wonAmount,
  lostCount: left.lostCount + right.lostCount, cycleCount: left.cycleCount + right.cycleCount,
  cycleDaysSum: left.cycleDaysSum + right.cycleDaysSum, openCount: left.openCount + right.openCount,
  activities: left.activities + right.activities, goalAmount: left.goalAmount + right.goalAmount,
})

export interface ScorecardMetrics {
  /** 0..1: won / (won + lost); 0 with no closed deals. */
  winRate: number
  averageWonAmount: number
  averageCycleDays: number
  /** Expected won amount per day: open count × win rate × average won amount ÷ cycle days. */
  velocity: number
  /** Won amount / goal as a percentage; 0 without a goal. */
  goalAttainment: number
}

export const scorecardMetrics = (sums: ScorecardSums): ScorecardMetrics => {
  const closedCount = sums.wonCount + sums.lostCount
  const winRate = closedCount > 0 ? Math.max(0, sums.wonCount) / closedCount : 0
  const averageWonAmount = sums.wonCount > 0 ? sums.wonAmount / sums.wonCount : 0
  const averageCycleDays = sums.cycleCount > 0 ? sums.cycleDaysSum / sums.cycleCount : 0
  return {
    winRate, averageWonAmount, averageCycleDays,
    velocity: averageCycleDays > 0 ? sums.openCount * winRate * averageWonAmount / averageCycleDays : 0,
    goalAttainment: sums.goalAmount > 0 ? Math.round(sums.wonAmount * 100 / sums.goalAmount) : 0,
  }
}

// ---------------------------------------------------------------------------------------------
// R7 — lost reasons. A deal counts once when its LAST state in the range is lost in this
// pipeline: a deal lost and reopened inside the range was not lost, and the reason is read from
// the row that lost it (the reopening row no longer carries it).
// ---------------------------------------------------------------------------------------------

export interface LostDeal {
  opportunityID: number
  reasonID: number
  ownerID: number
  stageID: number
  amount: number
}

export const lostDealsInRange = (rowsInRange: ICrmOpportunityStageLog[], pipelineID: number, ownerIDs: number[] = []): LostDeal[] => {
  const lostDeals: LostDeal[] = []
  for (const [opportunityID, opportunityRows] of rowsByOpportunity(rowsInRange)) {
    const lastRow = opportunityRows.at(-1)!
    if (lastRow.ToPipelineID !== pipelineID || lastRow.ToOutcome !== OUTCOME_LOST) continue
    if (ownerIDs.length && !ownerIDs.includes(lastRow.ToOwnerUserID)) continue
    lostDeals.push({
      opportunityID, reasonID: lastRow.LostReasonID, ownerID: lastRow.ToOwnerUserID, stageID: lastRow.ToStageID, amount: lastRow.ToAmount,
    })
  }
  return lostDeals
}

export interface ParetoRow {
  key: number
  count: number
  amount: number
  /** Share of all lost deals, and running share down the sorted list (0..100). */
  percent: number
  cumulativePercent: number
}

/** Lost deals grouped by a key (reason, seller, stage), largest count first, with running shares. */
export const paretoOf = (lostDeals: LostDeal[], keyOf: (deal: LostDeal) => number): ParetoRow[] => {
  const byKey = new Map<number, ParetoRow>()
  for (const deal of lostDeals) {
    const key = keyOf(deal)
    const paretoRow = byKey.get(key) ?? { key, count: 0, amount: 0, percent: 0, cumulativePercent: 0 }
    paretoRow.count++
    paretoRow.amount += deal.amount
    byKey.set(key, paretoRow)
  }
  const sorted = [...byKey.values()].sort((left, right) => right.count - left.count || right.amount - left.amount)
  let runningCount = 0
  for (const paretoRow of sorted) {
    runningCount += paretoRow.count
    paretoRow.percent = paretoRow.count * 100 / lostDeals.length
    paretoRow.cumulativePercent = runningCount * 100 / lostDeals.length
  }
  return sorted
}
