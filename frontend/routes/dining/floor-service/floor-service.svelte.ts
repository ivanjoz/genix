import { EmpresaParametrosService } from '#routes/company/configuration/empresas.svelte.ts'
import {
  CURRENCY_PEN, EXCHANGE_RATE_MAX_AGE_DAYS, calendarMonthKeys, effectiveExchangeRate, latestSellRate,
} from '#routes/finance/exchange-rate/exchange-rate.ts'
import { BcrpDefaultRates, ExchangeRatesService } from '#routes/finance/exchange-rate/exchange-rate.svelte.ts'
import { ProductStockSimpleService } from '#routes/logistics/products-stock/stock-movement.ts'
import { exchangeRateSpread } from '#routes/sales/sale_order_create/sale_order.ts'
import { GET, POST } from '#libs/ui-runtime.svelte.ts'
import { todayUnixDay } from '#libs/helpers.ts'
import type { ITableOrder } from '../dining.ts'
import type { IRoundDraftLine, makeRoundRequest } from './floor-service.ts'
import type { ITableChargeRequest } from './table-charge.ts'

// The round being composed for the selected table, shared by "Nuevo pedido" (RoundComposer, which
// restarts it each time it mounts) and the menu picker the page shows beside the table's layer.
export class RoundDraftState {
  draft = $state<IRoundDraftLine[]>([])
  // True while "Nuevo pedido" is on screen: the menu picker shows only then.
  isComposing = $state(false)
}

// What the menu (diningMenu) is built from besides the catalog, all delta-cached so it paints from
// IndexedDB before the sync answers: the stock rows (every warehouse; the menu keeps the layout's)
// and the rate a USD-priced product is restated in soles with, read the way the server reads it
// (latest sell rate, own or BCRP, plus the company's spread).
export class DiningMenuSources {
  stockService = new ProductStockSimpleService(true)
  exchangeRatesService = new ExchangeRatesService(true)
  companyService = new EmpresaParametrosService()
  bcrpDefaultRates = new BcrpDefaultRates()
  penRate = $derived(effectiveExchangeRate(
    latestSellRate(this.exchangeRatesService.records, this.bcrpDefaultRates.ratesByMonthKey, todayUnixDay(), EXCHANGE_RATE_MAX_AGE_DAYS),
    exchangeRateSpread(this.companyService.empresa.FlagValues), CURRENCY_PEN))

  constructor() {
    this.bcrpDefaultRates.load(calendarMonthKeys(new Date(), 2))
  }
}

// A round reserves stock and a charge or annulment settles or releases it: the menu's units left
// must sync again after any of them.
const TABLE_ORDER_REFRESH_ROUTES = ['dining-table-orders', 'products-stock']

// The orders a table received since yesterday's midnight, newest first ("Historial").
export const getTableOrderHistory = async (layoutID: number, tableShapeID: number) => {
  const orders = await GET({ route: `dining-table-order-history?layout-id=${layoutID}&table-shape-id=${tableShapeID}` }) as ITableOrder[]
  return orders || []
}

// Adds a round to a table, opening its order when it has none. The answer is the whole order.
export const postTableOrderRound = (request: ReturnType<typeof makeRoundRequest>) => {
  return POST({
    route: 'dining-table-order-round',
    data: request,
    refreshRoutes: TABLE_ORDER_REFRESH_ROUTES,
  }) as Promise<ITableOrder>
}

// Closes the bill into a sale. The order comes back charged (ss 2), with its SaleOrderID.
export const postTableOrderCharge = (request: ITableChargeRequest) => {
  return POST({
    route: 'dining-table-order-charge',
    data: request,
    refreshRoutes: TABLE_ORDER_REFRESH_ROUTES,
  }) as Promise<{ Order: ITableOrder, SaleOrder: { ID: number } }>
}

// Annuls an open bill, releasing its reservations. The order comes back annulled (ss 0).
export const postTableOrderAnnul = (request: { OrderID: number, Reason: string }) => {
  return POST({
    route: 'dining-table-order-annul',
    data: request,
    refreshRoutes: TABLE_ORDER_REFRESH_ROUTES,
  }) as Promise<ITableOrder>
}
