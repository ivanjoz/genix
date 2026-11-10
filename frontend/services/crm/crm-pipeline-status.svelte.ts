import { CrmCatalogService } from './crm-pipeline.svelte.ts'

// Mirrors backend/crm/types/pipeline_opportunity_status.go: the path of one opportunity through
// one pipeline, which the timeline (Gantt) report draws.

/** Visit outcomes: 0..2 as the opportunity's Outcome, plus "left for another pipeline". */
export const VISIT_OUTCOME_LEFT = 3

export interface ICrmOpportunityStageVisit {
  StageID: number
  Outcome: number
  /** Days since the row's StartDate. */
  DaysPassed: number
}

export interface ICrmPipelineOpportunityStatus {
  ID: number
  PipelineID: number
  OpportunityID: number
  /** UnixDay it entered the pipeline. */
  StartDate: number
  /** UnixDay of its last change in the pipeline. */
  LastDate: number
  Visits: ICrmOpportunityStageVisit[]
  ss: number
  upd: number
}

/** Every opportunity path of ONE pipeline, kept in sync by delta (only changed rows travel). */
export class CrmPipelineOpportunityStatusService extends CrmCatalogService<ICrmPipelineOpportunityStatus> {
  constructor(pipelineID: number) { super(`crm-pipeline-opportunity-status?pipeline-id=${pipelineID}`, true) }

  // colbin sends an empty list as null; every reader treats Visits as a list.
  handler(result: ICrmPipelineOpportunityStatus[]): void {
    super.handler((result || []).map((row) => ({ ...row, Visits: row.Visits ?? [] })))
  }
}
