<script lang="ts">
  import { untrack } from "svelte";
  import Input from '$components/form/Input.svelte';
  import SearchSelect from '$components/form/SearchSelect.svelte';
  import Button from '$components/buttons/Button.svelte';
  import { Loading, Notify } from '$libs/helpers';
  import { tr } from '$core/store.svelte';
  import { WarehousesService, CountryCitiesService } from '../business/branches-warehouses/branches-warehouses.svelte';
  import { initialDataDefaults, postInitialData } from './initial-data.svelte';

  interface Props {
    onSaved: () => void
  }

  let { onSaved }: Props = $props();

  const warehousesService = new WarehousesService()
  const paisCiudadesService = new CountryCitiesService(true)

  let form = $state(initialDataDefaults())
  let isSaving = $state(false)

  // A company may already have a site and only be missing the warehouse or the cash bank; in that
  // case the existing site is reused instead of creating a duplicate "Principal".
  $effect(() => {
    const sedes = warehousesService.Sedes
    untrack(() => {
      if(sedes.length > 0 && !form.SiteID){ form.SiteID = sedes[0].ID }
    })
  })

  const isCreatingSite = $derived(!form.SiteID)

  const saveInitialData = async () => {
    if(isCreatingSite){
      if((form.SiteName||"").length < 4){
        Notify.failure(tr("The branch name must be at least 4 characters.|El nombre de la sede debe tener al menos 4 caracteres."))
        return
      }
      if((form.SiteAddress||"").length < 4){
        Notify.failure(tr("The branch address must be at least 4 characters.|La dirección de la sede debe tener al menos 4 caracteres."))
        return
      }
      if(!form.CityID){
        Notify.failure(tr("Please select a city.|Debe seleccionar una ciudad."))
        return
      }
    }
    if((form.WarehouseName||"").length < 4){
      Notify.failure(tr("The warehouse name must be at least 4 characters.|El nombre del almacén debe tener al menos 4 caracteres."))
      return
    }
    if((form.CashBankName||"").length < 4){
      Notify.failure(tr("The cash register name must be at least 4 characters.|El nombre de la caja debe tener al menos 4 caracteres."))
      return
    }

    isSaving = true
    Loading.standard(tr("Saving initial data...|Guardando los datos iniciales..."))
    try {
      await postInitialData(form)
    } catch (error) {
      // No Notify here: the http layer already raised the toast with the server's message, and this
      // rejects with the raw response object, which Notiflix renders as an empty box.
      console.error("[initial-data] Save failed:", error)
      Loading.remove()
      isSaving = false
      return
    }

    Loading.remove()
    isSaving = false
    Notify.success(tr("Initial data saved.|Datos iniciales guardados."))
    onSaved()
  }
</script>

{#if warehousesService.isReady > 0}
  <div class="grid grid-cols-24 gap-10">
    {#if isCreatingSite}
      <Input bind:saveOn={form} save="SiteName" required={true}
        css="col-span-24 md:col-span-10" label="Branch|Sede"
      />
      <Input bind:saveOn={form} save="SiteAddress" required={true}
        css="col-span-24 md:col-span-14" label="Address|Dirección"
      />
      <SearchSelect bind:saveOn={form} save="CityID" required={true}
        css="col-span-24" label="Department | Province | District|Departamento | Provincia | Distrito"
        keyId="ID" keyName="_nombre" options={paisCiudadesService.distritos}
      />
    {:else}
      <SearchSelect bind:saveOn={form} save="SiteID" required={true}
        css="col-span-24" label="Branch|Sede"
        keyId="ID" keyName="Name" options={warehousesService.Sedes}
      />
    {/if}
    <Input bind:saveOn={form} save="WarehouseName" required={true}
      css="col-span-24 md:col-span-12" label="Warehouse|Almacén"
    />
    <Input bind:saveOn={form} save="CashBankName" required={true}
      css="col-span-24 md:col-span-12" label="Cash Register|Caja"
    />
  </div>

  <div class="flex justify-center mt-20">
    <Button color="blue" icon="icon-[fa--floppy-o]" disabled={isSaving}
      name="Continue|Continuar"
      label="Creates the branch, warehouse and cash register with these values."
      onClick={saveInitialData}
    />
  </div>
{/if}
