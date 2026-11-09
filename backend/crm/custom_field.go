package crm

import (
	"encoding/json"
	"fmt"

	"app/core"
	"app/crm/types"
	"app/db"
)

func GetCrmCustomFields(req *core.HandlerArgs) core.HandlerResponse {
	customFields := []types.CrmCustomField{}
	query := db.Query(&customFields)
	query.Select().CompanyID.Equals(req.User.CompanyID).Delta(req.GetDeltaCursor(), types.CrmStatusActive)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los campos personalizados:", err)
	}
	return req.MakeResponse(customFields)
}

// PostCrmCustomField creates or edits a catalog field; Status 0 deletes it. Deleting is soft, so
// values already stored on opportunities keep their label.
func PostCrmCustomField(req *core.HandlerArgs) core.HandlerResponse {
	payload := types.CrmCustomField{}
	if err := json.Unmarshal([]byte(*req.Body), &payload); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	companyID := req.User.CompanyID

	saveError := db.RunTransaction(func(transaction *db.Transaction) error {
		stored := types.CrmCustomField{}
		hasStoredValues := false
		if payload.ID != 0 {
			storedRecords := []types.CrmCustomField{{CompanyID: companyID, ID: payload.ID}}
			if err := db.LockByKeys(transaction, &storedRecords); err != nil {
				return err
			}
			if len(storedRecords) == 0 {
				return fmt.Errorf("El campo %d no existe.", payload.ID)
			}
			stored = storedRecords[0]
			if stored.Type != payload.Type {
				var err error
				if hasStoredValues, err = anyOpportunityHoldsFieldValue(companyID, stored.ID); err != nil {
					return err
				}
			}
		}

		saved, err := types.ApplyCustomFieldEdit(stored, payload, hasStoredValues)
		if err != nil {
			return err
		}
		saved.CompanyID, saved.UpdatedBy = companyID, req.User.ID
		records := []types.CrmCustomField{saved}
		if saved.ID == 0 {
			err = db.InsertIn(transaction, &records)
		} else {
			table := db.TableOf[types.CrmCustomField]()
			err = db.UpdateIn(transaction, &records, table.Name, table.Type, table.Options, table.Status, table.UpdatedBy)
		}
		payload = records[0]
		return err
	})
	if saveError != nil {
		return req.MakeErr("Error al guardar el campo personalizado:", saveError)
	}
	return req.MakeResponse(payload)
}

// loadActiveCustomFieldIDs returns the IDs of the company's active custom fields.
func loadActiveCustomFieldIDs(companyID int32) (map[int16]bool, error) {
	customFields := []types.CrmCustomField{}
	query := db.Query(&customFields)
	query.Select(query.ID, query.Status).CompanyID.Equals(companyID)
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer los campos personalizados: %v", err)
	}
	activeFieldIDs := map[int16]bool{}
	for _, customField := range customFields {
		if customField.Status == types.CrmStatusActive {
			activeFieldIDs[customField.ID] = true
		}
	}
	return activeFieldIDs, nil
}

// anyOpportunityHoldsFieldValue scans the company's opportunities; it only runs when an edit
// tries to change a field's type, which is rare.
func anyOpportunityHoldsFieldValue(companyID int32, fieldID int16) (bool, error) {
	opportunities := []types.CrmOpportunity{}
	query := db.Query(&opportunities)
	query.Select(query.ID, query.CustomValues).CompanyID.Equals(companyID)
	if err := query.Exec(); err != nil {
		return false, fmt.Errorf("error al leer las oportunidades: %v", err)
	}
	for _, opportunity := range opportunities {
		for _, value := range opportunity.CustomValues {
			if value.FieldID == fieldID {
				return true, nil
			}
		}
	}
	return false, nil
}
