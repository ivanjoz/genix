<script lang="ts">
  import { useUI } from '@genix/ui'
  import { confirmWarn, hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
  import Page from '#domain/Page.svelte'
  import Button from '#components/buttons/Button.svelte'
  import DateInput from '#components/form/DateInput.svelte'
  import Input from '#components/form/Input.svelte'
  import Modal from '#components/layers/Modal.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import { formatTime } from '#libs/helpers.ts'
  import {
    postTaxDueCalendar, TaxDueCalendarService, type ITaxDueCalendar,
  } from '#services/accounting/tax-due-calendar.svelte.ts'
  import {
    formatTaxPeriod, nextTaxPeriod, TAX_DUE_COLUMN_LABELS, TAX_DUE_COLUMNS, taxCalendarFormProblem,
  } from './tax-calendar.ts'

  // SaaS only: SUNAT's monthly due-date schedule every company's cash flow forecast dates its taxes
  // with. Transcribed from sunat.gob.pe/orientacion/cronogramas/<year>/cObligacionMensual<year>.html,
  // never computed: decreed non-working days move the dates.
  const ui = useUI()
  const CALENDAR_MODAL_ID = 43
  const taxCalendar = new TaxDueCalendarService(true)

  let calendarForm = $state({ Period: 0, DueDays: [], ss: 1 } as unknown as ITaxDueCalendar)
  let isStoredPeriod = $state(false)

  const openPeriod = (calendar?: ITaxDueCalendar) => {
    isStoredPeriod = !!calendar
    calendarForm = calendar
      ? { ...calendar, DueDays: [...calendar.DueDays] }
      : { Period: nextTaxPeriod(taxCalendar.calendars), DueDays: Array(TAX_DUE_COLUMNS).fill(0), ss: 1 } as ITaxDueCalendar
    ui.openModal(CALENDAR_MODAL_ID)
  }

  const savePeriod = async (isDelete = false) => {
    if (isDelete) {
      if (!(await confirmWarn({ title: tr('Delete|Eliminar'), message: tr('Delete this period?|¿Eliminar este periodo?') }))) return
    } else {
      const problem = taxCalendarFormProblem(calendarForm)
      if (problem) { notifyFailure(tr(problem)); return }
    }
    showLoading(tr('Saving...|Guardando...'))
    try {
      await postTaxDueCalendar(taxCalendar, [{ ...calendarForm, ss: isDelete ? 0 : 1 }])
      ui.closeModal(CALENDAR_MODAL_ID)
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }

  const columns: ITableColumn<ITaxDueCalendar>[] = [
    { header: 'Period|Periodo', css: 'px-6', highlight: true, getValue: (e) => formatTaxPeriod(e.Period) },
    ...TAX_DUE_COLUMN_LABELS.map((label, column) => ({
      header: label, css: 'px-6 text-center',
      getValue: (e: ITaxDueCalendar) => String(formatTime(e.DueDays[column], 'd-m')),
    })),
  ]
</script>

<Page title="Tax Calendar|Calendario Tributario">
  <div class="flex items-center justify-between mb-8">
    <div class="text-fg-muted">
      {tr('Due dates of the monthly obligations by the RUC last digit.|Vencimientos de las obligaciones mensuales por último dígito del RUC.')}
    </div>
    <Button color="green" icon="icon-[fa--plus]" name="New period|Nuevo periodo"
      label="Opens the modal to transcribe the due dates of a tax period.|Abrir el formulario para transcribir los vencimientos de un periodo tributario." onClick={() => openPeriod()} />
  </div>
  <VTable {columns} data={taxCalendar.calendars} css="w-full" onRowClick={(calendar) => openPeriod(calendar)} />
</Page>

<Modal id={CALENDAR_MODAL_ID} size={7} bodyCss="px-16 py-14"
  title={isStoredPeriod ? tr('Edit period|Editar periodo') : tr('New period|Nuevo periodo')}
  onSave={() => savePeriod()}
  onDelete={isStoredPeriod ? () => savePeriod(true) : undefined}
>
  <div class="grid grid-cols-24 gap-10" aria-label="Tax period due dates form">
    <Input bind:saveOn={calendarForm} save="Period" type="number" label="Period (YYMM)|Periodo (AAMM)" required={true}
      css="col-span-24 md:col-span-8" disabled={isStoredPeriod} />
    <div class="col-span-24 md:col-span-16 self-end text-fg-muted pb-6">
      {#if calendarForm.Period % 100 >= 1 && calendarForm.Period % 100 <= 12}{formatTaxPeriod(calendarForm.Period)}{/if}
    </div>
    {#each TAX_DUE_COLUMN_LABELS as label, column}
      <DateInput bind:saveOn={calendarForm.DueDays} save={column} required={true}
        label={column < 10 ? `${tr('RUC ending in|RUC terminado en')} ${column}` : label}
        css="col-span-12 md:col-span-6" />
    {/each}
  </div>
</Modal>
