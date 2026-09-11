// The invoicing half of creating a sale: which series it is issued under, and the
// electronic document that is written with it.
//
// The stamp is permanent — the series lives in the last two digits of the sale id,
// which is the sale's identity everywhere else in the system — and the document is
// created here rather than later, so a sale that cannot be invoiced is refused
// while the operator is still looking at the screen. Everything that can fail runs
// before the sale is written.

package sales

import (
	"app/cloud"
	"app/core"
	crm "app/crm/types"
	"app/db"
	invoicing "app/invoicing/types"
	"app/sales/types"
)

// resolveSaleOrderIssueSeries checks the series the till named and the buyer the
// document will have to identify, and returns the series the sale will be issued
// under. A nil series means the till named none and no document will exist.
func resolveSaleOrderIssueSeries(req *core.HandlerArgs, sale *types.SaleOrder) (*invoicing.InvoiceSeries, error) {
	if sale.IssueSeriesID == 0 {
		return nil, nil // the till named no series: no document will ever be issued for this sale
	}

	// The company config is a cached blob (memory, then object storage), so this
	// costs no cluster read per sale. It can be up to twenty seconds stale, which
	// only matters for a series activated or retired in that window.
	companyConfig, err := cloud.LoadCompanyConfig(req.User.CompanyID)
	if err != nil {
		return nil, core.Err("Error al leer la configuración de la empresa:", err)
	}

	series := invoicing.FindSeries(companyConfig.Sunat.Series, sale.IssueSeriesID)
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

	if !invoicing.RequiresCustomerIdentity(series.DocType, sale.TotalAmount) {
		return series, nil
	}

	name, registryNumber, err := saleOrderCustomerIdentity(req.User.CompanyID, sale)
	if err != nil {
		return nil, err
	}
	if err := invoicing.ValidateCustomerIdentity(
		series.DocType, sale.TotalAmount, name, registryNumber); err != nil {
		return nil, err
	}
	return series, nil
}

// saleOrderCustomerIdentity is who the sale will be billed to, resolved exactly the
// way the document builder resolves it: the client row is the base and the details
// typed at the till override it field by field. Reading it any other way would let
// a sale pass here and fail at emission.
func saleOrderCustomerIdentity(companyID int32, sale *types.SaleOrder) (string, string, error) {
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
