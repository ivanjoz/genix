<script lang="ts">
import { Core } from '$core/store.svelte';
import { formatN } from '$libs/helpers';
  import { type ProductoVenta, type VentaProducto } from "./sale_order.svelte";

  interface Props {
    idx: number
    productoStock: ProductoVenta
    isSelected: boolean
    ventaProducto?: VentaProducto
    filterText: string
    onadd: (cant: number, sku?: string) => void
    onselect: (idx: number) => void
    onmouseover: () => void
  }

  let {
    idx,
    productoStock,
    isSelected,
    ventaProducto,
    filterText,
    onadd,
    onselect,
    onmouseover
  }: Props = $props();

  let inputRef: HTMLInputElement | undefined = $state();

  // Highlight logic for search
  const highlightText = (text: string, highlight: string) => {
    if (!highlight) return text;
    const parts = text.split(new RegExp(`(${highlight})`, 'gi'));
    return parts.map((part, i) =>
      part.toLowerCase() === highlight.toLowerCase()
        ? `<span class="bg-yellow-200 text-black font-bold">${part}</span>`
        : part
    ).join('');
  };

  const isSku = $derived((productoStock.skus?.length || 0) > 0);
  const mobileSelectedCount = $derived(ventaProducto?.cantidad || 0)

  const getCant = $derived.by(() => {
     let ventaCant = ventaProducto?.cantidad || 0

     // Legacy logic for sub-units counting against parent stock
     // If this is a parent (P..), check if S.. exists
     // This logic was in legacy `ProductoVentaCard`.
     // We might need access to the whole map or pass down the related subunit quanity?
     // For now, simpler approach: pass accurate `ventaProducto` or handle in parent?
     // In legacy it received `ventasProductosMap`.

     // Let's assume passed `ventaProducto` corresponds to THIS card's key.
     // But strictly speaking, stock is shared if it's the main unit.
     // Let's implement basic stock substraction first.
     return productoStock.cant - ventaCant
  })

  const hasCriticalStock = $derived(getCant <= 2)

  // SKU Logic filtering
  const firstSkus = $derived.by(() => {
    let skus = productoStock.skus || []

    // Filter out SKUs that are fully in cart (no stock left)
    skus = skus.filter(s => {
        const inCart = ventaProducto?.skus?.get(s.SKU as string) || 0
        return (s.Quantity - inCart) > 0
    })

    return skus.slice(0, 5)
  })

  const price = $derived.by(() => {
      if(productoStock.isSubUnidad && productoStock.producto.SbnPreciFinal){
          return productoStock.producto.SbnPreciFinal
      }
      return productoStock.producto.PrecioFinal
  })

  const highlightedDisplayName = $derived.by(() => {
    const productName = highlightText(productoStock.producto.Nombre, filterText)
    if (!productoStock.presentationName) return productName

    const highlightedPresentationName = highlightText(productoStock.presentationName, filterText)
    return `${productName} <span class="text-blue-600 font-bold">(${highlightedPresentationName})</span>`
  })

  // Helper for quantities
  const desktopQuickQuantities = [2,3,4,5,6,8,10,12]
  const mobileQuickQuantities = [1,2,5,10]
  const quickQuantities = $derived.by(() => {
    const availableQuantities = Core.deviceType === 3 ? mobileQuickQuantities : desktopQuickQuantities
    return availableQuantities.filter((cantidad) => cantidad <= getCant)
  })

  const css = $derived.by(() => {
    let cn = "relative px-8 py-4 border border-transparent rounded-lg "
    if(isSelected){
      // Keep mobile selection quiet while restoring the stronger desktop ring from md and up.
      cn += "bg-white shadow-sm border-gray-200 md:ring-2 md:ring-blue-500 md:ring-inset md:shadow-[inset_0_0_12px_rgba(59,130,246,0.1)]"
    } else {
      cn += "bg-white hover:border-gray-200 hover:shadow-sm"
    }
    return cn
  })

  $effect(() => {
    if(isSelected && inputRef){
        // Focus or prepare input?
        // Parent controls the global input mostly, but we might want visual focus
    }
  })

</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<!-- svelte-ignore a11y_mouse_events_have_key_events -->
<div class={css}
  onmouseover={(e) => {
    e.stopPropagation()
    onmouseover()
  }}
>
  {#if mobileSelectedCount > 0}
    <div class="absolute -right-4 -top-4 z-20 flex h-28 min-w-28 items-center justify-center rounded-full bg-red-600 px-6 text-[12px] font-bold text-white shadow-[0_6px_14px_rgba(220,38,38,0.35)] md:hidden">
      {mobileSelectedCount}
    </div>
  {/if}
  <div class="flex relative flex-col gap-4 cursor-pointer group"
    onclick={() => onselect(idx)}
  >
    <!-- Header: Name + Line -->
    <div class="flex items-center gap-8">
       <div class="flex items-center gap-8 pr-20 leading-tight text-gray-700 md:pr-0">
          <span>{@html highlightedDisplayName}</span>
          {#if productoStock.isSubUnidad}
             <span class="text-gray-300">|</span>
             <span class="text-purple-600 font-bold text-xs">{productoStock.producto.SbnUnidad}</span>
          {/if}
          {#if isSku}
             <span class="text-xs font-bold text-purple-600 bg-purple-50 px-4 py-1 rounded">(SKU)</span>
          {/if}
       </div>
       <div class="h-[1px] bg-gray-200 grow group-hover:bg-gray-300 transition-colors"></div>
    </div>

    <!-- Body: Grid -->
    <div class="flex items-center gap-6">
        <!-- Col 1: Quick Actions / SKUs -->
        <div class="flex items-center">
            {#if isSku}
               <div class="flex flex-wrap gap-4">
                  {#each firstSkus as sku}
                      <button class="sku-button transition-colors"
                        onclick={(e) => {
                            e.stopPropagation()
                            onadd(1, sku.SKU)
                        }}
                      >
                       {sku.SKU}
                      </button>
                  {/each}
               </div>
            {:else}
              <div class={"flex w-50 shrink-0 flex-col items-center justify-center rounded bg-gray-50 py-2 leading-none md:hidden" + (hasCriticalStock ? " text-red-500" : " text-gray-600")}>
                  <span class="text-[13px] text-gray-500">Stock</span>
                  <span class="font-mono ff-bold">{getCant}</span>
              </div>
              <div class="z-10 flex flex-1 flex-wrap justify-center gap-4 md:flex-none md:opacity-0 md:duration-200 md:group-hover:opacity-100">
                  {#each quickQuantities as cantidad}
                     <button
                       class="flex h-30 w-[56px] min-w-[14%] items-center justify-center rounded bg-gray-100 text-xs font-bold text-gray-500 transition-colors hover:bg-blue-100 hover:text-blue-600 md:w-32 md:min-w-0"
                       onclick={(e) => {
                           e.stopPropagation()
                           onadd(cantidad)
                       }}
                     >
                        {cantidad}
                     </button>
                  {/each}
              </div>
            {/if}
        </div>

        <!-- Col 2: Input Placeholder (Visual only, actual input is often global or hidden/bound) -->
        <!-- In this design, we keep it simple. -->

        <!-- Col 3: Stock -->
        {#if !isSku}
          <div class={"absolute bottom-0 left-0 text-right text-sm text-gray-500 group-hover:invisible" + (hasCriticalStock ? " text-red-500 font-bold" : "")}>
            <div class="hidden w-50 font-mono md:block">
                {getCant}
            </div>
          </div>
        {/if}
        <!-- Col 4: Price -->
        <div class="font-mono ml-auto text-sm font-medium text-gray-700 text-right w-80">
            {formatN(price/100, 2)}
        </div>
    </div>
  </div>
</div>


<style>
  .sku-button {
    color: #285bf1;
    border-radius: 7px;
    padding: 5px 8px 4px 8px;
    background-color: #f6f6f6;
    border-bottom: 1px solid #b7b9d3;
    line-height: 1;
    font-size: 14px;
    font-family: mono;
  }
  .sku-button:hover {
    background-color: #5e84f7;
    border-bottom: 1px solid #5e84f7;
    color: white;
  }
</style>
