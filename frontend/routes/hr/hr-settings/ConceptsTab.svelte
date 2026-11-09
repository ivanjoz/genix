<script lang="ts">
  import { useUI } from '@genix/ui'
  import { confirmWarn, hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
  import Button from '#components/buttons/Button.svelte'
  import Checkbox from '#components/form/Checkbox.svelte'
  import CheckboxOptions from '#components/form/CheckboxOptions.svelte'
  import FilterInput from '#components/form/FilterInput.svelte'
  import Input from '#components/form/Input.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import Layer from '#components/layers/Layer.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import { CONCEPT_KIND_OPTIONS, INCOME_TAX_OPTIONS } from '#services/hr/hr-catalogs.ts'
  import { postConcept, type IConcept, type PayrollConfigService } from '#services/hr/payroll.svelte.ts'
  import {
    CONCEPT_KIND_DEDUCTION, CONCEPT_KIND_EARNING, DEDUCTION_ACCOUNT_OPTIONS, EARNING_ACCOUNT_OPTIONS,
    ConceptOrigin, allConcepts, conceptsOfOrigins, isLegalConceptID, makeEmptyConcept, normalizeConcept, plameCodeOptionsForKind,
  } from './hr-settings.ts'

  // Every concept in one table: the legal ones (IDs 1–200: what the engine calculates and the records
  // registered on the employee) and the company's own (201+). A row opens the side layer: a company
  // concept to edit, a legal one to read, since it is fixed in code.
  let { config }: { config: PayrollConfigService } = $props()

  const ui = useUI()
  const CONCEPT_LAYER_ID = 1
  const kindOptions = CONCEPT_KIND_OPTIONS.map((option) => ({ ...option, name: tr(option.name) }))
  const kindNames = new Map(kindOptions.map((option) => [option.id, option.name]))
  const companyKindOptions = kindOptions.filter((option) => option.id === CONCEPT_KIND_EARNING || option.id === CONCEPT_KIND_DEDUCTION)
  const incomeTaxOptions = INCOME_TAX_OPTIONS.map((option) => ({ ...option, name: tr(option.name) }))
  const incomeTaxNames = new Map(incomeTaxOptions.map((option) => [option.id, option.name]))
  const yesNo = (value?: boolean) => value ? tr('Yes|Sí') : ''
  const accountsOf = (concept: IConcept) => [concept.ExpenseAccountCode, concept.PayableAccountCode].filter(Boolean).join(' / ')

  const originOptions = [
    { id: ConceptOrigin.Legal, name: tr('Legal|Legales') },
    { id: ConceptOrigin.Company, name: tr("Company's|De la empresa") },
  ]

  let filterText = $state('')
  /** Both origins shown by default. */
  let conceptFilters = $state({ origins: [ConceptOrigin.Legal, ConceptOrigin.Company] as number[] })
  let conceptForm = $state({} as IConcept)
  const concepts = $derived(conceptsOfOrigins(allConcepts(config.Concepts), conceptFilters.origins))
  const isLegalForm = $derived(isLegalConceptID(conceptForm.ID))

  const openConcept = (concept: IConcept) => {
    conceptForm = { ...concept }
    ui.openSideLayer(CONCEPT_LAYER_ID)
  }

  const saveConcept = async (isDelete = false) => {
    if (isDelete && !(await confirmWarn({ title: tr('Delete|Eliminar'), message: tr(`Delete "${conceptForm.Name}"?|¿Eliminar "${conceptForm.Name}"?`) }))) return
    showLoading(tr('Saving concept...|Guardando concepto...'))
    try {
      const savedConcept = await postConcept(config, { ...normalizeConcept(conceptForm), ss: isDelete ? 0 : 1 })
      if (isDelete) ui.openSideLayer(0)
      else conceptForm = { ...savedConcept }
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }

  const columns: ITableColumn<IConcept>[] = [
    { header: 'ID', headerCss: 'w-54', css: 'text-center ff-bold', getValue: (e) => e.ID },
    { header: 'Concept|Concepto', css: 'px-6', highlight: true, getValue: (e) => tr(e.Name) },
    { header: 'Type|Tipo', css: 'px-6', getValue: (e) => kindNames.get(e.Kind) || '' },
    { header: 'PLAME', css: 'px-6', getValue: (e) => e.PlameCode },
    { header: 'Remunerative|Remunerativo', css: 'px-6 text-center', getValue: (e) => yesNo(e.IsRemunerative) },
    { header: 'EsSalud', css: 'px-6 text-center', getValue: (e) => yesNo(e.AffectsEssalud) },
    { header: 'Pension|Pensión', css: 'px-6 text-center', getValue: (e) => yesNo(e.AffectsPension) },
    { header: '5ta', css: 'px-6', getValue: (e) => e.Kind === CONCEPT_KIND_EARNING ? incomeTaxNames.get(e.IncomeTax) || '' : '' },
    { header: 'Account|Cuenta', css: 'px-6', getValue: (e) => accountsOf(e) },
    { header: 'Separate payslip|Boleta aparte', css: 'px-6 text-center', getValue: (e) => yesNo(e.SeparatePayslip) },
  ]
</script>

<Layer type="content">
  <div class="flex items-center justify-between mb-6 gap-12 flex-wrap" aria-label="Concepts toolbar with filter, origin toggle and create button">
    <div class="flex items-center gap-16 flex-wrap">
      <FilterInput bind:value={filterText} css="w-256" />
      <CheckboxOptions type="multiple" useButtonsSlim={true} required={true} saveOn={conceptFilters} save="origins"
        options={originOptions} keyId="id" keyName="name" />
    </div>
    <Button color="green" icon="icon-[fa--plus]" name="New|Nuevo" label="Opens the side layer to create a company concept."
      onClick={() => openConcept(makeEmptyConcept())} />
  </div>
  <VTable
    {columns}
    data={concepts}
    css="w-full"
    maxHeight="calc(100vh - 8rem - 16px)"
    {filterText}
    getFilterContent={(e) => [e.ID, tr(e.Name), e.PlameCode].join(' ').toLowerCase()}
    selected={conceptForm.ID}
    isSelected={(concept, selectedID) => concept.ID === selectedID}
    onRowClick={openConcept}
  />
</Layer>

<Layer
  id={CONCEPT_LAYER_ID}
  type="side"
  sideLayerSize={640}
  title={conceptForm.ID > 0 ? `${conceptForm.ID} · ${tr(conceptForm.Name)}` : 'New concept|Nuevo concepto'}
  titleCss="h2 mb-6"
  css="px-12 py-10"
  contentCss="px-0"
  onSave={isLegalForm ? undefined : () => saveConcept()}
  onDelete={conceptForm.ID > 0 && !isLegalForm ? () => saveConcept(true) : undefined}
  onClose={() => { conceptForm = {} as IConcept }}
>
  {#if isLegalForm}
    <!-- A legal concept is fixed in code (legal_concepts.go): its treatment is shown, not edited. -->
    <div class="text-gray-600 mt-8 mb-12">
      {tr('Legal concept: defined by the system, it cannot be edited.|Concepto legal: lo define el sistema, no se puede editar.')}
    </div>
    <div class="grid grid-cols-[180px_1fr] gap-y-8 gap-x-12" aria-label="Legal concept treatment">
      {#each [
        ['Type|Tipo', kindNames.get(conceptForm.Kind) || ''],
        ['PLAME code|Código PLAME', conceptForm.PlameCode],
        ['Remunerative|Remunerativo', yesNo(conceptForm.IsRemunerative) || tr('No|No')],
        ['Pays EsSalud|Aporta a EsSalud', yesNo(conceptForm.AffectsEssalud) || tr('No|No')],
        ['Pays pension|Aporta a pensiones', yesNo(conceptForm.AffectsPension) || tr('No|No')],
        ['CTS / grati base|Computable CTS / grati', yesNo(conceptForm.ComputableForBenefits) || tr('No|No')],
        ['Income tax 5th cat.|Renta de 5ta', conceptForm.Kind === CONCEPT_KIND_EARNING ? incomeTaxNames.get(conceptForm.IncomeTax) || '' : ''],
        ['Accounts|Cuentas', accountsOf(conceptForm)],
      ] as [label, value]}
        <div class="text-gray-500">{tr(label)}</div>
        <div>{value || '—'}</div>
      {/each}
    </div>
  {:else}
    <div class="grid grid-cols-24 gap-10 mt-8" aria-label="Company concept form">
      <Input bind:saveOn={conceptForm} save="Name" label="Name|Nombre" required={true} css="col-span-24" />
      <SearchSelect bind:saveOn={conceptForm} save="Kind" label="Type|Tipo" required={true}
        options={companyKindOptions} keyId="id" keyName="name" css="col-span-24 md:col-span-12" />
      {#if conceptForm.Kind === CONCEPT_KIND_EARNING}
        <SearchSelect bind:saveOn={conceptForm} save="IncomeTax" label="Income tax 5th cat.|Renta de 5ta" required={true}
          options={incomeTaxOptions} keyId="id" keyName="name" css="col-span-24 md:col-span-12" />
      {/if}
      <SearchSelect bind:saveOn={conceptForm} save="PlameCode" label="PLAME code (table 22)|Código PLAME (tabla 22)" required={true}
        options={plameCodeOptionsForKind(conceptForm.Kind)} keyId="code" keyName="label" css="col-span-24" />
      {#if conceptForm.Kind === CONCEPT_KIND_EARNING}
        <Checkbox bind:saveOn={conceptForm} save="IsRemunerative" label="Remunerative|Remunerativo" css="col-span-24" />
        {#if conceptForm.IsRemunerative}
          <Checkbox bind:saveOn={conceptForm} save="AffectsEssalud" label="Pays EsSalud|Aporta a EsSalud" css="col-span-24 md:col-span-8" />
          <Checkbox bind:saveOn={conceptForm} save="AffectsPension" label="Pays pension|Aporta a pensiones" css="col-span-24 md:col-span-8" />
          <Checkbox bind:saveOn={conceptForm} save="ComputableForBenefits" label="CTS / grati base|Computable CTS / grati"
            css="col-span-24 md:col-span-8" />
        {/if}
        <SearchSelect bind:saveOn={conceptForm} save="ExpenseAccountCode" label="Expense account|Cuenta de gasto" required={true}
          options={EARNING_ACCOUNT_OPTIONS} keyId="code" keyName="label" css="col-span-24" />
        <Checkbox bind:saveOn={conceptForm} save="SeparatePayslip"
          label="Paid on a separate payslip (extraordinary payment)|Se paga en boleta aparte (pago extraordinario)" css="col-span-24" />
      {:else}
        <SearchSelect bind:saveOn={conceptForm} save="PayableAccountCode" label="Owed to (account)|Cuenta por pagar" required={true}
          options={DEDUCTION_ACCOUNT_OPTIONS} keyId="code" keyName="label" css="col-span-24" />
      {/if}
      <div class="col-span-24 text-gray-600">
        {tr('Closed payslips keep the treatment and accounts they were calculated with.|Las boletas cerradas conservan el tratamiento y las cuentas con que se calcularon.')}
      </div>
    </div>
  {/if}
</Layer>
