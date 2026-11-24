<script lang="ts">
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import { formatN } from '#libs/helpers.ts'
  import { formatPeriod, periodOfDay } from '#routes/hr/payroll/payroll.ts'
  import { CASH_FLOW_STREAMS, PLANNED_STREAM_ID, type ICashFlowGrid, type ICashFlowRow } from './cash-flow.ts'

  // The lane's grid: one row per stream, one column per day (or month), between the opening cash and
  // the closing balance. A stream's cell opens its records (onCellClick); a negative closing is red.
  let { grid, isMonthly, onCellClick }: {
    grid: ICashFlowGrid
    isMonthly: boolean
    onCellClick: (row: ICashFlowRow, dayIndex: number) => void
  } = $props()

  interface ITableRow {
    label: string
    amounts: number[]
    total: number | null
    stream?: ICashFlowRow
    isBalance: boolean
  }

  const WEEKDAY_NAMES = ['Sun|Dom', 'Mon|Lun', 'Tue|Mar', 'Wed|Mié', 'Thu|Jue', 'Fri|Vie', 'Sat|Sáb']
  const streamNames = new Map(CASH_FLOW_STREAMS.map((stream) => [stream.id, stream.name]))
  const formatAmount = (cents: number) => cents ? formatN(cents / 100, 0) : ''

  const tableRows = $derived.by((): ITableRow[] => [
    { label: tr('Opening cash|Saldo inicial'), amounts: grid.days.map((_, dayIndex) => dayIndex === 0 ? grid.opening : 0),
      total: null, isBalance: true },
    ...grid.rows.map((row) => ({
      label: tr(row.streamID === PLANNED_STREAM_ID ? 'Planned|Planificado' : streamNames.get(row.streamID) || String(row.streamID)),
      amounts: row.cells.map((cell) => cell.amount), total: row.total, stream: row, isBalance: false,
    })),
    { label: tr('Closing balance|Saldo final'), amounts: grid.closing, total: null, isBalance: true },
  ])

  const columnHeader = (unixDay: number) => {
    if (isMonthly) return formatPeriod(periodOfDay(unixDay))
    const date = new Date(unixDay * 86400000)
    const dayMonth = `${String(date.getUTCDate()).padStart(2, '0')}/${String(date.getUTCMonth() + 1).padStart(2, '0')}`
    return `${tr(WEEKDAY_NAMES[date.getUTCDay()])} ${dayMonth}`
  }

  const columns = $derived.by((): ITableColumn<ITableRow>[] => [
    { id: 'label', header: 'Concept|Concepto', headerCss: 'w-220', css: 'px-6 whitespace-nowrap',
      setCellCss: (row) => row.isBalance ? 'ff-bold' : undefined, getValue: (row) => row.label },
    { id: 'total', header: 'Total|Total', headerCss: 'w-110', css: 'px-6 text-right ff-bold',
      getValue: (row) => row.total === null ? '' : formatAmount(row.total) },
    ...grid.days.map((unixDay, dayIndex): ITableColumn<ITableRow> => ({
      id: `day-${dayIndex}`, header: columnHeader(unixDay), headerCss: `${isMonthly ? 'w-120' : 'w-96'} whitespace-nowrap`,
      css: 'px-6 text-right',
      getValue: (row) => formatAmount(row.amounts[dayIndex]),
      setCellCss: (row) => [
        row.isBalance ? 'ff-bold' : 'cursor-pointer',
        row.amounts[dayIndex] < 0 && (row.isBalance || row.amounts[dayIndex]) ? 'text-red-fg' : '',
      ].join(' '),
      showHoverEffect: true,
      onCellClick: (row) => { if (row.stream && row.amounts[dayIndex]) onCellClick(row.stream, dayIndex) },
    })),
  ])
</script>

<div class="overflow-x-auto" aria-label="Cash flow by day">
  <VTable {columns} data={tableRows} disableVirtualizer={true} estimateSize={36} css="min-w-max"
    emptyMessage="No flows in the window.|Sin movimientos en el periodo." />
</div>
