import { GetHandler, POST } from '#libs/ui-runtime.svelte.ts'

// Mirrors backend/hr/types/contracts.go. Served only by the payroll access (it carries salaries).

export interface IContract {
  ID: number
  EmployeeID: number
  /** UnixDay this version takes effect. */
  ValidFrom: number
  ContractType: number
  StartDate: number
  EndDate: number
  /** Cents of PEN. */
  MonthlySalary: number
  DailyHours: number
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

/** Every contract version of the company: a few rows per employee, delta-synced. */
export class ContractsService extends GetHandler<IContract> {
  route = 'hr-contracts'
  useCache = { min: 5, ver: 1 }
  inferRemoveFromStatus = true

  handler(result: IContract[]): void {
    this.records = []
    this.recordsMap = new Map()
    this.addSavedRecords(...(result || []))
  }

  constructor(init = false) {
    super()
    if (init) this.fetch()
  }
}

export const postContract = (contract: IContract): Promise<IContract> =>
  POST({ data: contract, route: 'hr-contracts', refreshRoutes: ['hr-contracts'] })
