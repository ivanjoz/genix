<script lang="ts">
import Page from '$domain/Page.svelte'
import ProductStockMovement from './ProductStockMovement.svelte'
</script>

<Page title="productos-stock"
	options={[{ id: 1, name: "Movimiento" }, { id: 2, name: "Ingreso OC" }]}
>
  <ProductStockMovement />
</Page>
