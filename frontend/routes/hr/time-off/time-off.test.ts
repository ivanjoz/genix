import { describe, expect, it } from 'vitest'
import type { IEmployeeRecord } from '#services/hr/employee-records.ts'
import { balanceYears, serviceDays, vacationBalance } from './time-off.ts'

const day = (year: number, month: number, date: number) => Date.UTC(year, month - 1, date) / 86400000

describe('vacation balance', () => {
  it('counts service on 30-day months like the payroll', () => {
    // Hired 1 Oct 2025, until 15 Oct 2026: 12 whole months and 15 days.
    expect(serviceDays({ HireDate: day(2025, 10, 1), TerminationDate: 0 }, day(2026, 10, 15))).toBe(375)
    // A termination stops the count.
    expect(serviceDays({ HireDate: day(2025, 10, 1), TerminationDate: day(2025, 12, 31) }, day(2026, 10, 15))).toBe(90)
  })

  it('earns 30 days a year and subtracts the active and settled vacations only', () => {
    const employee = { ID: 1, HireDate: day(2025, 10, 1), TerminationDate: 0 }
    const records = [
      { EmployeeID: 1, ConceptID: 29, Quantity: 10, ss: 2 },
      { EmployeeID: 1, ConceptID: 29, Quantity: 4, ss: 3 },  // settled by a closed payslip
      { EmployeeID: 1, ConceptID: 29, Quantity: 5, ss: 0 },  // cancelled
      { EmployeeID: 1, ConceptID: 29, Quantity: 2, ss: 1 },  // pending approval
      { EmployeeID: 1, ConceptID: 30, Quantity: 3, ss: 2 },  // medical leave
      { EmployeeID: 2, ConceptID: 29, Quantity: 7, ss: 2 },  // someone else
    ] as IEmployeeRecord[]
    expect(vacationBalance(employee, records, day(2026, 10, 15))).toEqual({ earned: 31.2, taken: 14, balance: 17.2 })
  })

  it('reads the years back to the earliest hire, ten at most', () => {
    expect(balanceYears([{ HireDate: day(2024, 3, 1), TerminationDate: 0 }], 2026)).toEqual([2024, 2025, 2026])
    expect(balanceYears([{ HireDate: day(2001, 3, 1), TerminationDate: 0 }], 2026)).toHaveLength(10)
    expect(balanceYears([], 2026)).toEqual([2026])
  })
})
