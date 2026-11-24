import { periodOfDay } from '#routes/hr/payroll/payroll.ts'
import type { ICashFlowCommitment, ICashFlowEntry } from '#services/finance/cash-flow.svelte.ts'

// Pure logic of the cash flow forecast (plan/CASH_FLOW_FORECAST_PLAN.md §2, §13): it lays the
// dated commitments (the backend's, and the browser's Forecast layer) and the planned entries on a
// day grid for one lane (a site, the company lane or the total), and rolls the balance from today's
// cash. No Svelte, no fetch.

/** The daily view: 13 weeks. */
export const CASH_FLOW_HORIZON_DAYS = 91
/** The monthly view and what the page fetches: a year. */
export const CASH_FLOW_YEAR_DAYS = 365
export const CURRENCY_USD = 2

// Layers: how certain a flow is. Known and Scheduled come from the backend and the cash-documents
// frame; the browser forecasts the rest; Planned is the user's entries.
export const CASH_FLOW_LAYER_KNOWN = 1
export const CASH_FLOW_LAYER_SCHEDULED = 2
export const CASH_FLOW_LAYER_FORECAST = 3
export const CASH_FLOW_LAYER_PLANNED = 4
export const CASH_FLOW_LAYER_NAMES: Record<number, string> = {
  [CASH_FLOW_LAYER_KNOWN]: 'Known|Conocido', [CASH_FLOW_LAYER_SCHEDULED]: 'Scheduled|Programado',
  [CASH_FLOW_LAYER_FORECAST]: 'Forecast|Pronóstico', [CASH_FLOW_LAYER_PLANNED]: 'Planned|Planificado',
}

// The Forecast layer's streams, after the backend's 1–12 (browser only).
export const CASH_FLOW_STREAM_FORECAST_COLLECTIONS = 13
export const CASH_FLOW_STREAM_FORECAST_OUTFLOWS = 14
export const CASH_FLOW_STREAM_FORECAST_TAXES = 15

/** The lane of the whole company: every site plus the company lane. */
export const LANE_TOTAL = -1
/** Flows that belong to no site: taxes, payroll, loans, unassigned registers. */
export const LANE_COMPANY = 0
/**
 * The Banco de la Nación detracciones account: the detracciones still to deposit come in, and its
 * balance pays the taxes first. Out of the available cash, so the company's total leaves it out.
 */
export const LANE_DETRACTIONS = -2
export const CASH_BANK_TYPE_DETRACTIONS = 3
// The tax streams the detracciones account pays: IGV, renta, Nuevo RUS and the forecast taxes.
const TAX_STREAMS = [10, 11, 12, CASH_FLOW_STREAM_FORECAST_TAXES]

// Streams, mirroring backend CashFlowStream* (1–4 and 10–12 are built from the cash-documents frame).
// Their order is the table's row order.
export const CASH_FLOW_STREAMS: { id: number, name: string }[] = [
  { id: 1, name: 'Receivables|Cobranzas' },
  { id: 2, name: 'Purchase orders|Órdenes de compra' },
  { id: 3, name: 'Expenses|Gastos' },
  { id: 4, name: 'Assets|Activos' },
  { id: 5, name: 'Recurring bills|Gastos programados' },
  { id: 6, name: 'Payroll net pay|Planilla neto' },
  { id: 7, name: 'CTS|CTS' },
  { id: 8, name: 'Payroll contributions|Aportes de planilla' },
  { id: 9, name: 'Loan installments|Cuotas de préstamos' },
  { id: 10, name: 'IGV|IGV' },
  { id: 11, name: 'Income tax prepayment|Pago a cuenta renta' },
  { id: 12, name: 'Nuevo RUS|Nuevo RUS' },
  { id: CASH_FLOW_STREAM_FORECAST_COLLECTIONS, name: 'Forecast collections|Cobranzas pronosticadas' },
  { id: CASH_FLOW_STREAM_FORECAST_OUTFLOWS, name: 'Forecast purchases and expenses|Compras y gastos pronosticados' },
  { id: CASH_FLOW_STREAM_FORECAST_TAXES, name: 'Forecast taxes|Impuestos pronosticados' },
]
/** The row of the planned entries, after the backend's streams. */
export const PLANNED_STREAM_ID = 100

// Planned entry kinds, mirroring backend CashFlowEntry*, with the direction each one moves money.
export const CASH_FLOW_ENTRY_KINDS: { id: number, name: string, isInflow: boolean }[] = [
  { id: 1, name: 'Capital expenditure|Inversión (capex)', isInflow: false },
  { id: 2, name: 'Dividend|Dividendos', isInflow: false },
  { id: 3, name: 'Capital contribution|Aporte de capital', isInflow: true },
  { id: 4, name: 'Other income|Otro ingreso', isInflow: true },
  { id: 5, name: 'Other outflow|Otra salida', isInflow: false },
]

export interface ICashFlowCell {
  /** PEN cents. */
  amount: number
  /**
   * What the cell adds up, for the drill-down. kind and documentDate mark an item of the cash-documents
   * frame: it stands for documents the drill-down loads.
   */
  items: { label: string, date: number, dueDate: number, amountPEN: number, layer: number, kind?: number, documentDate?: number }[]
}

export interface ICashFlowRow {
  streamID: number
  /** One per grid day. */
  cells: ICashFlowCell[]
  total: number
}

export interface ICashFlowGrid {
  days: number[]
  rows: ICashFlowRow[]
  /** Today's cash of the lane, PEN cents. */
  opening: number
  /** The balance at the end of each day. */
  closing: number[]
}

/**
 * A register (or any record with a site) belongs to the lane: its site, the company lane, or any but
 * the detracciones account's.
 */
export const isInLane = (siteID: number, lane: number) => lane === LANE_TOTAL ? siteID !== LANE_DETRACTIONS : siteID === lane

/** The cash banks by lane: the detracciones account is its own lane, whatever its site. */
export const cashBanksByLane = <T extends { Type: number, SiteID: number }>(cashBanks: T[]): T[] =>
  cashBanks.map((cashBank) => cashBank.Type === CASH_BANK_TYPE_DETRACTIONS ? { ...cashBank, SiteID: LANE_DETRACTIONS } : cashBank)

/**
 * payTaxesFromDetractions pays each tax outflow from the detracciones account first, in date order, as
 * far as its balance that day goes (today's balance plus the detracciones deposited since, less the
 * taxes it already paid): that part moves to the detracciones lane, the rest stays where it was.
 * Amounts in PEN cents; the commitments are in soles (taxes and detracciones always are).
 */
export function payTaxesFromDetractions(commitments: ICashFlowCommitment[], detractionsOpening: number): ICashFlowCommitment[] {
  const deposits = commitments.filter((commitment) => commitment.SiteID === LANE_DETRACTIONS && commitment.Amount > 0)
    .sort((left, right) => left.Date - right.Date)
  const isTax = (commitment: ICashFlowCommitment) => TAX_STREAMS.includes(commitment.Stream) && commitment.Amount < 0 &&
    commitment.SiteID !== LANE_DETRACTIONS
  const taxesByDate = commitments.filter(isTax).sort((left, right) => left.Date - right.Date)
  const paidByTax = new Map<ICashFlowCommitment, number>()
  let balance = detractionsOpening, depositIndex = 0
  for (const tax of taxesByDate) {
    while (depositIndex < deposits.length && deposits[depositIndex].Date <= tax.Date) balance += deposits[depositIndex++].Amount
    const paid = Math.min(Math.max(balance, 0), -tax.Amount)
    if (paid <= 0) continue
    paidByTax.set(tax, paid)
    balance -= paid
  }
  return commitments.flatMap((commitment) => {
    const paid = paidByTax.get(commitment)
    if (!paid) return [commitment]
    const fromDetractions = { ...commitment, SiteID: LANE_DETRACTIONS, Amount: -paid,
      Label: `${commitment.Label} (cuenta de detracciones)` }
    return paid === -commitment.Amount ? [fromDetractions] : [{ ...commitment, Amount: commitment.Amount + paid }, fromDetractions]
  })
}

/** Cents in soles: dollars at the last sell rate (× 1000), held flat over the horizon. */
export const toPEN = (amount: number, currencyType: number, usdSellRate: number) =>
  currencyType === CURRENCY_USD ? Math.round(amount * usdSellRate / 1000) : amount

/** The planned entries the forecast counts: active and dated today or later. */
export const countedEntries = (entries: ICashFlowEntry[], today: number) =>
  entries.filter((entry) => entry.ss > 0 && entry.Date >= today)

/**
 * The lane's day grid: one row per stream that moves money in the window (the planned entries
 * last), the opening cash, and the closing balance of each day.
 */
export function buildCashFlowGrid(input: {
  commitments: ICashFlowCommitment[]
  entries: ICashFlowEntry[]
  cashBanks: { SiteID: number, CurrencyType: number, CurrentAmount: number }[]
  lane: number
  today: number
  horizonDays: number
  usdSellRate: number
}): ICashFlowGrid {
  const { lane, today, horizonDays, usdSellRate } = input
  const days = Array.from({ length: horizonDays }, (_, offset) => today + offset)
  const rowsByStream = new Map<number, ICashFlowRow>()

  const addToCell = (streamID: number, date: number, item: ICashFlowCell['items'][number]) => {
    const dayIndex = date - today
    if (dayIndex < 0 || dayIndex >= horizonDays) return
    let row = rowsByStream.get(streamID)
    if (!row) {
      row = { streamID, cells: days.map(() => ({ amount: 0, items: [] })), total: 0 }
      rowsByStream.set(streamID, row)
    }
    row.cells[dayIndex].amount += item.amountPEN
    row.cells[dayIndex].items.push(item)
    row.total += item.amountPEN
  }

  for (const commitment of input.commitments) {
    if (!isInLane(commitment.SiteID, lane)) continue
    addToCell(commitment.Stream, commitment.Date, {
      label: commitment.Label, date: commitment.Date, dueDate: commitment.DueDate,
      amountPEN: toPEN(commitment.Amount, commitment.CurrencyType, usdSellRate), layer: commitment.Layer,
      kind: commitment.Kind, documentDate: commitment.DocumentDate,
    })
  }
  for (const entry of countedEntries(input.entries, today)) {
    if (!isInLane(entry.SiteID, lane)) continue
    addToCell(PLANNED_STREAM_ID, entry.Date, {
      label: entry.Name, date: entry.Date, dueDate: entry.Date,
      amountPEN: toPEN(entry.Amount, entry.CurrencyType, usdSellRate), layer: CASH_FLOW_LAYER_PLANNED,
    })
  }

  const streamOrder = [...CASH_FLOW_STREAMS.map((stream) => stream.id), PLANNED_STREAM_ID]
  const rows = streamOrder.flatMap((streamID) => rowsByStream.get(streamID) || [])
  const opening = input.cashBanks
    .filter((cashBank) => isInLane(cashBank.SiteID, lane))
    .reduce((total, cashBank) => total + toPEN(cashBank.CurrentAmount, cashBank.CurrencyType, usdSellRate), 0)

  // closing(d) = closing(d − 1) + Σ flows(d), from today's cash.
  const closing: number[] = []
  let balance = opening
  for (let dayIndex = 0; dayIndex < horizonDays; dayIndex++) {
    balance += rows.reduce((total, row) => total + row.cells[dayIndex].amount, 0)
    closing.push(balance)
  }
  return { days, rows, opening, closing }
}

/**
 * The lanes the page offers: the total, then each active site, then the company lane, then the
 * detracciones account when the company has one.
 */
export function cashFlowLanes(sites: { ID: number, Name: string, ss: number }[], hasDetractionsAccount: boolean): { id: number, name: string }[] {
  return [
    { id: LANE_TOTAL, name: 'All the company|Toda la empresa' },
    ...sites.filter((site) => site.ss > 0).map((site) => ({ id: site.ID, name: site.Name })),
    { id: LANE_COMPANY, name: 'Company lane (no site)|Empresa (sin sede)' },
    ...(hasDetractionsAccount ? [{ id: LANE_DETRACTIONS, name: 'Detracciones (Banco de la Nación)' }] : []),
  ]
}

/** The detracciones account's balance today, PEN cents. */
export const detractionsOpeningOf = (cashBanks: { Type: number, CurrentAmount: number, ss: number }[]) =>
  cashBanks.filter((cashBank) => cashBank.Type === CASH_BANK_TYPE_DETRACTIONS && cashBank.ss > 0)
    .reduce((total, cashBank) => total + cashBank.CurrentAmount, 0)

/** The first day the balance goes below zero, or 0 when it never does. */
export const firstNegativeDay = (grid: ICashFlowGrid) =>
  grid.days[grid.closing.findIndex((balance) => balance < 0)] || 0

/**
 * The day grid folded by month (the monthly view): a column per calendar month, its first column the
 * rest of the current month. Each column is keyed by its first grid day; its cells add up the days
 * and its closing is the balance at its last day.
 */
export function monthlyGrid(dailyGrid: ICashFlowGrid): ICashFlowGrid {
  const periodOfGridDay = dailyGrid.days.map((day) => periodOfDay(day))
  const monthPeriods = [...new Set(periodOfGridDay)]
  const columnOfDay = periodOfGridDay.map((period) => monthPeriods.indexOf(period))
  const days = monthPeriods.map((period) => dailyGrid.days[periodOfGridDay.indexOf(period)])
  const rows = dailyGrid.rows.map((row) => {
    const cells: ICashFlowCell[] = days.map(() => ({ amount: 0, items: [] }))
    row.cells.forEach((cell, dayIndex) => {
      cells[columnOfDay[dayIndex]].amount += cell.amount
      cells[columnOfDay[dayIndex]].items.push(...cell.items)
    })
    return { ...row, cells }
  })
  const closing = monthPeriods.map((period) => dailyGrid.closing[periodOfGridDay.lastIndexOf(period)])
  return { days, rows, opening: dailyGrid.opening, closing }
}
