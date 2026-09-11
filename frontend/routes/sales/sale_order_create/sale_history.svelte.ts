// Reactive state over the till's local sale history: the rows the history view renders and
// the two writes that feed it. The rules that shape a row live in sale_history.ts and the
// persistence in sale_history.idb.ts — this is only the bridge between them and the page.

import { appendSaleHistory, clearSaleHistory, loadSaleHistory, type SaleHistoryRow } from './sale_history.idb'

export class SaleHistoryState {
  rows = $state([] as SaleHistoryRow[])
  // Distinguishes "no sales yet" from "not read from IndexedDB yet", which the empty
  // message has to tell apart or it flashes on every page load.
  isLoaded = $state(false)

  constructor() {
    this.load()
  }

  async load() {
    this.rows = await loadSaleHistory()
    this.isLoaded = true
  }

  // The row goes on screen before IndexedDB confirms it: a till that just made a sale must
  // see it immediately, and a failed local write is a lost reprint, not a lost sale.
  async register(row: SaleHistoryRow) {
    this.rows = [row, ...this.rows]
    await appendSaleHistory(row)
  }

  async clear() {
    await clearSaleHistory()
    this.rows = []
  }
}
