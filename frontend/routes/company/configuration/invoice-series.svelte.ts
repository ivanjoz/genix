import { GetHandler, POST } from '$libs/ui-runtime.svelte';
import type { IInvoiceSeries } from './invoice-series';

// Only what the series form shows. Declared here rather than imported from the
// branches route: routes are leaves, and this needs two fields of a site, not the
// warehouses service that also carries warehouses and their layouts.
export interface IInvoiceSeriesSite {
  ID: number
  Name: string
}

export class InvoiceSeriesSitesService extends GetHandler {
  route = "locations-warehouses"
  useCache = { min: 5, ver: 5 }

  Sites: IInvoiceSeriesSite[] = $state([])

  handler(result: { Sedes?: IInvoiceSeriesSite[] }): void {
    this.Sites = result.Sedes || []
  }

  constructor() {
    super()
    this.fetch()
  }
}

// The series save through their own endpoint even though they live on the company
// record, so the company form's Save only writes what that form shows. It sends one
// series, not the set: a set-shaped body lets a stale tab delete series it never saw.
//
// The reply carries the whole set back, which is what the caller writes into the
// company it already holds — cheaper and less disruptive than refetching the form.
export async function postInvoiceSeries(series: IInvoiceSeries): Promise<IInvoiceSeries[]> {
  // The company route is refreshed because its cached copy carries the series: a
  // reload inside the cache window would otherwise show the series missing, which
  // reads as a lost save.
  const response = await POST({
    data: series,
    route: "invoice-series",
    refreshRoutes: ["company-parametros"],
  })
  return (response?.InvoiceSeries || []) as IInvoiceSeries[]
}
