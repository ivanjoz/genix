<script lang="ts">
import Input from '$ui/components/Input';
import Modal from '$ui/components/Modal';
import Layer from '$ui/components/Layer';
import Page from '$ui/components/Page';
import SearchSelect from '$ui/components/SearchSelect';
import VTable from '$ui/components/VTable/index';
  import type { ITableColumn } from "$ui/VTable/types.ts"
import { Loading, Notify, formatTime } from '$core/lib/helpers';
import { throttle } from '$core/lib/helpers';
import { Core } from '$core/core/store.svelte';
  import AlmacenLayoutEditor from "./AlmacenLayoutEditor.svelte"
  import { 
    AlmacenesService, 
    PaisCiudadesService, 
    postSede, 
    postAlmacen,
    type ISede, 
    type IAlmacen,
    type IAlmacenLayout
  } from "./sedes-almacenes.svelte"

  const almacenesService = new AlmacenesService()
  const paisCiudadesService = new PaisCiudadesService()

  const pageOptions = [{ id: 1, name: "Sedes" }, { id: 2, name: "Almacenes" }]

  let filterText = $state("")
  let sedeForm = $state({} as ISede)
  let almacenForm = $state({} as IAlmacen)

  const saveSede = async (isDelete?: boolean) => {
    const form = sedeForm
    if((form.Nombre?.length||0) < 4 || (form.Direccion?.length||0) < 4){
      Notify.failure("El nombre y la dirección deben tener al menos 4 caracteres.")
      return
    }

    console.log("guardando sede::", form)

    Loading.standard("Creando /Actualizando Sede...")
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
    if((form.Nombre?.length||0) < 4){
      Notify.failure("El nombre debe tener al menos 4 caracteres.")
      return
    } else if(!form.SedeID){
      Notify.failure("Debe seleccionar una sede.")
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

    Loading.standard("Creando /Actualizando Almacén...")
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

  const sedesColumns: ITableColumn<ISede>[] = [
    {
      header: "ID",
      headerCss: "w-32",
      cellCss: "text-center text-purple-600 px-6",
      getValue: e => e.ID
    },
    {
      header: "Nombre",
      cellCss: "px-6",
      getValue: e => e.Nombre
    },
    {
      header: "Dirección",
      cellCss: "px-6",
      getValue: e => e.Direccion
    },
    {
      header: "Ciudad",
      getValue: e => {
        if(!e.Ciudad){ return "" }
        const arr = e.Ciudad.split("|")
        return arr[1] + " > " + arr[0]
      }
    },
    {
      header: "Actualizado",
      headerCss: "w-144",
      cellCss: "whitespace-nowrap px-6",
      getValue: e => formatTime(e.upd, "Y-m-d h:n") as string
    },
    {
      header: "...",
      headerCss: "w-32",
      cellCss: "text-center px-6",
      buttonEditHandler: (e) => {
        sedeForm = {...e}
        Core.openModal(1)
      }
    }
  ]

  const almacenesColumns: ITableColumn<IAlmacen>[] = [
    {
      header: "ID",
      headerCss: "w-32",
      cellCss: "text-center text-purple-600 px-6",
      getValue: e => e.ID
    },
    {
      header: "Sede",
      cellCss: "px-6",
      getValue: e => {
        const sede = almacenesService.SedesMap.get(e.SedeID)
        return sede?.Nombre || `Sede-${e.SedeID}`
      }
    },
    {
      header: "Nombre",
      cellCss: "px-6",
      getValue: e => e.Nombre
    },
    {
      header: "Layout",
      id: "layout",
      cellCss: "px-6",
      getValue: e => ""
    },
    {
      header: "Estado",
      getValue: e => e.ss
    },
    {
      header: "Actualizado",
      headerCss: "w-144",
      cellCss: "whitespace-nowrap px-6",
      getValue: e => formatTime(e.upd, "Y-m-d h:n") as string
    },
    {
      header: "...",
      headerCss: "w-32",
      cellCss: "text-center px-6",
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
      return e.Nombre?.toLowerCase().includes(text) ||
             e.Direccion?.toLowerCase().includes(text) ||
             e.Ciudad?.toLowerCase().includes(text)
    })
  })

  const filteredAlmacenes = $derived.by(() => {
    if (!filterText) return almacenesService.Almacenes
    const text = filterText.toLowerCase()
    return almacenesService.Almacenes.filter(e => {
      const sede = almacenesService.SedesMap.get(e.SedeID)
      return e.Nombre?.toLowerCase().includes(text) ||
             sede?.Nombre?.toLowerCase().includes(text)
    })
  })

  const handleLayoutEdit = (almacen: IAlmacen) => {
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

<Page title="Sedes & Almacenes" options={pageOptions}>
  {#if Core.pageOptionSelected === 1 /* Sedes */}
    <div class="flex items-center justify-between mb-6">
      <div class="i-search mr-16 w-256">
        <div><i class="icon-search"></i></div>
        <input class="w-full" autocomplete="off" type="text" onkeyup={ev => {
          ev.stopPropagation()
          throttle(() => {
            filterText = ((ev.target as any).value || "").toLowerCase().trim()
          }, 150)
        }} />
      </div>
      <div class="flex items-center">
        <button class="bx-green" aria-label="Agregar Sede" onclick={ev => {
          ev.stopPropagation()
          sedeForm = { ss: 1 } as ISede
          Core.openModal(1)
        }}>
          <i class="icon-plus"></i>
        </button>
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
      <div class="flex items-center justify-between mb-6">
        <div class="i-search mr-16 w-256">
          <div><i class="icon-search"></i></div>
          <input class="w-full" autocomplete="off" type="text" onkeyup={ev => {
            ev.stopPropagation()
            throttle(() => {
              filterText = ((ev.target as any).value || "").toLowerCase().trim()
            }, 150)
          }} />
        </div>
        <div class="flex items-center">
          <button class="bx-green" aria-label="Agregar Almacén" onclick={ev => {
            ev.stopPropagation()
            almacenForm = { ID: 0, SedeID: 0, Nombre: "", Descripcion: "", ss: 1, upd: 0, Layout: [] }
            Core.openModal(2)
          }}>
            <i class="icon-plus"></i>
          </button>
        </div>
      </div>
      <VTable 
        css="w-full"
        maxHeight="calc(80vh - 13rem)"
        columns={almacenesColumns}
        data={filteredAlmacenes}
      >
        {#snippet cellRenderer(record: IAlmacen, col: ITableColumn<IAlmacen>)}
          {#if col.id === "layout"}
            <div class="w-full flex items-center justify-between">
              {#if record.Layout && record.Layout.length > 0}
                {@const avgCols = record.Layout.reduce((sum, x) => sum + (x.ColCant || 0), 0) / record.Layout.length}
                {@const avgRows = record.Layout.reduce((sum, x) => sum + (x.RowCant || 0), 0) / record.Layout.length}
                <div class="flex items-center">
                  <div class="ff-bold h3">{record.Layout.length}</div>
                  <i class="icon-folder-empty"></i>
                  <div class="mr-4 ml-4 h6 text-slate-500">X</div>
                  <div class="ff-bold h3">{avgCols.toFixed(0)}</div>
                  <i class="icon-buffer"></i>
                  <div class="mr-4 ml-4 h6 text-slate-500">X</div>
                  <div class="ff-bold h3">{avgRows.toFixed(0)}</div>
                  <i class="icon-cube"></i>
                </div>
              {:else}
                <div></div>
              {/if}
              <button class="bnr2 b-blue b-card-1" 
                aria-label="Editar Layout"
                onclick={() => handleLayoutEdit(record)}
              >
                <i class="icon-pencil"></i>
              </button>
            </div>
          {/if}
        {/snippet}
      </VTable>
    </div>
  {/if}

  <!-- Sede Modal -->
  <Modal id={1} title={(sedeForm?.ID > 0 ? "Actualizar" : "Crear") + " Sede"} 
    size={7} bodyCss="px-16 py-14"
    onSave={() => { saveSede() }}
    onDelete={sedeForm?.ID > 0 ? () => { saveSede(true) } : undefined}
  >
    <div class="grid grid-cols-24 gap-10">
      <Input bind:saveOn={sedeForm} save="Nombre"
        css="col-span-24 md:col-span-10" label="Nombre" required={true}
        disabled={sedeForm?.ID > 0}
      />
      <Input bind:saveOn={sedeForm} save="Descripcion"
        css="col-span-24 md:col-span-14" label="Descripción"
      />
      <Input bind:saveOn={sedeForm} save="Telefono"
        css="col-span-24 md:col-span-10" label="Teléfono"
        disabled={sedeForm?.ID > 0}
      />
      <Input bind:saveOn={sedeForm} save="Direccion"
        css="col-span-24 md:col-span-14" label="Dirección" required={true}
      />
      <SearchSelect bind:saveOn={sedeForm} save="CiudadID"
        css="col-span-24" label="Departamento | Provincia | Distrito"
        keyId="ID" keyName="_nombre" options={paisCiudadesService.distritos}
        required={true}
      />
    </div>
  </Modal>

  <!-- Almacen Modal -->
  <Modal id={2} title={(almacenForm?.ID > 0 ? "Actualizar" : "Crear") + " Almacén"}
    size={7} bodyCss="px-16 py-14"
    onSave={() => { saveAlmacen() }}
    onDelete={almacenForm?.ID > 0 ? () => { saveAlmacen(true) } : undefined}
  >
    <div class="grid grid-cols-24 gap-10">
      <SearchSelect bind:saveOn={almacenForm} save="SedeID"
        css="col-span-24 md:col-span-12" label="Sede"
        keyId="ID" keyName="Nombre" options={almacenesService.Sedes}
        required={true}
      />
      <Input bind:saveOn={almacenForm} save="Nombre"
        css="col-span-24 md:col-span-12" label="Nombre" required={true}
      />
      <Input bind:saveOn={almacenForm} save="Descripcion"
        css="col-span-24" label="Descripción"
      />
    </div>
  </Modal>

  <!-- Layout Side Layer -->
  <Layer id={1} type="side" title={"Layout " + (almacenForm?.Nombre || "-")}
    contentCss="p-0" css="px-8 py-8 md:px-14 md:py-10" 
    titleCss="h2 ff-bold"
    onClose={() => {}}
    onSave={() => { saveAlmacen() }}
  >
    <AlmacenLayoutEditor bind:almacen={almacenForm} />
  </Layer>
</Page>
