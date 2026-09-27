// Turning a sale into the document that bills it.
//
// This lives in the types leaf rather than in the invoicing body because both
// sides of the flow need it: `sales` builds and numbers the document when the sale
// is created, and `invoicing` rebuilds it on every send. A module body may not
// import another module body, and everything here is legal for a leaf — db, core
// and other */types (docs/MODULE_BOUNDARIES.md).

package types

import (
	"app/core"
	crm "app/crm/types"
	"app/db"
	production "app/production/types"
	sales "app/sales/types"
	"errors"
	"fmt"

	"github.com/ivanjoz/facturago/model"
	"github.com/ivanjoz/facturago/utils"
)

// The ERP prices everything with IGV included — that is the number on the shelf
// and the number the customer pays — while SUNAT wants the operation split into
// a net value and the tax on it.
//
// The split is done here, in integers, and both halves are handed to facturago
// explicitly rather than letting it recompute them. That is what guarantees the
// invoice totals exactly what was charged: net is the gross rounded down through
// the rate, and the tax is whatever is left, so the two always add back up.
const (
	igvRateBasisPoints   = 1800  // 18.00 %
	igvGrossBasisPoints  = 11800 // 100 % + 18 %
	igvPercentHundredths = 1800
)

// splitGrossAmount divides an amount that includes IGV into its net value and
// the tax, exactly: net + tax == gross, always.
func splitGrossAmount(gross int64) (net int64, tax int64) {
	net = utils.MulDivRound(gross, 10_000, igvGrossBasisPoints)
	return net, gross - net
}

// SaleOrderToDocument turns a sale into the document that will be issued for it, and the
// identity that document bills — the value InvoiceDocument pins so the sales book can
// still read the same buyer years later.
//
// Everything the document needs beyond the sale itself — the customer, the product
// descriptions — is read here, because a document is a snapshot: once issued it must not
// change when a product is renamed.
//
// pinnedClientSnapshotID is `0` when the document is being reserved, and the identity the
// row already froze when it is being rebuilt for a send. A resend has to declare the same
// buyer it declared the first time, whatever CRM says today, so the rebuild reads the pin
// instead of resolving the customer again.
func SaleOrderToDocument(
	companyID int32, order *sales.SaleOrder, series *InvoiceSeries,
	pinnedClientSnapshotID int32,
) (*model.Document, int32, error) {

	if len(order.DetailProductsIDs) == 0 {
		return nil, 0, errors.New("la venta no tiene productos")
	}

	customer, clientSnapshotID, err := buildCustomer(
		companyID, order, series.DocType, pinnedClientSnapshotID)
	if err != nil {
		return nil, 0, err
	}
	products, err := loadProductDescriptions(companyID, order.DetailProductsIDs)
	if err != nil {
		return nil, 0, err
	}
	lines, _, err := buildLines(order, products)
	if err != nil {
		return nil, 0, err
	}

	document := &model.Document{
		Type:     sunatDocType(series.DocType),
		Series:   series.SeriesCode,
		IssuedAt: core.Now(),
		// The document bills what the sale charged: a USD sale's lines are already dollars.
		Currency: saleDocumentCurrency(order),
		Customer: customer,
		Lines:    lines,
		// A sale settled at the till is a cash sale. Credit terms would have to
		// come from the sale's payment plan, which the order does not carry yet.
		Payment: model.PaymentCash,
	}
	return document, clientSnapshotID, nil
}

// saleDocumentCurrency is the ISO 4217 code of the currency the sale was charged in.
func saleDocumentCurrency(order *sales.SaleOrder) string {
	if order.CurrencyType == CurrencyUSD {
		return "USD"
	}
	return model.DefaultCurrency
}

// buildCustomer resolves who is being billed, and which identity the row pins for them.
//
// The identity rules are checked here as well as when the sale was created, because a
// send happens later and from a different process: this is the last point where a
// document SUNAT would refuse can still be stopped.
func buildCustomer(
	companyID int32, order *sales.SaleOrder, docType int8, pinnedClientSnapshotID int32,
) (model.Party, int32, error) {

	customer, clientSnapshotID, err := readBuyerIdentity(
		companyID, order.ClientID, pinnedClientSnapshotID)
	if err != nil {
		return model.Party{}, 0, err
	}
	if err := ValidateCustomerIdentity(
		docType, order.TotalInPEN(), customer.LegalName, customer.DocNumber); err != nil {
		return model.Party{}, 0, err
	}

	// A small boleta usually has no customer at all: someone paid at the till and
	// left. SUNAT still requires the block, so an unidentified buyer is declared
	// as one — which is what every point of sale in the country does. There is nobody
	// to snapshot, so the id stays 0 and the book prints the same declared buyer.
	if docType != DocTypeFactura {
		if customer.DocNumber == "" {
			customer.DocType, customer.DocNumber = AnonymousDocType, AnonymousDocNumber
		}
		if customer.LegalName == "" {
			customer.LegalName = AnonymousBuyerName
		}
	}
	return customer, clientSnapshotID, nil
}

// readBuyerIdentity reads the buyer from the identity the document already pinned, or —
// while it is still being reserved and has none — from the client the sale resolved to.
//
// Those are the same identity at reservation and can diverge afterwards, which is the
// whole point: a client renamed between the reservation and the send must not change what
// this document declares. Creating the sale already wrote a walk-in's typed details into a
// ClientProvider row and pointed ClientID at it, so that row is the buyer either way.
func readBuyerIdentity(
	companyID, clientID, pinnedClientSnapshotID int32,
) (model.Party, int32, error) {

	if pinnedClientSnapshotID > 0 {
		snapshots := []crm.ClientProviderSnapshot{}
		query := db.Query(&snapshots)
		query.Select().CompanyID.Equals(companyID).ID.Equals(pinnedClientSnapshotID)
		if err := query.Exec(); err != nil {
			return model.Party{}, 0, fmt.Errorf("error al leer la identidad del cliente: %w", err)
		}
		if len(snapshots) == 0 {
			return model.Party{}, 0, fmt.Errorf(
				"el comprobante apunta a la identidad %v, que no existe", pinnedClientSnapshotID)
		}
		return model.Party{
			LegalName: snapshots[0].Name,
			DocNumber: snapshots[0].RegistryNumber,
			DocType:   crm.SunatIdentityDocCode(snapshots[0].IdentityDocType),
		}, pinnedClientSnapshotID, nil
	}

	if clientID == 0 {
		return model.Party{}, 0, nil
	}
	clients := []crm.ClientProvider{}
	query := db.Query(&clients)
	query.Select().CompanyID.Equals(companyID).ID.Equals(clientID)
	if err := query.Exec(); err != nil {
		return model.Party{}, 0, fmt.Errorf("error al leer el cliente: %w", err)
	}
	if len(clients) == 0 {
		return model.Party{}, 0, nil
	}
	return model.Party{
		LegalName: clients[0].Name,
		DocNumber: clients[0].RegistryNumber,
		DocType:   crm.SunatIdentityDocCode(clients[0].IdentityDocType),
	}, clients[0].SnapshotID, nil
}

// How an unidentified buyer is declared on a boleta.
//
// Exported because the Registro de Ventas has to print the same three values: a document
// with no pinned identity declared these, and the book may not invent different ones.
// AnonymousDocType is SUNAT's character, not a crm.IdentityDoc* id: it travels straight
// into model.Party, which speaks the catalog's printed form. It is the literal rather than
// crm.SunatIdentityDocCode(crm.IdentityDocNone) because a const cannot call a function —
// TestAnonymousDocTypeIsSunatNone is what keeps the two from drifting.
const (
	AnonymousDocType   = "0"
	AnonymousDocNumber = "00000000"
	AnonymousBuyerName = "CLIENTES VARIOS"
)

// buildLines turns the sale's parallel detail arrays into document lines.
//
// A sale line is packed (Units*QuantityLineScale + Sub) and its two halves were charged at
// different prices, so a line holding both becomes **two** document lines: whole units at
// DetailPrices, sub-units at DetailSubPrices. That keeps every emitted quantity a whole
// count — no fraction of a box ever reaches the XML — which matters because facturago
// scales quantities by 1e6 and a sixth of a unit is not representable there. It also
// matches what SUNAT wants: four candies invoice as four, not as 0.667 of a box.
// It takes the product descriptions rather than loading them, so the splitting rule can be
// tested without a database.
func buildLines(order *sales.SaleOrder, products map[int32]production.Product) ([]model.Line, []int32, error) {
	lines := make([]model.Line, 0, len(order.DetailProductsIDs))
	lineProductIDs := make([]int32, 0, len(order.DetailProductsIDs))

	appendLine := func(productID int32, description string, quantity int32, grossUnit int32, index int) {
		if quantity <= 0 || grossUnit <= 0 {
			return
		}
		// The line total is exact in cents because that is what was charged;
		// the split is what SUNAT reads.
		gross := int64(quantity) * int64(grossUnit)
		net, tax := splitGrossAmount(gross)
		lines = append(lines, model.Line{
			ProductCode: core.GetIndex(order.DetailProductSkus, index),
			Description: description,
			Quantity:    model.Units(int(quantity)),
			UnitCode:    model.DefaultUnitCode,
			IgvType:     model.IgvTaxed,
			IgvPercent:  model.Percent(igvPercentHundredths),
			Value:       model.Cents(net),
			IGV:         model.Cents(tax),
		})
		lineProductIDs = append(lineProductIDs, productID)
	}

	for index, productID := range order.DetailProductsIDs {
		lineQuantity := core.UnpackQuantityLine(core.GetIndex(order.DetailQuantities, index))

		description := products[productID].Name
		if description == "" {
			description = fmt.Sprintf("PRODUCTO %v", productID)
		}

		appendLine(productID, description, lineQuantity.Units, core.GetIndex(order.DetailPrices, index), index)

		if lineQuantity.Sub > 0 {
			// The sub-unit name is carried in the description because SUNAT's unit-of-measure
			// catalog (03) has no entry this free-text field maps to.
			subDescription := description
			if subUnit := products[productID].SbuUnit; subUnit != "" {
				subDescription = fmt.Sprintf("%s (%s)", description, subUnit)
			}
			appendLine(productID, subDescription, lineQuantity.Sub,
				core.GetIndex(order.DetailSubPrices, index), index)
		}
	}

	if len(lines) == 0 {
		return nil, nil, errors.New("la venta no tiene líneas con cantidad y precio válidos")
	}
	return lines, lineProductIDs, nil
}

// SaleOrderTaxAmount is the IGV inside the sale's total, split per line part exactly as
// buildLines splits it, so the sale and its comprobante never differ by a rounding cent.
// A sale with no comprobante still carries it: the sales book reads it from the sale.
func SaleOrderTaxAmount(order *sales.SaleOrder) int32 {
	taxAmount := int64(0)
	addLinePart := func(quantity int32, grossUnit int32) {
		if quantity <= 0 || grossUnit <= 0 {
			return
		}
		_, tax := splitGrossAmount(int64(quantity) * int64(grossUnit))
		taxAmount += tax
	}
	for index := range order.DetailProductsIDs {
		lineQuantity := core.UnpackQuantityLine(core.GetIndex(order.DetailQuantities, index))
		addLinePart(lineQuantity.Units, core.GetIndex(order.DetailPrices, index))
		addLinePart(lineQuantity.Sub, core.GetIndex(order.DetailSubPrices, index))
	}
	return int32(taxAmount)
}

// loadProductDescriptions reads what a document line needs to describe itself: the product
// name and, for a sub-unit line, the name of the sub-unit.
func loadProductDescriptions(companyID int32, productIDs []int32) (map[int32]production.Product, error) {
	descriptions := map[int32]production.Product{}
	if len(productIDs) == 0 {
		return descriptions, nil
	}

	products := []production.Product{}
	query := db.Query(&products)
	query.Select(query.ID, query.Name, query.SbuUnit).
		CompanyID.Equals(companyID).ID.In(productIDs...)

	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer los productos de la venta: %w", err)
	}
	for _, product := range products {
		descriptions[product.ID] = product
	}
	return descriptions, nil
}

// sunatDocType maps the stored numeric type to the catalog code facturago uses.
func sunatDocType(docType int8) model.DocType {
	switch docType {
	case DocTypeBoleta:
		return model.Boleta
	case DocTypeCreditNote:
		return model.CreditNote
	case DocTypeDebitNote:
		return model.DebitNote
	}
	return model.Factura
}
