import { GET, POST } from '#libs/ui-runtime.svelte.ts'

// Mirrors backend/crm/types/forecast.go.
export interface ICrmForecastSubmission {
  ID: number
  PipelineID: number
  UserID: number
  TeamID: number
  Period: number
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

export const getCrmForecastSubmissions = (pipelineID: number, period: number): Promise<ICrmForecastSubmission[]> =>
  GET({ route: `crm-forecast-submissions?pipeline-id=${pipelineID}&period=${period}` })

export const postCrmForecastSubmission = (payload: ICrmForecastSubmissionPayload): Promise<ICrmForecastSubmission> =>
  POST({ route: 'crm-forecast-submission', data: payload })
