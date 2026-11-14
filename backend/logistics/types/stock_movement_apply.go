// Stock movement engine. This is business logic in a `types` package on purpose: `sales` and
// `accounting` both have to write stock, and a module body may not import another module body
// (see backend/docs/MODULE_BOUNDARIES.md). Living here is what lets them call it while importing
// only `app/logistics/types`.
//
// Concurrency: every stock writer runs in a database transaction and locks the ProductStock rows it
// touches (db.LockByKeys) before reading anything under them, so two writers of one row run one
// after the other, in any process. A ProductStock row guards its ProductStockDetail rows: nothing
// writes a detail without locking its stock row first. A row that does not exist yet cannot be
// locked, so it is created with a plain INSERT: when two writers both create it, the loser's
// transaction fails on the duplicate key and RunTransaction reruns it against the winner's row. Lots
// follow the same rule through their unique (Date, SupplierID, Name) index.
//
// ApplyMovimientos and RecalcProductStockByMovements share the packed-key helper, which is why the
// recalc moved with the engine rather than staying behind and forcing both to be exported.

package types

import (
	"app/core"
	"app/db"
	"cmp"
	"fmt"
	"slices"
)

// PackProductStockID mirrors the ORM's KeyIntPacking for ProductStockV2 so the
// application can compute the packed key for lookups/detail wiring without
// round-tripping through inserts.
// Schema: WarehouseID.Size(17) + ProductID.Size(30) + PresentationID.Size(14). The ORM packs the
// slots from the bottom (db.PackSlotValues), so the 2 unused bits of the 63-bit budget sit on top.
func PackProductStockID(warehouseID int32, productID int32, presentationID int16) int64 {
	return int64(warehouseID)<<44 | int64(productID)<<14 | int64(presentationID)
}

const maxProductStockLastPrices = 8

func appendProductStockLastPrice(stock *ProductStock, movementQuantity int32, price int32) {
	if movementQuantity <= 0 || price <= 0 {
		return
	}

	// Normalize old rows defensively so both slices keep one-to-one positions.
	pairCount := min(len(stock.LastPricesPrice), len(stock.LastPricesQuantity))
	if pairCount != len(stock.LastPricesPrice) || pairCount != len(stock.LastPricesQuantity) {
		stock.LastPricesPrice = core.TrimSliceLeft(stock.LastPricesPrice, pairCount)
		stock.LastPricesQuantity = core.TrimSliceLeft(stock.LastPricesQuantity, pairCount)
	}

	// Keep price and quantity aligned as a compact chronological history.
	stock.LastPricesPrice = append(stock.LastPricesPrice, price)
	stock.LastPricesQuantity = append(stock.LastPricesQuantity, movementQuantity)

	stock.LastPricesPrice = core.TrimSliceLeft(stock.LastPricesPrice, maxProductStockLastPrices)
	stock.LastPricesQuantity = core.TrimSliceLeft(stock.LastPricesQuantity, maxProductStockLastPrices)
	core.Log("ApplyMovimientos actualizó historial de precios:",
		"stockID", stock.ID, "quantity", movementQuantity, "price", price, "entries", len(stock.LastPricesPrice))
}

// movementDivisorOf treats an unset divisor as "whole units only". A movement carrying
// sub-units without saying what they are is rejected rather than guessed at, because the
// guess would be silently wrong for exactly the products this feature exists for.
func movementDivisorOf(mov *InternalMovement) (int16, error) {
	if mov.SubDivisor > 0 {
		if err := core.ValidateQuantityDivisor(mov.SubDivisor); err != nil {
			return 0, err
		}
		return mov.SubDivisor, nil
	}
	if mov.SubQuantity != 0 {
		return 0, core.Err(fmt.Sprintf(
			"Movimiento con sub-cantidad %v sin divisor de sub-unidad (producto %v, almacén %v).",
			mov.SubQuantity, mov.ProductID, mov.WarehouseID))
	}
	return core.QuantityDivisorNone, nil
}

// stockDivisorOf reads the divisor a stock row is currently expressed in. A row that has
// never carried sub-units has none, and adopts whatever the next movement brings.
func stockDivisorOf(stock *ProductStock) int16 {
	if stock.SubDivisor > 0 {
		return stock.SubDivisor
	}
	return core.QuantityDivisorNone
}

// refineStockDivisor re-expresses a stock row and every detail row under it at a finer
// divisor. It must be atomic across the details: ProductStock.DetailSubQuantity is an
// absolute re-sum of them, so converting only some would leave the row summing sixths
// together with twelfths.
func refineStockDivisor(stock *ProductStock, details []*ProductStockDetail,
	toDivisor int16, updatedTime int32, userID int32) error {

	fromDivisor := stockDivisorOf(stock)
	if fromDivisor == toDivisor {
		stock.SubDivisor = toDivisor
		return nil
	}

	freeBucket, err := core.Quantity{Units: stock.Quantity, Sub: stock.SubQuantity}.
		ConvertToDivisor(fromDivisor, toDivisor)
	if err != nil {
		return err
	}
	detailBucket, err := core.Quantity{Units: stock.DetailQuantity, Sub: stock.DetailSubQuantity}.
		ConvertToDivisor(fromDivisor, toDivisor)
	if err != nil {
		return err
	}
	computedBucket, err := core.Quantity{Units: stock.DetailComputedQuantity, Sub: stock.DetailComputedSubQuantity}.
		ConvertToDivisor(fromDivisor, toDivisor)
	if err != nil {
		return err
	}
	reservedProduction, err := core.Quantity{Units: stock.ReservedProductionQuantity, Sub: stock.ReservedProductionSubQuantity}.
		ConvertToDivisor(fromDivisor, toDivisor)
	if err != nil {
		return err
	}
	blocked, err := core.Quantity{Units: stock.BlockedQuantity, Sub: stock.BlockedSubQuantity}.
		ConvertToDivisor(fromDivisor, toDivisor)
	if err != nil {
		return err
	}

	// Convert the details first so a failure leaves nothing half-converted.
	convertedDetails := make([]core.Quantity, len(details))
	for i, detail := range details {
		converted, err := core.Quantity{Units: detail.Quantity, Sub: detail.SubQuantity}.
			ConvertToDivisor(fromDivisor, toDivisor)
		if err != nil {
			return err
		}
		convertedDetails[i] = converted
	}
	for i, detail := range details {
		detail.Quantity, detail.SubQuantity = convertedDetails[i].Units, convertedDetails[i].Sub
		// Stamp so the write step sees the row as dirty; an unstamped detail is skipped and
		// would stay behind at the old divisor.
		detail.Updated = updatedTime
		detail.UpdatedBy = userID
	}

	stock.Quantity, stock.SubQuantity = freeBucket.Units, freeBucket.Sub
	stock.DetailQuantity, stock.DetailSubQuantity = detailBucket.Units, detailBucket.Sub
	stock.DetailComputedQuantity, stock.DetailComputedSubQuantity = computedBucket.Units, computedBucket.Sub
	stock.ReservedProductionQuantity, stock.ReservedProductionSubQuantity = reservedProduction.Units, reservedProduction.Sub
	stock.BlockedQuantity, stock.BlockedSubQuantity = blocked.Units, blocked.Sub
	stock.SubDivisor = toDivisor
	core.Log("ApplyMovimientos refinó el divisor de stock:",
		"stockID", stock.ID, "de", fromDivisor, "a", toDivisor, "detalles", len(details))
	return nil
}

// expandStockTransfers turns every movement that names a DestWarehouseID into its two ledger
// rows: the outflow at the origin and the inflow at the destination, each naming the other
// warehouse. Two rows, like a cash transfer, so each warehouse's history and Kardex come from its
// own movements. The inflow's value is left for ApplyMovimientos: it is what the outflow leg took
// out of the origin, known only inside the lock.
func expandStockTransfers(movements []InternalMovement) ([]InternalMovement, error) {
	expanded := make([]InternalMovement, 0, len(movements))
	for _, movement := range movements {
		if movement.DestWarehouseID == 0 {
			expanded = append(expanded, movement)
			continue
		}
		if movement.ReplaceQuantity {
			return nil, core.Err("Un traslado mueve una cantidad, no puede fijar el stock del almacén.")
		}
		if movement.DestWarehouseID == movement.WarehouseID {
			return nil, core.Err("El almacén de destino debe ser distinto al de origen.")
		}
		if movement.Quantity < 0 || movement.SubQuantity < 0 || (movement.Quantity == 0 && movement.SubQuantity == 0) {
			return nil, core.Err(fmt.Sprintf("La cantidad a trasladar del producto %v debe ser positiva.", movement.ProductID))
		}

		outflow := movement
		outflow.DestWarehouseID, outflow.Price, outflow.InflowValue = 0, 0, 0
		outflow.Quantity, outflow.SubQuantity = -movement.Quantity, -movement.SubQuantity
		outflow.transferPeerWarehouseID = movement.DestWarehouseID

		inflow := movement
		inflow.WarehouseID, inflow.DestWarehouseID, inflow.Price, inflow.InflowValue = movement.DestWarehouseID, 0, 0, 0
		inflow.transferPeerWarehouseID = movement.WarehouseID

		expanded = append(expanded, outflow, inflow)
	}
	return expanded, nil
}

// ApplyMovimientos writes the movements, the stock they leave and their ledger rows in a transaction
// of their own.
func ApplyMovimientos(req *core.HandlerArgs, movimientos []InternalMovement) error {
	return db.RunTransaction(func(transaction *db.Transaction) error {
		return ApplyMovimientosIn(transaction, req, movimientos)
	})
}

// ApplyMovimientosIn is ApplyMovimientos inside the caller's transaction, so the stock moves commit
// with the document that caused them (a sale's delivery). It locks the stock rows before reading
// under them, and its negative-stock check runs on those locked rows: it is the final word on
// whether the movements fit. Everything it writes derives from what it reads in the transaction, and
// it never mutates movimientos, so a rerun of the caller's body (deadlock, duplicate key) runs it cleanly.
func ApplyMovimientosIn(transaction *db.Transaction, req *core.HandlerArgs, movimientos []InternalMovement) error {
	_, err := ApplyMovimientosWithLedgerIn(transaction, req, movimientos)
	return err
}

// ApplyMovimientosWithLedgerIn is ApplyMovimientosIn answering the ledger rows it wrote, each with the
// MonetaryValue the engine stamped: one per movement, in order, after the no-ops (a zero delta) drop
// and a transfer expands into its two legs. The production close values its products from them.
func ApplyMovimientosWithLedgerIn(transaction *db.Transaction, req *core.HandlerArgs, movimientos []InternalMovement) ([]WarehouseProductMovement, error) {
	companyID := req.User.CompanyID
	userID := req.User.ID

	movimientos, err := expandStockTransfers(movimientos)
	if err != nil {
		return nil, err
	}

	// Filter out no-ops and validate lot/supplier prerequisites in one pass.
	activeMovements := make([]*InternalMovement, 0, len(movimientos))
	for i := range movimientos {
		mov := &movimientos[i]
		// A zero is only a no-op as a delta. As a "set stock to X" target it writes the row off.
		if mov.Quantity == 0 && mov.SubQuantity == 0 && !mov.ReplaceQuantity {
			continue
		}
		if mov.WarehouseID == 0 || mov.ProductID == 0 {
			return nil, core.Err("Movimiento inválido: falta WarehouseID o ProductID.")
		}
		// The writer always names the type; the engine never guesses it from the sign.
		if err := validateStockMovementDocument(mov.Type, mov.DocumentID); err != nil {
			return nil, err
		}
		if err := validateStockMovementDate(mov.Type, mov.Date); err != nil {
			return nil, err
		}
		// Outbound (Quantity < 0) must ship a resolved LotID; name-based lookup is inbound-only.
		if mov.LotID == 0 && mov.LotName != "" && mov.Quantity < 0 {
			return nil, core.Err(fmt.Sprintf("Movimiento con Lote %q sin LotID para salida (product %v, almacén %v).",
				mov.LotName, mov.ProductID, mov.WarehouseID))
		}
		// SupplierID == 0 is allowed for manual stock-adjustment lots; the lot's unique key
		// becomes (today, 0, name) which stays consistent within the day.
		activeMovements = append(activeMovements, mov)
	}
	if len(activeMovements) == 0 {
		return nil, nil
	}

	// Resolve missing LotIDs for inbound lot-by-name movements.
	lotQualityByID, err := resolveLotIDsForMovements(transaction, req, activeMovements, core.FechaUnix())
	if err != nil {
		return nil, err
	}

	// Compute V2 packed IDs per movement and bucket them for preload.
	stockIDByMovement := make([]int64, len(activeMovements))
	divisorByMovement := make([]int16, len(activeMovements))
	stockIDSet := core.SliceSet[int64]{}
	stockIDsWithDetails := core.SliceSet[int64]{}
	for i, mov := range activeMovements {
		id := PackProductStockID(mov.WarehouseID, mov.ProductID, mov.PresentationID)
		stockIDByMovement[i] = id
		stockIDSet.Add(id)
		if mov.HasDetail() {
			stockIDsWithDetails.Add(id)
		}
		movementDivisor, err := movementDivisorOf(mov)
		if err != nil {
			return nil, err
		}
		divisorByMovement[i] = movementDivisor
	}

	// Lock the stock rows first: every read below them (their details) then sees the last committed
	// state, and nobody else writes it until this transaction ends. In key order, so two writers of
	// overlapping rows queue instead of deadlocking.
	stockByID := map[int64]*ProductStock{}
	lockedStocks := make([]ProductStock, 0, len(stockIDSet.Values))
	slices.Sort(stockIDSet.Values)
	for _, stockID := range stockIDSet.Values {
		lockedStocks = append(lockedStocks, ProductStock{CompanyID: companyID, ID: stockID})
	}
	if err := db.LockByKeys(transaction, &lockedStocks); err != nil {
		return nil, core.Err("Error al obtener stock V2 previo:", err)
	}
	for i := range lockedStocks {
		stockByID[lockedStocks[i].ID] = &lockedStocks[i]
	}
	if err := lockMovementLotsIn(transaction, companyID, activeMovements, lotQualityByID); err != nil {
		return nil, err
	}
	// Rows this call creates: written with a plain INSERT, which fails (and reruns the transaction)
	// if a concurrent writer created the same row first.
	newStockIDs := core.SliceSet[int64]{}

	// The detail preload excludes Created/CreatedBy (flagging UPDATE at write time) and
	// Updated/UpdatedBy, so untouched preloaded details stay at Updated==0 and are skipped by the
	// write step. The locked stock rows guard them, so a plain read is enough.
	detailByKey := map[string]*ProductStockDetail{}
	detailKey := func(stockID int64, lotID int32, serial string) string {
		return db.MakeKeyConcat(stockID, lotID, serial)
	}
	if !stockIDsWithDetails.IsEmpty() {
		existing := []ProductStockDetail{}
		q := db.Query(&existing)
		q.Exclude(q.Created, q.CreatedBy, q.Updated, q.UpdatedBy).
			CompanyID.Equals(companyID).
			ProductStockID.In(stockIDsWithDetails.Values...)
		if err := q.Exec(); err != nil {
			return nil, core.Err("Error al obtener detalle de stock previo:", err)
		}
		for i := range existing {
			row := &existing[i]
			detailByKey[detailKey(row.ProductStockID, row.LotID, row.SerialNumber)] = row
		}
	}

	updatedTime := core.SUnixTime()
	dateUnix := core.FechaUnix()

	// Divisor reconciliation, before any mutation. A stock row records the divisor it was
	// last written under; when a movement arrives at a finer one the row converts itself,
	// which is what lets a divisor refinement stay lazy instead of rewriting the ledger.
	// Refinements are rare by design, so the extra detail load below almost never fires.
	refinementTargetByStockID := map[int64]int16{}
	for i := range activeMovements {
		stock := stockByID[stockIDByMovement[i]]
		if stock == nil {
			continue
		}
		currentDivisor := stockDivisorOf(stock)
		if currentDivisor == divisorByMovement[i] {
			continue
		}
		// Only a strictly finer movement divisor pulls the stock row forward. A coarser one
		// is converted up to the row instead, in the mutation loop.
		if !core.IsQuantityDivisorRefinement(currentDivisor, divisorByMovement[i]) {
			continue
		}
		if target, seen := refinementTargetByStockID[stock.ID]; !seen ||
			core.IsQuantityDivisorRefinement(target, divisorByMovement[i]) {
			refinementTargetByStockID[stock.ID] = divisorByMovement[i]
		}
	}

	if len(refinementTargetByStockID) > 0 {
		// A stock touched only by a free-bucket movement had no details preloaded, but its
		// details still hold sub-units at the old divisor and must convert with it.
		missingDetailStockIDs := core.SliceSet[int64]{}
		for stockID := range refinementTargetByStockID {
			if !stockIDsWithDetails.Include(stockID) {
				missingDetailStockIDs.Add(stockID)
			}
		}
		if !missingDetailStockIDs.IsEmpty() {
			existing := []ProductStockDetail{}
			q := db.Query(&existing)
			q.Exclude(q.Created, q.CreatedBy, q.Updated, q.UpdatedBy).
				CompanyID.Equals(companyID).
				ProductStockID.In(missingDetailStockIDs.Values...)
			if err := q.Exec(); err != nil {
				return nil, core.Err("Error al obtener detalle de stock para refinar el divisor:", err)
			}
			for i := range existing {
				row := &existing[i]
				detailByKey[detailKey(row.ProductStockID, row.LotID, row.SerialNumber)] = row
			}
		}

		detailsByStockID := map[int64][]*ProductStockDetail{}
		for _, detail := range detailByKey {
			if _, refining := refinementTargetByStockID[detail.ProductStockID]; refining {
				detailsByStockID[detail.ProductStockID] = append(detailsByStockID[detail.ProductStockID], detail)
			}
		}
		for stockID, targetDivisor := range refinementTargetByStockID {
			if err := refineStockDivisor(stockByID[stockID], detailsByStockID[stockID],
				targetDivisor, updatedTime, userID); err != nil {
				return nil, err
			}
		}
	}

	// Build ledger rows and mutate V2/Detail in place.
	warehouseMovements := make([]WarehouseProductMovement, 0, len(activeMovements))
	// Units held by each stock row (free + detail) as the batch replays, which is what an inflow
	// averages its cost against. DetailQuantity is only re-summed after the loop, so it cannot
	// be read mid-batch; this map is seeded from the loaded row and carries every delta since.
	balanceByStockID := map[int64]core.Quantity{}
	// The rows an outflow of this call touched: they are checked against their production reservation.
	outflowStockIDs := map[int64]bool{}
	// The value the last transfer outflow leg took out; expandStockTransfers puts its inflow leg
	// right after it.
	transferOutflowValue := int32(0)
	for i, mov := range activeMovements {
		stockID := stockIDByMovement[i]
		stock := stockByID[stockID]
		if stock == nil {
			newStockIDs.Add(stockID)
			stock = &ProductStock{
				ID:             stockID,
				CompanyID:      companyID,
				WarehouseID:    mov.WarehouseID,
				ProductID:      mov.ProductID,
				PresentationID: mov.PresentationID,
				SubDivisor:     divisorByMovement[i],
				Created:        updatedTime,
				CreatedBy:      userID,
			}
			stockByID[stockID] = stock
		}

		// The row's divisor wins. The reconciliation pass already pulled it forward if any
		// movement was finer, so anything still different here is a coarser movement, which
		// converts up to the row exactly.
		effectiveDivisor := stockDivisorOf(stock)
		movementQuantity := core.Quantity{Units: mov.Quantity, Sub: mov.SubQuantity}
		if divisorByMovement[i] != effectiveDivisor {
			converted, convertErr := movementQuantity.ConvertToDivisor(divisorByMovement[i], effectiveDivisor)
			if convertErr != nil {
				return nil, core.Err(fmt.Sprintf(
					"Movimiento incompatible con el divisor del stock (producto %v, almacén %v, divisor stock %v, divisor movimiento %v).",
					mov.ProductID, mov.WarehouseID, effectiveDivisor, divisorByMovement[i]))
			}
			movementQuantity = converted
		}

		balanceBefore, balanceSeeded := balanceByStockID[stockID]
		if !balanceSeeded {
			balanceBefore = core.Quantity{Units: stock.Quantity, Sub: stock.SubQuantity}.
				Add(core.Quantity{Units: stock.DetailQuantity, Sub: stock.DetailSubQuantity})
		}

		movement := WarehouseProductMovement{
			DocumentID:     mov.DocumentID,
			CompanyID:      companyID,
			WarehouseID:    mov.WarehouseID,
			ProductID:      mov.ProductID,
			PresentationID: mov.PresentationID,
			SerialNumber:   mov.SerialNumber,
			LotID:          mov.LotID,
			SubDivisor:     effectiveDivisor,
			Date:           core.Coalesce(mov.Date, dateUnix),
			Created:        updatedTime,
			CreatedBy:      userID,
			Period:         core.PeriodOfDay(core.Coalesce(mov.Date, dateUnix)),
			// What a reception posts to, stamped so the accounting operations never read the purchase.
			PurchaseAccountCode: mov.PurchaseAccountCode,
		}

		if mov.HasDetail() {
			key := detailKey(stockID, mov.LotID, mov.SerialNumber)
			detail := detailByKey[key]
			if detail == nil {
				detail = &ProductStockDetail{
					CompanyID:      companyID,
					ProductStockID: stockID,
					LotID:          mov.LotID,
					SerialNumber:   mov.SerialNumber,
					WarehouseID:    mov.WarehouseID,
					ProductID:      mov.ProductID,
					Created:        updatedTime,
					CreatedBy:      userID,
				}
				detailByKey[key] = detail
			}

			previous := core.Quantity{Units: detail.Quantity, Sub: detail.SubQuantity}
			var next core.Quantity
			if mov.ReplaceQuantity {
				// "Set stock to X": the ledger records the delta that gets it there.
				delta := movementQuantity.Add(previous.Negate())
				movement.Quantity, movement.SubQuantity = delta.Units, delta.Sub
				next = movementQuantity
			} else {
				movement.Quantity, movement.SubQuantity = movementQuantity.Units, movementQuantity.Sub
				next = previous.Add(movementQuantity)
			}
			detail.Quantity, detail.SubQuantity = next.Units, next.Sub
			// Stamp Updated/UpdatedBy so the partition step recognises this detail as dirty.
			detail.Updated = updatedTime
			detail.UpdatedBy = userID
			detail.Status = core.If(next.IsZero(), int8(0), int8(1))
		} else {
			// Free bucket: mutate V2.Quantity only, leave DetailQuantity for the final re-sum pass.
			previous := core.Quantity{Units: stock.Quantity, Sub: stock.SubQuantity}
			var next core.Quantity
			if mov.ReplaceQuantity {
				delta := movementQuantity.Add(previous.Negate())
				movement.Quantity, movement.SubQuantity = delta.Units, delta.Sub
				next = movementQuantity
			} else {
				movement.Quantity, movement.SubQuantity = movementQuantity.Units, movementQuantity.Sub
				next = previous.Add(movementQuantity)
			}
			stock.Quantity, stock.SubQuantity = next.Units, next.Sub
		}

		// Everything below reads the delta, never the requested quantity: for a "set stock to X"
		// movement the two differ, and lowering stock from 10 to 5 is an outflow of 5.
		delta := core.Quantity{Units: movement.Quantity, Sub: movement.SubQuantity}
		if delta.IsZero() {
			// A target equal to the current stock moves nothing and leaves no ledger row.
			continue
		}
		isInflow := !delta.IsNegative(effectiveDivisor)
		if err := validateStockMovementDirection(mov.Type, isInflow, mov.ProductID); err != nil {
			return nil, err
		}
		// A lot in Cuarentena or Rechazado is blocked stock (M22): it is neither sold, consumed nor
		// transferred, only disposed of by a manual exit or a shrinkage. Whatever moves in or out of
		// it moves its row's Blocked counter with it.
		if mov.LotID > 0 && IsLotQualityBlocked(lotQualityByID[mov.LotID]) {
			if !isInflow && mov.Type != StockMovementTypeManualExit && mov.Type != StockMovementTypeShrinkage {
				return nil, core.Err(fmt.Sprintf(
					"El lote %v del producto %v no está liberado: solo puede salir por salida manual o merma.",
					mov.LotID, mov.ProductID))
			}
			blocked := core.Quantity{Units: stock.BlockedQuantity, Sub: stock.BlockedSubQuantity}.Add(delta)
			stock.BlockedQuantity, stock.BlockedSubQuantity = blocked.Units, blocked.Sub
		}
		movement.Type = mov.Type
		movement.WarehouseRefID = mov.transferPeerWarehouseID

		balanceSubUnits := balanceBefore.TotalSubUnits(effectiveDivisor)
		deltaSubUnits := delta.TotalSubUnits(effectiveDivisor)

		if !isInflow {
			outflowStockIDs[stockID] = true
			// An outflow leaves at its share of the row's value; the average does not move.
			movement.MonetaryValue = StockOutflowValue(stock.StockValue, balanceSubUnits, -deltaSubUnits)
			stock.StockValue -= int64(movement.MonetaryValue)
			if mov.transferPeerWarehouseID > 0 {
				transferOutflowValue = movement.MonetaryValue
			}
		}

		if isInflow && mov.transferPeerWarehouseID > 0 {
			// A transfer's inflow enters with exactly the value its outflow leg, expanded right
			// before it, took out of the origin: moving goods between warehouses creates no value.
			mov.InflowValue = transferOutflowValue
		}

		if isInflow && mov.RejectInbound {
			return nil, core.Err(fmt.Sprintf(
				"El producto %v no puede aumentar su stock con un movimiento manual: la empresa sólo permite ingresos por orden de compra.",
				mov.ProductID))
		}
		if isInflow && mov.RequireInboundCost && mov.Price <= 0 {
			return nil, core.Err(fmt.Sprintf(
				"El producto %v necesita el costo de compra del stock que ingresa.", mov.ProductID))
		}
		// Stock is never let in without a value: with no exact value and no Price, an inflow is
		// valued at the row's average, so a row that has none cannot take one. This holds for every
		// document, not only manual adjustments; a transfer of uncosted stock is refused here too.
		if isInflow && mov.InflowValue <= 0 && mov.Price <= 0 && stock.AverageCost <= 0 {
			return nil, core.Err(fmt.Sprintf(
				"El producto %v no tiene costo unitario consolidado: el stock que ingresa necesita su costo unitario.",
				mov.ProductID))
		}

		if isInflow {
			// The ledger carries the inflow's total value. The sub-unit part is prorated: there
			// is only a unit price, with no separate sub-unit price to charge against.
			lineValue := mov.InflowValue
			if lineValue <= 0 {
				// Only a real price enters the last-prices history; the average standing in for a
				// missing one is not a price anybody paid.
				if mov.Price > 0 {
					appendProductStockLastPrice(stock, movement.Quantity, mov.Price)
				}
				unitPrice := core.If(mov.Price > 0, mov.Price, stock.AverageCost)
				var valueErr error
				if lineValue, valueErr = core.QuantityValueAtUnitPrice(delta, effectiveDivisor, unitPrice); valueErr != nil {
					return nil, valueErr
				}
			}
			movement.MonetaryValue = lineValue
			stock.StockValue, stock.AverageCost = StockAfterInflow(stock.StockValue, stock.AverageCost,
				balanceSubUnits, deltaSubUnits, lineValue, effectiveDivisor)
		}
		balanceByStockID[stockID] = balanceBefore.Add(delta)
		stock.Updated = updatedTime
		stock.UpdatedBy = userID
		// WarehouseQuantity on the ledger row uses the post-mutation total (free + detail).
		// DetailQuantity may be finalised in the re-sum pass, so we re-stamp after it below.
		warehouseMovements = append(warehouseMovements, movement)
	}

	// Single re-sum pass: any stock whose details changed gets its DetailQuantity/SubQuantity refreshed
	// from the current in-memory state (includes both mutated and preloaded-untouched rows).
	if len(detailByKey) > 0 {
		type detailSum struct{ quantity, subQuantity int32 }
		sumsByStockID := map[int64]detailSum{}
		for _, d := range detailByKey {
			s := sumsByStockID[d.ProductStockID]
			s.quantity += d.Quantity
			s.subQuantity += d.SubQuantity
			sumsByStockID[d.ProductStockID] = s
		}
		for stockID, sum := range sumsByStockID {
			if stock := stockByID[stockID]; stock != nil {
				stock.DetailQuantity = sum.quantity
				stock.DetailSubQuantity = sum.subQuantity
			}
		}
	}

	// An outflow may not eat what released production orders reserved (docs/MANUFACTURING_PLAN.md §5).
	// The owning order releases its own reservation before its consumption reaches the engine, so
	// only somebody else's outflow can trip this. Blocked lots cannot serve a reservation, so the
	// reservation must fit in the rest (§10).
	for stockID := range outflowStockIDs {
		stock := stockByID[stockID]
		reserved := core.Quantity{Units: stock.ReservedProductionQuantity, Sub: stock.ReservedProductionSubQuantity}
		if reserved.IsZero() {
			continue
		}
		stockDivisor := stockDivisorOf(stock)
		unblockedBalance := core.Quantity{Units: stock.Quantity, Sub: stock.SubQuantity}.
			Add(core.Quantity{Units: stock.DetailQuantity, Sub: stock.DetailSubQuantity}).
			Add(core.Quantity{Units: stock.BlockedQuantity, Sub: stock.BlockedSubQuantity}.Negate())
		if unblockedBalance.TotalSubUnits(stockDivisor) < reserved.TotalSubUnits(stockDivisor) {
			return nil, core.Err(fmt.Sprintf(
				"El producto %v del almacén %v tiene stock reservado por órdenes de producción: quedarían %v disponibles con %v reservados.",
				stock.ProductID, stock.WarehouseID, unblockedBalance.TotalSubUnits(stockDivisor), reserved.TotalSubUnits(stockDivisor)))
		}
	}

	// Backfill WarehouseQuantity on each ledger row now that DetailQuantity is final.
	for i := range warehouseMovements {
		if stock := stockByID[stockIDByMovement[i]]; stock != nil {
			warehouseMovements[i].WarehouseQuantity = stock.Quantity + stock.DetailQuantity
		}
	}

	// Flatten to value slices. Validate non-negative balances at the same time.
	// Untouched preloaded details (Updated==0) are skipped so InsertUpdateInclude only sees dirty rows.
	newStocks := []ProductStock{}
	lockedStocksToUpdate := []ProductStock{}
	for _, stock := range stockByID {
		// A plain Quantity < 0 test misses {0,-4}: four sub-units oversold, whose whole-unit
		// field is exactly zero. Both buckets have to be judged as normalized values.
		stockDivisor := stockDivisorOf(stock)
		freeBucket := core.Quantity{Units: stock.Quantity, Sub: stock.SubQuantity}
		detailBucket := core.Quantity{Units: stock.DetailQuantity, Sub: stock.DetailSubQuantity}
		if freeBucket.IsNegative(stockDivisor) || detailBucket.IsNegative(stockDivisor) {
			return nil, core.Err(fmt.Sprintf(
				"Stock resultante negativo: almacén %v producto %v presentación %v "+
					"(Quantity=%v SubQuantity=%v DetailQuantity=%v DetailSubQuantity=%v divisor=%v).",
				stock.WarehouseID, stock.ProductID, stock.PresentationID,
				stock.Quantity, stock.SubQuantity, stock.DetailQuantity, stock.DetailSubQuantity, stockDivisor))
		}
		if newStockIDs.Include(stock.ID) {
			newStocks = append(newStocks, *stock)
		} else {
			lockedStocksToUpdate = append(lockedStocksToUpdate, *stock)
		}
	}
	details := make([]ProductStockDetail, 0, len(detailByKey))
	for _, detail := range detailByKey {
		if detail.Updated == 0 {
			continue
		}
		detailDivisor := core.QuantityDivisorNone
		if parentStock := stockByID[detail.ProductStockID]; parentStock != nil {
			detailDivisor = stockDivisorOf(parentStock)
		}
		if (core.Quantity{Units: detail.Quantity, Sub: detail.SubQuantity}).IsNegative(detailDivisor) {
			return nil, core.Err(fmt.Sprintf(
				"Detalle de stock negativo: stockID %v lotID %v serial %q (Quantity=%v SubQuantity=%v divisor=%v).",
				detail.ProductStockID, detail.LotID, detail.SerialNumber,
				detail.Quantity, detail.SubQuantity, detailDivisor))
		}
		details = append(details, *detail)
	}

	core.Log("ApplyMovimientos writes:", "new stocks", len(newStocks), "updated stocks", len(lockedStocksToUpdate),
		"details", len(details), "movements", len(warehouseMovements))

	// One transaction runs on one connection, so the writes go one after the other.
	if len(newStocks) > 0 {
		if err := db.InsertIn(transaction, &newStocks); err != nil {
			return nil, core.Err("Error al crear stock V2:", err)
		}
	}
	if len(lockedStocksToUpdate) > 0 {
		stockTable := db.TableOf[ProductStock]()
		if err := db.UpdateIn(transaction, &lockedStocksToUpdate,
			// WarehouseID and Status are the delta index's pinned columns: written together.
			stockTable.WarehouseID,
			stockTable.Quantity, stockTable.SubQuantity,
			stockTable.DetailQuantity, stockTable.DetailSubQuantity,
			stockTable.DetailComputedQuantity, stockTable.DetailComputedSubQuantity,
			stockTable.SubDivisor,
			// Carried from the locked row; a divisor refinement converts them with the quantities.
			stockTable.ReservedProductionQuantity, stockTable.ReservedProductionSubQuantity,
			stockTable.BlockedQuantity, stockTable.BlockedSubQuantity,
			stockTable.LastPricesPrice, stockTable.LastPricesQuantity,
			stockTable.AverageCost, stockTable.StockValue,
			stockTable.Updated, stockTable.UpdatedBy, stockTable.Status,
		); err != nil {
			return nil, core.Err("Error al guardar stock V2:", err)
		}
	}
	if len(details) > 0 {
		detailTable := db.TableOf[ProductStockDetail]()
		if err := db.InsertUpdateIncludeIn(transaction, &details,
			func(e *ProductStockDetail) bool { return e.Created > 0 },
			db.Cols(
				// Keep the materialized view tuple consistent on updates.
				detailTable.WarehouseID,
				detailTable.Quantity, detailTable.SubQuantity,
				detailTable.Updated, detailTable.UpdatedBy, detailTable.Status,
			),
		); err != nil {
			return nil, core.Err("Error al guardar detalle de stock:", err)
		}
	}
	if err := db.UpsertIn(transaction, &warehouseMovements); err != nil {
		return nil, core.Err("Error al guardar los movimientos:", err)
	}
	return warehouseMovements, nil
}

// resolveLotIDsForMovements fills in InternalMovement.LotID for inbound movements that only carry a
// LotName: the lot (lotDate, SupplierID, Name), created when missing. The lot's unique index is what
// keeps it single: two transactions that both find it missing both insert it, the loser fails on the
// duplicate key, and its rerun finds the winner's lot here. It answers the QualityStatus of the lots
// it created: the insert holds their rows, so the caller does not lock them again.
func resolveLotIDsForMovements(transaction *db.Transaction, req *core.HandlerArgs, movements []*InternalMovement, lotDate int16) (map[int32]int8, error) {
	type lotNaturalKey struct {
		supplierID int32
		name       string
	}
	movementsByLotKey := map[lotNaturalKey][]*InternalMovement{}
	supplierIDs := core.SliceSet[int32]{}
	lotNames := core.SliceSet[string]{}
	for _, mov := range movements {
		if mov.LotID != 0 || mov.LotName == "" {
			continue
		}
		lotKey := lotNaturalKey{supplierID: mov.SupplierID, name: mov.LotName}
		movementsByLotKey[lotKey] = append(movementsByLotKey[lotKey], mov)
		supplierIDs.Add(mov.SupplierID)
		lotNames.Add(mov.LotName)
	}
	createdLotQualityByID := map[int32]int8{}
	if len(movementsByLotKey) == 0 {
		return createdLotQualityByID, nil
	}

	// One read through the unique index. The IN on both columns can return a pair nobody asked for
	// (supplier A's name with supplier B), which the map lookup below never picks.
	existingLots := []ProductStockLot{}
	query := db.Query(&existingLots)
	query.Select(query.ID, query.SupplierID, query.Name).
		CompanyID.Equals(req.User.CompanyID).
		Date.Equals(lotDate).
		SupplierID.In(supplierIDs.Values...).
		Name.In(lotNames.Values...)
	if err := query.Exec(); err != nil {
		return nil, core.Err("Error al buscar lotes existentes:", err)
	}
	lotIDByKey := map[lotNaturalKey]int32{}
	for _, lot := range existingLots {
		lotIDByKey[lotNaturalKey{supplierID: lot.SupplierID, name: lot.Name}] = lot.ID
	}

	// The missing lots, as a plain INSERT: the ORM mints their autoincrement IDs in place. The first
	// movement naming a lot is its origin, and carries its product's expiry and quality policy.
	nowTime := core.SUnixTime()
	lotsToInsert := []ProductStockLot{}
	for lotKey, lotMovements := range movementsByLotKey {
		if _, isFound := lotIDByKey[lotKey]; isFound {
			continue
		}
		originMovement := lotMovements[0]
		expirationDate := originMovement.LotExpirationDate
		if expirationDate == 0 && originMovement.LotShelfLifeDays > 0 {
			expirationDate = lotDate + originMovement.LotShelfLifeDays
		}
		lotsToInsert = append(lotsToInsert, ProductStockLot{
			CompanyID:          req.User.CompanyID,
			Date:               lotDate,
			Name:               lotKey.name,
			SupplierID:         lotKey.supplierID,
			OriginMovementType: originMovement.Type,
			OriginDocumentID:   originMovement.DocumentID,
			ExpirationDate:     expirationDate,
			QualityStatus:      core.If(originMovement.LotQualityHold, LotQualityQuarantine, LotQualityReleased),
			Created:            nowTime,
			CreatedBy:          req.User.ID,
			Updated:            nowTime,
			UpdatedBy:          req.User.ID,
			Status:             1,
		})
	}
	if len(lotsToInsert) > 0 {
		if err := db.InsertIn(transaction, &lotsToInsert); err != nil {
			return nil, core.Err("Error al crear lotes:", err)
		}
		for _, lot := range lotsToInsert {
			lotIDByKey[lotNaturalKey{supplierID: lot.SupplierID, name: lot.Name}] = lot.ID
			createdLotQualityByID[lot.ID] = lot.QualityStatus
		}
	}

	for lotKey, lotMovements := range movementsByLotKey {
		lotID := lotIDByKey[lotKey]
		if lotID == 0 {
			return nil, core.Err(fmt.Sprintf("No se pudo resolver el lote %q del proveedor %v.", lotKey.name, lotKey.supplierID))
		}
		for _, mov := range lotMovements {
			mov.LotID = lotID
		}
	}
	return createdLotQualityByID, nil
}

// lockMovementLotsIn locks the lots the movements name, after their stock rows (the order every lot
// writer follows, PostProductStockLotStatus included), and answers each lot's QualityStatus. Locked,
// so a status change cannot land between this read and the Blocked counters written from it.
func lockMovementLotsIn(transaction *db.Transaction, companyID int32, movements []*InternalMovement,
	lotQualityByID map[int32]int8) error {

	lockedLots := []ProductStockLot{}
	for _, lotID := range core.Map(movements, func(mov *InternalMovement) int32 { return mov.LotID }) {
		if _, isCreatedNow := lotQualityByID[lotID]; !isCreatedNow {
			lockedLots = append(lockedLots, ProductStockLot{CompanyID: companyID, ID: lotID})
		}
	}
	if len(lockedLots) == 0 {
		return nil
	}
	slices.SortFunc(lockedLots, func(a, b ProductStockLot) int { return cmp.Compare(a.ID, b.ID) })
	if err := db.LockByKeys(transaction, &lockedLots); err != nil {
		return core.Err("Error al obtener los lotes:", err)
	}
	for _, lot := range lockedLots {
		lotQualityByID[lot.ID] = lot.QualityStatus
	}
	return nil
}

// RecalcProductStockByMovements rebuilds ProductStockV2 + ProductStockDetail
// from the full WarehouseProductMovement ledger for one company.
// A movement with LotID==0 AND SerialNumber=="" contributes to V2.Quantity;
// any other movement contributes to the matching ProductStockDetail row, whose
// sum is mirrored into V2.DetailQuantity.
//
// It runs in one transaction that locks every stock row of the company: a live stock write waits
// until the replay commits, so none lands between the ledger read and the write of its result.
func RecalcProductStockByMovements(companyID int32) error {
	return db.RunTransaction(func(transaction *db.Transaction) error {
		return recalcProductStockIn(transaction, companyID)
	})
}

func recalcProductStockIn(transaction *db.Transaction, companyID int32) error {
	updatedTime := core.SUnixTime()

	// Start by locking the persisted stock rows, then reading their details. Rows the replay creates
	// go in newStockIDs and are written with a plain INSERT. Details exclude Created+CreatedBy, so a
	// detail still carrying Created==0 at the end was loaded from DB and is persisted as an UPDATE.
	newStockIDs := core.SliceSet[int64]{}
	stockByID := map[int64]*ProductStock{}
	detailByKey := map[string]*ProductStockDetail{}
	// Indexed by stock so a divisor refinement mid-replay can convert a row's whole detail
	// set at once; DetailSubQuantity is their absolute sum and cannot mix divisors.
	detailsByStock := map[int64][]*ProductStockDetail{}
	detailKey := func(stockID int64, lotID int32, serial string) string {
		return db.MakeKeyConcat(stockID, lotID, serial)
	}

	{
		existing := []ProductStock{}
		q := db.Query(&existing)
		q.Select(q.ID).CompanyID.Equals(companyID)
		if err := q.Exec(); err != nil {
			return core.Err("Error al obtener stock V2 previo:", err)
		}
		slices.SortFunc(existing, func(left, right ProductStock) int { return cmp.Compare(left.ID, right.ID) })
		for i := range existing {
			existing[i].CompanyID = companyID
		}
		if err := db.LockByKeys(transaction, &existing); err != nil {
			return core.Err("Error al bloquear el stock V2:", err)
		}
		for i := range existing {
			stock := &existing[i]
			// Reset so movements recompute from scratch; untouched rows will blank out below.
			// SubDivisor resets too: the ledger is the source of truth for it, so the replay
			// adopts whatever the movements carry rather than trusting the stale row.
			stock.Quantity, stock.SubQuantity = 0, 0
			stock.DetailQuantity, stock.DetailSubQuantity = 0, 0
			stock.BlockedQuantity, stock.BlockedSubQuantity = 0, 0
			stock.SubDivisor = 0
			// The value and the average are replayed from the ledger, like the quantities.
			stock.AverageCost, stock.StockValue = 0, 0
			stockByID[stock.ID] = stock
		}
	}
	{
		existing := []ProductStockDetail{}
		q := db.Query(&existing)
		q.Exclude(q.Created, q.CreatedBy, q.Updated, q.UpdatedBy).CompanyID.Equals(companyID)
		if err := q.Exec(); err != nil {
			return core.Err("Error al obtener detalle de stock previo:", err)
		}
		detailsByStockID := map[int64][]*ProductStockDetail{}
		for i := range existing {
			detail := &existing[i]
			detail.Quantity, detail.SubQuantity = 0, 0
			detailByKey[detailKey(detail.ProductStockID, detail.LotID, detail.SerialNumber)] = detail
			detailsByStockID[detail.ProductStockID] = append(detailsByStockID[detail.ProductStockID], detail)
		}
		detailsByStock = detailsByStockID
	}

	// The ledger can span a divisor refinement, so the scan cannot assume every movement for a
	// product speaks the same divisor. Each accumulator adopts the first divisor it sees and
	// then reconciles: a finer movement pulls the accumulator (and its details) forward, a
	// coarser one is converted up to it. Both directions are exact.
	//
	// balanceByStockID is each row's running free + detail total, at the row's divisor. The
	// average cost needs the units held before each inflow, and DetailQuantity is only rolled up
	// after the scan.
	balanceByStockID := map[int64]core.Quantity{}
	// What each sale still has out of each stock row, keyed (DocumentID, stockID), so a return the
	// ledger holds at 0 re-enters at the value its deliveries took, as PostSaleOrderAnnul does live.
	type saleDeliveredOut struct {
		quantity core.Quantity
		divisor  int16
		value    int64
	}
	saleOutByKey := map[[2]int64]*saleDeliveredOut{}
	// Movements whose MonetaryValue the replay filled in, written back after the scan.
	restampedMovements := []WarehouseProductMovement{}
	accumulate := func(warehouseID int32, quantity core.Quantity, movementDivisor int16,
		movement *WarehouseProductMovement) error {

		stockID := PackProductStockID(warehouseID, movement.ProductID, movement.PresentationID)
		stock := stockByID[stockID]
		if stock == nil {
			// Fresh V2 row (no historical record).
			newStockIDs.Add(stockID)
			stock = &ProductStock{
				ID:             stockID,
				CompanyID:      companyID,
				WarehouseID:    warehouseID,
				ProductID:      movement.ProductID,
				PresentationID: movement.PresentationID,
				Created:        updatedTime,
			}
			stockByID[stockID] = stock
		}

		if stock.SubDivisor <= 0 {
			stock.SubDivisor = movementDivisor
		} else if stock.SubDivisor != movementDivisor {
			if core.IsQuantityDivisorRefinement(stock.SubDivisor, movementDivisor) {
				refinedBalance, err := balanceByStockID[stockID].ConvertToDivisor(stock.SubDivisor, movementDivisor)
				if err != nil {
					return err
				}
				if err := refineStockDivisor(stock, detailsByStock[stockID],
					movementDivisor, updatedTime, 0); err != nil {
					return err
				}
				balanceByStockID[stockID] = refinedBalance
			} else {
				converted, err := quantity.ConvertToDivisor(movementDivisor, stock.SubDivisor)
				if err != nil {
					return core.Err(fmt.Sprintf(
						"Movimiento con divisor %v incompatible con el divisor %v del stock (producto %v, almacén %v).",
						movementDivisor, stock.SubDivisor, movement.ProductID, warehouseID))
				}
				quantity = converted
			}
		}

		// Same rule as ApplyMovimientos. A recorded value is a fact and is kept; only a movement
		// the ledger holds at 0 — every outflow and uncosted inflow written before StockValue
		// existed — is valued here and restamped.
		balanceBefore := balanceByStockID[stockID]
		balanceSubUnits := balanceBefore.TotalSubUnits(stock.SubDivisor)
		movedSubUnits := quantity.TotalSubUnits(stock.SubDivisor)
		recordedValue := movement.MonetaryValue
		isSaleMovement := movement.Type == StockMovementTypeSaleDelivery ||
			movement.Type == StockMovementTypeSaleAnnulmentReturn
		var saleOut *saleDeliveredOut
		if isSaleMovement {
			saleKey := [2]int64{movement.DocumentID, stockID}
			if saleOut = saleOutByKey[saleKey]; saleOut == nil {
				saleOut = &saleDeliveredOut{divisor: stock.SubDivisor}
				saleOutByKey[saleKey] = saleOut
			}
			if saleOut.divisor != stock.SubDivisor {
				// The row's divisor only gets finer, so the conversion is exact.
				converted, err := saleOut.quantity.ConvertToDivisor(saleOut.divisor, stock.SubDivisor)
				if err != nil {
					return err
				}
				saleOut.quantity, saleOut.divisor = converted, stock.SubDivisor
			}
		}

		if movedSubUnits < 0 {
			if movement.MonetaryValue <= 0 {
				movement.MonetaryValue = StockOutflowValue(stock.StockValue, balanceSubUnits, -movedSubUnits)
			}
			stock.StockValue -= int64(movement.MonetaryValue)
		} else if movedSubUnits > 0 {
			if movement.MonetaryValue <= 0 && movement.Type == StockMovementTypeSaleAnnulmentReturn {
				// A return re-enters at the value its sale's deliveries took out of this row.
				movement.MonetaryValue = StockOutflowValue(saleOut.value,
					saleOut.quantity.TotalSubUnits(stock.SubDivisor), movedSubUnits)
			}
			if movement.MonetaryValue <= 0 {
				// Any other uncosted inflow enters at the running average.
				value, err := core.QuantityValueAtUnitPrice(quantity, stock.SubDivisor, stock.AverageCost)
				if err != nil {
					return err
				}
				movement.MonetaryValue = value
			}
			stock.StockValue, stock.AverageCost = StockAfterInflow(stock.StockValue, stock.AverageCost,
				balanceSubUnits, movedSubUnits, movement.MonetaryValue, stock.SubDivisor)
		}
		if movement.MonetaryValue != recordedValue {
			movement.Period = core.PeriodOfDay(movement.Date)
			restampedMovements = append(restampedMovements, *movement)
		}
		if isSaleMovement {
			// Deliveries are negative on the ledger; what is still out is their negation.
			saleOut.quantity = saleOut.quantity.Add(quantity.Negate())
			saleOut.value += int64(core.If(movement.Type == StockMovementTypeSaleDelivery,
				movement.MonetaryValue, -movement.MonetaryValue))
		}
		balanceByStockID[stockID] = balanceBefore.Add(quantity)

		if movement.LotID == 0 && movement.SerialNumber == "" {
			next := core.Quantity{Units: stock.Quantity, Sub: stock.SubQuantity}.Add(quantity)
			stock.Quantity, stock.SubQuantity = next.Units, next.Sub
			return nil
		}
		key := detailKey(stockID, movement.LotID, movement.SerialNumber)
		detail := detailByKey[key]
		if detail == nil {
			detail = &ProductStockDetail{
				CompanyID:      companyID,
				ProductStockID: stockID,
				LotID:          movement.LotID,
				SerialNumber:   movement.SerialNumber,
				WarehouseID:    warehouseID,
				ProductID:      movement.ProductID,
				Created:        updatedTime,
			}
			detailByKey[key] = detail
			detailsByStock[stockID] = append(detailsByStock[stockID], detail)
		}
		next := core.Quantity{Units: detail.Quantity, Sub: detail.SubQuantity}.Add(quantity)
		detail.Quantity, detail.SubQuantity = next.Units, next.Sub
		return nil
	}

	// The value of an outflow depends on what the row held when it left, so the replay must run in
	// the order the movements happened. The ledger key (Date, WarehouseID, autoincrement) is not
	// that order within a day, so the rows are sorted by Created. Inside one Created second an
	// inflow goes first: the live engine never lets a row go negative, so that is the order that
	// could have happened.
	movements := []WarehouseProductMovement{}
	query := db.Query(&movements)
	query.CompanyID.Equals(companyID)
	if err := query.Exec(); err != nil {
		return core.Err("Error al obtener movimientos:", err)
	}
	isOutflow := func(movement *WarehouseProductMovement) bool {
		return movement.Quantity < 0 || (movement.Quantity == 0 && movement.SubQuantity < 0)
	}
	slices.SortStableFunc(movements, func(a, b WarehouseProductMovement) int {
		if a.Created != b.Created {
			return cmp.Compare(a.Created, b.Created)
		}
		if aOut, bOut := isOutflow(&a), isOutflow(&b); aOut != bOut {
			return core.If(aOut, 1, -1)
		}
		return cmp.Compare(a.ID, b.ID)
	})
	for i := range movements {
		movement := &movements[i]
		movementDivisor := movement.SubDivisor
		if movementDivisor <= 0 {
			movementDivisor = core.QuantityDivisorNone
		}
		moved := core.Quantity{Units: movement.Quantity, Sub: movement.SubQuantity}
		// A transfer is two rows, one per warehouse (expandStockTransfers), so each row only
		// accumulates on its own warehouse; WarehouseRefID just names the other side.
		if err := accumulate(movement.WarehouseID, moved, movementDivisor, movement); err != nil {
			return err
		}
	}

	// The lots that are not Liberado: their details roll up into Blocked too. The lots are read under
	// the company's stock locks, which every status change takes first.
	lots := []ProductStockLot{}
	lotQuery := db.Query(&lots)
	lotQuery.Select(lotQuery.ID, lotQuery.QualityStatus).CompanyID.Equals(companyID)
	if err := lotQuery.Exec(); err != nil {
		return core.Err("Error al obtener los lotes:", err)
	}
	blockedLotIDs := core.SliceSet[int32]{}
	for _, lot := range lots {
		if IsLotQualityBlocked(lot.QualityStatus) {
			blockedLotIDs.Add(lot.ID)
		}
	}

	// Roll up DetailQuantity onto each stock once every movement is accumulated. Every detail
	// under a stock is at that stock's divisor by now, so the sum is well defined.
	for _, detail := range detailByKey {
		if stock := stockByID[detail.ProductStockID]; stock != nil {
			stock.DetailQuantity += detail.Quantity
			stock.DetailSubQuantity += detail.SubQuantity
			if blockedLotIDs.Include(detail.LotID) {
				stock.BlockedQuantity += detail.Quantity
				stock.BlockedSubQuantity += detail.SubQuantity
			}
		}
		detail.Status = core.If((core.Quantity{Units: detail.Quantity, Sub: detail.SubQuantity}).IsZero(),
			int8(0), int8(1))
		detail.Updated = updatedTime
	}
	for _, stock := range stockByID {
		stock.SelfParse()
		stock.Updated = updatedTime
	}

	// Fresh stock rows go in as a plain INSERT, the locked ones as an UPDATE of the replayed columns.
	// Details route by the Created>0 marker: fresh ones go to INSERT, preloaded ones to UPDATE.
	newStocks, lockedStocksToUpdate := []ProductStock{}, []ProductStock{}
	for _, stock := range stockByID {
		if newStockIDs.Include(stock.ID) {
			newStocks = append(newStocks, *stock)
		} else {
			lockedStocksToUpdate = append(lockedStocksToUpdate, *stock)
		}
	}
	details := make([]ProductStockDetail, 0, len(detailByKey))
	for _, detail := range detailByKey {
		details = append(details, *detail)
	}

	core.Log("RecalcProductStockByMovements writes:", "new stocks", len(newStocks), "updated stocks", len(lockedStocksToUpdate),
		"details", len(details), "restamped movements", len(restampedMovements))

	if len(restampedMovements) > 0 {
		movementTable := db.TableOf[WarehouseProductMovement]()
		// The (date, product, type, sub_divisor) view and the period delta index need their key
		// columns written with any update; the movements were read whole, so they carry their stored
		// values, and Period was stamped from Date above.
		if err := db.UpdateIn(transaction, &restampedMovements, movementTable.MonetaryValue,
			movementTable.Date, movementTable.ProductID, movementTable.Type, movementTable.SubDivisor,
			movementTable.Period); err != nil {
			return core.Err("Error al guardar el valor de los movimientos:", err)
		}
	}

	if len(newStocks) > 0 {
		if err := db.InsertIn(transaction, &newStocks); err != nil {
			return core.Err("Error al crear stock V2 recalculado:", err)
		}
	}
	if len(lockedStocksToUpdate) > 0 {
		stockTable := db.TableOf[ProductStock]()
		if err := db.UpdateIn(transaction, &lockedStocksToUpdate,
			stockTable.Quantity, stockTable.SubQuantity,
			stockTable.DetailQuantity, stockTable.DetailSubQuantity,
			stockTable.BlockedQuantity, stockTable.BlockedSubQuantity,
			stockTable.SubDivisor, stockTable.AverageCost, stockTable.StockValue,
			// WarehouseID and Status are the delta index's pinned columns: written together.
			stockTable.Updated, stockTable.Status, stockTable.WarehouseID,
		); err != nil {
			return core.Err("Error al guardar stock V2 recalculado:", err)
		}
	}

	if len(details) > 0 {
		detailTable := db.TableOf[ProductStockDetail]()
		if err := db.InsertUpdateIncludeIn(transaction, &details,
			func(e *ProductStockDetail) bool { return e.Created > 0 },
			db.Cols(
				detailTable.Quantity, detailTable.SubQuantity,
				detailTable.Updated, detailTable.Status, detailTable.WarehouseID,
			),
		); err != nil {
			return core.Err("Error al guardar detalle de stock recalculado:", err)
		}
	}

	return nil
}
