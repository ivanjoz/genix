<script lang="ts">
import Input from '$components/form/Input.svelte';
import { Notify } from '$libs/helpers';
import Button from '$components/buttons/Button.svelte';
import { tr } from '$core/store.svelte';
import T from '$components/misc/T.svelte';
import pkg from 'notiflix'
const { Loading } = pkg;
import { EmpresaParametrosService, postEmpresaParametros, type ICompany } from "./empresas.svelte"

  const service = new EmpresaParametrosService()

  // Email is deliberately absent: seeded and imported companies carry none, and the global email
  // index on the companies table tolerates a blank value. PostEmpresaParametros enforces the same
  // three. Labels are repeated here so a refusal can name the field the user has to go fix.
  const requiredFields: [keyof ICompany, string][] = [
    ["Name", "Name|Nombre"],
    ["RUC", "RUC"],
    ["LegalName", "Legal Name|Razón Social"],
  ]

  async function saveEmpresa() {
    const form = service.empresa
    const missing = requiredFields
      .filter(([field]) => ((form[field] as string) || "").trim().length === 0)
      .map(([, label]) => tr(label))

    if (missing.length > 0) {
      Notify.failure(`${tr("Missing required data:|Faltan datos a guardar:")} ${missing.join(", ")}`)
      return
    }

    Loading.standard(tr("Saving...|Guardando..."))
    try {
      await postEmpresaParametros(form)
      Notify.success(tr("Data saved successfully|Datos guardados correctamente"))
    } catch (error) {
      // Error handled by POST
    }
    Loading.remove()
  }
</script>

<div class="flex justify-end items-center mb-8" aria-label="Company parameters header with save button">
  <Button color="blue" icon="icon-[fa--floppy-o]" name="Save|Guardar" label="Saves all company parameter changes." onClick={saveEmpresa} />
</div>
<div class="grid grid-cols-24 gap-14">
  <section class="col-span-24 lg:col-span-12 rounded-[12px] border border-slate-200 bg-white p-16 shadow-sm"
    aria-label="Company parameters form with name, RUC, legal address and contact fields">
    <div class="h3 ff-bold mb-14"><T text="Company Parameters|Parámetros de la Empresa" /></div>
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

  <section class="col-span-24 lg:col-span-12 rounded-[12px] border border-slate-200 bg-white p-16 shadow-sm"
    aria-label="Culqi payment gateway keys for the online store">
    <div class="h3 ff-bold mb-14"><T text="Culqi Configuration|Configuración Culqi" /></div>
    <div class="grid grid-cols-24 gap-10 content-start">
      <Input css="col-span-24" label="Public Key (Test)|Llave Pública (Pruebas)"
        save="PubKeyDev" bind:saveOn={service.empresa.CulqiConfig} />
      <Input css="col-span-24" label="Private Key (Test)|Llave Privada (Pruebas)"
        save="KeyDev" bind:saveOn={service.empresa.CulqiConfig} />
      <Input css="col-span-24" label="Public Key (Live)|Llave Pública (Live)"
        save="PubKeyLive" bind:saveOn={service.empresa.CulqiConfig} />
      <Input css="col-span-24" label="Private Key (Live)|Llave Privada (Live)"
        save="KeyLive" bind:saveOn={service.empresa.CulqiConfig} type="password" />
      <Input css="col-span-24" label="Culqi RSA Key ID"
        save="RsaKeyID" bind:saveOn={service.empresa.CulqiConfig} />
      <Input css="col-span-24" label="Culqi RSA Key"
        save="RsaKey" bind:saveOn={service.empresa.CulqiConfig} />
    </div>
  </section>
</div>
