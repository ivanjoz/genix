<script lang="ts">
import Button from '$components/buttons/Button.svelte'
import LabelCell from '$components/form/LabelCell.svelte'
import T from '$components/misc/T.svelte'
import { formatN, formatTime, Notify } from '$libs/helpers'
import { tr } from '$core/store.svelte'
import { getRecordByID } from '@genix/ui/cache'
import { formatQuantity, quantityAmount, unpackQuantityLine } from '$core/quantity'
import type { ISaleOrder } from '$routes/sales/sale_orders_status/sale_order_status.svelte'
import {
  hasInvoiceCDR, hasInvoiceXML, invoiceNumber, invoiceSaleOrderID, invoiceStateCss,
  invoiceStateLabels, type IInvoiceDocument,
} from './invoicing'
import { downloadInvoiceArtifact } from './invoicing.svelte'

interface Props {
  document: IInvoiceDocument
  // Resolved by the page from the company's series: the row only carries the id.
  seriesCode: string
}

// The by-ids cache keys and evicts on these three, so the shape it resolves has to
// carry them even when the panel only reads the name.
interface IProductName {
  ID: number
  Name: string
  ss: number
  upd: number
}

const { document: invoiceDocument, seriesCode }: Props = $props()

// The sale behind the document, fetched one id at a time through the by-ids cache.
// An annulled sale comes back undefined — the cache drops ss=0 rows — which is
// exactly the case of a voided document, so the panel says so instead of blanking.
let saleOrder = $state(undefined as ISaleOrder | undefined)
let saleOrderMissing = $state(false)
let productNames = $state(new Map<number, string>())

$effect(() => {
  const saleOrderID = invoiceSaleOrderID(invoiceDocument)
  saleOrder = undefined
  saleOrderMissing = false

  getRecordByID<ISaleOrder>("sale-order-by-ids", saleOrderID).then(record => {
    if (!record) { saleOrderMissing = true; return }
    saleOrder = record
    loadProductNames(record.DetailProductsIDs || [])
  })
})

// Only the products of this sale are fetched, which is the whole point of the
// by-ids cache: an accounting page must not download the product catalog.
async function loadProductNames(productIDs: number[]) {
  const resolvedNames = new Map<number, string>()
  await Promise.all(productIDs.map(async (productID) => {
    const product = await getRecordByID<IProductName>("p-products-ids", productID)
    if (product) resolvedNames.set(productID, product.Name)
  }))
  productNames = resolvedNames
}

const saleLines = $derived((saleOrder?.DetailProductsIDs || []).map((productID, index) => {
  const quantity = unpackQuantityLine(saleOrder?.DetailQuantities?.[index] || 0)
  const unitPrice = saleOrder?.DetailPrices?.[index] || 0
  const subUnitPrice = saleOrder?.DetailSubPrices?.[index] || 0
  return {
    productID,
    name: productNames.get(productID) || `#${productID}`,
    quantityLabel: formatQuantity(quantity, saleOrder?.DetailSubDivisor?.[index] || 1),
    amount: quantityAmount(quantity, unitPrice, subUnitPrice),
  }
}))

async function download(artifact: "xml" | "cdr") {
  const extension = artifact === "cdr" ? "zip" : "xml"
  const fileName = `${invoiceNumber(invoiceDocument, seriesCode)}-${artifact}.${extension}`
  try {
    await downloadInvoiceArtifact(invoiceDocument.ID, artifact, fileName)
  } catch (downloadError) {
    console.error("invoice artifact download failed", downloadError)
    Notify.failure(tr("The file could not be downloaded.|No se pudo descargar el archivo."))
  }
}
</script>

<div class="grid grid-cols-24 gap-10 mt-6 md:mt-16" aria-label="Invoice document detail">
  <LabelCell css="col-span-12" label="Status|Estado"
    valueCss={`h4 ${invoiceStateCss[invoiceDocument.State] || ""}`}
    value={tr(invoiceStateLabels[invoiceDocument.State] || "")}
  />
  <LabelCell css="col-span-12" label="Issue Date|Fecha de Emisión"
    value={formatTime(invoiceDocument.IssueDate, "d-m-Y") as string}
  />
  <LabelCell css="col-span-12" label="Total|Total"
    value={formatN(invoiceDocument.TotalAmount / 100, 2)}
  />
  <LabelCell css="col-span-12" label="IGV|IGV"
    value={formatN(invoiceDocument.TaxAmount / 100, 2)}
  />
</div>

<div class="mt-12 flex flex-wrap gap-8" aria-label="Invoice document downloads">
  <Button name="XML" icon="icon-[fa--download]" color="blue"
    disabled={!hasInvoiceXML(invoiceDocument.State)}
    onClick={() => download("xml")}
  />
  <Button name="CDR" icon="icon-[fa--download]" color="blue"
    disabled={!hasInvoiceCDR(invoiceDocument.State)}
    onClick={() => download("cdr")}
  />
</div>

<!-- What SUNAT said. Absent until the document was actually sent, which is why the
     whole block only appears when there is something in it. -->
{#if invoiceDocument.SunatCode || invoiceDocument.LastError || (invoiceDocument.SunatNotes || []).length}
  <div class="mt-16" aria-label="SUNAT response">
    <div class="h4 mb-6"><T text="SUNAT response|Respuesta de SUNAT" /></div>
    <div class="grid grid-cols-24 gap-10">
      {#if invoiceDocument.SunatCode}
        <LabelCell css="col-span-12" label="Code|Código" value={invoiceDocument.SunatCode} />
      {/if}
      {#if invoiceDocument.RetryCount > 0}
        <LabelCell css="col-span-12" label="Attempts|Intentos" value={invoiceDocument.RetryCount} />
      {/if}
    </div>
    {#each invoiceDocument.SunatNotes || [] as note}
      <div class="text-sm text-amber-700 mt-4">{note}</div>
    {/each}
    {#if invoiceDocument.LastError}
      <div class="text-sm text-red-600 mt-4">{invoiceDocument.LastError}</div>
    {/if}
  </div>
{/if}

<div class="mt-16" aria-label="Sale order detail">
  <div class="h4 mb-6"><T text="Sale|Venta" /> #{invoiceSaleOrderID(invoiceDocument)}</div>

  {#if saleOrderMissing}
    <div class="text-sm c-gray">
      <T text="The sale is annulled or no longer available.|La venta está anulada o ya no está disponible." />
    </div>
  {:else if saleOrder}
    <div class="grid grid-cols-24 gap-10 mb-10">
      <LabelCell css="col-span-12" label="Sale Date|Fecha de Venta"
        value={formatTime(saleOrder.Date, "d-m-Y") as string} />
      <LabelCell css="col-span-12" label="Sale Total|Total de la Venta"
        value={formatN(saleOrder.TotalAmount / 100, 2)} />
    </div>
    {#each saleLines as line (line.productID)}
      <div class="flex items-center gap-8 py-4 border-b border-gray-100 text-sm">
        <span class="c-blue ff-bold">{line.quantityLabel}</span>
        <span class="flex-1 truncate">{line.name}</span>
        <span>{formatN(line.amount / 100, 2)}</span>
      </div>
    {/each}
  {/if}
</div>
