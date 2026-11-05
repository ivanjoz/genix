// Every movement of a period as a journal entry — the operations of "Consolidado & Cuentas".
// Pure: no Svelte, no fetch.
//
// The backend (GET.accounting-operations) sends the period's raw records as numbers, one delta
// log per key (./DELTA_CACHE_PLAN.md). The service adds what they name outside the period (by-IDs
// cache) and the company-wide sets; here each record becomes an operation: its entry lines,
// whether it enters the official Diario, and its status. The user classifies only the nature
// lines (IsNature); IGV, payables, receivables and cash are derived and locked. The rules are
// ./PLAN.md §3.
//
// One builder serves the page, the Diario de gestión and the official 5.2: the official Diario
// is these same operations filtered by InOfficialBook.

import { findPCGEAccount, PCGE_KIND_ASSET, PCGE_KIND_EXPENSE, PCGE_KIND_SUPPLY } from '#core/pcge.ts'
import { quantityAmount, unpackQuantityLine } from '#core/quantity.ts'
import { DOC_TYPE_CREDIT_NOTE } from '#core/sunat-doc-type.ts'
import {
  PURCHASE_DOC_TYPE_NONE, purchaseDocTypeCreditsTax, purchaseDocumentTotal, type IPurchaseDocument,
} from '#core/purchase-document.ts'
import {
  BOOK_STATE_REJECTED, BOOK_STATE_VOIDED, bookDocumentSeriesID, documentValidityState,
  type IBookSeries, type ISalesBookDocument,
} from '../books/books'
import { PURCHASE_SOURCE_EXPENSE, PURCHASE_SOURCE_ORDER, type IPurchaseBookDocument } from '../books/books.purchases'
import { convertUSD, splitByWeights } from './accounts-consolidation.utils'

// Mirrors the Operation* structs in backend/accounting/accounting_operations_api.go.
export interface IOperationSale {
  ID: number
  Date: number
  ss: number
  CurrencyType: number
  ExchangeRate: number
  TotalAmount: number
  TaxAmount: number
  Lines?: { ProductID: number, Amount: number }[]
  SameDayCashBankID?: number
}

// What the builder reads off an asset (the GET.assets record).
export interface IOperationAsset extends IPurchaseDocument {
  ID: number
  Name: string
  SupplierID: number
  PurchaseAmount: number
  AcquisitionValue: number
  AcquisitionDate: number
  DisposalDate: number
  AccumulatedDepreciation: number
  AccountCode: number
  ss: number
  SameDayCashBankID?: number
}

// A reception (3, 4) at its cost, or a sale delivery (8) / annulment return (9) at the cost
// stamped on it. MonetaryValue is positive on both sides. A reception carries the purchase
// account its stock / variation pair follows from.
export interface IOperationStockMovement {
  ID: number
  Date: number
  Type: number
  DocumentID: number
  ProductID: number
  MonetaryValue: number
  PurchaseAccountCode?: number
}

export interface IOperationProduct { ID: number, ss: number, Origin: number, AccountCode: number }

export interface IOperationCashBank { ID: number, Name: string, Type: number, CurrencyType: number }

// A payment of a document (types 6, 8, 9, 10, 11) carries the counter-account it clears.
export interface IOperationCashMovement {
  ID: number
  CashBankID: number
  DocumentID: number
  Date: number
  Type: number
  CounterAccountCode: number
  Amount: number
  CashBankRefID: number
}

// A closed or paid payslip with the accounts stamped on each line at calculation (Kind: 1 earning,
// 2 deduction, 3 employer contribution, 4 provision — backend/hr/types/legal_concepts.go).
export interface IOperationPayslip {
  ID: number
  Period: number
  // ProvisionedAmount: the part of a benefit (gratificación, CTS, vacation pay) charged to the
  // provision it accrued (ProvisionAccountCode 4114 / 415 / 4115) instead of AccountCode.
  // OutsideNetPay: an earning owed to its PayableAccountCode, not through the net pay (the CTS deposit).
  Lines?: {
    Kind: number, AccountCode: number, PayableAccountCode: number, Amount: number
    ProvisionAccountCode?: number, ProvisionedAmount?: number, OutsideNetPay?: boolean
  }[]
  NetPay: number
  ss: number
}

// The period's delta keys, as GET.accounting-operations sends them.
export interface IAccountingOperationsResponse {
  SalesDocuments: ISalesBookDocument[]
  Sales: IOperationSale[]
  PurchaseOrders: IPurchaseBookDocument[]
  Expenses: IPurchaseBookDocument[]   // depreciation included (ExpenseType 4)
  StockMovements: IOperationStockMovement[]
  CashMovements: IOperationCashMovement[]   // the payments settled same-day are not among them
  Payslips?: IOperationPayslip[]            // closed and paid only
}

// Everything the builder reads: the period's keys, what they name outside the period (by-IDs
// cache), and the company-wide sets.
export interface IAccountingSources extends IAccountingOperationsResponse {
  FirstDay: number
  LastDay: number
  ReferencedSales: IOperationSale[]             // the older sales the period's credit notes correct
  ReferencedPurchases: IPurchaseBookDocument[]  // the older orders and expenses its receptions name
  Products: IOperationProduct[]
  Assets: IOperationAsset[]
  CashBanks: IOperationCashBank[]
  SellRates: number[]   // SellRates[i] is the sell rate (× 1000) of FirstDay + i
}

// periodBounds is a "YYYYMM" period's first and last UnixDay.
export function periodBounds(period: string): { firstDay: number, lastDay: number } {
  const year = Number(period.slice(0, 4))
  const monthIndex = Number(period.slice(4, 6)) - 1
  return {
    firstDay: Date.UTC(year, monthIndex, 1) / 86_400_000,
    lastDay: Date.UTC(year, monthIndex + 1, 0) / 86_400_000,
  }
}

// referencedRecordIDs is what the period names outside itself: the sales its credit notes
// correct, and the orders and expenses its receptions name, when the period does not hold them.
export function referencedRecordIDs(response: IAccountingOperationsResponse) {
  const periodSaleIDs = new Set((response.Sales || []).map(sale => sale.ID))
  const periodOrderIDs = new Set((response.PurchaseOrders || []).map(order => order.SourceID))
  const periodExpenseIDs = new Set((response.Expenses || []).map(expense => expense.SourceID))
  const saleIDs = new Set<number>()
  const orderIDs = new Set<number>()
  const expenseIDs = new Set<number>()
  for (const document of response.SalesDocuments || []) {
    if (document.AffectedDocID && !periodSaleIDs.has(document.AffectedDocID)) { saleIDs.add(document.AffectedDocID) }
  }
  for (const movement of response.StockMovements || []) {
    if (movement.Type === STOCK_MOVEMENT_PURCHASE_RECEPTION && !periodOrderIDs.has(movement.DocumentID)) {
      orderIDs.add(movement.DocumentID)
    } else if (movement.Type === STOCK_MOVEMENT_SUPPLIES_PURCHASE && !periodExpenseIDs.has(movement.DocumentID)) {
      expenseIDs.add(movement.DocumentID)
    }
  }
  return { saleIDs: [...saleIDs], orderIDs: [...orderIDs], expenseIDs: [...expenseIDs] }
}

// operationProductIDs are the products whose Origin or AccountCode the entries read: the sale
// lines (7012 / 7022) and the deliveries (the cost and stock accounts).
export function operationProductIDs(sales: IOperationSale[], stockMovements: IOperationStockMovement[]): number[] {
  const productIDs = new Set<number>()
  for (const sale of sales) {
    for (const line of sale.Lines || []) { productIDs.add(line.ProductID) }
  }
  for (const movement of stockMovements) {
    if (movement.Type === STOCK_MOVEMENT_SALE_DELIVERY || movement.Type === STOCK_MOVEMENT_SALE_RETURN) {
      productIDs.add(movement.ProductID)
    }
  }
  return [...productIDs]
}

// The by-IDs records, as much of them as the builder reads.
export interface ISaleOrderRecord {
  ID: number, Date: number, ss: number, upd: number, CurrencyType: number, ExchangeRate: number,
  TotalAmount: number, TaxAmount: number,
  Lines?: { ProductID: number, Quantity: number, Price: number, SubPrice: number }[]
}
export interface IPurchaseRecord extends Partial<IPurchaseDocument> { ID: number, ss: number, upd: number, Name?: string }
export interface IProductRecord extends IOperationProduct { upd: number }

// operationSaleOf is a by-IDs sale as the period's Sales key carries it (makeOperationSale).
export function operationSaleOf(record: ISaleOrderRecord): IOperationSale {
  return {
    ID: record.ID, Date: record.Date, ss: record.ss, CurrencyType: record.CurrencyType,
    ExchangeRate: record.ExchangeRate, TotalAmount: record.TotalAmount, TaxAmount: record.TaxAmount,
    Lines: (record.Lines || []).filter(line => line.ProductID > 0 && line.Quantity > 0).map(line => ({
      ProductID: line.ProductID,
      Amount: quantityAmount(unpackQuantityLine(line.Quantity), line.Price || 0, line.SubPrice || 0),
    })),
  }
}

// referencedPurchaseOf is a by-IDs order or expense, as much as a reception's label reads.
export const referencedPurchaseOf = (source: number, record: IPurchaseRecord): IPurchaseBookDocument => ({
  Source: source, SourceID: record.ID, Name: record.Name || "", ProviderID: 0, ProviderSnapshotID: 0,
  DueDate: 0, RecordAmount: 0, DocType: record.DocType || 0, DocSeries: record.DocSeries || "",
  DocNumber: record.DocNumber || 0, DocIssueDate: record.DocIssueDate || 0, TaxableAmount: 0, TaxAmount: 0,
  UntaxedAmount: 0, OtherAmount: 0, CurrencyType: record.CurrencyType || 0, ExchangeRate: record.ExchangeRate || 0,
})

// What an operation is. The key prefix (CUO) and the label follow from it.
export const OPERATION_SALE = 1
export const OPERATION_CREDIT_NOTE = 2
export const OPERATION_PURCHASE_ORDER = 3
export const OPERATION_EXPENSE = 4
export const OPERATION_RECEPTION = 5
export const OPERATION_ASSET = 6
export const OPERATION_DEPRECIATION = 7
export const OPERATION_DISPOSAL = 8
export const OPERATION_CASH = 9
export const OPERATION_COST_OF_SALES = 10
export const OPERATION_PAYROLL = 11
export const OPERATION_PAYROLL_PROVISION = 12

export const operationKindLabels: Record<number, string> = {
  [OPERATION_SALE]: "Sale|Venta",
  [OPERATION_CREDIT_NOTE]: "Credit note|Nota de crédito",
  [OPERATION_PURCHASE_ORDER]: "Purchase|Compra",
  [OPERATION_EXPENSE]: "Expense|Gasto",
  [OPERATION_RECEPTION]: "Stock reception|Ingreso a almacén",
  [OPERATION_ASSET]: "Fixed asset|Activo fijo",
  [OPERATION_DEPRECIATION]: "Depreciation|Depreciación",
  [OPERATION_DISPOSAL]: "Asset disposal|Baja de activo",
  [OPERATION_CASH]: "Cash movement|Movimiento de caja",
  [OPERATION_COST_OF_SALES]: "Cost of sales|Costo de ventas",
  [OPERATION_PAYROLL]: "Payroll|Planilla",
  [OPERATION_PAYROLL_PROVISION]: "Payroll provisions|Provisiones de planilla",
}

// ./PLAN.md §6.
export const OPERATION_STATUS_CLASSIFIED = 1
export const OPERATION_STATUS_UNCLASSIFIED = 2
export const OPERATION_STATUS_INCOMPLETE = 3

export interface IJournalLine {
  AccountCode: number   // 0 = unclassified
  Debit: number         // PEN cents
  Credit: number
  IsNature: boolean     // the line the user classifies; every other line is derived and locked
}

export interface IAccountingOperation {
  Key: string           // CUO: kind prefix + source id
  Kind: number          // OPERATION_*
  SourceID: number
  SourceType: number    // the cash movement type or the expense type; 0 otherwise
  Date: number          // UnixDay
  Description: string
  DocumentLabel: string // "SERIES-NUMBER"; "" when there is no comprobante
  InOfficialBook: boolean
  Amount: number        // PEN cents: the entry's Debe total
  Lines: IJournalLine[]
  Status: number        // OPERATION_STATUS_*
  Problem: string       // bilingual; why the status is incomplete
}

// The join the backend leaves to the browser: the series (document type) off the company record.
export interface IOperationsContext {
  seriesByID: Map<number, IBookSeries>
}

// The accounts the system picks. PCGE 2019, all postable in backend/accounting/types/pcge_accounts.go.
const ACCOUNT_CASH = 101                     // Caja
const ACCOUNT_BANK = 1041                    // Cuentas corrientes operativas
const ACCOUNT_RECEIVABLE = 1212              // Facturas por cobrar — emitidas en cartera
const ACCOUNT_IGV = 40111                    // IGV — cuenta propia
const ACCOUNT_SUPPLIERS = 4212               // Facturas por pagar — emitidas
const ACCOUNT_OTHER_PAYABLES = 4699          // Otras cuentas por pagar diversas
const ACCOUNT_ASSET_PAYABLES = 4654          // Pasivos por compra de activo inmovilizado — PPE
const ACCOUNT_CAPITAL = 501                  // Capital social
const ACCOUNT_MERCHANDISE_PURCHASE = 6011    // Compras — mercaderías
const ACCOUNT_MERCHANDISE_SALES = 7012       // Ventas — mercaderías, venta local
const ACCOUNT_FINISHED_GOODS_SALES = 7022    // Ventas — productos terminados, venta local
const ACCOUNT_DISPOSAL_COST = 6551           // Costo neto de enajenación de activos inmovilizados
const ACCOUNT_MERCHANDISE_STOCK = 2011       // Mercaderías
const ACCOUNT_FINISHED_GOODS_STOCK = 2111    // Productos terminados
const ACCOUNT_MERCHANDISE_COST = 6912        // Costo de ventas — mercaderías
const ACCOUNT_FINISHED_GOODS_COST = 6922     // Costo de ventas — productos terminados

const CURRENCY_USD = 2
const SALE_STATUS_ANNULLED = 0
const PRODUCT_ORIGIN_MADE = 2
const PRODUCT_STATUS_SUPPLY = 2
const CASH_BANK_TYPE_BANK = 2
const SERIES_DIGITS = 100 // a sale's series sits in its id's last two digits; 00 = no comprobante
// StockMovementType* (backend/logistics/types/stock_movement_type.go).
const STOCK_MOVEMENT_PURCHASE_RECEPTION = 3
const STOCK_MOVEMENT_SUPPLIES_PURCHASE = 4
const STOCK_MOVEMENT_SALE_DELIVERY = 8
const STOCK_MOVEMENT_SALE_RETURN = 9

// CashMovementType* (backend/finance/types/cash_banks.go).
const CASH_TRANSFER = 3
const CASH_SUPPLIER_PAYMENT = 6
const CASH_SALE_COLLECTION = 8
const CASH_EXPENSE_PAYMENT = 9
const CASH_ASSET_PAYMENT = 10
const CASH_SALE_REFUND = 11
const CASH_PAYROLL_PAYMENT = 12
const CASH_PAYROLL_CONTRIBUTIONS = 13

const PROBLEM_MISSING_RATE = "A dollar amount has no exchange rate.|Un monto en dólares no tiene tipo de cambio."
const PROBLEM_UNBALANCED = "Debit and credit do not match.|El Debe y el Haber no cuadran."

interface IOperationBase {
  Key: string
  Kind: number
  SourceID: number
  SourceType?: number
  Date: number
  Description: string
  DocumentLabel: string
  InOfficialBook: boolean
}

// buildAccountingOperations turns the period's records into one operation per source record.
export function buildAccountingOperations(
  sources: IAccountingSources, context: IOperationsContext,
): IAccountingOperation[] {
  const isInPeriod = (day: number) => day >= sources.FirstDay && day <= sources.LastDay
  // A USD amount is restated at the official sell rate of its day, unless its document names one.
  const sellRateOf = (day: number) => sources.SellRates?.[day - sources.FirstDay] || 0

  const salesByID = new Map([...(sources.Sales || []), ...(sources.ReferencedSales || [])].map(sale => [sale.ID, sale]))
  const purchaseKey = (source: number, sourceID: number) => `${source}|${sourceID}`
  const periodPurchases = [...(sources.PurchaseOrders || []), ...(sources.Expenses || [])]
  const purchasesByKey = new Map([...periodPurchases, ...(sources.ReferencedPurchases || [])].map(document =>
    [purchaseKey(document.Source, document.SourceID), document]))
  const assetsByID = new Map((sources.Assets || []).map(asset => [asset.ID, asset]))
  const cashBanksByID = new Map((sources.CashBanks || []).map(cashBank => [cashBank.ID, cashBank]))
  const productsByID = new Map((sources.Products || []).map(product => [product.ID, product]))
  // A sale's id carries its series; 00 means it never had a comprobante.
  const isSaleDocumented = (saleID: number) => saleID % SERIES_DIGITS !== 0

  const operations: IAccountingOperation[] = []

  // Sales with a comprobante: Dr 1212 · Cr 70x per product origin · Cr 40111. A credit note is the
  // same entry reversed. A voided or rejected comprobante declares nothing.
  for (const document of sources.SalesDocuments || []) {
    const validityState = documentValidityState(document.State)
    if (validityState === BOOK_STATE_VOIDED || validityState === BOOK_STATE_REJECTED) { continue }
    const series = context.seriesByID.get(bookDocumentSeriesID(document))
    const isCreditNote = series?.DocType === DOC_TYPE_CREDIT_NOTE
    const sale = salesByID.get(document.AffectedDocID || document.ID)
    const rate = solesRate(document.Currency, sellRateOf(document.IssueDate))
    const revenue = (document.TaxableAmount || 0) + (document.ExemptAmount || 0) + (document.UnaffectedAmount || 0)

    const lines = saleLines(sale, productsByID, convertUSD(revenue, rate), convertUSD(document.TaxAmount || 0, rate))
    operations.push(finishOperation({
      Key: `${isCreditNote ? "NC" : "V"}-${document.ID}`,
      Kind: isCreditNote ? OPERATION_CREDIT_NOTE : OPERATION_SALE,
      SourceID: document.ID, Date: document.IssueDate,
      Description: isCreditNote ? "Credit note on a sale|Nota de crédito de una venta" : "Sale|Venta",
      DocumentLabel: `${series?.SeriesCode || ""}-${document.Correlativo}`,
      InOfficialBook: true,
    }, isCreditNote ? reverseLines(lines) : lines, [
      rate ? "" : PROBLEM_MISSING_RATE,
      sale ? "" : "The sale of this comprobante was not found.|No se encontró la venta de este comprobante.",
    ]))
  }

  // Sales without a comprobante: the same entry, out of the official Diario.
  for (const sale of sources.Sales || []) {
    if (!isInPeriod(sale.Date) || isSaleDocumented(sale.ID) || sale.ss === SALE_STATUS_ANNULLED) { continue }
    const rate = solesRate(sale.CurrencyType, sellRateOf(sale.Date))
    const lines = saleLines(sale, productsByID,
      convertUSD((sale.TotalAmount || 0) - (sale.TaxAmount || 0), rate), convertUSD(sale.TaxAmount || 0, rate))
    operations.push(finishOperation({
      Key: `V-${sale.ID}`, Kind: OPERATION_SALE, SourceID: sale.ID, Date: sale.Date,
      Description: "Sale without comprobante|Venta sin comprobante", DocumentLabel: "", InOfficialBook: false,
    }, lines, [rate ? "" : PROBLEM_MISSING_RATE]))
  }

  // Purchase orders and expenses issued in the period. Depreciation has its own entry below, and
  // referenced older purchases are only lookups.
  for (const document of periodPurchases) {
    if (isDepreciation(document) || !isInPeriod(document.DocIssueDate)) { continue }
    const isOrder = document.Source === PURCHASE_SOURCE_ORDER
    const isDocumented = isPurchaseDocumented(document)
    const rate = solesRate(document.CurrencyType, isDocumented ? document.ExchangeRate : sellRateOf(document.DocIssueDate))
    const { natureAmount, taxAmount } = purchaseAmounts(document, isDocumented ? purchaseDocumentTotal(document) : document.RecordAmount)

    // An order posts per line (a product to 6011, a supply to its line's account); an expense
    // posts to its own account. The order's base is split by line value, remainder on the last.
    const natureSoles = convertUSD(natureAmount, rate)
    const lines: IJournalLine[] = []
    if (isOrder) {
      const orderLines = document.Lines || []
      const shares = splitByWeights(natureSoles, orderLines.map(line => (line.Price || 0) * (line.Quantity || 0)))
      orderLines.forEach((line, index) => {
        lines.push(postLine(line.AccountCode || ACCOUNT_MERCHANDISE_PURCHASE, shares[index], !!line.AccountCode))
      })
    } else {
      lines.push(postLine(document.AccountCode || 0, natureSoles, true))
    }
    const taxSoles = convertUSD(taxAmount, rate)
    lines.push(postLine(ACCOUNT_IGV, taxSoles))
    lines.push(postLine(purchasePayableAccount(document), -(natureSoles + taxSoles)))

    operations.push(finishOperation({
      Key: `${isOrder ? "OC" : "G"}-${document.SourceID}`,
      Kind: isOrder ? OPERATION_PURCHASE_ORDER : OPERATION_EXPENSE,
      SourceID: document.SourceID, SourceType: isOrder ? 0 : document.ExpenseType || 0,
      Date: document.DocIssueDate,
      Description: document.Name || (isOrder ? "Purchase order|Orden de compra" : "Expense|Gasto"),
      DocumentLabel: isDocumented ? `${document.DocSeries}-${document.DocNumber}` : "",
      InOfficialBook: isDocumented,
    }, lines, [rate ? "" : PROBLEM_MISSING_RATE]))
  }

  // Receptions: Dr the stock account · Cr the 61x variation, both the pair of the account the
  // purchase posted to, stamped on the reception. A purchase account with no stock pair (656,
  // 33x…) never enters stock. One operation per document and day; the purchase only labels it.
  const inflowGroups = new Map<string, IOperationStockMovement[]>()
  // Cost of sales: one operation per sale and day, its deliveries and returns together.
  const costGroups = new Map<string, IOperationStockMovement[]>()
  for (const movement of sources.StockMovements || []) {
    const isSaleMovement = movement.Type === STOCK_MOVEMENT_SALE_DELIVERY || movement.Type === STOCK_MOVEMENT_SALE_RETURN
    const groups = isSaleMovement ? costGroups : inflowGroups
    const groupKey = isSaleMovement
      ? `${movement.DocumentID}|${movement.Date}`
      : `${movement.Type}|${movement.DocumentID}|${movement.Date}`
    groups.set(groupKey, [...(groups.get(groupKey) || []), movement])
  }
  for (const inflows of inflowGroups.values()) {
    const { Type: movementType, DocumentID: documentID, Date: date } = inflows[0]
    const source = movementType === STOCK_MOVEMENT_PURCHASE_RECEPTION ? PURCHASE_SOURCE_ORDER : PURCHASE_SOURCE_EXPENSE
    const document = purchasesByKey.get(purchaseKey(source, documentID))
    const lines: IJournalLine[] = []
    let isUncosted = false
    for (const inflow of inflows) {
      const purchaseAccount = findPCGEAccount(inflow.PurchaseAccountCode || 0)
      if (!purchaseAccount?.stockCode || !purchaseAccount.variationCode) { continue }
      isUncosted ||= !inflow.MonetaryValue
      lines.push(postLine(purchaseAccount.stockCode, inflow.MonetaryValue || 0))
      lines.push(postLine(purchaseAccount.variationCode, -(inflow.MonetaryValue || 0)))
    }
    if (lines.length === 0) { continue }
    operations.push(finishOperation({
      Key: `IA-${movementType}-${documentID}-${date}`, Kind: OPERATION_RECEPTION, SourceID: documentID,
      SourceType: movementType, Date: date,
      Description: document?.Name || "Purchase reception|Ingreso por compra",
      DocumentLabel: document && isPurchaseDocumented(document) ? `${document.DocSeries}-${document.DocNumber}` : "",
      InOfficialBook: !document || isPurchaseDocumented(document),
    }, lines, [
      document ? "" : "The purchase of this reception was not found.|No se encontró la compra de este ingreso.",
      isUncosted ? "A reception has no cost.|Un ingreso no tiene costo." : "",
    ]))
  }

  // Cost of sales (docs/SALE_COST_PLAN.md §4): each delivery leaves stock at the cost stamped on
  // it — Dr 6912 · Cr 2011 for merchandise, Dr 6922 · Cr 2111 for a made product, Dr 6912 · Cr
  // the supply's stock account for a supply. An annulment return is the same entry reversed. It
  // follows its sale in or out of the official Diario, like the sale's collections.
  for (const movements of costGroups.values()) {
    const { DocumentID: saleID, Date: date } = movements[0]
    const lines: IJournalLine[] = []
    let isUncosted = false
    let hasNoStockAccount = false
    for (const movement of movements) {
      const product = productsByID.get(movement.ProductID)
      let costAccountCode = ACCOUNT_MERCHANDISE_COST
      let stockAccountCode = ACCOUNT_MERCHANDISE_STOCK
      if (product?.ss === PRODUCT_STATUS_SUPPLY) {
        stockAccountCode = findPCGEAccount(product.AccountCode)?.stockCode || 0
        hasNoStockAccount ||= !stockAccountCode
      } else if (product?.Origin === PRODUCT_ORIGIN_MADE) {
        costAccountCode = ACCOUNT_FINISHED_GOODS_COST
        stockAccountCode = ACCOUNT_FINISHED_GOODS_STOCK
      }
      const isDelivery = movement.Type === STOCK_MOVEMENT_SALE_DELIVERY
      isUncosted ||= isDelivery && !movement.MonetaryValue
      const signedCost = (isDelivery ? 1 : -1) * (movement.MonetaryValue || 0)
      lines.push(postLine(costAccountCode, signedCost))
      lines.push(postLine(stockAccountCode, -signedCost))
    }
    operations.push(finishOperation({
      Key: `CV-${saleID}-${date}`, Kind: OPERATION_COST_OF_SALES, SourceID: saleID, Date: date,
      Description: "Cost of a sale|Costo de una venta", DocumentLabel: "",
      InOfficialBook: isSaleDocumented(saleID),
    }, lines, [
      isUncosted ? "A delivery left the warehouse with no cost.|Una entrega salió del almacén sin costo." : "",
      hasNoStockAccount ? "A delivered supply's account has no stock account.|La cuenta de un insumo entregado no tiene cuenta de existencias." : "",
    ]))
  }

  for (const asset of sources.Assets || []) {
    const assetAccount = findPCGEAccount(asset.AccountCode)

    // Acquisition: Dr 33x · Dr 40111 · Cr 4654. A donated asset (nothing paid, no comprobante)
    // is an owner's contribution: Dr 33x · Cr 501 at its book value.
    const acquisitionDate = asset.DocIssueDate || asset.AcquisitionDate
    if (isInPeriod(acquisitionDate)) {
      const isDocumented = isPurchaseDocumented(asset)
      const isDonation = !asset.PurchaseAmount && !isDocumented
      const rate = solesRate(asset.CurrencyType, isDocumented ? asset.ExchangeRate : sellRateOf(acquisitionDate))
      const lines: IJournalLine[] = []
      if (isDonation) {
        const valueSoles = convertUSD(asset.AcquisitionValue || 0, rate)
        lines.push(postLine(asset.AccountCode || 0, valueSoles, true))
        lines.push(postLine(ACCOUNT_CAPITAL, -valueSoles))
      } else {
        const { natureAmount, taxAmount } = purchaseAmounts(asset, isDocumented ? purchaseDocumentTotal(asset) : asset.PurchaseAmount)
        const natureSoles = convertUSD(natureAmount, rate)
        const taxSoles = convertUSD(taxAmount, rate)
        lines.push(postLine(asset.AccountCode || 0, natureSoles, true))
        lines.push(postLine(ACCOUNT_IGV, taxSoles))
        lines.push(postLine(ACCOUNT_ASSET_PAYABLES, -(natureSoles + taxSoles)))
      }
      operations.push(finishOperation({
        Key: `AF-${asset.ID}`, Kind: OPERATION_ASSET, SourceID: asset.ID, Date: acquisitionDate,
        Description: isDonation ? `${asset.Name} (donated|donado)` : asset.Name,
        DocumentLabel: isDocumented ? `${asset.DocSeries}-${asset.DocNumber}` : "",
        // A donation is not a purchase, so it is an internal entry of the official Diario.
        InOfficialBook: isDocumented || isDonation,
      }, lines, [rate ? "" : PROBLEM_MISSING_RATE]))
    }

    // Disposal: Dr 3952x the accumulated depreciation · Dr 6551 the rest · Cr 33x the value.
    if (isInPeriod(asset.DisposalDate)) {
      const rate = solesRate(asset.CurrencyType, asset.ExchangeRate || sellRateOf(asset.DisposalDate))
      const valueSoles = convertUSD(asset.AcquisitionValue || 0, rate)
      const accumulatedSoles = convertUSD(asset.AccumulatedDepreciation || 0, rate)
      operations.push(finishOperation({
        Key: `BAJA-${asset.ID}`, Kind: OPERATION_DISPOSAL, SourceID: asset.ID, Date: asset.DisposalDate,
        Description: asset.Name, DocumentLabel: "", InOfficialBook: true,
      }, [
        postLine(assetAccount?.accumulatedDepreciationCode || 0, accumulatedSoles),
        postLine(ACCOUNT_DISPOSAL_COST, valueSoles - accumulatedSoles),
        postLine(asset.AccountCode || 0, -valueSoles),
      ], [
        rate ? "" : PROBLEM_MISSING_RATE,
        assetAccount?.accumulatedDepreciationCode ? "" : depreciationPairProblem(asset.AccountCode),
      ]))
    }
  }

  // Depreciation: Dr 6841x · Cr 3952x, the pair of the asset's 33x.
  for (const depreciation of sources.Expenses || []) {
    if (!isDepreciation(depreciation) || !isInPeriod(depreciation.Date || 0)) { continue }
    const asset = assetsByID.get(depreciation.AssetID || 0)
    const assetAccount = findPCGEAccount(asset?.AccountCode || 0)
    const rate = solesRate(asset?.CurrencyType || 0, asset?.ExchangeRate || sellRateOf(depreciation.Date || 0))
    const amountSoles = convertUSD(depreciation.RecordAmount || 0, rate)
    operations.push(finishOperation({
      Key: `DEP-${depreciation.SourceID}`, Kind: OPERATION_DEPRECIATION, SourceID: depreciation.SourceID,
      Date: depreciation.Date || 0, Description: asset?.Name || "", DocumentLabel: "", InOfficialBook: true,
    }, [
      postLine(assetAccount?.depreciationExpenseCode || 0, amountSoles),
      postLine(assetAccount?.accumulatedDepreciationCode || 0, -amountSoles),
    ], [
      rate ? "" : PROBLEM_MISSING_RATE,
      assetAccount?.depreciationExpenseCode ? "" : depreciationPairProblem(asset?.AccountCode || 0),
    ]))
  }

  // Cash: the cash side (101 till / 1041 bank) against the counter-account. A payment carries the
  // counter-account it clears; it enters the official Diario with the document it pays. The
  // payments that settled their document same-day are not read: they are inferred from it.
  const cashEntries = [
    ...(sources.CashMovements || []).map(movement => ({ movement, key: `CJ-${movement.ID}` })),
    ...inferSameDayPayments(sources).map(movement => ({ movement, key: inferredPaymentKey(movement) })),
  ]
  for (const { movement, key } of cashEntries) {
    // A transfer is two rows; the outgoing one carries the whole entry.
    if (movement.Type === CASH_TRANSFER && movement.Amount >= 0) { continue }
    const cashBank = cashBanksByID.get(movement.CashBankID)
    const rate = solesRate(cashBank?.CurrencyType || 0, sellRateOf(movement.Date))
    const amountSoles = convertUSD(movement.Amount || 0, rate)

    let counterAccountCode = movement.CounterAccountCode || 0
    let isNature = true
    let isOfficial = true
    switch (movement.Type) {
      case CASH_TRANSFER: {
        counterAccountCode = cashAccountOf(cashBanksByID.get(movement.CashBankRefID))
        isNature = false
        break
      }
      case CASH_SALE_COLLECTION:
      case CASH_SALE_REFUND: {
        isNature = false
        isOfficial = isSaleDocumented(movement.DocumentID)
        break
      }
      case CASH_SUPPLIER_PAYMENT:
      case CASH_EXPENSE_PAYMENT: {
        isNature = false
        // Only a purchase with a comprobante is owed on 4212.
        isOfficial = counterAccountCode === ACCOUNT_SUPPLIERS
        break
      }
      // Payroll pays a payable its payslips left open (4111, 1412, 1411, 40x, 417, 419).
      case CASH_PAYROLL_PAYMENT:
      case CASH_PAYROLL_CONTRIBUTIONS: {
        isNature = false
        break
      }
      case CASH_ASSET_PAYMENT: {
        const asset = assetsByID.get(movement.DocumentID)
        isNature = false
        isOfficial = !asset || isPurchaseDocumented(asset)
        break
      }
    }
    operations.push(finishOperation({
      Key: key, Kind: OPERATION_CASH, SourceID: movement.ID || movement.DocumentID, SourceType: movement.Type,
      Date: movement.Date, Description: cashBank?.Name || "", DocumentLabel: "", InOfficialBook: isOfficial,
    }, [
      postLine(cashAccountOf(cashBank), amountSoles),
      postLine(counterAccountCode, -amountSoles, isNature),
    ], [rate ? "" : PROBLEM_MISSING_RATE]))
  }

  // Payroll, on the last day of the period, from the accounts each payslip line stamped
  // (plan/HR_PAYROLL_PLAN.md §5.1). H1 + H2: Dr 62x per earning and the employer's 627x ·
  // Cr the deductions' payables (ONP 4032, AFP 417, 5ta 40173, advances 1412, loans 1411…), the
  // employer's 4031 and the net pay owed on 4111. H3: the provisions, Dr 62x · Cr 41x.
  for (const payslip of sources.Payslips || []) {
    const { payrollLines, provisionLines } = payslipJournalLines(payslip)
    const base = { SourceID: payslip.ID, Date: sources.LastDay, DocumentLabel: "", InOfficialBook: true }
    operations.push(finishOperation({ ...base, Key: `PL-${payslip.ID}`, Kind: OPERATION_PAYROLL,
      Description: "Payslip|Boleta de pago" }, payrollLines, []))
    if (provisionLines.length) {
      operations.push(finishOperation({ ...base, Key: `PP-${payslip.ID}`, Kind: OPERATION_PAYROLL_PROVISION,
        Description: "Payroll provisions|Provisiones de planilla" }, provisionLines, []))
    }
  }

  return operations.sort((first, second) => first.Date - second.Date || first.Key.localeCompare(second.Key))
}

const PAYSLIP_LINE_EARNING = 1
const PAYSLIP_LINE_DEDUCTION = 2
const PAYSLIP_LINE_PROVISION = 4
const ACCOUNT_SALARIES_PAYABLE = 4111 // Sueldos y salarios por pagar

// payslipJournalLines splits a payslip into its payroll entry and its provisions entry. Every
// line is posted as stamped: a missing account leaves the line unclassified, never guessed.
export function payslipJournalLines(payslip: IOperationPayslip) {
  const payrollLines: IJournalLine[] = []
  const provisionLines: IJournalLine[] = []
  for (const line of payslip.Lines || []) {
    if (line.Kind === PAYSLIP_LINE_EARNING) {
      // A benefit settles its provision first (H4/H5); only the excess is this period's expense.
      const provisioned = line.ProvisionAccountCode ? line.ProvisionedAmount || 0 : 0
      if (provisioned) payrollLines.push(postLine(line.ProvisionAccountCode!, provisioned))
      if (line.Amount > provisioned) payrollLines.push(postLine(line.AccountCode, line.Amount - provisioned))
      // Not in the net pay: the whole amount stays owed on its payable until it is deposited.
      if (line.OutsideNetPay) payrollLines.push(postLine(line.PayableAccountCode, -line.Amount))
    } else if (line.Kind === PAYSLIP_LINE_DEDUCTION) {
      payrollLines.push(postLine(line.PayableAccountCode, -line.Amount))
    } else {
      // An employer contribution or a provision: Dr its expense · Cr who is owed it.
      const target = line.Kind === PAYSLIP_LINE_PROVISION ? provisionLines : payrollLines
      target.push(postLine(line.AccountCode, line.Amount), postLine(line.PayableAccountCode, -line.Amount))
    }
  }
  payrollLines.push(postLine(ACCOUNT_SALARIES_PAYABLE, -payslip.NetPay))
  return { payrollLines, provisionLines }
}

// saleLines is a sale's entry: Dr 1212 · Cr 7012 / 7022 · Cr 40111. The revenue is split by the
// value of the lines whose product is made (7022) against the rest (7012).
function saleLines(
  sale: IOperationSale | undefined, productsByID: Map<number, IOperationProduct>,
  revenueSoles: number, taxSoles: number,
): IJournalLine[] {
  let boughtValue = 0
  let madeValue = 0
  for (const line of sale?.Lines || []) {
    if (productsByID.get(line.ProductID)?.Origin === PRODUCT_ORIGIN_MADE) {
      madeValue += line.Amount || 0
    } else {
      boughtValue += line.Amount || 0
    }
  }
  // With nothing made the whole revenue lands on 7012 (splitByWeights' last share).
  const [madeRevenue, boughtRevenue] = splitByWeights(revenueSoles, [madeValue, boughtValue])
  return [
    postLine(ACCOUNT_RECEIVABLE, revenueSoles + taxSoles),
    postLine(ACCOUNT_FINISHED_GOODS_SALES, -madeRevenue),
    postLine(ACCOUNT_MERCHANDISE_SALES, -boughtRevenue),
    postLine(ACCOUNT_IGV, -taxSoles),
  ]
}

const isPurchaseDocumented = (document: Partial<IPurchaseDocument>): boolean =>
  (document.DocType || PURCHASE_DOC_TYPE_NONE) !== PURCHASE_DOC_TYPE_NONE

// purchaseAmounts splits a purchase's total into what is cost and what is IGV credit. Only a
// comprobante that grants crédito fiscal separates the IGV; without one the whole total is cost.
function purchaseAmounts(document: Partial<IPurchaseDocument>, total: number) {
  const creditsTax = isPurchaseDocumented(document) && purchaseDocTypeCreditsTax(document.DocType || 0)
  const taxAmount = creditsTax ? document.TaxAmount || 0 : 0
  return { natureAmount: (total || 0) - taxAmount, taxAmount }
}

// A purchase with a comprobante is owed as an issued invoice; without one, as another payable.
const purchasePayableAccount = (document: IPurchaseBookDocument): number =>
  isPurchaseDocumented(document) ? ACCOUNT_SUPPLIERS : ACCOUNT_OTHER_PAYABLES

const EXPENSE_TYPE_DEPRECIATION = 4
const isDepreciation = (document: IPurchaseBookDocument): boolean =>
  document.Source === PURCHASE_SOURCE_EXPENSE && document.ExpenseType === EXPENSE_TYPE_DEPRECIATION

// inferSameDayPayments is the payment each document of the period was settled with on its own
// accounting date (SameDayCashBankID ≠ 0): one movement at that cash bank for the document's
// whole total — the movement the period's CashMovements leave out (finance.SameDaySettlement).
export function inferSameDayPayments(sources: IAccountingSources): IOperationCashMovement[] {
  const payment = (type: number, documentID: number, date: number, cashBankID: number, amount: number,
    counterAccountCode: number): IOperationCashMovement => ({
    ID: 0, CashBankID: cashBankID, DocumentID: documentID, Date: date, Type: type,
    CounterAccountCode: counterAccountCode, Amount: amount, CashBankRefID: 0,
  })
  const payments: IOperationCashMovement[] = []
  for (const sale of sources.Sales || []) {
    if (!sale.SameDayCashBankID) { continue }
    payments.push(payment(CASH_SALE_COLLECTION, sale.ID, sale.Date, sale.SameDayCashBankID,
      sale.TotalAmount || 0, ACCOUNT_RECEIVABLE))
  }
  for (const document of [...(sources.PurchaseOrders || []), ...(sources.Expenses || [])]) {
    if (!document.SameDayCashBankID) { continue }
    const isOrder = document.Source === PURCHASE_SOURCE_ORDER
    payments.push(payment(isOrder ? CASH_SUPPLIER_PAYMENT : CASH_EXPENSE_PAYMENT, document.SourceID,
      document.DocIssueDate || document.Date || 0, document.SameDayCashBankID, -(document.RecordAmount || 0),
      purchasePayableAccount(document)))
  }
  for (const asset of sources.Assets || []) {
    const accountingDate = asset.DocIssueDate || asset.AcquisitionDate
    if (!asset.SameDayCashBankID || accountingDate < sources.FirstDay || accountingDate > sources.LastDay) { continue }
    payments.push(payment(CASH_ASSET_PAYMENT, asset.ID, accountingDate, asset.SameDayCashBankID,
      -(asset.PurchaseAmount || 0), ACCOUNT_ASSET_PAYABLES))
  }
  return payments
}

// An inferred payment has no movement ID: its CUO names the document it settles.
const inferredPaymentKey = (movement: IOperationCashMovement): string =>
  `CJD-${movement.Type}-${movement.DocumentID}`

const cashAccountOf = (cashBank: IOperationCashBank | undefined): number =>
  cashBank?.Type === CASH_BANK_TYPE_BANK ? ACCOUNT_BANK : ACCOUNT_CASH

// solesRate is the rate × 1000 an amount is restated with: 1000 in soles, the given rate in
// dollars, and 0 when a dollar amount has none — which zeroes it and flags the operation.
const solesRate = (currencyType: number, rate: number): number =>
  currencyType === CURRENCY_USD ? rate || 0 : 1000

const depreciationPairProblem = (accountCode: number): string =>
  `Account ${accountCode} has no depreciation accounts.|La cuenta ${accountCode} no tiene cuentas de depreciación.`

// postLine is a line on the side its sign says: positive is Debe, negative is Haber.
function postLine(accountCode: number, signedAmount: number, isNature = false): IJournalLine {
  return {
    AccountCode: accountCode, Debit: Math.max(0, signedAmount), Credit: Math.max(0, -signedAmount),
    IsNature: isNature,
  }
}

const reverseLines = (lines: IJournalLine[]): IJournalLine[] =>
  lines.map(line => ({ ...line, Debit: line.Credit, Credit: line.Debit }))

// finishOperation drops the empty lines, merges the lines that hit the same account on the same
// side, and decides the status: a problem makes it incomplete, an account left at 0 makes it
// unclassified, and an entry whose sides differ is incomplete too.
function finishOperation(base: IOperationBase, lines: IJournalLine[], problems: string[]): IAccountingOperation {
  const mergedLines: IJournalLine[] = []
  for (const line of lines) {
    if (!line.Debit && !line.Credit) { continue }
    const sameSide = mergedLines.find(merged => merged.AccountCode === line.AccountCode &&
      merged.IsNature === line.IsNature && !!merged.Debit === !!line.Debit)
    if (sameSide) {
      sameSide.Debit += line.Debit
      sameSide.Credit += line.Credit
    } else {
      mergedLines.push({ ...line })
    }
  }

  const debitTotal = mergedLines.reduce((sum, line) => sum + line.Debit, 0)
  const creditTotal = mergedLines.reduce((sum, line) => sum + line.Credit, 0)
  let problem = problems.filter(Boolean).join(" ")
  let status = OPERATION_STATUS_CLASSIFIED
  if (!problem && debitTotal !== creditTotal) { problem = PROBLEM_UNBALANCED }
  if (problem) {
    status = OPERATION_STATUS_INCOMPLETE
  } else if (mergedLines.some(line => !line.AccountCode)) {
    status = OPERATION_STATUS_UNCLASSIFIED
  }
  return {
    ...base, SourceType: base.SourceType || 0, Amount: debitTotal, Lines: mergedLines,
    Status: status, Problem: problem,
  }
}

// One nature account the user can set, and the record field it lands on
// (PUT.accounting-classification). Kind is the classifiable operation kind; Line is a purchase
// order line's index.
export interface IClassificationTarget {
  Kind: number
  ID: number
  Line: number
  Label: string
  AccountCode: number
  PCGEKind: number          // PCGE_KIND_*; 0 for a cash movement, whose accounts are its type's options
  CashMovementType: number  // a cash movement's current type
  CashAmount: number        // a cash movement's signed amount; its options depend on the sign
}

const CLASSIFIABLE_CASH_TYPES = [1, 2, 4, 5, 7] // unspecified, count, withdrawal, loss, collection
const EXPENSE_TYPE_INVENTORY = 2

// classificationTargetsOf lists what the user can classify in an operation. Derived entries that
// take their accounts from a classifiable record point to it: depreciation and a disposal to
// their asset. Everything else has no target — its accounts are fixed.
export function classificationTargetsOf(
  operation: IAccountingOperation, sources: IAccountingSources,
): IClassificationTarget[] {
  const baseTarget = { Line: 0, CashMovementType: 0, CashAmount: 0 }
  switch (operation.Kind) {
    case OPERATION_EXPENSE: {
      const expense = (sources.Expenses || []).find(document => document.SourceID === operation.SourceID)
      if (!expense) { return [] }
      return [{
        ...baseTarget, Kind: OPERATION_EXPENSE, ID: expense.SourceID, Label: expense.Name || "",
        AccountCode: expense.AccountCode || 0,
        PCGEKind: expense.ExpenseType === EXPENSE_TYPE_INVENTORY ? PCGE_KIND_SUPPLY : PCGE_KIND_EXPENSE,
      }]
    }
    case OPERATION_PURCHASE_ORDER: {
      const order = (sources.PurchaseOrders || []).find(document => document.SourceID === operation.SourceID)
      // Only supply lines carry an account; a product line is always 6011.
      return (order?.Lines || []).flatMap((line, lineIndex) => line.AccountCode ? [{
        ...baseTarget, Kind: OPERATION_PURCHASE_ORDER, ID: order!.SourceID, Line: lineIndex,
        Label: `Line|Línea ${lineIndex + 1}`, AccountCode: line.AccountCode, PCGEKind: PCGE_KIND_SUPPLY,
      }] : [])
    }
    case OPERATION_ASSET:
    case OPERATION_DEPRECIATION:
    case OPERATION_DISPOSAL: {
      const assetID = operation.Kind === OPERATION_DEPRECIATION
        ? (sources.Expenses || []).find(depreciation => depreciation.SourceID === operation.SourceID)?.AssetID
        : operation.SourceID
      const asset = (sources.Assets || []).find(candidate => candidate.ID === assetID)
      if (!asset) { return [] }
      return [{
        ...baseTarget, Kind: OPERATION_ASSET, ID: asset.ID, Label: asset.Name, AccountCode: asset.AccountCode || 0,
        PCGEKind: PCGE_KIND_ASSET,
      }]
    }
    case OPERATION_CASH: {
      // Matched by its CUO: an inferred payment carries its document's ID, never a movement's.
      const movement = (sources.CashMovements || []).find(candidate => `CJ-${candidate.ID}` === operation.Key)
      if (!movement || !CLASSIFIABLE_CASH_TYPES.includes(movement.Type)) { return [] }
      return [{
        Kind: OPERATION_CASH, ID: movement.ID, Line: 0, Label: operation.Description,
        AccountCode: movement.CounterAccountCode || 0, PCGEKind: 0,
        CashMovementType: movement.Type, CashAmount: movement.Amount,
      }]
    }
  }
  return []
}

// bulkClassificationTargets gathers the targets of the selected operations for one account. A
// bulk apply needs one PCGE kind for all of them, so the picker can offer a single list; cash
// movements are left out, since each type has its own options.
export function bulkClassificationTargets(
  operations: IAccountingOperation[], sources: IAccountingSources,
): { pcgeKind: number, targets: IClassificationTarget[], skippedCount: number } {
  const targets: IClassificationTarget[] = []
  let skippedCount = 0
  for (const operation of operations) {
    const operationTargets = classificationTargetsOf(operation, sources).filter(target => target.PCGEKind)
    if (operationTargets.length === 0) { skippedCount++ }
    targets.push(...operationTargets)
  }
  const pcgeKinds = new Set(targets.map(target => target.PCGEKind))
  return { pcgeKind: pcgeKinds.size === 1 ? targets[0].PCGEKind : 0, targets, skippedCount }
}

// The views Movimientos filters the operations by.
export const STATUS_VIEW_ALL = 1
export const STATUS_VIEW_UNCLASSIFIED = 2
export const STATUS_VIEW_INCOMPLETE = 3
export const STATUS_VIEW_UNDOCUMENTED = 4

export const statusViewOptions = [
  { ID: STATUS_VIEW_ALL, Name: "All|Todas" },
  { ID: STATUS_VIEW_UNCLASSIFIED, Name: "Unclassified|Sin clasificar" },
  { ID: STATUS_VIEW_INCOMPLETE, Name: "Incomplete|Incompletas" },
  { ID: STATUS_VIEW_UNDOCUMENTED, Name: "No comprobante|Sin comprobante" },
]

export function isInStatusView(operation: IAccountingOperation, statusView: number): boolean {
  if (statusView === STATUS_VIEW_UNCLASSIFIED) { return operation.Status === OPERATION_STATUS_UNCLASSIFIED }
  if (statusView === STATUS_VIEW_INCOMPLETE) { return operation.Status === OPERATION_STATUS_INCOMPLETE }
  if (statusView === STATUS_VIEW_UNDOCUMENTED) { return !operation.InOfficialBook }
  return true
}

// One account on one side of a group, its amount summed over the group's operations.
export interface IGroupAccountLine {
  AccountCode: number   // 0 = unclassified
  Amount: number        // PEN cents
}

// A row of Movimientos: the operations that share a day, a kind, a glosa, the same accounts on
// the same sides, a status and the official-Diario flag. Sales of a day read as one row.
export interface IOperationGroup {
  Key: string
  Date: number
  Kind: number
  Description: string
  InOfficialBook: boolean
  Status: number
  Operations: IAccountingOperation[]
  DebitLines: IGroupAccountLine[]
  CreditLines: IGroupAccountLine[]
}

// groupOperations folds the operations into Movimientos rows, keeping the operations' order.
// The account signature is sorted, so the order a builder posts its lines in never splits a group.
export function groupOperations(operations: IAccountingOperation[]): IOperationGroup[] {
  const groupsByKey = new Map<string, IOperationGroup>()
  for (const operation of operations) {
    const accountSignature = operation.Lines
      .map(line => `${line.Debit ? "D" : "C"}${line.AccountCode}`)
      .sort().join(",")
    const groupKey = [operation.Date, operation.Kind, operation.Description, operation.InOfficialBook,
      operation.Status, accountSignature].join("|")
    let group = groupsByKey.get(groupKey)
    if (!group) {
      group = {
        Key: groupKey, Date: operation.Date, Kind: operation.Kind, Description: operation.Description,
        InOfficialBook: operation.InOfficialBook, Status: operation.Status, Operations: [],
        DebitLines: [], CreditLines: [],
      }
      groupsByKey.set(groupKey, group)
    }
    group.Operations.push(operation)
    addEntryLines(group, operation.Lines)
  }
  return [...groupsByKey.values()]
}

// entrySides is one operation's entry as the Diario shows it: its Debe and its Haber accounts.
export function entrySides(operation: IAccountingOperation): Pick<IOperationGroup, "DebitLines" | "CreditLines"> {
  const sides = { DebitLines: [], CreditLines: [] }
  addEntryLines(sides, operation.Lines)
  return sides
}

// addEntryLines adds each line to its side, summed into the side's line for the same account.
function addEntryLines(sides: Pick<IOperationGroup, "DebitLines" | "CreditLines">, lines: IJournalLine[]) {
  for (const line of lines) {
    const sideLines = line.Debit ? sides.DebitLines : sides.CreditLines
    const accountLine = sideLines.find(candidate => candidate.AccountCode === line.AccountCode)
    if (accountLine) {
      accountLine.Amount += line.Debit + line.Credit
    } else {
      sideLines.push({ AccountCode: line.AccountCode, Amount: line.Debit + line.Credit })
    }
  }
}

// One account of the period's balance de comprobación.
export interface ITrialBalanceRow {
  AccountCode: number       // 0 = the lines still unclassified
  Depth: number             // 0 for a 2-digit account, +1 per digit
  IsPostable: boolean
  Debit: number
  Credit: number
  Balance: number           // Debe − Haber
}

// buildTrialBalance sums the period's lines per account and rolls each sum up to every parent
// of the chart, so the tree reads like the PCGE: 40111 adds to 4011, 401 and 40. Only accounts
// with movement are listed, in chart order; the unclassified lines go last as account 0.
export function buildTrialBalance(operations: IAccountingOperation[]): ITrialBalanceRow[] {
  const rowsByCode = new Map<number, ITrialBalanceRow>()
  const addTo = (accountCode: number, debit: number, credit: number) => {
    const row = rowsByCode.get(accountCode) || {
      AccountCode: accountCode, Depth: accountCode ? String(accountCode).length - 2 : 0,
      IsPostable: !!findPCGEAccount(accountCode)?.postable, Debit: 0, Credit: 0, Balance: 0,
    }
    row.Debit += debit
    row.Credit += credit
    row.Balance = row.Debit - row.Credit
    rowsByCode.set(accountCode, row)
  }
  for (const operation of operations) {
    for (const line of operation.Lines) {
      addTo(line.AccountCode, line.Debit, line.Credit)
      for (let parentCode = Math.floor(line.AccountCode / 10); parentCode >= 10; parentCode = Math.floor(parentCode / 10)) {
        if (findPCGEAccount(parentCode)) { addTo(parentCode, line.Debit, line.Credit) }
      }
    }
  }
  // Comparing the codes as text is the chart's tree order: 10, 101, 1041, 12, 1212…
  return [...rowsByCode.values()].sort((first, second) => {
    if (!first.AccountCode || !second.AccountCode) { return second.AccountCode - first.AccountCode }
    return String(first.AccountCode).localeCompare(String(second.AccountCode))
  })
}

// The page header (./PLAN.md §6).
export interface IOperationsSummary {
  count: number
  classified: number
  unclassified: number
  incomplete: number
  debitTotal: number
  creditTotal: number
  undocumentedCount: number     // out of the official Diario
  undocumentedAmount: number
}

export function summarizeOperations(operations: IAccountingOperation[]): IOperationsSummary {
  const summary: IOperationsSummary = {
    count: operations.length, classified: 0, unclassified: 0, incomplete: 0,
    debitTotal: 0, creditTotal: 0, undocumentedCount: 0, undocumentedAmount: 0,
  }
  for (const operation of operations) {
    if (operation.Status === OPERATION_STATUS_CLASSIFIED) {
      summary.classified++
    } else if (operation.Status === OPERATION_STATUS_UNCLASSIFIED) {
      summary.unclassified++
    } else {
      summary.incomplete++
    }
    for (const line of operation.Lines) {
      summary.debitTotal += line.Debit
      summary.creditTotal += line.Credit
    }
    if (!operation.InOfficialBook) {
      summary.undocumentedCount++
      summary.undocumentedAmount += operation.Amount
    }
  }
  return summary
}
