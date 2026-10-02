import { getRecordsByID } from '@genix/ui/cache'
import { GetHandler, POST } from '#libs/ui-runtime.svelte.ts'

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
  /** An IDENTITY_DOC_OPTIONS id. 0 leaves it to the backend, which derives it from the
   *  registry number's shape. */
  IdentityDocType: number
  /** The frozen identity this row currently presents, in client_provider_snapshot.
   *  Documents copy it when issued; read it, never write it. */
  SnapshotID: number
  PersonType: number
  Email: string
  CountryID: number
  CityID: string
  ss: number
  upd: number
  UpdatedBy?: number
}

/** An identity as some document froze it. Immutable, so it is safe to cache forever. */
export interface IClientProviderSnapshot {
  ID: number
  Name: string
  RegistryNumber: string
  IdentityDocType: number
  CityID: string
  upv: number
  /** The table has no status column — nothing deletes a snapshot — so the backend sends none
   *  and the by-ids cache defaults it to 1. Declared because IMinimalRecord requires it. */
  ss: number
  upd: number
}

export const CLIENT_PROVIDER_SNAPSHOT_ROUTE = 'client-provider-snapshot-ids'

/** Resolves the frozen identities a set of documents pinned.
 *
 *  Through the by-ids cache rather than a join in the handler that produced the ids: a snapshot
 *  never changes, so a browser that has one keeps it, and the accounting books ask for the same
 *  few hundred identities every period. See packages/genix-ui/cache/CACHE_BY_IDS.md. */
export const getClientProviderSnapshots = (
  snapshotIDs: number[],
): Promise<Map<number, IClientProviderSnapshot>> =>
  getRecordsByID<IClientProviderSnapshot>(CLIENT_PROVIDER_SNAPSHOT_ROUTE, snapshotIDs)

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
