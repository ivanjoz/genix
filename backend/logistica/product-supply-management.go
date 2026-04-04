package logistica

import (
	"app/core"
	"app/db"
	logisticaTypes "app/logistica/types"
	negocioTypes "app/negocio/types"
	"encoding/json"
	"slices"
)

func GetProductSupply(req *core.HandlerArgs) core.HandlerResponse {
	productSupplyRecords := []logisticaTypes.ProductSupply{}
	productSupplyQuery := db.Query(&productSupplyRecords)
	productSupplyQuery.Select().
		CompanyID.Equals(req.Usuario.EmpresaID).
		Status.Equals(1)

	if queryError := productSupplyQuery.Exec(); queryError != nil {
		core.Log("GetProductSupply query error:", queryError)
		return req.MakeErr("Error al obtener la configuración de abastecimiento.", queryError)
	}

	core.Log("GetProductSupply result_count:", len(productSupplyRecords))
	return req.MakeResponse(productSupplyRecords)
}

func PostProductSupply(req *core.HandlerArgs) core.HandlerResponse {
	productSupplyRecord := logisticaTypes.ProductSupply{}
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
	productSupplyRecord.CompanyID = req.Usuario.EmpresaID
	productSupplyRecord.Status = 1
	productSupplyRecord.Updated = currentTimestamp
	productSupplyRecord.UpdatedBy = req.Usuario.ID

	productSupplyRecords := []logisticaTypes.ProductSupply{productSupplyRecord}
	if mergeError := db.Merge(&productSupplyRecords, nil,
		func(previousProductSupply, currentProductSupply *logisticaTypes.ProductSupply) bool {
			// Keep the product key immutable and refresh only mutable configuration fields.
			currentProductSupply.CompanyID = req.Usuario.EmpresaID
			currentProductSupply.ProductID = previousProductSupply.ProductID
			currentProductSupply.Status = 1
			currentProductSupply.Updated = currentTimestamp
			currentProductSupply.UpdatedBy = req.Usuario.ID
			return true
		},
		func(currentProductSupply *logisticaTypes.ProductSupply) {
			// Fill server-owned metadata on insert so clients only send configuration fields.
			currentProductSupply.CompanyID = req.Usuario.EmpresaID
			currentProductSupply.Status = 1
			currentProductSupply.Updated = currentTimestamp
			currentProductSupply.UpdatedBy = req.Usuario.ID
		},
	); mergeError != nil {
		core.Log("PostProductSupply merge error:", mergeError)
		return req.MakeErr("Error al guardar la configuración de abastecimiento.", mergeError)
	}

	core.Log("PostProductSupply saved:", "product_id=", productSupplyRecord.ProductID, "provider_count=", len(productSupplyRecord.ProviderSupply))
	return req.MakeResponse(productSupplyRecords[0])
}

func sanitizeProviderSupplyRows(providerSupplyRows []logisticaTypes.ProductSupplyProviderRow) []logisticaTypes.ProductSupplyProviderRow {
	sanitizedProviderSupplyRows := make([]logisticaTypes.ProductSupplyProviderRow, 0, len(providerSupplyRows))

	for _, providerSupplyRow := range providerSupplyRows {
		if providerSupplyRow.ProviderID <= 0 && providerSupplyRow.Capacity == 0 && providerSupplyRow.DeliveryTime == 0 && providerSupplyRow.Price == 0 {
			continue
		}
		sanitizedProviderSupplyRows = append(sanitizedProviderSupplyRows, providerSupplyRow)
	}

	return sanitizedProviderSupplyRows
}

func validateProviderSupplyRows(req *core.HandlerArgs, providerSupplyRows []logisticaTypes.ProductSupplyProviderRow) error {
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
			return core.Err("No se puede repetir el mismo proveedor en un producto.")
		}
		providerIDs = append(providerIDs, providerSupplyRow.ProviderID)
	}

	providers := []negocioTypes.ClientProvider{}
	providerQuery := db.Query(&providers)
	providerTable := db.Table[negocioTypes.ClientProvider]()
	providerQuery.Select(providerTable.ID, providerTable.Type, providerTable.Status).
		EmpresaID.Equals(req.Usuario.EmpresaID).
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
		if providerRecord.Type != negocioTypes.ClientProviderTypeProvider {
			return core.Err("Uno o más registros seleccionados no son proveedores.")
		}
	}

	return nil
}

/* GET: ALMACEN MOVIMIENTOS GROUPED */

type FechaProductoMovimientos struct {
	Fecha              int16
	DetailProductsIDs  []int32
	DetailInflows      []int32
	DetailOutflows     []int32
	DetailMinimumStock []int32
	Updated            int16 `json:"upd"`
}

func GetAlmacenMovimientosGrouped(req *core.HandlerArgs) core.HandlerResponse {

	movimientosFecha := int16(req.GetQueryInt("movimientos"))
	productosStockUpdated := req.GetQueryInt("productosStock")

	movimientos := []logisticaTypes.WarehouseProductMovement{}

	query := db.Query(&movimientos).
		CompanyID.Equals(req.Usuario.EmpresaID).
		Fecha.GreaterEqual(movimientosFecha)

	if err := query.GroupBy(query.Fecha, query.ProductoID, query.Tipo, query.Cantidad.Sum()).Exec(); err != nil {
		return req.MakeErr("Error al obtener los registros del almacén:", err)
	}

	core.PrintTable(movimientos, 30, 30, "Fecha", "ProductoID", "WarehouseID", "AlmacenCantidad")

	// Keep one row per date and reuse the product position to avoid duplicated product IDs.
	type fechaMovimientosAccumulator struct {
		record                  *FechaProductoMovimientos
		productIndexByProductID map[int32]int
	}

	dailyGroupedRecords := map[int16]*fechaMovimientosAccumulator{}
	for _, movimiento := range movimientos {
		fechaAccumulator, exists := dailyGroupedRecords[movimiento.Fecha]
		if !exists {
			fechaAccumulator = &fechaMovimientosAccumulator{
				record: &FechaProductoMovimientos{
					Fecha: movimiento.Fecha, Updated: movimiento.Fecha,
				},
				productIndexByProductID: map[int32]int{},
			}
			dailyGroupedRecords[movimiento.Fecha] = fechaAccumulator
		}

		productIndex, productExists := fechaAccumulator.productIndexByProductID[movimiento.ProductoID]
		if !productExists {
			productIndex = len(fechaAccumulator.record.DetailProductsIDs)
			fechaAccumulator.productIndexByProductID[movimiento.ProductoID] = productIndex
			fechaAccumulator.record.DetailProductsIDs = append(fechaAccumulator.record.DetailProductsIDs, movimiento.ProductoID)
			fechaAccumulator.record.DetailInflows = append(fechaAccumulator.record.DetailInflows, 0)
			fechaAccumulator.record.DetailOutflows = append(fechaAccumulator.record.DetailOutflows, 0)
		}

		// Split the signed grouped quantity into inflow/outflow columns for the cached payload.
		if movimiento.Cantidad > 0 {
			fechaAccumulator.record.DetailInflows[productIndex] += movimiento.Cantidad
		} else if movimiento.Cantidad < 0 {
			fechaAccumulator.record.DetailOutflows[productIndex] += -movimiento.Cantidad
		}
	}

	movimientosAgrupados := make([]FechaProductoMovimientos, 0, len(dailyGroupedRecords))
	for _, fechaAccumulator := range dailyGroupedRecords {
		movimientosAgrupados = append(movimientosAgrupados, *fechaAccumulator.record)
	}

	// Productos Stock
	productosStock := []logisticaTypes.ProductStock{}

	psQuery := db.Query(&productosStock).AllowFilter()
	psQuery.Select(psQuery.ID, psQuery.Updated, psQuery.Cantidad)

	if productosStockUpdated == 0 {
		psQuery.Status.Equals(1)
	} else {
		psQuery.Updated.GreaterEqual(productosStockUpdated)
	}

	if err := psQuery.Exec(); err != nil {
		return req.MakeErr("Error al obtener los productos stock:", err)
	}

	response := map[string]any{
		"productosStock": &productosStock,
		"movimientos":    &movimientosAgrupados,
	}

	return req.MakeResponse(response)
}
