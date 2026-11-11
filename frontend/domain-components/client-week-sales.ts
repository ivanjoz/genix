import type { IClientWeekSales } from '#services/sales/client-week-sales.svelte.ts'

/** How many weeks the customers list draws, the current one last (the backend syncs the same). */
export const CLIENT_WEEKS_SHOWN = 26

/** One bar of a client: what the client bought in the week and the part of it still unpaid. */
export interface IClientWeekBar {
  week: number
  purchases: number
  unpaid: number
}

/** The ISO week of a date as YYWW (2646), as the backend's core.FecSemana.Code. */
export function isoWeekCode(date: Date): number {
  const dayUTC = new Date(Date.UTC(date.getFullYear(), date.getMonth(), date.getDate()))
  // The Thursday of the date's Monday-based week decides which year the week belongs to.
  const isoWeekday = dayUTC.getUTCDay() || 7
  dayUTC.setUTCDate(dayUTC.getUTCDate() + 4 - isoWeekday)
  const isoYear = dayUTC.getUTCFullYear()
  const weekNumber = Math.ceil(((dayUTC.getTime() - Date.UTC(isoYear, 0, 1)) / 86_400_000 + 1) / 7)
  return (isoYear % 100) * 100 + weekNumber
}

/** The ISO weeks (YYWW) of the last `count` weeks up to today's, the oldest first. */
export function lastIsoWeeks(today: Date, count: number): number[] {
  const weeks: number[] = []
  for (let weeksBack = count - 1; weeksBack >= 0; weeksBack--) {
    const dayOfWeek = new Date(today.getFullYear(), today.getMonth(), today.getDate() - 7 * weeksBack)
    weeks.push(isoWeekCode(dayOfWeek))
  }
  return weeks
}

/** Each client's bars over `weeks`: a week's purchases summed over its days, and the unpaid part
 *  (purchases − payments, never below zero). A week with no row is an empty bar. */
export function clientWeekBarsByClient(rows: IClientWeekSales[], weeks: number[]): Map<number, IClientWeekBar[]> {
  const weekPosition = new Map(weeks.map((week, position) => [week, position]))
  const barsByClient = new Map<number, IClientWeekBar[]>()
  for (const row of rows) {
    const position = weekPosition.get(row.Week)
    if (position === undefined) continue
    let clientBars = barsByClient.get(row.ClientID)
    if (!clientBars) {
      clientBars = weeks.map((week) => ({ week, purchases: 0, unpaid: 0 }))
      barsByClient.set(row.ClientID, clientBars)
    }
    const purchases = (row.Purchases || []).reduce((sum, dayAmount) => sum + dayAmount, 0)
    const payments = (row.Payments || []).reduce((sum, dayAmount) => sum + dayAmount, 0)
    clientBars[position] = { week: row.Week, purchases, unpaid: Math.max(0, purchases - payments) }
  }
  return barsByClient
}
