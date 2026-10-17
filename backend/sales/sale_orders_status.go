package sales

import (
	"app/core"
	"app/db"
	"app/sales/types"
	"slices"
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

	// Status 0 is "annulled", and the delta log reads a status-0 write as a delete: the other tabs
	// drop an annulled order through it, but the Anuladas tab would never receive one. That tab
	// answers its whole list with the replace Metadata instead.
	if slices.Equal(orderStatusToQuery, []int8{types.OrderStatusAnnulled}) {
		annulledOrders := []types.SaleOrder{}
		query := db.Query(&annulledOrders).Limit(saleOrdersQueryLimit).OrderDesc()
		query.Exclude(query.CompanyID)
		query.CompanyID.Equals(req.User.CompanyID).Status.Equals(types.OrderStatusAnnulled)
		if err := query.Exec(); err != nil {
			return req.MakeErr("Error al obtener los registros de ventas:", err)
		}
		response := map[string]any{"records": core.WithDeltaReplace(annulledOrders)}
		return req.MakeResponse(&response)
	}

	syncFilterStatuses := make([]any, len(orderStatusToQuery))
	for i, status := range orderStatusToQuery {
		syncFilterStatuses[i] = status
	}
	loggedOrders := []types.SaleOrder{}
	query := db.Query(&loggedOrders).Limit(saleOrdersQueryLimit).OrderDesc()
	query.Exclude(query.CompanyID)
	query.CompanyID.Equals(req.User.CompanyID).Delta(req.GetDeltaCursor("records"), syncFilterStatuses...)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los registros de ventas:", err)
	}

	// Every tab reads the same log, which names every status: an order that moved to another tab
	// comes back as a live record, and the client is told to drop it from this one.
	saleOrders := []types.SaleOrder{}
	saleOrdersToRemoveIDs := []int64{}
	for _, saleOrder := range loggedOrders {
		if saleOrder.Metadata == nil && !slices.Contains(orderStatusToQuery, saleOrder.Status) {
			saleOrdersToRemoveIDs = append(saleOrdersToRemoveIDs, saleOrder.ID)
			continue
		}
		saleOrders = append(saleOrders, saleOrder)
	}

	response := map[string]any{
		"records":             &saleOrders,
		"records_IDsToRemove": &saleOrdersToRemoveIDs,
	}
	return req.MakeResponse(&response)
}
