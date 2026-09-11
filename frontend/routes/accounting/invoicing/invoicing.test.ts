import { describe, expect, it } from 'vitest'
import {
  canSendInvoice, countPendingInvoices, filterInvoices, hasInvoiceCDR, hasInvoiceXML,
  invoiceNumber, invoiceSaleOrderID, invoiceSeriesID, sunatVerdictMessage, INVOICE_ACCEPTED,
  INVOICE_EXCEPTION, INVOICE_OBSERVED, INVOICE_PENDING, INVOICE_REJECTED, INVOICE_VOIDED,
  type IInvoiceDocument,
} from './invoicing'

const makeDocument = (fields: Partial<IInvoiceDocument>): IInvoiceDocument => ({
  ID: 550301, Correlativo: 1, IssueDate: 20000, IssueTime: 0, Currency: 1,
  TotalAmount: 11800, TaxAmount: 1800, TaxableAmount: 10000, ExemptAmount: 0,
  UnaffectedAmount: 0, FreeAmount: 0, AffectedDocID: 0, NoteReasonCode: "", NoteReason: "",
  State: INVOICE_PENDING, SunatCode: "", SunatNotes: [], Ticket: "", DigestValue: "",
  RetryCount: 0, LastError: "", ss: 1, upd: 0, Created: 0, CreatedBy: 0,
  ...fields,
})

describe('the series and the sale live in the id', () => {
  it('reads the series off the last two digits', () => {
    expect(invoiceSeriesID(makeDocument({ ID: 550301 }))).toBe(1)
    expect(invoiceSeriesID(makeDocument({ ID: 550312 }))).toBe(12)
  })

  // A note is issued under its own series, so its own id is not the sale's.
  it('reaches the sale through the document a note corrects', () => {
    const note = makeDocument({ ID: 550303, AffectedDocID: 550301 })
    expect(invoiceSaleOrderID(note)).toBe(550301)
    expect(invoiceSaleOrderID(makeDocument({ ID: 550301 }))).toBe(550301)
  })

  it('falls back to the series id when the code is gone', () => {
    expect(invoiceNumber(makeDocument({ Correlativo: 45 }), "F001")).toBe("F001-45")
    expect(invoiceNumber(makeDocument({ ID: 550307, Correlativo: 45 }), "")).toBe("#7-45")
  })
})

describe('what can still be done to a document', () => {
  it('offers a send only while SUNAT has not answered', () => {
    expect(canSendInvoice(INVOICE_PENDING)).toBe(true)
    expect(canSendInvoice(INVOICE_EXCEPTION)).toBe(true)
    expect(canSendInvoice(INVOICE_ACCEPTED)).toBe(false)
    expect(canSendInvoice(INVOICE_REJECTED)).toBe(false)
    // A voided document must never go out, whatever the operator clicks.
    expect(canSendInvoice(INVOICE_VOIDED)).toBe(false)
  })

  it('has no artifacts before it was signed', () => {
    expect(hasInvoiceXML(INVOICE_PENDING)).toBe(false)
    expect(hasInvoiceXML(INVOICE_VOIDED)).toBe(false)
    expect(hasInvoiceXML(INVOICE_ACCEPTED)).toBe(true)
    // The CDR is SUNAT's answer: a rejection has one, an unsent document does not.
    expect(hasInvoiceCDR(INVOICE_REJECTED)).toBe(true)
    expect(hasInvoiceCDR(INVOICE_EXCEPTION)).toBe(false)
  })
})

describe('the report filter', () => {
  const documents = [
    makeDocument({ ID: 100001, Correlativo: 1, IssueDate: 20000, State: INVOICE_PENDING }),
    makeDocument({ ID: 200001, Correlativo: 2, IssueDate: 20005, State: INVOICE_ACCEPTED }),
    makeDocument({ ID: 300002, Correlativo: 3, IssueDate: 20010, State: INVOICE_PENDING }),
  ]
  const noFilter = { fromDate: 0, toDate: 0, idText: "", state: 0 }

  it('keeps everything when nothing is asked', () => {
    expect(filterInvoices(documents, noFilter)).toHaveLength(3)
  })

  it('bounds the date range inclusively', () => {
    expect(filterInvoices(documents, { ...noFilter, fromDate: 20005 })).toHaveLength(2)
    expect(filterInvoices(documents, { ...noFilter, toDate: 20005 })).toHaveLength(2)
    expect(filterInvoices(documents, { ...noFilter, fromDate: 20005, toDate: 20005 })).toHaveLength(1)
  })

  it('matches the document id, the sale id and the correlativo', () => {
    expect(filterInvoices(documents, { ...noFilter, idText: "300002" })).toHaveLength(1)
    expect(filterInvoices(documents, { ...noFilter, idText: "999" })).toHaveLength(0)
  })

  it('filters by state, and 0 means every state', () => {
    expect(filterInvoices(documents, { ...noFilter, state: INVOICE_PENDING })).toHaveLength(2)
    expect(filterInvoices(documents, { ...noFilter, state: INVOICE_ACCEPTED })).toHaveLength(1)
  })

  it('counts what is still waiting', () => {
    expect(countPendingInvoices(documents)).toBe(2)
  })
})

describe('what the operator is told after waiting for SUNAT', () => {
  it('names the verdict the document came back with', () => {
    expect(sunatVerdictMessage(makeDocument({ State: INVOICE_ACCEPTED })))
      .toContain("SUNAT aceptó el comprobante")
    expect(sunatVerdictMessage(makeDocument({ State: INVOICE_OBSERVED })))
      .toContain("observaciones")
  })

  // Losing the race against an annulment is not an error, and saying "enviado"
  // about a document that never left would be a lie.
  it('says a voided document was not sent', () => {
    expect(sunatVerdictMessage(makeDocument({ State: INVOICE_VOIDED })))
      .toContain("no se envió")
  })
})
