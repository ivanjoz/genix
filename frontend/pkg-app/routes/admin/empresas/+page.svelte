<script lang="ts">
import Input from '$ui/components/Input';
import Modal from '$ui/components/Modal';
import Page from '$ui/components/Page';
import VTable from '$ui/components/VTable/index';
  import type { ITableColumn } from "$ui/VTable/types.ts"
import { Notify, throttle } from '$core/lib/helpers';
import { Core, closeModal } from '$core/core/store.svelte';
import { formatTime } from '$core/lib/helpers';
  import pkg from 'notiflix'
const { Loading } = pkg
  import { EmpresasService, postEmpresa, type IEmpresa } from "./empresas.svelte"

  const empresasService = new EmpresasService()

  let filterText = $state("")
  let empresaForm = $state({} as IEmpresa)

  async function saveEmpresa(isDelete?: boolean) {
    const form = empresaForm

    if ((form.Nombre?.length || 0) < 3) {
      Notify.failure("El nombre de la empresa debe tener al menos 3 caracteres.")
      return
    }

    if ((form.RUC?.length || 0) < 8) {
      Notify.failure("El RUC debe tener al menos 8 caracteres.")
      return
    }

    if (isDelete) {
      form.ss = 0
    }

    Loading.standard("Guardando Empresa...")
    try {
      const result = await postEmpresa(form)

      if (isDelete) {
        empresasService.removeEmpresa(form.id)
      } else {
        if (!form.id) {
          form.id = result.id
        }
        empresasService.updateEmpresa(form)
      }

      closeModal(1)
      Notify.success("Empresa guardada correctamente")
    } catch (error) {
      Notify.failure(error as string)
    }
    Loading.remove()
  }

  const columns: ITableColumn<IEmpresa>[] = [
    {
      header: "ID",
      headerCss: "w-54",
      cellCss: "text-center ff-bold",
      getValue: e => e.id
    },
    {
      header: "Nombre",
      highlight: true,
      cellCss: "px-6 c-blue",
      getValue: e => e.Nombre
    },
    {
      header: "Razón Social",
      getValue: e => e.RazonSocial
    },
    {
      header: "RUC",
      headerCss: "w-120",
      cellCss: "px-6",
      getValue: e => e.RUC
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
        empresaForm = { ...rec }
        Core.openModal(1)
      }
    }
  ]
</script>

<Page title="Empresas">
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
          empresaForm = {
            ss: 1,
            SmtpConfig: {},
            CulquiConfig: {}
          } as IEmpresa
          Core.openModal(1)
        }} aria-label="Agregar empresa">
          <i class="icon-plus"></i>
        </button>
      </div>
    </div>

    <VTable
      columns={columns}
      data={empresasService.empresas}
      css="w-full"
      maxHeight="calc(80vh - 13rem)"
      filterText={filterText}
      getFilterContent={e => [e.Nombre, e.RazonSocial, e.RUC, e.Email].filter(x => x).join(" ").toLowerCase()}
    >
    </VTable>
  </div>

  <Modal
    id={1}
    size={6}
    title={(empresaForm?.id > 0 ? "Actualizar" : "Guardar") + " Empresa"}
    isEdit={empresaForm?.id > 0}
    onSave={() => saveEmpresa()}
    onDelete={empresaForm?.id > 0 ? () => saveEmpresa(true) : undefined}
  >
    <div class="grid grid-cols-24 gap-10 p-6">
      <Input
        bind:saveOn={empresaForm}
        save="Nombre"
        css="col-span-24 md:col-span-12"
        label="Nombre"
        required={true}
      />
      <Input
        bind:saveOn={empresaForm}
        save="RazonSocial"
        css="col-span-24 md:col-span-12"
        label="Razón Social"
      />
      <Input
        bind:saveOn={empresaForm}
        save="RUC"
        css="col-span-24 md:col-span-8"
        label="RUC"
        required={true}
      />
      <Input
        bind:saveOn={empresaForm}
        save="Email"
        css="col-span-24 md:col-span-8"
        label="Email"
        type="email"
      />
      <Input
        bind:saveOn={empresaForm}
        save="Telefono"
        css="col-span-24 md:col-span-8"
        label="Teléfono"
      />
      <Input
        bind:saveOn={empresaForm}
        save="Representante"
        css="col-span-24 md:col-span-12"
        label="Representante"
      />
      <Input
        bind:saveOn={empresaForm}
        save="Ciudad"
        css="col-span-24 md:col-span-12"
        label="Ciudad"
      />
      <Input
        bind:saveOn={empresaForm}
        save="Direccion"
        css="col-span-24"
        label="Dirección"
        useTextArea={true}
        rows={2}
      />
    </div>
  </Modal>
</Page>
