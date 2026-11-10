import { describe, expect, it } from 'vitest'
import type { ICrmForecastSubmission } from '#services/crm/crm-forecast.svelte.ts'
import type { ICrmPipelineStage } from '#services/crm/crm-pipeline.svelte.ts'
import { periodFirstDay } from '../reports/reports.ts'
import { attainmentPercent, computeForecast, latestSubmission } from './forecast.ts'

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

  it('has no attainment without a goal', () => {
    expect(attainmentPercent(50, 200)).toBe(25)
    expect(attainmentPercent(50, 0)).toBe(0)
  })
})
