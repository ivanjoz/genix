// Pure logic of the CRM reports (docs/CRM_REPORTS_QUOTES_PLAN.md §8). No Svelte, no fetch: every
// report is computed here from stage log rows, which carry the opportunity's state before and
// after each change, so any period can be aggregated from its own rows alone.
import type { ICrmOpportunityStageLog } from '#services/crm/crm-opportunity.svelte.ts'
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
