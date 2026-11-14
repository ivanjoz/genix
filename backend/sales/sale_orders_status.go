package sales

import (
	"app/core"
	"app/db"
	"app/sales/types"
)

var pendingStatusToStatus = map[int8][]int8{
	types.OrderStatusPaid:      {types.OrderStatusPending, types.OrderStatusDelivered},
	types.OrderStatusDelivered: {types.OrderStatusPending, types.OrderStatusPaid},
}

// saleOrdersQueryLimit caps a tab's listing.
const saleOrdersQueryLimit = 5000

// GetSaleOrders serves one tab per request. It supports delta sync via the "records" query
// parameter, the response key's own cursor.
func GetSaleOrders(req *core.HandlerArgs) core.HandlerResponse {
	// "Por regularizar": delivered sales whose made products' stock exit waits for the production close.
	if req.GetQueryInt("pending-stock") == 1 {
		saleOrders := []types.SaleOrder{}
		query := db.Query(&saleOrders).Limit(saleOrdersQueryLimit).OrderDesc()
		query.Exclude(query.CompanyID)
		query.CompanyID.Equals(req.User.CompanyID).PendingStockMovement.Equals(1).
			Delta(req.GetDeltaCursor("records"), types.OrderStatusDelivered, types.OrderStatusCompleted)
		if err := query.Exec(); err != nil {
			return req.MakeErr("Error al obtener las ventas por regularizar:", err)
		}
		return req.MakeResponse(&map[string]any{"records": &saleOrders})
	}

	orderPendingStatus := int8(req.GetQueryInt("pending-status"))
	// Read presence, not value: the Anuladas tab pins Status 0, which a `> 0` test cannot
	// distinguish from the parameter being absent.
	orderStatusValue, hasOrderStatus := req.Query["order-status"]
	orderStatus := int8(req.GetQueryInt("order-status"))

	// Clone the configured slice before appending so request-specific filters do not
	// mutate the shared backing array stored in the status map.
	orderStatusToQuery := append([]int8{}, pendingStatusToStatus[orderPendingStatus]...)
	if hasOrderStatus && orderStatusValue != "" {
		orderStatusToQuery = append(orderStatusToQuery, orderStatus)
	}
	if len(orderStatusToQuery) == 0 {
		return req.MakeErr("El order status es incorrecto.")
	}

	// Status is the delta index's pinned key, so the read covers exactly this tab's statuses (the
	// Anuladas tab pins 0 like any other value): an order that moved to another tab comes back as a
	// removed ID in the window Metadata, never as a record here.
	saleOrders := []types.SaleOrder{}
	query := db.Query(&saleOrders).Limit(saleOrdersQueryLimit).OrderDesc()
	query.Exclude(query.CompanyID)
	query.CompanyID.Equals(req.User.CompanyID).Delta(req.GetDeltaCursor("records"), core.ToAny(orderStatusToQuery)...)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los registros de ventas:", err)
	}

	response := map[string]any{"records": &saleOrders}
	return req.MakeResponse(&response)
}
