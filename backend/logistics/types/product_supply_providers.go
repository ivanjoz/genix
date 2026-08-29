// Provider rows of a supply are written from two modules — `logistics` saves the
// replenishment config of a product, and `production` saves an insumo together with its
// providers — so their sanitize/validate pass lives here rather than in either body
// (see backend/docs/MODULE_BOUNDARIES.md).
package types

import (
	"app/core"
	crm "app/crm/types"
	"app/db"
	"slices"
)

func SanitizeProviderSupplyRows(providerSupplyRows []ProductSupplyProviderRow) []ProductSupplyProviderRow {
	sanitizedProviderSupplyRows := make([]ProductSupplyProviderRow, 0, len(providerSupplyRows))

	for _, providerSupplyRow := range providerSupplyRows {
		if providerSupplyRow.ProviderID <= 0 && providerSupplyRow.Capacity == 0 && providerSupplyRow.DeliveryTime == 0 && providerSupplyRow.Price == 0 {
			continue
		}
		sanitizedProviderSupplyRows = append(sanitizedProviderSupplyRows, providerSupplyRow)
	}

	return sanitizedProviderSupplyRows
}

func ValidateProviderSupplyRows(companyID int32, providerSupplyRows []ProductSupplyProviderRow) error {
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
			return core.Err("No se puede repetir el mismo proveedor en un product.")
		}
		providerIDs = append(providerIDs, providerSupplyRow.ProviderID)
	}

	providers := []crm.ClientProvider{}
	providerQuery := db.Query(&providers)
	providerTable := db.TableOf[crm.ClientProvider]()
	providerQuery.Select(providerTable.ID, providerTable.Type, providerTable.Status).
		CompanyID.Equals(companyID).
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
		if providerRecord.Type != crm.ClientProviderTypeProvider {
			return core.Err("Uno o más registros seleccionados no son proveedores.")
		}
	}

	return nil
}
