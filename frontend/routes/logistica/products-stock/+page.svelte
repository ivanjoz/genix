<script lang="ts">
import { Core } from '$core/store.svelte'
import Page from '$domain/Page.svelte'
import ProductSupplyManagement from './ProductSupplyManagement.svelte'
import ProductStockMovement from './ProductStockMovement.svelte'

  const pageOptions = [{ id: 1, name: 'Gestión Stock' }, { id: 2, name: 'Abastecimiento' }]
</script>

<Page title="productos-stock" options={pageOptions}>
  {#if Core.pageOptionSelected === 1}
    <ProductStockMovement />
  {:else if Core.pageOptionSelected === 2}
    <ProductSupplyManagement />
  {/if}
</Page>
