// The Registro de Ventas, in the shape SUNAT reads it. Pure: no Svelte, no fetch.
//
// The backend sends the period's comprobantes as numbers and nothing else. Both joins happen
// here: the series — and with it the document type — comes off the company record the app
// already holds, and the buyer comes from the by-ids identity cache. The SUNAT rules live here
// too, because this is where the joined data is: the S/ 700 threshold, the daily consolidation
// of boletas into a contiguous range, and the zeroing of a comprobante SUNAT refused.
//
// Everything below is a pure function of its arguments, so every rule is testable with no
// component and no network — see books.test.ts.
//
// Field numbers in the comments are Anexo 3 of RS 112-2021. See ./PLAN.md.

import { DOC_TYPE_BOLETA } from '#core/sunat-doc-type.ts'
import { sunatIdentityDocCode } from '#services/crm/identity-doc.ts'
import type { IClientProviderSnapshot } from '#services/crm/client-provider.svelte.ts'

export const BOOK_SALES = "ventas"
export const BOOK_PURCHASES = "compras"
export const BOOK_JOURNAL = "diario"

export type BookKind = typeof BOOK_SALES | typeof BOOK_PURCHASES | typeof BOOK_JOURNAL

export const bookOptions: { ID: BookKind, Name: string }[] = [
  { ID: BOOK_SALES, Name: "Sales|Ventas" },
  { ID: BOOK_PURCHASES, Name: "Purchases|Compras" },
  { ID: BOOK_JOURNAL, Name: "Simplified Journal|Diario Simplificado" },
]

// Tabla 9 of RS 112-2021. Not a SUNAT field — it decides whether a row carries its amounts.
// BOOK_STATE_PENDING is not one of SUNAT's: it is a comprobante this ERP has not had answered,
// which cannot be declared as anything, so the page shows it and the period is not filable.
export const BOOK_STATE_ACTIVE = 1
export const BOOK_STATE_VOIDED = 2
export const BOOK_STATE_REJECTED = 3
export const BOOK_STATE_PENDING = 4

export const bookStateLabels: Record<number, string> = {
  [BOOK_STATE_ACTIVE]: "Active|Activo",
  [BOOK_STATE_VOIDED]: "Voided|Baja",
  [BOOK_STATE_REJECTED]: "Rejected|Rechazado",
  [BOOK_STATE_PENDING]: "Awaiting SUNAT|Pendiente SUNAT",
}

export const bookStateCss: Record<number, string> = {
  [BOOK_STATE_ACTIVE]: "text-green-600",
  [BOOK_STATE_VOIDED]: "text-gray-500",
  [BOOK_STATE_REJECTED]: "text-red-600",
  [BOOK_STATE_PENDING]: "text-amber-600",
}

// ISalesBookDocument is what the backend sends: the comprobante as numbers, with the two ids
// this file joins on. Mirrors SalesBookDocument in backend/accounting/sales_book_api.go.
export interface ISalesBookDocument {
  ID: number
  IssueDate: number       // UnixDay
  Correlativo: number
  ClientSnapshotID: number
  Currency: number
  TaxableAmount: number   // cents
  TaxAmount: number
  ExemptAmount: number
  UnaffectedAmount: number
  TotalAmount: number
  AffectedDocID: number
  State: number
}

export interface ISalesBookResponse {
  Period: string
  FirstDay: number
  LastDay: number
  Documents: ISalesBookDocument[]
  AffectedDocuments: ISalesBookDocument[]
  UninvoicedSales: IUninvoicedSale[]
}

// Only what a book row reads off a series. A structural type rather than IInvoiceSeries so this
// leaf does not import from another route to say it — the company record satisfies it as is.
export interface IBookSeries {
  SeriesID: number
  DocType: number
  SeriesCode: string
}

export interface ISalesBookRow {
  Period: string
  IssueDate: number       // UnixDay
  DocType: number         // DOC_TYPE_* — the catalog id, not SUNAT's "01"
  Series: string
  Number: number
  NumberFinal: number     // 0 unless the row consolidates a range
  BuyerDocType: string
  BuyerDocNumber: string
  BuyerName: string
  TaxableAmount: number   // cents
  TaxAmount: number
  ExemptAmount: number
  UnaffectedAmount: number
  TotalAmount: number
  Currency: string
  ModifiedIssueDate: number
  ModifiedDocType: number
  ModifiedSeries: string
  ModifiedNumber: number
  ValidityState: number
  ConsolidatedCount: number
  DocumentID: number
}

export interface IUninvoicedSale {
  ID: number
  Date: number
  TotalAmount: number
  TaxAmount: number
  ss: number
}

// SERIES_DIGITS is the width the series occupies at the tail of a document id, the same way
// invoicing.ts reads it. Mirrors SeriesDigits in backend/invoicing/types/invoice_document.go.
const SERIES_DIGITS = 100

export const bookDocumentSeriesID = (document: ISalesBookDocument): number =>
  document.ID % SERIES_DIGITS

// BOLETA_IDENTIFIED_FROM is the S/ 700 threshold in cents, above which a boleta must name its
// buyer and therefore cannot be folded into a range. Mirrors BoletaIdentifiedFrom in
// backend/invoicing/types/customer_identity.go.
export const BOLETA_IDENTIFIED_FROM = 70_000

// The send states the ERP tracks, as invoicing.ts numbers them. Only the three that map onto
// tabla 9 are named here; everything else is a comprobante SUNAT has not answered.
const INVOICE_VOIDED = 0
const INVOICE_ACCEPTED = 3
const INVOICE_OBSERVED = 4
const INVOICE_REJECTED = 5

// How an unidentified buyer is declared on a boleta. The document printed these three values,
// so the book prints them too — it may not invent different ones. Mirrors the Anonymous*
// constants in backend/invoicing/types/sale_order_to_cpe.go.
export const ANONYMOUS_DOC_TYPE = "0"
export const ANONYMOUS_DOC_NUMBER = "00000000"
export const ANONYMOUS_BUYER_NAME = "CLIENTES VARIOS"

// buildSalesBook turns the period's comprobantes into the book, consolidating what SUNAT
// allows to be consolidated and leaving everything else detailed.
//
// seriesByID and identityBySnapshotID are the two joins the backend deliberately did not do.
// A missing series or identity leaves its columns empty rather than throwing: a book that
// renders with a blank cell is debuggable, a page that fails to render is not.
export function buildSalesBook(
  response: ISalesBookResponse,
  seriesByID: Map<number, IBookSeries>,
  identityBySnapshotID: Map<number, IClientProviderSnapshot>,
): ISalesBookRow[] {
  const affectedByID = new Map<number, ISalesBookDocument>()
  for (const document of [...(response.Documents || []), ...(response.AffectedDocuments || [])]) {
    affectedByID.set(document.ID, document)
  }

  const detailed = (response.Documents || []).map(document => buildDetailedRow(
    document, response.Period, seriesByID, identityBySnapshotID, affectedByID,
  ))

  return consolidateBoletas(detailed).sort((first, second) =>
    first.IssueDate - second.IssueDate ||
    first.Series.localeCompare(second.Series) ||
    first.Number - second.Number,
  )
}

function buildDetailedRow(
  document: ISalesBookDocument,
  period: string,
  seriesByID: Map<number, IBookSeries>,
  identityBySnapshotID: Map<number, IClientProviderSnapshot>,
  affectedByID: Map<number, ISalesBookDocument>,
): ISalesBookRow {
  const series = seriesByID.get(bookDocumentSeriesID(document))
  const validityState = documentValidityState(document.State)

  const row: ISalesBookRow = {
    Period: period,
    IssueDate: document.IssueDate,
    DocType: series?.DocType || 0,
    Series: series?.SeriesCode || "",
    Number: document.Correlativo,
    NumberFinal: 0,
    BuyerDocType: "", BuyerDocNumber: "", BuyerName: "",
    TaxableAmount: 0, TaxAmount: 0, ExemptAmount: 0, UnaffectedAmount: 0, TotalAmount: 0,
    Currency: sunatCurrencyCode(document.Currency),
    ModifiedIssueDate: 0, ModifiedDocType: 0, ModifiedSeries: "", ModifiedNumber: 0,
    ValidityState: validityState,
    ConsolidatedCount: 0,
    DocumentID: document.ID,
  }

  // Nota 3 del Anexo 3: a comprobante whose CDR was not "aceptado", or whose baja was
  // communicated, is anotado with zeros. It keeps its row — the correlativo was spent and
  // SUNAT expects to see it — but it declares no operation.
  if (validityState === BOOK_STATE_ACTIVE || validityState === BOOK_STATE_PENDING) {
    row.TaxableAmount = document.TaxableAmount || 0
    row.TaxAmount = document.TaxAmount || 0
    row.ExemptAmount = document.ExemptAmount || 0
    row.UnaffectedAmount = document.UnaffectedAmount || 0
    row.TotalAmount = document.TotalAmount || 0
  }

  applyBuyer(document, identityBySnapshotID, row)
  applyModifiedDocument(document, seriesByID, affectedByID, row)
  return row
}

// applyBuyer writes columns 11-13.
//
// A document with no pinned identity is a till boleta nobody was asked to identify, and it
// declared SUNAT's unidentified buyer. The book has to print exactly what the document
// printed, so it reads the same constants rather than leaving the columns empty.
function applyBuyer(
  document: ISalesBookDocument,
  identityBySnapshotID: Map<number, IClientProviderSnapshot>,
  row: ISalesBookRow,
): void {
  if (!document.ClientSnapshotID) {
    row.BuyerDocType = ANONYMOUS_DOC_TYPE
    row.BuyerDocNumber = ANONYMOUS_DOC_NUMBER
    row.BuyerName = ANONYMOUS_BUYER_NAME
    return
  }
  const identity = identityBySnapshotID.get(document.ClientSnapshotID)
  if (!identity) return

  row.BuyerDocType = sunatIdentityDocCode(identity.IdentityDocType)
  row.BuyerDocNumber = identity.RegistryNumber
  row.BuyerName = identity.Name
}

// applyModifiedDocument writes columns 29-32 for a credit or debit note.
function applyModifiedDocument(
  document: ISalesBookDocument,
  seriesByID: Map<number, IBookSeries>,
  affectedByID: Map<number, ISalesBookDocument>,
  row: ISalesBookRow,
): void {
  if (!document.AffectedDocID) return
  const affected = affectedByID.get(document.AffectedDocID)
  if (!affected) return

  const affectedSeries = seriesByID.get(bookDocumentSeriesID(affected))
  row.ModifiedIssueDate = affected.IssueDate
  row.ModifiedDocType = affectedSeries?.DocType || 0
  row.ModifiedSeries = affectedSeries?.SeriesCode || ""
  row.ModifiedNumber = affected.Correlativo
}

// consolidateBoletas collapses every maximal run of contiguous boletas that may be declared
// as one range.
//
// A consolidated row declares a range of correlativos — Nro Inicial and Nro Final — so it is
// not simply "the day's small boletas". Three things end a run, and each of them becomes its
// own detailed row:
//
//   - a boleta of S/ 700 or more, which SUNAT requires detailed with its buyer;
//   - a boleta SUNAT rejected or whose baja was communicated, which has to appear at zero and
//     cannot do that while hidden inside somebody else's range;
//   - a gap in the correlativo sequence.
//
// A run of one is left as the detailed row it already was: a range of a single number says
// nothing a plain row does not.
function consolidateBoletas(detailed: ISalesBookRow[]): ISalesBookRow[] {
  const groups = new Map<string, ISalesBookRow[]>()
  const rows: ISalesBookRow[] = []

  for (const row of detailed) {
    if (!isConsolidable(row)) {
      rows.push(row)
      continue
    }
    const groupKey = `${row.IssueDate}|${row.Series}`
    const group = groups.get(groupKey)
    if (group) { group.push(row) } else { groups.set(groupKey, [row]) }
  }

  // Map iteration is insertion-ordered, so the groups come out in the order the day's first
  // boleta appeared — and buildSalesBook sorts the result anyway.
  for (const group of groups.values()) {
    group.sort((first, second) => first.Number - second.Number)

    let runStart = 0
    for (let index = 1; index <= group.length; index++) {
      const isContiguous = index < group.length && group[index].Number === group[index - 1].Number + 1
      if (isContiguous) continue
      rows.push(mergeRun(group.slice(runStart, index)))
      runStart = index
    }
  }
  return rows
}

// isConsolidable is whether a row may be folded into a range at all.
//
// An identified buyer does not prevent it: SUNAT's only threshold for a boleta is the S/ 700
// one, and the identity the ERP holds is not lost — it stays on the sale and on the document,
// it simply does not travel to the book.
export const isConsolidable = (row: ISalesBookRow): boolean =>
  row.DocType === DOC_TYPE_BOLETA &&
  row.ValidityState === BOOK_STATE_ACTIVE &&
  row.TotalAmount < BOLETA_IDENTIFIED_FROM

// mergeRun turns a contiguous run into the single row that declares it.
function mergeRun(run: ISalesBookRow[]): ISalesBookRow {
  if (run.length === 1) return run[0]

  const merged: ISalesBookRow = {
    ...run[0],
    NumberFinal: run[run.length - 1].Number,
    ConsolidatedCount: run.length,
    // The range is the row now, not any one document in it.
    DocumentID: 0,
    BuyerDocType: "", BuyerDocNumber: "", BuyerName: "",
  }

  for (const row of run.slice(1)) {
    merged.TaxableAmount += row.TaxableAmount
    merged.TaxAmount += row.TaxAmount
    merged.ExemptAmount += row.ExemptAmount
    merged.UnaffectedAmount += row.UnaffectedAmount
    merged.TotalAmount += row.TotalAmount
  }
  return merged
}

// documentValidityState reads tabla 9's validity off the send state the ERP tracks.
export function documentValidityState(sendState: number): number {
  switch (sendState) {
    case INVOICE_ACCEPTED:
    case INVOICE_OBSERVED:
      return BOOK_STATE_ACTIVE
    case INVOICE_VOIDED:
      return BOOK_STATE_VOIDED
    case INVOICE_REJECTED:
      return BOOK_STATE_REJECTED
  }
  return BOOK_STATE_PENDING
}

// sunatCurrencyCode maps the stored currency to ISO 4217, which is what field 27 wants.
const CURRENCY_USD = 2
const sunatCurrencyCode = (currency: number): string => currency === CURRENCY_USD ? "USD" : "PEN"

// The number as the book declares it: a single correlativo, or the range a consolidated row
// stands for (fields 9 and 10).
export const bookRowNumber = (row: ISalesBookRow): string =>
  row.NumberFinal > 0 ? `${row.Number} — ${row.NumberFinal}` : String(row.Number)

export const bookRowDocument = (row: ISalesBookRow): string =>
  `${row.Series}-${bookRowNumber(row)}`

// The document a note corrects (fields 29-32). Empty for everything else.
export const bookRowModifiedDocument = (row: ISalesBookRow): string =>
  row.ModifiedSeries ? `${row.ModifiedSeries}-${row.ModifiedNumber}` : ""

export interface IBookTotals {
  taxable: number
  tax: number
  exempt: number
  unaffected: number
  total: number
  comprobantes: number
}

// The figure that has to tie to the Formulario 621, which is the reason anyone opens this page
// at month end. Consolidated rows count as the boletas they stand for, not as one document.
export function sumSalesBook(rows: ISalesBookRow[]): IBookTotals {
  const totals: IBookTotals = {
    taxable: 0, tax: 0, exempt: 0, unaffected: 0, total: 0, comprobantes: 0,
  }
  for (const row of rows || []) {
    totals.taxable += row.TaxableAmount || 0
    totals.tax += row.TaxAmount || 0
    totals.exempt += row.ExemptAmount || 0
    totals.unaffected += row.UnaffectedAmount || 0
    totals.total += row.TotalAmount || 0
    totals.comprobantes += row.ConsolidatedCount || 1
  }
  return totals
}

// A period cannot be filed while it still holds documents SUNAT has not answered: they would
// be declared as operations that may yet be rejected.
export const countPendingRows = (rows: ISalesBookRow[]): number =>
  (rows || []).filter(row => row.ValidityState === BOOK_STATE_PENDING).length

export interface IPeriodOption {
  // The SUNAT period itself, YYYYMM (field 3). Empty means "a single day".
  ID: string
  Name: string
}

const monthNames = [
  "ENE", "FEB", "MAR", "ABR", "MAY", "JUN", "JUL", "AGO", "SET", "OCT", "NOV", "DIC",
]

// SINGLE_DAY_PERIOD is the option that hands control back to the date picker.
//
// A word and not "": SearchSelect reads a falsy id as "nothing is selected", so an empty
// string would leave the option unselectable and the field blank after picking it.
export const SINGLE_DAY_PERIOD = "day"

// isWholeMonthPeriod is whether the selection names a period, as opposed to the single-day
// escape hatch. It is what locks the date picker and what decides the shape of the request.
export const isWholeMonthPeriod = (period: string): boolean =>
  !!period && period !== SINGLE_DAY_PERIOD

// buildPeriodOptions lists the months a book can be asked for, newest first.
//
// The value is the SUNAT period verbatim, so what the selector holds and what the request
// sends are the same string — there is no second place where a month could be built wrong.
export function buildPeriodOptions(today: Date, monthCount: number = 24): IPeriodOption[] {
  const options: IPeriodOption[] = [
    { ID: SINGLE_DAY_PERIOD, Name: "Specific day|Día específico" },
  ]
  for (let monthsBack = 0; monthsBack < monthCount; monthsBack++) {
    const month = new Date(today.getFullYear(), today.getMonth() - monthsBack, 1)
    options.push({
      ID: periodCode(month.getFullYear(), month.getMonth()),
      Name: `${monthNames[month.getMonth()]} ${month.getFullYear()}`,
    })
  }
  return options
}

// periodCode writes SUNAT's six-character period. It takes the parts rather than a Date so
// there is no timezone to get wrong: a UnixDay and a local calendar month reach it the same way.
export const periodCode = (year: number, monthIndex: number): string =>
  `${year}${String(monthIndex + 1).padStart(2, "0")}`

// periodOfUnixDay is the period a picked day belongs to, so a single-day view can still say
// which period it would be filed under.
//
// Read in UTC, like the backend does: a UnixDay is a day number, and interpreting its midnight
// in a negative offset would land on the previous day — and, on the first of a month, on the
// previous period.
export function periodOfUnixDay(unixDay: number): string {
  if (!unixDay) return ""
  const date = new Date(unixDay * 86400 * 1000)
  return periodCode(date.getUTCFullYear(), date.getUTCMonth())
}

// periodLabel turns a stored period back into what the selector shows.
export function periodLabel(period: string): string {
  if (!period || period.length !== 6) return ""
  const monthIndex = Number(period.slice(4, 6)) - 1
  if (monthIndex < 0 || monthIndex > 11) return ""
  return `${monthNames[monthIndex]} ${period.slice(0, 4)}`
}
