package logistics

import (
	"app/core"
	"app/db"
	"app/logistics/types"
	production "app/production/types"
	"encoding/json"
	"fmt"

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
	WarehouseID    int32  `json:",omitempty"`
	ProductID      int32  `json:",omitempty"`
	PresentationID int16  `json:",omitempty"`
	Quantity       int32  `json:",omitempty"`
	SubQuantity    int32  `json:",omitempty"`
	SerialNumber   string `json:",omitempty"`
	LotID          int32  `json:",omitempty"`
	LotCode        string `json:",omitempty"`
	SupplierID     int32  `json:",omitempty"`
}

// loadProductSubDivisors reads the current sub-unit divisor of every product an adjustment
// touches. A product with no sub-unit maps to core.QuantityDivisorNone.
func loadProductSubDivisors(companyID int32, items []PostStockAdjustItem) (map[int32]int16, error) {
	productIDs := core.SliceSet[int32]{}
	for _, item := range items {
		productIDs.AddIf(item.ProductID)
	}
	divisorByProductID := map[int32]int16{}
	if productIDs.IsEmpty() {
		return divisorByProductID, nil
	}

	products := []production.Product{}
	query := db.Query(&products)
	query.Select(query.ID, query.SbuQuantity).
		CompanyID.Equals(companyID).
		ID.In(productIDs.Values...)
	if err := query.Exec(); err != nil {
		return nil, core.Err("Error al obtener la sub-unidad de los productos:", err)
	}
	for _, product := range products {
		divisorByProductID[product.ID] = core.If(
			product.SbuQuantity > core.QuantityDivisorNone, product.SbuQuantity, core.QuantityDivisorNone)
	}
	return divisorByProductID, nil
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
	subDivisorByProductID, err := loadProductSubDivisors(req.User.CompanyID, items)
	if err != nil {
		return req.MakeErr(err)
	}

	movimientos := make([]types.InternalMovement, 0, len(items))
	for _, item := range items {
		if item.WarehouseID == 0 || item.ProductID == 0 {
			return req.MakeErr("Hay un registro sin Almacén-ID o Producto-ID.")
		}
		subDivisor := subDivisorByProductID[item.ProductID]
		if item.SubQuantity != 0 && subDivisor <= core.QuantityDivisorNone {
			return req.MakeErr(fmt.Sprintf(
				"El producto %v no tiene sub-unidad configurada, pero el ajuste envía %v sub-unidades.",
				item.ProductID, item.SubQuantity))
		}
		movimientos = append(movimientos, types.InternalMovement{
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
		})
	}

	if err := types.ApplyMovimientos(req, movimientos); err != nil {
		return req.MakeErr(err)
	}

	return req.MakeResponse(items)
}

// GetProductStockLotsByIDs resolves ProductStockLot rows by ID.
// Static-lookup endpoint: no cache-version protocol, just `ids`. The frontend treats the response
// as immutable-per-ID so cached rows are never revalidated (see getStaticRecordsByID on the client).
func GetProductStockLotsByIDs(req *core.HandlerArgs) core.HandlerResponse {
	lotIDRecords := req.ExtractUpdatedVersionValues()
	if len(lotIDRecords) == 0 {
		return req.MakeErr("No se enviaron ids de lotes.")
	}

	// ProductStockLot.ID is int32; cache-version values come in as int64.
	lotIDs := core.Map(lotIDRecords, func(e db.IDUpdatedVersion) int32 { return int32(e.ID) })

	lots := []types.ProductStockLot{}
	query := db.Query(&lots)
	if err := query.CompanyID.Equals(req.User.CompanyID).ID.In(lotIDs...).Exec(); err != nil {
		return req.MakeErr("Error al obtener los lotes.", err)
	}

	return core.MakeResponse(req, &lots)
}

func GetAlmacenMovimientos(req *core.HandlerArgs) core.HandlerResponse {

	warehouseID := int32(req.GetQueryInt("warehouse-id"))
	dateInicio := req.GetQueryInt16("date-inicio")
	dateFin := req.GetQueryInt16("date-fin")
	productID := int32(req.GetQueryInt("product-id"))
	lotCode := req.GetQuery("lot-code")
	documentID := req.GetQueryInt64("document-id")
	serialNumber := req.GetQuery("serial-number")
	tipo := int8(req.GetQueryInt("tipo"))

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
		query.IncludeCachedGroup(cacheGroup.GroupHash, cacheGroup.UpdateCounter)
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
	ProductStock       []types.ProductStock
	ProductStockDetail []types.ProductStockDetail
}

func GetWarehouseProductStock(req *core.HandlerArgs) core.HandlerResponse {
	// Returns V2 stock rows + their detail rows + the lot catalog.
	// `updated` enables delta-cache fetches via the `upd` field on stocks/details.
	warehouseID := int32(req.GetQueryInt("warehouse-id"))
	// Each table advances its own "updated_version" sequence, so the two response keys carry
	// independent watermarks; the frontend sends one query param per key, named after it.
	productStockUpdatedSince := int32(req.GetQueryInt("ProductStock"))
	productStockDetailUpdatedSince := int32(req.GetQueryInt("ProductStockDetail"))

	result := GetProductsStockResult{}

	eg := errgroup.Group{}

	eg.Go(func() error {
		// Pinning WarehouseID routes Delta() to the [WarehouseID, Status] delta index and leaves
		// Status as the sync filter: active only on a first sync, both values afterwards.
		query := db.Query(&result.ProductStock)
		query.Select().
			CompanyID.Equals(req.User.CompanyID).
			WarehouseID.Equals(warehouseID).
			Delta(productStockUpdatedSince, 1)

		return query.Exec()
	})

	eg.Go(func() error {
		query := db.Query(&result.ProductStockDetail)
		query.Select().
			CompanyID.Equals(req.User.CompanyID).
			WarehouseID.Equals(warehouseID).
			Delta(productStockDetailUpdatedSince, 1)

		return query.Exec()
	})

	if err := eg.Wait(); err != nil {
		return req.MakeErr("Error al obtener el stock de productos del almacén.:", err)
	}

	return req.MakeResponse(result)
}

func GetProductsStock(req *core.HandlerArgs) core.HandlerResponse {
	updatedSince := req.GetQueryInt("upv")

	productsStock := []types.ProductStock{}
	// No WarehouseID pinned here, so Delta() routes to the [Status] delta index instead.
	query := db.Query(&productsStock)
	query.Select().
		CompanyID.Equals(req.User.CompanyID).
		Delta(updatedSince, 1)

	if err := query.Exec(); err != nil {
		return req.MakeErr("Arror al obtener los productos stock::", err)
	}

	return req.MakeResponse(productsStock)
}
