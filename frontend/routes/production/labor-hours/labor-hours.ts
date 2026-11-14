import { payrollActualCost } from '#core/payroll-cost.ts'
// Business rules of the "Horas vs planilla" report (docs/MANUFACTURING_PLAN.md §9.3, §9.5, M18, M24,
// M25, M27). No Svelte, no fetch: unit-testable in isolation. Money is PEN cents, periods YYMM.

import type { IContractAgreement } from '#services/hr/contracts-agreements.svelte.ts'
import type { IEmployeeRecord } from '#services/hr/employee-records.ts'
import { isLiveRecord } from '#services/hr/employee-records.ts'
import type { IProductionWorker } from '#services/production/production-settings.svelte.ts'
import { Engagement, LegalConcept } from '#services/hr/hr-catalogs.ts'
import type { IDepartment } from '#services/hr/organization.svelte.ts'
import type { IPayslip } from '#services/hr/payroll.svelte.ts'
import { contractValidOn, employeeFullName } from '#routes/hr/employees/employees.ts'
import { ConceptKind, PayrollStatus, periodFirstDay, periodLastDay } from '#routes/hr/payroll/payroll.ts'

/** Mirrors backend/production/types/production_labor_entry.go. */
export interface IProductionLaborEntry {
  ID: number
  OrderID: number
  EmployeeID: number
  LaborTypeID: number
  Date: number
  Period: number
  Minutes: number
  /** The worker's real rate when logged, PEN cents. */
  CostPerHour: number
  /** The valued cost after the monthly cap, PEN cents. */
  Amount: number
  ss: number
}

/** Department.CostFunction of the production workers: they are listed even with no hours. */
export const COST_FUNCTION_PRODUCTION = 3

export interface ILaborTypeMinutes {
  LaborTypeID: number
  Minutes: number
}

/** One Planilla worker in the month. The payroll figures are null while the month's payroll is pending. */
export interface IPayrollWorkerRow {
  /** 0 on the totals row. */
  EmployeeID: number
  LoggedMinutes: number
  MinutesByLaborType: ILaborTypeMinutes[]
  /** Σ Amount of their entries. */
  BookedAmount: number
  /** WorkedDays × DailyHours + overtime hours. */
  PaidHours: number | null
  /** logged ÷ paid, 0..1+; null when nothing was paid. */
  Utilization: number | null
  /** paid − logged, hours. */
  IdleHours: number | null
  /** EmployerCost + provisions, without the benefit paid from them. */
  ActualCost: number | null
  /** max(0, booked − actual). */
  Adjustment: number | null
  /** The month's payroll is closed but this worker has no payslip in it. */
  IsMissingPayslip: boolean
}

/** One worker off payroll (recibo por honorarios, temporal): no cap, no comparison, no adjustment. */
export interface IOffPayrollWorkerRow {
  /** 0 on the totals row. */
  EmployeeID: number
  LoggedMinutes: number
  MinutesByLaborType: ILaborTypeMinutes[]
  /** Their CostPerHour, averaged by minutes when it changed within the month. 0 on the totals row. */
  CostPerHour: number
  BookedAmount: number
}

export interface ILaborHoursReport {
  /** No closed payslip in the month yet: only the hours and the booked labor are shown. */
  isPayrollPending: boolean
  payrollRows: IPayrollWorkerRow[]
  payrollTotals: IPayrollWorkerRow
  offPayrollRows: IOffPayrollWorkerRow[]
  offPayrollTotals: IOffPayrollWorkerRow
  /** Σ adjustment of the month: what is posted as Dr 7111 · Cr 2111 (M27). */
  totalAdjustment: number
}

export interface ILaborHoursInputs {
  period: number
  entries: IProductionLaborEntry[]
  employees: IProductionWorker[]
  departments: IDepartment[]
  agreements: IContractAgreement[]
  payslips: IPayslip[]
  /** The employee records of the period's year (overtime is read from them). */
  records: IEmployeeRecord[]
}

/** Only a closed or paid payslip counts: an open run is a draft that is recalculated until it closes. */
export const isFinalPayslip = (payslip: IPayslip): boolean => payslip.ss >= PayrollStatus.Closed

/** The payroll cost the worker's month really had (§9.3). A benefit paid in the month (gratificación,
 * CTS, vacation) is charged against the provisions already accrued, so its provisioned part is left out
 * and only the month's own provision lines count: nothing is counted twice. */
export const payslipActualCost = (payslip: IPayslip): number => payrollActualCost(payslip.Lines)

/** Hours of overtime (concepts 25% and 35%) registered for the worker in the period. */
export const overtimeHours = (records: IEmployeeRecord[], employeeID: number, period: number): number =>
  records
    .filter((record) => record.EmployeeID === employeeID && record.Period === period && isLiveRecord(record)
      && (record.ConceptID === LegalConcept.Overtime25 || record.ConceptID === LegalConcept.Overtime35))
    .reduce((hours, record) => hours + record.Quantity, 0)

/** Hours paid: the payslip's worked days at the daily hours of the agreement in force on the period's
 * last day, plus the overtime paid. */
export const paidHours = (
  payslip: IPayslip, agreements: IContractAgreement[], records: IEmployeeRecord[], period: number,
): number => {
  const agreement = contractValidOn(agreements, payslip.EmployeeID, periodLastDay(period))
  return payslip.WorkedDays * (agreement?.DailyHours || 0) + overtimeHours(records, payslip.EmployeeID, period)
}

const isOffPayroll = (employee: IProductionWorker | undefined): boolean =>
  !!employee && employee.EngagementType !== Engagement.Payroll

/** A production worker who was employed some day of the period: listed even with no hours. */
const isProductionWorkerInPeriod = (employee: IProductionWorker, productionDepartmentIDs: Set<number>, period: number): boolean =>
  employee.ss > 0 && !isOffPayroll(employee) && productionDepartmentIDs.has(employee.DepartmentID)
  && employee.HireDate <= periodLastDay(period)
  && (!employee.TerminationDate || employee.TerminationDate >= periodFirstDay(period))

interface IWorkerEntries {
  minutes: number
  amount: number
  /** Σ Minutes × CostPerHour, for the off-payroll rate. */
  ratedMinutes: number
  minutesByLaborType: Map<number, number>
}

const groupEntriesByEmployee = (entries: IProductionLaborEntry[], period: number): Map<number, IWorkerEntries> => {
  const workerEntries = new Map<number, IWorkerEntries>()
  for (const entry of entries) {
    // ss 0 is a deleted entry; the period filter guards a stale cache of another month.
    if (!entry.ss || entry.Period !== period) continue
    let worker = workerEntries.get(entry.EmployeeID)
    if (!worker) {
      worker = { minutes: 0, amount: 0, ratedMinutes: 0, minutesByLaborType: new Map() }
      workerEntries.set(entry.EmployeeID, worker)
    }
    worker.minutes += entry.Minutes
    worker.amount += entry.Amount
    worker.ratedMinutes += entry.Minutes * entry.CostPerHour
    worker.minutesByLaborType.set(entry.LaborTypeID, (worker.minutesByLaborType.get(entry.LaborTypeID) || 0) + entry.Minutes)
  }
  return workerEntries
}

const toLaborTypeMinutes = (minutesByLaborType: Map<number, number>): ILaborTypeMinutes[] =>
  [...minutesByLaborType].map(([LaborTypeID, Minutes]) => ({ LaborTypeID, Minutes }))
    .sort((left, right) => right.Minutes - left.Minutes)

/** Fills the comparison of a row (or the totals) from what was logged, paid and booked. */
const withPayrollComparison = (row: IPayrollWorkerRow, paid: number, actualCost: number, adjustment: number): IPayrollWorkerRow => ({
  ...row,
  PaidHours: paid,
  Utilization: paid > 0 ? row.LoggedMinutes / 60 / paid : null,
  IdleHours: paid - row.LoggedMinutes / 60,
  ActualCost: actualCost,
  Adjustment: adjustment,
})

/** Builds the month's report: one row per Planilla worker who logged hours or works in production,
 * and a separate block for the workers off payroll who logged hours. */
export const buildLaborHoursReport = (inputs: ILaborHoursInputs): ILaborHoursReport => {
  const { period, agreements, records } = inputs
  const employeesByID = new Map(inputs.employees.map((employee) => [employee.ID, employee]))
  const workerEntries = groupEntriesByEmployee(inputs.entries, period)
  const finalPayslips = inputs.payslips.filter((payslip) => payslip.Period === period && isFinalPayslip(payslip))
  const payslipByEmployee = new Map(finalPayslips.map((payslip) => [payslip.EmployeeID, payslip]))
  const isPayrollPending = finalPayslips.length === 0

  const productionDepartmentIDs = new Set(inputs.departments
    .filter((department) => department.CostFunction === COST_FUNCTION_PRODUCTION).map((department) => department.ID))
  const payrollEmployeeIDs = new Set<number>()
  const offPayrollEmployeeIDs = new Set<number>()
  for (const employeeID of workerEntries.keys()) {
    // A worker missing from the list (a deleted employee) is taken as Planilla, the default engagement.
    if (isOffPayroll(employeesByID.get(employeeID))) offPayrollEmployeeIDs.add(employeeID)
    else payrollEmployeeIDs.add(employeeID)
  }
  for (const employee of inputs.employees) {
    if (isProductionWorkerInPeriod(employee, productionDepartmentIDs, period)) payrollEmployeeIDs.add(employee.ID)
  }

  const payrollTotals: IPayrollWorkerRow = {
    EmployeeID: 0, LoggedMinutes: 0, MinutesByLaborType: [], BookedAmount: 0,
    PaidHours: null, Utilization: null, IdleHours: null, ActualCost: null, Adjustment: null, IsMissingPayslip: false,
  }
  const totalsByLaborType = new Map<number, number>()
  let totalPaidHours = 0
  let totalActualCost = 0
  let totalAdjustment = 0

  const payrollRows = [...payrollEmployeeIDs].map((employeeID): IPayrollWorkerRow => {
    const worker = workerEntries.get(employeeID)
    const row: IPayrollWorkerRow = {
      ...payrollTotals,
      EmployeeID: employeeID,
      LoggedMinutes: worker?.minutes || 0,
      MinutesByLaborType: worker ? toLaborTypeMinutes(worker.minutesByLaborType) : [],
      BookedAmount: worker?.amount || 0,
    }
    payrollTotals.LoggedMinutes += row.LoggedMinutes
    payrollTotals.BookedAmount += row.BookedAmount
    for (const [laborTypeID, minutes] of worker?.minutesByLaborType || []) {
      totalsByLaborType.set(laborTypeID, (totalsByLaborType.get(laborTypeID) || 0) + minutes)
    }
    if (isPayrollPending) return row

    // A worker with no payslip in a closed month had no payroll cost: all their booked labor is excess.
    const payslip = payslipByEmployee.get(employeeID)
    const paid = payslip ? paidHours(payslip, agreements, records, period) : 0
    const actualCost = payslip ? payslipActualCost(payslip) : 0
    const adjustment = Math.max(0, row.BookedAmount - actualCost)
    totalPaidHours += paid
    totalActualCost += actualCost
    totalAdjustment += adjustment
    return { ...withPayrollComparison(row, paid, actualCost, adjustment), IsMissingPayslip: !payslip }
  })
  payrollTotals.MinutesByLaborType = toLaborTypeMinutes(totalsByLaborType)

  const offPayrollTotals: IOffPayrollWorkerRow = { EmployeeID: 0, LoggedMinutes: 0, MinutesByLaborType: [], CostPerHour: 0, BookedAmount: 0 }
  const offPayrollTotalsByLaborType = new Map<number, number>()
  const offPayrollRows = [...offPayrollEmployeeIDs].map((employeeID): IOffPayrollWorkerRow => {
    const worker = workerEntries.get(employeeID)!
    offPayrollTotals.LoggedMinutes += worker.minutes
    offPayrollTotals.BookedAmount += worker.amount
    for (const [laborTypeID, minutes] of worker.minutesByLaborType) {
      offPayrollTotalsByLaborType.set(laborTypeID, (offPayrollTotalsByLaborType.get(laborTypeID) || 0) + minutes)
    }
    return {
      EmployeeID: employeeID,
      LoggedMinutes: worker.minutes,
      MinutesByLaborType: toLaborTypeMinutes(worker.minutesByLaborType),
      CostPerHour: worker.minutes > 0 ? Math.round(worker.ratedMinutes / worker.minutes) : 0,
      BookedAmount: worker.amount,
    }
  })
  offPayrollTotals.MinutesByLaborType = toLaborTypeMinutes(offPayrollTotalsByLaborType)

  // HR lists people by surname; a worker missing from the list goes last.
  const sortKey = (employeeID: number) => {
    const employee = employeesByID.get(employeeID)
    return employee ? employeeFullName(employee) : `~${employeeID}`
  }
  const byEmployeeName = (left: { EmployeeID: number }, right: { EmployeeID: number }) =>
    sortKey(left.EmployeeID).localeCompare(sortKey(right.EmployeeID))
  payrollRows.sort(byEmployeeName)
  offPayrollRows.sort(byEmployeeName)

  return {
    isPayrollPending,
    payrollRows,
    payrollTotals: isPayrollPending ? payrollTotals
      : withPayrollComparison(payrollTotals, totalPaidHours, totalActualCost, totalAdjustment),
    offPayrollRows,
    offPayrollTotals,
    totalAdjustment,
  }
}
