<script lang="ts">
import Page from '$domain/Page.svelte'
import { Core } from '$core/store.svelte'
import ProductStockMovement from './ProductStockMovement.svelte'
import PurchaseOrderEntry from './PurchaseOrderEntry.svelte'
</script>

<Page title="productos-stock"
	options={[{ id: 1, name: "Movimiento" }, { id: 2, name: "Ingreso OC" }]}
>
  {#if Core.pageOptionSelected === 2}
    <PurchaseOrderEntry />
  {:else}
    <ProductStockMovement />
  {/if}
</Page>
