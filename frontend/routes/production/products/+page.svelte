<script lang="ts">
  import { PRODUCT_ORIGIN_MADE } from '#services/production/products.svelte.ts';
  import ProductCatalog from './ProductCatalog.svelte';
</script>

<!-- Productos: what the company makes (Origin = 2). Mercaderías lives in production/merchandise. -->
<ProductCatalog origin={PRODUCT_ORIGIN_MADE} />
