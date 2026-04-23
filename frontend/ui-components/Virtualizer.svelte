<script lang="ts" generics="TItem">
  import { onDestroy, onMount, tick, untrack, type Snippet } from 'svelte'

  interface VirtualizerProps<TItem> {
    items: TItem[]
    /** CSS height applied to the outer container. */
    height?: string
    /** Height used for items that haven't been measured yet. Closer to reality = less churn. */
    estimatedItemHeight?: number
    /** Extra items rendered above and below the viewport for smooth scrolling. */
    bufferSize?: number
    containerCss?: string
    viewportCss?: string
    emptyMessage?: string
    /** Enable console debug logs. Logs are prefixed `[Virtualizer:<id>]`. */
    debug?: boolean
    children?: Snippet<[TItem, number]>
  }

  let {
    items = [],
    height = '100%',
    estimatedItemHeight = 80,
    bufferSize = 6,
    containerCss = '',
    viewportCss = '',
    emptyMessage = '',
    debug = false,
    children,
  }: VirtualizerProps<TItem> = $props()

  const instanceTag = Math.random().toString(36).slice(2, 6)
  const dlog = (tag: string, payload?: unknown) => {
    if (!debug) { return }
    try {
      console.info(`[Virtualizer:${instanceTag}] ${tag}`, payload ?? '')
    } catch {
      // ignore
    }
  }

  let containerElement = $state<HTMLDivElement | undefined>(undefined)
  let viewportElement = $state<HTMLDivElement | undefined>(undefined)
  let viewportHeightPx = $state(0)
  let scrollTop = $state(0)
  // Hide content until initial measurements settle — otherwise the user briefly sees items at
  // estimated positions before real heights arrive.
  let ready = $state(false)
  let containerResizeObserver: ResizeObserver | undefined
  let itemResizeObserver: ResizeObserver | undefined

  // Debounced "ready" gate: any measurement activity resets the quiet timer; when nothing arrives
  // for READY_QUIET_MS, the content is revealed. READY_MAX_MS caps the wait so slow-loading fonts
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
    dlog('ready', { reason })
  }

  const pokeReadyQuietTimer = () => {
    if (ready) { return }
    if (readyQuietTimer !== null) { clearTimeout(readyQuietTimer) }
    readyQuietTimer = setTimeout(() => {
      readyQuietTimer = null
      flipReady('quiet')
    }, READY_QUIET_MS)
  }

  const updateViewportSize = () => {
    if (!viewportElement) { return }
    const next = Math.max(0, Math.floor(viewportElement.clientHeight))
    if (next !== viewportHeightPx) { viewportHeightPx = next }
  }

  // Per-item measured heights (0 = unmeasured, falls back to estimate).
  let itemHeights: number[] = []
  // Cumulative offset of item i from the top.
  let itemOffsets: number[] = []
  let totalHeight = $state(0)
  // Bumped whenever offsets are rewritten so derived values recompute.
  let offsetsVersion = $state(0)

  const heightForItem = (i: number): number => {
    const measured = itemHeights[i]
    return (measured && measured > 0) ? measured : estimatedItemHeight
  }

  const rebuildOffsetsFrom = (startIndex: number) => {
    const itemsCount = items.length
    if (itemsCount === 0) {
      itemOffsets = []
      totalHeight = 0
      offsetsVersion++
      return
    }
    let offset = startIndex === 0 ? 0 : (itemOffsets[startIndex - 1] || 0) + heightForItem(startIndex - 1)
    for (let i = startIndex; i < itemsCount; i++) {
      itemOffsets[i] = offset
      offset += heightForItem(i)
    }
    if (itemOffsets.length > itemsCount) {
      itemOffsets.length = itemsCount
    }
    totalHeight = offset
    offsetsVersion++
  }

  // Reset measurements when the items array reference changes — the item at a given index may be
  // different content, so prior heights are not reliable. Runs before DOM updates so the first
  // render uses fresh estimates.
  $effect.pre(() => {
    void items
    untrack(() => {
      itemHeights = new Array(items.length).fill(0)
      rebuildOffsetsFrom(0)
    })
  })

  // After items change, re-measure every mounted item. Elements kept across the change (same index
  // → same DOM node) don't get their action re-fired, so only ResizeObserver would otherwise catch
  // their new content's height — and RO fires in a later frame, causing a visible flicker between
  // estimated positions and measured positions. Sweeping inside tick() runs after the new DOM is
  // committed but before the next paint, so offsets update in the same event-loop tick.
  $effect(() => {
    void items
    let cancelled = false
    void tick().then(() => {
      if (cancelled) { return }
      sweepMountedItems('items-change-sweep')
    })
    return () => { cancelled = true }
  })

  $effect(() => {
    if (viewportElement) { untrack(updateViewportSize) }
  })

  // Binary search: largest index i with itemOffsets[i] <= target. Returns -1 if all > target.
  const bsearchLastLE = (target: number): number => {
    let lo = 0
    let hi = itemOffsets.length - 1
    let result = -1
    while (lo <= hi) {
      const mid = (lo + hi) >>> 1
      if (itemOffsets[mid] <= target) {
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
    const itemsCount = items.length
    if (itemsCount === 0 || viewportHeightPx <= 0) {
      return { start: 0, end: 0 }
    }
    const firstVisible = Math.max(0, bsearchLastLE(scrollTop))
    const lastVisible = Math.max(firstVisible, bsearchLastLE(scrollTop + viewportHeightPx))
    const start = Math.max(0, firstVisible - bufferSize)
    const end = Math.min(itemsCount, lastVisible + 1 + bufferSize)
    return { start, end }
  })

  const renderedItems = $derived.by(() => {
    void offsetsVersion
    const { start, end } = visibleRange
    const out: Array<{ item: TItem, index: number, top: number }> = []
    for (let i = start; i < end; i++) {
      out.push({ item: items[i], index: i, top: itemOffsets[i] || 0 })
    }
    return out
  })

  let scrollLogCount = 0
  const handleScroll = () => {
    if (!viewportElement) { return }
    scrollTop = viewportElement.scrollTop
    if (debug && scrollLogCount < 6) {
      dlog('scroll', { scrollTop, visibleRange, totalHeight })
      scrollLogCount++
    }
  }

  // Map observed wrapper element → current item index, set by the `use:` action.
  const elementToIndex = new WeakMap<Element, number>()

  const measureItemFromElement = (element: HTMLElement) => {
    // Any measurement activity (change or not) counts as "still settling" for the ready gate.
    pokeReadyQuietTimer()
    const index = elementToIndex.get(element)
    if (index == null) { return }
    if (index >= items.length) { return }

    const measured = element.getBoundingClientRect().height
    if (!Number.isFinite(measured) || measured <= 0) { return }

    const oldHeight = heightForItem(index)
    const hadMeasurement = itemHeights[index] > 0
    // Skip only if we already had a real measurement and the new one is basically the same. A first
    // measurement that happens to equal the estimate must still be recorded, otherwise future
    // ResizeObserver callbacks with the same value get filtered out too.
    if (hadMeasurement && Math.abs(oldHeight - measured) < 0.5) { return }

    const delta = measured - oldHeight
    itemHeights[index] = measured
    dlog('measure:update', { index, oldHeight, measured, delta, hadMeasurement })
    rebuildOffsetsFrom(index)

    // Scroll anchoring: when an item above the viewport changes height, shift scrollTop by the same
    // delta so visible content stays put.
    if (viewportElement && index < visibleRange.start) {
      const newScrollTop = Math.max(0, viewportElement.scrollTop + delta)
      viewportElement.scrollTop = newScrollTop
      scrollTop = newScrollTop
    }
  }

  // Re-measure every item currently in the DOM. Used when we suspect earlier measurements were
  // taken before CSS/fonts stabilised, or after items change (content at a given index is different).
  const sweepMountedItems = (reason: string) => {
    if (!viewportElement) { return }
    const els = viewportElement.querySelectorAll<HTMLElement>('.virtualizer-item')
    dlog('sweep', { reason, count: els.length })
    for (const el of els) { measureItemFromElement(el) }
  }

  // Create the item observer eagerly so items mounted on the very first render are observed.
  if (typeof ResizeObserver !== 'undefined' && !itemResizeObserver) {
    itemResizeObserver = new ResizeObserver((entries) => {
      for (const entry of entries) {
        measureItemFromElement(entry.target as HTMLElement)
      }
    })
  }

  const observeItem = (element: HTMLElement, index: number) => {
    elementToIndex.set(element, index)
    itemResizeObserver?.observe(element)
    measureItemFromElement(element)
    return {
      update(newIndex: number) {
        elementToIndex.set(element, newIndex)
        measureItemFromElement(element)
      },
      destroy() {
        itemResizeObserver?.unobserve(element)
        elementToIndex.delete(element)
      },
    }
  }

  onMount(() => {
    updateViewportSize()
    if (!containerElement) { return }

    containerResizeObserver = new ResizeObserver(() => { updateViewportSize() })
    containerResizeObserver.observe(containerElement)

    // After first paint, ensure viewport size is captured and sweep every mounted item. Sweep
    // triggers measureItemFromElement → pokeReadyQuietTimer, so ready flips only once measurements
    // stop arriving.
    void tick().then(() => {
      updateViewportSize()
      requestAnimationFrame(() => {
        requestAnimationFrame(() => {
          sweepMountedItems('ready-sweep')
          pokeReadyQuietTimer()
        })
      })
    })
    readyMaxTimer = setTimeout(() => {
      readyMaxTimer = null
      flipReady('max-cap')
    }, READY_MAX_MS)

    dlog('mount', {
      itemsLen: items.length,
      estimatedItemHeight,
      viewportHeightPx,
      hasItemObserver: !!itemResizeObserver,
    })
  })

  onDestroy(() => {
    containerResizeObserver?.disconnect()
    itemResizeObserver?.disconnect()
    if (readyQuietTimer !== null) { clearTimeout(readyQuietTimer); readyQuietTimer = null }
    if (readyMaxTimer !== null) { clearTimeout(readyMaxTimer); readyMaxTimer = null }
  })

  /** Scrolls to a specific item index, aligning it to the top of the viewport. */
  export const scrollToIndex = (index: number) => {
    if (!viewportElement) { return }
    const clamped = Math.max(0, Math.min(items.length - 1, Math.floor(index)))
    const target = itemOffsets[clamped] ?? clamped * estimatedItemHeight
    viewportElement.scrollTop = target
    scrollTop = target
  }
</script>

<div
  bind:this={containerElement}
  class={`virtualizer-container ${containerCss}`}
  style={`height:${height}`}
>
  {#if items.length === 0}
    {#if emptyMessage}
      <div class="virtualizer-empty-message">{emptyMessage}</div>
    {/if}
  {:else}
    <div
      bind:this={viewportElement}
      class={`virtualizer-viewport ${viewportCss}`}
      onscroll={handleScroll}
    >
      <div
        class="virtualizer-content"
        style={`height:${totalHeight}px;visibility:${ready ? 'visible' : 'hidden'}`}
      >
        {#each renderedItems as rendered (rendered.index)}
          <div
            class="virtualizer-item"
            style={`transform:translateY(${rendered.top}px)`}
            use:observeItem={rendered.index}
          >
            {@render children?.(rendered.item, rendered.index)}
          </div>
        {/each}
      </div>
    </div>
  {/if}
</div>

<style>
  .virtualizer-container {
    min-height: 0;
    overflow: hidden;
    position: relative;
    width: 100%;
  }

  .virtualizer-viewport {
    position: relative;
    width: 100%;
    height: 100%;
    overflow-y: auto;
    overflow-x: hidden;
    -webkit-overflow-scrolling: touch;
    /* Browser scroll anchoring conflicts with our explicit anchoring; disable it. */
    overflow-anchor: none;
  }

  .virtualizer-content {
    position: relative;
    width: 100%;
  }

  .virtualizer-item {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    will-change: transform;
  }

  .virtualizer-empty-message {
    color: #6b7280;
  }
</style>
