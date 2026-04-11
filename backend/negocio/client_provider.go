package negocio

import (
	"app/core"
	"app/db"
	s "app/negocio/types"
	"encoding/json"
	"net/mail"
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

func GetClientProvidersByIDs(req *core.HandlerArgs) core.HandlerResponse {
	clientProviderCachedIDs := req.ExtractCacheVersionValues()
	if len(clientProviderCachedIDs) == 0 {
		return req.MakeErr("No se enviaron ids a buscar.")
	}

	core.Log("GetClientProvidersByIDs cached_ids_count:", len(clientProviderCachedIDs))

	clientProviders := []s.ClientProvider{}
	// Query only stale or missing cached rows using the cache-version payload provided by the frontend.
	if queryError := db.QueryCachedIDs(&clientProviders, clientProviderCachedIDs); queryError != nil {
		core.Log("GetClientProvidersByIDs query error:", queryError)
		return req.MakeErr("Error al obtener clientes/proveedores.", queryError)
	}

	core.Log("GetClientProvidersByIDs result_count:", len(clientProviders))
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
	saveError := SaveClientProviders(&clientProvidersPayload, req.Usuario.EmpresaID, req.Usuario.ID, false)
	if saveError != nil {
		core.Log("PostClientProviders save error:", saveError)
		return req.MakeErr("Error al guardar clientes/proveedores:", saveError)
	}

	core.Log("PostClientProviders save done:", "result_count=", len(clientProvidersPayload))
	return req.MakeResponse(clientProvidersPayload)
}

func SaveClientProviders(clientProvidersPayload *[]s.ClientProvider, empresaID int32, usuarioID int32, onlyInsert bool) error {
	currentTimestamp := core.SUnixTime()
	clientProviderRegistryNumbers := core.SliceSet[string]{}
	clientProviderHashes := core.SliceSet[int64]{}

	for clientProviderIndex := range *clientProvidersPayload {
		clientProvider := &(*clientProvidersPayload)[clientProviderIndex]

		// Normalize text fields before validation so the backend owns the final persisted shape.
		clientProvider.Name = strings.TrimSpace(clientProvider.Name)
		clientProvider.Email = strings.TrimSpace(strings.ToLower(clientProvider.Email))
		clientProvider.RegistryNumber = strings.TrimSpace(clientProvider.RegistryNumber)
		clientProvider.CityID = strings.TrimSpace(clientProvider.CityID)

		if clientProvider.Type != s.ClientProviderTypeClient && clientProvider.Type != s.ClientProviderTypeProvider {
			return core.Err("El registro en posición", clientProviderIndex, "tiene un Type inválido. Use 1 (cliente) o 2 (proveedor).")
		}

		if len(clientProvider.Name) == 0 {
			return core.Err("El registro en posición", clientProviderIndex, "debe tener Name.")
		}

		if clientProvider.PersonType != s.PersonTypeNatural && clientProvider.PersonType != s.PersonTypeCompany {
			return core.Err("El registro en posición", clientProviderIndex, "tiene PersonType inválido. Use 1 (persona) o 2 (empresa).")
		}

		if len(clientProvider.Email) > 0 && !isValidEmailAddress(clientProvider.Email) {
			return core.Err("El registro en posición", clientProviderIndex, "tiene un Email inválido.")
		}

		if clientProvider.Type == s.ClientProviderTypeProvider && clientProvider.CountryID <= 0 {
			return core.Err("El registro en posición", clientProviderIndex, "debe tener CountryID válido.")
		}

		if clientProvider.Type == s.ClientProviderTypeProvider && len(clientProvider.CityID) == 0 {
			return core.Err("El registro en posición", clientProviderIndex, "debe tener CityID válido.")
		}

		// Providers must keep a numeric registry number; natural clients can omit it completely.
		if clientProvider.Type == s.ClientProviderTypeProvider && !companyRegistryNumberPattern.MatchString(clientProvider.RegistryNumber) {
			return core.Err("El registro en posición", clientProviderIndex, "debe tener RegistryNumber numérico de 7 a 12 dígitos para proveedores.")
		}

		// Keep the deduplication hash server-owned so DB lookups and writes use the same identity rule.
		clientProvider.SelfParse()

		// Enforce tenant and audit fields from the authenticated user, never from the client payload.
		clientProvider.EmpresaID = empresaID
		clientProvider.Created = currentTimestamp
		clientProvider.CreatedBy = usuarioID
		clientProvider.Status = 1
		clientProvider.Updated = currentTimestamp
		clientProvider.UpdatedBy = usuarioID

		if clientProvider.RegistryNumber != "" && clientProviderRegistryNumbers.Include(clientProvider.RegistryNumber) {
			return core.Err("El registro en posición", clientProviderIndex, "repite RegistryNumber dentro del payload.")
		}
		clientProviderRegistryNumbers.AddIf(clientProvider.RegistryNumber)

		if clientProviderHashes.Include(clientProvider.NameRegistryHash) {
			return core.Err("El registro en posición", clientProviderIndex, "repite NameRegistryHash dentro del payload.")
		}
		clientProviderHashes.Add(clientProvider.NameRegistryHash)
	}

	// Resolve existing IDs before Merge so equal identities update instead of inserting duplicates.
	existingByRegistryNumber := []s.ClientProvider{}
	if !clientProviderRegistryNumbers.IsEmpty() {
		q := db.Query(&existingByRegistryNumber).AllowFilter()
		err := q.Select(q.RegistryNumber, q.ID).
			EmpresaID.Equals(empresaID).
			RegistryNumber.In(clientProviderRegistryNumbers.Values...).Exec()

		if err != nil {
			return core.Err("Error al buscar clientes/proveedores por RegistryNumber.", err)
		}
	}

	existingByRegistryNumberMap := core.SliceToMapE(existingByRegistryNumber,
		func(e s.ClientProvider) string { return e.RegistryNumber })

	existingByHash := []s.ClientProvider{}
	if !clientProviderHashes.IsEmpty() {
		q := db.Query(&existingByHash).AllowFilter()
		err := q.Select(q.NameRegistryHash, q.ID).
			EmpresaID.Equals(empresaID).
			NameRegistryHash.In(clientProviderHashes.Values...).Exec()

		if err != nil {
			return core.Err("Error al buscar clientes/proveedores por NameRegistryHash.", err)
		}
	}

	existingByHashMap := core.SliceToMapE(existingByHash,
		func(e s.ClientProvider) int64 { return e.NameRegistryHash })

	for clientProviderIndex := range *clientProvidersPayload {
		currentClientProvider := &(*clientProvidersPayload)[clientProviderIndex]
		if currentClientProvider.ID > 0 {
			continue
		}

		if existing, ok := existingByRegistryNumberMap[currentClientProvider.RegistryNumber]; ok {
			currentClientProvider.ID = existing.ID
			continue
		}

		if existing, ok := existingByHashMap[currentClientProvider.NameRegistryHash]; ok {
			currentClientProvider.ID = existing.ID
		}
	}

	core.Log("saveClientProviders merge start:", "payload_count=", len(*clientProvidersPayload))
	clientProviderTable := db.Table[s.ClientProvider]()
	if mergeError := db.Merge(clientProvidersPayload,
		[]db.Coln{clientProviderTable.Created, clientProviderTable.CreatedBy},
		func(_ *s.ClientProvider, currentClientProvider *s.ClientProvider) bool {
			return !onlyInsert || currentClientProvider.ID <= 0
		},
		func(_ *s.ClientProvider) {},
	); mergeError != nil {
		return mergeError
	}

	return nil
}

func isValidEmailAddress(emailAddress string) bool {
	// Enforce plain addresses only so persisted emails stay normalized and free from display-name formats.
	parsedAddress, parseError := mail.ParseAddress(emailAddress)
	return parseError == nil && parsedAddress.Address == emailAddress
}
