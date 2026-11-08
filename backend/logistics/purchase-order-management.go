package logistics

import (
	accounting "app/accounting/types"
	"app/core"
	crm "app/crm/types"
	"app/db"
	finance "app/finance/types"
	"app/logistics/types"
	production "app/production/types"
	"encoding/json"
	"fmt"
)

const (
	PurchaseOrderActionConfirm = 1 // Cambia status de Pendiente (1) a Cumplido (2)
	PurchaseOrderActionEdit    = 2 // Edita campos no críticos cuando la orden está Pendiente o Cumplida
	PurchaseOrderActionPay     = 3 // Registra un pago (Confirmada o Cumplida): descuenta DebtAmount y crea movimiento de cashBank Tipo=6
	PurchaseOrderActionAnnul   = 4 // Anula la orden: cambia status a Cancelada (0). Solo desde Pendiente o Confirmada.
	// Registra o corrige el comprobante del proveedor. También en Cumplida: la factura suele
	// llegar con la mercadería o después, y sin ella la compra no entra al Registro de Compras.
	PurchaseOrderActionDocument = 5
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

// PostPurchaseOrderEntry recibe la mercadería de una orden de compra Confirmada
// (ver receivePurchaseOrder) y la persiste como Cumplida.
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

	for _, item := range payload.Items {
		if item.ProductID == 0 {
			return req.MakeErr("Hay un item sin ProductID.")
		}
		if item.Quantity <= 0 {
			return req.MakeErr("Hay un item con Cantidad inválida (debe ser > 0).")
		}
	}

	// The order is locked while its status is checked and the stock enters: a second reception
	// waits on the row, then finds it Fulfilled, so the goods can never enter twice.
	var order types.PurchaseOrder
	err := db.RunTransaction(func(transaction *db.Transaction) error {
		orders := []types.PurchaseOrder{{CompanyID: req.User.CompanyID, ID: payload.PurchaseOrderID}}
		if err := db.LockByKeys(transaction, &orders); err != nil {
			return core.Err("Error al obtener la orden de compra.", err)
		}
		if len(orders) == 0 {
			return core.Err("Orden de compra no encontrada.")
		}
		order = orders[0]
		if order.Status != types.PurchaseOrderStatusConfirmed {
			return core.Err("La orden no está en estado Confirmada y no puede recibirse.")
		}
		return receivePurchaseOrderIn(transaction, req, &order, payload.WarehouseID, payload.Items)
	})
	if err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(order)
}

// receivePurchaseOrderIn ingresa la mercadería de una orden al almacén, dentro de la transacción
// que bloqueó la orden:
//  1. compara los productos recibidos vs. los pedidos en la OC y calcula
//     DifferenceValue (Σ (recibido - pedido) * precio), firmado
//     (negativo = subentrega, positivo = sobreentrega);
//  2. llama a ApplyMovimientos para insertar el stock en el almacén;
//  3. deja la OC en Status=Fulfilled con la diferencia calculada y la escribe.
//
// La diferencia se registra pero NO se rechaza: la OC se cumple aunque haya
// mismatches, y los valores quedan asentados para reportería.
func receivePurchaseOrderIn(
	transaction *db.Transaction, req *core.HandlerArgs, order *types.PurchaseOrder, warehouseID int32,
	items []PostStockAdjustItem,
) error {
	// Indexar el detalle pedido por (ProductID, PresentationID).
	type orderKey struct {
		ProductID      int32
		PresentationID int16
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

	for _, line := range order.Lines {
		s := getStats(orderKey{ProductID: line.ProductID, PresentationID: line.PresentationID})
		s.ordered += line.Quantity
		if s.price == 0 {
			s.price = line.Price
		}
	}

	for _, item := range items {
		getStats(orderKey{ProductID: item.ProductID, PresentationID: item.PresentationID}).received += item.Quantity
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
	// The order's prices are gross and in the order's currency; the stock is costed in soles and
	// net of the IGV the comprobante lets the company recover. An order whose comprobante is not
	// registered yet enters at the gross price, which is what it has.
	orderDocument := order.PurchaseDocument()
	movimientos := make([]types.InternalMovement, 0, len(items))
	for _, item := range items {
		key := orderKey{ProductID: item.ProductID, PresentationID: item.PresentationID}
		movimientos = append(movimientos, types.InternalMovement{
			Type:           types.StockMovementTypePurchaseReception,
			DocumentID:     int64(order.ID),
			WarehouseID:    warehouseID,
			ProductID:      item.ProductID,
			PresentationID: item.PresentationID,
			SerialNumber:   item.SerialNumber,
			LotID:          item.LotID,
			LotName:        item.LotCode,
			SupplierID:     order.ProviderID,
			Quantity:       item.Quantity,
			SubQuantity:    item.SubQuantity,
			Price:          orderDocument.InventoryUnitCost(getStats(key).price),
			// What the reception posts to, so the accounting operations never read the order back.
			PurchaseAccountCode: order.ReceptionAccountOf(item.ProductID),
		})
	}
	if err := types.ApplyMovimientosIn(transaction, req, movimientos); err != nil {
		return err
	}

	order.Status = types.PurchaseOrderStatusFulfilled
	order.DifferenceValue = diffValue
	order.Updated = core.SUnixTime()
	order.UpdatedBy = req.User.ID
	order.StampPeriod()
	q := db.TableOf[types.PurchaseOrder]()
	if err := db.UpdateIn(transaction, &[]types.PurchaseOrder{*order},
		q.Status, q.DifferenceValue, q.Updated, q.UpdatedBy, q.Period,
	); err != nil {
		return core.Err("Error al actualizar la orden de compra.", err)
	}
	return nil
}

func GetPurchaseOrders(req *core.HandlerArgs) core.HandlerResponse {
	statusParam := int8(req.GetQueryInt("status"))

	if statusParam == 0 {
		statusParam = types.PurchaseOrderStatusPending
	}

	// A first sync lists the requested status only; later syncs read the log, which names every
	// status (a canceled order, status 0, is logged as deleted and leaves every tab).
	records := []types.PurchaseOrder{}
	query := db.Query(&records)
	query.CompanyID.Equals(req.User.CompanyID).Delta(req.GetDeltaCursor(), statusParam)

	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener órdenes de compra:", err)
	}

	return req.MakeResponse(records)
}

// GetPurchaseOrdersByIDs serves the by-IDs cache: the accounting operations read through it the
// older orders a period's receptions name.
func GetPurchaseOrdersByIDs(req *core.HandlerArgs) core.HandlerResponse {
	cachedIDsRequest, err := req.GetCachedIDsRequest(req.User.CompanyID)
	if err != nil {
		return req.MakeErr(err)
	}
	partitions, err := db.QueryCachedIDs[types.PurchaseOrder](cachedIDsRequest)
	if err != nil {
		return req.MakeErr("Error al obtener las órdenes de compra.", err)
	}
	return req.MakeResponse(core.HideCachedPartitions(partitions))
}

func PostPurchaseOrder(req *core.HandlerArgs) core.HandlerResponse {
	record := types.PurchaseOrder{}
	if err := json.Unmarshal([]byte(*req.Body), &record); err != nil {
		return req.MakeErr("Error al deserializar el body.", err)
	}
	// Ingreso express: la orden nace y se recibe completa en su almacén en la misma llamada.
	express := struct{ ExpressEntry bool }{}
	if err := json.Unmarshal([]byte(*req.Body), &express); err != nil {
		return req.MakeErr("Error al deserializar el body.", err)
	}

	// This only creates. An existing order changes through PUT's actions, which check its status
	// and keep its debt, payments and comprobante.
	if record.ID != 0 {
		return req.MakeErr("Una orden existente se modifica con su acción de edición, no se vuelve a enviar.")
	}
	if record.ProviderID <= 0 {
		return req.MakeErr("Debe seleccionar un proveedor.")
	}
	if express.ExpressEntry && record.WarehouseID <= 0 {
		return req.MakeErr("Debe seleccionar el almacén donde ingresa la mercadería.")
	}

	// Supplies are Product rows with Status=2, so they arrive as ordinary product lines.
	if len(record.Lines) == 0 {
		return req.MakeErr("Debe agregar al menos un producto o insumo.")
	}
	// The total is the lines', never the client's: the debt and the comprobante check read it.
	record.TotalAmount = 0
	for _, line := range record.Lines {
		if line.ProductID <= 0 {
			return req.MakeErr("Hay una línea con ID = 0")
		}
		if line.Quantity == 0 {
			return req.MakeErr("Hay una línea con cantidad = 0")
		}
		record.TotalAmount += line.Price * line.Quantity
	}
	if err := resolvePurchaseLineAccounts(req.User.CompanyID, record.Lines); err != nil {
		return req.MakeErr(err)
	}

	now := core.SUnixTime()
	todayFecha := core.FechaUnix()
	currentSemana := core.MakeSemanaFromFechaUnix(todayFecha, false)

	record.CompanyID = req.User.CompanyID
	record.Status = types.PurchaseOrderStatusPending
	record.Created = now
	record.CreatedBy = req.User.ID
	record.Updated = now
	record.UpdatedBy = req.User.ID
	record.Date = todayFecha
	record.Week = currentSemana.Code
	// La deuda inicial corresponde al monto total: cada Pago la reduce hasta llegar a 0.
	record.DebtAmount = record.TotalAmount
	if express.ExpressEntry {
		record.DeliveryDate = todayFecha
	}

	// Congela la identidad del proveedor al crear la orden: el Registro de Compras
	// imprime su razón social y RUC tal como estaban en la compra, y renombrarlo
	// después no puede reescribir un libro ya presentado.
	snapshotID, err := crm.ProviderSnapshotID(req.User.CompanyID, record.ProviderID)
	if err != nil {
		return req.MakeErr(err)
	}
	record.ProviderSnapshotID = snapshotID
	// El comprobante puede venir con la orden o registrarse después con su propia acción.
	record.CurrencyType = core.If(record.CurrencyType == 0, finance.CurrencyPEN, record.CurrencyType)
	if err := record.ApplyPurchaseDocument(); err != nil {
		return req.MakeErr(err)
	}
	record.StampPeriod()

	// An express order is inserted and received in one transaction: a failed reception leaves no
	// order behind. The movements need the order's ID, which the insert assigns.
	var order types.PurchaseOrder
	err = db.RunTransaction(func(transaction *db.Transaction) error {
		orders := []types.PurchaseOrder{record}
		if err := db.InsertIn(transaction, &orders); err != nil {
			return core.Err("Error al guardar la orden de compra.", err)
		}
		order = orders[0]
		if !express.ExpressEntry {
			return nil
		}
		// Se recibe exactamente lo pedido, sin lote ni serie: quien necesite registrarlos usa el
		// ingreso normal.
		entryItems := make([]PostStockAdjustItem, len(order.Lines))
		for lineIndex, line := range order.Lines {
			entryItems[lineIndex] = PostStockAdjustItem{
				ProductID: line.ProductID, PresentationID: line.PresentationID, Quantity: line.Quantity,
			}
		}
		if err := receivePurchaseOrderIn(transaction, req, &order, order.WarehouseID, entryItems); err != nil {
			return core.Err("No se pudo ingresar la mercadería:", err)
		}
		return nil
	})
	if err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(order)
}

// resolvePurchaseLineAccounts stamps the PCGE account on every line, from what each line buys:
//   - a supply posts to the account picked on the line, else the supply's default;
//   - a bought product is merchandise, whose account (6011) follows from its origin, so the line
//     carries none;
//   - a made product is a finished good the company produces, and cannot be purchased.
func resolvePurchaseLineAccounts(companyID int32, lines []types.PurchaseOrderLine) error {
	productIDs := core.SliceSet[int32]{}
	for _, line := range lines {
		productIDs.Add(line.ProductID)
	}
	catalogRows := []production.Product{}
	catalogQuery := db.Query(&catalogRows)
	catalogQuery.Select(catalogQuery.ID, catalogQuery.Name, catalogQuery.Status, catalogQuery.Origin,
		catalogQuery.AccountCode).
		CompanyID.Equals(companyID).ID.In(productIDs.Values...)
	if err := catalogQuery.Exec(); err != nil {
		return core.Err("Error al obtener los productos de la orden.", err)
	}
	catalogByID := make(map[int32]production.Product, len(catalogRows))
	for _, catalogRow := range catalogRows {
		catalogByID[catalogRow.ID] = catalogRow
	}

	for lineIndex := range lines {
		line := &lines[lineIndex]
		catalogRow, found := catalogByID[line.ProductID]
		if !found {
			return core.Err("No se encontró el producto de la línea:", line.ProductID)
		}
		if catalogRow.Status != production.ProductStatusSupply {
			if catalogRow.Origin == production.ProductOriginMade {
				return core.Err(fmt.Sprintf(`"%s" es un producto fabricado por la empresa y no se compra.`, catalogRow.Name))
			}
			line.AccountCode = 0
			continue
		}
		accountCode, err := accounting.ResolveSupplyAccount(line.AccountCode, catalogRow.AccountCode)
		if err != nil {
			return core.Err(fmt.Sprintf(`Insumo "%s": %v`, catalogRow.Name, err))
		}
		line.AccountCode = accountCode
	}
	return nil
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

	// Every action runs on the order read under its row lock: two actions on one order apply one
	// after the other, each checking the status the other left, so two payments can never both fit
	// the same debt and an action never writes back a status another one already moved.
	var orderCurrent types.PurchaseOrder
	err := db.RunTransaction(func(transaction *db.Transaction) error {
		existing := []types.PurchaseOrder{{CompanyID: req.User.CompanyID, ID: orderID}}
		if err := db.LockByKeys(transaction, &existing); err != nil {
			return core.Err("Error al obtener la orden de compra.", err)
		}
		if len(existing) == 0 {
			return core.Err("Orden de compra no encontrada.")
		}
		orderCurrent = existing[0]
		return applyPurchaseOrderActionIn(transaction, req, action, &orderCurrent)
	})
	if err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(orderCurrent)
}

// applyPurchaseOrderActionIn applies one PUT action to the order its caller locked, and writes it.
func applyPurchaseOrderActionIn(
	transaction *db.Transaction, req *core.HandlerArgs, action int32, orderCurrent *types.PurchaseOrder,
) error {
	// Every action writes Status, which the period delta index packs with Period.
	orderCurrent.StampPeriod()
	now := core.SUnixTime()
	q := db.TableOf[types.PurchaseOrder]()

	switch action {
	case PurchaseOrderActionConfirm:
		// Solo se puede confirmar si está en estado Pendiente (1)
		if orderCurrent.Status != types.PurchaseOrderStatusPending {
			return core.Err("La orden no está en estado Pendiente y no puede confirmarse.")
		}
		orderCurrent.Status = types.PurchaseOrderStatusConfirmed
		orderCurrent.Updated = now
		orderCurrent.UpdatedBy = req.User.ID

		if err := db.UpdateIn(transaction, &[]types.PurchaseOrder{*orderCurrent}, q.Status, q.Updated, q.UpdatedBy, q.Period); err != nil {
			return core.Err("Error al actualizar la orden de compra.", err)
		}
		return nil

	case PurchaseOrderActionEdit:
		// Solo se permite editar mientras la orden esté Pendiente (1) o Confirmada (2);
		// los demás estados (Cancelada, Cumplida) son inmutables.
		if orderCurrent.Status != types.PurchaseOrderStatusPending &&
			orderCurrent.Status != types.PurchaseOrderStatusConfirmed {
			return core.Err("La orden no se puede editar en su estado actual.")
		}

		// Decodifica únicamente los campos editables; ProviderID, Status, productos y totales
		// se preservan desde el registro existente para evitar modificaciones no autorizadas.
		patch := types.PurchaseOrder{}
		if err := json.Unmarshal([]byte(*req.Body), &patch); err != nil {
			return core.Err("Error al deserializar el body.", err)
		}

		orderCurrent.WarehouseID = patch.WarehouseID
		orderCurrent.DeliveryDate = patch.DeliveryDate
		orderCurrent.PaymentDate = patch.PaymentDate
		orderCurrent.Notes = patch.Notes
		orderCurrent.Updated = now
		orderCurrent.UpdatedBy = req.User.ID

		if err := db.UpdateIn(transaction, &[]types.PurchaseOrder{*orderCurrent},
			q.WarehouseID, q.DeliveryDate, q.PaymentDate, q.Notes, q.Updated, q.UpdatedBy, q.Status, q.Period,
		); err != nil {
			return core.Err("Error al actualizar la orden de compra.", err)
		}
		return nil

	case PurchaseOrderActionPay:
		// Se paga una orden Confirmada o Cumplida: al proveedor se le suele pagar después de
		// entregar, así que recibir la mercadería no puede cerrar la deuda. Pendiente debe pasar
		// primero por Confirmar (acción 1) y una Cancelada no se paga.
		if orderCurrent.Status != types.PurchaseOrderStatusConfirmed &&
			orderCurrent.Status != types.PurchaseOrderStatusFulfilled {
			return core.Err("La orden no está Confirmada ni Cumplida y no puede pagarse.")
		}

		payload := purchaseOrderPayPayload{}
		if err := json.Unmarshal([]byte(*req.Body), &payload); err != nil {
			return core.Err("Error al deserializar el body.", err)
		}
		if payload.CashBankID <= 0 {
			return core.Err("Debe seleccionar una cashBank para registrar el pago.")
		}
		if payload.Amount <= 0 {
			return core.Err("El monto del pago debe ser mayor a 0.")
		}
		if payload.Amount > orderCurrent.DebtAmount {
			return core.Err("El monto del pago excede la deuda pendiente de la orden.")
		}

		// DebtAmount is in the order's currency and the cash movement subtracts it as is, so a
		// soles register paying a dollar order would lose the wrong amount. No conversion here:
		// the payment has to come from a register in the order's currency.
		cashBank, err := finance.GetCaja(req.User.CompanyID, payload.CashBankID)
		if err != nil {
			return err
		}
		cashBankCurrency := core.Coalesce(cashBank.CurrencyType, finance.CurrencyPEN)
		orderCurrency := core.Coalesce(orderCurrent.CurrencyType, finance.CurrencyPEN)
		if cashBankCurrency != orderCurrency {
			return core.Err(fmt.Sprintf("La caja \"%v\" no está en la moneda de la orden de compra.", cashBank.Name))
		}

		// When this payment alone pays the whole order on its comprobante's day, the accounting
		// operations infer it from the order instead of downloading it.
		paymentDate := core.FechaUnix()
		isSameDayPayment := finance.IsSameDaySettlement(orderCurrent.DocIssueDate, orderCurrent.TotalAmount,
			orderCurrent.TotalAmount-orderCurrent.DebtAmount, paymentDate, payload.Amount)

		// El pago sale de la cashBank: Amount negativo para que ApplyCajaMovimientos descuente del saldo.
		movimiento := finance.InternalCashMovement{
			CashBankID:         payload.CashBankID,
			DocumentID:         int64(orderCurrent.ID),
			Date:               paymentDate,
			Type:               finance.CashMovementTypeSupplierPayment,
			CounterAccountCode: finance.PurchasePayableAccount(orderCurrent.DocType),
			Amount:             -payload.Amount,
			SameDayMovement:    int8(core.If(isSameDayPayment, 1, 0)),
		}
		if err := finance.ApplyCashBankMovementIn(transaction, req, []finance.InternalCashMovement{movimiento}); err != nil {
			return core.Err("Error al registrar el movimiento de cashBank:", err)
		}

		orderCurrent.DebtAmount -= payload.Amount
		orderCurrent.Updated = now
		orderCurrent.UpdatedBy = req.User.ID
		if isSameDayPayment {
			orderCurrent.SameDayCashBankID = payload.CashBankID
		}

		if err := db.UpdateIn(transaction, &[]types.PurchaseOrder{*orderCurrent},
			q.Status, q.DebtAmount, q.Updated, q.UpdatedBy, q.Period, q.SameDayCashBankID,
		); err != nil {
			return core.Err("Error al actualizar la deuda de la orden de compra.", err)
		}
		return nil

	case PurchaseOrderActionAnnul:
		// Solo se permite anular órdenes en estado Pendiente o Confirmada; las ya Canceladas
		// o Cumplidas son inmutables para preservar consistencia contable.
		if orderCurrent.Status != types.PurchaseOrderStatusPending &&
			orderCurrent.Status != types.PurchaseOrderStatusConfirmed {
			return core.Err("La orden no se puede anular en su estado actual.")
		}
		orderCurrent.Status = types.PurchaseOrderStatusCanceled
		orderCurrent.Updated = now
		orderCurrent.UpdatedBy = req.User.ID

		if err := db.UpdateIn(transaction, &[]types.PurchaseOrder{*orderCurrent}, q.Status, q.Updated, q.UpdatedBy, q.Period); err != nil {
			return core.Err("Error al anular la orden de compra.", err)
		}
		return nil

	case PurchaseOrderActionDocument:
		if orderCurrent.Status == types.PurchaseOrderStatusCanceled {
			return core.Err("Una orden anulada no se registra en el Registro de Compras.")
		}
		patch := types.PurchaseOrder{}
		if err := json.Unmarshal([]byte(*req.Body), &patch); err != nil {
			return core.Err("Error al deserializar el body.", err)
		}
		orderCurrent.SetPurchaseDocument(patch.PurchaseDocument())
		if err := orderCurrent.ApplyPurchaseDocument(); err != nil {
			return core.Err(err)
		}
		orderCurrent.Updated = now
		orderCurrent.UpdatedBy = req.User.ID
		// The comprobante moves the order's accounting date and decides the payable its payments
		// clear (4212 / 4699), so they are re-stamped against it.
		orderCurrent.StampPeriod()
		sameDayCashBankID, restampErr := finance.RestampDocumentPaymentsIn(transaction, req.User.CompanyID, int64(orderCurrent.ID),
			[]finance.CashMovementType{finance.CashMovementTypeSupplierPayment},
			orderCurrent.DocIssueDate, orderCurrent.TotalAmount, finance.PurchasePayableAccount(orderCurrent.DocType))
		if restampErr != nil {
			return core.Err(restampErr)
		}
		orderCurrent.SameDayCashBankID = sameDayCashBankID

		if err := db.UpdateIn(transaction, &[]types.PurchaseOrder{*orderCurrent},
			q.DocType, q.DocSeries, q.DocNumber, q.DocIssueDate, q.TaxableAmount, q.TaxAmount,
			q.UntaxedAmount, q.OtherAmount, q.CurrencyType, q.ExchangeRate,
			q.Updated, q.UpdatedBy, q.Status, q.Period, q.SameDayCashBankID,
		); err != nil {
			return core.Err("Error al registrar el comprobante de la orden de compra.", err)
		}
		return nil

	default:
		return core.Err("Acción no válida.")
	}
}
