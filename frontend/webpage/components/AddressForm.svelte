<script lang="ts">
  import Input from "#components/form/Input.svelte";
  import CiudadesSelector from "#ecommerce/components/CiudadesSelector.svelte";
  import type { IWebCustomerAddress } from "#ecommerce/services/web-checkout.ts";

  // A delivery address: Departamento / Provincia / Distrito, street and reference. Used by the
  // cart's "Nueva dirección" and by Mis Direcciones. `address` must be a $state object.
  // `onChange` fires when a city is picked or a text field loses focus with a new value.
  interface Props {
    address: Omit<IWebCustomerAddress, 'ID'> & { ID?: number }
    onChange?: () => void
  }

  const { address, onChange }: Props = $props();
</script>

<div class="grid grid-cols-12 gap-12">
  <CiudadesSelector saveOn={address} save="CityID" css="col-span-12 md:col-span-4" {onChange} />
  <Input label="Dirección" css="col-span-12 md:col-span-6" saveOn={address} save="Address" required={true} {onChange} />
  <Input label="Referencia" css="col-span-12 md:col-span-6" saveOn={address} save="Reference" {onChange} />
</div>
