package sales

import (
	"app/core"
	"app/db"
	production "app/production/types"
	"app/sales/types"
	"cmp"
	"slices"
)

// webCustomerOrdersLimit bounds "Mis Pedidos" to the latest orders.
const webCustomerOrdersLimit = 50

// GetWebCustomerOrders lists the logged-in customer's online orders, newest first, for "Mis Pedidos".
// SaleOrderID tells which ones became a sale (and so carry a boleta when the store issues them).
func GetWebCustomerOrders(req *core.HandlerArgs) core.HandlerResponse {
	companyID, customer, err := requireWebCustomer(req)
	if err != nil {
		return req.MakeErrCode(err.Error(), 401)
	}
	orders := []types.OnlineSaleOrder{}
	query := db.Query(&orders)
	query.Select(query.ID, query.Date, query.Status, query.Lines, query.TotalAmount, query.DebtAmount,
		query.ShippingAmount, query.DeliveryType, query.PickupSiteID, query.Shipping, query.SaleOrderID,
		query.FulfillmentStatus, query.PaymentMethod).
		CompanyID.Equals(companyID).WebCustomerID.Equals(customer.ID)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los pedidos:", err)
	}
	// The ID is a per-company sequence, so it orders the orders by when they were placed.
	slices.SortFunc(orders, func(a, b types.OnlineSaleOrder) int { return cmp.Compare(b.ID, a.ID) })
	if len(orders) > webCustomerOrdersLimit {
		orders = orders[:webCustomerOrdersLimit]
	}

	// The product names travel with the orders: the storefront's catalog may no longer list a
	// product a past order bought.
	productIDs := core.SliceSet[int32]{}
	for index := range orders {
		for _, productID := range orders[index].ProductIDs() {
			productIDs.Add(productID)
		}
	}
	productNames := map[int32]string{}
	if !productIDs.IsEmpty() {
		products := []production.Product{}
		productQuery := db.Query(&products)
		productQuery.Select(productQuery.ID, productQuery.Name).CompanyID.Equals(companyID).ID.In(productIDs.Values...)
		if err := productQuery.Exec(); err != nil {
			return req.MakeErr("Error al obtener los productos de los pedidos:", err)
		}
		for _, product := range products {
			productNames[product.ID] = product.Name
		}
	}
	return req.MakeResponse(map[string]any{"Orders": orders, "ProductNames": productNames})
}

// GetWebOrderPurchasePDF downloads the Orden de Compra of one of the customer's orders (?id=, the
// online order number).
func GetWebOrderPurchasePDF(req *core.HandlerArgs) core.HandlerResponse {
	companyID, customer, err := requireWebCustomer(req)
	if err != nil {
		return req.MakeErrCode(err.Error(), 401)
	}
	orders := []types.OnlineSaleOrder{}
	query := db.Query(&orders)
	query.Select().CompanyID.Equals(companyID).ID.Equals(int32(req.GetQueryInt("id")))
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener el pedido:", err)
	}
	// Another customer's order answers exactly like a missing one.
	if len(orders) == 0 || orders[0].WebCustomerID != customer.ID {
		return req.MakeErr("No se encontró el pedido.")
	}
	purchaseDocument, err := loadWebOrderPurchaseDocument(companyID, orders[0])
	if err != nil {
		return req.MakeErr(err)
	}
	pdfContent, err := buildWebOrderPurchasePDF(purchaseDocument)
	if err != nil {
		return req.MakeErr("No se pudo generar la orden de compra:", err)
	}
	response := req.MakeResponsePlain(&pdfContent)
	response.Headers = map[string]string{
		"Content-Type":        "application/pdf",
		"Content-Disposition": `attachment; filename="orden-de-compra-` + webOrderNumber(orders[0].ID) + `.pdf"`,
	}
	return response
}
