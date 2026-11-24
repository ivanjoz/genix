<script lang="ts">
  import Chart, { type IChartSeries } from '#components/charts/Chart.svelte'
  import { tr } from '#core/store.svelte.ts'
  import { formatN } from '#libs/helpers.ts'
  import { formatPeriod, periodOfDay } from '#routes/hr/payroll/payroll.ts'

  // The closing balance of each lane, by day or month: where cash runs short shows as a line under
  // zero. The table below holds the same numbers, so the chart needs no value labels.
  let { days, lanes, isMonthly }: {
    days: number[]
    lanes: { name: string, closing: number[] }[]
    isMonthly: boolean
  } = $props()

  // The validated categorical order (dataviz palette), assigned by lane position, never cycled.
  const LANE_COLORS = ['#2a78d6', '#eb6834', '#1baf7a', '#eda100', '#e87ba4', '#008300', '#6250d6', '#e34948']

  const series = $derived(lanes.slice(0, LANE_COLORS.length).map((lane, laneIndex): IChartSeries => ({
    name: tr(lane.name), color: LANE_COLORS[laneIndex], values: lane.closing.map((cents) => cents / 100),
  })))
  const formatColumn = (unixDay: string | number) => {
    if (isMonthly) return formatPeriod(periodOfDay(Number(unixDay)))
    const date = new Date(Number(unixDay) * 86400000)
    return `${String(date.getUTCDate()).padStart(2, '0')}/${String(date.getUTCMonth() + 1).padStart(2, '0')}`
  }
</script>

<Chart type="line" {series} labels={days} height={260} labelEvery={isMonthly ? 1 : 7} formatLabel={formatColumn}
  formatValue={(value) => formatN(value, 0)} />
