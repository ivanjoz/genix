<script lang="ts">
import Button from '$components/buttons/Button.svelte';
import Checkbox from '$components/form/Checkbox.svelte';
import Input from '$components/form/Input.svelte';
import SearchSelect from '$components/form/SearchSelect.svelte';
import Modal from '$components/layers/Modal.svelte';
import T from '$components/misc/T.svelte';
import TableGrid from '$components/vTable/TableGrid.svelte';
import type { ITableColumn } from '$components/vTable/types';
import { tr } from '$core/store.svelte';
import { useUI } from '@genix/ui';
import { Loading, Notify } from '$libs/helpers';
import { InvoiceSeriesSitesService, postInvoiceSeries } from './invoice-series.svelte';
import {
  DOC_TYPES, docTypeName, makeSeries, validateSeries,
  type IInvoiceSeries,
} from './invoice-series';
import type { ICompany } from './empresas.svelte';

  const { company }: { company: ICompany } = $props()

  const ui = useUI()
  const SERIES_MODAL_ID = 21

  const sitesService = new InvoiceSeriesSitesService()

  // The series being added or edited. SeriesID 0 marks a new one, which is what
  // tells the endpoint to allocate the next id.
  let seriesForm = $state({ SeriesID: 0, DocType: 0, SeriesCode: "", SiteID: 0 } as IInvoiceSeries)
  let isSaving = $state(false)

  // No site is not "missing": it is the company's own fiscal address, which is what
  // a single-site company issues from and what the seeded series carry. Tested for
  // falsiness rather than against 0, because the backend omits the field when it is
  // zero and it arrives undefined.
  const siteName = (siteID: number) =>
    !siteID
      ? tr("Legal address|Dirección legal")
      : sitesService.Sites.find(site => site.ID === siteID)?.Name || "—"

  // Every series, not just the active ones: hiding an inactive series would make it
  // unreachable, and the checkbox that brings it back lives in its own dialog. They
  // sort to the end, because they are the ones nobody is looking for.
  const allSeries = $derived(
    [...(company.InvoiceSeries || [])].sort((a, b) =>
      (b.ss || 0) - (a.ss || 0) || a.SeriesID - b.SeriesID))

  const openNewSeries = () => {
    seriesForm = { SeriesID: 0, DocType: 0, SeriesCode: "", SiteID: 0, IsDefault: 0, ss: 1 }
    ui.openModal(SERIES_MODAL_ID)
  }

  const openSeries = (series: IInvoiceSeries) => {
    seriesForm = { ...series }
    ui.openModal(SERIES_MODAL_ID)
  }

  // Saving writes through the series endpoint, not the company form: the reply is
  // the whole set, which replaces what this page holds.
  const saveSeries = async () => {
    if (!seriesForm.DocType || !seriesForm.SeriesCode) {
      Notify.failure(tr("Complete the document type and the series code.|Complete el tipo de comprobante y el código de serie."))
      return
    }

    const allSeries = company.InvoiceSeries || []
    const isNew = seriesForm.SeriesID === 0
    const candidate = isNew
      ? [...allSeries, makeSeries(allSeries, seriesForm.DocType, seriesForm.SeriesCode, seriesForm.SiteID)]
      : allSeries.map(series => series.SeriesID === seriesForm.SeriesID ? { ...seriesForm } : series)

    // Checked here so a bad series never reaches the endpoint; the backend checks
    // the same set again, because it is the one that has to be right.
    const problem = validateSeries(candidate)
    if (problem) {
      Notify.failure(tr(problem))
      return
    }

    isSaving = true
    Loading.standard(tr("Saving...|Guardando..."))
    try {
      const payload = isNew ? candidate[candidate.length - 1] : { ...seriesForm }
      company.InvoiceSeries = await postInvoiceSeries(payload)
      Notify.success(tr("Series saved|Serie guardada"))
      ui.closeModal(SERIES_MODAL_ID)
    } catch (error) {
      // Reported by POST.
    }
    Loading.remove()
    isSaving = false
  }

  const makeDefault = async (series: IInvoiceSeries) => {
    if (series.IsDefault === 1) return

    Loading.standard(tr("Saving...|Guardando..."))
    try {
      company.InvoiceSeries = await postInvoiceSeries({ ...series, IsDefault: 1 })
    } catch (error) {
      // Reported by POST.
    }
    Loading.remove()
  }

  const columns: ITableColumn<IInvoiceSeries>[] = [
    {
      id: 'seriesID', header: 'No.|N°', width: '46px', align: 'right', css: 'ff-mono',
      getValue: (series) => series.SeriesID,
    },
    {
      id: 'docType', header: 'Type|Tipo', width: 'minmax(90px, 1fr)',
      render: (series) => series.ss === 1
        ? tr(docTypeName(series.DocType))
        : `${tr(docTypeName(series.DocType))} <span class="text-red-600 ff-bold text-[14px]">${tr("inactive|inactiva")}</span>`,
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
      onCellClick: (series) => { void makeDefault(series) },
    },
    // No delete here: it lives in the dialog, so a destructive action is never one
    // click away in a grid.
    {
      id: 'actions', header: '', width: '44px', align: 'center',
      buttonEditHandler: (series) => openSeries(series),
    },
  ]
</script>

<section class="rounded-[12px] border border-slate-200 bg-white p-16 shadow-sm"
  aria-label="Electronic invoicing series maintainer">
  <div class="flex items-start gap-10 mb-12">
    <div class="flex-1 min-w-0">
      <div class="h3 ff-bold mb-4"><T text="Invoicing Series|Series de Facturación" /></div>
      <div class="text-[13px] text-slate-500">
        <T text="Each series saves on its own, not with the company parameters. The number on the left identifies the series internally, not the SUNAT code.|Cada serie se guarda por su cuenta, no con los parámetros de la empresa. El número de la izquierda identifica la serie internamente, no el código SUNAT." />
      </div>
    </div>
    <!-- No `name`: the Button renders icon-only, and `label` is what the agent
         registry and screen readers read. -->
    <Button color="green" icon="icon-[fa--plus]"
      label="Opens the dialog that adds an invoicing series."
      onClick={openNewSeries} />
  </div>

  {#if allSeries.length === 0}
    <div class="rounded-[8px] border border-dashed border-slate-300 p-14 text-center text-slate-500">
      <T text="No series yet. A sale cannot be invoiced until the company has one.|Aún no hay series. No se puede facturar una venta hasta que la empresa tenga una." />
    </div>
  {:else}
    <TableGrid {columns} data={allSeries} rowHeight={32}
      getRowId={(series) => series.SeriesID} />
  {/if}
</section>

<Modal id={SERIES_MODAL_ID} size={4}
  title={seriesForm.SeriesID === 0 ? tr("New series|Nueva serie") : tr("Series|Serie") + " " + seriesForm.SeriesCode}
  isEdit={seriesForm.SeriesID !== 0}
  saveIcon="icon-[fa--floppy-o]" saveButtonLabel={tr("Save|Guardar")}
  onSave={() => { if (!isSaving) void saveSeries() }}
>
  <div class="grid grid-cols-24 gap-10 content-start pt-6">
    <SearchSelect css="col-span-24" label="Type|Tipo"
      options={DOC_TYPES.map(type => ({ ID: type.ID, Name: tr(type.Name) }))}
      keyId="ID" keyName="Name" save="DocType" bind:saveOn={seriesForm} />
    <Input css="col-span-24 md:col-span-10" label="Series|Serie" save="SeriesCode"
      bind:saveOn={seriesForm} />
    <SearchSelect css="col-span-24 md:col-span-14" label="Branch|Sede"
      options={sitesService.Sites} keyId="ID" keyName="Name" save="SiteID"
      bind:saveOn={seriesForm} />
    <Checkbox css="col-span-24" label="Active|Activa" save="ss" useNumber={true}
      bind:saveOn={seriesForm} />
    <div class="col-span-24 text-[13px] text-slate-500">
      <T text="A series code has 4 characters and starts with F for invoices or B for receipts. A note carries the prefix of the document it corrects. Leave the branch empty to issue from the legal address.|El código de serie tiene 4 caracteres y empieza con F para facturas o B para boletas. Una nota lleva el prefijo del documento que corrige. Deje la sede vacía para emitir desde la dirección legal." />
    </div>
  </div>
</Modal>
