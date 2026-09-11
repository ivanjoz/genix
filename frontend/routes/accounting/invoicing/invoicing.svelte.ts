import { Env } from '$core/env'
import { GetHandler, POST, buildHeaders } from '$libs/ui-runtime.svelte'
import { Notify } from '$libs/helpers'
import { tr } from '$core/store.svelte'
import type { IInvoiceDocument } from './invoicing'

export type { IInvoiceDocument }

export interface IInvoicesResult {
  Invoices: IInvoiceDocument[]
}

// Documents sync incrementally on their write version, and a document is written
// several times over its life — reserved, signed, answered — so this list moves on
// its own while the page is open.
export class InvoicesService extends GetHandler<IInvoiceDocument> {
  route = "invoices"
  useCache = { min: 1, ver: 1 }

  records: IInvoiceDocument[] = $state([])

  handler(result: IInvoicesResult): void {
    // No status filter: a voided document is exactly what somebody comes here to
    // check, and the correlativo it spent is still part of the series.
    const documents = [...(result?.Invoices || [])]
    documents.sort((a, b) => b.ID - a.ID)
    this.records = documents
  }

  constructor(init: boolean = false) {
    super()
    if (init) this.fetch()
  }
}

// Sends a document now instead of waiting for the sweep, or again after a failure.
export const postInvoiceSend = (documentID: number) => {
  return POST({
    route: `invoice-retry?id=${documentID}`,
    data: {},
    refreshRoutes: ["invoices"],
  })
}

export type InvoiceArtifact = "xml" | "cdr"

// Downloads the signed XML or the CDR SUNAT returned.
//
// Fetched rather than linked: the endpoint is authenticated, so a plain anchor
// would arrive without the session headers and answer 401.
export async function downloadInvoiceArtifact(
  documentID: number, artifact: InvoiceArtifact, fileName: string,
): Promise<void> {

  const route = artifact === "cdr"
    ? `invoice-xml?id=${documentID}&tipo=cdr`
    : `invoice-xml?id=${documentID}`

  const response = await fetch(Env.makeRoute(route), {
    headers: buildHeaders('json', 'invoice-xml'),
  })
  if (!response.ok) {
    Notify.failure(tr("The file could not be downloaded.|No se pudo descargar el archivo."))
    return
  }

  const blob = await response.blob()
  const objectURL = URL.createObjectURL(blob)
  const link = document.createElement("a")
  link.href = objectURL
  link.download = fileName
  link.click()
  URL.revokeObjectURL(objectURL)
}
