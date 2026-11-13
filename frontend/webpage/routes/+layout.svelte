<script lang="ts">
  // @render 'svelte';
  import { onMount, untrack } from "svelte";
  import { browser } from "$app/env";
  import "./store.css";
  import "./tailwind.css";
  // Shared typography (Open Sans desktop / Inter mobile) — imported last so its
  // ≤749px remap wins the cascade. Same file the admin/builder uses.
  import "../../styles/fonts.css";

  import { preloadProductSearch } from "#core/product-search/product-search-runtime.ts";
  import { ecommerceCatalog, getEcommerceCatalog } from '#ecommerce/services/ecommerce-catalog.svelte.ts';
  import FloatingCart from "#ecommerce/components/FloatingCart.svelte";
  import ProductDetailLayer from "#ecommerce/components/ProductDetailLayer.svelte";
  import { capCartToStock, restoreCart } from "#ecommerce/components/store.svelte.ts";
  import { provideUi } from '@genix/ui';
  import { NotifyHost } from '@genix/ui/notify';
  import { genixUiRuntime } from '#libs/ui-runtime.svelte.ts';
  let { children } = $props();
  provideUi(genixUiRuntime);

  onMount(() => {
    restoreCart();
    // Kick off the single shared catalog load (fast main-thread first paint, then background
    // delta) and warm up the search index. Both share the same memoized catalog instance.
    void getEcommerceCatalog().catch((catalogLoadError) => {
      console.error("[StoreLayout] catalog load failed", catalogLoadError);
    });
    void preloadProductSearch().catch((productSearchPreloadError) => {
      console.error("[StoreLayout] ProductSearch preload failed", productSearchPreloadError);
    });
  });

  // Every catalog publish (first paint, then the delta) may lower what the store can sell: the cart
  // follows it.
  $effect(() => {
    if (ecommerceCatalog.version > 0) untrack(capCartToStock);
  });
</script>

{@render children()}
<!-- Client-only: the floating cart depends on client cart state and must not be
     baked into the prerendered HTML (it would flash empty/stale before hydration). -->
{#if browser}
  <FloatingCart />
  <!-- Opened by a grid card click (productDetailLayer in store.svelte.ts). -->
  <ProductDetailLayer />
  <!-- genix-ui raises session and request errors through notify. -->
  <NotifyHost />
{/if}
