import { describe, expect, it } from 'bun:test'
import { normalizeAccessFrontendRoutes, parseAccessCatalogForTest } from './access-list-catalog'

describe('access-list catalog routes', () => {
  it('expands a comma-separated frontend route list', () => {
    // One access can intentionally unlock closely related pages.
    expect(normalizeAccessFrontendRoutes('system/server-panel, /system/observability')).toEqual([
      'system/server-panel',
      'system/observability',
    ])
  })
})

describe('access catalog TOML parser', () => {
  it('parses groups, accesses and single-line arrays', () => {
    const parsed = parseAccessCatalogForTest(`
# a comment
[[groups]]
id = 1
name = "Mi Empresa"

[[access]]
id = 10
name = "Punto de Venta"
group = 3
levels = 14
frontend_routes = "sales/sale_order_create"
backend_apis = "POST.sale-order"
sub_accesses_ids = [2, 3]
sub_accesses_names = ["Anular venta", "Aplicar descuento"]
`)

    expect(parsed.groups).toEqual([{ id: 1, name: 'Mi Empresa' }] as never)
    expect(parsed.access).toHaveLength(1)
    expect(parsed.access[0]).toMatchObject({ id: 10, name: 'Punto de Venta', levels: 14 })
    expect((parsed.access[0] as never as Record<string, unknown>).sub_accesses_ids).toEqual([2, 3])
    expect((parsed.access[0] as never as Record<string, unknown>).sub_accesses_names)
      .toEqual(['Anular venta', 'Aplicar descuento'])
  })

  // Both are legal TOML and neither survives JSON.parse, so they are rejected with a message that
  // names the problem instead of surfacing a SyntaxError from three frames down.
  it('rejects a multi-line array', () => {
    expect(() => parseAccessCatalogForTest('[[access]]\nsub_accesses_ids = [\n  2,\n]\n'))
      .toThrow(/must fit on one line/)
  })

  it('rejects a trailing comma', () => {
    expect(() => parseAccessCatalogForTest('[[access]]\nsub_accesses_ids = [2, 3, ]\n'))
      .toThrow(/trailing comma/)
  })

  it('rejects an unknown section and a field with no record', () => {
    expect(() => parseAccessCatalogForTest('[[modules]]\nid = 1\n')).toThrow(/Unsupported access-catalog section/)
    expect(() => parseAccessCatalogForTest('id = 1\n')).toThrow(/before a \[\[section\]\]/)
  })

  it('rejects an unquoted scalar', () => {
    expect(() => parseAccessCatalogForTest('[[access]]\nname = Punto de Venta\n'))
      .toThrow(/integers, quoted strings or arrays/)
  })
})
