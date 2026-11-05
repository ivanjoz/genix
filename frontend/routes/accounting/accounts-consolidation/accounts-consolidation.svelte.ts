import { hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
import { getRecordsByID } from '@genix/ui/cache'
import { extractError, GET, PUT } from '#libs/ui-runtime.svelte.ts'
import { tr } from '#core/store.svelte.ts'
import {
  cajaMovimientoTipos, cashMovementAccountOptions,
  CajasService,
} from '../../finance/cash-banks/cajas.svelte'
import { EmpresaParametrosService } from '../../company/configuration/empresas.svelte'
import { BcrpDefaultRates, ExchangeRatesService } from '../../finance/exchange-rate/exchange-rate.svelte'
import { EXCHANGE_RATE_MAX_AGE_DAYS, latestSellRate, toMonthKey } from '../../finance/exchange-rate/exchange-rate'
import { AssetsService } from '../assets/assets.svelte'
import { PURCHASE_SOURCE_EXPENSE, PURCHASE_SOURCE_ORDER, type IPurchaseBookDocument } from '../books/books.purchases'
import {
  buildAccountingOperations, operationProductIDs, operationSaleOf, periodBounds, referencedPurchaseOf, referencedRecordIDs,
  type IAccountingOperationsResponse, type IAccountingSources, type IClassificationTarget, type IOperationProduct, type IOperationSale,
  type IProductRecord, type IPurchaseRecord, type ISaleOrderRecord,
} from './accounts-consolidation'

// The period's records, and what they name outside it, rebuilt into operations by the page.
//
// The period comes through the delta cache: every key is a delta log of the period
// (./DELTA_CACHE_PLAN.md), so reopening a month only travels what changed. `min: 0` asks the
// server every time — the TTL would otherwise skip the network and show a stale month.
export class AccountingOperationsService {
  response: IAccountingOperationsResponse | null = $state(null)
  referencedSales: IOperationSale[] = $state([])
  referencedPurchases: IPurchaseBookDocument[] = $state([])
  products: IOperationProduct[] = $state([])
  loadedPeriod = $state("")

  async fetch(period: string): Promise<void> {
    if (!period) { return }
    showLoading(tr("Reading the period...|Leyendo el periodo..."))
    try {
      const response = await GET({
        route: `accounting-operations?period=${period}`,
        // ver 4: payslip lines carry the provision they settle.
        useCache: { min: 0, ver: 4 },
        // A purchase document is keyed by the order's or the expense's own ID.
        keysIDs: { PurchaseOrders: "SourceID", Expenses: "SourceID" },
      }) as IAccountingOperationsResponse

      // What the period names outside itself, through the by-IDs cache.
      const { saleIDs, orderIDs, expenseIDs } = referencedRecordIDs(response)
      const [salesByID, ordersByID, expensesByID] = await Promise.all([
        saleIDs.length ? getRecordsByID<ISaleOrderRecord>("sale-order-by-ids", saleIDs) : new Map(),
        orderIDs.length ? getRecordsByID<IPurchaseRecord>("purchase-order-ids", orderIDs) : new Map(),
        expenseIDs.length ? getRecordsByID<IPurchaseRecord>("expense-ids", expenseIDs) : new Map(),
      ])
      const referencedSales = [...salesByID.values()].map(operationSaleOf)
      const productIDs = operationProductIDs([...(response.Sales || []), ...referencedSales], response.StockMovements || [])
      const productsByID = productIDs.length
        ? await getRecordsByID<IProductRecord>("p-products-ids", productIDs)
        : new Map<number, IProductRecord>()

      this.referencedSales = referencedSales
      this.referencedPurchases = [
        ...[...ordersByID.values()].map(order => referencedPurchaseOf(PURCHASE_SOURCE_ORDER, order)),
        ...[...expensesByID.values()].map(expense => referencedPurchaseOf(PURCHASE_SOURCE_EXPENSE, expense)),
      ]
      this.products = [...productsByID.values()]
      this.response = response
      this.loadedPeriod = period
    } catch (error) {
      this.response = null
      notifyFailure(extractError(error) || tr("The period could not be read.|No se pudo leer el periodo."))
    } finally {
      hideLoading()
    }
  }
}

// AccountingPeriodReader reads one period and builds its operations: the period's records, plus
// the company-wide sets the entries read (each through its own delta cache). Consolidado & Cuentas
// and the Estados Financieros read a period the same way, so both show the same entries.
export class AccountingPeriodReader {
  operationsService = new AccountingOperationsService()
  // The series (and so the document type) of a comprobante live on the company record.
  companyParameters = new EmpresaParametrosService()
  assetsService = new AssetsService(true)
  cashBanksService = new CajasService()
  exchangeRatesService = new ExchangeRatesService(true)
  bcrpRates = new BcrpDefaultRates()

  seriesByID = $derived(new Map(
    (this.companyParameters.empresa.InvoiceSeries || []).map(series => [series.SeriesID, series]),
  ))
  sources = $derived.by((): IAccountingSources | null => {
    const response = this.operationsService.response
    if (!response) { return null }
    const { firstDay, lastDay } = periodBounds(this.operationsService.loadedPeriod)
    return {
      ...response, FirstDay: firstDay, LastDay: lastDay,
      ReferencedSales: this.operationsService.referencedSales,
      ReferencedPurchases: this.operationsService.referencedPurchases,
      Products: this.operationsService.products,
      Assets: this.assetsService.records,
      CashBanks: this.cashBanksService.Cajas,
      // The rate a USD amount of each day is restated at — the same rule the sale applies.
      SellRates: Array.from({ length: lastDay - firstDay + 1 }, (_, dayIndex) => this.sellRateOf(firstDay + dayIndex)),
    }
  })
  operations = $derived(this.sources ? buildAccountingOperations(this.sources, { seriesByID: this.seriesByID }) : [])

  // The official sell rate (× 1000) of a day, the company's own else BCRP's, up to a week old.
  sellRateOf(day: number): number {
    return latestSellRate(this.exchangeRatesService.records, this.bcrpRates.ratesByMonthKey, day, EXCHANGE_RATE_MAX_AGE_DAYS)
  }

  // read loads the BCRP fallback rates of the period's month and the one before it (a rate may be
  // a week old), and the period's records.
  read(period: string): Promise<void> {
    const { firstDay } = periodBounds(period)
    const firstDate = new Date(firstDay * 86_400_000)
    const monthKey = toMonthKey(firstDate.getUTCFullYear(), firstDate.getUTCMonth())
    const previousDate = new Date((firstDay - 1) * 86_400_000)
    this.bcrpRates.load([monthKey, toMonthKey(previousDate.getUTCFullYear(), previousDate.getUTCMonth())])
    return this.operationsService.fetch(period)
  }
}

// One line of PUT.accounting-classification. Mirrors AccountClassification in
// backend/accounting/accounting_classification_api.go.
export interface IAccountClassification {
  Kind: number
  ID: number
  Line: number
  AccountCode: number
  CashMovementType: number
}

// A classification target as the entry layer edits it.
export interface IEditableTarget extends IClassificationTarget {
  CashOptionKey: string // a cash target's pick, "<type>|<account>"
}

export const putAccountClassifications =(classifications: IAccountClassification[]) =>
  PUT({ data: classifications, route: "accounting-classification" })

const CASH_UNSPECIFIED = 1
const CASH_WITHDRAWAL = 4
const CASH_LOSS = 5
const CASH_COLLECTION = 7

export interface ICashClassificationOption {
  key: string             // "<movement type>|<account>"
  name: string
  movementType: number
  accountCode: number
}

// cashClassificationOptions is what a cash movement may be classified as. An unspecified movement
// is retyped by the option picked, into a type the user could have registered by hand with that
// sign: money out → a withdrawal or a loss, money in → a collection. The backend applies the same
// rule (applyClassification).
export function cashClassificationOptions(target: IClassificationTarget): ICashClassificationOption[] {
  const candidateTypes = target.CashMovementType !== CASH_UNSPECIFIED
    ? [target.CashMovementType]
    : target.CashAmount < 0 ? [CASH_WITHDRAWAL, CASH_LOSS] : [CASH_COLLECTION]
  return candidateTypes.flatMap(movementType => {
    const typeName = cajaMovimientoTipos.find(movementTypeOption => movementTypeOption.id === movementType)?.name || ""
    return cashMovementAccountOptions(movementType, target.CashAmount).map(option => ({
      key: `${movementType}|${option.accountCode}`,
      name: `${typeName} · ${option.accountCode} ${tr(option.name)}`,
      movementType, accountCode: option.accountCode,
    }))
  })
}
