<script lang="ts">
  import { hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
  import Button from '#components/buttons/Button.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import Page from '#domain/Page.svelte'
  import { tr } from '#core/store.svelte.ts'
  import { formatN, formatTime, todayUnixDay } from '#libs/helpers.ts'
  import { GOAL_METRIC, GOAL_PERIOD } from '#services/crm/crm-goal-metric.ts'
  import { CrmGoalsService, postCrmGoals } from '#services/crm/crm-goal.svelte.ts'
  import { CrmPipelinesService } from '#services/crm/crm-pipeline.svelte.ts'
  import { CrmTeamsService } from '#services/crm/crm-team.svelte.ts'
  import { UsuariosService } from '../../security/users-profiles/users-profiles.svelte'
  import { currentShortYear } from '../reports/reports.svelte.ts'
  import {
    cellKey, cellValue, changedGoalCells, copyPreviousYear, fillRow, goalGridRows, isDailyPeriodType, isMoneyMetric, periodsOfScope,
    scopeCount, scopeIndexOfDay, teamTotal,
    type GoalGridRow,
  } from './goals.ts'

  // Goals per seller x pipeline x metric, one grid per pipeline / metric / year (plan §3.2).
  const pipelines = new CrmPipelinesService(true)
  const teams = new CrmTeamsService(true)
  const goals = new CrmGoalsService(true)
  const usersService = new UsuariosService()

  const MONTH_NAMES = ['Jan|Ene', 'Feb|Feb', 'Mar|Mar', 'Apr|Abr', 'May|May', 'Jun|Jun', 'Jul|Jul', 'Aug|Ago', 'Sep|Set', 'Oct|Oct', 'Nov|Nov', 'Dec|Dic']
  const metricOptions = [
    { id: GOAL_METRIC.WON_AMOUNT, name: tr('Won amount|Monto ganado') },
    { id: GOAL_METRIC.WON_COUNT, name: tr('Won count|Ganadas') },
    { id: GOAL_METRIC.NEW_OPPORTUNITIES, name: tr('New opportunities|Oportunidades nuevas') },
    { id: GOAL_METRIC.ACTIVITIES, name: tr('Activities|Actividades') },
  ]
  const yearOptions = Array.from({ length: 4 }, (_, index) => currentShortYear() + 1 - index)
    .map((year) => ({ id: year, name: String(2000 + year) }))

  let pipelineID = $state(0)
  let metric = $state<number>(GOAL_METRIC.WON_AMOUNT)
  let year = $state(currentShortYear())
  let draft = $state(new Map<string, number>())

  const selectedPipeline = $derived(pipelines.records.find((pipeline) => pipeline.ID === pipelineID))
  $effect(() => { if (!pipelineID && pipelines.records.length) pipelineID = pipelines.records[0].ID })

  const userName = (userID: number) => {
    const user = usersService.usuariosMap.get(userID)
    return user ? `${user.FirstName} ${user.LastName}`.trim() : `#${userID}`
  }
  const rows = $derived(goalGridRows(teams.records, selectedPipeline?.Members.map((member) => member.UserID) ?? [], userName))
  const valueOf = (userID: number, period: number) => cellValue(draft, goals.records, pipelineID, metric, userID, period)
  // The pipeline decides the unit: 12 months, or the ~52 weeks whose Monday falls in the year.
  const periodType = $derived(selectedPipeline?.GoalPeriodType || GOAL_PERIOD.MONTHLY)
  // A daily pipeline shows one month or one week of days at a time, picked here.
  let scopeIndex = $state(-1)
  const scopeOptions = $derived(Array.from({ length: scopeCount(periodType, year) }, (_, index) => ({
    id: index,
    name: periodType === GOAL_PERIOD.DAILY_BY_MONTH ? tr(MONTH_NAMES[index])
      : `${index + 1} · ${formatTime(periodsOfScope(periodType, year, index)[0].firstDay, 'd-M')}`,
  })))
  const activeScopeIndex = $derived(scopeIndex >= 0 ? scopeIndex : scopeIndexOfDay(periodType, year, todayUnixDay()))
  const periods = $derived(periodsOfScope(periodType, year, activeScopeIndex))
  const periodHeader = (index: number) => {
    if (isDailyPeriodType(periodType)) return formatTime(periods[index].firstDay, 'd-M') as string
    return periodType === GOAL_PERIOD.WEEKLY ? `${index + 1} · ${formatTime(periods[index].firstDay, 'd-M')}` : tr(MONTH_NAMES[index])
  }
  const rowValue = (row: GoalGridRow, period: number) =>
    row.isTeamRow ? teamTotal(row, rows, (userID) => valueOf(userID, period)) : valueOf(row.userID, period)
  const yearTotal = (row: GoalGridRow) => periods.reduce((total, { period }) => total + rowValue(row, period), 0)
  const format = (value: number) => isMoneyMetric(metric) ? formatN(value, 2) : String(value)

  const columns = $derived<ITableColumn<GoalGridRow>[]>([
    { header: 'Seller|Vendedor', css: 'px-6', highlight: true, getValue: (row) => row.label,
      setCellCss: (row) => row.isTeamRow ? 'ff-bold' : undefined },
    ...periods.map(({ period }, index): ITableColumn<GoalGridRow> => ({
      header: periodHeader(index), css: 'px-6 text-right ff-mono', cellInputType: 'number',
      getValue: (row) => row.isTeamRow ? format(rowValue(row, period)) : rowValue(row, period),
      disableCellInteractions: (row) => row.isTeamRow,
      onCellEdit: (row, value) => {
        draft.set(cellKey(row.userID, period), Math.max(0, Number(value) || 0))
        draft = new Map(draft)
      },
    })),
    { header: 'Total', css: 'px-6 text-right ff-mono ff-bold', getValue: (row) => format(yearTotal(row)) },
    { header: '', css: 'px-6', getValue: (row) => '', buttonEditIf: (row) => !row.isTeamRow, buttonEditHandler: (row) => {
        // "Fill row": repeat the first period's value across the year.
        draft = fillRow(draft, row.userID, periods, valueOf(row.userID, periods[0].period))
      } },
  ])

  const changedCells = $derived(changedGoalCells(draft, goals.records, pipelineID, metric))

  const saveChanges = async () => {
    showLoading(tr('Saving goals...|Guardando metas...'))
    try {
      await postCrmGoals(goals, changedCells)
      draft = new Map()
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }
</script>

<Page title="CRM Goals|Metas CRM">
  <div class="flex items-end gap-12 flex-wrap mb-12" aria-label="Goal filters">
    <SearchSelect options={pipelines.records} keyId="ID" keyName="Name" label="Pipeline" css="w-240"
      selected={pipelineID} onChange={(option) => { pipelineID = option.ID; draft = new Map(); scopeIndex = -1 }} />
    <SearchSelect options={metricOptions} keyId="id" keyName="name" label="Metric|Métrica" css="w-200"
      selected={metric} onChange={(option) => { metric = option.id; draft = new Map() }} />
    <SearchSelect options={yearOptions} keyId="id" keyName="name" label="Year|Año" css="w-120"
      selected={year} onChange={(option) => { year = option.id; scopeIndex = -1 }} />
    {#if scopeOptions.length}
      <SearchSelect options={scopeOptions} keyId="id" keyName="name" css="w-160"
        label={periodType === GOAL_PERIOD.DAILY_BY_MONTH ? 'Month|Mes' : 'Week|Semana'}
        selected={activeScopeIndex} onChange={(option) => { scopeIndex = option.id }} />
    {/if}
    <Button color="purple" icon="icon-[fa--copy]" name="Copy previous year|Copiar año anterior"
      label="Fills the grid with last year's targets for every seller of the pipeline."
      onClick={() => { draft = copyPreviousYear(goals.records, pipelineID, metric, rows.filter((row) => !row.isTeamRow).map((row) => row.userID), periods, periodsOfScope(periodType, year - 1, activeScopeIndex)) }} />
    <Button color="blue" icon="icon-[fa--floppy-o]" name="Save|Guardar" disabled={changedCells.length === 0}
      label="Saves the edited cells; a cell set to 0 clears its goal." onClick={saveChanges} />
  </div>
  <div class="text-fg-muted mb-8">
    {tr('Only members of the pipeline are listed. Team rows add up their members. The pipeline decides the goal unit (month, week or day). The pencil on a row repeats its first value across the columns shown.|Solo se listan los miembros del pipeline. El pipeline define la unidad de las metas (mes, semana o día). El lápiz de una fila repite su primer valor en todas las columnas mostradas.')}
  </div>
  {#key `${periodType}|${year}|${activeScopeIndex}`}
    <VTable {columns} data={rows} css="w-full" maxHeight="calc(100vh - 12rem)" />
  {/key}
</Page>
