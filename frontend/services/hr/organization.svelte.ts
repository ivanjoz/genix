import { GetHandler, POST } from '#libs/ui-runtime.svelte.ts'

// Mirrors backend/hr/types/organization.go.

export interface IDepartment {
  ID: number
  Name: string
  ParentID: number
  ManagerEmployeeID: number
  CostFunction: number
  ss: number
  upd: number
}

export interface IPosition {
  ID: number
  Name: string
  DepartmentID: number
  OccupationCode: string
  ss: number
  upd: number
}

interface IOrganizationResponse {
  Departments?: IDepartment[]
  Positions?: IPosition[]
}

/** Departments and positions in one response (`hr-org`), one delta cursor per key. */
export class OrganizationService extends GetHandler {
  route = 'hr-org'
  useCache = { min: 5, ver: 1 }

  Departments: IDepartment[] = $state([])
  DepartmentsMap: Map<number, IDepartment> = $state(new Map())
  Positions: IPosition[] = $state([])
  PositionsMap: Map<number, IPosition> = $state(new Map())

  handler(result: IOrganizationResponse): void {
    this.setDepartments((result.Departments || []).filter((department) => department.ss > 0))
    this.setPositions((result.Positions || []).filter((position) => position.ss > 0))
  }

  setDepartments(departments: IDepartment[]) {
    this.Departments = departments.sort((left, right) => left.Name.localeCompare(right.Name))
    this.DepartmentsMap = new Map(this.Departments.map((department) => [department.ID, department]))
  }

  setPositions(positions: IPosition[]) {
    this.Positions = positions.sort((left, right) => left.Name.localeCompare(right.Name))
    this.PositionsMap = new Map(this.Positions.map((position) => [position.ID, position]))
  }

  constructor(init = false) {
    super()
    if (init) this.fetch()
  }
}

export const postDepartment = (department: IDepartment): Promise<IDepartment> =>
  POST({ data: department, route: 'hr-departments', refreshRoutes: ['hr-org'] })

export const postPosition = (position: IPosition): Promise<IPosition> =>
  POST({ data: position, route: 'hr-positions', refreshRoutes: ['hr-org'] })
