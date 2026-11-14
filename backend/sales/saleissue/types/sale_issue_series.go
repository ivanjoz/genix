// The invoicing half of issuing a sale: which series it is issued under, and who the document
// will be billed to.
//
// The stamp is permanent — the series lives in the last two digits of the sale id, which is the
// sale's identity everywhere else in the system — and the document is created when the sale is,
// so a sale that cannot be invoiced is refused while the operator is still looking at the screen.

package types

import (
	"app/core"
	crm "app/crm/types"
	"app/db"
	invoicing "app/invoicing/types"
	sales "app/sales/types"
)

// ResolveSaleIssueSeries checks the series the sale names (IssueSeriesID) and the buyer the document
// will have to identify, and returns the series the sale will be issued under. A nil series means
// the sale named none and no document will exist. allSeries are the company's
// (cloud.LoadCompanyConfig(...).Sunat.Series): cloud imports sales/types, so the caller loads them.
// The sale must be priced: the S/ 700 identity threshold reads its total.
func ResolveSaleIssueSeries(companyID int32, sale *sales.SaleOrder, allSeries []invoicing.InvoiceSeries) (*invoicing.InvoiceSeries, error) {
	if sale.IssueSeriesID == 0 {
		return nil, nil // no series: no document will ever be issued for this sale
	}

	series := invoicing.FindSeries(allSeries, sale.IssueSeriesID)
	if series == nil {
		return nil, core.Err("La serie indicada para la venta no existe.")
	}
	if series.Status != 1 {
		return nil, core.Err("La serie", series.SeriesCode, "está inactiva y no puede emitir comprobantes.")
	}
	// A note corrects a document that already went out; issuing a sale under one
	// would mint an id no document can use.
	if invoicing.IsNote(series.DocType) {
		return nil, core.Err("Una venta no puede registrarse con una serie de nota de crédito o débito.")
	}

	// The S/ 700 threshold is in soles whatever the sale is charged in.
	totalInPEN := sale.TotalInPEN()
	if !invoicing.RequiresCustomerIdentity(series.DocType, totalInPEN) {
		return series, nil
	}

	name, registryNumber, err := saleCustomerIdentity(companyID, sale)
	if err != nil {
		return nil, err
	}
	if err := invoicing.ValidateCustomerIdentity(
		series.DocType, totalInPEN, name, registryNumber); err != nil {
		return nil, err
	}
	return series, nil
}

// saleCustomerIdentity is who the sale will be billed to, resolved exactly the way the document
// builder resolves it: the client row is the base and the details typed at the till override it
// field by field. Reading it any other way would let a sale pass here and fail at emission.
func saleCustomerIdentity(companyID int32, sale *sales.SaleOrder) (string, string, error) {
	name, registryNumber := "", ""

	if sale.ClientID > 0 {
		clients := []crm.ClientProvider{}
		query := db.Query(&clients)
		query.Select(query.ID, query.Name, query.RegistryNumber).
			CompanyID.Equals(companyID).ID.Equals(sale.ClientID)

		if err := query.Exec(); err != nil {
			return "", "", core.Err("Error al leer el cliente de la venta:", err)
		}
		if len(clients) == 0 {
			return "", "", core.Err("El cliente indicado en la venta no existe.")
		}
		name, registryNumber = clients[0].Name, clients[0].RegistryNumber
	}

	if sale.ClientInfo != nil {
		if sale.ClientInfo.Name != "" {
			name = sale.ClientInfo.Name
		}
		if sale.ClientInfo.RegistryNumber != "" {
			registryNumber = sale.ClientInfo.RegistryNumber
		}
	}
	return name, registryNumber, nil
}
