import { getRecordsByID } from '@genix/ui/cache'
import { notifyFailure } from '@genix/ui/notify'
import { todayUnixDay } from '#libs/helpers.ts'
import { security } from '#libs/ui-runtime.svelte.ts'
import type { ICrmOpportunity } from '#services/crm/crm-opportunity.svelte.ts'
import { CrmPipelinesService } from '#services/crm/crm-pipeline.svelte.ts'
import { CrmPipelineOpportunityStatusService } from '#services/crm/crm-pipeline-status.svelte.ts'
import { UsuariosService } from '../../security/users-profiles/users-profiles.svelte'
import { ACCESO_ID_CRM_FULL_REPORT, reportablePipelineIDs } from '../reports/reports.ts'
import { buildTimelineCards, filterTimelineCards, groupCardsBySeller, type OpportunityInfo, type TimelineDateMode } from './timeline.ts'

/** Days shown by default, today included. */
const DEFAULT_WINDOW_DAYS = 45

/**
 * State of the timeline page. The pipeline's opportunity paths sync by delta; the titles and
 * owners are joined through the records-by-ID cache, asking only for the IDs on screen.
 */
export class CrmTimelineState {
  pipelines = new CrmPipelinesService(true)
  users = new UsuariosService()
  today = todayUnixDay()

  userID = security.getUserInfo()?.ID || 0
  /** The pipelines the user may report on, from the cached pipelines list (no request of its own). */
  reportPipelineIDs = $derived(reportablePipelineIDs(this.pipelines.records, this.userID, security.checkAcceso(ACCESO_ID_CRM_FULL_REPORT)))
  selectedPipelineID = $state(0)
  statusRows = $state<CrmPipelineOpportunityStatusService | undefined>()
  opportunities = $state(new Map<number, OpportunityInfo>())
  filters = $state({ fromDay: todayUnixDay() - (DEFAULT_WINDOW_DAYS - 1), toDay: todayUnixDay(), mode: 'start' as TimelineDateMode })
  ownerIDs = $state<number[]>([])

  selectedPipeline = $derived(this.pipelines.records.find((pipeline) => pipeline.ID === this.selectedPipelineID))
  stageColors = $derived(new Map((this.selectedPipeline?.Stages ?? []).map((stage) => [stage.ID, stage.Color])))
  cards = $derived(buildTimelineCards(this.statusRows?.records ?? [], this.opportunities, this.stageColors, this.today))
  visibleCards = $derived(filterTimelineCards(this.cards, this.filters.mode, this.filters.fromDay, this.filters.toDay, this.ownerIDs))
  sellers = $derived(groupCardsBySeller(this.visibleCards, (userID) => this.ownerName(userID)))

  /** Picks the first reportable pipeline once the pipelines list arrives, unless one is chosen. */
  selectDefaultPipeline() {
    if (!this.selectedPipelineID && this.reportPipelineIDs.length) this.selectedPipelineID = this.reportPipelineIDs[0]
  }

  /** Opens the pipeline's delta cache; a new pipeline is a new cache. */
  selectPipeline(pipelineID: number) {
    this.statusRows = new CrmPipelineOpportunityStatusService(pipelineID)
  }

  /** Resolves the title and owner of every opportunity of the pipeline not resolved yet. */
  async resolveOpportunities(opportunityIDs: number[]) {
    const missingIDs = [...new Set(opportunityIDs)].filter((opportunityID) => !this.opportunities.has(opportunityID))
    if (!missingIDs.length) return
    try {
      const records = await getRecordsByID<ICrmOpportunity>('crm-opportunity-ids', missingIDs)
      const resolved = new Map(this.opportunities)
      for (const [opportunityID, record] of records) resolved.set(opportunityID, { Title: record.Title, OwnerUserID: record.OwnerUserID })
      this.opportunities = resolved
    } catch (error) {
      notifyFailure(error as string)
    }
  }

  ownerName(userID: number) {
    const user = this.users.usuariosMap.get(userID)
    return user ? `${user.FirstName} ${user.LastName}`.trim() : `#${userID}`
  }
}
