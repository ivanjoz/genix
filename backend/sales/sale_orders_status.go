package sales

import (
	"app/core"
	"app/db"
	"app/sales/types"
	"slices"

	"golang.org/x/sync/errgroup"
)

var pendingStatusToStatus = map[int8][]int8{
	types.OrderStatusPaid:      {types.OrderStatusPending, types.OrderStatusDelivered},
	types.OrderStatusDelivered: {types.OrderStatusPending, types.OrderStatusPaid},
}

// It supports delta sync via the "records" query parameter, the response key's own watermark.
func GetSaleOrders(req *core.HandlerArgs) core.HandlerResponse {
	// Watermarked by "upv", the write sequence number, not by a timestamp or the index-group counter:
	// two writes in the same second are distinguishable, so nothing is re-sent and nothing is skipped.
	updatedSince := req.GetQueryInt("records")
	orderPendingStatus := int8(req.GetQueryInt("pending-status"))
	// Read presence, not value: the Anuladas tab pins Status 0, which a `> 0` test cannot
	// distinguish from the parameter being absent.
	orderStatusValue, hasOrderStatus := req.Query["order-status"]
	orderStatus := int8(req.GetQueryInt("order-status"))
	queryGroup := errgroup.Group{}

	// Clone the configured slice before appending so request-specific filters do not
	// mutate the shared backing array stored in the status map.
	orderStatusToQuery := append([]int8{}, pendingStatusToStatus[orderPendingStatus]...)

	if hasOrderStatus && orderStatusValue != "" {
		orderStatusToQuery = append(orderStatusToQuery, orderStatus)
	}

	if len(orderStatusToQuery) == 0 {
		return req.MakeErr("El order status es incorrecto.")
	}

	// Every status the tab does NOT show: a record that moved into one of these has left this
	// tab, and the client is told to drop it. The statuses being queried are excluded because
	// telling a client to delete the rows just sent it would empty the tab on every delta sync.
	orderStatusToRemove := []int8{}
	for _, candidateStatus := range []int8{
		types.OrderStatusAnnulled, types.OrderStatusPending, types.OrderStatusPaid,
		types.OrderStatusDelivered, types.OrderStatusCompleted,
	} {
		if !slices.Contains(orderStatusToQuery, candidateStatus) {
			orderStatusToRemove = append(orderStatusToRemove, candidateStatus)
		}
	}

	saleOrdersByStatus := make([][]types.SaleOrder, len(orderStatusToQuery))
	for resultIndex, currentOrderStatus := range orderStatusToQuery {

		queryGroup.Go(func() error {
			query := db.Query(&saleOrdersByStatus[resultIndex]).Limit(5000).OrderDesc()
			query.Exclude(query.CompanyID)

			// Delta() with no filter values adds the exact watermark bound and nothing else, leaving
			// this query pinned to the one status the tab asked for.
			query.CompanyID.Equals(req.User.CompanyID).
				Status.Equals(currentOrderStatus).
				Delta(updatedSince)

			return query.Exec()
		})
	}

	saleOrdersToRemoveIDsGroups := make([][]int64, len(orderStatusToRemove))
	core.Log("orderStatusToRemove:", orderStatusToRemove)

	if updatedSince > 0 {
		for resultIndex, currentOrderStatus := range orderStatusToRemove {

			queryGroup.Go(func() error {
				idsToSave := &saleOrdersToRemoveIDsGroups[resultIndex]

				query := db.Query(&[]types.SaleOrder{})
				query.Select(query.ID)

				query.CompanyID.Equals(req.User.CompanyID).
					Status.Equals(currentOrderStatus).
					Delta(updatedSince)

				if err := query.ExecScan(func(record *types.SaleOrder) bool {
					(*idsToSave) = append((*idsToSave), record.ID)
					// Skip storing the decoded row because this query only needs the IDs.
					return true
				}); err != nil {
					return err
				}
				return nil
			})
		}
	}

	if err := queryGroup.Wait(); err != nil {
		return req.MakeErr("Error al obtener los registros de ventas:", err)
	}

	saleOrders := []types.SaleOrder{}
	for _, saleOrdersByCurrentStatus := range saleOrdersByStatus {
		saleOrders = append(saleOrders, saleOrdersByCurrentStatus...)
	}

	saleOrdersToRemoveIDs := []int64{}
	for _, idsToRemove := range saleOrdersToRemoveIDsGroups {
		saleOrdersToRemoveIDs = append(saleOrdersToRemoveIDs, idsToRemove...)
	}

	core.Log("saleOrdersToRemoveIDs::", saleOrdersToRemoveIDsGroups)

	response := map[string]any{
		"records":             &saleOrders,
		"records_IDsToRemove": &saleOrdersToRemoveIDs,
	}

	return req.MakeResponse(&response)
}
