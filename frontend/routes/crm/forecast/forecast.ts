import type { ICrmForecastSubmission } from '#services/crm/crm-forecast.svelte.ts'
import { FORECAST_CATEGORY } from '#services/crm/crm-forecast-category.ts'
import type { ICrmPipelineStage } from '#services/crm/crm-pipeline.svelte.ts'
import { periodOfDay } from '../goals/goals.ts'
import { unixDayOfSUnixTime, type OpenOpportunityNow } from '../reports/reports.ts'

// Pure logic of the Forecast page (plan §3.4): what each seller's open opportunities add up to for
// a month, by the forecast category of their stage, mirrored from backend types.ComputeForecast.

export interface ForecastTotals {
  commit: number
  bestCase: number
  pipeline: number
}

export const emptyForecast = (): ForecastTotals => ({ commit: 0, bestCase: 0, pipeline: 0 })

/** Open opportunities expected to close in the goal period (month or week), summed by their stage's category. */
export const computeForecast = (
  openNow: OpenOpportunityNow[], stages: ICrmPipelineStage[], periodType: number, period: number, ownerIDs: Set<number>,
): ForecastTotals => {
  const categoryByStageID = new Map(stages.map((stage) => [stage.ID, stage.ForecastCategory]))
  const totals = emptyForecast()
  for (const opportunity of openNow) {
    if (!ownerIDs.has(opportunity.OwnerUserID) || !opportunity.ExpectedCloseDate) continue
    if (periodOfDay(periodType, opportunity.ExpectedCloseDate) !== period) continue
    const category = categoryByStageID.get(opportunity.StageID)
    if (category === FORECAST_CATEGORY.COMMIT) totals.commit += opportunity.Amount
    else if (category === FORECAST_CATEGORY.BEST_CASE) totals.bestCase += opportunity.Amount
    else totals.pipeline += opportunity.Amount
  }
  return totals
}

/** Every submission of a seller (teamID 0) or of a team, oldest first: how the forecast evolved (R6). */
export const submissionHistory = (submissions: ICrmForecastSubmission[], userID: number, teamID: number) =>
  submissions.filter((submission) => teamID ? submission.TeamID === teamID : submission.TeamID === 0 && submission.UserID === userID)
    .sort((left, right) => left.Created - right.Created || left.ID - right.ID)

/** The newest submission of a seller (teamID 0) or of a team, if any. */
export const latestSubmission = (submissions: ICrmForecastSubmission[], userID: number, teamID: number) =>
  submissionHistory(submissions, userID, teamID).at(-1)

/**
 * The last submission made before the period started (P7: there is no cut-off date, so the report
 * shows this one next to the latest). firstDay is the period's first UnixDay.
 */
export const submissionBeforePeriod = (history: ICrmForecastSubmission[], firstDay: number) =>
  history.filter((submission) => unixDayOfSUnixTime(submission.Created) < firstDay).at(-1)

/** Won / goal as a percentage; 0 when there is no goal. */
export const attainmentPercent = (won: number, goal: number) => goal > 0 ? Math.round(won * 100 / goal) : 0
