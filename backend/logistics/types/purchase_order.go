package types

import "github.com/ivanjoz/genix-orm/scylla"

const (
	PurchaseOrderStatusCanceled  int8 = 0
	PurchaseOrderStatusPending   int8 = 1
	PurchaseOrderStatusConfirmed int8 = 2
	PurchaseOrderStatusFulfilled int8 = 4
)

type PurchaseOrder struct {
	scylla.TableStruct[PurchaseOrderTable, PurchaseOrder]
	ID           int32
	CompanyID    int32 `json:",omitempty"`
	ProviderID   int32 `json:",omitempty"`
	WarehouseID  int32 `json:",omitempty"`
	Date         int16 `json:",omitempty"`
	Week         int16 `json:",omitempty"`
	DeliveryDate int16 `json:",omitempty"`
	PaymentDate  int16 `json:",omitempty"`
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
	TotalAmount          int32   `json:",omitempty"`
	TaxAmount            int32   `json:",omitempty"`
	DebtAmount           int32   `json:",omitempty"`
	DifferenceQuantity   int32   `json:",omitempty"`
	DifferenceValue      int32   `json:",omitempty"`
	InvoiceNumber        string  `json:",omitempty"`
	Notes                string  `json:",omitempty"`

	Created   int32 `json:",omitempty"`
	CreatedBy int32 `json:",omitempty"`
	Updated   int32 `json:"upd,omitempty"`
	UpdatedBy int32 `json:",omitempty"`
	Status    int8  `json:"ss,omitempty"`
}

type PurchaseOrderTable struct {
	scylla.TableStruct[PurchaseOrderTable, PurchaseOrder]
	CompanyID                    scylla.Col[PurchaseOrderTable, int32]
	ID                           scylla.Col[PurchaseOrderTable, int32]
	ProviderID                   scylla.Col[PurchaseOrderTable, int32]
	WarehouseID                  scylla.Col[PurchaseOrderTable, int32]
	Date                         scylla.Col[PurchaseOrderTable, int16]
	Week                         scylla.Col[PurchaseOrderTable, int16]
	DeliveryDate                 scylla.Col[PurchaseOrderTable, int16]
	PaymentDate                  scylla.Col[PurchaseOrderTable, int16]
	DetailProductIDs             scylla.Col[PurchaseOrderTable, []int32]
	DetailProductQuantity        scylla.Col[PurchaseOrderTable, []int32]
	DetailProductPrice           scylla.Col[PurchaseOrderTable, []int32]
	DetailProductPresentationIDs scylla.Col[PurchaseOrderTable, []int32]
	DetailSupplyIDs              scylla.Col[PurchaseOrderTable, []int32]
	DetailSupplyQuantity         scylla.Col[PurchaseOrderTable, []int32]
	DetailSupplyPrice            scylla.Col[PurchaseOrderTable, []int32]
	TotalAmount                  scylla.Col[PurchaseOrderTable, int32]
	TaxAmount                    scylla.Col[PurchaseOrderTable, int32]
	DebtAmount                   scylla.Col[PurchaseOrderTable, int32]
	DifferenceQuantity           scylla.Col[PurchaseOrderTable, int32]
	DifferenceValue              scylla.Col[PurchaseOrderTable, int32]
	InvoiceNumber                scylla.Col[PurchaseOrderTable, string]
	Notes                        scylla.Col[PurchaseOrderTable, string]
	Created                      scylla.Col[PurchaseOrderTable, int32]
	CreatedBy                    scylla.Col[PurchaseOrderTable, int32]
	Updated                      scylla.Col[PurchaseOrderTable, int32]
	UpdatedBy                    scylla.Col[PurchaseOrderTable, int32]
	Status                       scylla.Col[PurchaseOrderTable, int8]
}

func (e PurchaseOrderTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:             "purchase_order",
		Partition:        e.CompanyID,
		UseListAsDefault: true,
		Keys:             scylla.Cols(e.ID.Autoincrement(0)),
		Indexes: []scylla.Index{
			{
				Type: scylla.TypeView,
				Keys: scylla.Cols(e.Status.Int32(), e.Updated.DecimalSize(8)),
			},
			{
				Keys:          scylla.Cols(e.Week),
				UseIndexGroup: true,
			},
			{
				Keys:          scylla.Cols(e.Week, e.DetailProductIDs),
				UseIndexGroup: true,
			},
			{
				Keys:          scylla.Cols(e.Week, e.Status, e.DetailProductIDs),
				UseIndexGroup: true,
			},
			{
				Keys:          scylla.Cols(e.Week, e.ProviderID),
				UseIndexGroup: true,
			},
			{
				Keys:          scylla.Cols(e.Week, e.Status, e.ProviderID),
				UseIndexGroup: true,
			},
		},
	}
}
