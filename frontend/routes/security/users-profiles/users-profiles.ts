// Pure sub-access editing rules. No Svelte, no fetch: the whole point is that the "Todos" rule and
// the wire packing can be tested without a component or a network call.
//
// The wire shape is `accesoID * 100 + subAccesoID`, readable on purpose. The profile is what a human
// edits, so it stays a list of numbers anyone can read in the database; the binary grant blobs are
// packed exactly once, in the backend, when a user's grants are computed
// (backend/core/accesos-blob.go). See RATIONALE.md.

// Sub-access id 1 is "Todos" and is never declared in the catalog. It satisfies every check on its
// access, so granting it and granting individual sub-accesses at the same time is contradictory —
// which is why selecting it clears the rest and selecting anything else clears it.
export const SUB_ACCESO_TODOS_ID = 1

export const SUB_ACCESO_TODOS_NAME = "Todos"

export interface ISubAccesoOption {
  id: number
  name: string
}

// buildSubAccesoOptions puts the synthetic "Todos" first, ahead of what the catalog declares.
// An access that declares no sub-access gets no options at all — not even Todos, which would have
// nothing to stand for.
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

  return [
    { id: SUB_ACCESO_TODOS_ID, name: SUB_ACCESO_TODOS_NAME },
    ...declaredIDs.map((subAccesoID, index) => ({ id: subAccesoID, name: declaredNames[index] })),
  ]
}

// toggleSubAcceso returns the new selection for one access, applying the "Todos" exclusivity in
// both directions. Returns a new array rather than mutating, so the caller decides when to write.
export function toggleSubAcceso(
  selectedSubAccesoIDs: number[],
  subAccesoID: number,
): number[] {
  if (selectedSubAccesoIDs.includes(subAccesoID)) {
    return selectedSubAccesoIDs.filter(selectedID => selectedID !== subAccesoID)
  }
  if (subAccesoID === SUB_ACCESO_TODOS_ID) {
    return [SUB_ACCESO_TODOS_ID]
  }
  return [...selectedSubAccesoIDs.filter(selectedID => selectedID !== SUB_ACCESO_TODOS_ID), subAccesoID]
    .sort((leftID, rightID) => leftID - rightID)
}

// packSubAccesos flattens the editable map into the wire array.
//
// It drops any sub-access whose parent access is not granted, because the backend rejects the whole
// profile over one — a sub-access qualifies a permission rather than granting one, so one with no
// parent has nothing to qualify. Dropping is right here and rejecting is right there: this is
// clearing a leftover of the operator's own editing, not accepting an unauthorized grant.
export function packSubAccesos(
  subAccesosMap: Map<number, number[]>,
  accesosMap: Map<number, number[]>,
): number[] {
  const packedSubAccesos: number[] = []

  for (const [accesoID, subAccesoIDs] of subAccesosMap) {
    if (!(accesosMap.get(accesoID)?.length)) { continue }
    for (const subAccesoID of subAccesoIDs) {
      packedSubAccesos.push(accesoID * 100 + subAccesoID)
    }
  }

  return packedSubAccesos.sort((leftRef, rightRef) => leftRef - rightRef)
}

// unpackSubAccesos rebuilds the editable map from what the backend stored.
export function unpackSubAccesos(packedSubAccesos?: number[]): Map<number, number[]> {
  const subAccesosMap = new Map<number, number[]>()

  for (const subAccesoRef of packedSubAccesos || []) {
    const accesoID = Math.floor(subAccesoRef / 100)
    const subAccesoID = subAccesoRef % 100
    const subAccesoIDs = subAccesosMap.get(accesoID)
    subAccesoIDs ? subAccesoIDs.push(subAccesoID) : subAccesosMap.set(accesoID, [subAccesoID])
  }

  return subAccesosMap
}
