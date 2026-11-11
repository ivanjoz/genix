import { GET, GETWithGroupCache } from '#libs/ui-runtime.svelte.ts'
import type { ICrmActivity } from './crm-activity.svelte.ts'
import type { ICrmOpportunityStageLog } from './crm-opportunity.svelte.ts'

// Connectors of backend/crm/report.go. Every CRM report is computed in the browser from the stage
// log (routes/crm/reports/reports.ts); these only fetch it.

/** GET.crm-report-scope: the pipelines the user may report on (crm/types/report_rules.go). */
export interface ICrmReportScope {
  PipelineIDs: number[]
  HoldsFullReport: boolean
}

export const getCrmReportScope = (): Promise<ICrmReportScope> => GET({ route: 'crm-report-scope' })

/**
 * The stage log of one pipeline for one calendar year (YY, e.g. 26). Cached per month: a closed
 * month never changes, so only the current one is downloaded again.
 */
export const getCrmReportStageLogOfYear = async (pipelineID: number, year: number): Promise<ICrmOpportunityStageLog[]> => {
  const groups = await GETWithGroupCache<ICrmOpportunityStageLog>('crm-report-stage-log', {
    'pipeline-id': String(pipelineID),
    'period-start': String(year * 100 + 1),
    'period-end': String(year * 100 + 12),
  })
  return groups.flatMap((group) => group.records || []).filter((stageLog) => (stageLog?.ID || 0) > 0)
}

/**
 * The activity entries written in one pipeline for one calendar year (YY), cached per month like the
 * stage log. Text and files are stripped by the server: the scorecard only counts them by author.
 */
export const getCrmReportActivitiesOfYear = async (pipelineID: number, year: number): Promise<ICrmActivity[]> => {
  const groups = await GETWithGroupCache<ICrmActivity>('crm-report-activities', {
    'pipeline-id': String(pipelineID),
    'period-start': String(year * 100 + 1),
    'period-end': String(year * 100 + 12),
  })
  return groups.flatMap((group) => group.records || []).filter((activity) => (activity?.ID || 0) > 0)
}

export interface ICrmReportOpenOpportunity {
  ID: number
  OwnerUserID: number
  StageID: number
  Amount: number
  /** UnixDay */
  ExpectedCloseDate: number
}

export const getCrmReportOpenOpportunities = (pipelineID: number): Promise<ICrmReportOpenOpportunity[]> =>
  GET({ route: `crm-report-open-opportunities?pipeline-id=${pipelineID}` })
