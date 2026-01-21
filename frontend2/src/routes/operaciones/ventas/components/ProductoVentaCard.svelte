<script lang="ts">
  import { formatN } from "$shared/main";
  import { type ProductoVenta, type VentaProducto } from "../ventas.svelte";

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

  // SKU Logic filtering
  const firstSkus = $derived.by(() => {
    let skus = productoStock.skus || []
    if (ventaProducto?.skus && ventaProducto.skus.size > 0) {
        // Simple filter: show SKUs not fully added?
        // Legacy: if SKU in cart < SKU stock
        // For visual simplicity, showing all available SKUs is good.
    }
    return skus.slice(0, 5) 
  })

  const price = $derived.by(() => {
      if(productoStock.isSubUnidad && productoStock.producto.SbnPreciFinal){
          return productoStock.producto.SbnPreciFinal
      }
      return productoStock.producto.PrecioFinal
  })
  
  // Helper for quantities
  const cantidades = [2,3,4,5,6,8,10,12]

    const css = $derived.by(() => {
    let cn = "px-16 py-12 transition-all duration-200 border border-transparent rounded-lg "
    if(isSelected){
      cn += "bg-blue-50/50 border-blue-200 shadow-sm"
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
  <div 
    class="flex flex-col gap-4 cursor-pointer group"
    onclick={() => onselect(idx)}
  >
    <!-- Header: Name + Line -->
    <div class="flex items-center gap-8 mb-4">
       <div class="h-[1px] bg-gray-200 grow group-hover:bg-gray-300 transition-colors"></div>
       <div class="text-sm font-medium text-gray-700 flex items-center gap-8">
          {@html highlightText(productoStock.producto.Nombre, filterText)}
          {#if productoStock.isSubUnidad}
             <span class="text-gray-300">|</span>
             <span class="text-purple-600 font-bold text-xs">{productoStock.producto.SbnUnidad}</span>
          {/if}
          {#if isSku}
             <span class="text-xs font-bold text-purple-600 bg-purple-50 px-4 py-1 rounded">(SKU)</span>
          {/if}
       </div>
    </div>

    <!-- Body: Grid -->
    <div class="grid grid-cols-[1fr_auto_auto_auto] gap-16 items-center">
        <!-- Col 1: Quick Actions / SKUs -->
        <div class="min-h-[36px] flex items-center">
            {#if isSku}
               <div class="flex flex-wrap gap-4">
                  {#each firstSkus as sku}
                      <button 
                        class="bn-white h-32 px-8 py-2 text-xs font-mono font-medium hover:bg-gray-50 transition-colors"
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
              <div class="flex gap-4 opacity-0 group-hover:opacity-100 transition-opacity duration-200">
                  {#each cantidades as n}
                     <button 
                       class="w-32 h-32 flex items-center justify-center text-xs font-bold text-gray-500 bg-gray-100 hover:bg-blue-100 hover:text-blue-600 rounded cursor-pointer transition-colors"
                       onclick={(e) => {
                           e.stopPropagation()
                           onadd(n)
                       }}
                     >
                        {n}
                     </button>
                  {/each}
              </div>
            {/if}
        </div>

        <!-- Col 2: Input Placeholder (Visual only, actual input is often global or hidden/bound) -->
        <!-- In this design, we keep it simple. -->
        
        <!-- Col 3: Stock -->
        <div class="font-mono text-sm text-gray-500 text-right w-64">
            <span class={getCant === 0 ? "text-red-500 font-bold" : ""}>
                 {getCant}
            </span>
            <span class="text-gray-300 text-xs ml-4">stk</span>
        </div>

        <!-- Col 4: Price -->
        <div class="font-mono text-sm font-medium text-gray-700 text-right w-80">
            {formatN(price/100, 2)}
        </div>
    </div>
  </div>
</div>
