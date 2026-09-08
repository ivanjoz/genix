<script lang="ts">
import Input from '$components/form/Input.svelte';
import Button from '$components/buttons/Button.svelte';
import T from '$components/misc/T.svelte';
import { saveCompanyParameters, type EmpresaParametrosService } from "./empresas.svelte"

  const { service }: { service: EmpresaParametrosService } = $props()
</script>

<div class="flex justify-end items-center mb-8" aria-label="Store configuration header with save button">
  <Button color="blue" icon="icon-[fa--floppy-o]" name="Save|Guardar" label="Saves the online store configuration." onClick={() => saveCompanyParameters(service.empresa)} />
</div>
<div class="grid grid-cols-24 gap-14">
  <section class="col-span-24 rounded-[12px] border border-slate-200 bg-white p-16 shadow-sm"
    aria-label="Culqi payment gateway credentials for the online store, split into test and live environments">
    <div class="h3 ff-bold mb-4"><T text="Culqi Configuration|Configuración Culqi" /></div>
    <div class="text-[14px] text-slate-500 mb-14">
      <T text="Payment gateway credentials used by the online store checkout.|Credenciales de la pasarela de pago que usa el checkout de la tienda." />
    </div>

    <div class="grid grid-cols-24 gap-14">
      <!-- Test and live credentials are separated because using one where the other belongs is the
           failure this page exists to prevent: live keys in a test checkout charge real cards. -->
      <div class="col-span-24 lg:col-span-12 rounded-[10px] border border-amber-200 bg-amber-50/40 p-14">
        <div class="flex items-center gap-8 mb-12">
          <span class="rounded-[6px] bg-amber-100 text-amber-800 px-8 py-2 text-[13px] ff-bold">
            <T text="TEST|PRUEBAS" />
          </span>
          <span class="text-[16px] ff-bold"><T text="Culqi Test|Culqi Pruebas" /></span>
        </div>
        <div class="grid grid-cols-24 gap-10 content-start">
          <Input css="col-span-24" label="Public Key|Llave Pública"
            save="PubKeyDev" bind:saveOn={service.empresa.CulqiConfig} />
          <Input css="col-span-24" label="Private Key|Llave Privada"
            save="KeyDev" bind:saveOn={service.empresa.CulqiConfig} type="password" />
        </div>
      </div>

      <div class="col-span-24 lg:col-span-12 rounded-[10px] border border-emerald-200 bg-emerald-50/40 p-14">
        <div class="flex items-center gap-8 mb-12">
          <span class="rounded-[6px] bg-emerald-100 text-emerald-800 px-8 py-2 text-[13px] ff-bold">
            <T text="LIVE|PRODUCCIÓN" />
          </span>
          <span class="text-[16px] ff-bold"><T text="Culqi Live|Culqi Live" /></span>
        </div>
        <div class="grid grid-cols-24 gap-10 content-start">
          <Input css="col-span-24" label="Public Key|Llave Pública"
            save="PubKeyLive" bind:saveOn={service.empresa.CulqiConfig} />
          <Input css="col-span-24" label="Private Key|Llave Privada"
            save="KeyLive" bind:saveOn={service.empresa.CulqiConfig} type="password" />
        </div>
      </div>

      <!-- One RSA pair, not one per environment: the company record stores a single key, so it sits
           outside both boxes instead of being duplicated into each. -->
      <div class="col-span-24 rounded-[10px] border border-slate-200 p-14">
        <div class="text-[16px] ff-bold mb-4"><T text="RSA Encryption|Encriptación RSA" /></div>
        <div class="text-[14px] text-slate-500 mb-12">
          <T text="Optional. Only needed when card data is sent encrypted.|Opcional. Sólo se necesita cuando los datos de la tarjeta se envían encriptados." />
        </div>
        <!-- The id sits on its own row: a field bottom-aligns in its grid cell by design
             (field-shell's margin-top:auto), so pairing it with the taller textarea would
             drop it to the textarea's baseline. -->
        <div class="grid grid-cols-24 gap-10">
          <Input css="col-span-24 lg:col-span-10" label="RSA Key ID|ID de Llave RSA"
            save="RsaKeyID" bind:saveOn={service.empresa.CulqiConfig} />
          <Input css="col-span-24" label="RSA Key|Llave RSA"
            save="RsaKey" bind:saveOn={service.empresa.CulqiConfig} useTextArea={true} rows={3} />
        </div>
      </div>
    </div>
  </section>
</div>
