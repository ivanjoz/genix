// The web store's sellable stock: one rule for the catalog file, its delta and checkout, so the store
// never lists what checkout would refuse. docs/ECOMMERCE_CATALOG_SNAPSHOT_PLAN.md, "Sellable rule".
//
// Stock is per (product, presentation), in whole units; the overselling margin (Product.WebExtraStock)
// is one per product, shared by all its presentations. A presentation's net is its stock less what
// undelivered sales and web orders reserved, negative once it is oversold. What the oversold
// presentations already took comes off the margin, and each presentation can sell its own positive
// net plus what is left of the margin.

package types

import (
	"app/core"
	"app/db"
)

// ReadWebStockRows reads, by product, the stock rows of each product's sellable presentations on the
// warehouses open to online sales: everything WebSellableStock and WebOrderFits need. By packed ID,
// so it suits a bounded set of products (a cart, a delta); a whole-catalog build reads the company.
func ReadWebStockRows(companyID int32, onlineWarehouseIDs []int32, presentationIDsByProduct map[int32][]int16) (map[int32][]ProductStock, error) {
	stockIDs := core.SliceSet[int64]{}
	for productID, presentationIDs := range presentationIDsByProduct {
		for _, warehouseID := range onlineWarehouseIDs {
			for _, presentationID := range presentationIDs {
				stockIDs.Add(PackProductStockID(warehouseID, productID, presentationID))
			}
		}
	}
	rowsByProduct := map[int32][]ProductStock{}
	if stockIDs.IsEmpty() {
		return rowsByProduct, nil
	}
	stockRows := []ProductStock{}
	query := db.Query(&stockRows)
	query.Select(query.ProductID, query.PresentationID, query.Quantity, query.DetailQuantity,
		query.PendingDeliveryStore, query.PendingDeliveryOnline).
		CompanyID.Equals(companyID).ID.In(stockIDs.Values...)
	if err := query.Exec(); err != nil {
		return nil, core.Err("Error al obtener el stock:", err)
	}
	for _, stockRow := range stockRows {
		rowsByProduct[stockRow.ProductID] = append(rowsByProduct[stockRow.ProductID], stockRow)
	}
	return rowsByProduct, nil
}

// webNetByPresentation sums the stock rows (already limited to the warehouses open to online sales)
// into each presentation's net units. A row of a presentation outside presentationIDs (a deleted
// one) is not sellable and is left out.
func webNetByPresentation(stockRows []ProductStock, presentationIDs []int16) map[int16]int32 {
	netByPresentation := make(map[int16]int32, len(presentationIDs))
	for _, presentationID := range presentationIDs {
		netByPresentation[presentationID] = 0
	}
	for _, stockRow := range stockRows {
		if _, isSellable := netByPresentation[stockRow.PresentationID]; isSellable {
			netByPresentation[stockRow.PresentationID] += stockRow.Quantity + stockRow.DetailQuantity -
				stockRow.PendingDeliveryStore - stockRow.PendingDeliveryOnline
		}
	}
	return netByPresentation
}

// webOversoldUnits is how far below zero the presentations' nets went, together: the share of the
// margin already used.
func webOversoldUnits(netByPresentation map[int16]int32) int32 {
	oversoldUnits := int32(0)
	for _, netUnits := range netByPresentation {
		oversoldUnits += max(-netUnits, 0)
	}
	return oversoldUnits
}

// WebSellableStock is what the web store can sell of each presentation of one product right now.
// Every presentation in presentationIDs gets an entry, 0 included.
func WebSellableStock(stockRows []ProductStock, presentationIDs []int16, webExtraStock int32) map[int16]int32 {
	netByPresentation := webNetByPresentation(stockRows, presentationIDs)
	marginLeft := max(webExtraStock-webOversoldUnits(netByPresentation), 0)
	sellableByPresentation := make(map[int16]int32, len(netByPresentation))
	for presentationID, netUnits := range netByPresentation {
		sellableByPresentation[presentationID] = max(netUnits, 0) + marginLeft
	}
	return sellableByPresentation
}

// WebOrderFits reports whether one product's ordered units (by presentation) can be sold online: what
// they oversell, added to what is already oversold, must stay within the margin. An order that
// oversells nothing always fits, even when a lowered margin is already exceeded. A presentation
// outside presentationIDs never fits.
func WebOrderFits(stockRows []ProductStock, presentationIDs []int16, webExtraStock int32, orderedUnits map[int16]int32) bool {
	netByPresentation := webNetByPresentation(stockRows, presentationIDs)
	oversoldBefore := webOversoldUnits(netByPresentation)
	for presentationID, units := range orderedUnits {
		if _, isSellable := netByPresentation[presentationID]; !isSellable {
			return false
		}
		netByPresentation[presentationID] -= units
	}
	return webOversoldUnits(netByPresentation) <= max(webExtraStock, oversoldBefore)
}
