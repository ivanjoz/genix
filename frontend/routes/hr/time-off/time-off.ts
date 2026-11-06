// Pure business logic of the time-off page: the vacation balance, derived (never stored) from the
// service time and the vacations taken, as the settlement calculates it on the server.

import { isLiveRecord, type IEmployeeRecord } from '#services/hr/employee-records.ts'
import { LegalConcept } from '#services/hr/hr-catalogs.ts'
import { nextPeriod, periodFirstDay, periodLastDay, periodOfDay } from '#routes/hr/payroll/payroll.ts'

const MONTH_DAYS = 30
const YEAR_DAYS = 360
const ANNUAL_VACATION_DAYS = 30
/** How far back the page reads time off for the balance. */
const MAX_HISTORY_YEARS = 10

interface IServiceDates {
  HireDate: number
  TerminationDate: number
}

export const yearOfDay = (unixDay: number): number => new Date(unixDay * 86400000).getUTCFullYear()

/** Days of service up to untilDay on the 30-day month of the payroll (EmployedDays in the backend):
 * a whole month counts 30, a partial one its days. */
export const serviceDays = (employee: IServiceDates, untilDay: number): number => {
  const lastDay = employee.TerminationDate > 0 ? Math.min(employee.TerminationDate, untilDay) : untilDay
  let days = 0
  for (let period = periodOfDay(employee.HireDate); period <= periodOfDay(lastDay); period = nextPeriod(period)) {
    const firstOfMonth = periodFirstDay(period), lastOfMonth = periodLastDay(period)
    const from = Math.max(employee.HireDate, firstOfMonth), to = Math.min(lastDay, lastOfMonth)
    if (from > to) continue
    days += from === firstOfMonth && to === lastOfMonth ? MONTH_DAYS : Math.min(to - from + 1, MONTH_DAYS)
  }
  return days
}

/** Vacation earned (30 days a year, accrued by day), taken (active or settled vacations) and the
 * balance. TODO: part-time workers earn 6 days a year; the page does not read contracts. */
export const vacationBalance = (employee: IServiceDates & { ID: number }, records: IEmployeeRecord[], today: number) => {
  const earned = Math.floor(serviceDays(employee, today) * ANNUAL_VACATION_DAYS / YEAR_DAYS * 10) / 10
  const taken = records
    .filter((leave) => leave.EmployeeID === employee.ID && isLiveRecord(leave) && leave.ConceptID === LegalConcept.Vacation)
    .reduce((total, leave) => total + leave.Quantity, 0)
  return { earned, taken, balance: Math.round((earned - taken) * 10) / 10 }
}

/** The years whose time off the balance needs: from the earliest hire of the active staff, at most
 * MAX_HISTORY_YEARS back. */
export const balanceYears = (employees: IServiceDates[], currentYear: number): number[] => {
  const earliestHire = Math.min(...employees.map((employee) => yearOfDay(employee.HireDate)), currentYear)
  const firstYear = Math.max(earliestHire, currentYear - MAX_HISTORY_YEARS + 1)
  return Array.from({ length: currentYear - firstYear + 1 }, (_, index) => firstYear + index)
}
