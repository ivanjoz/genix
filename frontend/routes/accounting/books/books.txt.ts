// The Anexo 3 replacement file: the TXT that is uploaded to SIRE to replace the RVIE proposal.
//
// It is written here and not in Go because the consolidated rows only exist here — they are the
// same ISalesBookRow[] the table on screen shows, so the file and the preview can never
// disagree. Phase 5 sends this text to the backend, which zips, hashes and uploads it.
//
// Field numbers below are Anexo 3 of RS 112-2021. Fields 1-33 are informed by the taxpayer;
// 34-40 SUNAT completes from the proposal and 41-57 are free use, so the record stops at 33.
// See ./PLAN.md §1.2 for the whole table and §1.3 for the file name.

import { DOC_TYPE_CREDIT_NOTE, sunatDocCode } from '#core/sunat-doc-type.ts'
import { downloadTextFile } from '#libs/helpers.ts'
import { countPendingRows, isWholeMonthPeriod, type ISalesBookRow } from './books'
import { sunatAmount, sunatDate, sunatText } from './books.utils'

// Who the book belongs to — fields 1 and 2. The company record already in the page.
export interface ISalesBookIssuer {
  RUC: string
  LegalName: string
}

// The code of the Registro de Ventas in SIRE. Not PLE's 140100: a file named 140100 is read as
// a different book and rejected.
export const BOOK_CODE_SALES = "140400"

// Tabla 6, the parts of the file name this exporter fixes.
const OPPORTUNITY_REPLACEMENT = "02" // 02 = reemplaza la propuesta
const OPERATIONS_ACTIVE = "1"        // 1 = empresa operativa
const CURRENCY_SOLES = "1"           // the book is kept in soles, whatever a row's currency is
const GENERATED_BY_MIGE = "2"        // fixed by the annex

// The caps Anexo 3 puts on the two free-text fields.
const LEGAL_NAME_MAX = 1500
const BUYER_NAME_MAX = 1500

// A credit note is anotada as a subtraction: it takes back base imponible and IGV that an
// earlier comprobante declared, so its importes travel negative. A debit note adds and stays
// positive.
//
// ⚠️ Unverified against the annex text. Fields 16 and 18 ("Dscto BI" / "Dscto IGV", negative,
// only for tipos 07 and 87) could instead be where the reduction belongs, with 15 and 17 left
// at zero. Confirm before a period is filed — this one function is the only place to change.
export const salesBookRowSign = (row: ISalesBookRow): number =>
  row.DocType === DOC_TYPE_CREDIT_NOTE ? -1 : 1

// salesBookExportBlockers lists every reason the period cannot be written as a file, in the
// bilingual form the page shows through tr(). An empty list means it can.
//
// A period with no comprobantes is not one of them: Tabla 6 has a "sin información" flag for
// exactly that, and a month with no sales still has to be declared.
export function salesBookExportBlockers(
  rows: ISalesBookRow[], issuer: ISalesBookIssuer, period: string,
): string[] {
  const blockers: string[] = []

  if (!isWholeMonthPeriod(period)) {
    blockers.push(
      "A single day is not a filable book — pick a month.|" +
      "Un día suelto no es un libro presentable — elige un mes.",
    )
  }
  if (!/^\d{11}$/.test(issuer.RUC || "")) {
    blockers.push(
      "The company has no valid RUC (field 1).|La empresa no tiene un RUC válido (campo 1).",
    )
  }

  const pendingCount = countPendingRows(rows)
  if (pendingCount > 0) {
    blockers.push(
      `${pendingCount} comprobantes are still awaiting SUNAT's answer.|` +
      `${pendingCount} comprobantes siguen esperando la respuesta de SUNAT.`,
    )
  }

  // The exchange rate (field 28) is mandatory the moment a row is not in soles, and this ERP
  // does not store one. Writing the file without it would file a record SUNAT rejects.
  const foreignCount = rows.filter(row => row.Currency !== "PEN").length
  if (foreignCount > 0) {
    blockers.push(
      `${foreignCount} comprobantes are not in PEN and the ERP stores no exchange rate (field 28).|` +
      `${foreignCount} comprobantes no están en PEN y el ERP no guarda tipo de cambio (campo 28).`,
    )
  }

  // A row whose series could not be joined has no tipo de comprobante and no serie — fields 7
  // and 8 would go out empty, which is a rejected record rather than a visible gap.
  const unresolvedCount = rows.filter(row => !row.DocType || !row.Series).length
  if (unresolvedCount > 0) {
    blockers.push(
      `${unresolvedCount} rows have no series — check the company's series configuration.|` +
      `${unresolvedCount} filas no tienen serie — revisa la configuración de series de la empresa.`,
    )
  }

  return blockers
}

// buildSalesBookFile writes the whole file: one record per book row, fields separated and
// terminated by `|`, records separated by CRLF.
export function buildSalesBookFile(rows: ISalesBookRow[], issuer: ISalesBookIssuer): string {
  return rows.map(row => buildSalesBookRecord(row, issuer)).join("\r\n")
}

// buildSalesBookRecord is one line of the file: Anexo 3 fields 1 to 33, in order.
//
// The fields this ERP can never fill go out as the literal "0.00" the annex requires, which is
// why they are written here and not carried on the row.
export function buildSalesBookRecord(row: ISalesBookRow, issuer: ISalesBookIssuer): string {
  const sign = salesBookRowSign(row)
  const amount = (cents: number) => sunatAmount(sign * (cents || 0))

  const fields = [
    issuer.RUC,                                  // 1  RUC del generador
    sunatText(issuer.LegalName, LEGAL_NAME_MAX), // 2  Razón social del generador
    row.Period,                                  // 3  Periodo YYYYMM
    "",                                          // 4  CAR — SUNAT lo completa
    sunatDate(row.IssueDate),                    // 5  Fecha de emisión
    "",                                          // 6  Fecha de vencimiento — sólo tipo 14
    sunatDocCode(row.DocType),                   // 7  Tipo de comprobante
    row.Series,                                  // 8  Serie
    String(row.Number),                          // 9  Número, o inicial del rango
    row.NumberFinal ? String(row.NumberFinal) : "", // 10 Número final — sólo consolidados
    row.BuyerDocType,                            // 11 Tipo de documento de identidad
    row.BuyerDocNumber,                          // 12 Número de documento
    sunatText(row.BuyerName, BUYER_NAME_MAX),    // 13 Apellidos y nombres o razón social
    "0.00",                                      // 14 Valor facturado de la exportación
    amount(row.TaxableAmount),                   // 15 Base imponible gravada
    "0.00",                                      // 16 Descuento de la base imponible
    amount(row.TaxAmount),                       // 17 IGV / IPM
    "0.00",                                      // 18 Descuento del IGV / IPM
    amount(row.ExemptAmount),                    // 19 Monto exonerado
    amount(row.UnaffectedAmount),                // 20 Monto inafecto
    "0.00",                                      // 21 ISC
    "0.00",                                      // 22 Base imponible gravada del IVAP
    "0.00",                                      // 23 IVAP
    "0.00",                                      // 24 ICBPER
    "0.00",                                      // 25 Otros tributos
    amount(row.TotalAmount),                     // 26 Importe total del comprobante
    row.Currency,                                // 27 Moneda, ISO 4217
    "",                                          // 28 Tipo de cambio — sólo si 27 ≠ PEN
    sunatDate(row.ModifiedIssueDate),            // 29 Fecha del documento modificado
    row.ModifiedDocType ? sunatDocCode(row.ModifiedDocType) : "", // 30 Tipo modificado
    row.ModifiedSeries,                          // 31 Serie del documento modificado
    row.ModifiedNumber ? String(row.ModifiedNumber) : "", // 32 Número del modificado
    "",                                          // 33 ID del proyecto de operadores
  ]

  // The record ends with a separator too: the trailing pipe is what closes field 33.
  return `${fields.join("|")}|`
}

// salesBookFileName builds the Tabla 6 name. Every position is meaning, not decoration — see
// PLAN.md §1.3 for the mask.
//
// Positions 20-21 are the day, and the RVIE is a monthly book, so they are "00".
export function salesBookFileName(
  ruc: string, period: string, hasRows: boolean,
): string {
  const contentFlag = hasRows ? "1" : "0"
  return "LE" + ruc + period + "00" + BOOK_CODE_SALES + OPPORTUNITY_REPLACEMENT +
    OPERATIONS_ACTIVE + contentFlag + CURRENCY_SOLES + GENERATED_BY_MIGE + ".TXT"
}

export const downloadSalesBookTxt = (
  rows: ISalesBookRow[], issuer: ISalesBookIssuer, period: string,
): void => downloadTextFile(
  salesBookFileName(issuer.RUC, period, rows.length > 0),
  buildSalesBookFile(rows, issuer),
)
