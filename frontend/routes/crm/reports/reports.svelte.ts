import { notifyFailure } from '@genix/ui/notify'
import { todayUnixDay } from '#libs/helpers.ts'
import { security } from '#libs/ui-runtime.svelte.ts'
import type { ICrmOpportunityStageLog } from '#services/crm/crm-opportunity.svelte.ts'
import type { ICrmActivity } from '#services/crm/crm-activity.svelte.ts'
import { CrmOpportunitiesService } from '#services/crm/crm-opportunity.svelte.ts'
import { CrmGoalsService } from '#services/crm/crm-goal.svelte.ts'
import { CrmLostReasonsService, CrmPipelinesService } from '#services/crm/crm-pipeline.svelte.ts'
import { getCrmReportActivitiesOfYear, getCrmReportStageLogOfYear } from '#services/crm/crm-reports.svelte.ts'
import { CrmTeamsService } from '#services/crm/crm-team.svelte.ts'
import { UsuariosService } from '../../security/users-profiles/users-profiles.svelte'
import { ACCESO_ID_CRM_FULL_REPORT, periodOfUnixDay, reportablePipelineIDs } from './reports.ts'

/** Two-digit year of today, e.g. 26. */
export const currentShortYear = () => Math.floor(periodOfUnixDay(todayUnixDay()) / 100)

/**
 * Filters and data of the CRM reports page. The stage log is loaded per pipeline and year (each
 * month a cache group); the daily state also needs every later year up to today, because it is
 * rebuilt backwards from today's open opportunities.
 */
export class CrmReportsState {
  pipelines = new CrmPipelinesService(true)
  users = new UsuariosService()
  // Small whole-table catalogs: team subtotals and goals (scorecard, forecast), lost reason names.
  teams = new CrmTeamsService(true)
  goals = new CrmGoalsService(true)
  lostReasons = new CrmLostReasonsService(true)

  userID = security.getUserInfo()?.ID || 0
  /** The pipelines the user may report on, from the cached pipelines list (no request of its own). */
  reportPipelineIDs = $derived(reportablePipelineIDs(this.pipelines.records, this.userID, security.checkAcceso(ACCESO_ID_CRM_FULL_REPORT)))
  selectedPipelineID = $state(0)
  /** YYMM range of the report; defaults to the whole current year. */
  periodFrom = $state(currentShortYear() * 100 + 1)
  periodTo = $state(currentShortYear() * 100 + 12)
  showAmounts = $state(false)
  ownerIDs = $state<number[]>([])

  /** Stage log rows of the selected pipeline by year, as loaded. */
  rowsByYear = $state(new Map<number, ICrmOpportunityStageLog[]>())
  /** The board's delta list of the selected pipeline: today's open opportunities, the anchor of the daily state. */
  openOpportunities = $state<CrmOpportunitiesService | undefined>()
  openNow = $derived(this.openOpportunities?.records ?? [])
  /** Activity entries of the selected pipeline by year: only the scorecard loads them. */
  activitiesByYear = $state(new Map<number, ICrmActivity[]>())
  isLoading = $state(false)

  selectedPipeline = $derived(this.pipelines.records.find((pipeline) => pipeline.ID === this.selectedPipelineID))
  /** Year of the range start: the stage log is loaded from it through the current year. */
  yearFrom = $derived(Math.floor(this.periodFrom / 100))
  /** Rows from the start year to today: what the daily state walks back through. */
  rowsToToday = $derived([...this.rowsByYear].filter(([year]) => year >= this.yearFrom).flatMap(([, rows]) => rows))
  /** Rows inside the selected periods (the range may span several years). */
  rowsInRange = $derived(this.rowsToToday.filter((row) => row.Period >= this.periodFrom && row.Period <= this.periodTo))
  activitiesInRange = $derived([...this.activitiesByYear].flatMap(([, activities]) => activities)
    .filter((activity) => activity.Period >= this.periodFrom && activity.Period <= this.periodTo))

  /** Picks the first reportable pipeline once the pipelines list arrives, unless one is chosen. */
  selectDefaultPipeline() {
    if (!this.selectedPipelineID && this.reportPipelineIDs.length) this.selectedPipelineID = this.reportPipelineIDs[0]
  }

  /** Loads the selected pipeline from the start year through the current one (later years have no log yet). */
  async load() {
    if (!this.selectedPipelineID) return
    const pipelineID = this.selectedPipelineID
    const years = Array.from({ length: currentShortYear() - this.yearFrom + 1 }, (_, index) => this.yearFrom + index)
    // Each pipeline is its own delta cache: opening it again only downloads what changed.
    if (this.openOpportunities?.route !== `crm-opportunities?pipeline-id=${pipelineID}`) {
      this.openOpportunities = new CrmOpportunitiesService(pipelineID)
    }
    this.isLoading = true
    try {
      const yearRows = await Promise.all(years.map((year) => getCrmReportStageLogOfYear(pipelineID, year)))
      // A pipeline switched while loading discards this answer.
      if (pipelineID !== this.selectedPipelineID) return
      this.rowsByYear = new Map(years.map((year, index) => [year, yearRows[index] as ICrmOpportunityStageLog[]]))
    } catch (error) {
      notifyFailure(error as string)
    } finally {
      this.isLoading = false
    }
  }

  /** Loads the activities of the range's years (never past the current one) for the scorecard. */
  async loadActivities() {
    if (!this.selectedPipelineID) return
    const pipelineID = this.selectedPipelineID
    const lastYear = Math.min(Math.floor(this.periodTo / 100), currentShortYear())
    const years = Array.from({ length: Math.max(0, lastYear - this.yearFrom + 1) }, (_, index) => this.yearFrom + index)
    try {
      const yearActivities = await Promise.all(years.map((year) => getCrmReportActivitiesOfYear(pipelineID, year)))
      if (pipelineID !== this.selectedPipelineID) return
      this.activitiesByYear = new Map(years.map((year, index) => [year, yearActivities[index]]))
    } catch (error) {
      notifyFailure(error as string)
    }
  }

  ownerName(userID: number) {
    const user = this.users.usuariosMap.get(userID)
    return user ? `${user.FirstName} ${user.LastName}`.trim() : `#${userID}`
  }
}

let sharedReportsState: CrmReportsState | undefined
/**
 * The three report pages (flow, conversion, performance) share one state, so the pipeline, range,
 * sellers and loaded rows survive moving between them. Another user logging in starts a fresh one.
 */
export const getCrmReportsState = (): CrmReportsState => {
  const userID = security.getUserInfo()?.ID || 0
  if (!sharedReportsState || sharedReportsState.userID !== userID) sharedReportsState = new CrmReportsState()
  return sharedReportsState
}
