<script lang="ts">
  import { untrack } from 'svelte'
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
  import { todayUnixDay } from '#libs/helpers.ts'
  import { CajasService } from '#routes/finance/cash-banks/cajas.svelte.ts'
  import {
    formatCents, formatPeriod, nextPeriod, payrollCashBankOptions, periodOfDay, previousPeriod,
  } from '#routes/hr/payroll/payroll.ts'
  import {
    EmployeeConceptsService, PayrollConfigService, postEmployeeConcept, type IEmployeeConcept,
  } from '#services/hr/payroll.svelte.ts'
  import { EmployeeRecordsService, postEmployeeRecord, postLoan } from '#services/hr/employee-records.svelte.ts'
  import { RecordStatus, RecordType, isLiveRecord, type IEmployeeRecord } from '#services/hr/employee-records.ts'

  // What an employee is paid or deducted every month besides the salary (recurring concepts), and
  // the loans recovered from his payslips with their installments (one record each, so one can move
  // to a later month). Payroll access only. The parent re-creates this component per employee
  // ({#key}), so the services are built once here.
  let { employeeID }: { employeeID: number } = $props()

  const ui = useUI()
  const RECURRING_MODAL_ID = 23
  const LOAN_MODAL_ID = 24
  const INSTALLMENT_MODAL_ID = 25

  const config = new PayrollConfigService(true)
  const recurringConcepts = new EmployeeConceptsService(untrack(() => employeeID))
  const employeeRecords = new EmployeeRecordsService(untrack(() => employeeID))
  const loans = $derived(employeeRecords.records.filter((record) => record.Type === RecordType.Loan && isLiveRecord(record)))
  const installments = $derived(employeeRecords.records
    .filter((record) => record.Type === RecordType.LoanInstallment && isLiveRecord(record))
    .sort((left, right) => left.Period - right.Period))
  const cajas = new CajasService()
  const cashBankOptions = $derived(payrollCashBankOptions(cajas.Cajas))

  // Periods offered: a year back to a year ahead.
  const periodOptions = $derived.by(() => {
    let period = periodOfDay(todayUnixDay())
    for (let i = 0; i < 12; i++) period = previousPeriod(period)
    return Array.from({ length: 25 }, () => {
      const option = { id: period, name: formatPeriod(period) }
      period = nextPeriod(period)
      return option
    })
  })

  let recurringForm = $state({} as IEmployeeConcept)
  let loanForm = $state({} as IEmployeeRecord)
  let installmentForm = $state({} as IEmployeeRecord)

  const saveRecurring = async (isDelete = false) => {
    if (isDelete && !(await confirmWarn({ title: tr('Delete|Eliminar'), message: tr('Delete this concept?|¿Eliminar este concepto?') }))) return
    showLoading(tr('Saving...|Guardando...'))
    try {
      recurringConcepts.addSavedRecords(await postEmployeeConcept({ ...recurringForm, EmployeeID: employeeID, ss: isDelete ? 0 : 1 }))
      ui.closeModal(RECURRING_MODAL_ID)
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }

  const saveLoan = async () => {
    showLoading(tr('Saving...|Guardando...'))
    try {
      employeeRecords.addSavedRecords(...await postLoan({ ...loanForm, Type: RecordType.Loan, EmployeeID: employeeID,
        ss: RecordStatus.Active }))
      ui.closeModal(LOAN_MODAL_ID)
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }

  const deleteLoan = async (loan: IEmployeeRecord) => {
    if (!(await confirmWarn({ title: tr('Delete loan|Eliminar préstamo'),
      message: tr('Delete this loan? Only possible before its first installment is closed.|¿Eliminar este préstamo? Solo es posible antes de cerrar su primera cuota.') }))) return
    showLoading(tr('Deleting...|Eliminando...'))
    try {
      employeeRecords.addSavedRecords(...await postLoan({ ...loan, ss: RecordStatus.Cancelled }))
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }

  // Moves an installment not yet recovered to another month.
  const saveInstallment = async () => {
    showLoading(tr('Saving...|Guardando...'))
    try {
      employeeRecords.addSavedRecords(await postEmployeeRecord(installmentForm))
      ui.closeModal(INSTALLMENT_MODAL_ID)
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }

  const recurringColumns: ITableColumn<IEmployeeConcept>[] = [
    { header: 'Concept|Concepto', css: 'px-6', highlight: true, getValue: (e) => config.ConceptsMap.get(e.ConceptID)?.Name || '' },
    { header: 'Monthly|Mensual', css: 'px-6 text-right', getValue: (e) => formatCents(e.Amount) },
    { header: 'From|Desde', css: 'px-6', getValue: (e) => formatPeriod(e.FromPeriod) },
    { header: 'To|Hasta', css: 'px-6', getValue: (e) => e.ToPeriod ? formatPeriod(e.ToPeriod) : '' },
  ]
  const loanColumns: ITableColumn<IEmployeeRecord>[] = [
    { header: 'Amount|Monto', css: 'px-6 text-right', highlight: true, getValue: (e) => formatCents(e.Amount) },
    { header: 'Installments|Cuotas', css: 'px-6 text-center', getValue: (e) => e.Quantity },
    { header: 'First installment|Primera cuota', css: 'px-6', getValue: (e) => formatPeriod(e.Period) },
    { header: 'Note|Nota', css: 'px-6', getValue: (e) => e.Note },
  ]
  const installmentColumns: ITableColumn<IEmployeeRecord>[] = [
    { header: 'Month|Mes', css: 'px-6', highlight: true, getValue: (e) => formatPeriod(e.Period) },
    { header: 'Amount|Monto', css: 'px-6 text-right', getValue: (e) => formatCents(e.Amount) },
    { header: 'Status|Estado', css: 'px-6',
      getValue: (e) => e.ss === RecordStatus.Settled ? tr('Recovered|Descontada') : tr('Pending|Pendiente') },
  ]
</script>

<div class="mt-12" aria-label="Employee pay items">
  <div class="flex items-center justify-between mb-8">
    <div class="h3 ff-bold">{tr('Fixed concepts|Conceptos fijos')}</div>
    <Button color="green" icon="icon-[fa--plus]" label="Opens the modal to add a recurring concept."
      onClick={() => { recurringForm = { FromPeriod: periodOfDay(todayUnixDay()), ss: 1 } as IEmployeeConcept; ui.openModal(RECURRING_MODAL_ID) }} />
  </div>
  <VTable columns={recurringColumns} data={recurringConcepts.records.filter((record) => record.ss > 0)} css="w-full"
    onRowClick={(record) => { recurringForm = { ...record }; ui.openModal(RECURRING_MODAL_ID) }} />

  <div class="flex items-center justify-between mt-20 mb-8">
    <div class="h3 ff-bold">{tr('Loans|Préstamos')}</div>
    <Button color="green" icon="icon-[fa--plus]" label="Opens the modal to grant a loan."
      onClick={() => {
        loanForm = { Date: todayUnixDay(), Quantity: 1, Period: periodOfDay(todayUnixDay()), ss: RecordStatus.Active } as IEmployeeRecord
        ui.openModal(LOAN_MODAL_ID)
      }} />
  </div>
  <VTable columns={loanColumns} data={loans} css="w-full" onRowClick={deleteLoan} />
  {#if installments.length > 0}
    <div class="h3 ff-bold mt-16 mb-8">{tr('Installments|Cuotas')}</div>
    <VTable columns={installmentColumns} data={installments} css="w-full"
      onRowClick={(installment) => {
        if (installment.ss !== RecordStatus.Active) return
        installmentForm = { ...installment }
        ui.openModal(INSTALLMENT_MODAL_ID)
      }} />
  {/if}
</div>

<Modal id={RECURRING_MODAL_ID} size={5} bodyCss="px-16 py-14"
  title={recurringForm.ID > 0 ? tr('Edit fixed concept|Editar concepto fijo') : tr('New fixed concept|Nuevo concepto fijo')}
  onSave={() => saveRecurring()}
  onDelete={recurringForm.ID > 0 ? () => saveRecurring(true) : undefined}
>
  <div class="grid grid-cols-24 gap-10" aria-label="Fixed concept form">
    <SearchSelect bind:saveOn={recurringForm} save="ConceptID" label="Concept|Concepto" required={true}
      options={config.Concepts} keyId="ID" keyName="Name" css="col-span-24 md:col-span-14" />
    <Input bind:saveOn={recurringForm} save="Amount" type="number" baseDecimals={2} label="Monthly amount (S/)|Monto mensual (S/)"
      required={true} css="col-span-24 md:col-span-10" />
    <SearchSelect bind:saveOn={recurringForm} save="FromPeriod" label="From|Desde" required={true}
      options={periodOptions} keyId="id" keyName="name" css="col-span-24 md:col-span-12" />
    <SearchSelect bind:saveOn={recurringForm} save="ToPeriod" label="To (optional)|Hasta (opcional)"
      options={periodOptions} keyId="id" keyName="name" css="col-span-24 md:col-span-12" />
    <Input bind:saveOn={recurringForm} save="Note" label="Note|Nota" css="col-span-24" />
  </div>
</Modal>

<Modal id={LOAN_MODAL_ID} size={5} bodyCss="px-16 py-14" title={tr('New loan|Nuevo préstamo')} onSave={saveLoan}>
  <div class="grid grid-cols-24 gap-10" aria-label="Loan form">
    <Input bind:saveOn={loanForm} save="Amount" type="number" baseDecimals={2} label="Amount (S/)|Monto (S/)" required={true}
      css="col-span-24 md:col-span-12" />
    <Input bind:saveOn={loanForm} save="Quantity" type="number" label="Installments|Cuotas" required={true}
      css="col-span-24 md:col-span-12" />
    <DateInput bind:saveOn={loanForm} save="Date" label="Grant date|Fecha del préstamo" required={true}
      css="col-span-24 md:col-span-12" />
    <SearchSelect bind:saveOn={loanForm} save="Period" label="First installment|Primera cuota" required={true}
      options={periodOptions} keyId="id" keyName="name" css="col-span-24 md:col-span-12" />
    <SearchSelect bind:saveOn={loanForm} save="CashBankID" label="Handed out from|Entregado desde" required={true}
      options={cashBankOptions} keyId="ID" keyName="Name" css="col-span-24" />
    <Input bind:saveOn={loanForm} save="Note" label="Note|Nota" css="col-span-24" />
  </div>
</Modal>

<Modal id={INSTALLMENT_MODAL_ID} size={4} bodyCss="px-16 py-14" title={tr('Move installment|Mover cuota')} onSave={saveInstallment}>
  <div class="grid grid-cols-24 gap-10" aria-label="Installment form">
    <SearchSelect bind:saveOn={installmentForm} save="Period" label="Recovered in|Se descuenta en" required={true}
      options={periodOptions} keyId="id" keyName="name" css="col-span-24" />
  </div>
</Modal>
