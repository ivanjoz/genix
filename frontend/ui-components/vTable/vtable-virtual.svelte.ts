// Virtualization helper tailored for VTable's <table>/<tr> markup.
//
// Maintains per-row measured heights via ResizeObserver, cumulative offsets,
// and derives the visible window with binary search. Consumers size top/bottom
// spacer <tr>s from `range.offsetAtStart` and `totalSize - range.offsetAtEnd`,
// and call `observeRow` as a Svelte action on each rendered <tr>. This avoids
// the absolute-positioned wrapper pattern (which breaks column alignment in a
// real <table>) used by ui-components/misc/Virtualizer.svelte.

export interface TableVirtualizerOptions {
  /** Returns the scroll container. Read once per attach(). */
  getScrollElement: () => HTMLElement | null
  /** Placeholder size for rows that haven't been measured yet. Called per read
   *  so changing the host's prop is picked up without re-instantiating. */
  estimateSize: () => number
  /** Extra rows rendered above and below the viewport for smoother scrolling. */
  overscan?: () => number
}

export interface TableVirtualRange {
  start: number
  end: number
  /** Absolute Y of the first rendered row — drives the top spacer's height. */
  offsetAtStart: number
  /** Absolute Y of the bottom edge of the last rendered row. */
  offsetAtEnd: number
}

export interface TableVirtualizer {
  readonly totalSize: number
  readonly range: TableVirtualRange
  /** Bind the virtualizer to the scroll element. Returns false if the element isn't ready yet. */
  attach: () => boolean
  detach: () => void
  /** Called by the host when the row count changes; resets measurements for the new dataset. */
  setCount: (count: number) => void
  /** Svelte action: registers a <tr> for ResizeObserver-based measurement. */
  observeRow: (element: HTMLElement, index: number) => {
    update: (newIndex: number) => void
    destroy: () => void
  }
}

export function createTableVirtualizer(options: TableVirtualizerOptions): TableVirtualizer {
  const readOverscan = () => options.overscan?.() ?? 6
  const readEstimateSize = () => options.estimateSize()

  let scrollElement: HTMLElement | null = null
  let scrollTop = $state(0)
  let viewportHeight = $state(0)
  // 0 means "unmeasured — fall back to estimate".
  let heights: number[] = []
  // Absolute Y of each row's top edge; offsets[i] + heightAt(i) = offsets[i+1].
  let offsets: number[] = []
  let totalSize = $state(0)
  // Bumped after every offset rewrite so the derived range recomputes.
  let offsetsVersion = $state(0)
  let rowCount = $state(0)

  const heightAt = (i: number) => {
    const measured = heights[i]
    return (measured && measured > 0) ? measured : readEstimateSize()
  }

  // Recompute offsets from `startIndex` to the end. Cheap because the prefix
  // below `startIndex` is unaffected when a single row resizes.
  const rebuildOffsetsFrom = (startIndex: number) => {
    if (rowCount === 0) {
      offsets = []
      totalSize = 0
      offsetsVersion++
      return
    }
    let cursor = startIndex === 0
      ? 0
      : (offsets[startIndex - 1] || 0) + heightAt(startIndex - 1)
    for (let i = startIndex; i < rowCount; i++) {
      offsets[i] = cursor
      cursor += heightAt(i)
    }
    if (offsets.length > rowCount) { offsets.length = rowCount }
    totalSize = cursor
    offsetsVersion++
  }

  // Called by VTable whenever the filtered data length changes. Stored heights
  // are dropped because the row at index `i` may now be a different record.
  const setCount = (next: number) => {
    if (next === rowCount && heights.length === next) { return }
    rowCount = next
    heights = new Array(next).fill(0)
    rebuildOffsetsFrom(0)
  }

  // Largest index `i` with offsets[i] <= target (or strict < target when `exclusive`).
  // Returns -1 if none exists.
  const findLastOffsetAtOrBelow = (target: number, exclusive = false): number => {
    let lo = 0
    let hi = offsets.length - 1
    let result = -1
    while (lo <= hi) {
      const mid = (lo + hi) >>> 1
      const ok = exclusive ? offsets[mid] < target : offsets[mid] <= target
      if (ok) { result = mid; lo = mid + 1 }
      else { hi = mid - 1 }
    }
    return result
  }

  const range = $derived.by<TableVirtualRange>(() => {
    void offsetsVersion
    if (rowCount === 0 || viewportHeight <= 0) {
      return { start: 0, end: 0, offsetAtStart: 0, offsetAtEnd: 0 }
    }
    const firstVisible = Math.max(0, findLastOffsetAtOrBelow(scrollTop))
    // Use strict `<` for the bottom edge so a row whose top sits exactly at the
    // viewport bottom — i.e. fully below the visible area — isn't pulled into
    // the rendered window.
    const lastVisible = Math.max(firstVisible, findLastOffsetAtOrBelow(scrollTop + viewportHeight, true))
    const overscan = readOverscan()
    const start = Math.max(0, firstVisible - overscan)
    const end = Math.min(rowCount, lastVisible + 1 + overscan)
    const offsetAtStart = offsets[start] || 0
    const lastRendered = end - 1
    const offsetAtEnd = lastRendered >= 0
      ? (offsets[lastRendered] || 0) + heightAt(lastRendered)
      : 0
    return { start, end, offsetAtStart, offsetAtEnd }
  })

  // ResizeObserver wiring -----------------------------------------------------

  // Maps a row's DOM element → its current row index. Updated by the action's
  // `update(newIndex)` callback when keyed each-blocks reuse a node for a
  // different row (e.g. during fast scroll).
  const elementToIndex = new WeakMap<Element, number>()
  let rowResizeObserver: ResizeObserver | null = null

  // Read the row's real rendered height and patch offsets — but ONLY take the
  // first real measurement per (index, element) pair.
  //
  // Why we lock heights after the first measurement: with table-layout: auto,
  // the set of currently-rendered rows determines column widths, so adding or
  // removing one row from the visible window re-wraps text in every other row
  // and changes their heights. If ResizeObserver kept feeding those changes
  // back into our offsets, we'd loop forever (rebuildOffsetsFrom → range
  // re-derives → row mounts/unmounts → columns shift → RO fires → repeat).
  // The first measurement after mount is "close enough" to the steady-state
  // height; consumers needing a fresh measurement should call rerenderRow(i)
  // which triggers an unmount/remount via the each-block key.
  const measureRow = (element: HTMLElement) => {
    const index = elementToIndex.get(element)
    if (index == null || index >= rowCount) { return }
    if (heights[index] > 0) { return }
    const measured = element.getBoundingClientRect().height
    if (!Number.isFinite(measured) || measured <= 0) { return }
    heights[index] = measured
    rebuildOffsetsFrom(index)
  }

  const observeRow = (element: HTMLElement, index: number) => {
    elementToIndex.set(element, index)
    // Observe so the first non-zero rect (which may not arrive until children
    // mount on the next layout pass) gets captured even if action init ran
    // before the row had real content. ResizeObserver fires an initial
    // callback for newly-observed elements at the next "deliver resize
    // observations" step (before paint), so a synchronous read here is
    // redundant — and worse, it can sample a pre-layout 0-height (the <tr>
    // exists but its <td> children may not be laid out yet) which still gets
    // skipped by the `measured <= 0` guard, leaving the offsets briefly using
    // estimateSize. Letting RO own the first read keeps measurement on a
    // single, post-layout path.
    rowResizeObserver?.observe(element)
    return {
      update(newIndex: number) {
        // The keyed each-block typically destroys + recreates rows when keys
        // change, but if it ever reuses a node for a new index, clear the old
        // measurement so the new content gets sampled once.
        if (newIndex !== elementToIndex.get(element)) {
          heights[newIndex] = 0
        }
        elementToIndex.set(element, newIndex)
        measureRow(element)
      },
      destroy() {
        rowResizeObserver?.unobserve(element)
        elementToIndex.delete(element)
      },
    }
  }

  // Lifecycle -----------------------------------------------------------------

  let scrollListener: (() => void) | null = null
  let scrollRafId: number | null = null
  let containerObserver: ResizeObserver | null = null

  const attach = () => {
    scrollElement = options.getScrollElement()
    if (!scrollElement) { return false }

    viewportHeight = scrollElement.clientHeight
    scrollTop = scrollElement.scrollTop

    // Coalesce native scroll events to one read per animation frame.
    scrollListener = () => {
      if (scrollRafId !== null) { cancelAnimationFrame(scrollRafId) }
      scrollRafId = requestAnimationFrame(() => {
        scrollRafId = null
        if (!scrollElement) { return }
        scrollTop = scrollElement.scrollTop
      })
    }
    scrollElement.addEventListener('scroll', scrollListener, { passive: true })

    containerObserver = new ResizeObserver(() => {
      if (!scrollElement) { return }
      viewportHeight = scrollElement.clientHeight
    })
    containerObserver.observe(scrollElement)

    if (typeof ResizeObserver !== 'undefined') {
      // Batch a burst of row resizes (one initial-delivery callback often
      // covers every newly-mounted overscan row) into a single rebuild from
      // the smallest changed index, instead of N separate offset rebuilds and
      // N range re-derivations in the same tick.
      rowResizeObserver = new ResizeObserver((entries) => {
        let minChangedIndex = Infinity
        for (const entry of entries) {
          const element = entry.target as HTMLElement
          const index = elementToIndex.get(element)
          if (index == null || index >= rowCount) { continue }
          // First-measurement-only: heights[index] > 0 means we've already
          // locked this row's height to avoid auto column-width feedback loops.
          if (heights[index] > 0) { continue }
          const measured = element.getBoundingClientRect().height
          if (!Number.isFinite(measured) || measured <= 0) { continue }
          heights[index] = measured
          if (index < minChangedIndex) { minChangedIndex = index }
        }
        if (minChangedIndex !== Infinity) { rebuildOffsetsFrom(minChangedIndex) }
      })
    }
    return true
  }

  const detach = () => {
    if (scrollElement && scrollListener) {
      scrollElement.removeEventListener('scroll', scrollListener)
    }
    if (scrollRafId !== null) { cancelAnimationFrame(scrollRafId); scrollRafId = null }
    containerObserver?.disconnect(); containerObserver = null
    rowResizeObserver?.disconnect(); rowResizeObserver = null
    scrollElement = null
    scrollListener = null
  }

  return {
    get totalSize() { return totalSize },
    get range() { return range },
    attach,
    detach,
    setCount,
    observeRow,
  }
}
