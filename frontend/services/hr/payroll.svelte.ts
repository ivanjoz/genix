import { GetHandler, POST } from '#libs/ui-runtime.svelte.ts'

// Mirrors backend/hr/types/concepts.go, payroll.go and employee_concepts.go (the dated records are
// in employee-records.svelte.ts). Money is cents of PEN,
// rates are basis points, periods are YYMM. The server calculates; these are its stored results.

export interface IConcept {
  ID: number
  Name: string
  Kind: number
  PlameCode: string
  IsRemunerative: boolean
  AffectsEssalud: boolean
  AffectsPension: boolean
  IncomeTax: number
  ComputableForBenefits: boolean
  ExpenseAccountCode: number
  PayableAccountCode: number
  ss: number
  upd: number
}

export interface IPayrollRun {
  ID: number
  Period: number
  PayDate: number
  Note: string
  EmployeeCount: number
  TotalNetPay: number
  TotalEmployerCost: number
  CalculatedAt: number
  ClosedAt: number
  ClosedBy: number
  PaidAt: number
  /** The cash or bank account that paid the net pay. */
  CashBankID: number
  /** Profit sharing of the month, when the run distributes it: the previous fiscal year's net income
   * before tax (cents) and the sector rate (basis points). Zero distributes nothing. */
  NetIncomeBeforeTax: number
  SharingRate: number
  ss: number
  upd: number
}

export interface IPayslipLine {
  LegalConceptID: number
  ConceptID: number
  Kind: number
  Base: number
  Rate: number
  Quantity: number
  Amount: number
  PlameCode: string
  IsRemunerative: boolean
  IncomeTax: number
  ComputableForBenefits: boolean
  AccountCode: number
  PayableAccountCode: number
  SourceID: number
  /** The part of a benefit paid from its provision (4114 / 415 / 4115). */
  ProvisionAccountCode: number
  ProvisionedAmount: number
  /** An earning not paid through the payslip (the CTS is deposited in the worker's bank): it adds to the earnings but not to the net pay. */
  OutsideNetPay: boolean
}

export interface IPayslipDiagnostic {
  Code: string
  Message: string
  Blocking: boolean
}

export interface IPayslipParameters {
  MinimumWage: number
  TaxUnit: number
  PensionSystem: number
  AfpFundRate: number
  AfpPremiumRate: number
  AfpCommissionRate: number
  OnpRate: number
  EssaludRate: number
  MaxInsurable: number
}

export interface IPayslip {
  ID: number
  RunID: number
  Period: number
  EmployeeID: number
  ContractID: number
  DepartmentID: number
  CostFunction: number
  WorkedDays: number
  Lines: IPayslipLine[]
  TotalEarnings: number
  TotalDeductions: number
  NetPay: number
  EmployerCost: number
  Parameters: IPayslipParameters
  Diagnostics: IPayslipDiagnostic[]
  CalculatedAt: number
  ss: number
  upd: number
}

export interface IEmployeeConcept {
  ID: number
  EmployeeID: number
  ConceptID: number
  Amount: number
  FromPeriod: number
  ToPeriod: number
  Note: string
  ss: number
  upd: number
}

interface IPayrollConfigResponse {
  Concepts?: IConcept[]
  Runs?: IPayrollRun[]
}

/** Company concepts and payroll runs in one response (`hr-payroll-config`), one cursor each. */
export class PayrollConfigService extends GetHandler {
  route = 'hr-payroll-config'
  useCache = { min: 5, ver: 1 }

  Concepts: IConcept[] = $state([])
  ConceptsMap: Map<number, IConcept> = $state(new Map())
  Runs: IPayrollRun[] = $state([])

  handler(result: IPayrollConfigResponse): void {
    this.Concepts = (result.Concepts || []).filter((concept) => concept.ss > 0)
      .sort((left, right) => left.Name.localeCompare(right.Name))
    this.ConceptsMap = new Map(this.Concepts.map((concept) => [concept.ID, concept]))
    this.Runs = (result.Runs || []).filter((run) => run.ss > 0)
      .sort((left, right) => right.Period - left.Period)
  }

  constructor(init = false) {
    super()
    if (init) this.fetch()
  }
}

/** A delta list pinned by one query value (a period or an employee): each value is its own cache. */
export abstract class PinnedListService<T extends { ID: number, ss: number }> extends GetHandler<T> {
  useCache = { min: 5, ver: 1 }
  inferRemoveFromStatus = true

  handler(result: T[]): void {
    this.records = []
    this.recordsMap = new Map()
    this.addSavedRecords(...(result || []))
  }

  constructor(route: string) {
    super()
    this.route = route
    this.fetch()
  }
}

// The subclasses declare no fields: a field initializer would run after super() and undo the
// route the base constructor set before fetching.
export class PayslipsService extends PinnedListService<IPayslip> {
  constructor(period: number) { super(`hr-payslips?period=${period}`) }
}

export class EmployeeConceptsService extends PinnedListService<IEmployeeConcept> {
  constructor(employeeID: number) { super(`hr-employee-concepts?employee-id=${employeeID}`) }
}

export const postConcept = (concept: IConcept): Promise<IConcept> =>
  POST({ data: concept, route: 'hr-concepts', refreshRoutes: ['hr-payroll-config'] })

/** Edits an open run's pay date, note and profit sharing; the server recalculates it. */
export const postPayrollRun = (run: IPayrollRun): Promise<unknown> =>
  POST({ data: run, route: 'hr-payroll-runs', refreshRoutes: ['hr-payroll-config'] })

export interface IPayrollEnsureResult {
  /** The first and last month (YYMM) the payroll list covers. */
  FirstPeriod: number
  LastPeriod: number
  Skipped: { Period: number, EmployeeID: number, Message: string }[]
}

/** Creates the months of the list that have something to pay and recalculates the open ones whose inputs changed. */
export const postPayrollEnsure = (): Promise<IPayrollEnsureResult> =>
  POST({ data: {}, route: 'hr-payroll-ensure', refreshRoutes: ['hr-payroll-config'] })

export type PayrollTransition = 'hr-payroll-calculate' | 'hr-payroll-close' | 'hr-payroll-reopen' | 'hr-payroll-paid'

/** Calculates, closes, reopens or pays a run (paying needs the CashBankID); closing and reopening
 * also settle or free the employee records it consumed. */
export const postPayrollTransition = (transition: PayrollTransition, run: IPayrollRun, cashBankID = 0): Promise<unknown> =>
  POST({
    data: { RunID: run.ID, CashBankID: cashBankID },
    route: transition,
    refreshRoutes: ['hr-payroll-config', `hr-payslips?period=${run.Period}`, 'hr-employee-records'],
  })

export interface IContributionsPayment {
  RunID: number
  CashBankID: number
  /** The payable it settles: 4031 EsSalud, 4032 ONP, 40173 5ta, 417 AFP, 419 judicial withholding. */
  AccountCode: number
  Amount: number
  Date: number
}

/** Pays to a third party what a closed run withheld or owes on one payable; answers what is still pending. */
export const postPayrollContributions = (payment: IContributionsPayment): Promise<{ Pending: number }> =>
  POST({ data: payment, route: 'hr-payroll-contributions', refreshRoutes: [] })

export const postEmployeeConcept = (record: IEmployeeConcept): Promise<IEmployeeConcept> =>
  POST({ data: record, route: 'hr-employee-concepts', refreshRoutes: [`hr-employee-concepts?employee-id=${record.EmployeeID}`] })
