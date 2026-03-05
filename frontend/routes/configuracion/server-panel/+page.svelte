<script lang="ts">
import Page from '$domain/Page.svelte';
import OptionsStrip from '$components/OptionsStrip.svelte';
import DashboardView from './DashboardView.svelte';
import MemoryView from './MemoryView.svelte';

type ServerPanelViewID = 'dashboard' | 'memory';

const panelViewOptions = [
  { id: 'dashboard', name: 'Dashboard' },
  { id: 'memory', name: 'Memory' }
];

let selectedPanelView = $state<ServerPanelViewID>('dashboard');
</script>

<Page title="Server Panel">
  <div class="flex flex-col gap-12">
    <OptionsStrip
      options={panelViewOptions}
      selected={selectedPanelView}
      keyId="id"
      keyName="name"
      onSelect={(nextPanelView) => {
        selectedPanelView = nextPanelView.id as ServerPanelViewID;
      }}
      css="border-b border-slate-200"
    />

    {#if selectedPanelView === 'dashboard'}
      <DashboardView />
    {:else}
      <MemoryView />
    {/if}
  </div>
</Page>
