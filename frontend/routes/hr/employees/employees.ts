// Business rules of the employees page. No Svelte, no fetch: unit-testable in isolation.
import type { IOrgTreeNode } from '#components/charts/org-tree-layout.ts'
import type { IContractAgreement } from '#services/hr/contracts-agreements.svelte.ts'
import type { IDependent, IEmployee } from '#services/hr/employees.svelte.ts'
import { Engagement, PayBasis } from '#services/hr/hr-catalogs.ts'

/** Employee `ss`: a terminated worker keeps every record, a deleted one leaves the list. */
export const EmployeeStatus = { DELETED: 0, ACTIVE: 1, TERMINATED: 2 } as const

/** Peru, the default nationality (the backend applies the same default). */
export const COUNTRY_ID_PERU = 604
export const IDENTITY_DOC_DNI = 1
export const PENSION_SYSTEM_ONP = 1
export const CONTRACT_TYPE_INDEFINITE = 1
/** Below this many hours a day the worker is part-time: no CTS, reduced vacations. */
export const PART_TIME_DAILY_HOURS = 4
export const FULL_TIME_DAILY_HOURS = 8

/** "Last Second-last, First": the order HR lists people in. */
export const employeeFullName = (employee: Pick<IEmployee, 'FirstName' | 'LastName' | 'SecondLastName'>): string => {
  const surnames = [employee.LastName, employee.SecondLastName].filter(Boolean).join(' ')
  return [surnames, employee.FirstName].filter(Boolean).join(', ')
}

export const isTerminated = (employee: IEmployee): boolean => employee.ss === EmployeeStatus.TERMINATED

export const isOnPayroll = (employee: Pick<IEmployee, 'EngagementType'>): boolean =>
  employee.EngagementType === Engagement.Payroll

/** The workers the payroll pages list (time off, payslips, payroll records): Planilla only. Workers
 * off payroll are registered in HR only to be picked on a labor entry (plan §9.5, M25). */
export const payrollEmployees = <T extends Pick<IEmployee, 'EngagementType'>>(employees: T[]): T[] =>
  employees.filter(isOnPayroll)

/** The PayBasis an engagement forces: Planilla is monthly, a fee receipt is per hour. A temporary
 * worker picks it, so there is none. */
export const fixedPayBasisOf = (engagementType: number): number | undefined => {
  if (engagementType === Engagement.Payroll) return PayBasis.Monthly
  if (engagementType === Engagement.FeeReceipt) return PayBasis.Hourly
  return undefined
}

/** ContractType, pension, CUSPP, AFP type, EPS and CTS only apply to a payroll contract. */
export const showsPayrollFields = (engagementType: number): boolean => engagementType === Engagement.Payroll

export const showsPayDayOfWeek = (payBasis: number): boolean => payBasis === PayBasis.Weekly

/** The PayAmount label: what one unit of the basis pays. */
export const payAmountLabel = (payBasis: number): string => {
  if (payBasis === PayBasis.Weekly) return 'Weekly pay (S/)|Pago semanal (S/)'
  if (payBasis === PayBasis.Hourly) return 'Hourly pay (S/)|Pago por hora (S/)'
  return 'Monthly pay (S/)|Pago mensual (S/)'
}

/** Aligns an agreement with the employee's engagement, as the backend validates it: the forced
 * PayBasis, a pay day only when weekly, and no payroll fields off payroll. */
export const alignAgreementWithEngagement = (agreement: IContractAgreement, engagementType: number): IContractAgreement => {
  const payBasis = fixedPayBasisOf(engagementType) || agreement.PayBasis || PayBasis.Monthly
  const aligned = { ...agreement, PayBasis: payBasis }
  if (!showsPayDayOfWeek(payBasis)) aligned.PayDayOfWeek = 0
  if (!showsPayrollFields(engagementType)) {
    Object.assign(aligned, {
      ContractType: 0, PensionSystem: 0, CUSPP: '', AfpCommissionType: 0, HasEPS: false, CtsBankID: 0, CtsAccount: '',
    })
  }
  return aligned
}

/** An AFP affiliate needs a CUSPP and a commission type; ONP does not. */
export const isAfpPensionSystem = (pensionSystem: number): boolean => pensionSystem >= 2 && pensionSystem <= 5

export const isPartTime = (contract: Pick<IContractAgreement, 'DailyHours'>): boolean =>
  contract.DailyHours > 0 && contract.DailyHours < PART_TIME_DAILY_HOURS

/** An employee's contract versions, newest first. */
export const employeeContracts = (contracts: IContractAgreement[], employeeID: number): IContractAgreement[] =>
  contracts
    .filter((contract) => contract.EmployeeID === employeeID && contract.ss > 0)
    .sort((left, right) => right.ValidFrom - left.ValidFrom)

/** The version in force on `unixDay`: the latest ValidFrom not after it. Undefined before the first one. */
export const contractValidOn = (contracts: IContractAgreement[], employeeID: number, unixDay: number): IContractAgreement | undefined =>
  employeeContracts(contracts, employeeID).find((contract) => contract.ValidFrom <= unixDay)

/** The years completed between two UnixDays. */
export const completedYears = (fromUnixDay: number, toUnixDay: number): number => {
  if (!fromUnixDay || toUnixDay < fromUnixDay) return 0
  const fromDate = new Date(fromUnixDay * 86400000)
  const toDate = new Date(toUnixDay * 86400000)
  let years = toDate.getUTCFullYear() - fromDate.getUTCFullYear()
  const beforeAnniversary = toDate.getUTCMonth() < fromDate.getUTCMonth() ||
    (toDate.getUTCMonth() === fromDate.getUTCMonth() && toDate.getUTCDate() < fromDate.getUTCDate())
  if (beforeAnniversary) years--
  return years
}

/** Whether the worker earns the asignación familiar on a day: the server derives the ranges from his
 * children (a minor, or studying up to 24) on every dependent change. */
export const hasFamilyAllowanceOn = (employee: Pick<IEmployee, 'FamilyAllowance'>, unixDay: number): boolean =>
  (employee.FamilyAllowance || []).some((range) => range.From <= unixDay && unixDay <= range.To)

/** Active employees per department, for the organization tab. */
export const countActiveByDepartment = (employees: IEmployee[]): Map<number, number> => {
  const counts = new Map<number, number>()
  for (const employee of employees) {
    if (employee.ss !== EmployeeStatus.ACTIVE) continue
    counts.set(employee.DepartmentID, (counts.get(employee.DepartmentID) || 0) + 1)
  }
  return counts
}

/** An org chart card names its department only where it changes: on a root, or when the boss sits
 * in another department. */
export const showsDepartmentInOrgChart = (employee: IEmployee, boss: IEmployee | undefined): boolean =>
  !boss || boss.DepartmentID !== employee.DepartmentID

/** The org chart: active employees nested by ManagerEmployeeID, sorted by name. An employee whose
 * boss is missing, terminated or deleted is a root; a cycle is cut where it closes. */
export const buildEmployeeTree = (employees: IEmployee[]): IOrgTreeNode<IEmployee>[] => {
  const active = employees.filter((employee) => employee.ss === EmployeeStatus.ACTIVE)
  const activeIDs = new Set(active.map((employee) => employee.ID))
  const byName = (left: IEmployee, right: IEmployee) => employeeFullName(left).localeCompare(employeeFullName(right))
  const subordinatesByManager = new Map<number, IEmployee[]>()
  for (const employee of active) {
    const managerID = activeIDs.has(employee.ManagerEmployeeID) ? employee.ManagerEmployeeID : 0
    subordinatesByManager.set(managerID, [...(subordinatesByManager.get(managerID) || []), employee])
  }
  const placed = new Set<number>()
  const buildNode = (employee: IEmployee): IOrgTreeNode<IEmployee> => {
    placed.add(employee.ID)
    const children = (subordinatesByManager.get(employee.ID) || [])
      .filter((subordinate) => !placed.has(subordinate.ID))
      .sort(byName)
      .map(buildNode)
    return { item: employee, children }
  }
  const roots = (subordinatesByManager.get(0) || []).sort(byName).map(buildNode)
  // People only reachable through a cycle become roots, so none disappears from the chart.
  for (const employee of active.sort(byName)) {
    if (!placed.has(employee.ID)) roots.push(buildNode(employee))
  }
  return roots
}

export const makeEmptyEmployee = (): IEmployee => ({
  ID: 0, UserID: 0, IdentityDocType: IDENTITY_DOC_DNI, DocumentNumber: '', FirstName: '', LastName: '',
  SecondLastName: '', BirthDate: 0, Sex: 0, CountryID: COUNTRY_ID_PERU, Address: '', CityID: 0, Email: '',
  Phone: '', DepartmentID: 0, PositionID: 0, ManagerEmployeeID: 0, HireDate: 0, TerminationDate: 0, TerminationReason: 0,
  EngagementType: Engagement.Payroll, DependentsCount: 0, FamilyAllowance: [], ss: EmployeeStatus.ACTIVE, upd: 0,
})

/** A new version starts from the one in force, so a raise only changes what changed. It follows the
 * employee's current engagement: a worker moved from Temporal to Planilla gets a monthly contract. */
export const makeNextContract = (employee: IEmployee, currentContract: IContractAgreement | undefined): IContractAgreement => {
  if (currentContract) {
    return alignAgreementWithEngagement({ ...currentContract, ID: 0, ValidFrom: 0, ss: 1, upd: 0 }, employee.EngagementType)
  }
  return alignAgreementWithEngagement({
    ID: 0, EmployeeID: employee.ID, ValidFrom: 0, ContractType: CONTRACT_TYPE_INDEFINITE,
    StartDate: employee.HireDate, EndDate: 0, PayAmount: 0, PayBasis: PayBasis.Monthly, PayDayOfWeek: 0,
    DailyHours: FULL_TIME_DAILY_HOURS, PensionSystem: PENSION_SYSTEM_ONP, CUSPP: '', AfpCommissionType: 0, HasEPS: false,
    SalaryBankID: 0, SalaryAccount: '', CtsBankID: 0, CtsAccount: '', ss: 1, upd: 0,
  }, employee.EngagementType)
}

export const makeEmptyDependent = (employeeID: number, todayUnixDay: number): IDependent => ({
  ID: 0, EmployeeID: employeeID, Relationship: 0, IdentityDocType: IDENTITY_DOC_DNI, DocumentNumber: '',
  FirstName: '', LastName: '', SecondLastName: '', BirthDate: 0, IsStudying: false,
  ValidFrom: todayUnixDay, ValidTo: 0, ss: 1, upd: 0,
})
