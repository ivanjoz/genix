import { describe, expect, it } from 'vitest'
import { DOC_TYPE_BOLETA, DOC_TYPE_CREDIT_NOTE, DOC_TYPE_FACTURA } from '$core/sunat-doc-type'
import type { IClientProviderSnapshot } from '$services/crm/client-provider.svelte'
import {
  bookRowDocument, bookRowModifiedDocument, bookRowNumber, buildPeriodOptions, buildSalesBook,
  countPendingRows, isWholeMonthPeriod, periodCode, periodLabel, periodOfUnixDay, sumSalesBook,
  ANONYMOUS_BUYER_NAME, ANONYMOUS_DOC_NUMBER, BOLETA_IDENTIFIED_FROM, BOOK_STATE_ACTIVE,
  BOOK_STATE_PENDING, BOOK_STATE_REJECTED, BOOK_STATE_VOIDED, SINGLE_DAY_PERIOD,
  type IBookSeries, type ISalesBookDocument, type ISalesBookResponse, type ISalesBookRow,
} from './books'

// The two joins the backend leaves to the page. Series 1 is the factura series, 2 the boleta
// one, 3 a credit-note series — the ids a document carries in its last two digits.
const FACTURA_SERIES_ID = 1
const BOLETA_SERIES_ID = 2
const CREDIT_NOTE_SERIES_ID = 3

const testSeries = (): Map<number, IBookSeries> => new Map([
  [FACTURA_SERIES_ID, { SeriesID: FACTURA_SERIES_ID, DocType: DOC_TYPE_FACTURA, SeriesCode: "F001" }],
  [BOLETA_SERIES_ID, { SeriesID: BOLETA_SERIES_ID, DocType: DOC_TYPE_BOLETA, SeriesCode: "B001" }],
  [CREDIT_NOTE_SERIES_ID, { SeriesID: CREDIT_NOTE_SERIES_ID, DocType: DOC_TYPE_CREDIT_NOTE, SeriesCode: "BC01" }],
])

const INVOICE_ACCEPTED = 3
const ISSUE_DAY = 20700

// A comprobante id is the sale's id with the series in its last two digits, so a document of
// series 2 numbered 10 lands on 1000 + 2. The correlativo is its own column.
const makeDocument = (
  seriesID: number, correlativo: number, totalAmount: number,
  fields: Partial<ISalesBookDocument> = {},
): ISalesBookDocument => ({
  ID: correlativo * 100 + seriesID,
  IssueDate: ISSUE_DAY,
  Correlativo: correlativo,
  ClientSnapshotID: 0,
  Currency: 1,
  TaxableAmount: Math.round(totalAmount / 1.18),
  TaxAmount: totalAmount - Math.round(totalAmount / 1.18),
  ExemptAmount: 0,
  UnaffectedAmount: 0,
  TotalAmount: totalAmount,
  AffectedDocID: 0,
  State: INVOICE_ACCEPTED,
  ...fields,
})

const boleta = (correlativo: number, totalAmount: number, fields: Partial<ISalesBookDocument> = {}) =>
  makeDocument(BOLETA_SERIES_ID, correlativo, totalAmount, fields)

const buildBook = (
  documents: ISalesBookDocument[],
  identities: Map<number, IClientProviderSnapshot> = new Map(),
  affectedDocuments: ISalesBookDocument[] = [],
): ISalesBookRow[] => buildSalesBook(
  {
    Period: "202609", FirstDay: 20697, LastDay: 20726,
    Documents: documents, AffectedDocuments: affectedDocuments, UninvoicedSales: [],
  } as ISalesBookResponse,
  testSeries(),
  identities,
)

const makeRow = (fields: Partial<ISalesBookRow>): ISalesBookRow => ({
  Period: "202609", IssueDate: 20700, DocType: DOC_TYPE_BOLETA, Series: "B001",
  Number: 10, NumberFinal: 0,
  BuyerDocType: "0", BuyerDocNumber: "00000000", BuyerName: "CLIENTES VARIOS",
  TaxableAmount: 10000, TaxAmount: 1800, ExemptAmount: 0, UnaffectedAmount: 0,
  TotalAmount: 11800, Currency: "PEN", ModifiedIssueDate: 0, ModifiedDocType: 0,
  ModifiedSeries: "", ModifiedNumber: 0, ValidityState: BOOK_STATE_ACTIVE,
  ConsolidatedCount: 0, DocumentID: 1002,
  ...fields,
})

describe('a row reads as the comprobante or the range it declares', () => {
  it('writes a plain correlativo when the row is detailed', () => {
    expect(bookRowNumber(makeRow({ Number: 10 }))).toBe("10")
    expect(bookRowDocument(makeRow({ Number: 10 }))).toBe("B001-10")
  })

  // Fields 9 and 10: a consolidated row covers everything between the two numbers.
  it('writes the range when the row consolidates boletas', () => {
    const consolidated = makeRow({ Number: 10, NumberFinal: 25 })
    expect(bookRowNumber(consolidated)).toBe("10 — 25")
    expect(bookRowDocument(consolidated)).toBe("B001-10 — 25")
  })

  it('only names a modified document when there is one', () => {
    expect(bookRowModifiedDocument(makeRow({}))).toBe("")
    expect(bookRowModifiedDocument(
      makeRow({ ModifiedSeries: "B001", ModifiedNumber: 7 }),
    )).toBe("B001-7")
  })
})

describe('the totals are what has to tie to the 621', () => {
  it('sums every amount column', () => {
    const totals = sumSalesBook([
      makeRow({ TaxableAmount: 10000, TaxAmount: 1800, TotalAmount: 11800 }),
      makeRow({ TaxableAmount: 20000, TaxAmount: 3600, TotalAmount: 23600 }),
    ])
    expect(totals.taxable).toBe(30000)
    expect(totals.tax).toBe(5400)
    expect(totals.total).toBe(35400)
  })

  // A consolidated row is one line of the book but several comprobantes.
  it('counts a consolidated row as the boletas it stands for', () => {
    const totals = sumSalesBook([
      makeRow({ ConsolidatedCount: 14 }),
      makeRow({ Number: 30 }),
    ])
    expect(totals.comprobantes).toBe(15)
  })

  it('survives an empty book', () => {
    expect(sumSalesBook([]).total).toBe(0)
  })
})

it('counts the rows that block the period from being filed', () => {
  const rows = [
    makeRow({}),
    makeRow({ ValidityState: BOOK_STATE_PENDING }),
    makeRow({ ValidityState: BOOK_STATE_REJECTED }),
    makeRow({ ValidityState: BOOK_STATE_PENDING }),
  ]
  // A rejected comprobante is settled — it goes in at zero. Only a pending one is unresolved.
  expect(countPendingRows(rows)).toBe(2)
})

describe('the period is SUNAT YYYYMM everywhere', () => {
  it('pads the month', () => {
    expect(periodCode(2026, 0)).toBe("202601")
    expect(periodCode(2026, 8)).toBe("202609")
    expect(periodCode(2026, 11)).toBe("202612")
  })

  it('offers the single-day option first, then months newest first', () => {
    const options = buildPeriodOptions(new Date(2026, 8, 15), 3)
    expect(options[0].ID).toBe(SINGLE_DAY_PERIOD)
    expect(options.slice(1).map(option => option.ID)).toEqual(["202609", "202608", "202607"])
    expect(options[1].Name).toBe("SET 2026")
  })

  it('walks back across a year boundary', () => {
    const options = buildPeriodOptions(new Date(2026, 0, 10), 2)
    expect(options.slice(1).map(option => option.ID)).toEqual(["202601", "202512"])
  })

  // A UnixDay is a day number: reading its midnight in a negative offset would land on the
  // previous day, and on the first of a month that is the previous period.
  it('reads a day into its own period, including the first of the month', () => {
    const firstOfSeptember2026 = Math.floor(Date.UTC(2026, 8, 1) / 86400000)
    expect(periodOfUnixDay(firstOfSeptember2026)).toBe("202609")

    const lastOfAugust2026 = firstOfSeptember2026 - 1
    expect(periodOfUnixDay(lastOfAugust2026)).toBe("202608")
  })

  it('has no period for no day', () => {
    expect(periodOfUnixDay(0)).toBe("")
  })

  it('labels a stored period and refuses a malformed one', () => {
    expect(periodLabel("202609")).toBe("SET 2026")
    expect(periodLabel("20260900")).toBe("")
    expect(periodLabel("202613")).toBe("")
    expect(periodLabel("")).toBe("")
    expect(periodLabel(SINGLE_DAY_PERIOD)).toBe("")
  })

  // SearchSelect reads a falsy id as "nothing selected", so the single-day option carries a
  // word rather than an empty string — otherwise picking it would leave the field blank.
  it('keeps the single-day sentinel selectable', () => {
    expect(SINGLE_DAY_PERIOD).toBeTruthy()
    expect(isWholeMonthPeriod(SINGLE_DAY_PERIOD)).toBe(false)
    expect(isWholeMonthPeriod("")).toBe(false)
    expect(isWholeMonthPeriod("202609")).toBe(true)
  })
})

// ── The SUNAT mapping, ported from the Go mapper it replaced ──────────────────────────────
//
// These are the rules a filed book stands on, so they are pinned here rather than left to the
// page. Field numbers are Anexo 3 of RS 112-2021.

describe('the daily consolidation of boletas declares a contiguous range', () => {
  it('folds a day of small boletas into one row', () => {
    const rows = buildBook([boleta(10, 5000), boleta(11, 6000), boleta(12, 7000)])

    expect(rows).toHaveLength(1)
    expect(rows[0].Number).toBe(10)
    expect(rows[0].NumberFinal).toBe(12)
    expect(rows[0].ConsolidatedCount).toBe(3)
    expect(rows[0].TotalAmount).toBe(18000)
  })

  // The range carries no buyer and stands for no single document.
  it('drops the buyer and the document id from a consolidated row', () => {
    const rows = buildBook([boleta(10, 5000), boleta(11, 6000)])

    expect(rows[0].BuyerName).toBe("")
    expect(rows[0].BuyerDocNumber).toBe("")
    expect(rows[0].DocumentID).toBe(0)
  })

  // The S/ 700 boleta must be detailed with its buyer, so it cannot sit inside a range — and
  // the numbers on either side of it are no longer contiguous.
  it('splits the run at a boleta of S/ 700 or more', () => {
    const rows = buildBook([
      boleta(10, 5000), boleta(11, BOLETA_IDENTIFIED_FROM), boleta(12, 6000),
    ])

    expect(rows).toHaveLength(3)
    expect(rows.every(row => row.NumberFinal === 0)).toBe(true)
  })

  it('splits the run at a gap in the correlativos', () => {
    const rows = buildBook([
      boleta(10, 1000), boleta(11, 1000), boleta(20, 1000), boleta(21, 1000),
    ])

    expect(rows).toHaveLength(2)
    expect([rows[0].Number, rows[0].NumberFinal]).toEqual([10, 11])
    expect([rows[1].Number, rows[1].NumberFinal]).toEqual([20, 21])
  })

  // Nota 3: a rejected comprobante goes in at zero, which it cannot do inside somebody's range.
  it('splits the run at a comprobante SUNAT refused', () => {
    const INVOICE_REJECTED = 5
    const rows = buildBook([
      boleta(10, 1000), boleta(11, 1000, { State: INVOICE_REJECTED }), boleta(12, 1000),
    ])

    expect(rows).toHaveLength(3)
    const rejected = rows.find(row => row.Number === 11)!
    expect(rejected.ValidityState).toBe(BOOK_STATE_REJECTED)
    expect(rejected.TotalAmount).toBe(0)
  })

  it('leaves a run of one as the detailed row it already was', () => {
    const rows = buildBook([boleta(10, 5000)])

    expect(rows[0].NumberFinal).toBe(0)
    expect(rows[0].ConsolidatedCount).toBe(0)
    expect(rows[0].DocumentID).not.toBe(0)
  })

  it('never consolidates facturas, however small', () => {
    const rows = buildBook([
      makeDocument(FACTURA_SERIES_ID, 10, 1000),
      makeDocument(FACTURA_SERIES_ID, 11, 1000),
    ])

    expect(rows).toHaveLength(2)
  })

  it('never merges across two days', () => {
    const rows = buildBook([
      boleta(10, 1000), boleta(11, 1000, { IssueDate: ISSUE_DAY + 1 }),
    ])

    expect(rows).toHaveLength(2)
  })

  // An identity is not one of SUNAT's criteria, so a sub-S/700 boleta that has one still folds.
  it('consolidates a boleta whose buyer happened to be identified', () => {
    const identities = new Map<number, IClientProviderSnapshot>([
      [42, { ID: 42, Name: "JUAN PEREZ", RegistryNumber: "12345678", IdentityDocType: 1, CityID: "", upv: 1, ss: 1, upd: 0 }],
    ])
    const rows = buildBook(
      [boleta(10, 5000, { ClientSnapshotID: 42 }), boleta(11, 6000)], identities,
    )

    expect(rows).toHaveLength(1)
    expect(rows[0].ConsolidatedCount).toBe(2)
  })
})

describe('nota 3 — a comprobante SUNAT did not accept is anotado at zero', () => {
  it('zeroes a comprobante whose baja was communicated', () => {
    const INVOICE_VOIDED = 0
    const rows = buildBook([makeDocument(FACTURA_SERIES_ID, 10, 11800, { State: INVOICE_VOIDED })])

    expect(rows[0].ValidityState).toBe(BOOK_STATE_VOIDED)
    expect(rows[0].TotalAmount).toBe(0)
    expect(rows[0].TaxAmount).toBe(0)
  })

  // A pending one is real, it just has no answer yet: it keeps its amounts and is flagged so
  // the period is not closed over it.
  it('keeps the amounts of a comprobante SUNAT has not answered', () => {
    const INVOICE_PENDING = 1
    const rows = buildBook([boleta(10, 5000, { State: INVOICE_PENDING })])

    expect(rows[0].ValidityState).toBe(BOOK_STATE_PENDING)
    expect(rows[0].TotalAmount).toBe(5000)
    expect(rows[0].NumberFinal).toBe(0)
  })
})

describe('the buyer is whatever the comprobante declared', () => {
  it('reads the pinned identity, converted to SUNAT characters', () => {
    const identities = new Map<number, IClientProviderSnapshot>([
      [42, { ID: 42, Name: "ACME SAC", RegistryNumber: "20123456789", IdentityDocType: 6, CityID: "", upv: 1, ss: 1, upd: 0 }],
    ])
    const rows = buildBook(
      [boleta(10, BOLETA_IDENTIFIED_FROM + 100, { ClientSnapshotID: 42 })], identities,
    )

    expect(rows[0].BuyerName).toBe("ACME SAC")
    expect(rows[0].BuyerDocNumber).toBe("20123456789")
    expect(rows[0].BuyerDocType).toBe("6")
  })

  it('declares SUNAT’s unidentified buyer when nothing was pinned', () => {
    const rows = buildBook([boleta(10, 1000)])

    expect(rows[0].BuyerDocNumber).toBe(ANONYMOUS_DOC_NUMBER)
    expect(rows[0].BuyerName).toBe(ANONYMOUS_BUYER_NAME)
  })
})

// Fields 29-32. The document a note corrects is very often in an earlier month, which is why
// the backend sends it apart.
it('names the document a credit note modifies, from an earlier period', () => {
  const affected = boleta(10, 5000, { IssueDate: ISSUE_DAY - 40 })
  const note = makeDocument(CREDIT_NOTE_SERIES_ID, 1, 5000, { AffectedDocID: affected.ID })

  const rows = buildBook([note], new Map(), [affected])

  expect(rows[0].DocType).toBe(DOC_TYPE_CREDIT_NOTE)
  expect(rows[0].ModifiedDocType).toBe(DOC_TYPE_BOLETA)
  expect(rows[0].ModifiedSeries).toBe("B001")
  expect(rows[0].ModifiedNumber).toBe(10)
  expect(rows[0].ModifiedIssueDate).toBe(affected.IssueDate)
})

// A series the company no longer has must not take the page down with it.
it('renders a comprobante whose series cannot be resolved', () => {
  const rows = buildBook([makeDocument(47, 10, 1000)])

  expect(rows).toHaveLength(1)
  expect(rows[0].Series).toBe("")
  expect(rows[0].DocType).toBe(0)
})
