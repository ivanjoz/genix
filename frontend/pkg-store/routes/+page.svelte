<script lang="ts">
import MobileMenu from '$ui/MobileMenu.svelte';
import MainCarrusel from '$store/components/MainCarrusel.svelte';
import Header from '$store/components/Header.svelte';
import ProductCards from '$store/components/ProductCards.svelte';
import type { ICategoriaProducto } from '$store/components/store.svelte';

  let categorias: ICategoriaProducto[] = [
    { Name: "Perfumes", Image: "images/categoria_11.webp" },
    { Name: "Casacas", Image: "images/uwgGHwD2aYl3.webp" },
    { Name: "Zapatos", Image: "images/MqcNHwj3iKeK.webp" },
    { Name: "Relojes", Image: "images/7ogGDobEysil.webp" },
    { Name: "Decoración", Image: "images/lHSFC4IPj2mN.webp" },
  ];
</script>

<Header />
<MobileMenu />
<MainCarrusel {categorias} />
<div class="h-800 _1">
  <div>hola</div>
  <ProductCards />
</div>
<div class="h-800 _1 flex"></div>

<style>
  ._1 {
    background-color: #f0eff3;
  }
</style>
