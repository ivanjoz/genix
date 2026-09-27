// The Anexo 11 replacement file: the TXT uploaded to SIRE to replace the RCE proposal.
//
// Written here for the same reason as the sales file: the rows on screen are the rows filed.
// Fields 1-37 are informed by the taxpayer; 38-41 SUNAT completes, and the free-use fields
// 42-80 are left out together with their pipes, so the record stops at 37.
// See ../docs/SUNAT-RCE-Diario-annexes-summary.md §1.2 for the file name and §1.4 for the fields.

import { sunatDocCode } from '$core/sunat-doc-type'
import { downloadTextFile } from '$libs/helpers'
import type { IPurchaseBookRow, IPurchasesBookIssuer } from './books.purchases'
import { sunatAmount, sunatDate, sunatText } from './books.utils'

// The code of the Registro de Compras in SIRE (8.4).
export const BOOK_CODE_PURCHASES = "080400"

// Tabla 13, the parts of the file name this exporter fixes.
const OPPORTUNITY_REPLACEMENT = "02" // 02 = reemplaza la propuesta
const OPERATIONS_ACTIVE = "1"        // 1 = empresa operativa
const CURRENCY_SOLES = "1"           // the book is kept in soles, whatever a row's currency is
const GENERATED_BY_SIRE = "2"        // fixed by the annex

const LEGAL_NAME_MAX = 1500
const SUPPLIER_NAME_MAX = 1500

// A dollar rate travels as #.### (field 27).
const sunatExchangeRate = (rate: number): string => rate ? (rate / 1000).toFixed(3) : ""

export function buildPurchasesBookFile(rows: IPurchaseBookRow[], issuer: IPurchasesBookIssuer): string {
  return rows.map(row => buildPurchasesBookRecord(row, issuer)).join("\r\n")
}

// buildPurchasesBookRecord is one line of the file: Anexo 11 fields 1 to 37, in order.
//
// The ERP books every purchase with crédito fiscal as destined to taxed sales (DG, 15-16):
// DGNG and DNG (17-20) only exist for a company that also makes untaxed sales.
export function buildPurchasesBookRecord(row: IPurchaseBookRow, issuer: IPurchasesBookIssuer): string {
  const fields = [
    issuer.RUC,                                    // 1  RUC del generador
    sunatText(issuer.LegalName, LEGAL_NAME_MAX),   // 2  Razón social del generador
    row.Period,                                    // 3  Periodo AAAAMM
    "",                                            // 4  CAR — SUNAT lo completa
    sunatDate(row.IssueDate),                      // 5  Fecha de emisión
    sunatDate(row.DueDate),                        // 6  Fecha de vencimiento — sólo tipo 14
    sunatDocCode(row.DocType),                     // 7  Tipo de comprobante
    row.Series,                                    // 8  Serie
    "",                                            // 9  Año de la DUA/DSI — importaciones
    String(row.Number),                            // 10 Número
    "",                                            // 11 Número final — sin consolidación
    row.SupplierDocType,                           // 12 Tipo de documento del proveedor
    row.SupplierDocNumber,                         // 13 Número de documento del proveedor
    sunatText(row.SupplierName, SUPPLIER_NAME_MAX), // 14 Razón social del proveedor
    sunatAmount(row.TaxableAmount),                // 15 Base imponible DG
    sunatAmount(row.TaxAmount),                    // 16 IGV DG
    "0.00",                                        // 17 Base imponible DGNG
    "0.00",                                        // 18 IGV DGNG
    "0.00",                                        // 19 Base imponible DNG
    "0.00",                                        // 20 IGV DNG
    sunatAmount(row.UntaxedAmount),                // 21 Adquisiciones no gravadas
    "0.00",                                        // 22 ISC
    "0.00",                                        // 23 ICBPER
    sunatAmount(row.OtherAmount),                  // 24 Otros tributos y cargos
    sunatAmount(row.TotalAmount),                  // 25 Importe total
    row.Currency,                                  // 26 Moneda, ISO 4217
    sunatExchangeRate(row.ExchangeRate),           // 27 Tipo de cambio — sólo si 26 ≠ PEN
    "",                                            // 28 Fecha del documento modificado
    "",                                            // 29 Tipo del documento modificado
    "",                                            // 30 Serie del documento modificado
    "",                                            // 31 Código de la dependencia aduanera
    "",                                            // 32 Número del documento modificado
    "",                                            // 33 Clasificación — sólo ingresos > 1500 UIT
    "",                                            // 34 ID del proyecto de operadores
    "",                                            // 35 Porcentaje de participación
    "",                                            // 36 Impuesto beneficiado — Ley 31053
    "",                                            // 37 CAR original — sólo ajustes
  ]
  return `${fields.join("|")}|`
}

// purchasesBookFileName builds the Tabla 13 name:
// LE RRRRRRRRRRR AAAAMM 00 080400 02 O I M 2 .TXT
export function purchasesBookFileName(ruc: string, period: string, hasRows: boolean): string {
  const contentFlag = hasRows ? "1" : "0"
  return "LE" + ruc + period + "00" + BOOK_CODE_PURCHASES + OPPORTUNITY_REPLACEMENT +
    OPERATIONS_ACTIVE + contentFlag + CURRENCY_SOLES + GENERATED_BY_SIRE + ".TXT"
}

export const downloadPurchasesBookTxt = (
  rows: IPurchaseBookRow[], issuer: IPurchasesBookIssuer, period: string,
): void => downloadTextFile(
  purchasesBookFileName(issuer.RUC, period, rows.length > 0),
  buildPurchasesBookFile(rows, issuer),
)
