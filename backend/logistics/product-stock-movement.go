package logistics

import (
	"app/cloud"
	"app/core"
	"app/db"
	"app/logistics/types"
	production "app/production/types"
	"encoding/json"
	"fmt"
	"slices"

	"golang.org/x/sync/errgroup"
)

// PostStockAdjustItem is one absolute "set stock to X" instruction from the client.
// ReemplazarCantidad is always true for this handler (the UI shows current stock and
// the user types the new value).
//
// Lot resolution:
//   - LotID > 0          → reference an existing lot.
//   - LotID == 0 + LotCode → resolve / create a lot via Hash(today, SupplierID, LotCode).
//     SupplierID is optional here (manual adjustments have no supplier context).
type PostStockAdjustItem struct {
	WarehouseID    int32  `cb:"1"`
	ProductID      int32  `cb:"2"`
	PresentationID int16  `cb:"3"`
	Quantity       int32  `cb:"4"`
	SubQuantity    int32  `cb:"5"`
	SerialNumber   string `cb:"6"`
	LotID          int32  `cb:"7"`
	LotCode        string `cb:"8"`
	SupplierID     int32  `cb:"9"`
	// UnitCost is the purchase cost of one unit of the stock this adjustment adds, in PEN cents,
	// taken as typed: a manual entry has no comprobante, so there is no IGV to recover from it.
	// Only an increase reads it. A purchase-order reception ignores it and costs from the order.
	UnitCost int32 `cb:"10"`
	// Type is the reason the user chose for the change, one of manualStockAdjustmentTypes. The
	// page picks it per direction (one for the increases, one for the decreases); the engine
	// refuses it if the delta turns out to go the other way.
	Type types.StockMovementType `cb:"11"`
	// LotExpirationDate is the expiry (UnixDay) of the lot a LotCode creates; 0 takes the
	// product's shelf life.
	LotExpirationDate int16 `cb:"12"`
}

// manualStockAdjustmentTypes are the reasons a user may give a manual stock change. Every other
// type is written by its own document.
var manualStockAdjustmentTypes = []types.StockMovementType{
	types.StockMovementTypeManualEntry, types.StockMovementTypeOpeningStock,
	types.StockMovementTypeManualExit, types.StockMovementTypeShrinkage, types.StockMovementTypeInternalConsumption,
}

// The company flag that governs manual stock adjustments (backend/company_flags.toml).
const companyFlagBlockManualStockInbound = int16(6) // stock only enters through a purchase order

// loadStockItemProducts reads what a stock entry needs of every product it touches: the current
// sub-unit divisor and the lot policy (docs/MANUFACTURING_PLAN.md §10).
func loadStockItemProducts(companyID int32, items []PostStockAdjustItem) (map[int32]production.Product, error) {
	productByID := map[int32]production.Product{}
	productIDs := core.Map(items, func(item PostStockAdjustItem) int32 { return item.ProductID })
	if len(productIDs) == 0 {
		return productByID, nil
	}
	products := []production.Product{}
	query := db.Query(&products)
	query.Select(query.ID, query.Name, query.SbuQuantity, query.TracksLots, query.ShelfLifeDays, query.QualityHoldOnEntry).
		CompanyID.Equals(companyID).
		ID.In(productIDs...)
	if err := query.Exec(); err != nil {
		return nil, core.Err("Error al obtener los productos:", err)
	}
	for _, product := range products {
		productByID[product.ID] = product
	}
	return productByID, nil
}

// productSubDivisor is a product's sub-unit divisor; one with no sub-unit has core.QuantityDivisorNone.
func productSubDivisor(product production.Product) int16 {
	return max(product.SbuQuantity, core.QuantityDivisorNone)
}

// withProductLotPolicy stamps on a movement what a lot it creates takes from its product: the shelf
// life its expiry defaults to and whether it is born in Cuarentena.
func withProductLotPolicy(movement types.InternalMovement, product production.Product, lotExpirationDate int16) types.InternalMovement {
	movement.LotExpirationDate = lotExpirationDate
	movement.LotShelfLifeDays = product.ShelfLifeDays
	movement.LotQualityHold = product.QualityHoldOnEntry
	return movement
}

func PostAlmacenStock(req *core.HandlerArgs) core.HandlerResponse {
	// Handler takes absolute target quantities; each item maps to one InternalMovement{ReemplazarCantidad:true}.
	items := []PostStockAdjustItem{}
	if err := json.Unmarshal([]byte(*req.Body), &items); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	if len(items) == 0 {
		return req.MakeErr("No se enviaron registros.")
	}

	// The divisor is resolved from the catalog, not taken from the client: a movement that
	// carries sub-units without one is rejected by the stock engine, and guessing it would be
	// silently wrong for exactly the products this exists for.
	productByID, err := loadStockItemProducts(req.User.CompanyID, items)
	if err != nil {
		return req.MakeErr(err)
	}

	// The page enforces this too, but only the engine knows whether a "set stock to X" item is
	// an increase, so the flag travels on each movement and is judged there. So is the cost an
	// increase needs when its row has no AverageCost.
	companyConfig, err := cloud.LoadCompanyConfig(req.User.CompanyID)
	if err != nil {
		return req.MakeErr("Error al leer la configuración de la empresa:", err)
	}
	rejectManualInbound := core.HasCompanyFlag(companyConfig.Company.Flags, companyFlagBlockManualStockInbound)

	movimientos := make([]types.InternalMovement, 0, len(items))
	for _, item := range items {
		if item.WarehouseID == 0 || item.ProductID == 0 {
			return req.MakeErr("Hay un registro sin Almacén-ID o Producto-ID.")
		}
		if item.UnitCost < 0 {
			return req.MakeErr(fmt.Sprintf("El costo del producto %v no puede ser negativo.", item.ProductID))
		}
		if !slices.Contains(manualStockAdjustmentTypes, item.Type) {
			return req.MakeErr(fmt.Sprintf("Seleccione el motivo del cambio de stock del producto %v.", item.ProductID))
		}
		// Opening stock is how a company onboards, so flag 6 does not block it; and every
		// valuation starts from it, so it always carries its cost, average or not.
		isOpeningStock := item.Type == types.StockMovementTypeOpeningStock
		subDivisor := productSubDivisor(productByID[item.ProductID])
		if item.SubQuantity != 0 && subDivisor <= core.QuantityDivisorNone {
			return req.MakeErr(fmt.Sprintf(
				"El producto %v no tiene sub-unidad configurada, pero el ajuste envía %v sub-unidades.",
				item.ProductID, item.SubQuantity))
		}
		movimientos = append(movimientos, withProductLotPolicy(types.InternalMovement{
			Type:            item.Type,
			SubDivisor:      subDivisor,
			ReplaceQuantity: true,
			WarehouseID:     item.WarehouseID,
			ProductID:       item.ProductID,
			PresentationID:  item.PresentationID,
			SerialNumber:    item.SerialNumber,
			LotID:           item.LotID,
			// LotName is the internal field; PostStockAdjustItem exposes it as LotCode on the wire.
			LotName:     item.LotCode,
			SupplierID:  item.SupplierID,
			Quantity:    item.Quantity,
			SubQuantity: item.SubQuantity,
			Price:       item.UnitCost,

			RejectInbound:      rejectManualInbound && !isOpeningStock,
			RequireInboundCost: isOpeningStock,
		}, productByID[item.ProductID], item.LotExpirationDate))
	}

	if err := types.ApplyMovimientos(req, movimientos); err != nil {
		return req.MakeErr(err)
	}

	return req.MakeResponse(items)
}

// GetProductStockLotsByIDs resolves ProductStockLot rows by ID.
// Static-lookup endpoint: the frontend treats the response as immutable-per-ID so cached rows are
// never revalidated (see getStaticRecordsByID on the client).
func GetProductStockLotsByIDs(req *core.HandlerArgs) core.HandlerResponse {
	requestedIDs, err := req.GetStaticRequestIDs()
	if err != nil {
		return req.MakeErr(err)
	}
	// ProductStockLot.ID is int32; the request decodes int64.
	lotIDs := core.Map(requestedIDs, func(id int64) int32 { return int32(id) })

	lots := []types.ProductStockLot{}
	query := db.Query(&lots)
	if err := query.CompanyID.Equals(req.User.CompanyID).ID.In(lotIDs...).Exec(); err != nil {
		return req.MakeErr("Error al obtener los lotes.", err)
	}

	return req.MakeResponse(core.StaticCachedPartition(lots))
}

type productStockLotStatusRequest struct {
	LotID          int32
	QualityStatus  int8
	ExpirationDate int16 // UnixDay, 0 = none
}

// readLotDetailStockIDs answers the stock rows holding a detail of the lot, sorted.
func readLotDetailStockIDs(companyID int32, lotID int32) ([]int64, error) {
	details := []types.ProductStockDetail{}
	query := db.Query(&details)
	query.Select(query.ProductStockID).CompanyID.Equals(companyID).LotID.Equals(lotID)
	if err := query.Exec(); err != nil {
		return nil, core.Err("Error al obtener el stock del lote:", err)
	}
	stockIDs := core.Map(details, func(detail types.ProductStockDetail) int64 { return detail.ProductStockID })
	slices.Sort(stockIDs)
	return stockIDs, nil
}

// PostProductStockLotStatus changes a lot's quality status and expiry (docs/MANUFACTURING_PLAN.md §10,
// M22). In one transaction: the stock rows holding the lot are locked in key order, then the lot (the
// engine's order), and a lot entering or leaving Liberado moves its detail quantities into or out of
// each row's Blocked counter.
func PostProductStockLotStatus(req *core.HandlerArgs) core.HandlerResponse {
	body := productStockLotStatusRequest{}
	if err := json.Unmarshal([]byte(*req.Body), &body); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	if body.LotID <= 0 {
		return req.MakeErr("Se requiere el lote.")
	}
	if body.QualityStatus != types.LotQualityReleased && !types.IsLotQualityBlocked(body.QualityStatus) {
		return req.MakeErr(fmt.Sprintf("Estado de calidad inválido: %v.", body.QualityStatus))
	}
	if body.ExpirationDate < 0 {
		return req.MakeErr("La fecha de vencimiento es inválida.")
	}
	companyID := req.User.CompanyID
	var savedLot types.ProductStockLot

	err := db.RunTransaction(func(transaction *db.Transaction) error {
		stockIDs, err := readLotDetailStockIDs(companyID, body.LotID)
		if err != nil {
			return err
		}
		lockedStocks := []types.ProductStock{}
		for _, stockID := range stockIDs {
			lockedStocks = append(lockedStocks, types.ProductStock{CompanyID: companyID, ID: stockID})
		}
		if len(lockedStocks) > 0 {
			if err := db.LockByKeys(transaction, &lockedStocks); err != nil {
				return core.Err("Error al bloquear el stock del lote:", err)
			}
		}
		lots := []types.ProductStockLot{{CompanyID: companyID, ID: body.LotID}}
		if err := db.LockByKeys(transaction, &lots); err != nil {
			return core.Err("Error al obtener el lote:", err)
		}
		if len(lots) == 0 {
			return core.Err(fmt.Sprintf("El lote %v no existe.", body.LotID))
		}
		lot := &lots[0]

		// Read again under the lot's lock: a movement that put the lot in a new stock row before it
		// committed is now visible, and that row is not locked by this transaction.
		stockIDsUnderLock, err := readLotDetailStockIDs(companyID, body.LotID)
		if err != nil {
			return err
		}
		if !slices.Equal(stockIDs, stockIDsUnderLock) {
			return core.Err("El lote recibió movimientos mientras se cambiaba su estado: intente de nuevo.")
		}

		if wasBlocked, isBlocked := types.IsLotQualityBlocked(lot.QualityStatus), types.IsLotQualityBlocked(body.QualityStatus); wasBlocked != isBlocked && len(lockedStocks) > 0 {
			details := []types.ProductStockDetail{}
			query := db.Query(&details)
			query.Select(query.ProductStockID, query.Quantity, query.SubQuantity).CompanyID.Equals(companyID).LotID.Equals(body.LotID)
			if err := query.Exec(); err != nil {
				return core.Err("Error al obtener el stock del lote:", err)
			}
			stockByID := map[int64]*types.ProductStock{}
			for index := range lockedStocks {
				stockByID[lockedStocks[index].ID] = &lockedStocks[index]
			}
			// A detail is at its stock row's divisor, so its quantity adds to Blocked as it is.
			for _, detail := range details {
				detailQuantity := core.Quantity{Units: detail.Quantity, Sub: detail.SubQuantity}
				if !isBlocked {
					detailQuantity = detailQuantity.Negate()
				}
				stock := stockByID[detail.ProductStockID]
				blocked := core.Quantity{Units: stock.BlockedQuantity, Sub: stock.BlockedSubQuantity}.Add(detailQuantity)
				stock.BlockedQuantity, stock.BlockedSubQuantity = blocked.Units, blocked.Sub
			}
			stockTable := db.TableOf[types.ProductStock]()
			// Written with WarehouseID and Status, the delta index's pinned columns.
			if err := db.UpdateIn(transaction, &lockedStocks, stockTable.BlockedQuantity, stockTable.BlockedSubQuantity,
				stockTable.WarehouseID, stockTable.Status); err != nil {
				return core.Err("Error al guardar el stock bloqueado del lote:", err)
			}
		}

		lot.QualityStatus, lot.ExpirationDate = body.QualityStatus, body.ExpirationDate
		lot.Updated, lot.UpdatedBy = core.SUnixTime(), req.User.ID
		lotTable := db.TableOf[types.ProductStockLot]()
		if err := db.UpdateIn(transaction, &lots, lotTable.QualityStatus, lotTable.ExpirationDate,
			lotTable.Updated, lotTable.UpdatedBy); err != nil {
			return core.Err("Error al guardar el lote:", err)
		}
		savedLot = *lot
		return nil
	})
	if err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(savedLot)
}

func GetAlmacenMovimientos(req *core.HandlerArgs) core.HandlerResponse {

	warehouseID := int32(req.GetQueryInt("warehouse-id"))
	dateInicio := req.GetQueryInt16("date-inicio")
	dateFin := req.GetQueryInt16("date-fin")
	productID := int32(req.GetQueryInt("product-id"))
	lotCode := req.GetQuery("lot-code")
	documentID := req.GetQueryInt64("document-id")
	serialNumber := req.GetQuery("serial-number")
	tipo := types.StockMovementType(req.GetQueryInt("tipo"))

	// Resolve lot-code → all matching lotIDs (same Name can exist across multiple entries).
	var lotIDs []int32

	if lotCode != "" {
		lots := []types.ProductStockLot{}
		query := db.Query(&lots).CompanyID.Equals(req.User.CompanyID)

		if err := query.Select(query.ID).Name.Equals(lotCode).Exec(); err != nil {
			return req.MakeErr("Error al buscar el lote:", err)
		}
		if len(lots) == 0 {
			return req.MakeErr("No se encontró un lote con ese código.")
		}
		for _, lot := range lots {
			lotIDs = append(lotIDs, lot.ID)
		}
	}

	movimientos := []db.RecordGroup[types.WarehouseProductMovement]{}

	// Direct-lookup path: SerialNumber / lotIDs / DocumentID target non-grouped local indexes,
	// so plain Query applies and the Date range is ignored.
	if serialNumber != "" || len(lotIDs) > 0 || documentID > 0 {
		flat := []types.WarehouseProductMovement{}
		query := db.Query(&flat)
		query.CompanyID.Equals(req.User.CompanyID)
		switch {
		case serialNumber != "":
			query.SerialNumber.Equals(serialNumber)
		case documentID > 0:
			query.DocumentID.Equals(documentID)
		case len(lotIDs) > 0:
			query.LotID.In(lotIDs...).AllowFilter()
		}
		if err := query.Exec(); err != nil {
			return req.MakeErr("Error al obtener los movimientos:", err)
		}
		if len(flat) > 0 {
			movimientos = append(movimientos, db.RecordGroup[types.WarehouseProductMovement]{
				IndexID: -1,
				Records: flat,
			})
		}
		return req.MakeResponse(movimientos)
	}

	if dateInicio <= 0 || dateFin <= 0 {
		return req.MakeErr("Debe especificar el rango de dates.")
	}
	if dateFin < dateInicio {
		return req.MakeErr("La date final no puede ser menor a la date inicial.")
	}
	if dateFin-dateInicio > 120 {
		return req.MakeErr("Sólo se pueden consultar hasta 120 días a la vez.")
	}

	cacheGroupHashes, err := core.ExtractGroupIndexCacheValues(req)
	if err != nil {
		return req.MakeErr(err)
	}

	query := db.QueryIndexGroup(&movimientos).
		CompanyID.Equals(req.User.CompanyID)

	for _, cacheGroup := range cacheGroupHashes {
		query.IncludeCachedGroup(cacheGroup.GroupHash, cacheGroup.LastWrite)
	}

	// Date is the single BETWEEN required by QueryIndexGroup; the most specific
	// compatible grouped index is picked from the remaining equality filters.
	query.Date.Between(dateInicio, dateFin)

	switch {
	case tipo > 0 && warehouseID > 0:
		// Uses the raw group: Date + Type + WarehouseID.
		query.Type.Equals(tipo).WarehouseID.Equals(warehouseID)
	case tipo > 0:
		// Uses the raw group: Date + Type.
		query.Type.Equals(tipo)
	case warehouseID > 0:
		// Uses the raw group: Date + WarehouseID.
		query.WarehouseID.Equals(warehouseID)
	case productID > 0:
		// Uses the raw group: Date + ProductID.
		query.ProductID.Equals(productID)
	}

	if err := query.Exec(); err != nil {
		core.Log("Error querying movement groups:", err)
		return req.MakeErr("Error al obtener los movimientos del almacén.")
	}

	return req.MakeResponse(movimientos)
}

type GetProductsStockResult struct {
	ProductStock []types.ProductStock
	// The details answer whole, with the replace Metadata: their composite key has no delta log.
	ProductStockDetail []any
	// The lots of those details, whole too: a lot's quality status and expiry change, so the stock
	// screen reads them fresh with the details rather than from the immutable by-IDs cache.
	ProductStockLot []any
}

func GetWarehouseProductStock(req *core.HandlerArgs) core.HandlerResponse {
	warehouseID := int32(req.GetQueryInt("warehouse-id"))
	result := GetProductsStockResult{}
	eg := errgroup.Group{}

	eg.Go(func() error {
		// Pinning WarehouseID routes Delta() to the [WarehouseID, Status] delta index and leaves
		// Status as the sync filter a first sync lists.
		query := db.Query(&result.ProductStock)
		query.Select().
			CompanyID.Equals(req.User.CompanyID).
			WarehouseID.Equals(warehouseID).
			Delta(req.GetDeltaCursor("ProductStock"), 1)

		return query.Exec()
	})

	eg.Go(func() error {
		details := []types.ProductStockDetail{}
		query := db.Query(&details)
		query.Select().
			CompanyID.Equals(req.User.CompanyID).
			WarehouseID.Equals(warehouseID).
			Status.Equals(1)
		if err := query.Exec(); err != nil {
			return err
		}
		result.ProductStockDetail = core.WithDeltaReplace(details)

		lots := []types.ProductStockLot{}
		if lotIDs := core.Map(details, func(detail types.ProductStockDetail) int32 { return detail.LotID }); len(lotIDs) > 0 {
			lotQuery := db.Query(&lots)
			if err := lotQuery.CompanyID.Equals(req.User.CompanyID).ID.In(lotIDs...).Exec(); err != nil {
				return err
			}
		}
		result.ProductStockLot = core.WithDeltaReplace(lots)
		return nil
	})

	if err := eg.Wait(); err != nil {
		return req.MakeErr("Error al obtener el stock de productos del almacén.:", err)
	}

	return req.MakeResponse(result)
}

func GetProductsStock(req *core.HandlerArgs) core.HandlerResponse {
	productsStock := []types.ProductStock{}
	// No WarehouseID pinned here, so Delta() routes to the [Status] delta index instead.
	query := db.Query(&productsStock)
	query.Select().
		CompanyID.Equals(req.User.CompanyID).
		Delta(req.GetDeltaCursor(), 1)

	if err := query.Exec(); err != nil {
		return req.MakeErr("Arror al obtener los productos stock::", err)
	}

	return req.MakeResponse(productsStock)
}
