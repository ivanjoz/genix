package logistics

import (
	"app/core"
	"app/db"
	"app/logistics/types"
	production "app/production/types"
	"encoding/json"
)

func GetProductSupply(req *core.HandlerArgs) core.HandlerResponse {
	productSupplyRecords := []types.ProductSupply{}
	productSupplyQuery := db.Query(&productSupplyRecords)
	productSupplyQuery.Select().
		CompanyID.Equals(req.User.CompanyID).
		Status.Equals(1)

	if queryError := productSupplyQuery.Exec(); queryError != nil {
		core.Log("GetProductSupply query error:", queryError)
		return req.MakeErr("Error al obtener la configuración de abastecimiento.", queryError)
	}

	core.Log("GetProductSupply result_count:", len(productSupplyRecords))
	return req.MakeResponse(productSupplyRecords)
}

// maxProductSupplyBulkRecords bounds one bulk write. The Excel import batches on the client, so a
// payload above this is a malformed request rather than a legitimate save.
const maxProductSupplyBulkRecords = 1000

func PostProductSupply(req *core.HandlerArgs) core.HandlerResponse {
	productSupplyRecords := []types.ProductSupply{}
	if deserializeError := json.Unmarshal([]byte(*req.Body), &productSupplyRecords); deserializeError != nil {
		core.Log("PostProductSupply deserialization error:", deserializeError)
		return req.MakeErr("Error al deserializar el body.", deserializeError)
	}

	if len(productSupplyRecords) == 0 {
		return req.MakeErr("Debe enviar al menos un registro de abastecimiento.")
	}
	if len(productSupplyRecords) > maxProductSupplyBulkRecords {
		return req.MakeErr("No se pueden guardar más de", maxProductSupplyBulkRecords, "registros de abastecimiento por solicitud.")
	}

	productIDs := make([]int32, 0, len(productSupplyRecords))
	seenProductIDs := map[int32]bool{}
	providerSupplyRowsPerRecord := make([][]types.ProductSupplyProviderRow, 0, len(productSupplyRecords))

	for recordIndex := range productSupplyRecords {
		productSupplyRecord := &productSupplyRecords[recordIndex]
		productSupplyRecord.ProviderSupply = types.SanitizeProviderSupplyRows(productSupplyRecord.ProviderSupply)

		if productSupplyRecord.ProductID <= 0 {
			return req.MakeErr("Debe enviar un ProductID válido.")
		}
		if productSupplyRecord.MinimunStock < 0 {
			return req.MakeErr("El stock mínimo no puede ser negativo.")
		}
		if productSupplyRecord.SalesPerDayEstimated < 0 {
			return req.MakeErr("Las ventas por día estimadas no pueden ser negativas.")
		}
		// Two rows for the same product would collide inside the same merge, and which one wins
		// would depend on slice order rather than on anything the caller decided.
		if seenProductIDs[productSupplyRecord.ProductID] {
			return req.MakeErr("El producto", productSupplyRecord.ProductID, "está repetido en el envío.")
		}

		seenProductIDs[productSupplyRecord.ProductID] = true
		productIDs = append(productIDs, productSupplyRecord.ProductID)
		providerSupplyRowsPerRecord = append(providerSupplyRowsPerRecord, productSupplyRecord.ProviderSupply)
	}

	if validationError := validateProductSupplyProductIDs(req.User.CompanyID, productIDs); validationError != nil {
		return req.MakeErr(validationError)
	}
	if validationError := types.ValidateProviderSupplyRowsBatch(req.User.CompanyID, providerSupplyRowsPerRecord); validationError != nil {
		return req.MakeErr(validationError)
	}

	currentTimestamp := core.SUnixTime()
	for recordIndex := range productSupplyRecords {
		productSupplyRecords[recordIndex].CompanyID = req.User.CompanyID
		productSupplyRecords[recordIndex].Status = 1
		productSupplyRecords[recordIndex].Updated = currentTimestamp
		productSupplyRecords[recordIndex].UpdatedBy = req.User.ID
	}

	if mergeError := db.Merge(&productSupplyRecords, nil,
		func(previousProductSupply, currentProductSupply *types.ProductSupply) bool {
			// Keep the product key immutable and refresh only mutable configuration fields.
			currentProductSupply.CompanyID = req.User.CompanyID
			currentProductSupply.ProductID = previousProductSupply.ProductID
			currentProductSupply.Status = 1
			currentProductSupply.Updated = currentTimestamp
			currentProductSupply.UpdatedBy = req.User.ID
			return true
		},
		func(currentProductSupply *types.ProductSupply) {
			// Fill server-owned metadata on insert so clients only send configuration fields.
			currentProductSupply.CompanyID = req.User.CompanyID
			currentProductSupply.Status = 1
			currentProductSupply.Updated = currentTimestamp
			currentProductSupply.UpdatedBy = req.User.ID
		},
	); mergeError != nil {
		core.Log("PostProductSupply merge error:", mergeError)
		return req.MakeErr("Error al guardar la configuración de abastecimiento.", mergeError)
	}

	core.Log("PostProductSupply saved:", "records=", len(productSupplyRecords))
	return req.MakeResponse(productSupplyRecords)
}

// validateProductSupplyProductIDs rejects supply rows pointing at products that do not exist in the
// company. The Excel import resolves products by name, so a typo would otherwise persist a supply
// configuration attached to nothing.
func validateProductSupplyProductIDs(companyID int32, productIDs []int32) error {
	products := []production.Product{}
	productQuery := db.Query(&products)
	productQuery.Select(productQuery.ID, productQuery.Status).
		CompanyID.Equals(companyID).
		ID.In(productIDs...)

	if queryError := productQuery.Exec(); queryError != nil {
		return core.Err("Error al validar los productos.", queryError)
	}

	foundProductIDs := map[int32]bool{}
	for _, productRecord := range products {
		foundProductIDs[productRecord.ID] = true
	}
	for _, productID := range productIDs {
		if !foundProductIDs[productID] {
			return core.Err("El producto", productID, "no existe.")
		}
	}

	return nil
}

/* GET: WAREHOUSE MOVIMIENTOS GROUPED */

type DateProductMovements struct {
	Date               int16
	DetailProductsIDs  []int32
	DetailInflows      []int32
	DetailOutflows     []int32
	DetailMinimumStock []int32
	Updated            int16 `json:"upd"`
}

func GetAlmacenMovimientosGrouped(req *core.HandlerArgs) core.HandlerResponse {

	movimientosFecha := req.GetQueryInt16("movimientos")
	productosStockUpdated := req.GetQueryInt("productosStock")

	movimientos := []types.WarehouseProductMovement{}

	query := db.Query(&movimientos).
		CompanyID.Equals(req.User.CompanyID).
		Date.GreaterEqual(movimientosFecha)

	// SubDivisor is in the group key so each sum stays within one divisor; the fold below
	// converts each group's sub-units to whole units before adding them together.
	if err := query.GroupBy(query.Date, query.ProductID, query.Type, query.SubDivisor,
		query.Quantity.Sum(), query.SubQuantity.Sum()).Exec(); err != nil {
		return req.MakeErr("Error al obtener los registros del almacén:", err)
	}

	core.PrintTable(movimientos, 30, 30, "Date", "ProductID", "WarehouseID", "AlmacenCantidad")

	// Keep one row per date and reuse the product position to avoid duplicated product IDs.
	type dateMovimientosAccumulator struct {
		record                  *DateProductMovements
		productIndexByProductID map[int32]int
	}

	dailyGroupedRecords := map[int16]*dateMovimientosAccumulator{}
	for _, movimiento := range movimientos {
		dateAccumulator, exists := dailyGroupedRecords[movimiento.Date]
		if !exists {
			dateAccumulator = &dateMovimientosAccumulator{
				record: &DateProductMovements{
					Date: movimiento.Date, Updated: movimiento.Date,
				},
				productIndexByProductID: map[int32]int{},
			}
			dailyGroupedRecords[movimiento.Date] = dateAccumulator
		}

		productIndex, productExists := dateAccumulator.productIndexByProductID[movimiento.ProductID]
		if !productExists {
			productIndex = len(dateAccumulator.record.DetailProductsIDs)
			dateAccumulator.productIndexByProductID[movimiento.ProductID] = productIndex
			dateAccumulator.record.DetailProductsIDs = append(dateAccumulator.record.DetailProductsIDs, movimiento.ProductID)
			dateAccumulator.record.DetailInflows = append(dateAccumulator.record.DetailInflows, 0)
			dateAccumulator.record.DetailOutflows = append(dateAccumulator.record.DetailOutflows, 0)
		}

		// This report is whole-unit by design (supply thresholds are), so the sub-unit part is
		// folded in truncated rather than dropped: a product sold only in candies would
		// otherwise show no movement at all and read as dead stock to supply planning.
		groupedQuantity := movimiento.Quantity
		if movimiento.SubQuantity != 0 && movimiento.SubDivisor > 0 {
			groupedQuantity = int32(core.Quantity{
				Units: movimiento.Quantity, Sub: movimiento.SubQuantity,
			}.TotalSubUnits(movimiento.SubDivisor) / int64(movimiento.SubDivisor))
		}

		// Split the signed grouped quantity into inflow/outflow columns for the cached payload.
		if groupedQuantity > 0 {
			dateAccumulator.record.DetailInflows[productIndex] += groupedQuantity
		} else if groupedQuantity < 0 {
			dateAccumulator.record.DetailOutflows[productIndex] += -groupedQuantity
		}
	}

	movimientosAgrupados := make([]DateProductMovements, 0, len(dailyGroupedRecords))
	for _, dateAccumulator := range dailyGroupedRecords {
		movimientosAgrupados = append(movimientosAgrupados, *dateAccumulator.record)
	}

	// Productos Stock (V2). "Quantity" on the response is the combined bucket
	// so consumers stay compatible with the old shape without needing detail rows here.
	productosStockV2 := []types.ProductStock{}

	psQuery := db.Query(&productosStockV2)
	// No WarehouseID pinned, so Delta() routes to the [Status] delta index.
	psQuery.Select(psQuery.ID, psQuery.Updated, psQuery.UpdatedVersion, psQuery.Quantity, psQuery.DetailQuantity).
		CompanyID.Equals(req.User.CompanyID).
		Delta(productosStockUpdated, 1)

	if err := psQuery.Exec(); err != nil {
		return req.MakeErr("Error al obtener los productos stock:", err)
	}

	response := map[string]any{
		"productosStock": &productosStockV2,
		"movimientos":    &movimientosAgrupados,
	}

	return req.MakeResponse(response)
}
