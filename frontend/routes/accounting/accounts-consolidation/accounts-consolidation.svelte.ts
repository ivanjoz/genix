import { hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
import { getRecordsByID } from '@genix/ui/cache'
import { extractError, GET, PUT } from '#libs/ui-runtime.svelte.ts'
import { tr } from '#core/store.svelte.ts'
import {
  cajaMovimientoTipos, cashMovementAccountOptions,
} from '../../finance/cash-banks/cajas.svelte'
import { PURCHASE_SOURCE_EXPENSE, PURCHASE_SOURCE_ORDER, type IPurchaseBookDocument } from '../books/books.purchases'
import {
  operationProductIDs, operationSaleOf, referencedPurchaseOf, referencedRecordIDs,
  type IAccountingOperationsResponse, type IClassificationTarget, type IOperationProduct, type IOperationSale,
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
        useCache: { min: 0, ver: 2 },
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
