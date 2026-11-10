<script lang="ts">
  import { untrack } from 'svelte'
  import { useUI } from '@genix/ui'
  import { hideLoading, notifyFailure, notifySuccess, showLoading } from '@genix/ui/notify'
  import Input from '#components/form/Input.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import Modal from '#components/layers/Modal.svelte'
  import Button from '#components/buttons/Button.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import Page from '#domain/Page.svelte'
  import { tr } from '#core/store.svelte.ts'
  import { formatN, formatTime } from '#libs/helpers.ts'
  import { GOAL_METRIC, GOAL_PERIOD } from '#services/crm/crm-goal-metric.ts'
  import { security } from '#libs/ui-runtime.svelte.ts'
  import { activeStagesOf } from '../pipeline/pipeline.ts'
  import { cellValue, goalGridRows, goalPeriodDays, isDailyPeriodType, type GoalGridRow } from '../goals/goals.ts'
  import { aggregateNetProgress, unixDayOfSUnixTime, WON_COLUMN } from '../reports/reports.ts'
  import { attainmentPercent, computeForecast, latestSubmission, type ForecastTotals } from './forecast.ts'
  import { CrmForecastState, currentShortYear } from './forecast.svelte.ts'

  // Forecast of the month per seller and team: goal, what is already won, the open pipeline by
  // forecast category, and the last forecast the seller or the leader submitted (plan §3.4).
  const MODAL_ID = 1
  const ui = useUI()
  const forecast = new CrmForecastState()
  const report = forecast.report
  const userID = security.getUserInfo()?.ID || 0

  const MONTH_NAMES = ['Jan|Ene', 'Feb|Feb', 'Mar|Mar', 'Apr|Abr', 'May|May', 'Jun|Jun', 'Jul|Jul', 'Aug|Ago', 'Sep|Set', 'Oct|Oct', 'Nov|Nov', 'Dec|Dic']
  // Months, or weeks labelled by their number and Monday, as the pipeline's goals are set.
  const periodOptions = $derived(forecast.periods.map(({ period, firstDay }, index) => ({
    id: period,
    name: isDailyPeriodType(forecast.periodType) ? (formatTime(firstDay, 'd-M-Y') as string)
      : forecast.periodType === GOAL_PERIOD.WEEKLY ? `${index + 1} · ${formatTime(firstDay, 'd-M')}` : tr(MONTH_NAMES[index]),
  })))
  const yearOptions = Array.from({ length: 4 }, (_, index) => currentShortYear() + 1 - index)
    .map((year) => ({ id: year, name: String(2000 + year) }))
  const pipelineOptions = $derived(report.scope.PipelineIDs.map((pipelineID) => ({
    id: pipelineID, name: report.pipelines.records.find((pipeline) => pipeline.ID === pipelineID)?.Name ?? `Pipeline #${pipelineID}`,
  })))

  report.loadScope()
  $effect(() => {
    if (!report.selectedPipelineID) return
    void [report.year, report.selectedPipelineID, report.selectedPipeline?.GoalPeriodType]
    untrack(() => { forecast.selectDefaultPeriod(); forecast.load() })
  })
  // Another period of the same year only reloads that period's submissions.
  const selectPeriod = (period: number) => { forecast.period = period; forecast.load() }

  const memberUserIDs = $derived(report.selectedPipeline?.Members.map((member) => member.UserID) ?? [])
  const gridRows = $derived(goalGridRows(forecast.teams.records, memberUserIDs, (id) => report.ownerName(id)))

  interface ForecastRow {
    row: GoalGridRow
    goal: number
    won: number
    totals: ForecastTotals
    lastCommit: number | undefined
    canSubmit: boolean
  }

  // What was won inside the period: the stage log rows of its days, whatever their month.
  const wonByOwner = $derived.by(() => {
    const [firstDay, lastDay] = goalPeriodDays(forecast.periodType, forecast.period)
    const periodRows = report.rowsToToday.filter((row) => {
      const unixDay = unixDayOfSUnixTime(row.Time)
      return unixDay >= firstDay && unixDay <= lastDay
    })
    const wonAmounts = new Map<number, number>()
    for (const cell of aggregateNetProgress(periodRows, report.selectedPipelineID, 'period')) {
      if (cell.column === WON_COLUMN) wonAmounts.set(cell.ownerID, (wonAmounts.get(cell.ownerID) ?? 0) + cell.amount)
    }
    return wonAmounts
  })

  const forecastRows = $derived(gridRows.map((row): ForecastRow => {
    const owners = row.isTeamRow
      ? gridRows.filter((candidate) => !candidate.isTeamRow && candidate.teamID === row.teamID).map((candidate) => candidate.userID)
      : [row.userID]
    const goalOf = (ownerID: number) =>
      cellValue(new Map(), forecast.goals.records, report.selectedPipelineID, GOAL_METRIC.WON_AMOUNT, ownerID, forecast.period)
    const lastSubmission = latestSubmission(forecast.submissions, row.userID, row.isTeamRow ? row.teamID : 0)
    const leaderID = forecast.teams.records.find((team) => team.ID === row.teamID)?.LeaderUserID
    return {
      row,
      goal: owners.reduce((total, ownerID) => total + goalOf(ownerID), 0),
      won: owners.reduce((total, ownerID) => total + (wonByOwner.get(ownerID) ?? 0) / 100, 0),
      totals: computeForecast(report.openNow, report.selectedPipeline ? activeStagesOf(report.selectedPipeline) : [],
        forecast.periodType, forecast.period, new Set(owners)),
      lastCommit: lastSubmission ? lastSubmission.CommitAmount / 100 : undefined,
      canSubmit: row.isTeamRow ? leaderID === userID : row.userID === userID,
    }
  }))
  const money = (value: number) => formatN(value, 2)
  const inCurrency = (amount: number) => amount / 100

  const columns: ITableColumn<ForecastRow>[] = [
    { header: 'Seller|Vendedor', css: 'px-6', highlight: true, getValue: (item) => item.row.label,
      setCellCss: (item) => item.row.isTeamRow ? 'ff-bold' : undefined },
    { header: 'Goal|Meta', css: 'px-6 text-right ff-mono', getValue: (item) => money(item.goal) },
    { header: 'Won|Ganado', css: 'px-6 text-right ff-mono', getValue: (item) => money(item.won) },
    { header: '% goal|% meta', css: 'px-6 text-right ff-mono', getValue: (item) => `${attainmentPercent(item.won, item.goal)}%` },
    { header: 'Commit|Compromiso', css: 'px-6 text-right ff-mono', getValue: (item) => money(inCurrency(item.totals.commit)) },
    { header: 'Best case|Mejor caso', css: 'px-6 text-right ff-mono', getValue: (item) => money(inCurrency(item.totals.bestCase)) },
    { header: 'Pipeline', css: 'px-6 text-right ff-mono', getValue: (item) => money(inCurrency(item.totals.pipeline)) },
    { header: 'Won + commit|Ganado + compromiso', css: 'px-6 text-right ff-mono',
      getValue: (item) => money(item.won + inCurrency(item.totals.commit)),
      setCellCss: (item) => item.goal > 0 && item.won + inCurrency(item.totals.commit) < item.goal ? 'text-red-500' : undefined },
    { header: 'Last submitted|Último enviado', css: 'px-6 text-right ff-mono',
      getValue: (item) => item.lastCommit === undefined ? '—' : money(item.lastCommit) },
    { header: '', css: 'px-6', getValue: () => '', buttonEditIf: (item) => item.canSubmit, buttonEditHandler: (item) => openSubmit(item) },
  ]

  // The modal edits one submission; the amounts start from the computed forecast, in currency units.
  let submitTarget = $state<ForecastRow | null>(null)
  let submitForm = $state({ CommitAmount: 0, BestCaseAmount: 0, Note: '' })
  const openSubmit = (item: ForecastRow) => {
    submitTarget = item
    submitForm = { CommitAmount: inCurrency(item.totals.commit), BestCaseAmount: inCurrency(item.totals.bestCase), Note: '' }
    ui.openModal(MODAL_ID)
  }
  const confirmSubmit = async () => {
    if (!submitTarget) return
    showLoading(tr('Submitting forecast...|Enviando forecast...'))
    try {
      await forecast.submit({
        TeamID: submitTarget.row.isTeamRow ? submitTarget.row.teamID : 0,
        // Stored in the opportunities' amount scale.
        CommitAmount: Math.round(Number(submitForm.CommitAmount) * 100),
        BestCaseAmount: Math.round(Number(submitForm.BestCaseAmount) * 100),
        Note: submitForm.Note,
      })
      ui.closeModal(MODAL_ID)
      notifySuccess(tr('Forecast submitted|Forecast enviado'))
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }
</script>

<Page title="CRM Forecast|Forecast CRM">
  <div class="flex items-end gap-12 flex-wrap mb-12" aria-label="Forecast filters">
    <SearchSelect options={pipelineOptions} keyId="id" keyName="name" label="Pipeline" css="w-240"
      selected={report.selectedPipelineID} onChange={(option) => { report.selectedPipelineID = option.id }} />
    <SearchSelect options={yearOptions} keyId="id" keyName="name" label="Year|Año" css="w-120"
      selected={report.year} onChange={(option) => { report.year = option.id }} />
    <SearchSelect options={periodOptions} keyId="id" keyName="name" css="w-160"
      label={isDailyPeriodType(forecast.periodType) ? 'Day|Día' : forecast.periodType === GOAL_PERIOD.WEEKLY ? 'Week|Semana' : 'Month|Mes'}
      selected={forecast.period} onChange={(option) => selectPeriod(option.id)} />
    {#if report.isLoading}<span class="text-fg-muted">{tr('Loading...|Cargando...')}</span>{/if}
  </div>
  <div class="text-fg-muted mb-8">
    {tr('Open opportunities expected to close in the month, by the forecast category of their stage. Each seller submits their own forecast; a team leader submits the team row.|Oportunidades abiertas que se esperan cerrar en el mes, según la categoría de forecast de su etapa. Cada vendedor envía su forecast; el líder envía la fila del equipo.')}
  </div>
  {#if !report.scope.PipelineIDs.length}
    <div class="text-fg-muted p-16">{tr('You have no pipelines to report on.|No tiene pipelines para reportar.')}</div>
  {:else}
    <VTable {columns} data={forecastRows} css="w-full" maxHeight="calc(100vh - 12rem)" />
  {/if}

  <Modal id={MODAL_ID} size={4} bodyCss="px-16 py-14" title={tr('Submit forecast|Enviar forecast')} onSave={confirmSubmit}>
    <div class="grid grid-cols-24 gap-10" aria-label="Submit forecast form">
      <div class="col-span-24 font-bold">{submitTarget?.row.label}</div>
      <Input bind:saveOn={submitForm} save="CommitAmount" type="number" label="Commit|Compromiso" css="col-span-12" />
      <Input bind:saveOn={submitForm} save="BestCaseAmount" type="number" label="Best case|Mejor caso" css="col-span-12" />
      <Input bind:saveOn={submitForm} save="Note" label="Note|Nota" css="col-span-24" />
    </div>
  </Modal>
</Page>
