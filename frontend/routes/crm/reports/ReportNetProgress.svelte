<script lang="ts">
  import { Chart, type IChartSeries } from '#components/charts/index.ts'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import { formatN, formatTime } from '#libs/helpers.ts'
  import { activeStagesOf } from '../pipeline/pipeline.ts'
  import type { CrmReportsState } from './reports.svelte.ts'
  import {
    aggregateNetProgress, LOST_COLUMN, netProgressByOwner, netProgressColumns, periodFirstDay, WON_COLUMN,
    type NetProgressCell,
  } from './reports.ts'

  // R1: net progress by seller (plan §8.1). Each stage column is "opportunities that reached it",
  // net of steps back, losses and hand-overs; a deal that went back and forth counts once.

  interface Props { report: CrmReportsState }
  const { report }: Props = $props()

  interface ProgressRow {
    key: number
    label: string
    totals: Map<string, { count: number, amount: number }>
  }

  const SERIES_COLORS = ['#94a3b8', '#38bdf8', '#f59e0b', '#a855f7', '#ec4899', '#14b8a6', '#f97316', '#6366f1']
  const WON_COLOR = '#22c55e'

  let selectedOwnerID = $state(0)

  const stages = $derived(activeStagesOf(report.selectedPipeline))
  const includesOwner = (ownerID: number) => !report.ownerIDs.length || report.ownerIDs.includes(ownerID)
  const monthCells = $derived(aggregateNetProgress(report.rowsInRange, report.selectedPipelineID, 'period')
    .filter((cell) => includesOwner(cell.ownerID)))
  const columns = $derived(netProgressColumns(monthCells, stages.length))

  const columnLabel = (column: string) => {
    if (column === WON_COLUMN) return 'Won|Ganadas'
    if (column === LOST_COLUMN) return 'Lost|Perdidas'
    if (column.startsWith('won@')) {
      const stageName = stages[Number(column.slice(4)) - 1]?.Name ?? column.slice(4)
      return `Won from ${stageName}|Ganada en ${stageName}`
    }
    const position = Number(column.slice(1))
    return stages[position - 1]?.Name ?? `Stage ${position}|Etapa ${position}`
  }
  const valueOf = (total: { count: number, amount: number } | undefined) =>
    report.showAmounts ? (total?.amount ?? 0) / 100 : (total?.count ?? 0)
  const formatValue = (value: number) => report.showAmounts ? formatN(value, 2) : String(value)

  /** Seller rows sorted by name, plus a total row (key 0) at the end. */
  const sellerRows = $derived.by((): ProgressRow[] => {
    const byOwner = netProgressByOwner(monthCells)
    const rows = [...byOwner].map(([ownerID, totals]) => ({ key: ownerID, label: report.ownerName(ownerID), totals }))
      .sort((left, right) => left.label.localeCompare(right.label))
    const grandTotal = netProgressByOwner(monthCells.map((cell) => ({ ...cell, ownerID: 0 }))).get(0) ?? new Map()
    return rows.length ? [...rows, { key: 0, label: tr('Total|Total'), totals: grandTotal }] : []
  })

  /** The selected seller, day by day: only the days with a net movement. */
  const dayRows = $derived.by((): ProgressRow[] => {
    if (!selectedOwnerID) return []
    const dayCells: NetProgressCell[] = aggregateNetProgress(report.rowsInRange, report.selectedPipelineID, 'day')
      .filter((cell) => cell.ownerID === selectedOwnerID)
    const byDay = netProgressByOwner(dayCells.map((cell) => ({ ...cell, ownerID: cell.bucket })))
    return [...byDay].sort((left, right) => right[0] - left[0])
      .map(([unixDay, totals]) => ({ key: unixDay, label: formatTime(unixDay, 'd-M-Y') as string, totals }))
  })

  const tableColumns = (firstHeader: string): ITableColumn<ProgressRow>[] => [
    { header: firstHeader, css: 'px-6', highlight: true, getValue: (row) => row.label },
    ...columns.map((column): ITableColumn<ProgressRow> => ({
      header: columnLabel(column), css: 'px-6 text-right ff-mono',
      getValue: (row) => formatValue(valueOf(row.totals.get(column))),
      setCellCss: (row) => valueOf(row.totals.get(column)) < 0 ? 'text-red-500' : undefined,
    })),
  ]

  // Monthly chart: the net count (or amount) of every stage column and Won, all sellers shown.
  const monthLabels = $derived(Array.from({ length: report.monthTo - report.monthFrom + 1 }, (_, index) => report.periodFrom + index))
  const chartSeries = $derived.by((): IChartSeries[] => {
    const chartColumns = columns.filter((column) => column.startsWith('p') || column === WON_COLUMN)
    return chartColumns.map((column, index) => ({
      name: tr(columnLabel(column)),
      color: column === WON_COLUMN ? WON_COLOR : SERIES_COLORS[index % SERIES_COLORS.length],
      values: monthLabels.map((period) => monthCells
        .filter((cell) => cell.bucket === period && cell.column === column)
        .reduce((total, cell) => total + (report.showAmounts ? cell.amount / 100 : cell.count), 0)),
    }))
  })
</script>

<div class="flex flex-col gap-16" aria-label="Net progress by seller">
  <div class="text-fg-muted">
    {tr('Opportunities that reached each stage, net of steps back, losses and hand-overs. A deal that went back and forth counts once; negative numbers mean the seller lost ground.|Oportunidades que llegaron a cada etapa, netas de retrocesos, pérdidas y reasignaciones. Una oportunidad que avanzó y retrocedió cuenta una sola vez; los negativos indican que el vendedor perdió terreno.')}
  </div>

  {#if !sellerRows.length}
    <div class="text-fg-muted p-16">{tr('No movements in the selected months.|No hay movimientos en los meses elegidos.')}</div>
  {:else}
    {#key columns.join()}
      <VTable columns={tableColumns(tr('Seller|Vendedor'))} data={sellerRows} css="w-full"
        selected={selectedOwnerID} isSelected={(row, id) => row.key === id && row.key !== 0}
        onRowClick={(row) => { selectedOwnerID = row.key === selectedOwnerID ? 0 : row.key }} />
    {/key}

    {#if selectedOwnerID}
      <div class="h3 ff-bold mt-8">{tr('Day by day|Día a día')} · {report.ownerName(selectedOwnerID)}</div>
      {#key columns.join()}
        <VTable columns={tableColumns(tr('Day|Día'))} data={dayRows} css="w-full" maxHeight="400px" />
      {/key}
    {/if}

    {#if monthLabels.length > 1}
      <Chart type="bar" series={chartSeries} labels={monthLabels} height={280}
        formatLabel={(period) => formatTime(periodFirstDay(Number(period)), 'M') as string}
        formatValue={(value) => formatValue(value)} />
    {/if}
  {/if}
</div>
