package types

import "app/db"

const (
	PurchaseOrderStatusCanceled  int8 = 0
	PurchaseOrderStatusPending   int8 = 1
	PurchaseOrderStatusFulfilled int8 = 2
)

type PurchaseOrder struct {
	db.TableStruct[PurchaseOrderTable, PurchaseOrder]
	ID                    int32
	CompanyID             int32   `json:",omitempty"`
	ProviderID            int32   `json:",omitempty"`
	WarehouseID           int32   `json:",omitempty"`
	Date                  int16   `json:",omitempty"`
	Week                  int16   `json:",omitempty"`
	DateOfDelivery        int16   `json:",omitempty"`
	DetailProductIDs      []int32 `json:",omitempty"`
	DetailQuantities      []int32 `json:",omitempty"`
	DetailPrices          []int32 `json:",omitempty"`
	DetailPresentationIDs []int32 `json:",omitempty"`
	TotalAmount           int32   `json:",omitempty"`
	TaxAmount             int32   `json:",omitempty"`
	Notes                 string  `json:",omitempty"`

	Created   int32 `json:",omitempty"`
	CreatedBy int32 `json:",omitempty"`
	Updated   int32 `json:"upd,omitempty"`
	UpdatedBy int32 `json:",omitempty"`
	Status    int8  `json:"ss,omitempty"`
}

type PurchaseOrderTable struct {
	db.TableStruct[PurchaseOrderTable, PurchaseOrder]
	CompanyID             db.Col[PurchaseOrderTable, int32]
	ID                    db.Col[PurchaseOrderTable, int32]
	ProviderID            db.Col[PurchaseOrderTable, int32]
	WarehouseID           db.Col[PurchaseOrderTable, int32]
	Date                  db.Col[PurchaseOrderTable, int16]
	Week                  db.Col[PurchaseOrderTable, int16]
	DateOfDelivery        db.Col[PurchaseOrderTable, int16]
	DetailProductIDs      db.Col[PurchaseOrderTable, []int32]
	DetailQuantities      db.Col[PurchaseOrderTable, []int32]
	DetailPrices          db.Col[PurchaseOrderTable, []int32]
	DetailPresentationIDs db.Col[PurchaseOrderTable, []int32]
	TotalAmount           db.Col[PurchaseOrderTable, int32]
	TaxAmount             db.Col[PurchaseOrderTable, int32]
	Notes                 db.Col[PurchaseOrderTable, string]
	Created               db.Col[PurchaseOrderTable, int32]
	CreatedBy             db.Col[PurchaseOrderTable, int32]
	Updated               db.Col[PurchaseOrderTable, int32]
	UpdatedBy             db.Col[PurchaseOrderTable, int32]
	Status                db.Col[PurchaseOrderTable, int8]
}

func (e PurchaseOrderTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:                 "purchase_order",
		Partition:            e.CompanyID,
		Keys:                 []db.Coln{e.ID.Autoincrement(0)},
		Indexes: []db.Index{
			{
				Type:     db.TypeView,
				Keys:     []db.Coln{e.Status.Int32(), e.Updated.DecimalSize(8)},
				KeepPart: true,
			},			
			{
				Keys: []db.Coln{e.Week}, 
				UseIndexGroup: true,
			},
			{
				Keys: []db.Coln{e.Week, e.DetailProductIDs}, 
				UseIndexGroup: true,
			},
			{
				Keys: []db.Coln{e.Week, e.Status, e.DetailProductIDs}, 
				UseIndexGroup: true,
			},
			{
				Keys: []db.Coln{e.Week, e.ProviderID}, 
				UseIndexGroup: true,
			},
			{
				Keys: []db.Coln{e.Week, e.Status, e.ProviderID}, 
				UseIndexGroup: true,
			},
		},
	}
}
