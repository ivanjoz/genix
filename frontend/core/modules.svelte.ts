// The module registry, read from db/pages.toml and db/modules.toml, and the module navigation of
// the logged-in session: which modules it can enter and which one the side menu shows.

import { browser } from '$app/env';
import { Env, moduleNavigationStorageKey } from '#core/env.ts';
import { security } from '#libs/ui-runtime.svelte.ts';
import pagesTomlContent from '../../db/pages.toml?raw';
import modulesTomlContent from '../../db/modules.toml?raw';
import {
  SAAS_COMPANY_ID, findModuleOfPath, getAccessibleModuleEntries, getMenuModule, isSaaSOnlyRoute, parseModuleCatalogs,
  type IModuleNavigationMemory,
} from './modules';

export const Modules = parseModuleCatalogs(pagesTomlContent, modulesTomlContent)

const getStoredModuleNavigation = (): IModuleNavigationMemory => {
  const storedNavigation = browser ? localStorage.getItem(moduleNavigationStorageKey) : null
  return storedNavigation ? JSON.parse(storedNavigation) : { lastModuleCode: '', lastRouteByModule: {} }
}

export const ModuleNavigation = $state<IModuleNavigationMemory>(getStoredModuleNavigation())

export const isSaaSRouteDenied = (route: string) => isSaaSOnlyRoute(Modules, route) && Env.getCompanyID() !== SAAS_COMPANY_ID

// A route the session can open: its access (backend/access.toml) and, for SYSTEM, the SaaS company.
export const canOpenRoute = (route: string) => !isSaaSRouteDenied(route) && security.canAccessRoute(route)

export const getSessionModuleEntries = () => getAccessibleModuleEntries(Modules, ModuleNavigation, canOpenRoute)

export const getSessionMenuModule = (pathname: string) =>
  getMenuModule(getSessionModuleEntries().map((moduleEntry) => moduleEntry.module), pathname, ModuleNavigation.lastModuleCode)

// The module the user picked in the selector. A page shared by several modules (crm/customers is in
// COMMERCIAL and CRM) stays in the last module, so picking the other one while standing on that page
// must switch the menu itself: the navigation target is the page already open and does nothing.
export const rememberSelectedModule = (moduleCode: string) => {
  ModuleNavigation.lastModuleCode = moduleCode
  localStorage.setItem(moduleNavigationStorageKey, JSON.stringify(ModuleNavigation))
}

// Called with every page the session opens: it becomes the remembered page of its module, and that
// module the one a shared page and "/" stay in.
export const rememberModulePage = (pathname: string) => {
  const pageModule = findModuleOfPath(Modules, pathname, ModuleNavigation.lastModuleCode)
  if (!pageModule) { return }
  if (ModuleNavigation.lastModuleCode === pageModule.code && ModuleNavigation.lastRouteByModule[pageModule.code] === pathname) { return }
  ModuleNavigation.lastModuleCode = pageModule.code
  ModuleNavigation.lastRouteByModule[pageModule.code] = pathname
  localStorage.setItem(moduleNavigationStorageKey, JSON.stringify(ModuleNavigation))
}
