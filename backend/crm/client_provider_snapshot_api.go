package crm

import (
	"app/core"
	"app/crm/types"
	"app/db"
)

// GetClientProviderSnapshotsByIDs resolves frozen identities through the by-ids cache.
//
// A document pins a snapshot id and whoever renders it joins here, rather than the handler
// that produced the id doing the join. That is the better trade on this table specifically:
// nothing ever updates a snapshot, so a browser that has one can keep it forever, and the
// accounting books read the same few hundred identities on every period.
//
// The rows carry no Status column — there is nothing to delete — and the cache reads a missing
// "ss" as active, so none is invented here.
func GetClientProviderSnapshotsByIDs(req *core.HandlerArgs) core.HandlerResponse {
	snapshotCachedIDs := req.ExtractUpdatedVersionValues()
	if len(snapshotCachedIDs) == 0 {
		return req.MakeErr("No se enviaron ids a buscar.")
	}

	snapshots := []types.ClientProviderSnapshot{}
	if queryError := db.QueryCachedIDs(&snapshots, snapshotCachedIDs); queryError != nil {
		core.Log("GetClientProviderSnapshotsByIDs query error:", queryError)
		return req.MakeErr("Error al obtener las identidades de cliente/proveedor.", queryError)
	}

	core.Log("GetClientProviderSnapshotsByIDs result_count:", len(snapshots))
	return req.MakeResponse(snapshots)
}
