import { GetHandler, POST } from '#libs/ui-runtime.svelte.ts'
import type { IEmployee } from '#services/hr/employees.svelte.ts'

// The production settings (docs/MANUFACTURING_PLAN.md §9.1, §9.4): the labor types a recipe and an
// order plan in minutes, and the stages of the orders board.

// IProductionLaborType mirrors production.ProductionLaborType. CostPerHour is PEN cents and only
// estimates: the actual labor comes from the hours logged.
export interface IProductionLaborType {
  ID: number
  Name: string
  CostPerHour: number
  ss: number
  upd: number
}

// IProductionStage mirrors production.ProductionStage: a column of the board, ordered by Position.
export interface IProductionStage {
  ID: number
  Name: string
  Position: number
  ss: number
  upd: number
}

export class ProductionLaborTypesService extends GetHandler<IProductionLaborType> {
  route = 'production-labor-types'
  useCache = { min: 5, ver: 1 }

  handler(result: IProductionLaborType[]): void {
    const laborTypes = (result || []).filter((laborType) => laborType.ss === 1)
    laborTypes.sort((leftType, rightType) => leftType.Name.localeCompare(rightType.Name))
    this.records = laborTypes
    this.recordsMap = new Map(laborTypes.map((laborType) => [laborType.ID, laborType]))
  }

  constructor(init = false) {
    super()
    if (init) this.fetch()
  }
}

export class ProductionStagesService extends GetHandler<IProductionStage> {
  route = 'production-stages'
  useCache = { min: 5, ver: 1 }

  handler(result: IProductionStage[]): void {
    const stages = (result || []).filter((stage) => stage.ss === 1)
    stages.sort((leftStage, rightStage) => leftStage.Position - rightStage.Position || leftStage.ID - rightStage.ID)
    this.records = stages
    this.recordsMap = new Map(stages.map((stage) => [stage.ID, stage]))
  }

  constructor(init = false) {
    super()
    if (init) this.fetch()
  }
}

// Saves one labor type; ss = 0 deletes it.
export const postProductionLaborType = (laborType: IProductionLaborType) => {
  return POST({
    route: 'production-labor-type',
    data: laborType,
    refreshRoutes: ['production-labor-types'],
  }) as Promise<IProductionLaborType>
}

// Saves one stage; ss = 0 deletes it.
export const postProductionStage = (stage: IProductionStage) => {
  return POST({
    route: 'production-stage',
    data: stage,
    refreshRoutes: ['production-stages'],
  }) as Promise<IProductionStage>
}

// The company's workers as production reads them (GET.production-workers): the name, the engagement,
// the department and the employment dates. The labor entries and the "Horas vs planilla" report use it,
// so they need no access to the HR records.
export type IProductionWorker = Pick<IEmployee,
  'ID' | 'FirstName' | 'LastName' | 'SecondLastName' | 'EngagementType' | 'DepartmentID' | 'HireDate' | 'TerminationDate' | 'ss' | 'upd'>

export class ProductionWorkersService extends GetHandler<IProductionWorker> {
  route = 'production-workers'
  useCache = { min: 5, ver: 1 }
  inferRemoveFromStatus = true

  handler(result: IProductionWorker[]): void {
    this.records = []
    this.recordsMap = new Map()
    this.addSavedRecords(...(result || []))
  }

  constructor(init = false) {
    super()
    if (init) this.fetch()
  }
}
