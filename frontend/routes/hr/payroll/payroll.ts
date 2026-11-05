// Pure business logic of the payroll pages: periods, run states, and how a payslip reads on the
// boleta. No Svelte, no fetch. The server calculates; nothing here computes pay.

import type { IConcept, IPayrollRun, IPayslip, IPayslipLine } from '#services/hr/payroll.svelte.ts'
import { LEGAL_CONCEPTS } from '#services/hr/hr-catalogs.ts'

/** Mirrors backend/hr/types/payroll.go and legal_concepts.go. */
export const PayrollStatus = { Deleted: 0, Open: 1, Closed: 2, Paid: 3 } as const
export const ConceptKind = { Earning: 1, Deduction: 2, Employer: 3, Provision: 4 } as const

/** Profit sharing rates by sector (DL 892), basis points. */
export const PROFIT_SHARING_RATE_OPTIONS = [
  { id: 1000, name: '10 % — fishing, telecommunications, industry|10 % — pesca, telecomunicaciones, industria' },
  { id: 800, name: '8 % — mining, trade, restaurants|8 % — minería, comercio, restaurantes' },
  { id: 500, name: '5 % — other activities|5 % — otras actividades' },
]

/** One month of the payroll list; Run is missing when nobody is paid in it. */
export interface IPayrollMonth {
  Period: number
  Run?: IPayrollRun
}

/** Every month from firstPeriod to lastPeriod (the window the server answers), newest first, each
 * with its run if it has one. Without a window (the user cannot calculate) only the runs show. */
export const payrollMonths = (runs: IPayrollRun[], window?: { FirstPeriod: number, LastPeriod: number }): IPayrollMonth[] => {
  const runByPeriod = new Map(runs.filter((run) => run.ss > 0).map((run) => [run.Period, run]))
  const periods = new Set(runByPeriod.keys())
  if (window) {
    for (let period = window.FirstPeriod; period <= window.LastPeriod; period = nextPeriod(period)) periods.add(period)
  }
  return [...periods].sort((left, right) => right - left).map((period) => ({ Period: period, Run: runByPeriod.get(period) }))
}

const MONTH_NAMES = ['Ene', 'Feb', 'Mar', 'Abr', 'May', 'Jun', 'Jul', 'Ago', 'Set', 'Oct', 'Nov', 'Dic']

/** 2610 → "Oct 2026". */
export const formatPeriod = (period: number): string =>
  `${MONTH_NAMES[(period % 100) - 1] || '?'} ${2000 + Math.floor(period / 100)}`

/** The YYMM of a UnixDay. */
export const periodOfDay = (unixDay: number): number => {
  const date = new Date(unixDay * 86400 * 1000)
  return (date.getUTCFullYear() % 100) * 100 + date.getUTCMonth() + 1
}

export const nextPeriod = (period: number): number => (period % 100 === 12 ? period - 11 + 100 : period + 1)
export const previousPeriod = (period: number): number => (period % 100 === 1 ? period + 11 - 100 : period - 1)

/** UnixDay of the first and last day of a YYMM period. */
export const periodFirstDay = (period: number): number =>
  Date.UTC(2000 + Math.floor(period / 100), (period % 100) - 1, 1) / 86400000
export const periodLastDay = (period: number): number => periodFirstDay(nextPeriod(period)) - 1

export type RunState = 'draft' | 'calculated' | 'closed' | 'paid'

/** Draft and calculated are both Status 1, told apart by CalculatedAt. */
export const runState = (run: IPayrollRun): RunState => {
  if (run.ss === PayrollStatus.Paid) return 'paid'
  if (run.ss === PayrollStatus.Closed) return 'closed'
  return run.CalculatedAt > 0 ? 'calculated' : 'draft'
}

export const RUN_STATE_LABELS: Record<RunState, string> = {
  draft: 'Draft|Borrador',
  calculated: 'Calculated|Calculada',
  closed: 'Closed|Cerrada',
  paid: 'Paid|Pagada',
}

/** Which transitions the run allows, before checking the user's sub-accesses. */
export const allowedTransitions = (run: IPayrollRun) => {
  const state = runState(run)
  return {
    calculate: state === 'draft' || state === 'calculated',
    close: state === 'calculated',
    reopen: state === 'closed',
    pay: state === 'closed',
    edit: state === 'draft' || state === 'calculated',
  }
}

const legalConceptsByID = new Map(LEGAL_CONCEPTS.map((concept) => [concept.id, concept]))

/** The label of a payslip line: a legal concept from the catalog, or the company concept. */
export const payslipLineName = (line: IPayslipLine, companyConcepts: Map<number, IConcept>): string => {
  if (line.LegalConceptID > 0) return legalConceptsByID.get(line.LegalConceptID)?.name || `#${line.LegalConceptID}`
  return companyConcepts.get(line.ConceptID)?.Name || `#${line.ConceptID}`
}

/** The boleta's sections: earnings, worker deductions, employer contributions. Provisions are accounting only. */
export const payslipSections = (payslip: IPayslip) => ({
  earnings: (payslip.Lines || []).filter((line) => line.Kind === ConceptKind.Earning),
  deductions: (payslip.Lines || []).filter((line) => line.Kind === ConceptKind.Deduction),
  employer: (payslip.Lines || []).filter((line) => line.Kind === ConceptKind.Employer),
  provisions: (payslip.Lines || []).filter((line) => line.Kind === ConceptKind.Provision),
})

export const hasBlockingDiagnostics = (payslip: IPayslip): boolean =>
  (payslip.Diagnostics || []).some((diagnostic) => diagnostic.Blocking)

/** Totals of the payslips of one run, for the run header. */
export const runTotals = (payslips: IPayslip[]) => payslips.reduce((totals, payslip) => {
  totals.earnings += payslip.TotalEarnings
  totals.deductions += payslip.TotalDeductions
  totals.netPay += payslip.NetPay
  totals.employerCost += payslip.EmployerCost
  totals.blocked += hasBlockingDiagnostics(payslip) ? 1 : 0
  return totals
}, { earnings: 0, deductions: 0, netPay: 0, employerCost: 0, blocked: 0 })

const CURRENCY_PEN = 1

/** Payroll is paid in soles: only active PEN cash registers and bank accounts can pay it. */
export const payrollCashBankOptions = <T extends { ID: number, Name: string, CurrencyType: number, ss: number }>(cashBanks: T[]) =>
  cashBanks.filter((cashBank) => cashBank.ss > 0 && cashBank.CurrencyType === CURRENCY_PEN)

/** The payables a run owes to third parties, paid through PLAME, AFPnet or to a beneficiary. */
export const CONTRIBUTION_ACCOUNTS = [
  { code: 4031, name: 'EsSalud|EsSalud' },
  { code: 4032, name: 'ONP|ONP' },
  { code: 40173, name: 'Income tax 5th category|Renta de 5ta categoría' },
  { code: 417, name: 'AFP|AFP' },
  { code: 419, name: 'Judicial withholdings|Retenciones judiciales' },
]

/** What a run's payslips credited to each contribution payable (before payments, which the server nets). */
export const runContributionTotals = (payslips: IPayslip[]): Map<number, number> => {
  const contributionCodes = new Set(CONTRIBUTION_ACCOUNTS.map((account) => account.code))
  const totals = new Map<number, number>()
  for (const payslip of payslips) {
    for (const line of payslip.Lines || []) {
      const owedToThirdParty = line.Kind === ConceptKind.Deduction || line.Kind === ConceptKind.Employer
      if (owedToThirdParty && contributionCodes.has(line.PayableAccountCode)) {
        totals.set(line.PayableAccountCode, (totals.get(line.PayableAccountCode) || 0) + line.Amount)
      }
    }
  }
  return totals
}

/** Cents → "1,234.56". */
export const formatCents = (cents: number): string =>
  (cents / 100).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })

/** Basis points → "9.00 %". */
export const formatRate = (basisPoints: number): string => `${(basisPoints / 100).toFixed(2)} %`
