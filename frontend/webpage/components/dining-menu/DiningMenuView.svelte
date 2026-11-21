<script lang="ts">
  import {
    MENU_ALIGN, MENU_BLOCK, MENU_FONT_FAMILIES, MENU_PRODUCTS_VARIANT, readableTextColor,
    type IDiningMenuTheme, type IMenuCatalogProduct, type IResolvedMenuBlock,
  } from '#core/dining-menu.ts'
  import MenuProductItem from './MenuProductItem.svelte'

  // What the builder wires onto the view: the selection, the drag-and-drop state and its handlers.
  // The QR page passes none, and gets a plain read-only (orderable) carta.
  export interface IMenuBuilderHooks {
    selectedIndex: number
    // Where a drop would insert, as an index in the carta's blocks; null when nothing is dragged over.
    dropIndex: number | null
    // The products block a dragged product would join instead of making a new block.
    appendTargetIndex: number | null
    onSelect: (blockIndex: number) => void
    onDragStart: (event: DragEvent, blockIndex: number) => void
    onDragOver: (event: DragEvent, blockIndex: number) => void
    onDragEnd: () => void
  }

  // The same renderer draws the carta in the builder and on the customer's page, so the preview is
  // exactly what is served (docs/DINING_MENU_PLAN.md M5).
  let { resolvedBlocks, theme, builder, onAddProduct }: {
    resolvedBlocks: IResolvedMenuBlock[]
    theme: IDiningMenuTheme
    builder?: IMenuBuilderHooks
    onAddProduct?: (product: IMenuCatalogProduct) => void
  } = $props()

  const alignCss = (align?: number) => align === MENU_ALIGN.CENTER ? 'text-center' : 'text-left'
  const productsLayoutCss = (variant?: number) =>
    variant === MENU_PRODUCTS_VARIANT.CARDS ? 'grid grid-cols-2 gap-12' : 'flex flex-col gap-12'
</script>

{#snippet dropPlaceholder()}
  <div class="h-40 my-4 rounded-md border-2 border-dashed border-blue-solid bg-blue-solid/10" aria-hidden="true"></div>
{/snippet}

<div class="min-h-full p-16 flex flex-col gap-16"
  style:--menu-accent={theme.AccentColor} style:background={theme.BackgroundColor}
  style:color={readableTextColor(theme.BackgroundColor)} style:font-family={MENU_FONT_FAMILIES[theme.Font]}>
  {#each resolvedBlocks as resolved (resolved.index)}
    {@const block = resolved.block}
    {#if builder?.dropIndex === resolved.index}{@render dropPlaceholder()}{/if}

    <!-- In the builder each block is a drag source and a click target; on the QR page it is plain content. -->
    <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
    <div class="relative {builder ? 'cursor-pointer rounded-md outline-offset-4' : ''}"
      class:outline-2={builder && (builder.selectedIndex === resolved.index || builder.appendTargetIndex === resolved.index)}
      class:outline-blue-solid={builder && builder.selectedIndex === resolved.index}
      class:outline-green-solid={builder && builder.appendTargetIndex === resolved.index}
      role={builder ? 'button' : undefined} tabindex={builder ? 0 : undefined}
      draggable={builder ? true : undefined}
      onclick={builder ? () => builder.onSelect(resolved.index) : undefined}
      onkeydown={builder ? (event) => { if (event.key === 'Enter') builder.onSelect(resolved.index) } : undefined}
      ondragstart={builder ? (event) => builder.onDragStart(event, resolved.index) : undefined}
      ondragover={builder ? (event) => builder.onDragOver(event, resolved.index) : undefined}
      ondragend={builder ? () => builder.onDragEnd() : undefined}>
      <!-- While dragging, children must not steal the drag events from the block wrapper. -->
      <div class:pointer-events-none={!!builder}>
        {#if block.Kind === MENU_BLOCK.TITLE}
          <h2 class="text-[24px] font-bold leading-tight {alignCss(block.Align)}" style:color="var(--menu-accent)">{block.Text}</h2>
        {:else if block.Kind === MENU_BLOCK.SUBTITLE}
          <h3 class="text-[18px] font-semibold leading-tight {alignCss(block.Align)}">{block.Text}</h3>
        {:else if block.Kind === MENU_BLOCK.TEXT}
          <p class="whitespace-pre-line {alignCss(block.Align)}">{block.Text}</p>
        {:else if block.Kind === MENU_BLOCK.IMAGE}
          <figure class="flex flex-col gap-4">
            <img class="w-full rounded-md object-cover" src={block.ImageURL} alt={block.Text || ''} />
            {#if block.Text}<figcaption class="text-center opacity-70">{block.Text}</figcaption>{/if}
          </figure>
        {:else if block.Kind === MENU_BLOCK.DIVIDER}
          <hr class="border-t-2" style:border-color="var(--menu-accent)" />
        {:else if resolved.groups.length === 0}
          <!-- Only the builder reaches here: the public resolve drops an empty product block. -->
          <div class="p-12 rounded-md border border-dashed border-current opacity-60 text-center">
            {block.Kind === MENU_BLOCK.CATEGORY && !block.CategoryID ? 'Elige una categoría en el panel' : 'Sin productos: elige algunos en el panel'}
          </div>
        {:else}
          <div class="flex flex-col gap-12">
            {#each resolved.groups as group}
              <section class="flex flex-col gap-8">
                {#if group.heading}
                  <h3 class="text-[18px] font-bold border-b-2 pb-2" style:color="var(--menu-accent)" style:border-color="var(--menu-accent)">{group.heading}</h3>
                {/if}
                <div class={productsLayoutCss(block.Variant)}>
                  {#each group.products as product (product.ID)}
                    <MenuProductItem {product} variant={block.Variant || MENU_PRODUCTS_VARIANT.COMPACT} onAdd={onAddProduct} />
                  {/each}
                </div>
              </section>
            {/each}
          </div>
        {/if}
      </div>
    </div>
  {/each}
  {#if builder && builder.dropIndex !== null && builder.dropIndex >= resolvedBlocks.length}{@render dropPlaceholder()}{/if}
</div>
