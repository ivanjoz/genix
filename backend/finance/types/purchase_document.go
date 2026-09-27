// The supplier's comprobante a purchase was booked with.
//
// The Registro de Compras (SIRE RCE, Anexo 11 of RS 112-2021) is a register of comprobantes,
// not of orders: a purchase order, an expense and a fixed asset are each RCE rows only through
// the document the supplier issued for them. The three tables carry the same document columns,
// and this file holds the rules they share, so an electricity bill entered as an expense and a
// factura entered on a purchase order are validated identically. It lives in finance because
// the currency and the exchange rate a document is validated against live here.
//
// Field numbers in the comments are Anexo 11. See frontend/routes/accounting/books/PLAN.md §2.2.

package types

import (
	"app/core"
	crm "app/crm/types"
	"fmt"
	"math"
	"strings"
)

// Purchase document types: the SUNAT catalog 01 ids a purchase can be booked with (field 7).
// PurchaseDocTypeNone is not SUNAT's: the purchase has no comprobante, so it is not an RCE row.
//
// Credit and debit notes (07/08) are deliberately absent. A note needs the document it
// modifies (fields 28-32), and where a supplier note lives is still an open decision.
const (
	PurchaseDocTypeNone        int8 = 0
	PurchaseDocTypeFactura     int8 = 1
	PurchaseDocTypeFeeReceipt  int8 = 2 // Recibo por honorarios.
	PurchaseDocTypeBoleta      int8 = 3
	PurchaseDocTypeTicket      int8 = 12 // Ticket de máquina registradora.
	PurchaseDocTypeUtilityBill int8 = 14 // Recibo de servicios públicos.
)

type PurchaseDocTypeOption struct {
	ID   int8
	Name string
	// CreditsTax is 1 when the document can carry IGV with crédito fiscal (fields 15-16). A
	// boleta or a recibo por honorarios grants none, so its whole amount is untaxed (field 21).
	CreditsTax int8
}

var PurchaseDocTypeOptions = []PurchaseDocTypeOption{
	{ID: PurchaseDocTypeFactura, Name: "Invoice|Factura", CreditsTax: 1},
	{ID: PurchaseDocTypeFeeReceipt, Name: "Fee receipt|Recibo por Honorarios"},
	{ID: PurchaseDocTypeBoleta, Name: "Receipt|Boleta"},
	{ID: PurchaseDocTypeTicket, Name: "Register ticket|Ticket", CreditsTax: 1},
	{ID: PurchaseDocTypeUtilityBill, Name: "Utility bill|Recibo de Servicios Públicos", CreditsTax: 1},
}

// purchaseDocSeriesMax is Anexo 11's cap on the series (field 8).
const purchaseDocSeriesMax = 20

// PurchaseDocument is the document columns of a purchase, lifted out of whichever table holds
// them so the rules below are written once.
type PurchaseDocument struct {
	DocType       int8
	DocSeries     string
	DocNumber     int64 // Numeric on every type the ERP books; supermarket tickets run past int32.
	DocIssueDate  int16 // UnixDay. Picks the RCE period.
	TaxableAmount int32 // Field 15, base imponible with crédito fiscal.
	TaxAmount     int32 // Field 16, its IGV.
	UntaxedAmount int32 // Field 21, exonerado + inafecto.
	OtherAmount   int32 // Field 24, charges outside the base.
	CurrencyType  int8
	ExchangeRate  int32 // Field 27, × 1000. Only when the currency is not PEN.
}

// Total is field 25. It is never stored: the annex defines it as the sum, so storing it would
// only create a second number that could disagree.
func (document PurchaseDocument) Total() int32 {
	return document.TaxableAmount + document.TaxAmount + document.UntaxedAmount + document.OtherAmount
}

// NormalizePurchaseDocument validates a document and puts it in its stored shape.
//
// purchaseDate is the record's own date. A purchase with no comprobante stores it as
// DocIssueDate, so the single DocIssueDate index also finds the purchases the book lists as
// "sin comprobante" — for those rows the column reads as "the date the purchase is booked on".
//
// dueDate is the record's payment due date, which a recibo de servicios públicos must declare
// (field 6). supplierID is who issued it: fields 12-14 are mandatory on every RCE row.
func NormalizePurchaseDocument(
	document *PurchaseDocument, supplierID int32, purchaseDate int16, dueDate int16,
) error {
	if document.DocType == PurchaseDocTypeNone {
		*document = PurchaseDocument{DocIssueDate: purchaseDate, CurrencyType: document.CurrencyType}
		return nil
	}

	option := findPurchaseDocType(document.DocType)
	if option == nil {
		return core.Err("El tipo de comprobante de compra no es válido.")
	}
	if supplierID <= 0 {
		return core.Err("Un comprobante de compra debe indicar el proveedor que lo emitió.")
	}

	document.DocSeries = strings.ToUpper(strings.TrimSpace(document.DocSeries))
	if document.DocSeries == "" || document.DocNumber <= 0 {
		return core.Err("Debe indicar la serie y el número del comprobante del proveedor.")
	}
	if len(document.DocSeries) > purchaseDocSeriesMax {
		return core.Err("La serie del comprobante admite hasta 20 caracteres.")
	}
	if !isSunatAlphanumeric(document.DocSeries) {
		return core.Err("La serie del comprobante sólo admite letras, números y guiones.")
	}

	if document.DocIssueDate <= 0 {
		return core.Err("Debe indicar la fecha de emisión del comprobante del proveedor.")
	}
	if document.DocIssueDate > core.FechaUnix() {
		return core.Err("La fecha de emisión del comprobante no puede ser futura.")
	}
	if document.DocType == PurchaseDocTypeUtilityBill && dueDate <= 0 {
		return core.Err("Un recibo de servicios públicos debe indicar su fecha de vencimiento.")
	}

	if document.TaxableAmount < 0 || document.TaxAmount < 0 ||
		document.UntaxedAmount < 0 || document.OtherAmount < 0 {
		return core.Err("Los importes del comprobante no pueden ser negativos.")
	}
	if document.Total() <= 0 {
		return core.Err("El comprobante debe tener un importe mayor a 0.")
	}
	if option.CreditsTax == 0 && (document.TaxableAmount != 0 || document.TaxAmount != 0) {
		return core.Err("Un comprobante sin crédito fiscal (boleta, recibo por honorarios) " +
			"se registra entero como importe no gravado.")
	}
	if document.TaxAmount > 0 && document.TaxableAmount == 0 {
		return core.Err("El IGV del comprobante necesita una base imponible.")
	}

	document.CurrencyType = core.If(document.CurrencyType == 0, CurrencyPEN, document.CurrencyType)
	switch document.CurrencyType {
	case CurrencyPEN:
		document.ExchangeRate = 0
	case CurrencyUSD:
		if document.ExchangeRate <= 0 || document.ExchangeRate > ExchangeRateMax {
			return core.Err("Un comprobante en dólares debe indicar el tipo de cambio de su fecha de emisión.")
		}
	default:
		return core.Err("Moneda del comprobante inválida (1 = PEN, 2 = USD).")
	}
	return nil
}

// InventoryUnitCost is what one unit bought on this document costs the stock, in PEN cents:
// the gross unit price the purchase was entered at, less the IGV the company recovers as
// crédito fiscal, restated in soles at the document's rate.
//
// The IGV comes off in the document's own proportion — net over total — so a comprobante
// mixing taxed and untaxed lines, or carrying other charges, spreads the same way over every
// unit. A document with no recoverable IGV (boleta, no comprobante yet) keeps it in the cost.
// A dollar document with no rate returns 0: the unit enters uncosted rather than at a guess.
func (document PurchaseDocument) InventoryUnitCost(grossUnitPrice int32) int32 {
	netUnitPrice := grossUnitPrice
	if total := document.Total(); document.TaxAmount > 0 && total > 0 {
		netUnitPrice = int32(math.Round(float64(grossUnitPrice) * float64(total-document.TaxAmount) / float64(total)))
	}
	if document.CurrencyType != CurrencyUSD {
		return netUnitPrice
	}
	if document.ExchangeRate <= 0 {
		return 0
	}
	return ConvertUnitPrice(netUnitPrice, CurrencyUSD, CurrencyPEN, document.ExchangeRate)
}

// ApplyPurchaseDocument validates the supplier's comprobante on an expense and pins the
// supplier's identity. The document is the bill for this expense, so its total is the
// expense's Amount — two different figures would put two different purchases in the books.
func (expense *Expense) ApplyPurchaseDocument(companyID int32) error {
	document := expense.PurchaseDocument()
	if err := NormalizePurchaseDocument(
		&document, expense.SupplierID, expense.Date, expense.DueDate,
	); err != nil {
		return err
	}
	if document.DocType != PurchaseDocTypeNone && document.Total() != expense.Amount {
		return core.Err(fmt.Sprintf("El total del comprobante (%.2f) no coincide con el monto del gasto (%.2f).",
			float64(document.Total())/100, float64(expense.Amount)/100))
	}
	expense.SetPurchaseDocument(document)

	snapshotID, err := crm.ProviderSnapshotID(companyID, expense.SupplierID)
	if err != nil {
		return err
	}
	expense.ProviderSnapshotID = snapshotID
	return nil
}

func findPurchaseDocType(docType int8) *PurchaseDocTypeOption {
	for index := range PurchaseDocTypeOptions {
		if PurchaseDocTypeOptions[index].ID == docType {
			return &PurchaseDocTypeOptions[index]
		}
	}
	return nil
}

// isSunatAlphanumeric keeps a series to what fits a pipe-separated SUNAT record.
func isSunatAlphanumeric(value string) bool {
	for _, character := range value {
		isLetter := (character >= 'A' && character <= 'Z') || (character >= 'a' && character <= 'z')
		isDigit := character >= '0' && character <= '9'
		if !isLetter && !isDigit && character != '-' {
			return false
		}
	}
	return true
}
