// Creating a sale (SaleOrder) is one path for every place a sale is born: the till (PostSaleOrder),
// a web store order that is paid or delivered (settleOnlineSaleOrder) and a restaurant table that is
// charged (dining). Each caller prices its lines and names the series; the rest — debt, client,
// reservations, stock check, payment and delivery stamps, the sale's id and its comprobante, and the
// cash and stock movements — is decided here, so the three can never drift apart.
//
// It lives in its own package because it needs invoicing/types, config/types and crm/types, which
// all import sales/types.

package types

import (
	"app/cloud"
	"app/core"
	crm "app/crm/types"
	"app/db"
	finance "app/finance/types"
	invoicing "app/invoicing/types"
	logistics "app/logistics/types"
	production "app/production/types"
	sales "app/sales/types"
	"slices"
)

// NewSaleRequest is a sale ready to be created, as its caller resolved it.
type NewSaleRequest struct {
	// Sale holds the priced lines, TotalAmount, TaxAmount, CurrencyType (sales.ValidateSaleCurrency
	// already ran), ExchangeRate, ActionsIncluded (2 paid, 3 delivered), WarehouseID, ClientID or
	// ClientInfo, OnlineSaleOrderID and, when paid, LastPaymentCajaID and DebtAmount = what stays
	// owed after the payment.
	Sale sales.SaleOrder
	// Products are the catalog rows of the sale's products (sales.ReadSaleProducts).
	Products map[int32]production.Product
	// IssueSeries is the series of its comprobante (ResolveSaleIssueSeries); nil = no comprobante.
	IssueSeries *invoicing.InvoiceSeries
	// ReservedWarehouseID is where the order the sale comes from already reserved its units (a web
	// order or a restaurant table); 0 when nothing is reserved yet (the till).
	ReservedWarehouseID int32
	// IsOnlineReservation is whether those reservations are a web order's (PendingDeliveryWeb*).
	IsOnlineReservation bool
}

// PreparedSale is a sale with its id minted and every write it makes computed, waiting for its
// transaction (WriteNewSaleIn).
type PreparedSale struct {
	Sale                   sales.SaleOrder
	document               *invoicing.InvoiceDocument
	cashMovements          []finance.InternalCashMovement
	stockMovements         []logistics.InternalMovement
	pendingDeliveryChanges []logistics.PendingDeliveryChange
}

// PrepareNewSale validates the sale and mints its id, reserving its comprobante number when it has a
// series. Everything that can refuse it runs before the number is spent, so a refused sale leaves no
// gap in the series and nothing persisted (but a new client named by ClientInfo).
func PrepareNewSale(req *core.HandlerArgs, newSale NewSaleRequest) (PreparedSale, error) {
	companyID, userID := req.User.CompanyID, req.User.ID
	nowTime, today := core.SUnixTime(), core.FechaUnix()
	sale := newSale.Sale
	isPayment := slices.Contains(sale.ActionsIncluded, 2)
	isDelivery := slices.Contains(sale.ActionsIncluded, 3)

	sale.CompanyID, sale.Date, sale.Created = companyID, today, nowTime
	sale.Status, sale.Updated, sale.UpdatedBy = sales.OrderStatusPending, nowTime, userID

	if sale.ClientInfo != nil {
		clientID, err := ResolveSaleClientID(sale.ClientInfo, companyID, userID)
		if err != nil {
			return PreparedSale{}, core.Err("Error al guardar el cliente de la venta:", err)
		}
		sale.ClientID = clientID
	}

	// The detracción or retención of a factura, from its products, its client and the company.
	if newSale.IssueSeries != nil && newSale.IssueSeries.DocType == invoicing.DocTypeFactura {
		withholdingContext, err := saleWithholdingContext(companyID, sale.ClientID)
		if err != nil {
			return PreparedSale{}, err
		}
		sale.ApplySaleWithholdings(newSale.Products, withholdingContext)
	}

	// The server owns the debt: a new sale owes what the customer must pay the company (the total less
	// a retención), and a payment collects the drop to what the caller says stays owed. The caller
	// speaks only of what the till collects: an open detracción stays owed on top, until it is
	// deposited into the detracciones account, which a sale is never created paid into. Only a payment
	// names a cash-bank: an unpaid sale that kept the till's default one would later be paid into it,
	// whatever register the operator picks then.
	collectedAmount := int32(0)
	if isPayment {
		cashBank, err := finance.GetCaja(companyID, sale.LastPaymentCajaID)
		if err != nil {
			return PreparedSale{}, err
		}
		if cashBank.Type == finance.CashBankTypeDetractions {
			return PreparedSale{}, core.Err("Una venta nueva no se cobra en la cuenta de detracciones: registre el depósito después.")
		}
		remainingDebt := sale.DebtAmount + sale.OpenDetractionAmount()
		if collectedAmount, err = sales.SaleOrderCollectedAmount(sale.CollectibleAmount(), remainingDebt); err != nil {
			return PreparedSale{}, err
		}
		sale.DebtAmount = remainingDebt
	} else {
		sale.DebtAmount, sale.LastPaymentCajaID = sale.CollectibleAmount(), 0
	}
	if err := sale.StampPaymentDueDate(); err != nil {
		return PreparedSale{}, err
	}

	// The stock the sale holds while undelivered (ProductStock.PendingDelivery*). An undelivered sale
	// keeps what its order reserved, or reserves its units now; a delivery releases them from where
	// they were reserved, which it may not ship from. A made product's line is delivered without its
	// stock exit, which waits for the daily production close: it keeps (or takes) a reservation on
	// the delivery warehouse until then.
	isReserved := newSale.ReservedWarehouseID > 0
	pendingDeliveryChanges := []logistics.PendingDeliveryChange{}
	if isDelivery {
		sale.MarkStockExitPending(sales.MadeProductIDsOf(newSale.Products))
		pendingDeliveryChanges = sales.DeliveryPendingChanges(sale.Lines, newSale.ReservedWarehouseID,
			sale.WarehouseID, newSale.IsOnlineReservation)
	} else if !isReserved {
		pendingDeliveryChanges = sales.PendingDeliveryChanges(sale.Lines, sale.WarehouseID, newSale.IsOnlineReservation, 1)
	}

	// An early check on a plain read, before the number is minted: almost every shortage is refused
	// here with nothing spent. The delivery's movements re-check against the rows they lock inside
	// the transaction, so two tills selling the last units at once cannot both deliver; the loser
	// leaves a rare gap in the series (accepted, 2026-11-07). An undelivered sale whose units are
	// already reserved has nothing new to check. A made line takes the margin when it reserves on a
	// warehouse that did not hold it yet.
	if isDelivery || !isReserved {
		checksMadeLines := !isReserved || newSale.ReservedWarehouseID != sale.WarehouseID
		if err := sales.ValidateSaleStock(companyID, sale, isReserved, newSale.Products, checksMadeLines); err != nil {
			return PreparedSale{}, err
		}
	}

	if isPayment {
		if err := sales.StampSalePayment(&sale, userID, nowTime); err != nil {
			return PreparedSale{}, err
		}
	}
	isSameDayCollection := sales.StampSameDayCollection(&sale, collectedAmount, 0, today)
	sale.StampPeriod()
	if err := sale.StampSiteID(); err != nil {
		return PreparedSale{}, err
	}
	if isDelivery {
		if err := sales.StampSaleDelivery(&sale, userID, nowTime); err != nil {
			return PreparedSale{}, err
		}
	}

	// The sale id carries the correlativo of its series, so minting it and reserving the
	// comprobante's number are one act, and the document is validated before that number is spent.
	// Without a series the id comes from the series-0 counter, which nothing is declared from.
	prepared := PreparedSale{pendingDeliveryChanges: pendingDeliveryChanges}
	if newSale.IssueSeries != nil {
		document, err := invoicing.PrepareDocumentForSale(companyID, userID, &sale, newSale.IssueSeries, newSale.Products)
		if err != nil {
			return PreparedSale{}, core.Err("No se pudo generar el comprobante de la venta:", err)
		}
		prepared.document = document
	} else {
		saleID, err := sales.MakeSaleOrderID(companyID, 0)
		if err != nil {
			return PreparedSale{}, core.Err("Error al obtener el ID de la venta:", err)
		}
		sale.ID = saleID
	}

	prepared.cashMovements = sales.SaleCollectionMovements(&sale, collectedAmount, today, isSameDayCollection)
	if isDelivery {
		prepared.stockMovements = sales.SaleDeliveryMovements(&sale)
	}
	prepared.Sale = sale
	return prepared, nil
}

// WriteNewSaleIn writes a prepared sale in the caller's transaction: the sale, its comprobante, the
// collection with the cash-bank's balance, the delivery with the stock it takes, its reservations
// and its client's week row. Nothing can be left half written. A rerun of the transaction writes
// the same rows again: each step derives its writes from what it reads inside it.
func WriteNewSaleIn(transaction *db.Transaction, req *core.HandlerArgs, prepared *PreparedSale) error {
	companyID := req.User.CompanyID
	// The id was just reserved from the series counter: nothing is stored under it.
	if err := db.InsertIn(transaction, &[]sales.SaleOrder{prepared.Sale}); err != nil {
		return core.Err("Error al registrar la venta:", err)
	}
	// Every send rebuilds the document from the sale, which this transaction stores with it.
	if prepared.document != nil {
		if err := invoicing.SaveNewDocument(transaction, prepared.document); err != nil {
			return core.Err("No se pudo guardar el comprobante de la venta:", err)
		}
	}
	if err := finance.ApplyCashBankMovementIn(transaction, req, prepared.cashMovements); err != nil {
		return core.Err("Error al registrar el cobro de la venta:", err)
	}
	if len(prepared.stockMovements) > 0 {
		if err := logistics.ApplyMovimientosIn(transaction, req, prepared.stockMovements); err != nil {
			return core.Err("Error al procesar la salida de almacén:", err)
		}
	}
	if err := logistics.ApplyPendingDeliveryIn(transaction, companyID, req.User.ID, prepared.pendingDeliveryChanges); err != nil {
		return err
	}
	return sales.ApplyClientWeekSaleChangesIn(transaction, companyID, sales.ClientWeekSaleChanges(nil, &prepared.Sale))
}

// ScheduleNewSaleDocument sends the sale's comprobante, once its transaction committed.
func ScheduleNewSaleDocument(prepared *PreparedSale) {
	if prepared.document != nil {
		invoicing.ScheduleEmitPendingSweep(prepared.Sale.CompanyID)
	}
}

// saleWithholdingContext reads what decides a factura's retención: whether its client is a retention
// agent and whether the company is a buen contribuyente.
func saleWithholdingContext(companyID int32, clientID int32) (sales.SaleWithholdingContext, error) {
	isRetentionAgent, err := crm.IsClientRetentionAgent(companyID, clientID)
	if err != nil {
		return sales.SaleWithholdingContext{}, err
	}
	companyConfig, err := cloud.LoadCompanyConfig(companyID)
	if err != nil {
		return sales.SaleWithholdingContext{}, core.Err("Error al leer la configuración de la empresa:", err)
	}
	return sales.SaleWithholdingContext{IsFactura: true, IsRetentionAgent: isRetentionAgent,
		IsGoodTaxpayer: companyConfig.Tax.IsGoodTaxpayer}, nil
}
