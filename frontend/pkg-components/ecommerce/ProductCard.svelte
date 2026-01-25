<script lang="ts">
import { formatN, formatTime } from '$core/lib/helpers';
  import type { IProducto } from "$services/productos.svelte";

  const {
    producto = null as IProducto, css = ""
  } = $props();

  import ImageHash from "./Imagehash.svelte";
    import { ProductsSelectedMap } from "./store.svelte";
  import s1 from "./styles.module.css";

  const prodCant = $derived.by(() => {
    return ProductsSelectedMap.get(producto.ID)?.cant || 0
  })
</script>

<div class="relative _8">
  <div class={["_1", css].join(" ")}>
    <ImageHash css="w-full h-[36vw] md:h-200" src={producto.Image?.n} folder="img-productos"/>
    <div class="_3 pb-2">
      <div class="_5 mt-6 mb-4 min-h-26 md:min-h-32 fx-c">
        {producto.Nombre}
      </div>
      <div class="px-4 ff-bold fs17">s/. {formatN(producto.PrecioFinal/100,2)}</div>
      <div class="_6 fx-c h-30 w-32">
        <i class="icon1-basket"></i>
      </div>
    </div>
    <div class="_2"
      onclick={ev => {
        ev.stopPropagation()
        console.log("hola")
        ProductsSelectedMap.set(producto.ID, { cant: prodCant + 1, producto })
        console.log("ProductsSelectedMap", ProductsSelectedMap)
      }}
    >
      {#if prodCant === 0}
        Agregar <i class="icon1-basket"></i>
      {/if}
      {#if prodCant > 0}
        Agregar más ({prodCant}) <i class="icon1-basket"></i>
      {/if}
    </div>
  </div>
</div>

<style>
  ._1 {
    position: relative;
    background-color: white;
    box-shadow: rgba(54, 56, 67, 0.2) 0px 2px 8px 0px;
    min-height: 180px;
    padding: 6px;
    border-radius: 10px;
    margin-bottom: 24px;
  }
  ._2 {
    color: rgb(100, 67, 160);
    position: absolute;
    border-radius: 0 0 10px 10px;
    height: 36px;
    align-items: center;
    justify-content: center;
    display: none;
    width: 100%;
    bottom: 0;
    left: 0;
    user-select: none;
  }
  ._3 {
    position: relative;
    z-index: 10;
    width: 100%;
  }
  ._5 {
    text-align: center;
  }
  ._6 {
    border-radius: 7px;
    background-color: rgb(228, 221, 248);
    color: rgb(100, 67, 160);
    flex-shrink: 0;
    position: absolute;
    bottom: -2px;
    right: -2px;
    outline: 2px solid rgba(255, 255, 255);
    user-select: none;
  }

  @media (min-width: 739px) {
    ._8:hover ._1 {
      padding-bottom: 22px;
      margin-bottom: -10px;
      outline: 2px solid rgb(202, 173, 255);
    }
    ._8:hover ._2{
      display: flex;
      background-color: rgb(228, 221, 248);
      cursor: pointer;
    }
    ._8:hover ._3{
      background-color: white;
      margin-top: -14px;
      margin-bottom: 14px;
    }
    ._2:hover, ._8 ._2:hover {
      background-color: rgb(111, 82, 179);
      height: 38px;
      padding-bottom: 2px;
      margin-bottom: -2px;
      margin-left: -2px;
      margin-right: -2px;
      width: calc(100% + 4px);
      color: white;
    }
    ._8:hover ._6 {
      display: none;
    }
  }

  @media (max-width: 740px) {
    ._8 ._2 {
      display: flex;
      background-color: rgb(228, 221, 248);
      cursor: pointer;
    }
    ._8 ._1 {
      padding-bottom: 38px;
      border-bottom: 2px solid rgb(172, 153, 226);
      margin-bottom: 12px;
    }
    ._6 {
      display: none;
    }
  }

</style>
