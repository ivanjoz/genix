// Recomputing the pending-delivery counters (ProductStock.PendingDeliveryStore / Online).
//
// The counters were added after sales existed, and from then on every sale write keeps them by
// increments. This rebuilds them whole from the undelivered SaleOrders, the pending web orders
// that have no sale yet and the restaurant's open table orders, so it also repairs counters that
// drifted. Before that it fixes the orders
// placed before the counters: a web order without PendingWarehouseID gets the company's first
// online warehouse, and an undelivered sale without WarehouseID (a web order only paid) gets its web
// order's. Run it while nothing sells for the company, or a sale written meanwhile is lost.
// Every row it writes gets its Status re-derived (ProductStock.SelfParse), so a row with nothing on
// hand that owes deliveries comes back to the stock screens.

package exec

import (
	business "app/business/types"
	config "app/config/types"
	"app/core"
	"app/db"
	dining "app/dining/types"
	logistics "app/logistics/types"
	sales "app/sales/types"
	"fmt"
)

type pendingDeliveryRecomputeReport struct {
	companies     int
	pendingSales  int
	pendingOrders int
	// openTableOrders are the restaurant's open table orders.
	openTableOrders int
	fixedOrders     int
	fixedSales      int
	changedRows     int
}

// RecomputePendingDelivery rebuilds the counters of every company (or one).
//
// fn-recompute-pending-delivery [apply] [companyID]
//
// Without `apply` it only reports what it would write. Re-running is safe: it rebuilds the same counters.
func RecomputePendingDelivery(args *core.ExecArgs) core.FuncResponse {
	shouldWrite, onlyCompanyID := parseIdentityBackfillArguments(args.Message)

	companies := []config.Company{}
	companyQuery := db.Query(&companies)
	companyQuery.Status.GreaterEqual(0).AllowFilter()
	if queryError := companyQuery.Exec(); queryError != nil {
		return args.MakeErr("no se pudieron obtener las empresas:", queryError)
	}

	report := pendingDeliveryRecomputeReport{}
	for _, company := range companies {
		if onlyCompanyID > 0 && company.ID != onlyCompanyID {
			continue
		}
		report.companies++
		if err := recomputeCompanyPendingDelivery(company.ID, shouldWrite, &report); err != nil {
			return args.MakeErr(err)
		}
	}

	mode := core.If(shouldWrite, "APLICADO", "SIMULACIÓN (agregue 'apply' para escribir)")
	message := fmt.Sprintf("Recálculo de stock por entregar %v: %d empresa(s) · %d venta(s) y %d pedido(s) web pendientes · "+
		"%d pedido(s) de mesa abiertos · %d pedido(s) y %d venta(s) sin almacén corregidos · %d fila(s) de stock cambiadas",
		mode, report.companies, report.pendingSales, report.pendingOrders, report.openTableOrders, report.fixedOrders,
		report.fixedSales, report.changedRows)
	core.Log(message)
	return core.FuncResponse{Message: message}
}

func recomputeCompanyPendingDelivery(companyID int32, shouldWrite bool, report *pendingDeliveryRecomputeReport) error {
	warehouses := []business.Warehouse{}
	warehouseQuery := db.Query(&warehouses)
	warehouseQuery.Select(warehouseQuery.ID, warehouseQuery.Status, warehouseQuery.ExcludeFromOnlineSales).
		CompanyID.Equals(companyID)
	if err := warehouseQuery.Exec(); err != nil {
		return core.Err("Error al leer los almacenes de la empresa", companyID, err)
	}
	firstOnlineWarehouseID := business.FirstOnlineWarehouseID(warehouses)

	onlineOrders := []sales.OnlineSaleOrder{}
	orderQuery := db.Query(&onlineOrders)
	orderQuery.Select(orderQuery.ID, orderQuery.Lines, orderQuery.Status, orderQuery.SaleOrderID, orderQuery.PendingWarehouseID).
		CompanyID.Equals(companyID)
	if err := orderQuery.Exec(); err != nil {
		return core.Err("Error al leer los pedidos web de la empresa", companyID, err)
	}
	saleOrders := []sales.SaleOrder{}
	saleQuery := db.Query(&saleOrders)
	saleQuery.Select(saleQuery.ID, saleQuery.Lines, saleQuery.Status, saleQuery.WarehouseID, saleQuery.OnlineSaleOrderID,
		saleQuery.PendingStockMovement).
		CompanyID.Equals(companyID)
	if err := saleQuery.Exec(); err != nil {
		return core.Err("Error al leer las ventas de la empresa", companyID, err)
	}

	// What every undelivered order reserves, as one positive change per line.
	reservations := []logistics.PendingDeliveryChange{}
	ordersToFix := []sales.OnlineSaleOrder{}
	pendingWarehouseByOrderID := map[int32]int32{}
	for _, order := range onlineOrders {
		isUndelivered := order.Status == sales.OrderStatusPending || order.Status == sales.OrderStatusPaid
		if order.PendingWarehouseID == 0 && isUndelivered {
			order.CompanyID, order.PendingWarehouseID = companyID, firstOnlineWarehouseID
			ordersToFix = append(ordersToFix, order)
		}
		pendingWarehouseByOrderID[order.ID] = order.PendingWarehouseID
		// One that became a sale is counted from its sale.
		if order.Status == sales.OrderStatusPending && order.SaleOrderID == 0 {
			report.pendingOrders++
			reservations = append(reservations, sales.PendingDeliveryChanges(order.Lines, order.PendingWarehouseID, true, 1)...)
		}
	}
	salesToFix := []sales.SaleOrder{}
	for _, sale := range saleOrders {
		// A delivered sale still holds its made products' lines until the production close.
		if sale.PendingStockMovement == 1 && !sale.IsDeliveryPending() {
			reservations = append(reservations, sales.PendingDeliveryChanges(
				sales.LinesWithStockExitPending(sale.Lines, true), sale.WarehouseID, sale.OnlineSaleOrderID > 0, 1)...)
			continue
		}
		if !sale.IsDeliveryPending() {
			continue
		}
		report.pendingSales++
		if sale.WarehouseID == 0 {
			sale.CompanyID = companyID
			sale.WarehouseID = core.Coalesce(pendingWarehouseByOrderID[sale.OnlineSaleOrderID], firstOnlineWarehouseID)
			salesToFix = append(salesToFix, sale)
		}
		reservations = append(reservations, sales.PendingDeliveryChanges(sale.Lines, sale.WarehouseID, sale.OnlineSaleOrderID > 0, 1)...)
	}
	// The restaurant's open table orders hold their live lines too (docs/DINING_PLAN.md §2.2).
	tableOrders := []dining.TableOrder{}
	tableOrderQuery := db.Query(&tableOrders)
	tableOrderQuery.Select(tableOrderQuery.ID, tableOrderQuery.Lines, tableOrderQuery.WarehouseID, tableOrderQuery.Status).
		CompanyID.Equals(companyID)
	if err := tableOrderQuery.Exec(); err != nil {
		return core.Err("Error al leer los pedidos de mesa de la empresa", companyID, err)
	}
	for _, tableOrder := range tableOrders {
		if tableOrder.Status == dining.TableOrderOpen {
			report.openTableOrders++
			reservations = append(reservations, dining.TableOrderReservationChanges(tableOrder.Lines, tableOrder.WarehouseID, 1)...)
		}
	}
	report.fixedOrders += len(ordersToFix)
	report.fixedSales += len(salesToFix)

	// The change per row is the target minus what the row holds now, so rows no order accounts for
	// any more drop to zero.
	stocks := []logistics.ProductStock{}
	stockQuery := db.Query(&stocks)
	stockQuery.Select(stockQuery.WarehouseID, stockQuery.ProductID, stockQuery.PresentationID,
		stockQuery.PendingDeliveryStore, stockQuery.PendingDeliveryOnline).
		CompanyID.Equals(companyID)
	if err := stockQuery.Exec(); err != nil {
		return core.Err("Error al leer el stock de la empresa", companyID, err)
	}
	changes := reservations
	for _, stock := range stocks {
		if stock.PendingDeliveryStore != 0 || stock.PendingDeliveryOnline != 0 {
			changes = append(changes, logistics.PendingDeliveryChange{
				WarehouseID: stock.WarehouseID, ProductID: stock.ProductID, PresentationID: stock.PresentationID,
				Store: -stock.PendingDeliveryStore, Online: -stock.PendingDeliveryOnline,
			})
		}
	}
	for _, units := range logistics.PendingDeliveryOf(changes) {
		if units != 0 {
			report.changedRows++
		}
	}
	core.Log("recomputeCompanyPendingDelivery:: empresa", companyID, "ventas pendientes", report.pendingSales,
		"pedidos pendientes", report.pendingOrders, "pedidos/ventas sin almacén", len(ordersToFix), len(salesToFix))
	if !shouldWrite {
		return nil
	}

	return db.RunTransaction(func(transaction *db.Transaction) error {
		if len(ordersToFix) > 0 {
			orderTable := db.TableOf[sales.OnlineSaleOrder]()
			if err := db.UpdateIn(transaction, &ordersToFix, orderTable.PendingWarehouseID); err != nil {
				return core.Err("Error al corregir el almacén de los pedidos web:", err)
			}
		}
		if len(salesToFix) > 0 {
			saleTable := db.TableOf[sales.SaleOrder]()
			if err := db.UpdateIn(transaction, &salesToFix, saleTable.WarehouseID); err != nil {
				return core.Err("Error al corregir el almacén de las ventas:", err)
			}
		}
		return logistics.ApplyPendingDeliveryIn(transaction, companyID, 0, changes)
	})
}
