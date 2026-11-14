package types

import "app/db"

// ProductionClose is one run of the daily production close (docs/PRODUCTION_PLAN.md §5.6): what it
// produced in WarehouseID for the sales delivered there on Date, and what it consumed from
// SuppliesWarehouseID. Its ID is the DocumentID of its stock movements (types 15 and 16). A day can
// be closed more than once: each run settles the sales pending when it ran.
type ProductionClose struct {
	db.TableStruct[ProductionCloseTable, ProductionClose]
	CompanyID           int32 `cb:"1"`
	ID                  int32 `cb:"2"`
	WarehouseID         int32 `cb:"3"`
	SuppliesWarehouseID int32 `cb:"4"`
	// Date is the day closed (UnixDay). Its movements are dated max(Date, yesterday).
	Date int16 `cb:"5"`
	// IsStrict is company flag 15 when it ran: a short supply blocked it instead of being shared.
	IsStrict     bool                     `cb:"6"`
	Products     []ProductionCloseProduct `cb:"7"`
	Supplies     []ProductionCloseSupply  `cb:"8"`
	SaleOrderIDs []int64                  `cb:"9"`
	Status       int8                     `json:"ss" cb:"ss,10"`
	Created      int32                    `cb:"11"`
	CreatedBy    int32                    `cb:"12"`
	Updated      int32                    `json:"upd" cb:"upd,13"`
	Version      int16                    `json:"ver" cb:"ver,14"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata *string `json:"mdt" cb:"mdt,15"`
}

type ProductionCloseTable struct {
	db.TableStruct[ProductionCloseTable, ProductionClose]
	CompanyID           db.Col[*ProductionCloseTable, int32]
	ID                  db.Col[*ProductionCloseTable, int32]
	WarehouseID         db.Col[*ProductionCloseTable, int32]
	SuppliesWarehouseID db.Col[*ProductionCloseTable, int32]
	Date                db.Col[*ProductionCloseTable, int16]
	IsStrict            db.Col[*ProductionCloseTable, bool]
	Products            db.Col[*ProductionCloseTable, []ProductionCloseProduct]
	Supplies            db.Col[*ProductionCloseTable, []ProductionCloseSupply]
	SaleOrderIDs        db.ColSlice[*ProductionCloseTable, int64]
	Status              db.Col[*ProductionCloseTable, int8]
	Created             db.Col[*ProductionCloseTable, int32]
	CreatedBy           db.Col[*ProductionCloseTable, int32]
	Updated             db.Col[*ProductionCloseTable, int32]
	Version             db.Col[*ProductionCloseTable, int16]
}

func (e ProductionCloseTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        96,
		Name:      "production_close",
		Partition: e.CompanyID,
		Keys:      db.Cols(e.ID.Autoincrement(0)),
		Indexes: []db.Index{
			// The history of a warehouse's closes, read with the WarehouseID pinned by an equality.
			{Type: db.TypeDelta, Keys: db.Cols(e.WarehouseID)},
		},
	}
}
