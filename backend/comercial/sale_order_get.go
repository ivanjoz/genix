package comercial

import (
	"app/comercial/types"
	"app/core"
	"app/db"
)

// GetSaleOrders returns sales orders for the authenticated company.
// It supports delta sync via the "upd" query parameter.
func GetSaleOrders(req *core.HandlerArgs) core.HandlerResponse {
	updated := int32(req.GetQueryInt64("upd"))
	orderStatus := req.GetQueryIntSlice("ss")
	
	sales := []types.SaleOrder{}
	
	for _, status := range orderStatus {
		query := db.Query(&sales)
		query.EmpresaID.Equals(req.Usuario.EmpresaID).Status.Equals(int8(status))
		
		if updated > 0 {
			// Delta sync: return everything updated after the last sync, including deleted/canceled
			query.Updated.GreaterThan(updated)
		}
		
		if err := query.AllowFilter().Exec(); err != nil {
			core.Log("Error querying sale orders:", err)
			return req.MakeErr("Error al obtener las órdenes de venta.")
		}
	}
	
	return req.MakeResponse(sales)
}
