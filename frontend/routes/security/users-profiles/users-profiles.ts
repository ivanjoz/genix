// Pure access-granting rules. No Svelte, no fetch: the whole point is that the "Todos" rule and the
// conversion to the stored shape can be tested without a component or a network call.
//
// What is stored is a list of `IAccesoGrant` — one record per access, carrying its nivel and its
// sub-accesses — the identical shape a profile and a user both save. The binary grant blobs are
// packed exactly once, in the backend, when a user's effective grants are computed
// (backend/core/accesos-blob.go). See RATIONALE.md.

import Modules from '$core/modules'
import type { IAccesoGrant, IProfile, IUser } from '$core/types/common'
import {
  normalizeAccessFrontendRoutes,
  type IAccessGroupCatalogEntry,
  type IAccessListCatalogEntry
} from './access-list-catalog'
import type { IAccess } from './users-profiles.svelte'

// Sub-access id 1 is "Todos" and is never declared in the catalog. It satisfies every check on its
// access, so it is never offered as a checkbox: it is implied by ticking every declared sub-access,
// and that is the only way it is ever stored.
export const SUB_ACCESO_TODOS_ID = 1

export const SUB_ACCESO_TODOS_NAME = "Todos"

export interface ISubAccesoOption {
  id: number
  name: string
}

// buildSubAccesoOptions returns exactly what the catalog declares — the editor's checkboxes.
export function buildSubAccesoOptions(
  subAccesosIDs?: number[],
  subAccesosNames?: string[],
): ISubAccesoOption[] {
  const declaredIDs = subAccesosIDs || []
  const declaredNames = subAccesosNames || []

  // The two catalog arrays are parallel and the backend refuses to load a mismatched pair, so a
  // mismatch here can only mean the text parser dropped something. Showing nothing is the honest
  // answer: naming the wrong sub-access is how an operator grants the wrong permission.
  if (declaredIDs.length === 0 || declaredIDs.length !== declaredNames.length) { return [] }

  return declaredIDs.map((subAccesoID, index) => ({ id: subAccesoID, name: declaredNames[index] }))
}

// A stored "Todos" stands for every declared sub-access, so the editor expands it back into the
// individual ids before it can tick or untick one of them.
export function expandSubAccesos(selectedSubAccesoIDs: number[], declaredSubAccesoIDs: number[]): number[] {
  return selectedSubAccesoIDs.includes(SUB_ACCESO_TODOS_ID)
    ? [...declaredSubAccesoIDs]
    : [...selectedSubAccesoIDs]
}

// ...and collapses it back on the way out: a selection covering every declared sub-access is stored
// as the single "Todos" flag, never as the enumeration.
export function collapseSubAccesos(selectedSubAccesoIDs: number[], declaredSubAccesoIDs: number[]): number[] {
  if (declaredSubAccesoIDs.length === 0) { return [] }

  const isEveryDeclaredSelected = declaredSubAccesoIDs.every(
    declaredID => selectedSubAccesoIDs.includes(declaredID)
  )
  return isEveryDeclaredSelected ? [SUB_ACCESO_TODOS_ID] : selectedSubAccesoIDs
}

// What a checkbox renders as: "Todos" ticks every box, because that is what it means.
export function isSubAccesoChecked(selectedSubAccesoIDs: number[], subAccesoID: number): boolean {
  return selectedSubAccesoIDs.includes(SUB_ACCESO_TODOS_ID) || selectedSubAccesoIDs.includes(subAccesoID)
}

// toggleSubAcceso returns the new selection for one access, in the stored shape. Returns a new
// array rather than mutating, so the caller decides when to write.
export function toggleSubAcceso(
  selectedSubAccesoIDs: number[],
  subAccesoID: number,
  declaredSubAccesoIDs: number[],
): number[] {
  const expandedSubAccesoIDs = expandSubAccesos(selectedSubAccesoIDs, declaredSubAccesoIDs)

  const nextSubAccesoIDs = expandedSubAccesoIDs.includes(subAccesoID)
    ? expandedSubAccesoIDs.filter(selectedID => selectedID !== subAccesoID)
    : [...expandedSubAccesoIDs, subAccesoID].sort((leftID, rightID) => leftID - rightID)

  return collapseSubAccesos(nextSubAccesoIDs, declaredSubAccesoIDs)
}

// toAccesoGrants flattens the two editable maps into the list that is stored.
//
// A sub-access on an access nobody granted is dropped rather than sent: the backend rejects the
// whole record over one, and a sub-access qualifies a permission rather than granting one, so one
// with no parent has nothing to qualify. Dropping is right here and rejecting is right there —
// this is clearing a leftover of the operator's own editing, not accepting an unauthorized grant.
export function toAccesoGrants(
  accesosMap: Map<number, number>,
  subAccesosMap?: Map<number, number[]>,
): IAccesoGrant[] {
  const accesoGrants: IAccesoGrant[] = []

  for (const [accesoID, nivel] of accesosMap) {
    if (!accesoID || !nivel) { continue }

    const subAccesoIDs = subAccesosMap?.get(accesoID) || []
    accesoGrants.push(subAccesoIDs.length > 0
      ? { AccesoID: accesoID, Nivel: nivel, SubAccesos: [...subAccesoIDs].sort((leftID, rightID) => leftID - rightID) }
      : { AccesoID: accesoID, Nivel: nivel })
  }

  return accesoGrants.sort((leftGrant, rightGrant) => leftGrant.AccesoID - rightGrant.AccesoID)
}

// fromAccesoGrants rebuilds the two editable maps from what was stored.
export function fromAccesoGrants(accesoGrants?: IAccesoGrant[]): {
  accesosMap: Map<number, number>
  subAccesosMap: Map<number, number[]>
} {
  const accesosMap = new Map<number, number>()
  const subAccesosMap = new Map<number, number[]>()

  for (const accesoGrant of accesoGrants || []) {
    if (!accesoGrant?.AccesoID || !accesoGrant.Nivel) { continue }

    accesosMap.set(accesoGrant.AccesoID, accesoGrant.Nivel)
    if (accesoGrant.SubAccesos?.length) {
      subAccesosMap.set(accesoGrant.AccesoID, [...accesoGrant.SubAccesos])
    }
  }

  return { accesosMap, subAccesosMap }
}

// A user's effective level on an access is the widest one any of its sources grants — the profiles
// it holds, plus the grants saved on the user itself. Same "highest wins" merge the backend runs
// when it packs the stored blob, so the table shows what the user will actually be able to do.
export function buildEffectiveNivelByAcceso(
  profileAccesosMaps: Map<number, number>[],
  ownAccesoGrants?: IAccesoGrant[],
): Map<number, number> {
  const nivelByAccesoID = new Map<number, number>()

  const grantedNiveles: [number, number][] = [
    ...profileAccesosMaps.flatMap(profileAccesosMap => [...profileAccesosMap]),
    ...(ownAccesoGrants || []).map(accesoGrant => [accesoGrant?.AccesoID, accesoGrant?.Nivel] as [number, number]),
  ]

  for (const [accesoID, nivel] of grantedNiveles) {
    if (accesoID && nivel > (nivelByAccesoID.get(accesoID) || 0)) { nivelByAccesoID.set(accesoID, nivel) }
  }

  return nivelByAccesoID
}

// The same merge for one user: the profiles it holds plus its own grants. Both access views read
// this, so a bar and a row can never disagree about what someone can do.
export function buildUsuarioEffectiveNiveles(
  usuario: IUser,
  perfilesByID: Map<number, IProfile>,
): Map<number, number> {
  const profileAccesosMaps = (usuario.ProfileIDs || [])
    .map(profileID => perfilesByID.get(profileID)?.accesosMap)
    .filter((accesosMap): accesosMap is Map<number, number> => !!accesosMap)

  return buildEffectiveNivelByAcceso(profileAccesosMaps, usuario.AccesosGrants)
}

export interface IUsuarioEffectiveAccess {
  usuarioID: number
  usuarioName: string
  nivelByAccesoID: Map<number, number>
}

export interface IAccessUserRow {
  rowKey: string
  accesoID: number
  accesoName: string
  groupName: string
  nivel: number
  // The id travels with the name because a card is clickable: it opens that user.
  usuarios: { usuarioID: number, usuarioName: string }[]
}

// How many users an access hands out at or above a level. The write levels (nivel > 1) are what
// order the view, so this is called once with 2 and once with 1 — the whole ranking rule.
const countUsuariosFromNivel = (accessRows: IAccessUserRow[], fromNivel: number) =>
  accessRows
    .filter(accessRow => accessRow.nivel >= fromNivel)
    .reduce((usuarioCount, accessRow) => usuarioCount + accessRow.usuarios.length, 0)

// The "by access" view, read the other way round: one row per level of an access, carrying whoever
// holds it at exactly that level. Exactly, not "at least": a user granted TODO appears on the TODO
// row only, the same way they count once in the TODO segment of their bar.
//
// Only rows somebody holds are listed, and the accesses run in order of how many users hold them at
// a write level, widest level first inside each access. The top of the table is therefore what the
// most people can change — where an access review starts.
export function buildAccessUserRows(
  usuarioEffectiveAccesses: IUsuarioEffectiveAccess[],
  accessCatalogEntries: IAccessListCatalogEntry[],
  accessGroupEntries: IAccessGroupCatalogEntry[],
): IAccessUserRow[] {
  const groupNameByID = new Map(accessGroupEntries.map(groupEntry => [groupEntry.id, groupEntry.name]))

  const accessRowsByAccess = accessCatalogEntries.map((accessEntry) => {
    // An access with no declared level is still viewable, otherwise it could never be granted.
    const declaredNiveles = decodeAccessLevels(accessEntry.levels)
    return (declaredNiveles.length > 0 ? declaredNiveles : [1])
      .sort((leftNivel, rightNivel) => rightNivel - leftNivel)
      .map(nivel => ({
        rowKey: `${accessEntry.id}-${nivel}`,
        accesoID: accessEntry.id,
        accesoName: accessEntry.name,
        groupName: groupNameByID.get(accessEntry.group) || "",
        nivel,
        usuarios: usuarioEffectiveAccesses
          .filter(usuarioEffectiveAccess => usuarioEffectiveAccess.nivelByAccesoID.get(accessEntry.id) === nivel)
          .map(({ usuarioID, usuarioName }) => ({ usuarioID, usuarioName })),
      }))
      .filter(accessRow => accessRow.usuarios.length > 0)
  })

  return accessRowsByAccess
    .filter(accessRows => accessRows.length > 0)
    .sort((leftAccessRows, rightAccessRows) =>
      countUsuariosFromNivel(rightAccessRows, 2) - countUsuariosFromNivel(leftAccessRows, 2)
      || countUsuariosFromNivel(rightAccessRows, 1) - countUsuariosFromNivel(leftAccessRows, 1)
      || leftAccessRows[0].accesoID - rightAccessRows[0].accesoID)
    .flat()
}

export interface IAccessGroupBar {
  groupID: number
  groupName: string
  totalAccessCount: number
  segments: { nivel: number, accessCount: number }[]
}

// One bar per access group: the group's whole catalog is the bar's width, and each level the user
// holds fills the fraction of it that level covers — 3 of the group's 8 accesses at TODO is 3/8 of
// the bar. Segments run widest level first. Groups the user holds nothing in are left out; an empty
// bar for each of the nine would be most of every table row.
export function buildAccessGroupBars(
  nivelByAccesoID: Map<number, number>,
  accessCatalogEntries: IAccessListCatalogEntry[],
  accessGroupEntries: IAccessGroupCatalogEntry[],
): IAccessGroupBar[] {
  return accessGroupEntries.map((accessGroupEntry) => {
    const groupAccessEntries = accessCatalogEntries.filter(entry => entry.group === accessGroupEntry.id)
    const grantedNiveles = groupAccessEntries
      .map(entry => nivelByAccesoID.get(entry.id) || 0)
      .filter(nivel => nivel > 0)

    return {
      groupID: accessGroupEntry.id,
      groupName: accessGroupEntry.name,
      totalAccessCount: groupAccessEntries.length,
      segments: [...new Set(grantedNiveles)]
        .sort((leftNivel, rightNivel) => rightNivel - leftNivel)
        .map(nivel => ({ nivel, accessCount: grantedNiveles.filter(granted => granted === nivel).length })),
    }
  }).filter(accessGroupBar => accessGroupBar.segments.length > 0)
}

// The catalog stores an access's levels as concatenated digits, e.g. 14 => VER + TODO.
export function decodeAccessLevels(accessLevelMask: number): number[] {
  return [...String(accessLevelMask)]
    .map(levelDigit => Number(levelDigit))
    .filter(levelValue => levelValue > 0)
}

// Every menu option's route, mapped to the module and group it belongs to. The access catalog only
// names a route, so this is what turns it into the module/group an access is filed under.
export function buildRouteCatalogIndex() {
  const routeCatalogIndex = new Map<string, {
    moduleID: number
    moduleName: string
    groupID: number
    groupName: string
  }>()

  for (const moduleRecord of Modules) {
    for (const menuRecord of moduleRecord.menus) {
      for (const menuOption of menuRecord.options || []) {
        const normalizedRoute = (menuOption.route || "").replace(/^\//, "")
        if (!normalizedRoute) { continue }
        routeCatalogIndex.set(normalizedRoute, {
          moduleID: moduleRecord.id,
          moduleName: moduleRecord.name,
          groupID: menuRecord.id || 0,
          groupName: menuRecord.name
        })
      }
    }
  }

  return routeCatalogIndex
}

// buildAccesosCatalog turns the parsed TOML entries into the IAccess shape AccessCard renders.
// Shared by the profiles tab and the user layer so both offer exactly the same accesses and levels.
export function buildAccesosCatalog(
  accessListEntries: IAccessListCatalogEntry[],
  routeCatalogIndex: ReturnType<typeof buildRouteCatalogIndex>,
): IAccess[] {
  const catalogAccesos = accessListEntries.map((accessListEntry) => {
    const normalizedRoute = normalizeAccessFrontendRoutes(accessListEntry.frontend_routes)[0] || ""
    const routeMeta = routeCatalogIndex.get(normalizedRoute)
    const catalogActions = decodeAccessLevels(accessListEntry.levels)

    return {
      id: accessListEntry.id,
      nombre: accessListEntry.name,
      descripcion: normalizedRoute,
      orden: accessListEntry.id,
      // An access with no declared level is still viewable, otherwise it could never be granted.
      acciones: catalogActions.length > 0 ? catalogActions : [1],
      subAccesos: buildSubAccesoOptions(accessListEntry.sub_accesses_ids, accessListEntry.sub_accesses_names),
      grupo: accessListEntry.group || routeMeta?.groupID || 0,
      modulosIDs: routeMeta ? [routeMeta.moduleID] : [],
      ss: 1,
      upd: 0
    }
  })

  return catalogAccesos.sort((leftAccess, rightAccess) => leftAccess.orden - rightAccess.orden)
}
