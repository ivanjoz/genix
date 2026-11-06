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
  import { todayUnixDay } from '#libs/helpers.ts'
  import { CajasService } from '#routes/finance/cash-banks/cajas.svelte.ts'
  import {
    formatCents, formatPeriod, payrollCashBankOptions, periodOfDay, periodOptionsAround,
  } from '#routes/hr/payroll/payroll.ts'
  import { postEmployeeRecord, postLoan, type EmployeeRecordsService } from '#services/hr/employee-records.svelte.ts'
  import { RecordStatus, type IEmployeeRecord } from '#services/hr/employee-records.ts'
  import { LegalConcept } from '#services/hr/hr-catalogs.ts'
  import { employeeLoanInstallments, employeeLoans, isEditableRecord } from './employee-records.ts'

  // The employee's loans, recovered from his payslips with their installments (one record each, so
  // one can move to a later month). Granting a loan moves cash out of the chosen cash/bank account.
  let { employeeID, records }: { employeeID: number, records: EmployeeRecordsService } = $props()

  const ui = useUI()
  const LOAN_MODAL_ID = 24
  const INSTALLMENT_MODAL_ID = 25

  const currentPeriod = periodOfDay(todayUnixDay())
  const periodOptions = periodOptionsAround(currentPeriod)
  const cajas = new CajasService()
  const cashBankOptions = $derived(payrollCashBankOptions(cajas.Cajas))
  const loans = $derived(employeeLoans(records.records))
  const installments = $derived(employeeLoanInstallments(records.records))

  let loanForm = $state({} as IEmployeeRecord)
  let installmentForm = $state({} as IEmployeeRecord)

  const saveLoan = async () => {
    showLoading(tr('Saving...|Guardando...'))
    try {
      records.addSavedRecords(...await postLoan({ ...loanForm, ConceptID: LegalConcept.Loan, EmployeeID: employeeID,
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
      records.addSavedRecords(...await postLoan({ ...loan, ss: RecordStatus.Cancelled }))
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }

  // Moves an installment not yet recovered to another month.
  const saveInstallment = async () => {
    showLoading(tr('Saving...|Guardando...'))
    try {
      records.addSavedRecords(await postEmployeeRecord(installmentForm))
      ui.closeModal(INSTALLMENT_MODAL_ID)
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }

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

<div class="mt-12" aria-label="Employee loans">
  <div class="flex items-center justify-between mb-8">
    <div class="h3 ff-bold">{tr('Loans|Préstamos')}</div>
    <Button color="green" icon="icon-[fa--plus]" label="Opens the modal to grant a loan."
      onClick={() => {
        loanForm = { Date: todayUnixDay(), Quantity: 1, Period: currentPeriod, ss: RecordStatus.Active } as IEmployeeRecord
        ui.openModal(LOAN_MODAL_ID)
      }} />
  </div>
  <VTable columns={loanColumns} data={loans} css="w-full" onRowClick={deleteLoan} />
  {#if installments.length > 0}
    <div class="h3 ff-bold mt-16 mb-8">{tr('Installments|Cuotas')}</div>
    <VTable columns={installmentColumns} data={installments} css="w-full"
      onRowClick={(installment) => {
        if (!isEditableRecord(installment)) return
        installmentForm = { ...installment }
        ui.openModal(INSTALLMENT_MODAL_ID)
      }} />
  {/if}
</div>

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
