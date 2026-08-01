package config

import (
	"app/core"
	"app/db"
)

// GetTableRecordsByIDs resolves IDs to a flat label shape for any table that opts in through
// TableSchema.GenericRecord. It replaces the need for a dedicated "*-ids" handler per table when the
// caller only needs a display label, and it returns far less than the full record.
//
// The company scope is NOT read here: mainHandler already strips a client-sent "cmp" from private
// routes, so ExtractUpdatedVersionValues resolves the partition from the user token.
func GetTableRecordsByIDs(req *core.HandlerArgs) core.HandlerResponse {
	tableName := req.GetQuery("table")
	if len(tableName) == 0 {
		return req.MakeErr("No se envió la tabla a consultar.")
	}

	cachedIDs := req.ExtractUpdatedVersionValues()
	if len(cachedIDs) == 0 {
		return req.MakeErr("No se enviaron ids a buscar.")
	}

	records, err := db.QueryCachedGenericByIDs(tableName, cachedIDs)
	if err != nil {
		return req.MakeErr("Error al obtener los registros genéricos.", err)
	}

	return core.MakeResponse(req, &records)
}
