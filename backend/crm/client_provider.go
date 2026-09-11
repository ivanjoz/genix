package crm

import (
	"app/core"
	"app/crm/types"
	"app/db"
	"encoding/json"
)

func GetClientProviders(req *core.HandlerArgs) core.HandlerResponse {
	// Delta syncs are watermarked by "upv", the write sequence number, not by a timestamp: two
	// writes in the same second are distinguishable, so nothing is re-sent and nothing is skipped.
	updatedSince := req.GetUpVersion()
	requestedClientProviderType := req.GetQueryInt("type")

	// Type is required so the query hits the packed delta view.
	if requestedClientProviderType != int32(types.ClientProviderTypeClient) && requestedClientProviderType != int32(types.ClientProviderTypeProvider) {
		return req.MakeErr("Debe enviar type=1 (cliente) o type=2 (proveedor).")
	}

	core.Log("GetClientProviders params:", "empresa_id=", req.User.CompanyID, "type=", requestedClientProviderType, "upv_since=", updatedSince)

	clientProviders := []types.ClientProvider{}
	clientProvidersQuery := db.Query(&clientProviders)
	// Delta() keeps only active rows on a first sync and every status afterwards, so the frontend can
	// evict deleted ones from its cache.
	clientProvidersQuery.Select().
		CompanyID.Equals(req.User.CompanyID).
		Type.Equals(int8(requestedClientProviderType)).
		Delta(updatedSince, 1)

	if queryExecutionError := clientProvidersQuery.Exec(); queryExecutionError != nil {
		core.Log("GetClientProviders query error:", queryExecutionError)
		return req.MakeErr("Error al obtener clientes/proveedores:", queryExecutionError)
	}

	core.Log("GetClientProviders result_count:", len(clientProviders))
	return req.MakeResponse(clientProviders)
}

func GetClientProvidersByIDs(req *core.HandlerArgs) core.HandlerResponse {
	clientProviderCachedIDs := req.ExtractUpdatedVersionValues()
	if len(clientProviderCachedIDs) == 0 {
		return req.MakeErr("No se enviaron ids a buscar.")
	}

	core.Log("GetClientProvidersByIDs cached_ids_count:", len(clientProviderCachedIDs))

	clientProviders := []types.ClientProvider{}
	// Query only stale or missing cached rows, using the slot versions the frontend sent back.
	if queryError := db.QueryCachedIDs(&clientProviders, clientProviderCachedIDs); queryError != nil {
		core.Log("GetClientProvidersByIDs query error:", queryError)
		return req.MakeErr("Error al obtener clientes/proveedores.", queryError)
	}

	core.Log("GetClientProvidersByIDs result_count:", len(clientProviders))
	return req.MakeResponse(clientProviders)
}

func PostClientProviders(req *core.HandlerArgs) core.HandlerResponse {
	clientProvidersPayload := []types.ClientProvider{}
	if deserializeError := json.Unmarshal([]byte(*req.Body), &clientProvidersPayload); deserializeError != nil {
		core.Log("PostClientProviders deserialization error:", deserializeError)
		return req.MakeErr("Error al deserializar el body:", deserializeError)
	}

	if len(clientProvidersPayload) == 0 {
		return req.MakeErr("No se enviaron clientes/proveedores.")
	}

	core.Log("PostClientProviders payload_count:", len(clientProvidersPayload), "empresa_id=", req.User.CompanyID)
	saveError := types.SaveClientProviders(&clientProvidersPayload, req.User.CompanyID, req.User.ID, false)
	if saveError != nil {
		core.Log("PostClientProviders save error:", saveError)
		return req.MakeErr("Error al guardar clientes/proveedores:", saveError)
	}

	core.Log("PostClientProviders save done:", "result_count=", len(clientProvidersPayload))
	return req.MakeResponse(clientProvidersPayload)
}
