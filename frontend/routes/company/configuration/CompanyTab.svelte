<script lang="ts">
import Input from '#components/form/Input.svelte';
import SearchSelect from '#components/form/SearchSelect.svelte';
import Button from '#components/buttons/Button.svelte';
import T from '#components/misc/T.svelte';
import CompanyFlagsPanel from './CompanyFlagsPanel.svelte';
import { CountryCitiesService } from '#services/business/country-cities.svelte.ts';
import Checkbox from '#components/form/Checkbox.svelte';
import { tr } from '#core/store.svelte.ts';
import { saveCompanyParameters, type EmpresaParametrosService } from "./empresas.svelte"
import {
  defaultIncomeTaxPrepaymentRate, NRUS_CATEGORY_OPTIONS, TAX_REGIME_NRUS, TAX_REGIME_OPTIONS,
} from './company-tax.ts'

  const { service }: { service: EmpresaParametrosService } = $props()

  const taxRegimeOptions = TAX_REGIME_OPTIONS.map((option) => ({ ...option, name: tr(option.name) }))
  const nrusCategoryOptions = NRUS_CATEGORY_OPTIONS.map((option) => ({ ...option, name: tr(option.name) }))

  // The district is the company's ubigeo, which every electronic document declares
  // as the issuer's fiscal address. Same catalog and same selector the sites use,
  // so the two cannot disagree about what a district is.
  const citiesService = new CountryCitiesService(true)
</script>

<div class="grid grid-cols-24 gap-14 items-start">
  <section class="col-span-24 lg:col-span-12 rounded-[12px] border border-line bg-surface p-16 shadow-sm"
    aria-label="Company parameters form with name, RUC, legal address and contact fields">
    <!-- Save sits inside this panel, not above the page: the series alongside it save
         through their own endpoint, so a button at the top would claim to write both. -->
    <div class="flex items-center gap-10 mb-14">
      <div class="h3 ff-bold flex-1 min-w-0"><T text="Company Parameters|Parámetros de la Empresa" /></div>
      <Button color="blue" icon="icon-[fa--floppy-o]" name="Save|Guardar"
        label="Saves the company parameters on this panel.|Guardar los parámetros de la empresa."
        onClick={() => saveCompanyParameters(service.empresa)} />
    </div>
    <div class="grid grid-cols-24 gap-10 content-start">
      <Input css="col-span-24" label="Name|Nombre" save="Name" bind:saveOn={service.empresa}
        required={true} />
      <Input css="col-span-24" label="Legal Name|Razón Social" save="LegalName"
        bind:saveOn={service.empresa} required={true} />
      <Input css="col-span-12" label="RUC" save="RUC" bind:saveOn={service.empresa}
        required={true} />
      <Input css="col-span-12" label="Phone|Teléfono" save="Phone" bind:saveOn={service.empresa} />
      <Input css="col-span-24" label="Email|Correo Electrónico" save="Email"
        bind:saveOn={service.empresa} />
      <Input css="col-span-24" label="Representative|Representante" save="Representative"
        bind:saveOn={service.empresa} />
      <Input css="col-span-24" label="Legal Address|Dirección Legal" save="Address"
        bind:saveOn={service.empresa} />
      <SearchSelect css="col-span-24"
        label="Department | Province | District|Departamento | Provincia | Distrito"
        keyId="ID" keyName="_nombre" options={citiesService.distritos}
        save="CityID" bind:saveOn={service.empresa}
      />
    </div>
    <!-- Tax setup: the cash flow forecast sizes and dates the monthly taxes with it. Same record
         and same Save as the fields above. -->
    <div class="h4 ff-bold mt-16 mb-8"><T text="Tax Setup|Configuración Tributaria" /></div>
    <div class="grid grid-cols-24 gap-10 content-start" aria-label="Company tax setup">
      <SearchSelect css="col-span-24 md:col-span-12" label="Tax Regime|Régimen Tributario"
        keyId="id" keyName="name" options={taxRegimeOptions}
        save="TaxRegime" bind:saveOn={service.empresa}
        onChange={(regime) => {
          service.empresa.IncomeTaxPrepaymentRate = defaultIncomeTaxPrepaymentRate(regime?.id || 0)
        }}
      />
      {#if service.empresa.TaxRegime === TAX_REGIME_NRUS}
        <SearchSelect css="col-span-24 md:col-span-12" label="NRUS Category|Categoría NRUS"
          keyId="id" keyName="name" options={nrusCategoryOptions}
          save="NrusCategory" bind:saveOn={service.empresa} />
      {:else}
        <Input css="col-span-24 md:col-span-12" label="Income Tax Prepayment (%)|Pago a Cuenta Renta (%)"
          save="IncomeTaxPrepaymentRate" type="number" baseDecimals={2} bind:saveOn={service.empresa} />
      {/if}
      <Checkbox css="col-span-24" save="IsGoodTaxpayer" bind:saveOn={service.empresa}
        label="Good taxpayer (own column in the SUNAT calendar)|Buen contribuyente (columna propia en el cronograma SUNAT)" />
    </div>
  </section>
  <!-- The flags sit beside the parameters because they are the same record and the same Save:
       the invoicing panels that used to be here write through their own endpoints and moved to
       their own tab. -->
  <div class="col-span-24 lg:col-span-12">
    <CompanyFlagsPanel {service} />
  </div>
</div>
