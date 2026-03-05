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
	DetailProductsIDs []int32  `json:",omitempty" db:",list"`
	DetailPrices      []int32  `json:",omitempty" db:",list"`
	DetailQuantities  []int32  `json:",omitempty" db:",list"`
	DetailProductSkus []string `json:",omitempty" db:",list"`

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
		Views: []db.View{
			{Cols: []db.Coln{e.Fecha, e.Updated}, KeepPart: true},
		},
	}
}

// Table to save the summary per day
type SaleSummary struct {
	db.TableStruct[SaleSummaryTable, SaleSummary]
	EmpresaID int32 `json:",omitempty"`
	Fecha     int16 `json:",omitempty"`
	// uint16 productos and quantity
	ProductIDs_16              []uint16 `json:",omitempty"`
	Quantity_16                []uint16 `json:",omitempty"`
	QuantityPendingDelivery_16 []uint16 `json:",omitempty"`
	TotalAmount_16             []int32  `json:",omitempty" db:",list"`
	TotalDebtAmount_16         []int32  `json:",omitempty" db:",list"`
	// uint32 productos and quantity
	ProductIDs_32              []int32 `json:",omitempty" db:",list"`
	Quantity_32                []int32 `json:",omitempty" db:",list"`
	QuantityPendingDelivery_32 []int32 `json:",omitempty" db:",list"`
	TotalAmount_32             []int32 `json:",omitempty" db:",list"`
	TotalDebtAmount_32         []int32 `json:",omitempty" db:",list"`
	Updated                    int32   `json:"upd,omitempty"`
}

type SaleSummaryTable struct {
	db.TableStruct[SaleSummaryTable, SaleSummary]
	EmpresaID                  db.Col[SaleSummaryTable, int32]
	Fecha                      db.Col[SaleSummaryTable, int16]
	ProductIDs_16              db.Col[SaleSummaryTable, []uint16]
	Quantity_16                db.Col[SaleSummaryTable, []uint16]
	QuantityPendingDelivery_16 db.Col[SaleSummaryTable, []uint16]
	TotalAmount_16             db.Col[SaleSummaryTable, []int32]
	TotalDebtAmount_16         db.Col[SaleSummaryTable, []int32]
	ProductIDs_32              db.Col[SaleSummaryTable, []int32]
	Quantity_32                db.Col[SaleSummaryTable, []int32]
	QuantityPendingDelivery_32 db.Col[SaleSummaryTable, []int32]
	TotalAmount_32             db.Col[SaleSummaryTable, []int32]
	TotalDebtAmount_32         db.Col[SaleSummaryTable, []int32]
	Updated                    db.Col[SaleSummaryTable, int32]
}

func (e SaleSummaryTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "sale_summary",
		Partition: e.EmpresaID,
		Keys:      []db.Coln{e.Fecha},
	}
}


// FlattenSaleSummary converts 16-bit packed slices into the 32-bit slices and
// clears *_16 fields so clients consume one single flat shape.
func (summary *SaleSummary) FlattenSaleSummary() {
	if summary == nil {
		return
	}

	// Keep existing int32 rows and only append converted int16 rows.
	for index, productID := range summary.ProductIDs_16 {
		if productID == 0 {
			continue
		}
		summary.ProductIDs_32 = append(summary.ProductIDs_32, int32(productID))
		summary.Quantity_32 = append(summary.Quantity_32, int32(summary.Quantity_16[index]))
		summary.QuantityPendingDelivery_32 = append(summary.QuantityPendingDelivery_32, int32(summary.QuantityPendingDelivery_16[index]))
		summary.TotalAmount_32 = append(summary.TotalAmount_32, summary.TotalAmount_16[index])
		summary.TotalDebtAmount_32 = append(summary.TotalDebtAmount_32, summary.TotalDebtAmount_16[index])
	}

	// Clear packed 16-bit slices from API output.
	summary.ProductIDs_16 = nil
	summary.Quantity_16 = nil
	summary.QuantityPendingDelivery_16 = nil
	summary.TotalAmount_16 = nil
	summary.TotalDebtAmount_16 = nil
}
