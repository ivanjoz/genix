import { GetHandler } from '$libs/ui-runtime.svelte';

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
