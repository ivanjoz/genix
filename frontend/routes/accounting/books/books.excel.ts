import { downloadExcel, type ExcelTableColumn } from '@genix/ui/excel'
import { formatTime } from '#libs/helpers.ts'
import { tr } from '#core/store.svelte.ts'
import { docTypeName, sunatDocCode } from '#core/sunat-doc-type.ts'
import {
  bookRowModifiedDocument, bookRowNumber, bookStateLabels, periodLabel,
  type ISalesBookRow,
} from './books'
import type { IPurchaseBookRow } from './books.purchases'
import { purchaseDocTypeName } from '#core/purchase-document.ts'

// The sheet is the book, column for column, in SUNAT's own field order.
//
// Headers carry the Anexo 3 field number because that is how an accountant checks a book: down
// the annex, column by column. Amounts leave as numbers, not strings, so the file can be summed.
const centsToSheet = (cents: number) => Number(((cents || 0) / 100).toFixed(2))

export const salesBookExcelColumns = (): ExcelTableColumn<ISalesBookRow>[] => [
  {
    header: "3 · Periodo",
    getValue: (row) => row.Period,
  },
  {
    header: "5 · Fecha emisión",
    getValue: (row) => formatTime(row.IssueDate, "d-m-Y") as string,
  },
  {
    header: "7 · Tipo",
    // The annex wants the two-character code, not the id the row carries.
    getValue: (row) => sunatDocCode(row.DocType),
  },
  {
    header: "8 · Serie",
    getValue: (row) => row.Series,
  },
  {
    header: "9-10 · Número",
    getValue: (row) => bookRowNumber(row),
  },
  {
    header: "11 · Tipo doc.",
    getValue: (row) => row.BuyerDocType,
  },
  {
    header: "12 · Nro doc.",
    getValue: (row) => row.BuyerDocNumber,
  },
  {
    header: "13 · Razón social",
    getValue: (row) => row.BuyerName,
  },
  {
    header: "15 · Base imponible",
    getValue: (row) => centsToSheet(row.TaxableAmount),
  },
  {
    header: "17 · IGV",
    getValue: (row) => centsToSheet(row.TaxAmount),
  },
  {
    header: "19 · Exonerado",
    getValue: (row) => centsToSheet(row.ExemptAmount),
  },
  {
    header: "20 · Inafecto",
    getValue: (row) => centsToSheet(row.UnaffectedAmount),
  },
  {
    header: "26 · Total",
    getValue: (row) => centsToSheet(row.TotalAmount),
  },
  {
    header: "27 · Moneda",
    getValue: (row) => row.Currency,
  },
  {
    header: "29-32 · Doc. modificado",
    getValue: (row) => bookRowModifiedDocument(row),
  },
  // Not SUNAT fields. They exist so whoever reads the sheet can see why a row is at zero and
  // how many boletas a consolidated line stands for.
  {
    header: "Estado",
    getValue: (row) => tr(bookStateLabels[row.ValidityState] || ""),
  },
  {
    header: "Boletas consolidadas",
    getValue: (row) => row.ConsolidatedCount || "",
  },
  {
    header: "Comprobante",
    getValue: (row) => tr(docTypeName(row.DocType)),
  },
]

export const exportSalesBookToExcel = async (
  rows: ISalesBookRow[], period: string,
): Promise<void> => {
  await downloadExcel({
    fileName: `registro-ventas-${period}.xlsx`,
    creator: 'Genix',
    includeTitleRow: true,
    includeGroupedHeaders: true,
    headerRowIndex: 2,
    sheet: {
      sheetName: 'Registro de Ventas',
      title: `Registro de Ventas — ${periodLabel(period) || period}`,
      columns: salesBookExcelColumns(),
      records: rows,
    },
  })
}

// The Registro de Compras sheet, in Anexo 11 field order. Amounts stay in the document's
// currency, as in the file; field 26 says which one.
export const purchasesBookExcelColumns = (): ExcelTableColumn<IPurchaseBookRow>[] => [
  { header: "3 · Periodo", getValue: (row) => row.Period },
  { header: "5 · Fecha emisión", getValue: (row) => formatTime(row.IssueDate, "d-m-Y") as string },
  { header: "6 · Vencimiento", getValue: (row) => row.DueDate ? formatTime(row.DueDate, "d-m-Y") as string : "" },
  { header: "7 · Tipo", getValue: (row) => sunatDocCode(row.DocType) },
  { header: "8 · Serie", getValue: (row) => row.Series },
  { header: "10 · Número", getValue: (row) => row.Number },
  { header: "12 · Tipo doc.", getValue: (row) => row.SupplierDocType },
  { header: "13 · Nro doc.", getValue: (row) => row.SupplierDocNumber },
  { header: "14 · Razón social", getValue: (row) => row.SupplierName },
  { header: "15 · Base imponible", getValue: (row) => centsToSheet(row.TaxableAmount) },
  { header: "16 · IGV", getValue: (row) => centsToSheet(row.TaxAmount) },
  { header: "21 · No gravado", getValue: (row) => centsToSheet(row.UntaxedAmount) },
  { header: "24 · Otros cargos", getValue: (row) => centsToSheet(row.OtherAmount) },
  { header: "25 · Total", getValue: (row) => centsToSheet(row.TotalAmount) },
  { header: "26 · Moneda", getValue: (row) => row.Currency },
  { header: "27 · Tipo de cambio", getValue: (row) => row.ExchangeRate ? row.ExchangeRate / 1000 : "" },
  { header: "Comprobante", getValue: (row) => tr(purchaseDocTypeName(row.DocType)) },
]

export const exportPurchasesBookToExcel = async (
  rows: IPurchaseBookRow[], period: string,
): Promise<void> => {
  await downloadExcel({
    fileName: `registro-compras-${period}.xlsx`,
    creator: 'Genix',
    includeTitleRow: true,
    includeGroupedHeaders: true,
    headerRowIndex: 2,
    sheet: {
      sheetName: 'Registro de Compras',
      title: `Registro de Compras — ${periodLabel(period) || period}`,
      columns: purchasesBookExcelColumns(),
      records: rows,
    },
  })
}
