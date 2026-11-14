import { describe, expect, it } from 'vitest'
import type { IContractAgreement } from '#services/hr/contracts-agreements.svelte.ts'
import type { IEmployeeRecord } from '#services/hr/employee-records.ts'
import type { IEmployee } from '#services/hr/employees.svelte.ts'
import type { IPayslip, IPayslipLine } from '#services/hr/payroll.svelte.ts'
import { periodFirstDay } from '#routes/hr/payroll/payroll.ts'
import { buildLaborHoursReport, payslipActualCost, type ILaborHoursInputs, type IProductionLaborEntry } from './labor-hours.ts'

const PERIOD = 2610
const PRODUCTION_DEPARTMENT = 7
const SALES_DEPARTMENT = 8

const makeEmployee = (ID: number, overrides: Partial<IEmployee> = {}): IEmployee =>
  ({ ID, FirstName: `Worker${ID}`, LastName: `L${ID}`, SecondLastName: '', DepartmentID: SALES_DEPARTMENT,
    EngagementType: 1, HireDate: periodFirstDay(2501), TerminationDate: 0, ss: 1, ...overrides }) as IEmployee

const makeEntry = (EmployeeID: number, LaborTypeID: number, Minutes: number, Amount: number, overrides: Partial<IProductionLaborEntry> = {}): IProductionLaborEntry =>
  ({ ID: 0, OrderID: 1, EmployeeID, LaborTypeID, Date: periodFirstDay(PERIOD), Period: PERIOD, Minutes, CostPerHour: 1000, Amount, ss: 1, ...overrides })

const makeAgreement = (EmployeeID: number, DailyHours: number): IContractAgreement =>
  ({ ID: EmployeeID, EmployeeID, ValidFrom: periodFirstDay(2501), DailyHours, ss: 1 }) as IContractAgreement

// As the payroll engine writes it: EmployerCost is the earnings plus the employer contributions, so a
// salary line makes up what the given earning (Kind 1) and employer (Kind 3) lines leave of it.
const makePayslip = (EmployeeID: number, WorkedDays: number, EmployerCost: number, Lines: Partial<IPayslipLine>[] = [], ss = 2): IPayslip => {
  const givenCost = Lines.filter((line) => line.Kind === 1 || line.Kind === 3).reduce((total, line) => total + (line.Amount || 0), 0)
  const earningLine = { Kind: 1, Amount: EmployerCost - givenCost }
  return ({ ID: EmployeeID, Period: PERIOD, EmployeeID, WorkedDays, EmployerCost, Lines: [earningLine, ...Lines] as IPayslipLine[], ss }) as IPayslip
}

const makeOvertime = (EmployeeID: number, ConceptID: number, Quantity: number): IEmployeeRecord =>
  ({ ID: 0, EmployeeID, ConceptID, Period: PERIOD, Quantity, ss: 2 }) as IEmployeeRecord

const makeInputs = (overrides: Partial<ILaborHoursInputs> = {}): ILaborHoursInputs => ({
  period: PERIOD, entries: [], employees: [], agreements: [], payslips: [], records: [],
  departments: [{ ID: PRODUCTION_DEPARTMENT, CostFunction: 3 }, { ID: SALES_DEPARTMENT, CostFunction: 2 }] as ILaborHoursInputs['departments'],
  ...overrides,
})

describe('labor hours vs payroll', () => {
  it('compares a payroll worker with hours against their payslip', () => {
    const report = buildLaborHoursReport(makeInputs({
      employees: [makeEmployee(1)],
      entries: [makeEntry(1, 10, 120 * 60, 120000), makeEntry(1, 11, 30 * 60, 30000), makeEntry(1, 10, 600, 999, { ss: 0 })],
      agreements: [makeAgreement(1, 8)],
      // 4000 of provisions on top of the employer cost.
      payslips: [makePayslip(1, 25, 160000, [{ Kind: 4, Amount: 3000 }, { Kind: 4, Amount: 1000 }, { Kind: 3, Amount: 500 }])],
    }))
    expect(report.isPayrollPending).toBe(false)
    const [row] = report.payrollRows
    expect(row.LoggedMinutes).toBe(150 * 60)
    expect(row.MinutesByLaborType).toEqual([{ LaborTypeID: 10, Minutes: 7200 }, { LaborTypeID: 11, Minutes: 1800 }])
    expect(row.PaidHours).toBe(200)
    expect(row.Utilization).toBe(0.75)
    expect(row.IdleHours).toBe(50)
    expect(row.BookedAmount).toBe(150000)
    expect(row.ActualCost).toBe(164000)
    // Booked below the actual cost: a shortfall needs no adjustment.
    expect(row.Adjustment).toBe(0)
    expect(report.totalAdjustment).toBe(0)
  })

  it('adjusts only the excess of booked over actual, per employee', () => {
    const report = buildLaborHoursReport(makeInputs({
      employees: [makeEmployee(1), makeEmployee(2)],
      entries: [makeEntry(1, 10, 200 * 60, 200000), makeEntry(2, 10, 100 * 60, 50000)],
      agreements: [makeAgreement(1, 8), makeAgreement(2, 8)],
      payslips: [makePayslip(1, 26, 170000, [{ Kind: 4, Amount: 10000 }]), makePayslip(2, 26, 90000)],
    }))
    const rowsByEmployee = new Map(report.payrollRows.map((row) => [row.EmployeeID, row]))
    expect(rowsByEmployee.get(1)?.Adjustment).toBe(20000)
    // Employee 2 is 40000 short: it does not offset employee 1's excess.
    expect(rowsByEmployee.get(2)?.Adjustment).toBe(0)
    expect(report.totalAdjustment).toBe(20000)
    expect(report.payrollTotals.Adjustment).toBe(20000)
    expect(report.payrollTotals.PaidHours).toBe(416)
    expect(report.payrollTotals.BookedAmount).toBe(250000)
  })

  it('leaves out the benefit paid from its provision', () => {
    // A gratificación of 200000 fully provisioned, plus the month's own provision of 15000.
    const payslip = makePayslip(1, 26, 380000, [
      { Kind: 1, Amount: 200000, ProvisionedAmount: 200000 }, { Kind: 4, Amount: 15000 },
    ])
    expect(payslipActualCost(payslip)).toBe(195000)
  })

  it('shows hours and booked labor only while the payroll is pending', () => {
    const report = buildLaborHoursReport(makeInputs({
      employees: [makeEmployee(1)],
      entries: [makeEntry(1, 10, 600, 10000)],
      agreements: [makeAgreement(1, 8)],
      // An open (draft) run does not count.
      payslips: [makePayslip(1, 26, 160000, [], 1)],
    }))
    expect(report.isPayrollPending).toBe(true)
    const [row] = report.payrollRows
    expect(row.LoggedMinutes).toBe(600)
    expect(row.BookedAmount).toBe(10000)
    expect(row.PaidHours).toBeNull()
    expect(row.Adjustment).toBeNull()
    expect(report.payrollTotals.Utilization).toBeNull()
    expect(report.totalAdjustment).toBe(0)
  })

  it('lists a production worker with no hours at 0%', () => {
    const report = buildLaborHoursReport(makeInputs({
      employees: [
        makeEmployee(1, { DepartmentID: PRODUCTION_DEPARTMENT }),
        // Not listed: sales without hours, a production worker terminated before the month, one hired after it.
        makeEmployee(2),
        makeEmployee(3, { DepartmentID: PRODUCTION_DEPARTMENT, TerminationDate: periodFirstDay(PERIOD) - 1, ss: 2 }),
        makeEmployee(4, { DepartmentID: PRODUCTION_DEPARTMENT, HireDate: periodFirstDay(2611) }),
      ],
      agreements: [makeAgreement(1, 8)],
      payslips: [makePayslip(1, 26, 160000)],
    }))
    expect(report.payrollRows.map((row) => row.EmployeeID)).toEqual([1])
    const [row] = report.payrollRows
    expect(row.LoggedMinutes).toBe(0)
    expect(row.Utilization).toBe(0)
    expect(row.IdleHours).toBe(208)
    expect(row.Adjustment).toBe(0)
  })

  it('puts the workers off payroll in their own block, without comparison', () => {
    const report = buildLaborHoursReport(makeInputs({
      employees: [makeEmployee(1, { EngagementType: 2, DepartmentID: PRODUCTION_DEPARTMENT }), makeEmployee(2, { EngagementType: 3 }),
        makeEmployee(3, { EngagementType: 3, DepartmentID: PRODUCTION_DEPARTMENT })],
      entries: [
        makeEntry(1, 10, 60, 1500, { CostPerHour: 1500 }), makeEntry(1, 10, 180, 6000, { CostPerHour: 2000 }),
        makeEntry(2, 11, 120, 2000),
      ],
    }))
    expect(report.payrollRows).toEqual([])
    // Worker 3 is in production but off payroll with no hours: not listed.
    expect(report.offPayrollRows.map((row) => row.EmployeeID)).toEqual([1, 2])
    const [worker1] = report.offPayrollRows
    expect(worker1.LoggedMinutes).toBe(240)
    expect(worker1.CostPerHour).toBe(1875)
    expect(worker1.BookedAmount).toBe(7500)
    expect(report.offPayrollTotals.LoggedMinutes).toBe(360)
    expect(report.offPayrollTotals.BookedAmount).toBe(9500)
    expect(report.totalAdjustment).toBe(0)
  })

  it('adds the overtime of the month to the paid hours', () => {
    const report = buildLaborHoursReport(makeInputs({
      employees: [makeEmployee(1)],
      entries: [makeEntry(1, 10, 60, 1000)],
      agreements: [makeAgreement(1, 8)],
      payslips: [makePayslip(1, 26, 160000)],
      records: [
        makeOvertime(1, 3, 6), makeOvertime(1, 4, 2),
        // Ignored: cancelled, another month, another concept, another employee.
        { ...makeOvertime(1, 3, 50), ss: 0 }, { ...makeOvertime(1, 3, 50), Period: 2609 },
        makeOvertime(1, 23, 50), makeOvertime(2, 3, 50),
      ],
    }))
    expect(report.payrollRows[0].PaidHours).toBe(26 * 8 + 8)
  })

  it('takes all booked labor as excess when a payroll worker has no payslip in a closed month', () => {
    const report = buildLaborHoursReport(makeInputs({
      employees: [makeEmployee(1), makeEmployee(2)],
      entries: [makeEntry(1, 10, 60, 1000), makeEntry(2, 10, 60, 1000)],
      payslips: [makePayslip(1, 26, 160000)],
    }))
    const missing = report.payrollRows.find((row) => row.EmployeeID === 2)!
    expect(missing.IsMissingPayslip).toBe(true)
    expect(missing.PaidHours).toBe(0)
    expect(missing.Utilization).toBeNull()
    expect(missing.Adjustment).toBe(1000)
  })
})
