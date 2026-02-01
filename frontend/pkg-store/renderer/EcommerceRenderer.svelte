<script lang="ts" module>
  export * from './renderer-types';
</script>

<script lang="ts">
  import type { ComponentAST, ColorPalette } from './renderer-types';
  import { resolveTokens, generatePaletteStyles } from './token-resolver';
  import ProductCard from '$store/components/ProductCard.svelte';
  import ProductCardHorizonal from '$store/components/ProductCardHorizonal.svelte';

  interface Props {
    elements: ComponentAST | ComponentAST[];
    values?: Record<string, string>;
    palette?: ColorPalette;
    isRoot?: boolean;
  }

  const {
    elements,
    values = {},
    palette,
    isRoot = true
  }: Props = $props();

  const handleClick = (element: ComponentAST) => {
    if (element.onClick) {
      element.onClick(element.id || 0);
    }
  };

  function getResolvedCss(element: ComponentAST) {
    return resolveTokens(element.css, element.variables, values, palette);
  }

  function getResolvedLineCss(css: string, variables: any[]) {
     return resolveTokens(css, variables, values, palette);
  }

  const paletteStyles = $derived(isRoot ? generatePaletteStyles(palette) : '');
</script>

{#snippet renderElement(element: ComponentAST, depth: number)}
  {#if !element}
    <!-- Skip null elements -->
  {:else if element.tagName === 'ProductCard'}
    {#if element.productos}
      {#each element.productos as producto}
        <ProductCard productoID={producto.ID} css={getResolvedCss(element)}/>
      {/each}
    {/if}
  {:else if element.tagName === 'ProductCardHorizonal'}
    {#if element.productos}
      {#each element.productos as producto}
        <ProductCardHorizonal producto={producto} css={getResolvedCss(element)}/>
      {/each}
    {/if}
  {:else}
    {@const Tag = element.semanticTag || (element.tagName as any) || 'div'}
    <Tag 
      class={getResolvedCss(element)} 
      style={`${element.style || ''} ${depth === 0 ? paletteStyles : ''}`}
      onclick={() => handleClick(element)}
      {...element.attributes}
      aria-label={element.aria?.label}
      role={element.aria?.role}
      aria-hidden={element.aria?.hidden}
    >
      {#if element.text}
        {element.text}
      {/if}

      {#if element.textLines}
        {#each element.textLines as line}
          {@const LineTag = line.tag || 'span'}
          <LineTag class={getResolvedLineCss(line.css, element.variables || [])}>
            {line.text}
          </LineTag>
        {/each}
      {/if}

      {#if element.children}
        {#each element.children as child}
          {@render renderElement(child, depth + 1)}
        {/each}
      {/if}
    </Tag>
  {/if}
{/snippet}

{#if Array.isArray(elements)}
  {#each elements as element}
    {@render renderElement(element, 0)}
  {/each}
{:else}
  {@render renderElement(elements, 0)}
{/if}
