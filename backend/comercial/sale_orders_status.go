package comercial

import (
	"app/comercial/types"
	"app/core"
	"app/db"

	"golang.org/x/sync/errgroup"
)

var pendingStatusToStatus = map[int8][]int8{
	types.OrderStatusPaid:      {types.OrderStatusPending, types.OrderStatusDelivered},
	types.OrderStatusDelivered: {types.OrderStatusPending, types.OrderStatusPaid},
}

// It supports delta sync via the "upc" query parameter.
func GetSaleOrders(req *core.HandlerArgs) core.HandlerResponse {
	updateCounter := req.GetQueryInt("records")
	orderPendingStatus := int8(req.GetQueryInt("pending-status"))
	orderStatus := int8(req.GetQueryInt("order-status"))
	queryGroup := errgroup.Group{}

	// Clone the configured slice before appending so request-specific filters do not
	// mutate the shared backing array stored in the status map.
	orderStatusToQuery := append([]int8{}, pendingStatusToStatus[orderPendingStatus]...)

	if orderStatus > 0 {
		orderStatusToQuery = append(orderStatusToQuery, orderStatus)
	}

	if len(orderStatusToQuery) == 0 {
		return req.MakeErr("El order status es incorrecto.")
	}

	orderStatusToRemove := []int8{types.OrderStatusCompleted, types.OrderStatusAnnulled}
	if orderPendingStatus > 0 {
		orderStatusToRemove = append(orderStatusToRemove, orderPendingStatus)
	}

	saleOrdersByStatus := make([][]types.SaleOrder, len(orderStatusToQuery))
	for resultIndex, currentOrderStatus := range orderStatusToQuery {

		queryGroup.Go(func() error {
			query := db.Query(&saleOrdersByStatus[resultIndex]).Limit(5000).OrderDesc()
			query.Exclude(query.CompanyID)

			query.CompanyID.Equals(req.Usuario.EmpresaID).
				Status.Equals(currentOrderStatus).
				UpdateCounter.GreaterEqual(updateCounter)

			return query.Exec()
		})
	}

	saleOrdersToRemoveIDsGroups := make([][]int64, len(orderStatusToRemove))
	core.Log("orderStatusToRemove:", orderStatusToRemove)
	
	if updateCounter > 0 {
		for resultIndex, currentOrderStatus := range orderStatusToRemove {

			queryGroup.Go(func() error {
				idsToSave := &saleOrdersToRemoveIDsGroups[resultIndex]

				query := db.Query(&[]types.SaleOrder{})
				query.Select(query.ID)

				query.CompanyID.Equals(req.Usuario.EmpresaID).
					Status.Equals(currentOrderStatus).
					UpdateCounter.GreaterEqual(updateCounter)

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
	
	core.Log("saleOrdersToRemoveIDs::",saleOrdersToRemoveIDsGroups)
	
	response := map[string]any{
		"records":             &saleOrders,
		"records_IDsToRemove": &saleOrdersToRemoveIDs,
	}

	return req.MakeResponse(&response)
}
