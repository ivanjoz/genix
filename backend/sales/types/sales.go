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
	Date        int16 `json:",omitempty"`
	WarehouseID int32 `json:",omitempty"`
	ID          int64

	//Table: Following slices must be same size
	DetailProductsIDs []int32 `json:",omitempty" db:",list"`
	// DetailPrices is the price of one whole unit; DetailSubPrices the price of one sub-unit.
	// Both are needed per line and neither is derivable from the other: selling loose is
	// deliberately not proportional (four candies from a 5000-cent box of six is not 3333).
	// A summary reprocess has to reconstruct what was charged, not what the product costs now.
	DetailPrices    []int32 `json:",omitempty" db:",list"`
	DetailSubPrices []int32 `json:",omitempty" db:",list"`
	// DetailQuantities is the packed line quantity: Units*core.QuantityLineScale + Sub, so
	// 1001 at divisor 6 is one box plus one candy. A sale order is a document — it is never
	// accumulated and never SUM()-ed — so it packs both halves into one column, where the
	// stock and ledger tables keep them separate. core.UnpackQuantityLine splits it.
	DetailQuantities []int32 `json:",omitempty" db:",list"`
	// DetailSubDivisor is the divisor each line's packed sub-unit part is expressed in,
	// recorded per line so a later refinement of the product never restates a past sale.
	DetailSubDivisor           []int16  `json:",omitempty" db:",list"`
	DetailProductSkus          []string `json:",omitempty" db:",list"`
	DetailProductLotIDs        []int32  `json:",omitempty" db:",list"`
	DetailProductPresentations []int16  `json:",omitempty" db:",list"`

	TotalAmount    int32 `json:",omitempty"`
	TaxAmount      int32 `json:",omitempty"`
	DebtAmount     int32 `json:",omitempty"`
	ClientID       int32 `json:",omitempty"`
	Created        int32 `json:",omitempty"`
	Updated        int32 `json:"upd,omitempty"`
	UpdateCounter  int32 `json:"upc,omitempty"`
	UpdatedVersion int32 `json:"upv,omitempty"`
	UpdatedBy      int32 `json:",omitempty"`
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
	// IssueSeriesID names the invoicing series this sale will be issued under. It
	// is a request field, not a column: it is folded into the last two digits of
	// the id on creation, and read back with SeriesID(). Zero when the till does
	// not say.
	IssueSeriesID  int8  `json:",omitempty"`
	PaymentDueDate int16 `json:",omitempty" db:"payment_due_date"`
	// AnnulReason is why the sale was annulled, required by the annul endpoint. There is no
	// AnnulledTime/AnnulledUser beside it: annulment is terminal, so Updated and UpdatedBy are
	// already the "when and by whom" and a second pair could only drift from them.
	AnnulReason string `json:",omitempty"`
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
	CompanyID                  db.Col[*SaleOrderTable, int32]
	ID                         db.Col[*SaleOrderTable, int64]
	Date                       db.Col[*SaleOrderTable, int16]
	WarehouseID                db.Col[*SaleOrderTable, int32]
	LastPaymentCajaID          db.Col[*SaleOrderTable, int32]
	DetailProductsIDs          db.Col[*SaleOrderTable, []int32]
	DetailPrices               db.Col[*SaleOrderTable, []int32]
	DetailSubPrices            db.Col[*SaleOrderTable, []int32]
	DetailQuantities           db.Col[*SaleOrderTable, []int32]
	DetailSubDivisor           db.Col[*SaleOrderTable, []int16]
	DetailProductSkus          db.Col[*SaleOrderTable, []string]
	DetailProductLotIDs        db.Col[*SaleOrderTable, []int32]
	DetailProductPresentations db.Col[*SaleOrderTable, []int16]
	TotalAmount                db.Col[*SaleOrderTable, int32]
	TaxAmount                  db.Col[*SaleOrderTable, int32]
	DebtAmount                 db.Col[*SaleOrderTable, int32]
	Created                    db.Col[*SaleOrderTable, int32]
	ClientID                   db.Col[*SaleOrderTable, int32]
	Updated                    db.Col[*SaleOrderTable, int32]
	UpdateCounter              db.Col[*SaleOrderTable, int32]
	UpdatedVersion             db.Col[*SaleOrderTable, int32]
	UpdatedBy                  db.Col[*SaleOrderTable, int32]
	Status                     db.Col[*SaleOrderTable, int8]
	LastPaymentTime            db.Col[*SaleOrderTable, int32]
	LastPaymentUser            db.Col[*SaleOrderTable, int32]
	DeliveryTime               db.Col[*SaleOrderTable, int32]
	DeliveryUser               db.Col[*SaleOrderTable, int32]
	PaymentDueDate             db.Col[*SaleOrderTable, int16]
	AnnulReason                db.Col[*SaleOrderTable, string]
}

func (e SaleOrderTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        25,
		Name:      "sale_order",
		Partition: e.CompanyID,
		// A plain key. The id is built by MakeSaleOrderID, which packs the counter,
		// two random digits and the invoicing series into it — a layout the ORM's
		// autoincrement cannot express.
		Keys: db.Cols(e.ID),
		// Sizes the Status slot of the delta index.
		FixedValues: []db.FixedValues{
			{Col: e.Status, Min: 0, Max: 4},
		},
		Indexes: []db.Index{
			{
				Type: db.TypeLocalIndex,
				Keys: db.Cols(e.Updated),
			},
			{
				Keys:          db.Cols(e.Date),
				UseIndexGroup: true,
			},
			{
				Keys:          db.Cols(e.Date, e.Status),
				UseIndexGroup: true,
			},
			{
				Keys:          db.Cols(e.Date, e.ClientID),
				UseIndexGroup: true,
			},
			{
				Keys:          db.Cols(e.Date, e.ClientID, e.DetailProductsIDs),
				UseIndexGroup: true,
			},
			{
				Keys:          db.Cols(e.Date, e.DetailProductsIDs),
				UseIndexGroup: true,
			},
			// The status tabs pin Status themselves and take only the watermark from Delta(upv), so
			// this replaces the [Status, UpdateCounter] view the delta read used to use.
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
		},
	}
}

// SaleSummary is the day-level response shape of GET.sale-summary. It is not persisted:
// buildSaleSummariesFromProductRows assembles it at read time from ProductSaleSummary rows.
type SaleSummary struct {
	CompanyID int32 `json:",omitempty"`
	Date      int16 `json:",omitempty"`
	// Products are sent columnar: every slice is aligned by index with ProductIDs.
	ProductIDs []int32 `json:",omitempty"`
	// Quantity/SubQuantity are the split pair: a summary is accumulated across sales, so it
	// cannot use the packed document form (1004 + 1004 would be 2008, not 3002).
	Quantity                   []int32 `json:",omitempty"`
	QuantityPendingDelivery    []int32 `json:",omitempty"`
	SubQuantity                []int16 `json:",omitempty"`
	SubQuantityPendingDelivery []int16 `json:",omitempty"`
	SubDivisor                 []int16 `json:",omitempty"`
	TotalAmount                []int32 `json:",omitempty"`
	TotalDebtAmount            []int32 `json:",omitempty"`
	Updated                    int32   `json:"upd,omitempty"`
}

/* Sale summary 2 */
// SaleOrderProductStats is packed by libs.SerializeInt30Struct, which stores every field in
// at most 30 bits and saturates silently past that.
//
// Unlike the ledger and stock rows, this one keeps its pairs **normalized**: it is updated
// by a Go-side read-modify-write, not by a Scylla SUM(), so carrying on every write costs
// nothing. That is what bounds the Sub* fields — a normalized Sub is always below the
// divisor, and the divisor tops out at core.MaxQuantityDivisor (1000) — so they are int16.
type SaleOrderProductStats struct {
	Quantity int32 `cb:"1,minimal"`
	// Bounded by Quantity, so it carries the same width. Only the Sub* halves are bounded by
	// the divisor and can narrow.
	QuantityPendingDelivery    int32 `cb:"2"`
	SubQuantity                int16 `cb:"3"`
	SubQuantityPendingDelivery int16 `cb:"4"`
	// SubDivisor the two Sub* fields are counted in. One per (date, product) row: a
	// refinement mid-day would otherwise have the row adding sixths to twelfths.
	SubDivisor      int16 `cb:"5"`
	TotalAmount     int32 `cb:"6"`
	TotalDebtAmount int32 `cb:"7"`
}

type ProductSaleSummary struct {
	db.TableStruct[ProductSaleSummaryTable, ProductSaleSummary]
	CompanyID int32  `json:",omitempty"`
	Date      int16  `json:",omitempty"`
	ProductID int32  `json:",omitempty"`
	Updated   int32  `json:",omitempty"`
	Stats     []byte `json:",omitempty"`
}

type ProductSaleSummaryTable struct {
	db.TableStruct[ProductSaleSummaryTable, ProductSaleSummary]
	CompanyID db.Col[*ProductSaleSummaryTable, int32]
	Date      db.Col[*ProductSaleSummaryTable, int16]
	ProductID db.Col[*ProductSaleSummaryTable, int32]
	Stats     db.Col[*ProductSaleSummaryTable, []byte]
	Updated   db.Col[*ProductSaleSummaryTable, int32]
}

func (e ProductSaleSummaryTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        19,
		Name:      "product_sale_summary",
		Partition: e.CompanyID,
		Keys:      db.Cols(e.Date, e.ProductID),

		DisableDefaultColumns: true,
	}
}

const (
	OrderStatusPending   = int8(1)
	OrderStatusPaid      = int8(2)
	OrderStatusDelivered = int8(3)
	OrderStatusCompleted = int8(4)
	OrderStatusAnnulled  = int8(0)
)

// The ids of the "Gestión Ventas" entry in backend/access.toml. Named here rather than written
// as literals in the handler so a catalog edit can be found by grep from the code that depends
// on it — the catalog is data and the compiler cannot check the link.
const (
	AccesoIDGestionVentas  = int32(11)
	SubAccesoIDAnularVenta = int32(2)
)

// MaxAnnulReasonLength bounds the free-text reason so the column cannot be used as storage.
const MaxAnnulReasonLength = 200
