import { describe, expect, it } from 'vitest'
import { DOC_TYPE_BOLETA, DOC_TYPE_CREDIT_NOTE, DOC_TYPE_FACTURA } from '$core/sunat-doc-type'
import { BOOK_STATE_ACTIVE, BOOK_STATE_PENDING, SINGLE_DAY_PERIOD, type ISalesBookRow } from './books'
import {
  buildSalesBookFile, buildSalesBookRecord, salesBookExportBlockers, salesBookFileName,
  salesBookRowSign, type ISalesBookIssuer,
} from './books.txt'
import { sunatAmount, sunatDate, sunatText } from './books.utils'

const ISSUE_DAY = 20700 // 2026-09-04

const issuer: ISalesBookIssuer = { RUC: "20601000001", LegalName: "COMERCIAL GENIX S.A.C." }

const makeRow = (fields: Partial<ISalesBookRow> = {}): ISalesBookRow => ({
  Period: "202609", IssueDate: ISSUE_DAY, DocType: DOC_TYPE_FACTURA, Series: "F001",
  Number: 10, NumberFinal: 0,
  BuyerDocType: "6", BuyerDocNumber: "20601000002", BuyerName: "CLIENTE UNO S.A.",
  TaxableAmount: 10000, TaxAmount: 1800, ExemptAmount: 0, UnaffectedAmount: 0,
  TotalAmount: 11800, Currency: "PEN", ModifiedIssueDate: 0, ModifiedDocType: 0,
  ModifiedSeries: "", ModifiedNumber: 0, ValidityState: BOOK_STATE_ACTIVE,
  ConsolidatedCount: 0, DocumentID: 1001,
  ...fields,
})

// The fields of a record, 1-indexed the way the annex numbers them.
const fieldsOf = (record: string): string[] => {
  const withoutTerminator = record.slice(0, -1)
  return ["", ...withoutTerminator.split("|")]
}

describe('the flat-file formatters follow Tabla 5', () => {
  it('writes a UnixDay as DD/MM/AAAA in UTC', () => {
    expect(sunatDate(ISSUE_DAY)).toBe("04/09/2026")
    expect(sunatDate(0)).toBe("")
  })

  it('writes amounts with exactly two decimals and no separators', () => {
    expect(sunatAmount(11800)).toBe("118.00")
    expect(sunatAmount(123456789)).toBe("1234567.89")
    expect(sunatAmount(-11800)).toBe("-118.00")
    expect(sunatAmount(0)).toBe("0.00")
  })

  it('drops the three characters a field may not contain', () => {
    expect(sunatText("A|B/C\\D", 50)).toBe("ABCD")
  })

  it('collapses line breaks, which would split the record in two', () => {
    expect(sunatText("LINEA UNO\nLINEA DOS", 50)).toBe("LINEA UNO LINEA DOS")
  })

  it('truncates at the annex length', () => {
    expect(sunatText("ABCDEFGHIJ", 4)).toBe("ABCD")
  })
})

describe('a record carries Anexo 3 fields 1 to 33', () => {
  it('ends every record with the separator that closes field 33', () => {
    const record = buildSalesBookRecord(makeRow(), issuer)
    expect(record.endsWith("|")).toBe(true)
    expect(fieldsOf(record)).toHaveLength(34) // the empty slot 0 plus 33 fields
  })

  it('puts the issuer, the period and the comprobante where the annex says', () => {
    const fields = fieldsOf(buildSalesBookRecord(makeRow(), issuer))
    expect(fields[1]).toBe("20601000001")
    expect(fields[2]).toBe("COMERCIAL GENIX S.A.C.")
    expect(fields[3]).toBe("202609")
    expect(fields[5]).toBe("04/09/2026")
    expect(fields[7]).toBe("01")
    expect(fields[8]).toBe("F001")
    expect(fields[9]).toBe("10")
  })

  it('leaves the CAR empty — SUNAT assigns it from the proposal', () => {
    expect(fieldsOf(buildSalesBookRecord(makeRow(), issuer))[4]).toBe("")
  })

  it('leaves the vencimiento empty, which is only informed for tipo 14', () => {
    expect(fieldsOf(buildSalesBookRecord(makeRow(), issuer))[6]).toBe("")
  })

  it('writes the buyer in fields 11 to 13', () => {
    const fields = fieldsOf(buildSalesBookRecord(makeRow(), issuer))
    expect(fields[11]).toBe("6")
    expect(fields[12]).toBe("20601000002")
    expect(fields[13]).toBe("CLIENTE UNO S.A.")
  })

  it('writes the amounts as 12,2 decimals in their own fields', () => {
    const fields = fieldsOf(buildSalesBookRecord(makeRow({
      TaxableAmount: 10000, TaxAmount: 1800, ExemptAmount: 500, UnaffectedAmount: 700,
      TotalAmount: 13000,
    }), issuer))
    expect(fields[15]).toBe("100.00")
    expect(fields[17]).toBe("18.00")
    expect(fields[19]).toBe("5.00")
    expect(fields[20]).toBe("7.00")
    expect(fields[26]).toBe("130.00")
    expect(fields[27]).toBe("PEN")
  })

  // The annex marks several of these "obligatorio" even when the taxpayer has nothing to put in
  // them, which is why the row does not carry them and the writer emits the literal.
  it('emits 0.00 for every concept this ERP cannot produce', () => {
    const fields = fieldsOf(buildSalesBookRecord(makeRow(), issuer))
    for (const fieldNumber of [14, 16, 18, 21, 22, 23, 24, 25]) {
      expect(fields[fieldNumber]).toBe("0.00")
    }
  })

  it('leaves the exchange rate empty for a comprobante in soles', () => {
    expect(fieldsOf(buildSalesBookRecord(makeRow(), issuer))[28]).toBe("")
  })

  it('declares a consolidated range in field 10 and names no buyer', () => {
    const fields = fieldsOf(buildSalesBookRecord(makeRow({
      DocType: DOC_TYPE_BOLETA, Series: "B001", Number: 40, NumberFinal: 52,
      ConsolidatedCount: 13, BuyerDocType: "", BuyerDocNumber: "", BuyerName: "",
    }), issuer))
    expect(fields[7]).toBe("03")
    expect(fields[9]).toBe("40")
    expect(fields[10]).toBe("52")
    expect(fields[11]).toBe("")
    expect(fields[13]).toBe("")
  })

  it('leaves field 10 empty on a detailed row, which declares one number', () => {
    expect(fieldsOf(buildSalesBookRecord(makeRow(), issuer))[10]).toBe("")
  })

  it('writes the modified document in fields 29 to 32 for a note', () => {
    const fields = fieldsOf(buildSalesBookRecord(makeRow({
      DocType: DOC_TYPE_CREDIT_NOTE, Series: "FC01", Number: 3,
      ModifiedIssueDate: ISSUE_DAY - 10, ModifiedDocType: DOC_TYPE_FACTURA,
      ModifiedSeries: "F001", ModifiedNumber: 7,
    }), issuer))
    expect(fields[7]).toBe("07")
    expect(fields[29]).toBe("25/08/2026")
    expect(fields[30]).toBe("01")
    expect(fields[31]).toBe("F001")
    expect(fields[32]).toBe("7")
  })

  it('leaves fields 29 to 32 empty when the row modifies nothing', () => {
    const fields = fieldsOf(buildSalesBookRecord(makeRow(), issuer))
    for (const fieldNumber of [29, 30, 31, 32]) {
      expect(fields[fieldNumber]).toBe("")
    }
  })

  it('takes a credit note back out of the period, and leaves a debit note adding', () => {
    expect(salesBookRowSign(makeRow({ DocType: DOC_TYPE_CREDIT_NOTE }))).toBe(-1)
    expect(salesBookRowSign(makeRow())).toBe(1)

    const fields = fieldsOf(buildSalesBookRecord(makeRow({
      DocType: DOC_TYPE_CREDIT_NOTE, TaxableAmount: 10000, TaxAmount: 1800, TotalAmount: 11800,
    }), issuer))
    expect(fields[15]).toBe("-100.00")
    expect(fields[17]).toBe("-18.00")
    expect(fields[26]).toBe("-118.00")
  })

  it('never lets a razón social carry a separator into the file', () => {
    const fields = fieldsOf(buildSalesBookRecord(
      makeRow({ BuyerName: "PANIFICADORA S.A. | EIRL" }),
      { RUC: "20601000001", LegalName: "GENIX S.A.C. / SUCURSAL" },
    ))
    expect(fields[2]).toBe("GENIX S.A.C. SUCURSAL")
    expect(fields[13]).toBe("PANIFICADORA S.A. EIRL")
    expect(fields).toHaveLength(34)
  })
})

describe('the file is one record per row', () => {
  it('separates records with CRLF and nothing else', () => {
    const file = buildSalesBookFile([makeRow({ Number: 1 }), makeRow({ Number: 2 })], issuer)
    const records = file.split("\r\n")
    expect(records).toHaveLength(2)
    expect(fieldsOf(records[0])[9]).toBe("1")
    expect(fieldsOf(records[1])[9]).toBe("2")
  })

  it('is empty for a period with no comprobantes', () => {
    expect(buildSalesBookFile([], issuer)).toBe("")
  })
})

describe('the file name is the Tabla 6 mask', () => {
  it('names the period, the book code and the replacement flag', () => {
    expect(salesBookFileName("20601000001", "202609", true))
      .toBe("LE2060100000120260900140400021112.TXT")
  })

  // Position 30 is the "contenido" flag: a month with no sales is still declared, as a file
  // that says it carries no information.
  it('flips the content flag for a period with no rows', () => {
    expect(salesBookFileName("20601000001", "202609", false))
      .toBe("LE2060100000120260900140400021012.TXT")
  })
})

describe('a period says why it cannot be filed', () => {
  it('lets a clean month through', () => {
    expect(salesBookExportBlockers([makeRow()], issuer, "202609")).toEqual([])
  })

  it('files a month with no comprobantes — that is what the content flag is for', () => {
    expect(salesBookExportBlockers([], issuer, "202609")).toEqual([])
  })

  it('refuses a single day', () => {
    const blockers = salesBookExportBlockers([makeRow()], issuer, SINGLE_DAY_PERIOD)
    expect(blockers).toHaveLength(1)
    expect(blockers[0]).toContain("single day")
  })

  it('refuses a company with no valid RUC', () => {
    const blockers = salesBookExportBlockers([makeRow()], { RUC: "123", LegalName: "X" }, "202609")
    expect(blockers[0]).toContain("RUC")
  })

  it('refuses a period SUNAT has not answered in full', () => {
    const blockers = salesBookExportBlockers(
      [makeRow(), makeRow({ ValidityState: BOOK_STATE_PENDING })], issuer, "202609",
    )
    expect(blockers).toHaveLength(1)
    expect(blockers[0]).toContain("awaiting")
  })

  // Field 28 is mandatory the moment a row is not in soles, and no exchange rate is stored.
  it('refuses a period holding a comprobante that is not in PEN', () => {
    const blockers = salesBookExportBlockers([makeRow({ Currency: "USD" })], issuer, "202609")
    expect(blockers).toHaveLength(1)
    expect(blockers[0]).toContain("exchange rate")
  })

  it('refuses a row whose series never joined', () => {
    const blockers = salesBookExportBlockers(
      [makeRow({ DocType: 0, Series: "" })], issuer, "202609",
    )
    expect(blockers).toHaveLength(1)
    expect(blockers[0]).toContain("no series")
  })
})
