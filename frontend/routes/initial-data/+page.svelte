<script lang="ts">
  import { untrack } from "svelte";
  import Input from '$components/form/Input.svelte';
  import SearchSelect from '$components/form/SearchSelect.svelte';
  import Button from '$components/buttons/Button.svelte';
  import T from '$components/misc/T.svelte';
  import { Loading, Notify } from '$libs/helpers';
  import { Env } from '$core/env';
  import { tr } from '$core/store.svelte';
  import { WarehousesService, CountryCitiesService } from '../business/branches-warehouses/branches-warehouses.svelte';
  import { initialDataDefaults, postInitialData } from './initial-data.svelte';

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
      Notify.failure(error as string)
      Loading.remove()
      isSaving = false
      return
    }

    Loading.remove()
    isSaving = false
    Notify.success(tr("Initial data saved.|Datos iniciales guardados."))
    Env.navigate("/")
  }
</script>

<div class="flex items-center justify-center min-h-screen initial-data-bg px-8 py-20">
  <div class="initial-data-card">
    <div class="initial-data-tt flex items-center text-xl">
      <T text="Initial Data|Datos Iniciales" />
    </div>
    <div class="initial-data-logo-c relative mb-2">
      <img class="w-full h-full" src="/images/genix_logo.svg" alt="Genix Logo" />
    </div>
    <p class="text-center text-gray-600 mb-16">
      <T text="Your company needs a branch, a warehouse and a cash register to start operating. You can accept the suggested names.|Su empresa necesita una sede, un almacén y una caja para poder operar. Puede aceptar los nombres sugeridos." />
    </p>

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
        <Button color="blue" icon="icon-[fa--check]" disabled={isSaving}
          name="Accept and Continue|Aceptar y Continuar"
          label="Creates the branch, warehouse and cash register with these values."
          onClick={saveInitialData}
        />
      </div>
    {/if}
  </div>
</div>

<style>
  .initial-data-bg {
    background-color: #eef0f8;
  }

  .initial-data-card {
    width: 40rem;
    max-width: 92vw;
    background-color: white;
    box-shadow: rgba(17, 17, 26, 0.1) 0px 4px 16px, rgba(17, 17, 26, 0.1) 0px 8px 24px;
    border-radius: 14px;
    position: relative;
    border-top: 4px solid #5759b1;
    padding: 2rem;
  }

  .initial-data-tt {
    position: absolute;
    top: -2.2rem;
    height: 2.2rem;
    left: 2rem;
    padding: 0 11px;
    background-color: #5759b1;
    color: white;
    border-radius: 11px 11px 0 0;
  }

  .initial-data-logo-c {
    height: 6rem;
  }

  .initial-data-logo-c > img {
    object-fit: contain;
  }
</style>
