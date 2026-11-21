// Loan schedule calculator and loan reading rules. Pure: no Svelte, no fetch.
//
// The calculator only proposes installments: the lender's official schedule always prevails, so the
// user edits the proposal to match it before saving (plan/CASH_FLOW_FORECAST_PLAN.md §8.2).

import type { CalendarActivity } from '#components/calendar/calendar.ts'
import type { ILoan, ILoanInstallment } from '#services/finance/loans.svelte.ts'

export const LOAN_METHOD_FRENCH = 1 // constant installment
export const LOAN_METHOD_GERMAN = 2 // constant principal
export const LOAN_METHOD_BULLET = 3 // interest only, principal at the end

export const LOAN_METHOD_OPTIONS = [
  { id: LOAN_METHOD_FRENCH, name: 'French (constant installment)|Francés (cuota constante)' },
  { id: LOAN_METHOD_GERMAN, name: 'German (constant principal)|Alemán (amortización constante)' },
  { id: LOAN_METHOD_BULLET, name: 'Bullet (principal at the end)|Bullet (capital al final)' },
]

/** AnnualRate is the TEA in percent × 10000: 12.5 % → 125000. */
export const ANNUAL_RATE_SCALE = 1_000_000

export interface ILoanScheduleInput {
  /** Cents. */
  principal: number
  /** TEA in percent × 10000. */
  annualRate: number
  months: number
  /** UnixDay the money arrived: the first period runs from it. */
  disbursementDate: number
  /** UnixDay of the first installment; the next ones fall on the same day of each following month. */
  firstDueDate: number
  method: number
  /** Interest-only installments before amortization starts. */
  graceMonths: number
  /** Fixed fee per installment (desgravamen, commissions), cents. */
  fee: number
}

const MS_PER_DAY = 86400000

/** The same day of the month `monthsAhead` months later, clamped to the month's last day. */
export function addMonthsToUnixDay(unixDay: number, monthsAhead: number): number {
  const date = new Date(unixDay * MS_PER_DAY)
  const targetMonthIndex = date.getUTCMonth() + monthsAhead
  const lastDayOfTarget = new Date(Date.UTC(date.getUTCFullYear(), targetMonthIndex + 1, 0)).getUTCDate()
  const dayOfMonth = Math.min(date.getUTCDate(), lastDayOfTarget)
  return Math.round(Date.UTC(date.getUTCFullYear(), targetMonthIndex, dayOfMonth) / MS_PER_DAY)
}

/** The rate of one period from the TEA, by its actual days over a 360-day year: (1 + TEA)^(d/360) − 1. */
export const periodRate = (annualRate: number, days: number): number =>
  Math.pow(1 + annualRate / ANNUAL_RATE_SCALE, days / 360) - 1

/**
 * Proposes the installments of a loan. Interest of each period is the outstanding balance at that
 * period's rate. French: a constant principal + interest payment over the amortizing periods, with
 * unequal periods (C = P / Σ_k Π_{j≤k} 1/(1+i_j)). German: a constant principal. Bullet: interest
 * only, the principal on the last one. Grace months pay interest only. The last installment absorbs
 * the rounding, so the principals add up to the loan exactly.
 */
export function buildLoanSchedule(input: ILoanScheduleInput): ILoanInstallment[] {
  const { principal, annualRate, months, disbursementDate, firstDueDate, method, fee } = input
  const graceMonths = Math.min(Math.max(input.graceMonths, 0), months - 1)
  if (principal <= 0 || months <= 0 || firstDueDate <= disbursementDate) return []

  const dueDates = Array.from({ length: months }, (_, index) => addMonthsToUnixDay(firstDueDate, index))
  const rates = dueDates.map((dueDate, index) => periodRate(annualRate, dueDate - (index ? dueDates[index - 1] : disbursementDate)))
  const amortizingCount = months - graceMonths

  // French: the payment that a discount over the amortizing periods' rates turns into the principal.
  let discountFactorsSum = 0
  let discountFactor = 1
  for (let index = graceMonths; index < months; index++) {
    discountFactor /= 1 + rates[index]
    discountFactorsSum += discountFactor
  }
  const frenchPayment = discountFactorsSum > 0 ? principal / discountFactorsSum : 0

  let balance = principal
  return dueDates.map((dueDate, index) => {
    const interest = Math.round(balance * rates[index])
    const isLast = index === months - 1
    let installmentPrincipal = 0
    if (index >= graceMonths) {
      if (isLast) {
        installmentPrincipal = balance
      } else if (method === LOAN_METHOD_FRENCH) {
        installmentPrincipal = Math.max(0, Math.round(frenchPayment - balance * rates[index]))
      } else if (method === LOAN_METHOD_GERMAN) {
        installmentPrincipal = Math.round(principal / amortizingCount)
      }
    }
    installmentPrincipal = Math.min(installmentPrincipal, balance)
    balance -= installmentPrincipal
    return { Number: index + 1, DueDate: dueDate, Principal: installmentPrincipal, Interest: interest, Fees: fee, PaidAmount: 0 }
  })
}

export const installmentTotal = (installment: ILoanInstallment): number =>
  installment.Principal + installment.Interest + installment.Fees

/** The principal a payment already covered: a lender applies it to fees, then interest, then principal. */
const paidPrincipalOf = (installment: ILoanInstallment): number =>
  Math.max(0, installment.PaidAmount - installment.Fees - installment.Interest)

/** Principal still owed. */
export const outstandingPrincipal = (loan: ILoan): number =>
  loan.Installments.reduce((total, installment) => total + installment.Principal - paidPrincipalOf(installment), 0)

/** The first installment not fully paid, or undefined when the loan is paid off. */
export const nextPendingInstallment = (loan: ILoan): ILoanInstallment | undefined =>
  loan.Installments.find((installment) => installment.PaidAmount < installmentTotal(installment))

export const LOAN_INSTALLMENT_PAID = 1
export const LOAN_INSTALLMENT_DUE = 2
export const LOAN_INSTALLMENT_OVERDUE = 3

export function installmentState(installment: ILoanInstallment, today: number): number {
  if (installment.PaidAmount >= installmentTotal(installment)) return LOAN_INSTALLMENT_PAID
  return installment.DueDate < today ? LOAN_INSTALLMENT_OVERDUE : LOAN_INSTALLMENT_DUE
}

const STATE_COLORS: Record<number, CalendarActivity['color']> = {
  [LOAN_INSTALLMENT_PAID]: 'green',
  [LOAN_INSTALLMENT_DUE]: 'blue',
  [LOAN_INSTALLMENT_OVERDUE]: 'red',
}

/** Every installment of the active loans as a calendar card, colored paid / due / overdue. */
export function loanCalendarActivities(
  loans: ILoan[], today: number, formatAmount: (cents: number) => string,
): CalendarActivity[] {
  return loans.flatMap((loan) => loan.Installments.map((installment) => ({
    date: installment.DueDate,
    title: `${loan.Name} #${installment.Number}`,
    text: formatAmount(installmentTotal(installment) - installment.PaidAmount),
    color: STATE_COLORS[installmentState(installment, today)],
    icon: 'icon-[fa--university]',
  })))
}

/** What the loan form misses before it can be saved; the backend checks the schedule again. */
export function loanFormProblem(loan: ILoan): string {
  if (!loan.Name?.trim()) return 'Enter the loan name|Ingrese el nombre del préstamo'
  if (!loan.LenderID || !loan.CashBankID) return 'Select the lender and the account|Seleccione la entidad y la caja o banco'
  if (!(loan.Principal > 0) || !loan.DisbursementDate) return 'Enter the amount and the disbursement date|Ingrese el monto y la fecha del desembolso'
  if (!loan.Installments.length) return 'Calculate or enter the installments|Calcule o ingrese las cuotas'
  const principalsTotal = loan.Installments.reduce((total, installment) => total + installment.Principal, 0)
  if (principalsTotal !== loan.Principal) {
    return 'The installments\' principal does not add up to the loan|El capital de las cuotas no suma el monto del préstamo'
  }
  return ''
}
