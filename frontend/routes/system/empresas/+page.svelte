<script lang="ts">
import Input from '$components/form/Input.svelte';
import Modal from '$components/layers/Modal.svelte';
import Page from '$domain/Page.svelte';
import VTable from '$components/vTable/VTable.svelte';
import type { ITableColumn } from '$components/vTable/types';
import { Notify } from '$libs/helpers';
import FilterInput from '$components/form/FilterInput.svelte';
import Button from '$components/buttons/Button.svelte';
import { Core, closeModal } from '$core/store.svelte';
import { formatTime } from '$libs/helpers';
  import pkg from 'notiflix'
const { Loading } = pkg
  import { EmpresasService, postEmpresa, type ICompany } from "./empresas.svelte"

  const empresasService = new EmpresasService()

  let filterText = $state("")
  let empresaForm = $state({} as ICompany)

  async function saveEmpresa(isDelete?: boolean) {
    const form = empresaForm

    if ((form.Name?.length || 0) < 3) {
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

  const columns: ITableColumn<ICompany>[] = [
    {
      header: "ID",
      headerCss: "w-54",
      css: "text-center ff-bold",
      getValue: e => e.id
    },
    {
      header: "Nombre",
      highlight: true,
      css: "px-6 c-blue",
      getValue: e => e.Name
    },
    {
      header: "Razón Social",
      getValue: e => e.LegalName
    },
    {
      header: "RUC",
      headerCss: "w-120",
      css: "px-6",
      getValue: e => e.RUC
    },
    {
      header: "Estado",
      headerCss: "w-80",
      css: "text-center",
      getValue: e => e.ss
    },
    {
      header: "Actualizado",
      headerCss: "w-144",
      css: "px-6 nowrap",
      getValue: e => formatTime(e.upd, "Y-m-d h:n") as string
    },
    {
      header: "...",
      headerCss: "w-42",
      css: "t-c",
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
    <div class="flex items-center justify-between mb-6" aria-label="Companies toolbar with filter and create button">
      <FilterInput bind:value={filterText} css="mr-16 w-256" />
      <div class="flex items-center">
        <Button color="green" icon="icon-plus" label="Opens the modal to create a new company." onClick={() => {
          empresaForm = { ss: 1, SmtpConfig: {}, CulquiConfig: {} } as ICompany
          Core.openModal(1)
        }} />
      </div>
    </div>

    <VTable
      columns={columns}
      data={empresasService.empresas}
      css="w-full"
      maxHeight="calc(80vh - 13rem)"
      filterText={filterText}
      getFilterContent={e => [e.Name, e.LegalName, e.RUC, e.Email].filter(x => x).join(" ").toLowerCase()}
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
    <div class="grid grid-cols-24 gap-10 p-6" aria-label="Company form with name, RUC, email, phone, representative, city, and address">
      <Input
        bind:saveOn={empresaForm}
        save="Name"
        css="col-span-24 md:col-span-12"
        label="Nombre"
        required={true}
      />
      <Input
        bind:saveOn={empresaForm}
        save="LegalName"
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
