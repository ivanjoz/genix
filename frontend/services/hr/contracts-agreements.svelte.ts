import { GetHandler, POST } from '#libs/ui-runtime.svelte.ts'

// Mirrors backend/hr/types/contracts_agreements.go. One row is the terms of one worker from one date:
// a payroll contract (Planilla) or a non-payroll agreement (plan §9.5, M30). Served only by the
// payroll access (it carries pay amounts).

export interface IContractAgreement {
  ID: number
  EmployeeID: number
  /** UnixDay this version takes effect. */
  ValidFrom: number
  /** Payroll only (T-Registro); 0 on a non-payroll agreement. */
  ContractType: number
  StartDate: number
  EndDate: number
  /** Cents of PEN per month, week or hour, by PayBasis. */
  PayAmount: number
  /** PayBasis.Monthly, Weekly or Hourly (hr-catalogs.ts). A payroll contract is always monthly. */
  PayBasis: number
  /** Weekly pay day, 1 Monday … 7 Sunday; 0 unless weekly. */
  PayDayOfWeek: number
  DailyHours: number
  // Payroll only: 0 / empty on a non-payroll agreement.
  PensionSystem: number
  CUSPP: string
  AfpCommissionType: number
  HasEPS: boolean
  SalaryBankID: number
  SalaryAccount: string
  CtsBankID: number
  CtsAccount: string
  ss: number
  upd: number
}

/** Every contract and agreement version of the company: a few rows per employee, delta-synced. */
export class ContractsAgreementsService extends GetHandler<IContractAgreement> {
  route = 'hr-contracts-agreements'
  useCache = { min: 5, ver: 2 }
  inferRemoveFromStatus = true

  handler(result: IContractAgreement[]): void {
    this.records = []
    this.recordsMap = new Map()
    this.addSavedRecords(...(result || []))
  }

  constructor(init = false) {
    super()
    if (init) this.fetch()
  }
}

export const postContractAgreement = (contract: IContractAgreement): Promise<IContractAgreement> =>
  POST({ data: contract, route: 'hr-contracts-agreements', refreshRoutes: ['hr-contracts-agreements'] })
