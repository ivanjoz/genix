<script lang="ts">
import Input from '$components/form/Input.svelte';
import Modal from '$components/layers/Modal.svelte';
import Layer from '$components/layers/Layer.svelte';
import Page from '$domain/Page.svelte';
import SearchSelect from '$components/form/SearchSelect.svelte';
import VTable from '$components/vTable/VTable.svelte';
import type { ITableColumn } from '$components/vTable/types';
import { Loading, Notify, formatTime } from '$libs/helpers';
import FilterInput from '$components/form/FilterInput.svelte';
import Button from '$components/buttons/Button.svelte';
import { Core, tr } from '$core/store.svelte';
import T from '$components/misc/T.svelte';
import AlmacenLayoutEditor from './AlmacenLayoutEditor.svelte';

import {
    AlmacenesService,
    PaisCiudadesService,
    postSede,
    postAlmacen,
    type ISite,
    type IWarehouse,
    type IWarehouseLayout
  } from "./sedes-almacenes.svelte"

  const almacenesService = new AlmacenesService()
  const paisCiudadesService = new PaisCiudadesService(true)

  const pageOptions = [{ id: 1, name: "Branches|Sedes" }, { id: 2, name: "Warehouses|Almacenes" }]

  let filterText = $state("")
  let sedeForm = $state({} as ISite)
  let almacenForm = $state({} as IWarehouse)

  const saveSede = async (isDelete?: boolean) => {
    const form = sedeForm
    if((form.Name?.length||0) < 4 || (form.Direccion?.length||0) < 4){
      Notify.failure(tr("Name and address must be at least 4 characters.|El nombre y la dirección deben tener al menos 4 caracteres."))
      return
    }

    console.log("guardando sede::", form)

    Loading.standard(tr("Creating/Updating Branch...|Creando /Actualizando Sede..."))
    try {
      var result = await postSede(form)
    } catch (error) {
      Notify.failure(error as string)
      Loading.remove()
      return
    }

    let sedes_ = [...almacenesService.Sedes]

    if(form.ID){
      const selected = almacenesService.Sedes.find(x => x.ID === form.ID)
      if(selected){ Object.assign(selected, form) }
      if(isDelete){ sedes_ = sedes_.filter(x => x.ID !== form.ID) }
    } else {
      form.ID = result.ID
      sedes_.unshift(form)
    }

    almacenesService.Sedes = sedes_
    Core.closeModal(1)
    Loading.remove()
  }

  const saveAlmacen = async (isDelete?: boolean) => {
    const form = almacenForm
    if((form.Name?.length||0) < 4){
      Notify.failure(tr("Name must be at least 4 characters.|El nombre debe tener al menos 4 caracteres."))
      return
    } else if(!form.SiteID){
      Notify.failure(tr("Please select a branch.|Debe seleccionar una sede."))
      return
    }

    // Process layout bloques
    for(let layout of form.Layout||[]){
      layout.Bloques = []
      for(let key in layout){
        if(key.substring(0,3) === 'xy_'){
          const [_, rw, co] = key.split("_")
          layout.Bloques.push({
            nm: layout[key] as string,
            rw: parseInt(rw),
            co: parseInt(co)
          })
        }
      }
    }

    console.log("guardando almacen::", form)

    Loading.standard(tr("Creating/Updating Warehouse...|Creando /Actualizando Almacén..."))
    try {
      var result = await postAlmacen(form)
    } catch (error) {
      Notify.failure(error as string)
      Loading.remove()
      return
    }

    let almacenes_ = [...almacenesService.Almacenes]

    if(form.ID){
      const selected = almacenesService.Almacenes.find(x => x.ID === form.ID)
      if(selected){ Object.assign(selected, form) }
      if(isDelete){ almacenes_ = almacenes_.filter(x => x.ID !== form.ID) }
    } else {
      form.ID = result.ID
      almacenes_.unshift(form)
    }

    almacenesService.Almacenes = almacenes_
    Core.closeModal(2)
    Core.hideSideLayer()
    Loading.remove()
  }

  const sedesColumns: ITableColumn<ISite>[] = [
    {
      header: "ID",
      headerCss: "w-32",
      css: "text-center text-purple-600 px-6",
      getValue: e => e.ID
    },
    {
      header: "Name|Nombre",
      css: "px-6",
      getValue: e => e.Name
    },
    {
      header: "Address|Dirección",
      css: "px-6",
      getValue: e => e.Direccion
    },
    {
      header: "City|Ciudad",
      getValue: e => {
        if(!e.Ciudad){ return "" }
        const arr = e.Ciudad.split("|")
        return arr[1] + " > " + arr[0]
      }
    },
    {
      header: "Updated|Actualizado",
      headerCss: "w-144",
      css: "whitespace-nowrap px-6",
      getValue: e => formatTime(e.upd, "Y-m-d h:n") as string
    },
    {
      header: "...",
      headerCss: "w-32",
      css: "text-center px-6",
      buttonEditHandler: (e) => {
        sedeForm = {...e}
        Core.openModal(1)
      }
    }
  ]

  const almacenesColumns: ITableColumn<IWarehouse>[] = [
    {
      header: "ID",
      headerCss: "w-32",
      css: "text-center text-purple-600 px-6",
      getValue: e => e.ID
    },
    {
      header: "Branch|Sede",
      css: "px-6",
      getValue: e => {
        const sede = almacenesService.SedesMap.get(e.SiteID)
        return sede?.Name || `Sede-${e.SiteID}`
      }
    },
    {
      header: "Name|Nombre",
      css: "px-6",
      getValue: e => e.Name
    },
    {
      header: "Layout",
      id: "layout",
      css: "px-6",
      getValue: e => ""
    },
    {
      header: "Status|Estado",
      getValue: e => e.ss
    },
    {
      header: "Updated|Actualizado",
      headerCss: "w-144",
      css: "whitespace-nowrap px-6",
      getValue: e => formatTime(e.upd, "Y-m-d h:n") as string
    },
    {
      header: "...",
      headerCss: "w-32",
      css: "text-center px-6",
      buttonEditHandler: (e) => {
        almacenForm = JSON.parse(JSON.stringify(e))
        Core.openModal(2)
      }
    }
  ]

  const filteredSedes = $derived.by(() => {
    if (!filterText) return almacenesService.Sedes
    const text = filterText.toLowerCase()
    return almacenesService.Sedes.filter(e => {
      return e.Name?.toLowerCase().includes(text) ||
             e.Direccion?.toLowerCase().includes(text) ||
             e.Ciudad?.toLowerCase().includes(text)
    })
  })

  const filteredAlmacenes = $derived.by(() => {
    if (!filterText) return almacenesService.Almacenes
    const text = filterText.toLowerCase()
    return almacenesService.Almacenes.filter(e => {
      const sede = almacenesService.SedesMap.get(e.SiteID)
      return e.Name?.toLowerCase().includes(text) ||
             sede?.Name?.toLowerCase().includes(text)
    })
  })

  const handleLayoutEdit = (almacen: IWarehouse) => {
    almacenForm = JSON.parse(JSON.stringify(almacen))
    // Process layout to convert Bloques to xy_ properties
    for(let layout of almacenForm.Layout||[]){
      for(let e of layout.Bloques||[]){
        layout[`xy_${e.rw}_${e.co}`] = e.nm
      }
    }
    console.log("ejecutando open side")
    Core.openSideLayer(1)
  }
</script>

<Page title="Branches & Warehouses|Sedes & Almacenes" options={pageOptions}>
  {#if Core.pageOptionSelected === 1 /* Sedes */}
    <div class="flex items-center justify-between mb-6" aria-label="Sedes list toolbar with search filter and create button">
      <FilterInput bind:value={filterText} css="mr-16 w-256" />
      <div class="flex items-center">
        <Button color="green" icon="icon-[fa--plus]" label="Opens the modal to create a new business location (sede)." onClick={() => {
          sedeForm = { ss: 1 } as ISite
          Core.openModal(1)
        }} />
      </div>
    </div>
    <VTable css="w-full"
      maxHeight="calc(80vh - 13rem)"
      columns={sedesColumns}
      data={filteredSedes}
    />
  {/if}

  {#if Core.pageOptionSelected === 2 /* Almacenes */}
    <div class="w-full">
      <div class="flex items-center justify-between mb-6" aria-label="Almacenes list toolbar with search filter and create button">
        <FilterInput bind:value={filterText} css="mr-16 w-256" />
        <div class="flex items-center">
          <Button color="green" icon="icon-[fa--plus]" label="Opens the modal to create a new warehouse linked to a sede." onClick={() => {
            almacenForm = { ID: 0, SiteID: 0, Name: "", Description: "", ss: 1, upd: 0, Layout: [] }
            Core.openModal(2)
          }} />
        </div>
      </div>
      <VTable
        css="w-full"
        maxHeight="calc(80vh - 13rem)"
        columns={almacenesColumns}
        data={filteredAlmacenes}
      >
        {#snippet cellRenderer(record: IWarehouse, col: ITableColumn<IWarehouse>)}
          {#if col.id === "layout"}
            <div class="w-full flex items-center justify-between">
              {#if record.Layout && record.Layout.length > 0}
                {@const avgCols = record.Layout.reduce((sum, x) => sum + (x.ColCant || 0), 0) / record.Layout.length}
                {@const avgRows = record.Layout.reduce((sum, x) => sum + (x.RowCant || 0), 0) / record.Layout.length}
                <div class="flex items-center">
                  <div class="ff-bold h3">{record.Layout.length}</div>
                  <i class="icon-[fa--folder]"></i>
                  <div class="mr-4 ml-4 h6 text-slate-500">X</div>
                  <div class="ff-bold h3">{avgCols.toFixed(0)}</div>
                  <i class="icon-[fa--th-large]"></i>
                  <div class="mr-4 ml-4 h6 text-slate-500">X</div>
                  <div class="ff-bold h3">{avgRows.toFixed(0)}</div>
                  <i class="icon-[fa--cube]"></i>
                </div>
              {:else}
                <div></div>
              {/if}
              <button class="bnr2 b-blue b-card-1"
                aria-label="Editar Layout"
                onclick={() => handleLayoutEdit(record)}
              >
                <i class="icon-[fa--pencil]"></i>
              </button>
            </div>
          {/if}
        {/snippet}
      </VTable>
    </div>
  {/if}

  <!-- Sede Modal -->
  <Modal id={1} title={(sedeForm?.ID > 0 ? tr("Update|Actualizar") : tr("Create|Crear")) + " " + tr("Branch|Sede")}
    size={7} bodyCss="px-16 py-14"
    onSave={() => { saveSede() }}
    onDelete={sedeForm?.ID > 0 ? () => { saveSede(true) } : undefined}
  >
    <div class="grid grid-cols-24 gap-10" aria-label="Sede form with name, description, phone, address, and location">
      <Input bind:saveOn={sedeForm} save="Name"
        css="col-span-24 md:col-span-10" label="Name|Nombre" required={true}
        disabled={sedeForm?.ID > 0}
      />
      <Input bind:saveOn={sedeForm} save="Description"
        css="col-span-24 md:col-span-14" label="Description|Descripción"
      />
      <Input bind:saveOn={sedeForm} save="Telefono"
        css="col-span-24 md:col-span-10" label="Phone|Teléfono"
        disabled={sedeForm?.ID > 0}
      />
      <Input bind:saveOn={sedeForm} save="Direccion"
        css="col-span-24 md:col-span-14" label="Address|Dirección" required={true}
      />
      <SearchSelect bind:saveOn={sedeForm} save="CityID"
        css="col-span-24" label="Department | Province | District|Departamento | Provincia | Distrito"
        keyId="ID" keyName="_nombre" options={paisCiudadesService.distritos}
        required={true}
      />
    </div>
  </Modal>

  <!-- Almacen Modal -->
  <Modal id={2} title={(almacenForm?.ID > 0 ? tr("Update|Actualizar") : tr("Create|Crear")) + " " + tr("Warehouse|Almacén")}
    size={7} bodyCss="px-16 py-14"
    onSave={() => { saveAlmacen() }}
    onDelete={almacenForm?.ID > 0 ? () => { saveAlmacen(true) } : undefined}
  >
    <div class="grid grid-cols-24 gap-10" aria-label="Almacen form with sede, name, and description">
      <SearchSelect bind:saveOn={almacenForm} save="SiteID"
        css="col-span-24 md:col-span-12" label="Branch|Sede"
        keyId="ID" keyName="Name" options={almacenesService.Sedes}
        required={true}
      />
      <Input bind:saveOn={almacenForm} save="Name"
        css="col-span-24 md:col-span-12" label="Name|Nombre" required={true}
      />
      <Input bind:saveOn={almacenForm} save="Description"
        css="col-span-24" label="Description|Descripción"
      />
    </div>
  </Modal>

  <!-- Layout Side Layer -->
  <Layer id={1} type="side" title={"Layout " + (almacenForm?.Name || "-")}
    contentCss="p-0" css="px-8 py-8 md:px-14 md:py-10"
    titleCss="h2 ff-bold"
    onClose={() => {}}
    onSave={() => { saveAlmacen() }}
  >
    <AlmacenLayoutEditor bind:almacen={almacenForm} />
  </Layer>
</Page>
