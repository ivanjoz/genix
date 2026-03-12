<script lang="ts">
import Input from '$components/Input.svelte';
import Layer from '$components/Layer.svelte';
import Page from '$domain/Page.svelte';
import SearchCard from '$components/SearchCard.svelte';
import VTable from '$components/vTable/VTable.svelte';
import type { ITableColumn } from '$components/vTable/types';
import { Notify, throttle } from '$libs/helpers';
import { Core } from '$core/store.svelte';
import { formatTime } from '$libs/helpers';
  import pkg from 'notiflix'
const { Loading } = pkg
  import { UsuariosService, PerfilesService, postUsuario, type IUsuario } from "./usuarios.svelte"

  const usuariosService = new UsuariosService()
  const perfilesService = new PerfilesService()

  let filterText = $state("")
  let usuarioForm = $state({} as IUsuario)

  const resetUsuarioForm = () => {
    // Initialize the create form with an active status so the layer always opens in a valid default state.
    usuarioForm = { Status: 1 } as IUsuario
  }

  const openCreateUsuarioLayer = () => {
    resetUsuarioForm()
    console.log("openCreateUsuarioLayer::")
    Core.openSideLayer(1)
  }

  const openEditUsuarioLayer = (selectedUsuario: IUsuario) => {
    // Clone the selected record so the table does not update optimistically while the user edits the layer.
    usuarioForm = { ...selectedUsuario }
    console.log("openEditUsuarioLayer::", $state.snapshot(usuarioForm))
    Core.openSideLayer(1)
  }

  async function saveUsuario(isDelete?: boolean) {
    const form = usuarioForm

    if ((form.Usuario?.length || 0) < 4 || (form.Nombres?.length || 0) < 4) {
      Notify.failure("El usuario y el nombre deben tener al menos 4 caracteres.")
      return
    }

    if (form.Password) form.Password = form.Password.trim()
    if (form.Password2) form.Password2 = form.Password2.trim()

    if (!form.ID || form.Password) {
      let err = ""
      if ((form.Password?.length || 0) < 6) {
        err = "El password tiene menos de 6 caracteres."
      } else if (form.Password !== form.Password2) {
        err = "Los password no coinciden."
      }
      if (err) {
        Notify.failure(err)
        return
      }
    }

    Loading.standard("Creando/Actualizando Usuario...")
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

  const columns: ITableColumn<IUsuario>[] = [
    {
      header: "ID",
      headerCss: "w-54",
      cellCss: "text-center ff-bold",
      getValue: e => e.ID
    },
    {
      header: "Usuario", highlight: true,
      cellCss: "px-6 c-purple",
      getValue: e => e.Usuario
    },
    {
      header: "Nombres", highlight: true,
      getValue: e => `${e.Nombres} ${e.Apellidos || ""}`
    },
    {
      header: "Email",
      cellCss: "px-6",
      getValue: e => e.Email
    },
    {
      header: "Estado",
      headerCss: "w-80",
      cellCss: "text-center",
      getValue: e => e.Status
    },
    {
      header: "Actualizado",
      headerCss: "w-144",
      cellCss: "px-6 nowrap",
      getValue: e => formatTime(e.Updated, "Y-m-d h:n") as string
    }
  ]
</script>

<Page title="Usuarios">
  <Layer type="content">
    <div class="h-full w-full">
      <div class="flex items-center justify-between mb-6">
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
            openCreateUsuarioLayer()
          }} aria-label="Agregar usuario">
            <i class="icon-plus"></i>
          </button>
        </div>
      </div>

      <VTable
        columns={columns}
        data={usuariosService.usuarios}
        css="w-full"
        maxHeight="calc(80vh - 13rem)"
        filterText={filterText}
        getFilterContent={e => [e.Usuario, e.Nombres, e.Apellidos, e.Email].filter(x => x).join(" ").toLowerCase()}
        selected={usuarioForm?.ID}
        isSelected={(usuarioRecord, selectedUsuarioID) => usuarioRecord.ID === selectedUsuarioID}
        onRowClick={(selectedUsuario) => {
          openEditUsuarioLayer(selectedUsuario)
        }}
      >
      </VTable>
    </div>
  </Layer>

  <Layer
    id={1}
    type="side"
    sideLayerSize={760}
    title={(usuarioForm?.ID > 0 ? "Actualizar" : "Guardar") + " Usuario"}
    titleCss="h2 mb-6"
    css="px-12 py-10"
    contentCss="px-0"
    onSave={() => saveUsuario()}
    onDelete={usuarioForm?.ID > 0 ? () => saveUsuario(true) : undefined}
    onClose={() => {
      resetUsuarioForm()
    }}
  >
    <div class="grid grid-cols-24 gap-10 mt-8">
      <Input
        bind:saveOn={usuarioForm}
        save="Usuario"
        css="col-span-24 md:col-span-12"
        label="Usuario"
        required={true}
        disabled={usuarioForm?.ID > 0}
      />
      <Input
        bind:saveOn={usuarioForm}
        save="Nombres"
        css="col-span-24 md:col-span-12"
        label="Nombres"
        required={true}
      />
      <Input
        bind:saveOn={usuarioForm}
        save="Apellidos"
        css="col-span-24 md:col-span-12"
        label="Apellidos"
      />
      <Input
        bind:saveOn={usuarioForm}
        save="DocumentoNro"
        css="col-span-24 md:col-span-12"
        label="Nº Documento"
      />
      <Input
        bind:saveOn={usuarioForm}
        save="Cargo"
        css="col-span-24 md:col-span-12"
        label="Cargo"
      />
      <Input
        bind:saveOn={usuarioForm}
        save="Email"
        css="col-span-24 md:col-span-12"
        label="Email"
      />
      <SearchCard
        bind:saveOn={usuarioForm}
        save="PerfilesIDs"
        css="col-span-24"
        options={perfilesService.perfiles}
        keyId="ID"
        keyName="Nombre"
        label="PERFILES ::"
      />
      <Input
        bind:saveOn={usuarioForm}
        save="Password"
        css="col-span-24 md:col-span-12"
        label="Password"
        type="password"
        required={!usuarioForm.ID}
        placeholder={usuarioForm.ID > 0 ? "SIN CAMBIAR" : ""}
      />
      <Input
        bind:saveOn={usuarioForm}
        save="Password2"
        css="col-span-24 md:col-span-12"
        label="Password (Repetir)"
        type="password"
        required={!usuarioForm.ID}
      />
    </div>
  </Layer>
</Page>
