<script lang="ts">
import Input from '$components/Input.svelte';
import Modal from '$components/Modal.svelte';
import Page from '$ui/Page.svelte';
import Layer from '$components/Layer.svelte';
import SearchCard from '$components/SearchCard.svelte';
import SearchSelect from '$components/SearchSelect.svelte';
import CheckboxOptions from '$components/CheckboxOptions.svelte';
import VTable from '$components/vTable/VTable.svelte';
import type { ITableColumn } from '$components/vTable/types';
import { Notify, throttle } from '$core/helpers';
import { Core, closeModal } from '$core/store.svelte';
import Modules from '$core/modules';
  import pkg from 'notiflix'
const { Loading } = pkg
  import {
    AccesosService,
    PerfilesService,
    postSeguridadAccesos,
    postPerfil,
    accesosGrupos,
    accesoAcciones,
    arrayToMapN,
    type IAcceso,
    type IPerfil
  } from "./perfiles-accesos.svelte"
import AccesoCard from '$routes/admin/perfiles-accesos/AccesoCard.svelte';

  const accesosService = new AccesosService()
  const perfilesService = new PerfilesService()

  const accesosGruposMap = arrayToMapN(accesosGrupos, 'id')
  const modulesMap = arrayToMapN(Modules, 'id')

  let accesoForm = $state({} as IAcceso)
  let perfilForm = $state({} as IPerfil)
  let moduleSelected = $state(0)
  let filterText = $state("")
  let accesoEdit = $state(false)

  const accesosGrouped = $derived.by(() => {
    const gruposMap: Map<string, IAcceso[]> = new Map()
    const moduleSelectedID = moduleSelected

    for (let acs of accesosService.accesos) {
      for (let md of acs.modulosIDs) {
        if (moduleSelectedID && moduleSelectedID !== md) { continue }
        const key = [md, acs.grupo].join("_")
        if (!gruposMap.has(key)) { gruposMap.set(key, []) }
        gruposMap.get(key)!.push(acs)
        break
      }
    }

    const accesosGrouped_: {
      moduleID: number
      group: number
      accesos: IAcceso[]
      groupName: string
      moduleName: string
    }[] = []

    for (let [key, accesosGroup] of gruposMap) {
      const [moduleID, group] = key.split("_").map(x => parseInt(x))
      accesosGrouped_.push({
        moduleID,
        group,
        accesos: accesosGroup,
        groupName: accesosGruposMap.get(group)?.name || "",
        moduleName: moduleSelectedID ? "" : modulesMap.get(moduleID)?.name || ""
      })
    }

    return accesosGrouped_
  })

  async function saveAcceso() {
    const form = accesoForm
    if (!form.nombre || !form.orden || (form.acciones?.length || 0) == 0
      || !form.grupo || (form.modulosIDs?.length || 0) === 0) {
      Notify.failure("Faltan propiedades para agregar el acceso.")
      return
    }

    Loading.standard("Actualizando Acceso...")

    try {
      const result = await postSeguridadAccesos(form)

      if ((form.id || 0) <= 0) form.id = result.id
      accesosService.updateAcceso(form)

      accesoForm = {} as IAcceso
      closeModal(1)
      Notify.success("Acceso guardado correctamente")
    } catch (error) {
      Notify.failure(error as string)
    }
    Loading.remove()
  }

  async function savePerfil(onDelete?: boolean, isAccesos?: boolean) {
    const form = perfilForm
    if (!form.nombre) {
      Notify.failure("Faltan propiedades para agregar el perfil.")
      return
    }

    if (isAccesos) {
      form.accesos = []
      for (let [accesoID, niveles] of form.accesosMap) {
        if (niveles.length === 0) { form.accesosMap.delete(accesoID) }
        for (let n of niveles) {
          form.accesos.push(accesoID * 10 + n)
        }
      }
      const accesosFiltered = accesosService.accesos.filter(x => form.accesosMap.has(x.id))
      const modulosIDSet: Set<number> = new Set()
      for (let e of accesosFiltered) {
        for (let md of e.modulosIDs) { modulosIDSet.add(md) }
      }
      form.modulosIDs = [...modulosIDSet]
    }

    Loading.standard("Actualizando Perfil...")

    try {
      const result = await postPerfil(form)

      if ((form.id || 0) <= 0) form.id = result.id
      form._open = false
      perfilesService.updatePerfil(form)

      perfilForm = {} as IPerfil
      closeModal(2)
      Notify.success("Perfil guardado correctamente")
    } catch (error) {
      Notify.failure(error as string)
    }
    Loading.remove()
  }

  const columns: ITableColumn<IPerfil>[] = [
    {
      header: "ID",
      headerCss: "w-54",
      cellCss: "text-center c-purple",
      getValue: e => e.id
    },
    {
      header: "Perfil", highlight: true,
      getValue: e => e.nombre
    },
    {
      header: "...",
      headerCss: "w-42",
      cellCss: "text-center",
      id: "actions",
      buttonEditHandler: (rec) => {
        perfilForm = { ...rec, accesosMap: new Map(rec.accesosMap) }
        accesoEdit = false
        Core.openModal(2)
      }
    }
  ]
</script>

<Page title="Perfiles & Accesos">
  <div class="flex justify-between h-full gap-8 max-md:flex-col">
    <!-- Left side: Profiles table -->
    <div class="w-full md:w-[32%]">
      <div class="flex justify-between items-center w-full mb-10">
        <div class="i-search mr-16 w-256">
          <div><i class="icon-search"></i></div>
          <input class="w-full" autocomplete="off" type="text" onkeyup={ev => {
            ev.stopPropagation()
            throttle(() => {
              filterText = ((ev.target as HTMLInputElement).value || "").toLowerCase().trim()
            }, 150)
          }}>
        </div>
        <div class="flex items-center">
          <button class="bx-green" onclick={ev => {
            ev.stopPropagation()
            perfilForm = { ss: 1, accesosMap: new Map() } as IPerfil
            Core.openModal(2)
          }} aria-label="Agregar perfil">
            <i class="icon-plus"></i>
          </button>
        </div>
      </div>
      <VTable
        css="w-full selectable"
        columns={columns}
        maxHeight="calc(100vh - 8rem - 16px)"
        data={perfilesService.perfiles}
        selected={perfilForm.id}
        filterText={filterText}
        getFilterContent={e => [e.nombre].filter(x => x).join(" ").toLowerCase()}
        isSelected={(e, id) => e.id === id}
        onRowClick={e => {
          accesoEdit = false
          if (e.id === perfilForm.id) {
            perfilForm = {} as IPerfil
          } else {
            perfilForm = { ...e, _open: true, accesosMap: new Map(e.accesosMap) }
          }
        }}
      />
    </div>

    <!-- Right side: Accesos -->
    <div class="w-full md:w-[66.5%]">
      <div class="flex justify-between w-full mb-6">
        <div class="ff-bold text-xl">
          <span>Accesos</span>
          {#if accesoEdit}
            <span class="text-red-500">(Modo Edición)</span>
          {/if}
          {#if perfilForm.id > 0}
            <span class="mr-4">:</span>
            <span class="c-purple ml-4">{perfilForm.nombre}</span>
          {/if}
        </div>
        <div class="flex items-center max-md:absolute max-md:top-0 max-md:right-0">
          {#if accesoEdit}
            <button class="bx-green mr-8" onclick={ev => {
              ev.stopPropagation()
              accesoForm = { ss: 1, modulosIDs: [], acciones: [], id: 0, nombre: "", orden: 0, grupo: 0, upd: 0 }
              Core.openModal(1)
            }} aria-label="Agregar acceso">
              <i class="icon-plus"></i>
            </button>
          {/if}
          {#if perfilForm.id > 0}
            <button class="bx-blue mr-8" onclick={ev => {
              ev.stopPropagation()
              savePerfil(false, true)
            }} aria-label="Guardar">
              <i class="icon-floppy"></i>
              <span class="max-md:hidden">Guardar</span>
            </button>
          {/if}
          {#if !perfilForm.id}
            <button class="bn-white" onclick={ev => {
              ev.stopPropagation()
              accesoEdit = !accesoEdit
            }} aria-label="Editar">
              {#if accesoEdit}
                <i class="text-red-500 icon-cancel"></i>
              {:else}
                <i class="icon-pencil"></i>
              {/if}
            </button>
          {/if}
        </div>
      </div>
      {#if !accesoEdit && !perfilForm.id}
        <div class="mb-8 px-12 py-8 bg-red-100 border border-red-400 text-red-700 rounded w-fit">
          Debe seleccionar un perfil para editar sus accesos.
        </div>
      {/if}
      {#each accesosGrouped as ag}
        <div class="ff-bold h3 mb-6">
          {ag.moduleName}{ag.moduleName ? " > " : ""}{ag.groupName}
        </div>
        <div class="grid grid-cols-3 gap-x-12 gap-y-8 mb-16">
          {#each ag.accesos as acceso}
            <AccesoCard
              {acceso}
              isEdit={accesoEdit}
              bind:perfilForm
              onEdit={() => {
                accesoForm = { ...acceso }
                Core.openModal(1)
              }}
            />
          {/each}
        </div>
      {/each}
    </div>
  </div>

  <!-- Modal for Acceso -->
  <Modal
    id={1}
    size={6}
    title={(accesoForm?.id > 0 ? "Editando" : "Creando") + " Acceso"}
    isEdit={!!accesoForm?.id}
    onSave={() => saveAcceso()}
  >
    <div class="grid grid-cols-24 gap-10 p-6">
      <Input
        bind:saveOn={accesoForm}
        save="nombre"
        css="col-span-14"
        label="Nombre"
        required={true}
      />
      <SearchSelect
        bind:saveOn={accesoForm}
        save="grupo"
        css="col-span-10"
        label="Grupo"
        required={true}
        options={accesosGrupos}
        keyId="id"
        keyName="name"
      />
      <Input
        bind:saveOn={accesoForm}
        save="descripcion"
        css="col-span-16"
        label="Descripción"
      />
      <Input
        bind:saveOn={accesoForm}
        save="orden"
        css="col-span-8"
        label="Orden"
        type="number"
      />
      <SearchCard
        bind:saveOn={accesoForm}
        save="modulosIDs"
        css="col-span-24"
        label="Módulos"
        options={Modules}
        keyId="id"
        keyName="name"
      />
      <div class="col-span-24 flex items-center gap-12 mb-4">
        <div class="h-[1px] grow bg-gray-300"></div>
        <div class="ff-bold text-lg">Acciones</div>
        <div class="h-[1px] grow bg-gray-300"></div>
      </div>
      <CheckboxOptions
        bind:saveOn={accesoForm}
        save="acciones"
        css="col-span-24 flex-wrap gap-y-8"
        options={accesoAcciones}
        keyId="id"
        keyName="name"
        type="multiple"
      />
    </div>
  </Modal>

  <!-- Modal for Perfil -->
  <Modal
    id={2}
    size={5}
    title={(perfilForm?.id > 0 ? "Editando" : "Creando") + " Perfil"}
    isEdit={!!perfilForm?.id}
    onSave={() => savePerfil()}
    onClose={() => { perfilForm = {} as IPerfil }}
  >
    <div class="grid grid-cols-24 gap-10 p-6">
      <Input
        bind:saveOn={perfilForm}
        save="nombre"
        css="col-span-24"
        label="Nombre"
        required={true}
      />
      <Input
        bind:saveOn={perfilForm}
        save="descripcion"
        css="col-span-24"
        label="Descripción"
      />
    </div>
  </Modal>
</Page>
