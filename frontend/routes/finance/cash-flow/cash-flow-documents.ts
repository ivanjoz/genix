import type { FrameFileRows } from '@genix/ui/cache'
import { periodOfDay } from '#routes/hr/payroll/payroll.ts'
import type {
  ICashDocumentDetail, ICashFlowCommitment, ICashFlowOldPayables, ICashFlowServerWarning, ICashFlowTaxOutlook,
} from '#services/finance/cash-flow.svelte.ts'
import {
  igvOfGross, isMemoTaxGroup, TAX_GROUP_DETRACTION, TAX_GROUP_FREE, TAX_GROUP_NO_DOCUMENT, TAX_GROUP_RETENTION, TAX_GROUP_TAXED_18,
} from '#core/tax-groups.ts'
import { CASH_FLOW_LAYER_KNOWN, CURRENCY_USD, LANE_DETRACTIONS } from './cash-flow.ts'

// The cash-documents frame in the browser (plan/CASH_DOCUMENTS_FRAME_PLAN.md): every sale, expense,
// purchase order and asset purchase, summed per day, site, Kind, tax group, currency and due offset.
// From it come the Known layer of the open documents, the taxes of the closed periods, the bases of
// the open ones, and the document warnings, as counts and amounts. Pure: no Svelte, no fetch.

// Kinds: the source and whether the amount is still open (backend finance/types/cash_documents.go).
export const KIND_SALE_OPEN = 1
export const KIND_SALE_SETTLED = 2
export const KIND_EXPENSE_OPEN = 10
export const KIND_EXPENSE_SETTLED = 11
export const KIND_SCHEDULED_EXPENSE_OPEN = 12
export const KIND_SCHEDULED_EXPENSE_SETTLED = 13
export const KIND_PURCHASE_ORDER_OPEN = 20
export const KIND_PURCHASE_ORDER_SETTLED = 21
export const KIND_ASSET_OPEN = 30
export const KIND_ASSET_SETTLED = 31

/** DueOffset of a document without a due date. */
export const NO_DUE_DATE = 1023
const MAX_DUE_OFFSET = 1022
/** DocumentOffset of a dollar document whose comprobante is dated on its own date. */
export const SAME_DAY_OFFSET = 128
/** Receivables overdue longer than this are left out: probably uncollectible, or a payment nobody recorded. */
export const STALE_RECEIVABLE_DAYS = 90
/** Days of the frame the page reads; older open payables come from the backend as one total. */
export const DOCUMENT_WINDOW_DAYS = 365

// The frame's columns: Keys [Date, SiteID], Rows [Kind, TaxGroup, DocumentOffset, DueOffset], Sums [Amount].
const KEY_DATE = 0
const KEY_SITE = 1
const ROW_KIND = 0
const ROW_TAX_GROUP = 1
const ROW_DOCUMENT_OFFSET = 2
const ROW_DUE_OFFSET = 3
const SUM_AMOUNT = 0

export interface ICashDocumentRow {
  date: number
  siteID: number
  kind: number
  taxGroup: number
  documentOffset: number
  dueOffset: number
  /** Cents of local currency, or of dollars when documentOffset > 0. */
  amount: number
}

/** The rows of the frame's files, flat. */
export function cashDocumentRowsOf(files: FrameFileRows[]): ICashDocumentRow[] {
  const rows: ICashDocumentRow[] = []
  for (const file of files) {
    const kinds = file.rows[ROW_KIND] as number[]
    for (let rowIndex = 0; rowIndex < kinds.length; rowIndex++) {
      rows.push({
        date: file.keys[KEY_DATE] as number, siteID: file.keys[KEY_SITE] as number, kind: kinds[rowIndex],
        taxGroup: (file.rows[ROW_TAX_GROUP] as number[])[rowIndex], documentOffset: (file.rows[ROW_DOCUMENT_OFFSET] as number[])[rowIndex],
        dueOffset: (file.rows[ROW_DUE_OFFSET] as number[])[rowIndex], amount: (file.sums[SUM_AMOUNT] as number[])[rowIndex],
      })
    }
  }
  return rows
}

export const isSaleKind = (kind: number) => kind === KIND_SALE_OPEN || kind === KIND_SALE_SETTLED
export const isScheduledExpenseKind = (kind: number) => kind === KIND_SCHEDULED_EXPENSE_OPEN || kind === KIND_SCHEDULED_EXPENSE_SETTLED
const isExpenseKind = (kind: number) => kind >= KIND_EXPENSE_OPEN && kind <= KIND_SCHEDULED_EXPENSE_SETTLED
/** The spending a Kind belongs to, open and settled together: 10 expenses, 20 purchase orders, 30 assets. */
export const spendingGroupOf = (kind: number) => Math.floor(kind / 10) * 10

/** The due date of a row, 0 without one. */
export const dueDateOf = (row: { date: number, dueOffset: number }) => row.dueOffset === NO_DUE_DATE ? 0 : row.date + row.dueOffset

/** The due offset the frame stores for a document's due date (clipped as the backend clips it). */
export const dueOffsetOf = (date: number, dueDate: number) =>
  dueDate === 0 ? NO_DUE_DATE : Math.min(Math.max(dueDate - date, 0), MAX_DUE_OFFSET)

/**
 * A sell rate (× 1000) per day: the SUNAT rate of the day, or of the last day before it that has one
 * (weekends, holidays), else the fallback.
 */
export function sellRateLookup(rates: { unixDay: number, sell: number }[], fallbackRate: number): (day: number) => number {
  const sorted = [...rates].sort((left, right) => left.unixDay - right.unixDay)
  return (day) => {
    let low = 0, high = sorted.length - 1, found = -1
    while (low <= high) {
      const middle = (low + high) >> 1
      if (sorted[middle].unixDay <= day) { found = middle; low = middle + 1 } else { high = middle - 1 }
    }
    return found >= 0 ? Math.round(sorted[found].sell * 1000) : fallbackRate
  }
}

/** A row in local cents: dollars at the sell rate of the comprobante's day (Date + DocumentOffset − 128). */
export const rowAmountLocal = (row: ICashDocumentRow, sellRateOfDay: (day: number) => number) =>
  row.documentOffset > 0 ? Math.round(row.amount * sellRateOfDay(row.date + row.documentOffset - SAME_DAY_OFFSET) / 1000) : row.amount

// ── Known layer ──

const STREAM_OF_OPEN_KIND: Record<number, number> = {
  [KIND_SALE_OPEN]: 1, [KIND_PURCHASE_ORDER_OPEN]: 2, [KIND_EXPENSE_OPEN]: 3, [KIND_SCHEDULED_EXPENSE_OPEN]: 3, [KIND_ASSET_OPEN]: 4,
}
const LABEL_OF_OPEN_KIND: Record<number, string> = {
  [KIND_SALE_OPEN]: 'Unpaid sales|Ventas por cobrar', [KIND_PURCHASE_ORDER_OPEN]: 'Unpaid purchase orders|Órdenes de compra por pagar',
  [KIND_EXPENSE_OPEN]: 'Pending expenses|Gastos pendientes', [KIND_SCHEDULED_EXPENSE_OPEN]: 'Pending recurring bills|Gastos programados pendientes',
  [KIND_ASSET_OPEN]: 'Unpaid assets|Activos por pagar',
}

/** A warning record of the frame: a document day and site, with its amount in local cents. */
export interface IDocumentWarningRecord { date: number, siteID: number, amountLocal: number }

export interface IKnownDocuments {
  commitments: ICashFlowCommitment[]
  /** Records of the warnings by code. */
  warningRecords: Map<string, IDocumentWarningRecord[]>
}

const addWarningRecord = (warningRecords: Map<string, IDocumentWarningRecord[]>, code: string, record: IDocumentWarningRecord) => {
  const records = warningRecords.get(code) || []
  records.push(record)
  warningRecords.set(code, records)
}

/**
 * knownDocuments dates the open rows (§3): on their due date, or their document date without one; a
 * date already past moves to today. A receivable overdue more than STALE_RECEIVABLE_DAYS is left out.
 * Rows that only differ by tax group are one commitment. Dollar rows stay in dollars.
 */
export function knownDocuments(rows: ICashDocumentRow[], today: number, sellRateOfDay: (day: number) => number): IKnownDocuments {
  const commitmentByKey = new Map<string, ICashFlowCommitment>()
  const warningRecords = new Map<string, IDocumentWarningRecord[]>()
  for (const row of rows) {
    const stream = STREAM_OF_OPEN_KIND[row.kind]
    if (!stream || row.amount === 0) continue
    const dueDate = dueDateOf(row)
    const expectedDate = dueDate || row.date
    const record = { date: row.date, siteID: row.siteID, amountLocal: rowAmountLocal(row, sellRateOfDay) }
    const isSale = row.kind === KIND_SALE_OPEN
    // A stale receivable leaves its detracción out with it; the memo is not counted twice.
    if (isSale && today - expectedDate > STALE_RECEIVABLE_DAYS) {
      if (!isMemoTaxGroup(row.taxGroup)) addWarningRecord(warningRecords, 'receivable-stale', record)
      continue
    }
    // An open detracción is collected into the detracciones account: its share of the receivable moves
    // from the site to the detracciones lane. Every other memo moves no cash.
    if (isMemoTaxGroup(row.taxGroup)) {
      if (row.taxGroup === TAX_GROUP_DETRACTION && isSale) moveToDetractionsLane(commitmentByKey, row, expectedDate, dueDate, today)
      continue
    }
    if (isSale && !dueDate) addWarningRecord(warningRecords, 'sale-without-due-date', record)
    if (row.kind === KIND_PURCHASE_ORDER_OPEN && !dueDate) addWarningRecord(warningRecords, 'purchase-order-without-payment-date', record)
    if (!isSale && expectedDate < today) addWarningRecord(warningRecords, 'payable-overdue', record)

    knownCommitmentOf(commitmentByKey, row, row.siteID, expectedDate, dueDate, today).Amount += isSale ? row.amount : -row.amount
  }
  return { commitments: [...commitmentByKey.values()].filter((commitment) => commitment.Amount !== 0), warningRecords }
}

/** The commitment of an open row's document day, due offset and currency in a lane. */
function knownCommitmentOf(commitmentByKey: Map<string, ICashFlowCommitment>, row: ICashDocumentRow, siteID: number,
  expectedDate: number, dueDate: number, today: number): ICashFlowCommitment {
  const currencyType = row.documentOffset > 0 ? CURRENCY_USD : 1
  const key = `${row.kind}_${siteID}_${row.date}_${row.dueOffset}_${currencyType}`
  let commitment = commitmentByKey.get(key)
  if (!commitment) {
    commitment = {
      Stream: STREAM_OF_OPEN_KIND[row.kind], Layer: CASH_FLOW_LAYER_KNOWN, SiteID: siteID, SourceID: 0,
      Label: siteID === LANE_DETRACTIONS ? DETRACTIONS_LABEL : LABEL_OF_OPEN_KIND[row.kind],
      Date: Math.max(expectedDate, today), DueDate: dueDate, DocumentDate: row.date, Kind: row.kind, Amount: 0, CurrencyType: currencyType,
    }
    commitmentByKey.set(key, commitment)
  }
  return commitment
}

const DETRACTIONS_LABEL = 'Detracciones to deposit|Detracciones por depositar'

/** An open detracción: collected into the detracciones lane instead of its site's. */
function moveToDetractionsLane(commitmentByKey: Map<string, ICashFlowCommitment>, row: ICashDocumentRow,
  expectedDate: number, dueDate: number, today: number) {
  knownCommitmentOf(commitmentByKey, row, row.siteID, expectedDate, dueDate, today).Amount -= row.amount
  knownCommitmentOf(commitmentByKey, row, LANE_DETRACTIONS, expectedDate, dueDate, today).Amount += row.amount
}

/** The open payables older than the window, owed today (payables-before-window). */
export function oldPayableCommitments(oldPayables: ICashFlowOldPayables, today: number): ICashFlowCommitment[] {
  const label = 'Payables older than a year|Cuentas por pagar de más de un año'
  return [[oldPayables.Local, 1], [oldPayables.Dollars, CURRENCY_USD]]
    .filter(([amount]) => amount > 0)
    .map(([amount, currencyType]) => ({
      Stream: 3, Layer: CASH_FLOW_LAYER_KNOWN, SiteID: 0, SourceID: 0, Label: label, Date: today, DueDate: 0,
      Amount: -amount, CurrencyType: currencyType,
    }))
}

// ── Taxes ──

/** What a tax period's taxes are computed from, in local cents. */
export interface ITaxPeriodBase {
  period: number
  dueDate: number
  salesTax: number
  salesNet: number
  purchaseTaxCredit: number
  /** The retenciones the clients withheld from the period's sales: a credit against its IGV. */
  retentionCredit: number
  paidIgv: number
  paidIncome: number
}

/**
 * taxPeriodBases sums each period of the outlook from the rows dated in it: the IGV and the net of the
 * sales, and the crédito fiscal of the spending (expenses, purchase orders and assets), each derived
 * from its tax group. Dollar rows at the SUNAT sell rate of the comprobante's day.
 */
export function taxPeriodBases(rows: ICashDocumentRow[], outlook: ICashFlowTaxOutlook, sellRateOfDay: (day: number) => number): ITaxPeriodBase[] {
  const bases = (outlook.Periods || []).map((taxPeriod): ITaxPeriodBase => ({
    period: taxPeriod.Period, dueDate: taxPeriod.DueDate, salesTax: 0, salesNet: 0, purchaseTaxCredit: 0, retentionCredit: 0,
    paidIgv: taxPeriod.PaidIgvLocal, paidIncome: taxPeriod.PaidIncomeLocal,
  }))
  const baseByPeriod = new Map(bases.map((base) => [base.period, base]))
  for (const row of rows) {
    const base = baseByPeriod.get(periodOfDay(row.date))
    if (!base) continue
    const amountLocal = rowAmountLocal(row, sellRateOfDay)
    // A free line was charged nothing, so it adds no net sale, but its IGV is owed. Its row does not
    // keep the product's rate: taken as 18 % (a free 10 % line is a restaurant's rare case).
    if (row.taxGroup === TAX_GROUP_FREE) {
      base.salesTax += igvOfGross(TAX_GROUP_TAXED_18, amountLocal)
      continue
    }
    if (row.taxGroup === TAX_GROUP_RETENTION) base.retentionCredit += amountLocal
    // The other memos (a detracción) are part of their sale's total, already counted.
    if (isMemoTaxGroup(row.taxGroup)) continue
    const igv = igvOfGross(row.taxGroup, amountLocal)
    if (isSaleKind(row.kind)) {
      base.salesTax += igv
      base.salesNet += amountLocal - igv
    } else {
      base.purchaseTaxCredit += igv
    }
  }
  return bases
}

export interface ITaxRules { ChargesIgv: boolean, IncomeTaxRate: number, NrusQuota: number }

export interface ITaxPeriodDue { period: number, dueDate: number, igv: number, incomeTax: number, nrus: number }

/**
 * closedPeriodTaxes computes what each closed period owes (§7.3), bases in period order: IGV is the
 * sales IGV less the crédito fiscal, the retenciones and the saldo a favor carried from earlier periods (a negative
 * result carries forward); renta is the rate over net sales; a Nuevo RUS company pays its fixed cuota
 * instead of both. What the period's due month already paid is subtracted. creditCarried is the saldo
 * a favor left after the last period, which the forecast periods start from.
 */
export function closedPeriodTaxes(bases: ITaxPeriodBase[], rules: ITaxRules): { dues: ITaxPeriodDue[], creditCarried: number } {
  let creditCarried = 0
  const dues = bases.map((base): ITaxPeriodDue => {
    const due = { period: base.period, dueDate: base.dueDate, igv: 0, incomeTax: 0, nrus: 0 }
    if (rules.NrusQuota > 0) {
      due.nrus = Math.max(0, rules.NrusQuota - base.paidIgv - base.paidIncome)
      return due
    }
    if (rules.ChargesIgv) {
      const igv = base.salesTax - base.purchaseTaxCredit - base.retentionCredit - creditCarried
      creditCarried = Math.max(0, -igv)
      due.igv = Math.max(0, igv - base.paidIgv)
    }
    due.incomeTax = Math.max(0, base.salesNet * rules.IncomeTaxRate / 10000 - base.paidIncome)
    return due
  })
  return { dues, creditCarried }
}

/** The IGV share of the gross sales over the bases; 0 without sales. */
export function salesTaxRatioOf(bases: ITaxPeriodBase[]): number {
  const salesTax = bases.reduce((total, base) => total + base.salesTax, 0)
  const salesGross = bases.reduce((total, base) => total + base.salesTax + base.salesNet, 0)
  return salesGross > 0 ? salesTax / salesGross : 0
}

/**
 * The closed periods' dues as Known commitments in the company lane, on their SUNAT date or today
 * when it passed (tax-period-unpaid).
 */
export function closedTaxCommitments(dues: ITaxPeriodDue[], today: number): { commitments: ICashFlowCommitment[], unpaid: { ID: number, Label: string }[], unpaidAmount: number } {
  const commitments: ICashFlowCommitment[] = []
  const unpaid: { ID: number, Label: string }[] = []
  let unpaidAmount = 0
  for (const due of dues) {
    for (const [stream, name, amount] of [[10, 'IGV', due.igv], [11, 'Renta', due.incomeTax], [12, 'Nuevo RUS', due.nrus]] as const) {
      if (amount <= 0) continue
      const label = `${name} ${due.period}`
      if (due.dueDate < today) {
        unpaid.push({ ID: due.period, Label: label })
        unpaidAmount += amount
      }
      commitments.push({
        Stream: stream, Layer: CASH_FLOW_LAYER_KNOWN, SiteID: 0, SourceID: due.period, Label: label,
        Date: Math.max(due.dueDate, today), DueDate: due.dueDate, Amount: -Math.round(amount), CurrencyType: 1,
      })
    }
  }
  return { commitments, unpaid, unpaidAmount }
}

/** Splits the outlook's bases at the open period: the closed ones, then the open and following ones. */
export function splitTaxPeriods(bases: ITaxPeriodBase[], today: number) {
  const openPeriod = periodOfDay(today)
  return { closed: bases.filter((base) => base.period < openPeriod), open: bases.filter((base) => base.period >= openPeriod) }
}

// ── Warnings ──

/**
 * The scan warnings over the closed and open tax periods (the last months): sales and expenses without
 * a site (they land in the company lane) and expenses without a comprobante (no crédito fiscal).
 */
export function scanWarningRecords(rows: ICashDocumentRow[], firstPeriod: number, sellRateOfDay: (day: number) => number) {
  const warningRecords = new Map<string, IDocumentWarningRecord[]>()
  for (const row of rows) {
    if (periodOfDay(row.date) < firstPeriod || isMemoTaxGroup(row.taxGroup)) continue
    const record = { date: row.date, siteID: row.siteID, amountLocal: rowAmountLocal(row, sellRateOfDay) }
    if (isSaleKind(row.kind) && !row.siteID) addWarningRecord(warningRecords, 'sale-without-site', record)
    if (isExpenseKind(row.kind) && !row.siteID) addWarningRecord(warningRecords, 'expense-without-site', record)
    if (isExpenseKind(row.kind) && row.taxGroup === TAX_GROUP_NO_DOCUMENT) addWarningRecord(warningRecords, 'purchase-without-document', record)
  }
  return warningRecords
}

/**
 * Warnings from frame records: one record per document day and site (the frame keeps no documents),
 * the amount at stake summed.
 */
export function documentWarnings(warningRecords: Map<string, IDocumentWarningRecord[]>, recordLabel: (date: number, siteID: number) => string): ICashFlowServerWarning[] {
  return [...warningRecords].map(([code, records]) => {
    const amountByDaySite = new Map<string, IDocumentWarningRecord>()
    for (const record of records) {
      const key = `${record.date}_${record.siteID}`
      const merged = amountByDaySite.get(key) || { ...record, amountLocal: 0 }
      merged.amountLocal += record.amountLocal
      amountByDaySite.set(key, merged)
    }
    const daySites = [...amountByDaySite.values()].sort((left, right) => left.date - right.date || left.siteID - right.siteID)
    return {
      Code: code, AmountAtStakePEN: daySites.reduce((total, record) => total + Math.abs(record.amountLocal), 0),
      Records: daySites.slice(0, 50).map((record) => ({ ID: record.date, Label: recordLabel(record.date, record.siteID) })),
      RecordCount: daySites.length,
    }
  })
}

// The sale-client-debt frame: Keys [ClientBucket], Rows [ClientID], Sums [DebtLocal].
const DEBT_ROW_CLIENT = 0
const DEBT_SUM_AMOUNT = 0

/**
 * client-credit-days-missing: the clients that owe money on cash terms (no credit days), so their
 * sales fall due on the sale date. The debts come from the sale-client-debt frame, the clients from
 * their delta cache.
 */
export function clientCreditDaysWarning(debtFiles: FrameFileRows[], clientByID: Map<number, { Name: string, CreditDays: number }>): ICashFlowServerWarning[] {
  const records: { ID: number, Label: string }[] = []
  let amountAtStake = 0
  for (const file of debtFiles) {
    const clientIDs = file.rows[DEBT_ROW_CLIENT] as number[]
    clientIDs.forEach((clientID, rowIndex) => {
      const debt = (file.sums[DEBT_SUM_AMOUNT] as number[])[rowIndex]
      const client = clientByID.get(clientID)
      if (debt <= 0 || !client || client.CreditDays > 0) return
      records.push({ ID: clientID, Label: client.Name })
      amountAtStake += debt
    })
  }
  return records.length ? [{ Code: 'client-credit-days-missing', AmountAtStakePEN: amountAtStake, Records: records.slice(0, 50), RecordCount: records.length }] : []
}

// ── Drill-down ──

/** A drill-down item that stands for frame rows: its Kind, document day and due date. */
export interface IDocumentItem { kind?: number, documentDate?: number, dueDate: number }

/** The document dates to ask GET.cash-documents-day for, per open Kind, from a cell's items. */
export function drilldownDatesByKind(items: IDocumentItem[]): Map<number, number[]> {
  const datesByKind = new Map<number, Set<number>>()
  for (const item of items) {
    if (!item.kind || !item.documentDate) continue
    const dates = datesByKind.get(item.kind) || new Set<number>()
    dates.add(item.documentDate)
    datesByKind.set(item.kind, dates)
  }
  return new Map([...datesByKind].map(([kind, dates]) => [kind, [...dates].sort((left, right) => left - right)]))
}

/**
 * The documents behind a cell's items of one Kind: those whose document day and due offset match an
 * item's, since the backend answers every open document of those days, wherever its cash lands.
 */
export function documentsOfItems(items: IDocumentItem[], kind: number, documents: ICashDocumentDetail[]): ICashDocumentDetail[] {
  const itemKeys = new Set(items.filter((item) => item.kind === kind && item.documentDate)
    .map((item) => `${item.documentDate}_${dueOffsetOf(item.documentDate as number, item.dueDate)}`))
  return documents.filter((document) => itemKeys.has(`${document.Date}_${dueOffsetOf(document.Date, document.DueDate)}`))
}

/** The first period the scan warnings look at: the oldest closed tax period of the outlook. */
export const firstScanPeriod = (outlook: ICashFlowTaxOutlook, today: number) => outlook.Periods?.[0]?.Period ?? periodOfDay(today)
