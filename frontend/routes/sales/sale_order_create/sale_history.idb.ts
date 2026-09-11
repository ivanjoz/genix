// Local persistence for the till's own sale history. Backed by Dexie/IndexedDB so the
// operator can reprint the ticket of a sale made minutes ago without querying the server.
//
// There is no server-side counterpart and none is wanted: this is the log of what *this
// browser* sold, scoped per company+env exactly like core/notifications.idb.ts. A sale made
// at another till is not here, and that is the point — reprinting is a till-local act.
//
// A row stores references and nothing that can be resolved from them. The series code, the
// document type, the client, the warehouse, the caja and the product names all come back from
// the services the page already loads; the sale id alone carries the series and the
// correlativo. What IS stored is what nothing can recover later: the quantities, the divisor
// and the two prices the sale was actually made at.

import Dexie from 'dexie'
import { Env } from '$core/env'
import type { Quantity } from '$core/quantity'

const LOG_PREFIX = '[sale-history:idb]'
const SALE_HISTORY_DB_PREFIX = 'sale_history'
const SALE_HISTORY_DB_VERSION = 1

// Hard cap on persisted rows: the newest MAX_SALE_HISTORY_ROWS survive, the rest are dropped
// on load. A till does hundreds of sales a day and only the recent ones are ever reprinted.
export const MAX_SALE_HISTORY_ROWS = 200

export interface SaleHistoryLine {
  productID: number
  presentationID: number
  quantity: Quantity
  // The divisor and both prices are the terms the sale was closed on. The catalog is free to
  // reprice or reconfigure the product afterwards, which is why these three are copied and
  // the product's name is not.
  subDivisor: number
  unitPrice: number
  subUnitPrice: number
  // A serial is data the sale carries, not a lookup: "SN-0007" or "SN-0007 x2".
  serialNumbers: string[]
}

export interface SaleHistoryRow {
  // Primary key. The sale id is already unique and is what the ticket is numbered by — its
  // last two digits are the invoicing series and the rest is the correlativo
  // (backend/sales/types/sale_order_id.go) — so a separate local id would be a second name
  // for the same row.
  saleID: number
  // Milliseconds on the browser clock. This is a local log, not a persisted business date —
  // the sale's own date lives on the server record.
  savedAt: number
  // 0 = the till named no client ("VARIOS"). A client typed at the till was created by the
  // backend, which returns its id on the sale.
  clientID: number
  warehouseID: number
  // The user who rang the sale up, resolved against the session that reprints it.
  cashierID: number
  cashBankID: number
  // SaleOrder.Status exactly as the backend persisted it: 0 anulado, 1 generado, 2 pagado,
  // 3 entregado, 4 pagado + entregado. Kept raw so both flags derive from one field.
  status: number
  // Cents.
  totalAmount: number
  taxAmount: number
  lines: SaleHistoryLine[]
}

class SaleHistoryDatabase extends Dexie {
  sales!: Dexie.Table<SaleHistoryRow, number>

  constructor(databaseName: string) {
    super(databaseName)
    this.version(SALE_HISTORY_DB_VERSION).stores({
      // saleID is the primary key, savedAt the index the history view reads in order.
      sales: 'saleID, savedAt',
    })
  }
}

const databasesByName = new Map<string, SaleHistoryDatabase>()

const getDatabase = (): SaleHistoryDatabase => {
  const name = `${Env.getCompanyID() || 0}_${SALE_HISTORY_DB_PREFIX}_${Env.enviroment || '000000'}`
  const existing = databasesByName.get(name)
  if (existing) return existing
  const created = new SaleHistoryDatabase(name)
  databasesByName.set(name, created)
  return created
}

// Newest first, which is the order the history view renders.
export const loadSaleHistory = async (): Promise<SaleHistoryRow[]> => {
  try {
    const rows = await getDatabase().sales.orderBy('savedAt').reverse().toArray()
    if (rows.length <= MAX_SALE_HISTORY_ROWS) return rows

    // Pruned on read rather than on write: one delete per load instead of one per sale.
    const expiredSaleIDs = rows.slice(MAX_SALE_HISTORY_ROWS).map((row) => row.saleID)
    getDatabase().sales.bulkDelete(expiredSaleIDs)
      .catch((error) => console.warn(`${LOG_PREFIX} prune failed`, error))
    return rows.slice(0, MAX_SALE_HISTORY_ROWS)
  } catch (error) {
    console.warn(`${LOG_PREFIX} load failed`, error)
    return []
  }
}

export const appendSaleHistory = async (row: SaleHistoryRow): Promise<void> => {
  try {
    await getDatabase().sales.put(row)
  } catch (error) {
    console.warn(`${LOG_PREFIX} append failed`, error)
  }
}

export const clearSaleHistory = async (): Promise<void> => {
  try {
    await getDatabase().sales.clear()
  } catch (error) {
    console.warn(`${LOG_PREFIX} clear failed`, error)
  }
}
