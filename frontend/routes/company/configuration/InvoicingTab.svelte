<script lang="ts">
import InvoiceSeriesTable from './InvoiceSeriesTable.svelte';
import CompanySecretsPanel from './CompanySecretsPanel.svelte';
import type { EmpresaParametrosService } from './empresas.svelte';

  const { service }: { service: EmpresaParametrosService } = $props()
</script>

<!-- The series and the certificate that signs them are one subject, and neither writes through the
     company parameters endpoint: the series have their own, the credentials another. That is why
     this tab carries no Save of its own — each panel saves itself. -->
<div class="grid grid-cols-24 gap-14 items-start">
  <div class="col-span-24 lg:col-span-12">
    <InvoiceSeriesTable company={service.empresa} />
  </div>
  <div class="col-span-24 lg:col-span-12">
    <CompanySecretsPanel />
  </div>
</div>
