import { hideLoading, notifyFailure, showLoading } from '@genix/ui/notify';
import { extractError, GET } from '#libs/ui-runtime.svelte.ts'
import { tr } from '#core/store.svelte.ts'
import {
  getClientProviderSnapshots, type IClientProviderSnapshot,
} from '#services/crm/client-provider.svelte.ts'
import type { ISalesBookResponse } from './books'
import type { IPurchasesBookResponse } from './books.purchases'

// A book is read for a period, not synced.
//
// There is no delta cache on the period read itself: a filed period does not change, and the
// rows are a slice of the month rather than a growing list. The identities are the opposite —
// a snapshot is immutable and the same few hundred come back every period — so those go
// through the by-ids cache and are downloaded once, ever.
export class BookService<TResponse extends { Period: string }> {
  response: TResponse | null = $state(null)
  identityBySnapshotID: Map<number, IClientProviderSnapshot> = $state(new Map())
  isLoading = $state(false)
  // What was last asked for, so the page can tell "nothing yet" from "this period is empty".
  loadedPeriod = $state("")

  // route is the book's endpoint; snapshotIDsOf lists the counterpart identities a response pins.
  constructor(
    private route: string,
    private snapshotIDsOf: (response: TResponse) => number[],
  ) {}

  // Fetches a whole period (YYYYMM) or, when unixDay is given, a single day inside one.
  async fetch(period: string, unixDay: number = 0): Promise<void> {
    if (!period && !unixDay) {
      this.reset()
      return
    }

    const route = period ? `${this.route}?period=${period}` : `${this.route}?date=${unixDay}`
    this.isLoading = true
    showLoading(tr("Reading the book...|Leyendo el libro..."))
    try {
      const response = await GET({ route }) as TResponse
      this.response = response
      this.loadedPeriod = response?.Period || period
      // The second-grade read: a period whose counterparts are already known costs no request.
      const snapshotIDs = response ? this.snapshotIDsOf(response).filter(snapshotID => snapshotID > 0) : []
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

export const makeSalesBookService = () => new BookService<ISalesBookResponse>(
  "sales-book",
  response => (response.Documents || []).map(document => document.ClientSnapshotID),
)

export const makePurchasesBookService = () => new BookService<IPurchasesBookResponse>(
  "purchases-book",
  response => (response.Documents || []).map(document => document.ProviderSnapshotID),
)
