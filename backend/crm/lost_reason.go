package crm

import (
	"encoding/json"
	"fmt"
	"strings"

	"app/core"
	"app/crm/types"
	"app/db"
)

func GetCrmLostReasons(req *core.HandlerArgs) core.HandlerResponse {
	lostReasons := []types.CrmLostReason{}
	query := db.Query(&lostReasons)
	query.Select().CompanyID.Equals(req.User.CompanyID).Delta(req.GetDeltaCursor(), types.CrmStatusActive)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los motivos de pérdida:", err)
	}
	return req.MakeResponse(lostReasons)
}

// PostCrmLostReason creates or renames a lost reason; Status 0 deletes it (soft, closed
// opportunities keep pointing at it).
func PostCrmLostReason(req *core.HandlerArgs) core.HandlerResponse {
	payload := types.CrmLostReason{}
	if err := json.Unmarshal([]byte(*req.Body), &payload); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	payload.Name = strings.TrimSpace(payload.Name)
	if payload.Name == "" {
		return req.MakeErr("El nombre del motivo es obligatorio.")
	}
	payload.CompanyID, payload.UpdatedBy = req.User.CompanyID, req.User.ID
	// Status 0 deletes an existing reason; a new one is always active.
	if payload.ID == 0 || payload.Status != types.CrmStatusDeleted {
		payload.Status = types.CrmStatusActive
	}

	records := []types.CrmLostReason{payload}
	var saveError error
	if payload.ID == 0 {
		saveError = db.Insert(&records)
	} else {
		storedRecords := []types.CrmLostReason{}
		query := db.Query(&storedRecords)
		query.CompanyID.Equals(payload.CompanyID).ID.Equals(payload.ID)
		if err := query.Exec(); err != nil || len(storedRecords) == 0 {
			return req.MakeErr(fmt.Sprintf("El motivo %d no existe.", payload.ID), err)
		}
		table := db.TableOf[types.CrmLostReason]()
		saveError = db.Update(&records, table.Name, table.Status, table.UpdatedBy)
	}
	if saveError != nil {
		return req.MakeErr("Error al guardar el motivo de pérdida:", saveError)
	}
	return req.MakeResponse(records[0])
}
