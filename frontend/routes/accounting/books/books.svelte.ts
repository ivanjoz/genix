import { hideLoading, notifyFailure, showLoading } from '@genix/ui/notify';
import { getRecordsByID } from '@genix/ui/cache'
import { extractError, GET } from '#libs/ui-runtime.svelte.ts'
import { tr } from '#core/store.svelte.ts'
import {
  getClientProviderSnapshots, type IClientProviderSnapshot,
} from '#services/crm/client-provider.svelte.ts'
import {
  periodOfUnixDay, salesBookOfPeriod, type ISalesBookDocument, type ISalesBookPeriod, type ISalesBookResponse,
} from './books'
import { purchasesBookOfPeriod, type IPurchasesBookPeriod, type IPurchasesBookResponse } from './books.purchases'

// A book is synced a period at a time: each response key is one of the period's delta logs, so
// reopening a month only travels what changed. `min: 0` asks the server every time — the TTL would
// otherwise skip the network and show a stale month. A single day is cut from its period here.
// The identities are the opposite — a snapshot is immutable and the same few hundred come back
// every period — so those go through the by-ids cache and are downloaded once, ever.
export class BookService<TPeriodRecords, TResponse extends { Period: string }> {
  response: TResponse | null = $state(null)
  identityBySnapshotID: Map<number, IClientProviderSnapshot> = $state(new Map())
  isLoading = $state(false)
  // What was last asked for, so the page can tell "nothing yet" from "this period is empty".
  loadedPeriod = $state("")

  // route is the book's endpoint; resolveBook cuts the synced period down to the book asked for;
  // snapshotIDsOf lists the counterpart identities a response pins.
  constructor(
    private route: string,
    private keysIDs: Record<string, string>,
    private resolveBook: (periodRecords: TPeriodRecords, period: string, unixDay: number) => Promise<TResponse>,
    private snapshotIDsOf: (response: TResponse) => number[],
  ) {}

  // Reads a whole period (YYYYMM) or, when no period is given, the single day unixDay.
  async fetch(period: string, unixDay: number = 0): Promise<void> {
    if (!period && !unixDay) {
      this.reset()
      return
    }

    const bookPeriod = period || periodOfUnixDay(unixDay)
    this.isLoading = true
    showLoading(tr("Reading the book...|Leyendo el libro..."))
    try {
      const periodRecords = await GET({
        route: `${this.route}?period=${bookPeriod}`,
        useCache: { min: 0, ver: 1 },
        keysIDs: this.keysIDs,
      }) as TPeriodRecords
      const response = await this.resolveBook(periodRecords, bookPeriod, period ? 0 : unixDay)
      this.response = response
      this.loadedPeriod = bookPeriod
      // The second-grade read: a period whose counterparts are already known costs no request.
      const snapshotIDs = this.snapshotIDsOf(response).filter(snapshotID => snapshotID > 0)
      this.identityBySnapshotID = snapshotIDs.length > 0
        ? await getClientProviderSnapshots(snapshotIDs)
        : new Map()
    } catch (error) {
      this.reset()
      // The rejection is the raw response, so String() on it says "[object Object]".
      notifyFailure(
        extractError(error) || tr("The book could not be read.|No se pudo leer el libro."),
      )
    } finally {
      this.isLoading = false
      hideLoading()
    }
  }

  reset(): void {
    this.response = null
    this.identityBySnapshotID = new Map()
    this.loadedPeriod = ""
  }
}

// The comprobantes a note corrects in an earlier period come through the by-IDs cache.
const resolveSalesBook = async (periodRecords: ISalesBookPeriod, period: string, unixDay: number) => {
  const { response, olderAffectedIDs } = salesBookOfPeriod(periodRecords, period, unixDay)
  if (olderAffectedIDs.length > 0) {
    const olderAffected = await getRecordsByID<ISalesBookDocument & { ss: number, upd: number }>("invoice-document-ids", olderAffectedIDs)
    response.AffectedDocuments.push(...olderAffected.values())
  }
  return response
}

export const makeSalesBookService = () => new BookService<ISalesBookPeriod, ISalesBookResponse>(
  "sales-book",
  {},
  resolveSalesBook,
  response => (response.Documents || []).map(document => document.ClientSnapshotID),
)

export const makePurchasesBookService = () => new BookService<IPurchasesBookPeriod, IPurchasesBookResponse>(
  "purchases-book",
  // A purchase document is keyed by the order's, the expense's or the asset's own ID.
  { PurchaseOrders: "SourceID", Expenses: "SourceID", Assets: "SourceID" },
  async (periodRecords, period, unixDay) => purchasesBookOfPeriod(periodRecords, period, unixDay),
  response => (response.Documents || []).map(document => document.ProviderSnapshotID),
)
