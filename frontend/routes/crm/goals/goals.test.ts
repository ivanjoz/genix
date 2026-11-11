import { describe, expect, it } from 'vitest'
import { GOAL_METRIC, GOAL_PERIOD } from '#services/crm/crm-goal-metric.ts'
import type { ICrmGoal } from '#services/crm/crm-goal.svelte.ts'
import { changedGoalCells, copyPreviousYear, fillRow, goalGridRows, storedTarget, teamTotal, cellKey, periodsOfYear, periodOfDay, goalPeriodDays, periodsOfScope, scopeCount, scopeIndexOfDay, goalTargetsInRange } from './goals.ts'

const goal = (userID: number, period: number, target: number, metric = GOAL_METRIC.WON_AMOUNT): ICrmGoal =>
  ({ ID: 1, PipelineID: 1, UserID: userID, Period: period, Metric: metric, Target: target, ss: 1, upd: 0 })

describe('goals grid', () => {
  it('sums each seller targets over the periods starting inside a day range', () => {
    const goals = [goal(1, 2601, 100), goal(1, 2602, 200), goal(1, 2603, 400), goal(2, 2602, 50), goal(1, 2602, 9, GOAL_METRIC.WON_COUNT)]
    const [fromDay] = goalPeriodDays(GOAL_PERIOD.MONTHLY, 2602)
    const [, toDay] = goalPeriodDays(GOAL_PERIOD.MONTHLY, 2603)
    const targets = goalTargetsInRange(goals, 1, GOAL_METRIC.WON_AMOUNT, GOAL_PERIOD.MONTHLY, fromDay, toDay)
    expect([...targets]).toEqual([[1, 600], [2, 50]])
  })

  it('shows money goals in currency units', () => {
    expect(storedTarget([goal(1, 2601, 150000)], 1, GOAL_METRIC.WON_AMOUNT, 1, 2601)).toBe(1500)
    expect(storedTarget([goal(1, 2601, 4, GOAL_METRIC.WON_COUNT)], 1, GOAL_METRIC.WON_COUNT, 1, 2601)).toBe(4)
  })

  it('groups sellers by team and leaves the unassigned last', () => {
    const teams = [{ ID: 1, Name: 'North', LeaderUserID: 1, MemberUserIDs: [1, 2], ss: 1, upd: 0 },
      { ID: 2, Name: 'Empty here', LeaderUserID: 9, MemberUserIDs: [9], ss: 1, upd: 0 }]
    const rows = goalGridRows(teams, [3, 2, 1], (userID) => `u${userID}`)
    expect(rows.map((row) => row.label)).toEqual(['North', 'u1', 'u2', 'u3'])
    expect(teamTotal(rows[0], rows, (userID) => userID * 10)).toBe(30)
  })

  it('sends only the cells that changed, scaled back to the amount scale', () => {
    const goals = [goal(1, 2601, 100000)]
    const draft = new Map([[cellKey(1, 2601), 1000], [cellKey(1, 2602), 50]])
    expect(changedGoalCells(draft, goals, 1, GOAL_METRIC.WON_AMOUNT)).toEqual([
      { PipelineID: 1, UserID: 1, Period: 2602, Metric: 1, Target: 5000 },
    ])
  })

  it('fills a row and copies the previous year', () => {
    const monthly = periodsOfYear(GOAL_PERIOD.MONTHLY, 26)
    expect(fillRow(new Map(), 1, monthly, 7).size).toBe(12)
    const copied = copyPreviousYear([goal(1, 2503, 20000)], 1, GOAL_METRIC.WON_AMOUNT, [1], monthly, periodsOfYear(GOAL_PERIOD.MONTHLY, 25))
    expect(copied.get(cellKey(1, 2603))).toBe(200)
    expect(copied.size).toBe(1)
  })

  it('lists the weeks of a year, Monday to Sunday', () => {
    const weeks = periodsOfYear(GOAL_PERIOD.WEEKLY, 26)
    expect(weeks.length).toBeGreaterThanOrEqual(52)
    expect(weeks.length).toBeLessThanOrEqual(53)
    // Every week starts on a Monday of 2026 and the next one follows seven days later.
    for (const week of weeks) expect(new Date(week.firstDay * 86400000).getUTCDay()).toBe(1)
    expect(new Date(weeks[0].firstDay * 86400000).getUTCFullYear()).toBe(2026)
    expect(weeks[1].period).toBe(weeks[0].period + 1)
    const [first, last] = goalPeriodDays(GOAL_PERIOD.WEEKLY, weeks[0].period)
    expect(last - first).toBe(6)
    expect(periodOfDay(GOAL_PERIOD.WEEKLY, first + 3)).toBe(weeks[0].period)
    expect(periodOfDay(GOAL_PERIOD.MONTHLY, first + 3)).toBe(periodOfDay(GOAL_PERIOD.MONTHLY, first + 3))
  })

  it('daily pipelines show one month or one week of days at a time', () => {
    const november = periodsOfScope(GOAL_PERIOD.DAILY_BY_MONTH, 26, 10)
    expect(november).toHaveLength(30)
    expect(november[0].period).toBe(periodsOfYear(GOAL_PERIOD.DAILY_BY_MONTH, 26)[304].period)
    const week = periodsOfScope(GOAL_PERIOD.DAILY_BY_WEEK, 26, 0)
    expect(week).toHaveLength(7)
    expect(new Date(week[0].firstDay * 86400000).getUTCDay()).toBe(1)
    expect(periodsOfYear(GOAL_PERIOD.DAILY_BY_WEEK, 26)).toHaveLength(365)
    expect(scopeCount(GOAL_PERIOD.DAILY_BY_MONTH, 26)).toBe(12)
    expect(scopeCount(GOAL_PERIOD.MONTHLY, 26)).toBe(0)
    // A day maps to its own period and back to its scope.
    const day = november[4].firstDay
    expect(periodOfDay(GOAL_PERIOD.DAILY_BY_MONTH, day)).toBe(day)
    expect(scopeIndexOfDay(GOAL_PERIOD.DAILY_BY_MONTH, 26, day)).toBe(10)
    expect(periodsOfScope(GOAL_PERIOD.DAILY_BY_WEEK, 26, scopeIndexOfDay(GOAL_PERIOD.DAILY_BY_WEEK, 26, day)).some((d) => d.period === day)).toBe(true)
  })
})
