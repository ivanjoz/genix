package logistica

import (
	"app/core"
	"app/db"
	"app/finanzas"
	finanzasTypes "app/finanzas/types"
	logisticaTypes "app/logistica/types"
	"encoding/json"
	"slices"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	PurchaseOrderActionConfirm = 1 // Cambia status de Pendiente (1) a Cumplido (2)
	PurchaseOrderActionEdit    = 2 // Edita campos no críticos cuando la orden está Pendiente o Cumplida
	PurchaseOrderActionPay     = 3 // Registra un pago: descuenta DebtAmount y crea movimiento de caja Tipo=6
	PurchaseOrderActionAnnul   = 4 // Anula la orden: cambia status a Cancelada (0). Solo desde Pendiente o Confirmada.
)

// Tipo del movimiento de caja para pagos a proveedor (Pago Proveedor).
const cajaMovimientoTipoPagoProveedor int8 = 6

// Body esperado para PurchaseOrderActionPay.
type purchaseOrderPayPayload struct {
	CajaID int32
	Monto  int32 // Monto a pagar en céntimos (positivo). Se enviará como negativo a la caja.
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
//     DifferenceQuantity (Σ recibido - pedido) y DifferenceValue (Σ (recibido - pedido) * precio),
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
	existing := []logisticaTypes.PurchaseOrder{}
	if err := db.Query(&existing).
		CompanyID.Equals(req.Usuario.EmpresaID).
		ID.Equals(payload.PurchaseOrderID).Limit(1).Exec(); err != nil {
		return req.MakeErr("Error al obtener la orden de compra.", err)
	}
	if len(existing) == 0 {
		return req.MakeErr("Orden de compra no encontrada.")
	}
	order := existing[0]
	if order.Status != logisticaTypes.PurchaseOrderStatusConfirmed {
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

	// Diferencia firmada: positiva = sobreentrega, negativa = subentrega, cero = exacto.
	// price=0 cubre productos recibidos que no están en la OC (suman a quantity, no a value).
	var diffQuantity, diffValue int32
	for _, s := range statsByKey {
		diff := s.received - s.ordered
		if diff == 0 {
			continue
		}
		diffQuantity += diff
		diffValue += diff * s.price
	}

	// Construir movimientos: ReemplazarCantidad=false (suma a stock), DocumentID enlaza
	// el ledger con la OC, SupplierID se completa con el ProviderID de la OC para
	// que la resolución de lotes use el hash (fecha, proveedor, nombre).
	movimientos := make([]logisticaTypes.MovimientoInterno, 0, len(payload.Items))
	for _, item := range payload.Items {
		key := orderKey{ProductID: item.ProductID, PresentationID: int32(item.PresentationID)}
		movimientos = append(movimientos, logisticaTypes.MovimientoInterno{
			DocumentID:     int64(order.ID),
			WarehouseID:    payload.WarehouseID,
			ProductoID:     item.ProductID,
			PresentacionID: item.PresentationID,
			SerialNumber:   item.SerialNumber,
			LotID:          item.LotID,
			LotName:        item.LotCode,
			SupplierID:     order.ProviderID,
			Cantidad:       item.Quantity,
			SubCantidad:    item.SubQuantity,
			Price:          getStats(key).price,
		})
	}
	if err := ApplyMovimientos(req, movimientos); err != nil {
		return req.MakeErr(err)
	}

	// Cumplir la OC: Status=Fulfilled + diferencias calculadas.
	now := core.SUnixTime()
	order.Status = logisticaTypes.PurchaseOrderStatusFulfilled
	order.DifferenceQuantity = diffQuantity
	order.DifferenceValue = diffValue
	order.Updated = now
	order.UpdatedBy = req.Usuario.ID

	q := db.Table[logisticaTypes.PurchaseOrder]()
	if err := db.Update(&[]logisticaTypes.PurchaseOrder{order},
		q.Status, q.DifferenceQuantity, q.DifferenceValue, q.Updated, q.UpdatedBy,
	); err != nil {
		return req.MakeErr("Error al actualizar la orden de compra.", err)
	}
	return req.MakeResponse(order)
}

func GetPurchaseOrders(req *core.HandlerArgs) core.HandlerResponse {
	updated := req.GetQueryInt("updated")
	statusParam := int8(req.GetQueryInt("status"))

	if statusParam == 0 {
		statusParam = logisticaTypes.PurchaseOrderStatusPending
	}

	status := []int8{statusParam}
	if updated > 0 {
		status = []int8{0, 1, 2, 4}
	}

	statusRecords := make([][]logisticaTypes.PurchaseOrder, len(status))
	group := errgroup.Group{}

	for index, currentStatus := range status {

		group.Go(func() error {
			recordsForStatus := []logisticaTypes.PurchaseOrder{}
			query := db.Query(&recordsForStatus)
			query.CompanyID.Equals(req.Usuario.EmpresaID).
				Status.Equals(currentStatus).
				Updated.GreaterThan(updated)
			if err := query.Exec(); err != nil {
				return err
			}

			// Preserve grouped query results and merge them after all workers finish.
			statusRecords[index] = recordsForStatus
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return req.MakeErr("Error al obtener órdenes de compra:", err)
	}

	totalRecords := 0
	for _, recordsForStatus := range statusRecords {
		totalRecords += len(recordsForStatus)
	}

	records := make([]logisticaTypes.PurchaseOrder, 0, totalRecords)
	for _, recordsForStatus := range statusRecords {
		records = append(records, recordsForStatus...)
	}

	return req.MakeResponse(records)
}

func PostPurchaseOrder(req *core.HandlerArgs) core.HandlerResponse {
	record := logisticaTypes.PurchaseOrder{}
	if err := json.Unmarshal([]byte(*req.Body), &record); err != nil {
		return req.MakeErr("Error al deserializar el body.", err)
	}

	if record.ProviderID <= 0 {
		return req.MakeErr("Debe seleccionar un proveedor.")
	}

	productLineCount := len(record.DetailProductIDs)
	supplyLineCount := len(record.DetailSupplyIDs)

	// Productos e insumos son listas independientes; la orden necesita al menos una línea entre ambas.
	if productLineCount == 0 && supplyLineCount == 0 {
		return req.MakeErr("Debe agregar al menos un producto o insumo.")
	}

	// Validar la lista de productos (si tiene contenido).
	if productLineCount > 0 {
		if len(record.DetailProductQuantity) != productLineCount || len(record.DetailProductPrice) != productLineCount {
			return req.MakeErr("Los detalles de productos de la orden son inconsistentes.")
		}
		if len(record.DetailProductPresentationIDs) > 0 && len(record.DetailProductPresentationIDs) != productLineCount {
			return req.MakeErr("Inconsistencia en el detalle de Presentaciones IDs.")
		}
		if slices.Contains(record.DetailProductIDs, 0) {
			return req.MakeErr("Hay un producto con ID = 0")
		}
		if slices.Contains(record.DetailProductQuantity, 0) {
			return req.MakeErr("Hay un producto con cantidad = 0")
		}
	}

	// Validar la lista de insumos (si tiene contenido).
	if supplyLineCount > 0 {
		if len(record.DetailSupplyQuantity) != supplyLineCount || len(record.DetailSupplyPrice) != supplyLineCount {
			return req.MakeErr("Los detalles de insumos de la orden son inconsistentes.")
		}
		if slices.Contains(record.DetailSupplyIDs, 0) {
			return req.MakeErr("Hay un insumo con ID = 0")
		}
		if slices.Contains(record.DetailSupplyQuantity, 0) {
			return req.MakeErr("Hay un insumo con cantidad = 0")
		}
	}

	now := core.SUnixTime()
	todayFecha := core.TimeToFechaUnix(time.Now())
	currentSemana := core.MakeSemanaFromFechaUnix(todayFecha, false)

	record.CompanyID = req.Usuario.EmpresaID
	record.Status = logisticaTypes.PurchaseOrderStatusPending
	record.Updated = now
	record.UpdatedBy = req.Usuario.ID
	if record.ID == 0 {
		record.Created = now
		record.CreatedBy = req.Usuario.ID
		record.Date = todayFecha
		record.Week = currentSemana.Code
		// La deuda inicial corresponde al monto total: cada Pago la reduce hasta llegar a 0.
		record.DebtAmount = record.TotalAmount
	}

	records := []logisticaTypes.PurchaseOrder{record}
	if err := db.Merge(&records, nil,
		func(prev, curr *logisticaTypes.PurchaseOrder) bool {
			curr.CompanyID = req.Usuario.EmpresaID
			curr.Created = prev.Created
			curr.CreatedBy = prev.CreatedBy
			curr.Date = prev.Date
			curr.Week = prev.Week
			curr.Status = logisticaTypes.PurchaseOrderStatusPending
			curr.Updated = now
			curr.UpdatedBy = req.Usuario.ID
			return true
		},
		func(curr *logisticaTypes.PurchaseOrder) {
			curr.CompanyID = req.Usuario.EmpresaID
			curr.Status = logisticaTypes.PurchaseOrderStatusPending
			curr.Updated = now
			curr.UpdatedBy = req.Usuario.ID
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
	existing := []logisticaTypes.PurchaseOrder{}
	if err := db.Query(&existing).
		CompanyID.Equals(req.Usuario.EmpresaID).
		ID.Equals(orderID).Limit(1).Exec(); err != nil {
		return req.MakeErr("Error al obtener la orden de compra.", err)
	}
	if len(existing) == 0 {
		return req.MakeErr("Orden de compra no encontrada.")
	}

	orderCurrent := existing[0]
	now := core.SUnixTime()
	q := db.Table[logisticaTypes.PurchaseOrder]()

	switch action {
	case PurchaseOrderActionConfirm:
		// Solo se puede confirmar si está en estado Pendiente (1)
		if orderCurrent.Status != logisticaTypes.PurchaseOrderStatusPending {
			return req.MakeErr("La orden no está en estado Pendiente y no puede confirmarse.")
		}
		orderCurrent.Status = logisticaTypes.PurchaseOrderStatusConfirmed
		orderCurrent.Updated = now
		orderCurrent.UpdatedBy = req.Usuario.ID

		if err := db.Update(&[]logisticaTypes.PurchaseOrder{orderCurrent}, q.Status, q.Updated, q.UpdatedBy); err != nil {
			return req.MakeErr("Error al actualizar la orden de compra.", err)
		}
		return req.MakeResponse(orderCurrent)

	case PurchaseOrderActionEdit:
		// Solo se permite editar mientras la orden esté Pendiente (1) o Confirmada (2);
		// los demás estados (Cancelada, Cumplida) son inmutables.
		if orderCurrent.Status != logisticaTypes.PurchaseOrderStatusPending &&
			orderCurrent.Status != logisticaTypes.PurchaseOrderStatusConfirmed {
			return req.MakeErr("La orden no se puede editar en su estado actual.")
		}

		// Decodifica únicamente los campos editables; ProviderID, Status, productos y totales
		// se preservan desde el registro existente para evitar modificaciones no autorizadas.
		patch := logisticaTypes.PurchaseOrder{}
		if err := json.Unmarshal([]byte(*req.Body), &patch); err != nil {
			return req.MakeErr("Error al deserializar el body.", err)
		}

		orderCurrent.WarehouseID = patch.WarehouseID
		orderCurrent.DeliveryDate = patch.DeliveryDate
		orderCurrent.PaymentDate = patch.PaymentDate
		orderCurrent.InvoiceNumber = patch.InvoiceNumber
		orderCurrent.Notes = patch.Notes
		orderCurrent.Updated = now
		orderCurrent.UpdatedBy = req.Usuario.ID

		if err := db.Update(&[]logisticaTypes.PurchaseOrder{orderCurrent},
			q.WarehouseID, q.DeliveryDate, q.PaymentDate, q.InvoiceNumber, q.Notes, q.Updated, q.UpdatedBy, q.Status,
		); err != nil {
			return req.MakeErr("Error al actualizar la orden de compra.", err)
		}
		return req.MakeResponse(orderCurrent)

	case PurchaseOrderActionPay:
		// Solo se registra pago cuando la orden está Confirmada: Pendiente debe pasar primero
		// por Confirmar (acción 1) y los demás estados son inmutables.
		if orderCurrent.Status != logisticaTypes.PurchaseOrderStatusConfirmed {
			return req.MakeErr("La orden no está en estado Confirmada y no puede pagarse.")
		}

		payload := purchaseOrderPayPayload{}
		if err := json.Unmarshal([]byte(*req.Body), &payload); err != nil {
			return req.MakeErr("Error al deserializar el body.", err)
		}
		if payload.CajaID <= 0 {
			return req.MakeErr("Debe seleccionar una caja para registrar el pago.")
		}
		if payload.Monto <= 0 {
			return req.MakeErr("El monto del pago debe ser mayor a 0.")
		}
		if payload.Monto > orderCurrent.DebtAmount {
			return req.MakeErr("El monto del pago excede la deuda pendiente de la orden.")
		}

		// El pago sale de la caja: Monto negativo para que ApplyCajaMovimientos descuente del saldo.
		movimiento := finanzasTypes.CajaMovimientoInterno{
			CajaID:     payload.CajaID,
			DocumentID: int64(orderCurrent.ID),
			Tipo:       cajaMovimientoTipoPagoProveedor,
			Monto:      -payload.Monto,
		}
		if err := finanzas.ApplyCajaMovimientos(req, []finanzasTypes.CajaMovimientoInterno{movimiento}); err != nil {
			return req.MakeErr("Error al registrar el movimiento de caja:", err)
		}

		orderCurrent.DebtAmount -= payload.Monto
		orderCurrent.Updated = now
		orderCurrent.UpdatedBy = req.Usuario.ID

		if err := db.Update(&[]logisticaTypes.PurchaseOrder{orderCurrent}, q.Status, q.DebtAmount, q.Updated, q.UpdatedBy); err != nil {
			return req.MakeErr("Error al actualizar la deuda de la orden de compra.", err)
		}
		return req.MakeResponse(orderCurrent)

	case PurchaseOrderActionAnnul:
		// Solo se permite anular órdenes en estado Pendiente o Confirmada; las ya Canceladas
		// o Cumplidas son inmutables para preservar consistencia contable.
		if orderCurrent.Status != logisticaTypes.PurchaseOrderStatusPending &&
			orderCurrent.Status != logisticaTypes.PurchaseOrderStatusConfirmed {
			return req.MakeErr("La orden no se puede anular en su estado actual.")
		}
		orderCurrent.Status = logisticaTypes.PurchaseOrderStatusCanceled
		orderCurrent.Updated = now
		orderCurrent.UpdatedBy = req.Usuario.ID

		if err := db.Update(&[]logisticaTypes.PurchaseOrder{orderCurrent}, q.Status, q.Updated, q.UpdatedBy); err != nil {
			return req.MakeErr("Error al anular la orden de compra.", err)
		}
		return req.MakeResponse(orderCurrent)

	default:
		return req.MakeErr("Acción no válida.")
	}
}
