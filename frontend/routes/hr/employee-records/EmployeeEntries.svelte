<script lang="ts">
  import { useUI } from '@genix/ui'
  import { confirmWarn, hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
  import Button from '#components/buttons/Button.svelte'
  import DateInput from '#components/form/DateInput.svelte'
  import Input from '#components/form/Input.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import Modal from '#components/layers/Modal.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import { formatTime, todayUnixDay } from '#libs/helpers.ts'
  import { conceptName, formatCents, formatPeriodMonth, periodOfDay } from '#routes/hr/payroll/payroll.ts'
  import { postEmployeeRecord, type EmployeeRecordsService } from '#services/hr/employee-records.svelte.ts'
  import { RecordStatus, type IEmployeeRecord } from '#services/hr/employee-records.ts'
  import type { PayrollConfigService } from '#services/hr/payroll.svelte.ts'
  import {
    employeeEntries, entryConceptOptions, entryInput, entryMonthOptions, isEditableRecord, makeEmptyEntry,
  } from './employee-records.ts'

  // The employee's Registros of every month: overtime, absences and company concept amounts (a
  // concept flagged "separate payslip" is an extraordinary payment). Each has the date it happened
  // and the payroll month that pays it (by default the date's month; a later one of the same year
  // for a late report). The payroll is not recalculated here: opening the Payroll page recalculates
  // every open month whose inputs changed. The server refuses writes to a month whose payroll is closed.
  let { employeeID, records, config }: {
    employeeID: number, records: EmployeeRecordsService, config: PayrollConfigService
  } = $props()

  const ui = useUI()
  const ENTRY_MODAL_ID = 31

  let entryForm = $state({} as IEmployeeRecord)
  const entries = $derived(employeeEntries(records.records))
  const conceptOptions = $derived(entryConceptOptions(config.Concepts).map((option) => ({ ...option, Name: tr(option.Name) })))
  const monthOptions = $derived(entryMonthOptions(entryForm.Date || todayUnixDay()))
  const formInput = $derived(entryInput(entryForm.ConceptID))

  const saveEntry = async (isDelete = false) => {
    if (isDelete && !(await confirmWarn({ title: tr('Delete|Eliminar'), message: tr('Delete this record?|¿Eliminar este registro?') }))) return
    showLoading(tr('Saving...|Guardando...'))
    try {
      records.addSavedRecords(await postEmployeeRecord({ ...entryForm,
        ss: isDelete ? RecordStatus.Cancelled : RecordStatus.Active }))
      ui.closeModal(ENTRY_MODAL_ID)
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }

  const columns: ITableColumn<IEmployeeRecord>[] = [
    { header: 'Date|Fecha', css: 'px-6 whitespace-nowrap', highlight: true,
      getValue: (e) => e.Date ? formatTime(e.Date, 'Y-m-d') as string : '' },
    { header: 'Month|Mes', css: 'px-6 text-center', getValue: (e) => formatPeriodMonth(e.Period) },
    { header: 'Concept|Concepto', css: 'px-6', getValue: (e) => tr(conceptName(e.ConceptID, config.ConceptsMap)) },
    { header: 'Detail|Detalle', css: 'px-6',
      getValue: (e) => {
        const input = entryInput(e.ConceptID)
        if (input === 'amount') return `S/ ${formatCents(e.Amount)}`
        return `${e.Quantity} ${input === 'days' ? tr('days|días') : tr('hours|horas')}`
      } },
    { header: 'Note|Nota', css: 'px-6', getValue: (e) => e.Note },
    { header: 'Status|Estado', css: 'px-6',
      getValue: (e) => e.ss === RecordStatus.Settled ? tr('Settled|Liquidado') : tr('Open|Abierto') },
  ]
</script>

<div class="mt-12" aria-label="Employee records">
  <div class="flex items-center justify-between mb-8">
    <div class="text-gray-600">
      {tr('Overtime, absences, bonuses and other concepts of the employee.|Horas extras, inasistencias, bonos y otros conceptos del trabajador.')}
    </div>
    <Button color="green" icon="icon-[fa--plus]" name="Add|Agregar" label="Opens the modal to add a record to the employee."
      onClick={() => { entryForm = makeEmptyEntry(employeeID, todayUnixDay()); ui.openModal(ENTRY_MODAL_ID) }} />
  </div>
  <VTable {columns} data={entries} css="w-full"
    onRowClick={(entry) => { if (isEditableRecord(entry)) { entryForm = { ...entry }; ui.openModal(ENTRY_MODAL_ID) } }} />
</div>

<Modal id={ENTRY_MODAL_ID} size={5} bodyCss="px-16 py-14"
  title={entryForm.ID > 0 ? tr('Edit record|Editar registro') : tr('New record|Nuevo registro')}
  onSave={() => saveEntry()}
  onDelete={entryForm.ID > 0 ? () => saveEntry(true) : undefined}
>
  <div class="grid grid-cols-24 gap-10" aria-label="Employee record form">
    <!-- A new date moves the payroll month to the date's month; the user can still pick a later one. -->
    <DateInput bind:saveOn={entryForm} save="Date" label="Date|Fecha" required={true} css="col-span-24 md:col-span-8"
      onChange={() => { if (entryForm.Date) entryForm.Period = periodOfDay(entryForm.Date) }} />
    <SearchSelect bind:saveOn={entryForm} save="Period" label="Payroll month|Mes de planilla" required={true}
      options={monthOptions} keyId="id" keyName="name" css="col-span-24 md:col-span-6" />
    <SearchSelect bind:saveOn={entryForm} save="ConceptID" label="Concept|Concepto" required={true}
      options={conceptOptions} keyId="ID" keyName="Name" css="col-span-24 md:col-span-10" />
    <!-- The concept decides the input: overtime hours, absence days, or a company concept's amount. -->
    {#if formInput === 'amount'}
      <Input bind:saveOn={entryForm} save="Amount" type="number" baseDecimals={2} label="Amount (S/)|Monto (S/)"
        required={true} css="col-span-24 md:col-span-8" />
    {:else if formInput}
      <Input bind:saveOn={entryForm} save="Quantity" type="number"
        label={formInput === 'days' ? 'Days|Días' : 'Hours|Horas'} required={true} css="col-span-24 md:col-span-8" />
    {/if}
    <Input bind:saveOn={entryForm} save="Note" label="Note|Nota" css="col-span-24 {formInput ? 'md:col-span-16' : ''}" />
  </div>
</Modal>
