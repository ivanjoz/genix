// Rules for the SUNAT series a company issues under. Pure: no Svelte, no fetch.
//
// They mirror backend/invoicing/types/invoice_series.go, which is the authority —
// this copy exists so the form can refuse a bad series before a round trip, not
// so the backend can trust it.

export interface IInvoiceSeries {
  SeriesID: number
  DocType: number
  SeriesCode: string
  SiteID: number
  IsDefault: number
  ss: number
}

// SUNAT catalog 01, mirroring the DocType constants in invoice_document.go.
export const DOC_TYPE_FACTURA = 1
export const DOC_TYPE_BOLETA = 3
export const DOC_TYPE_CREDIT_NOTE = 7
export const DOC_TYPE_DEBIT_NOTE = 8

export const DOC_TYPES = [
  { ID: DOC_TYPE_FACTURA, Name: "Invoice|Factura" },
  { ID: DOC_TYPE_BOLETA, Name: "Receipt|Boleta" },
  { ID: DOC_TYPE_CREDIT_NOTE, Name: "Credit Note|Nota de Crédito" },
  { ID: DOC_TYPE_DEBIT_NOTE, Name: "Debit Note|Nota de Débito" },
]

export const docTypeName = (docType: number): string =>
  DOC_TYPES.find(type => type.ID === docType)?.Name || "—"

// A note carries the prefix of the document it corrects, so each note type needs
// one series per family. This is why a new company is seeded with six.
export const isNoteDocType = (docType: number): boolean =>
  docType === DOC_TYPE_CREDIT_NOTE || docType === DOC_TYPE_DEBIT_NOTE

// A series id is the two-digit suffix a document id carries, so 99 is the ceiling.
export const MAX_SERIES_ID = 99

// Ids are never reused: a document issued under series 3 points at it forever, so
// deleting 3 must not free that id for the next series.
export function nextSeriesID(allSeries: IInvoiceSeries[]): number {
  return allSeries.reduce((highest, series) => Math.max(highest, series.SeriesID), 0) + 1
}

// validateSeries returns the first problem in the set, or "" when it is sound.
// The rules that matter are about the set, not the row being edited: ids and
// active codes have to be unique, and a type can only have one default.
export function validateSeries(allSeries: IInvoiceSeries[]): string {
  const seenID = new Set<number>()
  const seenCode = new Set<string>()
  const defaultOfType = new Set<number>()

  for (const series of allSeries) {
    const code = (series.SeriesCode || "").trim().toUpperCase()

    if (!DOC_TYPES.some(type => type.ID === series.DocType)) {
      return "Select the document type for every series.|Indique el tipo de comprobante de cada serie."
    }
    if (series.SeriesID <= 0 || series.SeriesID > MAX_SERIES_ID) {
      return "The series number must be between 1 and 99.|El número interno de la serie debe estar entre 1 y 99."
    }
    if (seenID.has(series.SeriesID)) {
      return "Two series share the same number.|Hay dos series con el mismo número interno."
    }
    seenID.add(series.SeriesID)

    if (code.length !== 4) {
      return "A series code has 4 characters, for example F001.|El código de serie debe tener 4 caracteres, por ejemplo F001."
    }
    // SUNAT's rule, not a preference: the wrong first letter means every document
    // numbered under the series is rejected.
    if (code[0] !== "F" && code[0] !== "B") {
      return "A series code must start with F or B.|El código de serie debe empezar con F o con B."
    }
    if (series.DocType === DOC_TYPE_FACTURA && code[0] !== "F") {
      return "An invoice needs a series starting with F.|Una factura necesita una serie que empiece con F."
    }
    if (series.DocType === DOC_TYPE_BOLETA && code[0] !== "B") {
      return "A receipt needs a series starting with B.|Una boleta necesita una serie que empiece con B."
    }

    // Only active codes have to be unique: a retired series keeps its code so the
    // documents issued under it still read correctly.
    if (series.ss === 1) {
      if (seenCode.has(code)) {
        return "There is already an active series with that code.|Ya existe una serie activa con ese código."
      }
      seenCode.add(code)
    }

    // No branch is allowed: it means the company's own fiscal address, which is
    // SUNAT's main establishment and what a single-site company issues from.

    if (series.IsDefault === 1 && series.ss === 1) {
      if (defaultOfType.has(series.DocType)) {
        return "Only one series per document type can be the default.|Solo una serie puede ser la predeterminada por tipo de comprobante."
      }
      defaultOfType.add(series.DocType)
    }
  }
  return ""
}

// makeSeries builds the row an "add" produces, already carrying its id.
export function makeSeries(allSeries: IInvoiceSeries[], docType: number, code: string,
  siteID: number): IInvoiceSeries {

  return {
    SeriesID: nextSeriesID(allSeries),
    DocType: docType,
    SeriesCode: (code || "").trim().toUpperCase(),
    SiteID: siteID,
    // The first series of a document type is its default, because a till that
    // names no series still has to get one.
    IsDefault: allSeries.some(s => s.DocType === docType && s.ss === 1) ? 0 : 1,
    ss: 1,
  }
}
