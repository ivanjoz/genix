package comercial

import (
	"app/comercial/types"
	"app/core"
	"app/db"

	"golang.org/x/sync/errgroup"
)

// GetSaleOrders returns sales orders for the authenticated company.
// It supports delta sync via the "upd" query parameter.
func GetSaleOrders(req *core.HandlerArgs) core.HandlerResponse {
	updated := core.Coalesce(req.GetQueryInt("upd"), req.GetQueryInt("updated"))
	orderPendingStatus := req.GetQueryInt("pending-status")
	orderStatus := req.GetQueryInt("order-status")

	sales := []types.SaleOrder{}
	statusTracesToQuery := []int8{}
	statusTracesCompletedToQuery := []int8{}

	if orderPendingStatus > 0 {
		statusTracesToQuery = GetSaleOrderStatusTracesByPendingStatus(int8(orderPendingStatus))
		statusTracesCompletedToQuery = []int8{SaleOrderTraceStage3GeneratedDeliveredPaid}
		if orderPendingStatus == 2 {
			statusTracesCompletedToQuery = append(statusTracesCompletedToQuery,
				SaleOrderTraceStage2GeneratedPaid, SaleOrderTraceStage2CompletedFromGeneratedDelivered)
		} else if orderPendingStatus == 3 {
			statusTracesCompletedToQuery = append(statusTracesCompletedToQuery,
				SaleOrderTraceStage2GeneratedDelivered, SaleOrderTraceStage2CompletedFromGeneratedPaid)
		}
	} else if orderStatus > 0 {
		statusTracesToQuery = GetSaleOrderStatusTracesByOrderStatus(int8(orderStatus))
	}

	// Get the delta data
	if updated > 0 {
		// Delta sync for pending views must include completed traces too so the frontend can
		// evict rows that left the pending group after the last sync.
		pendingSales, err := getSaleOrdersByStatusTraces(req.Usuario.EmpresaID, statusTracesToQuery, updated)
		if err != nil {
			core.Log("Error querying pending sale orders:", err)
			return req.MakeErr("Error al obtener las órdenes de venta.")
		}
		sales = append(sales, pendingSales...)

		completedSales, err := getSaleOrdersByStatusTraces(req.Usuario.EmpresaID, statusTracesCompletedToQuery, updated)
		if err != nil {
			core.Log("Error querying completed sale orders:", err)
			return req.MakeErr("Error al obtener las órdenes de venta.")
		}
		sales = append(sales, completedSales...)

		salesByID := map[int64]*types.SaleOrder{}
		for i := range sales {
			sale := &sales[i]
			salesByID[sale.ID] = sale
		}
		sales = core.MapToSlice(salesByID)

		// Get the full data
	} else {
		filteredSales, err := getSaleOrdersByStatusTraces(req.Usuario.EmpresaID, statusTracesToQuery, 0)
		if err != nil {
			core.Log("Error querying sale orders:", err)
			return req.MakeErr("Error al obtener las órdenes de venta.")
		}
		sales = filteredSales
	}

	return req.MakeResponse(sales)
}

func getSaleOrdersByStatusTraces(companyID int32, statusTraces []int8, updated int32) ([]types.SaleOrder, error) {
	if len(statusTraces) == 0 {
		return nil, nil
	}

	queryResults := make([][]types.SaleOrder, len(statusTraces))
	queryGroup := errgroup.Group{}

	for traceIndex, statusTrace := range statusTraces {
		queryGroup.Go(func() error {
			traceSales := []types.SaleOrder{}
			query := db.Query(&traceSales)
			query.Exclude(query.CompanyID)
			query.CompanyID.Equals(companyID).StatusTrace.Equals(statusTrace)
			// Always keep Updated as a range predicate so the planner can route to the
			// packed StatusTrace+Updated view instead of falling back to base-table filtering.
			query.Updated.GreaterEqual(updated)

			if err := query.Exec(); err != nil {
				return err
			}

			queryResults[traceIndex] = traceSales
			return nil
		})
	}

	if err := queryGroup.Wait(); err != nil {
		return nil, err
	}

	mergedSales := []types.SaleOrder{}
	for _, traceSales := range queryResults {
		mergedSales = append(mergedSales, traceSales...)
	}
	return mergedSales, nil
}
