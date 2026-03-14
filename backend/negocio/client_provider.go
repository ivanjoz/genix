package negocio

import (
	"app/core"
	"app/db"
	s "app/negocio/types"
	"encoding/json"
	"regexp"
	"strings"
)

var companyRegistryNumberPattern = regexp.MustCompile(`^\d{7,12}$`)

func GetClientProviders(req *core.HandlerArgs) core.HandlerResponse {
	updatedSince := core.Coalesce(req.GetQueryInt("upd"), req.GetQueryInt("updated"))
	requestedClientProviderType := req.GetQueryInt("type")

	// Type is required so the query hits the type+updated composite view.
	if requestedClientProviderType != int32(s.ClientProviderTypeClient) && requestedClientProviderType != int32(s.ClientProviderTypeProvider) {
		return req.MakeErr("Debe enviar type=1 (cliente) o type=2 (proveedor).")
	}

	core.Log("GetClientProviders params:", "empresa_id=", req.Usuario.EmpresaID, "type=", requestedClientProviderType, "updated_since=", updatedSince)

	clientProviders := []s.ClientProvider{}
	clientProvidersQuery := db.Query(&clientProviders)
	clientProvidersQuery.Select().
		EmpresaID.Equals(req.Usuario.EmpresaID).
		Type.Equals(int8(requestedClientProviderType))

	// Delta sync includes deleted rows so the frontend can evict them from cache.
	if updatedSince > 0 {
		clientProvidersQuery.Updated.GreaterThan(updatedSince)
	} else {
		clientProvidersQuery.Status.Equals(1)
	}

	if queryExecutionError := clientProvidersQuery.Exec(); queryExecutionError != nil {
		core.Log("GetClientProviders query error:", queryExecutionError)
		return req.MakeErr("Error al obtener clientes/proveedores:", queryExecutionError)
	}

	core.Log("GetClientProviders result_count:", len(clientProviders))
	return req.MakeResponse(clientProviders)
}

func PostClientProviders(req *core.HandlerArgs) core.HandlerResponse {
	clientProvidersPayload := []s.ClientProvider{}
	if deserializeError := json.Unmarshal([]byte(*req.Body), &clientProvidersPayload); deserializeError != nil {
		core.Log("PostClientProviders deserialization error:", deserializeError)
		return req.MakeErr("Error al deserializar el body:", deserializeError)
	}

	if len(clientProvidersPayload) == 0 {
		return req.MakeErr("No se enviaron clientes/proveedores.")
	}

	core.Log("PostClientProviders payload_count:", len(clientProvidersPayload), "empresa_id=", req.Usuario.EmpresaID)

	currentTimestamp := core.SUnixTime()

	for clientProviderIndex := range clientProvidersPayload {
		currentClientProvider := &clientProvidersPayload[clientProviderIndex]

		// Normalize text fields before validation so the backend owns the final persisted shape.
		currentClientProvider.Name = strings.TrimSpace(currentClientProvider.Name)
		currentClientProvider.Email = strings.TrimSpace(strings.ToLower(currentClientProvider.Email))
		currentClientProvider.RegistryNumber = strings.TrimSpace(currentClientProvider.RegistryNumber)
		currentClientProvider.CityID = strings.TrimSpace(currentClientProvider.CityID)

		if currentClientProvider.Type != s.ClientProviderTypeClient && currentClientProvider.Type != s.ClientProviderTypeProvider {
			return req.MakeErr("El registro en posición", clientProviderIndex, "tiene un Type inválido. Use 1 (cliente) o 2 (proveedor).")
		}

		if len(currentClientProvider.Name) == 0 {
			return req.MakeErr("El registro en posición", clientProviderIndex, "debe tener Name.")
		}

		if currentClientProvider.PersonType != s.PersonTypeNatural && currentClientProvider.PersonType != s.PersonTypeCompany {
			return req.MakeErr("El registro en posición", clientProviderIndex, "tiene PersonType inválido. Use 1 (persona) o 2 (empresa).")
		}

		if len(currentClientProvider.Email) == 0 || !strings.Contains(currentClientProvider.Email, "@") {
			return req.MakeErr("El registro en posición", clientProviderIndex, "debe tener un Email válido.")
		}

		if currentClientProvider.CountryID <= 0 {
			return req.MakeErr("El registro en posición", clientProviderIndex, "debe tener CountryID válido.")
		}

		if len(currentClientProvider.CityID) == 0 {
			return req.MakeErr("El registro en posición", clientProviderIndex, "debe tener CityID válido.")
		}

		// RegistryNumber is required only for companies and must remain numeric with 7..12 digits.
		if currentClientProvider.PersonType == s.PersonTypeCompany && !companyRegistryNumberPattern.MatchString(currentClientProvider.RegistryNumber) {
			return req.MakeErr("El registro en posición", clientProviderIndex, "debe tener RegistryNumber numérico de 7 a 12 dígitos para empresas.")
		}

		// Enforce tenant and audit fields from the authenticated user, never from the client payload.
		currentClientProvider.EmpresaID = req.Usuario.EmpresaID
		currentClientProvider.Status = 1
		currentClientProvider.Updated = currentTimestamp
		currentClientProvider.UpdatedBy = req.Usuario.ID
	}

	core.Log("PostClientProviders merge start:", "payload_count=", len(clientProvidersPayload))
	if mergeError := db.Merge(&clientProvidersPayload, nil,
		func(previousClientProvider, currentClientProvider *s.ClientProvider) bool {
			// Preserve key ownership while refreshing mutable fields on update.
			currentClientProvider.EmpresaID = req.Usuario.EmpresaID
			currentClientProvider.ID = previousClientProvider.ID
			currentClientProvider.Status = 1
			currentClientProvider.Updated = currentTimestamp
			currentClientProvider.UpdatedBy = req.Usuario.ID
			return true
		},
		func(currentClientProvider *s.ClientProvider) {
			// Fill server-owned fields on insert so clients cannot forge tenancy or audit data.
			currentClientProvider.EmpresaID = req.Usuario.EmpresaID
			currentClientProvider.Status = 1
			currentClientProvider.Updated = currentTimestamp
			currentClientProvider.UpdatedBy = req.Usuario.ID
		},
	); mergeError != nil {
		core.Log("PostClientProviders merge error:", mergeError)
		return req.MakeErr("Error al guardar clientes/proveedores:", mergeError)
	}

	core.Log("PostClientProviders merge done:", "result_count=", len(clientProvidersPayload))
	return req.MakeResponse(clientProvidersPayload)
}
