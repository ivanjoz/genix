import { GetHandler, POST } from '$libs/http.svelte'

export const ClientProviderType = {
  CLIENT: 1,
  PROVIDER: 2,
} as const

export const PersonType = {
  PERSON: 1,
  COMPANY: 2,
} as const

export interface IClientProvider {
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

export const postClientProviders = (clientProvidersPayload: IClientProvider[]) => {
  return POST({
    data: clientProvidersPayload,
    route: 'client-provider',
    // Keep both client-provider caches aligned because the shared save flow serves both routes.
    refreshRoutes: ['client-provider?type=1', 'client-provider?type=2'],
  })
}

export class ClientProviderService extends GetHandler<IClientProvider> {
  route = ''
  routeByID = "client-provider-ids"
  keyID = 'ID'
  useCache = { min: 5, ver: 1 }

  records: IClientProvider[] = $state([])
  recordsMap: Map<number, IClientProvider> = $state(new Map())

  constructor(clientProviderType: number = 0, init: boolean = false) {
    super()
    this.route = `client-provider?type=${clientProviderType}`
    if (init) {
      this.fetch()
    }
  }

  handler(result: IClientProvider[]): void {
    // Filter deleted rows from delta cache responses so both pages render only active records.
    const fetchedClientProviders = (result || []).filter(clientProviderRecord => (clientProviderRecord.ss || 0) > 0)
    this.records = fetchedClientProviders
    this.recordsMap = new Map(fetchedClientProviders.map(clientProviderRecord => [clientProviderRecord.ID, clientProviderRecord]))
  }
}
