<script lang="ts">
import Input from '$components/Input.svelte';
import Modal from '$components/Modal.svelte';
import VTable from '$components/vTable/VTable.svelte';
import type { ITableColumn } from '$components/vTable/types';
import Modules from '$core/modules';
import { closeModal, Core } from '$core/store.svelte';
import Page from '$domain/Page.svelte';
import { arrayToMapN, Loading, Notify, throttle } from '$libs/helpers';
import { onMount } from 'svelte';
import AccesoCard from './AccesoCard.svelte';
import {
  fetchAccessListCatalog,
  type IAccessGroupCatalogEntry,
  type IAccessListCatalogEntry
} from './access-list-catalog';
import {
    PerfilesService,
    postPerfil,
    type IAcceso,
    type IPerfil
} from "./perfiles-accesos.svelte";

  const perfilesService = new PerfilesService()

  const modulesMap = arrayToMapN(Modules, 'id')
  const routeCatalogIndex = new Map<string, {
    moduleID: number
    moduleName: string
    groupID: number
    groupName: string
  }>()

  for (const moduleRecord of Modules) {
    for (const menuRecord of moduleRecord.menus) {
      for (const menuOption of menuRecord.options || []) {
        const normalizedRoute = (menuOption.route || "").replace(/^\//, "")
        if (!normalizedRoute) { continue }
        routeCatalogIndex.set(normalizedRoute, {
          moduleID: moduleRecord.id,
          moduleName: moduleRecord.name,
          groupID: menuRecord.id || 0,
          groupName: menuRecord.name
        })
      }
    }
  }

  let perfilForm = $state({} as IPerfil)
  let moduleSelected = $state(0)
  let filterText = $state("")
  let accessGroups = $state([] as IAccessGroupCatalogEntry[])
  let accessListEntries = $state([] as IAccessListCatalogEntry[])
  let accessListLoadError = $state("")

  const accessGroupNameByID = $derived.by(() => {
    const accessGroupMap = new Map<number, string>()
    for (const accessGroupRecord of accessGroups) {
      accessGroupMap.set(accessGroupRecord.id, accessGroupRecord.name)
    }
    return accessGroupMap
  })

  function decodeAccessLevels(accessLevelMask: number): number[] {
    const levelDigitMap = new Map<number, number>([
      [1, 1],
      [2, 2],
      [3, 3],
      [4, 4],
    ])

    // The YAML catalog stores levels as concatenated digits, e.g. 19 => VER + TODO.
    return [...String(accessLevelMask)]
      .map(levelDigit => levelDigitMap.get(Number(levelDigit)) || 0)
      .filter(levelValue => levelValue > 0)
  }

  const accesosCatalog = $derived.by(() => {
    if (accessListEntries.length === 0) { return [] as IAcceso[] }

    const catalogAccesos = accessListEntries.map((accessListEntry) => {
      const normalizedRoute = (accessListEntry.frontend_routes || "").replace(/^\//, "")
      const routeMeta = routeCatalogIndex.get(normalizedRoute)
      const catalogActions = decodeAccessLevels(accessListEntry.levels)

      return {
        id: accessListEntry.id,
        nombre: accessListEntry.name,
        descripcion: normalizedRoute,
        orden: accessListEntry.id,
        acciones: catalogActions.length > 0 ? catalogActions : [1],
        grupo: accessListEntry.group || routeMeta?.groupID || 0,
        modulosIDs: routeMeta ? [routeMeta.moduleID] : [],
        ss: 1,
        upd: 0
      }
    })

    return catalogAccesos.sort((leftAccess, rightAccess) => leftAccess.orden - rightAccess.orden)
  })

  const accesosGrouped = $derived.by(() => {
    const gruposMap: Map<string, IAcceso[]> = new Map()
    const moduleSelectedID = moduleSelected

    for (const accessRecord of accesosCatalog) {
      const routeMeta = routeCatalogIndex.get((accessRecord.descripcion || "").replace(/^\//, ""))
      const moduleIDs = accessRecord.modulosIDs.length > 0
        ? accessRecord.modulosIDs
        : routeMeta ? [routeMeta.moduleID] : [0]
      const groupID = accessRecord.grupo || routeMeta?.groupID || 0

      for (const moduleID of moduleIDs) {
        if (moduleSelectedID && moduleSelectedID !== moduleID) { continue }
        const groupKey = [moduleID, groupID].join("_")
        if (!gruposMap.has(groupKey)) { gruposMap.set(groupKey, []) }
        gruposMap.get(groupKey)!.push(accessRecord)
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
        groupName: accessGroupNameByID.get(group)
          || routeCatalogIndex.get((accesosGroup[0]?.descripcion || "").replace(/^\//, ""))?.groupName
          || "Sin grupo",
        moduleName: moduleSelectedID ? "" : routeCatalogIndex.get((accesosGroup[0]?.descripcion || "").replace(/^\//, ""))?.moduleName
          || modulesMap.get(moduleID)?.name || "Sin módulo"
      })
    }

    return accesosGrouped_.sort((leftGroup, rightGroup) => {
      if (leftGroup.moduleID !== rightGroup.moduleID) {
        return leftGroup.moduleID - rightGroup.moduleID
      }
      return leftGroup.group - rightGroup.group
    })
  })

  onMount(async () => {
    try {
      // Load the hashed static catalog so this page can compare backend records against the CDN-shipped source of truth.
      const accessListPayload = await fetchAccessListCatalog()
      accessGroups = accessListPayload.access_groups || []
      accessListEntries = accessListPayload.access_list || []
      console.info('[access-list] Catalog loaded', {
        totalGroups: accessGroups.length,
        totalEntries: accessListEntries.length
      })
    } catch (error) {
      accessListLoadError = error as string
      console.error('[access-list] Catalog load failed', { error })
    }
  })

  async function savePerfil(onDelete?: boolean, isAccesos?: boolean) {
    const form = perfilForm
    if (!form.Nombre) {
      Notify.failure("Faltan propiedades para agregar el perfil.")
      return
    }

    if (isAccesos) {
      form.Accesos = []
      for (let [accesoID, niveles] of form.accesosMap) {
        if (niveles.length === 0) { form.accesosMap.delete(accesoID) }
        for (let n of niveles) {
          form.Accesos.push(accesoID * 10 + n)
        }
      }
      const accesosFiltered = accesosCatalog.filter(x => form.accesosMap.has(x.id))
      const modulosIDSet: Set<number> = new Set()
      for (let e of accesosFiltered) {
        for (let md of e.modulosIDs) { modulosIDSet.add(md) }
      }
      form.Modulos = [...modulosIDSet]
    }

    Loading.standard("Actualizando Perfil...")

    try {
      const result = await postPerfil(form)

      if ((form.ID || 0) <= 0) form.ID = result.ID
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
      getValue: e => e.ID
    },
    {
      header: "Perfil", highlight: true,
      getValue: e => e.Nombre
    },
    {
      header: "...",
      headerCss: "w-42",
      cellCss: "text-center",
      id: "actions",
      buttonEditHandler: (rec) => {
        perfilForm = { ...rec, accesosMap: new Map(rec.accesosMap) }
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
        selected={perfilForm.ID}
        filterText={filterText}
        getFilterContent={e => [e.Nombre].filter(x => x).join(" ").toLowerCase()}
        isSelected={(e, id) => e.ID === id}
        onRowClick={e => {
          if (e.ID === perfilForm.ID) {
            perfilForm = {} as IPerfil
          } else {
            perfilForm = { ...e, _open: true, accesosMap: new Map(e.accesosMap) }
          }
        }}
      />
    </div>

    <!-- Right side: Accesos -->
    <div class="w-full md:w-[66.5%]">
      {#if perfilForm.ID}
	      <div class="flex justify-between w-full mb-6">
	        <div class="ff-bold text-xl">
	          <span>Accesos</span>
	          {#if perfilForm.ID > 0}
	            <span class="mr-2">de</span>
	            <span class="c-purple ml-4">{perfilForm.Nombre}</span>
	          {/if}
	        </div>
	        <div class="flex items-center max-md:absolute max-md:top-0 max-md:right-0">
	          {#if perfilForm.ID > 0}
	            <button class="bx-blue mr-8" onclick={ev => {
	              ev.stopPropagation()
	              savePerfil(false, true)
	            }} aria-label="Guardar">
	              <i class="icon-floppy"></i>
	              <span class="max-md:hidden">Guardar</span>
	            </button>
	          {/if}
	        </div>
	      </div>
      {:else}
	      <div class="mb-8 px-12 py-8 bg-red-100 border border-red-400 text-red-700 rounded w-fit">
	        Debe seleccionar un perfil para editar sus accesos.
	      </div>
      {/if}
      {#if accessListLoadError}
        <div class="mb-8 px-12 py-8 bg-amber-100 border border-amber-400 text-amber-700 rounded w-fit">
          {accessListLoadError}
        </div>
      {/if}
      <div class="max-h-[calc(100vh-var(--header-height)-1rem-64px)] overflow-y-auto p-2">
	      {#each accesosGrouped as ag}
	        <div class="ff-bold h3 mb-3 _access-group-title">
	          {ag.groupName}
	        </div>
	        <div class="grid grid-cols-3 gap-x-12 gap-y-8 mb-16">
	          {#each ag.accesos as acceso}
	            <AccesoCard
	              {acceso}
	              bind:perfilForm
	            />
	          {/each}
	        </div>
	      {/each}
      </div>
    </div>
  </div>

  <!-- Modal for Perfil -->
  <Modal
    id={2}
    size={5}
    title={(perfilForm?.ID > 0 ? "Editando" : "Creando") + " Perfil"}
    isEdit={!!perfilForm?.ID}
    onSave={() => savePerfil()}
    onClose={() => { perfilForm = {} as IPerfil }}
  >
    <div class="grid grid-cols-24 gap-10 p-6">
      <Input
        bind:saveOn={perfilForm}
        save="Nombre"
        css="col-span-24"
        label="Nombre"
        required={true}
      />
      <Input
        bind:saveOn={perfilForm}
        save="Descripcion"
        css="col-span-24"
        label="Descripción"
      />
    </div>
  </Modal>
</Page>

<style>
  ._access-group-title {
    color: #5f6dff;
  }
</style>
