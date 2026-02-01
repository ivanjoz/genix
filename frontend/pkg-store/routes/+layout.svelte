<script lang="ts">
  // @render 'svelte';
  import "./store.css";
  import "$ui/libs/fontello-prerender.css";
import { productosServiceState } from '$services/services/productos.svelte';
  let { children, data } = $props();

  productosServiceState.categorias = data.productos.categorias
  productosServiceState.productos = data.productos.productos
</script>

<svelte:head>
  <link rel="stylesheet" href="libs/fontello-embedded.css">
</svelte:head>

{@render children()}
