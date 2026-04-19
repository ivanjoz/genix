package logistica

import (
	"app/core"
	"app/db"
	logisticaTypes "app/logistica/types"
)

// RecalcProductStockByMovements rebuilds ProductStockV2 + ProductStockDetail
// from the full WarehouseProductMovement ledger for one company.
// A movement with LotID==0 AND SerialNumber=="" contributes to V2.Quantity;
// any other movement contributes to the matching ProductStockDetail row, whose
// sum is mirrored into V2.DetailQuantity.
func RecalcProductStockByMovements(companyID int32) error {
	companyLock := getApplyMovimientosCompanyLock(companyID)

	core.Log("RecalcProductStockByMovements esperando lock empresa:", companyID)
	companyLock.Lock()
	defer companyLock.Unlock()

	updatedTime := core.SUnixTime()

	// Start by loading the persisted rows. Excluding Created+CreatedBy (and Updated/UpdatedBy on
	// details) means any in-memory row still carrying Created==0 at the end of the pass was
	// loaded from DB and must be persisted as an UPDATE; Created>0 flags INSERT.
	stockByID := map[int64]*logisticaTypes.ProductStockV2{}
	detailByKey := map[string]*logisticaTypes.ProductStockDetail{}
	detailKey := func(stockID int64, lotID int32, serial string) string {
		return db.MakeKeyConcat(stockID, lotID, serial)
	}

	{
		existing := []logisticaTypes.ProductStockV2{}
		q := db.Query(&existing)
		q.Exclude(q.Created, q.CreatedBy).CompanyID.Equals(companyID)
		if err := q.Exec(); err != nil {
			return core.Err("Error al obtener stock V2 previo:", err)
		}
		for i := range existing {
			stock := &existing[i]
			// Reset so movements recompute from scratch; untouched rows will blank out below.
			stock.Quantity, stock.SubQuantity = 0, 0
			stock.DetailQuantity, stock.DetailSubQuantity = 0, 0
			stockByID[stock.ID] = stock
		}
	}
	{
		existing := []logisticaTypes.ProductStockDetail{}
		q := db.Query(&existing)
		q.Exclude(q.Created, q.CreatedBy, q.Updated, q.UpdatedBy).CompanyID.Equals(companyID)
		if err := q.Exec(); err != nil {
			return core.Err("Error al obtener detalle de stock previo:", err)
		}
		for i := range existing {
			detail := &existing[i]
			detail.Quantity, detail.SubQuantity = 0, 0
			detailByKey[detailKey(detail.ProductStockID, detail.LotID, detail.SerialNumber)] = detail
		}
	}

	accumulate := func(warehouseID int32, quantity int32, subQuantity int32, movement *logisticaTypes.WarehouseProductMovement) {
		stockID := packProductStockID(warehouseID, movement.ProductoID, movement.PresentacionID)
		stock := stockByID[stockID]
		if stock == nil {
			// Fresh V2 row (no historical record): Created stamps it as INSERT at write time.
			stock = &logisticaTypes.ProductStockV2{
				ID:             stockID,
				CompanyID:      companyID,
				WarehouseID:    warehouseID,
				ProductID:      movement.ProductoID,
				PresentationID: movement.PresentacionID,
				Created:        updatedTime,
			}
			stockByID[stockID] = stock
		}
		if movement.LotID == 0 && movement.SerialNumber == "" {
			stock.Quantity += quantity
			stock.SubQuantity += subQuantity
			return
		}
		key := detailKey(stockID, movement.LotID, movement.SerialNumber)
		detail := detailByKey[key]
		if detail == nil {
			detail = &logisticaTypes.ProductStockDetail{
				CompanyID:      companyID,
				ProductStockID: stockID,
				LotID:          movement.LotID,
				SerialNumber:   movement.SerialNumber,
				WarehouseID:    warehouseID,
				ProductID:      movement.ProductoID,
				Created:        updatedTime,
			}
			detailByKey[key] = detail
		}
		detail.Quantity += quantity
		detail.SubQuantity += subQuantity
	}

	query := db.Query(&[]logisticaTypes.WarehouseProductMovement{})
	query.CompanyID.Equals(companyID)
	if err := query.ExecScan(func(movement *logisticaTypes.WarehouseProductMovement) bool {
		accumulate(movement.WarehouseID, movement.Quantity, movement.SubQuantity, movement)
		if movement.WarehouseRefID > 0 {
			// Transfers mirror an outbound leg on the source warehouse.
			accumulate(movement.WarehouseRefID, -movement.Quantity, -movement.SubQuantity, movement)
		}
		return true
	}); err != nil {
		return core.Err("Error al escanear movimientos:", err)
	}

	// Roll up DetailQuantity onto each stock once every movement is accumulated.
	for _, detail := range detailByKey {
		if stock := stockByID[detail.ProductStockID]; stock != nil {
			stock.DetailQuantity += detail.Quantity
			stock.DetailSubQuantity += detail.SubQuantity
		}
		detail.Status = core.If(detail.Quantity == 0 && detail.SubQuantity == 0, int8(0), int8(1))
		detail.Updated = updatedTime
	}
	for _, stock := range stockByID {
		stock.Status = core.If(stock.Quantity == 0 && stock.DetailQuantity == 0, int8(0), int8(1))
		stock.Updated = updatedTime
	}

	// Flatten and let InsertUpdateInclude route by the Created>0 marker: fresh rows go to INSERT,
	// preloaded rows go to UPDATE (touching only the listed columns).
	stocks := make([]logisticaTypes.ProductStockV2, 0, len(stockByID))
	for _, stock := range stockByID {
		stocks = append(stocks, *stock)
	}
	details := make([]logisticaTypes.ProductStockDetail, 0, len(detailByKey))
	for _, detail := range detailByKey {
		details = append(details, *detail)
	}

	core.Log("RecalcProductStockByMovements writes:", "stocks", len(stocks), "details", len(details))

	if len(stocks) > 0 {
		stockTable := db.Table[logisticaTypes.ProductStockV2]()
		if err := db.InsertUpdateInclude(&stocks,
			func(e *logisticaTypes.ProductStockV2) bool { return e.Created > 0 },
			[]db.Coln{
				stockTable.Quantity, stockTable.SubQuantity,
				stockTable.DetailQuantity, stockTable.DetailSubQuantity,
				stockTable.Updated, stockTable.Status,
			},
		); err != nil {
			return core.Err("Error al guardar stock V2 recalculado:", err)
		}
	}

	if len(details) > 0 {
		detailTable := db.Table[logisticaTypes.ProductStockDetail]()
		if err := db.InsertUpdateInclude(&details,
			func(e *logisticaTypes.ProductStockDetail) bool { return e.Created > 0 },
			[]db.Coln{
				detailTable.Quantity, detailTable.SubQuantity,
				detailTable.Updated, detailTable.Status,
			},
		); err != nil {
			return core.Err("Error al guardar detalle de stock recalculado:", err)
		}
	}

	return nil
}
