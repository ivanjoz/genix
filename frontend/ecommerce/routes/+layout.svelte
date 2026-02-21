<script lang="ts">
  // @render 'svelte';
  import "./store.css";
  import "./tailwind.css";
  import "$domain/libs/fontello-prerender.css";
import { productosServiceState } from '$services/services/productos.svelte';
    import { Env } from "$core/env";
  let { children, data } = $props();

  productosServiceState.categorias = data.productos.categorias
  productosServiceState.productos = data.productos.productos
  
  const productsIndexUrl = Env.makeCDNRoute("public",`c${1}_products.idx`)
  console.log("productsIndexUrl", productsIndexUrl)
  
  fetch(productsIndexUrl).then(e => e.bytes()).then(e => {
  	console.log("products.idx", e.length)
  })
  
</script>

<svelte:head>
  <link rel="stylesheet" href="libs/fontello-embedded.css">
</svelte:head>

{@render children()}
