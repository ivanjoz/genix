// The module registry, read from db/pages.toml and db/modules.toml, and the module navigation of
// the logged-in session: which modules it can enter and which one the side menu shows.

import { browser } from '$app/env';
import { Env } from '#core/env.ts';
import { security } from '#libs/ui-runtime.svelte.ts';
import pagesTomlContent from '../../db/pages.toml?raw';
import modulesTomlContent from '../../db/modules.toml?raw';
import {
  SAAS_COMPANY_ID, findModuleOfPath, getAccessibleModuleEntries, getMenuModule, isSaaSOnlyRoute, parseModuleCatalogs,
  type IModuleNavigationMemory,
} from './modules';

export const Modules = parseModuleCatalogs(pagesTomlContent, modulesTomlContent)

const MODULE_NAVIGATION_STORAGE_KEY = 'genix_module_navigation'

const getStoredModuleNavigation = (): IModuleNavigationMemory => {
  const storedNavigation = browser ? localStorage.getItem(MODULE_NAVIGATION_STORAGE_KEY) : null
  return storedNavigation ? JSON.parse(storedNavigation) : { lastModuleCode: '', lastRouteByModule: {} }
}

export const ModuleNavigation = $state<IModuleNavigationMemory>(getStoredModuleNavigation())

export const isSaaSRouteDenied = (route: string) => isSaaSOnlyRoute(Modules, route) && Env.getCompanyID() !== SAAS_COMPANY_ID

// A route the session can open: its access (backend/access.toml) and, for SYSTEM, the SaaS company.
export const canOpenRoute = (route: string) => !isSaaSRouteDenied(route) && security.canAccessRoute(route)

export const getSessionModuleEntries = () => getAccessibleModuleEntries(Modules, ModuleNavigation, canOpenRoute)

export const getSessionMenuModule = (pathname: string) =>
  getMenuModule(getSessionModuleEntries().map((moduleEntry) => moduleEntry.module), pathname, ModuleNavigation.lastModuleCode)

// Called with every page the session opens: it becomes the remembered page of its module, and that
// module the one a shared page and "/" stay in.
export const rememberModulePage = (pathname: string) => {
  const pageModule = findModuleOfPath(Modules, pathname, ModuleNavigation.lastModuleCode)
  if (!pageModule) { return }
  if (ModuleNavigation.lastModuleCode === pageModule.code && ModuleNavigation.lastRouteByModule[pageModule.code] === pathname) { return }
  ModuleNavigation.lastModuleCode = pageModule.code
  ModuleNavigation.lastRouteByModule[pageModule.code] = pathname
  localStorage.setItem(MODULE_NAVIGATION_STORAGE_KEY, JSON.stringify(ModuleNavigation))
}
