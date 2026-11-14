package sales

import (
	business "app/business/types"
	config "app/config/types"
	"app/core"
	"app/db"
	finance "app/finance/types"
	invoicing "app/invoicing/types"
	"app/libs/culqi"
	logistics "app/logistics/types"
	production "app/production/types"
	saleissue "app/sales/saleissue/types"
	"app/sales/types"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
)

// The web store checkout: POST.p-web-checkout places an OnlineSaleOrder for the logged-in web
// customer, charges it through Culqi when it is paid online, and answers with the order number. See
// docs/WEB_CHECKOUT_PLAN.md §3 and docs/ONLINE_SALE_ORDER_PLAN.md.
//
// The online order is inserted pending BEFORE the charge, so a declined charge leaves an annulled
// order (an audit trail) and never money without an order: no refund path is needed. No SaleOrder
// exists until the money (or the goods) moves, so a declined or abandoned order spends no boleta
// number. A Culqi that cannot be reached leaves the order pending, because then nobody knows whether
// the money moved; its onlineSaleOrderID travels in the charge's metadata for reconciliation.

// webMaxPendingOrdersPerCustomer caps the unpaid orders one customer can hold: a deferred order
// costs nothing to place, so without a cap one session could flood the store with them.
const webMaxPendingOrdersPerCustomer = 3

// webDevSimulatedPaymentToken stands in for a Culqi token in local testing: the checkout settles
// the order as paid without calling Culqi. Honored only by a `go run . dev` process and only for
// company 1 (the dev company); anywhere else it reaches Culqi and is rejected as an invalid token.
const webDevSimulatedPaymentToken = "dev_simulated_payment"

type webCheckoutLine struct {
	ProductID      int32
	PresentationID int16
	// Quantity is whole units: the web store does not sell sub-units.
	Quantity int32
}

type webCheckoutRequest struct {
	Lines        []webCheckoutLine
	DeliveryType int8 // types.DeliveryTypeHome / types.DeliveryTypePickup
	PickupSiteID int32
	// PaymentMethod is types.WebPaymentCard, WebPaymentYape or WebPaymentDeferred.
	PaymentMethod int8
	// CulqiToken is the card ("tkn_") or Yape ("ype_") token; empty on a deferred payment.
	CulqiToken string
	// AddressID picks a saved address for home delivery. 0 means "Nueva dirección": the address
	// in Shipping is used and saved to the customer's addresses.
	AddressID int32
	// Shipping carries the contact always (the email is taken from the session, never from here)
	// and the address only when AddressID is 0.
	Shipping types.SaleOrderShipping
}

// Document kinds the customer receives for an order.
const (
	webOrderDocumentNone          = int8(0)
	webOrderDocumentReceipt       = int8(1) // a boleta, issued with the sale when the order is paid
	webOrderDocumentPurchaseOrder = int8(2) // the Orden de Compra of an order paid later
)

// PostWebCheckout places a web order. See the file comment for the order of the steps.
func PostWebCheckout(req *core.HandlerArgs) core.HandlerResponse {
	companyID, customer, err := requireWebCustomer(req)
	if err != nil {
		return req.MakeErrCode(err.Error(), 401)
	}
	checkout := webCheckoutRequest{}
	if err := json.Unmarshal([]byte(*req.Body), &checkout); err != nil {
		return req.MakeErr("Error al deserializar el body: " + err.Error())
	}
	companyConfig, err := loadWebStoreConfig(companyID)
	if err != nil {
		return req.MakeErr(err)
	}
	companyFlags := companyConfig.Company.Flags

	// The sale helpers shared with the till read the company from req.User. A web order has no
	// back-office user: it is written by the system (user 0) for the store the session belongs to.
	req.User = &core.UsuarioToken{CompanyID: companyID}

	// One checkout per customer at a time, so the pending-orders cap below cannot be raced.
	checkoutLock, lockErr := core.AcquireLock(context.Background(), core.ActionWebCheckoutByCustomer,
		int64(companyID)<<32|int64(customer.ID), 1)
	if lockErr != nil {
		return lockErr.Response(req)
	}
	defer checkoutLock.Release()

	// ── 1. Validate the request ────────────────────────────────────────────────────────────
	if len(checkout.Lines) == 0 {
		return req.MakeErr("El carrito está vacío.")
	}
	// pricing is a SaleOrder only to run the till's pricing rules on the lines; it is never stored.
	pricing := types.SaleOrder{CompanyID: companyID, CurrencyType: finance.CurrencyPEN}
	for _, line := range checkout.Lines {
		if line.ProductID <= 0 || line.Quantity <= 0 || line.Quantity > core.MaxQuantityLineUnits {
			return req.MakeErr("Hay un producto con una cantidad inválida.")
		}
		pricing.Lines = append(pricing.Lines, types.SaleOrderLine{
			ProductID: line.ProductID, PresentationID: line.PresentationID,
			Quantity: line.Quantity * core.QuantityLineScale,
		})
	}

	isOnlinePayment := types.IsOnlineWebPayment(checkout.PaymentMethod)
	switch checkout.PaymentMethod {
	case types.WebPaymentCard:
	case types.WebPaymentYape:
		if !companyConfig.Culqi.YapeEnabled {
			return req.MakeErr("La tienda no acepta pagos con Yape.")
		}
	case types.WebPaymentDeferred:
		if !core.HasCompanyFlag(companyFlags, types.CompanyFlagWebCashOnDelivery) {
			return req.MakeErr("La tienda no acepta el pago contra entrega.")
		}
	default:
		return req.MakeErr("Debe elegir un medio de pago.")
	}
	if isOnlinePayment && strings.TrimSpace(checkout.CulqiToken) == "" {
		return req.MakeErr("Falta el token de pago de Culqi.")
	}
	if !isOnlinePayment && checkout.CulqiToken != "" {
		return req.MakeErr("Un pedido con pago contra entrega no lleva un token de pago.")
	}
	isSimulatedPayment := isOnlinePayment && checkout.CulqiToken == webDevSimulatedPaymentToken &&
		core.Env.IS_DEV_ARG && companyID == 1

	// The contact: the email is the session's, the rest is typed and checked like Mis Datos.
	contact := webCustomerProfile{FirstName: checkout.Shipping.FirstName, LastName: checkout.Shipping.LastName,
		Phone: checkout.Shipping.Phone, DocumentNumber: checkout.Shipping.DocumentNumber}
	if err := validateWebCustomerProfile(&contact, companyFlags); err != nil {
		return req.MakeErr(err)
	}
	order := types.OnlineSaleOrder{
		CompanyID: companyID, WebCustomerID: customer.ID, PaymentMethod: checkout.PaymentMethod,
		Shipping: types.SaleOrderShipping{FirstName: contact.FirstName, LastName: contact.LastName,
			Email: customer.Email, Phone: contact.Phone, DocumentNumber: contact.DocumentNumber},
	}

	// The destination.
	var newAddress *types.WebCustomerAddress
	switch checkout.DeliveryType {
	case types.DeliveryTypePickup:
		isPickupSite := slices.ContainsFunc(companyConfig.Sites, func(site config.CompanyConfigSite) bool {
			return site.ID == checkout.PickupSiteID && site.IsPickupPoint && site.Status > 0
		})
		if !isPickupSite {
			return req.MakeErr("La tienda elegida no es un punto de recojo.")
		}
		order.PickupSiteID = checkout.PickupSiteID
	case types.DeliveryTypeHome:
		address := types.WebCustomerAddress{CityID: checkout.Shipping.CityID,
			Address: checkout.Shipping.Address, Reference: checkout.Shipping.Reference}
		if checkout.AddressID > 0 {
			savedAddresses, err := listWebCustomerAddresses(companyID, customer.ID)
			if err != nil {
				return req.MakeErr(err)
			}
			savedIndex := slices.IndexFunc(savedAddresses, func(saved types.WebCustomerAddress) bool {
				return saved.ID == checkout.AddressID
			})
			if savedIndex < 0 {
				return req.MakeErr("La dirección elegida no pertenece al cliente.")
			}
			address = savedAddresses[savedIndex]
		} else {
			if err := validateWebCustomerAddress(&address); err != nil {
				return req.MakeErr(err)
			}
			newAddress = &address
		}
		order.Shipping.CityID, order.Shipping.Address, order.Shipping.Reference = address.CityID, address.Address, address.Reference
	default:
		return req.MakeErr("Debe elegir entrega a domicilio o recojo en tienda.")
	}
	order.DeliveryType = checkout.DeliveryType

	// ── 2. Price from the catalog, plus the delivery ─────────────────────────────────────────
	saleProducts, err := validateSaleOrderLines(req, &pricing)
	if err != nil {
		return req.MakeErr(err)
	}
	if order.DeliveryType == types.DeliveryTypeHome {
		if pricing.ShippingAmount, err = resolveWebShippingAmount(companyID, order.Shipping.CityID); err != nil {
			return req.MakeErr(err)
		}
		pricing.TotalAmount += pricing.ShippingAmount
	}
	order.Lines, order.ExchangeRate = pricing.Lines, pricing.ExchangeRate
	order.TotalAmount, order.ShippingAmount = pricing.TotalAmount, pricing.ShippingAmount

	// With flag 12 the sale this order becomes issues a boleta, which must name its buyer from
	// S/ 700: the DNI is asked now, while the shopper is here, and the CRM client created from it.
	issuesReceipt := core.HasCompanyFlag(companyFlags, types.CompanyFlagWebReceiptAutomatic)
	if issuesReceipt {
		receiptSeries := invoicing.FindWebStoreSeries(companyConfig.Sunat.Series)
		if receiptSeries == nil {
			return req.MakeErr(`La tienda no tiene una serie de boleta marcada "Usar en Tienda Web".`)
		}
		if invoicing.RequiresCustomerIdentity(receiptSeries.DocType, pricing.TotalInPEN()) {
			if order.Shipping.DocumentNumber == "" {
				return req.MakeErr("Para compras desde S/ 700 debe ingresar su DNI.")
			}
			clientID, err := saleissue.ResolveSaleClientID(&types.SaleOrderClientInfo{
				Name: order.Shipping.FirstName + " " + order.Shipping.LastName, RegistryNumber: order.Shipping.DocumentNumber,
			}, companyID, 0)
			if err != nil {
				return req.MakeErr("Error al registrar el cliente del comprobante:", err)
			}
			order.ClientID = clientID
		}
	}

	// ── 3. Stock ────────────────────────────────────────────────────────────────────────
	if order.PendingWarehouseID, err = validateWebStock(companyID, order.Lines, saleProducts); err != nil {
		return req.MakeErr(err)
	}

	// The money must have somewhere to land before a card is charged.
	var culqiCashBank *finance.CashBank
	if isOnlinePayment {
		if culqiCashBank, err = finance.FindCulqiCashBank(companyID); err != nil {
			return req.MakeErr(err)
		}
		if culqiCashBank == nil {
			return req.MakeErr("La tienda no tiene configurada la cuenta que recibe los pagos de Culqi.")
		}
		if types.NormalizeCurrency(culqiCashBank.CurrencyType) != finance.CurrencyPEN {
			return req.MakeErr("La cuenta que recibe los pagos de Culqi debe ser en soles.")
		}
		if companyConfig.Culqi.SecretKey() == "" && !isSimulatedPayment {
			return req.MakeErr("La tienda no tiene configurada la llave secreta de Culqi.")
		}
	} else {
		pendingOrders, err := countPendingWebOrders(companyID, customer.ID)
		if err != nil {
			return req.MakeErr(err)
		}
		if pendingOrders >= webMaxPendingOrdersPerCustomer {
			return req.MakeErr(fmt.Sprintf("Tiene %v pedidos pendientes de pago. Espere a que se completen para hacer otro.", pendingOrders))
		}
	}

	// ── 4. Insert the online order as pending ────────────────────────────────────────────────
	nowTime := core.SUnixTime()
	order.Date = core.FechaUnix()
	order.Created, order.Updated = nowTime, nowTime
	order.Status = types.OrderStatusPending
	order.DebtAmount = order.TotalAmount
	// The order and the units it reserves commit together. A rerun after a deadlock inserts from the
	// unsaved order again, not from the ID a failed attempt was given.
	insertedOrder := order
	err = db.RunTransaction(func(transaction *db.Transaction) error {
		orders := []types.OnlineSaleOrder{order}
		if err := db.InsertIn(transaction, &orders); err != nil {
			return core.Err("Error al registrar el pedido:", err)
		}
		insertedOrder = orders[0]
		return logistics.ApplyPendingDeliveryIn(transaction, companyID, 0,
			types.PendingDeliveryChanges(order.Lines, order.PendingWarehouseID, true, 1))
	})
	if err != nil {
		return req.MakeErr(err)
	}
	order = insertedOrder
	core.Log("PostWebCheckout:: pedido registrado", order.ID, "cliente", customer.ID, "total", order.TotalAmount)

	// Remembered for the next checkout. A failure here costs a retyped address, not the order.
	if newAddress != nil {
		if _, err := saveWebCustomerAddresses(companyID, customer.ID, []types.WebCustomerAddress{*newAddress}); err != nil {
			core.Log("PostWebCheckout:: no se pudo guardar la dirección nueva", err)
		}
	}
	rememberWebCustomerContact(customer, contact)

	if !isOnlinePayment {
		// ── Deferred payment stops here: the order waits, the Orden de Compra is mailed. Its sale
		// is created in Pedidos Web when the payment or the delivery is registered.
		sendWebOrderEmail(companyID, order, true)
		return makeWebCheckoutResponse(req, order, webOrderDocumentPurchaseOrder)
	}

	// ── 5. Charge Culqi ───────────────────────────────────────────────────────────────────
	var charge *culqi.Charge
	var chargeErr error
	if isSimulatedPayment {
		// Dev testing: a fake charge ID, so the rest of the flow (sale, boleta, email) runs as if paid.
		charge = &culqi.Charge{ID: fmt.Sprintf("dev_sim_%v", order.ID), Amount: order.TotalAmount}
		core.Log("PostWebCheckout:: pago simulado (dev) para el pedido", order.ID)
	} else {
		charge, chargeErr = culqi.CreateCharge(companyConfig.Culqi.SecretKey(), culqi.ChargeRequest{
			Amount: order.TotalAmount, CurrencyCode: "PEN", Email: customer.Email, SourceID: checkout.CulqiToken,
			Metadata: map[string]string{"onlineSaleOrderID": fmt.Sprintf("%v", order.ID), "webCustomerID": fmt.Sprintf("%v", customer.ID)},
		})
	}
	declined := &culqi.ChargeError{}
	if errors.As(chargeErr, &declined) {
		// ── 6b. Declined: the order is annulled. No sale and no boleta number exist to undo.
		if _, annulErr := annulOnlineSaleOrder(companyID, order.ID, "Culqi: "+declined.UserMessage, 0); annulErr != nil {
			core.Log("PostWebCheckout:: no se pudo anular el pedido rechazado", order.ID, annulErr)
		}
		return req.MakeErr(declined.UserMessage)
	}
	if chargeErr != nil {
		core.Log("PostWebCheckout:: cargo de Culqi sin respuesta clara para el pedido", order.ID, chargeErr)
		return req.MakeErr(fmt.Sprintf("No se pudo confirmar el pago. Su pedido N° %v quedó pendiente: "+
			"no vuelva a pagar, la tienda lo verificará con Culqi.", order.ID))
	}

	// ── 6. Charged: the charge is recorded first, on its own, so the money is traceable even if
	// creating the sale fails; Pedidos Web then shows the order as charged and finishes it.
	order.CulqiChargeID = charge.ID
	orderTable := db.TableOf[types.OnlineSaleOrder]()
	if err := db.Update(&[]types.OnlineSaleOrder{order}, orderTable.CulqiChargeID, orderTable.Updated); err != nil {
		core.Log("PostWebCheckout:: no se pudo guardar el cargo de Culqi", order.ID, charge.ID, err)
	}
	paidOrder, sale, err := settleOnlineSaleOrder(req, onlineSaleOrderSettleRequest{
		ID: order.ID, ActionsIncluded: []int8{2}, CashBankID: culqiCashBank.ID, // 2 = payment
	}, charge.ID)
	if err != nil {
		// The money moved: this order must be finished by the store, never charged again.
		core.Log("PostWebCheckout:: cobrado en Culqi pero sin venta", order.ID, charge.ID, err)
		return req.MakeErr(fmt.Sprintf("Su pago fue recibido (Culqi %v) pero no pudo registrarse en el pedido N° %v. "+
			"La tienda lo completará; no vuelva a pagar.", charge.ID, order.ID))
	}
	sendWebOrderEmail(companyID, paidOrder, false)
	documentType := core.If(issuesReceipt, webOrderDocumentReceipt, webOrderDocumentNone)
	response := makeWebCheckoutResponse(req, paidOrder, documentType)
	core.Log("PostWebCheckout:: pedido", paidOrder.ID, "pagado, venta", sale.ID)
	return response
}

func makeWebCheckoutResponse(req *core.HandlerArgs, order types.OnlineSaleOrder, documentType int8) core.HandlerResponse {
	return req.MakeResponse(map[string]any{
		"OnlineSaleOrderID": order.ID,
		"OrderNumber":       webOrderNumber(order.ID),
		"SaleOrderID":       order.SaleOrderID,
		"TotalAmount":       order.TotalAmount,
		"ShippingAmount":    order.ShippingAmount,
		"Status":            order.Status,
		"DocumentType":      documentType,
	})
}

// resolveWebShippingAmount reads the district's delivery rate, falling back to its province's and
// then its department's. No rate on any level means the store does not deliver there.
func resolveWebShippingAmount(companyID, districtID int32) (int32, error) {
	shippingCosts := []types.ShippingCost{}
	query := db.Query(&shippingCosts)
	query.Select(query.CityID, query.FlatAmount).CompanyID.Equals(companyID).
		CityID.In(types.ShippingRateCityIDs(districtID)...)
	if err := query.Exec(); err != nil {
		return 0, core.Err("Error al obtener el costo de envío:", err)
	}
	ratesByCityID := map[int32]int32{}
	for _, shippingCost := range shippingCosts {
		ratesByCityID[shippingCost.CityID] = shippingCost.FlatAmount
	}
	shippingAmount, found := types.ResolveShippingAmount(districtID, ratesByCityID)
	if !found {
		return 0, core.Err("No hacemos envíos a este distrito.")
	}
	return shippingAmount, nil
}

// GetWebShippingAmount answers the storefront's "how much is delivery to this district"
// (?cid=&city-id=), with the same fallback the checkout charges. Public and session-less: rates
// are what the store shows any visitor.
func GetWebShippingAmount(req *core.HandlerArgs) core.HandlerResponse {
	companyID := webStoreCompanyID(req)
	districtID := int32(req.GetQueryInt("city-id"))
	if companyID <= 0 || districtID < 10000 {
		return req.MakeErr("Debe indicar la tienda y el distrito.")
	}
	shippingAmount, err := resolveWebShippingAmount(companyID, districtID)
	if err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(map[string]any{"ShippingAmount": shippingAmount})
}

// validateWebStock checks the order against the web store's sellable rule (logistics.WebOrderFits):
// per presentation, the stock of the warehouses open to online sales less what undelivered sales and
// web orders reserved, with the product's WebExtraStock shared by its presentations. A line with
// sub-units needs one more whole unit, as its reservation does. Stock is only checked here, never
// moved: the delivery is registered in the back office, from whichever warehouse ships it.
//
// It returns the warehouse the order's units are reserved on: the first (lowest ID) of those.
func validateWebStock(companyID int32, lines []types.SaleOrderLine, productsByID map[int32]production.Product) (int32, error) {
	// A product hidden from the store may still sit in a cart cached before it was hidden, and a line
	// may name a presentation the product no longer sells.
	presentationIDsByProduct := map[int32][]int16{}
	for _, line := range lines {
		product := productsByID[line.ProductID]
		if product.WebHidden {
			return 0, core.Err(fmt.Sprintf(`El producto "%v" ya no está disponible en la tienda.`, product.Name))
		}
		presentationIDsByProduct[line.ProductID] = product.WebPresentationIDs()
		if !slices.Contains(presentationIDsByProduct[line.ProductID], line.PresentationID) {
			return 0, core.Err(fmt.Sprintf(`La presentación elegida de "%v" ya no está disponible.`, product.Name))
		}
	}

	warehouses := []business.Warehouse{}
	warehouseQuery := db.Query(&warehouses)
	warehouseQuery.Select(warehouseQuery.ID, warehouseQuery.Status, warehouseQuery.ExcludeFromOnlineSales).
		CompanyID.Equals(companyID)
	if err := warehouseQuery.Exec(); err != nil {
		return 0, core.Err("Error al obtener los almacenes:", err)
	}
	pendingWarehouseID := business.FirstOnlineWarehouseID(warehouses)
	if pendingWarehouseID == 0 {
		return 0, core.Err("La tienda no tiene almacenes habilitados para la venta en línea.")
	}

	// What the order takes of each product, by presentation, in the units its reservation will hold.
	orderedUnitsByProduct := map[int32]map[int16]int32{}
	for _, change := range types.PendingDeliveryChanges(lines, pendingWarehouseID, true, 1) {
		if orderedUnitsByProduct[change.ProductID] == nil {
			orderedUnitsByProduct[change.ProductID] = map[int16]int32{}
		}
		orderedUnitsByProduct[change.ProductID][change.PresentationID] += change.Online
	}
	stockRowsByProduct, err := logistics.ReadWebStockRows(companyID, business.OnlineWarehouseIDs(warehouses), presentationIDsByProduct)
	if err != nil {
		return 0, err
	}
	for productID, orderedUnits := range orderedUnitsByProduct {
		product := productsByID[productID]
		if logistics.WebOrderFits(stockRowsByProduct[productID], presentationIDsByProduct[productID], product.WebExtraStock, orderedUnits) {
			continue
		}
		// One presentation ordered: say how many are left. Several share the margin, so no single
		// figure is right for each.
		if len(orderedUnits) == 1 {
			sellableUnits := logistics.WebSellableStock(stockRowsByProduct[productID], presentationIDsByProduct[productID], product.WebExtraStock)
			for presentationID := range orderedUnits {
				return 0, core.Err(fmt.Sprintf(`No hay stock suficiente de "%v": quedan %v.`, product.Name, sellableUnits[presentationID]))
			}
		}
		return 0, core.Err(fmt.Sprintf(`No hay stock suficiente de "%v" para las presentaciones elegidas.`, product.Name))
	}
	return pendingWarehouseID, nil
}

// countPendingWebOrders counts the customer's online orders still waiting for payment or delivery.
func countPendingWebOrders(companyID, customerID int32) (int, error) {
	orders := []types.OnlineSaleOrder{}
	query := db.Query(&orders)
	query.Select(query.ID, query.Status).CompanyID.Equals(companyID).WebCustomerID.Equals(customerID)
	if err := query.Exec(); err != nil {
		return 0, core.Err("Error al obtener los pedidos del cliente:", err)
	}
	pending := 0
	for _, order := range orders {
		if order.Status == types.OrderStatusPending {
			pending++
		}
	}
	return pending, nil
}

// rememberWebCustomerContact fills, on the customer, the contact fields they had left empty, so the
// next checkout starts with them. It never overwrites (Mis Datos does) nor confirms the email.
func rememberWebCustomerContact(customer *types.WebCustomer, contact webCustomerProfile) {
	if customer.Phone != "" && customer.DocumentNumber != "" {
		return
	}
	customer.Phone = core.If(customer.Phone == "", contact.Phone, customer.Phone)
	customer.DocumentNumber = core.If(customer.DocumentNumber == "", contact.DocumentNumber, customer.DocumentNumber)
	customer.Updated = core.SUnixTime()
	customerTable := db.TableOf[types.WebCustomer]()
	if err := db.Update(&[]types.WebCustomer{*customer}, customerTable.Phone, customerTable.DocumentNumber,
		customerTable.Updated); err != nil {
		core.Log("rememberWebCustomerContact:: no se pudo guardar el contacto", err)
	}
}

// sendWebOrderEmail confirms the order to the customer; a deferred order carries its Orden de
// Compra. A failed email is logged: the order stands, and the PDF is downloadable from Mis Pedidos.
func sendWebOrderEmail(companyID int32, order types.OnlineSaleOrder, withPurchaseOrder bool) {
	purchaseDocument, err := loadWebOrderPurchaseDocument(companyID, order)
	if err != nil {
		core.Log("sendWebOrderEmail:: no se pudo leer el pedido", order.ID, err)
		return
	}
	message := core.EmailMessage{
		To:       []string{order.Shipping.Email},
		FromName: purchaseDocument.StoreName,
		Subject:  fmt.Sprintf("%v — pedido %v", purchaseDocument.StoreName, webOrderNumber(order.ID)),
		HTMLBody: fmt.Sprintf(`<html><head><meta http-equiv="Content-Type" content="text/html; charset=utf-8" /></head>
<body style="font-family:system-ui,sans-serif;color:#1e293b;line-height:1.55">
	<h2 style="color:#4f46e5">¡Gracias por su compra, %v!</h2>
	<p>Registramos su pedido <b>%v</b> por <b>%v</b>.</p>
	<p>%v</p>
</body></html>`, order.Shipping.FirstName, webOrderNumber(order.ID), formatCents(order.TotalAmount),
			core.If(withPurchaseOrder, "Adjuntamos su Orden de Compra: preséntela al recibir o recoger su pedido.",
				"Su pago fue confirmado.")),
	}
	if withPurchaseOrder {
		pdfContent, err := buildWebOrderPurchasePDF(purchaseDocument)
		if err != nil {
			core.Log("sendWebOrderEmail:: no se pudo generar la orden de compra", order.ID, err)
		} else {
			message.Attachments = []core.EmailAttachment{{
				Name: fmt.Sprintf("orden-de-compra-%v.pdf", webOrderNumber(order.ID)), Content: pdfContent}}
		}
	}
	if err := core.SendEmail(companyID, message); err != nil {
		core.Log("sendWebOrderEmail:: no se pudo enviar el correo del pedido", order.ID, err)
	}
}
