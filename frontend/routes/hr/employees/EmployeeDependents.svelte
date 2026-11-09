<script lang="ts">
  import { untrack } from 'svelte'
  import { useUI } from '@genix/ui'
  import { confirmWarn, hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
  import Button from '#components/buttons/Button.svelte'
  import Checkbox from '#components/form/Checkbox.svelte'
  import DateInput from '#components/form/DateInput.svelte'
  import Input from '#components/form/Input.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import Modal from '#components/layers/Modal.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import { formatTime, todayUnixDay } from '#libs/helpers.ts'
  import { IDENTITY_DOC_OPTIONS } from '#services/crm/identity-doc.ts'
  import {
    DependentsService, postDependent, type EmployeesService, type IDependent,
  } from '#services/hr/employees.svelte.ts'
  import { RELATIONSHIP_OPTIONS } from '#services/hr/hr-catalogs.ts'
  import { employeeFullName, hasFamilyAllowanceOn, makeEmptyDependent } from './employees.ts'

  // The parent re-creates this component per employee ({#key}), so the service is built once here.
  // The family allowance is read from the employee's summary, which the server rewrites on save.
  let { employeeID, employees }: { employeeID: number, employees: EmployeesService } = $props()

  const ui = useUI()
  const DEPENDENT_MODAL_ID = 22
  const RELATIONSHIP_CHILD = 3

  const dependents = new DependentsService(untrack(() => employeeID))
  const relationshipOptions = RELATIONSHIP_OPTIONS.map((option) => ({ ...option, name: tr(option.name) }))
  const relationshipNames = new Map(relationshipOptions.map((option) => [option.id, option.name]))
  const identityDocOptions = IDENTITY_DOC_OPTIONS.filter((option) => option.id !== 6)

  let dependentForm = $state({} as IDependent)

  const employee = $derived(employees.recordsMap.get(employeeID))
  const grantsFamilyAllowance = $derived(!!employee && hasFamilyAllowanceOn(employee, todayUnixDay()))

  const openDependent = (dependent: IDependent) => {
    dependentForm = { ...dependent }
    ui.openModal(DEPENDENT_MODAL_ID)
  }

  const saveDependent = async (isDelete = false) => {
    if (isDelete && !(await confirmWarn({
      title: tr('Delete dependent|Eliminar derechohabiente'),
      message: tr('To record that a dependent no longer counts, set the end date instead. Delete anyway?|Para registrar que ya no cuenta, use la fecha de baja. ¿Eliminar de todos modos?'),
    }))) return

    showLoading(tr('Saving dependent...|Guardando derechohabiente...'))
    try {
      await postDependent(dependents, { ...dependentForm, ss: isDelete ? 0 : 1 })
      ui.closeModal(DEPENDENT_MODAL_ID)
      await employees.fetch()
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }

  const columns: ITableColumn<IDependent>[] = [
    { header: 'Name|Nombre', css: 'px-6', getValue: (e) => employeeFullName(e) },
    { header: 'Relationship|Vínculo', headerCss: 'w-120', css: 'px-6', getValue: (e) => relationshipNames.get(e.Relationship) || '' },
    { header: 'Document|Documento', headerCss: 'w-110', css: 'px-6', getValue: (e) => e.DocumentNumber },
    {
      header: 'Birth|Nacimiento', headerCss: 'w-100', css: 'px-6 whitespace-nowrap',
      getValue: (e) => (e.BirthDate ? formatTime(e.BirthDate, 'd-m-Y') : '') as string,
    },
  ]
</script>

<div class="mt-8" aria-label="Dependents of the employee">
  <div class="flex items-center justify-between mb-8">
    <div class="text-fg-muted">
      {grantsFamilyAllowance
        ? tr('Earns the family allowance (asignación familiar).|Percibe asignación familiar.')
        : tr('No child grants the family allowance today.|Ningún hijo otorga asignación familiar hoy.')}
    </div>
    <Button color="green" icon="icon-[fa--plus]" name={tr('Add|Agregar')}
      label="Opens the modal to register a dependent of the employee."
      onClick={() => openDependent(makeEmptyDependent(employeeID, todayUnixDay()))} />
  </div>
  <VTable {columns} data={dependents.records} css="w-full" onRowClick={openDependent} />
</div>

<Modal id={DEPENDENT_MODAL_ID} size={7} bodyCss="px-16 py-14"
  title={dependentForm.ID > 0 ? tr('Edit dependent|Editar derechohabiente') : tr('New dependent|Nuevo derechohabiente')}
  onSave={() => saveDependent()}
  onDelete={dependentForm.ID > 0 ? () => saveDependent(true) : undefined}
>
  <div class="grid grid-cols-24 gap-10" aria-label="Dependent form">
    <SearchSelect bind:saveOn={dependentForm} save="Relationship" label="Relationship|Vínculo" required={true}
      options={relationshipOptions} keyId="id" keyName="name" css="col-span-24 md:col-span-10" />
    <Input bind:saveOn={dependentForm} save="DocumentNumber" label="Identity document|Documento de identidad"
      leftOptions={identityDocOptions} saveLeft="IdentityDocType" required={true} css="col-span-24 md:col-span-14" />
    <Input bind:saveOn={dependentForm} save="FirstName" label="First names|Nombres" required={true}
      css="col-span-24 md:col-span-8" />
    <Input bind:saveOn={dependentForm} save="LastName" label="Paternal surname|Apellido paterno" required={true}
      css="col-span-24 md:col-span-8" />
    <Input bind:saveOn={dependentForm} save="SecondLastName" label="Maternal surname|Apellido materno"
      css="col-span-24 md:col-span-8" />
    <DateInput bind:saveOn={dependentForm} save="BirthDate" label="Birth date|Fecha de nacimiento"
      required={dependentForm.Relationship === RELATIONSHIP_CHILD} css="col-span-24 md:col-span-8" />
    <DateInput bind:saveOn={dependentForm} save="ValidFrom" label="Registered from|Alta" required={true}
      css="col-span-24 md:col-span-8" />
    <DateInput bind:saveOn={dependentForm} save="ValidTo" label="Removed on|Baja" css="col-span-24 md:col-span-8" />
    {#if dependentForm.Relationship === RELATIONSHIP_CHILD}
      <Checkbox bind:saveOn={dependentForm} save="IsStudying" label="Studying (counts up to 24)|Estudia (cuenta hasta los 24)"
        css="col-span-24" />
    {/if}
  </div>
</Modal>
