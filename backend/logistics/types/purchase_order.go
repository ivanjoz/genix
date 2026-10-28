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

type PurchaseOrder struct {
	db.TableStruct[PurchaseOrderTable, PurchaseOrder]
	ID         int32 `cb:"ID,1"`
	CompanyID  int32 `cb:"CompanyID,2"`
	ProviderID int32 `cb:"ProviderID,3"`
	// ProviderSnapshotID freezes the supplier's name and RUC as of this purchase, in
	// crm.ClientProviderSnapshot. The Registro de Compras prints them (fields 10-12 of PLE
	// 8.3) and a filed book may not move when the supplier is later renamed.
	ProviderSnapshotID int32 `cb:"ProviderSnapshotID,4"`
	WarehouseID        int32 `cb:"WarehouseID,5"`
	Date               int16 `cb:"Date,6"`
	Week               int16 `cb:"Week,7"`
	DeliveryDate       int16 `cb:"DeliveryDate,8"`
	PaymentDate        int16 `cb:"PaymentDate,9"`
	// Parallel arrays in the same order — one row per line. Supplies and materials are
	// Product rows with Status=2, so they are ordinary product lines here and move stock
	// through the same reception path as anything else.
	DetailProductIDs             []int32 `cb:"DetailProductIDs,10"`
	DetailProductQuantity        []int32 `cb:"DetailProductQuantity,11"`
	DetailProductPrice           []int32 `cb:"DetailProductPrice,12"`
	DetailProductPresentationIDs []int32 `cb:"DetailProductPresentationIDs,13"`
	TotalAmount                  int32   `cb:"TotalAmount,14"`
	DebtAmount                   int32   `cb:"DebtAmount,15"`
	// The supplier's comprobante — the Registro de Compras row. See finance.PurchaseDocument.
	// Registered with its own action, because the invoice usually arrives after the order and
	// often after the goods. Its total is the invoice's, which may differ from TotalAmount.
	DocType       int8   `cb:"DocType,16"`
	DocSeries     string `cb:"DocSeries,17"`
	DocNumber     int64  `cb:"DocNumber,18"`
	DocIssueDate  int16  `cb:"DocIssueDate,19"`
	TaxableAmount int32  `cb:"TaxableAmount,20"`
	TaxAmount     int32  `cb:"TaxAmount,21"`
	UntaxedAmount int32  `cb:"UntaxedAmount,22"`
	OtherAmount   int32  `cb:"OtherAmount,23"`
	CurrencyType  int8   `cb:"CurrencyType,24"`
	ExchangeRate  int32  `cb:"ExchangeRate,25"`
	// DifferenceValue is the signed money gap between what was ordered and what arrived.
	// There is no DifferenceQuantity companion: it summed quantities across product lines,
	// so it added kilograms to screws and had no unit. Money is the figure that sums.
	DifferenceValue int32  `cb:"DifferenceValue,26"`
	Notes           string `cb:"Notes,27"`

	Created   int32 `cb:"Created,28"`
	CreatedBy int32 `cb:"CreatedBy,29"`
	Updated   int32 `json:"upd" cb:"upd,30"`
	Version   int16 `json:"ver" cb:"ver,31"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata  *string `json:"mdt" cb:"mdt,32"`
	UpdatedBy int32   `cb:"UpdatedBy,33"`
	Status    int8    `json:"ss" cb:"ss,34"`
}

type PurchaseOrderTable struct {
	db.TableStruct[PurchaseOrderTable, PurchaseOrder]
	CompanyID                    db.Col[*PurchaseOrderTable, int32]
	ID                           db.Col[*PurchaseOrderTable, int32]
	ProviderID                   db.Col[*PurchaseOrderTable, int32]
	ProviderSnapshotID           db.Col[*PurchaseOrderTable, int32]
	WarehouseID                  db.Col[*PurchaseOrderTable, int32]
	Date                         db.Col[*PurchaseOrderTable, int16]
	Week                         db.Col[*PurchaseOrderTable, int16]
	DeliveryDate                 db.Col[*PurchaseOrderTable, int16]
	PaymentDate                  db.Col[*PurchaseOrderTable, int16]
	DetailProductIDs             db.Col[*PurchaseOrderTable, []int32]
	DetailProductQuantity        db.Col[*PurchaseOrderTable, []int32]
	DetailProductPrice           db.Col[*PurchaseOrderTable, []int32]
	DetailProductPresentationIDs db.Col[*PurchaseOrderTable, []int32]
	TotalAmount                  db.Col[*PurchaseOrderTable, int32]
	DebtAmount                   db.Col[*PurchaseOrderTable, int32]
	DocType                      db.Col[*PurchaseOrderTable, int8]
	DocSeries                    db.Col[*PurchaseOrderTable, string]
	DocNumber                    db.Col[*PurchaseOrderTable, int64]
	DocIssueDate                 db.Col[*PurchaseOrderTable, int16]
	TaxableAmount                db.Col[*PurchaseOrderTable, int32]
	TaxAmount                    db.Col[*PurchaseOrderTable, int32]
	UntaxedAmount                db.Col[*PurchaseOrderTable, int32]
	OtherAmount                  db.Col[*PurchaseOrderTable, int32]
	CurrencyType                 db.Col[*PurchaseOrderTable, int8]
	ExchangeRate                 db.Col[*PurchaseOrderTable, int32]
	DifferenceValue              db.Col[*PurchaseOrderTable, int32]
	Notes                        db.Col[*PurchaseOrderTable, string]
	Created                      db.Col[*PurchaseOrderTable, int32]
	CreatedBy                    db.Col[*PurchaseOrderTable, int32]
	Updated                      db.Col[*PurchaseOrderTable, int32]
	Version                      db.Col[*PurchaseOrderTable, int16]
	UpdatedBy                    db.Col[*PurchaseOrderTable, int32]
	Status                       db.Col[*PurchaseOrderTable, int8]
}

func (e PurchaseOrderTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:               24,
		Name:             "purchase_order",
		Partition:        e.CompanyID,
		UseListAsDefault: true,
		Keys:             db.Cols(e.ID.Autoincrement(0)),
		// Delta() enumerates its filter column, so every Status value must be declared. Listing them
		// rather than a 0..4 range keeps the delta fan-out at four values instead of five.
		FixedValues: []db.FixedValues{
			{Col: e.Status, Values: []int64{
				int64(PurchaseOrderStatusCanceled), int64(PurchaseOrderStatusPending),
				int64(PurchaseOrderStatusConfirmed), int64(PurchaseOrderStatusFulfilled),
			}},
		},
		Indexes: []db.Index{
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
			{
				Keys:          db.Cols(e.Week),
				UseIndexGroup: true,
			},
			{
				Keys:          db.Cols(e.Week, e.DetailProductIDs),
				UseIndexGroup: true,
			},
			{
				Keys:          db.Cols(e.Week, e.Status, e.DetailProductIDs),
				UseIndexGroup: true,
			},
			{
				Keys:          db.Cols(e.Week, e.ProviderID),
				UseIndexGroup: true,
			},
			{
				Keys:          db.Cols(e.Week, e.Status, e.ProviderID),
				UseIndexGroup: true,
			},
			// The Registro de Compras reads a period of documents, one indexed read per day.
			{Keys: db.Cols(e.DocIssueDate), UseIndexGroup: true},
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
