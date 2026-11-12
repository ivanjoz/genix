<script lang="ts">
  import Input from "#components/form/Input.svelte";
  import CiudadesSelector from "#ecommerce/components/CiudadesSelector.svelte";
  import type { IWebCustomerAddress } from "#ecommerce/services/web-checkout.ts";

  // A delivery address: Departamento / Provincia / Distrito, street and reference. Used by the
  // cart's "Nueva dirección" and by Mis Direcciones. `address` must be a $state object.
  interface Props {
    address: Omit<IWebCustomerAddress, 'ID'> & { ID?: number }
    onCityChange?: () => void
  }

  const { address, onCityChange }: Props = $props();
</script>

<div class="grid grid-cols-12 gap-12">
  <CiudadesSelector saveOn={address} save="CityID" css="col-span-12 md:col-span-4"
    onChange={() => onCityChange?.()}
  />
  <Input label="Dirección" css="col-span-12 md:col-span-6" saveOn={address} save="Address" required={true} />
  <Input label="Referencia" css="col-span-12 md:col-span-6" saveOn={address} save="Reference" />
</div>
