// Stock movement engine. This is business logic in a `types` package on purpose: `sales` and
// `accounting` both have to write stock, and a module body may not import another module body
// (see backend/docs/MODULE_BOUNDARIES.md). Living here is what lets them call it while importing
// only `app/logistics/types`.
//
// ApplyMovimientos and RecalcProductStockByMovements share one per-company write lock and the
// packed-key helper, which is why the recalc moved with the engine rather than staying behind and
// forcing both to be exported.

package types

import (
	"app/core"
	"app/db"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"
)

var applyMovimientosLockByCompany = map[int32]*sync.Mutex{}
var applyMovimientosLockMapMu sync.Mutex

func getApplyMovimientosCompanyLock(companyID int32) *sync.Mutex {
	// One mutex per company so stock writes for the same tenant stay serialized.
	applyMovimientosLockMapMu.Lock()
	companyLock := applyMovimientosLockByCompany[companyID]
	if companyLock == nil {
		companyLock = &sync.Mutex{}
		applyMovimientosLockByCompany[companyID] = companyLock
	}
	applyMovimientosLockMapMu.Unlock()
	return companyLock
}

// PackProductStockID mirrors the ORM's KeyIntPacking for ProductStockV2 so the
// application can compute the packed key for lookups/detail wiring without
// round-tripping through inserts.
// Schema: WarehouseID.DecimalSize(5) + ProductID.DecimalSize(9) + PresentationID.DecimalSize(4).
// Starting budget is 19 digits (see db/insert-update.go).
func PackProductStockID(warehouseID int32, productID int32, presentationID int16) int64 {
	return int64(warehouseID)*1e14 + int64(productID)*1e5 + int64(presentationID)*10
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

func ApplyMovimientos(req *core.HandlerArgs, movimientos []InternalMovement) error {
	companyID := req.User.CompanyID
	userID := req.User.ID

	companyLock := getApplyMovimientosCompanyLock(companyID)
	core.Log("ApplyMovimientos esperando lock company:", companyID, "movimientos:", len(movimientos))
	companyLock.Lock()
	defer companyLock.Unlock()

	// Filter out no-ops and validate lot/supplier prerequisites in one pass.
	activeMovements := make([]*InternalMovement, 0, len(movimientos))
	for i := range movimientos {
		mov := &movimientos[i]
		if mov.Quantity == 0 && mov.SubQuantity == 0 {
			continue
		}
		if mov.WarehouseID == 0 || mov.ProductID == 0 {
			return core.Err("Movimiento inválido: falta WarehouseID o ProductID.")
		}
		// Outbound (Quantity < 0) must ship a resolved LotID; name-based lookup is inbound-only.
		if mov.LotID == 0 && mov.LotName != "" && mov.Quantity < 0 {
			return core.Err(fmt.Sprintf("Movimiento con Lote %q sin LotID para salida (product %v, almacén %v).",
				mov.LotName, mov.ProductID, mov.WarehouseID))
		}
		// SupplierID == 0 is allowed for manual stock-adjustment lots; the dedup hash
		// becomes (today, 0, name) which stays consistent within the day.
		activeMovements = append(activeMovements, mov)
	}
	if len(activeMovements) == 0 {
		return nil
	}

	// Resolve missing LotIDs for inbound lot-by-name movements.
	if err := resolveLotIDsForMovements(req, activeMovements, core.FechaUnix()); err != nil {
		return err
	}

	// Compute V2 packed IDs per movement and bucket them for preload.
	stockIDByMovement := make([]int64, len(activeMovements))
	stockIDSet := core.SliceSet[int64]{}
	stockIDsWithDetails := core.SliceSet[int64]{}
	for i, mov := range activeMovements {
		id := PackProductStockID(mov.WarehouseID, mov.ProductID, mov.PresentationID)
		stockIDByMovement[i] = id
		stockIDSet.Add(id)
		if mov.HasDetail() {
			stockIDsWithDetails.Add(id)
		}
	}

	// Preload V2 rows and detail rows in parallel. Each goroutine owns its own map
	// so no shared state is mutated concurrently.
	// - V2 preload excludes Created/CreatedBy so loaded rows carry them as 0, flagging UPDATE at write time.
	// - Detail preload also excludes Updated/UpdatedBy so untouched preloaded details stay at Updated==0
	//   and get skipped by the write step.
	stockByID := map[int64]*ProductStock{}
	detailByKey := map[string]*ProductStockDetail{}
	detailKey := func(stockID int64, lotID int32, serial string) string {
		return db.MakeKeyConcat(stockID, lotID, serial)
	}

	preloadGroup := errgroup.Group{}
	preloadGroup.Go(func() error {
		existing := []ProductStock{}
		q := db.Query(&existing)
		q.Exclude(q.Created, q.CreatedBy).
			CompanyID.Equals(companyID).
			ID.In(stockIDSet.Values...)
		if err := q.Exec(); err != nil {
			return core.Err("Error al obtener stock V2 previo:", err)
		}
		for i := range existing {
			stockByID[existing[i].ID] = &existing[i]
		}
		return nil
	})
	if !stockIDsWithDetails.IsEmpty() {
		preloadGroup.Go(func() error {
			existing := []ProductStockDetail{}
			q := db.Query(&existing)
			q.Exclude(q.Created, q.CreatedBy, q.Updated, q.UpdatedBy).
				CompanyID.Equals(companyID).
				ProductStockID.In(stockIDsWithDetails.Values...)
			if err := q.Exec(); err != nil {
				return core.Err("Error al obtener detalle de stock previo:", err)
			}
			for i := range existing {
				row := &existing[i]
				detailByKey[detailKey(row.ProductStockID, row.LotID, row.SerialNumber)] = row
			}
			return nil
		})
	}
	if err := preloadGroup.Wait(); err != nil {
		return err
	}

	updatedTime := core.SUnixTime()
	dateUnix := core.FechaUnix()

	// Build ledger rows and mutate V2/Detail in place.
	warehouseMovements := make([]WarehouseProductMovement, 0, len(activeMovements))
	for i, mov := range activeMovements {
		stockID := stockIDByMovement[i]
		stock := stockByID[stockID]
		if stock == nil {
			// New V2 row: stamp Created/CreatedBy so the partition step can flag it for insert.
			stock = &ProductStock{
				ID:             stockID,
				CompanyID:      companyID,
				WarehouseID:    mov.WarehouseID,
				ProductID:      mov.ProductID,
				PresentationID: mov.PresentationID,
				Created:        updatedTime,
				CreatedBy:      userID,
			}
			stockByID[stockID] = stock
		}

		movement := WarehouseProductMovement{
			DocumentID:     mov.DocumentID,
			CompanyID:      companyID,
			WarehouseID:    mov.WarehouseID,
			ProductID:      mov.ProductID,
			PresentationID: mov.PresentationID,
			SerialNumber:   mov.SerialNumber,
			LotID:          mov.LotID,
			Type:           core.Coalesce(mov.Type, core.If(mov.Quantity > 0, int8(1), int8(2))),
			Date:           dateUnix,
			Created:        updatedTime,
			CreatedBy:      userID,
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

			prevQuantity, prevSubQuantity := detail.Quantity, detail.SubQuantity
			if mov.ReplaceQuantity {
				movement.Quantity = mov.Quantity - prevQuantity
				movement.SubQuantity = mov.SubQuantity - prevSubQuantity
				detail.Quantity = mov.Quantity
				detail.SubQuantity = mov.SubQuantity
			} else {
				movement.Quantity = mov.Quantity
				movement.SubQuantity = mov.SubQuantity
				detail.Quantity = prevQuantity + mov.Quantity
				detail.SubQuantity = prevSubQuantity + mov.SubQuantity
			}
			// Stamp Updated/UpdatedBy so the partition step recognises this detail as dirty.
			detail.Updated = updatedTime
			detail.UpdatedBy = userID
			detail.Status = core.If(detail.Quantity == 0 && detail.SubQuantity == 0, int8(0), int8(1))
		} else {
			// Free bucket: mutate V2.Quantity only, leave DetailQuantity for the final re-sum pass.
			prevQuantity, prevSubQuantity := stock.Quantity, stock.SubQuantity
			if mov.ReplaceQuantity {
				movement.Quantity = mov.Quantity - prevQuantity
				movement.SubQuantity = mov.SubQuantity - prevSubQuantity
				stock.Quantity = mov.Quantity
				stock.SubQuantity = mov.SubQuantity
			} else {
				movement.Quantity = mov.Quantity
				movement.SubQuantity = mov.SubQuantity
				stock.Quantity = prevQuantity + mov.Quantity
				stock.SubQuantity = prevSubQuantity + mov.SubQuantity
			}
		}

		appendProductStockLastPrice(stock, movement.Quantity, mov.Price)
		if mov.Price > 0 {
			// Persist the unit price on the ledger as total line value for later reports.
			movement.MonetaryValue = movement.Quantity * mov.Price
		}
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

	// Backfill WarehouseQuantity on each ledger row now that DetailQuantity is final.
	for i := range warehouseMovements {
		if stock := stockByID[stockIDByMovement[i]]; stock != nil {
			warehouseMovements[i].WarehouseQuantity = stock.Quantity + stock.DetailQuantity
		}
	}

	// Flatten to value slices. Validate non-negative balances at the same time.
	// Untouched preloaded details (Updated==0) are skipped so InsertUpdateInclude only sees dirty rows.
	stocks := make([]ProductStock, 0, len(stockByID))
	for _, stock := range stockByID {
		if stock.Quantity < 0 || stock.DetailQuantity < 0 {
			return core.Err(fmt.Sprintf(
				"Stock resultante negativo: almacén %v product %v presentación %v (Quantity=%v DetailQuantity=%v).",
				stock.WarehouseID, stock.ProductID, stock.PresentationID, stock.Quantity, stock.DetailQuantity))
		}
		stocks = append(stocks, *stock)
	}
	details := make([]ProductStockDetail, 0, len(detailByKey))
	for _, detail := range detailByKey {
		if detail.Updated == 0 {
			continue
		}
		if detail.Quantity < 0 {
			return core.Err(fmt.Sprintf(
				"Detalle de stock negativo: stockID %v lotID %v serial %q (Quantity=%v).",
				detail.ProductStockID, detail.LotID, detail.SerialNumber, detail.Quantity))
		}
		details = append(details, *detail)
	}

	core.Log("ApplyMovimientos writes:",
		"stocks", len(stocks), "details", len(details), "movements", len(warehouseMovements))

	// Three writes run in parallel — the stock, detail, and movement tables don't share state,
	// so a failure in one doesn't affect the others' progress. InsertUpdateInclude routes each
	// record to INSERT or UPDATE by the Created>0 predicate.
	writeGroup := errgroup.Group{}
	if len(stocks) > 0 {
		writeGroup.Go(func() error {
			stockTable := db.TableOf[ProductStock]()
			if err := db.InsertUpdateInclude(&stocks,
				func(e *ProductStock) bool { return e.Created > 0 },
				db.Cols(
					// Keep the materialized view tuple consistent on updates.
					stockTable.WarehouseID,
					stockTable.Quantity, stockTable.SubQuantity,
					stockTable.DetailQuantity, stockTable.DetailSubQuantity,
					stockTable.LastPricesPrice, stockTable.LastPricesQuantity,
					stockTable.Updated, stockTable.UpdatedBy, stockTable.Status,
				),
			); err != nil {
				return core.Err("Error al guardar stock V2:", err)
			}
			return nil
		})
	}
	if len(details) > 0 {
		writeGroup.Go(func() error {
			detailTable := db.TableOf[ProductStockDetail]()
			if err := db.InsertUpdateInclude(&details,
				func(e *ProductStockDetail) bool { return e.Created > 0 },
				db.Cols(
					// Keep the materialized view tuple consistent on updates.
					detailTable.WarehouseID,
					detailTable.Quantity, detailTable.SubQuantity,
					detailTable.Updated, detailTable.UpdatedBy, detailTable.Status,
				),
			); err != nil {
				return core.Err("Error al guardar detalle de stock:", err)
			}
			return nil
		})
	}
	writeGroup.Go(func() error {
		if err := db.Insert(&warehouseMovements); err != nil {
			return core.Err("Error al guardar los movimientos:", err)
		}
		return nil
	})
	return writeGroup.Wait()
}

// resolveLotIDsForMovements fills in InternalMovement.LotID for inbound movements
// that only carry a LotName. It dedups by Hash(today, SupplierID, Name) against
// ProductStockLot and creates any missing lot rows in one batch.
func resolveLotIDsForMovements(req *core.HandlerArgs, movements []*InternalMovement, lotDate int16) error {
	// Group movements by hash so we only touch each unique (date, supplier, name) lot once.
	type lotLookupKey struct {
		hash       string
		supplierID int32
		name       string
	}
	movementsByHash := map[string][]*InternalMovement{}
	hashToKey := map[string]lotLookupKey{}

	for _, mov := range movements {
		if mov.LotID != 0 || mov.LotName == "" {
			continue
		}
		hash := db.MakeKeyConcat(lotDate, mov.SupplierID, mov.LotName)
		movementsByHash[hash] = append(movementsByHash[hash], mov)
		hashToKey[hash] = lotLookupKey{hash: hash, supplierID: mov.SupplierID, name: mov.LotName}
	}
	if len(movementsByHash) == 0 {
		return nil
	}

	hashes := make([]string, 0, len(movementsByHash))
	for hash := range movementsByHash {
		hashes = append(hashes, hash)
	}

	// Look up existing lots by the dedup hash index, one hash per query and in parallel.
	// Hash is a global index, so its capability signature carries no partition prefix and a
	// CompanyID + Hash query cannot route through it: Scylla then sees a plain restriction on a
	// non-key column, which it only accepts for a single value. Batching the hashes into one IN
	// would need ALLOW FILTERING — a full partition scan of every lot the tenant ever created.
	lotsByHashIndex := make([][]ProductStockLot, len(hashes))
	lotLookupGroup := errgroup.Group{}
	lotLookupGroup.SetLimit(8)

	for hashIndex, hash := range hashes {
		lotLookupGroup.Go(func() error {
			query := db.Query(&lotsByHashIndex[hashIndex])
			query.Select().
				CompanyID.Equals(req.User.CompanyID).
				Hash.Equals(hash)
			return query.Exec()
		})
	}
	if err := lotLookupGroup.Wait(); err != nil {
		return core.Err("Error al buscar lotes existentes:", err)
	}

	lotIDByHash := map[string]int32{}
	for _, lotsForHash := range lotsByHashIndex {
		for _, lot := range lotsForHash {
			lotIDByHash[lot.Hash] = lot.ID
		}
	}

	// Insert any missing lots in one batch. The ORM assigns autoincrement IDs in-place.
	lotsToInsert := []ProductStockLot{}
	insertedHashOrder := []string{}
	for hash, key := range hashToKey {
		if _, found := lotIDByHash[hash]; found {
			continue
		}
		lotsToInsert = append(lotsToInsert, ProductStockLot{
			CompanyID:  req.User.CompanyID,
			Date:       lotDate,
			Name:       key.name,
			SupplierID: key.supplierID,
			Created:    core.SUnixTime(),
			CreatedBy:  req.User.ID,
		})
		insertedHashOrder = append(insertedHashOrder, hash)
	}
	if len(lotsToInsert) > 0 {
		if err := db.Insert(&lotsToInsert); err != nil {
			return core.Err("Error al crear lotes:", err)
		}
		for i, hash := range insertedHashOrder {
			lotIDByHash[hash] = lotsToInsert[i].ID
		}
	}

	// Fan the resolved IDs back out to every movement that referenced them.
	for hash, movs := range movementsByHash {
		lotID := lotIDByHash[hash]
		if lotID == 0 {
			return core.Err(fmt.Sprintf("No se pudo resolver el LotID para hash %v", hash))
		}
		for _, mov := range movs {
			mov.LotID = lotID
		}
	}
	return nil
}

// RecalcProductStockByMovements rebuilds ProductStockV2 + ProductStockDetail
// from the full WarehouseProductMovement ledger for one company.
// A movement with LotID==0 AND SerialNumber=="" contributes to V2.Quantity;
// any other movement contributes to the matching ProductStockDetail row, whose
// sum is mirrored into V2.DetailQuantity.
func RecalcProductStockByMovements(companyID int32) error {
	companyLock := getApplyMovimientosCompanyLock(companyID)

	core.Log("RecalcProductStockByMovements esperando lock company:", companyID)
	companyLock.Lock()
	defer companyLock.Unlock()

	updatedTime := core.SUnixTime()

	// Start by loading the persisted rows. Excluding Created+CreatedBy (and Updated/UpdatedBy on
	// details) means any in-memory row still carrying Created==0 at the end of the pass was
	// loaded from DB and must be persisted as an UPDATE; Created>0 flags INSERT.
	stockByID := map[int64]*ProductStock{}
	detailByKey := map[string]*ProductStockDetail{}
	detailKey := func(stockID int64, lotID int32, serial string) string {
		return db.MakeKeyConcat(stockID, lotID, serial)
	}

	{
		existing := []ProductStock{}
		q := db.Query(&existing)
		q.Exclude(q.Created, q.CreatedBy).CompanyID.Equals(companyID)
		if err := q.Exec(); err != nil {
			return core.Err("Error al obtener stock V2 previo:", err)
		}
		for i := range existing {
			stock := &existing[i]
			// Reset so movements recompute from scratch; untouched rows will blank out below.
			stock.Quantity, stock.SubQuantity = 0, 0
			stock.DetailQuantity, stock.DetailSubQuantity = 0, 0
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
		for i := range existing {
			detail := &existing[i]
			detail.Quantity, detail.SubQuantity = 0, 0
			detailByKey[detailKey(detail.ProductStockID, detail.LotID, detail.SerialNumber)] = detail
		}
	}

	accumulate := func(warehouseID int32, quantity int32, subQuantity int32, movement *WarehouseProductMovement) {
		stockID := PackProductStockID(warehouseID, movement.ProductID, movement.PresentationID)
		stock := stockByID[stockID]
		if stock == nil {
			// Fresh V2 row (no historical record): Created stamps it as INSERT at write time.
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
		if movement.LotID == 0 && movement.SerialNumber == "" {
			stock.Quantity += quantity
			stock.SubQuantity += subQuantity
			return
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
		}
		detail.Quantity += quantity
		detail.SubQuantity += subQuantity
	}

	query := db.Query(&[]WarehouseProductMovement{})
	query.CompanyID.Equals(companyID)
	if err := query.ExecScan(func(movement *WarehouseProductMovement) bool {
		accumulate(movement.WarehouseID, movement.Quantity, movement.SubQuantity, movement)
		if movement.WarehouseRefID > 0 {
			// Transfers mirror an outbound leg on the source warehouse.
			accumulate(movement.WarehouseRefID, -movement.Quantity, -movement.SubQuantity, movement)
		}
		return true
	}); err != nil {
		return core.Err("Error al escanear movimientos:", err)
	}

	// Roll up DetailQuantity onto each stock once every movement is accumulated.
	for _, detail := range detailByKey {
		if stock := stockByID[detail.ProductStockID]; stock != nil {
			stock.DetailQuantity += detail.Quantity
			stock.DetailSubQuantity += detail.SubQuantity
		}
		detail.Status = core.If(detail.Quantity == 0 && detail.SubQuantity == 0, int8(0), int8(1))
		detail.Updated = updatedTime
	}
	for _, stock := range stockByID {
		stock.Status = core.If(stock.Quantity == 0 && stock.DetailQuantity == 0, int8(0), int8(1))
		stock.Updated = updatedTime
	}

	// Flatten and let InsertUpdateInclude route by the Created>0 marker: fresh rows go to INSERT,
	// preloaded rows go to UPDATE (touching only the listed columns).
	stocks := make([]ProductStock, 0, len(stockByID))
	for _, stock := range stockByID {
		stocks = append(stocks, *stock)
	}
	details := make([]ProductStockDetail, 0, len(detailByKey))
	for _, detail := range detailByKey {
		details = append(details, *detail)
	}

	core.Log("RecalcProductStockByMovements writes:", "stocks", len(stocks), "details", len(details))

	if len(stocks) > 0 {
		stockTable := db.TableOf[ProductStock]()
		if err := db.InsertUpdateInclude(&stocks,
			func(e *ProductStock) bool { return e.Created > 0 },
			db.Cols(
				stockTable.Quantity, stockTable.SubQuantity,
				stockTable.DetailQuantity, stockTable.DetailSubQuantity,
				stockTable.Updated, stockTable.Status,
			),
		); err != nil {
			return core.Err("Error al guardar stock V2 recalculado:", err)
		}
	}

	if len(details) > 0 {
		detailTable := db.TableOf[ProductStockDetail]()
		if err := db.InsertUpdateInclude(&details,
			func(e *ProductStockDetail) bool { return e.Created > 0 },
			db.Cols(
				detailTable.Quantity, detailTable.SubQuantity,
				detailTable.Updated, detailTable.Status,
			),
		); err != nil {
			return core.Err("Error al guardar detalle de stock recalculado:", err)
		}
	}

	return nil
}
