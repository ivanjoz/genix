package logistics

import (
	"app/core"
	"app/db"
	finance "app/finance/types"
	"app/logistics/types"
	"encoding/json"
	"slices"
)

const (
	PurchaseOrderActionConfirm = 1 // Cambia status de Pendiente (1) a Cumplido (2)
	PurchaseOrderActionEdit    = 2 // Edita campos no críticos cuando la orden está Pendiente o Cumplida
	PurchaseOrderActionPay     = 3 // Registra un pago: descuenta DebtAmount y crea movimiento de cashBank Tipo=6
	PurchaseOrderActionAnnul   = 4 // Anula la orden: cambia status a Cancelada (0). Solo desde Pendiente o Confirmada.
)

// Tipo del movimiento de cashBank para pagos a proveedor (Pago Proveedor).

// Body esperado para PurchaseOrderActionPay.
type purchaseOrderPayPayload struct {
	CashBankID int32
	Amount     int32 // Amount to pay in cents (positive). Sent as negative to the cashBank.
}

// Body esperado para PostPurchaseOrderEntry.
// Items reutiliza PostStockAdjustItem por simetría con POST.productos-stock; aquí
// la cantidad es siempre un ingreso (suma a stock), no un reemplazo.
type purchaseOrderEntryPayload struct {
	PurchaseOrderID int32
	WarehouseID     int32
	Items           []PostStockAdjustItem
}

// PostPurchaseOrderEntry recibe la mercadería de una orden de compra Confirmada:
//  1. compara los productos recibidos vs. los pedidos en la OC y calcula
//     DifferenceValue (Σ (recibido - pedido) * precio),
//     ambos firmados (negativo = subentrega, positivo = sobreentrega);
//  2. llama a ApplyMovimientos para insertar el stock en el almacén;
//  3. actualiza la OC: Status=Fulfilled + diferencias calculadas.
//
// La diferencia se registra pero NO se rechaza: la OC se cumple aunque haya
// mismatches, y los valores quedan asentados para reportería.
func PostPurchaseOrderEntry(req *core.HandlerArgs) core.HandlerResponse {
	payload := purchaseOrderEntryPayload{}
	if err := json.Unmarshal([]byte(*req.Body), &payload); err != nil {
		return req.MakeErr("Error al deserializar el body.", err)
	}
	if payload.PurchaseOrderID <= 0 {
		return req.MakeErr("Debe especificar PurchaseOrderID.")
	}
	if payload.WarehouseID <= 0 {
		return req.MakeErr("Debe especificar el Almacén destino.")
	}
	if len(payload.Items) == 0 {
		return req.MakeErr("No se enviaron registros para ingresar.")
	}

	// Obtener la OC para validar estado y mapear precios pedidos.
	existing := []types.PurchaseOrder{}
	if err := db.Query(&existing).
		CompanyID.Equals(req.User.CompanyID).
		ID.Equals(payload.PurchaseOrderID).Limit(1).Exec(); err != nil {
		return req.MakeErr("Error al obtener la orden de compra.", err)
	}
	if len(existing) == 0 {
		return req.MakeErr("Orden de compra no encontrada.")
	}
	order := existing[0]
	if order.Status != types.PurchaseOrderStatusConfirmed {
		return req.MakeErr("La orden no está en estado Confirmada y no puede recibirse.")
	}

	// Indexar el detalle pedido por (ProductID, PresentationID). El ORM almacena la
	// presentación como int32 pero MovimientoInterno la maneja como int16; usamos int32
	// para la clave a fin de no perder información si una OC contiene presentaciones > int16.
	type orderKey struct {
		ProductID      int32
		PresentationID int32
	}

	// Un único stats por clave: ordered/received se suman, price se conserva en su primera
	// aparición (la diferencia se pondera por el delta total, no línea a línea).
	type orderStats struct {
		ordered  int32
		received int32
		price    int32
	}

	statsByKey := map[orderKey]*orderStats{}

	getStats := func(key orderKey) *orderStats {
		if statsByKey[key] == nil {
			statsByKey[key] = &orderStats{}
		}
		return statsByKey[key]
	}

	for i, productID := range order.DetailProductIDs {
		key := orderKey{ProductID: productID}
		if i < len(order.DetailProductPresentationIDs) {
			key.PresentationID = order.DetailProductPresentationIDs[i]
		}
		s := getStats(key)
		s.ordered += core.GetIndex(order.DetailProductQuantity, i)
		if s.price == 0 {
			s.price = core.GetIndex(order.DetailProductPrice, i)
		}
	}

	for _, item := range payload.Items {
		if item.ProductID == 0 {
			return req.MakeErr("Hay un item sin ProductID.")
		}
		if item.Quantity <= 0 {
			return req.MakeErr("Hay un item con Cantidad inválida (debe ser > 0).")
		}
		getStats(orderKey{ProductID: item.ProductID, PresentationID: int32(item.PresentationID)}).received += item.Quantity
	}

	// Diferencia firmada en dinero: positiva = sobreentrega, negativa = subentrega.
	// price=0 cubre productos recibidos que no están en la OC (no suman a value).
	var diffValue int32
	for _, s := range statsByKey {
		diff := s.received - s.ordered
		if diff == 0 {
			continue
		}
		diffValue += diff * s.price
	}

	// Construir movimientos: ReemplazarCantidad=false (suma a stock), DocumentID enlaza
	// el ledger con la OC, SupplierID se completa con el ProviderID de la OC para
	// que la resolución de lotes use el hash (date, proveedor, nombre).
	movimientos := make([]types.InternalMovement, 0, len(payload.Items))
	for _, item := range payload.Items {
		key := orderKey{ProductID: item.ProductID, PresentationID: int32(item.PresentationID)}
		movimientos = append(movimientos, types.InternalMovement{
			DocumentID:     int64(order.ID),
			WarehouseID:    payload.WarehouseID,
			ProductID:      item.ProductID,
			PresentationID: item.PresentationID,
			SerialNumber:   item.SerialNumber,
			LotID:          item.LotID,
			LotName:        item.LotCode,
			SupplierID:     order.ProviderID,
			Quantity:       item.Quantity,
			SubQuantity:    item.SubQuantity,
			Price:          getStats(key).price,
		})
	}
	if err := types.ApplyMovimientos(req, movimientos); err != nil {
		return req.MakeErr(err)
	}

	// Cumplir la OC: Status=Fulfilled + diferencias calculadas.
	now := core.SUnixTime()
	order.Status = types.PurchaseOrderStatusFulfilled
	order.DifferenceValue = diffValue
	order.Updated = now
	order.UpdatedBy = req.User.ID

	q := db.TableOf[types.PurchaseOrder]()
	if err := db.Update(&[]types.PurchaseOrder{order},
		q.Status, q.DifferenceValue, q.Updated, q.UpdatedBy,
	); err != nil {
		return req.MakeErr("Error al actualizar la orden de compra.", err)
	}
	return req.MakeResponse(order)
}

func GetPurchaseOrders(req *core.HandlerArgs) core.HandlerResponse {
	// Delta syncs are watermarked by "upv", the write sequence number, not by a timestamp: two
	// writes in the same second are distinguishable, so nothing is re-sent and nothing is skipped.
	updatedSince := req.GetQueryInt("upv")
	statusParam := int8(req.GetQueryInt("status"))

	if statusParam == 0 {
		statusParam = types.PurchaseOrderStatusPending
	}

	// Delta() reproduces exactly what this handler used to fan out by hand: the requested status only
	// on a first sync, every declared status afterwards so the client can evict rows that moved.
	records := []types.PurchaseOrder{}
	query := db.Query(&records)
	query.CompanyID.Equals(req.User.CompanyID).Delta(updatedSince, int64(statusParam))

	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener órdenes de compra:", err)
	}

	return req.MakeResponse(records)
}

func PostPurchaseOrder(req *core.HandlerArgs) core.HandlerResponse {
	record := types.PurchaseOrder{}
	if err := json.Unmarshal([]byte(*req.Body), &record); err != nil {
		return req.MakeErr("Error al deserializar el body.", err)
	}

	if record.ProviderID <= 0 {
		return req.MakeErr("Debe seleccionar un proveedor.")
	}

	// Supplies are Product rows with Status=2, so they arrive as ordinary product lines.
	productLineCount := len(record.DetailProductIDs)
	if productLineCount == 0 {
		return req.MakeErr("Debe agregar al menos un producto o insumo.")
	}
	if len(record.DetailProductQuantity) != productLineCount || len(record.DetailProductPrice) != productLineCount {
		return req.MakeErr("Los detalles de productos de la orden son inconsistentes.")
	}
	if len(record.DetailProductPresentationIDs) > 0 && len(record.DetailProductPresentationIDs) != productLineCount {
		return req.MakeErr("Inconsistencia en el detalle de Presentaciones IDs.")
	}
	if slices.Contains(record.DetailProductIDs, 0) {
		return req.MakeErr("Hay una línea con ID = 0")
	}
	if slices.Contains(record.DetailProductQuantity, 0) {
		return req.MakeErr("Hay una línea con cantidad = 0")
	}

	now := core.SUnixTime()
	todayFecha := core.FechaUnix()
	currentSemana := core.MakeSemanaFromFechaUnix(todayFecha, false)

	record.CompanyID = req.User.CompanyID
	record.Status = types.PurchaseOrderStatusPending
	record.Updated = now
	record.UpdatedBy = req.User.ID
	if record.ID == 0 {
		record.Created = now
		record.CreatedBy = req.User.ID
		record.Date = todayFecha
		record.Week = currentSemana.Code
		// La deuda inicial corresponde al monto total: cada Pago la reduce hasta llegar a 0.
		record.DebtAmount = record.TotalAmount
	}

	records := []types.PurchaseOrder{record}
	if err := db.Merge(&records, nil,
		func(prev, curr *types.PurchaseOrder) bool {
			curr.CompanyID = req.User.CompanyID
			curr.Created = prev.Created
			curr.CreatedBy = prev.CreatedBy
			curr.Date = prev.Date
			curr.Week = prev.Week
			curr.Status = types.PurchaseOrderStatusPending
			curr.Updated = now
			curr.UpdatedBy = req.User.ID
			return true
		},
		func(curr *types.PurchaseOrder) {
			curr.CompanyID = req.User.CompanyID
			curr.Status = types.PurchaseOrderStatusPending
			curr.Updated = now
			curr.UpdatedBy = req.User.ID
			curr.Date = todayFecha
			curr.Week = currentSemana.Code
		},
	); err != nil {
		return req.MakeErr("Error al guardar la orden de compra.", err)
	}

	return req.MakeResponse(records[0])
}

func PutPurchaseOrder(req *core.HandlerArgs) core.HandlerResponse {
	action := req.GetQueryInt("action")
	if action == 0 {
		return req.MakeErr("Debe especificar el parámetro 'action'.")
	}

	orderID := req.GetQueryInt("id")
	if orderID == 0 {
		return req.MakeErr("Debe especificar el parámetro 'id'.")
	}

	// Obtener la orden actual para validar su estado antes de modificarla
	existing := []types.PurchaseOrder{}
	if err := db.Query(&existing).
		CompanyID.Equals(req.User.CompanyID).
		ID.Equals(orderID).Limit(1).Exec(); err != nil {
		return req.MakeErr("Error al obtener la orden de compra.", err)
	}
	if len(existing) == 0 {
		return req.MakeErr("Orden de compra no encontrada.")
	}

	orderCurrent := existing[0]
	now := core.SUnixTime()
	q := db.TableOf[types.PurchaseOrder]()

	switch action {
	case PurchaseOrderActionConfirm:
		// Solo se puede confirmar si está en estado Pendiente (1)
		if orderCurrent.Status != types.PurchaseOrderStatusPending {
			return req.MakeErr("La orden no está en estado Pendiente y no puede confirmarse.")
		}
		orderCurrent.Status = types.PurchaseOrderStatusConfirmed
		orderCurrent.Updated = now
		orderCurrent.UpdatedBy = req.User.ID

		if err := db.Update(&[]types.PurchaseOrder{orderCurrent}, q.Status, q.Updated, q.UpdatedBy); err != nil {
			return req.MakeErr("Error al actualizar la orden de compra.", err)
		}
		return req.MakeResponse(orderCurrent)

	case PurchaseOrderActionEdit:
		// Solo se permite editar mientras la orden esté Pendiente (1) o Confirmada (2);
		// los demás estados (Cancelada, Cumplida) son inmutables.
		if orderCurrent.Status != types.PurchaseOrderStatusPending &&
			orderCurrent.Status != types.PurchaseOrderStatusConfirmed {
			return req.MakeErr("La orden no se puede editar en su estado actual.")
		}

		// Decodifica únicamente los campos editables; ProviderID, Status, productos y totales
		// se preservan desde el registro existente para evitar modificaciones no autorizadas.
		patch := types.PurchaseOrder{}
		if err := json.Unmarshal([]byte(*req.Body), &patch); err != nil {
			return req.MakeErr("Error al deserializar el body.", err)
		}

		orderCurrent.WarehouseID = patch.WarehouseID
		orderCurrent.DeliveryDate = patch.DeliveryDate
		orderCurrent.PaymentDate = patch.PaymentDate
		orderCurrent.InvoiceNumber = patch.InvoiceNumber
		orderCurrent.Notes = patch.Notes
		orderCurrent.Updated = now
		orderCurrent.UpdatedBy = req.User.ID

		if err := db.Update(&[]types.PurchaseOrder{orderCurrent},
			q.WarehouseID, q.DeliveryDate, q.PaymentDate, q.InvoiceNumber, q.Notes, q.Updated, q.UpdatedBy, q.Status,
		); err != nil {
			return req.MakeErr("Error al actualizar la orden de compra.", err)
		}
		return req.MakeResponse(orderCurrent)

	case PurchaseOrderActionPay:
		// Solo se registra pago cuando la orden está Confirmada: Pendiente debe pasar primero
		// por Confirmar (acción 1) y los demás estados son inmutables.
		if orderCurrent.Status != types.PurchaseOrderStatusConfirmed {
			return req.MakeErr("La orden no está en estado Confirmada y no puede pagarse.")
		}

		payload := purchaseOrderPayPayload{}
		if err := json.Unmarshal([]byte(*req.Body), &payload); err != nil {
			return req.MakeErr("Error al deserializar el body.", err)
		}
		if payload.CashBankID <= 0 {
			return req.MakeErr("Debe seleccionar una cashBank para registrar el pago.")
		}
		if payload.Amount <= 0 {
			return req.MakeErr("El monto del pago debe ser mayor a 0.")
		}
		if payload.Amount > orderCurrent.DebtAmount {
			return req.MakeErr("El monto del pago excede la deuda pendiente de la orden.")
		}

		// El pago sale de la cashBank: Amount negativo para que ApplyCajaMovimientos descuente del saldo.
		movimiento := finance.InternalCashMovement{
			CashBankID: payload.CashBankID,
			DocumentID: int64(orderCurrent.ID),
			Type:       finance.CashMovementTypeSupplierPayment,
			Amount:     -payload.Amount,
		}
		if err := finance.ApplyCashBankMovement(req, []finance.InternalCashMovement{movimiento}); err != nil {
			return req.MakeErr("Error al registrar el movimiento de cashBank:", err)
		}

		orderCurrent.DebtAmount -= payload.Amount
		orderCurrent.Updated = now
		orderCurrent.UpdatedBy = req.User.ID

		if err := db.Update(&[]types.PurchaseOrder{orderCurrent}, q.Status, q.DebtAmount, q.Updated, q.UpdatedBy); err != nil {
			return req.MakeErr("Error al actualizar la deuda de la orden de compra.", err)
		}
		return req.MakeResponse(orderCurrent)

	case PurchaseOrderActionAnnul:
		// Solo se permite anular órdenes en estado Pendiente o Confirmada; las ya Canceladas
		// o Cumplidas son inmutables para preservar consistencia contable.
		if orderCurrent.Status != types.PurchaseOrderStatusPending &&
			orderCurrent.Status != types.PurchaseOrderStatusConfirmed {
			return req.MakeErr("La orden no se puede anular en su estado actual.")
		}
		orderCurrent.Status = types.PurchaseOrderStatusCanceled
		orderCurrent.Updated = now
		orderCurrent.UpdatedBy = req.User.ID

		if err := db.Update(&[]types.PurchaseOrder{orderCurrent}, q.Status, q.Updated, q.UpdatedBy); err != nil {
			return req.MakeErr("Error al anular la orden de compra.", err)
		}
		return req.MakeResponse(orderCurrent)

	default:
		return req.MakeErr("Acción no válida.")
	}
}
