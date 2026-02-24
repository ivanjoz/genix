<script lang="ts">
  // @render 'svelte';
  import { onMount } from "svelte";
  import "./store.css";
  import "./tailwind.css";
  import "$domain/libs/fontello-prerender.css";

  import { preloadProductSearch } from "$core/product-search/product-search-runtime";
  import { productosServiceState } from '$services/services/productos.svelte';
  let { children, data } = $props();

  productosServiceState.categorias = data.productos.categorias
  productosServiceState.productos = data.productos.productos

  onMount(() => {
    // Warm up product-search index as soon as store layout mounts.
    void preloadProductSearch().catch((productSearchPreloadError) => {
      console.error("[StoreLayout] ProductSearch preload failed", productSearchPreloadError);
    });
  });
</script>

<svelte:head>
  <link rel="stylesheet" href="libs/fontello-embedded.css">
</svelte:head>

{@render children()}
