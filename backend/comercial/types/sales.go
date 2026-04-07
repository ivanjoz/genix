package types

import (
	"app/core"
	"app/db"
)

type SaleOrderClientInfo struct {
	Name           string `json:",omitempty"`
	RegistryNumber string `json:",omitempty"`
	OnlyInsert     bool   `json:",omitempty"`
}

type SaleOrder struct {
	db.TableStruct[SaleOrderTable, SaleOrder]
	CompanyID   int32 `json:",omitempty"`
	Fecha       int16 `json:",omitempty"`
	WarehouseID int32 `json:",omitempty"`
	ID          int64

	//Table: Following slices must be same size
	DetailProductsIDs          []int32  `json:",omitempty" db:",list"`
	DetailPrices               []int32  `json:",omitempty" db:",list"`
	DetailQuantities           []int32  `json:",omitempty" db:",list"`
	DetailProductSkus          []string `json:",omitempty" db:",list"`
	DetailProductLots          []string `json:",omitempty" db:",list"`
	DetailProductPresentations []int16  `json:",omitempty" db:",list"`

	TotalAmount    int32 `json:",omitempty"`
	TaxAmount      int32 `json:",omitempty"`
	DebtAmount     int32 `json:",omitempty"`
	DeliveryStatus int8  `json:",omitempty"`
	ClientID       int32 `json:",omitempty"`
	Created        int32 `json:",omitempty"`
	Updated        int32 `json:"upd,omitempty"`
	UpdatedBy      int32 `json:",omitempty"`
	// 0 = Anulado, 1 = Generado, 2 = Pagado, 3 = Entregado, 4 = Pagado + Entregado
	Status      int8 `json:"ss,omitempty"`
	StatusTrace int8 `json:",omitempty"`
	// Last payment caja used for payment action tracking.
	// Keep db column mapped to legacy `caja_id_` to avoid data migration breaks.
	LastPaymentCajaID int32 `json:",omitempty" db:"caja_id_"`
	// If contains 2 = the payment is done
	// If contains 3 = the delivery of the product is done
	ActionsIncluded []int8 `json:",omitempty"`
	// Audit trail fields for payment and delivery actions.
	LastPaymentTime int32                `json:",omitempty"`
	LastPaymentUser int32                `json:",omitempty"`
	DeliveryTime    int32                `json:",omitempty"`
	DeliveryUser    int32                `json:",omitempty"`
	ClientInfo      *SaleOrderClientInfo `json:",omitempty"`
}

func (e *SaleOrder) AddStatus(orderState int8) error {
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
	return nil
}

type SaleOrderTable struct {
	db.TableStruct[SaleOrderTable, SaleOrder]
	CompanyID                  db.Col[SaleOrderTable, int32]
	ID                         db.Col[SaleOrderTable, int64]
	Fecha                      db.Col[SaleOrderTable, int16]
	WarehouseID                db.Col[SaleOrderTable, int32]
	LastPaymentCajaID          db.Col[SaleOrderTable, int32]
	DetailProductsIDs          db.Col[SaleOrderTable, []int32]
	DetailPrices               db.Col[SaleOrderTable, []int32]
	DetailQuantities           db.Col[SaleOrderTable, []int32]
	DetailProductSkus          db.Col[SaleOrderTable, []string]
	DetailProductPresentations db.Col[SaleOrderTable, []int16]
	TotalAmount                db.Col[SaleOrderTable, int32]
	TaxAmount                  db.Col[SaleOrderTable, int32]
	DebtAmount                 db.Col[SaleOrderTable, int32]
	Created                    db.Col[SaleOrderTable, int32]
	ClientID                   db.Col[SaleOrderTable, int32]
	Updated                    db.Col[SaleOrderTable, int32]
	UpdatedBy                  db.Col[SaleOrderTable, int32]
	Status                     db.Col[SaleOrderTable, int8]
	LastPaymentTime            db.Col[SaleOrderTable, int32]
	LastPaymentUser            db.Col[SaleOrderTable, int32]
	DeliveryTime               db.Col[SaleOrderTable, int32]
	DeliveryUser               db.Col[SaleOrderTable, int32]
	StatusTrace                db.Col[SaleOrderTable, int8]
}

func (e SaleOrderTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "sale_order",
		Partition: e.CompanyID,
		Keys:      []db.Coln{e.ID.Autoincrement(2)},
		Indexes: []db.Index{
			{
				Type: db.TypeLocalIndex,
				Keys: []db.Coln{e.Updated},
			},
			{
				Keys:          []db.Coln{e.Fecha},
				UseIndexGroup: true,
			},
			{
				Keys:          []db.Coln{e.Fecha.StoreAsWeek(), e.Status},
				UseIndexGroup: true,
			},
			{
				Keys:          []db.Coln{e.Fecha.StoreAsWeek(), e.ClientID},
				UseIndexGroup: true,
			},
			{
				Keys:          []db.Coln{e.Fecha.StoreAsWeek(), e.ClientID, e.DetailProductsIDs},
				UseIndexGroup: true,
			},
			{
				Keys:          []db.Coln{e.Fecha.StoreAsWeek(), e.DetailProductsIDs},
				UseIndexGroup: true,
			},
			{
				Type:     db.TypeViewTable,
				Keys:     []db.Coln{e.DetailProductsIDs, e.Fecha},
				Cols:     []db.Coln{e.Updated},
				KeepPart: true,
			},
			{
				Type:     db.TypeView,
				Keys:     []db.Coln{e.Status.Int32(), e.Updated.DecimalSize(8)},
				KeepPart: true,
			},
			{
				Type:     db.TypeView,
				Keys:     []db.Coln{e.StatusTrace.Int32(), e.Updated.DecimalSize(8)},
				Cols:     []db.Coln{e.ClientID, e.Updated, e.DetailProductsIDs, e.Status},
				KeepPart: true,
			},
		},
	}
}

// Table to save the summary per day
type SaleSummary struct {
	db.TableStruct[SaleSummaryTable, SaleSummary]
	EmpresaID int32 `json:",omitempty"`
	Fecha     int16 `json:",omitempty"`
	// Single int32 representation keeps the summary format simple and stable.
	ProductIDs              []int32 `json:",omitempty" db:",list"`
	Quantity                []int32 `json:",omitempty" db:",list"`
	QuantityPendingDelivery []int32 `json:",omitempty" db:",list"`
	TotalAmount             []int32 `json:",omitempty" db:",list"`
	TotalDebtAmount         []int32 `json:",omitempty" db:",list"`
	Updated                 int32   `json:"upd,omitempty"`
	ReprocessUpdated        int32   `json:"-,omitempty"`
}

type SaleSummaryTable struct {
	db.TableStruct[SaleSummaryTable, SaleSummary]
	EmpresaID               db.Col[SaleSummaryTable, int32]
	Fecha                   db.Col[SaleSummaryTable, int16]
	ProductIDs              db.Col[SaleSummaryTable, []int32]
	Quantity                db.Col[SaleSummaryTable, []int32]
	QuantityPendingDelivery db.Col[SaleSummaryTable, []int32]
	TotalAmount             db.Col[SaleSummaryTable, []int32]
	TotalDebtAmount         db.Col[SaleSummaryTable, []int32]
	Updated                 db.Col[SaleSummaryTable, int32]
	ReprocessUpdated        db.Col[SaleSummaryTable, int32]
}

func (e SaleSummaryTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "sale_summary",
		Partition: e.EmpresaID,
		Keys:      []db.Coln{e.Fecha},
	}
}

/* Sale summary 2 */
type SaleOrderProductStats struct {
	Quantity                int32
	QuantityPendingDelivery int32
	TotalAmount             int32
	TotalDebtAmount         int32
}

type ProductSaleSummary struct {
	db.TableStruct[ProductSaleSummaryTable, ProductSaleSummary]
	CompanyID int32  `json:",omitempty"`
	Fecha     int16  `json:",omitempty"`
	ProductID int32  `json:",omitempty"`
	Updated   int32  `json:",omitempty"`
	Stats     []byte `json:",omitempty"`
}

type ProductSaleSummaryTable struct {
	db.TableStruct[ProductSaleSummaryTable, ProductSaleSummary]
	CompanyID db.Col[ProductSaleSummaryTable, int32]
	Fecha     db.Col[ProductSaleSummaryTable, int16]
	ProductID db.Col[ProductSaleSummaryTable, int32]
	Stats     db.Col[ProductSaleSummaryTable, []byte]
	Updated   db.Col[ProductSaleSummaryTable, int32]
}

func (e ProductSaleSummaryTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:                 "product_sale_summary",
		Partition:            e.CompanyID,
		Keys:                 []db.Coln{e.Fecha, e.ProductID},
		DisableUpdateCounter: true,
	}
}
