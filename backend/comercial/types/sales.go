package types

import "app/db"

type SaleOrder struct {
	db.TableStruct[SaleOrderTable, SaleOrder]
	EmpresaID int32 `json:",omitempty"`
	Fecha     int16 `json:",omitempty"`
	Week      int16 `json:",omitempty"` // Example 2508
	AlmacenID int32 `json:",omitempty"`
	ID        int64

	//Tabla: Following slices must be same size
	DetailProductsIDs []int32  `json:",omitempty"`
	DetailPrices      []int32  `json:",omitempty"`
	DetailQuantities  []int32  `json:",omitempty"`
	DetailProductSkus []string `json:",omitempty"`

	TotalAmount    int32 `json:",omitempty"`
	TaxAmount      int32 `json:",omitempty"`
	DebtAmount     int32 `json:",omitempty"`
	DeliveryStatus int8  `json:",omitempty"`
	CajaID_        int32 `json:",omitempty"`

	// If contains 2 = the payment is done
	// If contains 3 = the delivery of the product is done
	ProcessesIncluded_ []int8 `json:",omitempty"`
	Created            int32  `json:",omitempty"`
	Updated            int32  `json:"upd,omitempty"`
	UpdatedBy          int32  `json:",omitempty"`
	// 0 = Anulado, 1 = Generado, 2 = Pagado, 3 = Entregado, 4 = Pagado + Entregado
	Status int8 `json:"ss,omitempty"`
}

type SaleOrderTable struct {
	db.TableStruct[SaleOrderTable, SaleOrder]
	EmpresaID          db.Col[SaleOrderTable, int32]
	ID                 db.Col[SaleOrderTable, int64]
	Fecha              db.Col[SaleOrderTable, int16]
	Week               db.Col[SaleOrderTable, int16]
	AlmacenID          db.Col[SaleOrderTable, int32]
	DetailProductsIDs  db.Col[SaleOrderTable, []int32]
	DetailPrices       db.Col[SaleOrderTable, []int32]
	DetailQuantities   db.Col[SaleOrderTable, []int32]
	DetailProductSkus  db.Col[SaleOrderTable, []string]
	TotalAmount        db.Col[SaleOrderTable, int32]
	TaxAmount          db.Col[SaleOrderTable, int32]
	DebtAmount         db.Col[SaleOrderTable, int32]
	Created            db.Col[SaleOrderTable, int32]
	Updated            db.Col[SaleOrderTable, int32]
	UpdatedBy          db.Col[SaleOrderTable, int32]
	Status             db.Col[SaleOrderTable, int8]
}

func (e SaleOrderTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:         "sale_order",
		Partition:    e.EmpresaID,
		Keys:         []db.Coln{e.ID.Autoincrement(3)},
		LocalIndexes: []db.Coln{e.Updated},
		HashIndexes: [][]db.Coln{
			{e.DetailProductsIDs, e.Week.CompositeBucketing(1,4,5)},
		},
		Views: []db.View{
			{Cols: []db.Coln{e.Fecha, e.Updated}, KeepPart: true},
		},
	}
}
