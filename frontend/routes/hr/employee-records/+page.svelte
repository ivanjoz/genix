<script lang="ts">
  import { useUI } from '@genix/ui'
  import Checkbox from '#components/form/Checkbox.svelte'
  import FilterInput from '#components/form/FilterInput.svelte'
  import Layer from '#components/layers/Layer.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import Page from '#domain/Page.svelte'
  import { employeeFullName, isTerminated } from '#routes/hr/employees/employees.ts'
  import { EmployeeRecordsService } from '#services/hr/employee-records.svelte.ts'
  import { EmployeesService, type IEmployee } from '#services/hr/employees.svelte.ts'
  import { PayrollConfigService } from '#services/hr/payroll.svelte.ts'
  import EmployeeEntries from './EmployeeEntries.svelte'
  import EmployeeLoans from './EmployeeLoans.svelte'

  // Employee first: pick a worker, then record his Registros and loans. The month-to-month records
  // live here; the contract and fixed concepts live in Employees & Contracts.
  const ui = useUI()
  const RECORDS_LAYER_ID = 1
  const VIEW_ENTRIES = 1
  const VIEW_LOANS = 2

  const employees = new EmployeesService(true)
  const config = new PayrollConfigService(true)

  let filterText = $state('')
  let showTerminated = $state(false)
  let selectedEmployee = $state<IEmployee>()
  /** All records of the selected employee: one cached list feeds both the Registros and the loans. */
  let employeeRecords = $state<EmployeeRecordsService>()
  let layerView = $state(VIEW_ENTRIES)

  // Terminated workers stay reachable behind the checkbox: they can still have a pending installment.
  const shownEmployees = $derived(employees.records.filter((employee) =>
    employee.ss > 0 && (showTerminated || !isTerminated(employee))))

  const openEmployee = (employee: IEmployee) => {
    selectedEmployee = employee
    if (employeeRecords?.route !== `hr-employee-records?employee-id=${employee.ID}`) {
      employeeRecords = new EmployeeRecordsService(employee.ID)
    }
    ui.openSideLayer(RECORDS_LAYER_ID)
  }

  const columns: ITableColumn<IEmployee>[] = [
    { header: 'ID', headerCss: 'w-54', css: 'text-center ff-bold', getValue: (e) => e.ID },
    { header: 'Name|Nombre', css: 'px-8', highlight: true, getValue: (e) => employeeFullName(e) },
    { header: 'Document|Documento', headerCss: 'w-120', css: 'px-6', getValue: (e) => e.DocumentNumber },
    { header: 'Status|Estado', headerCss: 'w-100', css: 'px-6',
      getValue: (e) => tr(isTerminated(e) ? 'Terminated|Cesado' : 'Active|Activo') },
  ]
</script>

<Page title="Employee Records|Registros Empleados">
  <Layer type="content">
    <div class="flex items-center gap-16 mb-6 flex-wrap" aria-label="Employees toolbar with filter and terminated toggle">
      <FilterInput bind:value={filterText} css="w-256" />
      <Checkbox label="Show terminated|Mostrar cesados" checked={showTerminated}
        onToggle={(isChecked) => { showTerminated = isChecked }} />
    </div>
    <VTable
      {columns}
      data={shownEmployees}
      css="w-full"
      maxHeight="calc(100vh - 8rem - 16px)"
      {filterText}
      getFilterContent={(e) => [employeeFullName(e), e.DocumentNumber].join(' ').toLowerCase()}
      selected={selectedEmployee?.ID}
      isSelected={(employee, selectedID) => employee.ID === selectedID}
      onRowClick={openEmployee}
    />
  </Layer>

  <Layer
    id={RECORDS_LAYER_ID}
    type="side"
    sideLayerSize={820}
    title={selectedEmployee ? employeeFullName(selectedEmployee) : ''}
    titleIcon="icon-[fa--list-alt] w-20 h-20 mb-6 c-purple"
    titleCss="h2 mb-6"
    options={[[VIEW_ENTRIES, 'Records|Registros'], [VIEW_LOANS, 'Loans|Préstamos']]}
    bind:selected={layerView}
    css="px-12 py-10"
    contentCss="px-0"
    onClose={() => { selectedEmployee = undefined }}
  >
    {#if selectedEmployee && employeeRecords}
      {#if layerView === VIEW_ENTRIES}
        <EmployeeEntries employeeID={selectedEmployee.ID} records={employeeRecords} {config} />
      {:else}
        <EmployeeLoans employeeID={selectedEmployee.ID} records={employeeRecords} />
      {/if}
    {/if}
  </Layer>
</Page>
