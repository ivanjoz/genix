package comercial

import (
	"app/comercial/types"
	"app/core"
	"app/db"
)

// GetSaleOrders returns sales orders for the authenticated company.
// It supports delta sync via the "upd" query parameter.
func GetSaleOrders(req *core.HandlerArgs) core.HandlerResponse {
	updated := core.Coalesce(req.GetQueryInt("upd"), req.GetQueryInt("updated"))
	orderPendingStatus := req.GetQueryInt("pending-status")
	orderStatus := req.GetQueryInt("order-status")
	
	sales := []types.SaleOrder{}
	
	// Get the delta data
	if updated > 0 {
		query := db.Query(&sales).EmpresaID.Equals(req.Usuario.EmpresaID)
		
		if orderPendingStatus == 2 { // Pending payment
			query.OrderPendingPaymentUpdated.GreaterEqual(updated)
		} else if orderPendingStatus == 3 { // Pending delivery
			query.OrderPendingDeliveryUpdated.GreaterEqual(updated)
		} else { // Finished
			query.OrderCompletedUpdated.GreaterEqual(updated)
		}
		
		if err := query.AllowFilter().Exec(); err != nil {
			core.Log("Error querying sale orders:", err)
			return req.MakeErr("Error al obtener las órdenes de venta.")
		}
		
	// Get the full data
	} else {
		orderStatusToQuery := []int8{}
		
		if orderStatus > 0 { 
			orderStatusToQuery = []int8{int8(orderStatus)}
		} else if orderPendingStatus > 0 {
			orderStatusNotCompleted := []int8{1,2,3}
			
			for _, id := range orderStatusNotCompleted {
				if id != int8(orderPendingStatus) {
					orderStatusToQuery = append(orderStatusToQuery, id)
				}
			}
		}
		
		for _, status := range orderStatusToQuery {
			query := db.Query(&sales)
			query.EmpresaID.Equals(req.Usuario.EmpresaID).Status.Equals(status)
			
			if err := query.AllowFilter().Exec(); err != nil {
				core.Log("Error querying sale orders:", err)
				return req.MakeErr("Error al obtener las órdenes de venta.")
			}
		}	
	}
	
	
	
	return req.MakeResponse(sales)
}
