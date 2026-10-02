import { describe, expect, it } from 'vitest'
import type { IClientProviderSnapshot } from '#services/crm/client-provider.svelte.ts'
import { purchaseDocumentTotal, splitPurchaseTotal } from '#core/purchase-document.ts'
import { SINGLE_DAY_PERIOD } from './books'
import {
  buildPurchasesBook, purchasesBookExportBlockers, sumPurchasesBook, PURCHASE_SOURCE_ASSET,
  PURCHASE_SOURCE_EXPENSE, PURCHASE_SOURCE_ORDER, type IPurchaseBookDocument,
  type IPurchasesBookResponse,
} from './books.purchases'
import {
  buildPurchasesBookRecord, purchasesBookFileName,
} from './books.purchases.txt'

const ISSUE_DAY = 20700 // 2026-09-04
const issuer = { RUC: "20601000001", LegalName: "COMERCIAL GENIX S.A.C." }

const supplier: IClientProviderSnapshot = {
  ID: 50, Name: "PROVEEDOR UNO S.A.C.", RegistryNumber: "20600000050", IdentityDocType: 6,
  CityID: "", upv: 0, ss: 1, upd: 0,
}
const identities = new Map([[supplier.ID, supplier]])

const makeDocument = (fields: Partial<IPurchaseBookDocument> = {}): IPurchaseBookDocument => ({
  Source: PURCHASE_SOURCE_ORDER, SourceID: 1, Name: "", ProviderID: 7, ProviderSnapshotID: 50,
  DueDate: 0, RecordAmount: 11800,
  DocType: 1, DocSeries: "F001", DocNumber: 123, DocIssueDate: ISSUE_DAY,
  TaxableAmount: 10000, TaxAmount: 1800, UntaxedAmount: 0, OtherAmount: 0,
  CurrencyType: 1, ExchangeRate: 0,
  ...fields,
})

const makeResponse = (documents: IPurchaseBookDocument[]): IPurchasesBookResponse => ({
  Period: "202609", FirstDay: ISSUE_DAY - 3, LastDay: ISSUE_DAY + 26, Documents: documents,
})

const fieldsOf = (record: string): string[] => ["", ...record.slice(0, -1).split("|")]

describe('buildPurchasesBook', () => {
  it('maps a factura onto a row with the pinned supplier identity', () => {
    const { rows, undocumented } = buildPurchasesBook(makeResponse([makeDocument()]), identities)
    expect(undocumented).toHaveLength(0)
    expect(rows).toHaveLength(1)
    expect(rows[0]).toMatchObject({
      Period: "202609", DocType: 1, Series: "F001", Number: 123,
      SupplierDocType: "6", SupplierDocNumber: "20600000050", SupplierName: "PROVEEDOR UNO S.A.C.",
      TaxableAmount: 10000, TaxAmount: 1800, TotalAmount: 11800, Currency: "PEN",
    })
  })

  it('keeps purchases without a comprobante out of the book', () => {
    const { rows, undocumented } = buildPurchasesBook(makeResponse([
      makeDocument({ DocType: 0, Source: PURCHASE_SOURCE_EXPENSE, SourceID: 9, Name: "Taxi", RecordAmount: 2500 }),
    ]), identities)
    expect(rows).toHaveLength(0)
    expect(undocumented).toEqual([expect.objectContaining({
      Source: PURCHASE_SOURCE_EXPENSE, SourceID: 9, Name: "Taxi", Amount: 2500, Date: ISSUE_DAY,
    })])
  })

  it('folds the records that share one factura into a single row', () => {
    // An acquisition of two serial-tracked assets: two rows, one factura, each with its share.
    const share = { Source: PURCHASE_SOURCE_ASSET, TaxableAmount: 5000, TaxAmount: 900 }
    const { rows } = buildPurchasesBook(makeResponse([
      makeDocument({ ...share, SourceID: 1 }),
      makeDocument({ ...share, SourceID: 2 }),
    ]), identities)
    expect(rows).toHaveLength(1)
    expect(rows[0].TotalAmount).toBe(11800)
    expect(rows[0].Sources.map(source => source.SourceID)).toEqual([1, 2])
  })

  it('does not fold the same number from two suppliers', () => {
    const { rows } = buildPurchasesBook(makeResponse([
      makeDocument({ ProviderID: 7 }), makeDocument({ ProviderID: 8 }),
    ]), identities)
    expect(rows).toHaveLength(2)
  })

  it('declares a due date only on a recibo de servicios públicos', () => {
    const { rows } = buildPurchasesBook(makeResponse([
      makeDocument({ DueDate: ISSUE_DAY + 10 }),
      makeDocument({ DocType: 14, DocSeries: "S001", DueDate: ISSUE_DAY + 10 }),
    ]), identities)
    expect(rows.find(row => row.DocType === 1)?.DueDate).toBe(0)
    expect(rows.find(row => row.DocType === 14)?.DueDate).toBe(ISSUE_DAY + 10)
  })
})

describe('sumPurchasesBook', () => {
  it('totals in soles, converting a dollar comprobante at its own rate', () => {
    const { rows } = buildPurchasesBook(makeResponse([
      makeDocument(),
      makeDocument({ DocNumber: 124, CurrencyType: 2, ExchangeRate: 3500 }),
    ]), identities)
    expect(sumPurchasesBook(rows)).toEqual({
      taxable: 10000 + 35000, tax: 1800 + 6300, untaxed: 0, total: 11800 + 41300, comprobantes: 2,
    })
  })
})

describe('purchasesBookExportBlockers', () => {
  it('lets a clean month through, and an empty one too', () => {
    const { rows } = buildPurchasesBook(makeResponse([makeDocument()]), identities)
    expect(purchasesBookExportBlockers(rows, issuer, "202609")).toEqual([])
    expect(purchasesBookExportBlockers([], issuer, "202609")).toEqual([])
  })

  it('stops a single day, a missing supplier identity, a dollar row without rate and a recibo without due date', () => {
    const { rows } = buildPurchasesBook(makeResponse([
      makeDocument({ ProviderSnapshotID: 0 }),
      makeDocument({ DocNumber: 2, CurrencyType: 2, ExchangeRate: 0 }),
      makeDocument({ DocNumber: 3, DocType: 14 }),
    ]), identities)
    expect(purchasesBookExportBlockers(rows, issuer, SINGLE_DAY_PERIOD)).toHaveLength(4)
  })
})

describe('the Anexo 11 record', () => {
  it('writes fields 1-37 in order and closes with a pipe', () => {
    const { rows } = buildPurchasesBook(makeResponse([makeDocument()]), identities)
    const record = buildPurchasesBookRecord(rows[0], issuer)
    const fields = fieldsOf(record)

    expect(record.endsWith("|")).toBe(true)
    expect(fields).toHaveLength(38)
    expect(fields[1]).toBe("20601000001")
    expect(fields[3]).toBe("202609")
    expect(fields[4]).toBe("")
    expect(fields[5]).toBe("04/09/2026")
    expect(fields[7]).toBe("01")
    expect(fields[8]).toBe("F001")
    expect(fields[10]).toBe("123")
    expect(fields[12]).toBe("6")
    expect(fields[13]).toBe("20600000050")
    expect(fields[15]).toBe("100.00")
    expect(fields[16]).toBe("18.00")
    expect(fields[23]).toBe("0.00")
    expect(fields[25]).toBe("118.00")
    expect(fields[26]).toBe("PEN")
    expect(fields[27]).toBe("")
  })

  it('writes a dollar rate with three decimals', () => {
    const { rows } = buildPurchasesBook(makeResponse([makeDocument({ CurrencyType: 2, ExchangeRate: 3752 })]), identities)
    const fields = fieldsOf(buildPurchasesBookRecord(rows[0], issuer))
    expect(fields[26]).toBe("USD")
    expect(fields[27]).toBe("3.752")
  })

  it('names the file after Tabla 13', () => {
    expect(purchasesBookFileName("20601000001", "202609", true))
      .toBe("LE2060100000120260900080400021112.TXT")
    expect(purchasesBookFileName("20601000001", "202609", false))
      .toBe("LE2060100000120260900080400021012.TXT")
  })
})

describe('the purchase document helpers', () => {
  it('splits a total at the general IGV rate only when the document grants crédito fiscal', () => {
    const facturaSplit = splitPurchaseTotal(1, 11800)
    expect(facturaSplit).toMatchObject({ TaxableAmount: 10000, TaxAmount: 1800 })
    expect(purchaseDocumentTotal(facturaSplit)).toBe(11800)
    expect(splitPurchaseTotal(3, 5000)).toMatchObject({ TaxableAmount: 0, TaxAmount: 0, UntaxedAmount: 5000 })
  })

  it('keeps the untaxed part and other charges, and splits only the remainder', () => {
    const facturaSplit = splitPurchaseTotal(1, 13800, 1500, 500)
    expect(facturaSplit).toMatchObject({ TaxableAmount: 10000, TaxAmount: 1800, UntaxedAmount: 1500, OtherAmount: 500 })
    expect(purchaseDocumentTotal(facturaSplit)).toBe(13800)
    // Without crédito fiscal the untaxed part is the remainder, whatever was typed.
    const boletaSplit = splitPurchaseTotal(3, 5000, 999, 300)
    expect(boletaSplit).toMatchObject({ TaxableAmount: 0, TaxAmount: 0, UntaxedAmount: 4700, OtherAmount: 300 })
  })
})
