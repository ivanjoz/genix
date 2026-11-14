<script lang="ts">
  import { PRODUCT_ORIGIN_BOUGHT } from '#services/production/products.svelte.ts';
  import ProductCatalog from '#routes/production/products/ProductCatalog.svelte';
</script>

<!-- Mercaderías: what the company buys to resell (Origin = 1). Same list and form as Productos. -->
<ProductCatalog origin={PRODUCT_ORIGIN_BOUGHT} />
