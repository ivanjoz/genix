<script lang="ts">
  import FilterInput from '#components/form/FilterInput.svelte'
  import OptionsStrip from '#components/navigation/OptionsStrip.svelte'
  import { tr } from '#core/store.svelte.ts'
  import { formatN } from '#libs/helpers.ts'
  import type { IMenuCatalogProduct, IMenuCategory } from '#core/dining-menu.ts'
  import { MENU_BLOCK_TILES, MENU_DRAG } from './menu-builder.ts'

  // The builder's left panel: everything that can be dragged onto the carta — a block kind, a
  // product or a category. The catalog tabs have a search box; dropping is the canvas's job.
  let { products, categories }: { products: IMenuCatalogProduct[], categories: IMenuCategory[] } = $props()

  const TAB_BLOCKS = 1, TAB_PRODUCTS = 2, TAB_CATEGORIES = 3
  let activeTab = $state(TAB_BLOCKS)
  let filterText = $state('')

  const filteredProducts = $derived(products.filter((product) => product.Name.toLowerCase().includes(filterText)))
  const filteredCategories = $derived(categories.filter((category) => category.Name.toLowerCase().includes(filterText)))

  function startDrag(event: DragEvent, mimeType: string, payload: number) {
    if (!event.dataTransfer) return
    event.dataTransfer.setData(mimeType, String(payload))
    event.dataTransfer.effectAllowed = 'copy'
  }
</script>

<div class="flex flex-col gap-8 h-full">
  <OptionsStrip selected={activeTab} useMobileGrid containerCss="w-full" buttonCss="flex-1 justify-center"
    options={[[TAB_BLOCKS, 'Blocks|Bloques'], [TAB_PRODUCTS, 'Products|Productos'], [TAB_CATEGORIES, 'Categories|Categorías']]}
    onSelect={(option) => { activeTab = option[0] as number; filterText = '' }} />

  {#if activeTab !== TAB_BLOCKS}
    <FilterInput bind:value={filterText} placeholder={tr('Search|Buscar')} />
  {/if}

  <div class="flex flex-col gap-6 overflow-auto min-h-0">
    {#if activeTab === TAB_BLOCKS}
      {#each MENU_BLOCK_TILES as tile (tile.kind)}
        <div class="flex items-center gap-8 p-8 rounded-md border border-line bg-surface cursor-grab hover:bg-surface-soft"
          draggable="true" role="listitem" ondragstart={(event) => startDrag(event, MENU_DRAG.NEW_BLOCK, tile.kind)}>
          <i class="{tile.icon} text-fg-muted"></i>
          <span>{tr(tile.label)}</span>
        </div>
      {/each}
    {:else if activeTab === TAB_PRODUCTS}
      {#each filteredProducts as product (product.ID)}
        <div class="flex items-center justify-between gap-8 p-8 rounded-md border border-line bg-surface cursor-grab hover:bg-surface-soft"
          draggable="true" role="listitem" ondragstart={(event) => startDrag(event, MENU_DRAG.PRODUCT, product.ID)}>
          <span class="min-w-0 truncate">{product.Name}</span>
          <span class="shrink-0 text-fg-muted">S/ {formatN(product.Price / 100, 2)}</span>
        </div>
      {:else}
        <div class="p-8 text-fg-muted">{tr('No products sold in the restaurant.|No hay productos que se vendan en el restaurante.')}</div>
      {/each}
    {:else}
      {#each filteredCategories as category (category.ID)}
        <div class="flex items-center gap-8 p-8 rounded-md border border-line bg-surface cursor-grab hover:bg-surface-soft"
          draggable="true" role="listitem" ondragstart={(event) => startDrag(event, MENU_DRAG.CATEGORY, category.ID)}>
          <i class="icon-[fa--tag] text-fg-muted"></i>
          <span class="min-w-0 truncate">{category.Name}</span>
        </div>
      {:else}
        <div class="p-8 text-fg-muted">{tr('No categories.|No hay categorías.')}</div>
      {/each}
    {/if}
  </div>
  <div class="text-fg-muted">{tr('Drag onto the phone. A product dropped on a "Picked products" block joins it.|Arrastra al celular. Un producto soltado sobre un bloque "Productos puntuales" se añade a él.')}</div>
</div>
