package types

import (
	"github.com/ivanjoz/genix-orm/scylla"
)

// SalesPlanningWeek is one week of the planned quantity for a product.
type SalesPlanningWeek struct {
	Week     int16 `json:",omitempty"`
	Quantity int16 `json:",omitempty"`
}

// SalesPlanning holds the sales projection assumptions/output for a product:
// a base weekly volume, an optional seasonality curve, and the resolved
// per-week planned quantities.
type SalesPlanning struct {
	scylla.TableStruct[SalesPlanningTable, SalesPlanning]
	CompanyID          int32               `json:",omitempty"`
	ID                 int32               `json:",omitempty"`
	TempID             int32               `json:",omitempty"`
	ProductID          int32               `json:",omitempty"`
	BaseQuantity       int32               `json:",omitempty"`
	SeasonalityCurveID int32               `json:",omitempty"`
	WeeklyQuantity     []SalesPlanningWeek `json:",omitempty"`
	Status             int8                `json:"ss,omitempty"`
	Updated            int32               `json:"upd,omitempty"`
	UpdatedBy          int32               `json:",omitempty"`
	Created            int32               `json:",omitempty"`
}

type SalesPlanningTable struct {
	scylla.TableStruct[SalesPlanningTable, SalesPlanning]
	CompanyID          scylla.Col[SalesPlanningTable, int32]
	ID                 scylla.Col[SalesPlanningTable, int32]
	ProductID          scylla.Col[SalesPlanningTable, int32]
	BaseQuantity       scylla.Col[SalesPlanningTable, int32]
	SeasonalityCurveID scylla.Col[SalesPlanningTable, int32]
	WeeklyQuantity     scylla.Col[SalesPlanningTable, []SalesPlanningWeek]
	Status             scylla.Col[SalesPlanningTable, int8]
	Updated            scylla.Col[SalesPlanningTable, int32]
	UpdatedBy          scylla.Col[SalesPlanningTable, int32]
	Created            scylla.Col[SalesPlanningTable, int32]
}

func (e SalesPlanningTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:      "sales_planning",
		Partition: e.CompanyID,
		Keys:      scylla.Cols(e.ID.Autoincrement(0)),
		Indexes: []scylla.Index{
			{Type: scylla.TypeLocalIndex, Keys: scylla.Cols(e.ProductID)},
			// Delta-cache view: queried with Status equality + Updated range.
			{Type: scylla.TypeView, Keys: scylla.Cols(e.Status.Int32(), e.Updated.DecimalSize(8))},
		},
	}
}

// SeasonalityCurveWeek is one week's multiplier within a seasonality curve.
// Percent is stored as the multiplier * 1000 (3 decimal digits as an integer):
// e.g. 0.500 -> 500, 1.000 -> 1000, 2.000 -> 2000.
type SeasonalityCurveWeek struct {
	Week    int16 `json:",omitempty"`
	Percent int16 `json:",omitempty"`
}

// SeasonalityCurve is a reusable per-week multiplier table that can be
// assigned to many products.
type SeasonalityCurve struct {
	scylla.TableStruct[SeasonalityCurveTable, SeasonalityCurve]
	CompanyID int32                  `json:",omitempty"`
	ID        int32                  `json:",omitempty"`
	TempID    int32                  `json:",omitempty"`
	Name      string                 `json:",omitempty"`
	Curve     []SeasonalityCurveWeek `json:",omitempty"`
	Status    int8                   `json:"ss,omitempty"`
	Updated   int32                  `json:"upd,omitempty"`
	UpdatedBy int32                  `json:",omitempty"`
	Created   int32                  `json:",omitempty"`
}

type SeasonalityCurveTable struct {
	scylla.TableStruct[SeasonalityCurveTable, SeasonalityCurve]
	CompanyID scylla.Col[SeasonalityCurveTable, int32]
	ID        scylla.Col[SeasonalityCurveTable, int32]
	Name      scylla.Col[SeasonalityCurveTable, string]
	Curve     scylla.Col[SeasonalityCurveTable, []SeasonalityCurveWeek]
	Status    scylla.Col[SeasonalityCurveTable, int8]
	Updated   scylla.Col[SeasonalityCurveTable, int32]
	UpdatedBy scylla.Col[SeasonalityCurveTable, int32]
	Created   scylla.Col[SeasonalityCurveTable, int32]
}

func (e SeasonalityCurveTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:      "seasonality_curve",
		Partition: e.CompanyID,
		Keys:      scylla.Cols(e.ID.Autoincrement(0)),
		Indexes: []scylla.Index{
			// Delta-cache view: queried with Status equality + Updated range.
			{Type: scylla.TypeView, Keys: scylla.Cols(e.Status.Int32(), e.Updated.DecimalSize(8))},
		},
	}
}
