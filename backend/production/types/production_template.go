package types

import (
	"app/core"
	"app/db"
	"fmt"
	"strings"
)

// ProductionTemplate is a reusable list of products in their size runs: "Zapatos Verano"
// (docs/MANUFACTURING_PLAN.md §4, M1, M2, M16). An order is created from it for a number of series,
// and from then on is independent of it.
type ProductionTemplate struct {
	db.TableStruct[ProductionTemplateTable, ProductionTemplate]
	CompanyID int32                    `cb:"1"`
	ID        int32                    `cb:"2"`
	Name      string                   `cb:"3"`
	Lines     []ProductionTemplateLine `cb:"4"`
	// DefaultSeries prefills a new order. The warehouses are copied to it too.
	DefaultSeries       int32  `cb:"5"`
	WarehouseID         int32  `cb:"6"`
	SuppliesWarehouseID int32  `cb:"7"`
	Notes               string `cb:"8"`
	Status              int8   `json:"ss" cb:"ss,9"`
	Created             int32  `cb:"10"`
	CreatedBy           int32  `cb:"11"`
	Updated             int32  `json:"upd" cb:"upd,12"`
	UpdatedBy           int32  `cb:"13"`
	Version             int16  `json:"ver" cb:"ver,14"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata *string `json:"mdt" cb:"mdt,15"`
}

type ProductionTemplateTable struct {
	db.TableStruct[ProductionTemplateTable, ProductionTemplate]
	CompanyID           db.Col[*ProductionTemplateTable, int32]
	ID                  db.Col[*ProductionTemplateTable, int32]
	Name                db.Col[*ProductionTemplateTable, string]
	Lines               db.Col[*ProductionTemplateTable, []ProductionTemplateLine]
	DefaultSeries       db.Col[*ProductionTemplateTable, int32]
	WarehouseID         db.Col[*ProductionTemplateTable, int32]
	SuppliesWarehouseID db.Col[*ProductionTemplateTable, int32]
	Notes               db.Col[*ProductionTemplateTable, string]
	Status              db.Col[*ProductionTemplateTable, int8]
	Created             db.Col[*ProductionTemplateTable, int32]
	CreatedBy           db.Col[*ProductionTemplateTable, int32]
	Updated             db.Col[*ProductionTemplateTable, int32]
	UpdatedBy           db.Col[*ProductionTemplateTable, int32]
	Version             db.Col[*ProductionTemplateTable, int16]
}

func (e ProductionTemplateTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:          102,
		Name:        "production_template",
		Partition:   e.CompanyID,
		Keys:        db.Cols(e.ID.Autoincrement(0)),
		FixedValues: []db.FixedValues{{Col: e.Status, Min: 0, Max: 1}},
		Indexes: []db.Index{
			// The company's templates, delta-synced by the templates and orders pages.
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
		},
	}
}

// ProductionTemplateLine is one product in its size run: how many units of each presentation one
// series holds. 36×1, 37×2, 38×3, 39×3, 40×2, 41×1 is a series of 12 pairs.
type ProductionTemplateLine struct {
	ProductID int32                  `cb:"1"`
	Run       []PresentationQuantity `cb:"2"`
}

// PresentationQuantity is the units of one presentation (a size) in a series.
type PresentationQuantity struct {
	PresentationID int16 `cb:"1"`
	Units          int32 `cb:"2"`
}

// ValidateProductionTemplate refuses a template without a name or with a line that could not become
// an order line: the same checks as an order's lines, for one series.
func ValidateProductionTemplate(template ProductionTemplate, productsByID map[int32]Product) error {
	if len(strings.TrimSpace(template.Name)) < 2 {
		return core.Err("El nombre de la plantilla debe tener al menos 2 caracteres.")
	}
	if template.DefaultSeries < 0 {
		return core.Err("Las series por defecto no pueden ser negativas.")
	}
	seriesLines := []ProductionOrderLine{}
	seenProductIDs := map[int32]bool{}
	for _, templateLine := range template.Lines {
		if seenProductIDs[templateLine.ProductID] {
			return core.Err(fmt.Sprintf("El producto %v está repetido en la plantilla.", templateLine.ProductID))
		}
		seenProductIDs[templateLine.ProductID] = true
		if len(templateLine.Run) == 0 {
			return core.Err(fmt.Sprintf("El producto %v de la plantilla no tiene unidades.", templateLine.ProductID))
		}
		for _, sizeUnits := range templateLine.Run {
			seriesLines = append(seriesLines, ProductionOrderLine{
				ProductID: templateLine.ProductID, PresentationID: sizeUnits.PresentationID, PlannedUnits: sizeUnits.Units,
			})
		}
	}
	return ValidateProductionOrderLines(seriesLines, productsByID)
}
