package types

import (
	"app/core"
	"app/db"
)

type SaleOrder struct {
	db.TableStruct[SaleOrderTable, SaleOrder]
	EmpresaID int32 `json:",omitempty"`
	Fecha     int16 `json:",omitempty"`
	AlmacenID int32 `json:",omitempty"`
	ID        int64

	//Table: Following slices must be same size
	DetailProductsIDs []int32  `json:",omitempty"`
	DetailPrices      []int32  `json:",omitempty"`
	DetailQuantities  []int32  `json:",omitempty"`
	DetailProductSkus []string `json:",omitempty"`

	TotalAmount    int32 `json:",omitempty"`
	TaxAmount      int32 `json:",omitempty"`
	DebtAmount     int32 `json:",omitempty"`
	DeliveryStatus int8  `json:",omitempty"`
	Created        int32 `json:",omitempty"`
	Updated        int32 `json:"upd,omitempty"`
	UpdatedBy      int32 `json:",omitempty"`
	// 0 = Anulado, 1 = Generado, 2 = Pagado, 3 = Entregado, 4 = Pagado + Entregado
	Status int8 `json:"ss,omitempty"`

	// Extra field for post request
	CajaID_ int32 `json:",omitempty"`
	// If contains 2 = the payment is done
	// If contains 3 = the delivery of the product is done
	ProcessesIncluded_ []int8 `json:",omitempty"`
	// This fields are used for fast filtering the last pending orders
	OrderPendingPaymentUpdated  int32 `json:",omitempty"`
	OrderPendingDeliveryUpdated int32 `json:",omitempty"`
	OrderCompletedUpdated       int32 `json:",omitempty"`
}

func (e *SaleOrder) AddStatus(orderState int8) error {
	updated := core.SUnixTime()
	if orderState == 2 {
		if e.Status == 1 || e.Status == 3 {
			e.Status += 1
		} else {
			core.Log("Error: No se puede agregar el estado Pagado a ", e.Status)
			return core.Err("Error: No se puede agregar el estado Pagado a ", e.Status)
		}
	} else if orderState == 3 {
		if e.Status == 1 || e.Status == 2 {
			e.Status += 2
		} else {
			core.Log("Error: No se puede agregar el estado Entregado a ", e.Status)
			return core.Err("Error: No se puede agregar el estado Entregado a ", e.Status)
		}
	}

	if e.OrderPendingPaymentUpdated > 0 && e.OrderPendingPaymentUpdated != updated {
		e.OrderPendingPaymentUpdated = updated
	}

	if e.OrderPendingDeliveryUpdated > 0 && e.OrderPendingDeliveryUpdated != updated {
		e.OrderPendingDeliveryUpdated = updated
	}

	if e.Status == 4 {
		e.OrderCompletedUpdated = updated
	}
	return nil
}

type SaleOrderTable struct {
	db.TableStruct[SaleOrderTable, SaleOrder]
	EmpresaID                   db.Col[SaleOrderTable, int32]
	ID                          db.Col[SaleOrderTable, int64]
	Fecha                       db.Col[SaleOrderTable, int16]
	AlmacenID                   db.Col[SaleOrderTable, int32]
	DetailProductsIDs           db.Col[SaleOrderTable, []int32]
	DetailPrices                db.Col[SaleOrderTable, []int32]
	DetailQuantities            db.Col[SaleOrderTable, []int32]
	DetailProductSkus           db.Col[SaleOrderTable, []string]
	TotalAmount                 db.Col[SaleOrderTable, int32]
	TaxAmount                   db.Col[SaleOrderTable, int32]
	DebtAmount                  db.Col[SaleOrderTable, int32]
	Created                     db.Col[SaleOrderTable, int32]
	Updated                     db.Col[SaleOrderTable, int32]
	UpdatedBy                   db.Col[SaleOrderTable, int32]
	Status                      db.Col[SaleOrderTable, int8]
	OrderPendingPaymentUpdated  db.Col[SaleOrderTable, int32]
	OrderPendingDeliveryUpdated db.Col[SaleOrderTable, int32]
	OrderCompletedUpdated       db.Col[SaleOrderTable, int32]
}

func (e SaleOrderTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:         "sale_order",
		Partition:    e.EmpresaID,
		Keys:         []db.Coln{e.ID.Autoincrement(3)},
		LocalIndexes: []db.Coln{e.Updated},
		HashIndexes: [][]db.Coln{
			{e.DetailProductsIDs, e.Fecha.CompositeBucketing(2, 6, 12, 20)},
		},
		Indexes: [][]db.Coln{
			{e.OrderPendingPaymentUpdated},
			{e.OrderPendingDeliveryUpdated},
			{e.OrderCompletedUpdated},
		},
		GlobalIndexes: [][]db.Coln{
			{e.Status.Int32(), e.Updated.DecimalSize(8)},
		},
		ViewsDeprecated: []db.View{
			{Cols: []db.Coln{e.Fecha, e.Updated}, KeepPart: true},
		},
	}
}
