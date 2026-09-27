// The Registro de Compras (SIRE RCE), in the shape SUNAT reads it. Pure: no Svelte, no fetch.
//
// The backend sends the period's purchases from the three tables that carry a supplier
// comprobante — purchase orders, expenses and fixed assets — as numbers. The supplier's
// identity comes from the by-ids snapshot cache, and everything else is decided here: which
// purchases are book rows, how a factura split across several records becomes one row, and
// what the period still needs before it can be filed.
//
// Field numbers in the comments are Anexo 11 of RS 112-2021. See ./PLAN.md §2.2 and
// ../docs/SUNAT-RCE-Diario-annexes-summary.md §1.4.

import { sunatIdentityDocCode } from '$services/crm/identity-doc'
import type { IClientProviderSnapshot } from '$services/crm/client-provider.svelte'
import {
  PURCHASE_DOC_TYPE_NONE, PURCHASE_DOC_TYPE_UTILITY_BILL, purchaseDocumentTotal,
  type IPurchaseDocument,
} from '$core/purchase-document'
import { isWholeMonthPeriod } from './books'

// Where a row came from. Mirrors PurchaseSource* in backend/accounting/purchases_book_api.go.
export const PURCHASE_SOURCE_ORDER = 1
export const PURCHASE_SOURCE_EXPENSE = 2
export const PURCHASE_SOURCE_ASSET = 3

export const purchaseSourceLabels: Record<number, string> = {
  [PURCHASE_SOURCE_ORDER]: "Purchase order|Orden de compra",
  [PURCHASE_SOURCE_EXPENSE]: "Expense|Gasto",
  [PURCHASE_SOURCE_ASSET]: "Fixed asset|Activo fijo",
}

// Mirrors PurchaseBookDocument in backend/accounting/purchases_book_api.go.
export interface IPurchaseBookDocument extends IPurchaseDocument {
  Source: number
  SourceID: number
  Name: string
  ProviderID: number
  ProviderSnapshotID: number
  DueDate: number
  RecordAmount: number
}

export interface IPurchasesBookResponse {
  Period: string
  FirstDay: number
  LastDay: number
  Documents: IPurchaseBookDocument[]
}

export interface IPurchaseSource {
  Source: number
  SourceID: number
}

export interface IPurchaseBookRow {
  Period: string
  IssueDate: number
  DueDate: number          // 0 unless the document is a recibo de servicios públicos (field 6)
  DocType: number          // catalog id, not SUNAT's "01"
  Series: string
  Number: number
  SupplierDocType: string
  SupplierDocNumber: string
  SupplierName: string
  TaxableAmount: number    // cents, in the document's currency
  TaxAmount: number
  UntaxedAmount: number
  OtherAmount: number
  TotalAmount: number
  Currency: string         // ISO 4217
  ExchangeRate: number     // × 1000; 0 in soles
  Sources: IPurchaseSource[]
}

// A purchase with no comprobante. Not a book row: listed behind a checkbox for review.
export interface IUndocumentedPurchase extends IPurchaseSource {
  Date: number
  Name: string
  SupplierName: string
  Amount: number
  Currency: string
}

export interface IPurchasesBook {
  rows: IPurchaseBookRow[]
  undocumented: IUndocumentedPurchase[]
}

const CURRENCY_USD = 2
const currencyCode = (currencyType: number): string => currencyType === CURRENCY_USD ? "USD" : "PEN"

// buildPurchasesBook splits the period's purchases into book rows and purchases without a
// comprobante, and folds the records that share one comprobante into a single row.
//
// Several records can carry the same document: an acquisition of five serial-tracked assets
// is five rows sharing one factura, each holding its share. Anexo 11 forbids duplicates, so the
// book keys a document by supplier, type, series and number and sums the shares back.
export function buildPurchasesBook(
  response: IPurchasesBookResponse,
  identityBySnapshotID: Map<number, IClientProviderSnapshot>,
): IPurchasesBook {
  const rowsByDocumentKey = new Map<string, IPurchaseBookRow>()
  const undocumented: IUndocumentedPurchase[] = []

  for (const document of response.Documents || []) {
    const identity = identityBySnapshotID.get(document.ProviderSnapshotID)

    if ((document.DocType || PURCHASE_DOC_TYPE_NONE) === PURCHASE_DOC_TYPE_NONE) {
      undocumented.push({
        Source: document.Source, SourceID: document.SourceID,
        // Without a comprobante the column holds the purchase's own date.
        Date: document.DocIssueDate,
        Name: document.Name || "",
        SupplierName: identity?.Name || "",
        Amount: document.RecordAmount || 0,
        Currency: currencyCode(document.CurrencyType),
      })
      continue
    }

    const documentKey = [document.ProviderID, document.DocType, document.DocSeries, document.DocNumber].join("|")
    const existing = rowsByDocumentKey.get(documentKey)
    if (existing) {
      existing.TaxableAmount += document.TaxableAmount || 0
      existing.TaxAmount += document.TaxAmount || 0
      existing.UntaxedAmount += document.UntaxedAmount || 0
      existing.OtherAmount += document.OtherAmount || 0
      existing.TotalAmount += purchaseDocumentTotal(document)
      existing.Sources.push({ Source: document.Source, SourceID: document.SourceID })
      continue
    }

    rowsByDocumentKey.set(documentKey, {
      Period: response.Period,
      IssueDate: document.DocIssueDate,
      DueDate: document.DocType === PURCHASE_DOC_TYPE_UTILITY_BILL ? document.DueDate || 0 : 0,
      DocType: document.DocType,
      Series: document.DocSeries || "",
      Number: document.DocNumber || 0,
      SupplierDocType: identity ? sunatIdentityDocCode(identity.IdentityDocType) : "",
      SupplierDocNumber: identity?.RegistryNumber || "",
      SupplierName: identity?.Name || "",
      TaxableAmount: document.TaxableAmount || 0,
      TaxAmount: document.TaxAmount || 0,
      UntaxedAmount: document.UntaxedAmount || 0,
      OtherAmount: document.OtherAmount || 0,
      TotalAmount: purchaseDocumentTotal(document),
      Currency: currencyCode(document.CurrencyType),
      ExchangeRate: document.CurrencyType === CURRENCY_USD ? document.ExchangeRate || 0 : 0,
      Sources: [{ Source: document.Source, SourceID: document.SourceID }],
    })
  }

  const rows = [...rowsByDocumentKey.values()].sort((first, second) =>
    first.IssueDate - second.IssueDate ||
    first.Series.localeCompare(second.Series) ||
    first.Number - second.Number,
  )
  undocumented.sort((first, second) => first.Date - second.Date)
  return { rows, undocumented }
}

export const purchaseRowDocument = (row: IPurchaseBookRow): string => `${row.Series}-${row.Number}`

// toSoles converts an amount of the row's currency, which is what the Formulario 621 totals are
// in. A dollar row with no rate converts to 0 rather than to a made-up figure; the export
// blockers already stop that period.
export const toSoles = (cents: number, row: { Currency: string, ExchangeRate: number }): number =>
  row.Currency === "PEN" ? cents : Math.round(cents * (row.ExchangeRate || 0) / 1000)

export interface IPurchasesBookTotals {
  taxable: number
  tax: number
  untaxed: number
  total: number
  comprobantes: number
}

// The figures in soles that tie to the crédito fiscal of the month.
export function sumPurchasesBook(rows: IPurchaseBookRow[]): IPurchasesBookTotals {
  const totals: IPurchasesBookTotals = { taxable: 0, tax: 0, untaxed: 0, total: 0, comprobantes: 0 }
  for (const row of rows || []) {
    totals.taxable += toSoles(row.TaxableAmount, row)
    totals.tax += toSoles(row.TaxAmount, row)
    totals.untaxed += toSoles(row.UntaxedAmount, row)
    totals.total += toSoles(row.TotalAmount, row)
    totals.comprobantes += 1
  }
  return totals
}

export interface IPurchasesBookIssuer {
  RUC: string
  LegalName: string
}

// purchasesBookExportBlockers lists every reason the period cannot be written as a file, in
// the bilingual form the page shows through tr(). An empty list means it can.
//
// A period with no purchases is not one of them: it is filed with the "sin información" flag.
// Purchases without a comprobante are not either — they are simply not part of the book.
export function purchasesBookExportBlockers(
  rows: IPurchaseBookRow[], issuer: IPurchasesBookIssuer, period: string,
): string[] {
  const blockers: string[] = []

  if (!isWholeMonthPeriod(period)) {
    blockers.push(
      "A single day is not a filable book — pick a month.|" +
      "Un día suelto no es un libro presentable — elige un mes.",
    )
  }
  if (!/^\d{11}$/.test(issuer.RUC || "")) {
    blockers.push("The company has no valid RUC (field 1).|La empresa no tiene un RUC válido (campo 1).")
  }

  // Fields 12-14 are mandatory on every detailed row. A supplier with no pinned identity — one
  // nobody has saved since identities were introduced — would go out blank.
  const unidentifiedCount = rows.filter(row => !row.SupplierDocNumber).length
  if (unidentifiedCount > 0) {
    blockers.push(
      `${unidentifiedCount} comprobantes have no supplier identity (fields 12-14) — save the supplier in CRM.|` +
      `${unidentifiedCount} comprobantes no tienen la identidad del proveedor (campos 12-14) — guarda el proveedor en CRM.`,
    )
  }

  const missingRateCount = rows.filter(row => row.Currency !== "PEN" && !row.ExchangeRate).length
  if (missingRateCount > 0) {
    blockers.push(
      `${missingRateCount} comprobantes in dollars have no exchange rate (field 27).|` +
      `${missingRateCount} comprobantes en dólares no tienen tipo de cambio (campo 27).`,
    )
  }

  const missingDueDateCount = rows.filter(row =>
    row.DocType === PURCHASE_DOC_TYPE_UTILITY_BILL && !row.DueDate).length
  if (missingDueDateCount > 0) {
    blockers.push(
      `${missingDueDateCount} utility bills have no due date (field 6).|` +
      `${missingDueDateCount} recibos de servicios públicos no tienen fecha de vencimiento (campo 6).`,
    )
  }

  return blockers
}
