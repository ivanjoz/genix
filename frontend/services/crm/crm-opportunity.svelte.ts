import { GET } from '#libs/ui-runtime.svelte.ts'
import { CrmCatalogService } from './crm-pipeline.svelte.ts'

// Mirrors backend/crm/types/opportunity.go and stage_log.go.

export const OpportunityOutcome = { OPEN: 0, WON: 1, LOST: 2 } as const

export interface ICrmCustomFieldValue {
  FieldID: number
  Number: number
  Text: string
  OptionIDs: number[]
}

export interface ICrmOpportunity {
  ID: number
  PipelineID: number
  StageID: number
  Title: string
  ClientID: number
  OwnerUserID: number
  Amount: number
  Probability: number
  ExpectedCloseDate: number
  Outcome: number
  LostReasonID: number
  LostNote: string
  ClosedTime: number
  StageEnteredTime: number
  PreviousPipelineID: number
  PreviousStageID: number
  CustomValues: ICrmCustomFieldValue[]
  CreatedBy: number
  ss: number
  upd: number
}

export interface ICrmOpportunityStageLog {
  ID: number
  OpportunityID: number
  FromPipelineID: number
  FromStageID: number
  ToPipelineID: number
  ToStageID: number
  Outcome: number
  Time: number
  UserID: number
}

/** The open opportunities of ONE pipeline: each pipeline is its own delta cache, and a card that
 *  is closed or moved elsewhere reaches the browser as a removed ID. */
const crmOpportunitiesRoute = (pipelineID: number) => `crm-opportunities?pipeline-id=${pipelineID}`

export class CrmOpportunitiesService extends CrmCatalogService<ICrmOpportunity> {
  constructor(pipelineID: number) { super(crmOpportunitiesRoute(pipelineID), true) }
}

// Every write goes through the board's opportunities service: the write brings back the delta of
// that pipeline's cache in its own response (GetHandler.writeAndSync), so no GET follows it. A move
// to another pipeline also marks the target pipeline's cache, read when the user opens it.
export const postCrmOpportunity = (opportunities: CrmOpportunitiesService, opportunity: Partial<ICrmOpportunity>) =>
  opportunities.writeAndSync<ICrmOpportunity>('crm-opportunity', opportunity)

export const postCrmOpportunityMove = (
  opportunities: CrmOpportunitiesService, opportunityID: number, pipelineID: number, stageID: number,
) =>
  opportunities.writeAndSync<ICrmOpportunity>(
    'crm-opportunity-move', { ID: opportunityID, PipelineID: pipelineID, StageID: stageID },
    [crmOpportunitiesRoute(pipelineID)],
  )

export const postCrmOpportunityClose = (
  opportunities: CrmOpportunitiesService, opportunityID: number, outcome: number, lostReasonID: number, lostNote: string,
) =>
  opportunities.writeAndSync<ICrmOpportunity>(
    'crm-opportunity-close', { ID: opportunityID, Outcome: outcome, LostReasonID: lostReasonID, LostNote: lostNote },
  )

export const getCrmOpportunityLog = (opportunityID: number): Promise<ICrmOpportunityStageLog[]> =>
  GET({ route: `crm-opportunity-log?opportunity-id=${opportunityID}` })
