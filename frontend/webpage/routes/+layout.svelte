<script lang="ts">
  // @render 'svelte';
  import { onMount } from "svelte";
  import { browser } from "$app/environment";
  import "./store.css";
  import "./tailwind.css";
  import "$domain/libs/fontello-prerender.css";

  import { preloadProductSearch } from "$core/product-search/product-search-runtime";
  import { getProductEcommerceData } from '$ecommerce/services/productos.svelte';
  import FloatingCart from "$ecommerce/components/FloatingCart.svelte";
  let { children } = $props();

  onMount(() => {
    // Kick off the single shared catalog load (fast main-thread first paint, then background
    // delta) and warm up the search index. Both share the same memoized catalog instance.
    void getProductEcommerceData().catch((catalogLoadError) => {
      console.error("[StoreLayout] catalog load failed", catalogLoadError);
    });
    void preloadProductSearch().catch((productSearchPreloadError) => {
      console.error("[StoreLayout] ProductSearch preload failed", productSearchPreloadError);
    });
  });
</script>

{@render children()}
<!-- Client-only: the floating cart depends on client cart state and must not be
     baked into the prerendered HTML (it would flash empty/stale before hydration). -->
{#if browser}
  <FloatingCart />
{/if}
