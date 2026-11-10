import { notifyFailure } from '@genix/ui/notify'
import { todayUnixDay } from '#libs/helpers.ts'
import { GOAL_PERIOD } from '#services/crm/crm-goal-metric.ts'
import { CrmGoalsService } from '#services/crm/crm-goal.svelte.ts'
import {
  getCrmForecastSubmissions, postCrmForecastSubmission,
  type ICrmForecastSubmission, type ICrmForecastSubmissionPayload,
} from '#services/crm/crm-forecast.svelte.ts'
import { CrmTeamsService } from '#services/crm/crm-team.svelte.ts'
import { periodOfDay, periodsOfYear } from '../goals/goals.ts'
import { CrmReportsState, currentShortYear } from '../reports/reports.svelte.ts'

/**
 * Data of the Forecast page. It reuses the reports state for scope, pipelines, users, today's open
 * opportunities and the stage log of the month (what is already won), and adds goals, teams and
 * the forecasts submitted for the month.
 */
export class CrmForecastState {
  report = new CrmReportsState()
  teams = new CrmTeamsService(true)
  goals = new CrmGoalsService(true)

  /** The goal period being forecast: a YYMM month or a week, by the pipeline's GoalPeriodType. */
  period = $state(0)
  submissions = $state<ICrmForecastSubmission[]>([])

  periodType = $derived(this.report.selectedPipeline?.GoalPeriodType || GOAL_PERIOD.MONTHLY)
  periods = $derived(periodsOfYear(this.periodType, this.report.year))

  /** Picks the period holding today, or the year's first one when today is in another year. */
  selectDefaultPeriod() {
    const todayPeriod = periodOfDay(this.periodType, todayUnixDay())
    this.period = this.periods.some((candidate) => candidate.period === todayPeriod) ? todayPeriod : (this.periods[0]?.period ?? 0)
  }

  async load() {
    // The stage log from the selected year to today holds the period's wins; the page filters it by day.
    await this.report.load()
    await this.loadSubmissions()
  }

  async loadSubmissions() {
    if (!this.report.selectedPipelineID) return
    try {
      this.submissions = await getCrmForecastSubmissions(this.report.selectedPipelineID, this.period)
    } catch (error) {
      notifyFailure(error as string)
    }
  }

  async submit(payload: Omit<ICrmForecastSubmissionPayload, 'PipelineID' | 'Period'>) {
    await postCrmForecastSubmission({ ...payload, PipelineID: this.report.selectedPipelineID, Period: this.period })
    await this.loadSubmissions()
  }
}

export { currentShortYear }
