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
  import type { EmployeesService } from '#services/hr/employees.svelte.ts'
  import { COST_FUNCTION_OPTIONS } from '#services/hr/hr-catalogs.ts'
  import {
    postDepartment, postPosition, type IDepartment, type IPosition, type OrganizationService,
  } from '#services/hr/organization.svelte.ts'
  import { countActiveByDepartment, employeeFullName } from '#routes/hr/employees/employees.ts'

  let { organization, employees }: { organization: OrganizationService, employees: EmployeesService } = $props()

  const ui = useUI()
  const DEPARTMENT_MODAL_ID = 11
  const POSITION_MODAL_ID = 12

  const costFunctionOptions = COST_FUNCTION_OPTIONS.map((option) => ({ ...option, name: tr(option.name) }))
  const costFunctionNames = new Map(costFunctionOptions.map((option) => [option.id, option.name]))

  let departmentForm = $state({} as IDepartment)
  let positionForm = $state({} as IPosition)

  const activeByDepartment = $derived(countActiveByDepartment(employees.records))
  const managerOptions = $derived(
    employees.records
      .filter((employee) => employee.ss === 1)
      .map((employee) => ({ ID: employee.ID, Name: employeeFullName(employee) })),
  )
  // A department cannot report to itself.
  const parentOptions = $derived(organization.Departments.filter((department) => department.ID !== departmentForm.ID))

  const confirmDelete = (name: string) => confirmWarn({
    title: tr('Delete|Eliminar'),
    message: tr(`Delete "${name}"?|¿Eliminar "${name}"?`),
  })

  // The server refuses to delete what is still in use and says by whom; the page shows that message.
  const saveDepartment = async (isDelete = false) => {
    if (isDelete && !(await confirmDelete(departmentForm.Name))) return
    showLoading(tr('Saving department...|Guardando área...'))
    try {
      const savedDepartment = await postDepartment({ ...departmentForm, ss: isDelete ? 0 : 1 })
      const otherDepartments = organization.Departments.filter((department) => department.ID !== savedDepartment.ID)
      organization.setDepartments(isDelete ? otherDepartments : [...otherDepartments, savedDepartment])
      ui.closeModal(DEPARTMENT_MODAL_ID)
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }

  const savePosition = async (isDelete = false) => {
    if (isDelete && !(await confirmDelete(positionForm.Name))) return
    showLoading(tr('Saving position...|Guardando cargo...'))
    try {
      const savedPosition = await postPosition({ ...positionForm, ss: isDelete ? 0 : 1 })
      const otherPositions = organization.Positions.filter((position) => position.ID !== savedPosition.ID)
      organization.setPositions(isDelete ? otherPositions : [...otherPositions, savedPosition])
      ui.closeModal(POSITION_MODAL_ID)
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }

  const departmentColumns: ITableColumn<IDepartment>[] = [
    { header: 'Department|Área', css: 'px-6', highlight: true, getValue: (e) => e.Name },
    {
      header: 'Reports to|Depende de', css: 'px-6',
      getValue: (e) => organization.DepartmentsMap.get(e.ParentID)?.Name || '',
    },
    { header: 'Cost function|Función de costo', css: 'px-6', getValue: (e) => costFunctionNames.get(e.CostFunction) || '' },
    { header: 'Staff|Personal', headerCss: 'w-70', css: 'text-center', getValue: (e) => activeByDepartment.get(e.ID) || 0 },
  ]

  const positionColumns: ITableColumn<IPosition>[] = [
    { header: 'Position|Cargo', css: 'px-6', highlight: true, getValue: (e) => e.Name },
    { header: 'Department|Área', css: 'px-6', getValue: (e) => organization.DepartmentsMap.get(e.DepartmentID)?.Name || '' },
  ]
</script>

<div class="grid grid-cols-1 lg:grid-cols-2 gap-20" aria-label="Departments and positions">
  <div>
    <div class="flex items-center justify-between mb-8">
      <div class="h3 ff-bold">{tr('Departments|Áreas')}</div>
      <Button color="green" icon="icon-[fa--plus]" label="Opens the modal to create a department."
        onClick={() => { departmentForm = { CostFunction: 1, ss: 1 } as IDepartment; ui.openModal(DEPARTMENT_MODAL_ID) }} />
    </div>
    <VTable columns={departmentColumns} data={organization.Departments} css="w-full"
      onRowClick={(department) => { departmentForm = { ...department }; ui.openModal(DEPARTMENT_MODAL_ID) }} />
  </div>
  <div>
    <div class="flex items-center justify-between mb-8">
      <div class="h3 ff-bold">{tr('Positions|Cargos')}</div>
      <Button color="green" icon="icon-[fa--plus]" label="Opens the modal to create a position."
        onClick={() => { positionForm = { ss: 1 } as IPosition; ui.openModal(POSITION_MODAL_ID) }} />
    </div>
    <VTable columns={positionColumns} data={organization.Positions} css="w-full"
      onRowClick={(position) => { positionForm = { ...position }; ui.openModal(POSITION_MODAL_ID) }} />
  </div>
</div>

<Modal id={DEPARTMENT_MODAL_ID} size={6} bodyCss="px-16 py-14"
  title={departmentForm.ID > 0 ? tr('Edit department|Editar área') : tr('New department|Nueva área')}
  onSave={() => saveDepartment()}
  onDelete={departmentForm.ID > 0 ? () => saveDepartment(true) : undefined}
>
  <div class="grid grid-cols-24 gap-10" aria-label="Department form">
    <Input bind:saveOn={departmentForm} save="Name" label="Name|Nombre" required={true} css="col-span-24" />
    <SearchSelect bind:saveOn={departmentForm} save="CostFunction" label="Cost function|Función de costo" required={true}
      options={costFunctionOptions} keyId="id" keyName="name" css="col-span-24 md:col-span-12" />
    <SearchSelect bind:saveOn={departmentForm} save="ParentID" label="Reports to|Depende de"
      options={parentOptions} keyId="ID" keyName="Name" css="col-span-24 md:col-span-12" />
    <SearchSelect bind:saveOn={departmentForm} save="ManagerEmployeeID" label="Manager|Responsable"
      options={managerOptions} keyId="ID" keyName="Name" css="col-span-24" />
  </div>
</Modal>

<Modal id={POSITION_MODAL_ID} size={5} bodyCss="px-16 py-14"
  title={positionForm.ID > 0 ? tr('Edit position|Editar cargo') : tr('New position|Nuevo cargo')}
  onSave={() => savePosition()}
  onDelete={positionForm.ID > 0 ? () => savePosition(true) : undefined}
>
  <div class="grid grid-cols-24 gap-10" aria-label="Position form">
    <Input bind:saveOn={positionForm} save="Name" label="Name|Nombre" required={true} css="col-span-24" />
    <SearchSelect bind:saveOn={positionForm} save="DepartmentID" label="Department|Área" required={true}
      options={organization.Departments} keyId="ID" keyName="Name" css="col-span-24" />
    <Input bind:saveOn={positionForm} save="OccupationCode" label="SUNAT occupation code|Código de ocupación SUNAT"
      css="col-span-24" />
  </div>
</Modal>
