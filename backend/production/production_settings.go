package production

import (
	"app/core"
	"app/db"
	"app/production/types"
	"encoding/json"
	"strings"
)

// GetProductionLaborTypes is the delta of the company's labor types (docs/MANUFACTURING_PLAN.md
// §9.1); a deleted one arrives with ss = 0.
func GetProductionLaborTypes(req *core.HandlerArgs) core.HandlerResponse {
	laborTypes := []types.ProductionLaborType{}
	query := db.Query(&laborTypes)
	query.Select().CompanyID.Equals(req.User.CompanyID).Delta(req.GetDeltaCursor(), int8(1))
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los tipos de trabajo:", err)
	}
	return req.MakeResponse(laborTypes)
}

// PostProductionLaborType creates or updates one labor type. ss = 0 deletes it: the recipes that
// name it keep their lines, and refuse to save again until the line is removed.
func PostProductionLaborType(req *core.HandlerArgs) core.HandlerResponse {
	laborType := types.ProductionLaborType{}
	if err := json.Unmarshal([]byte(*req.Body), &laborType); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	laborType.Name = strings.TrimSpace(laborType.Name)
	if laborType.Status != 0 {
		laborType.Status = 1
		if err := types.ValidateProductionLaborType(laborType); err != nil {
			return req.MakeErr(err)
		}
	}
	laborType.CompanyID, laborType.Updated, laborType.UpdatedBy = req.User.CompanyID, core.SUnixTime(), req.User.ID

	records := []types.ProductionLaborType{laborType}
	if laborType.ID <= 0 {
		if laborType.Status == 0 {
			return req.MakeErr("No se puede eliminar un tipo de trabajo que no se ha guardado.")
		}
		records[0].ID, records[0].Created, records[0].CreatedBy = 0, laborType.Updated, req.User.ID
		if err := db.Insert(&records); err != nil {
			return req.MakeErr("Error al guardar el tipo de trabajo:", err)
		}
		return req.MakeResponse(records[0])
	}

	stored := []types.ProductionLaborType{}
	query := db.Query(&stored)
	query.Select(query.ID).CompanyID.Equals(req.User.CompanyID).ID.Equals(laborType.ID)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener el tipo de trabajo:", err)
	}
	if len(stored) == 0 {
		return req.MakeErr("El tipo de trabajo no existe.")
	}
	table := db.TableOf[types.ProductionLaborType]()
	if err := db.Update(&records, table.Name, table.CostPerHour, table.Status, table.Updated, table.UpdatedBy); err != nil {
		return req.MakeErr("Error al guardar el tipo de trabajo:", err)
	}
	return req.MakeResponse(records[0])
}

// GetProductionStages is the delta of the company's board stages (docs/MANUFACTURING_PLAN.md §9.4).
func GetProductionStages(req *core.HandlerArgs) core.HandlerResponse {
	stages := []types.ProductionStage{}
	query := db.Query(&stages)
	query.Select().CompanyID.Equals(req.User.CompanyID).Delta(req.GetDeltaCursor(), int8(1))
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener las etapas de producción:", err)
	}
	return req.MakeResponse(stages)
}

// PostProductionStage creates or updates one stage. ss = 0 deletes it: an order on it shows in no
// column until it is moved.
func PostProductionStage(req *core.HandlerArgs) core.HandlerResponse {
	stage := types.ProductionStage{}
	if err := json.Unmarshal([]byte(*req.Body), &stage); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	stage.Name = strings.TrimSpace(stage.Name)
	if stage.Status != 0 {
		stage.Status = 1
		if err := types.ValidateProductionStage(stage); err != nil {
			return req.MakeErr(err)
		}
	}
	stage.CompanyID, stage.Updated, stage.UpdatedBy = req.User.CompanyID, core.SUnixTime(), req.User.ID

	records := []types.ProductionStage{stage}
	if stage.ID <= 0 {
		if stage.Status == 0 {
			return req.MakeErr("No se puede eliminar una etapa que no se ha guardado.")
		}
		records[0].ID, records[0].Created, records[0].CreatedBy = 0, stage.Updated, req.User.ID
		if err := db.Insert(&records); err != nil {
			return req.MakeErr("Error al guardar la etapa:", err)
		}
		return req.MakeResponse(records[0])
	}

	stored := []types.ProductionStage{}
	query := db.Query(&stored)
	query.Select(query.ID).CompanyID.Equals(req.User.CompanyID).ID.Equals(stage.ID)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener la etapa:", err)
	}
	if len(stored) == 0 {
		return req.MakeErr("La etapa no existe.")
	}
	table := db.TableOf[types.ProductionStage]()
	if err := db.Update(&records, table.Name, table.Position, table.Status, table.Updated, table.UpdatedBy); err != nil {
		return req.MakeErr("Error al guardar la etapa:", err)
	}
	return req.MakeResponse(records[0])
}
