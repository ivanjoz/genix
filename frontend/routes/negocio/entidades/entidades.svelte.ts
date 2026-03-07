import { GetHandler, POST } from '$libs/http.svelte'

export const EntityType = {
  CLIENT: 1,
  PROVIDER: 2,
} as const

export const PersonType = {
  PERSON: 1,
  COMPANY: 2,
} as const

export interface IEntity {
  ID: number
  Type: number
  Name: string
  RegistryNumber: string
  PersonType: number
  Email: string
  CountryID: number
  CityID: string
  ss: number
  upd: number
  UpdatedBy?: number
}

export const postEntities = (entitiesPayload: IEntity[]) => {
  return POST({
    data: entitiesPayload,
    route: 'entities',
    // Keep both entity-type routes hot after create/update mutations.
    refreshRoutes: ['entities?type=1', 'entities?type=2'],
  })
}

export class EntitiesService extends GetHandler<IEntity> {
  route = ''
  keyID = 'ID'
  useCache = { min: 5, ver: 1 }

  records: IEntity[] = $state([])
  recordsMap: Map<number, IEntity> = $state(new Map())

  constructor(entityType: number) {
    super()
    this.route = `entities?type=${entityType}`
    this.fetch()
  }

  handler(result: IEntity[]): void {
    const fetchedEntities = (result || []).filter(entityRecord => (entityRecord.ss || 0) > 0)
    this.records = fetchedEntities
    this.recordsMap = new Map(fetchedEntities.map(entityRecord => [entityRecord.ID, entityRecord]))
  }
}
