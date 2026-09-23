import { extractError, GET } from '$libs/ui-runtime.svelte'
import { Loading, Notify } from '$libs/helpers'
import { tr } from '$core/store.svelte'
import {
  getClientProviderSnapshots, type IClientProviderSnapshot,
} from '$services/crm/client-provider.svelte'
import type { ISalesBookResponse } from './books'

// A book is read for a period, not synced.
//
// There is no delta cache on the period read itself: a filed period does not change, and the
// rows are a slice of the month rather than a growing list. The identities are the opposite —
// a snapshot is immutable and the same few hundred come back every period — so those go
// through the by-ids cache and are downloaded once, ever.
export class SalesBookService {
  response: ISalesBookResponse | null = $state(null)
  identityBySnapshotID: Map<number, IClientProviderSnapshot> = $state(new Map())
  isLoading = $state(false)
  // What was last asked for, so the page can tell "nothing yet" from "this period is empty".
  loadedPeriod = $state("")

  // Fetches a whole period (YYYYMM) or, when unixDay is given, a single day inside one.
  async fetch(period: string, unixDay: number = 0): Promise<void> {
    if (!period && !unixDay) {
      this.reset()
      return
    }

    const route = period ? `sales-book?period=${period}` : `sales-book?date=${unixDay}`
    this.isLoading = true
    Loading.standard(tr("Reading the book...|Leyendo el libro..."))
    try {
      const response = await GET({ route }) as ISalesBookResponse
      this.response = response
      this.loadedPeriod = response?.Period || period
      this.identityBySnapshotID = await this.resolveIdentities(response)
    } catch (error) {
      this.reset()
      // The rejection is the raw response, so String() on it says "[object Object]".
      Notify.failure(
        extractError(error) || tr("The book could not be read.|No se pudo leer el libro."),
      )
    } finally {
      this.isLoading = false
      Loading.remove()
    }
  }

  // The second-grade read: the documents carry a snapshot id and the identities come from the
  // by-ids cache, so a period whose buyers are already known costs no request at all.
  private async resolveIdentities(
    response: ISalesBookResponse,
  ): Promise<Map<number, IClientProviderSnapshot>> {
    const snapshotIDs = (response?.Documents || [])
      .map(document => document.ClientSnapshotID)
      .filter(snapshotID => snapshotID > 0)

    if (snapshotIDs.length === 0) return new Map()
    return await getClientProviderSnapshots(snapshotIDs)
  }

  private reset(): void {
    this.response = null
    this.identityBySnapshotID = new Map()
    this.loadedPeriod = ""
  }
}
