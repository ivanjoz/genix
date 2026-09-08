<script lang="ts">
import Input from '$components/form/Input.svelte';
import Button from '$components/buttons/Button.svelte';
import T from '$components/misc/T.svelte';
import InvoiceSeriesTable from './InvoiceSeriesTable.svelte';
import CompanySecretsPanel from './CompanySecretsPanel.svelte';
import { saveCompanyParameters, type EmpresaParametrosService } from "./empresas.svelte"

  const { service }: { service: EmpresaParametrosService } = $props()
</script>

<div class="grid grid-cols-24 gap-14 items-start">
  <section class="col-span-24 lg:col-span-12 rounded-[12px] border border-slate-200 bg-white p-16 shadow-sm"
    aria-label="Company parameters form with name, RUC, legal address and contact fields">
    <!-- Save sits inside this panel, not above the page: the series alongside it save
         through their own endpoint, so a button at the top would claim to write both. -->
    <div class="flex items-center gap-10 mb-14">
      <div class="h3 ff-bold flex-1 min-w-0"><T text="Company Parameters|Parámetros de la Empresa" /></div>
      <Button color="blue" icon="icon-[fa--floppy-o]" name="Save|Guardar"
        label="Saves the company parameters on this panel."
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
      <Input css="col-span-24" label="City|Ciudad" save="City" bind:saveOn={service.empresa} />
    </div>
  </section>
  <!-- The invoicing panels stack in their own column: the certificate belongs under the
       series it signs, not under the company form on the other side of the page. -->
  <div class="col-span-24 lg:col-span-12 flex flex-col gap-14">
    <InvoiceSeriesTable company={service.empresa} />
    <CompanySecretsPanel />
  </div>
</div>
