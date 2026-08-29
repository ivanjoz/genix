package invoicing

import (
	"app/core"
	crm "app/crm/types"
	"app/db"
	"app/invoicing/types"
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

// SaleOrderToDocument turns a sale into the document that will be issued for it.
//
// Everything the document needs beyond the sale itself — the customer's identity
// document, the product descriptions — is read here, because a document is a
// snapshot: once issued it must not change when a product is renamed.
// A line with a sub-unit part becomes two document lines, so the second return value maps
// each generated line back to the product it came from.
func SaleOrderToDocument(
	companyID int32, order *sales.SaleOrder, series *types.InvoiceSeries,
) (*model.Document, []int32, error) {

	if len(order.DetailProductsIDs) == 0 {
		return nil, nil, errors.New("la venta no tiene productos")
	}

	customer, err := buildCustomer(companyID, order, series.DocType)
	if err != nil {
		return nil, nil, err
	}
	products, err := loadProductDescriptions(companyID, order.DetailProductsIDs)
	if err != nil {
		return nil, nil, err
	}
	lines, lineProductIDs, err := buildLines(order, products)
	if err != nil {
		return nil, nil, err
	}

	issuedAt := core.Now()
	document := &model.Document{
		Type:     sunatDocType(series.DocType),
		Series:   series.SeriesCode,
		IssuedAt: issuedAt,
		Currency: model.DefaultCurrency,
		Customer: customer,
		Lines:    lines,
		// A sale settled at the till is a cash sale. Credit terms would have to
		// come from the sale's payment plan, which the order does not carry yet.
		Payment: model.PaymentCash,
	}
	return document, lineProductIDs, nil
}

// buildCustomer resolves who is being billed.
//
// A factura is business to business and SUNAT only accepts a RUC on one, so the
// check happens here rather than after a correlativo has been spent.
func buildCustomer(companyID int32, order *sales.SaleOrder, docType int8) (model.Party, error) {
	name, registryNumber := "", ""

	if order.ClientID > 0 {
		clients := []crm.ClientProvider{}
		query := db.Query(&clients)
		query.Select().CompanyID.Equals(companyID).ID.Equals(order.ClientID)
		if err := query.Exec(); err != nil {
			return model.Party{}, fmt.Errorf("error al leer el cliente: %w", err)
		}
		if len(clients) > 0 {
			name, registryNumber = clients[0].Name, clients[0].RegistryNumber
		}
	}
	// A walk-in sale carries the buyer's details on the order itself.
	if order.ClientInfo != nil {
		if order.ClientInfo.Name != "" {
			name = order.ClientInfo.Name
		}
		if order.ClientInfo.RegistryNumber != "" {
			registryNumber = order.ClientInfo.RegistryNumber
		}
	}

	customer := model.Party{
		DocNumber: registryNumber,
		LegalName: name,
		DocType:   identityDocTypeOf(registryNumber),
	}

	// A factura always names a business, so a missing RUC is an error.
	if docType == types.DocTypeFactura {
		if customer.DocType != model.IDDocRUC {
			return model.Party{}, errors.New(
				"una factura necesita un cliente con RUC de 11 dígitos")
		}
		if customer.LegalName == "" {
			return model.Party{}, errors.New("el cliente de la factura no tiene nombre")
		}
		return customer, nil
	}

	// A boleta usually has no customer at all: someone paid at the till and
	// left. SUNAT still requires the block, so an unidentified buyer is declared
	// as one — which is what every point of sale in the country does.
	if customer.DocNumber == "" {
		customer.DocType = model.IDDocNone
		customer.DocNumber = anonymousDocNumber
	}
	if customer.LegalName == "" {
		customer.LegalName = anonymousCustomerName
	}
	return customer, nil
}

// How an unidentified buyer is declared on a boleta.
const (
	anonymousDocNumber    = "00000000"
	anonymousCustomerName = "CLIENTES VARIOS"
)

// identityDocTypeOf reads the kind of identity document from its shape, which is
// unambiguous in Peru: eleven digits is a RUC and eight is a DNI.
//
// A dedicated column on the client would be better and is planned; until then
// this is the same inference the rest of the ERP makes.
func identityDocTypeOf(registryNumber string) string {
	switch len(registryNumber) {
	case 11:
		return model.IDDocRUC
	case 8:
		return model.IDDocDNI
	case 0:
		return ""
	}
	return model.IDDocForeign
}

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
	case types.DocTypeBoleta:
		return model.Boleta
	case types.DocTypeCreditNote:
		return model.CreditNote
	case types.DocTypeDebitNote:
		return model.DebitNote
	}
	return model.Factura
}
