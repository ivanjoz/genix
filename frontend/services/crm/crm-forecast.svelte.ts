import { GETWithGroupCache, POST } from '#libs/ui-runtime.svelte.ts'

// Mirrors backend/crm/types/forecast.go.
export interface ICrmForecastSubmission {
  ID: number
  PipelineID: number
  UserID: number
  TeamID: number
  Period: number
  /** Week of the period's first day: the cache-group key the submissions are fetched by. */
  Week: number
  ComputedCommit: number
  ComputedBestCase: number
  ComputedPipeline: number
  ComputedClosed: number
  CommitAmount: number
  BestCaseAmount: number
  Note: string
  Created: number
}

export interface ICrmForecastSubmissionPayload {
  PipelineID: number
  Period: number
  TeamID: number
  CommitAmount: number
  BestCaseAmount: number
  Note: string
}

/**
 * The submissions of one pipeline stored in a range of weeks: one cache group per week, so a past
 * week is downloaded once. The caller asks for the weeks its goal period spans and keeps the
 * period's own rows (routes/crm/forecast/forecast.ts).
 */
export const getCrmForecastSubmissionsOfWeeks = async (
  pipelineID: number, weekStart: number, weekEnd: number,
): Promise<ICrmForecastSubmission[]> => {
  const groups = await GETWithGroupCache<ICrmForecastSubmission>('crm-forecast-submissions', {
    'pipeline-id': String(pipelineID),
    'week-start': String(weekStart),
    'week-end': String(weekEnd),
  })
  return groups.flatMap((group) => group.records || []).filter((submission) => (submission?.ID || 0) > 0)
}

export const postCrmForecastSubmission = (payload: ICrmForecastSubmissionPayload): Promise<ICrmForecastSubmission> =>
  POST({ route: 'crm-forecast-submission', data: payload })
