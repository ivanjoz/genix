<script lang="ts">
import Page from '$domain/Page.svelte';
import OptionsStrip from '$components/navigation/OptionsStrip.svelte';
import { security } from '$libs/ui-runtime.svelte';
import CompanyTab from './CompanyTab.svelte';
import StoreTab from './StoreTab.svelte';
import BackupsTab from './BackupsTab.svelte';
import { CONFIGURATION_ACCESS_ID, EmpresaParametrosService } from './empresas.svelte';
import { BACKUPS_ACCESS_ID } from './backups.svelte';

  // Both accesses point at this route, so canAccessRoute only proves the page is reachable:
  // it grants entry to anyone holding either one. Each tab is gated on its own access id and
  // dropped from the strip when the user cannot read it. Store shares the company-parameters
  // access because it edits the same record through the same endpoint.
  const tabOptions = ([
    [1, "My Company|Mi Empresa", CONFIGURATION_ACCESS_ID],
    [3, "Store|Tienda", CONFIGURATION_ACCESS_ID],
    [2, "Backups", BACKUPS_ACCESS_ID]
  ] as [number, string, number][]).filter(([, , accessID]) => security.checkAcceso(accessID, 1))

  let tabSelected = $state(tabOptions[0]?.[0] || 1)

  // One service for both company tabs: two instances would each hold their own copy of the record
  // and the last save would overwrite the other tab's edits, since the endpoint writes it whole.
  const service = new EmpresaParametrosService()
</script>

<Page title="Configuration|Configuración">
  <OptionsStrip css="mb-12" buttonCss="ff-bold" options={tabOptions} selected={tabSelected}
    onSelect={(opt) => { tabSelected = opt[0] }} />
  {#if tabSelected === 1}
    <CompanyTab {service} />
  {/if}
  {#if tabSelected === 3}
    <StoreTab {service} />
  {/if}
  {#if tabSelected === 2}
    <BackupsTab />
  {/if}
</Page>
