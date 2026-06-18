<script lang="ts">
import Page from '$domain/Page.svelte'
import { Core } from '$core/store.svelte'
import ProductStockMovement from './ProductStockMovement.svelte'
import PurchaseOrderEntry from './PurchaseOrderEntry.svelte'
</script>

<Page title="Product Stock|Inventario de Productos"
	options={[{ id: 1, name: "Movement|Movimiento" }, { id: 2, name: "PO Entry|Ingreso OC" }]}
>
  {#if Core.pageOptionSelected === 2}
    <PurchaseOrderEntry />
  {:else}
    <ProductStockMovement />
  {/if}
</Page>
