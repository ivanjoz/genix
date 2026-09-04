import { describe, expect, it } from 'bun:test'
import {
  buildSubAccesoOptions,
  packSubAccesos,
  SUB_ACCESO_TODOS_ID,
  toggleSubAcceso,
  unpackSubAccesos,
} from './users-profiles'

describe('buildSubAccesoOptions', () => {
  it('puts the synthetic Todos ahead of what the catalog declares', () => {
    expect(buildSubAccesoOptions([2, 3], ["Recibir Pago", "Despachar Producto"])).toEqual([
      { id: SUB_ACCESO_TODOS_ID, name: "Todos" },
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
    expect(toggleSubAcceso([], 3)).toEqual([3])
    expect(toggleSubAcceso([2, 3], 3)).toEqual([2])
  })

  it('keeps the selection sorted', () => {
    expect(toggleSubAcceso([3], 2)).toEqual([2, 3])
  })

  // Todos satisfies every check on its access, so holding it alongside individual sub-accesses is
  // contradictory. The exclusivity runs in both directions.
  it('clears everything else when Todos is selected', () => {
    expect(toggleSubAcceso([2, 3], SUB_ACCESO_TODOS_ID)).toEqual([SUB_ACCESO_TODOS_ID])
  })

  it('clears Todos when an individual sub-access is selected', () => {
    expect(toggleSubAcceso([SUB_ACCESO_TODOS_ID], 2)).toEqual([2])
  })

  it('deselects Todos without selecting anything in its place', () => {
    expect(toggleSubAcceso([SUB_ACCESO_TODOS_ID], SUB_ACCESO_TODOS_ID)).toEqual([])
  })
})

describe('the wire shape', () => {
  it('packs to accesoID * 100 + subAccesoID, sorted', () => {
    const subAccesosMap = new Map([[10, [3, 2]], [7, [SUB_ACCESO_TODOS_ID]]])
    const accesosMap = new Map([[10, [4]], [7, [1]]])

    expect(packSubAccesos(subAccesosMap, accesosMap)).toEqual([701, 1002, 1003])
  })

  // The backend rejects the whole profile over a sub-access with no parent access, so a leftover of
  // the operator's own editing — ticking sub-accesses, then clearing the access — is dropped here
  // rather than turned into an error they cannot act on.
  it('drops a sub-access whose access is no longer granted', () => {
    const subAccesosMap = new Map([[10, [2]], [7, [SUB_ACCESO_TODOS_ID]]])
    const accesosMap = new Map([[10, [4]], [7, []]])

    expect(packSubAccesos(subAccesosMap, accesosMap)).toEqual([1002])
  })

  it('round-trips through the unpacker', () => {
    expect([...unpackSubAccesos([701, 1002, 1003])]).toEqual([[7, [1]], [10, [2, 3]]])
    expect([...unpackSubAccesos(undefined)]).toEqual([])
  })
})
