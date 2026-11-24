import type { FrameFileRows } from '@genix/ui/cache'
import { GET, GETFrameFiles, GetHandler } from '#libs/ui-runtime.svelte.ts'

// Mirrors backend/finance/types/cash_flow.go and cash_flow_entry.go: the cash flow forecast's Known
// and Scheduled layers (GET.cash-flow-commitments, computed on every read) and its planned entries
// (cash_flow_entry, delta-synced). Also the history frames the Forecast layer learns from.

export interface ICashFlowCommitment {
  /** CashFlowStream* (cash-flow.ts). */
  Stream: number
  /** CASH_FLOW_LAYER_* (cash-flow.ts): 1 Known, 2 Scheduled; the browser adds 3 Forecast. */
  Layer: number
  /** 0 = the company lane. */
  SiteID: number
  SourceID: number
  Label: string
  /** UnixDay the cash is expected to move. */
  Date: number
  /** The record's own due date, 0 when it has none. */
  DueDate: number
  /** Browser only, on the open documents of the cash-documents frame: their document day and Kind. */
  DocumentDate?: number
  Kind?: number
  /** Signed cents of CurrencyType: + in, − out. */
  Amount: number
  CurrencyType: number
}

export interface ICashFlowWarningRecord {
  ID: number
  Label: string
}

export interface ICashFlowServerWarning {
  Code: string
  AmountAtStakePEN: number
  /** The first 50. */
  Records: ICashFlowWarningRecord[] | null
  RecordCount: number
}

export interface ICashFlowTaxPeriod {
  /** YYMM. */
  Period: number
  DueDate: number
  /** What the 401x expenses of its due month already paid, local cents. */
  PaidIgvLocal: number
  PaidIncomeLocal: number
}

export interface ICashFlowTaxOutlook {
  Rules: { ChargesIgv: boolean, IncomeTaxRate: number, NrusQuota: number }
  /** The last closed periods, then the open one and those falling due in the window. */
  Periods: ICashFlowTaxPeriod[] | null
}

/** Open payables dated before the frame window, cents per currency. */
export interface ICashFlowOldPayables {
  Local: number
  Dollars: number
}

export interface ICashFlowCommitmentsResponse {
  Commitments: ICashFlowCommitment[] | null
  Warnings: ICashFlowServerWarning[] | null
  TaxOutlook: ICashFlowTaxOutlook
  OldPayables: ICashFlowOldPayables
}

/** One open document behind a cash flow cell (GET.cash-documents-day). */
export interface ICashDocumentDetail {
  ID: number
  Label: string
  Date: number
  /** 0 without one. */
  DueDate: number
  /** What it still owes, cents of CurrencyType. */
  Amount: number
  CurrencyType: number
}

export interface ICashFlowEntry {
  ID: number
  /** UnixDay. */
  Date: number
  /** 0 = the company lane. */
  SiteID: number
  /** Signed cents: + in, − out; the sign follows the Kind. */
  Amount: number
  CurrencyType: number
  /** CASH_FLOW_ENTRY_* (cash-flow.ts). */
  Kind: number
  Name: string
  ss: number
  upd: number
}

/** Not delta-cached: one computed answer, read when the page opens. */
export const getCashFlowCommitments = (horizonDays: number): Promise<ICashFlowCommitmentsResponse> =>
  GET({ route: `cash-flow-commitments?horizon=${horizonDays}` })

/** The open documents of one Kind dated on the given days; without a site, every site's. */
export const getCashDocumentsDay = (kind: number, dates: number[], siteID?: number): Promise<ICashDocumentDetail[] | null> =>
  GET({ route: `cash-documents-day?kind=${kind}&dates=${dates.join(',')}${siteID === undefined ? '' : `&site=${siteID}`}` })

// sale-client-debt files are keyed by ClientID >> 12 (backend SaleClientDebtBucketBits): every bucket.
const CLIENT_DEBT_LAST_BUCKET = 2147483647 >> 12

/** What each client owes on its open sales (the sale-client-debt frame); only changed buckets travel. */
export const fetchClientDebtFiles = async (): Promise<FrameFileRows[]> =>
  (await GETFrameFiles('sale-client-debt', 'sale-client-debt', { fromKey: 0, toKey: CLIENT_DEBT_LAST_BUCKET })).files

/** Active planned entries (`cash-flow-entries`). */
export class CashFlowEntriesService extends GetHandler<ICashFlowEntry> {
  route = 'cash-flow-entries'
  useCache = { min: 5, ver: 1 }
  inferRemoveFromStatus = true

  constructor(init = false) {
    super()
    if (init) this.fetch()
  }

  handler(result: ICashFlowEntry[]): void {
    this.records = []
    this.recordsMap = new Map()
    this.addSavedRecords(...(result || []))
  }
}

// The history frames are read at most 400 days per request (backend cashFlowFrameMaxDays, and
// GET.sale-summary's own bound).
const HISTORY_FRAME_MAX_DAYS = 400

/**
 * fetchHistoryFrameFiles reads a frame's day files over [fromDay, toDay], 400 days per request. The
 * files stay in the browser's frame cache, so a later read only receives the days that changed.
 * Frames: cash-documents (every document per day, site, Kind and tax group), cash-delay (payments
 * per day, register and delay).
 */
export const fetchHistoryFrameFiles = async (route: string, frame: string, fromDay: number, toDay: number): Promise<FrameFileRows[]> => {
  const files: FrameFileRows[] = []
  for (let chunkFromDay = fromDay; chunkFromDay <= toDay; chunkFromDay += HISTORY_FRAME_MAX_DAYS) {
    const chunk = await GETFrameFiles(route, frame, { fromKey: chunkFromDay, toKey: Math.min(chunkFromDay + HISTORY_FRAME_MAX_DAYS - 1, toDay) })
    files.push(...chunk.files)
  }
  return files
}

/** Saves (or, with ss = 0, removes) one entry and brings back the delta of `cash-flow-entries`. */
export const postCashFlowEntry = (service: CashFlowEntriesService, entry: ICashFlowEntry): Promise<ICashFlowEntry> =>
  service.writeAndSync('cash-flow-entries', entry)
