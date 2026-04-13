<script lang="ts" generics="TItem">
  import SvelteVirtualList from '@humanspeak/svelte-virtual-list'
  import { onDestroy, onMount, untrack, type Snippet } from 'svelte'

  interface VirtualCardsProps<TItem> {
    items: TItem[]
    height?: string
    maxColumns?: number
    mobileBreakpointPx?: number
    estimatedRowHeight?: number
    bufferSize?: number
    containerCss?: string
    rowGapPx?: number
    columnGapPx?: number
    emptyMessage?: string
    children?: Snippet<[TItem, number]>
  }

  let {
    items = [],
    height = 'calc(100vh - 180px - var(--header-height))',
    maxColumns = 3,
    mobileBreakpointPx = 600,
    estimatedRowHeight = 220,
    bufferSize = 4,
    containerCss = '',
    rowGapPx = 12,
    columnGapPx = 12,
    emptyMessage = 'No se encontraron registros.',
    children,
  }: VirtualCardsProps<TItem> = $props()

  let containerElement = $state<HTMLDivElement | undefined>(undefined)
  let containerWidthPx = $state(0)
  let viewportWidthPx = $state(0)
  let resizeObserver: ResizeObserver | undefined

  const updateContainerWidth = () => {
    if (!containerElement) { return }
    containerWidthPx = Math.max(0, Math.floor(containerElement.clientWidth))
  }

  const updateViewportWidth = () => {
    // Use the page viewport as the primary responsive signal so cards collapse consistently across layouts.
    viewportWidthPx = Math.max(0, Math.floor(window.innerWidth || 0))
  }

  const activeColumns = $derived.by(() => {
    const responsiveWidthPx = viewportWidthPx || containerWidthPx

    if (responsiveWidthPx > 0 && responsiveWidthPx <= mobileBreakpointPx) {
      return 1
    }
    return Math.max(1, Math.floor(maxColumns || 1))
  })

  const itemRows = $derived.by(() => {
    const rows: Array<Array<TItem | null>> = []

    for (let rowStartIndex = 0; rowStartIndex < items.length; rowStartIndex += activeColumns) {
      const rowItems = items.slice(rowStartIndex, rowStartIndex + activeColumns)
      rows.push(Array.from({ length: activeColumns }, (_, columnIndex) => rowItems[columnIndex] || null))
    }

    untrack(() => {
      console.debug('[VirtualCards] rows rebuild', {
        itemsCount: items.length,
        activeColumns,
        rowsCount: rows.length,
        estimatedRowHeight,
      })
    })

    return rows
  })

  onMount(() => {
    updateViewportWidth()
    updateContainerWidth()

    if (!containerElement) { return }

    // Keep responsive columns tied to the page width, not only to the card shell width.
    window.addEventListener('resize', updateViewportWidth)
    resizeObserver = new ResizeObserver(() => {
      updateContainerWidth()
    })
    resizeObserver.observe(containerElement)

    return () => {
      window.removeEventListener('resize', updateViewportWidth)
    }
  })

  onDestroy(() => {
    resizeObserver?.disconnect()
  })
</script>

<div
  bind:this={containerElement}
  class={`virtual-cards-container ${containerCss}`}
  style={`height:${height}`}
>
  {#if items.length === 0}
    <div class="virtual-cards-empty-message">
      {emptyMessage}
    </div>
  {:else}
    <SvelteVirtualList
      items={itemRows}
      defaultEstimatedItemHeight={estimatedRowHeight}
      bufferSize={bufferSize}
      itemsClass="w-full [&>div]:w-full"
    >
      {#snippet renderItem(itemRow, rowIndex)}
        <div class="w-full">
          <div
            class="grid"
            style={`grid-template-columns:repeat(${activeColumns}, minmax(0, 1fr));column-gap:${columnGapPx}px`}
          >
            {#each itemRow as item, columnIndex (`${rowIndex}-${columnIndex}`)}
              {#if item}
                <div style={`margin-bottom:${rowGapPx}px`}>
                  {@render children?.(item, (rowIndex * activeColumns) + columnIndex)}
                </div>
              {:else}
                <div></div>
              {/if}
            {/each}
          </div>
        </div>
      {/snippet}
    </SvelteVirtualList>
  {/if}
</div>

<style>
  .virtual-cards-container {
    min-height: 0;
    overflow: hidden;
  }

  .virtual-cards-container :global(.virtual-list-container) {
    height: 100%;
  }

  .virtual-cards-container :global(.virtual-list-items) {
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding-bottom: 12px;
  }

  .virtual-cards-empty-message {
    color: #6b7280;
  }
</style>
