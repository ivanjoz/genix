package comercial

import (
	"app/comercial/types"
	"app/core"
	"app/db"
)

func SaleOrderQuery(req *core.HandlerArgs) core.HandlerResponse {
	fechaStart := req.GetQueryInt16("fecha-start")
	fechaEnd := req.GetQueryInt16("fecha-end")
	productID := req.GetQueryInt("product-id")
	clientID := req.GetQueryInt("client-id")
	saleOrderStatus := req.GetQueryInt("status")

	if fechaStart <= 0 || fechaEnd <= 0 {
		return req.MakeErr("Debe especificar el rango de fechas.")
	}
	if fechaEnd < fechaStart {
		return req.MakeErr("La fecha final no puede ser menor a la fecha inicial.")
	}
	if fechaEnd-fechaStart > 120 {
		return req.MakeErr("Sólo se pueden consultar hasta 120 días a la vez.")
	}

	saleOrderCacheGroupHashes, err := core.ExtractGroupIndexCacheValues(req)
	if err != nil {
		return req.MakeErr(err)
	}

	saleOrders := []db.RecordGroup[types.SaleOrder]{}
	query := db.QueryIndexGroup(&saleOrders).
		CompanyID.Equals(req.Usuario.EmpresaID)

	for _, cacheGroup := range saleOrderCacheGroupHashes {
		query.IncludeCachedGroup(cacheGroup.GroupHash, cacheGroup.UpdateCounter)
	}

	// Keep Fecha as the only BETWEEN predicate because QueryIndexGroup requires exactly one range
	// and then picks the most specific compatible grouped index from the remaining equality filters.
	query.Fecha.Between(fechaStart, fechaEnd)

	switch {
	case clientID > 0 && productID > 0:
		// Uses the most specific raw group: Fecha + ClientID + DetailProductsIDs.
		query.ClientID.Equals(clientID).DetailProductsIDs.Contains(productID)
	case productID > 0:
		// Uses the raw group: Fecha + DetailProductsIDs.
		query.DetailProductsIDs.Contains(productID)
	case clientID > 0:
		// Uses the raw group: Fecha + ClientID.
		query.ClientID.Equals(clientID)
	case saleOrderStatus > 0:
		// Uses the raw group: Fecha + Status.
		query.Status.Equals(int8(saleOrderStatus))
	}

	if err := query.Exec(); err != nil {
		core.Log("Error querying sale order groups:", err)
		return req.MakeErr("Error al obtener las órdenes de venta agrupadas.")
	}

	return core.MakeResponse(req, &saleOrders)
}
