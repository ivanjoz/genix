<script lang="ts">
import { getProductEcommerceData, type ProductCatalog } from '$ecommerce/services/products.svelte';
import ProductCard from '$ecommerce/components/ProductCard.svelte';
  import s1 from "./styles.module.css";

  let catalog = $state<ProductCatalog | null>(null);

  getProductEcommerceData().then((loaded) => { catalog = loaded; });

  // Reactive: re-renders on the Phase 1 → Phase 2 catalog re-publish.
  const productos = $derived(catalog?.productos ?? []);
</script>

<div class="w-full flex justify-center overflow-x-hidden pt-2">
  <div class={"grid grid-cols-2 gap-x-12 md:gap-x-20 md:flex md:flex-wrap md:justify-center max-w-1680 w100-p12 p-8 md:p-0 " +
      s1.product_cards_ctn}
  >
    {#each productos as producto}
      <ProductCard css="w-full md:w-240" productoID={producto.ID} />
    {/each}
  </div>
</div>
