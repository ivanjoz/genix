<script lang="ts">
  import { untrack } from 'svelte'
  import { useUI } from '@genix/ui'
  import { confirmWarn, hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
  import Button from '#components/buttons/Button.svelte'
  import Input from '#components/form/Input.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import Modal from '#components/layers/Modal.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import { todayUnixDay } from '#libs/helpers.ts'
  import { formatCents, formatPeriod, periodOfDay, periodOptionsAround } from '#routes/hr/payroll/payroll.ts'
  import {
    EmployeeConceptsService, PayrollConfigService, postEmployeeConcept, type IEmployeeConcept,
  } from '#services/hr/payroll.svelte.ts'

  // What an employee is paid or deducted every month besides the salary (fixed concepts). Payroll
  // access only. The month-to-month records and the loans live in Employee Records. The parent
  // re-creates this component per employee ({#key}), so the services are built once here.
  let { employeeID }: { employeeID: number } = $props()

  const ui = useUI()
  const FIXED_CONCEPT_MODAL_ID = 23

  const config = new PayrollConfigService(true)
  const fixedConcepts = new EmployeeConceptsService(untrack(() => employeeID))
  const periodOptions = periodOptionsAround(periodOfDay(todayUnixDay()))

  let fixedConceptForm = $state({} as IEmployeeConcept)

  const saveFixedConcept = async (isDelete = false) => {
    if (isDelete && !(await confirmWarn({ title: tr('Delete|Eliminar'), message: tr('Delete this concept?|¿Eliminar este concepto?') }))) return
    showLoading(tr('Saving...|Guardando...'))
    try {
      fixedConcepts.addSavedRecords(await postEmployeeConcept({ ...fixedConceptForm, EmployeeID: employeeID, ss: isDelete ? 0 : 1 }))
      ui.closeModal(FIXED_CONCEPT_MODAL_ID)
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }

  const fixedConceptColumns: ITableColumn<IEmployeeConcept>[] = [
    { header: 'Concept|Concepto', css: 'px-6', highlight: true, getValue: (e) => config.ConceptsMap.get(e.ConceptID)?.Name || '' },
    { header: 'Monthly|Mensual', css: 'px-6 text-right', getValue: (e) => formatCents(e.Amount) },
    { header: 'From|Desde', css: 'px-6', getValue: (e) => formatPeriod(e.FromPeriod) },
    { header: 'To|Hasta', css: 'px-6', getValue: (e) => e.ToPeriod ? formatPeriod(e.ToPeriod) : '' },
  ]
</script>

<div class="mt-12" aria-label="Employee fixed concepts">
  <div class="flex items-center justify-between mb-8">
    <div class="h3 ff-bold">{tr('Fixed concepts|Conceptos fijos')}</div>
    <Button color="green" icon="icon-[fa--plus]" label="Opens the modal to add a fixed concept."
      onClick={() => { fixedConceptForm = { FromPeriod: periodOfDay(todayUnixDay()), ss: 1 } as IEmployeeConcept; ui.openModal(FIXED_CONCEPT_MODAL_ID) }} />
  </div>
  <VTable columns={fixedConceptColumns} data={fixedConcepts.records.filter((record) => record.ss > 0)} css="w-full"
    onRowClick={(record) => { fixedConceptForm = { ...record }; ui.openModal(FIXED_CONCEPT_MODAL_ID) }} />
</div>

<Modal id={FIXED_CONCEPT_MODAL_ID} size={5} bodyCss="px-16 py-14"
  title={fixedConceptForm.ID > 0 ? tr('Edit fixed concept|Editar concepto fijo') : tr('New fixed concept|Nuevo concepto fijo')}
  onSave={() => saveFixedConcept()}
  onDelete={fixedConceptForm.ID > 0 ? () => saveFixedConcept(true) : undefined}
>
  <div class="grid grid-cols-24 gap-10" aria-label="Fixed concept form">
    <SearchSelect bind:saveOn={fixedConceptForm} save="ConceptID" label="Concept|Concepto" required={true}
      options={config.Concepts} keyId="ID" keyName="Name" css="col-span-24 md:col-span-14" />
    <Input bind:saveOn={fixedConceptForm} save="Amount" type="number" baseDecimals={2} label="Monthly amount (S/)|Monto mensual (S/)"
      required={true} css="col-span-24 md:col-span-10" />
    <SearchSelect bind:saveOn={fixedConceptForm} save="FromPeriod" label="From|Desde" required={true}
      options={periodOptions} keyId="id" keyName="name" css="col-span-24 md:col-span-12" />
    <SearchSelect bind:saveOn={fixedConceptForm} save="ToPeriod" label="To (optional)|Hasta (opcional)"
      options={periodOptions} keyId="id" keyName="name" css="col-span-24 md:col-span-12" />
    <Input bind:saveOn={fixedConceptForm} save="Note" label="Note|Nota" css="col-span-24" />
  </div>
</Modal>
