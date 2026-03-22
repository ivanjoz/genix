package comercial

import (
	"app/comercial/types"
	"app/core"
	"app/db"
	"app/finanzas"
	finanzasTypes "app/finanzas/types"
	"app/logistica"
	logisticaTypes "app/logistica/types"
	coreTypes "app/core/types"
	"encoding/json"
	"slices"
	"time"

	"golang.org/x/sync/errgroup"
)

func PostSaleOrder(req *core.HandlerArgs) core.HandlerResponse {
	nowTime := core.SUnixTime()
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
	stockSolicitadoPorID := map[string]int32{}
	stockActualPorID := map[string]*logisticaTypes.ProductoStock{}
	if isUpdate {
		core.Log("PostSaleOrder update requested. SaleID:", saleRequest.ID, "ActionsIncluded:", saleRequest.ActionsIncluded)
		existingSales := []types.SaleOrder{}
		query := db.Query(&existingSales)
		query.EmpresaID.Equals(req.Usuario.EmpresaID).ID.Equals(saleRequest.ID).Limit(1)
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
		if saleRequest.AlmacenID > 0 {
			sale.AlmacenID = saleRequest.AlmacenID
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

		sale.Fecha = core.TimeToFechaUnix(time.Now())
		sale.Created = nowTime
		sale.Status = 1
	}

	if !isUpdate || slices.Contains(sale.ActionsIncluded, 3) {
		// Agrupa por stock real para validar una sola vez por combinacion almacen-producto-presentacion-sku-lote.
		for index, productID := range sale.DetailProductsIDs {
			stockID := db.Concat62(
				sale.AlmacenID,
				productID,
				core.GetIndex(sale.DetailProductPresentations, index),
				core.GetIndex(sale.DetailProductSkus, index),
				core.GetIndex(sale.DetailProductLots, index),
			)
			stockSolicitadoPorID[stockID] += sale.DetailQuantities[index]
		}

		if len(stockSolicitadoPorID) > 0 {
			productosStock := []logisticaTypes.ProductoStock{}
			query := db.Query(&productosStock)
			err := query.Select(
				query.ID,
				query.Cantidad,
				query.CostoUn,
				query.Status,
			).
				EmpresaID.Equals(req.Usuario.EmpresaID).
				ID.In(core.MapToKeys(stockSolicitadoPorID)...).
				Exec()

			if err != nil {
				return req.MakeErr("Error al obtener el stock de productos:", err)
			}

			for index := range productosStock {
				stockActualPorID[productosStock[index].ID] = &productosStock[index]
			}

			for stockID, cantidadSolicitada := range stockSolicitadoPorID {
				stockActual := stockActualPorID[stockID]
				cantidadDisponible := int32(0)
				if stockActual != nil && stockActual.Status == 1 {
					cantidadDisponible = stockActual.Cantidad
				}
				if cantidadDisponible < cantidadSolicitada {
					return req.MakeErr("No hay stock suficiente para completar la venta.")
				}
			}

			core.Log("PostSaleOrder stock validado. Items:", len(stockSolicitadoPorID), "Stocks encontrados:", len(stockActualPorID))
		}
	}

	sale.EmpresaID = req.Usuario.EmpresaID
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
	} else if !isUpdate {
		sale.OrderPendingPaymentUpdated = sale.Updated
	}

	// 3 = Entrega (Movimiento de Almacén)
	if slices.Contains(sale.ActionsIncluded, 3) {
		sale.AddStatus(3)
		// Track when and who executed the latest delivery action.
		sale.DeliveryTime = nowTime
		sale.DeliveryUser = req.Usuario.ID
		if sale.AlmacenID == 0 {
			return req.MakeErr("Se requiere AlmacenID para procesar la entrega.")
		}

		if len(sale.DetailProductsIDs) == 0 {
			return req.MakeErr("No hay productos en el detalle para procesar la entrega.")
		}

		// Revisa si hay stock necesario

	} else if !isUpdate {
		sale.OrderPendingDeliveryUpdated = sale.Updated
	}

	if sale.Status == 4 {
		sale.OrderCompletedUpdated = sale.Updated
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

			stockID := db.Concat62(
				sale.AlmacenID,
				productoID,
				core.GetIndex(sale.DetailProductPresentations, i),
				core.GetIndex(sale.DetailProductSkus, i),
				core.GetIndex(sale.DetailProductLots, i),
			)

			movimientosInternos = append(movimientosInternos, logisticaTypes.MovimientoInterno{
				AlmacenID:      sale.AlmacenID,
				ProductoID:     productoID,
				PresentacionID: core.GetIndex(sale.DetailProductPresentations, i),
				SKU:            core.GetIndex(sale.DetailProductSkus, i),
				Lote:           core.GetIndex(sale.DetailProductLots, i),
				DocumentID:     sale.ID,
				Tipo:           8,         // Entrega a cliente final (Venta)
				Cantidad:       -cantidad, // Salida de almacén
				Stock:          stockActualPorID[stockID],
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
			saleTable.Fecha,
			saleTable.AlmacenID,
			saleTable.LastPaymentCajaID,
			saleTable.DebtAmount,
			saleTable.Updated,
			saleTable.UpdatedBy,
			saleTable.Status,
			saleTable.OrderPendingPaymentUpdated,
			saleTable.OrderPendingDeliveryUpdated,
			saleTable.OrderCompletedUpdated,
			saleTable.LastPaymentTime,
			saleTable.LastPaymentUser,
			saleTable.DeliveryTime,
			saleTable.DeliveryUser,
			saleTable.DetailProductsIDs,
		); err != nil {
			return req.MakeErr("Error al actualizar la venta:", err)
		}
	}

	if err := updateSaleSummaryForChange(sale, saleActions...); err != nil {
		core.Log("Error actualizando resumen de ventas:", err)
	}
	
	core.ScheduleCronAction(coreTypes.CronAction{
		CompanyID: req.Usuario.EmpresaID,
		ActionID: 2, 
		Params: coreTypes.CronActionParams{ Param1: int64(sale.Fecha) },
	},10)

	return req.MakeResponse(sale)
}
