import { POST } from '#libs/ui-runtime.svelte.ts'
import type { IEmployeeRecord } from './employee-records.ts'
import { PinnedListService } from './payroll.svelte.ts'

// The services of the employee records (types and constants in employee-records.ts).

/** Every record of one year (`hr-employee-records?year=`): time off, and the months the payroll pages filter. */
export class EmployeeRecordsYearService extends PinnedListService<IEmployeeRecord> {
  constructor(year: number) { super(`hr-employee-records?year=${year}`) }
}

/** Every record of one employee (`hr-employee-records?employee-id=`): his loans and their installments. */
export class EmployeeRecordsService extends PinnedListService<IEmployeeRecord> {
  constructor(employeeID: number) { super(`hr-employee-records?employee-id=${employeeID}`) }
}

/** Refreshes are prefix-matched: this one marks every year and employee list stale, since a record
 * lives in one of each (and a time off's year can change on edit). */
const EMPLOYEE_RECORDS_ROUTES = ['hr-employee-records']

/** Creates, edits or cancels (ss 0) a novedad or a time off, or moves a loan installment to another month. */
export const postEmployeeRecord = (record: IEmployeeRecord): Promise<IEmployeeRecord> =>
  POST({ data: record, route: 'hr-employee-records', refreshRoutes: EMPLOYEE_RECORDS_ROUTES })

/** Registers advances of several employees at once, or voids one (ss 0). */
export const postPayAdvances = (advances: IEmployeeRecord[]): Promise<IEmployeeRecord[]> =>
  POST({ data: advances, route: 'hr-pay-advances', refreshRoutes: EMPLOYEE_RECORDS_ROUTES })

/** Grants a loan (answers it with its installments) or deletes one (ss 0, answers what it cancelled). */
export const postLoan = (loan: IEmployeeRecord): Promise<IEmployeeRecord[]> =>
  POST({ data: loan, route: 'hr-loans', refreshRoutes: EMPLOYEE_RECORDS_ROUTES })
