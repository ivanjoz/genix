<script lang="ts">
import Button from '$components/buttons/Button.svelte';
import Input from '$components/form/Input.svelte';
import SearchSelect from '$components/form/SearchSelect.svelte';
import T from '$components/misc/T.svelte';
import TableGrid from '$components/vTable/TableGrid.svelte';
import type { ITableColumn } from '$components/vTable/types';
import { tr } from '$core/store.svelte';
import { Notify } from '$libs/helpers';
import { InvoiceSeriesSitesService } from './invoice-series.svelte';
import {
  DOC_TYPES, docTypeName, makeSeries, setDefaultSeries, validateSeries,
  type IInvoiceSeries,
} from './invoice-series';
import type { ICompany } from './empresas.svelte';

  const { company }: { company: ICompany } = $props()

  const sitesService = new InvoiceSeriesSitesService()

  // The form the "add" button reads. A series is only appended once all three are
  // set, so it never produces a row the backend would refuse.
  let newSeries = $state({ DocType: 0, SeriesCode: "", SiteID: 0 })

  // Zero is not "missing": it is the company's own fiscal address, which is what a
  // single-site company issues from and what the seeded series carry.
  const siteName = (siteID: number) =>
    siteID === 0
      ? tr("Legal address|Dirección legal")
      : sitesService.Sites.find(site => site.ID === siteID)?.Name || "—"

  const activeSeries = $derived((company.InvoiceSeries || []).filter(series => series.ss === 1))

  const addSeries = () => {
    if (!newSeries.DocType || !newSeries.SeriesCode) {
      Notify.failure(tr("Complete the document type and the series code.|Complete el tipo de comprobante y el código de serie."))
      return
    }
    const allSeries = company.InvoiceSeries || []
    const candidate = [...allSeries, makeSeries(allSeries, newSeries.DocType,
      newSeries.SeriesCode, newSeries.SiteID)]

    const problem = validateSeries(candidate)
    if (problem) {
      Notify.failure(tr(problem))
      return
    }
    company.InvoiceSeries = candidate
    newSeries = { DocType: 0, SeriesCode: "", SiteID: 0 }
  }

  // Retiring, not removing: a document issued under this series points at its id
  // forever, so the row stays and stops being offered.
  const retireSeries = (series: IInvoiceSeries) => {
    company.InvoiceSeries = (company.InvoiceSeries || []).map(current =>
      current.SeriesID === series.SeriesID ? { ...current, ss: 0, IsDefault: 0 } : current)
  }

  const columns: ITableColumn<IInvoiceSeries>[] = [
    {
      id: 'seriesID', header: 'No.|N°', width: '46px', align: 'right', css: 'ff-mono',
      getValue: (series) => series.SeriesID,
    },
    {
      id: 'docType', header: 'Type|Tipo', width: 'minmax(90px, 1fr)',
      render: (series) => tr(docTypeName(series.DocType)),
    },
    {
      id: 'seriesCode', header: 'Series|Serie', width: '72px', css: 'ff-mono ff-bold',
      getValue: (series) => series.SeriesCode,
    },
    {
      id: 'siteID', header: 'Branch|Sede', width: 'minmax(90px, 1fr)',
      render: (series) => siteName(series.SiteID),
    },
    {
      id: 'isDefault', header: 'Default|Predet.', width: '74px', align: 'center',
      showHoverEffect: true,
      render: (series) => series.IsDefault === 1
        ? `<i class="icon-[fa--check] text-green-600"></i>`
        : `<span class="text-slate-300">—</span>`,
      onCellClick: (series) => {
        company.InvoiceSeries = setDefaultSeries(company.InvoiceSeries || [], series.SeriesID)
      },
    },
    {
      id: 'actions', header: '', width: '44px', align: 'center',
      buttonDeleteHandler: (series) => retireSeries(series),
    },
  ]
</script>

<section class="col-span-24 lg:col-span-12 rounded-[12px] border border-slate-200 bg-white p-16 shadow-sm"
  aria-label="Electronic invoicing series maintainer">
  <div class="h3 ff-bold mb-4"><T text="Invoicing Series|Series de Facturación" /></div>
  <div class="text-[13px] text-slate-500 mb-12">
    <T text="Series are saved with the company. The number on the left is what identifies the series internally, not the SUNAT code.|Las series se guardan junto con la empresa. El número de la izquierda es lo que identifica la serie internamente, no el código SUNAT." />
  </div>

  <div class="grid grid-cols-24 gap-8 items-end mb-12">
    <!-- Not `required`: on a pristine form that paints an error marker on fields the user has
         not reached yet. Adding validates all three and names the one that is missing. -->
    <SearchSelect css="col-span-24 md:col-span-9" label="Type|Tipo"
      options={DOC_TYPES.map(type => ({ ID: type.ID, Name: tr(type.Name) }))}
      keyId="ID" keyName="Name" save="DocType" bind:saveOn={newSeries} />
    <Input css="col-span-10 md:col-span-5" label="Series|Serie" save="SeriesCode"
      bind:saveOn={newSeries} />
    <SearchSelect css="col-span-14 md:col-span-7" label="Branch|Sede"
      options={sitesService.Sites} keyId="ID" keyName="Name" save="SiteID"
      bind:saveOn={newSeries} />
    <!-- Left empty the series issues from the company's legal address. -->
    <Button css="col-span-24 md:col-span-3" color="blue" icon="icon-[fa--plus]"
      name="Add|Agregar" label="Adds the series to the company." onClick={addSeries} />
  </div>

  {#if activeSeries.length === 0}
    <div class="rounded-[8px] border border-dashed border-slate-300 p-14 text-center text-slate-500">
      <T text="No series yet. A sale cannot be invoiced until the company has one.|Aún no hay series. No se puede facturar una venta hasta que la empresa tenga una." />
    </div>
  {:else}
    <TableGrid {columns} data={activeSeries} rowHeight={32}
      getRowId={(series) => series.SeriesID} />
  {/if}
</section>
