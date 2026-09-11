import { describe, expect, it } from 'vitest'
import {
  TICKET_58MM, TICKET_80MM, centerText, formatTicketAmount, isFiscalSale, labelledLines,
  padColumns, productDisplayName, renderSaleTicket, renderTicketLine, saleCorrelativo,
  saleDocumentNumber, saleDocumentTitle, saleSeriesID, wrapText, type TicketContext,
} from './sale_ticket'
import type { SaleHistoryLine, SaleHistoryRow } from './sale_history.idb'
import type { IInvoiceSeries } from '$routes/company/configuration/invoice-series'
import type { IProduct } from '$services/production/products.svelte'

const BOLETA_SERIES: IInvoiceSeries = {
  SeriesID: 1, DocType: 3, SeriesCode: "B001", SiteID: 0, IsDefault: 1, ss: 1,
}

const makeProduct = (fields: Partial<IProduct>): IProduct => ({
  ID: 10, Name: "Zumo Sunny Delight Florida botella 31 cl.", SbuUnit: "", SbuQuantity: 0,
  Presentations: [], FinalPrice: 255, SbuFinalPrice: 0,
  ...fields,
} as unknown as IProduct)

const makeContext = (fields: Partial<TicketContext>): TicketContext => ({
  company: {
    name: "Bodega Central",
    legalName: "Distribuidora Central S.A.C.",
    ruc: "20123456789",
    address: "Av. Siempre Viva 742, Lima",
  },
  seriesByID: new Map([[1, BOLETA_SERIES]]),
  productsByID: new Map([[10, makeProduct({})]]),
  clientsByID: new Map([[7, { ID: 7, Name: "Juan Pérez", RegistryNumber: "12345678" } as any]]),
  warehousesByID: new Map([[3, { ID: 3, Name: "Almacén Principal" } as any]]),
  cashBanksByID: new Map([[5, { ID: 5, Name: "Caja Principal" } as any]]),
  cashierNamesByID: new Map([[2, "Ivan Angulo"]]),
  ...fields,
})

const makeLine = (fields: Partial<SaleHistoryLine>): SaleHistoryLine => ({
  productID: 10,
  presentationID: 0,
  quantity: { units: 2, sub: 0 },
  subDivisor: 1,
  unitPrice: 255,
  subUnitPrice: 0,
  serialNumbers: [],
  ...fields,
})

const makeRow = (fields: Partial<SaleHistoryRow>): SaleHistoryRow => ({
  // correlativo 130, random 11, series 1
  saleID: 1301101,
  savedAt: new Date(2026, 8, 10, 14, 32).getTime(),
  clientID: 7,
  warehouseID: 3,
  cashierID: 2,
  cashBankID: 5,
  status: 4,
  totalAmount: 2010,
  taxAmount: 307,
  lines: [makeLine({})],
  ...fields,
})

describe('the comprobante is read off the sale id', () => {
  it('splits the id into correlativo and series', () => {
    expect(saleSeriesID(1301101)).toBe(1)
    expect(saleCorrelativo(1301101)).toBe(130)
  })

  it('treats the series-0 tail as a sale with no comprobante', () => {
    expect(isFiscalSale(420700)).toBe(false)
    expect(isFiscalSale(1301101)).toBe(true)
  })

  it('pads the correlativo to eight digits', () => {
    expect(saleDocumentNumber(1301101, BOLETA_SERIES)).toBe("B001-00000130")
  })

  it('falls back to the series number when the series is gone from the company', () => {
    expect(saleDocumentNumber(1301101, undefined)).toBe("#1-00000130")
  })

  it('numbers a sale with no series by its own id', () => {
    expect(saleDocumentNumber(420700, undefined)).toBe("VENTA #420700")
  })

  it('titles the document by its type, and an internal sale as a note', () => {
    expect(saleDocumentTitle(1301101, BOLETA_SERIES)).toBe("BOLETA DE VENTA ELECTRÓNICA")
    expect(saleDocumentTitle(1301101, { ...BOLETA_SERIES, DocType: 1 })).toBe("FACTURA ELECTRÓNICA")
    expect(saleDocumentTitle(420700, undefined)).toBe("NOTA DE VENTA")
  })
})

describe('a product is named with its presentation', () => {
  const product = makeProduct({
    Name: "Zumo", Presentations: [{ id: 2, nm: "Pack x6" }] as unknown as IProduct["Presentations"],
  })

  it('appends the presentation when the line carries one', () => {
    expect(productDisplayName(product, 2)).toBe("Zumo (Pack x6)")
    expect(productDisplayName(product, 0)).toBe("Zumo")
  })

  it('says so when the catalog no longer has the product', () => {
    expect(productDisplayName(undefined, 0)).toBe("(producto no encontrado)")
  })
})

describe('amounts', () => {
  it('always carries two decimals', () => {
    expect(formatTicketAmount(2010)).toBe("20.10")
    expect(formatTicketAmount(5)).toBe("0.05")
    expect(formatTicketAmount(0)).toBe("0.00")
  })

  it('groups thousands and keeps the sign', () => {
    expect(formatTicketAmount(123456)).toBe("1,234.56")
    expect(formatTicketAmount(-2010)).toBe("-20.10")
  })
})

describe('rows are built for a fixed column count', () => {
  it('right-aligns the amount and cuts the left side instead', () => {
    expect(padColumns("CANT", "9.90", 12)).toBe("CANT    9.90")
    const cut = padColumns("Un nombre larguísimo de producto", "9.90", 12)
    expect(cut).toHaveLength(12)
    expect(cut.endsWith("9.90")).toBe(true)
  })

  it('centers without padding the right edge', () => {
    expect(centerText("TOTAL", 11)).toBe("   TOTAL")
  })

  it('wraps on words and cuts a word wider than the paper', () => {
    expect(wrapText("uno dos tres cuatro", 9)).toEqual(["uno dos", "tres", "cuatro"])
    expect(wrapText("supercalifragilistico", 8)).toEqual(["supercal", "ifragili", "stico"])
  })

  it('hangs a wrapped label under its value', () => {
    expect(labelledLines("CLIENTE", "Juan Pérez de la Torre", 20)).toEqual([
      "CLIENTE: Juan Pérez",
      "         de la Torre",
    ])
  })
})

describe('a cart line prices its two halves separately', () => {
  it('prints one priced row for whole units', () => {
    const rows = renderTicketLine(makeLine({}), makeProduct({ Name: "Agua" }), TICKET_58MM)
    expect(rows[0]).toBe("Agua")
    expect(rows[1]).toBe(padColumns("  2 x 2.55", "5.10", TICKET_58MM))
  })

  it('prints a second row for sub-units at their own price', () => {
    const rows = renderTicketLine(makeLine({
      quantity: { units: 1, sub: 250 }, subDivisor: 1000, unitPrice: 3000, subUnitPrice: 3,
    }), makeProduct({ Name: "Jamón", SbuUnit: "g" }), TICKET_58MM)

    expect(rows[1]).toBe(padColumns("  1 x 30.00", "30.00", TICKET_58MM))
    expect(rows[2]).toBe(padColumns("  250 g x 0.03", "7.50", TICKET_58MM))
  })

  it('lists every serial number under the line', () => {
    const rows = renderTicketLine(
      makeLine({ serialNumbers: ["SN-0001", "SN-0002 x2"] }),
      makeProduct({ Name: "Taladro" }), TICKET_58MM)
    expect(rows).toContain("  S/N: SN-0001")
    expect(rows).toContain("  S/N: SN-0002 x2")
  })
})

describe('the whole ticket fits the paper', () => {
  const longNameRow = makeRow({
    lines: [
      makeLine({ productID: 10 }),
      makeLine({ productID: 11, serialNumbers: ["ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"] }),
    ],
  })
  const longNameContext = makeContext({
    productsByID: new Map([
      [10, makeProduct({ Name: "Zumo pacífico zero sin azúcar añadido Bifrutas sin gluten pack de 3 briks de 330 ml." })],
      [11, makeProduct({ ID: 11, Name: "Agua" })],
    ]),
  })

  it.each([TICKET_58MM, TICKET_80MM])('never exceeds %i columns', (columns) => {
    const ticketRows = renderSaleTicket(longNameRow, longNameContext, columns).split("\n")
    for (const ticketRow of ticketRows) {
      expect(ticketRow.length).toBeLessThanOrEqual(columns)
    }
  })

  it('names the document, its number and the parties it resolved', () => {
    const ticket = renderSaleTicket(makeRow({}), makeContext({}), TICKET_58MM)
    expect(ticket).toContain("BOLETA DE VENTA ELECTRÓNICA")
    expect(ticket).toContain("B001-00000130")
    expect(ticket).toContain("RUC 20123456789")
    expect(ticket).toContain("CLIENTE: Juan Pérez")
    expect(ticket).toContain("CAJERO: Ivan Angulo")
    expect(ticket).toContain("PAGADO - Caja Principal")
    expect(ticket).toContain("ENTREGADO")
  })

  it('declares itself a printed copy only when it is a comprobante', () => {
    const fiscal = renderSaleTicket(makeRow({}), makeContext({}), TICKET_80MM)
    expect(fiscal).toContain("Representación impresa")

    const internal = renderSaleTicket(makeRow({ saleID: 420700 }), makeContext({}), TICKET_80MM)
    expect(internal).not.toContain("Representación impresa")
    expect(internal).toContain("NOTA DE VENTA")
  })

  it('falls back to VARIOS when no client was named', () => {
    const ticket = renderSaleTicket(makeRow({ clientID: 0 }), makeContext({}), TICKET_58MM)
    expect(ticket).toContain("CLIENTE: VARIOS")
  })

  it('says the sale is unpaid when its status says so', () => {
    const ticket = renderSaleTicket(makeRow({ status: 3 }), makeContext({}), TICKET_58MM)
    expect(ticket).toContain("PENDIENTE DE PAGO")
  })

  it('prints no cashier line for a sale rung up by another user', () => {
    const ticket = renderSaleTicket(makeRow({ cashierID: 99 }), makeContext({}), TICKET_58MM)
    expect(ticket).not.toContain("CAJERO")
  })
})
