package types

import "app/db"

type PurchaseOrder struct {
	db.TableStruct[PurchaseOrderTable, PurchaseOrder]
	ID                        int32
	CompanyID                 int32 `json:",omitempty"`
  Date         							int16 `json:",omitempty"`
  Week         							int16 `json:",omitempty"`
	DetailProductIDs          []int32 `json:",omitempty"`
	DetailQuantities          []int32 `json:",omitempty"`
	DetailPrices              []int32 `json:",omitempty"`
	DetailPresentationIDs     []int32 `json:",omitempty"`

	Created   int32 `json:",omitempty"`
	CreatedBy int32 `json:",omitempty"`
	Updated   int32 `json:"upd,omitempty"`
	UpdatedBy int32 `json:",omitempty"`
	Status    int8  `json:"ss,omitempty"`
}

type PurchaseOrderTable struct {
	db.TableStruct[PurchaseOrderTable, PurchaseOrder]
	CompanyID            db.Col[PurchaseOrderTable, int32]
	ID                   db.Col[PurchaseOrderTable, int32]
	Week                   db.Col[PurchaseOrderTable, int16]
	DetailProductIDs db.Col[PurchaseOrderTable, []int32]
}

func (e PurchaseOrderTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:                 "purchase_order",
		Partition:            e.CompanyID,
		Keys:                 []db.Coln{e.ID},
		Indexes: []db.Index{
			{
				Keys: []db.Coln{  e.Week, e.DetailProductIDs },
				UseIndexGroup: true,
			},
		},
	}
}
