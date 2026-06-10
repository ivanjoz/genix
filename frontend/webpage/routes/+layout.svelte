<script lang="ts">
  // @render 'svelte';
  import { onMount } from "svelte";
  import { browser } from "$app/environment";
  import "./store.css";
  import "./tailwind.css";
  import "$domain/libs/fontello-prerender.css";

  import { preloadProductSearch } from "$core/product-search/product-search-runtime";
  import { ensureProductosLoaded } from '$services/services/productos.svelte';
  import FloatingCart from "$ecommerce/components/FloatingCart.svelte";
  let { children } = $props();

  onMount(() => {
    // Load the single shared catalog (lists + category maps) and warm up the search index. Both
    // resolve to the same shared instance, so the catalog is fetched only once.
    void ensureProductosLoaded().catch((catalogLoadError) => {
      console.error("[StoreLayout] catalog load failed", catalogLoadError);
    });
    void preloadProductSearch().catch((productSearchPreloadError) => {
      console.error("[StoreLayout] ProductSearch preload failed", productSearchPreloadError);
    });
  });
</script>

<svelte:head>
  <link rel="stylesheet" href="libs/fontello-embedded.css">
</svelte:head>

{@render children()}
<!-- Client-only: the floating cart depends on client cart state and must not be
     baked into the prerendered HTML (it would flash empty/stale before hydration). -->
{#if browser}
  <FloatingCart />
{/if}
