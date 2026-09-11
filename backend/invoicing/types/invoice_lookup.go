// Invoice lookups that other modules have to make. This is business logic in a `types` package
// for the same reason ApplyCashBankMovement is: a module body may not import another module body
// (see backend/docs/MODULE_BOUNDARIES.md), and `sales` has to ask whether a sale was invoiced
// before it will annul it. Living here is what lets `sales` call it while importing only
// `app/invoicing/types`.
//
// It depends on nothing but db and this package's own tables, so it stays a leaf.

package types

import (
	"app/core"
	"app/db"
	"fmt"
)

// FindBySaleOrder answers whether a sale has already been invoiced. Asked before
// every emission, because the second document for one sale is the mistake that
// cannot be undone without a credit note — and before an annulment, because a sale
// with a live document cannot simply vanish either.
//
// It is a primary-key read: the document that bills a sale is keyed by that sale.
// A sale whose tail is 00 was not registered for electronic invoicing and has no
// document, so this correctly finds nothing for it.
func FindBySaleOrder(companyID int32, saleOrderID int64) (*InvoiceDocument, error) {
	documents := []InvoiceDocument{}
	query := db.Query(&documents)
	query.Select().CompanyID.Equals(companyID).ID.Equals(saleOrderID)

	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al verificar si la venta ya fue facturada: %w", err)
	}
	for index := range documents {
		// A rejected document does not exist for SUNAT, and neither does one
		// that was discarded, so in both cases the sale may be invoiced again.
		if documents[index].Status == 0 || documents[index].State == InvoiceRejected {
			continue
		}
		return &documents[index], nil
	}
	return nil, nil
}

// VoidPendingDocumentForSale clears the way for a sale to be annulled.
//
// Every sale issued under a series carries a document from the moment it is
// created, so an annulment always finds one. What decides the outcome is whether
// it ever left: a document still waiting for the sweep is voided here and never
// sent, and one that reached SUNAT is returned so the caller can refuse.
//
// A nil document means the sale is free to annul. The correlativo of a voided
// document stays spent — the series keeps a gap that SUNAT expects to be declared
// through a comunicación de baja, which this system does not emit yet.
//
// The caller must hold the sale's ActionInvoiceSaleOrder lock: the sweep re-reads
// the state inside that same lock, and that is what stops it from sending a
// document while this is voiding it.
func VoidPendingDocumentForSale(companyID int32, saleOrderID int64) (*InvoiceDocument, error) {
	document, err := FindBySaleOrder(companyID, saleOrderID)
	if err != nil || document == nil {
		return nil, err
	}
	if document.State != InvoicePending {
		return document, nil
	}

	document.State = InvoiceVoided
	document.Status = 0
	document.Updated = core.SUnixTime()

	table := db.TableOf[InvoiceDocument]()
	rows := &[]InvoiceDocument{*document}
	// State travels with the delta view's key, which the ORM requires to be
	// written together with Status and Updated.
	if err := db.Update(rows, table.State, table.Status, table.Updated); err != nil {
		return nil, fmt.Errorf("no se pudo anular el comprobante de la venta: %w", err)
	}
	return nil, nil
}
