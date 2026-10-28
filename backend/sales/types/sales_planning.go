package types

import (
	"app/db"
)

// SalesPlanningWeek is one week of the planned quantity for a product.
type SalesPlanningWeek struct {
	Week     int16 `cb:"Week,1"`
	Quantity int16 `cb:"Quantity,2"`
}

// SalesPlanning holds the sales projection assumptions/output for a product:
// a base weekly volume, an optional seasonality curve, and the resolved
// per-week planned quantities.
type SalesPlanning struct {
	db.TableStruct[SalesPlanningTable, SalesPlanning]
	CompanyID          int32               `cb:"CompanyID,1"`
	ID                 int32               `cb:"ID,2"`
	TempID             int32               `cb:"TempID,3"`
	ProductID          int32               `cb:"ProductID,4"`
	BaseQuantity       int32               `cb:"BaseQuantity,5"`
	SeasonalityCurveID int32               `cb:"SeasonalityCurveID,6"`
	WeeklyQuantity     []SalesPlanningWeek `cb:"WeeklyQuantity,7"`
	Status             int8                `json:"ss" cb:"ss,8"`
	Updated            int32               `json:"upd" cb:"upd,9"`
	Version            int16               `json:"ver" cb:"ver,10"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata  *string `json:"mdt" cb:"mdt,11"`
	UpdatedBy int32   `cb:"UpdatedBy,12"`
	Created   int32   `cb:"Created,13"`
}

type SalesPlanningTable struct {
	db.TableStruct[SalesPlanningTable, SalesPlanning]
	CompanyID          db.Col[*SalesPlanningTable, int32]
	ID                 db.Col[*SalesPlanningTable, int32]
	ProductID          db.Col[*SalesPlanningTable, int32]
	BaseQuantity       db.Col[*SalesPlanningTable, int32]
	SeasonalityCurveID db.Col[*SalesPlanningTable, int32]
	WeeklyQuantity     db.Col[*SalesPlanningTable, []SalesPlanningWeek]
	Status             db.Col[*SalesPlanningTable, int8]
	Updated            db.Col[*SalesPlanningTable, int32]
	Version            db.Col[*SalesPlanningTable, int16]
	UpdatedBy          db.Col[*SalesPlanningTable, int32]
	Created            db.Col[*SalesPlanningTable, int32]
}

func (e SalesPlanningTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        27,
		Name:      "sales_planning",
		Partition: e.CompanyID,
		Keys:      db.Cols(e.ID.Autoincrement(0)),
		// Delta() enumerates its filter column, so every Status value must be declared.
		FixedValues: []db.FixedValues{
			{Col: e.Status, Values: []int64{0, 1}},
		},
		Indexes: []db.Index{
			{Type: db.TypeLocalIndex, Keys: db.Cols(e.ProductID)},
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
		},
	}
}

// SeasonalityCurveWeek is one week's multiplier within a seasonality curve.
// Percent is stored as the multiplier * 1000 (3 decimal digits as an integer):
// e.g. 0.500 -> 500, 1.000 -> 1000, 2.000 -> 2000.
type SeasonalityCurveWeek struct {
	Week    int16 `cb:"Week,1"`
	Percent int16 `cb:"Percent,2"`
}

// SeasonalityCurve is a reusable per-week multiplier table that can be
// assigned to many products.
type SeasonalityCurve struct {
	db.TableStruct[SeasonalityCurveTable, SeasonalityCurve]
	CompanyID int32                  `cb:"CompanyID,1"`
	ID        int32                  `cb:"ID,2"`
	TempID    int32                  `cb:"TempID,3"`
	Name      string                 `cb:"Name,4"`
	Curve     []SeasonalityCurveWeek `cb:"Curve,5"`
	Status    int8                   `json:"ss" cb:"ss,6"`
	Updated   int32                  `json:"upd" cb:"upd,7"`
	Version   int16                  `json:"ver" cb:"ver,8"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata  *string `json:"mdt" cb:"mdt,9"`
	UpdatedBy int32   `cb:"UpdatedBy,10"`
	Created   int32   `cb:"Created,11"`
}

type SeasonalityCurveTable struct {
	db.TableStruct[SeasonalityCurveTable, SeasonalityCurve]
	CompanyID db.Col[*SeasonalityCurveTable, int32]
	ID        db.Col[*SeasonalityCurveTable, int32]
	Name      db.Col[*SeasonalityCurveTable, string]
	Curve     db.Col[*SeasonalityCurveTable, []SeasonalityCurveWeek]
	Status    db.Col[*SeasonalityCurveTable, int8]
	Updated   db.Col[*SeasonalityCurveTable, int32]
	Version   db.Col[*SeasonalityCurveTable, int16]
	UpdatedBy db.Col[*SeasonalityCurveTable, int32]
	Created   db.Col[*SeasonalityCurveTable, int32]
}

func (e SeasonalityCurveTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        28,
		Name:      "seasonality_curve",
		Partition: e.CompanyID,
		Keys:      db.Cols(e.ID.Autoincrement(0)),
		// Delta() enumerates its filter column, so every Status value must be declared.
		FixedValues: []db.FixedValues{
			{Col: e.Status, Values: []int64{0, 1}},
		},
		Indexes: []db.Index{
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
		},
	}
}
