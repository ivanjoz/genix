// The sales tool's tables and charts (agent/salestools/answer.go), as the chat shows them. Values
// arrive in display units (soles, not cents) and dates as "YYYY-MM-DD"; this file only formats and
// lays them out. Pure: AgentAnswer.svelte renders what it returns.

import { formatN } from '#libs/helpers.ts'

export type AgentCellFormat = 'text' | 'number' | 'money' | 'date'

export interface AgentTableColumn {
  Label: string
  Format: AgentCellFormat
  Values: Array<string | number | null>
  InlineBar?: boolean
}

export interface AgentTable {
  Columns: AgentTableColumn[]
}

export interface AgentChartColumn {
  Label: string
  Axis: 'x' | 'y' | 'y2'
  Values: Array<string | number | null>
  Type?: 'bar' | 'line'
  Format?: 'number' | 'money' | 'date'
}

export interface AgentChart {
  Columns: AgentChartColumn[]
}

// The app's categorical palette, in order: one color per series.
export const AGENT_CHART_COLORS = ['#3b82f6', '#f97316', '#14b8a6', '#eab308', '#ec4899', '#22c55e', '#a855f7', '#ef4444']

// A line reads better than bars on a long date axis.
const LINE_ABOVE_DATE_POINTS = 31

const MONTH_ABBREVIATIONS = ['ENE', 'FEB', 'MAR', 'ABR', 'MAY', 'JUN', 'JUL', 'AGO', 'SEP', 'OCT', 'NOV', 'DIC']

// formatAgentDate turns "2026-09-28" into "SEP-28": month and day only, the year is left out. No
// Date: the day is a calendar day, and parsing it would shift it by the browser's offset.
export const formatAgentDate = (isoDate: string): string => {
  const [year, month, day] = isoDate.split('-')
  const monthAbbreviation = MONTH_ABBREVIATIONS[Number(month) - 1]
  return year && monthAbbreviation && day ? `${monthAbbreviation}-${day}` : isoDate
}

// formatAgentCell is one cell as text: '—' for no value, money with 2 decimals, ratios under 1
// with 4 so they don't read as 0.
export const formatAgentCell = (value: string | number | null | undefined, format: AgentCellFormat | undefined): string => {
  if (value === null || value === undefined) return '—'
  if (typeof value === 'string') return format === 'date' ? formatAgentDate(value) : value
  if (format === 'money') return formatN(value, 2)
  if (Number.isInteger(value)) return formatN(value)
  return formatN(value, Math.abs(value) < 1 ? 4 : 2)
}

// formatAgentChartValue is a chart's axis tick or tooltip value. Unlike a table cell, money shows
// cents only when the value has them: axis ticks are round numbers, and "4,000.00" on every tick
// widens the axis without saying more.
export const formatAgentChartValue = (value: number, format: 'number' | 'money'): string => {
  const hasCents = Math.abs(value - Math.round(value)) >= 0.005
  if (format === 'money') return formatN(value, hasCents ? 2 : 0)
  return formatN(value, -1)
}

// inlineBarPercents is each row's bar length: its size over the column's largest (negatives by size).
export const inlineBarPercents = (column: AgentTableColumn): number[] => {
  const sizes = column.Values.map((value) => (typeof value === 'number' ? Math.abs(value) : 0))
  const largest = Math.max(0, ...sizes)
  return sizes.map((size) => (largest > 0 ? (size / largest) * 100 : 0))
}

export interface AgentChartSeries {
  name: string
  color: string
  values: Array<number | null>
}

// AgentChartPlan is one Chart component: the y series share a scale, y2 gets a second chart below.
export interface AgentChartPlan {
  type: 'bar' | 'line'
  labels: string[]
  series: AgentChartSeries[]
  format: 'number' | 'money'
}

// planAgentChart splits a chart into its scales. A series without a type is a bar, or a line on a
// date axis of more than 31 points; the y2 scale is a line. Colors run in order across both scales.
export const planAgentChart = (chart: AgentChart): AgentChartPlan[] => {
  const xColumn = chart.Columns.find((column) => column.Axis === 'x')
  const labels = (xColumn?.Values || []).map((value) => xColumn?.Format === 'date' && typeof value === 'string'
    ? formatAgentDate(value) : String(value ?? ''))
  const isLongDateAxis = xColumn?.Format === 'date' && labels.length > LINE_ABOVE_DATE_POINTS
  const plans: AgentChartPlan[] = []
  let colorIndex = 0
  for (const axis of ['y', 'y2'] as const) {
    const columns = chart.Columns.filter((column) => column.Axis === axis)
    if (columns.length === 0) continue
    const defaultType = axis === 'y2' || isLongDateAxis ? 'line' : 'bar'
    plans.push({
      type: columns[0].Type || defaultType,
      labels,
      format: columns[0].Format === 'money' ? 'money' : 'number',
      series: columns.map((column) => ({
        name: column.Label,
        color: AGENT_CHART_COLORS[colorIndex++ % AGENT_CHART_COLORS.length],
        values: column.Values.map((value) => (typeof value === 'number' ? value : null)),
      })),
    })
  }
  return plans
}
