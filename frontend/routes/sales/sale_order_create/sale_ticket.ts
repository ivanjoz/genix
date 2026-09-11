// Renders a sale as the plain monospace text a thermal printer puts on paper.
// Pure: no Svelte, no fetch, no DOM.
//
// A ticket printer has no layout engine — it prints a fixed number of characters per row and
// nothing else — so the whole ticket is built here as a string of padded rows, and the Svelte
// side only drops it into a <pre>. Every helper below exists because of that one constraint.
//
// Nothing is read off the stored row that a service can resolve: the history keeps ids, and
// the page hands the record maps in as a TicketContext. Whether the ticket is a comprobante or
// an internal note is read off the sale id itself, whose last two digits are the invoicing
// series — 0 meaning the till named none (backend/sales/types/sale_order_id.go).

import { normalizeQuantity } from '$core/quantity'
import { DOC_TYPE_BOLETA, DOC_TYPE_FACTURA, type IInvoiceSeries } from '$routes/company/configuration/invoice-series'
import type { IWarehouse } from '$routes/business/branches-warehouses/branches-warehouses.svelte'
import type { ICashBank } from '$routes/finance/cash-banks/cajas.svelte'
import type { IClientProvider } from '$services/crm/client-provider.svelte'
import type { IProduct } from '$services/production/products.svelte'
import type { SaleHistoryLine, SaleHistoryRow } from './sale_history.idb'
import { isSaleDelivered, isSalePaid } from './sale_history'

// The two dominant ESC/POS roll widths, in characters at Font A: 58 mm prints 32 columns,
// 80 mm prints 48. The number of columns is the only thing the renderer needs to know.
export const TICKET_58MM = 32
export const TICKET_80MM = 48

export const TICKET_WIDTH_OPTIONS = [
  { ID: TICKET_58MM, Name: "58 mm" },
  { ID: TICKET_80MM, Name: "80 mm" },
]

export interface TicketCompany {
  name: string
  legalName: string
  ruc: string
  address: string
}

// Composition, not copies: the ticket is given the same records the page already holds, and
// resolves each id against them at render time.
export interface TicketContext {
  company: TicketCompany
  seriesByID: Map<number, IInvoiceSeries>
  productsByID: Map<number, IProduct>
  clientsByID: Map<number, IClientProvider>
  warehousesByID: Map<number, IWarehouse>
  cashBanksByID: Map<number, ICashBank>
  // Only the session user is known here — there is no users service at the till — so a ticket
  // rung up by someone else simply prints no cashier line.
  cashierNamesByID: Map<number, string>
}

// The legal titles SUNAT expects on a printed representation. These are not translatable
// strings: they are what the document is called. Anything else is an internal note.
const DOCUMENT_TITLES = new Map<number, string>([
  [DOC_TYPE_FACTURA, "FACTURA ELECTRÓNICA"],
  [DOC_TYPE_BOLETA, "BOLETA DE VENTA ELECTRÓNICA"],
])

// The two halves the sale id is packed with. Mirrors SaleOrder.SeriesID / .Correlativo in Go.
export const saleSeriesID = (saleID: number): number => saleID % 100
export const saleCorrelativo = (saleID: number): number => Math.floor(saleID / 10000)

export const isFiscalSale = (saleID: number): boolean => saleSeriesID(saleID) !== 0

export const saleDocumentTitle = (saleID: number, series?: IInvoiceSeries): string => {
  if (!isFiscalSale(saleID)) return "NOTA DE VENTA"
  return DOCUMENT_TITLES.get(series?.DocType || 0) || "COMPROBANTE ELECTRÓNICO"
}

// Padded to 8 digits, which is how the number is printed on the document itself. The
// invoicing panel shows it unpadded because it is a list column, not a comprobante.
export const saleDocumentNumber = (saleID: number, series?: IInvoiceSeries): string => {
  const seriesID = saleSeriesID(saleID)
  if (seriesID === 0) return `VENTA #${saleID}`
  // A series retired after the sale may no longer be in the company's list, and the ticket
  // still has to read as something — the same fallback the invoicing panel uses.
  const seriesCode = series?.SeriesCode || `#${seriesID}`
  return `${seriesCode}-${String(saleCorrelativo(saleID)).padStart(8, "0")}`
}

// The product as the cart named it: the presentation is part of the identity of what was
// sold, so a ticket that omits it does not say which box left the shelf.
export const productDisplayName = (product: IProduct | undefined, presentationID: number): string => {
  if (!product) return "(producto no encontrado)"
  if (!presentationID) return product.Name
  const presentation = product.Presentations?.find((option) => option.id === presentationID)
  return presentation ? `${product.Name} (${presentation.nm})` : product.Name
}

// Own money formatter rather than $libs/helpers.formatN: that module initializes notiflix at
// import time, and this one has to stay importable without a DOM.
export const formatTicketAmount = (cents: number): string => {
  const rounded = Math.round(cents)
  const absolute = Math.abs(rounded)
  const wholeUnits = String(Math.floor(absolute / 100)).replace(/\B(?=(\d{3})+(?!\d))/g, ",")
  return `${rounded < 0 ? "-" : ""}${wholeUnits}.${String(absolute % 100).padStart(2, "0")}`
}

export const formatTicketDateTime = (epochMilliseconds: number): string => {
  const at = new Date(epochMilliseconds)
  const pad = (value: number) => String(value).padStart(2, "0")
  return `${pad(at.getDate())}/${pad(at.getMonth() + 1)}/${at.getFullYear()} ${pad(at.getHours())}:${pad(at.getMinutes())}`
}

export const dividerLine = (columns: number): string => "-".repeat(columns)

// No trailing padding: a printer advances to the next row on the newline, so spaces on the
// right are only wasted characters.
export const centerText = (text: string, columns: number): string => {
  if (text.length >= columns) return text.slice(0, columns)
  return " ".repeat(Math.floor((columns - text.length) / 2)) + text
}

// The right side is always the amount and is never truncated — the left side gives way,
// because a cut price is a wrong ticket while a cut name is only an ugly one.
export const padColumns = (left: string, right: string, columns: number): string => {
  const roomForLeft = columns - right.length - 1
  if (roomForLeft <= 0) return right.slice(-columns)
  const leftText = left.length > roomForLeft ? left.slice(0, roomForLeft) : left
  return leftText + " ".repeat(columns - leftText.length - right.length) + right
}

export const wrapText = (text: string, columns: number): string[] => {
  const words = (text || "").split(/\s+/).filter(Boolean)
  if (words.length === 0 || columns <= 0) return []

  const wrappedLines: string[] = []
  let currentLine = ""

  for (const word of words) {
    if (!currentLine) {
      currentLine = word
    } else if (currentLine.length + 1 + word.length <= columns) {
      currentLine += " " + word
    } else {
      wrappedLines.push(currentLine)
      currentLine = word
    }
    // A single word wider than the paper has to be cut, or the printer drops the overflow.
    while (currentLine.length > columns) {
      wrappedLines.push(currentLine.slice(0, columns))
      currentLine = currentLine.slice(columns)
    }
  }

  if (currentLine) wrappedLines.push(currentLine)
  return wrappedLines
}

// "CLIENTE: Nombre muy largo" with a hanging indent, so a wrapped value stays under the
// value and not under the label.
export const labelledLines = (label: string, value: string, columns: number): string[] => {
  const prefix = `${label}: `
  const wrapped = wrapText(value, columns - prefix.length)
  return wrapped.map((text, index) => index === 0 ? prefix + text : " ".repeat(prefix.length) + text)
}

// One cart line as its rows: the name wrapped over the full width, then a priced row per
// half of the quantity. Whole units and sub-units are charged at their own prices — never
// prorated — so they are printed as the two separate rows they are.
export const renderTicketLine = (
  line: SaleHistoryLine, product: IProduct | undefined, columns: number,
): string[] => {

  const rows = wrapText(productDisplayName(product, line.presentationID), columns)
  const quantity = normalizeQuantity(line.quantity, line.subDivisor || 1)

  if (quantity.units !== 0) {
    rows.push(padColumns(
      `  ${quantity.units} x ${formatTicketAmount(line.unitPrice)}`,
      formatTicketAmount(quantity.units * line.unitPrice), columns))
  }
  if (quantity.sub !== 0) {
    const subUnitLabel = product?.SbuUnit ? ` ${product.SbuUnit}` : ""
    rows.push(padColumns(
      `  ${quantity.sub}${subUnitLabel} x ${formatTicketAmount(line.subUnitPrice)}`,
      formatTicketAmount(quantity.sub * line.subUnitPrice), columns))
  }
  // Through labelledLines and not wrapText: wrapText splits on whitespace and would eat the
  // indent that ties the serial to its line.
  for (const serialNumber of line.serialNumbers || []) {
    rows.push(...labelledLines("  S/N", serialNumber, columns))
  }
  return rows
}

export const renderSaleTicket = (
  row: SaleHistoryRow, context: TicketContext, columns: number,
): string => {

  const { company } = context
  const series = context.seriesByID.get(saleSeriesID(row.saleID))
  const client = context.clientsByID.get(row.clientID)

  const ticketRows: string[] = []
  const pushCentered = (text: string) => {
    for (const wrapped of wrapText(text, columns)) ticketRows.push(centerText(wrapped, columns))
  }

  pushCentered(company.legalName || company.name)
  // The trade name is only worth a second header when it is not the legal name repeated.
  if (company.name && company.legalName && company.name !== company.legalName) {
    pushCentered(company.name)
  }
  if (company.ruc) pushCentered(`RUC ${company.ruc}`)
  pushCentered(company.address)

  ticketRows.push(dividerLine(columns))
  pushCentered(saleDocumentTitle(row.saleID, series))
  pushCentered(saleDocumentNumber(row.saleID, series))
  ticketRows.push(dividerLine(columns))

  ticketRows.push(...labelledLines("FECHA", formatTicketDateTime(row.savedAt), columns))

  const cashierName = context.cashierNamesByID.get(row.cashierID)
  if (cashierName) ticketRows.push(...labelledLines("CAJERO", cashierName, columns))

  const warehouse = context.warehousesByID.get(row.warehouseID)
  if (warehouse) ticketRows.push(...labelledLines("ALMACEN", warehouse.Name, columns))

  ticketRows.push(...labelledLines("CLIENTE", client?.Name || "VARIOS", columns))
  if (client?.RegistryNumber) {
    ticketRows.push(...labelledLines("DOC", client.RegistryNumber, columns))
  }

  ticketRows.push(dividerLine(columns))
  ticketRows.push(padColumns("DESCRIPCION", "IMPORTE", columns))
  ticketRows.push(dividerLine(columns))
  for (const line of row.lines) {
    ticketRows.push(...renderTicketLine(line, context.productsByID.get(line.productID), columns))
  }

  ticketRows.push(dividerLine(columns))
  ticketRows.push(padColumns("OP. GRAVADA",
    formatTicketAmount(row.totalAmount - row.taxAmount), columns))
  ticketRows.push(padColumns("IGV", formatTicketAmount(row.taxAmount), columns))
  ticketRows.push(padColumns("TOTAL S/", formatTicketAmount(row.totalAmount), columns))
  ticketRows.push(dividerLine(columns))

  const cashBank = context.cashBanksByID.get(row.cashBankID)
  pushCentered(isSalePaid(row.status)
    ? `PAGADO${cashBank ? ` - ${cashBank.Name}` : ""}`
    : "PENDIENTE DE PAGO")
  if (isSaleDelivered(row.status)) pushCentered("ENTREGADO")

  ticketRows.push("")
  // SUNAT requires the printed copy of an electronic document to say that it is one.
  if (isFiscalSale(row.saleID)) {
    pushCentered("Representación impresa del comprobante electrónico")
    ticketRows.push("")
  }
  pushCentered("¡Gracias por su compra!")

  return ticketRows.join("\n")
}
