package types

import (
	"app/core"
	"app/db"
	logistics "app/logistics/types"
	production "app/production/types"
	"fmt"
	"strings"

	"golang.org/x/sync/errgroup"
)

// ValidateSaleStock ensures each sale-order line has enough stock available. The lines of made
// products follow validateMadeLinesSellable; checksMadeLines is whether they reserve on the sale's
// warehouse now (a delivery of a sale reserved there already does not). The rest of the lines:
// lines with no LotID and no SerialNumber draw from ProductStockV2.Quantity; for a sale that holds
// no reservation yet, less the units undelivered sales and web orders reserved on the row
// (PendingDelivery*). The delivery of a sale that does hold one (isReserved) only needs the physical
// stock: its units are already among the reserved, which may exceed the stock after an oversold web
// order. Lines with a LotID or SerialNumber draw from the matching ProductStockDetail row:
// reservations name no lot.
//
// It is an early check, on a plain read: a delivery's movements re-check against the stock rows
// they lock inside the sale's transaction.
func ValidateSaleStock(companyID int32, sale SaleOrder, isReserved bool,
	productByID map[int32]production.Product, checksMadeLines bool) error {
	if sale.WarehouseID == 0 {
		return core.Err("Se requiere WarehouseID para validar el stock.")
	}
	madeLines, boughtLines := []SaleOrderLine{}, []SaleOrderLine{}
	for _, line := range sale.Lines {
		if productByID[line.ProductID].Origin == production.ProductOriginMade {
			madeLines = append(madeLines, line)
		} else {
			boughtLines = append(boughtLines, line)
		}
	}
	if err := validateMadeLinesSellable(companyID, sale.WarehouseID, madeLines, productByID, checksMadeLines); err != nil {
		return err
	}
	sale.Lines = boughtLines

	// Aggregate requested quantity by the physical storage bucket it hits.
	type lineKey struct {
		stockID      int64
		lotID        int32
		serialNumber string
	}
	requestedByKey := map[lineKey]core.Quantity{}
	divisorByKey := map[lineKey]int16{}
	for _, line := range sale.Lines {
		if line.Quantity == 0 {
			continue
		}
		quantity := core.UnpackQuantityLine(line.Quantity)
		lineDivisor := line.SubDivisor
		if lineDivisor <= 0 {
			lineDivisor = core.QuantityDivisorNone
		}
		key := lineKey{
			stockID:      logistics.PackProductStockID(sale.WarehouseID, line.ProductID, line.PresentationID),
			lotID:        line.LotID,
			serialNumber: line.Sku,
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
		q.Select(q.ID, q.Quantity, q.SubQuantity, q.DetailQuantity, q.DetailSubQuantity, q.SubDivisor,
			q.PendingDeliveryStore, q.PendingDeliveryOnline).
			CompanyID.Equals(companyID).
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
				CompanyID.Equals(companyID).
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
			if !isReserved {
				available.Units -= stock.PendingDeliveryStore + stock.PendingDeliveryOnline
			}
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
			// Decompose the packed stock ID (see logistics.PackProductStockID) for a human-readable error.
			warehouseID := key.stockID >> 44
			productID := (key.stockID >> 14) & (1<<30 - 1)
			presentationID := key.stockID & (1<<14 - 1)
			metadata := []string{
				fmt.Sprintf("Almacén: %v", warehouseID),
				fmt.Sprintf("Producto: %v", productID),
			}
			if presentationID > 0 {
				metadata = append(metadata, fmt.Sprintf("Presentación: %v", presentationID))
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

// validateMadeLinesSellable checks the lines of made products. A made product enters stock only at
// the daily production close, into the free bucket, so its line may not name a lot or a serial. Until
// that close it sells against the sellable rule on the sale's warehouse (logistics.WebOrderFits): its
// stock less what is reserved, plus the product's "Sobreventa permitida" (WebExtraStock), not against
// physical stock. checksMargin is false when the units are reserved on that warehouse already.
func validateMadeLinesSellable(companyID int32, warehouseID int32, madeLines []SaleOrderLine,
	productByID map[int32]production.Product, checksMargin bool) error {
	presentationIDsByProduct := map[int32][]int16{}
	for _, line := range madeLines {
		product := productByID[line.ProductID]
		if line.LotID > 0 || line.Sku != "" {
			return core.Err(fmt.Sprintf(`"%s" es un producto fabricado: su venta no puede indicar lote ni serie.`, product.Name))
		}
		presentationIDsByProduct[line.ProductID] = product.WebPresentationIDs()
	}
	if !checksMargin || len(madeLines) == 0 {
		return nil
	}

	// What the lines take of each product, by presentation, in the whole units their reservation holds.
	orderedUnitsByProduct := map[int32]map[int16]int32{}
	for _, change := range PendingDeliveryChanges(madeLines, warehouseID, false, 1) {
		if orderedUnitsByProduct[change.ProductID] == nil {
			orderedUnitsByProduct[change.ProductID] = map[int16]int32{}
		}
		orderedUnitsByProduct[change.ProductID][change.PresentationID] += change.Store
	}
	stockRowsByProduct, err := logistics.ReadWebStockRows(companyID, []int32{warehouseID}, presentationIDsByProduct)
	if err != nil {
		return err
	}
	for productID, orderedUnits := range orderedUnitsByProduct {
		product := productByID[productID]
		if !logistics.WebOrderFits(stockRowsByProduct[productID], presentationIDsByProduct[productID], product.WebExtraStock, orderedUnits) {
			return core.Err(fmt.Sprintf(`"%s" no tiene stock ni sobreventa permitida suficiente en el almacén %v.`, product.Name, warehouseID))
		}
	}
	return nil
}
