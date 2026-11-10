import { notifyFailure } from '@genix/ui/notify'
import { todayUnixDay } from '#libs/helpers.ts'
import type { ICrmOpportunityStageLog } from '#services/crm/crm-opportunity.svelte.ts'
import { CrmPipelinesService } from '#services/crm/crm-pipeline.svelte.ts'
import {
  getCrmReportOpenOpportunities, getCrmReportScope, getCrmReportStageLogOfYear,
  type ICrmReportOpenOpportunity, type ICrmReportScope,
} from '#services/crm/crm-reports.svelte.ts'
import { UsuariosService } from '../../security/users-profiles/users-profiles.svelte'
import { periodOfUnixDay } from './reports.ts'

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

  scope = $state<ICrmReportScope>({ PipelineIDs: [], HoldsFullReport: false })
  selectedPipelineID = $state(0)
  year = $state(currentShortYear())
  monthFrom = $state(1)
  monthTo = $state(12)
  showAmounts = $state(false)
  ownerIDs = $state<number[]>([])

  /** Stage log rows of the selected pipeline by year, as loaded. */
  rowsByYear = $state(new Map<number, ICrmOpportunityStageLog[]>())
  openNow = $state<ICrmReportOpenOpportunity[]>([])
  isLoading = $state(false)

  selectedPipeline = $derived(this.pipelines.records.find((pipeline) => pipeline.ID === this.selectedPipelineID))
  periodFrom = $derived(this.year * 100 + this.monthFrom)
  periodTo = $derived(this.year * 100 + this.monthTo)
  /** Rows of the selected year inside the selected months. */
  rowsInRange = $derived((this.rowsByYear.get(this.year) ?? []).filter((row) => row.Period >= this.periodFrom && row.Period <= this.periodTo))
  /** Rows from the selected year to today: what the daily state walks back through. */
  rowsToToday = $derived([...this.rowsByYear].filter(([year]) => year >= this.year).flatMap(([, rows]) => rows))

  async loadScope() {
    try {
      this.scope = await getCrmReportScope()
      if (!this.selectedPipelineID && this.scope.PipelineIDs.length) this.selectedPipelineID = this.scope.PipelineIDs[0]
    } catch (error) {
      notifyFailure(error as string)
    }
  }

  /** Loads the selected pipeline from the selected year through the current one. */
  async load() {
    if (!this.selectedPipelineID) return
    const pipelineID = this.selectedPipelineID
    const years = Array.from({ length: currentShortYear() - this.year + 1 }, (_, index) => this.year + index)
    this.isLoading = true
    try {
      const [openNow, ...yearRows] = await Promise.all([
        getCrmReportOpenOpportunities(pipelineID),
        ...years.map((year) => getCrmReportStageLogOfYear(pipelineID, year)),
      ])
      // A pipeline switched while loading discards this answer.
      if (pipelineID !== this.selectedPipelineID) return
      this.openNow = openNow as ICrmReportOpenOpportunity[]
      this.rowsByYear = new Map(years.map((year, index) => [year, yearRows[index] as ICrmOpportunityStageLog[]]))
    } catch (error) {
      notifyFailure(error as string)
    } finally {
      this.isLoading = false
    }
  }

  ownerName(userID: number) {
    const user = this.users.usuariosMap.get(userID)
    return user ? `${user.FirstName} ${user.LastName}`.trim() : `#${userID}`
  }
}
