import { GetHandler, POST } from '#libs/ui-runtime.svelte.ts'

// Mirrors backend/hr/types/employees.go. The employee record is label-safe on purpose: salary,
// pension and bank data live in the contracts (payroll access only).

export interface IEmployee {
  ID: number
  UserID: number
  IdentityDocType: number
  DocumentNumber: string
  FirstName: string
  LastName: string
  SecondLastName: string
  BirthDate: number
  Sex: number
  CountryID: number
  Address: string
  CityID: number
  Email: string
  Phone: string
  DepartmentID: number
  PositionID: number
  /** Immediate boss (0 = none): the edge the org chart draws. */
  ManagerEmployeeID: number
  HireDate: number
  TerminationDate: number
  TerminationReason: number
  /** Summary of the dependents, kept by the server on every dependent change. */
  DependentsCount: number
  /** The day ranges a child earns the worker the asignación familiar; the payroll reads them. */
  FamilyAllowance: IDayRange[]
  ss: number
  upd: number
}

export interface IDayRange {
  From: number
  To: number
}

export interface IDependent {
  ID: number
  EmployeeID: number
  Relationship: number
  IdentityDocType: number
  DocumentNumber: string
  FirstName: string
  LastName: string
  SecondLastName: string
  BirthDate: number
  IsStudying: boolean
  ValidFrom: number
  ValidTo: number
  ss: number
  upd: number
}

/** Active and terminated employees, delta-synced. `routeByID` resolves any other ID a fact carries. */
export class EmployeesService extends GetHandler<IEmployee> {
  route = 'hr-employees'
  routeByID = 'hr-employees-ids'
  useCache = { min: 5, ver: 2 }
  inferRemoveFromStatus = true
  prependOnSave = true

  handler(result: IEmployee[]): void {
    this.records = []
    this.recordsMap = new Map()
    this.addSavedRecords(...(result || []))
  }

  constructor(init = false) {
    super()
    if (init) this.fetch()
  }
}

export const postEmployee = (employee: IEmployee): Promise<IEmployee> =>
  POST({ data: employee, route: 'hr-employees', refreshRoutes: ['hr-employees'] })

/** One employee's dependents: a delta slice pinned by `employee-id`, fetched when his panel opens. */
export class DependentsService extends GetHandler<IDependent> {
  route = ''
  useCache = { min: 5, ver: 2 }
  inferRemoveFromStatus = true

  handler(result: IDependent[]): void {
    this.records = []
    this.recordsMap = new Map()
    this.addSavedRecords(...(result || []))
  }

  constructor(employeeID: number) {
    super()
    this.route = `hr-dependents?employee-id=${employeeID}`
    this.fetch()
  }
}

// The employee's dependents come back with the write; the server rewrites the employee's dependents
// summary too, so `hr-employees` is marked and read by the page that shows it.
export const postDependent = (dependents: DependentsService, dependent: IDependent): Promise<IDependent> =>
  dependents.writeAndSync('hr-dependents', dependent, ['hr-employees'])
