<script lang="ts">
  // @render 'svelte';
  import "./store.css";
  import "./tailwind.css";
  import "$domain/libs/fontello-prerender.css";
  import { ProductIndex } from "$libs/index-decoder/product-index"
  import { readBuildSunixTimeFromHeader } from "$libs/index-decoder/decoder";
  
  import { productosServiceState } from '$services/services/productos.svelte';
  import { Env } from "$core/env";
  let { children, data } = $props();

  productosServiceState.categorias = data.productos.categorias
  productosServiceState.productos = data.productos.productos
  
  const productsIndexUrl = Env.makeCDNRoute("live",`c${1}_products.idx`)
  console.log("productsIndexUrl", productsIndexUrl)
  
  fetch(productsIndexUrl)
    .then((response) => response.bytes())
    .then((binary) => {
      const updatedSunix = readBuildSunixTimeFromHeader(binary);
      const updatedUnix = updatedSunix * 2 + 1_000_000_000;
      const createdAt = new Date(updatedUnix * 1000);
      console.log("products.idx updated_sunix", updatedSunix);
      console.log("products.idx updated_unix", updatedUnix);
      console.log("products.idx created_at", createdAt.toISOString());
      try {
        const productIndex = new ProductIndex(binary);
        console.log("products.idx", productIndex);
      } catch (decodeError) {
        console.error("products.idx decode failed", decodeError);
      }
    })
    .catch((fetchError) => {
      console.error("products.idx fetch failed", fetchError);
    });
  
</script>

<svelte:head>
  <link rel="stylesheet" href="libs/fontello-embedded.css">
</svelte:head>

{@render children()}
