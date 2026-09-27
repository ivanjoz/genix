// The supplier's comprobante a purchase is booked with — the Registro de Compras row.
//
// Purchase orders, expenses and fixed assets carry the same document columns, and the
// accounting books read them, so the shape and its rules live in core rather than in any one
// of those routes. Mirrors finance.PurchaseDocument in backend/finance/types/purchase_document.go.

export interface IPurchaseDocTypeOption {
  id: number
  name: string
  // 1 when the document can carry IGV with crédito fiscal. A boleta or a recibo por honorarios
  // grants none, so its whole amount is untaxed.
  creditsTax?: number
}

// 0 is not in the list: it is "no comprobante", which SearchSelect shows as its empty state.
//CATALOG:finance.PurchaseDocTypeOptions
export const PURCHASE_DOC_TYPES: IPurchaseDocTypeOption[] = [
  { id: 1, name: 'Invoice|Factura', creditsTax: 1 },
  { id: 2, name: 'Fee receipt|Recibo por Honorarios' },
  { id: 3, name: 'Receipt|Boleta' },
  { id: 12, name: 'Register ticket|Ticket', creditsTax: 1 },
  { id: 14, name: 'Utility bill|Recibo de Servicios Públicos', creditsTax: 1 },
]

export const PURCHASE_DOC_TYPE_NONE = 0
export const PURCHASE_DOC_TYPE_INVOICE = 1
export const PURCHASE_DOC_TYPE_UTILITY_BILL = 14

export interface IPurchaseDocument {
  DocType: number
  DocSeries: string
  DocNumber: number
  DocIssueDate: number   // UnixDay
  TaxableAmount: number  // cents
  TaxAmount: number
  UntaxedAmount: number
  OtherAmount: number
  CurrencyType: number
  ExchangeRate: number   // × 1000
}

export const purchaseDocTypeName = (docType: number): string =>
  PURCHASE_DOC_TYPES.find(option => option.id === docType)?.name || "No comprobante|Sin comprobante"

export const purchaseDocTypeCreditsTax = (docType: number): boolean =>
  PURCHASE_DOC_TYPES.find(option => option.id === docType)?.creditsTax === 1

// Field 25 of the RCE. Never stored: the annex defines it as the sum.
export const purchaseDocumentTotal = (document: Partial<IPurchaseDocument>): number =>
  (document.TaxableAmount || 0) + (document.TaxAmount || 0) +
  (document.UntaxedAmount || 0) + (document.OtherAmount || 0)

// IGV_RATE_PERCENT is the general rate. Only used to propose a split the user then corrects —
// the stored figures are always the ones printed on the supplier's document.
const IGV_RATE_PERCENT = 18

// splitPurchaseTotal completes the amounts of a document so they add up to its total. What is
// left after other charges is base + IGV at the general rate (minus the untaxed part) when the
// document grants crédito fiscal, otherwise it is all untaxed.
export function splitPurchaseTotal(
  docType: number, total: number, untaxed = 0, other = 0,
): Partial<IPurchaseDocument> {
  if (!purchaseDocTypeCreditsTax(docType)) {
    return { TaxableAmount: 0, TaxAmount: 0, UntaxedAmount: total - other, OtherAmount: other }
  }
  const taxedTotal = total - untaxed - other
  const taxable = Math.round(taxedTotal * 100 / (100 + IGV_RATE_PERCENT))
  return { TaxableAmount: taxable, TaxAmount: taxedTotal - taxable, UntaxedAmount: untaxed, OtherAmount: other }
}
