// Turns a sale the backend just accepted into the local history row, and reads back the few
// facts a card or a ticket needs from it. Pure: no Svelte, no fetch.

import type { SaleHistoryLine, SaleHistoryRow } from './sale_history.idb'
import type { ISaleOrder, VentaProducto } from './sale_order.svelte'

// Sale.Status (`ss`) is the persisted state of the sale, and the only honest source for these
// two: ActionsIncluded is what the till *asked* for, while the backend may refuse half of it.
// 0 = Anulado, 1 = Generado, 2 = Pagado, 3 = Entregado, 4 = Pagado + Entregado.
export const isSalePaid = (status: number): boolean => status === 2 || status === 4
export const isSaleDelivered = (status: number): boolean => status === 3 || status === 4

export const buildSaleHistoryLine = (soldProduct: VentaProducto): SaleHistoryLine => ({
  productID: soldProduct.productoID,
  presentationID: soldProduct.presentationID,
  // Spelled out field by field rather than passed through: the cart is `$state`, so its nested
  // objects are reactive proxies, and IndexedDB refuses to structured-clone one.
  quantity: { units: soldProduct.cantidad.units, sub: soldProduct.cantidad.sub },
  subDivisor: soldProduct.subDivisor,
  unitPrice: soldProduct.producto?.FinalPrice || 0,
  subUnitPrice: soldProduct.producto?.SbuFinalPrice || 0,
  // The cart keeps serials as serial → quantity; the ticket lists each one once and only
  // says "x2" when the same serial covers more than one item.
  serialNumbers: [...(soldProduct.serialNumbers?.entries() || [])]
    .map(([serialNumber, serialQuantity]) =>
      serialQuantity > 1 ? `${serialNumber} x${serialQuantity}` : serialNumber),
})

// soldProducts is the cart as it stood at submit time: postSaleOrder hands it over before
// clearing it, because the sale response carries packed lines and not the products behind them.
export const buildSaleHistoryRow = (
  sale: ISaleOrder, soldProducts: VentaProducto[], savedAt: number,
): SaleHistoryRow => ({
  saleID: sale.ID,
  savedAt,
  clientID: sale.ClientID || 0,
  warehouseID: sale.WarehouseID || 0,
  cashierID: sale.UpdatedBy || 0,
  cashBankID: sale.LastPaymentCajaID || 0,
  status: sale.ss,
  totalAmount: sale.TotalAmount,
  taxAmount: sale.TaxAmount,
  lines: soldProducts.map(buildSaleHistoryLine),
})
