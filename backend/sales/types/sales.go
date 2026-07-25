package types

import (
	"app/core"
	"github.com/ivanjoz/genix-orm/scylla"
)

type SaleOrderClientInfo struct {
	Name           string `json:",omitempty"`
	RegistryNumber string `json:",omitempty"`
	OnlyInsert     bool   `json:",omitempty"`
}

type SaleOrder struct {
	scylla.TableStruct[SaleOrderTable, SaleOrder]
	CompanyID   int32 `json:",omitempty"`
	Date        int16 `json:",omitempty"`
	WarehouseID int32 `json:",omitempty"`
	ID          int64

	//Table: Following slices must be same size
	DetailProductsIDs          []int32  `json:",omitempty" db:",list"`
	DetailPrices               []int32  `json:",omitempty" db:",list"`
	DetailQuantities           []int32  `json:",omitempty" db:",list"`
	DetailProductSkus          []string `json:",omitempty" db:",list"`
	DetailProductLotIDs        []int32  `json:",omitempty" db:",list"`
	DetailProductPresentations []int16  `json:",omitempty" db:",list"`

	TotalAmount   int32 `json:",omitempty"`
	TaxAmount     int32 `json:",omitempty"`
	DebtAmount    int32 `json:",omitempty"`
	ClientID      int32 `json:",omitempty"`
	Created       int32 `json:",omitempty"`
	Updated       int32 `json:"upd,omitempty"`
	UpdateCounter int32 `json:"upc,omitempty"`
	UpdatedBy     int32 `json:",omitempty"`
	// 0 = Anulado, 1 = Generado, 2 = Pagado, 3 = Entregado, 4 = Pagado + Entregado
	Status            int8  `json:"ss,omitempty"`
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
	PaymentDueDate  int16                `json:",omitempty" db:"payment_due_date"`
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
	scylla.TableStruct[SaleOrderTable, SaleOrder]
	CompanyID                  scylla.Col[SaleOrderTable, int32]
	ID                         scylla.Col[SaleOrderTable, int64]
	Date                       scylla.Col[SaleOrderTable, int16]
	WarehouseID                scylla.Col[SaleOrderTable, int32]
	LastPaymentCajaID          scylla.Col[SaleOrderTable, int32]
	DetailProductsIDs          scylla.Col[SaleOrderTable, []int32]
	DetailPrices               scylla.Col[SaleOrderTable, []int32]
	DetailQuantities           scylla.Col[SaleOrderTable, []int32]
	DetailProductSkus          scylla.Col[SaleOrderTable, []string]
	DetailProductLotIDs        scylla.Col[SaleOrderTable, []int32]
	DetailProductPresentations scylla.Col[SaleOrderTable, []int16]
	TotalAmount                scylla.Col[SaleOrderTable, int32]
	TaxAmount                  scylla.Col[SaleOrderTable, int32]
	DebtAmount                 scylla.Col[SaleOrderTable, int32]
	Created                    scylla.Col[SaleOrderTable, int32]
	ClientID                   scylla.Col[SaleOrderTable, int32]
	Updated                    scylla.Col[SaleOrderTable, int32]
	UpdateCounter              scylla.Col[SaleOrderTable, int32]
	UpdatedBy                  scylla.Col[SaleOrderTable, int32]
	Status                     scylla.Col[SaleOrderTable, int8]
	LastPaymentTime            scylla.Col[SaleOrderTable, int32]
	LastPaymentUser            scylla.Col[SaleOrderTable, int32]
	DeliveryTime               scylla.Col[SaleOrderTable, int32]
	DeliveryUser               scylla.Col[SaleOrderTable, int32]
	PaymentDueDate             scylla.Col[SaleOrderTable, int16]
}

func (e SaleOrderTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:      "sale_order",
		Partition: e.CompanyID,
		Keys:      scylla.Cols(e.ID.Autoincrement(2)),
		Indexes: []scylla.Index{
			{
				Type: scylla.TypeLocalIndex,
				Keys: scylla.Cols(e.Updated),
			},
			{
				Keys:          scylla.Cols(e.Date),
				UseIndexGroup: true,
			},
			{
				Keys:          scylla.Cols(e.Date.StoreAsWeek(), e.Status),
				UseIndexGroup: true,
			},
			{
				Keys:          scylla.Cols(e.Date.StoreAsWeek(), e.ClientID),
				UseIndexGroup: true,
			},
			{
				Keys:          scylla.Cols(e.Date.StoreAsWeek(), e.ClientID, e.DetailProductsIDs),
				UseIndexGroup: true,
			},
			{
				Keys:          scylla.Cols(e.Date.StoreAsWeek(), e.DetailProductsIDs),
				UseIndexGroup: true,
			},
			{
				Type: scylla.TypeView,
				Keys: scylla.Cols(e.Status.Int32(), e.UpdateCounter.DecimalSize(8)),
			},
		},
	}
}

// Table to save the summary per day
type SaleSummary struct {
	scylla.TableStruct[SaleSummaryTable, SaleSummary]
	CompanyID int32 `json:",omitempty"`
	Date      int16 `json:",omitempty"`
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
	scylla.TableStruct[SaleSummaryTable, SaleSummary]
	CompanyID               scylla.Col[SaleSummaryTable, int32]
	Date                    scylla.Col[SaleSummaryTable, int16]
	ProductIDs              scylla.Col[SaleSummaryTable, []int32]
	Quantity                scylla.Col[SaleSummaryTable, []int32]
	QuantityPendingDelivery scylla.Col[SaleSummaryTable, []int32]
	TotalAmount             scylla.Col[SaleSummaryTable, []int32]
	TotalDebtAmount         scylla.Col[SaleSummaryTable, []int32]
	Updated                 scylla.Col[SaleSummaryTable, int32]
	ReprocessUpdated        scylla.Col[SaleSummaryTable, int32]
}

func (e SaleSummaryTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:      "sale_summary",
		Partition: e.CompanyID,
		Keys:      scylla.Cols(e.Date),
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
	scylla.TableStruct[ProductSaleSummaryTable, ProductSaleSummary]
	CompanyID int32  `json:",omitempty"`
	Date      int16  `json:",omitempty"`
	ProductID int32  `json:",omitempty"`
	Updated   int32  `json:",omitempty"`
	Stats     []byte `json:",omitempty"`
}

type ProductSaleSummaryTable struct {
	scylla.TableStruct[ProductSaleSummaryTable, ProductSaleSummary]
	CompanyID scylla.Col[ProductSaleSummaryTable, int32]
	Date      scylla.Col[ProductSaleSummaryTable, int16]
	ProductID scylla.Col[ProductSaleSummaryTable, int32]
	Stats     scylla.Col[ProductSaleSummaryTable, []byte]
	Updated   scylla.Col[ProductSaleSummaryTable, int32]
}

func (e ProductSaleSummaryTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:                 "product_sale_summary",
		Partition:            e.CompanyID,
		Keys:                 scylla.Cols(e.Date, e.ProductID),
		DisableUpdateCounter: true,
	}
}

const (
	OrderStatusPending   = int8(1)
	OrderStatusPaid      = int8(2)
	OrderStatusDelivered = int8(3)
	OrderStatusCompleted = int8(4)
	OrderStatusAnnulled  = int8(0)
)
