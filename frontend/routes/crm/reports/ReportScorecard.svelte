<script lang="ts">
  import { untrack } from 'svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import { formatN, todayUnixDay } from '#libs/helpers.ts'
  import { GOAL_METRIC, GOAL_PERIOD } from '#services/crm/crm-goal-metric.ts'
  import { goalGridRows, goalTargetsInRange, type GoalGridRow } from '../goals/goals.ts'
  import type { CrmReportsState } from './reports.svelte.ts'
  import {
    addScorecardSums, emptyScorecardSums, periodFirstDay, periodLastDay, scorecardMetrics, scorecardSumsByOwner,
    type ScorecardMetrics, type ScorecardSums,
  } from './reports.ts'

  // R5: one row per seller, grouped by team. Team rows add their members' sums and derive their
  // own rates. Open count is today's; everything else is the selected range.

  interface Props { report: CrmReportsState }
  const { report }: Props = $props()

  // Activities are only downloaded for this view; reload when the pipeline or the years change.
  $effect(() => {
    if (!report.selectedPipelineID) return
    void [report.yearFrom, report.periodTo]
    untrack(() => report.loadActivities())
  })

  const fromDay = $derived(periodFirstDay(report.periodFrom))
  const toDay = $derived(Math.min(periodLastDay(report.periodTo), todayUnixDay()))
  const sumsByOwner = $derived(scorecardSumsByOwner({
    rowsInRange: report.rowsInRange, rowsToToday: report.rowsToToday, pipelineID: report.selectedPipelineID,
    openNow: report.openNow, activityAuthorIDs: report.activitiesInRange.map((activity) => activity.AuthorID),
    goalAmountByOwner: goalTargetsInRange(report.goals.records, report.selectedPipelineID, GOAL_METRIC.WON_AMOUNT,
      report.selectedPipeline?.GoalPeriodType || GOAL_PERIOD.MONTHLY, fromDay, periodLastDay(report.periodTo)),
  }))

  interface ScorecardRow {
    row: GoalGridRow
    sums: ScorecardSums
    metrics: ScorecardMetrics
  }

  const scorecardRows = $derived.by((): ScorecardRow[] => {
    const ownerIDs = [...sumsByOwner.keys()].filter((ownerID) => !report.ownerIDs.length || report.ownerIDs.includes(ownerID))
      // Leaderboard order inside each team: most won first.
      .sort((left, right) => sumsByOwner.get(right)!.wonAmount - sumsByOwner.get(left)!.wonAmount)
    const gridRows = goalGridRows(report.teams.records, ownerIDs, (ownerID) => report.ownerName(ownerID))
    return gridRows.map((row) => {
      const sums = row.isTeamRow
        ? gridRows.filter((member) => !member.isTeamRow && member.teamID === row.teamID)
          .reduce((total, member) => addScorecardSums(total, sumsByOwner.get(member.userID)!), emptyScorecardSums())
        : sumsByOwner.get(row.userID)!
      return { row, sums, metrics: scorecardMetrics(sums) }
    })
  })

  const money = (amount: number) => formatN(amount / 100, 2)
  const numberColumn = (header: string, getValue: (item: ScorecardRow) => string): ITableColumn<ScorecardRow> =>
    ({ header, css: 'px-6 text-right ff-mono', getValue })
  const columns: ITableColumn<ScorecardRow>[] = [
    { header: 'Seller|Vendedor', css: 'px-6', highlight: true, getValue: (item) => item.row.label,
      setCellCss: (item) => item.row.isTeamRow ? 'ff-bold' : undefined },
    numberColumn('Won|Ganadas', (item) => String(item.sums.wonCount)),
    numberColumn('Lost|Perdidas', (item) => String(item.sums.lostCount)),
    numberColumn('Win rate|Tasa de cierre', (item) => `${Math.round(item.metrics.winRate * 100)}%`),
    numberColumn('Won amount|Monto ganado', (item) => money(item.sums.wonAmount)),
    numberColumn('Avg. won|Prom. ganado', (item) => money(item.metrics.averageWonAmount)),
    numberColumn('Cycle days|Días de ciclo', (item) => item.sums.cycleCount ? formatN(item.metrics.averageCycleDays, 1) : '—'),
    numberColumn('Open today|Abiertas hoy', (item) => String(item.sums.openCount)),
    numberColumn('Velocity / day|Velocidad / día', (item) => money(item.metrics.velocity)),
    numberColumn('Activities|Actividades', (item) => String(item.sums.activities)),
    numberColumn('Goal|Meta', (item) => item.sums.goalAmount ? money(item.sums.goalAmount) : '—'),
    { header: '% goal|% meta', css: 'px-6 text-right ff-mono',
      getValue: (item) => item.sums.goalAmount ? `${item.metrics.goalAttainment}%` : '—',
      setCellCss: (item) => item.sums.goalAmount && item.metrics.goalAttainment < 100 ? 'text-red-500' : undefined },
  ]
</script>

<div class="flex flex-col gap-8" aria-label="Seller scorecard">
  {#if !scorecardRows.length}
    <div class="text-fg-muted p-16">{tr('No sellers with movements in the selected months.|No hay vendedores con movimientos en los meses elegidos.')}</div>
  {:else}
    <VTable {columns} data={scorecardRows} css="w-full" />
    <div class="text-fg-muted">
      {tr('Win rate = won ÷ (won + lost). Cycle = days from creation to won, for deals created since the first selected year. Velocity = open today × win rate × average won ÷ cycle days.|Tasa de cierre = ganadas ÷ (ganadas + perdidas). Ciclo = días desde la creación hasta ganar, para oportunidades creadas desde el primer año elegido. Velocidad = abiertas hoy × tasa de cierre × promedio ganado ÷ días de ciclo.')}
    </div>
  {/if}
</div>
