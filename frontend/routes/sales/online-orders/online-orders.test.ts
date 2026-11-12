import { describe, expect, it } from 'vitest'
import type { ICashBank } from '#routes/finance/cash-banks/cajas.svelte.ts'
import type { IWarehouse } from '#routes/business/branches-warehouses/branches-warehouses.svelte.ts'
import {
  DELIVERY_TYPE, ONLINE_ORDER_GROUP, ONLINE_ORDER_STATUS, SETTLE_ACTION, buildAnnulRequest, buildSettleRequest, canAnnul,
  canSettle, customerName, defaultDeliveryWarehouseID, defaultPaymentCashBankID, needsCulqiReconciliation,
  onlineOrderStatusLabel, statusGroupOf, type IOnlineSaleOrder,
} from './online-orders.ts'

const makeOrder = (overrides: Partial<IOnlineSaleOrder> = {}): IOnlineSaleOrder => ({
  ID: 7, Date: 20000, Lines: [], TotalAmount: 5000, DebtAmount: 5000, ShippingAmount: 1000, ExchangeRate: 0,
  DeliveryType: DELIVERY_TYPE.HOME, PickupSiteID: 0, WebCustomerID: 1, ClientID: 0, PaymentMethod: 3,
  CulqiChargeID: '', SaleOrderID: 0, AnnulReason: '', ss: ONLINE_ORDER_STATUS.PENDING, Created: 0, upd: 0, ver: 0,
  ...overrides,
})

const cashBank = (ID: number, overrides: Partial<ICashBank> = {}): ICashBank => ({
  ID, SiteID: 0, Name: `c${ID}`, Description: '', CurrencyType: 1, ReconciliationDate: 0, CurrentAmount: 0,
  ReconciliationAmount: 0, Type: 1, ReceivesCulqiPayments: false, ss: 1, upd: 0, ...overrides,
})

const warehouse = (ID: number, SiteID: number, overrides: Partial<IWarehouse> = {}): IWarehouse => ({
  ID, SiteID, Name: `w${ID}`, Description: '', Layout: [], ExcludeFromOnlineSales: false, ss: 1, upd: 0, ...overrides,
})

describe('online-orders', () => {
  it('maps every status to its tab', () => {
    expect(statusGroupOf(ONLINE_ORDER_STATUS.PENDING)).toBe(ONLINE_ORDER_GROUP.PENDING)
    expect(statusGroupOf(ONLINE_ORDER_STATUS.PAID)).toBe(ONLINE_ORDER_GROUP.IN_PROGRESS)
    expect(statusGroupOf(ONLINE_ORDER_STATUS.DELIVERED)).toBe(ONLINE_ORDER_GROUP.IN_PROGRESS)
    expect(statusGroupOf(ONLINE_ORDER_STATUS.COMPLETED)).toBe(ONLINE_ORDER_GROUP.COMPLETED)
    expect(statusGroupOf(ONLINE_ORDER_STATUS.ANNULLED)).toBe(ONLINE_ORDER_GROUP.ANNULLED)
    expect(onlineOrderStatusLabel(9)).toBe('Unknown|Desconocido')
  })

  it('only accepts actions on a pending order with no sale', () => {
    expect(canSettle(makeOrder())).toBe(true)
    expect(canAnnul(makeOrder())).toBe(true)
    expect(canSettle(makeOrder({ SaleOrderID: 1301101 }))).toBe(false)
    expect(canSettle(makeOrder({ ss: ONLINE_ORDER_STATUS.PAID }))).toBe(false)
    expect(canAnnul(makeOrder({ ss: ONLINE_ORDER_STATUS.ANNULLED }))).toBe(false)
  })

  it('flags a Culqi charge whose sale was never created', () => {
    expect(needsCulqiReconciliation(makeOrder({ CulqiChargeID: 'chr_1' }))).toBe(true)
    expect(needsCulqiReconciliation(makeOrder())).toBe(false)
    expect(needsCulqiReconciliation(makeOrder({ CulqiChargeID: 'chr_1', SaleOrderID: 5 }))).toBe(false)
    expect(needsCulqiReconciliation(makeOrder({ CulqiChargeID: 'chr_1', ss: ONLINE_ORDER_STATUS.ANNULLED }))).toBe(false)
  })

  it('defaults payment to the Culqi account only for a Culqi charge, among active PEN accounts', () => {
    const cashBanks = [cashBank(1, { ss: 0 }), cashBank(2, { CurrencyType: 2 }), cashBank(3), cashBank(4, { ReceivesCulqiPayments: true })]
    expect(defaultPaymentCashBankID(makeOrder(), cashBanks)).toBe(3)
    expect(defaultPaymentCashBankID(makeOrder({ CulqiChargeID: 'chr_1' }), cashBanks)).toBe(4)
    expect(defaultPaymentCashBankID(makeOrder(), [])).toBe(0)
  })

  it('defaults delivery to a warehouse of the pickup store', () => {
    const warehouses = [warehouse(1, 10, { ExcludeFromOnlineSales: true }), warehouse(2, 10), warehouse(3, 20)]
    expect(defaultDeliveryWarehouseID(makeOrder(), warehouses)).toBe(2)
    expect(defaultDeliveryWarehouseID(makeOrder({ DeliveryType: DELIVERY_TYPE.PICKUP, PickupSiteID: 20 }), warehouses)).toBe(3)
  })

  it('builds the settle request and validates its account and warehouse', () => {
    const order = makeOrder()
    expect(buildSettleRequest(order, [SETTLE_ACTION.PAYMENT], 4, 9).request)
      .toEqual({ ID: 7, ActionsIncluded: [2], CashBankID: 4, WarehouseID: 0 })
    expect(buildSettleRequest(order, [SETTLE_ACTION.DELIVERY, SETTLE_ACTION.PAYMENT], 4, 9).request)
      .toEqual({ ID: 7, ActionsIncluded: [2, 3], CashBankID: 4, WarehouseID: 9 })
    expect(buildSettleRequest(order, [SETTLE_ACTION.PAYMENT], 0, 9).error).toContain('caja/banco')
    expect(buildSettleRequest(order, [SETTLE_ACTION.DELIVERY], 4, 0).error).toContain('almacén')
    expect(buildSettleRequest(order, [], 4, 9).error).toBeTruthy()
    expect(buildSettleRequest(makeOrder({ SaleOrderID: 3 }), [SETTLE_ACTION.PAYMENT], 4, 9).error).toBeTruthy()
  })

  it('requires an annul reason', () => {
    expect(buildAnnulRequest(makeOrder(), '  duplicado  ').request).toEqual({ ID: 7, Reason: 'duplicado' })
    expect(buildAnnulRequest(makeOrder(), '   ').error).toBeTruthy()
  })

  it('names the customer from the shipping snapshot', () => {
    const shipping = { FirstName: 'Ana', LastName: 'Ruiz', Email: '', CityID: 0, Address: '', Reference: '', Phone: '', DocumentNumber: '' }
    expect(customerName(makeOrder({ Shipping: shipping }))).toBe('Ana Ruiz')
    expect(customerName(makeOrder())).toBe('-')
  })
})
