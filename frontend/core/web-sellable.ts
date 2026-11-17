import type { IProductStock } from '#routes/logistics/products-stock/stock-movement.ts'
import type { IProduct } from '#services/production/products.svelte.ts'

// The sellable-stock rule of backend/logistics/types/web_sellable.go, for the menus the browser
// builds from its delta caches. The server applies the same rule when the order arrives, so this
// copy only decides what is offered.
//
// Stock is per (product, presentation), in whole units; the overselling margin (WebExtraStock) is
// one per product, shared by all its presentations. A presentation's net is its stock less its
// blocked lots and what undelivered sales and web orders reserved, negative once it is oversold.
// What the oversold presentations took comes off the margin; each presentation sells its own
// positive net plus what is left of the margin.

type IStockRow = Pick<IProductStock, 'PresentationID' | 'Quantity' | 'DetailQuantity' | 'BlockedQuantity'
  | 'PendingDeliveryStore' | 'PendingDeliveryOnline'>

// The active presentations, or the product itself (presentation 0) when it has none. Mirrors
// Product.WebPresentationIDs.
export function webPresentationIDs(product: Pick<IProduct, 'Presentations'>): number[] {
  const presentationIDs = (product.Presentations || []).filter((presentation) => presentation.ss > 0).map((presentation) => presentation.id)
  return presentationIDs.length ? presentationIDs : [0]
}

// What can be sold of each presentation of one product right now, from its stock rows (Status 1,
// already limited to the warehouses that count). Every presentation gets an entry, 0 included; a row
// of a presentation outside presentationIDs (a deleted one) is not sellable. Mirrors WebSellableStock.
export function webSellableStock(stockRows: IStockRow[], presentationIDs: number[], webExtraStock: number): Map<number, number> {
  const netByPresentation = new Map(presentationIDs.map((presentationID) => [presentationID, 0]))
  for (const stockRow of stockRows) {
    const presentationID = stockRow.PresentationID || 0
    const netUnits = netByPresentation.get(presentationID)
    if (netUnits === undefined) continue
    // Zero fields arrive absent (omitempty). Blocked lots (Cuarentena, Rechazado) are not for sale.
    netByPresentation.set(presentationID, netUnits + (stockRow.Quantity || 0) + (stockRow.DetailQuantity || 0)
      - (stockRow.BlockedQuantity || 0) - (stockRow.PendingDeliveryStore || 0) - (stockRow.PendingDeliveryOnline || 0))
  }
  let oversoldUnits = 0
  for (const netUnits of netByPresentation.values()) oversoldUnits += Math.max(-netUnits, 0)
  const marginLeft = Math.max((webExtraStock || 0) - oversoldUnits, 0)
  return new Map([...netByPresentation].map(([presentationID, netUnits]) => [presentationID, Math.max(netUnits, 0) + marginLeft]))
}
