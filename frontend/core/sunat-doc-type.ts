// SUNAT catalog 01 — what kind of comprobante a document is.
//
// Mirrors the DocType constants in backend/invoicing/types/invoice_document.go. It lives in
// core because three features read it and none of them owns it: the series form in company
// configuration, the till in sale_order_create, and the accounting books.
//
// The stored value is the id, everywhere — on InvoiceSeries.DocType and on SalesBookRow.DocType.
// The two-character code SUNAT prints is produced only by whatever writes to SUNAT, which is
// what sunatDocCode is for.

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

// sunatDocCode is the id padded to the two characters SUNAT's catalog uses — campo 7 of the
// Anexo 3 sales book, and the same value the XML carries. Twin of SunatDocCode in
// backend/accounting/types/sales_book.go; unpadded is rejected.
export const sunatDocCode = (docType: number): string =>
  docType ? String(docType).padStart(2, "0") : ""
