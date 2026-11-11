import { GETWithGroupCache } from '#libs/ui-runtime.svelte.ts'
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
  /** The approval task the card waits on (0 = none): the card cannot move or close meanwhile. */
  PendingApprovalTaskID: number
  CreatedBy: number
  ss: number
  upd: number
}

/** Mirrors crm/types/stage_log.go: each row carries the reportable state before and after. */
export interface ICrmOpportunityStageLog {
  ID: number
  OpportunityID: number
  Kind: number
  Period: number
  Time: number
  UserID: number
  FromPipelineID: number
  FromStageID: number
  FromPosition: number
  FromOutcome: number
  FromOwnerUserID: number
  FromAmount: number
  FromExpectedCloseDate: number
  ToPipelineID: number
  ToStageID: number
  ToPosition: number
  ToOutcome: number
  ToOwnerUserID: number
  ToAmount: number
  ToExpectedCloseDate: number
  LostReasonID: number
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

/** The opportunity's stage history: one cache group per opportunity, downloaded again only after a new change. */
export const getCrmOpportunityLog = async (opportunityID: number): Promise<ICrmOpportunityStageLog[]> => {
  const groups = await GETWithGroupCache<ICrmOpportunityStageLog>('crm-opportunity-log', { 'opportunity-id': String(opportunityID) })
  return groups.flatMap((group) => group.records || []).filter((stageLog) => (stageLog?.ID || 0) > 0)
}
