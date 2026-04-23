<script lang="ts" generics="TItem">
  import { onDestroy, onMount, tick, untrack, type Snippet } from 'svelte'

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
    useInnerPadding?: boolean
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
    useInnerPadding = false,
    emptyMessage = 'No se encontraron registros.',
    children,
  }: VirtualCardsProps<TItem> = $props()

  // DEBUG: toggle with window.__VC_DEBUG__ = true in the browser console (or leave on for now).
  const DEBUG = true
  const instanceTag = Math.random().toString(36).slice(2, 6)
  const dlog = (tag: string, payload?: unknown) => {
    if (!DEBUG) { return }
    try {
      console.info(`[VC:${instanceTag}] ${tag}`, payload ?? '')
    } catch {
      // ignore
    }
  }

  let containerElement = $state<HTMLDivElement | undefined>(undefined)
  let viewportElement = $state<HTMLDivElement | undefined>(undefined)
  let containerWidthPx = $state(0)
  let viewportWidthPx = $state(0)
  let viewportHeightPx = $state(0)
  let scrollTop = $state(0)
  // Hide content until initial measurements have settled — otherwise the user briefly sees rows
  // positioned with the estimated row height before ResizeObserver callbacks update them.
  let ready = $state(false)
  let containerResizeObserver: ResizeObserver | undefined
  let itemResizeObserver: ResizeObserver | undefined
  // Debounce-to-settle ready flip: any measurement activity resets the timer; when nothing arrives
  // for READY_QUIET_MS the content is revealed. READY_MAX_MS is a hard cap so slow-loading fonts
  // or never-settling layouts still reveal eventually.
  const READY_QUIET_MS = 80
  const READY_MAX_MS = 500
  let readyQuietTimer: ReturnType<typeof setTimeout> | null = null
  let readyMaxTimer: ReturnType<typeof setTimeout> | null = null

  const flipReady = (reason: string) => {
    if (ready) { return }
    ready = true
    if (readyQuietTimer !== null) { clearTimeout(readyQuietTimer); readyQuietTimer = null }
    if (readyMaxTimer !== null) { clearTimeout(readyMaxTimer); readyMaxTimer = null }
    dlog('ready', {
      reason,
      totalHeight,
      rowsLen: itemRows.length,
      viewportHeightPx,
      measuredCount: rowHeights.filter((h) => h > 0).length,
      firstHeights: rowHeights.slice(0, 8),
    })
  }

  const pokeReadyQuietTimer = () => {
    if (ready) { return }
    if (readyQuietTimer !== null) { clearTimeout(readyQuietTimer) }
    readyQuietTimer = setTimeout(() => {
      readyQuietTimer = null
      flipReady('quiet')
    }, READY_QUIET_MS)
  }

  const updateContainerWidth = () => {
    if (!containerElement) { return }
    containerWidthPx = Math.max(0, Math.floor(containerElement.clientWidth))
  }

  const updateViewportSize = () => {
    if (!viewportElement) { return }
    const next = Math.max(0, Math.floor(viewportElement.clientHeight))
    if (next !== viewportHeightPx) {
      dlog('viewport-size', { viewportHeightPx: next, prev: viewportHeightPx })
      viewportHeightPx = next
    }
  }

  const updateViewportWidth = () => {
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
    return rows
  })

  // Per-row measured heights (0 = unmeasured, falls back to estimate). Includes the row gap.
  let rowHeights: number[] = []
  // Cumulative offset of row i from the top (rowOffsets[i] = sum of heights for rows 0..i-1).
  let rowOffsets: number[] = []
  let totalHeight = $state(0)
  // Bumped whenever rowOffsets is rewritten so the visible-range derived recomputes.
  let offsetsVersion = $state(0)

  const heightForRow = (i: number): number => {
    const measured = rowHeights[i]
    return (measured && measured > 0) ? measured : estimatedRowHeight
  }

  const rebuildOffsetsFrom = (startIndex: number) => {
    const rowsCount = itemRows.length
    if (rowsCount === 0) {
      rowOffsets = []
      totalHeight = 0
      offsetsVersion++
      dlog('rebuild:empty', { rowsCount })
      return
    }
    let offset = startIndex === 0 ? 0 : (rowOffsets[startIndex - 1] || 0) + heightForRow(startIndex - 1)
    for (let i = startIndex; i < rowsCount; i++) {
      rowOffsets[i] = offset
      offset += heightForRow(i)
    }
    if (rowOffsets.length > rowsCount) {
      rowOffsets.length = rowsCount
    }
    totalHeight = offset
    offsetsVersion++
    dlog('rebuild', {
      startIndex,
      rowsCount,
      totalHeight,
      firstOffsets: rowOffsets.slice(0, 6),
      firstHeights: rowHeights.slice(0, 6),
    })
  }

  // Capture viewport size when the viewport binds (it may mount later if items starts empty).
  $effect(() => {
    if (viewportElement) {
      untrack(updateViewportSize)
    }
  })

  // Reset measured cache when items reference or column count changes — row index → content mapping
  // changes in those cases, so prior measurements are no longer valid. Runs before DOM updates so the
  // first render after a change uses fresh estimates rather than stale measurements.
  $effect.pre(() => {
    void items
    void activeColumns
    untrack(() => {
      rowHeights = new Array(itemRows.length).fill(0)
      rebuildOffsetsFrom(0)
    })
  })

  // After items/columns change and new content renders, force a re-measurement of every mounted
  // row. Elements kept across the change (same rowIndex → same DOM node) don't get their action
  // re-fired, so only ResizeObserver would otherwise catch their new content's height — and RO
  // fires in a later frame, causing a visible flicker between estimated positions and measured
  // positions. Sweeping inside tick() runs after the new DOM is committed but before the next paint,
  // so offsets update in the same event-loop tick.
  $effect(() => {
    void items
    void activeColumns
    let cancelled = false
    void tick().then(() => {
      if (cancelled) { return }
      sweepMountedRows('items-change-sweep')
    })
    return () => { cancelled = true }
  })

  // Binary search: largest index i with rowOffsets[i] <= target. Returns -1 if all > target.
  const bsearchLastLE = (target: number): number => {
    let lo = 0
    let hi = rowOffsets.length - 1
    let result = -1
    while (lo <= hi) {
      const mid = (lo + hi) >>> 1
      if (rowOffsets[mid] <= target) {
        result = mid
        lo = mid + 1
      } else {
        hi = mid - 1
      }
    }
    return result
  }

  const visibleRange = $derived.by(() => {
    void offsetsVersion
    const rowsCount = itemRows.length
    if (rowsCount === 0 || viewportHeightPx <= 0) {
      return { start: 0, end: 0 }
    }
    const firstVisible = Math.max(0, bsearchLastLE(scrollTop))
    const lastVisible = Math.max(firstVisible, bsearchLastLE(scrollTop + viewportHeightPx))
    const start = Math.max(0, firstVisible - bufferSize)
    const end = Math.min(rowsCount, lastVisible + 1 + bufferSize)
    return { start, end }
  })

  const renderedRows = $derived.by(() => {
    void offsetsVersion
    const { start, end } = visibleRange
    const out: Array<{ row: Array<TItem | null>, rowIndex: number, top: number }> = []
    for (let i = start; i < end; i++) {
      out.push({ row: itemRows[i], rowIndex: i, top: rowOffsets[i] || 0 })
    }
    return out
  })

  let scrollLogCount = 0
  const handleScroll = () => {
    if (!viewportElement) { return }
    scrollTop = viewportElement.scrollTop
    // Throttle scroll logs — the first few are enough to diagnose.
    if (scrollLogCount < 6) {
      dlog('scroll', { scrollTop, visibleRange, totalHeight })
      scrollLogCount++
    }
  }

  // Map from observed wrapper element back to its row index, set on `use:` action.
  const elementToRowIndex = new WeakMap<Element, number>()

  const measureRowFromElement = (element: HTMLElement) => {
    // Any measurement activity (change or not) counts as "still settling" for the ready gate.
    pokeReadyQuietTimer()
    const rowIndex = elementToRowIndex.get(element)
    if (rowIndex == null) { dlog('measure:no-index'); return }
    if (rowIndex >= itemRows.length) { dlog('measure:stale-index', { rowIndex }); return }

    // padding-bottom on the row provides the inter-row gap, so it's already part of the measured height.
    const rect = element.getBoundingClientRect()
    const measured = rect.height
    if (!Number.isFinite(measured) || measured <= 0) {
      dlog('measure:invalid', { rowIndex, measured, width: rect.width })
      return
    }

    const oldHeight = heightForRow(rowIndex)
    const hadMeasurement = rowHeights[rowIndex] > 0
    // Skip only if we already had a real measurement and the new one is basically the same. When the
    // row has never been measured we MUST record the value even if it happens to equal the estimate —
    // otherwise future ResizeObserver callbacks with the same value will be filtered out too.
    if (hadMeasurement && Math.abs(oldHeight - measured) < 0.5) {
      dlog('measure:unchanged', { rowIndex, measured, oldHeight })
      return
    }

    const delta = measured - oldHeight
    rowHeights[rowIndex] = measured
    dlog('measure:update', {
      rowIndex,
      oldHeight,
      measured,
      delta,
      hadMeasurement,
      width: rect.width,
      visibleStart: visibleRange.start,
    })
    rebuildOffsetsFrom(rowIndex)

    // Scroll anchoring: when a row above the viewport changes height, shift scrollTop by the same delta
    // so visible content stays put. Without this, items in view would visibly jump.
    if (viewportElement && rowIndex < visibleRange.start) {
      const newScrollTop = Math.max(0, viewportElement.scrollTop + delta)
      dlog('anchor', { rowIndex, delta, newScrollTop, prev: viewportElement.scrollTop })
      viewportElement.scrollTop = newScrollTop
      scrollTop = newScrollTop
    }
  }

  // Create the item observer eagerly so rows mounted on the very first render are observed. If we
  // wait for onMount, the first batch of rows mounts before the observer exists and they never get
  // measured — which is what caused rows to stay stacked at estimated heights until a scroll forced
  // new rows in/out of view.
  if (typeof ResizeObserver !== 'undefined' && !itemResizeObserver) {
    itemResizeObserver = new ResizeObserver((entries) => {
      for (const entry of entries) {
        measureRowFromElement(entry.target as HTMLElement)
      }
    })
  }

  // Re-measure every row currently in the DOM. Used when we suspect earlier measurements were taken
  // before CSS/fonts were stable, or after items change (content at a given row index is different).
  const sweepMountedRows = (reason: string) => {
    if (!viewportElement) { return }
    const rowEls = viewportElement.querySelectorAll<HTMLElement>('.virtual-cards-row')
    dlog('sweep', { reason, count: rowEls.length })
    for (const el of rowEls) {
      measureRowFromElement(el)
    }
  }

  const observeRow = (element: HTMLElement, rowIndex: number) => {
    elementToRowIndex.set(element, rowIndex)
    if (itemResizeObserver) {
      itemResizeObserver.observe(element)
    } else {
      dlog('observe:no-observer', { rowIndex })
    }
    dlog('observe:mount', {
      rowIndex,
      syncRect: {
        height: element.getBoundingClientRect().height,
        width: element.getBoundingClientRect().width,
      },
    })
    // Synchronous first measurement so the initial layout converges before paint.
    measureRowFromElement(element)
    return {
      update(newRowIndex: number) {
        elementToRowIndex.set(element, newRowIndex)
        measureRowFromElement(element)
      },
      destroy() {
        if (itemResizeObserver) {
          itemResizeObserver.unobserve(element)
        }
        elementToRowIndex.delete(element)
      },
    }
  }

  onMount(() => {
    updateViewportWidth()
    updateContainerWidth()
    updateViewportSize()
    dlog('mount', {
      itemsLen: items.length,
      rowsLen: itemRows.length,
      activeColumns,
      estimatedRowHeight,
      viewportHeightPx,
      viewportWidthPx,
      containerWidthPx,
      hasItemObserver: !!itemResizeObserver,
    })

    if (!containerElement) { return }

    window.addEventListener('resize', updateViewportWidth)
    containerResizeObserver = new ResizeObserver(() => {
      updateContainerWidth()
      updateViewportSize()
    })
    containerResizeObserver.observe(containerElement)

    // After first paint, ensure viewport size is captured and sweep every mounted row. The sweep
    // triggers measureRowFromElement → pokeReadyQuietTimer, so the ready gate flips only once
    // measurements stop arriving. The hard cap guarantees content eventually appears even if
    // something is continuously resizing.
    void tick().then(() => {
      updateViewportSize()
      requestAnimationFrame(() => {
        requestAnimationFrame(() => {
          sweepMountedRows('ready-sweep')
          pokeReadyQuietTimer()
        })
      })
    })
    readyMaxTimer = setTimeout(() => {
      readyMaxTimer = null
      flipReady('max-cap')
    }, READY_MAX_MS)

    // Expose a snapshot helper: in the browser console run `window.__vcDump()` to get a frozen
    // picture of the current state for the first mounted instance.
    if (typeof window !== 'undefined') {
      ;(window as unknown as { __vcDump?: () => unknown }).__vcDump = () => {
        const snapshot = {
          instanceTag,
          itemsLen: items.length,
          rowsLen: itemRows.length,
          activeColumns,
          viewportHeightPx,
          viewportWidthPx,
          containerWidthPx,
          scrollTop,
          totalHeight,
          ready,
          visibleRange,
          measuredCount: rowHeights.filter((h) => h > 0).length,
          firstHeights: rowHeights.slice(0, 20),
          firstOffsets: rowOffsets.slice(0, 20),
          viewportScrollTopFromDom: viewportElement?.scrollTop,
          viewportClientHeightFromDom: viewportElement?.clientHeight,
          viewportScrollHeightFromDom: viewportElement?.scrollHeight,
        }
        console.info('[VC] dump', snapshot)
        return snapshot
      }
    }

    return () => {
      window.removeEventListener('resize', updateViewportWidth)
    }
  })

  onDestroy(() => {
    containerResizeObserver?.disconnect()
    itemResizeObserver?.disconnect()
    if (readyQuietTimer !== null) { clearTimeout(readyQuietTimer); readyQuietTimer = null }
    if (readyMaxTimer !== null) { clearTimeout(readyMaxTimer); readyMaxTimer = null }
  })

  const containerCompensationCss = $derived.by(() => {
    return useInnerPadding ? ' virtual-cards-container-with-inner-padding' : ''
  })

  const viewportClass = $derived.by(() => {
    return useInnerPadding
      ? 'virtual-cards-viewport virtual-cards-viewport-with-inner-padding'
      : 'virtual-cards-viewport'
  })
</script>

<div
  bind:this={containerElement}
  class={`virtual-cards-container${containerCompensationCss} ${containerCss}`}
  style={`height:${height}`}
>
  {#if items.length === 0}
    <div class="virtual-cards-empty-message">
      {emptyMessage}
    </div>
  {:else}
    <div
      bind:this={viewportElement}
      class={viewportClass}
      onscroll={handleScroll}
    >
      <div
        class="virtual-cards-content"
        style={`height:${totalHeight}px;visibility:${ready ? 'visible' : 'hidden'}`}
      >
        {#each renderedRows as renderedRow (renderedRow.rowIndex)}
          <div
            class="virtual-cards-row"
            style={`transform:translateY(${renderedRow.top}px);grid-template-columns:repeat(${activeColumns}, minmax(0, 1fr));column-gap:${columnGapPx}px;padding-bottom:${rowGapPx}px`}
            use:observeRow={renderedRow.rowIndex}
          >
            {#each renderedRow.row as item, columnIndex (`${renderedRow.rowIndex}-${columnIndex}`)}
              {#if item}
                <div class="virtual-cards-cell">
                  {@render children?.(item, (renderedRow.rowIndex * activeColumns) + columnIndex)}
                </div>
              {:else}
                <div></div>
              {/if}
            {/each}
          </div>
        {/each}
      </div>
    </div>
  {/if}
</div>

<style>
  .virtual-cards-container {
    min-height: 0;
    overflow: hidden;
    position: relative;
  }

  .virtual-cards-container.virtual-cards-container-with-inner-padding {
    margin-left: -6px;
    margin-right: -6px;
    width: calc(100% + 12px);
  }

  .virtual-cards-viewport {
    position: relative;
    width: 100%;
    height: 100%;
    overflow-y: auto;
    overflow-x: hidden;
    -webkit-overflow-scrolling: touch;
    /* Disable browser scroll anchoring so our explicit anchoring stays authoritative. */
    overflow-anchor: none;
  }

  .virtual-cards-viewport.virtual-cards-viewport-with-inner-padding {
    padding-left: 6px;
    padding-right: 6px;
    padding-top: 2px;
  }

  .virtual-cards-content {
    position: relative;
    width: 100%;
  }

  .virtual-cards-row {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    display: grid;
    width: 100%;
    will-change: transform;
  }

  .virtual-cards-cell {
    width: 100%;
  }

  .virtual-cards-empty-message {
    color: #6b7280;
  }
</style>
