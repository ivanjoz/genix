package types

import (
	"app/core"
	"app/db"
	finance "app/finance/types"
)

// SaleOrderClientInfo is a walk-in client typed at the till. It is request-only — SaleOrderTable has
// no column for it — and is turned into a ClientProvider row, which (with its snapshot) is the only
// place these fields are stored.
type SaleOrderClientInfo struct {
	Name           string `json:",omitempty" cb:"Name,1"`
	RegistryNumber string `json:",omitempty" cb:"RegistryNumber,2"`
	// IdentityDocType is the SUNAT document type the till picked (crm.IdentityDoc*). 0 leaves
	// SaveClientProviders to derive it from the number's shape.
	IdentityDocType int8 `json:",omitempty" cb:"IdentityDocType,3"`
	OnlyInsert      bool `json:",omitempty" cb:"OnlyInsert,4"`
}

type SaleOrder struct {
	db.TableStruct[SaleOrderTable, SaleOrder]
	CompanyID   int32 `json:",omitempty" cb:"CompanyID,1"`
	Date        int16 `json:",omitempty" cb:"Date,2"`
	WarehouseID int32 `json:",omitempty" cb:"WarehouseID,3"`
	ID          int64 `cb:"ID,4"`

	//Table: Following slices must be same size
	DetailProductsIDs []int32 `json:",omitempty" db:",list" cb:"DetailProductsIDs,5"`
	// DetailPrices is the price of one whole unit; DetailSubPrices the price of one sub-unit.
	// Both are needed per line and neither is derivable from the other: selling loose is
	// deliberately not proportional (four candies from a 5000-cent box of six is not 3333).
	// A summary reprocess has to reconstruct what was charged, not what the product costs now.
	DetailPrices    []int32 `json:",omitempty" db:",list" cb:"DetailPrices,6"`
	DetailSubPrices []int32 `json:",omitempty" db:",list" cb:"DetailSubPrices,7"`
	// DetailQuantities is the packed line quantity: Units*core.QuantityLineScale + Sub, so
	// 1001 at divisor 6 is one box plus one candy. A sale order is a document — it is never
	// accumulated and never SUM()-ed — so it packs both halves into one column, where the
	// stock and ledger tables keep them separate. core.UnpackQuantityLine splits it.
	DetailQuantities []int32 `json:",omitempty" db:",list" cb:"DetailQuantities,8"`
	// DetailSubDivisor is the divisor each line's packed sub-unit part is expressed in,
	// recorded per line so a later refinement of the product never restates a past sale.
	DetailSubDivisor           []int16  `json:",omitempty" db:",list" cb:"DetailSubDivisor,9"`
	DetailProductSkus          []string `json:",omitempty" db:",list" cb:"DetailProductSkus,10"`
	DetailProductLotIDs        []int32  `json:",omitempty" db:",list" cb:"DetailProductLotIDs,11"`
	DetailProductPresentations []int16  `json:",omitempty" db:",list" cb:"DetailProductPresentations,12"`

	// DetailPrices, DetailSubPrices and the three amounts below are all in the sale's
	// currency: a USD sale stores dollars, converted per unit from the catalog on creation.
	TotalAmount int32 `json:",omitempty" cb:"TotalAmount,13"`
	TaxAmount   int32 `json:",omitempty" cb:"TaxAmount,14"`
	DebtAmount  int32 `json:",omitempty" cb:"DebtAmount,15"`
	// CurrencyType is the sale's currency: 1 = PEN, 2 = USD (the cash-bank codes). A paid
	// sale takes its cash-bank's currency; an unpaid one takes what the till picked.
	CurrencyType int8 `json:",omitempty" cb:"CurrencyType,16"`
	// ExchangeRate is the effective rate × 1000 the lines were converted with, spread
	// included — what a report needs to restate a USD sale in PEN. A USD sale always has one;
	// a PEN sale only when it sold a USD-priced product.
	ExchangeRate  int32 `json:",omitempty" cb:"ExchangeRate,17"`
	ClientID      int32 `json:",omitempty" cb:"ClientID,18"`
	Created       int32 `json:",omitempty" cb:"Created,19"`
	Updated       int32 `json:"upd,omitempty" cb:"upd,20"`
	UpdateCounter int32 `json:"upc,omitempty" cb:"upc,21"`
	Version       int16 `json:"ver,omitempty" cb:"ver,22"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata  *string `json:"mdt,omitempty" cb:"mdt,23"`
	UpdatedBy int32   `json:",omitempty" cb:"UpdatedBy,24"`
	// 0 = Anulado, 1 = Generado, 2 = Pagado, 3 = Entregado, 4 = Pagado + Entregado
	Status            int8  `json:"ss,omitempty" cb:"ss,25"`
	LastPaymentCajaID int32 `json:",omitempty" db:"caja_id_" cb:"LastPaymentCajaID,26"`
	// If contains 2 = the payment is done
	// If contains 3 = the delivery of the product is done
	ActionsIncluded []int8 `json:",omitempty" cb:"ActionsIncluded,27"`
	// Audit trail fields for payment and delivery actions.
	LastPaymentTime int32                `json:",omitempty" cb:"LastPaymentTime,28"`
	LastPaymentUser int32                `json:",omitempty" cb:"LastPaymentUser,29"`
	DeliveryTime    int32                `json:",omitempty" cb:"DeliveryTime,30"`
	DeliveryUser    int32                `json:",omitempty" cb:"DeliveryUser,31"`
	ClientInfo      *SaleOrderClientInfo `json:",omitempty" cb:"ClientInfo,32"`
	// IssueSeriesID names the invoicing series this sale will be issued under. It
	// is a request field, not a column: it is folded into the last two digits of
	// the id on creation, and read back with SeriesID(). Zero when the till does
	// not say.
	IssueSeriesID  int8  `json:",omitempty" cb:"IssueSeriesID,33"`
	PaymentDueDate int16 `json:",omitempty" db:"payment_due_date" cb:"PaymentDueDate,34"`
	// AnnulReason is why the sale was annulled, required by the annul endpoint. There is no
	// AnnulledTime/AnnulledUser beside it: annulment is terminal, so Updated and UpdatedBy are
	// already the "when and by whom" and a second pair could only drift from them.
	AnnulReason string `json:",omitempty" cb:"AnnulReason,35"`
}

// SaleSummaryFrame is one sale line as the "sale-summary" DataFrame sums it: one file per day, one row
// per product and sub-unit divisor. Amounts are in soles; quantities are the split pair, never the
// packed document form, because a row adds many sales (1004 + 1004 is not 2008).
type SaleSummaryFrame struct {
	Date      int16 `frame:"key"`
	ProductID int32 `frame:"row"`
	// SubDivisor is its own row, so a product refined mid-day keeps sixths and twelfths apart: the
	// reader converts them, the frame never mixes them.
	SubDivisor    int16 `frame:"row"`
	AmountPEN     int32 `frame:"sum"`
	DebtAmountPEN int32 `frame:"sum"`
	SoldUnits     int32 `frame:"sum"`
	SoldSub       int32 `frame:"sum"`
	PendingUnits  int32 `frame:"sum"`
	PendingSub    int32 `frame:"sum"`
}

// ToSaleSummaryFrames is what the sale adds to its day summary, from its current state alone: an
// annulled sale adds nothing, an undelivered one keeps its quantities pending and an unpaid one its
// amount as debt. The frame subtracts what the sale added before each write, so no write has to
// undo anything by hand. It appends to lines, which a rebuild reuses from sale to sale.
func (e *SaleOrder) ToSaleSummaryFrames(lines []SaleSummaryFrame) []SaleSummaryFrame {
	if e.Status == OrderStatusAnnulled {
		return lines
	}
	isDeliveryPending := e.Status == OrderStatusPending || e.Status == OrderStatusPaid
	isPaymentPending := e.Status == OrderStatusPending || e.Status == OrderStatusDelivered
	for lineIndex, productID := range e.DetailProductsIDs {
		packedQuantity := core.GetIndex(e.DetailQuantities, lineIndex)
		if productID <= 0 || packedQuantity <= 0 {
			continue
		}
		quantity := core.UnpackQuantityLine(packedQuantity)
		subDivisor := core.GetIndex(e.DetailSubDivisor, lineIndex)
		if subDivisor <= 0 {
			subDivisor = core.QuantityDivisorNone
		}
		// Sub-units are charged at their own price, so the line total is exact, not prorated.
		line := SaleSummaryFrame{
			Date: e.Date, ProductID: productID, SubDivisor: subDivisor,
			AmountPEN: e.AmountInPEN(core.QuantityAmount(quantity,
				core.GetIndex(e.DetailPrices, lineIndex), core.GetIndex(e.DetailSubPrices, lineIndex))),
			SoldUnits: quantity.Units, SoldSub: quantity.Sub,
		}
		if isDeliveryPending {
			line.PendingUnits, line.PendingSub = quantity.Units, quantity.Sub
		}
		if isPaymentPending {
			line.DebtAmountPEN = line.AmountPEN
		}
		lines = append(lines, line)
	}
	return lines
}

// AmountInPEN restates an amount of this sale in soles, with the rate the sale was converted
// with — a USD sale always carries one. The sale summaries and the S/ 700 boleta rule are
// both in soles whatever the sale was charged in.
func (e *SaleOrder) AmountInPEN(amount int32) int32 {
	if e.CurrencyType != finance.CurrencyUSD {
		return amount
	}
	return finance.ConvertUnitPrice(amount, finance.CurrencyUSD, finance.CurrencyPEN, e.ExchangeRate)
}

// TotalInPEN is the total restated in soles (see AmountInPEN).
func (e *SaleOrder) TotalInPEN() int32 {
	return e.AmountInPEN(e.TotalAmount)
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
	CurrencyType               db.Col[*SaleOrderTable, int8]
	ExchangeRate               db.Col[*SaleOrderTable, int32]
	Created                    db.Col[*SaleOrderTable, int32]
	ClientID                   db.Col[*SaleOrderTable, int32]
	Updated                    db.Col[*SaleOrderTable, int32]
	UpdateCounter              db.Col[*SaleOrderTable, int32]
	Version                    db.Col[*SaleOrderTable, int16]
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
		// A plain key. The id is built by MakeSaleOrderID, which packs the series'
		// correlativo, two random digits and the series itself into it — a layout
		// the ORM's autoincrement cannot express.
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
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
		},
		DataFrames: []db.DataFrame{{
			// The day/product sales summary of /sales/sale_orders_charts. Most products have no
			// sub-units and most sales are settled, so the zero sums mask drops the Sub and
			// pending/debt columns of a day that has none. A read compacts the company's files
			// once 100 sales changed or an hour passed since the last compaction.
			Name: "sale-summary", Version: 1, UseZeroSumsMask: true,
			CompactOn: db.CompactOn{Records: 100, Seconds: 3600},
			ToFrames:  db.ToFrames((*SaleOrder).ToSaleSummaryFrames),
		}},
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
