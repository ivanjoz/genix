package crm

import (
	"app/core"
	"app/crm/types"
	"app/db"
	"encoding/json"
)

func GetClientProviders(req *core.HandlerArgs) core.HandlerResponse {
	requestedClientProviderType := req.GetQueryInt("type")

	// Type is required: it is the delta index's pinned key, one log per type.
	if requestedClientProviderType != int32(types.ClientProviderTypeClient) && requestedClientProviderType != int32(types.ClientProviderTypeProvider) {
		return req.MakeErr("Debe enviar type=1 (cliente) o type=2 (proveedor).")
	}

	clientProviders := []types.ClientProvider{}
	clientProvidersQuery := db.Query(&clientProviders)
	// A first sync lists the active rows; later syncs read the log, whose deleted entries evict.
	clientProvidersQuery.Select().
		CompanyID.Equals(req.User.CompanyID).
		Type.Equals(int8(requestedClientProviderType)).
		Delta(req.GetDeltaCursor(), int8(1))

	if queryExecutionError := clientProvidersQuery.Exec(); queryExecutionError != nil {
		core.Log("GetClientProviders query error:", queryExecutionError)
		return req.MakeErr("Error al obtener clientes/proveedores:", queryExecutionError)
	}

	core.Log("GetClientProviders result_count:", len(clientProviders))
	return req.MakeResponse(clientProviders)
}

func GetClientProvidersByIDs(req *core.HandlerArgs) core.HandlerResponse {
	cachedIDsRequest, err := req.GetCachedIDsRequest(req.User.CompanyID)
	if err != nil {
		return req.MakeErr(err)
	}
	// Only the records whose slot moved past the tick the client holds come back.
	partitions, err := db.QueryCachedIDs[types.ClientProvider](cachedIDsRequest)
	if err != nil {
		return req.MakeErr("Error al obtener clientes/proveedores.", err)
	}
	return req.MakeResponse(core.HideCachedPartitions(partitions))
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
