<script lang="ts">
  import Input from "$components/Input.svelte"
  import Modal from "$components/Modal.svelte"
  import Page from "$components/Page.svelte"
  import SearchCard from "$components/SearchCard.svelte"
  import VTable from "$components/VTable/vTable.svelte"
  import type { ITableColumn } from "$components/VTable/types"
  import { Notify, throttle } from "$core/helpers"
  import { Core, closeModal } from '$core/core/store.svelte'
  import { formatTime } from '$core/helpers'
  import pkg from 'notiflix'
const { Loading } = pkg
  import { UsuariosService, PerfilesService, postUsuario, type IUsuario } from "./usuarios.svelte"

  const usuariosService = new UsuariosService()
  const perfilesService = new PerfilesService()

  let filterText = $state("")
  let usuarioForm = $state({} as IUsuario)

  async function saveUsuario(isDelete?: boolean) {
    const form = usuarioForm

    if ((form.usuario?.length || 0) < 4 || (form.nombres?.length || 0) < 4) {
      Notify.failure("El usuario y el nombre deben tener al menos 4 caracteres.")
      return
    }

    if (form.password1) form.password1 = form.password1.trim()
    if (form.password2) form.password2 = form.password2.trim()

    if (!form.id || form.password1) {
      let err = ""
      if ((form.password1?.length || 0) < 6) {
        err = "El password tiene menos de 6 caracteres."
      } else if (form.password1 !== form.password2) {
        err = "Los password no coinciden."
      }
      if (err) {
        Notify.failure(err)
        return
      }
    }

    Loading.standard("Creando/Actualizando Usuario...")
    try {
      const result = await postUsuario(form)

      if (isDelete) {
        usuariosService.removeUsuario(form.id)
      } else {
        if (!form.id) {
          form.id = result.id
        }
        usuariosService.updateUsuario(form)
      }

      closeModal(1)
    } catch (error) {
      Notify.failure(error as string)
    }
    Loading.remove()
  }

  const columns: ITableColumn<IUsuario>[] = [
    {
      header: "ID",
      headerCss: "w-54",
      cellCss: "text-center ff-bold",
      getValue: e => e.id
    },
    {
      header: "Usuario", highlight: true,
      cellCss: "px-6 c-purple",
      getValue: e => e.usuario
    },
    {
      header: "Nombres", highlight: true,
      getValue: e => `${e.nombres} ${e.apellidos||""}`
    },
    {
      header: "Email",
      cellCss: "px-6",
      getValue: e => e.email
    },
    {
      header: "Estado",
      headerCss: "w-80",
      cellCss: "text-center",
      getValue: e => e.ss
    },
    {
      header: "Actualizado",
      headerCss: "w-144",
      cellCss: "px-6 nowrap",
      getValue: e => formatTime(e.upd, "Y-m-d h:n") as string
    },
    {
      header: "...",
      headerCss: "w-42",
      cellCss: "t-c",
      id: "actions",
      buttonEditHandler: (rec) => {
        usuarioForm = { ...rec }
        Core.openModal(1)
      }
    }
  ]
</script>

<Page title="Usuarios">
  <div class="h-full">
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
          usuarioForm = { ss: 1 } as IUsuario
          Core.openModal(1)
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
      getFilterContent={e => [e.usuario, e.nombres, e.apellidos, e.email].filter(x => x).join(" ").toLowerCase()}
    >
    </VTable>
  </div>

  <Modal 
    id={1}
    size={5}
    title={(usuarioForm?.id > 0 ? "Actualizar" : "Guardar") + " Usuario"}
    isEdit={usuarioForm?.id > 0}
    onSave={() => saveUsuario()}
    onDelete={usuarioForm?.id > 0 ? () => saveUsuario(true) : undefined}
  >
    <div class="grid grid-cols-24 gap-10 p-6">
      <Input 
        bind:saveOn={usuarioForm} 
        save="usuario"
        css="col-span-12" 
        label="Usuario" 
        required={true}
        disabled={usuarioForm?.id > 0}
      />
      <Input 
        bind:saveOn={usuarioForm} 
        save="nombres"
        css="col-span-12" 
        label="Nombres" 
        required={true}
      />
      <Input 
        bind:saveOn={usuarioForm} 
        save="apellidos"
        css="col-span-12" 
        label="Apellidos"
      />
      <Input 
        bind:saveOn={usuarioForm} 
        save="documentoNro"
        css="col-span-12" 
        label="Nº Documento"
      />
      <Input 
        bind:saveOn={usuarioForm} 
        save="cargo"
        css="col-span-12" 
        label="Cargo"
      />
      <Input 
        bind:saveOn={usuarioForm} 
        save="email"
        css="col-span-12" 
        label="Email"
      />
      <SearchCard 
        bind:saveOn={usuarioForm} 
        save="perfilesIDs"
        css="col-span-24"
        options={perfilesService.perfiles}
        keyId="id"
        keyName="nombre"
        label="PERFILES ::"
      />
      <Input 
        bind:saveOn={usuarioForm} 
        save="password1"
        css="col-span-12" 
        label="Password" 
        type="password"
        required={!usuarioForm.id}
        placeholder={usuarioForm.id > 0 ? "SIN CAMBIAR" : ""}
      />
      <Input 
        bind:saveOn={usuarioForm} 
        save="password2"
        css="col-span-12" 
        label="Password (Repetir)" 
        type="password"
        required={!usuarioForm.id}
      />
    </div>
  </Modal>
</Page>

