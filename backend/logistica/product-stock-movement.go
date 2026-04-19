package logistica

import (
	"app/core"
	coretypes "app/core/types"
	"app/db"
	logisticaTypes "app/logistica/types"
	negocioTypes "app/negocio/types"
	"encoding/json"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"
)

// PostStockAdjustItem is one absolute "set stock to X" instruction from the client.
// ReemplazarCantidad is always true for this handler (the UI shows current stock and
// the user types the new value).
type PostStockAdjustItem struct {
	WarehouseID    int32  `json:",omitempty"`
	ProductID      int32  `json:",omitempty"`
	PresentationID int16  `json:",omitempty"`
	Quantity       int32  `json:",omitempty"`
	SubQuantity    int32  `json:",omitempty"`
	SerialNumber   string `json:",omitempty"`
	LotID          int32  `json:",omitempty"`
	LotName        string `json:",omitempty"`
	SupplierID     int32  `json:",omitempty"`
}

func PostAlmacenStock(req *core.HandlerArgs) core.HandlerResponse {
	// Handler takes absolute target quantities; each item maps to one MovimientoInterno{ReemplazarCantidad:true}.
	items := []PostStockAdjustItem{}
	if err := json.Unmarshal([]byte(*req.Body), &items); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	if len(items) == 0 {
		return req.MakeErr("No se enviaron registros.")
	}

	movimientos := make([]logisticaTypes.MovimientoInterno, 0, len(items))
	for _, item := range items {
		if item.WarehouseID == 0 || item.ProductID == 0 {
			return req.MakeErr("Hay un registro sin Almacén-ID o Producto-ID.")
		}
		movimientos = append(movimientos, logisticaTypes.MovimientoInterno{
			ReemplazarCantidad: true,
			WarehouseID:        item.WarehouseID,
			ProductoID:         item.ProductID,
			PresentacionID:     item.PresentationID,
			SerialNumber:       item.SerialNumber,
			LotID:              item.LotID,
			LotName:            item.LotName,
			SupplierID:         item.SupplierID,
			Cantidad:           item.Quantity,
			SubCantidad:        item.SubQuantity,
		})
	}

	if err := ApplyMovimientos(req, movimientos); err != nil {
		return req.MakeErr(err)
	}

	return req.MakeResponse(items)
}

func GetAlmacenMovimientos(req *core.HandlerArgs) core.HandlerResponse {

	almacenID := req.GetQueryInt("almacen-id")
	fechaInicio := req.GetQueryInt16("fecha-inicio")
	fechaFin := req.GetQueryInt16("fecha-fin")

	if almacenID == 0 || fechaInicio == 0 || fechaFin == 0 {
		return req.MakeErr("Faltan parámetros.")
	}

	type Result struct {
		Movimientos []logisticaTypes.WarehouseProductMovement
		Usuarios    []coretypes.Usuario
		Productos   []negocioTypes.Producto
	}

	result := Result{}

	query := db.Query(&result.Movimientos)

	query.CompanyID.Equals(req.Usuario.EmpresaID).
		WarehouseID.Equals(almacenID).
		Fecha.Between(fechaInicio, fechaFin).OrderDesc().Limit(1000)

	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los registros del almacén:", err)
	}

	core.Log("movimientos encontrados:", len(result.Movimientos))

	if len(result.Movimientos) == 0 {
		return req.MakeResponse(result)
	}

	usuariosSet := core.SliceSet[int32]{}
	productosSet := core.SliceSet[int32]{}

	for _, e := range result.Movimientos {
		usuariosSet.Add(e.CreatedBy)
		productosSet.Add(e.ProductoID)
	}

	errGroup := errgroup.Group{}

	errGroup.Go(func() error {
		query := db.Query(&result.Productos)
		query.Select(query.ID, query.Nombre, query.Precio).
			EmpresaID.Equals(req.Usuario.EmpresaID).
			ID.In(productosSet.Values...)
		err := query.Exec()
		if err != nil {
			err = core.Err("Error al obtener los movimientos de almacén:", err)
		}
		return err
	})

	errGroup.Go(func() error {
		query := db.Query(&result.Usuarios)
		query.Select(query.ID, query.Usuario, query.Nombres, query.Apellidos).
			EmpresaID.Equals(req.Usuario.EmpresaID).
			ID.In(usuariosSet.Values...)
		err := query.Exec()
		if err != nil {
			err = core.Err("Error al obtener los usuarios:", err)
		}
		return err
	})

	if err := errGroup.Wait(); err != nil {
		return req.MakeErr(err.Error())
	}

	return req.MakeResponse(result)
}

type GetProductosStockResult struct {
	Stocks  []logisticaTypes.ProductStockV2     `json:"stocks"`
	Details []logisticaTypes.ProductStockDetail `json:"details"`
	Lots    []logisticaTypes.ProductStockLot    `json:"lots"`
}

func GetProductosStock(req *core.HandlerArgs) core.HandlerResponse {
	// Returns V2 stock rows + their detail rows + the lot catalog.
	// `updated` enables delta-cache fetches via the `upd` field on stocks/details.
	almacenID := int32(req.GetQueryInt("almacen-id"))
	updated := int32(req.GetQueryInt("updated"))

	result := GetProductosStockResult{}

	eg := errgroup.Group{}

	eg.Go(func() error {
		query := db.Query(&result.Stocks)
		query.Select().
			CompanyID.Equals(req.Usuario.EmpresaID).
			WarehouseID.Equals(almacenID)
		if updated > 0 {
			query.Updated.GreaterEqual(updated)
		}
		return query.Exec()
	})

	eg.Go(func() error {
		// Detail rows don't have WarehouseID on the partition/cluster key path, so we filter in memory.
		details := []logisticaTypes.ProductStockDetail{}
		query := db.Query(&details)
		query.Select().CompanyID.Equals(req.Usuario.EmpresaID)
		if updated > 0 {
			query.Updated.GreaterEqual(updated).AllowFilter()
		}
		if err := query.Exec(); err != nil {
			return err
		}
		for _, detail := range details {
			if detail.WarehouseID == almacenID {
				result.Details = append(result.Details, detail)
			}
		}
		return nil
	})

	eg.Go(func() error {
		// Full lot catalog is small and shared across warehouses; a delta filter is unnecessary for now.
		query := db.Query(&result.Lots)
		query.Select().CompanyID.Equals(req.Usuario.EmpresaID)
		return query.Exec()
	})

	if err := eg.Wait(); err != nil {
		return req.MakeErr("Error al obtener los registros del almacén:", err)
	}

	return req.MakeResponse(result)
}

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

// packProductStockID mirrors the ORM's KeyIntPacking for ProductStockV2 so the
// application can compute the packed key for lookups/detail wiring without
// round-tripping through inserts.
// Schema: WarehouseID.DecimalSize(5) + ProductID.DecimalSize(9) + PresentationID.DecimalSize(4).
// Starting budget is 19 digits (see db/insert-update.go).
func packProductStockID(warehouseID int32, productID int32, presentationID int16) int64 {
	return int64(warehouseID)*1e14 + int64(productID)*1e5 + int64(presentationID)*10
}

func ApplyMovimientos(req *core.HandlerArgs, movimientos []logisticaTypes.MovimientoInterno) error {
	companyID := req.Usuario.EmpresaID
	userID := req.Usuario.ID

	companyLock := getApplyMovimientosCompanyLock(companyID)
	core.Log("ApplyMovimientos esperando lock empresa:", companyID, "movimientos:", len(movimientos))
	companyLock.Lock()
	defer companyLock.Unlock()

	// Filter out no-ops and validate lot/supplier prerequisites in one pass.
	activeMovements := make([]*logisticaTypes.MovimientoInterno, 0, len(movimientos))
	for i := range movimientos {
		mov := &movimientos[i]
		if mov.Cantidad == 0 && mov.SubCantidad == 0 {
			continue
		}
		if mov.WarehouseID == 0 || mov.ProductoID == 0 {
			return core.Err("Movimiento inválido: falta WarehouseID o ProductoID.")
		}
		// Outbound (Cantidad < 0) must ship a resolved LotID; name-based lookup is inbound-only.
		if mov.LotID == 0 && mov.LotName != "" && mov.Cantidad < 0 {
			return core.Err(fmt.Sprintf("Movimiento con Lote %q sin LotID para salida (producto %v, almacén %v).",
				mov.LotName, mov.ProductoID, mov.WarehouseID))
		}
		// Inbound lot-by-name needs SupplierID for the dedup hash.
		if mov.LotID == 0 && mov.LotName != "" && mov.SupplierID == 0 {
			return core.Err(fmt.Sprintf("Movimiento con Lote %q sin SupplierID (producto %v, almacén %v).",
				mov.LotName, mov.ProductoID, mov.WarehouseID))
		}
		activeMovements = append(activeMovements, mov)
	}
	if len(activeMovements) == 0 {
		return nil
	}

	// Resolve missing LotIDs for inbound lot-by-name movements.
	if err := resolveLotIDsForMovements(req, activeMovements, req.EffectiveFechaUnix()); err != nil {
		return err
	}

	// Compute V2 packed IDs per movement and bucket them for preload.
	stockIDByMovement := make([]int64, len(activeMovements))
	stockIDSet := core.SliceSet[int64]{}
	stockIDsWithDetails := core.SliceSet[int64]{}
	for i, mov := range activeMovements {
		id := packProductStockID(mov.WarehouseID, mov.ProductoID, mov.PresentacionID)
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
	stockByID := map[int64]*logisticaTypes.ProductStockV2{}
	detailByKey := map[string]*logisticaTypes.ProductStockDetail{}
	detailKey := func(stockID int64, lotID int32, serial string) string {
		return db.MakeKeyConcat(stockID, lotID, serial)
	}

	preloadGroup := errgroup.Group{}
	preloadGroup.Go(func() error {
		existing := []logisticaTypes.ProductStockV2{}
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
			existing := []logisticaTypes.ProductStockDetail{}
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

	updatedTime := req.EffectiveSUnixTime()
	fechaUnix := req.EffectiveFechaUnix()

	// Build ledger rows and mutate V2/Detail in place.
	warehouseMovements := make([]logisticaTypes.WarehouseProductMovement, 0, len(activeMovements))
	for i, mov := range activeMovements {
		stockID := stockIDByMovement[i]
		stock := stockByID[stockID]
		if stock == nil {
			// New V2 row: stamp Created/CreatedBy so the partition step can flag it for insert.
			stock = &logisticaTypes.ProductStockV2{
				ID:             stockID,
				CompanyID:      companyID,
				WarehouseID:    mov.WarehouseID,
				ProductID:      mov.ProductoID,
				PresentationID: mov.PresentacionID,
				Created:        updatedTime,
				CreatedBy:      userID,
			}
			stockByID[stockID] = stock
		}

		movement := logisticaTypes.WarehouseProductMovement{
			DocumentID:     mov.DocumentID,
			CompanyID:      companyID,
			WarehouseID:    mov.WarehouseID,
			ProductoID:     mov.ProductoID,
			PresentacionID: mov.PresentacionID,
			SerialNumber:   mov.SerialNumber,
			LotID:          mov.LotID,
			Tipo:           core.Coalesce(mov.Tipo, core.If(mov.Cantidad > 0, int8(1), int8(2))),
			Fecha:          fechaUnix,
			Created:        updatedTime,
			CreatedBy:      userID,
		}

		if mov.HasDetail() {
			key := detailKey(stockID, mov.LotID, mov.SerialNumber)
			detail := detailByKey[key]
			if detail == nil {
				detail = &logisticaTypes.ProductStockDetail{
					CompanyID:      companyID,
					ProductStockID: stockID,
					LotID:          mov.LotID,
					SerialNumber:   mov.SerialNumber,
					WarehouseID:    mov.WarehouseID,
					ProductID:      mov.ProductoID,
					Created:        updatedTime,
					CreatedBy:      userID,
				}
				detailByKey[key] = detail
			}

			prevQuantity, prevSubQuantity := detail.Quantity, detail.SubQuantity
			if mov.ReemplazarCantidad {
				movement.Quantity = mov.Cantidad - prevQuantity
				movement.SubQuantity = mov.SubCantidad - prevSubQuantity
				detail.Quantity = mov.Cantidad
				detail.SubQuantity = mov.SubCantidad
			} else {
				movement.Quantity = mov.Cantidad
				movement.SubQuantity = mov.SubCantidad
				detail.Quantity = prevQuantity + mov.Cantidad
				detail.SubQuantity = prevSubQuantity + mov.SubCantidad
			}
			// Stamp Updated/UpdatedBy so the partition step recognises this detail as dirty.
			detail.Updated = updatedTime
			detail.UpdatedBy = userID
			detail.Status = core.If(detail.Quantity == 0 && detail.SubQuantity == 0, int8(0), int8(1))
		} else {
			// Free bucket: mutate V2.Quantity only, leave DetailQuantity for the final re-sum pass.
			prevQuantity, prevSubQuantity := stock.Quantity, stock.SubQuantity
			if mov.ReemplazarCantidad {
				movement.Quantity = mov.Cantidad - prevQuantity
				movement.SubQuantity = mov.SubCantidad - prevSubQuantity
				stock.Quantity = mov.Cantidad
				stock.SubQuantity = mov.SubCantidad
			} else {
				movement.Quantity = mov.Cantidad
				movement.SubQuantity = mov.SubCantidad
				stock.Quantity = prevQuantity + mov.Cantidad
				stock.SubQuantity = prevSubQuantity + mov.SubCantidad
			}
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
	stocks := make([]logisticaTypes.ProductStockV2, 0, len(stockByID))
	for _, stock := range stockByID {
		if stock.Quantity < 0 || stock.DetailQuantity < 0 {
			return core.Err(fmt.Sprintf(
				"Stock resultante negativo: almacén %v producto %v presentación %v (Quantity=%v DetailQuantity=%v).",
				stock.WarehouseID, stock.ProductID, stock.PresentationID, stock.Quantity, stock.DetailQuantity))
		}
		stocks = append(stocks, *stock)
	}
	details := make([]logisticaTypes.ProductStockDetail, 0, len(detailByKey))
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
			stockTable := db.Table[logisticaTypes.ProductStockV2]()
			if err := db.InsertUpdateInclude(&stocks,
				func(e *logisticaTypes.ProductStockV2) bool { return e.Created > 0 },
				[]db.Coln{
					stockTable.Quantity, stockTable.SubQuantity,
					stockTable.DetailQuantity, stockTable.DetailSubQuantity,
					stockTable.Updated, stockTable.UpdatedBy, stockTable.Status,
				},
			); err != nil {
				return core.Err("Error al guardar stock V2:", err)
			}
			return nil
		})
	}
	if len(details) > 0 {
		writeGroup.Go(func() error {
			detailTable := db.Table[logisticaTypes.ProductStockDetail]()
			if err := db.InsertUpdateInclude(&details,
				func(e *logisticaTypes.ProductStockDetail) bool { return e.Created > 0 },
				[]db.Coln{
					detailTable.Quantity, detailTable.SubQuantity,
					detailTable.Updated, detailTable.UpdatedBy, detailTable.Status,
				},
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

// resolveLotIDsForMovements fills in MovimientoInterno.LotID for inbound movements
// that only carry a LotName. It dedups by Hash(today, SupplierID, Name) against
// ProductStockLot and creates any missing lot rows in one batch.
func resolveLotIDsForMovements(req *core.HandlerArgs, movements []*logisticaTypes.MovimientoInterno, lotDate int16) error {
	// Group movements by hash so we only touch each unique (date, supplier, name) lot once.
	type lotLookupKey struct {
		hash       string
		supplierID int32
		name       string
	}
	movementsByHash := map[string][]*logisticaTypes.MovimientoInterno{}
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

	// Look up existing lots by the dedup hash index.
	existingLots := []logisticaTypes.ProductStockLot{}
	q := db.Query(&existingLots)
	q.Select().
		CompanyID.Equals(req.Usuario.EmpresaID).
		Hash.In(hashes...)
	if err := q.Exec(); err != nil {
		return core.Err("Error al buscar lotes existentes:", err)
	}

	lotIDByHash := map[string]int32{}
	for _, lot := range existingLots {
		lotIDByHash[lot.Hash] = lot.ID
	}

	// Insert any missing lots in one batch. The ORM assigns autoincrement IDs in-place.
	lotsToInsert := []logisticaTypes.ProductStockLot{}
	insertedHashOrder := []string{}
	for hash, key := range hashToKey {
		if _, found := lotIDByHash[hash]; found {
			continue
		}
		lotsToInsert = append(lotsToInsert, logisticaTypes.ProductStockLot{
			CompanyID:  req.Usuario.EmpresaID,
			Date:       lotDate,
			Name:       key.name,
			SupplierID: key.supplierID,
			Created:    req.EffectiveSUnixTime(),
			CreatedBy:  req.Usuario.ID,
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
