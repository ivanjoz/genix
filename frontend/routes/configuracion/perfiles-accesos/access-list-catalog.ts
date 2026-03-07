import { accessListUrl, accessListHash } from '$core/generated/access-list';

export interface IAccessListCatalogEntry {
  id: number
  name: string
  levels: number
  frontend_routes: string
  backend_apis: string
}

export interface IAccessListCatalogPayload {
  access_list: IAccessListCatalogEntry[]
}

export async function fetchAccessListCatalog(): Promise<IAccessListCatalogPayload> {
  // The hash is emitted at build time so CDN clients can cache forever and refresh on content changes.
  console.info('[access-list] Fetching access catalog', { accessListHash, accessListUrl })

  const accessListResponse = await fetch(accessListUrl, {
    headers: {
      'cache-control': 'no-cache'
    }
  })

  if (!accessListResponse.ok) {
    console.error('[access-list] Failed to fetch access catalog', {
      status: accessListResponse.status,
      accessListUrl
    })
    throw new Error(`No se pudo cargar el catálogo de accesos: ${accessListResponse.status}`)
  }

  return accessListResponse.json() as Promise<IAccessListCatalogPayload>
}
