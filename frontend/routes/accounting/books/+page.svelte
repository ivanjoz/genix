<script lang="ts">
import Button from '$components/buttons/Button.svelte'
import DateInput from '$components/form/DateInput.svelte'
import SearchSelect from '$components/form/SearchSelect.svelte'
import Layer from '$components/layers/Layer.svelte'
import T from '$components/misc/T.svelte'
import OptionsStrip from '$components/navigation/OptionsStrip.svelte'
import VTable from '$components/vTable/VTable.svelte'
import type { ExcelTableColumn } from '@genix/ui/excel'
import Page from '$domain/Page.svelte'
import { tr } from '$core/store.svelte'
import { sendUserNotification } from '$core/notifications.svelte'
import { formatN, formatTime, Notify } from '$libs/helpers'
import { docTypeName } from '$core/sunat-doc-type'
import { EmpresaParametrosService } from '../../company/configuration/empresas.svelte'
import {
  bookOptions, bookRowDocument, bookRowModifiedDocument, bookStateCss, bookStateLabels,
  buildPeriodOptions, buildSalesBook, isWholeMonthPeriod, periodCode,
  periodLabel, periodOfUnixDay, sumSalesBook, BOOK_SALES, BOOK_PURCHASES,
  type BookKind, type ISalesBookRow, type IUninvoicedSale,
} from './books'
import { exportSalesBookToExcel } from './books.excel'
import {
  downloadSalesBookTxt, salesBookExportBlockers, salesBookFileName,
  type ISalesBookIssuer,
} from './books.txt'
import { SalesBookService } from './books.svelte'

const salesBook = new SalesBookService()
// The series code and the document type are not on a comprobante — they live on the company
// record, and the row only carries the series id in the tail of its own id. Same join the
// invoicing page does.
const companyParameters = new EmpresaParametrosService()

const today = new Date()

// DateInput and SearchSelect write through saveOn/save, so the selection is an object.
// The current month is the default: a book is a monthly document, and a day is the exception.
let selection = $state({
  book: BOOK_SALES as BookKind,
  period: periodCode(today.getFullYear(), today.getMonth()),
  date: 0,
})

// 1 = the book's comprobantes, 2 = the period's sales that have none.
let salesView = $state(1)

const periodOptions = buildPeriodOptions(today)
const bookSelectOptions = bookOptions.map(option => ({ ID: option.ID, Name: tr(option.Name) }))

// A month locks the date picker: the book is the whole period, and a day inside it would only
// contradict the title.
const isWholeMonth = $derived(isWholeMonthPeriod(selection.period))
// What the rows would be filed under, whichever way the period was chosen.
const shownPeriod = $derived(isWholeMonth ? selection.period : periodOfUnixDay(selection.date))

const seriesByID = $derived(new Map(
  (companyParameters.empresa.InvoiceSeries || []).map(series => [series.SeriesID, series]),
))
const rows = $derived(
  salesBook.response
    ? buildSalesBook(salesBook.response, seriesByID, salesBook.identityBySnapshotID)
    : [],
)
const uninvoiced = $derived(salesBook.response?.UninvoicedSales || [])
const totals = $derived(sumSalesBook(rows))
const uninvoicedTotal = $derived(
  uninvoiced.reduce((runningTotal, sale) => runningTotal + (sale.TotalAmount || 0), 0),
)

async function loadBook() {
  if (selection.book !== BOOK_SALES) {
    Notify.failure(tr("This book is not available yet.|Este libro aún no está disponible."))
    return
  }
  if (!isWholeMonth && !selection.date) {
    Notify.failure(tr("Pick a month or a date.|Elige un mes o una fecha."))
    return
  }
  await salesBook.fetch(isWholeMonth ? selection.period : "", selection.date)
  if (exportBlockers.length > 0) notifyExportBlockers()
}

// Why the TXT button is off, sent once per read. Each line is one thing to fix, not a general warning.
function notifyExportBlockers() {
  const periodName = periodLabel(shownPeriod)
  sendUserNotification(
    exportBlockers.map(blocker => `• ${tr(blocker)}`).join("\n"),
    {
      title: tr(`Period ${periodName} cannot be filed yet|El periodo ${periodName} todavía no se puede presentar`),
      subtitle: tr("Accounting Books · Sales Register|Libros Contables · Registro de Ventas"),
      color: "yellow",
    },
  )
}

async function exportBook() {
  if (rows.length === 0) {
    Notify.failure(tr("There is nothing to export.|No hay nada que exportar."))
    return
  }
  await exportSalesBookToExcel(rows, shownPeriod)
}

// Fields 1-2 of the file. LegalName is the razón social SUNAT matches against the RUC; Name is
// the commercial one, and it is only a fallback for a company nobody has filled in yet.
const issuer = $derived({
  RUC: companyParameters.empresa.RUC || "",
  LegalName: companyParameters.empresa.LegalName || companyParameters.empresa.Name || "",
} as ISalesBookIssuer)

// Everything standing between this period and a filable file. Empty means the TXT is writable —
// including for a month with no comprobantes, which is filed as a "sin información" book.
const exportBlockers = $derived(
  salesBook.loadedPeriod ? salesBookExportBlockers(rows, issuer, shownPeriod) : [],
)
const txtFileName = $derived(
  exportBlockers.length === 0 && salesBook.loadedPeriod
    ? salesBookFileName(issuer.RUC, shownPeriod, rows.length > 0)
    : "",
)

function exportBookTxt() {
  if (exportBlockers.length > 0) {
    Notify.failure(tr(exportBlockers[0]))
    return
  }
  downloadSalesBookTxt(rows, issuer, shownPeriod)
}

// Columns are in SUNAT's field order, and the header says which field each one is — that is how
// a book gets checked against the annex.
const salesColumns: ExcelTableColumn<ISalesBookRow>[] = [
  {
    header: "Comprobante|Comprobante",
    highlight: true,
    css: "c-blue",
    headerCss: "w-150",
    getValue: (row) => bookRowDocument(row),
    mobile: { order: 1, css: "col-span-12 ff-bold", icon: "[fa--file-text-o]" },
  },
  {
    header: "Type|Tipo",
    headerCss: "w-90",
    getValue: (row) => tr(docTypeName(row.DocType)),
    mobile: { order: 2, css: "col-span-12", labelLeft: "Tipo:" },
  },
  {
    header: "Issued|Emisión",
    headerCss: "w-100",
    getValue: (row) => formatTime(row.IssueDate, "d-m-Y") as string,
    mobile: { order: 3, css: "col-span-12", labelLeft: "Emisión:" },
  },
  {
    header: "Buyer|Cliente",
    getValue: (row) => row.BuyerName,
    useLineClamp: true,
    mobile: { order: 4, css: "col-span-24", labelLeft: "Cliente:" },
  },
  {
    header: "Doc.|Doc.",
    headerCss: "w-110",
    getValue: (row) => row.BuyerDocNumber,
    mobile: { order: 5, css: "col-span-12", labelLeft: "Doc:" },
  },
  {
    header: "Taxable|Base imp.",
    css: "text-right",
    getValue: (row) => formatN(row.TaxableAmount / 100, 2),
    mobile: { order: 6, css: "col-span-12", labelLeft: "Base:" },
  },
  {
    header: "IGV",
    css: "text-right",
    getValue: (row) => formatN(row.TaxAmount / 100, 2),
    mobile: { order: 7, css: "col-span-12", labelLeft: "IGV:" },
  },
  {
    header: "Total",
    css: "text-right",
    getValue: (row) => formatN(row.TotalAmount / 100, 2),
    mobile: { order: 8, css: "col-span-12 ff-bold", labelLeft: "Total:" },
  },
  {
    header: "Modifies|Modifica",
    headerCss: "w-120",
    getValue: (row) => bookRowModifiedDocument(row),
    mobile: { order: 9, css: "col-span-12", labelLeft: "Modifica:" },
  },
  {
    header: "Status|Estado",
    headerCss: "w-120",
    setCellCss: (row) => bookStateCss[row.ValidityState] || "",
    getValue: (row) => {
      const stateLabel = tr(bookStateLabels[row.ValidityState] || "")
      // A consolidated row stands for several boletas; saying how many is what makes the
      // range readable next to a detailed row.
      return row.ConsolidatedCount > 1 ? `${stateLabel} · ${row.ConsolidatedCount} blt.` : stateLabel
    },
    mobile: { order: 10, css: "col-span-12", labelLeft: "Estado:" },
  },
]

const uninvoicedColumns: ExcelTableColumn<IUninvoicedSale>[] = [
  {
    header: "Sale|Venta",
    highlight: true,
    css: "c-blue",
    getValue: (sale) => String(sale.ID),
    mobile: { order: 1, css: "col-span-12 ff-bold" },
  },
  {
    header: "Date|Fecha",
    getValue: (sale) => formatTime(sale.Date, "d-m-Y") as string,
    mobile: { order: 2, css: "col-span-12", labelLeft: "Fecha:" },
  },
  {
    header: "Total",
    css: "text-right",
    getValue: (sale) => formatN(sale.TotalAmount / 100, 2),
    mobile: { order: 3, css: "col-span-12", labelLeft: "Total:" },
  },
]
</script>

<Page title="Accounting Books|Libros Contables">
  <div class="grid grid-cols-12 md:flex md:flex-row md:items-center gap-8 mb-8">
    <SearchSelect label="Book|Libro" css="col-span-12 md:w-220"
      keyId="ID" keyName="Name"
      options={bookSelectOptions}
      saveOn={selection} save="book"
    />
    <DateInput label="Date|Fecha" css="col-span-6 md:w-150"
      disabled={isWholeMonth}
      saveOn={selection} save="date"
    />
    <SearchSelect label="Month|Mes" css="col-span-6 md:w-190"
      keyId="ID" keyName="Name"
      options={periodOptions}
      saveOn={selection} save="period"
    />
    <Button name="Read|Consultar" color="blue" icon="icon-[fa--search]"
      css="col-span-6 md:ml-4" onClick={loadBook} />
    <Button name="Excel" color="green" icon="icon-[fa--file-excel-o]"
      css="col-span-6" onClick={exportBook} />
    <Button name="TXT SUNAT" color="blue" icon="icon-[fa--file-text-o]"
      css="col-span-6" disabled={!salesBook.loadedPeriod || exportBlockers.length > 0}
      onClick={exportBookTxt} />

    {#if shownPeriod}
      <div class="col-span-12 md:ml-auto text-sm flex items-center gap-6">
        <T text="Period|Periodo" />
        <span class="ff-mono font-semibold">{shownPeriod}</span>
        <span class="text-gray-500">{periodLabel(shownPeriod)}</span>
      </div>
    {/if}
  </div>

  {#if selection.book === BOOK_SALES}
    <div class="flex flex-wrap items-center gap-16 mb-8">
      <OptionsStrip
        selected={salesView}
        options={[
          [1, "Booked|Contabilizadas"],
          [2, "No comprobante|Sin Comprobante"],
        ]}
        onSelect={(option) => { salesView = option[0] as number }}
      />
      {#if salesView === 1 && rows.length > 0}
        <div class="flex flex-wrap items-center gap-16 md:ml-auto text-sm">
          <div><T text="Comprobantes|Comprobantes" />: <span class="font-semibold">{totals.comprobantes}</span></div>
          <div><T text="Taxable|Base imp." />: <span class="font-semibold">{formatN(totals.taxable / 100, 2)}</span></div>
          <div>IGV: <span class="font-semibold">{formatN(totals.tax / 100, 2)}</span></div>
          <div><T text="Total|Total" />: <span class="font-semibold">{formatN(totals.total / 100, 2)}</span></div>
        </div>
      {/if}
    </div>

    {#if txtFileName}
      <div class="mb-8 text-sm text-gray-600 flex items-center gap-6">
        <i class="icon-[fa--file-text-o] shrink-0"></i>
        <T text="Replacement file|Archivo de reemplazo" />
        <span class="ff-mono">{txtFileName}</span>
      </div>
    {/if}

    <Layer type="content">
      {#if salesView === 1}
        <VTable columns={salesColumns} data={rows} mobileCardCss="mb-2" />
      {:else}
        <div class="text-sm text-gray-600 mb-8">
          <T text="Not part of the book — the Registro de Ventas registers comprobantes. Shown so the period can be reviewed.|No forman parte del libro — el Registro de Ventas registra comprobantes. Se muestran para revisar el periodo." />
        </div>
        <VTable columns={uninvoicedColumns} data={uninvoiced} mobileCardCss="mb-2" />
        {#if uninvoiced.length > 0}
          <div class="mt-8 px-8 py-8 bg-gray-50 rounded text-sm">
            <T text="Total|Total" />: <span class="font-semibold">{formatN(uninvoicedTotal / 100, 2)}</span>
          </div>
        {/if}
      {/if}
    </Layer>
  {:else}
    <Layer type="content">
      <div class="py-40 text-center text-gray-500">
        <T text="This book is not built yet.|Este libro aún no está construido." />
        {#if selection.book === BOOK_PURCHASES}
          <div class="mt-6 text-sm">
            <T text="Purchases need the supplier's comprobante on each purchase and expense.|Compras necesita el comprobante del proveedor en cada compra y gasto." />
          </div>
        {/if}
      </div>
    </Layer>
  {/if}
</Page>
