<script lang="ts">
  import { useUI } from '@genix/ui'
  import { confirmWarn, hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
  import Button from '#components/buttons/Button.svelte'
  import Checkbox from '#components/form/Checkbox.svelte'
  import DateInput from '#components/form/DateInput.svelte'
  import FilterInput from '#components/form/FilterInput.svelte'
  import Input from '#components/form/Input.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import Layer from '#components/layers/Layer.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import { formatTime, todayUnixDay } from '#libs/helpers.ts'
  import { IDENTITY_DOC_OPTIONS } from '#services/crm/identity-doc.ts'
  import type { ContractsService } from '#services/hr/contracts.svelte.ts'
  import { postEmployee, type EmployeesService, type IEmployee } from '#services/hr/employees.svelte.ts'
  import { SEX_OPTIONS, TERMINATION_REASON_OPTIONS } from '#services/hr/hr-catalogs.ts'
  import type { OrganizationService } from '#services/hr/organization.svelte.ts'
  import EmployeeContracts from './EmployeeContracts.svelte'
  import EmployeeDependents from './EmployeeDependents.svelte'
  import EmployeePayItems from './EmployeePayItems.svelte'
  import { EmployeeStatus, employeeFullName, hasFamilyAllowanceOn, isTerminated, makeEmptyEmployee } from './employees.ts'

  let { employees, organization, contracts }: {
    employees: EmployeesService
    organization: OrganizationService
    /** Undefined without the payroll access: the contracts tab is not offered. */
    contracts?: ContractsService
  } = $props()

  const ui = useUI()
  const today = todayUnixDay()
  const EMPLOYEE_LAYER_ID = 1
  const VIEW_INFO = 1
  const VIEW_CONTRACTS = 3
  const VIEW_DEPENDENTS = 4
  const VIEW_PAY_ITEMS = 5

  let filterText = $state('')
  let showTerminated = $state(false)
  let employeeForm = $state(makeEmptyEmployee())
  let layerView = $state(VIEW_INFO)

  const translateOptions = <T extends { name: string }>(options: T[]) =>
    options.map((option) => ({ ...option, name: tr(option.name) }))
  const sexOptions = translateOptions(SEX_OPTIONS)
  const terminationReasonOptions = translateOptions(TERMINATION_REASON_OPTIONS)
  const identityDocOptions = IDENTITY_DOC_OPTIONS.filter((option) => option.id !== 6)

  const shownEmployees = $derived(
    employees.records.filter((employee) => showTerminated || !isTerminated(employee)),
  )
  const departmentPositions = $derived(
    organization.Positions.filter((position) => position.DepartmentID === employeeForm.DepartmentID),
  )

  // Any other active employee can be the boss; the server refuses a cycle.
  const possibleManagers = $derived(
    employees.records
      .filter((employee) => employee.ss === EmployeeStatus.ACTIVE && employee.ID !== employeeForm.ID)
      .map((employee) => ({ ID: employee.ID, fullName: employeeFullName(employee) })),
  )

  // A saved employee unlocks the tabs that hang from his ID.
  const layerOptions = $derived.by(() => {
    const options: [number, string][] = [[VIEW_INFO, 'Information|Información']]
    if (employeeForm.ID > 0 && contracts) options.push([VIEW_CONTRACTS, 'Contracts|Contratos'])
    if (employeeForm.ID > 0) options.push([VIEW_DEPENDENTS, 'Dependents|Derechohabientes'])
    if (employeeForm.ID > 0 && contracts) options.push([VIEW_PAY_ITEMS, 'Concepts & loans|Conceptos y préstamos'])
    return options
  })

  const openEmployee = (employee: IEmployee) => {
    employeeForm = { ...employee }
    ui.openSideLayer(EMPLOYEE_LAYER_ID)
  }

  const openNewEmployee = () => {
    employeeForm = makeEmptyEmployee()
    layerView = VIEW_INFO
    ui.openSideLayer(EMPLOYEE_LAYER_ID)
  }

  // The server validates everything and answers a descriptive error; the page only shows it.
  const saveEmployee = async (isDelete = false) => {
    if (isDelete && !(await confirmWarn({
      title: tr('Delete employee|Eliminar trabajador'),
      message: tr('To record a departure, set the termination date instead. Delete anyway?|Para registrar una salida use la fecha de cese. ¿Eliminar de todos modos?'),
    }))) return

    showLoading(tr('Saving employee...|Guardando trabajador...'))
    try {
      const savedEmployee = await postEmployee({ ...employeeForm, ss: isDelete ? EmployeeStatus.DELETED : employeeForm.ss })
      employees.addSavedRecords(savedEmployee)
      if (isDelete) {
        ui.openSideLayer(0)
      } else {
        employeeForm = { ...savedEmployee }
      }
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }

  const columns: ITableColumn<IEmployee>[] = [
    {
      header: 'ID', headerCss: 'w-54', css: 'text-center ff-bold', getValue: (e) => e.ID,
      mobile: { order: 1, css: 'col-span-6 ff-bold', icon: '[fa--tag]' },
    },
    {
      header: 'Name|Nombre', css: 'px-8', highlight: true, getValue: (e) => employeeFullName(e),
      mobile: { order: 2, css: 'col-span-18', render: (e) => `<strong>${employeeFullName(e)}</strong>` },
    },
    {
      header: 'Document|Documento', headerCss: 'w-120', css: 'px-6', getValue: (e) => e.DocumentNumber,
      mobile: { order: 3, css: 'col-span-12', labelLeft: 'Doc:' },
    },
    {
      header: 'Department|Área', css: 'px-6',
      getValue: (e) => organization.DepartmentsMap.get(e.DepartmentID)?.Name || '',
      mobile: { order: 4, css: 'col-span-12' },
    },
    {
      header: 'Position|Cargo', css: 'px-6',
      getValue: (e) => organization.PositionsMap.get(e.PositionID)?.Name || '',
      mobile: { order: 5, css: 'col-span-24', labelLeft: 'Cargo:' },
    },
    {
      header: 'Hire date|Ingreso', headerCss: 'w-100', css: 'px-6 whitespace-nowrap',
      getValue: (e) => formatTime(e.HireDate, 'd-m-Y') as string,
      mobile: { order: 6, css: 'col-span-12', labelLeft: 'Ingreso:' },
    },
    {
      header: 'Dependents|Derechohabientes', headerCss: 'w-130', css: 'px-6 text-center',
      getValue: (e) => e.DependentsCount
        ? `${e.DependentsCount}${hasFamilyAllowanceOn(e, today) ? ` · ${tr('Fam. allow.|Asig. fam.')}` : ''}` : '',
      mobile: { order: 8, css: 'col-span-12', labelLeft: 'Derechohab.:' },
    },
    {
      header: 'Status|Estado', headerCss: 'w-100', css: 'px-6',
      getValue: (e) => tr(isTerminated(e) ? 'Terminated|Cesado' : 'Active|Activo'),
      mobile: { order: 7, css: 'col-span-12' },
    },
  ]
</script>

<Layer type="content">
  <div class="flex items-center justify-between mb-6 gap-12 flex-wrap" aria-label="Employees toolbar with filter, terminated toggle and create button">
    <div class="flex items-center gap-16">
      <FilterInput bind:value={filterText} css="w-256" />
      <Checkbox label="Show terminated|Mostrar cesados" checked={showTerminated}
        onToggle={(isChecked) => { showTerminated = isChecked }} />
    </div>
    <Button color="green" icon="icon-[fa--plus]" label="Opens the side layer to register a new employee."
      onClick={openNewEmployee} />
  </div>
  <VTable
    {columns}
    data={shownEmployees}
    css="w-full"
    maxHeight="calc(80vh - 13rem)"
    {filterText}
    getFilterContent={(e) => [employeeFullName(e), e.DocumentNumber, e.Email].join(' ').toLowerCase()}
    selected={employeeForm?.ID}
    isSelected={(employee, selectedID) => employee.ID === selectedID}
    onRowClick={openEmployee}
  />
</Layer>

<Layer
  id={EMPLOYEE_LAYER_ID}
  type="side"
  sideLayerSize={820}
  title={employeeForm.ID > 0 ? employeeFullName(employeeForm) : 'New employee|Nuevo trabajador'}
  titleIcon="icon-[fa--id-card] w-20 h-20 mb-6 c-purple"
  titleCss="h2 mb-6"
  options={layerOptions}
  bind:selected={layerView}
  css="px-12 py-10"
  contentCss="px-0"
  onSave={layerView === VIEW_INFO ? () => saveEmployee() : undefined}
  onDelete={employeeForm.ID > 0 && layerView === VIEW_INFO ? () => saveEmployee(true) : undefined}
  onClose={() => { employeeForm = makeEmptyEmployee() }}
>
  {#if layerView === VIEW_INFO}
    <div class="grid grid-cols-24 gap-10 mt-8" aria-label="Employee information form with personal data, department, position and dates">
      <Input bind:saveOn={employeeForm} save="DocumentNumber" label="Identity document|Documento de identidad"
        leftOptions={identityDocOptions} saveLeft="IdentityDocType" required={true}
        css="col-span-24 md:col-span-12" />
      <Input bind:saveOn={employeeForm} save="FirstName" label="First names|Nombres" required={true}
        css="col-span-24 md:col-span-12" />
      <Input bind:saveOn={employeeForm} save="LastName" label="Paternal surname|Apellido paterno" required={true}
        css="col-span-24 md:col-span-12" />
      <Input bind:saveOn={employeeForm} save="SecondLastName" label="Maternal surname|Apellido materno"
        css="col-span-24 md:col-span-12" />
      <DateInput bind:saveOn={employeeForm} save="BirthDate" label="Birth date|Fecha de nacimiento" required={true}
        css="col-span-24 md:col-span-8" />
      <SearchSelect bind:saveOn={employeeForm} save="Sex" label="Sex|Sexo" required={true}
        options={sexOptions} keyId="id" keyName="name" css="col-span-24 md:col-span-8" />
      <Input bind:saveOn={employeeForm} save="Phone" label="Phone|Teléfono" css="col-span-24 md:col-span-8" />
      <Input bind:saveOn={employeeForm} save="Email" label="Email" css="col-span-24 md:col-span-12" />
      <Input bind:saveOn={employeeForm} save="Address" label="Address|Dirección" css="col-span-24 md:col-span-12" />
      <SearchSelect bind:saveOn={employeeForm} save="DepartmentID" label="Department|Área" required={true}
        options={organization.Departments} keyId="ID" keyName="Name" css="col-span-24 md:col-span-12"
        onChange={() => { employeeForm.PositionID = 0 }} />
      <SearchSelect bind:saveOn={employeeForm} save="PositionID" label="Position|Cargo"
        options={departmentPositions} keyId="ID" keyName="Name" css="col-span-24 md:col-span-12" />
      <SearchSelect bind:saveOn={employeeForm} save="ManagerEmployeeID" label="Immediate boss|Jefe inmediato"
        options={possibleManagers} keyId="ID" keyName="fullName" css="col-span-24 md:col-span-12" />
      <DateInput bind:saveOn={employeeForm} save="HireDate" label="Hire date|Fecha de ingreso" required={true}
        css="col-span-24 md:col-span-8" />
      <DateInput bind:saveOn={employeeForm} save="TerminationDate" label="Termination date|Fecha de cese"
        css="col-span-24 md:col-span-8" />
      <SearchSelect bind:saveOn={employeeForm} save="TerminationReason" label="Termination reason|Motivo de cese"
        options={terminationReasonOptions} keyId="id" keyName="name" css="col-span-24 md:col-span-8" />
    </div>
  {:else if layerView === VIEW_CONTRACTS && contracts}
    <EmployeeContracts employee={employeeForm} {contracts} />
  {:else if layerView === VIEW_DEPENDENTS}
    {#key employeeForm.ID}
      <EmployeeDependents employeeID={employeeForm.ID} {employees} />
    {/key}
  {:else if layerView === VIEW_PAY_ITEMS && contracts}
    {#key employeeForm.ID}
      <EmployeePayItems employeeID={employeeForm.ID} />
    {/key}
  {/if}
</Layer>
