package comercial

import (
	"app/comercial/types"
	"app/core"
	"app/db"
	"app/finanzas"
	finanzasTypes "app/finanzas/types"
	"app/logistica"
	logisticaTypes "app/logistica/types"
	"app/negocio"
	negocioTypes "app/negocio/types"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"golang.org/x/sync/errgroup"
)

func PostSaleOrder(req *core.HandlerArgs) core.HandlerResponse {
	nowTime := req.EffectiveSUnixTime()
	saleRequest := types.SaleOrder{}
	err := json.Unmarshal([]byte(*req.Body), &saleRequest)
	if err != nil {
		return req.MakeErr("Error al deserializar el body: " + err.Error())
	}

	isUpdate := saleRequest.ID > 0
	if isUpdate {
		if len(saleRequest.ActionsIncluded) == 0 {
			return req.MakeErr("Se requiere ActionsIncluded para actualizar la venta.")
		}
		for _, actionID := range saleRequest.ActionsIncluded {
			if actionID != 2 && actionID != 3 {
				return req.MakeErr("ActionsIncluded solo permite 2 (pago) y 3 (entrega).")
			}
		}
	}

	sale := saleRequest

	if isUpdate {
		core.Log("PostSaleOrder update requested. SaleID:", saleRequest.ID, "ActionsIncluded:", saleRequest.ActionsIncluded)
		existingSales := []types.SaleOrder{}
		query := db.Query(&existingSales)
		query.CompanyID.Equals(req.Usuario.EmpresaID).ID.Equals(saleRequest.ID).Limit(1)
		if err := query.Exec(); err != nil {
			return req.MakeErr("Error al obtener la venta a actualizar:", err)
		}
		if len(existingSales) == 0 {
			return req.MakeErr("No se encontró la venta a actualizar.")
		}

		sale = existingSales[0]
		sale.ActionsIncluded = saleRequest.ActionsIncluded
		// Preserve existing payment caja on delivery-only updates (payload may omit LastPaymentCajaID).
		if saleRequest.LastPaymentCajaID > 0 {
			sale.LastPaymentCajaID = saleRequest.LastPaymentCajaID
		}
		if saleRequest.WarehouseID > 0 {
			sale.WarehouseID = saleRequest.WarehouseID
		}
		if slices.Contains(saleRequest.ActionsIncluded, 2) {
			sale.DebtAmount = saleRequest.DebtAmount
		}
	} else {
		// Create rule: detail slices must keep one-to-one cardinality.
		if len(sale.DetailProductsIDs) != len(sale.DetailPrices) || len(sale.DetailProductsIDs) != len(sale.DetailQuantities) {
			return req.MakeErr("El registro posee propiedades incorrectas.")
		}

		for _, value := range slices.Concat(sale.DetailProductsIDs, sale.DetailQuantities, sale.DetailPrices) {
			if value == 0 {
				return req.MakeErr("Hay un valor incorrecto.")
			}
		}

		sale.Fecha = req.EffectiveFechaUnix()
		sale.Created = nowTime
		sale.Status = 1
	}

	if saleRequest.ClientInfo != nil {
		clientID, resolveClientError := resolveSaleOrderClientID(saleRequest.ClientInfo, req.Usuario.EmpresaID, req.Usuario.ID)
		if resolveClientError != nil {
			return req.MakeErr("Error al guardar el cliente de la venta:", resolveClientError)
		}
		sale.ClientID = clientID
	}

	if !isUpdate || slices.Contains(sale.ActionsIncluded, 3) {
		// Validate stock availability against the V2 split:
		//  - line items without LotID AND without SerialNumber draw from ProductStockV2.Quantity
		//  - items with a LotID/SerialNumber draw from the matching ProductStockDetail row
		if err := validateSaleStock(req, sale); err != nil {
			return req.MakeErr(err)
		}
	}

	sale.CompanyID = req.Usuario.EmpresaID
	sale.Updated = nowTime
	sale.UpdatedBy = req.Usuario.ID

	// 2 = Pago (Registro en Caja)
	if slices.Contains(sale.ActionsIncluded, 2) {
		sale.AddStatus(2)
		// Track when and who executed the latest payment action.
		sale.LastPaymentTime = nowTime
		sale.LastPaymentUser = req.Usuario.ID
		if sale.LastPaymentCajaID == 0 {
			return req.MakeErr("Se requiere LastPaymentCajaID para procesar el pago.")
		}
	}

	// 3 = Entrega (Movimiento de Almacén)
	if slices.Contains(sale.ActionsIncluded, 3) {
		sale.AddStatus(3)
		// Track when and who executed the latest delivery action.
		sale.DeliveryTime = nowTime
		sale.DeliveryUser = req.Usuario.ID
		if sale.WarehouseID == 0 {
			return req.MakeErr("Se requiere WarehouseID para procesar la entrega.")
		}

		if len(sale.DetailProductsIDs) == 0 {
			return req.MakeErr("No hay productos en el detalle para procesar la entrega.")
		}
	}

	saleActions := []int8{}
	if !isUpdate {
		sales := []types.SaleOrder{sale}
		saleActions = append(saleActions, 1)

		// Insertar el registro de venta para obtener el ID (autoincrement)
		if err := db.Insert(&sales); err != nil {
			return req.MakeErr("Error al registrar la venta:", err)
		}

		if sale.ID = sales[0].ID; sale.ID == 0 {
			return req.MakeErr("Error al obtener el ID de la venta.")
		}
	}

	eg := errgroup.Group{}

	// 2 = Pago (Registro en Caja)
	if slices.Contains(sale.ActionsIncluded, 2) {

		montoPago := sale.TotalAmount - sale.DebtAmount
		if montoPago != 0 {
			movimiento := finanzasTypes.CajaMovimientoInterno{
				CajaID:     sale.LastPaymentCajaID,
				DocumentID: sale.ID,
				Tipo:       8, // Cobro (Venta)
				Monto:      montoPago,
			}

			eg.Go(func() error {
				if err := finanzas.ApplyCajaMovimientos(req, []finanzasTypes.CajaMovimientoInterno{movimiento}); err != nil {
					core.Log("Error al aplicar movimiento de caja:", err)
					return core.Err("Error al registrar el movimiento de caja:", err)
				}
				return nil
			})
		}

		if sale.DebtAmount == 0 {
			saleActions = append(saleActions, 2)
		}
	}

	// 3 = Entrega (Movimiento de Almacén)
	if slices.Contains(sale.ActionsIncluded, 3) {
		core.Log("Incluyendo movimientos internos...", len(sale.DetailProductsIDs))

		movimientosInternos := []logisticaTypes.MovimientoInterno{}
		for i, productoID := range sale.DetailProductsIDs {
			if i >= len(sale.DetailQuantities) {
				break
			}
			cantidad := sale.DetailQuantities[i]
			if cantidad == 0 {
				continue
			}

			movimientosInternos = append(movimientosInternos, logisticaTypes.MovimientoInterno{
				WarehouseID:    sale.WarehouseID,
				ProductoID:     productoID,
				PresentacionID: core.GetIndex(sale.DetailProductPresentations, i),
				SerialNumber:   core.GetIndex(sale.DetailProductSkus, i),
				LotID:          core.GetIndex(sale.DetailProductLotIDs, i),
				DocumentID:     sale.ID,
				Tipo:           8,         // Entrega a cliente final (Venta)
				Cantidad:       -cantidad, // Salida de almacén
			})
		}

		core.Print(movimientosInternos)

		if len(movimientosInternos) > 0 {
			eg.Go(func() error {
				if err := logistica.ApplyMovimientos(req, movimientosInternos); err != nil {
					core.Log("Error al aplicar movimientos de almacén:", err)
					return core.Err("Error al procesar la salida de almacén:", err)
				}
				return nil
			})
		}

		saleActions = append(saleActions, 3)
	}

	if err := eg.Wait(); err != nil {
		return req.MakeErr(err)
	}

	if isUpdate {
		saleTable := db.Table[types.SaleOrder]()
		salesToUpdate := []types.SaleOrder{sale}
		if err := db.Update(&salesToUpdate,
			// Keep composite view columns in sync: {Fecha, Updated} must be updated together.
			saleTable.WarehouseID,
			saleTable.LastPaymentCajaID,
			saleTable.DebtAmount,
			saleTable.Updated,
			saleTable.UpdatedBy,
			saleTable.Status,
			saleTable.StatusTrace,
			saleTable.LastPaymentTime,
			saleTable.LastPaymentUser,
			saleTable.DeliveryTime,
			saleTable.DeliveryUser,
			saleTable.ClientID,
		); err != nil {
			return req.MakeErr("Error al actualizar la venta:", err)
		}
	}

	go func() {
		core.ScheduleCronAction(core.CronAction{
			CompanyID: req.Usuario.EmpresaID,
			ActionID:  2,
			// Keep the cron payload compact: company and fecha are the only inputs the reprocess action needs.
			Params: core.ExecArgs{Param1: int64(req.Usuario.EmpresaID), Param2: int64(sale.Fecha)},
		}, 10)

		if err := updateSaleSummaryForChange(sale, saleActions...); err != nil {
			core.Log("Error actualizando resumen de ventas:", err)
		}
	}()

	return req.MakeResponse(sale)
}

func resolveSaleOrderClientID(clientInfo *types.SaleOrderClientInfo, empresaID int32, usuarioID int32) (int32, error) {
	clientName := strings.TrimSpace(clientInfo.Name)
	clientRegistryNumber := strings.TrimSpace(clientInfo.RegistryNumber)
	if clientName == "" {
		return 0, core.Err("ClientInfo.Name es obligatorio.")
	}

	clientPersonType := negocioTypes.PersonTypeNatural
	if clientRegistryNumber != "" {
		// Preserve the provided registry number so identity matching can reuse existing client rows.
		clientPersonType = negocioTypes.PersonTypeCompany
	}

	clientProviders := []negocioTypes.ClientProvider{{
		Type:           negocioTypes.ClientProviderTypeClient,
		Name:           clientName,
		RegistryNumber: clientRegistryNumber,
		PersonType:     clientPersonType,
	}}
	// Sale-order client creation must never update an existing client record from frontend input
	// to prevent accidental data corruption of shared client/provider records.
	saveError := negocio.SaveClientProviders(&clientProviders, empresaID, usuarioID, true)
	if saveError != nil {
		return 0, saveError
	}
	if len(clientProviders) == 0 || clientProviders[0].ID <= 0 {
		return 0, core.Err("No se pudo resolver el ClientID de la venta.")
	}

	return clientProviders[0].ID, nil
}

// validateSaleStock ensures each sale-order line has enough stock available.
// Lines with no LotID and no SerialNumber draw from ProductStockV2.Quantity;
// lines with either field draw from the matching ProductStockDetail row.
func validateSaleStock(req *core.HandlerArgs, sale types.SaleOrder) error {
	if sale.WarehouseID == 0 {
		return core.Err("Se requiere WarehouseID para validar el stock.")
	}

	// Aggregate requested quantity by the physical storage bucket it hits.
	type lineKey struct {
		stockID      int64
		lotID        int32
		serialNumber string
	}
	requestedByKey := map[lineKey]int32{}
	for index, productID := range sale.DetailProductsIDs {
		quantity := core.GetIndex(sale.DetailQuantities, index)
		if quantity == 0 {
			continue
		}
		presentationID := core.GetIndex(sale.DetailProductPresentations, index)
		key := lineKey{
			stockID:      packProductStockIDForSale(sale.WarehouseID, productID, presentationID),
			lotID:        core.GetIndex(sale.DetailProductLotIDs, index),
			serialNumber: core.GetIndex(sale.DetailProductSkus, index),
		}
		requestedByKey[key] += quantity
	}
	if len(requestedByKey) == 0 {
		return nil
	}

	// Collect the distinct stock IDs so we can preload V2 and detail rows in parallel.
	stockIDSet := core.SliceSet[int64]{}
	needsDetailFetch := false
	for key := range requestedByKey {
		stockIDSet.Add(key.stockID)
		if key.lotID > 0 || key.serialNumber != "" {
			needsDetailFetch = true
		}
	}

	stockByID := map[int64]logisticaTypes.ProductStockV2{}
	detailsByKey := map[lineKey]logisticaTypes.ProductStockDetail{}

	eg := errgroup.Group{}
	eg.Go(func() error {
		stocks := []logisticaTypes.ProductStockV2{}
		q := db.Query(&stocks)
		q.Select(q.ID, q.Quantity, q.DetailQuantity).
			CompanyID.Equals(req.Usuario.EmpresaID).
			ID.In(stockIDSet.Values...)
		if err := q.Exec(); err != nil {
			return core.Err("Error al obtener el stock de productos:", err)
		}
		for _, stock := range stocks {
			stockByID[stock.ID] = stock
		}
		return nil
	})
	if needsDetailFetch {
		eg.Go(func() error {
			details := []logisticaTypes.ProductStockDetail{}
			q := db.Query(&details)
			q.Select().
				CompanyID.Equals(req.Usuario.EmpresaID).
				ProductStockID.In(stockIDSet.Values...)
			if err := q.Exec(); err != nil {
				return core.Err("Error al obtener el detalle de stock:", err)
			}
			for _, detail := range details {
				detailsByKey[lineKey{stockID: detail.ProductStockID, lotID: detail.LotID, serialNumber: detail.SerialNumber}] = detail
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}

	for key, requested := range requestedByKey {
		var available int32
		if key.lotID == 0 && key.serialNumber == "" {
			available = stockByID[key.stockID].Quantity
		} else {
			available = detailsByKey[key].Quantity
		}
		if available < requested {
			// Decompose the packed stock ID for a human-readable error.
			warehouseDigits := key.stockID / 1e14
			productDigits := (key.stockID / 1e5) % 1e9
			presentationDigits := (key.stockID / 10) % 1e4
			metadata := []string{
				fmt.Sprintf("Almacén: %v", warehouseDigits),
				fmt.Sprintf("Producto: %v", productDigits),
			}
			if presentationDigits > 0 {
				metadata = append(metadata, fmt.Sprintf("Presentación: %v", presentationDigits))
			}
			if key.serialNumber != "" {
				metadata = append(metadata, fmt.Sprintf("SKU: %v", key.serialNumber))
			}
			if key.lotID > 0 {
				metadata = append(metadata, fmt.Sprintf("Lote: %v", key.lotID))
			}
			return core.Err(strings.Join(metadata, " | ")+". ",
				fmt.Sprintf("Se necesita %v. Se posee en stock: %v", requested, available))
		}
	}
	return nil
}

// packProductStockIDForSale mirrors logistica.packProductStockID without importing the package
// to avoid a circular dependency (logistica already depends on comercial via some paths).
// Schema: WarehouseID.DecimalSize(5) + ProductID.DecimalSize(9) + PresentationID.DecimalSize(4),
// 19-digit starting budget.
func packProductStockIDForSale(warehouseID int32, productID int32, presentationID int16) int64 {
	return int64(warehouseID)*1e14 + int64(productID)*1e5 + int64(presentationID)*10
}

func GetSaleOrderByIDs(req *core.HandlerArgs) core.HandlerResponse {
	saleOrderIDRecords := req.ExtractCacheVersionValues()

	saleOrderIDs := core.Map(saleOrderIDRecords, func(e db.IDCacheVersion) int64 { return e.ID })

	saleOrders := []types.SaleOrder{}
	query := db.Query(&saleOrders).CompanyID.Equals(req.Usuario.EmpresaID)

	if err := query.ID.In(saleOrderIDs...).Exec(); err != nil {
		return req.MakeErr("Error al obtener los productos.", err)
	}

	return core.MakeResponse(req, &saleOrders)
}
