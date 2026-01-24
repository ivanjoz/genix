<script lang="ts">
  import { formatN } from "$lib/helpers";
  import type { IProducto } from "$services/productos.svelte";
  import ImageHash from "./Imagehash.svelte";
  import { addProductoCant, ProductsSelectedMap } from "./store.svelte";

  const {
    producto = null as IProducto, css = ""
  } = $props();

  const prodCant = $derived.by(() => {
    return ProductsSelectedMap.get(producto.ID)?.cant || 0
  })
</script>

<div class="flex relative _1 h-88 md:h-110 rounded-[7px]">
  <div class="p-8">
    <ImageHash css="w-88 md:w-100 h-[100%]" src={producto.Image?.n} folder="img-productos" />
  </div>
  <div class="flex flex-col h-full relative pt-6 pb-4 w-full">
    <div>{producto.Nombre}</div>
    <div class="flex items-center mt-auto w-full">
      <div class="flex">
        <button class="_2 h-28 w-28 rounded-[50%]" onclick={ev => {
          ev.stopPropagation()
          addProductoCant(producto,null,-1)
        }}>-</button>
        <input class="_3 text-center h-28 w-58" type="number"
          value={prodCant}
          onchange={ev => {
            ev.stopPropagation()
            addProductoCant(producto,parseInt((ev.target as HTMLInputElement).value))
          }}
        >
        <button class="_2 h-28 w-28 rounded-[50%]" onclick={ev => {
          ev.stopPropagation()
          addProductoCant(producto,null,1)
        }}>+</button>
      </div>
      <div class="flex ml-auto mr-6 fs17">
        <div class="mr-4">s/.</div>
        <div class="ff-bold">{ formatN(producto.Precio/100,2) }</div>
      </div>
    </div>
  </div>
  <button class="_4 absolute outline-0 border-none fx-c rounded-[50%] h-28 w-28 top-[-4px] right-[-4px]"
    onclick={ev => {
      ev.stopPropagation()
      addProductoCant(producto,0)
    }}
  >
    <i class="icon-cancel"></i>
  </button>
</div>

<style>
  ._1 {
    box-shadow: rgba(50, 50, 105, 0.15) 0px 2px 5px 0px, rgba(0, 0, 0, 0.1) 0px 0px 2px 0px;
  }
  ._2 {
    background-color: #e5e3f4;
    border: none;
  }
  ._2:hover {
    background-color: #5e51c0;
    color: white;
  }
  ._2, ._3 {
    outline: none;
  }
  ._4 {
    background-color: rgb(255, 233, 233);
    color: rgb(197, 71, 71);
  }
  ._4:hover {
    color: white;
    background-color: rgb(228, 94, 94);
  }
</style>
