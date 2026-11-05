import { describe, expect, it } from 'vitest'
import type { IPayrollRun, IPayslip } from '#services/hr/payroll.svelte.ts'
import {
  allowedTransitions, formatCents, formatPeriod, nextPeriod, payrollMonths, payslipLineName, payslipSections,
  periodFirstDay, periodLastDay, periodOfDay, previousPeriod, runContributionTotals, runState, runTotals,
} from './payroll.ts'

const run = (fields: Partial<IPayrollRun>) => ({ ID: 1, Period: 2610, CalculatedAt: 0, ss: 1, ...fields }) as IPayrollRun

describe('periods', () => {
  it('formats and walks YYMM periods', () => {
    expect(formatPeriod(2610)).toBe('Oct 2026')
    expect(nextPeriod(2612)).toBe(2701)
    expect(previousPeriod(2701)).toBe(2612)
  })
  it('matches the backend day helpers', () => {
    expect(periodFirstDay(2610)).toBe(20727)
    expect(periodLastDay(2610)).toBe(20757)
    expect(periodOfDay(20758)).toBe(2611)
  })
  it('lists every month of the window newest first, with its run when it has one', () => {
    const months = payrollMonths([run({ ID: 5, Period: 2611 }), run({ ID: 4, Period: 2610 })], { FirstPeriod: 2609, LastPeriod: 2612 })
    expect(months.map((month) => month.Period)).toEqual([2612, 2611, 2610, 2609])
    expect(months.map((month) => month.Run?.ID)).toEqual([undefined, 5, 4, undefined])
  })
  it('crosses the year inside the window and keeps a run outside it', () => {
    const months = payrollMonths([run({ Period: 2511 })], { FirstPeriod: 2612, LastPeriod: 2702 })
    expect(months.map((month) => month.Period)).toEqual([2702, 2701, 2612, 2511])
  })
  it('shows only the runs when the user has no window, and hides deleted ones', () => {
    const months = payrollMonths([run({ Period: 2610 }), run({ Period: 2609, ss: 0 })])
    expect(months.map((month) => month.Period)).toEqual([2610])
  })
})

describe('run state', () => {
  it('tells draft from calculated by CalculatedAt', () => {
    expect(runState(run({}))).toBe('draft')
    expect(runState(run({ CalculatedAt: 5 }))).toBe('calculated')
    expect(runState(run({ ss: 2 }))).toBe('closed')
  })
  it('only closes a calculated run and only pays a closed one', () => {
    expect(allowedTransitions(run({})).close).toBe(false)
    expect(allowedTransitions(run({ CalculatedAt: 5 })).close).toBe(true)
    expect(allowedTransitions(run({ ss: 2 }))).toMatchObject({ pay: true, reopen: true, calculate: false })
  })
})

describe('payslip', () => {
  const payslip = {
    Lines: [
      { LegalConceptID: 1, ConceptID: 0, Kind: 1, Amount: 200000 },
      { LegalConceptID: 0, ConceptID: 7, Kind: 1, Amount: 15000 },
      { LegalConceptID: 23, ConceptID: 0, Kind: 2, Amount: 26000 },
      { LegalConceptID: 40, ConceptID: 0, Kind: 3, Amount: 18000 },
      { LegalConceptID: 52, ConceptID: 0, Kind: 4, Amount: 19444 },
    ],
    TotalEarnings: 215000, TotalDeductions: 26000, NetPay: 189000, EmployerCost: 233000,
    Diagnostics: [{ Code: 'HR-003', Message: '', Blocking: true }],
  } as unknown as IPayslip

  it('splits the lines into the boleta sections', () => {
    const sections = payslipSections(payslip)
    expect([sections.earnings.length, sections.deductions.length, sections.employer.length, sections.provisions.length])
      .toEqual([2, 1, 1, 1])
  })
  it('names legal and company concepts', () => {
    const concepts = new Map([[7, { Name: 'Movilidad' }]]) as any
    expect(payslipLineName(payslip.Lines[0], concepts)).toContain('Remuneración básica')
    expect(payslipLineName(payslip.Lines[1], concepts)).toBe('Movilidad')
  })
  it('sums what the run owes to third parties by payable', () => {
    const withPayables = { ...payslip, Lines: [
      { Kind: 2, PayableAccountCode: 4032, Amount: 26000 },
      { Kind: 2, PayableAccountCode: 1412, Amount: 30000 },
      { Kind: 3, PayableAccountCode: 4031, Amount: 18000 },
      { Kind: 4, PayableAccountCode: 415, Amount: 19444 },
    ] } as unknown as IPayslip
    const totals = runContributionTotals([withPayables, withPayables])
    expect([...totals.entries()]).toEqual([[4032, 52000], [4031, 36000]])
  })
  it('totals a run and counts blocked payslips', () => {
    expect(runTotals([payslip, payslip])).toMatchObject({ netPay: 378000, blocked: 2 })
    expect(formatCents(123456)).toBe('1,234.56')
  })
})
