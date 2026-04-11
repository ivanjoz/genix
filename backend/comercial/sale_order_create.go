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
	previousStatusTrace := int8(0)

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
		previousStatusTrace = sale.StatusTrace
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
		stockSolicitadoPorID := map[string]int32{}

		// Agrupa por stock real para validar una sola vez por combinacion almacen-producto-presentacion-sku-lote.
		for index, productID := range sale.DetailProductsIDs {
			stockID := db.MakeKeyConcat(
				sale.WarehouseID,
				productID,
				core.GetIndex(sale.DetailProductPresentations, index),
				core.GetIndex(sale.DetailProductSkus, index),
				core.GetIndex(sale.DetailProductLots, index),
			)
			stockSolicitadoPorID[stockID] += sale.DetailQuantities[index]
		}

		if len(stockSolicitadoPorID) > 0 {
			productsCurrentStock := []logisticaTypes.ProductStock{}

			query := db.Query(&productsCurrentStock)
			err := query.Select(query.ID, query.Quantity, query.Status).
				CompanyID.Equals(req.Usuario.EmpresaID).
				ID.In(core.MapToKeys(stockSolicitadoPorID)...).
				Exec()

			if err != nil {
				return req.MakeErr("Error al obtener el stock de productos:", err)
			}

			currentStockByID := core.SliceToMapE(productsCurrentStock,
				func(e logisticaTypes.ProductStock) string { return e.ID })

			for stockID, cantidadSolicitada := range stockSolicitadoPorID {
				stockActual := currentStockByID[stockID]
				if stockActual.Quantity < cantidadSolicitada {
					keyParser := db.KeyParser{Key: stockID}
					metadata := []string{
						fmt.Sprintf(`Almacén: %v`, keyParser.GetNumber(0)),
						fmt.Sprintf(`Producto: %v`, keyParser.GetNumber(1)),
					}

					if keyParser.GetNumber(2) > 0 {
						metadata = append(metadata, fmt.Sprintf(`Presentación: %v`, keyParser.GetNumber(2)))
					}
					if keyParser.GetString(3) != "" {
						metadata = append(metadata, fmt.Sprintf(`Sku: %v`, keyParser.GetString(3)))
					}
					if keyParser.GetString(4) != "" {
						metadata = append(metadata, fmt.Sprintf(`Lote: %v`, keyParser.GetString(3)))
					}

					return req.MakeErr(strings.Join(metadata, " | ")+". ", fmt.Sprintf(`Se necesita %v. Se posee en stock: %v`, cantidadSolicitada, stockActual.Quantity))
				}
			}

			core.Log("PostSaleOrder stock validado. Items:", len(stockSolicitadoPorID), "Stocks encontrados:", len(currentStockByID))
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

		// Revisa si hay stock necesario
	}

	// Keep the compact 1..9 trace synchronized with the executed action flow.
	nextStatusTrace, err := CalculateSaleOrderStatusTrace(previousStatusTrace, sale.ActionsIncluded)
	if err != nil {
		return req.MakeErr("No se pudo calcular el StatusTrace de la venta:", err)
	}
	sale.StatusTrace = nextStatusTrace

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
				SKU:            core.GetIndex(sale.DetailProductSkus, i),
				Lote:           core.GetIndex(sale.DetailProductLots, i),
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
