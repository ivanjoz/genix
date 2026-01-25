<script lang="ts">
import MobileMenu from '$core/core/MobileMenu.svelte';
import MainCarrusel from '$components/ecommerce/MainCarrusel.svelte';
import Header from '$components/ecommerce/Header.svelte';
import ProductCards from '$components/ecommerce/ProductCards.svelte';
import { ICategoriaProducto } from '$components/ecommerce/store.svelte';

  let categorias: ICategoriaProducto[] = [
    { Name: "Perfumes", Image: "images/categoria_1.webp" },
    { Name: "Casacas", Image: "images/casaca_icon_sm3.webp" },
    { Name: "Zapatos", Image: "images/categoria_1.webp" },
    { Name: "Relojes", Image: "images/categoria_1.webp" },
    { Name: "Decoración", Image: "images/categoria_1.webp" },
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
