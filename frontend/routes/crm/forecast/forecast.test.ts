import { describe, expect, it } from 'vitest'
import type { ICrmForecastSubmission } from '#services/crm/crm-forecast.svelte.ts'
import type { ICrmPipelineStage } from '#services/crm/crm-pipeline.svelte.ts'
import { GOAL_PERIOD } from '#services/crm/crm-goal-metric.ts'
import { weekOfDay } from '../goals/goals.ts'
import { periodFirstDay, periodLastDay } from '../reports/reports.ts'
import {
  attainmentPercent, computeForecast, forecastWeekRange, latestSubmission, submissionBeforePeriod, submissionHistory,
  submissionsOfPeriod,
} from './forecast.ts'

const stage = (ID: number, ForecastCategory: number): ICrmPipelineStage =>
  ({ ID, Name: `s${ID}`, Color: '', Probability: 0, StaleDays: 0, Status: 1, ForecastCategory })

describe('forecast', () => {
  it('sums open opportunities of the month by stage category and owner', () => {
    const january = periodFirstDay(2601) + 5
    const february = periodFirstDay(2602) + 5
    const totals = computeForecast([
      { OwnerUserID: 1, StageID: 1, Amount: 10, ExpectedCloseDate: january },
      { OwnerUserID: 1, StageID: 2, Amount: 20, ExpectedCloseDate: january },
      { OwnerUserID: 1, StageID: 3, Amount: 30, ExpectedCloseDate: january },
      { OwnerUserID: 1, StageID: 3, Amount: 99, ExpectedCloseDate: february },
      { OwnerUserID: 2, StageID: 3, Amount: 7, ExpectedCloseDate: january },
    ], [stage(1, 1), stage(2, 2), stage(3, 3)], 1, 2601, new Set([1]))
    expect(totals).toEqual({ commit: 30, bestCase: 20, pipeline: 10 })
  })

  it('can forecast a week instead of a month', () => {
    const monday = periodFirstDay(2601) + 6 - ((periodFirstDay(2601) + 6 + 3) % 7)
    const week = Math.floor((monday + 3) / 7)
    const totals = computeForecast([
      { OwnerUserID: 1, StageID: 3, Amount: 30, ExpectedCloseDate: monday + 6 },
      { OwnerUserID: 1, StageID: 3, Amount: 99, ExpectedCloseDate: monday + 7 },
    ], [stage(3, 3)], 2, week, new Set([1]))
    expect(totals.commit).toBe(30)
  })

  it('picks the newest submission of a seller or a team', () => {
    const make = (ID: number, UserID: number, TeamID: number, Created: number) => ({ ID, UserID, TeamID, Created }) as ICrmForecastSubmission
    const submissions = [make(1, 4, 0, 10), make(2, 4, 0, 20), make(3, 4, 7, 30), make(4, 5, 0, 40)]
    expect(latestSubmission(submissions, 4, 0)?.ID).toBe(2)
    expect(latestSubmission(submissions, 4, 7)?.ID).toBe(3)
    expect(latestSubmission(submissions, 9, 0)).toBeUndefined()
  })

  it('finds the last submission made before the period started', () => {
    // SUnixTime of noon of a UnixDay.
    const timeOfDay = (unixDay: number) => Math.floor((unixDay * 86400 + 43200 - 1e9) / 2)
    const firstDay = periodFirstDay(2602)
    const make = (ID: number, unixDay: number) => ({ ID, UserID: 4, TeamID: 0, Created: timeOfDay(unixDay) }) as ICrmForecastSubmission
    const history = submissionHistory([make(3, firstDay + 2), make(1, firstDay - 5), make(2, firstDay - 1)], 4, 0)
    expect(history.map((submission) => submission.ID)).toEqual([1, 2, 3])
    expect(submissionBeforePeriod(history, firstDay)?.ID).toBe(2)
    expect(submissionBeforePeriod(history.slice(2), firstDay)).toBeUndefined()
  })

  it('has no attainment without a goal', () => {
    expect(attainmentPercent(50, 200)).toBe(25)
    expect(attainmentPercent(50, 0)).toBe(0)
  })

  it('fetches a period as the weeks it spans, starting at the week its submissions are stored in', () => {
    const [monthStart, monthEnd] = forecastWeekRange(GOAL_PERIOD.MONTHLY, 2602)
    expect(monthStart).toBe(weekOfDay(periodFirstDay(2602)))
    expect(monthEnd).toBe(weekOfDay(periodLastDay(2602)))
    expect(forecastWeekRange(GOAL_PERIOD.WEEKLY, 2950)).toEqual([2950, 2950])
    const day = periodFirstDay(2602) + 3
    expect(forecastWeekRange(GOAL_PERIOD.DAILY_BY_MONTH, day)).toEqual([weekOfDay(day), weekOfDay(day)])
  })

  it('keeps only the period own submissions of the fetched weeks', () => {
    const submissions = [{ ID: 1, Period: 2602 }, { ID: 2, Period: 2950 }] as ICrmForecastSubmission[]
    expect(submissionsOfPeriod(submissions, 2602).map((submission) => submission.ID)).toEqual([1])
  })
})
