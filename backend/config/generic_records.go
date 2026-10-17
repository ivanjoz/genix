package config

import (
	"app/core"
	"app/db"
)

// GetTableRecordsByIDs resolves IDs to a flat label shape for any table that opts in through
// TableSchema.GenericRecord. It replaces the need for a dedicated "*-ids" handler per table when the
// caller only needs a display label, and it returns far less than the full record.
//
// The partition is the token's company: mainHandler strips a client-sent "cmp" from private routes.
func GetTableRecordsByIDs(req *core.HandlerArgs) core.HandlerResponse {
	tableName := req.GetQuery("table")
	if len(tableName) == 0 {
		return req.MakeErr("No se envió la tabla a consultar.")
	}

	cachedIDsRequest, err := req.GetCachedIDsRequest(req.User.CompanyID)
	if err != nil {
		return req.MakeErr(err)
	}
	partitions, err := db.QueryCachedGenericByIDs(tableName, cachedIDsRequest)
	if err != nil {
		return req.MakeErr("Error al obtener los registros genéricos.", err)
	}
	return req.MakeResponse(core.HideCachedPartitions(partitions))
}
