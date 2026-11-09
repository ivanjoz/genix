import { notifyFailure } from '@genix/ui/notify'
import { security } from '#libs/ui-runtime.svelte.ts'
import { ClientProviderService } from '#services/crm/client-provider.svelte.ts'
import { CrmActivityTypesService } from '#services/crm/crm-activity.svelte.ts'
import {
  CrmOpportunitiesService, postCrmOpportunity, postCrmOpportunityClose, postCrmOpportunityMove,
  type ICrmOpportunity,
} from '#services/crm/crm-opportunity.svelte.ts'
import {
  CrmCustomFieldsService, CrmLostReasonsService, CrmPipelinesService,
} from '#services/crm/crm-pipeline.svelte.ts'
import { UsuariosService } from '../../security/users-profiles/users-profiles.svelte'
import { activityTypeOptions, memberLevel, visiblePipelines } from './pipeline.ts'

const LAST_PIPELINE_STORAGE_KEY = 'crm-last-pipeline'

/** Board state: the catalogs, the selected pipeline and its cards, and the actions on a card. */
export class PipelineBoard {
  pipelines = new CrmPipelinesService(true)
  customFields = new CrmCustomFieldsService(true)
  lostReasons = new CrmLostReasonsService(true)
  activityTypes = new CrmActivityTypesService(true)
  clients = new ClientProviderService(1, true)
  users = new UsuariosService()
  userID = security.getUserInfo()?.ID || 0

  selectedPipelineID = $state(0)
  opportunities = $state<CrmOpportunitiesService | undefined>()

  visiblePipelines = $derived(visiblePipelines(this.pipelines.records, this.userID))
  selectedPipeline = $derived(this.visiblePipelines.find((pipeline) => pipeline.ID === this.selectedPipelineID))
  level = $derived(memberLevel(this.selectedPipeline, this.userID))
  customFieldsMap = $derived(new Map(this.customFields.records.map((customField) => [customField.ID, customField])))
  activityTypeOptions = $derived(activityTypeOptions(this.activityTypes.records))

  /** Opens the remembered pipeline, or the first one the user can see. */
  selectInitialPipeline() {
    const rememberedID = Number(localStorage.getItem(LAST_PIPELINE_STORAGE_KEY))
    const initial = this.visiblePipelines.find((pipeline) => pipeline.ID === rememberedID) ?? this.visiblePipelines[0]
    if (initial) this.selectPipeline(initial.ID)
  }

  selectPipeline(pipelineID: number) {
    this.selectedPipelineID = pipelineID
    localStorage.setItem(LAST_PIPELINE_STORAGE_KEY, String(pipelineID))
    this.opportunities = new CrmOpportunitiesService(pipelineID)
  }

  // Each write below refreshes the board's cards from the delta its own response carries
  // (GetHandler.writeAndSync): there is no GET after it.

  /** Moves the card at once and puts it back if the server refuses (permission, required fields). */
  async moveOpportunity(opportunity: ICrmOpportunity, toPipelineID: number, toStageID: number) {
    if (!this.opportunities) return
    const previousStageID = opportunity.StageID
    if (toPipelineID === opportunity.PipelineID) opportunity.StageID = toStageID
    try {
      await postCrmOpportunityMove(this.opportunities, opportunity.ID, toPipelineID, toStageID)
    } catch (error) {
      opportunity.StageID = previousStageID
      notifyFailure(error as string)
    }
  }

  async closeOpportunity(opportunityID: number, outcome: number, lostReasonID = 0, lostNote = '') {
    if (!this.opportunities) return
    await postCrmOpportunityClose(this.opportunities, opportunityID, outcome, lostReasonID, lostNote)
  }

  async saveOpportunity(opportunity: Partial<ICrmOpportunity>) {
    if (!this.opportunities) return
    return await postCrmOpportunity(this.opportunities, opportunity)
  }
}
