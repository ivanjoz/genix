// What an electronic document is and how it reads. Pure: no Svelte, no fetch.
//
// Mirrors backend/invoicing/types/invoice_document.go. A document is created with
// the sale it bills and waits in PENDING until the cron sweep sends it, so most of
// what this page shows is a state and what SUNAT answered about it.

export interface IInvoiceDocument {
  ID: number
  Correlativo: number
  IssueDate: number   // UnixDay
  IssueTime: number   // SUnixTime
  Currency: number
  TotalAmount: number // cents
  TaxAmount: number
  TaxableAmount: number
  ExemptAmount: number
  UnaffectedAmount: number
  FreeAmount: number
  AffectedDocID: number
  NoteReasonCode: string
  NoteReason: string
  State: number
  SunatCode: string
  SunatNotes: string[]
  Ticket: string
  DigestValue: string
  RetryCount: number
  LastError: string
  ss: number
  upd: number
  Created: number
  CreatedBy: number
}

// The states, in the order the backend numbers them.
export const INVOICE_VOIDED = 0
export const INVOICE_PENDING = 1
export const INVOICE_QUEUED = 2
export const INVOICE_ACCEPTED = 3
export const INVOICE_OBSERVED = 4
export const INVOICE_REJECTED = 5
export const INVOICE_EXCEPTION = 6

export const invoiceStateLabels: Record<number, string> = {
  [INVOICE_VOIDED]: "Voided|Anulado",
  [INVOICE_PENDING]: "Pending|Pendiente de envío",
  [INVOICE_QUEUED]: "Sending|Enviando",
  [INVOICE_ACCEPTED]: "Accepted|Aceptado",
  [INVOICE_OBSERVED]: "Accepted with remarks|Aceptado con observaciones",
  [INVOICE_REJECTED]: "Rejected|Rechazado",
  [INVOICE_EXCEPTION]: "Failed|Error de envío",
}

// Green is "SUNAT has it", red is "it will never be accepted as it stands", amber
// is "still on its way". The operator reads this column before anything else.
export const invoiceStateCss: Record<number, string> = {
  [INVOICE_VOIDED]: "text-gray-500",
  [INVOICE_PENDING]: "text-amber-600",
  [INVOICE_QUEUED]: "text-blue-600",
  [INVOICE_ACCEPTED]: "text-green-600",
  [INVOICE_OBSERVED]: "text-green-700",
  [INVOICE_REJECTED]: "text-red-600",
  [INVOICE_EXCEPTION]: "text-red-600",
}

// SUNAT has it, whatever it thought of it.
export const wasSentToSunat = (state: number): boolean =>
  state === INVOICE_ACCEPTED || state === INVOICE_OBSERVED || state === INVOICE_REJECTED

// Whether "enviar ahora" applies: it pushes a document out ahead of the sweep, or
// again after a transport failure. Anything SUNAT already answered is final, and a
// voided document must never go out.
export const canSendInvoice = (state: number): boolean =>
  state === INVOICE_PENDING || state === INVOICE_QUEUED || state === INVOICE_EXCEPTION

// The XML exists from the moment the document was signed, which is the step that
// moves it out of PENDING. The CDR only exists once SUNAT answered.
export const hasInvoiceXML = (state: number): boolean =>
  state !== INVOICE_PENDING && state !== INVOICE_VOIDED

export const hasInvoiceCDR = (state: number): boolean => wasSentToSunat(state)

// The series a document was issued under lives in the last two digits of its id,
// not in a column — the same packing the sale id uses.
export const SERIES_DIGITS = 100

export const invoiceSeriesID = (document: IInvoiceDocument): number =>
  document.ID % SERIES_DIGITS

// The sale a document bills. A note is issued under its own series and lands on a
// neighbouring id, so it reaches the sale through the document it corrects.
export const invoiceSaleOrderID = (document: IInvoiceDocument): number =>
  document.AffectedDocID || document.ID

// The number as SUNAT writes it. The series code belongs to the company record, so
// the caller resolves it and passes it in; a document whose series was deleted
// still has to read as something.
export const invoiceNumber = (document: IInvoiceDocument, seriesCode: string): string =>
  `${seriesCode || `#${invoiceSeriesID(document)}`}-${document.Correlativo}`

export interface InvoiceFilter {
  // UnixDay bounds, inclusive. 0 means unbounded.
  fromDate: number
  toDate: number
  // Matched against the document id, the sale id and the correlativo.
  idText: string
  // 0 shows every state.
  state: number
}

// filterInvoices is the report's whole query. Filtering happens here rather than on
// the server because the list is delta-synced: after the first load the browser
// already holds every document, so a date range is a scan over memory.
export function filterInvoices(
  documents: IInvoiceDocument[], filter: InvoiceFilter,
): IInvoiceDocument[] {

  const searchedID = (filter.idText || "").trim()

  return (documents || []).filter(document => {
    if (filter.fromDate && document.IssueDate < filter.fromDate) return false
    if (filter.toDate && document.IssueDate > filter.toDate) return false
    if (filter.state && document.State !== filter.state) return false

    if (searchedID) {
      const matchesAnyID = String(document.ID).includes(searchedID)
        || String(invoiceSaleOrderID(document)).includes(searchedID)
        || String(document.Correlativo).includes(searchedID)
      if (!matchesAnyID) return false
    }
    return true
  })
}

// How many documents are waiting, so the page can say so without the operator
// having to filter for it.
export const countPendingInvoices = (documents: IInvoiceDocument[]): number =>
  (documents || []).filter(document => document.State === INVOICE_PENDING).length

// What to tell the operator after a send that the request waited for. Only the
// states a successful call can land on are named: a rejection or a transport
// failure comes back as an error and carries SUNAT's own message.
export function sunatVerdictMessage(document: IInvoiceDocument): string {
  if (document.State === INVOICE_ACCEPTED) {
    return "SUNAT accepted the document.|SUNAT aceptó el comprobante."
  }
  if (document.State === INVOICE_OBSERVED) {
    return "SUNAT accepted it with observations.|SUNAT lo aceptó con observaciones."
  }
  if (document.State === INVOICE_VOIDED) {
    return "The document was voided and was not sent.|El comprobante fue anulado y no se envió."
  }
  return "The document was sent.|El comprobante fue enviado."
}
