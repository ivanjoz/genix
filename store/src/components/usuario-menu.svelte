<script>
  import { layerOpenedState } from "./store.svelte";

  function toggleLayer() {
    layerOpenedState.id = layerOpenedState.id === 2 ? 0 : 2;
  }
</script>

<div class="relative w-120 hidden md:block">
  <button class="bn1 w-full" onclick={toggleLayer}>
    <i class="icon1-user"></i>
    <span>Mi Usuario</span>
  </button>

  {#if layerOpenedState.id === 2}
    <div class="cart-content absolute">¡Contenido del carrito aquí!</div>
  {/if}
</div>
