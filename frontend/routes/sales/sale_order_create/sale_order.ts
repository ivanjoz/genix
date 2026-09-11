// Who a sale has to identify before it can be stamped with an invoicing series.
// Pure: no Svelte, no fetch.
//
// Mirrors backend/invoicing/types/customer_identity.go, which is the authority —
// this copy exists so the till refuses the sale before the round trip, not so the
// backend can trust it. The stamp is permanent (the series lives in the last two
// digits of the sale id), so a sale that gets this wrong can never be invoiced.

import { DOC_TYPE_BOLETA, DOC_TYPE_FACTURA } from '$routes/company/configuration/invoice-series'

// Cents. SUNAT requires the buyer to be named on a boleta from 700 soles up.
export const BOLETA_IDENTIFIED_FROM = 70000

// A DNI is 8 digits and a RUC is 11, but a foreign document has no fixed length.
const MIN_IDENTITY_DOCUMENT_LENGTH = 8

export function requiresCustomerIdentity(docType: number, totalAmount: number): boolean {
  return docType === DOC_TYPE_FACTURA
    || (docType === DOC_TYPE_BOLETA && totalAmount >= BOLETA_IDENTIFIED_FROM)
}

// validateCustomerIdentity returns the problem to show, or "" when the buyer
// satisfies the document. A note is never issued from this form, so only the two
// selling types are checked.
export function validateCustomerIdentity(docType: number, totalAmount: number,
  name: string, registryNumber: string): string {

  if (!requiresCustomerIdentity(docType, totalAmount)) return ""

  const buyerName = (name || "").trim()
  const document = (registryNumber || "").trim()

  if (docType === DOC_TYPE_FACTURA) {
    // SUNAT accepts nothing but a RUC on a factura, so the shape is checked and
    // not just the presence.
    if (document.length !== 11 || !/^\d+$/.test(document)) {
      return "An invoice needs a client with an 11-digit RUC.|Una factura necesita un cliente con RUC de 11 dígitos."
    }
    if (!buyerName) {
      return "An invoice needs the client's legal name.|Una factura necesita la razón social del cliente."
    }
    return ""
  }

  if (document.length < MIN_IDENTITY_DOCUMENT_LENGTH) {
    return "A receipt from S/ 700 needs the client's DNI or RUC.|Una boleta desde S/ 700 necesita el DNI o RUC del cliente."
  }
  if (!buyerName) {
    return "A receipt from S/ 700 needs the client's name.|Una boleta desde S/ 700 necesita el nombre del cliente."
  }
  return ""
}
