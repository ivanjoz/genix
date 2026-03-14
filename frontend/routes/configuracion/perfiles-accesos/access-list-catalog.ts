import { parse as parseYaml } from 'yaml';
import accessListYamlContent from '../../../../backend/access_list.yml?raw';

export interface IAccessGroupCatalogEntry {
  id: number
  name: string
}

export interface IAccessListCatalogEntry {
  id: number
  name: string
  group: number
  levels: number
  frontend_routes: string
  backend_apis: string
}

export interface IAccessListCatalogPayload {
  access_groups: IAccessGroupCatalogEntry[]
  access_list: IAccessListCatalogEntry[]
}

// Parse the backend-owned YAML once so every screen reads the same catalog without a generated module.
const accessListCatalogPayload = parseYaml(accessListYamlContent) as IAccessListCatalogPayload

export async function fetchAccessListCatalog(): Promise<IAccessListCatalogPayload> {
  console.info('[access-list] Loaded access catalog from backend/access_list.yml', {
    accessGroupCount: accessListCatalogPayload?.access_groups?.length || 0,
    accessEntryCount: accessListCatalogPayload?.access_list?.length || 0
  })

  return accessListCatalogPayload
}
