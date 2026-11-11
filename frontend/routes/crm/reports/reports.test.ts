import { describe, expect, it } from 'vitest'
import type { ICrmOpportunityStageLog } from '#services/crm/crm-opportunity.svelte.ts'
import {
  addMonthsToPeriod, addScorecardSums, aggregateNetProgress, dailyStageStates, emptyScorecardSums, funnelReport,
  lostDealsInRange, medianOf, netProgressByOwner, paretoOf, periodsBetween, scorecardMetrics, scorecardSumsByOwner,
  reportablePipelineIDs, stageOccupancyDeltas, stageReachDeltas, waterfallByPeriod, waterfallDeltas, waterfallSummary,
  wonFromPositionsOf,
} from './reports.ts'
import type { ICrmPipeline } from '#services/crm/crm-pipeline.svelte.ts'

describe('reportablePipelineIDs', () => {
  // Mirrors backend crm/types/report_rules_test.go.
  const member = { UserID: 7, Level: 1 }
  const pipelines = [
    { ID: 3, ss: 1, Members: [member] },
    { ID: 1, ss: 1, Members: [] },
    { ID: 2, ss: 0, Members: [member] },
  ] as unknown as ICrmPipeline[]
  it('gives a member the active pipelines they belong to', () => {
    expect(reportablePipelineIDs(pipelines, 7, false)).toEqual([3])
  })
  it('opens every active pipeline to the full-report access, never a deleted one', () => {
    expect(reportablePipelineIDs(pipelines, 7, true)).toEqual([1, 3])
  })
  it('gives a non-member nothing', () => {
    expect(reportablePipelineIDs(pipelines, 99, false)).toEqual([])
  })
})

describe('period helpers', () => {
  it('adds months across year boundaries both ways', () => {
    expect(addMonthsToPeriod(2611, 12)).toBe(2711)
    expect(addMonthsToPeriod(2612, 1)).toBe(2701)
    expect(addMonthsToPeriod(2601, -1)).toBe(2512)
  })
  it('lists every period of a range spanning years', () => {
    expect(periodsBetween(2511, 2602)).toEqual([2511, 2512, 2601, 2602])
    expect(periodsBetween(2603, 2602)).toEqual([])
  })
})

const DAY_SECONDS = 86400
// SUnixTime of noon of a UnixDay.
const timeOfDay = (unixDay: number) => Math.floor((unixDay * DAY_SECONDS + DAY_SECONDS / 2 - 1e9) / 2)

/** A log row in pipeline 1, owner 7, amount 100, moving between positions (stage ID = 10 + position). */
const row = (day: number, kind: number, from: [number, number] | null, to: [number, number], overrides: Partial<ICrmOpportunityStageLog> = {}) => ({
  ID: day, OpportunityID: 1, Kind: kind, Period: 0, Time: timeOfDay(day), UserID: 1,
  FromPipelineID: from ? 1 : 0, FromStageID: from ? 10 + from[0] : 0, FromPosition: from?.[0] ?? 0, FromOutcome: from?.[1] ?? 0,
  FromOwnerUserID: from ? 7 : 0, FromAmount: from ? 100 : 0, FromExpectedCloseDate: 0,
  ToPipelineID: 1, ToStageID: 10 + to[0], ToPosition: to[0], ToOutcome: to[1], ToOwnerUserID: 7, ToAmount: 100,
  ToExpectedCloseDate: 0, LostReasonID: 0, ...overrides,
}) as ICrmOpportunityStageLog

const totalsOf = (rows: ICrmOpportunityStageLog[]) => {
  const totals = netProgressByOwner(aggregateNetProgress(rows, 1, 'period')).get(7) ?? new Map()
  return Object.fromEntries([...totals].map(([column, total]) => [column, total.count]))
}

describe('net progress (signed reach)', () => {
  it('counts a deal that goes back and forth only once', () => {
    // Created at 3, then 3→4, 4→3, 3→4: the user's example from the plan.
    const rows = [row(1, 1, null, [3, 0]), row(2, 2, [3, 0], [4, 0]), row(3, 2, [4, 0], [3, 0]), row(4, 2, [3, 0], [4, 0])]
    expect(totalsOf(rows)).toEqual({ p1: 1, p2: 1, p3: 1, p4: 1 })
    // The step back on its own is a −1 on position 4.
    expect(stageReachDeltas(rows[2], 1).filter((cell) => cell.count < 0).map((cell) => cell.column)).toEqual(['p1', 'p2', 'p3', 'p4'])
  })

  it('keeps only the stages actually reached when won, and records where it was won from', () => {
    const wonAtTwo = [row(1, 1, null, [1, 0]), row(2, 2, [1, 0], [2, 0]), row(3, 3, [2, 0], [2, 1])]
    expect(totalsOf(wonAtTwo)).toEqual({ p1: 1, p2: 1, won: 1, 'won@2': 1 })
  })

  it('subtracts the reached stages when lost, and gives them back on reopen', () => {
    const lost = [row(1, 1, null, [2, 0]), row(2, 3, [2, 0], [2, 2])]
    expect(totalsOf(lost)).toEqual({ lost: 1 })
    expect(totalsOf([...lost, row(3, 3, [2, 2], [2, 0])])).toEqual({ p1: 1, p2: 1 })
  })

  it('moves the count between sellers on an owner change', () => {
    const handover = row(2, 4, [2, 0], [2, 0], { ToOwnerUserID: 8 })
    const byOwner = netProgressByOwner(aggregateNetProgress([row(1, 1, null, [2, 0]), handover], 1, 'period'))
    expect(byOwner.get(7)).toBeUndefined()
    expect(byOwner.get(8)?.get('p2')?.count).toBe(1)
  })

  it('treats a transfer as out of the source pipeline and into the target', () => {
    const transfer = row(2, 2, [2, 0], [1, 0], { ToPipelineID: 2 })
    expect(stageReachDeltas(transfer, 1).every((cell) => cell.count === -1)).toBe(true)
    expect(stageReachDeltas(transfer, 2).map((cell) => cell.column)).toEqual(['p1'])
  })

  it('nets an amount change without moving the counts', () => {
    const renegotiated = row(2, 4, [2, 0], [2, 0], { FromAmount: 100, ToAmount: 150 })
    const cells = aggregateNetProgress([renegotiated], 1, 'day')
    expect(cells.every((cell) => cell.count === 0 && cell.amount === 50)).toBe(true)
  })

  it('lists the won-from positions present', () => {
    const cells = aggregateNetProgress([row(1, 3, [4, 0], [4, 1]), row(2, 3, [2, 0], [2, 1])], 1, 'day')
    expect(wonFromPositionsOf(cells)).toEqual([2, 4])
  })
})

describe('daily stage state', () => {
  it('rebuilds past days backwards from the open opportunities of today', () => {
    // Day 10 created in stage 11, day 12 moved to stage 12; today (day 14) it sits in stage 12.
    const rows = [row(10, 1, null, [1, 0]), row(12, 2, [1, 0], [2, 0])]
    const states = dailyStageStates(rows, 1, [{ OwnerUserID: 7, StageID: 12, Amount: 100 }], 9, 13, 14)
    const countIn = (day: number, stageID: number) => states.find((state) => state.day === day)?.byStage.get(stageID)?.count ?? 0
    expect(states.map((state) => state.day)).toEqual([9, 10, 11, 12, 13])
    expect([countIn(9, 11), countIn(9, 12)]).toEqual([0, 0])
    expect([countIn(11, 11), countIn(11, 12)]).toEqual([1, 0])
    expect([countIn(12, 11), countIn(12, 12)]).toEqual([0, 1])
  })

  it('filters by owner', () => {
    const states = dailyStageStates([], 1, [{ OwnerUserID: 7, StageID: 12, Amount: 100 }], 14, 14, 14, [8])
    expect(states[0].byStage.get(12)?.count ?? 0).toBe(0)
  })
})

describe('waterfall', () => {
  const stepsOf = (rowToSplit: ICrmOpportunityStageLog, pipelineID = 1) =>
    waterfallDeltas(rowToSplit, pipelineID).map((delta) => [delta.ownerID, delta.step, delta.amount])

  it('splits every kind of change into its step', () => {
    expect(stepsOf(row(1, 1, null, [1, 0]))).toEqual([[7, 'new', 100]])
    expect(stepsOf(row(2, 4, [1, 0], [1, 0], { ToAmount: 130 }))).toEqual([[7, 'increased', 30]])
    expect(stepsOf(row(2, 4, [1, 0], [1, 0], { ToAmount: 80 }))).toEqual([[7, 'decreased', -20]])
    expect(stepsOf(row(2, 3, [2, 0], [2, 1]))).toEqual([[7, 'won', -100]])
    expect(stepsOf(row(2, 3, [2, 0], [2, 2]))).toEqual([[7, 'lost', -100]])
    expect(stepsOf(row(3, 3, [2, 2], [2, 0]))).toEqual([[7, 'reopened', 100]])
    expect(stepsOf(row(2, 4, [2, 0], [2, 0], { ToOwnerUserID: 8 }))).toEqual([[7, 'transferredOut', -100], [8, 'transferredIn', 100]])
    expect(stepsOf(row(2, 2, [2, 0], [1, 0], { ToPipelineID: 2 }))).toEqual([[7, 'transferredOut', -100]])
    expect(stepsOf(row(2, 2, [2, 0], [1, 0], { ToPipelineID: 2 }), 2)).toEqual([[7, 'transferredIn', 100]])
    // A stage move of an open deal does not change the open amount.
    expect(stepsOf(row(2, 2, [1, 0], [2, 0]))).toEqual([])
  })

  it('always sums to the occupancy delta of the row', () => {
    const rows = [row(1, 1, null, [1, 0]), row(2, 4, [1, 0], [1, 0], { ToAmount: 130, ToOwnerUserID: 8 }), row(3, 3, [2, 0], [2, 2])]
    for (const checkedRow of rows) {
      const stepAmount = waterfallDeltas(checkedRow, 1).reduce((total, delta) => total + delta.amount, 0)
      const occupancyAmount = stageOccupancyDeltas(checkedRow, 1).reduce((total, delta) => total + delta.amount, 0)
      expect(stepAmount).toBe(occupancyAmount)
    }
  })

  it('opens each month with the closing of the one before, walking back from today', () => {
    // Jan 2026 (UnixDay 20454..20484): created on day 20460, still open today (Feb, day 20500).
    const rows = [row(20460, 1, null, [1, 0]), row(20495, 4, [1, 0], [1, 0], { ToAmount: 150 })]
    const waterfall = waterfallByPeriod(rows, 1, [{ OwnerUserID: 7, StageID: 11, Amount: 150 }], 2601, 2612, 20500)
    expect([...waterfall.keys()].sort()).toEqual([2601, 2602])
    expect(waterfall.get(2601)?.opening.amount).toBe(0)
    expect(waterfall.get(2601)?.closing.amount).toBe(100)
    expect(waterfall.get(2602)?.opening.amount).toBe(100)
    expect(waterfall.get(2602)?.steps.get('increased')?.amount).toBe(50)
    expect(waterfall.get(2602)?.closing.amount).toBe(150)
    const summary = waterfallSummary(waterfall)
    expect([summary?.opening.amount, summary?.steps.get('new')?.amount, summary?.closing.amount]).toEqual([0, 100, 150])
  })
})

describe('funnel', () => {
  it('counts the deepest position ever reached and only finished stays', () => {
    const rows = [
      row(10, 1, null, [1, 0]), row(12, 2, [1, 0], [3, 0]), row(13, 3, [3, 0], [3, 2]), // lost at 3
      row(10, 1, null, [1, 0], { OpportunityID: 2 }), // still open at 1: its stay is unfinished
    ]
    const funnel = funnelReport(rows, 1, 0, 100)
    expect(funnel.entered).toBe(2)
    expect(funnel.reachedByPosition).toEqual([2, 1, 1])
    expect(funnel.lost).toBe(1)
    expect(funnel.stayDaysByStage.get(11)).toEqual([2])
    expect(funnel.stayDaysByStage.get(13)).toEqual([1])
  })

  it('takes deals transferred in as entries, and leaves out those that entered before the range', () => {
    const transferredIn = row(20, 2, [2, 0], [1, 0], { FromPipelineID: 2 })
    expect(funnelReport([transferredIn], 1, 15, 25).entered).toBe(1)
    expect(funnelReport([row(5, 1, null, [1, 0])], 1, 15, 25).entered).toBe(0)
  })

  it('takes the median of even and odd lists', () => {
    expect(medianOf([3, 1, 2])).toBe(2)
    expect(medianOf([4, 1, 3, 2])).toBe(2.5)
    expect(medianOf([])).toBe(0)
  })
})

describe('seller scorecard', () => {
  it('nets reopenings, measures the cycle from creation and derives the rates from the sums', () => {
    const rows = [
      row(10, 1, null, [1, 0]), row(14, 3, [1, 0], [1, 1]), // won in 4 days
      row(10, 1, null, [1, 0], { OpportunityID: 2 }), row(11, 3, [1, 0], [1, 2], { OpportunityID: 2 }),
    ]
    const sums = scorecardSumsByOwner({
      rowsInRange: rows, rowsToToday: rows, pipelineID: 1, openNow: [{ OwnerUserID: 7, StageID: 11, Amount: 100 }],
      activityAuthorIDs: [7, 7, 8], goalAmountByOwner: new Map([[7, 400]]),
    })
    const seller = sums.get(7)!
    expect([seller.wonCount, seller.lostCount, seller.cycleCount, seller.openCount, seller.activities]).toEqual([1, 1, 1, 1, 2])
    expect(seller.cycleDaysSum).toBeCloseTo(4)
    const metrics = scorecardMetrics(seller)
    expect(metrics.winRate).toBe(0.5)
    expect(metrics.goalAttainment).toBe(25)
    // 1 open × 0.5 × 100 ÷ 4 days.
    expect(metrics.velocity).toBeCloseTo(12.5)
    expect(sums.get(8)?.activities).toBe(1)
  })

  it('adds a team from its members sums', () => {
    const team = addScorecardSums({ ...emptyScorecardSums(), wonCount: 1, lostCount: 1 }, { ...emptyScorecardSums(), wonCount: 2 })
    expect(scorecardMetrics(team).winRate).toBe(0.75)
  })
})

describe('lost reasons', () => {
  it('counts a deal once by its last state in the range, with the reason of the row that lost it', () => {
    const rows = [
      row(1, 3, [2, 0], [2, 2], { LostReasonID: 5 }),
      row(1, 3, [1, 0], [1, 2], { OpportunityID: 2, LostReasonID: 6 }), row(2, 3, [1, 2], [1, 0], { OpportunityID: 2 }), // reopened
      row(3, 3, [3, 0], [3, 2], { OpportunityID: 3, LostReasonID: 5 }),
    ]
    const lostDeals = lostDealsInRange(rows, 1)
    expect(lostDeals.map((deal) => deal.opportunityID)).toEqual([1, 3])
    const pareto = paretoOf(lostDeals, (deal) => deal.reasonID)
    expect(pareto.map((paretoRow) => [paretoRow.key, paretoRow.count, paretoRow.cumulativePercent])).toEqual([[5, 2, 100]])
  })
})
