package operaciones

import (
	"app/core"
	"app/db"
	negocioTypes "app/negocio/types"
	s "app/types"
	"encoding/json"
	"slices"
)

func GetProductSupply(req *core.HandlerArgs) core.HandlerResponse {
	productSupplyRecords := []s.ProductSupply{}
	productSupplyQuery := db.Query(&productSupplyRecords)
	productSupplyQuery.Select().
		CompanyID.Equals(req.Usuario.EmpresaID).
		Status.Equals(1)

	if queryError := productSupplyQuery.Exec(); queryError != nil {
		core.Log("GetProductSupply query error:", queryError)
		return req.MakeErr("Error al obtener la configuración de abastecimiento.", queryError)
	}

	core.Log("GetProductSupply result_count:", len(productSupplyRecords))
	return req.MakeResponse(productSupplyRecords)
}

func PostProductSupply(req *core.HandlerArgs) core.HandlerResponse {
	productSupplyRecord := s.ProductSupply{}
	if deserializeError := json.Unmarshal([]byte(*req.Body), &productSupplyRecord); deserializeError != nil {
		core.Log("PostProductSupply deserialization error:", deserializeError)
		return req.MakeErr("Error al deserializar el body.", deserializeError)
	}

	productSupplyRecord.ProviderSupply = sanitizeProviderSupplyRows(productSupplyRecord.ProviderSupply)

	if productSupplyRecord.ProductID <= 0 {
		return req.MakeErr("Debe enviar un ProductID válido.")
	}
	if productSupplyRecord.MinimunStock < 0 {
		return req.MakeErr("El stock mínimo no puede ser negativo.")
	}
	if productSupplyRecord.SalesPerDayEstimated < 0 {
		return req.MakeErr("Las ventas por día estimadas no pueden ser negativas.")
	}

	if validationError := validateProviderSupplyRows(req, productSupplyRecord.ProviderSupply); validationError != nil {
		return req.MakeErr(validationError)
	}

	currentTimestamp := core.SUnixTime()
	productSupplyRecord.CompanyID = req.Usuario.EmpresaID
	productSupplyRecord.Status = 1
	productSupplyRecord.Updated = currentTimestamp
	productSupplyRecord.UpdatedBy = req.Usuario.ID

	productSupplyRecords := []s.ProductSupply{productSupplyRecord}
	if mergeError := db.Merge(&productSupplyRecords, nil,
		func(previousProductSupply, currentProductSupply *s.ProductSupply) bool {
			// Keep the product key immutable and refresh only mutable configuration fields.
			currentProductSupply.CompanyID = req.Usuario.EmpresaID
			currentProductSupply.ProductID = previousProductSupply.ProductID
			currentProductSupply.Status = 1
			currentProductSupply.Updated = currentTimestamp
			currentProductSupply.UpdatedBy = req.Usuario.ID
			return true
		},
		func(currentProductSupply *s.ProductSupply) {
			// Fill server-owned metadata on insert so clients only send configuration fields.
			currentProductSupply.CompanyID = req.Usuario.EmpresaID
			currentProductSupply.Status = 1
			currentProductSupply.Updated = currentTimestamp
			currentProductSupply.UpdatedBy = req.Usuario.ID
		},
	); mergeError != nil {
		core.Log("PostProductSupply merge error:", mergeError)
		return req.MakeErr("Error al guardar la configuración de abastecimiento.", mergeError)
	}

	core.Log("PostProductSupply saved:", "product_id=", productSupplyRecord.ProductID, "provider_count=", len(productSupplyRecord.ProviderSupply))
	return req.MakeResponse(productSupplyRecords[0])
}

func sanitizeProviderSupplyRows(providerSupplyRows []s.ProductSupplyProviderRow) []s.ProductSupplyProviderRow {
	sanitizedProviderSupplyRows := make([]s.ProductSupplyProviderRow, 0, len(providerSupplyRows))

	for _, providerSupplyRow := range providerSupplyRows {
		if providerSupplyRow.ProviderID <= 0 && providerSupplyRow.Capacity == 0 && providerSupplyRow.DeliveryTime == 0 && providerSupplyRow.Price == 0 {
			continue
		}
		sanitizedProviderSupplyRows = append(sanitizedProviderSupplyRows, providerSupplyRow)
	}

	return sanitizedProviderSupplyRows
}

func validateProviderSupplyRows(req *core.HandlerArgs, providerSupplyRows []s.ProductSupplyProviderRow) error {
	if len(providerSupplyRows) == 0 {
		return nil
	}

	providerIDs := make([]int32, 0, len(providerSupplyRows))
	for _, providerSupplyRow := range providerSupplyRows {
		if providerSupplyRow.ProviderID <= 0 {
			return core.Err("Cada fila de proveedor debe tener un proveedor válido.")
		}
		if providerSupplyRow.Capacity < 0 {
			return core.Err("La capacidad no puede ser negativa.")
		}
		if providerSupplyRow.DeliveryTime < 0 {
			return core.Err("El tiempo de entrega no puede ser negativo.")
		}
		if providerSupplyRow.Price < 0 {
			return core.Err("El precio no puede ser negativo.")
		}
		if slices.Contains(providerIDs, providerSupplyRow.ProviderID) {
			return core.Err("No se puede repetir el mismo proveedor en un producto.")
		}
		providerIDs = append(providerIDs, providerSupplyRow.ProviderID)
	}

	providers := []negocioTypes.ClientProvider{}
	providerQuery := db.Query(&providers)
	providerTable := db.Table[negocioTypes.ClientProvider]()
	providerQuery.Select(providerTable.ID, providerTable.Type, providerTable.Status).
		EmpresaID.Equals(req.Usuario.EmpresaID).
		ID.In(providerIDs...)

	if queryError := providerQuery.Exec(); queryError != nil {
		return core.Err("Error al validar los proveedores.", queryError)
	}

	if len(providers) != len(providerIDs) {
		return core.Err("Uno o más proveedores no existen.")
	}

	for _, providerRecord := range providers {
		if providerRecord.Status <= 0 {
			return core.Err("Uno o más proveedores están inactivos.")
		}
		if providerRecord.Type != negocioTypes.ClientProviderTypeProvider {
			return core.Err("Uno o más registros seleccionados no son proveedores.")
		}
	}

	return nil
}
