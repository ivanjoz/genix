package logistics

import (
	"app/core"
	"app/db"
	logisticsTypes "app/logistics/types"
	businessTypes "app/business/types"
	"encoding/json"
	"slices"
)

func GetProductSupply(req *core.HandlerArgs) core.HandlerResponse {
	productSupplyRecords := []logisticsTypes.ProductSupply{}
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

func PostProductSupply(req *core.HandlerArgs) core.HandlerResponse {
	productSupplyRecord := logisticsTypes.ProductSupply{}
	if deserializeError := json.Unmarshal([]byte(*req.Body), &productSupplyRecord); deserializeError != nil {
		core.Log("PostProductSupply deserialization error:", deserializeError)
		return req.MakeErr("Error al deserializar el body.", deserializeError)
	}

	productSupplyRecord.ProviderSupply = sanitizeProviderSupplyRows(productSupplyRecord.ProviderSupply)

	if productSupplyRecord.ProductID <= 0 {
		return req.MakeErr("Debe enviar un ProductID válido.")
	}
	if productSupplyRecord.MinimunStock < 0 {
		return req.MakeErr("El stock mínimo no puede ser negativo.")
	}
	if productSupplyRecord.SalesPerDayEstimated < 0 {
		return req.MakeErr("Las ventas por día estimadas no pueden ser negativas.")
	}

	if validationError := validateProviderSupplyRows(req, productSupplyRecord.ProviderSupply); validationError != nil {
		return req.MakeErr(validationError)
	}

	currentTimestamp := core.SUnixTime()
	productSupplyRecord.CompanyID = req.User.CompanyID
	productSupplyRecord.Status = 1
	productSupplyRecord.Updated = currentTimestamp
	productSupplyRecord.UpdatedBy = req.User.ID

	productSupplyRecords := []logisticsTypes.ProductSupply{productSupplyRecord}
	if mergeError := db.Merge(&productSupplyRecords, nil,
		func(previousProductSupply, currentProductSupply *logisticsTypes.ProductSupply) bool {
			// Keep the product key immutable and refresh only mutable configuration fields.
			currentProductSupply.CompanyID = req.User.CompanyID
			currentProductSupply.ProductID = previousProductSupply.ProductID
			currentProductSupply.Status = 1
			currentProductSupply.Updated = currentTimestamp
			currentProductSupply.UpdatedBy = req.User.ID
			return true
		},
		func(currentProductSupply *logisticsTypes.ProductSupply) {
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

	core.Log("PostProductSupply saved:", "product_id=", productSupplyRecord.ProductID, "provider_count=", len(productSupplyRecord.ProviderSupply))
	return req.MakeResponse(productSupplyRecords[0])
}

func sanitizeProviderSupplyRows(providerSupplyRows []logisticsTypes.ProductSupplyProviderRow) []logisticsTypes.ProductSupplyProviderRow {
	sanitizedProviderSupplyRows := make([]logisticsTypes.ProductSupplyProviderRow, 0, len(providerSupplyRows))

	for _, providerSupplyRow := range providerSupplyRows {
		if providerSupplyRow.ProviderID <= 0 && providerSupplyRow.Capacity == 0 && providerSupplyRow.DeliveryTime == 0 && providerSupplyRow.Price == 0 {
			continue
		}
		sanitizedProviderSupplyRows = append(sanitizedProviderSupplyRows, providerSupplyRow)
	}

	return sanitizedProviderSupplyRows
}

func validateProviderSupplyRows(req *core.HandlerArgs, providerSupplyRows []logisticsTypes.ProductSupplyProviderRow) error {
	if len(providerSupplyRows) == 0 {
		return nil
	}

	providerIDs := make([]int32, 0, len(providerSupplyRows))
	for _, providerSupplyRow := range providerSupplyRows {
		if providerSupplyRow.ProviderID <= 0 {
			return core.Err("Cada fila de proveedor debe tener un proveedor válido.")
		}
		if providerSupplyRow.Capacity < 0 {
			return core.Err("La capacidad no puede ser negativa.")
		}
		if providerSupplyRow.DeliveryTime < 0 {
			return core.Err("El tiempo de entrega no puede ser negativo.")
		}
		if providerSupplyRow.Price < 0 {
			return core.Err("El precio no puede ser negativo.")
		}
		if slices.Contains(providerIDs, providerSupplyRow.ProviderID) {
			return core.Err("No se puede repetir el mismo proveedor en un product.")
		}
		providerIDs = append(providerIDs, providerSupplyRow.ProviderID)
	}

	providers := []businessTypes.ClientProvider{}
	providerQuery := db.Query(&providers)
	providerTable := db.Table[businessTypes.ClientProvider]()
	providerQuery.Select(providerTable.ID, providerTable.Type, providerTable.Status).
		CompanyID.Equals(req.User.CompanyID).
		ID.In(providerIDs...)

	if queryError := providerQuery.Exec(); queryError != nil {
		return core.Err("Error al validar los proveedores.", queryError)
	}

	if len(providers) != len(providerIDs) {
		return core.Err("Uno o más proveedores no existen.")
	}

	for _, providerRecord := range providers {
		if providerRecord.Status <= 0 {
			return core.Err("Uno o más proveedores están inactivos.")
		}
		if providerRecord.Type != businessTypes.ClientProviderTypeProvider {
			return core.Err("Uno o más registros seleccionados no son proveedores.")
		}
	}

	return nil
}

/* GET: WAREHOUSE MOVIMIENTOS GROUPED */

type DateProductMovements struct {
	Date              int16
	DetailProductsIDs  []int32
	DetailInflows      []int32
	DetailOutflows     []int32
	DetailMinimumStock []int32
	Updated            int16 `json:"upd"`
}

func GetAlmacenMovimientosGrouped(req *core.HandlerArgs) core.HandlerResponse {

	movimientosFecha := int16(req.GetQueryInt("movimientos"))
	productosStockUpdated := req.GetQueryInt("productosStock")

	movimientos := []logisticsTypes.WarehouseProductMovement{}

	query := db.Query(&movimientos).
		CompanyID.Equals(req.User.CompanyID).
		Date.GreaterEqual(movimientosFecha)

	if err := query.GroupBy(query.Date, query.ProductID, query.Tipo, query.Quantity.Sum()).Exec(); err != nil {
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

		// Split the signed grouped quantity into inflow/outflow columns for the cached payload.
		if movimiento.Quantity > 0 {
			dateAccumulator.record.DetailInflows[productIndex] += movimiento.Quantity
		} else if movimiento.Quantity < 0 {
			dateAccumulator.record.DetailOutflows[productIndex] += -movimiento.Quantity
		}
	}

	movimientosAgrupados := make([]DateProductMovements, 0, len(dailyGroupedRecords))
	for _, dateAccumulator := range dailyGroupedRecords {
		movimientosAgrupados = append(movimientosAgrupados, *dateAccumulator.record)
	}

	// Productos Stock (V2). "Quantity" on the response is the combined bucket
	// so consumers stay compatible with the old shape without needing detail rows here.
	productosStockV2 := []logisticsTypes.ProductStock{}

	psQuery := db.Query(&productosStockV2).AllowFilter()
	psQuery.Select(psQuery.ID, psQuery.Updated, psQuery.Quantity, psQuery.DetailQuantity)

	if productosStockUpdated == 0 {
		psQuery.Status.Equals(1)
	} else {
		psQuery.Updated.GreaterEqual(productosStockUpdated)
	}

	if err := psQuery.Exec(); err != nil {
		return req.MakeErr("Error al obtener los productos stock:", err)
	}

	response := map[string]any{
		"productosStock": &productosStockV2,
		"movimientos":    &movimientosAgrupados,
	}

	return req.MakeResponse(response)
}
