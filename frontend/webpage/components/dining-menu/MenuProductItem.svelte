<script lang="ts">
  import ImageHash from '#components/files/Imagehash.svelte'
  import { MENU_PRODUCTS_VARIANT, type IMenuCatalogProduct } from '#core/dining-menu.ts'
  import { formatN } from '#libs/helpers.ts'

  // One product of the carta in one of the three presets (compact / with photo / cards). On the QR
  // page `onAdd` makes it orderable; in the builder it is a plain, non-interactive preview.
  let { product, variant, onAdd }: {
    product: IMenuCatalogProduct, variant: number, onAdd?: (product: IMenuCatalogProduct) => void
  } = $props()

  const priceLabel = $derived(`S/ ${formatN(product.Price / 100, 2)}`)
</script>

{#snippet photo(sizeCss: string)}
  {#if product.ImageName}
    <ImageHash css="{sizeCss} object-cover rounded-md" src={product.ImageName} folder="img-productos" />
  {:else}
    <div class="{sizeCss} rounded-md bg-black/10"></div>
  {/if}
{/snippet}

<!-- The row is a button only when it can be ordered, so the builder preview has no dead controls. -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<svelte:element this={onAdd ? 'button' : 'div'} type={onAdd ? 'button' : undefined}
  class="w-full text-left {variant === MENU_PRODUCTS_VARIANT.CARDS ? 'flex flex-col gap-6' : 'flex items-center gap-10'}"
  onclick={onAdd ? () => onAdd(product) : undefined}>
  {#if variant === MENU_PRODUCTS_VARIANT.CARDS}
    {@render photo('w-full aspect-[4/3]')}
    <div class="font-semibold leading-tight">{product.Name}</div>
    <div class="font-bold" style:color="var(--menu-accent)">{priceLabel}</div>
  {:else if variant === MENU_PRODUCTS_VARIANT.WITH_PHOTO}
    {@render photo('w-64 h-64 shrink-0')}
    <div class="flex-1 min-w-0">
      <div class="font-semibold leading-tight">{product.Name}</div>
      {#if product.Description}<div class="line-clamp-2 opacity-70">{product.Description}</div>{/if}
    </div>
    <div class="font-bold shrink-0" style:color="var(--menu-accent)">{priceLabel}</div>
  {:else}
    <div class="font-semibold leading-tight">{product.Name}</div>
    <div class="flex-1 border-b border-dotted border-current opacity-30 mb-4"></div>
    <div class="font-bold shrink-0" style:color="var(--menu-accent)">{priceLabel}</div>
  {/if}
</svelte:element>
