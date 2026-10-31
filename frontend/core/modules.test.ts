import { describe, expect, it } from 'bun:test'
import { existsSync, readFileSync } from 'node:fs'
import path from 'node:path'
import { findModuleOfPath, getAccessibleModuleEntries, getMenuModule, isSaaSOnlyRoute, parseModuleCatalogs } from './modules'

const repoRoot = path.resolve(import.meta.dir, '../..')
const readCatalog = (fileName: string) => readFileSync(path.join(repoRoot, 'db', fileName), 'utf8')

describe('db/pages.toml and db/modules.toml', () => {
  // Parsing throws on a module route missing from pages.toml.
  const modules = parseModuleCatalogs(readCatalog('pages.toml'), readCatalog('modules.toml'))
  const listedPages = modules.flatMap((module) => module.menus.flatMap((menuGroup) => menuGroup.pages))

  it('lists every declared page in at least one module', () => {
    const declaredRoutes = readCatalog('pages.toml').match(/^route = "([^"]+)"$/gm)!
      .map((routeLine) => `/${routeLine.slice(9, -1)}`)
    const listedRoutes = new Set(listedPages.map((listedPage) => listedPage.route))
    expect(declaredRoutes.filter((route) => !listedRoutes.has(route))).toEqual([])
  })

  it('points every built page at a real route folder, and no pending page at one', () => {
    const routeHasPage = (route: string) => existsSync(path.join(repoRoot, 'frontend/routes', route, '+page.svelte'))
    expect(listedPages.filter((listedPage) => !listedPage.pending && !routeHasPage(listedPage.route)).map((listedPage) => listedPage.route)).toEqual([])
    expect(listedPages.filter((listedPage) => listedPage.pending && routeHasPage(listedPage.route)).map((listedPage) => listedPage.route)).toEqual([])
  })

  // The plugin silently renders nothing for a misspelled class or an icon its set lacks.
  it('uses only icons that exist in the installed Iconify sets', () => {
    const iconClasses = [...modules.map((module) => module.icon), ...listedPages.map((listedPage) => listedPage.icon || '')].filter(Boolean)
    const missingIcons = iconClasses.filter((iconClass) => {
      const iconMatch = /^icon-\[([a-z0-9-]+?)--([a-z0-9-]+)\]$/.exec(iconClass)
      if (!iconMatch) { return true }
      const iconSetPath = path.join(repoRoot, 'frontend/node_modules/@iconify-json', iconMatch[1], 'icons.json')
      if (!existsSync(iconSetPath)) { return true }
      const iconSet = JSON.parse(readFileSync(iconSetPath, 'utf8'))
      return !iconSet.icons[iconMatch[2]] && !iconSet.aliases?.[iconMatch[2]]
    })
    expect(missingIcons).toEqual([])
  })

  it('keeps module codes unique', () => {
    const moduleCodes = modules.map((module) => module.code)
    expect(new Set(moduleCodes).size).toBe(moduleCodes.length)
  })
})

describe('module navigation rules', () => {
  const pagesToml = `
[[pages]]
route = "production/products"
name = "Products|Productos"

[[pages]]
route = "sales/sale_order_create"
name = "POS|Punto de Venta"

[[pages]]
route = "finance/cash-flow"
name = "Cash Flow|Flujo de Caja"
pending = true

[[pages]]
route = "system/companies"
name = "Companies|Empresas"
`
  const modulesToml = `
[[modules]]
code = "commercial"
name = "Commercial|Comercial"
icon = "icon-[fa--shopping-cart]"

[[modules.menus]]
name = "Sales|Ventas"
routes = "sales/sale_order_create, production/products"

[[modules]]
code = "operations"
name = "Operations|Operaciones"
icon = "icon-[fa--cubes]"

[[modules.menus]]
name = "Production|Producción"
routes = "production/products, finance/cash-flow"

[[modules]]
code = "system"
name = "System|System"
icon = "icon-[fa--cogs]"
saas_only = true

[[modules.menus]]
name = "Platform|Plataforma"
routes = "system/companies"
`
  const modules = parseModuleCatalogs(pagesToml, modulesToml)

  it('attaches each [[modules.menus]] to the module above it', () => {
    expect(modules.map((module) => module.menus.map((menuGroup) => menuGroup.pages.map((menuPage) => menuPage.route))))
      .toEqual([[['/sales/sale_order_create', '/production/products']], [['/production/products', '/finance/cash-flow']], [['/system/companies']]])
  })

  it('keeps a shared page in the module the user came from', () => {
    expect(findModuleOfPath(modules, '/production/products', 'operations')?.code).toBe('operations')
    expect(findModuleOfPath(modules, '/production/products', 'commercial')?.code).toBe('commercial')
    // Coming from a module that does not list it: the first module listing it.
    expect(findModuleOfPath(modules, '/production/products', 'system')?.code).toBe('commercial')
  })

  it('shows the last module on a page no module lists', () => {
    expect(findModuleOfPath(modules, '/', 'operations')).toBeUndefined()
    expect(getMenuModule(modules, '/', 'operations')?.code).toBe('operations')
    expect(getMenuModule(modules, '/', '')?.code).toBe('commercial')
  })

  it('opens a module on its last page, never on a pending one', () => {
    const navigationMemory = { lastModuleCode: '', lastRouteByModule: { commercial: '/production/products' } }
    const entries = getAccessibleModuleEntries(modules, navigationMemory, () => true)
    expect(entries.map((entry) => [entry.module.code, entry.entryRoute])).toEqual([
      ['commercial', '/production/products'],
      ['operations', '/production/products'],
      ['system', '/system/companies'],
    ])
    // A module whose only openable page is pending cannot be entered.
    const withoutProducts = getAccessibleModuleEntries(modules, navigationMemory, (route) => route !== '/production/products')
    expect(withoutProducts.map((entry) => entry.module.code)).toEqual(['commercial', 'system'])
  })

  it('marks only the saas_only modules routes as SaaS-only', () => {
    expect(isSaaSOnlyRoute(modules, '/system/companies')).toBe(true)
    expect(isSaaSOnlyRoute(modules, '/production/products')).toBe(false)
  })

  it('rejects a module route pages.toml does not declare', () => {
    expect(() => parseModuleCatalogs(pagesToml, '[[modules]]\ncode = "x"\n[[modules.menus]]\nroutes = "missing/page"\n'))
      .toThrow(/does not declare/)
  })
})
