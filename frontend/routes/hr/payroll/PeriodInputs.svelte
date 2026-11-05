<script lang="ts">
  import { useUI } from '@genix/ui'
  import { confirmWarn, hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
  import Button from '#components/buttons/Button.svelte'
  import Input from '#components/form/Input.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import Modal from '#components/layers/Modal.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import { employeeFullName } from '#routes/hr/employees/employees.ts'
  import type { EmployeesService } from '#services/hr/employees.svelte.ts'
  import { postEmployeeRecord, type EmployeeRecordsYearService } from '#services/hr/employee-records.svelte.ts'
  import { RecordStatus, RecordType, isLiveRecord, type IEmployeeRecord } from '#services/hr/employee-records.ts'
  import { INPUT_TYPE_OPTIONS } from '#services/hr/hr-catalogs.ts'
  import type { IPayrollRun, PayrollConfigService } from '#services/hr/payroll.svelte.ts'
  import { ConceptKind, formatCents, runState } from './payroll.ts'

  // The novedades of the run's period: absences, overtime, one-month concept amounts and
  // extraordinary payments. They come from the year's employee records; editable while the run is
  // open, and onChanged recalculates the run.
  let { run, records, employees, config, onChanged }: {
    run: IPayrollRun, records: EmployeeRecordsYearService, employees: EmployeesService, config: PayrollConfigService,
    onChanged: () => Promise<void>
  } = $props()

  const ui = useUI()
  const INPUT_MODAL_ID = 31
  const inputTypeOptions = INPUT_TYPE_OPTIONS.map((option) => ({ ...option, name: tr(option.name) }))
  const inputTypeNames = new Map(INPUT_TYPE_OPTIONS.map((option) => [option.id, tr(option.name)]))
  const earningConcepts = $derived(config.Concepts.filter((concept) => concept.Kind === ConceptKind.Earning))

  let inputForm = $state({} as IEmployeeRecord)
  const isEditable = $derived(runState(run) === 'draft' || runState(run) === 'calculated')
  const employeeOptions = $derived(employees.records.filter((employee) => employee.ss > 0)
    .map((employee) => ({ ID: employee.ID, Name: employeeFullName(employee) })))
  const visibleInputs = $derived(records.records.filter((record) =>
    record.Period === run.Period && inputTypeNames.has(record.Type) && isLiveRecord(record)))
  const isConceptAmount = $derived(inputForm.Type === RecordType.ConceptAmount || inputForm.Type === RecordType.ExtraordinaryPayment)

  const saveInput = async (isDelete = false) => {
    if (isDelete && !(await confirmWarn({ title: tr('Delete|Eliminar'), message: tr('Delete this entry?|¿Eliminar esta novedad?') }))) return
    showLoading(tr('Saving...|Guardando...'))
    try {
      const savedInput = await postEmployeeRecord({ ...inputForm, Period: run.Period,
        ss: isDelete ? RecordStatus.Cancelled : RecordStatus.Active })
      records.addSavedRecords(savedInput)
      ui.closeModal(INPUT_MODAL_ID)
      await onChanged()
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }

  const columns: ITableColumn<IEmployeeRecord>[] = [
    { header: 'Employee|Trabajador', css: 'px-6', highlight: true,
      getValue: (e) => employeeOptions.find((option) => option.ID === e.EmployeeID)?.Name || `#${e.EmployeeID}` },
    { header: 'Entry|Novedad', css: 'px-6', getValue: (e) => inputTypeNames.get(e.Type) || '' },
    { header: 'Detail|Detalle', css: 'px-6',
      getValue: (e) => e.ConceptID > 0
        ? `${config.ConceptsMap.get(e.ConceptID)?.Name || ''}: S/ ${formatCents(e.Amount)}` : String(e.Quantity) },
    { header: 'Status|Estado', css: 'px-6',
      getValue: (e) => e.ss === RecordStatus.Settled ? tr('Settled|Liquidada') : tr('Open|Abierta') },
  ]
</script>

<div class="flex items-center justify-between mb-8">
  <div class="text-gray-600">
    {tr('Absences, overtime, one-off amounts and extraordinary payments of the month.|Inasistencias, horas extras, montos y pagos extraordinarios del mes.')}
  </div>
  {#if isEditable}
    <Button color="green" icon="icon-[fa--plus]" name="Add|Agregar" label="Opens the modal to add a payroll entry."
      onClick={() => { inputForm = { Type: RecordType.ConceptAmount, ss: RecordStatus.Active } as IEmployeeRecord; ui.openModal(INPUT_MODAL_ID) }} />
  {/if}
</div>
<VTable {columns} data={visibleInputs} css="w-full"
  onRowClick={(input) => { if (isEditable && input.ss === RecordStatus.Active) { inputForm = { ...input }; ui.openModal(INPUT_MODAL_ID) } }} />

<Modal id={INPUT_MODAL_ID} size={5} bodyCss="px-16 py-14"
  title={inputForm.ID > 0 ? tr('Edit entry|Editar novedad') : tr('New entry|Nueva novedad')}
  onSave={() => saveInput()}
  onDelete={inputForm.ID > 0 ? () => saveInput(true) : undefined}
>
  <div class="grid grid-cols-24 gap-10" aria-label="Payroll entry form">
    <SearchSelect bind:saveOn={inputForm} save="EmployeeID" label="Employee|Trabajador" required={true}
      disabled={inputForm.ID > 0} options={employeeOptions} keyId="ID" keyName="Name" css="col-span-24" />
    <SearchSelect bind:saveOn={inputForm} save="Type" label="Entry type|Tipo de novedad" required={true}
      options={inputTypeOptions} keyId="id" keyName="name" css="col-span-24" />
    {#if isConceptAmount}
      <SearchSelect bind:saveOn={inputForm} save="ConceptID" label="Concept|Concepto" required={true}
        options={inputForm.Type === RecordType.ExtraordinaryPayment ? earningConcepts : config.Concepts}
        keyId="ID" keyName="Name" css="col-span-24 md:col-span-14" />
      <Input bind:saveOn={inputForm} save="Amount" type="number" baseDecimals={2} label="Amount (S/)|Monto (S/)"
        required={true} css="col-span-24 md:col-span-10" />
    {:else}
      <Input bind:saveOn={inputForm} save="Quantity" type="number"
        label={inputForm.Type === RecordType.AbsenceDays ? 'Days|Días' : 'Hours|Horas'} required={true} css="col-span-24 md:col-span-10" />
    {/if}
    <Input bind:saveOn={inputForm} save="Note" label="Note|Nota" css="col-span-24" />
  </div>
</Modal>
