// Modules are virtual: db/pages.toml declares every page once, and db/modules.toml arranges those
// pages into modules and menu groups, regardless of the route folders. A page may be listed in
// several modules. Pure rules only, no Svelte and no session: core/modules.svelte.ts loads the two
// files and binds these rules to the logged-in session.

import { parseTomlTables, splitCatalogRoutes } from '#libs/toml-tables.utils.ts'

export interface IPage {
  // With the leading "/": the TOML omits it, the router needs it.
  route: string
  name: string
  icon?: string
  description?: string
  // Planned but not built: listed disabled, never a module's entry page.
  pending?: boolean
}

export interface IMenuGroup {
  // Position in its module, from 1. Only the side menu's open/closed state uses it.
  id: number
  name: string
  minName?: string
  pages: IPage[]
}

export interface IModule {
  code: string
  name: string
  icon: string
  saasOnly: boolean
  menus: IMenuGroup[]
}

export interface IModuleNavigationMemory {
  // Module of the last page visited: a shared page stays in it, and "/" keeps showing its menu.
  lastModuleCode: string
  // Last page visited in each module, by module code: going A → B → A lands back on A's page.
  lastRouteByModule: Record<string, string>
}

export interface IModuleEntry {
  module: IModule
  entryRoute: string
}

// The SYSTEM module administers the whole platform (companies, server, crons), not a tenant: only
// the company that owns the SaaS sees it. The backend repeats the restriction per endpoint.
export const SAAS_COMPANY_ID = 1

// Builds the modules from the two catalogs. A module route missing from pages.toml throws: the menu
// would otherwise drop it silently.
export function parseModuleCatalogs(pagesTomlContent: string, modulesTomlContent: string): IModule[] {
  const pageRecords = parseTomlTables(pagesTomlContent, ['pages'], 'pages-catalog').pages
  const pagesByRoute = new Map<string, IPage>()
  for (const pageRecord of pageRecords) {
    const route = `/${splitCatalogRoutes(pageRecord.route as string)[0] || ''}`
    if (pagesByRoute.has(route)) { throw new Error(`pages.toml declares "${route}" twice`) }
    pagesByRoute.set(route, {
      route,
      name: pageRecord.name as string,
      icon: pageRecord.icon as string | undefined,
      description: pageRecord.description as string | undefined,
      pending: pageRecord.pending === true,
    })
  }

  const moduleRecords = parseTomlTables(modulesTomlContent, ['modules', 'modules.menus'], 'modules-catalog').modules
  return moduleRecords.map((moduleRecord) => ({
    code: moduleRecord.code as string,
    name: moduleRecord.name as string,
    icon: moduleRecord.icon as string,
    saasOnly: moduleRecord.saas_only === true,
    menus: ((moduleRecord.menus || []) as Record<string, unknown>[]).map((menuRecord, menuIndex) => ({
      id: menuIndex + 1,
      name: menuRecord.name as string,
      minName: menuRecord.min_name as string | undefined,
      pages: splitCatalogRoutes(menuRecord.routes as string).map((route) => {
        const menuPage = pagesByRoute.get(`/${route}`)
        if (!menuPage) { throw new Error(`modules.toml: module "${moduleRecord.code}" lists "${route}", which pages.toml does not declare`) }
        return menuPage
      }),
    })),
  }))
}

const isPathUnder = (pathname: string, basePath: string) => pathname === basePath || pathname.startsWith(`${basePath}/`)

const listModulePages = (module: IModule) => module.menus.flatMap((menuGroup) => menuGroup.pages)

// Routes of the saas_only modules: blocked for every other company, even through a typed URL.
export function isSaaSOnlyRoute(modules: IModule[], pathname: string): boolean {
  return modules.some((module) => module.saasOnly && listModulePages(module).some((modulePage) => isPathUnder(pathname, modulePage.route)))
}

// A module opens on the page last visited in it while that page is still one of its built pages and
// the session can open it; otherwise on its first such page. Undefined when there is none: the
// session cannot enter the module.
export function getModuleEntryRoute(
  module: IModule, navigationMemory: IModuleNavigationMemory, canOpenRoute: (route: string) => boolean,
): string | undefined {
  const openableRoutes = listModulePages(module)
    .filter((modulePage) => !modulePage.pending && canOpenRoute(modulePage.route))
    .map((modulePage) => modulePage.route)
  const lastRoute = navigationMemory.lastRouteByModule[module.code]
  return openableRoutes.includes(lastRoute) ? lastRoute : openableRoutes[0]
}

// The modules a session can enter: those with at least one page it can open.
export function getAccessibleModuleEntries(
  modules: IModule[], navigationMemory: IModuleNavigationMemory, canOpenRoute: (route: string) => boolean,
): IModuleEntry[] {
  return modules
    .map((module) => ({ module, entryRoute: getModuleEntryRoute(module, navigationMemory, canOpenRoute) }))
    .filter((moduleEntry): moduleEntry is IModuleEntry => !!moduleEntry.entryRoute)
}

// The module a page belongs to. A page shared by several modules stays in the one the user came
// from (lastModuleCode) when that one lists it; otherwise it goes to the first module listing it.
// Undefined for a page no module lists (e.g. "/").
export function findModuleOfPath(
  modules: IModule[], pathname: string, lastModuleCode: string,
): IModule | undefined {
  const listingModules = modules.filter((module) => listModulePages(module).some((modulePage) => isPathUnder(pathname, modulePage.route)))
  return listingModules.find((module) => module.code === lastModuleCode) || listingModules[0]
}

// The module whose menu the side menu shows: the page's module; on a page no module lists, the
// last module visited, else the first one.
export function getMenuModule(
  modules: IModule[], pathname: string, lastModuleCode: string,
): IModule | undefined {
  return findModuleOfPath(modules, pathname, lastModuleCode)
    || modules.find((module) => module.code === lastModuleCode)
    || modules[0]
}
