import { GetHandler } from '#libs/ui-runtime.svelte.ts'

/** One client's sales of one ISO week, summed per weekday (index 0 = Monday … 6 = Sunday), in
 *  céntimos of soles: what the client bought and how much of it is paid. A payment counts on the
 *  day of the sale it pays, so Purchases − Payments is what is still owed of that day's sales.
 *  Backend: sales/types/client_week_sales.go. */
export interface IClientWeekSales {
  ID: number
  /** ISO week as YYWW (2646). */
  Week: number
  ClientID: number
  Purchases: number[]
  Payments: number[]
  upd: number
  ver?: number
}

/** The client week rows of the last 26 weeks, kept in sync by the delta cache. */
export class ClientWeekSalesService extends GetHandler<IClientWeekSales> {
  route = 'client-week-sales'
  useCache = { min: 1, ver: 1 }

  records: IClientWeekSales[] = $state([])

  constructor(init = false) {
    super()
    if (init) {
      this.fetch()
    }
  }

  handler(result: IClientWeekSales[]): void {
    // The delta read ends with a metadata record (negative ID): it is not a row.
    this.records = (result || []).filter((clientWeekRow) => clientWeekRow.ID > 0)
  }
}
