<script lang="ts">
import Page from '$domain/Page.svelte'
import { useUI } from '@genix/ui'
import ProductStockMovement from './ProductStockMovement.svelte'
import PurchaseOrderEntry from './PurchaseOrderEntry.svelte'
const ui = useUI()
</script>

<Page title="Product Stock|Inventario de Productos"
	options={[{ id: 1, name: "Movement|Movimiento" }, { id: 2, name: "PO Entry|Ingreso OC" }]}
>
  {#if ui.state.pageOptionSelected === 2}
    <PurchaseOrderEntry />
  {:else}
    <ProductStockMovement />
  {/if}
</Page>
