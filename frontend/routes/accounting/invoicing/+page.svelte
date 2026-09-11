<script lang="ts">
import { useUI } from '@genix/ui'
import Button from '$components/buttons/Button.svelte'
import DateInput from '$components/form/DateInput.svelte'
import FilterInput from '$components/form/FilterInput.svelte'
import SearchSelect from '$components/form/SearchSelect.svelte'
import Layer from '$components/layers/Layer.svelte'
import T from '$components/misc/T.svelte'
import VTable from '$components/vTable/VTable.svelte'
import type { ExcelTableColumn } from '@genix/ui/excel'
import Page from '$domain/Page.svelte'
import { tr } from '$core/store.svelte'
import { formatN, formatTime, Loading, Notify } from '$libs/helpers'
import { EmpresaParametrosService } from '../../company/configuration/empresas.svelte'
import InvoiceDetailLayer from './InvoiceDetailLayer.svelte'
import {
  canSendInvoice, countPendingInvoices, filterInvoices, invoiceNumber, invoiceSaleOrderID,
  invoiceSeriesID, invoiceStateCss, invoiceStateLabels, sunatVerdictMessage,
  type IInvoiceDocument, type InvoiceFilter,
} from './invoicing'
import { InvoicesService, postInvoiceSend } from './invoicing.svelte'

const ui = useUI()

const invoices = new InvoicesService(true)
// The series code is not on the document — it lives on the company record, and the
// document only carries the series id in the tail of its own id.
const companyParameters = new EmpresaParametrosService()

// DateInput and SearchSelect write through saveOn/save, so the filter is an object.
let filter = $state({ fromDate: 0, toDate: 0, idText: "", state: 0 } as InvoiceFilter)
let idFilterText = $state("")
let selectedDocument = $state(null as IInvoiceDocument | null)

const seriesCodeByID = $derived(new Map(
  (companyParameters.empresa.InvoiceSeries || []).map(series => [series.SeriesID, series.SeriesCode]),
))
const seriesCodeOf = (document: IInvoiceDocument) =>
  seriesCodeByID.get(invoiceSeriesID(document)) || ""

const visibleDocuments = $derived(
  filterInvoices(invoices.records, { ...filter, idText: idFilterText }),
)
const pendingCount = $derived(countPendingInvoices(invoices.records))

const stateOptions = [
  { ID: 0, Name: "All states|Todos los estados" },
  ...Object.entries(invoiceStateLabels).map(([state, label]) => ({ ID: Number(state), Name: label })),
].map(option => ({ ID: option.ID, Name: tr(option.Name) }))

const invoiceColumns: ExcelTableColumn<IInvoiceDocument>[] = [
  {
    header: "Document|Comprobante",
    highlight: true,
    css: "c-blue",
    headerCss: "w-120",
    getValue: (document) => invoiceNumber(document, seriesCodeOf(document)),
    mobile: { order: 1, css: "col-span-12 ff-bold", icon: "[fa--file-text-o]" },
  },
  {
    header: "Issued|Emitido",
    getValue: (document) => formatTime(document.IssueDate, "d-m-Y") as string,
    mobile: { order: 2, css: "col-span-12", labelLeft: "Emitido:" },
  },
  {
    header: "Sale|Venta",
    getValue: (document) => invoiceSaleOrderID(document),
    mobile: { order: 3, css: "col-span-12", labelLeft: "Venta:" },
  },
  {
    header: "Total|Total",
    css: "text-right",
    getValue: (document) => formatN(document.TotalAmount / 100, 2),
    mobile: { order: 4, css: "col-span-12", labelLeft: "Total:" },
  },
  {
    header: "Status|Estado",
    setCellCss: (document) => invoiceStateCss[document.State] || "",
    getValue: (document) => tr(invoiceStateLabels[document.State] || ""),
    mobile: { order: 5, css: "col-span-12", labelLeft: "Estado:" },
  },
  {
    header: "SUNAT",
    getValue: (document) => document.SunatCode || document.LastError || "",
    useLineClamp: true,
    mobile: { order: 6, css: "col-span-24", labelLeft: "SUNAT:" },
  },
]

async function sendSelectedDocument() {
  if (!selectedDocument) return

  // The request waits for SUNAT, which regularly takes tens of seconds, so the
  // message says what is being waited on rather than a generic "guardando".
  Loading.standard(tr("Sending to SUNAT, this may take a moment...|Enviando a SUNAT, puede demorar un momento..."))
  try {
    // What comes back is the document with SUNAT's verdict already on it.
    const sentDocument = await postInvoiceSend(selectedDocument.ID) as IInvoiceDocument
    Notify.success(tr(sunatVerdictMessage(sentDocument)))
    // The panel stays open on the answer: a rejection is what the operator needs
    // to read, and closing the layer would hide the code that explains it.
    selectedDocument = sentDocument
  } catch (sendError) {
    console.error("invoice send failed", sendError)
  } finally {
    // Either way the document moved, so the list has to show where it landed.
    await invoices.fetch()
    Loading.remove()
  }
}
</script>

<Page title="Invoicing|Facturación">
  <div class="grid grid-cols-12 md:flex md:flex-row md:items-center gap-8 mb-8">
    <DateInput label="From|Desde" css="col-span-6 md:w-150" saveOn={filter} save="fromDate" />
    <DateInput label="To|Hasta" css="col-span-6 md:w-150" saveOn={filter} save="toDate" />
    <SearchSelect label="Status|Estado" css="col-span-12 md:w-200"
      keyId="ID" keyName="Name"
      options={stateOptions}
      saveOn={filter} save="state"
    />
    <FilterInput label="Document or sale ID|ID de comprobante o venta"
      css="col-span-12 md:w-220"
      icon="icon-[fa--search]"
      bind:value={idFilterText}
    />
    {#if pendingCount > 0}
      <div class="col-span-12 md:ml-auto text-sm text-amber-700 flex items-center gap-6">
        <i class="icon-[fa--clock-o] shrink-0"></i>
        <T text={`${pendingCount} pending to send|${pendingCount} pendientes de envío`} />
      </div>
    {/if}
  </div>

  <Layer type="content">
    <VTable
      columns={invoiceColumns}
      data={visibleDocuments}
      selected={selectedDocument?.ID}
      isSelected={(document, selectedID) => document.ID === selectedID}
      onRowClick={(document) => { selectedDocument = document; ui.openSideLayer(1) }}
      mobileCardCss="mb-2"
    />
  </Layer>

  <Layer
    type="side"
    id={1}
    sideLayerSize={640}
    css="px-8 py-8 md:px-16 md:py-10"
    contentCss="px-0 md:px-0"
    title={selectedDocument ? invoiceNumber(selectedDocument, seriesCodeOf(selectedDocument)) : ""}
    titleCss="h2 mb-6"
    saveButtonName="Send now|Enviar ahora"
    saveButtonIcon="icon-[fa--paper-plane]"
    onSave={canSendInvoice(selectedDocument?.State ?? -1) ? sendSelectedDocument : undefined}
    onClose={() => { selectedDocument = null }}
  >
    {#if selectedDocument}
      <InvoiceDetailLayer document={selectedDocument} seriesCode={seriesCodeOf(selectedDocument)} />
    {/if}
  </Layer>
</Page>
