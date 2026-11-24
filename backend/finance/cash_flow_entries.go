package finance

import (
	business "app/business/types"
	"app/core"
	"app/db"
	"app/finance/types"
	"encoding/json"
)

// GetCashFlowEntries syncs the company's planned cash flow entries.
func GetCashFlowEntries(req *core.HandlerArgs) core.HandlerResponse {
	entries := []types.CashFlowEntry{}
	query := db.Query(&entries)
	query.Select().CompanyID.Equals(req.User.CompanyID).Delta(req.GetDeltaCursor(), 1)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener las entradas planificadas:", err)
	}
	return core.MakeResponse(req, &entries)
}

// PostCashFlowEntries creates, edits or (ss = 0) removes one planned entry.
func PostCashFlowEntries(req *core.HandlerArgs) core.HandlerResponse {
	entry := types.CashFlowEntry{}
	if err := json.Unmarshal([]byte(*req.Body), &entry); err != nil {
		return req.MakeErr("Error al deserializar la entrada planificada:", err)
	}
	entry.CompanyID = req.User.CompanyID
	entry.Updated, entry.UpdatedBy = core.SUnixTime(), req.User.ID

	if entry.Status != 0 {
		entry.Status = 1
		if err := types.ValidateCashFlowEntry(&entry); err != nil {
			return req.MakeErr(err)
		}
		if err := business.ValidateSiteID(entry.CompanyID, entry.SiteID); err != nil {
			return req.MakeErr(err)
		}
	} else if entry.ID <= 0 {
		return req.MakeErr("No se puede eliminar una entrada que no se ha guardado.")
	}

	entries := []types.CashFlowEntry{entry}
	if entry.ID <= 0 {
		entries[0].Created, entries[0].CreatedBy = entry.Updated, req.User.ID
		if err := db.Insert(&entries); err != nil {
			return req.MakeErr("Error al registrar la entrada planificada:", err)
		}
		return req.MakeResponse(entries[0])
	}

	// An update must hit an entry of the company: the client's ID is not trusted.
	storedEntries := []types.CashFlowEntry{}
	storedQuery := db.Query(&storedEntries)
	storedQuery.Select(storedQuery.ID).CompanyID.Equals(entry.CompanyID).ID.Equals(entry.ID)
	if err := storedQuery.Exec(); err != nil {
		return req.MakeErr("Error al obtener la entrada planificada:", err)
	}
	if len(storedEntries) == 0 {
		return req.MakeErr("La entrada planificada no existe.")
	}
	table := db.TableOf[types.CashFlowEntry]()
	columnsToUpdate := db.Cols(table.Status, table.Updated, table.UpdatedBy)
	// A delete only changes the status.
	if entry.Status != 0 {
		columnsToUpdate = append(columnsToUpdate, table.Date, table.SiteID, table.Amount, table.CurrencyType, table.Kind, table.Name)
	}
	if err := db.Update(&entries, columnsToUpdate...); err != nil {
		return req.MakeErr("Error al guardar la entrada planificada:", err)
	}
	return req.MakeResponse(entries[0])
}
