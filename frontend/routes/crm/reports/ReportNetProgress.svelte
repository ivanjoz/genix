<script lang="ts">
  import { Chart, type IChartSeries } from '#components/charts/index.ts'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import { formatN, formatTime } from '#libs/helpers.ts'
  import { activeStagesOf } from '../pipeline/pipeline.ts'
  import type { CrmReportsState } from './reports.svelte.ts'
  import {
    aggregateNetProgress, LOST_COLUMN, netProgressByOwner, netProgressColumns, periodFirstDay, periodsBetween, WON_COLUMN,
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
  // The same colors as Tailwind classes (index-aligned), for the table bars of each stage.
  const SERIES_BAR_CSS = ['bg-slate-400', 'bg-sky-400', 'bg-amber-500', 'bg-purple-500', 'bg-pink-500', 'bg-teal-500', 'bg-orange-500', 'bg-indigo-500']
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
    // "won@N" sits under the "Won" group header, so its subheader is just the stage name.
    const position = Number(column.startsWith('won@') ? column.slice(4) : column.slice(1))
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

  const valueColumn = (column: string, header: string): ITableColumn<ProgressRow> => ({
    header, css: 'px-6 text-right ff-mono',
    getValue: (row) => formatValue(valueOf(row.totals.get(column))),
    setCellCss: (row) => valueOf(row.totals.get(column)) < 0 ? 'text-red-500' : undefined,
  })

  /** A value cell underlined by a bar proportional to the largest value of the column among the rows
   *  (the Total row is not drawn: it would always dwarf the sellers). */
  const barColumn = (column: string, header: string, rows: ProgressRow[], barCss: string): ITableColumn<ProgressRow> => {
    const maxValue = Math.max(0, ...rows.filter((row) => row.key !== 0).map((row) => valueOf(row.totals.get(column))))
    return {
      ...valueColumn(column, header),
      renderAbsolute: (row) => {
        const value = valueOf(row.totals.get(column))
        if (maxValue <= 0 || value <= 0 || row.key === 0) return undefined
        // Inset by the cell's 6px side padding so a full bar spans exactly the content width.
        return { css: `bottom-3 left-2 h-3 ${barCss}`, width: `calc((100% - 8px) * ${value / maxValue})` }
      },
    }
  }

  // Won is a grouped header: one subcolumn per stage the deals were won from, then the Won total.
  // Won total and Lost carry a green/red bar, each "won from" stage the bar color of its stage
  // in the chart, so sellers compare at a glance.
  const tableColumns = (firstHeader: string, rows: ProgressRow[]): ITableColumn<ProgressRow>[] => {
    const wonFromColumns = columns.filter((column) => column.startsWith('won@'))
    const stageBarCss = (wonFrom: string) => SERIES_BAR_CSS[(Number(wonFrom.slice(4)) - 1) % SERIES_BAR_CSS.length]
    const tableValueColumns = columns.filter((column) => !column.startsWith('won@')).map((column) => {
      if (column === LOST_COLUMN) return barColumn(LOST_COLUMN, columnLabel(LOST_COLUMN), rows, 'bg-red-500')
      if (column !== WON_COLUMN) return valueColumn(column, columnLabel(column))
      return { header: 'Won|Ganadas', subcols: [
        ...wonFromColumns.map((wonFrom) => barColumn(wonFrom, columnLabel(wonFrom), rows, stageBarCss(wonFrom))),
        barColumn(WON_COLUMN, 'Total|Total', rows, 'bg-green-500'),
      ] }
    })
    return [{ header: firstHeader, css: 'px-6', highlight: true, getValue: (row) => row.label }, ...tableValueColumns]
  }

  // Monthly chart: the net count (or amount) of every stage column and Won, all sellers shown.
  const monthLabels = $derived(periodsBetween(report.periodFrom, report.periodTo))
  // A range spanning several years labels each month with its year.
  const monthLabelLayout = $derived(Math.floor(report.periodFrom / 100) === Math.floor(report.periodTo / 100) ? 'M' : 'M y')
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

  {#if !sellerRows.length}
    <div class="text-fg-muted p-16">{tr('No movements in the selected months.|No hay movimientos en los meses elegidos.')}</div>
  {:else}
    {#key columns.join()}
      <VTable columns={tableColumns(tr('Seller|Vendedor'), sellerRows)} data={sellerRows} css="w-full"
        selected={selectedOwnerID} isSelected={(row, id) => row.key === id && row.key !== 0}
        onRowClick={(row) => { selectedOwnerID = row.key === selectedOwnerID ? 0 : row.key }} />
    {/key}

    {#if selectedOwnerID}
      <div class="h3 ff-bold mt-8">{tr('Day by day|Día a día')} · {report.ownerName(selectedOwnerID)}</div>
      {#key columns.join()}
        <VTable columns={tableColumns(tr('Day|Día'), dayRows)} data={dayRows} css="w-full" maxHeight="400px" />
      {/key}
    {/if}

    {#if monthLabels.length > 1}
      <div class="bg-white/80 rounded-lg p-12">
        <Chart type="bar" series={chartSeries} labels={monthLabels} height={280}
          formatLabel={(period) => formatTime(periodFirstDay(Number(period)), monthLabelLayout) as string}
          formatValue={(value) => formatValue(value)} />
      </div>
    {/if}
  {/if}
</div>
