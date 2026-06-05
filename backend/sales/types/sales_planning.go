package types

import (
	"app/db"
)

// SalesPlanningWeek is one week of the planned quantity for a product.
type SalesPlanningWeek struct {
	Week     int16 `json:",omitempty" cbor:"1,keyasint,omitempty"`
	Quantity int16 `json:",omitempty" cbor:"2,keyasint,omitempty"`
}

// SalesPlanning holds the sales projection assumptions/output for a product:
// a base weekly volume, an optional seasonality curve, and the resolved
// per-week planned quantities.
type SalesPlanning struct {
	db.TableStruct[SalesPlanningTable, SalesPlanning]
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
	db.TableStruct[SalesPlanningTable, SalesPlanning]
	CompanyID          db.Col[SalesPlanningTable, int32]
	ID                 db.Col[SalesPlanningTable, int32]
	ProductID          db.Col[SalesPlanningTable, int32]
	BaseQuantity       db.Col[SalesPlanningTable, int32]
	SeasonalityCurveID db.Col[SalesPlanningTable, int32]
	WeeklyQuantity     db.Col[SalesPlanningTable, []SalesPlanningWeek]
	Status             db.Col[SalesPlanningTable, int8]
	Updated            db.Col[SalesPlanningTable, int32]
	UpdatedBy          db.Col[SalesPlanningTable, int32]
	Created            db.Col[SalesPlanningTable, int32]
}

func (e SalesPlanningTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "sales_planning",
		Partition: e.CompanyID,
		Keys:      []db.Coln{e.ID.Autoincrement(0)},
		Indexes: []db.Index{
			{Type: db.TypeLocalIndex, Keys: []db.Coln{e.ProductID}},
			// Delta-cache view: queried with Status equality + Updated range.
			{Type: db.TypeView, Keys: []db.Coln{e.Status.Int32(), e.Updated.DecimalSize(8)}, KeepPart: true},
		},
	}
}

// SeasonalityCurveWeek is one week's multiplier within a seasonality curve.
// Percent is stored as the multiplier * 1000 (3 decimal digits as an integer):
// e.g. 0.500 -> 500, 1.000 -> 1000, 2.000 -> 2000.
type SeasonalityCurveWeek struct {
	Week    int16 `json:",omitempty" cbor:"1,keyasint,omitempty"`
	Percent int16 `json:",omitempty" cbor:"2,keyasint,omitempty"`
}

// SeasonalityCurve is a reusable per-week multiplier table that can be
// assigned to many products.
type SeasonalityCurve struct {
	db.TableStruct[SeasonalityCurveTable, SeasonalityCurve]
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
	db.TableStruct[SeasonalityCurveTable, SeasonalityCurve]
	CompanyID db.Col[SeasonalityCurveTable, int32]
	ID        db.Col[SeasonalityCurveTable, int32]
	Name      db.Col[SeasonalityCurveTable, string]
	Curve     db.Col[SeasonalityCurveTable, []SeasonalityCurveWeek]
	Status    db.Col[SeasonalityCurveTable, int8]
	Updated   db.Col[SeasonalityCurveTable, int32]
	UpdatedBy db.Col[SeasonalityCurveTable, int32]
	Created   db.Col[SeasonalityCurveTable, int32]
}

func (e SeasonalityCurveTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "seasonality_curve",
		Partition: e.CompanyID,
		Keys:      []db.Coln{e.ID.Autoincrement(0)},
		Indexes: []db.Index{
			// Delta-cache view: queried with Status equality + Updated range.
			{Type: db.TypeView, Keys: []db.Coln{e.Status.Int32(), e.Updated.DecimalSize(8)}, KeepPart: true},
		},
	}
}
