package logistics

import (
	"app/core"
	"app/db"
	logisticsTypes "app/logistics/types"
	"encoding/json"
)

// GetSupplyMaterials returns the supply-material catalog using the delta-cache
// protocol: with `upd=0` only active rows are returned; with `upd>0` every row
// updated after that watermark (including soft-deleted Status=0) is returned so
// the client can evict stale local entries.
func GetSupplyMaterials(req *core.HandlerArgs) core.HandlerResponse {
	updatedWatermark := core.Coalesce(req.GetQueryInt("upd"), req.GetQueryInt("updated"))

	supplyMaterialRecords := []logisticsTypes.SupplyMaterial{}
	supplyMaterialQuery := db.Query(&supplyMaterialRecords)
	supplyMaterialQuery.Select().CompanyID.Equals(req.User.CompanyID)

	if updatedWatermark > 0 {
		supplyMaterialQuery.Updated.GreaterThan(updatedWatermark)
	} else {
		supplyMaterialQuery.Status.Equals(1)
	}

	if queryError := supplyMaterialQuery.Exec(); queryError != nil {
		core.Log("GetSupplyMaterials query error:", queryError)
		return req.MakeErr("Error al obtener los insumos.", queryError)
	}

	core.Log("GetSupplyMaterials result_count:", len(supplyMaterialRecords))
	return req.MakeResponse(supplyMaterialRecords)
}

// supplyMaterialIDMapping is the shape the frontend GetHandler.postAndSync expects
// back: for each input record, the resolved server ID paired with the client-sent
// (possibly negative/temporary) ID so optimistic local rows can be reconciled.
type supplyMaterialIDMapping struct {
	ID     int32 `json:",omitempty"`
	TempID int32 `json:",omitempty"`
}

// PostSupplyMaterial upserts a batch of supply-material records. Mirrors the
// productos POST contract: payload is an array, response is []{ID, TempID} so
// the frontend can map its temp IDs to the server-assigned ones. New rows
// (ID<=0) get an autoincrement ID; existing rows are updated while protecting
// Created/CreatedBy.
func PostSupplyMaterial(req *core.HandlerArgs) core.HandlerResponse {
	incomingSupplyMaterials := []logisticsTypes.SupplyMaterial{}
	if deserializeError := json.Unmarshal([]byte(*req.Body), &incomingSupplyMaterials); deserializeError != nil {
		core.Log("PostSupplyMaterial deserialization error:", deserializeError)
		return req.MakeErr("Error al deserializar el body.", deserializeError)
	}
	if len(incomingSupplyMaterials) == 0 {
		return req.MakeErr("No se recibieron insumos para guardar.")
	}

	currentTimestamp := core.SUnixTime()
	// Preserve the client-sent ID (TempID) per record so we can return ID mappings
	// after the autoincrement assigns real IDs to the inserts.
	clientSentIDs := make([]int32, len(incomingSupplyMaterials))

	// Bucket the batch into inserts and updates to use the right ORM call per group.
	toInsert := []logisticsTypes.SupplyMaterial{}
	toUpdate := []logisticsTypes.SupplyMaterial{}
	for recordIndex := range incomingSupplyMaterials {
		supplyMaterialRecord := &incomingSupplyMaterials[recordIndex]
		clientSentIDs[recordIndex] = supplyMaterialRecord.ID

		// Server is the source of truth — validate every record before persisting any.
		if len(supplyMaterialRecord.Name) < 2 {
			return req.MakeErr("El nombre del insumo debe tener al menos 2 caracteres.")
		}
		if supplyMaterialRecord.Price < 0 {
			return req.MakeErr("El precio no puede ser negativo.")
		}

		// Reuse the existing provider-row sanitization/validation from product-supply.
		supplyMaterialRecord.ProviderSupply = sanitizeProviderSupplyRows(supplyMaterialRecord.ProviderSupply)
		if validationError := validateProviderSupplyRows(req, supplyMaterialRecord.ProviderSupply); validationError != nil {
			return req.MakeErr(validationError)
		}

		supplyMaterialRecord.CompanyID = req.User.CompanyID
		supplyMaterialRecord.Updated = currentTimestamp
		supplyMaterialRecord.UpdatedBy = req.User.ID

		if supplyMaterialRecord.ID <= 0 {
			// New row → ORM auto-assigns the ID via the schema's Autoincrement(0).
			supplyMaterialRecord.ID = 0
			supplyMaterialRecord.Status = 1
			supplyMaterialRecord.Created = currentTimestamp
			supplyMaterialRecord.CreatedBy = req.User.ID
			toInsert = append(toInsert, *supplyMaterialRecord)
		} else {
			toUpdate = append(toUpdate, *supplyMaterialRecord)
		}
	}

	if len(toInsert) > 0 {
		if insertError := db.Insert(&toInsert); insertError != nil {
			core.Log("PostSupplyMaterial insert error:", insertError)
			return req.MakeErr("Error al guardar los insumos.", insertError)
		}
	}
	if len(toUpdate) > 0 {
		supplyMaterialTable := db.Table[logisticsTypes.SupplyMaterial]()
		// Protect immutable creation fields from being overwritten on update.
		if updateError := db.UpdateExclude(&toUpdate, supplyMaterialTable.Created, supplyMaterialTable.CreatedBy); updateError != nil {
			core.Log("PostSupplyMaterial update error:", updateError)
			return req.MakeErr("Error al actualizar los insumos.", updateError)
		}
	}

	// Rebuild the mapping list in the original input order so the frontend can
	// match by index — autoincrement only filled IDs in the insert slice.
	insertCursor := 0
	updateCursor := 0
	idMappings := make([]supplyMaterialIDMapping, len(incomingSupplyMaterials))
	for recordIndex, originalID := range clientSentIDs {
		if originalID <= 0 {
			idMappings[recordIndex] = supplyMaterialIDMapping{ID: toInsert[insertCursor].ID, TempID: originalID}
			insertCursor++
		} else {
			idMappings[recordIndex] = supplyMaterialIDMapping{ID: toUpdate[updateCursor].ID, TempID: originalID}
			updateCursor++
		}
	}

	core.Log("PostSupplyMaterial saved:", "inserted=", len(toInsert), "updated=", len(toUpdate))
	return req.MakeResponse(idMappings)
}
