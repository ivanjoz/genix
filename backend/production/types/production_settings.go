package types

import (
	"app/core"
	"app/db"
	"fmt"
	"strings"
)

// ProductionLaborType is a kind of work a recipe and a production order plan in minutes:
// "Zapatero", "Maquinista" (docs/MANUFACTURING_PLAN.md §9.1). CostPerHour only estimates: the
// recipe's cost, a template's and a release's. The actual labor value comes from the hours logged.
type ProductionLaborType struct {
	db.TableStruct[ProductionLaborTypeTable, ProductionLaborType]
	CompanyID int32  `cb:"1"`
	ID        int32  `cb:"2"`
	Name      string `cb:"3"`
	// CostPerHour is PEN cents. It is also the rate of a worker off payroll who has no agreement (M29).
	CostPerHour int32 `cb:"4"`
	Status      int8  `json:"ss" cb:"ss,5"`
	Created     int32 `cb:"6"`
	CreatedBy   int32 `cb:"7"`
	Updated     int32 `json:"upd" cb:"upd,8"`
	UpdatedBy   int32 `cb:"9"`
	Version     int16 `json:"ver" cb:"ver,10"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata *string `json:"mdt" cb:"mdt,11"`
}

type ProductionLaborTypeTable struct {
	db.TableStruct[ProductionLaborTypeTable, ProductionLaborType]
	CompanyID   db.Col[*ProductionLaborTypeTable, int32]
	ID          db.Col[*ProductionLaborTypeTable, int32]
	Name        db.Col[*ProductionLaborTypeTable, string]
	CostPerHour db.Col[*ProductionLaborTypeTable, int32]
	Status      db.Col[*ProductionLaborTypeTable, int8]
	Created     db.Col[*ProductionLaborTypeTable, int32]
	CreatedBy   db.Col[*ProductionLaborTypeTable, int32]
	Updated     db.Col[*ProductionLaborTypeTable, int32]
	UpdatedBy   db.Col[*ProductionLaborTypeTable, int32]
	Version     db.Col[*ProductionLaborTypeTable, int16]
}

func (e ProductionLaborTypeTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:          99,
		Name:        "production_labor_type",
		Partition:   e.CompanyID,
		Keys:        db.Cols(e.ID.Autoincrement(0)),
		FixedValues: []db.FixedValues{{Col: e.Status, Min: 0, Max: 1}},
		Indexes: []db.Index{
			// The company's labor types, delta-synced by the settings page and the recipe form.
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
		},
	}
}

// ProductionStage is a column of the production orders board: "Corte", "Aparado", "Armado"
// (docs/MANUFACTURING_PLAN.md §9.4). It is information only: it moves no stock and changes no cost.
type ProductionStage struct {
	db.TableStruct[ProductionStageTable, ProductionStage]
	CompanyID int32  `cb:"1"`
	ID        int32  `cb:"2"`
	Name      string `cb:"3"`
	// Position orders the board's columns, left to right.
	Position  int16 `cb:"4"`
	Status    int8  `json:"ss" cb:"ss,5"`
	Created   int32 `cb:"6"`
	CreatedBy int32 `cb:"7"`
	Updated   int32 `json:"upd" cb:"upd,8"`
	UpdatedBy int32 `cb:"9"`
	Version   int16 `json:"ver" cb:"ver,10"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata *string `json:"mdt" cb:"mdt,11"`
}

type ProductionStageTable struct {
	db.TableStruct[ProductionStageTable, ProductionStage]
	CompanyID db.Col[*ProductionStageTable, int32]
	ID        db.Col[*ProductionStageTable, int32]
	Name      db.Col[*ProductionStageTable, string]
	Position  db.Col[*ProductionStageTable, int16]
	Status    db.Col[*ProductionStageTable, int8]
	Created   db.Col[*ProductionStageTable, int32]
	CreatedBy db.Col[*ProductionStageTable, int32]
	Updated   db.Col[*ProductionStageTable, int32]
	UpdatedBy db.Col[*ProductionStageTable, int32]
	Version   db.Col[*ProductionStageTable, int16]
}

func (e ProductionStageTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:          100,
		Name:        "production_stage",
		Partition:   e.CompanyID,
		Keys:        db.Cols(e.ID.Autoincrement(0)),
		FixedValues: []db.FixedValues{{Col: e.Status, Min: 0, Max: 1}},
		Indexes: []db.Index{
			// The company's stages, delta-synced by the settings page and the board.
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
		},
	}
}

// ValidateProductionLaborType refuses a labor type without a name or with a negative rate. A rate
// of 0 is allowed: the type then plans hours without estimating their cost.
func ValidateProductionLaborType(laborType ProductionLaborType) error {
	if len(strings.TrimSpace(laborType.Name)) < 2 {
		return core.Err("El nombre del tipo de trabajo debe tener al menos 2 caracteres.")
	}
	if laborType.CostPerHour < 0 {
		return core.Err(fmt.Sprintf(`Tipo de trabajo "%s": el costo por hora no puede ser negativo.`, laborType.Name))
	}
	return nil
}

// ValidateProductionStage refuses a stage without a name or with a negative position.
func ValidateProductionStage(stage ProductionStage) error {
	if len(strings.TrimSpace(stage.Name)) < 2 {
		return core.Err("El nombre de la etapa debe tener al menos 2 caracteres.")
	}
	if stage.Position < 0 {
		return core.Err(fmt.Sprintf(`Etapa "%s": la posición no puede ser negativa.`, stage.Name))
	}
	return nil
}
