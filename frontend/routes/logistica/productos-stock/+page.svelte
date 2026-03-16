<script lang="ts">
import { Core } from '$core/store.svelte'
import Page from '$domain/Page.svelte'
import AbastecimientoView from './AbastecimientoView.svelte'
import GestionStockView from './GestionStockView.svelte'

  const pageOptions = [{ id: 1, name: 'Gestión Stock' }, { id: 2, name: 'Abastecimiento' }]
</script>

<Page title="productos-stock" options={pageOptions}>
  {#if Core.pageOptionSelected === 1}
    <GestionStockView />
  {:else if Core.pageOptionSelected === 2}
    <AbastecimientoView />
  {/if}
</Page>
