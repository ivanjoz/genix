<script lang="ts">
  import type { ComponentAST, ColorPalette, ComponentVariable } from '../../pkg-store/renderer/renderer-types';
  import { resolveTokens } from '../../pkg-store/renderer/token-resolver';
  import ProductCard from '$store/components/ProductCard.svelte';
  import ProductsByCategory from '$store/ecommerce-components/ProductsByCategory.svelte';
  import ProductCardHorizonal from '$store/components/ProductCardHorizonal.svelte';

  interface Props {
    element: ComponentAST;
    index: number;
    isSelected: boolean;
    isDragging: boolean;
    values: Record<string, string>;
    palette?: ColorPalette;
    paletteStyles: string;
    onSelect: (element: ComponentAST, index: number) => void;
    onDragStart: (e: DragEvent, index: number) => void;
    onDragEnd: (e: DragEvent) => void;
    onDragOver: (e: DragEvent, index: number) => void;
  }

  let {
    element,
    index,
    isSelected,
    isDragging,
    values,
    palette,
    paletteStyles,
    onSelect,
    onDragStart,
    onDragEnd,
    onDragOver
  }: Props = $props();

  function getResolvedCss(el: ComponentAST) {
    return resolveTokens(el.css, el.variables, values, palette);
  }

  function getResolvedLineCss(css: string, variables: ComponentVariable[] = []) {
    return resolveTokens(css, variables, values, palette);
  }

  const Tag = $derived(element?.semanticTag || (element?.tagName as any) || 'section');
</script>

{#snippet renderChildElement(child: ComponentAST)}
  {#if !child}
    <!-- Skip null elements -->
  {:else if child.tagName === 'ProductCard'}
    {#if child.productosIDs||[]}
      {#each child.productosIDs as productoID}
        <ProductCard productoID={productoID} css={getResolvedCss(child)}/>
      {/each}
    {/if}
  {:else if child.tagName === 'ProductCardHorizonal'}
    {#if child.productosIDs}
      {#each child.productosIDs as productoID}
        <ProductCardHorizonal productoID={productoID} css={getResolvedCss(child)}/>
      {/each}
    {/if}
  {:else if child.tagName === 'ProductsByCategory'}
  	<ProductsByCategory categoryID={(child.categoriasIDs||[])[0]} limit={child.limit}
   		css={getResolvedCss(child)}
    />
  {:else}
    {@const Tag = child.semanticTag || (child.tagName as any) || 'div'}
    <svelte:element 
      this={Tag}
      class={getResolvedCss(child)} 
      style={child.style || ''}
      {...child.attributes}
      aria-label={child.aria?.label}
      role={child.aria?.role}
      aria-hidden={child.aria?.hidden}
    >
      {#if child.text}
        {child.text}
      {/if}

      {#if child.textLines}
        {#each child.textLines as line}
          {@const LineTag = line.tag || 'span'}
          <svelte:element this={LineTag} class={getResolvedLineCss(line.css, element.variables)}>
            {line.text}
          </svelte:element>
        {/each}
      {/if}

      {#if child.children}
        {#each child.children as grandchild}
          {@render renderChildElement(grandchild)}
        {/each}
      {/if}
    </svelte:element>
  {/if}
{/snippet}

<div
  class="section-wrapper"
  class:section-selected={isSelected}
  class:is-dragging={isDragging}
  
  draggable="true"
  ondragstart={(e) => onDragStart(e, index)}
  ondragend={onDragEnd}
  ondragover={(e) => onDragOver(e, index)}
  
  onclick={() => onSelect(element, index)}
  role="button"
  tabindex="0"
  onkeydown={(e) => e.key === 'Enter' && onSelect(element, index)}
>
  <div class="section-outline"></div>
  <div class="section-label">
    <span class="section-label-icon">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <rect x="3" y="3" width="18" height="18" rx="2"/>
        <path d="M3 9h18M9 21V9"/>
      </svg>
    </span>
    <span>{element.tagName || 'section'}</span>
    <span class="section-label-hint">Click to edit • Drag to move</span>
  </div>
  
  <svelte:element 
    this={Tag}
    class={getResolvedCss(element)} 
    style={`${element.style || ''} ${paletteStyles}`}
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
        <svelte:element this={LineTag} class={getResolvedLineCss(line.css, element.variables)}>
          {line.text}
        </svelte:element>
      {/each}
    {/if}

    {#if element.children}
      {#each element.children as child}
        {@render renderChildElement(child)}
      {/each}
    {/if}
  </svelte:element>
</div>

<style>
  .section-wrapper {
    position: relative;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .section-wrapper.is-dragging {
    opacity: 0.4;
    cursor: grabbing;
  }

  .section-outline {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    border: 2px dashed #3b82f6;
    pointer-events: none;
    opacity: 0;
    transition: opacity 0.2s ease;
    z-index: 9999;
  }

  .section-wrapper:hover .section-outline {
    opacity: 1;
    box-shadow: inset 0 0 0 6px rgb(37 99 235 / 29%);
  }

  .section-wrapper.section-selected .section-outline {
    border: 2px solid #2563eb;
  }

  .section-label {
    position: absolute;
    top: -32px;
    left: 8px;
    display: flex;
    align-items: center;
    gap: 8px;
    background: #2563eb;
    color: white;
    font-size: 12px;
    font-weight: 600;
    padding: 6px 12px;
    border-radius: 6px;
    z-index: 100;
    opacity: 0;
    transform: translateY(8px);
    transition: all 0.2s ease;
    pointer-events: none;
    font-family: system-ui, -apple-system, sans-serif;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    box-shadow: 0 4px 12px rgba(37, 99, 235, 0.3);
  }

  .section-label-icon {
    display: flex;
    align-items: center;
    opacity: 0.8;
  }

  .section-label-hint {
    font-size: 10px;
    font-weight: 400;
    text-transform: none;
    letter-spacing: 0;
    opacity: 0.7;
    margin-left: 4px;
  }

  .section-wrapper:hover .section-label,
  .section-wrapper.section-selected .section-label {
    opacity: 1;
    transform: translateY(0);
  }

  .section-wrapper.section-selected .section-label {
    background: #1d4ed8;
  }

  .section-wrapper.section-selected .section-label-hint {
    display: none;
  }
</style>
