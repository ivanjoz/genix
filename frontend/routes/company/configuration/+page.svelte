<script lang="ts">
import Page from '$domain/Page.svelte';
import OptionsStrip from '$components/navigation/OptionsStrip.svelte';
import { security } from '$libs/ui-runtime.svelte';
import CompanyTab from './CompanyTab.svelte';
import BackupsTab from './BackupsTab.svelte';
import { CONFIGURATION_ACCESS_ID } from './empresas.svelte';
import { BACKUPS_ACCESS_ID } from './backups.svelte';

  // Both accesses point at this route, so canAccessRoute only proves the page is reachable:
  // it grants entry to anyone holding either one. Each tab is gated on its own access id and
  // dropped from the strip when the user cannot read it.
  const tabOptions = ([
    [1, "My Company|Mi Empresa", CONFIGURATION_ACCESS_ID],
    [2, "Backups", BACKUPS_ACCESS_ID]
  ] as [number, string, number][]).filter(([, , accessID]) => security.checkAcceso(accessID, 1))

  let tabSelected = $state(tabOptions[0]?.[0] || 1)
</script>

<Page title="Configuration|Configuración">
  <OptionsStrip css="mb-12" buttonCss="ff-bold" options={tabOptions} selected={tabSelected}
    onSelect={(opt) => { tabSelected = opt[0] }} />
  {#if tabSelected === 1}
    <CompanyTab />
  {/if}
  {#if tabSelected === 2}
    <BackupsTab />
  {/if}
</Page>
