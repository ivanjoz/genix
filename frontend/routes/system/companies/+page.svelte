<script lang="ts">
  import { useUI } from '@genix/ui';
  import Input from '$components/form/Input.svelte';
  import Modal from '$components/layers/Modal.svelte';
  import Page from '$domain/Page.svelte';
  import { Notify } from '$libs/helpers';
  import { tr } from '$core/store.svelte';
  import pkg from 'notiflix';
  import CompanyCards from './CompanyCards.svelte';
  import CompanyCreditBudget from './CompanyCreditBudget.svelte';
  import { EmpresasService, postEmpresa, type ICompany } from './empresas.svelte';

  const { Loading } = pkg;
  const ui = useUI();
  const empresasService = new EmpresasService();

  let empresaForm = $state({} as ICompany);
  let companyRefreshVersion = $state(0);

  const openCreateCompany = () => {
    empresaForm = { ss: 1, SmtpConfig: {}, CulquiConfig: {} } as ICompany;
    ui.openModal(1);
  };

  const openEditCompany = (company: ICompany) => {
    empresaForm = { ...company };
    ui.openModal(1);
  };

  async function saveEmpresa(isDelete?: boolean) {
    const form = empresaForm;

    if ((form.Name?.length || 0) < 3) {
      Notify.failure(tr('Company name must be at least 3 characters.|El nombre de la empresa debe tener al menos 3 caracteres.'));
      return;
    }
    if ((form.RUC?.length || 0) < 8) {
      Notify.failure(tr('RUC must be at least 8 characters.|El RUC debe tener al menos 8 caracteres.'));
      return;
    }
    if (isDelete) form.ss = 0;

    Loading.standard(tr('Saving Company...|Guardando Empresa...'));
    try {
      const result = await postEmpresa(form);
      if (isDelete) {
        empresasService.removeEmpresa(form.id);
      } else {
        if (!form.id) form.id = Number(result.id || result.ID || 0);
        empresasService.updateEmpresa(form);
      }
      companyRefreshVersion += 1;
      ui.closeModal(1);
      Notify.success(tr('Company saved successfully|Empresa guardada correctamente'));
    } catch (error) {
      Notify.failure(error as string);
    } finally {
      Loading.remove();
    }
  }
</script>

<Page title="Companies|Empresas">
  <CompanyCards
    companies={empresasService.empresas}
    refreshVersion={companyRefreshVersion}
    onCreate={openCreateCompany}
    onEdit={openEditCompany}
  />

  <Modal
    id={1}
    size={6}
    title={(empresaForm?.id > 0 ? tr('Update|Actualizar') : tr('Save|Guardar')) + ' ' + tr('Company|Empresa')}
    isEdit={empresaForm?.id > 0}
    onSave={() => saveEmpresa()}
    onDelete={empresaForm?.id > 0 ? () => saveEmpresa(true) : undefined}
  >
    <div class="grid grid-cols-24 gap-10 p-6" aria-label="Company form with name, RUC, email, phone, representative, city, and address">
      <Input
        bind:saveOn={empresaForm}
        save="Name"
        css="col-span-24 md:col-span-12"
        label="Name|Nombre"
        required={true}
      />
      <Input
        bind:saveOn={empresaForm}
        save="LegalName"
        css="col-span-24 md:col-span-12"
        label="Legal Name|Razón Social"
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
        label="Phone|Teléfono"
      />
      <Input
        bind:saveOn={empresaForm}
        save="Representante"
        css="col-span-24 md:col-span-12"
        label="Representative|Representante"
      />
      <Input
        bind:saveOn={empresaForm}
        save="Ciudad"
        css="col-span-24 md:col-span-12"
        label="City|Ciudad"
      />
      <Input
        bind:saveOn={empresaForm}
        save="Direccion"
        css="col-span-24"
        label="Address|Dirección"
        useTextArea={true}
        rows={2}
      />
      {#if empresaForm?.id > 0}
        <CompanyCreditBudget companyID={empresaForm.id} />
      {/if}
    </div>
  </Modal>
</Page>
