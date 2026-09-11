package sales

import (
	"app/core"
	crm "app/crm/types"
	"app/db"
	finance "app/finance/types"
	invoicing "app/invoicing/types"
	logistics "app/logistics/types"
	production "app/production/types"
	"app/sales/types"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

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

	// The series the sale is issued under, resolved in the create branch below and
	// used to number its document. Nil when the till named none, and always nil on
	// an update: the series is part of the sale id and cannot change.
	var issueSeries *invoicing.InvoiceSeries

	if isUpdate {
		core.Log("PostSaleOrder update requested. SaleID:", saleRequest.ID, "ActionsIncluded:", saleRequest.ActionsIncluded)
		existingSales := []types.SaleOrder{}
		query := db.Query(&existingSales)
		query.CompanyID.Equals(req.User.CompanyID).ID.Equals(saleRequest.ID).Limit(1)
		if err := query.Exec(); err != nil {
			return req.MakeErr("Error al obtener la venta a actualizar:", err)
		}
		if len(existingSales) == 0 {
			return req.MakeErr("No se encontró la venta a actualizar.")
		}

		sale = existingSales[0]
		sale.ActionsIncluded = saleRequest.ActionsIncluded
		// Preserve existing payment cashBank on delivery-only updates (payload may omit LastPaymentCajaID).
		if saleRequest.LastPaymentCajaID > 0 {
			sale.LastPaymentCajaID = saleRequest.LastPaymentCajaID
		}
		if saleRequest.WarehouseID > 0 {
			sale.WarehouseID = saleRequest.WarehouseID
		}
		// PaymentDueDate is editable post-creation (e.g., reschedule a payment).
		sale.PaymentDueDate = saleRequest.PaymentDueDate
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

		// Resolve the divisor and the prices against the catalog rather than trusting what
		// the client sent. This is the only place that loads products, and it is what stops
		// a crafted request from booking a sale at an invented price.
		if err := validateSaleOrderLines(req, &sale); err != nil {
			return req.MakeErr(err)
		}

		// Runs on the total the catalog just resolved, and before the client is
		// created: a sale that cannot be issued must not leave a client row behind.
		// The series it returns is what the document is numbered in, further down.
		issueSeries, err = resolveSaleOrderIssueSeries(req, &sale)
		if err != nil {
			return req.MakeErr(err)
		}

		sale.Date = core.FechaUnix()
		sale.Created = nowTime
		sale.Status = 1
	}

	if saleRequest.ClientInfo != nil {
		clientID, resolveClientError := resolveSaleOrderClientID(saleRequest.ClientInfo, req.User.CompanyID, req.User.ID)
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

	sale.CompanyID = req.User.CompanyID
	sale.Updated = nowTime
	sale.UpdatedBy = req.User.ID

	// 2 = Pago (Registro en CashBank)
	if slices.Contains(sale.ActionsIncluded, 2) {
		sale.AddStatus(2)
		// Track when and who executed the latest payment action.
		sale.LastPaymentTime = nowTime
		sale.LastPaymentUser = req.User.ID
		if sale.LastPaymentCajaID == 0 {
			return req.MakeErr("Se requiere LastPaymentCajaID para procesar el pago.")
		}
	}

	// 3 = Entrega (Movimiento de Almacén)
	if slices.Contains(sale.ActionsIncluded, 3) {
		sale.AddStatus(3)
		// Track when and who executed the latest delivery action.
		sale.DeliveryTime = nowTime
		sale.DeliveryUser = req.User.ID
		if sale.WarehouseID == 0 {
			return req.MakeErr("Se requiere WarehouseID para procesar la entrega.")
		}

		if len(sale.DetailProductsIDs) == 0 {
			return req.MakeErr("No hay productos en el detalle para procesar la entrega.")
		}
	}

	saleActions := []int8{}
	if !isUpdate {
		// The sale id carries the correlativo of its series, so minting it and
		// reserving the comprobante's number are one act — and the document is
		// validated before that number is spent, so a sale that cannot be invoiced
		// is refused with nothing persisted and no gap left in the series.
		var pendingDocument *invoicing.InvoiceDocument
		if issueSeries != nil {
			pendingDocument, err = invoicing.PrepareDocumentForSale(
				req.User.CompanyID, req.User.ID, &sale, issueSeries)
			if err != nil {
				return req.MakeErr("No se pudo generar el comprobante de la venta:", err)
			}
		} else {
			// No comprobante: the id comes from the series-0 counter, which nothing
			// is ever declared from.
			saleID, idErr := types.MakeSaleOrderID(req.User.CompanyID, 0)
			if idErr != nil {
				return req.MakeErr("Error al obtener el ID de la venta:", idErr)
			}
			sale.ID = saleID
		}

		sales := []types.SaleOrder{sale}
		saleActions = append(saleActions, 1)

		if err := db.Insert(&sales); err != nil {
			return req.MakeErr("Error al registrar la venta:", err)
		}

		// Written after the sale, because every send rebuilds the document from it.
		// This is the one step that can leave the two out of step: the sale is
		// already committed and there is no transaction to undo it, so the failure
		// is reported and logged rather than silently swallowed.
		if pendingDocument != nil {
			if docErr := invoicing.SaveNewDocument(pendingDocument); docErr != nil {
				core.Log("VENTA SIN COMPROBANTE:: la venta", sale.ID,
					"se registró pero su comprobante no pudo guardarse:", docErr)
				return req.MakeErr("La venta se registró pero no se pudo guardar su comprobante:", docErr)
			}
			invoicing.ScheduleEmitPendingSweep(req.User.CompanyID)
		}
	}

	eg := errgroup.Group{}

	// 2 = Pago (Registro en CashBank)
	if slices.Contains(sale.ActionsIncluded, 2) {

		montoPago := sale.TotalAmount - sale.DebtAmount
		if montoPago != 0 {
			movimiento := finance.InternalCashMovement{
				CashBankID: sale.LastPaymentCajaID,
				DocumentID: sale.ID,
				Type:       finance.CashMovementTypeSaleCollection,
				Amount:     montoPago,
			}

			eg.Go(func() error {
				if err := finance.ApplyCashBankMovement(req, []finance.InternalCashMovement{movimiento}); err != nil {
					core.Log("Error al aplicar movimiento de cashBank:", err)
					return core.Err("Error al registrar el movimiento de cashBank:", err)
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

		movimientosInternos := []logistics.InternalMovement{}
		for i, productID := range sale.DetailProductsIDs {
			if i >= len(sale.DetailQuantities) {
				break
			}
			packedQuantity := sale.DetailQuantities[i]
			if packedQuantity == 0 {
				continue
			}

			// The order line is packed; the ledger keeps the halves apart so it can accumulate
			// and be SUM()-ed. This is the single conversion point between the two forms.
			lineQuantity := core.UnpackQuantityLine(packedQuantity).Negate() // Salida de almacén
			lineDivisor := core.GetIndex(sale.DetailSubDivisor, i)
			if lineDivisor <= 0 {
				lineDivisor = core.QuantityDivisorNone
			}

			movimientosInternos = append(movimientosInternos, logistics.InternalMovement{
				WarehouseID:    sale.WarehouseID,
				ProductID:      productID,
				PresentationID: core.GetIndex(sale.DetailProductPresentations, i),
				SerialNumber:   core.GetIndex(sale.DetailProductSkus, i),
				LotID:          core.GetIndex(sale.DetailProductLotIDs, i),
				DocumentID:     sale.ID,
				Type:           8, // Entrega a cliente final (Venta)
				Quantity:       lineQuantity.Units,
				SubQuantity:    lineQuantity.Sub,
				SubDivisor:     lineDivisor,
			})
		}

		core.Print(movimientosInternos)

		if len(movimientosInternos) > 0 {
			eg.Go(func() error {
				if err := logistics.ApplyMovimientos(req, movimientosInternos); err != nil {
					core.Log("Error al aplicar movimientos de almacén:", err)
					return core.Err("Error al procesar la salida de almacén:", err)
				}
				return nil
			})
		}

		saleActions = append(saleActions, 3)
	}

	// Scheduled last so the closure reads the final saleActions built by the blocks above.
	eg.Go(func() error {
		if err := updateSaleSummaryForChange(sale, saleActions...); err != nil {
			core.Log("Error actualizando resumen de ventas:", err)
			return core.Err("Error al actualizar el resumen de ventas:", err)
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		return req.MakeErr(err)
	}

	if isUpdate {
		saleTable := db.TableOf[types.SaleOrder]()
		salesToUpdate := []types.SaleOrder{sale}
		if err := db.Update(&salesToUpdate,
			// Keep composite view columns in sync: {Date, Updated} must be updated together.
			saleTable.WarehouseID,
			saleTable.LastPaymentCajaID,
			saleTable.DebtAmount,
			saleTable.Updated,
			saleTable.UpdatedBy,
			saleTable.Status,
			saleTable.LastPaymentTime,
			saleTable.LastPaymentUser,
			saleTable.DeliveryTime,
			saleTable.DeliveryUser,
			saleTable.ClientID,
			saleTable.PaymentDueDate,
		); err != nil {
			return req.MakeErr("Error al actualizar la venta:", err)
		}
	}

	go func() {
		core.ScheduleCronAction(core.CronAction{
			CompanyID: req.User.CompanyID,
			ActionID:  2,
			// Keep the cron payload compact: company and date are the only inputs the reprocess action needs.
			Params: core.ExecArgs{Param1: int64(req.User.CompanyID), Param2: int64(sale.Date)},
		}, 10)
	}()

	return req.MakeResponse(sale)
}

// validateSaleOrderLines resolves each line against the product catalog: the sub-unit
// divisor, whether a sub-unit part is even allowed, and the prices. PostSaleOrder is the
// only place that loads products, and without it DetailPrices and TotalAmount arrive
// straight off the request body unchecked — a client could book a sale at any price.
func validateSaleOrderLines(req *core.HandlerArgs, sale *types.SaleOrder) error {
	productIDs := core.SliceSet[int32]{}
	for _, productID := range sale.DetailProductsIDs {
		productIDs.AddIf(productID)
	}
	if productIDs.IsEmpty() {
		return nil
	}

	products := []production.Product{}
	query := db.Query(&products)
	query.Select(query.ID, query.Name, query.FinalPrice, query.SbuQuantity, query.SbuFinalPrice).
		CompanyID.Equals(req.User.CompanyID).
		ID.In(productIDs.Values...)
	if err := query.Exec(); err != nil {
		return core.Err("Error al obtener los productos de la venta:", err)
	}

	productByID := make(map[int32]production.Product, len(products))
	for _, product := range products {
		productByID[product.ID] = product
	}

	sale.DetailSubDivisor = make([]int16, len(sale.DetailProductsIDs))
	sale.DetailSubPrices = make([]int32, len(sale.DetailProductsIDs))

	for lineIndex, productID := range sale.DetailProductsIDs {
		product, found := productByID[productID]
		if !found {
			return core.Err(fmt.Sprintf("El producto %v de la venta no existe.", productID))
		}

		lineQuantity := core.UnpackQuantityLine(sale.DetailQuantities[lineIndex])
		productDivisor := product.SbuQuantity
		if productDivisor <= 0 {
			productDivisor = core.QuantityDivisorNone
		}

		if lineQuantity.Sub > 0 {
			if productDivisor <= core.QuantityDivisorNone {
				return core.Err(fmt.Sprintf(
					`El producto "%s" no tiene sub-unidad configurada, pero la venta envía %v sub-unidades.`,
					product.Name, lineQuantity.Sub))
			}
			if product.SbuFinalPrice <= 0 {
				return core.Err(fmt.Sprintf(
					`El producto "%s" no tiene precio de sub-unidad; no se puede vender fraccionado.`,
					product.Name))
			}
			// Packing normalizes, so a Sub at or past the divisor means the client packed a
			// value the catalog cannot represent.
			if lineQuantity.Sub >= int32(productDivisor) {
				return core.Err(fmt.Sprintf(
					`El producto "%s" admite hasta %v sub-unidades por unidad; la venta envía %v.`,
					product.Name, productDivisor-1, lineQuantity.Sub))
			}
		}

		// Prices come from the catalog, never from the request.
		sale.DetailPrices[lineIndex] = product.FinalPrice
		sale.DetailSubPrices[lineIndex] = product.SbuFinalPrice
		sale.DetailSubDivisor[lineIndex] = productDivisor
	}

	// Recompute the order total from the resolved lines for the same reason.
	totalAmount := int32(0)
	for lineIndex := range sale.DetailProductsIDs {
		totalAmount += core.QuantityAmount(
			core.UnpackQuantityLine(sale.DetailQuantities[lineIndex]),
			sale.DetailPrices[lineIndex], sale.DetailSubPrices[lineIndex])
	}
	sale.TotalAmount = totalAmount
	if sale.DebtAmount > totalAmount {
		return core.Err("El monto adeudado no puede superar el total de la venta.")
	}
	return nil
}

func resolveSaleOrderClientID(clientInfo *types.SaleOrderClientInfo, companyID int32, userID int32) (int32, error) {
	clientName := strings.TrimSpace(clientInfo.Name)
	clientRegistryNumber := strings.TrimSpace(clientInfo.RegistryNumber)
	if clientName == "" {
		return 0, core.Err("ClientInfo.Name es obligatorio.")
	}

	clientPersonType := crm.PersonTypeNatural
	if clientRegistryNumber != "" {
		// Preserve the provided registry number so identity matching can reuse existing client rows.
		clientPersonType = crm.PersonTypeCompany
	}

	clientProviders := []crm.ClientProvider{{
		Type:           crm.ClientProviderTypeClient,
		Name:           clientName,
		RegistryNumber: clientRegistryNumber,
		PersonType:     clientPersonType,
	}}
	// Sale-order client creation must never update an existing client record from frontend input
	// to prevent accidental data corruption of shared client/provider records.
	saveError := crm.SaveClientProviders(&clientProviders, companyID, userID, true)
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
	requestedByKey := map[lineKey]core.Quantity{}
	divisorByKey := map[lineKey]int16{}
	for index, productID := range sale.DetailProductsIDs {
		packedQuantity := core.GetIndex(sale.DetailQuantities, index)
		if packedQuantity == 0 {
			continue
		}
		quantity := core.UnpackQuantityLine(packedQuantity)
		lineDivisor := core.GetIndex(sale.DetailSubDivisor, index)
		if lineDivisor <= 0 {
			lineDivisor = core.QuantityDivisorNone
		}
		presentationID := core.GetIndex(sale.DetailProductPresentations, index)
		key := lineKey{
			stockID:      logistics.PackProductStockID(sale.WarehouseID, productID, presentationID),
			lotID:        core.GetIndex(sale.DetailProductLotIDs, index),
			serialNumber: core.GetIndex(sale.DetailProductSkus, index),
		}
		requestedByKey[key] = requestedByKey[key].Add(quantity)
		divisorByKey[key] = lineDivisor
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

	stockByID := map[int64]logistics.ProductStock{}
	detailsByKey := map[lineKey]logistics.ProductStockDetail{}

	eg := errgroup.Group{}
	eg.Go(func() error {
		stocks := []logistics.ProductStock{}
		q := db.Query(&stocks)
		q.Select(q.ID, q.Quantity, q.SubQuantity, q.DetailQuantity, q.DetailSubQuantity, q.SubDivisor).
			CompanyID.Equals(req.User.CompanyID).
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
			details := []logistics.ProductStockDetail{}
			q := db.Query(&details)
			q.Select().
				CompanyID.Equals(req.User.CompanyID).
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
		stock := stockByID[key.stockID]
		// The stock row and the line can sit at different divisors while a refinement is
		// still rolling out, so compare both at the finer of the two.
		comparisonDivisor := divisorByKey[key]
		if stockDivisor := stock.SubDivisor; stockDivisor > 0 && stockDivisor != comparisonDivisor {
			if core.IsQuantityDivisorRefinement(comparisonDivisor, stockDivisor) {
				converted, convErr := requested.ConvertToDivisor(comparisonDivisor, stockDivisor)
				if convErr != nil {
					return convErr
				}
				requested, comparisonDivisor = converted, stockDivisor
			}
		}

		var available core.Quantity
		if key.lotID == 0 && key.serialNumber == "" {
			available = core.Quantity{Units: stock.Quantity, Sub: stock.SubQuantity}
		} else {
			detail := detailsByKey[key]
			available = core.Quantity{Units: detail.Quantity, Sub: detail.SubQuantity}
		}
		if stock.SubDivisor > 0 && stock.SubDivisor != comparisonDivisor {
			converted, convErr := available.ConvertToDivisor(stock.SubDivisor, comparisonDivisor)
			if convErr != nil {
				return convErr
			}
			available = converted
		}

		if core.CompareQuantities(available, requested, comparisonDivisor) < 0 {
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
				fmt.Sprintf("Se necesita %v+%v/%v. Se posee en stock: %v+%v/%v",
					requested.Units, requested.Sub, comparisonDivisor,
					available.Units, available.Sub, comparisonDivisor))
		}
	}
	return nil
}

func GetSaleOrderByIDs(req *core.HandlerArgs) core.HandlerResponse {
	saleOrderIDRecords := req.ExtractUpdatedVersionValues()

	saleOrderIDs := core.Map(saleOrderIDRecords, func(e db.IDUpdatedVersion) int64 { return e.ID })

	saleOrders := []types.SaleOrder{}
	query := db.Query(&saleOrders).CompanyID.Equals(req.User.CompanyID)

	if err := query.ID.In(saleOrderIDs...).Exec(); err != nil {
		return req.MakeErr("Error al obtener los productos.", err)
	}

	return core.MakeResponse(req, &saleOrders)
}
