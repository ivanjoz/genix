import { describe, expect, it } from 'bun:test'
import type { IAccessListCatalogEntry } from './access-list-catalog'
import {
  buildAccessGroupBars,
  buildAccessUserRows,
  buildEffectiveNivelByAcceso,
  buildSubAccesoOptions,
  decodeAccessLevels,
  fromAccesoGrants,
  isSubAccesoChecked,
  SUB_ACCESO_TODOS_ID,
  toAccesoGrants,
  toggleSubAcceso,
} from './users-profiles'

const PUNTO_DE_VENTA_SUB_ACCESOS = [2, 3]

describe('buildSubAccesoOptions', () => {
  // Todos is not a checkbox: it is what ticking every box is stored as.
  it('offers exactly what the catalog declares', () => {
    expect(buildSubAccesoOptions([2, 3], ["Recibir Pago", "Despachar Producto"])).toEqual([
      { id: 2, name: "Recibir Pago" },
      { id: 3, name: "Despachar Producto" },
    ])
  })

  it('offers nothing for an access that declares no sub-access', () => {
    expect(buildSubAccesoOptions(undefined, undefined)).toEqual([])
    expect(buildSubAccesoOptions([], [])).toEqual([])
  })

  // Naming the wrong sub-access is how an operator grants the wrong permission, so a pair that
  // drifted out of step shows nothing rather than a shifted list.
  it('offers nothing when the parallel arrays disagree in length', () => {
    expect(buildSubAccesoOptions([2, 3], ["Recibir Pago"])).toEqual([])
  })
})

describe('toggleSubAcceso', () => {
  it('adds and removes an individual sub-access', () => {
    expect(toggleSubAcceso([], 3, [2, 3, 4])).toEqual([3])
    expect(toggleSubAcceso([2, 3], 3, [2, 3, 4])).toEqual([2])
  })

  it('keeps the selection sorted', () => {
    expect(toggleSubAcceso([4], 2, [2, 3, 4])).toEqual([2, 4])
  })

  // "Todos" is implicit: it is never ticked, it is what a complete selection collapses to.
  it('stores Todos once the last declared sub-access is ticked', () => {
    expect(toggleSubAcceso([2], 3, PUNTO_DE_VENTA_SUB_ACCESOS)).toEqual([SUB_ACCESO_TODOS_ID])
  })

  it('expands a stored Todos before unticking one of its boxes', () => {
    expect(toggleSubAcceso([SUB_ACCESO_TODOS_ID], 2, PUNTO_DE_VENTA_SUB_ACCESOS)).toEqual([3])
  })

  // An access declaring a single sub-access has ticking it and granting Todos mean the same thing.
  it('collapses a single-sub-access selection to Todos too', () => {
    expect(toggleSubAcceso([], 2, [2])).toEqual([SUB_ACCESO_TODOS_ID])
    expect(toggleSubAcceso([SUB_ACCESO_TODOS_ID], 2, [2])).toEqual([])
  })
})

describe('isSubAccesoChecked', () => {
  it('ticks every declared box while Todos is stored', () => {
    expect(isSubAccesoChecked([SUB_ACCESO_TODOS_ID], 2)).toBe(true)
    expect(isSubAccesoChecked([SUB_ACCESO_TODOS_ID], 3)).toBe(true)
  })

  it('ticks only what is listed otherwise', () => {
    expect(isSubAccesoChecked([3], 3)).toBe(true)
    expect(isSubAccesoChecked([3], 2)).toBe(false)
    expect(isSubAccesoChecked([], 2)).toBe(false)
  })
})

describe('buildEffectiveNivelByAcceso', () => {
  it('keeps the widest level any source grants', () => {
    const profileAccesosMaps = [new Map([[10, 1], [7, 4]]), new Map([[10, 4]])]
    const ownGrants = [{ AccesoID: 7, Nivel: 1 }, { AccesoID: 12, Nivel: 4 }]

    const effectiveNivelEntries = [...buildEffectiveNivelByAcceso(profileAccesosMaps, ownGrants)]
      .sort(([leftAccesoID], [rightAccesoID]) => leftAccesoID - rightAccesoID)

    expect(effectiveNivelEntries).toEqual([
      [7, 4], [10, 4], [12, 4],
    ])
  })

  it('ignores a grant that grants nothing', () => {
    expect([...buildEffectiveNivelByAcceso([], [{ AccesoID: 0, Nivel: 4 }, { AccesoID: 9, Nivel: 0 }])])
      .toEqual([])
  })
})

describe('buildAccessGroupBars', () => {
  const accessGroups = [{ id: 3, name: "Comercial" }, { id: 5, name: "Finanzas" }]
  // Four accesses in Comercial, one in Finanzas — the bar's denominator is the catalog, not the grant.
  const accessCatalog = [
    { id: 1, name: "Gestión Ventas", group: 3 },
    { id: 2, name: "Reporte Ventas", group: 3 },
    { id: 3, name: "Punto de Venta", group: 3 },
    { id: 4, name: "Clientes", group: 3 },
    { id: 5, name: "Flujo de Caja", group: 5 },
  ] as IAccessListCatalogEntry[]

  it('measures each level against the group total, widest level first', () => {
    const bars = buildAccessGroupBars(new Map([[1, 4], [2, 1], [3, 1]]), accessCatalog, accessGroups)

    expect(bars).toEqual([{
      groupID: 3,
      groupName: "Comercial",
      totalAccessCount: 4,
      segments: [{ nivel: 4, accessCount: 1 }, { nivel: 1, accessCount: 2 }],
    }])
  })

  it('leaves out a group the user holds nothing in', () => {
    const bars = buildAccessGroupBars(new Map([[5, 1]]), accessCatalog, accessGroups)
    expect(bars.map(bar => bar.groupName)).toEqual(["Finanzas"])
  })

  it('has nothing to draw for a user with no accesses', () => {
    expect(buildAccessGroupBars(new Map(), accessCatalog, accessGroups)).toEqual([])
  })
})

describe('buildAccessUserRows', () => {
  const accessGroups = [{ id: 3, name: "Comercial" }, { id: 5, name: "Finanzas" }]
  const accessCatalog = [
    { id: 9, name: "Flujo de Caja", group: 5, levels: 14 },
    { id: 1, name: "Gestión Ventas", group: 3, levels: 14 },
    { id: 2, name: "Reporte Ventas", group: 3, levels: 1 },
  ] as IAccessListCatalogEntry[]

  const usuarios = [
    { usuarioID: 11, usuarioName: "Ana Ruiz", nivelByAccesoID: new Map([[1, 4], [2, 1]]) },
    { usuarioID: 12, usuarioName: "Beto Paz", nivelByAccesoID: new Map([[1, 1], [9, 4]]) },
    { usuarioID: 13, usuarioName: "Cora Gil", nivelByAccesoID: new Map([[1, 4]]) },
  ]

  // Gestión Ventas hands TODO to two people, Flujo de Caja to one, Reporte Ventas to nobody at a
  // write level — and each access keeps its own levels together, widest first.
  it('ranks the accesses by how many users hold them at a write level', () => {
    const rows = buildAccessUserRows(usuarios, accessCatalog, accessGroups)

    expect(rows.map(row => [row.accesoName, row.nivel])).toEqual([
      ["Gestión Ventas", 4], ["Gestión Ventas", 1],
      ["Flujo de Caja", 4],
      ["Reporte Ventas", 1],
    ])
  })

  // Exactly at that level, not "at least": a user granted TODO is on the TODO row only.
  it('files each user under the single level they effectively hold', () => {
    const rows = buildAccessUserRows(usuarios, accessCatalog, accessGroups)

    expect(rows.filter(row => row.accesoID === 1).map(row => row.usuarios)).toEqual([
      [{ usuarioID: 11, usuarioName: "Ana Ruiz" }, { usuarioID: 13, usuarioName: "Cora Gil" }],
      [{ usuarioID: 12, usuarioName: "Beto Paz" }],
    ])
  })

  it('leaves out every level nobody holds, and an access nobody holds at all', () => {
    const rows = buildAccessUserRows(
      [{ usuarioID: 11, usuarioName: "Ana Ruiz", nivelByAccesoID: new Map([[1, 1]]) }],
      accessCatalog, accessGroups)

    expect(rows.map(row => [row.accesoName, row.nivel])).toEqual([["Gestión Ventas", 1]])
  })
})

describe('the stored grant shape', () => {
  it('nests each access sub-accesses inside its grant, sorted by access', () => {
    const accesosMap = new Map([[10, 4], [7, 1]])
    const subAccesosMap = new Map([[10, [3, 2]], [7, [SUB_ACCESO_TODOS_ID]]])

    expect(toAccesoGrants(accesosMap, subAccesosMap)).toEqual([
      { AccesoID: 7, Nivel: 1, SubAccesos: [SUB_ACCESO_TODOS_ID] },
      { AccesoID: 10, Nivel: 4, SubAccesos: [2, 3] },
    ])
  })

  it('leaves SubAccesos off an access that has none', () => {
    expect(toAccesoGrants(new Map([[10, 4]]))).toEqual([{ AccesoID: 10, Nivel: 4 }])
  })

  // The backend rejects the whole record over a sub-access with no parent access, so a leftover of
  // the operator's own editing — ticking sub-accesses, then clearing the access — is dropped here
  // rather than turned into an error they cannot act on.
  it('drops a sub-access whose access is no longer granted', () => {
    const accesosMap = new Map([[10, 4]])
    const subAccesosMap = new Map([[10, [2]], [7, [SUB_ACCESO_TODOS_ID]]])

    expect(toAccesoGrants(accesosMap, subAccesosMap)).toEqual([
      { AccesoID: 10, Nivel: 4, SubAccesos: [2] },
    ])
  })

  it('round-trips back into the two editable maps', () => {
    const grants = toAccesoGrants(new Map([[10, 4], [7, 1]]), new Map([[10, [2, 3]]]))
    const { accesosMap, subAccesosMap } = fromAccesoGrants(grants)

    expect([...accesosMap]).toEqual([[7, 1], [10, 4]])
    expect([...subAccesosMap]).toEqual([[10, [2, 3]]])
  })

  it('ignores a grant with no access or no level, which grants nothing', () => {
    const { accesosMap } = fromAccesoGrants([
      { AccesoID: 0, Nivel: 4 },
      { AccesoID: 10, Nivel: 0 },
    ])
    expect([...accesosMap]).toEqual([])
    expect([...fromAccesoGrants(undefined).accesosMap]).toEqual([])
  })
})

describe('decodeAccessLevels', () => {
  it('expands the catalog digit mask into individual levels', () => {
    expect(decodeAccessLevels(14)).toEqual([1, 4])
    expect(decodeAccessLevels(1)).toEqual([1])
  })

  it('drops the zero digit, which stands for no level', () => {
    expect(decodeAccessLevels(104)).toEqual([1, 4])
  })
})
