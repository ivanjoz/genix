package types

import (
	"app/core"
	"app/db"
	finance "app/finance/types"
	"fmt"
)

const (
	PurchaseOrderStatusCanceled  int8 = 0
	PurchaseOrderStatusPending   int8 = 1
	PurchaseOrderStatusConfirmed int8 = 2
	PurchaseOrderStatusFulfilled int8 = 4
)

// PurchaseOrderLine is one product line of a purchase order. Price is the gross price of one
// unit, in the order's currency.
type PurchaseOrderLine struct {
	ProductID      int32 `cb:"1"`
	Quantity       int32 `cb:"2"`
	Price          int32 `cb:"3"`
	PresentationID int16 `cb:"4"`
	// AccountCode is the PCGE account a supply line posts to, copied from the supply's default
	// and editable per line. 0 on a product line: merchandise always posts to 6011.
	AccountCode int32 `cb:"5"`
}

// DetailProductIDs is the product of every line, in line order. It feeds the computed column of
// the same name, which the product index groups hash.
func (e *PurchaseOrder) DetailProductIDs() []int32 {
	productIDs := make([]int32, 0, len(e.Lines))
	for _, line := range e.Lines {
		productIDs = append(productIDs, line.ProductID)
	}
	return productIDs
}

type PurchaseOrder struct {
	db.TableStruct[PurchaseOrderTable, PurchaseOrder]
	ID         int32 `cb:"1"`
	CompanyID  int32 `cb:"2"`
	ProviderID int32 `cb:"3"`
	// ProviderSnapshotID freezes the supplier's name and RUC as of this purchase, in
	// crm.ClientProviderSnapshot. The Registro de Compras prints them (fields 10-12 of PLE
	// 8.3) and a filed book may not move when the supplier is later renamed.
	ProviderSnapshotID int32 `cb:"4"`
	WarehouseID        int32 `cb:"5"`
	Date               int16 `cb:"6"`
	Week               int16 `cb:"7"`
	DeliveryDate       int16 `cb:"8"`
	PaymentDate        int16 `cb:"9"`
	// Lines is stored as one colbin blob. Supplies and materials are Product rows with Status=2,
	// so they are ordinary product lines here and move stock through the same reception path as
	// anything else. Its product IDs are indexed through the computed DetailProductIDs column.
	Lines       []PurchaseOrderLine `cb:"10"`
	TotalAmount int32               `cb:"14"`
	DebtAmount  int32               `cb:"15"`
	// The supplier's comprobante — the Registro de Compras row. See finance.PurchaseDocument.
	// Registered with its own action, because the invoice usually arrives after the order and
	// often after the goods. Its total is the invoice's, which may differ from TotalAmount.
	DocType       int8   `cb:"16"`
	DocSeries     string `cb:"17"`
	DocNumber     int64  `cb:"18"`
	DocIssueDate  int16  `cb:"19"`
	TaxableAmount int32  `cb:"20"`
	TaxAmount     int32  `cb:"21"`
	UntaxedAmount int32  `cb:"22"`
	OtherAmount   int32  `cb:"23"`
	CurrencyType  int8   `cb:"24"`
	ExchangeRate  int32  `cb:"25"`
	// DifferenceValue is the signed money gap between what was ordered and what arrived.
	// There is no DifferenceQuantity companion: it summed quantities across product lines,
	// so it added kilograms to screws and had no unit. Money is the figure that sums.
	DifferenceValue int32  `cb:"26"`
	Notes           string `cb:"27"`

	Created   int32 `cb:"28"`
	CreatedBy int32 `cb:"29"`
	Updated   int32 `json:"upd" cb:"upd,30"`
	Version   int16 `json:"ver" cb:"ver,31"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata  *string `json:"mdt" cb:"mdt,32"`
	UpdatedBy int32   `cb:"33"`
	Status    int8    `json:"ss" cb:"ss,34"`
	// Period is the YYMM of DocIssueDate (core.PeriodOfDay): the pinned key of the period delta
	// index the accounting operations sync a month through. Set by StampPeriod on every write.
	Period int16 `cb:"35"`
	// SameDayCashBankID is the cash bank of the one supplier payment that settled the whole order
	// on its DocIssueDate; 0 when no payment did. See finance.SameDaySettlement.
	SameDayCashBankID int32 `cb:"36"`
}

// ReceptionAccountOf is the account a received product posts to: its supply line's account, else
// merchandise 6011 (a product line carries none). It is stamped on the reception's movements.
func (e *PurchaseOrder) ReceptionAccountOf(productID int32) int32 {
	for _, line := range e.Lines {
		if line.ProductID == productID && line.AccountCode != 0 {
			return line.AccountCode
		}
	}
	return finance.PCGEMerchandisePurchase
}

// StampPeriod sets Period from the order's accounting date, its DocIssueDate (the order's own
// date when it has no comprobante, see finance.NormalizePurchaseDocument).
func (e *PurchaseOrder) StampPeriod() {
	e.Period = core.PeriodOfDay(e.DocIssueDate)
}

type PurchaseOrderTable struct {
	db.TableStruct[PurchaseOrderTable, PurchaseOrder]
	CompanyID          db.Col[*PurchaseOrderTable, int32]
	ID                 db.Col[*PurchaseOrderTable, int32]
	ProviderID         db.Col[*PurchaseOrderTable, int32]
	ProviderSnapshotID db.Col[*PurchaseOrderTable, int32]
	WarehouseID        db.Col[*PurchaseOrderTable, int32]
	Date               db.Col[*PurchaseOrderTable, int16]
	Week               db.Col[*PurchaseOrderTable, int16]
	DeliveryDate       db.Col[*PurchaseOrderTable, int16]
	PaymentDate        db.Col[*PurchaseOrderTable, int16]
	Lines              db.Col[*PurchaseOrderTable, []PurchaseOrderLine]
	// DetailProductIDs has no field in PurchaseOrder: GetSchema computes it from Lines.
	DetailProductIDs db.Col[*PurchaseOrderTable, []int32]
	TotalAmount      db.Col[*PurchaseOrderTable, int32]
	DebtAmount       db.Col[*PurchaseOrderTable, int32]
	DocType          db.Col[*PurchaseOrderTable, int8]
	DocSeries        db.Col[*PurchaseOrderTable, string]
	DocNumber        db.Col[*PurchaseOrderTable, int64]
	DocIssueDate     db.Col[*PurchaseOrderTable, int16]
	TaxableAmount    db.Col[*PurchaseOrderTable, int32]
	TaxAmount        db.Col[*PurchaseOrderTable, int32]
	UntaxedAmount    db.Col[*PurchaseOrderTable, int32]
	OtherAmount      db.Col[*PurchaseOrderTable, int32]
	CurrencyType     db.Col[*PurchaseOrderTable, int8]
	ExchangeRate     db.Col[*PurchaseOrderTable, int32]
	DifferenceValue  db.Col[*PurchaseOrderTable, int32]
	Notes            db.Col[*PurchaseOrderTable, string]
	Created          db.Col[*PurchaseOrderTable, int32]
	CreatedBy        db.Col[*PurchaseOrderTable, int32]
	Updated          db.Col[*PurchaseOrderTable, int32]
	Version          db.Col[*PurchaseOrderTable, int16]
	UpdatedBy        db.Col[*PurchaseOrderTable, int32]
	Status           db.Col[*PurchaseOrderTable, int8]
	Period           db.Col[*PurchaseOrderTable, int16]
	// SameDayCashBankID is the cash bank of the payment that settled the order on its DocIssueDate.
	SameDayCashBankID db.Col[*PurchaseOrderTable, int32]
}

func (e PurchaseOrderTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:               24,
		Name:             "purchase_order",
		Partition:        e.CompanyID,
		UseListAsDefault: true,
		// A period's reception of an older order reaches it through the by-IDs cache.
		CacheByIDs: true,
		Keys:       db.Cols(e.ID.Autoincrement(0)),
		// Delta() enumerates its filter column, so every Status value must be declared. Listing them
		// rather than a 0..4 range keeps the delta fan-out at four values instead of five.
		FixedValues: []db.FixedValues{
			{Col: e.Status, Values: []int64{
				int64(PurchaseOrderStatusCanceled), int64(PurchaseOrderStatusPending),
				int64(PurchaseOrderStatusConfirmed), int64(PurchaseOrderStatusFulfilled),
			}},
			{Col: e.Period, Min: 0, Max: 9912},
		},
		// Only the product index groups below read it, from the record in memory.
		Computed: []db.ComputedCol{
			{Col: e.DetailProductIDs, From: db.Cols(e.Lines), Value: db.Compute((*PurchaseOrder).DetailProductIDs)},
		},
		Indexes: []db.Index{
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
			// One log per accounting period: GET.accounting-operations syncs a month through it.
			{Type: db.TypeDelta, Keys: db.Cols(e.Period, e.Status)},
			{
				Type: db.TypeCacheGroup,
				Keys: db.Cols(e.Week),
			},
			// No Week+Status+DetailProductIDs group: Status changes at every step of an order, and
			// each change rewrites one entry per product (~1,200 index writes per order measured).
			// A status + product query reads this group and the client filters the status.
			{
				Type: db.TypeCacheGroup,
				Keys: db.Cols(e.Week, e.DetailProductIDs),
			},
			{
				Type: db.TypeCacheGroup,
				Keys: db.Cols(e.Week, e.ProviderID),
			},
			{
				Type: db.TypeCacheGroup,
				Keys: db.Cols(e.Week, e.Status, e.ProviderID),
			},
			// The Registro de Compras reads a period of documents, one indexed read per day.
			{Type: db.TypeCacheGroup, Keys: db.Cols(e.DocIssueDate)},
		},
	}
}

func (e *PurchaseOrder) PurchaseDocument() finance.PurchaseDocument {
	return finance.PurchaseDocument{
		DocType: e.DocType, DocSeries: e.DocSeries, DocNumber: e.DocNumber, DocIssueDate: e.DocIssueDate,
		TaxableAmount: e.TaxableAmount, TaxAmount: e.TaxAmount, UntaxedAmount: e.UntaxedAmount,
		OtherAmount: e.OtherAmount, CurrencyType: e.CurrencyType, ExchangeRate: e.ExchangeRate,
	}
}

func (e *PurchaseOrder) SetPurchaseDocument(document finance.PurchaseDocument) {
	e.DocType, e.DocSeries, e.DocNumber = document.DocType, document.DocSeries, document.DocNumber
	e.DocIssueDate = document.DocIssueDate
	e.TaxableAmount, e.TaxAmount = document.TaxableAmount, document.TaxAmount
	e.UntaxedAmount, e.OtherAmount = document.UntaxedAmount, document.OtherAmount
	e.CurrencyType, e.ExchangeRate = document.CurrencyType, document.ExchangeRate
}

// ApplyPurchaseDocument validates the supplier's comprobante carried on the order and puts it in
// its stored shape. The comprobante is the bill for the order, so its total must be TotalAmount.
// Date and PaymentDate must already be set: they are the booking date and the due date.
func (e *PurchaseOrder) ApplyPurchaseDocument() error {
	document := e.PurchaseDocument()
	if err := finance.NormalizePurchaseDocument(&document, e.ProviderID, e.Date, e.PaymentDate); err != nil {
		return err
	}
	if document.DocType != finance.PurchaseDocTypeNone && document.Total() != e.TotalAmount {
		return core.Err(fmt.Sprintf("El total del comprobante (%.2f) no coincide con el total de la orden (%.2f).",
			float64(document.Total())/100, float64(e.TotalAmount)/100))
	}
	e.SetPurchaseDocument(document)
	return nil
}
