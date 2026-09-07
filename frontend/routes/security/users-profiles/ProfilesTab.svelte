<script lang="ts">
  import { useUI } from '@genix/ui';
  const ui = useUI();
import Input from '$components/form/Input.svelte';
import Modal from '$components/layers/Modal.svelte';
import VTable from '$components/vTable/VTable.svelte';
import type { ITableColumn } from '$components/vTable/types';
import Modules from '$core/modules';
import { Core, tr } from '$core/store.svelte';
import T from '$components/misc/T.svelte';
import { arrayToMapN, Loading, Notify } from '$libs/helpers';
import FilterInput from '$components/form/FilterInput.svelte';
import Button from '$components/buttons/Button.svelte';
import AccesoCard from './AccessCard.svelte';
import {
  type IAccessGroupCatalogEntry,
  type IAccessListCatalogEntry
} from './access-list-catalog';
import {
    PerfilesService,
    postPerfil,
    type IAccess,
    type IProfile
} from "./users-profiles.svelte";
import { buildAccesosCatalog, buildRouteCatalogIndex, toAccesoGrants } from './users-profiles';

  // Services and catalog live in +page.svelte so both tabs share one copy of the profiles.
  let { perfilesService, accessGroups, accessListEntries, accessListLoadError = "" }: {
    perfilesService: PerfilesService
    accessGroups: IAccessGroupCatalogEntry[]
    accessListEntries: IAccessListCatalogEntry[]
    accessListLoadError?: string
  } = $props()

  const modulesMap = arrayToMapN(Modules, 'id')
  const routeCatalogIndex = buildRouteCatalogIndex()

  let perfilForm = $state({} as IProfile)
  let moduleSelected = $state(0)
  let filterText = $state("")

  const accessGroupNameByID = $derived.by(() => {
    const accessGroupMap = new Map<number, string>()
    for (const accessGroupRecord of accessGroups) {
      accessGroupMap.set(accessGroupRecord.id, accessGroupRecord.name)
    }
    return accessGroupMap
  })

  const accesosCatalog = $derived(buildAccesosCatalog(accessListEntries, routeCatalogIndex))

  const accesosGrouped = $derived.by(() => {
    const gruposMap: Map<string, IAccess[]> = new Map()
    const moduleSelectedID = moduleSelected

    for (const accessRecord of accesosCatalog) {
      const routeMeta = routeCatalogIndex.get((accessRecord.descripcion || "").replace(/^\//, ""))
      const moduleIDs = accessRecord.modulosIDs.length > 0
        ? accessRecord.modulosIDs
        : routeMeta ? [routeMeta.moduleID] : [Modules[0].id]
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
      accesos: IAccess[]
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
          || tr("No group|Sin grupo"),
        moduleName: moduleSelectedID ? "" : routeCatalogIndex.get((accesosGroup[0]?.descripcion || "").replace(/^\//, ""))?.moduleName
          || modulesMap.get(moduleID)?.name || tr("No module|Sin módulo")
      })
    }

    return accesosGrouped_.sort((leftGroup, rightGroup) => {
      if (leftGroup.moduleID !== rightGroup.moduleID) {
        return leftGroup.moduleID - rightGroup.moduleID
      }
      return leftGroup.group - rightGroup.group
    })
  })

  async function savePerfil(onDelete?: boolean, isAccesos?: boolean) {
    const form = perfilForm
    if (!form.Name) {
      Notify.failure(tr("Missing required properties to add the profile.|Faltan propiedades para agregar el perfil."))
      return
    }

    if (isAccesos) {
      // toAccesoGrants drops any sub-access whose parent access the operator cleared during this
      // same edit, so a leftover tick cannot reach the backend and be rejected there.
      form.AccesosGrants = toAccesoGrants(form.accesosMap, form.subAccesosMap)

      const accesosFiltered = accesosCatalog.filter(x => form.accesosMap.has(x.id))
      const modulosIDSet: Set<number> = new Set()
      for (let e of accesosFiltered) {
        for (let md of e.modulosIDs) { modulosIDSet.add(md) }
      }
      form.Modulos = [...modulosIDSet]
    }

    Loading.standard(tr("Updating Profile...|Actualizando Perfil..."))

    try {
      const result = await postPerfil(form)

      if ((form.ID || 0) <= 0) form.ID = result.ID
      perfilesService.updatePerfil(form)

      perfilForm = {} as IProfile
      ui.closeModal(2)
      Notify.success(tr("Profile saved successfully|Perfil guardado correctamente"))
    } catch (error) {
      Notify.failure(error as string)
    }
    Loading.remove()
  }

  const columns: ITableColumn<IProfile>[] = [
    {
      header: "ID",
      headerCss: "w-54",
      css: "text-center c-purple",
      getValue: e => e.ID,
      mobile: { order: 1, css: "col-span-6 ff-bold", icon: "[fa--tag]" }
    },
    {
      header: "Profile|Perfil", highlight: true,
      getValue: e => e.Name,
      mobile: { order: 2, css: "col-span-14", render: e => `<strong>${e.Name || ''}</strong>` }
    },
    {
      header: "...",
      headerCss: "w-42",
      css: "text-center",
      id: "actions",
      buttonEditHandler: (rec) => {
        perfilForm = { ...rec, accesosMap: new Map(rec.accesosMap), subAccesosMap: new Map(rec.subAccesosMap) }
        ui.openModal(2)
      },
      mobile: { order: 3, css: "col-span-4 justify-end" }
    }
  ]
</script>

  <div class="flex justify-between h-full gap-8 max-md:flex-col">
    <!-- Left side: Profiles table -->
    <div class="w-full md:w-[32%]">
      <div class="flex justify-between items-center w-full mb-10" aria-label="Profiles toolbar with filter and create button">
        <FilterInput bind:value={filterText} css="mr-16 w-256" />
        <div class="flex items-center">
          <Button color="green" icon="icon-[fa--plus]" label="Opens the modal to create a new access profile." onClick={() => {
            perfilForm = { ss: 1, accesosMap: new Map(), subAccesosMap: new Map() } as IProfile
            ui.openModal(2)
          }} />
        </div>
      </div>
      <VTable
        css="w-full selectable"
        columns={columns}
        maxHeight="calc(100vh - 8rem - 16px)"
        data={perfilesService.perfiles}
        selected={perfilForm.ID}
        filterText={filterText}
        getFilterContent={e => [e.Name].filter(x => x).join(" ").toLowerCase()}
        isSelected={(e, id) => e.ID === id}
        onRowClick={e => {
          if (e.ID === perfilForm.ID) {
            perfilForm = {} as IProfile
          } else {
            perfilForm = { ...e, accesosMap: new Map(e.accesosMap), subAccesosMap: new Map(e.subAccesosMap) }
          }
        }}
      />
    </div>

    <!-- Right side: Accesos -->
    <div class="w-full md:w-[66.5%]">
      {#if perfilForm.ID}
	      <div class="flex justify-between w-full mb-6">
	        <div class="ff-bold text-xl">
	          <span><T text="Access|Accesos" /></span>
	          {#if perfilForm.ID > 0}
	            <span class="mr-2"><T text="of|de" /></span>
	            <span class="c-purple ml-4">{perfilForm.Name}</span>
	          {/if}
	        </div>
	        <div class="flex items-center max-md:absolute max-md:top-0 max-md:right-0">
	          {#if perfilForm.ID > 0}
	            <Button color="blue" icon="icon-[fa--floppy-o]" name="Save|Guardar" css="mr-8"
	              hideNameOnMobile label="Saves the access permissions for this profile." onClick={() => savePerfil(false, true)} />
	          {/if}
	        </div>
	      </div>
      {:else}
	      <div class="mb-8 px-12 py-8 bg-red-100 border border-red-400 text-red-700 rounded w-fit">
	        <T text="Select a profile to edit its access permissions.|Debe seleccionar un perfil para editar sus accesos." />
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
	        <div class="grid grid-cols-1 md:grid-cols-3 gap-x-12 gap-y-8 mb-16">
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
    title={(perfilForm?.ID > 0 ? tr("Editing|Editando") : tr("Creating|Creando")) + " " + tr("Profile|Perfil")}
    isEdit={!!perfilForm?.ID}
    onSave={() => savePerfil()}
    onClose={() => { perfilForm = {} as IProfile }}
  >
    <div class="grid grid-cols-24 gap-10 p-6" aria-label="Profile form with name and description">
      <Input
        bind:saveOn={perfilForm}
        save="Name"
        css="col-span-24"
        label="Name|Nombre"
        required={true}
      />
      <Input
        bind:saveOn={perfilForm}
        save="Description"
        css="col-span-24"
        label="Description|Descripción"
      />
    </div>
  </Modal>

<style>
  ._access-group-title {
    color: #5f6dff;
  }
</style>
