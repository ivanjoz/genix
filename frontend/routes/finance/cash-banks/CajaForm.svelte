<script lang="ts">
import Input from '$components/form/Input.svelte'
import SearchSelect from '$components/form/SearchSelect.svelte'
import Checkbox from '$components/form/Checkbox.svelte'
import type { ISite } from '$routes/business/branches-warehouses/branches-warehouses.svelte'
import { cajaTipos, cajaMonedaTipos, type ICashBank } from './cajas.svelte'

let {
  form = $bindable(),
  sedes,
}: {
  form: ICashBank
  sedes: ISite[]
} = $props()
</script>

<div class="grid grid-cols-24 gap-10" aria-label="Cash register form with type, name, description, and sede">
  <SearchSelect bind:saveOn={form} save="Type" css="col-span-24 md:col-span-10"
    label="Type|Tipo" keyId="id" keyName="name" options={cajaTipos}
    placeholder="" required={true}
  />
  <Input bind:saveOn={form} save="Name"
    css="col-span-24 md:col-span-14" label="Name|Nombre" required={true}
  />
  <Input bind:saveOn={form} save="Description"
    css="col-span-24" label="Description|Descripción"
  />
  <SearchSelect bind:saveOn={form} save="SiteID"
    css="col-span-24 md:col-span-10" label="Branch|Sede" required={true} options={sedes}
    keyId="ID" keyName="Name"
  />
  <SearchSelect bind:saveOn={form} save="CurrencyType" css="col-span-24 md:col-span-10"
    label="Currency|Moneda" keyId="id" keyName="name" options={cajaMonedaTipos}
    placeholder="" required={true}
  />
  <div class="col-span-24 flex justify-between items-center">
    <div></div>
    <Checkbox label="Negative Balance|Saldo Negativo" bind:saveOn={form} save="Name" />
  </div>
</div>
