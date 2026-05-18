package types

import "app/db"

const (
	PurchaseOrderStatusCanceled  int8 = 0
	PurchaseOrderStatusPending   int8 = 1
	PurchaseOrderStatusConfirmed int8 = 2
	PurchaseOrderStatusFulfilled int8 = 4
)

type PurchaseOrder struct {
	db.TableStruct[PurchaseOrderTable, PurchaseOrder]
	ID                    int32
	CompanyID             int32   `json:",omitempty"`
	ProviderID            int32   `json:",omitempty"`
	WarehouseID           int32   `json:",omitempty"`
	Date                  int16   `json:",omitempty"`
	Week                  int16   `json:",omitempty"`
	DeliveryDate          int16   `json:",omitempty"`
	PaymentDate           int16   `json:",omitempty"`
	// Producto: parallel arrays in the same order — one row per product line.
	DetailProductIDs             []int32 `json:",omitempty"`
	DetailProductQuantity        []int32 `json:",omitempty"`
	DetailProductPrice           []int32 `json:",omitempty"`
	DetailProductPresentationIDs []int32 `json:",omitempty"`
	// Insumo (supply_material): parallel arrays in the same order — one row per supply line.
	// Independiente de la lista de productos: una orden puede mezclar ambas.
	DetailSupplyIDs      []int32 `json:",omitempty"`
	DetailSupplyQuantity []int32 `json:",omitempty"`
	DetailSupplyPrice    []int32 `json:",omitempty"`
	TotalAmount           int32   `json:",omitempty"`
	TaxAmount             int32   `json:",omitempty"`
	DebtAmount            int32   `json:",omitempty"`
	DifferenceQuantity    int32   `json:",omitempty"`
	DifferenceValue       int32   `json:",omitempty"`
	InvoiceNumber         string  `json:",omitempty"`
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
	DeliveryDate          db.Col[PurchaseOrderTable, int16]
	PaymentDate           db.Col[PurchaseOrderTable, int16]
	DetailProductIDs             db.Col[PurchaseOrderTable, []int32]
	DetailProductQuantity        db.Col[PurchaseOrderTable, []int32]
	DetailProductPrice           db.Col[PurchaseOrderTable, []int32]
	DetailProductPresentationIDs db.Col[PurchaseOrderTable, []int32]
	DetailSupplyIDs              db.Col[PurchaseOrderTable, []int32]
	DetailSupplyQuantity         db.Col[PurchaseOrderTable, []int32]
	DetailSupplyPrice            db.Col[PurchaseOrderTable, []int32]
	TotalAmount           db.Col[PurchaseOrderTable, int32]
	TaxAmount             db.Col[PurchaseOrderTable, int32]
	DebtAmount            db.Col[PurchaseOrderTable, int32]
	DifferenceQuantity    db.Col[PurchaseOrderTable, int32]
	DifferenceValue       db.Col[PurchaseOrderTable, int32]
	InvoiceNumber         db.Col[PurchaseOrderTable, string]
	Notes                 db.Col[PurchaseOrderTable, string]
	Created               db.Col[PurchaseOrderTable, int32]
	CreatedBy             db.Col[PurchaseOrderTable, int32]
	Updated               db.Col[PurchaseOrderTable, int32]
	UpdatedBy             db.Col[PurchaseOrderTable, int32]
	Status                db.Col[PurchaseOrderTable, int8]
}

func (e PurchaseOrderTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:             "purchase_order",
		Partition:        e.CompanyID,
		UseListAsDefault: true,
		Keys:             []db.Coln{e.ID.Autoincrement(0)},
		Indexes: []db.Index{
			{
				Type:     db.TypeView,
				Keys:     []db.Coln{e.Status.Int32(), e.Updated.DecimalSize(8)},
				KeepPart: true,
			},
			{
				Keys:          []db.Coln{e.Week},
				UseIndexGroup: true,
			},
			{
				Keys:          []db.Coln{e.Week, e.DetailProductIDs},
				UseIndexGroup: true,
			},
			{
				Keys:          []db.Coln{e.Week, e.Status, e.DetailProductIDs},
				UseIndexGroup: true,
			},
			{
				Keys:          []db.Coln{e.Week, e.ProviderID},
				UseIndexGroup: true,
			},
			{
				Keys:          []db.Coln{e.Week, e.Status, e.ProviderID},
				UseIndexGroup: true,
			},
		},
	}
}
