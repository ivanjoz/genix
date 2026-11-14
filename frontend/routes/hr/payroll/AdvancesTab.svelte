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
  import { employeeFullName, payrollEmployees } from '#routes/hr/employees/employees.ts'
  import type { EmployeesService } from '#services/hr/employees.svelte.ts'
  import { EmployeeRecordsYearService, postPayAdvances } from '#services/hr/employee-records.svelte.ts'
  import {
    RecordStatus, isLiveRecord, yearOfPeriod, type IEmployeeRecord,
  } from '#services/hr/employee-records.ts'
  import { ConceptGroup, LegalConcept, legalConceptsOfGroup } from '#services/hr/hr-catalogs.ts'
  import { CajasService } from '#routes/finance/cash-banks/cajas.svelte.ts'
  import {
    formatCents, formatPeriod, nextPeriod, payrollCashBankOptions, periodFirstDay, periodOfDay, previousPeriod,
  } from './payroll.ts'

  // Weekly / biweekly payments made before the monthly payslip, which nets them (plan §3.6). They
  // are listed by the period they settle against (the month of the last day they cover; they cover
  // one month); a payment is voided, never edited.
  let { employees }: { employees: EmployeesService } = $props()

  const ui = useUI()
  const ADVANCE_MODAL_ID = 33
  const advanceTypeOptions = legalConceptsOfGroup(ConceptGroup.Advance).map((concept) => ({ id: concept.id, name: tr(concept.name) }))
  const advanceTypeNames = new Map(advanceTypeOptions.map((option) => [option.id, option.name]))

  const currentPeriod = periodOfDay(todayUnixDay())
  const periodOptions = Array.from({ length: 13 }, (_, index) => {
    let period = nextPeriod(currentPeriod)
    for (let step = 0; step < index; step++) period = previousPeriod(period)
    return { id: period, name: formatPeriod(period) }
  })

  let selected = $state({ Period: currentPeriod })
  // The year's employee records, of which this tab keeps the advances of the selected month.
  let yearRecords = $state(new EmployeeRecordsYearService(yearOfPeriod(currentPeriod)))
  let advanceForm = $state({} as IEmployeeRecord)
  const cajas = new CajasService()
  const cashBankOptions = $derived(payrollCashBankOptions(cajas.Cajas))

  // Advances are paid back through the payslip: Planilla workers only (plan §9.5).
  const employeeOptions = $derived(payrollEmployees(employees.records).filter((employee) => employee.ss === 1)
    .map((employee) => ({ ID: employee.ID, Name: employeeFullName(employee) })))
  const visibleAdvances = $derived(yearRecords.records.filter((record) =>
    record.Period === selected.Period && advanceTypeNames.has(record.ConceptID) && isLiveRecord(record)))

  // A month of another year switches to that year's cached list.
  const selectPeriod = (period: number) => {
    if (yearRecords.route !== `hr-employee-records?year=${yearOfPeriod(period)}`) {
      yearRecords = new EmployeeRecordsYearService(yearOfPeriod(period))
    }
  }

  const openNewAdvance = () => {
    const today = todayUnixDay()
    const coveredFrom = Math.max(today - 6, periodFirstDay(periodOfDay(today)))
    advanceForm = { ConceptID: LegalConcept.AdvanceWeekly, Date: today, FromDate: coveredFrom, ToDate: today,
      ss: RecordStatus.Active } as IEmployeeRecord
    ui.openModal(ADVANCE_MODAL_ID)
  }

  const saveAdvance = async () => {
    showLoading(tr('Saving...|Guardando...'))
    try {
      const saved = await postPayAdvances([advanceForm])
      yearRecords.addSavedRecords(...saved.filter((advance) => advance.Year === yearOfPeriod(selected.Period)))
      ui.closeModal(ADVANCE_MODAL_ID)
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }

  const voidAdvance = async (advance: IEmployeeRecord) => {
    if (advance.ss !== RecordStatus.Active) return
    const name = employeeOptions.find((option) => option.ID === advance.EmployeeID)?.Name || ''
    if (!(await confirmWarn({ title: tr('Void payment|Anular pago'),
      message: tr(`Void the payment of S/ ${formatCents(advance.Amount)} to ${name}? The money returns to the account that paid it.|¿Anular el pago de S/ ${formatCents(advance.Amount)} a ${name}? El dinero vuelve a la cuenta que lo pagó.`) }))) return
    showLoading(tr('Voiding...|Anulando...'))
    try {
      const saved = await postPayAdvances([{ ...advance, ss: RecordStatus.Cancelled }])
      yearRecords.addSavedRecords(...saved)
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }

  const columns: ITableColumn<IEmployeeRecord>[] = [
    { header: 'Employee|Trabajador', css: 'px-6', highlight: true,
      getValue: (e) => employeeOptions.find((option) => option.ID === e.EmployeeID)?.Name || `#${e.EmployeeID}` },
    { header: 'Type|Tipo', css: 'px-6', getValue: (e) => advanceTypeNames.get(e.ConceptID) || '' },
    { header: 'Covers|Cubre', css: 'px-6',
      getValue: (e) => `${formatTime(e.FromDate, 'd-m-Y')} → ${formatTime(e.ToDate, 'd-m-Y')}` },
    { header: 'Paid on|Pagado el', css: 'px-6', getValue: (e) => String(formatTime(e.Date, 'd-m-Y')) },
    { header: 'Amount|Monto', css: 'px-6 text-right', getValue: (e) => formatCents(e.Amount) },
    { header: 'Status|Estado', css: 'px-6',
      getValue: (e) => e.ss === RecordStatus.Settled ? tr('Settled in payslip|Descontado en boleta') : tr('Paid|Pagado') },
  ]
</script>

<div class="flex flex-wrap items-end gap-10 mb-10" aria-label="Weekly payments">
  <SearchSelect bind:saveOn={selected} save="Period" label="Settles in period|Se descuenta en" options={periodOptions}
    keyId="id" keyName="name" css="w-200"
    onChange={(option) => { if (option) selectPeriod(option.id) }} />
  <div class="text-gray-600 mr-auto">
    {tr('Payments made before the monthly payslip, which deducts them.|Pagos hechos antes de la boleta mensual, que los descuenta.')}
  </div>
  <Button color="green" icon="icon-[fa--plus]" name="Register payment|Registrar pago"
    label="Opens the modal to register a weekly or biweekly payment." onClick={openNewAdvance} />
</div>
<VTable {columns} data={visibleAdvances} css="w-full" onRowClick={voidAdvance} />

<Modal id={ADVANCE_MODAL_ID} size={5} bodyCss="px-16 py-14" title={tr('Register payment|Registrar pago')}
  onSave={saveAdvance}
>
  <div class="grid grid-cols-24 gap-10" aria-label="Payment form">
    <SearchSelect bind:saveOn={advanceForm} save="EmployeeID" label="Employee|Trabajador" required={true}
      options={employeeOptions} keyId="ID" keyName="Name" css="col-span-24" />
    <SearchSelect bind:saveOn={advanceForm} save="ConceptID" label="Type|Tipo" required={true}
      options={advanceTypeOptions} keyId="id" keyName="name" css="col-span-24 md:col-span-12" />
    <Input bind:saveOn={advanceForm} save="Amount" type="number" baseDecimals={2} label="Amount (S/)|Monto (S/)"
      required={true} css="col-span-24 md:col-span-12" />
    <DateInput bind:saveOn={advanceForm} save="FromDate" label="Covers from|Cubre desde" required={true} css="col-span-24 md:col-span-8" />
    <DateInput bind:saveOn={advanceForm} save="ToDate" label="Covers to|Cubre hasta" required={true} css="col-span-24 md:col-span-8" />
    <DateInput bind:saveOn={advanceForm} save="Date" label="Paid on|Pagado el" required={true} css="col-span-24 md:col-span-8" />
    <SearchSelect bind:saveOn={advanceForm} save="CashBankID" label="Paid from|Pagado desde" required={true}
      options={cashBankOptions} keyId="ID" keyName="Name" css="col-span-24 md:col-span-12" />
    <Input bind:saveOn={advanceForm} save="Note" label="Note|Nota" css="col-span-24 md:col-span-12" />
  </div>
</Modal>
