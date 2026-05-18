package sales

import (
	"app/sales/types"
	"app/core"
	"app/db"
)

func SaleOrderQuery(req *core.HandlerArgs) core.HandlerResponse {
	dateStart := req.GetQueryInt16("date-start")
	dateEnd := req.GetQueryInt16("date-end")
	productID := req.GetQueryInt("product-id")
	clientID := req.GetQueryInt("client-id")
	saleOrderStatus := req.GetQueryInt("status")

	if dateStart <= 0 || dateEnd <= 0 {
		return req.MakeErr("Debe especificar el rango de dates.")
	}
	if dateEnd < dateStart {
		return req.MakeErr("La date final no puede ser menor a la date inicial.")
	}
	if dateEnd-dateStart > 120 {
		return req.MakeErr("Sólo se pueden consultar hasta 120 días a la vez.")
	}

	saleOrderCacheGroupHashes, err := core.ExtractGroupIndexCacheValues(req)
	if err != nil {
		return req.MakeErr(err)
	}

	saleOrders := []db.RecordGroup[types.SaleOrder]{}
	query := db.QueryIndexGroup(&saleOrders).
		CompanyID.Equals(req.User.CompanyID)

	for _, cacheGroup := range saleOrderCacheGroupHashes {
		query.IncludeCachedGroup(cacheGroup.GroupHash, cacheGroup.UpdateCounter)
	}

	// Keep Date as the only BETWEEN predicate because QueryIndexGroup requires exactly one range
	// and then picks the most specific compatible grouped index from the remaining equality filters.
	query.Date.Between(dateStart, dateEnd)

	switch {
	case clientID > 0 && productID > 0:
		// Uses the most specific raw group: Date + ClientID + DetailProductsIDs.
		query.ClientID.Equals(clientID).DetailProductsIDs.Contains(productID)
	case productID > 0:
		// Uses the raw group: Date + DetailProductsIDs.
		query.DetailProductsIDs.Contains(productID)
	case clientID > 0:
		// Uses the raw group: Date + ClientID.
		query.ClientID.Equals(clientID)
	case saleOrderStatus > 0:
		// Uses the raw group: Date + Status.
		query.Status.Equals(int8(saleOrderStatus))
	}

	if err := query.Exec(); err != nil {
		core.Log("Error querying sale order groups:", err)
		return req.MakeErr("Error al obtener las órdenes de venta agrupadas.")
	}

	return core.MakeResponse(req, &saleOrders)
}
