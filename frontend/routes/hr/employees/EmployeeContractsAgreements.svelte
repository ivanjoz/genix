<script lang="ts">
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
  import { formatN, formatTime, todayUnixDay } from '#libs/helpers.ts'
  import { postContractAgreement, type ContractsAgreementsService, type IContractAgreement } from '#services/hr/contracts-agreements.svelte.ts'
  import type { IEmployee } from '#services/hr/employees.svelte.ts'
  import {
    AFP_COMMISSION_TYPE_OPTIONS, BANK_OPTIONS, CONTRACT_TYPE_OPTIONS, PAY_BASIS_OPTIONS, PAY_DAY_OF_WEEK_OPTIONS,
    PENSION_SYSTEM_OPTIONS,
  } from '#services/hr/hr-catalogs.ts'
  import {
    alignAgreementWithEngagement, CONTRACT_TYPE_INDEFINITE, contractValidOn, employeeContracts, fixedPayBasisOf,
    isAfpPensionSystem, makeNextContract, payAmountLabel, showsPayDayOfWeek, showsPayrollFields,
  } from './employees.ts'

  let { employee, contracts }: { employee: IEmployee, contracts: ContractsAgreementsService } = $props()

  const ui = useUI()
  const CONTRACT_MODAL_ID = 21

  const translateOptions = <T extends { name: string }>(options: T[]) =>
    options.map((option) => ({ ...option, name: tr(option.name) }))
  const contractTypeOptions = translateOptions(CONTRACT_TYPE_OPTIONS)
  const commissionTypeOptions = translateOptions(AFP_COMMISSION_TYPE_OPTIONS)
  const payBasisOptions = translateOptions(PAY_BASIS_OPTIONS)
  const payDayOfWeekOptions = translateOptions(PAY_DAY_OF_WEEK_OPTIONS)
  const contractTypeNames = new Map(contractTypeOptions.map((option) => [option.id, option.name]))
  const pensionSystemNames = new Map(PENSION_SYSTEM_OPTIONS.map((option) => [option.id, option.name]))
  const payBasisNames = new Map(payBasisOptions.map((option) => [option.id, option.name]))
  const payDayOfWeekNames = new Map(payDayOfWeekOptions.map((option) => [option.id, option.name]))

  let contractForm = $state({} as IContractAgreement)

  const versions = $derived(employeeContracts(contracts.records, employee.ID))
  const currentContractID = $derived(contractValidOn(contracts.records, employee.ID, todayUnixDay())?.ID || 0)
  // The form only shows what applies to the worker's engagement (plan §9.5): the payroll fields for
  // Planilla, the basis selector for a temporary worker (Planilla is monthly, a fee receipt hourly).
  const hasPayrollFields = $derived(showsPayrollFields(employee.EngagementType))
  const choosesPayBasis = $derived(fixedPayBasisOf(employee.EngagementType) === undefined)

  const openContract = (contract: IContractAgreement) => {
    contractForm = { ...contract }
    ui.openModal(CONTRACT_MODAL_ID)
  }

  const saveContract = async (isDelete = false) => {
    if (isDelete && !(await confirmWarn({
      title: tr('Delete version|Eliminar versión'),
      message: tr('The previous version will apply from this date. Continue?|Desde esta fecha aplicará la versión anterior. ¿Continuar?'),
    }))) return

    showLoading(tr('Saving...|Guardando...'))
    try {
      const agreement = alignAgreementWithEngagement(contractForm, employee.EngagementType)
      const savedContract = await postContractAgreement({ ...agreement, ss: isDelete ? 0 : 1 })
      contracts.addSavedRecords(savedContract)
      ui.closeModal(CONTRACT_MODAL_ID)
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }

  const columns: ITableColumn<IContractAgreement>[] = [
    {
      header: 'Valid from|Vigente desde', headerCss: 'w-120', css: 'px-6 whitespace-nowrap',
      getValue: (e) => (formatTime(e.ValidFrom, 'd-m-Y') as string) + (e.ID === currentContractID ? ' ✔' : ''),
    },
    { header: 'Type|Tipo', css: 'px-6', getValue: (e) => contractTypeNames.get(e.ContractType) || '' },
    {
      header: 'Pay|Pago', headerCss: 'w-170', css: 'px-6 text-right whitespace-nowrap',
      getValue: (e) => `${formatN(e.PayAmount / 100, 2)} · ${payBasisNames.get(e.PayBasis) || ''}`,
    },
    {
      header: 'Pay day|Día de pago', headerCss: 'w-100', css: 'px-6',
      getValue: (e) => payDayOfWeekNames.get(e.PayDayOfWeek) || '',
    },
    { header: 'Hours|Horas', headerCss: 'w-60', css: 'text-center', getValue: (e) => e.DailyHours },
    { header: 'Pension|Pensión', css: 'px-6', getValue: (e) => pensionSystemNames.get(e.PensionSystem) || '' },
  ]
</script>

<div class="mt-8" aria-label="Contract and agreement versions of the employee">
  <div class="flex items-center justify-between mb-8">
    <div class="text-fg-muted">
      {tr('Each raise or change is a new version; the one in force on each date applies.|Cada aumento o cambio es una nueva versión; se aplica la vigente en cada fecha.')}
    </div>
    <Button color="green" icon="icon-[fa--plus]" name={tr('New version|Nueva versión')}
      label="Opens the modal to add a contract or agreement version, prefilled from the one in force."
      onClick={() => openContract(makeNextContract(employee, versions.find((e) => e.ID === currentContractID) || versions[0]))} />
  </div>
  <VTable {columns} data={versions} css="w-full" onRowClick={openContract} />
</div>

<Modal id={CONTRACT_MODAL_ID} size={8} bodyCss="px-16 py-14"
  title={contractForm.ID > 0 ? tr('Edit version|Editar versión') : tr('New version|Nueva versión')}
  onSave={() => saveContract()}
  onDelete={contractForm.ID > 0 ? () => saveContract(true) : undefined}
>
  <div class="grid grid-cols-24 gap-10" aria-label="Contract or agreement version form">
    <DateInput bind:saveOn={contractForm} save="ValidFrom" label="Valid from|Vigente desde"
      placeholder={tr('= start date|= fecha de inicio')} css="col-span-24 md:col-span-8" />
    {#if hasPayrollFields}
      <SearchSelect bind:saveOn={contractForm} save="ContractType" label="Contract type|Tipo de contrato" required={true}
        options={contractTypeOptions} keyId="id" keyName="name" css="col-span-24 md:col-span-16"
        onChange={() => { if (contractForm.ContractType === CONTRACT_TYPE_INDEFINITE) contractForm.EndDate = 0 }} />
    {:else if choosesPayBasis}
      <SearchSelect bind:saveOn={contractForm} save="PayBasis" label="Pay basis|Base de pago" required={true}
        options={payBasisOptions} keyId="id" keyName="name" css="col-span-24 md:col-span-8"
        onChange={() => { if (!showsPayDayOfWeek(contractForm.PayBasis)) contractForm.PayDayOfWeek = 0 }} />
    {/if}
    {#if showsPayDayOfWeek(contractForm.PayBasis)}
      <SearchSelect bind:saveOn={contractForm} save="PayDayOfWeek" label="Pay day|Día de pago" required={true}
        options={payDayOfWeekOptions} keyId="id" keyName="name" css="col-span-24 md:col-span-8" />
    {/if}
    <DateInput bind:saveOn={contractForm} save="StartDate" label="Start date|Fecha de inicio" required={true}
      css="col-span-24 md:col-span-8" />
    <DateInput bind:saveOn={contractForm} save="EndDate" label="End date|Fecha de fin"
      disabled={hasPayrollFields && contractForm.ContractType === CONTRACT_TYPE_INDEFINITE} css="col-span-24 md:col-span-8" />
    <Input bind:saveOn={contractForm} save="DailyHours" type="number" label="Daily hours|Horas diarias" required={true}
      css="col-span-24 md:col-span-8" />
    <Input bind:saveOn={contractForm} save="PayAmount" type="number" baseDecimals={2}
      label={payAmountLabel(contractForm.PayBasis)} required={true} css="col-span-24 md:col-span-8" />
    {#if hasPayrollFields}
      <SearchSelect bind:saveOn={contractForm} save="PensionSystem" label="Pension system|Sistema de pensiones" required={true}
        options={PENSION_SYSTEM_OPTIONS} keyId="id" keyName="name" css="col-span-24 md:col-span-8" />
      <Checkbox bind:saveOn={contractForm} save="HasEPS" label="Has EPS|Tiene EPS" css="col-span-24 md:col-span-8 mt-24" />
      {#if isAfpPensionSystem(contractForm.PensionSystem)}
        <Input bind:saveOn={contractForm} save="CUSPP" label="CUSPP" required={true} css="col-span-24 md:col-span-12" />
        <SearchSelect bind:saveOn={contractForm} save="AfpCommissionType" label="AFP commission|Comisión AFP" required={true}
          options={commissionTypeOptions} keyId="id" keyName="name" css="col-span-24 md:col-span-12" />
      {/if}
    {/if}
    <SearchSelect bind:saveOn={contractForm} save="SalaryBankID" label="Pay bank|Banco de pago"
      options={BANK_OPTIONS} keyId="id" keyName="name" css="col-span-24 md:col-span-8" />
    <Input bind:saveOn={contractForm} save="SalaryAccount" label="Pay account|Cuenta de pago"
      css="col-span-24 md:col-span-16" />
    {#if hasPayrollFields}
      <SearchSelect bind:saveOn={contractForm} save="CtsBankID" label="CTS bank|Banco CTS"
        options={BANK_OPTIONS} keyId="id" keyName="name" css="col-span-24 md:col-span-8" />
      <Input bind:saveOn={contractForm} save="CtsAccount" label="CTS account|Cuenta CTS" css="col-span-24 md:col-span-16" />
    {/if}
  </div>
</Modal>
