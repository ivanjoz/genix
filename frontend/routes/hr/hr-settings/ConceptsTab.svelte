<script lang="ts">
  import { useUI } from '@genix/ui'
  import { confirmWarn, hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
  import Button from '#components/buttons/Button.svelte'
  import Checkbox from '#components/form/Checkbox.svelte'
  import Input from '#components/form/Input.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import Layer from '#components/layers/Layer.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import { CONCEPT_KIND_OPTIONS, INCOME_TAX_OPTIONS, LEGAL_CONCEPTS, type ILegalConcept } from '#services/hr/hr-catalogs.ts'
  import { postConcept, type IConcept, type PayrollConfigService } from '#services/hr/payroll.svelte.ts'
  import {
    CONCEPT_KIND_DEDUCTION, CONCEPT_KIND_EARNING, DEDUCTION_ACCOUNT_OPTIONS, EARNING_ACCOUNT_OPTIONS,
    makeEmptyConcept, normalizeConcept, plameCodeOptionsForKind,
  } from './hr-settings.ts'

  // The company's own earnings and deductions, next to the legal concepts the engine calculates.
  let { config }: { config: PayrollConfigService } = $props()

  const ui = useUI()
  const CONCEPT_LAYER_ID = 1
  const kindOptions = CONCEPT_KIND_OPTIONS.map((option) => ({ ...option, name: tr(option.name) }))
  const kindNames = new Map(kindOptions.map((option) => [option.id, option.name]))
  const companyKindOptions = kindOptions.filter((option) => option.id === CONCEPT_KIND_EARNING || option.id === CONCEPT_KIND_DEDUCTION)
  const incomeTaxOptions = INCOME_TAX_OPTIONS.map((option) => ({ ...option, name: tr(option.name) }))
  const incomeTaxNames = new Map(incomeTaxOptions.map((option) => [option.id, option.name]))
  const yesNo = (value?: boolean) => value ? tr('Yes|Sí') : ''

  let conceptForm = $state({} as IConcept)

  const saveConcept = async (isDelete = false) => {
    if (isDelete && !(await confirmWarn({ title: tr('Delete|Eliminar'), message: tr(`Delete "${conceptForm.Name}"?|¿Eliminar "${conceptForm.Name}"?`) }))) return
    showLoading(tr('Saving concept...|Guardando concepto...'))
    try {
      await postConcept({ ...normalizeConcept(conceptForm), ss: isDelete ? 0 : 1 })
      await config.fetch()
      ui.openSideLayer(0)
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }

  const conceptColumns: ITableColumn<IConcept>[] = [
    { header: 'Concept|Concepto', css: 'px-6', highlight: true, getValue: (e) => e.Name },
    { header: 'Type|Tipo', css: 'px-6', getValue: (e) => kindNames.get(e.Kind) || '' },
    { header: 'PLAME', css: 'px-6', getValue: (e) => e.PlameCode },
    { header: 'Remunerative|Remunerativo', css: 'px-6 text-center', getValue: (e) => yesNo(e.IsRemunerative) },
    { header: '5ta', css: 'px-6', getValue: (e) => e.Kind === CONCEPT_KIND_EARNING ? incomeTaxNames.get(e.IncomeTax) || '' : '' },
    { header: 'Account|Cuenta', css: 'px-6', getValue: (e) => String(e.ExpenseAccountCode || e.PayableAccountCode || '') },
  ]
  const legalColumns: ITableColumn<ILegalConcept>[] = [
    { header: 'Legal concept|Concepto legal', css: 'px-6', getValue: (e) => tr(e.name) },
    { header: 'Type|Tipo', css: 'px-6', getValue: (e) => kindNames.get(e.kind) || '' },
    { header: 'PLAME', css: 'px-6', getValue: (e) => e.plameCode || '' },
    { header: 'EsSalud', css: 'px-6 text-center', getValue: (e) => yesNo(e.affectsEssalud) },
    { header: 'Pension|Pensión', css: 'px-6 text-center', getValue: (e) => yesNo(e.affectsPension) },
    { header: 'Account|Cuenta', css: 'px-6', getValue: (e) => [e.expenseAccountCode, e.payableAccountCode].filter(Boolean).join(' / ') },
  ]
</script>

<div class="grid grid-cols-1 xl:grid-cols-2 gap-20">
  <div aria-label="Company concepts">
    <div class="flex items-center justify-between mb-8">
      <div class="h3 ff-bold">{tr('Company concepts|Conceptos de la empresa')}</div>
      <Button color="green" icon="icon-[fa--plus]" name="New|Nuevo" label="Opens the panel to create a payroll concept."
        onClick={() => { conceptForm = makeEmptyConcept(); ui.openSideLayer(CONCEPT_LAYER_ID) }} />
    </div>
    <Layer type="content">
      <VTable columns={conceptColumns} data={config.Concepts} css="w-full" selected={conceptForm.ID}
        isSelected={(concept, id) => concept.ID === id}
        onRowClick={(concept) => { conceptForm = { ...concept }; ui.openSideLayer(CONCEPT_LAYER_ID) }} />
    </Layer>
  </div>
  <div aria-label="Legal concepts">
    <div class="h3 ff-bold mb-8">{tr('Legal concepts (calculated)|Conceptos legales (calculados)')}</div>
    <VTable columns={legalColumns} data={LEGAL_CONCEPTS} css="w-full" />
  </div>
</div>

<Layer type="side" id={CONCEPT_LAYER_ID} sideLayerSize={620} titleCss="h2 mb-6"
  title={conceptForm.ID > 0 ? conceptForm.Name : tr('New concept|Nuevo concepto')}
  onSave={() => saveConcept()}
  onDelete={conceptForm.ID > 0 ? () => saveConcept(true) : undefined}
  onClose={() => { conceptForm = {} as IConcept }}
>
  <div class="grid grid-cols-24 gap-10 mt-12" aria-label="Concept form">
    <Input bind:saveOn={conceptForm} save="Name" label="Name|Nombre" required={true} css="col-span-24" />
    <SearchSelect bind:saveOn={conceptForm} save="Kind" label="Type|Tipo" required={true}
      options={companyKindOptions} keyId="id" keyName="name" css="col-span-24 md:col-span-12" />
    <SearchSelect bind:saveOn={conceptForm} save="PlameCode" label="PLAME code (table 22)|Código PLAME (tabla 22)" required={true}
      options={plameCodeOptionsForKind(conceptForm.Kind)} keyId="code" keyName="label" css="col-span-24 md:col-span-12" />
    {#if conceptForm.Kind === CONCEPT_KIND_EARNING}
      <Checkbox bind:saveOn={conceptForm} save="IsRemunerative" label="Remunerative|Remunerativo" css="col-span-24 md:col-span-12" />
      <SearchSelect bind:saveOn={conceptForm} save="IncomeTax" label="Income tax 5th cat.|Renta de 5ta" required={true}
        options={incomeTaxOptions} keyId="id" keyName="name" css="col-span-24 md:col-span-12" />
      {#if conceptForm.IsRemunerative}
        <Checkbox bind:saveOn={conceptForm} save="AffectsEssalud" label="Pays EsSalud|Aporta a EsSalud" css="col-span-24 md:col-span-8" />
        <Checkbox bind:saveOn={conceptForm} save="AffectsPension" label="Pays pension|Aporta a pensiones" css="col-span-24 md:col-span-8" />
        <Checkbox bind:saveOn={conceptForm} save="ComputableForBenefits" label="CTS / grati base|Computable CTS / grati"
          css="col-span-24 md:col-span-8" />
      {/if}
      <SearchSelect bind:saveOn={conceptForm} save="ExpenseAccountCode" label="Expense account|Cuenta de gasto" required={true}
        options={EARNING_ACCOUNT_OPTIONS} keyId="code" keyName="label" css="col-span-24" />
    {:else}
      <SearchSelect bind:saveOn={conceptForm} save="PayableAccountCode" label="Owed to (account)|Cuenta por pagar" required={true}
        options={DEDUCTION_ACCOUNT_OPTIONS} keyId="code" keyName="label" css="col-span-24" />
    {/if}
    <div class="col-span-24 text-gray-600">
      {tr('Closed payslips keep the treatment and accounts they were calculated with.|Las boletas cerradas conservan el tratamiento y las cuentas con que se calcularon.')}
    </div>
  </div>
</Layer>
