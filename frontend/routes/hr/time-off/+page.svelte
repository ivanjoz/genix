<script lang="ts">
  import { untrack } from 'svelte'
  import { useUI } from '@genix/ui'
  import { confirmWarn, hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
  import Page from '#domain/Page.svelte'
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
  import { EmployeesService, type IEmployee } from '#services/hr/employees.svelte.ts'
  import { ConceptGroup, LegalConcept, legalConceptsOfGroup } from '#services/hr/hr-catalogs.ts'
  import { EmployeeRecordsYearService, postEmployeeRecord } from '#services/hr/employee-records.svelte.ts'
  import { RecordStatus, type IEmployeeRecord } from '#services/hr/employee-records.ts'
  import { balanceYears, vacationBalance, yearOfDay } from './time-off.ts'

  // Vacations and leaves. HR registers them already approved; the monthly payroll pays vacation
  // days as vacation pay and leaves unpaid days out. Requests and approvals come in phase 7.
  const ui = useUI()
  const TIME_OFF_MODAL_ID = 51
  const today = todayUnixDay()
  const currentYear = yearOfDay(today)

  const employees = new EmployeesService(true)
  // Time off belongs to the payroll: workers off payroll are not listed (plan §9.5).
  const activeEmployees = $derived(payrollEmployees(employees.records).filter((employee) => employee.ss === 1))
  const employeeOptions = $derived(activeEmployees.map((employee) => ({ ID: employee.ID, Name: employeeFullName(employee) })))
  const timeOffTypeOptions = legalConceptsOfGroup(ConceptGroup.TimeOff).map((concept) => ({ id: concept.id, name: tr(concept.name) }))
  const timeOffTypeNames = new Map(timeOffTypeOptions.map((option) => [option.id, option.name]))
  const isTimeOff = (record: IEmployeeRecord) => timeOffTypeNames.has(record.ConceptID)

  // One cached list per year (every record type; this page keeps the time off), and the balance
  // needs every year back to the earliest hire.
  const years = $derived(balanceYears(activeEmployees, currentYear))
  let recordsByYear = $state(new Map<number, EmployeeRecordsYearService>())
  $effect(() => {
    const neededYears = years
    untrack(() => {
      const missing = neededYears.filter((year) => !recordsByYear.has(year))
      if (missing.length === 0) return
      const next = new Map(recordsByYear)
      for (const year of missing) next.set(year, new EmployeeRecordsYearService(year))
      recordsByYear = next
    })
  })
  const allTimeOff = $derived([...recordsByYear.values()].flatMap((service) => service.records).filter(isTimeOff))

  let selected = $state({ Year: currentYear })
  const yearOptions = $derived([...years].reverse().map((year) => ({ id: year, name: String(year) })))
  const yearTimeOff = $derived((recordsByYear.get(selected.Year)?.records || [])
    .filter((leave) => isTimeOff(leave) && leave.ss === RecordStatus.Active)
    .sort((left, right) => right.FromDate - left.FromDate))

  let timeOffForm = $state({} as IEmployeeRecord)

  const saveTimeOff = async (isCancel = false) => {
    if (isCancel && !(await confirmWarn({ title: tr('Cancel time off|Anular ausencia'),
      message: tr('Cancel this time off? Its days go back to the payroll.|¿Anular esta ausencia? Sus días vuelven a la planilla.') }))) return
    showLoading(tr('Saving...|Guardando...'))
    try {
      const saved = await postEmployeeRecord({ ...timeOffForm, ss: isCancel ? RecordStatus.Cancelled : RecordStatus.Active })
      // The record lives in the list of the year it starts in: a change of year moves it.
      for (const service of recordsByYear.values()) {
        if (service.recordsMap.has(saved.ID)) service.addSavedRecords({ ...saved, ss: RecordStatus.Cancelled })
      }
      recordsByYear.get(saved.Year)?.addSavedRecords(saved)
      ui.closeModal(TIME_OFF_MODAL_ID)
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }

  const openNew = () => {
    timeOffForm = { ConceptID: LegalConcept.Vacation, FromDate: today, ToDate: today, ss: RecordStatus.Active } as IEmployeeRecord
    ui.openModal(TIME_OFF_MODAL_ID)
  }

  const employeeName = (employeeID: number) => employeeOptions.find((option) => option.ID === employeeID)?.Name || `#${employeeID}`

  const timeOffColumns: ITableColumn<IEmployeeRecord>[] = [
    { header: 'Employee|Trabajador', css: 'px-6', highlight: true, getValue: (e) => employeeName(e.EmployeeID) },
    { header: 'Type|Tipo', css: 'px-6', getValue: (e) => timeOffTypeNames.get(e.ConceptID) || '' },
    { header: 'From|Desde', css: 'px-6', getValue: (e) => String(formatTime(e.FromDate, 'd-m-Y')) },
    { header: 'To|Hasta', css: 'px-6', getValue: (e) => String(formatTime(e.ToDate, 'd-m-Y')) },
    { header: 'Days|Días', css: 'px-6 text-center', getValue: (e) => e.Quantity },
    { header: 'Note|Nota', css: 'px-6', getValue: (e) => e.Note },
  ]

  const balanceColumns: ITableColumn<IEmployee>[] = [
    { header: 'Employee|Trabajador', css: 'px-6', highlight: true, getValue: (e) => employeeFullName(e) },
    { header: 'Hired|Ingreso', css: 'px-6', getValue: (e) => String(formatTime(e.HireDate, 'd-m-Y')) },
    { header: 'Earned|Ganados', css: 'px-6 text-right', getValue: (e) => vacationBalance(e, allTimeOff, today).earned },
    { header: 'Taken|Gozados', css: 'px-6 text-right', getValue: (e) => vacationBalance(e, allTimeOff, today).taken },
    { header: 'Balance|Saldo', css: 'px-6 text-right', getValue: (e) => vacationBalance(e, allTimeOff, today).balance },
  ]

  const pageOptions = [
    { id: 1, name: 'Time off|Ausencias' },
    { id: 2, name: 'Vacation balances|Saldos de vacaciones' },
  ]
</script>

<Page title="Time Off|Vacaciones y Licencias" options={pageOptions}>
  {#if ui.state.pageOptionSelected === 2}
    <div class="text-gray-600 mb-8">
      {tr('30 days a year, earned day by day since hiring, less the approved vacations.|30 días por año, ganados día a día desde el ingreso, menos las vacaciones aprobadas.')}
    </div>
    <VTable columns={balanceColumns} data={activeEmployees} css="w-full" />
  {:else}
    <div class="flex flex-wrap items-end gap-10 mb-10" aria-label="Time off">
      <SearchSelect bind:saveOn={selected} save="Year" label="Starting in|Que empiezan en" options={yearOptions}
        keyId="id" keyName="name" css="w-160" />
      <Button color="green" icon="icon-[fa--plus]" name="Register|Registrar" css="ml-auto"
        label="Opens the modal to register a vacation or leave." onClick={openNew} />
    </div>
    <VTable columns={timeOffColumns} data={yearTimeOff} css="w-full"
      onRowClick={(leave) => { timeOffForm = { ...leave }; ui.openModal(TIME_OFF_MODAL_ID) }} />
  {/if}
</Page>

<Modal id={TIME_OFF_MODAL_ID} size={5} bodyCss="px-16 py-14"
  title={timeOffForm.ID > 0 ? tr('Edit time off|Editar ausencia') : tr('Register time off|Registrar ausencia')}
  onSave={() => saveTimeOff()}
  onDelete={timeOffForm.ID > 0 ? () => saveTimeOff(true) : undefined}
>
  <div class="grid grid-cols-24 gap-10" aria-label="Time off form">
    <SearchSelect bind:saveOn={timeOffForm} save="EmployeeID" label="Employee|Trabajador" required={true}
      disabled={timeOffForm.ID > 0} options={employeeOptions} keyId="ID" keyName="Name" css="col-span-24" />
    <SearchSelect bind:saveOn={timeOffForm} save="ConceptID" label="Type|Tipo" required={true}
      options={timeOffTypeOptions} keyId="id" keyName="name" css="col-span-24" />
    <DateInput bind:saveOn={timeOffForm} save="FromDate" label="From|Desde" required={true} css="col-span-24 md:col-span-12" />
    <DateInput bind:saveOn={timeOffForm} save="ToDate" label="To (included)|Hasta (incluido)" required={true} css="col-span-24 md:col-span-12" />
    <Input bind:saveOn={timeOffForm} save="Note" label="Note|Nota" css="col-span-24" />
  </div>
</Modal>
