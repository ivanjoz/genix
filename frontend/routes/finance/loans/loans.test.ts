import { describe, expect, it } from 'vitest'
import type { ILoan } from '#services/finance/loans.svelte.ts'
import {
  addMonthsToUnixDay, buildLoanSchedule, installmentState, installmentTotal, LOAN_INSTALLMENT_DUE,
  LOAN_INSTALLMENT_OVERDUE, LOAN_INSTALLMENT_PAID, LOAN_METHOD_BULLET, LOAN_METHOD_FRENCH, LOAN_METHOD_GERMAN,
  nextPendingInstallment, outstandingPrincipal, periodRate, type ILoanScheduleInput,
} from './loans.ts'

const dayOf = (year: number, monthIndex: number, day: number) => Math.round(Date.UTC(year, monthIndex, day) / 86400000)

const baseInput: ILoanScheduleInput = {
  principal: 15_000_000, annualRate: 180000, months: 24, disbursementDate: dayOf(2026, 0, 15),
  firstDueDate: dayOf(2026, 1, 15), method: LOAN_METHOD_FRENCH, graceMonths: 0, fee: 0,
}
const principalsOf = (installments: { Principal: number }[]) => installments.reduce((total, e) => total + e.Principal, 0)

describe('loan schedule', () => {
  it('moves a due date month by month, clamping to the month end', () => {
    expect(addMonthsToUnixDay(dayOf(2026, 0, 31), 1)).toBe(dayOf(2026, 1, 28))
    expect(addMonthsToUnixDay(dayOf(2026, 10, 15), 2)).toBe(dayOf(2027, 0, 15))
  })

  it('compounds the TEA by the actual days over 360', () => {
    expect(periodRate(180000, 360)).toBeCloseTo(0.18, 10)
    expect(periodRate(180000, 30)).toBeCloseTo(Math.pow(1.18, 30 / 360) - 1, 12)
  })

  it('French: principals add up to the loan and the payment stays constant on equal periods', () => {
    const schedule = buildLoanSchedule({ ...baseInput, disbursementDate: dayOf(2026, 0, 1), firstDueDate: dayOf(2026, 0, 31), months: 12 })
    expect(schedule).toHaveLength(12)
    expect(principalsOf(schedule)).toBe(baseInput.principal)
    // Unequal months make the payment drift by a few soles; it never moves more than 1 %.
    const payments = schedule.slice(0, -1).map(installmentTotal)
    expect(Math.max(...payments) - Math.min(...payments)).toBeLessThan(payments[0] * 0.01)
  })

  it('French: every installment but the last pays C = P / Σ_k Π_{j≤k} 1/(1+i_j) over the actual periods', () => {
    const schedule = buildLoanSchedule(baseInput)
    let discountFactor = 1
    let discountFactorsSum = 0
    schedule.forEach((installment, index) => {
      const previousDate = index ? schedule[index - 1].DueDate : baseInput.disbursementDate
      discountFactor /= 1 + periodRate(baseInput.annualRate, installment.DueDate - previousDate)
      discountFactorsSum += discountFactor
    })
    const expectedPayment = baseInput.principal / discountFactorsSum
    for (const installment of schedule.slice(0, -1)) {
      expect(Math.abs(installmentTotal(installment) - expectedPayment)).toBeLessThanOrEqual(1)
    }
    // The last one absorbs the rounding of the 23 before it: a few cents at most.
    expect(Math.abs(installmentTotal(schedule[schedule.length - 1]) - expectedPayment)).toBeLessThan(100)
  })

  it('German: constant principal, interest falling with the balance', () => {
    const schedule = buildLoanSchedule({ ...baseInput, method: LOAN_METHOD_GERMAN, months: 4 })
    expect(schedule.slice(0, 3).map((e) => e.Principal)).toEqual([3_750_000, 3_750_000, 3_750_000])
    expect(principalsOf(schedule)).toBe(baseInput.principal)
    expect(schedule[3].Interest).toBeLessThan(schedule[0].Interest)
  })

  it('Bullet: interest only, the whole principal on the last installment', () => {
    const schedule = buildLoanSchedule({ ...baseInput, method: LOAN_METHOD_BULLET, months: 6 })
    expect(schedule.slice(0, 5).every((e) => e.Principal === 0)).toBe(true)
    expect(schedule[5].Principal).toBe(baseInput.principal)
  })

  it('grace months pay interest only, then amortize; fees go on every installment', () => {
    const schedule = buildLoanSchedule({ ...baseInput, graceMonths: 3, fee: 2500 })
    expect(schedule.slice(0, 3).every((e) => e.Principal === 0 && e.Interest > 0)).toBe(true)
    expect(schedule[3].Principal).toBeGreaterThan(0)
    expect(schedule.every((e) => e.Fees === 2500)).toBe(true)
    expect(principalsOf(schedule)).toBe(baseInput.principal)
  })

  it('proposes nothing for a first due date on or before the disbursement', () => {
    expect(buildLoanSchedule({ ...baseInput, firstDueDate: baseInput.disbursementDate })).toEqual([])
  })
})

describe('loan reading', () => {
  const loan = {
    Installments: [
      { Number: 1, DueDate: 100, Principal: 5000, Interest: 200, Fees: 50, PaidAmount: 5250 },
      { Number: 2, DueDate: 130, Principal: 5000, Interest: 100, Fees: 50, PaidAmount: 1150 },
    ],
  } as ILoan

  it('counts only the principal a payment covered after fees and interest', () => {
    // Installment 2 received 1150: 50 fees, 100 interest, 1000 principal.
    expect(outstandingPrincipal(loan)).toBe(4000)
  })

  it('reads the next pending installment and each state', () => {
    expect(nextPendingInstallment(loan)?.Number).toBe(2)
    expect(installmentState(loan.Installments[0], 200)).toBe(LOAN_INSTALLMENT_PAID)
    expect(installmentState(loan.Installments[1], 120)).toBe(LOAN_INSTALLMENT_DUE)
    expect(installmentState(loan.Installments[1], 131)).toBe(LOAN_INSTALLMENT_OVERDUE)
  })
})
