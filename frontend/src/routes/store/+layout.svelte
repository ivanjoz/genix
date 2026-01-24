<script lang="ts">
  // @render 'svelte';
  import "../../app.css";
  import "$lib/fontello-prerender.css";
  import { productosServiceState } from "$services/productos.svelte";
  let { children, data } = $props();

  productosServiceState.categorias = data.productos.categorias
  productosServiceState.productos = data.productos.productos
</script>

<svelte:head>
  {@html `<link rel="stylesheet" href="libs/fontello-embedded.css">`}
</svelte:head>

{@render children()}