<script lang="ts">
  import { useUI } from '@genix/ui';
  const ui = useUI();
import Input from '$components/form/Input.svelte';
import LabelCell from '$components/form/LabelCell.svelte';
import Layer from '$components/layers/Layer.svelte';
import UserProfilesAccessSelector from './UserProfilesAccessSelector.svelte';
import VTable from '$components/vTable/VTable.svelte';
import type { ITableColumn } from '$components/vTable/types';
import { Notify } from '$libs/helpers';
import FilterInput from '$components/form/FilterInput.svelte';
import Button from '$components/buttons/Button.svelte';
import OptionsStrip from '$components/navigation/OptionsStrip.svelte';
import { tr } from '$core/store.svelte';
import { formatTime } from '$libs/helpers';
  import pkg from 'notiflix'
const { Loading } = pkg
  import type { IAccessGroupCatalogEntry, IAccessListCatalogEntry } from "./access-list-catalog"
  import { UsuariosService, PerfilesService, postUser, type IUser } from "./users-profiles.svelte"
  import AccessGroupBars from "./AccessGroupBars.svelte"
  import AccessUsersTable from "./AccessUsersTable.svelte"

  // The profiles service and the access catalog are owned by +page.svelte so both tabs read the
  // same records: a profile created on the Profiles tab is selectable here without a refetch.
  let { perfilesService, accessGroupEntries, accessCatalogEntries, accessCatalogLoadError = "" }: {
    perfilesService: PerfilesService
    accessGroupEntries: IAccessGroupCatalogEntry[]
    accessCatalogEntries: IAccessListCatalogEntry[]
    accessCatalogLoadError?: string
  } = $props()

  const usuariosService = new UsuariosService()

  let filterText = $state("")
  let usuarioForm = $state({} as IUser)

  const sortedPerfiles = $derived.by(() => {
    // Keep profile ordering stable for the selector without re-sorting inside the child component.
    return [...perfilesService.perfiles].sort((leftProfile, rightProfile) => {
      const nameComparison = leftProfile.Name.localeCompare(rightProfile.Name)
      return nameComparison !== 0 ? nameComparison : leftProfile.ID - rightProfile.ID
    })
  })

  const perfilesByID = $derived.by(() => {
    return new Map(sortedPerfiles.map((profileRecord) => [profileRecord.ID, profileRecord]))
  })

  // The layer titles itself with whoever is being edited, so the name has to survive a record with
  // only a login, and a brand-new record with neither.
  const usuarioLayerTitle = $derived.by(() => {
    const fullName = [usuarioForm?.FirstName, usuarioForm?.LastName].filter(Boolean).join(" ").trim()
    return fullName || usuarioForm?.User || "New User|Nuevo Usuario"
  })

  // The same grants, read from either end: "Por Usuario" answers what one person can do, "Por
  // Acceso" answers who can do one thing — the question an audit actually starts from.
  const USUARIOS_VIEW_BY_USER = 1
  const USUARIOS_VIEW_BY_ACCESS = 2
  let usuariosView = $state(USUARIOS_VIEW_BY_USER)

  const USUARIO_VIEW_INFO = 1
  const USUARIO_VIEW_ACCESS = 2
  let usuarioLayerView = $state(USUARIO_VIEW_INFO)

  const resetUsuarioForm = () => {
    // Initialize the create form with an active status so the layer always opens in a valid default state.
    usuarioForm = { Status: 1, ProfileIDs: [], AccesosGrants: [] } as unknown as IUser
  }

  const openCreateUsuarioLayer = () => {
    resetUsuarioForm()
    console.log("openCreateUsuarioLayer::")
    usuarioLayerView = USUARIO_VIEW_INFO
    ui.openSideLayer(1)
  }

  const openEditUsuarioLayer = (selectedUsuario: IUser) => {
    // Clone the selected record so the table does not update optimistically while the user edits the layer.
    usuarioForm = {
      ...selectedUsuario,
      ProfileIDs: [...(selectedUsuario.ProfileIDs || [])],
      AccesosGrants: [...(selectedUsuario.AccesosGrants || [])]
    }
    console.log("openEditUsuarioLayer::", $state.snapshot(usuarioForm))
    // The view is NOT reset here: reviewing accesses across several users is one task, and
    // snapping back to Información on every row click restarts it. Creating a user does reset,
    // since the required identity fields live on the Información tab.
    ui.openSideLayer(1)
  }

  async function saveUsuario(isDelete?: boolean) {
    const form = usuarioForm

    if ((form.User?.length || 0) < 4 || (form.FirstName?.length || 0) < 4) {
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
      const result = await postUser(form)

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
      ui.openSideLayer(0)
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
      getValue: e => e.ID,
      mobile: { order: 1, css: "col-span-6 ff-bold", icon: "[fa--tag]" }
    },
    {
      id: "usuario_info",
      header: "Username|Usuario", highlight: true,
      css: "px-8 py-6 align-top",
      getValue: e => e.User,
      mobile: { order: 2, css: "col-span-18" }
    },
    {
      id: "usuario_accesos", headerCss: "w-[47%]",
      header: "Access|Accesos", highlight: true,
      css: "px-8 py-6 align-top _usuario-access-td",
      getValue: e => `${e.FirstName} ${e.LastName || ""}`,
      mobile: { order: 5, css: "col-span-24", labelTop: "Access|Accesos" }
    },
    {
      header: "Email",
      css: "px-6",
      getValue: e => e.Email,
      mobile: { order: 3, css: "col-span-24", labelLeft: "Email:", if: e => !!e.Email }
    },
    {
      header: "Status|Estado",
      headerCss: "w-80",
      css: "text-center",
      getValue: e => e.Status,
      mobile: { order: 4, css: "col-span-12", labelLeft: "Estado:" }
    },
    {
      header: "Updated|Actualizado",
      headerCss: "w-144",
      css: "px-6 nowrap",
      getValue: e => formatTime(e.Updated, "Y-m-d h:n") as string,
      mobile: { order: 6, css: "col-span-24", labelLeft: "Actualizado:" }
    }
  ]
</script>

  <Layer type="content">
    <div class="h-full w-full">
      <div class="flex items-center justify-between mb-6" aria-label="Users toolbar with view switch, filter and create button">
        <div class="flex items-center">
          <OptionsStrip
            selected={usuariosView}
            options={[
              [USUARIOS_VIEW_BY_USER, "By User|Por Usuario"],
              [USUARIOS_VIEW_BY_ACCESS, "By Access|Por Acceso"],
            ]}
            buttonCss="ff-bold"
            css="mr-16"
            onSelect={(selectedOption) => { usuariosView = selectedOption[0] as number }}
          />
          <FilterInput bind:value={filterText} css="w-256" />
        </div>
        <div class="flex items-center">
          <Button color="green" icon="icon-[fa--plus]" label="Opens the side layer to create a new user."
            onClick={openCreateUsuarioLayer} />
        </div>
      </div>

      {#if usuariosView === USUARIOS_VIEW_BY_ACCESS}
      <AccessUsersTable
        usuarios={usuariosService.usuarios}
        {perfilesByID}
        {accessCatalogEntries}
        {accessGroupEntries}
        {filterText}
        onUsuarioSelect={(selectedUsuario) => {
          // Editing a user belongs to the other view, so picking one there hands the tab over to it
          // — with the record already open, which is the step the operator was heading for.
          usuariosView = USUARIOS_VIEW_BY_USER
          usuarioLayerView = USUARIO_VIEW_ACCESS
          openEditUsuarioLayer(selectedUsuario)
        }}
      />
      {:else}
      <VTable
        columns={columns}
        data={usuariosService.usuarios}
        css="w-full"
        maxHeight="calc(80vh - 13rem)"
        estimateSize={72}
        filterText={filterText}
        getFilterContent={e => [e.User, e.FirstName, e.LastName, e.Email].filter(x => x).join(" ").toLowerCase()}
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
              <div class="_usuario-info-login">{usuarioRecord.User}</div>
            </div>
          {:else if columnDefinition.id === "usuario_accesos"}
            <AccessGroupBars usuario={usuarioRecord} {perfilesByID} {accessCatalogEntries} {accessGroupEntries} />
          {/if}
        {/snippet}
      </VTable>
      {/if}
    </div>
  </Layer>

  <Layer
    id={1}
    type="side"
    sideLayerSize={760}
    title={usuarioLayerTitle}
    titleIcon="icon-[fa--user] w-20 h-20 mb-6 c-purple"
    titleCss="h2 mb-6"
    options={[[USUARIO_VIEW_INFO, "Information|Información"], [USUARIO_VIEW_ACCESS, "Access|Accesos"]]}
    bind:selected={usuarioLayerView}
    css="px-12 py-10"
    contentCss="px-0"
    onSave={() => saveUsuario()}
    onDelete={usuarioForm?.ID > 0 ? () => saveUsuario(true) : undefined}
    onClose={() => {
      resetUsuarioForm()
    }}
  >
    {#if usuarioLayerView === USUARIO_VIEW_INFO}
    <div class="grid grid-cols-24 gap-10 mt-8" aria-label="User form with username, names, email and password">
      <Input
        bind:saveOn={usuarioForm}
        save="User"
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
    {:else}
    <!-- Unmounting the selector is safe: its editable maps are rebuilt from the form's AccesosGrants
         every time it mounts, and that field is written on every grant. -->
    <div class="grid grid-cols-24 gap-10 mt-8" aria-label="User profiles and individual accesses">
      <!-- Read-only echo of the identity fields: who is being granted this is on the other tab. -->
      <LabelCell css="col-span-8" valueCss="text-[15px]" label="Username|Usuario" value={usuarioForm?.User} />
      <LabelCell css="col-span-8" valueCss="text-[15px]" label="Job Title|Cargo" value={usuarioForm?.JobTitle} />
      <LabelCell css="col-span-8" valueCss="text-[15px]" label="Email" value={usuarioForm?.Email} />
      <UserProfilesAccessSelector
        bind:saveOn={usuarioForm}
        perfiles={sortedPerfiles}
        accessGroupEntries={accessGroupEntries}
        accessCatalogEntries={accessCatalogEntries}
        accessCatalogLoadError={accessCatalogLoadError}
        css="col-span-24"
      />
    </div>
    {/if}
  </Layer>

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

</style>
