import { describe, expect, it } from 'vitest'
import type { ICrmOpportunityStageLog } from '#services/crm/crm-opportunity.svelte.ts'
import {
  aggregateNetProgress, dailyStageStates, netProgressByOwner, stageReachDeltas, wonFromPositionsOf,
} from './reports.ts'

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
