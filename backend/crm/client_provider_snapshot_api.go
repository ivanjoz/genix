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
	cachedIDsRequest, err := req.GetCachedIDsRequest(req.User.CompanyID)
	if err != nil {
		return req.MakeErr(err)
	}
	partitions, err := db.QueryCachedIDs[types.ClientProviderSnapshot](cachedIDsRequest)
	if err != nil {
		return req.MakeErr("Error al obtener las identidades de cliente/proveedor.", err)
	}
	return req.MakeResponse(core.HideCachedPartitions(partitions))
}
