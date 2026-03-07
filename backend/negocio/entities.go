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

func GetEntities(req *core.HandlerArgs) core.HandlerResponse {
	updatedSince := core.Coalesce(req.GetQueryInt("upd"), req.GetQueryInt("updated"))
	requestedEntityType := req.GetQueryInt("type")

	// Type is required by contract to route queries through the Type+Updated composite view.
	if requestedEntityType != int32(s.EntityTypeClient) && requestedEntityType != int32(s.EntityTypeProvider) {
		return req.MakeErr("Debe enviar type=1 (cliente) o type=2 (proveedor).")
	}

	core.Log("GetEntities params:", "empresa_id=", req.Usuario.EmpresaID, "type=", requestedEntityType, "updated_since=", updatedSince)

	entities := []s.Entity{}
	entitiesQuery := db.Query(&entities)
	entitiesQuery.Select().
		EmpresaID.Equals(req.Usuario.EmpresaID).
		Type.Equals(int8(requestedEntityType))

	// Delta sync includes status=0 records so the client can delete from local cache.
	if updatedSince > 0 {
		entitiesQuery.Updated.GreaterThan(updatedSince)
	} else {
		entitiesQuery.Status.Equals(1)
	}

	if queryExecutionError := entitiesQuery.Exec(); queryExecutionError != nil {
		core.Log("GetEntities query error:", queryExecutionError)
		return req.MakeErr("Error al obtener entidades:", queryExecutionError)
	}

	core.Log("GetEntities result_count:", len(entities))
	return req.MakeResponse(entities)
}

func PostEntities(req *core.HandlerArgs) core.HandlerResponse {
	entitiesPayload := []s.Entity{}
	if deserializeError := json.Unmarshal([]byte(*req.Body), &entitiesPayload); deserializeError != nil {
		core.Log("PostEntities deserialization error:", deserializeError)
		return req.MakeErr("Error al deserializar el body:", deserializeError)
	}

	if len(entitiesPayload) == 0 {
		return req.MakeErr("No se enviaron entidades.")
	}

	core.Log("PostEntities payload_count:", len(entitiesPayload), "empresa_id=", req.Usuario.EmpresaID)

	currentTimestamp := core.SUnixTime()

	for entityIndex := range entitiesPayload {
		currentEntity := &entitiesPayload[entityIndex]

		// Normalize text fields before applying validation and persistence.
		currentEntity.Name = strings.TrimSpace(currentEntity.Name)
		currentEntity.Email = strings.TrimSpace(strings.ToLower(currentEntity.Email))
		currentEntity.RegistryNumber = strings.TrimSpace(currentEntity.RegistryNumber)
		currentEntity.CityID = strings.TrimSpace(currentEntity.CityID)

		if currentEntity.Type != s.EntityTypeClient && currentEntity.Type != s.EntityTypeProvider {
			return req.MakeErr("La entidad en posición", entityIndex, "tiene un Type inválido. Use 1 (cliente) o 2 (proveedor).")
		}

		if len(currentEntity.Name) == 0 {
			return req.MakeErr("La entidad en posición", entityIndex, "debe tener Name.")
		}

		if currentEntity.PersonType != s.PersonTypeNatural && currentEntity.PersonType != s.PersonTypeCompany {
			return req.MakeErr("La entidad en posición", entityIndex, "tiene PersonType inválido. Use 1 (persona) o 2 (empresa).")
		}

		if len(currentEntity.Email) == 0 || !strings.Contains(currentEntity.Email, "@") {
			return req.MakeErr("La entidad en posición", entityIndex, "debe tener un Email válido.")
		}

		if currentEntity.CountryID <= 0 {
			return req.MakeErr("La entidad en posición", entityIndex, "debe tener CountryID válido.")
		}

		if len(currentEntity.CityID) == 0 {
			return req.MakeErr("La entidad en posición", entityIndex, "debe tener CityID válido.")
		}

		// RegistryNumber is required only for companies and must be numeric with 7..12 digits.
		if currentEntity.PersonType == s.PersonTypeCompany && !companyRegistryNumberPattern.MatchString(currentEntity.RegistryNumber) {
			return req.MakeErr("La entidad en posición", entityIndex, "debe tener RegistryNumber numérico de 7 a 12 dígitos para empresas.")
		}

		// Enforce tenant, audit, and active state from server side.
		currentEntity.EmpresaID = req.Usuario.EmpresaID
		currentEntity.Status = 1
		currentEntity.Updated = currentTimestamp
		currentEntity.UpdatedBy = req.Usuario.ID
	}

	core.Log("PostEntities merge start:", "payload_count=", len(entitiesPayload))
	if mergeError := db.Merge(&entitiesPayload, nil,
		func(previousEntity, currentEntity *s.Entity) bool {
			// Update path: keep tenant and key integrity while refreshing mutable fields.
			currentEntity.EmpresaID = req.Usuario.EmpresaID
			currentEntity.ID = previousEntity.ID
			currentEntity.Status = 1
			currentEntity.Updated = currentTimestamp
			currentEntity.UpdatedBy = req.Usuario.ID
			return true
		},
		func(currentEntity *s.Entity) {
			// Insert path: enforce tenant and audit fields server-side.
			currentEntity.EmpresaID = req.Usuario.EmpresaID
			currentEntity.Status = 1
			currentEntity.Updated = currentTimestamp
			currentEntity.UpdatedBy = req.Usuario.ID
		},
	); mergeError != nil {
		core.Log("PostEntities merge error:", mergeError)
		return req.MakeErr("Error al guardar entidades:", mergeError)
	}

	core.Log("PostEntities merge done:", "result_count=", len(entitiesPayload))
	return req.MakeResponse(entitiesPayload)
}
