// Recomputing the production reservations (ProductStock.ReservedProduction*,
// docs/MANUFACTURING_PLAN.md §5).
//
// Every release, receipt, completion and annulment keeps them by increments. This rebuilds them
// whole from the open orders' Supplies[].Reserved, so it repairs counters that drifted. Run it while
// nothing is released or received for the company, or a change written meanwhile is lost.

package exec

import (
	business "app/business/types"
	config "app/config/types"
	"app/core"
	"app/db"
	logistics "app/logistics/types"
	production "app/production/types"
	"fmt"
)

type productionReservationRecomputeReport struct {
	companies   int
	openOrders  int
	changedRows int
}

// RecomputeProductionReservation rebuilds the counters of every company (or one).
//
// fn-recompute-production-reservation [apply] [companyID]
//
// Without `apply` it only reports what it would write. Re-running is safe: it rebuilds the same counters.
func RecomputeProductionReservation(args *core.ExecArgs) core.FuncResponse {
	shouldWrite, onlyCompanyID := parseIdentityBackfillArguments(args.Message)

	companies := []config.Company{}
	companyQuery := db.Query(&companies)
	companyQuery.Status.GreaterEqual(0).AllowFilter()
	if queryError := companyQuery.Exec(); queryError != nil {
		return args.MakeErr("no se pudieron obtener las empresas:", queryError)
	}

	report := productionReservationRecomputeReport{}
	for _, company := range companies {
		if onlyCompanyID > 0 && company.ID != onlyCompanyID {
			continue
		}
		report.companies++
		if err := recomputeCompanyProductionReservation(company.ID, shouldWrite, &report); err != nil {
			return args.MakeErr(err)
		}
	}

	mode := core.If(shouldWrite, "APLICADO", "SIMULACIÓN (agregue 'apply' para escribir)")
	message := fmt.Sprintf("Recálculo de reservas de producción %v: %d empresa(s) · %d orden(es) abiertas · %d fila(s) de stock cambiadas",
		mode, report.companies, report.openOrders, report.changedRows)
	core.Log(message)
	return core.FuncResponse{Message: message}
}

func recomputeCompanyProductionReservation(companyID int32, shouldWrite bool, report *productionReservationRecomputeReport) error {
	orders := []production.ProductionOrder{}
	orderQuery := db.Query(&orders)
	orderQuery.Select(orderQuery.ID, orderQuery.SuppliesWarehouseID, orderQuery.SubcontractorID, orderQuery.Supplies, orderQuery.Status).
		CompanyID.Equals(companyID)
	if err := orderQuery.Exec(); err != nil {
		return core.Err("Error al leer las órdenes de producción de la empresa", companyID, err)
	}
	// What each row should hold, in total sub-units at the reservation's divisor.
	type reservationTarget struct {
		subUnits   int64
		subDivisor int16
	}
	targetByStockID := map[int64]*reservationTarget{}
	addTarget := func(warehouseID int32, orderSupply production.ProductionOrderSupply, subUnits int32) error {
		stockID := logistics.PackProductStockID(warehouseID, orderSupply.SupplyID, 0)
		target := targetByStockID[stockID]
		if target == nil {
			target = &reservationTarget{subDivisor: max(orderSupply.SubDivisor, core.QuantityDivisorNone)}
			targetByStockID[stockID] = target
		}
		if target.subDivisor != max(orderSupply.SubDivisor, core.QuantityDivisorNone) {
			return core.Err(fmt.Sprintf("Las órdenes reservan el insumo %v con divisores distintos: vuelva a liberarlas.", orderSupply.SupplyID))
		}
		target.subUnits += int64(subUnits)
		return nil
	}
	// A subcontracted order also holds what it sent to its workshop, on the workshop's row (§8).
	warehouses := []business.Warehouse{}
	warehouseQuery := db.Query(&warehouses)
	warehouseQuery.Select(warehouseQuery.ID, warehouseQuery.Status, warehouseQuery.SubcontractorID).CompanyID.Equals(companyID)
	if err := warehouseQuery.Exec(); err != nil {
		return core.Err("Error al leer los almacenes de la empresa", companyID, err)
	}
	workshopWarehouseBySubcontractor := map[int32]int32{}
	for _, warehouse := range warehouses {
		if warehouse.Status > 0 && warehouse.SubcontractorID > 0 {
			workshopWarehouseBySubcontractor[warehouse.SubcontractorID] = warehouse.ID
		}
	}
	for _, order := range orders {
		if order.Status != production.ProductionOrderStatusReleased && order.Status != production.ProductionOrderStatusInProgress {
			continue
		}
		report.openOrders++
		for _, orderSupply := range order.Supplies {
			if orderSupply.Reserved > 0 {
				if err := addTarget(order.SuppliesWarehouseID, orderSupply, orderSupply.Reserved); err != nil {
					return err
				}
			}
			if workshopWarehouseID := workshopWarehouseBySubcontractor[order.SubcontractorID]; orderSupply.AtWorkshop > 0 && workshopWarehouseID > 0 {
				if err := addTarget(workshopWarehouseID, orderSupply, orderSupply.AtWorkshop); err != nil {
					return err
				}
			}
		}
	}

	stocks := []logistics.ProductStock{}
	stockQuery := db.Query(&stocks)
	stockQuery.Select(stockQuery.ID, stockQuery.WarehouseID, stockQuery.ProductID, stockQuery.SubDivisor, stockQuery.Quantity,
		stockQuery.SubQuantity, stockQuery.DetailQuantity, stockQuery.DetailSubQuantity, stockQuery.PendingDeliveryStore,
		stockQuery.PendingDeliveryOnline, stockQuery.ReservedProductionQuantity, stockQuery.ReservedProductionSubQuantity).
		CompanyID.Equals(companyID)
	if err := stockQuery.Exec(); err != nil {
		return core.Err("Error al leer el stock de la empresa", companyID, err)
	}
	changedStocks := []logistics.ProductStock{}
	for _, stock := range stocks {
		stockDivisor := max(stock.SubDivisor, core.QuantityDivisorNone)
		nextReserved := core.Quantity{}
		if target := targetByStockID[stock.ID]; target != nil {
			converted, err := core.Quantity{Units: int32(target.subUnits / int64(target.subDivisor)), Sub: int32(target.subUnits % int64(target.subDivisor))}.
				ConvertToDivisor(target.subDivisor, stockDivisor)
			if err != nil {
				return core.Err(fmt.Sprintf("El stock %v está en una sub-unidad anterior a la de su reserva.", stock.ID))
			}
			nextReserved = core.Quantity{Units: int32(converted.TotalSubUnits(stockDivisor) / int64(stockDivisor)),
				Sub: int32(converted.TotalSubUnits(stockDivisor) % int64(stockDivisor))}
			delete(targetByStockID, stock.ID)
		}
		if nextReserved.Units == stock.ReservedProductionQuantity && nextReserved.Sub == stock.ReservedProductionSubQuantity {
			continue
		}
		stock.CompanyID = companyID
		stock.ReservedProductionQuantity, stock.ReservedProductionSubQuantity = nextReserved.Units, nextReserved.Sub
		changedStocks = append(changedStocks, stock)
	}
	if len(targetByStockID) > 0 {
		core.Log("recomputeCompanyProductionReservation:: reservas sin fila de stock", len(targetByStockID), "empresa", companyID)
	}
	report.changedRows += len(changedStocks)
	if !shouldWrite || len(changedStocks) == 0 {
		return nil
	}
	stockTable := db.TableOf[logistics.ProductStock]()
	// Written with WarehouseID and Status, the delta index's pinned columns (Status re-derived by
	// SelfParse from the quantities read above).
	return db.Update(&changedStocks, stockTable.ReservedProductionQuantity, stockTable.ReservedProductionSubQuantity,
		stockTable.Status, stockTable.WarehouseID)
}
