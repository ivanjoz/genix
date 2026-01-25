<script lang="ts">
  // @render 'svelte';
  import "../../app.css";
  import "$lib/fontello-prerender.css";
import blurhashScript from '$ui/libs/blurhash';
import { productosServiceState } from '$services/services/productos.svelte';
  let { children, data } = $props();

  productosServiceState.categorias = data.productos.categorias
  productosServiceState.productos = data.productos.productos
</script>

<svelte:head>
	{@html '<script>' + blurhashScript + '</script>'}
  {@html `<link rel="stylesheet" href="libs/fontello-embedded.css">`}
</svelte:head>

{@render children()}
