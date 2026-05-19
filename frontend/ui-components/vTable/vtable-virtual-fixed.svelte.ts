// Fixed-height virtualization helper for VTable/<table>-style markup.
//
// Specialization of vtable-virtual.svelte.ts for the case where every row is
// the same known height. The dynamic version needs per-row ResizeObservers, a
// heights[] array, an offsets[] array, and binary search because rows can
// resize at any time (table-layout: auto re-wraps text as columns shift). With
// a known fixed height, all of that collapses to multiplication and division
// against a single `rowHeight` value, and the rendered <tr>s need no
// measurement at all.
//
// API matches createTableVirtualizer so a consumer can swap implementations
// without touching its template; `observeRow` is kept as a no-op action.

export interface FixedTableVirtualizerOptions {
  /** Returns the scroll container. Read once per attach(). */
  getScrollElement: () => HTMLElement | null
  /** Fixed height in px applied to every row. Called per range read so
   *  changing the host's prop is picked up without re-instantiating. */
  rowHeight: () => number
  /** Extra rows rendered above and below the viewport for smoother scrolling. */
  overscan?: () => number
}

export interface FixedTableVirtualRange {
  start: number
  end: number
  /** Absolute Y of the first rendered row — drives the top spacer's height. */
  offsetAtStart: number
  /** Absolute Y of the bottom edge of the last rendered row. */
  offsetAtEnd: number
}

export interface FixedTableVirtualizer {
  readonly totalSize: number
  readonly range: FixedTableVirtualRange
  /** Bind the virtualizer to the scroll element. Returns false if the element isn't ready yet. */
  attach: () => boolean
  detach: () => void
  /** Called by the host when the row count changes. */
  setCount: (count: number) => void
  /** API-parity no-op so templates written against the dynamic virtualizer
   *  (use:virtualizer.observeRow={rowIndex}) work unchanged here. */
  observeRow: (element: HTMLElement, index: number) => {
    update: (newIndex: number) => void
    destroy: () => void
  }
}

export function createFixedTableVirtualizer(options: FixedTableVirtualizerOptions): FixedTableVirtualizer {
  const readOverscan = () => options.overscan?.() ?? 6
  const readRowHeight = () => options.rowHeight()

  let scrollElement: HTMLElement | null = null
  let scrollTop = $state(0)
  let viewportHeight = $state(0)
  let rowCount = $state(0)

  const totalSize = $derived(rowCount * readRowHeight())

  const range = $derived.by<FixedTableVirtualRange>(() => {
    const rowHeight = readRowHeight()
    if (rowCount === 0 || viewportHeight <= 0 || rowHeight <= 0) {
      return { start: 0, end: 0, offsetAtStart: 0, offsetAtEnd: 0 }
    }
    // Clamp firstVisible so an over-scroll past the dataset doesn't produce
    // an invalid start > end interval before the overscan widens it.
    const firstVisible = Math.min(
      rowCount - 1,
      Math.max(0, Math.floor(scrollTop / rowHeight)),
    )
    // Strict-less analogue of the dynamic version's findLastOffsetAtOrBelow
    // with `exclusive=true`: largest i where i * rowHeight < scrollTop+viewport.
    const lastVisibleRaw = Math.ceil((scrollTop + viewportHeight) / rowHeight) - 1
    const lastVisible = Math.max(firstVisible, Math.min(rowCount - 1, lastVisibleRaw))
    const overscan = readOverscan()
    const start = Math.max(0, firstVisible - overscan)
    const end = Math.min(rowCount, lastVisible + 1 + overscan)
    return {
      start,
      end,
      offsetAtStart: start * rowHeight,
      offsetAtEnd: end * rowHeight,
    }
  })

  const setCount = (next: number) => {
    if (next === rowCount) { return }
    rowCount = next
  }

  // No measurement to do — kept so templates using `use:virtualizer.observeRow`
  // can switch between dynamic and fixed implementations without edits.
  const observeRow = (_element: HTMLElement, _index: number) => {
    return {
      update(_newIndex: number) { },
      destroy() { },
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
    return true
  }

  const detach = () => {
    if (scrollElement && scrollListener) {
      scrollElement.removeEventListener('scroll', scrollListener)
    }
    if (scrollRafId !== null) { cancelAnimationFrame(scrollRafId); scrollRafId = null }
    containerObserver?.disconnect(); containerObserver = null
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
