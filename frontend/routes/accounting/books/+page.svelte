<script lang="ts">
import Button from '$components/buttons/Button.svelte'
import Checkbox from '$components/form/Checkbox.svelte'
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
import { makePurchasesBookService, makeSalesBookService } from './books.svelte'
import {
  buildPurchasesBook, purchaseRowDocument, purchasesBookExportBlockers, purchaseSourceLabels,
  sumPurchasesBook, type IPurchaseBookRow, type IUndocumentedPurchase,
} from './books.purchases'
import { downloadPurchasesBookTxt, purchasesBookFileName } from './books.purchases.txt'
import { exportPurchasesBookToExcel } from './books.excel'
import { purchaseDocTypeName } from '$core/purchase-document'

const salesBook = makeSalesBookService()
const purchasesBook = makePurchasesBookService()
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
// Purchases without a comprobante are not book rows; they are shown only on request.
let purchasesView = $state({ showUndocumented: false })

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

const purchases = $derived(
  purchasesBook.response
    ? buildPurchasesBook(purchasesBook.response, purchasesBook.identityBySnapshotID)
    : { rows: [], undocumented: [] },
)
const purchaseTotals = $derived(sumPurchasesBook(purchases.rows))

const isPurchases = $derived(selection.book === BOOK_PURCHASES)
const loadedPeriod = $derived(isPurchases ? purchasesBook.loadedPeriod : salesBook.loadedPeriod)
const bookRowCount = $derived(isPurchases ? purchases.rows.length : rows.length)

async function loadBook() {
  if (selection.book !== BOOK_SALES && selection.book !== BOOK_PURCHASES) {
    Notify.failure(tr("This book is not available yet.|Este libro aún no está disponible."))
    return
  }
  if (!isWholeMonth && !selection.date) {
    Notify.failure(tr("Pick a month or a date.|Elige un mes o una fecha."))
    return
  }
  const book = isPurchases ? purchasesBook : salesBook
  await book.fetch(isWholeMonth ? selection.period : "", selection.date)
  if (exportBlockers.length > 0) notifyExportBlockers()
}

// Why the TXT button is off, sent once per read. Each line is one thing to fix, not a general warning.
function notifyExportBlockers() {
  const periodName = periodLabel(shownPeriod)
  sendUserNotification(
    exportBlockers.map(blocker => `• ${tr(blocker)}`).join("\n"),
    {
      title: tr(`Period ${periodName} cannot be filed yet|El periodo ${periodName} todavía no se puede presentar`),
      subtitle: tr(isPurchases
        ? "Accounting Books · Purchases Register|Libros Contables · Registro de Compras"
        : "Accounting Books · Sales Register|Libros Contables · Registro de Ventas"),
      color: "yellow",
    },
  )
}

async function exportBook() {
  if (bookRowCount === 0) {
    Notify.failure(tr("There is nothing to export.|No hay nada que exportar."))
    return
  }
  if (isPurchases) {
    await exportPurchasesBookToExcel(purchases.rows, shownPeriod)
  } else {
    await exportSalesBookToExcel(rows, shownPeriod)
  }
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
  !loadedPeriod ? []
    : isPurchases ? purchasesBookExportBlockers(purchases.rows, issuer, shownPeriod)
    : salesBookExportBlockers(rows, issuer, shownPeriod),
)
const txtFileName = $derived(
  exportBlockers.length > 0 || !loadedPeriod ? ""
    : isPurchases ? purchasesBookFileName(issuer.RUC, shownPeriod, bookRowCount > 0)
    : salesBookFileName(issuer.RUC, shownPeriod, bookRowCount > 0),
)

function exportBookTxt() {
  if (exportBlockers.length > 0) {
    Notify.failure(tr(exportBlockers[0]))
    return
  }
  if (isPurchases) {
    downloadPurchasesBookTxt(purchases.rows, issuer, shownPeriod)
  } else {
    downloadSalesBookTxt(rows, issuer, shownPeriod)
  }
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

const purchaseColumns: ExcelTableColumn<IPurchaseBookRow>[] = [
  {
    header: "Comprobante|Comprobante",
    highlight: true,
    css: "c-blue",
    headerCss: "w-150",
    getValue: (row) => purchaseRowDocument(row),
    mobile: { order: 1, css: "col-span-12 ff-bold", icon: "[fa--file-text-o]" },
  },
  {
    header: "Type|Tipo",
    headerCss: "w-110",
    getValue: (row) => tr(purchaseDocTypeName(row.DocType)),
    mobile: { order: 2, css: "col-span-12", labelLeft: "Tipo:" },
  },
  {
    header: "Issued|Emisión",
    headerCss: "w-100",
    getValue: (row) => formatTime(row.IssueDate, "d-m-Y") as string,
    mobile: { order: 3, css: "col-span-12", labelLeft: "Emisión:" },
  },
  {
    header: "Supplier|Proveedor",
    getValue: (row) => row.SupplierName,
    useLineClamp: true,
    mobile: { order: 4, css: "col-span-24", labelLeft: "Proveedor:" },
  },
  {
    header: "RUC",
    headerCss: "w-110",
    getValue: (row) => row.SupplierDocNumber,
    mobile: { order: 5, css: "col-span-12", labelLeft: "RUC:" },
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
    header: "Untaxed|No grav.",
    css: "text-right",
    getValue: (row) => formatN(row.UntaxedAmount / 100, 2),
    mobile: { order: 8, css: "col-span-12", labelLeft: "No grav.:" },
  },
  {
    header: "Total",
    css: "text-right",
    getValue: (row) => `${formatN(row.TotalAmount / 100, 2)}${row.Currency === "PEN" ? "" : ` ${row.Currency}`}`,
    mobile: { order: 9, css: "col-span-12 ff-bold", labelLeft: "Total:" },
  },
  {
    header: "Source|Origen",
    headerCss: "w-140",
    getValue: (row) => row.Sources.map(source => `${tr(purchaseSourceLabels[source.Source])} ${source.SourceID}`).join(", "),
    useLineClamp: true,
    mobile: { order: 10, css: "col-span-24", labelLeft: "Origen:" },
  },
]

const undocumentedColumns: ExcelTableColumn<IUndocumentedPurchase>[] = [
  {
    header: "Source|Origen",
    highlight: true,
    css: "c-blue",
    headerCss: "w-170",
    getValue: (purchase) => `${tr(purchaseSourceLabels[purchase.Source])} ${purchase.SourceID}`,
    mobile: { order: 1, css: "col-span-12 ff-bold", icon: "[fa--exclamation-triangle]" },
  },
  {
    header: "Date|Fecha",
    headerCss: "w-100",
    getValue: (purchase) => formatTime(purchase.Date, "d-m-Y") as string,
    mobile: { order: 2, css: "col-span-12", labelLeft: "Fecha:" },
  },
  {
    header: "Purchase|Compra",
    getValue: (purchase) => purchase.Name,
    useLineClamp: true,
    mobile: { order: 3, css: "col-span-24" },
  },
  {
    header: "Supplier|Proveedor",
    getValue: (purchase) => purchase.SupplierName,
    useLineClamp: true,
    mobile: { order: 4, css: "col-span-12", labelLeft: "Proveedor:" },
  },
  {
    header: "Amount|Monto",
    css: "text-right",
    getValue: (purchase) => `${formatN(purchase.Amount / 100, 2)}${purchase.Currency === "PEN" ? "" : ` ${purchase.Currency}`}`,
    mobile: { order: 5, css: "col-span-12", labelLeft: "Monto:" },
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
      css="col-span-6" disabled={!loadedPeriod || exportBlockers.length > 0}
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
  {:else if isPurchases}
    <div class="flex flex-wrap items-center gap-16 mb-8">
      <div class="flex items-center gap-6">
        <i class="icon-[fa--exclamation-triangle] text-amber-500 shrink-0"></i>
        <Checkbox saveOn={purchasesView} save="showUndocumented"
          label={tr(`Purchases without comprobante (${purchases.undocumented.length})|Compras sin comprobante (${purchases.undocumented.length})`)} />
      </div>
      {#if purchases.rows.length > 0}
        <div class="flex flex-wrap items-center gap-16 md:ml-auto text-sm">
          <div><T text="Comprobantes|Comprobantes" />: <span class="font-semibold">{purchaseTotals.comprobantes}</span></div>
          <div><T text="Taxable|Base imp." />: <span class="font-semibold">{formatN(purchaseTotals.taxable / 100, 2)}</span></div>
          <div>IGV: <span class="font-semibold">{formatN(purchaseTotals.tax / 100, 2)}</span></div>
          <div><T text="Total (S/)|Total (S/)" />: <span class="font-semibold">{formatN(purchaseTotals.total / 100, 2)}</span></div>
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
      <VTable columns={purchaseColumns} data={purchases.rows} mobileCardCss="mb-2" />
      {#if purchasesView.showUndocumented}
        <div class="mt-16 mb-8 text-sm text-amber-700 flex items-center gap-6">
          <i class="icon-[fa--exclamation-triangle] shrink-0"></i>
          <T text="Not part of the book — these purchases have no supplier comprobante, so they give no crédito fiscal. Register it on the purchase to book it.|No forman parte del libro — estas compras no tienen comprobante del proveedor, así que no dan crédito fiscal. Regístralo en la compra para anotarla." />
        </div>
        <VTable columns={undocumentedColumns} data={purchases.undocumented} mobileCardCss="mb-2" />
      {/if}
    </Layer>
  {:else}
    <Layer type="content">
      <div class="py-40 text-center text-gray-500">
        <T text="This book is not built yet.|Este libro aún no está construido." />
      </div>
    </Layer>
  {/if}
</Page>
