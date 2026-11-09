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
  import { LEGAL_PARAMETER_OPTIONS, PENSION_SYSTEM_OPTIONS } from '#services/hr/hr-catalogs.ts'
  import {
    postAfpRates, postLegalParameter, type IAfpRate, type ILegalParameter, type LegalParametersService,
  } from '#services/hr/legal-parameters.svelte.ts'

  // The country's legal payroll values and AFP rates. Editable only on the SaaS System page; the
  // HR settings page shows the same panel read-only so a company sees what its payroll uses.
  let { legal, editable = false }: { legal: LegalParametersService, editable?: boolean } = $props()

  const ui = useUI()
  const PARAMETER_MODAL_ID = 41
  const AFP_MODAL_ID = 42
  const AFP_PENSION_SYSTEMS = PENSION_SYSTEM_OPTIONS.filter((option) => option.id >= 2)
  const parameterOptions = LEGAL_PARAMETER_OPTIONS.map((option) => ({ ...option, name: tr(option.name) }))
  const parameterOptionsByID = new Map(parameterOptions.map((option) => [option.id, option]))
  const afpNames = new Map(AFP_PENSION_SYSTEMS.map((option) => [option.id, tr(option.name)]))

  let parameterForm = $state({} as ILegalParameter)
  // One AFP table per period: four rows edited together, in percent with two decimals.
  let afpForm = $state({ Period: 0, rates: [] as IAfpRate[] })

  /** Cents show as soles, basis points as a percent: both are the stored value / 100. */
  const formatParameterValue = (parameter: ILegalParameter) => {
    const unit = parameterOptionsByID.get(parameter.Key)?.unit
    const value = (parameter.Value / 100).toLocaleString('en-US', { minimumFractionDigits: 2 })
    return unit === 'bp' ? `${value} %` : `S/ ${value}`
  }
  const formatPercent = (basisPoints: number) => `${(basisPoints / 100).toFixed(2)} %`

  const saveParameter = async (isDelete = false) => {
    if (isDelete && !(await confirmWarn({ title: tr('Delete|Eliminar'), message: tr('Delete this value?|¿Eliminar este valor?') }))) return
    showLoading(tr('Saving...|Guardando...'))
    try {
      await postLegalParameter(legal, { ...parameterForm, ss: isDelete ? 0 : 1 })
      ui.closeModal(PARAMETER_MODAL_ID)
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }

  const openAfpPeriod = (period: number) => {
    const stored = legal.AfpRates.filter((afpRate) => afpRate.Period === period)
    afpForm = {
      Period: period,
      rates: AFP_PENSION_SYSTEMS.map((option) => ({ ...(stored.find((afpRate) => afpRate.PensionSystem === option.id)
        || { FundRate: 1000, PremiumRate: 137 }), PensionSystem: option.id, Period: period, ss: 1 }) as IAfpRate),
    }
    ui.openModal(AFP_MODAL_ID)
  }

  const saveAfpRates = async () => {
    if (afpForm.Period % 100 < 1 || afpForm.Period % 100 > 12) { notifyFailure(tr('Enter a YYMM period|Ingrese un periodo AAMM')); return }
    showLoading(tr('Saving...|Guardando...'))
    try {
      await postAfpRates(legal, afpForm.rates.map((afpRate) => ({ ...afpRate, Period: afpForm.Period })))
      ui.closeModal(AFP_MODAL_ID)
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }

  const parameterColumns: ITableColumn<ILegalParameter>[] = [
    { header: 'Parameter|Parámetro', css: 'px-6', highlight: true, getValue: (e) => parameterOptionsByID.get(e.Key)?.name || '' },
    { header: 'Valid from|Vigente desde', css: 'px-6', getValue: (e) => String(formatTime(e.ValidFrom, 'd-m-Y')) },
    { header: 'Value|Valor', css: 'px-6 text-right', getValue: (e) => formatParameterValue(e) },
  ]
  const afpColumns: ITableColumn<IAfpRate>[] = [
    { header: 'Period|Periodo', css: 'px-6', highlight: true, getValue: (e) => e.Period },
    { header: 'AFP|AFP', css: 'px-6', getValue: (e) => afpNames.get(e.PensionSystem) || '' },
    { header: 'Fund|Fondo', css: 'px-6 text-right', getValue: (e) => formatPercent(e.FundRate) },
    { header: 'Premium|Prima', css: 'px-6 text-right', getValue: (e) => formatPercent(e.PremiumRate) },
    { header: 'Flow comm.|Com. flujo', css: 'px-6 text-right', getValue: (e) => formatPercent(e.FlowCommission) },
    { header: 'Balance comm.|Com. saldo', css: 'px-6 text-right', getValue: (e) => formatPercent(e.MixedCommission) },
  ]
</script>

<div class="grid grid-cols-1 xl:grid-cols-2 gap-20" aria-label="Legal payroll parameters">
  <div>
    <div class="flex items-center justify-between mb-8">
      <div class="h3 ff-bold">{tr('Legal values|Valores legales')}</div>
      {#if editable}
        <Button color="green" icon="icon-[fa--plus]" name="New value|Nuevo valor" label="Opens the modal to add a dated legal value."
          onClick={() => { parameterForm = { Key: 1, ValidFrom: todayUnixDay(), ss: 1 } as ILegalParameter; ui.openModal(PARAMETER_MODAL_ID) }} />
      {/if}
    </div>
    <VTable columns={parameterColumns} data={legal.Parameters} css="w-full"
      onRowClick={(parameter) => { if (editable) { parameterForm = { ...parameter }; ui.openModal(PARAMETER_MODAL_ID) } }} />
  </div>
  <div>
    <div class="flex items-center justify-between mb-8">
      <div class="h3 ff-bold">{tr('AFP rates (SBS)|Tasas AFP (SBS)')}</div>
      {#if editable}
        <Button color="green" icon="icon-[fa--plus]" name="Period rates|Tasas del periodo"
          label="Opens the modal to enter the four AFP rates of a period." onClick={() => openAfpPeriod(0)} />
      {/if}
    </div>
    <VTable columns={afpColumns} data={legal.AfpRates} css="w-full"
      onRowClick={(afpRate) => { if (editable) openAfpPeriod(afpRate.Period) }} />
  </div>
</div>

{#if editable}
  <Modal id={PARAMETER_MODAL_ID} size={4} bodyCss="px-16 py-14"
    title={parameterForm.ID > 0 ? tr('Edit legal value|Editar valor legal') : tr('New legal value|Nuevo valor legal')}
    onSave={() => saveParameter()}
    onDelete={parameterForm.ID > 0 ? () => saveParameter(true) : undefined}
  >
    <div class="grid grid-cols-24 gap-10" aria-label="Legal value form">
      <SearchSelect bind:saveOn={parameterForm} save="Key" label="Parameter|Parámetro" required={true}
        options={parameterOptions} keyId="id" keyName="name" css="col-span-24" />
      <DateInput bind:saveOn={parameterForm} save="ValidFrom" label="Valid from|Vigente desde" required={true}
        css="col-span-24 md:col-span-12" />
      <Input bind:saveOn={parameterForm} save="Value" type="number" baseDecimals={2} required={true}
        label={parameterOptionsByID.get(parameterForm.Key)?.unit === 'bp' ? 'Rate (%)|Tasa (%)' : 'Amount (S/)|Monto (S/)'}
        css="col-span-24 md:col-span-12" />
    </div>
  </Modal>

  <Modal id={AFP_MODAL_ID} size={7} bodyCss="px-16 py-14" title={tr('AFP rates of the period|Tasas AFP del periodo')}
    onSave={saveAfpRates}
  >
    <div class="grid grid-cols-24 gap-10" aria-label="AFP rates form">
      <Input bind:saveOn={afpForm} save="Period" type="number" label="Period (YYMM)|Periodo (AAMM)" required={true}
        css="col-span-24 md:col-span-8" />
      {#each afpForm.rates as afpRate}
        <div class="col-span-24 md:col-span-4 mt-28 ff-bold">{afpNames.get(afpRate.PensionSystem)}</div>
        <Input saveOn={afpRate} save="FundRate" type="number" baseDecimals={2} label="Fund %|Fondo %" css="col-span-12 md:col-span-5" />
        <Input saveOn={afpRate} save="PremiumRate" type="number" baseDecimals={2} label="Premium %|Prima %" css="col-span-12 md:col-span-5" />
        <Input saveOn={afpRate} save="FlowCommission" type="number" baseDecimals={2} label="Flow %|Flujo %" css="col-span-12 md:col-span-5" />
        <Input saveOn={afpRate} save="MixedCommission" type="number" baseDecimals={2} label="Balance %|Saldo %" css="col-span-12 md:col-span-5" />
      {/each}
    </div>
  </Modal>
{/if}
