<script lang="ts">
import Input from '$components/form/Input.svelte'
import SearchSelect from '$components/form/SearchSelect.svelte'
import Checkbox from '$components/form/Checkbox.svelte'
import type { ISede } from '$routes/negocio/sedes-almacenes/sedes-almacenes.svelte'
import { cajaTipos, type ICaja } from './cajas.svelte'

let {
  form = $bindable(),
  sedes,
}: {
  form: ICaja
  sedes: ISede[]
} = $props()
</script>

<div class="grid grid-cols-24 gap-10" aria-label="Cash register form with type, name, description, and sede">
  <SearchSelect bind:saveOn={form} save="Tipo" css="col-span-24 md:col-span-10"
    label="Tipo" keyId="id" keyName="name" options={cajaTipos}
    placeholder="" required={true}
  />
  <Input bind:saveOn={form} save="Nombre"
    css="col-span-24 md:col-span-14" label="Nombre" required={true}
  />
  <Input bind:saveOn={form} save="Descripcion"
    css="col-span-24" label="Descripcion"
  />
  <SearchSelect bind:saveOn={form} save="SedeID"
    css="col-span-24 md:col-span-10" label="Sede" required={true} options={sedes}
    keyId="ID" keyName="Nombre"
  />
  <div class="col-span-24 flex justify-between items-center">
    <div></div>
    <Checkbox label="Saldo Negativo" bind:saveOn={form} save="Nombre" />
  </div>
</div>
