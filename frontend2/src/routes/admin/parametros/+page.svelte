<script lang="ts">
  import Input from "$components/Input.svelte"
  import Page from "$components/Page.svelte"
  import { Notify } from "$core/helpers"
    import { Loading } from "notiflix";
  import { EmpresaParametrosService, postEmpresaParametros } from "./empresas.svelte"

  const service = new EmpresaParametrosService()
  
  async function saveEmpresa() {
    const form = service.empresa
    if((form.RUC||"").length === 0 || (form.Nombre||"").length === 0 || 
      (form.RazonSocial||"").length === 0 || (form.Email||"").length === 0 ){
      Notify.failure("Faltan datos a guardar.")
      return
    }

    Loading.standard("Guardando...")
    try {
      await postEmpresaParametros(form)
      Notify.success("Datos guardados correctamente")
    } catch (error) {
      // Error handled by POST
    }
    Loading.remove()
  }
</script>

<Page title="Parámetros">
    <div class="flex justify-between items-center mb-8">
      <div><div class="h3 ff-bold">Parámetros de la Empresa</div></div>
      <div class="flex items-center">
        <button class="bx-blue" onclick={ev => {
          ev.stopPropagation()
          saveEmpresa()
        }}>
          <i class="icon-floppy mr-2"></i>Guardar
        </button>
      </div>
    </div>
    <div class="w-full grid grid-cols-1 lg:grid-cols-[4fr_3fr] gap-4">
      <div class="grid grid-cols-24 gap-10 content-start">
        <Input css="col-span-14" label="Nombre" save="Nombre" bind:saveOn={service.empresa}
          required={true} />
        <Input css="col-span-10" label="RUC" save="RUC" bind:saveOn={service.empresa} 
          required={true} />
        <Input css="col-span-14" label="Razón Social" save="RazonSocial" 
          bind:saveOn={service.empresa} required={true} />
        <Input css="col-span-10" label="Teléfono" save="Telefono" bind:saveOn={service.empresa} />
        <Input css="col-span-14" label="Correo Electrónico" save="Email" 
          bind:saveOn={service.empresa} required={true} />
        <Input css="col-span-10" label="Representante" save="Representante" bind:saveOn={service.empresa} />
        <Input css="col-span-14" label="Dirección Legal" save="Direccion" 
          bind:saveOn={service.empresa} />
        <Input css="col-span-10" label="Ciudad" save="Ciudad" bind:saveOn={service.empresa} />
        
        <div class="col-span-24 mb-2"><div class="h3 ff-bold">Ecommerce</div></div>
        <Input css="col-span-12" label="Llave Pública (Pruebas)"
          save="LlavePubDev" bind:saveOn={service.empresa.CulquiConfig} />
        <Input css="col-span-12" label="Llave Privada (Pruebas)"
          save="LlaveDev" bind:saveOn={service.empresa.CulquiConfig} />
        <Input css="col-span-12" label="Llave Pública (Live)"
          save="LlavePubLive" bind:saveOn={service.empresa.CulquiConfig} />
        <Input css="col-span-12" label="Llave Privada (Live)"
          save="LlaveLive" bind:saveOn={service.empresa.CulquiConfig} />
        <Input css="col-span-12" label="Culqui LLave RSA ID" 
          save="RsaKeyID" bind:saveOn={service.empresa.CulquiConfig} />
        <Input css="col-span-12" label="Culqui LLave RSA" 
          save="RsaKey" bind:saveOn={service.empresa.CulquiConfig} />
      </div>
      <div class="" style="margin-left: 16px">
        <div class="w-full py-10 px-12 bg-white rounded shadow-sm">
          <div class="w-full mb-8">Parámetros SMTP para notificaciones</div>
          <div class="grid grid-cols-24 gap-4">
            <Input css="col-span-12" label="Host" save="Host" bind:saveOn={service.empresa.SmtpConfig} />
            <Input css="col-span-12" label="Port" save="Port" bind:saveOn={service.empresa.SmtpConfig} type="number" />
            <Input css="col-span-12" label="Usuario" save="User" bind:saveOn={service.empresa.SmtpConfig} />
            <Input css="col-span-12" label="Password" save="Password" bind:saveOn={service.empresa.SmtpConfig} />
            <Input css="col-span-24" label="Email" save="Email" bind:saveOn={service.empresa.SmtpConfig} />
          </div>
        </div>
      </div>
    </div>
</Page>
