package logistica

import (
	"app/core"
	"app/db"
	"app/finanzas"
	finanzasTypes "app/finanzas/types"
	logisticaTypes "app/logistica/types"
	"encoding/json"
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
	if len(record.DetailProductIDs) == 0 {
		return req.MakeErr("Debe agregar al menos un producto.")
	}
	n := len(record.DetailProductIDs)
	if len(record.DetailQuantities) != n || len(record.DetailPrices) != n {
		return req.MakeErr("Los detalles de la orden son inconsistentes.")
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
