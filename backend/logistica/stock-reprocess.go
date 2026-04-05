package logistica

import (
	"app/core"
	"app/db"
	logisticaTypes "app/logistica/types"
)

func RecalcProductStockByMovements(companyID int32, useFechaDelta bool) error {
	companyLock := getApplyMovimientosCompanyLock(companyID)

	core.Log("RecalcProductStockByMovements esperando lock de empresa:", companyID, "useFechaDelta:", useFechaDelta)
	companyLock.Lock()
	// Keep recalculation serialized with movement application for the same company.
	defer companyLock.Unlock()
	core.Log("RecalcProductStockByMovements lock adquirido para empresa:", companyID)

	computedMovementsByKey := make(map[string]*logisticaTypes.ProductStock)
	updatedTime := core.SUnixTime()

	accumulateMovement := func(warehouseID int32, quantity int32, subQuantity int32, movementRecord *logisticaTypes.WarehouseProductMovement) {
		key := db.MakeKeyConcat(warehouseID, movementRecord.ProductoID, movementRecord.PresentacionID, movementRecord.SKU, movementRecord.Lote)

		if computedMovementsByKey[key] == nil {
			// Keep one accumulator per exact warehouse-product-lot key for the mutable stock row.
			computedMovementsByKey[key] = &logisticaTypes.ProductStock{
				CompanyID:      companyID,
				ID:             key,
				WarehouseID:    warehouseID,
				ProductID:      movementRecord.ProductoID,
				PresentationID: movementRecord.PresentacionID,
				SKU:            movementRecord.SKU,
				Lote:           movementRecord.Lote,
				Updated:        updatedTime,
			}
		}

		computedMovementsByKey[key].Quantity += quantity
		computedMovementsByKey[key].SubQuantity += subQuantity

		keyGroup := db.MakeKeyConcat(warehouseID, movementRecord.ProductoID, movementRecord.PresentacionID)

		if computedMovementsByKey[keyGroup] == nil {
			// Keep one accumulator for the warehouse-product aggregate used by WarehouseProductQuantity.
			computedMovementsByKey[keyGroup] = &logisticaTypes.ProductStock{
				CompanyID:      companyID,
				ID:             keyGroup,
				WarehouseID:    warehouseID,
				ProductID:      movementRecord.ProductoID,
				PresentationID: movementRecord.PresentacionID,
				Updated:        updatedTime,
			}
		}

		computedMovementsByKey[keyGroup].WarehouseProductQuantity += quantity
	}

	query := db.Query(&[]logisticaTypes.WarehouseProductMovement{})
	query.CompanyID.Equals(companyID)

	if err := query.ExecScan(func(movementRecord *logisticaTypes.WarehouseProductMovement) bool {
		// Aggregate the row on its target warehouse.
		accumulateMovement(movementRecord.WarehouseID, movementRecord.Quantity, movementRecord.SubQuantity, movementRecord)

		if movementRecord.WarehouseRefID > 0 {
			// A transfer creates the mirrored outbound movement on the source warehouse.
			accumulateMovement(movementRecord.WarehouseRefID, -movementRecord.Quantity, -movementRecord.SubQuantity, movementRecord)
		}
		return true
	}); err != nil {
		return core.Err("Error al obtener movimientos:", err)
	}

	stocksToUpdate := []logisticaTypes.ProductStock{}
	stocksToInsert := []logisticaTypes.ProductStock{}

	stockQuery := db.Query(&[]logisticaTypes.ProductStock{})
	stockQuery.Select().CompanyID.Equals(companyID)

	if err := stockQuery.ExecScan(func(existingStockRecord *logisticaTypes.ProductStock) bool {
		computedStockRecord := computedMovementsByKey[existingStockRecord.ID]
		if computedStockRecord == nil {
			return true
		}

		delete(computedMovementsByKey, existingStockRecord.ID)

		shouldUpdate := existingStockRecord.Quantity != computedStockRecord.Quantity ||
			existingStockRecord.SubQuantity != computedStockRecord.SubQuantity ||
			existingStockRecord.WarehouseProductQuantity != computedStockRecord.WarehouseProductQuantity

		if shouldUpdate {
			stocksToUpdate = append(stocksToUpdate, *computedStockRecord)
		}
		return true
	}); err != nil {
		return core.Err("Error al obtener el stock actual:", err)
	}

	for _, computedStockRecord := range computedMovementsByKey {
		stocksToInsert = append(stocksToInsert, *computedStockRecord)
	}

	core.Log("stocksToInsert:", len(stocksToInsert), "| stocksToUpdate:", len(stocksToUpdate))

	productStockTable := db.Table[logisticaTypes.ProductStock]()

	if len(stocksToUpdate) > 0 {
		if err := db.Update(&stocksToUpdate,
			productStockTable.Quantity,
			productStockTable.SubQuantity,
			productStockTable.Updated,
			productStockTable.Status,
			productStockTable.WarehouseProductQuantity,
			productStockTable.IsWarehouseProductStatus,
		); err != nil {
			return core.Err("Error al actualizar el stock recalculado:", err)
		}
	}

	if len(stocksToInsert) > 0 {
		if err := db.Insert(&stocksToInsert); err != nil {
			return core.Err("Error al insertar el stock recalculado:", err)
		}
	}

	// 5. Update Producto.StockStatus and CategoriasConStock
	/*
		productos := []negocioTypes.Producto{}
		qProd := db.Query(&productos)
		q1 := db.Table[negocioTypes.Producto]()
		qProd.Select(q1.ID, q1.EmpresaID, q1.CategoriasIDs, q1.StockStatus).
			EmpresaID.Equals(empresaID)

		if err := qProd.Exec(); err != nil {
			return core.Err("Error al obtener productos:", err)
		}

		productosToUpdate := []negocioTypes.Producto{}
		for _, p := range productos {
			totalCantidad := productoTotalStock[p.ID]
			oldStatus := p.StockStatus

			if totalCantidad > 0 {
				p.StockStatus = 1
			} else {
				p.StockStatus = 0
			}

			// Always update if we need to ensure consistency,
			// or at least when status changed. We'll update all for safety.
			p.FillCategoriasConStock()

			// Only append if something logically changed, or just update all
			if oldStatus != p.StockStatus || len(p.CategoriasConStock) > 0 || oldStatus > 0 {
				productosToUpdate = append(productosToUpdate, p)
			}
		}

		if len(productosToUpdate) > 0 {
			qTable := db.Table[negocioTypes.Producto]()
			if err := db.Update(&productosToUpdate, qTable.StockStatus, qTable.CategoriasConStock); err != nil {
				return core.Err("Error al actualizar el stock y categorias en productos:", err)
			}
		}
	*/
	return nil
}
