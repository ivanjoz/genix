// Pedidos Web: the business rules of the back-office page that lists the web store's online sale
// orders and turns them into SaleOrders. Pure: no Svelte, no fetch. See docs/ONLINE_SALE_ORDER_PLAN.md.

import type { ISaleOrderLine } from '#routes/sales/sale_order_create/sale_order.svelte.ts'
import type { ICashBank } from '#routes/finance/cash-banks/cajas.svelte.ts'
import type { IWarehouse } from '#routes/business/branches-warehouses/branches-warehouses.svelte.ts'
import { CURRENCY_PEN, normalizeCurrency } from '#routes/finance/exchange-rate/exchange-rate.ts'

// Contact and address snapshot the shopper typed at checkout.
export interface IOnlineSaleOrderShipping {
  FirstName: string
  LastName: string
  Email: string
  // District ubigeo.
  CityID: number
  Address: string
  Reference: string
  Phone: string
  DocumentNumber: string
}

export interface IOnlineSaleOrder {
  // Per-company incremental: the number the shopper sees ("Pedido N° 125").
  ID: number
  Date: number
  Lines?: ISaleOrderLine[]
  // Amounts in cents, PEN.
  TotalAmount: number
  DebtAmount: number
  ShippingAmount: number
  ExchangeRate: number
  DeliveryType: number
  // A Sede ID, pickup orders only.
  PickupSiteID: number
  Shipping?: IOnlineSaleOrderShipping
  WebCustomerID: number
  ClientID: number
  PaymentMethod: number
  CulqiChargeID: string
  // 0 until the SaleOrder exists; from then on the SaleOrder rules and this page is read-only.
  SaleOrderID: number
  AnnulReason: string
  ss: number
  Created: number
  upd: number
  ver: number
  mdt?: string
}

export interface IOnlineSaleOrderSettleRequest {
  ID: number
  ActionsIncluded: number[]
  CashBankID: number
  WarehouseID: number
}

export interface IOnlineSaleOrderAnnulRequest {
  ID: number
  Reason: string
}

// Same status values as SaleOrder.
export const ONLINE_ORDER_STATUS = {
  ANNULLED: 0,
  PENDING: 1,
  PAID: 2,
  DELIVERED: 3,
  COMPLETED: 4,
} as const

// The `status-group` query param of GET online-sale-orders.
export const ONLINE_ORDER_GROUP = {
  ANNULLED: 0,
  PENDING: 1,
  IN_PROGRESS: 2,
  COMPLETED: 3,
} as const

// Tabs in display order.
export const ONLINE_ORDER_GROUP_TABS: [number, string][] = [
  [ONLINE_ORDER_GROUP.PENDING, 'Pending|Pendientes'],
  [ONLINE_ORDER_GROUP.IN_PROGRESS, 'In progress|En proceso'],
  [ONLINE_ORDER_GROUP.COMPLETED, 'Completed|Completados'],
  [ONLINE_ORDER_GROUP.ANNULLED, 'Annulled|Anulados'],
]

export const DELIVERY_TYPE = { HOME: 1, PICKUP: 2 } as const
export const PAYMENT_METHOD = { CARD: 1, YAPE: 2, DEFERRED: 3 } as const
// SaleOrder actions the settle route accepts.
export const SETTLE_ACTION = { PAYMENT: 2, DELIVERY: 3 } as const

// Labels are bilingual "English|Español" strings for tr().
export const onlineOrderStatusLabel = (status: number): string => {
  switch (status) {
    case ONLINE_ORDER_STATUS.ANNULLED: return 'Annulled|Anulado'
    case ONLINE_ORDER_STATUS.PENDING: return 'Pending|Pendiente'
    case ONLINE_ORDER_STATUS.PAID: return 'Paid, not delivered|Pagado, por entregar'
    case ONLINE_ORDER_STATUS.DELIVERED: return 'Delivered, not paid|Entregado, por cobrar'
    case ONLINE_ORDER_STATUS.COMPLETED: return 'Completed|Completado'
    default: return 'Unknown|Desconocido'
  }
}

export const deliveryTypeLabel = (deliveryType: number): string => {
  if (deliveryType === DELIVERY_TYPE.HOME) return 'Home delivery|Entrega a domicilio'
  if (deliveryType === DELIVERY_TYPE.PICKUP) return 'Store pickup|Recojo en tienda'
  return '-'
}

export const paymentMethodLabel = (paymentMethod: number): string => {
  if (paymentMethod === PAYMENT_METHOD.CARD) return 'Card|Tarjeta'
  if (paymentMethod === PAYMENT_METHOD.YAPE) return 'Yape'
  if (paymentMethod === PAYMENT_METHOD.DEFERRED) return 'On delivery / in store|Contra entrega / pago en tienda'
  return '-'
}

// The tab an order belongs to, mirroring the backend's status-group filter.
export const statusGroupOf = (status: number): number => {
  if (status === ONLINE_ORDER_STATUS.PENDING) return ONLINE_ORDER_GROUP.PENDING
  if (status === ONLINE_ORDER_STATUS.PAID || status === ONLINE_ORDER_STATUS.DELIVERED) return ONLINE_ORDER_GROUP.IN_PROGRESS
  if (status === ONLINE_ORDER_STATUS.COMPLETED) return ONLINE_ORDER_GROUP.COMPLETED
  return ONLINE_ORDER_GROUP.ANNULLED
}

// Once a SaleOrder exists every later action runs on it in Gestión Ventas, so the online order
// only accepts actions while it is pending and unlinked. The backend enforces the same rule.
export const canSettle = (order: IOnlineSaleOrder): boolean =>
  order.ss === ONLINE_ORDER_STATUS.PENDING && !order.SaleOrderID

// Annulling has the same precondition: with no SaleOrder there is no document nor stock to undo.
export const canAnnul = (order: IOnlineSaleOrder): boolean => canSettle(order)

// Culqi approved the charge but creating the SaleOrder failed: the money is in, the sale is not.
// Registrar pago into the Culqi account finishes it.
export const needsCulqiReconciliation = (order: IOnlineSaleOrder): boolean =>
  !!order.CulqiChargeID && !order.SaleOrderID && order.ss === ONLINE_ORDER_STATUS.PENDING

export const customerName = (order: IOnlineSaleOrder): string =>
  `${order.Shipping?.FirstName || ''} ${order.Shipping?.LastName || ''}`.trim() || '-'

// Online orders are priced in PEN, so only active PEN accounts can collect them.
export const paymentCashBankOptions = (cashBanks: ICashBank[]): ICashBank[] =>
  cashBanks.filter((cashBank) => (cashBank.ss || 0) > 0 && normalizeCurrency(cashBank.CurrencyType) === CURRENCY_PEN)

// A Culqi-charged order must be collected into the account that receives Culqi payments, where
// the money actually is; any other order defaults to the first eligible account.
export const defaultPaymentCashBankID = (order: IOnlineSaleOrder, cashBanks: ICashBank[]): number => {
  const eligibleCashBanks = paymentCashBankOptions(cashBanks)
  if (needsCulqiReconciliation(order)) {
    const culqiCashBank = eligibleCashBanks.find((cashBank) => cashBank.ReceivesCulqiPayments)
    if (culqiCashBank) return culqiCashBank.ID
  }
  return eligibleCashBanks[0]?.ID || 0
}

// Warehouses that serve the web store: active and not excluded from online sales.
export const deliveryWarehouseOptions = (warehouses: IWarehouse[]): IWarehouse[] =>
  warehouses.filter((warehouse) => (warehouse.ss || 0) > 0 && !warehouse.ExcludeFromOnlineSales)

// A pickup order leaves from a warehouse of the pickup store when there is one.
export const defaultDeliveryWarehouseID = (order: IOnlineSaleOrder, warehouses: IWarehouse[]): number => {
  const eligibleWarehouses = deliveryWarehouseOptions(warehouses)
  if (order.DeliveryType === DELIVERY_TYPE.PICKUP && order.PickupSiteID) {
    const pickupWarehouse = eligibleWarehouses.find((warehouse) => warehouse.SiteID === order.PickupSiteID)
    if (pickupWarehouse) return pickupWarehouse.ID
  }
  return eligibleWarehouses[0]?.ID || 0
}

// Validates a settle (payment, delivery or both) and builds the POST body. The error is a
// bilingual message for tr(); the backend validates again.
export const buildSettleRequest = (
  order: IOnlineSaleOrder, actionsIncluded: number[], cashBankID: number, warehouseID: number,
): { request?: IOnlineSaleOrderSettleRequest; error?: string } => {
  if (!canSettle(order)) {
    return { error: 'Only a pending order without a sale can be settled here.|Solo un pedido pendiente y sin venta se puede registrar aquí.' }
  }
  const includesPayment = actionsIncluded.includes(SETTLE_ACTION.PAYMENT)
  const includesDelivery = actionsIncluded.includes(SETTLE_ACTION.DELIVERY)
  if (!includesPayment && !includesDelivery) {
    return { error: 'Choose payment, delivery or both.|Elija pago, entrega o ambos.' }
  }
  if (includesPayment && !(cashBankID > 0)) {
    return { error: 'Select the cash/bank account that receives the payment.|Seleccione la caja/banco que recibe el pago.' }
  }
  if (includesDelivery && !(warehouseID > 0)) {
    return { error: 'Select the warehouse the goods leave from.|Seleccione el almacén del que salen los productos.' }
  }
  const sortedActions = [SETTLE_ACTION.PAYMENT, SETTLE_ACTION.DELIVERY].filter((action) => actionsIncluded.includes(action))
  return {
    request: {
      ID: order.ID,
      ActionsIncluded: sortedActions,
      CashBankID: includesPayment ? cashBankID : 0,
      WarehouseID: includesDelivery ? warehouseID : 0,
    },
  }
}

export const buildAnnulRequest = (
  order: IOnlineSaleOrder, reason: string,
): { request?: IOnlineSaleOrderAnnulRequest; error?: string } => {
  if (!canAnnul(order)) {
    return { error: 'Only a pending order without a sale can be annulled here.|Solo un pedido pendiente y sin venta se puede anular aquí.' }
  }
  const trimmedReason = reason.trim()
  if (!trimmedReason) {
    return { error: 'A reason is required to annul.|Se requiere un motivo para anular.' }
  }
  return { request: { ID: order.ID, Reason: trimmedReason } }
}
