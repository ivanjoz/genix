<script lang="ts">
import Input from '$components/form/Input.svelte';
import Layer from '$components/layers/Layer.svelte';
import Page from '$domain/Page.svelte';
import UserProfilesAccessSelector from './UserProfilesAccessSelector.svelte';
import VTable from '$components/vTable/VTable.svelte';
import type { ITableColumn } from '$components/vTable/types';
import { Notify } from '$libs/helpers';
import FilterInput from '$components/form/FilterInput.svelte';
import Button from '$components/buttons/Button.svelte';
import { Core, tr } from '$core/store.svelte';
import T from '$components/misc/T.svelte';
import { formatTime } from '$libs/helpers';
import { onMount } from 'svelte';
  import pkg from 'notiflix'
const { Loading } = pkg
  import { fetchAccessListCatalog, type IAccessGroupCatalogEntry, type IAccessListCatalogEntry } from "../perfiles-accesos/access-list-catalog"
  import { accesoAcciones } from "../perfiles-accesos/perfiles-accesos.svelte"
  import { UsuariosService, PerfilesService, postUsuario, type IUser } from "./usuarios.svelte"

  const usuariosService = new UsuariosService()
  const perfilesService = new PerfilesService()
  const accessActionShortNameByID = new Map(accesoAcciones.map((accessActionRecord) => [accessActionRecord.id, accessActionRecord.short]))

  let filterText = $state("")
  let usuarioForm = $state({} as IUser)
  let accessGroupEntries = $state([] as IAccessGroupCatalogEntry[])
  let accessCatalogEntries = $state([] as IAccessListCatalogEntry[])
  let accessCatalogLoadError = $state("")

  interface IUsuarioAccessSummary {
    readableAccessNames: string[]
    editableAccessNames: string[]
  }

  function formatAccessSummary(accessNames: string[], maxVisibleAccesses = 8): string {
    // Keep the table compact by truncating long access lists with a clear remaining-count suffix.
    if (accessNames.length <= maxVisibleAccesses) {
      return accessNames.join(", ")
    }

    const visibleAccessNames = accessNames.slice(0, maxVisibleAccesses)
    const hiddenAccessCount = accessNames.length - maxVisibleAccesses
    return `${visibleAccessNames.join(", ")} ... (${hiddenAccessCount} más)`
  }

  function decodeAccessLevels(accessLevelMask: number): number[] {
    // Expand the compact catalog format (for example 14) into individual selectable access levels.
    return [...String(accessLevelMask)]
      .map((levelDigit) => Number(levelDigit))
      .filter((levelValue) => levelValue > 0)
  }

  const sortedPerfiles = $derived.by(() => {
    // Keep profile ordering stable for the selector without re-sorting inside the child component.
    return [...perfilesService.perfiles].sort((leftProfile, rightProfile) => {
      const nameComparison = leftProfile.Name.localeCompare(rightProfile.Name)
      return nameComparison !== 0 ? nameComparison : leftProfile.ID - rightProfile.ID
    })
  })

  const accessLevelOptions = $derived.by(() => {
    const flattenedAccessLevelOptions: { ID: number, Name: string }[] = []

    // Build and sort access options once per catalog refresh so the child component stays render-only.
    for (const accessCatalogEntry of accessCatalogEntries) {
      const accessLevels = decodeAccessLevels(accessCatalogEntry.levels)
      for (const accessLevel of accessLevels) {
        const accessActionShortName = accessActionShortNameByID.get(accessLevel) || `N${accessLevel}`
        flattenedAccessLevelOptions.push({
          ID: accessCatalogEntry.id * 10 + accessLevel,
          Name: `${accessCatalogEntry.name} (${accessActionShortName})`
        })
      }
    }

    return flattenedAccessLevelOptions.sort((leftOption, rightOption) => {
      const nameComparison = leftOption.Name.localeCompare(rightOption.Name)
      return nameComparison !== 0 ? nameComparison : leftOption.ID - rightOption.ID
    })
  })

  const perfilesByID = $derived.by(() => {
    return new Map(sortedPerfiles.map((profileRecord) => [profileRecord.ID, profileRecord]))
  })

  const accessCatalogNameByID = $derived.by(() => {
    const accessNameByID = new Map<number, string>()

    for (const accessCatalogEntry of accessCatalogEntries) {
      accessNameByID.set(accessCatalogEntry.id, accessCatalogEntry.name)
    }

    return accessNameByID
  })

  function summarizeUsuarioAccesses(usuarioRecord: IUser): IUsuarioAccessSummary {
    const accessLevelsByAccessID = new Map<number, Set<number>>()

    // Merge profile-derived and user-specific access levels so the table shows the effective access summary.
    for (const profileID of usuarioRecord.ProfileIDs || []) {
      const profileRecord = perfilesByID.get(profileID)
      if (!profileRecord?.accesosMap) { continue }

      for (const [accessID, accessLevels] of profileRecord.accesosMap) {
        if (!accessLevelsByAccessID.has(accessID)) {
          accessLevelsByAccessID.set(accessID, new Set())
        }
        for (const accessLevel of accessLevels) {
          accessLevelsByAccessID.get(accessID)!.add(accessLevel)
        }
      }
    }

    for (const encodedAccessLevelID of usuarioRecord.AccessLevelIDs || []) {
      const accessID = Math.floor(encodedAccessLevelID / 10)
      const accessLevel = encodedAccessLevelID % 10
      if (!accessLevelsByAccessID.has(accessID)) {
        accessLevelsByAccessID.set(accessID, new Set())
      }
      accessLevelsByAccessID.get(accessID)!.add(accessLevel)
    }

    const readableAccessNames = new Set<string>()
    const editableAccessNames = new Set<string>()

    for (const [accessID, accessLevels] of accessLevelsByAccessID) {
      const accessName = accessCatalogNameByID.get(accessID)
      if (!accessName) { continue }

      if (accessLevels.has(1)) {
        readableAccessNames.add(accessName)
      }

      if ([...accessLevels].some((accessLevel) => accessLevel > 1)) {
        editableAccessNames.add(accessName)
      }
    }

    return {
      readableAccessNames: [...readableAccessNames].sort((leftName, rightName) => leftName.localeCompare(rightName)),
      editableAccessNames: [...editableAccessNames].sort((leftName, rightName) => leftName.localeCompare(rightName))
    }
  }

  const resetUsuarioForm = () => {
    // Initialize the create form with an active status so the layer always opens in a valid default state.
    usuarioForm = { Status: 1, ProfileIDs: [], AccessLevelIDs: [] } as unknown as IUser
  }

  const openCreateUsuarioLayer = () => {
    resetUsuarioForm()
    console.log("openCreateUsuarioLayer::")
    Core.openSideLayer(1)
  }

  const openEditUsuarioLayer = (selectedUsuario: IUser) => {
    // Clone the selected record so the table does not update optimistically while the user edits the layer.
    usuarioForm = {
      ...selectedUsuario,
      ProfileIDs: [...(selectedUsuario.ProfileIDs || [])],
      AccessLevelIDs: [...(selectedUsuario.AccessLevelIDs || [])]
    }
    console.log("openEditUsuarioLayer::", $state.snapshot(usuarioForm))
    Core.openSideLayer(1)
  }

  onMount(async () => {
    try {
      const accessCatalogPayload = await fetchAccessListCatalog()
      accessGroupEntries = accessCatalogPayload.access_groups || []
      accessCatalogEntries = accessCatalogPayload.access_list || []
      console.info("usuarios::accessCatalogLoaded", {
        totalGroups: accessGroupEntries.length,
        totalEntries: accessCatalogEntries.length
      })
    } catch (error) {
      accessCatalogLoadError = error as string
      console.error("usuarios::accessCatalogLoadError", { error })
    }
  })

  async function saveUsuario(isDelete?: boolean) {
    const form = usuarioForm

    if ((form.Usuario?.length || 0) < 4 || (form.FirstName?.length || 0) < 4) {
      Notify.failure(tr("Username and first name must be at least 4 characters.|El usuario y el nombre deben tener al menos 4 caracteres."))
      return
    }

    if (form.Password) form.Password = form.Password.trim()
    if (form.Password2) form.Password2 = form.Password2.trim()

    if (!form.ID || form.Password) {
      let err = ""
      if ((form.Password?.length || 0) < 6) {
        err = tr("Password must be at least 6 characters.|El password tiene menos de 6 caracteres.")
      } else if (form.Password !== form.Password2) {
        err = tr("Passwords do not match.|Los password no coinciden.")
      }
      if (err) {
        Notify.failure(err)
        return
      }
    }

    Loading.standard(tr("Creating/Updating User...|Creando/Actualizando Usuario..."))
    console.log("saveUsuario payload::", { isDelete: !!isDelete, form: $state.snapshot(form) })
    try {
      const result = await postUsuario(form)

      if (isDelete) {
        usuariosService.removeUsuario(form.ID)
      } else {
        // Keep local state synchronized with backend-assigned ID for brand-new users.
        if (!form.ID) {
          form.ID = result.ID
        }
        usuariosService.updateUsuario(form)
      }

      console.log("saveUsuario result::", result)
      Core.hideSideLayer()
      resetUsuarioForm()
    } catch (error) {
      console.warn("saveUsuario error::", error)
      Notify.failure(error as string)
    }
    Loading.remove()
  }

  const columns: ITableColumn<IUser>[] = [
    {
      header: "ID",
      headerCss: "w-54",
      css: "text-center ff-bold",
      getValue: e => e.ID
    },
    {
      id: "usuario_info",
      header: "Username|Usuario", highlight: true,
      css: "px-8 py-6 align-top",
      getValue: e => e.Usuario
    },
    {
      id: "usuario_accesos", headerCss: "w-[47%]",
      header: "Access|Accesos", highlight: true,
      css: "px-8 py-6 align-top _usuario-access-td",
      getValue: e => `${e.FirstName} ${e.LastName || ""}`
    },
    {
      header: "Email",
      css: "px-6",
      getValue: e => e.Email
    },
    {
      header: "Status|Estado",
      headerCss: "w-80",
      css: "text-center",
      getValue: e => e.Status
    },
    {
      header: "Updated|Actualizado",
      headerCss: "w-144",
      css: "px-6 nowrap",
      getValue: e => formatTime(e.Updated, "Y-m-d h:n") as string
    }
  ]
</script>

<Page title="Users|Usuarios">
  <Layer type="content">
    <div class="h-full w-full">
      <div class="flex items-center justify-between mb-6" aria-label="Users toolbar with filter and create button">
        <FilterInput bind:value={filterText} css="mr-16 w-256" />
        <div class="flex items-center">
          <Button color="green" icon="icon-plus" label="Opens the side layer to create a new user."
            onClick={openCreateUsuarioLayer} />
        </div>
      </div>

      {#snippet accessSummaryRow(iconName: string, iconCss: string, accessNames: string[])}
        <div class="_usuario-access-row">
          <i class={`${iconName} _usuario-access-icon ${iconCss}`}></i>
          <span class="text-sm">{formatAccessSummary(accessNames)}</span>
        </div>
      {/snippet}

      <VTable
        columns={columns}
        data={usuariosService.usuarios}
        css="w-full"
        maxHeight="calc(80vh - 13rem)"
        estimateSize={72}
        filterText={filterText}
        getFilterContent={e => [e.Usuario, e.FirstName, e.LastName, e.Email].filter(x => x).join(" ").toLowerCase()}
        selected={usuarioForm?.ID}
        isSelected={(usuarioRecord, selectedUsuarioID) => usuarioRecord.ID === selectedUsuarioID}
        onRowClick={(selectedUsuario) => {
          openEditUsuarioLayer(selectedUsuario)
        }}
      >
        {#snippet cellRenderer(usuarioRecord: IUser, columnDefinition: ITableColumn<IUser>)}
          {#if columnDefinition.id === "usuario_info"}
            <div class="_usuario-info-cell">
              <div class="_usuario-info-name ff-semibold">{usuarioRecord.FirstName} {usuarioRecord.LastName || ""}</div>
              <div class="_usuario-info-login">{usuarioRecord.Usuario}</div>
            </div>
          {:else if columnDefinition.id === "usuario_accesos"}
            {@const usuarioAccessSummary = summarizeUsuarioAccesses(usuarioRecord)}
            <div class="_usuario-access-cell">
              {#if usuarioAccessSummary.readableAccessNames.length > 0}
                {@render accessSummaryRow("icon-eye", "text-blue-600", usuarioAccessSummary.readableAccessNames)}
              {/if}
              {#if usuarioAccessSummary.editableAccessNames.length > 0}
                {@render accessSummaryRow("icon-pencil", "text-red-600", usuarioAccessSummary.editableAccessNames)}
              {/if}
            </div>
          {/if}
        {/snippet}
      </VTable>
    </div>
  </Layer>

  <Layer
    id={1}
    type="side"
    sideLayerSize={760}
    title={(usuarioForm?.ID > 0 ? tr("Update|Actualizar") : tr("Save|Guardar")) + " " + tr("User|Usuario")}
    titleCss="h2 mb-6"
    css="px-12 py-10"
    contentCss="px-0"
    onSave={() => saveUsuario()}
    onDelete={usuarioForm?.ID > 0 ? () => saveUsuario(true) : undefined}
    onClose={() => {
      resetUsuarioForm()
    }}
  >
    <div class="grid grid-cols-24 gap-10 mt-8" aria-label="User form with username, names, email, profiles, and password">
      <Input
        bind:saveOn={usuarioForm}
        save="Usuario"
        css="col-span-24 md:col-span-12"
        label="Username|Usuario"
        required={true}
        disabled={usuarioForm?.ID > 0}
      />
      <Input
        bind:saveOn={usuarioForm}
        save="FirstName"
        css="col-span-24 md:col-span-12"
        label="First Name|Nombres"
        required={true}
      />
      <Input
        bind:saveOn={usuarioForm}
        save="LastName"
        css="col-span-24 md:col-span-12"
        label="Last Name|Apellidos"
      />
      <Input
        bind:saveOn={usuarioForm}
        save="DocumentNumber"
        css="col-span-24 md:col-span-12"
        label="Document #|Nº Documento"
      />
      <Input
        bind:saveOn={usuarioForm}
        save="JobTitle"
        css="col-span-24 md:col-span-12"
        label="Job Title|Cargo"
      />
      <Input
        bind:saveOn={usuarioForm}
        save="Email"
        css="col-span-24 md:col-span-12"
        label="Email"
      />
      <UserProfilesAccessSelector
        bind:saveOn={usuarioForm}
        perfiles={sortedPerfiles}
        accessGroupEntries={accessGroupEntries}
        accessCatalogEntries={accessCatalogEntries}
        accessLevelOptions={accessLevelOptions}
        accessCatalogLoadError={accessCatalogLoadError}
        css="col-span-24"
      />
      <Input
        bind:saveOn={usuarioForm}
        save="Password"
        css="col-span-24 md:col-span-12"
        label="Password"
        type="password"
        required={!usuarioForm.ID}
        placeholder={usuarioForm.ID > 0 ? tr("UNCHANGED|SIN CAMBIAR") : ""}
      />
      <Input
        bind:saveOn={usuarioForm}
        save="Password2"
        css="col-span-24 md:col-span-12"
        label="Confirm Password|Password (Repetir)"
        type="password"
        required={!usuarioForm.ID}
      />
    </div>
  </Layer>
</Page>

<style>
  :global(.vtable-row > td._usuario-access-td) {
    text-overflow: initial;
    white-space: normal;
  }

  ._usuario-info-cell {
    display: flex;
    flex-direction: column;
    gap: 2px;
    line-height: 1.2;
  }

  ._usuario-info-name {
    color: #304256;
  }

  ._usuario-info-login {
    color: #7762ba;
    line-height: 16px;
  }

  ._usuario-access-cell {
    display: flex;
    flex-direction: column;
    gap: 4px;
    line-height: 1.25;
    max-width: 100%;
  }

  ._usuario-access-row {
    align-items: flex-start;
    color: #435166;
    display: flex;
    gap: 6px;
  }

  ._usuario-access-icon {
    flex: 0 0 auto;
    margin-top: 2px;
  }

  ._usuario-access-row span {
    min-width: 0;
    overflow-wrap: anywhere;
    white-space: normal;
  }
</style>
