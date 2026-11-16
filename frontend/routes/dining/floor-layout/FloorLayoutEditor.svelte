<script lang="ts">
  import { tick } from 'svelte'
  import { tr } from '#core/store.svelte.ts'
  import LayoutCanvas from '../LayoutCanvas.svelte'
  import { isLineShape, shapeAt, shapeBounds, type IGridPoint, type IGridRect } from '../dining.ts'
  import {
    GRID_MARGIN, LAYOUT_TOOL_DRAG_TYPE, cellAt, editorCanvasRegion, makeShape, moveLineEnd, moveShapes, normalizeRect,
    overlappingTableIDs, resizeShapeTo, shapeIDsInRect, type LayoutTool,
  } from './floor-layout.ts'
  import type { FloorLayoutEditorState } from './floor-layout.svelte.ts'

  // The drawing surface of the floor-layout editor: pointer gestures and keyboard shortcuts over
  // LayoutCanvas, and the drop of a palette element dragged from FloorLayoutTools. The geometry is in
  // floor-layout.ts; this only routes the gestures to it. The plan is a window over an open sheet:
  // the viewport's size, slid or enlarged only as the shapes need (editorCanvasRegion).
  let { editor }: { editor: FloorLayoutEditorState } = $props()

  const HANDLE_HIT_DISTANCE = 0.6 // cells
  // A drag held this close to the viewport's edge scrolls the plan one cell per tick, so a shape can
  // be carried past the visible area.
  const EDGE_SCROLL_ZONE_PX = 24
  const EDGE_SCROLL_TICK_MS = 60

  // The gesture in progress, from pointerdown to pointerup. Shape drags follow the pointer's travel on
  // screen (plus any scrolling), not its cell on the plan: the window slides as the shapes move, so the
  // cell under a still pointer changes, and following it would feed the drag back into itself.
  type ScreenStart = { startClient: IGridPoint, startScroll: IGridPoint }
  type Drag =
    | ScreenStart & { kind: 'move', startCorner: IGridPoint }
    | ScreenStart & { kind: 'resize', shapeID: number, startCorner: IGridPoint }
    | ScreenStart & { kind: 'line-end', shapeID: number, end: 1 | 2, startEnd: IGridPoint }
    | { kind: 'band', origin: IGridPoint, current: IGridPoint }
  let drag = $state<Drag>()
  const isDragging = $derived(!!drag)

  let planViewport = $state<HTMLDivElement>()
  // The content box, inside the scrollbars: it shrinks when one appears, so the plan never overflows
  // into a second scrollbar just to fill the space the first one took.
  let viewportContentRect = $state<DOMRectReadOnly>()
  let lastPointerClient = { x: 0, y: 0 }

  // The window of the sheet drawn: the viewport measured in cells, slid to hold every shape with its
  // margin, larger only when they do not fit.
  const canvasRegion = $derived(editorCanvasRegion(editor.draft?.Shapes || [], editor.viewAnchor,
    Math.floor((viewportContentRect?.width || 0) / editor.cellSize), Math.floor((viewportContentRect?.height || 0) / editor.cellSize)))

  const overlapIDs = $derived(overlappingTableIDs(editor.draft?.Shapes || []))
  const singleSelected = $derived(editor.selectedShapeIDs.length === 1 ? editor.selectedShapes()[0] : undefined)

  // The top-left corner of the selection as a whole.
  function selectionCorner(): IGridPoint {
    const bounds = editor.selectedShapes().map(shapeBounds)
    return { x: Math.min(...bounds.map((rect) => rect.x1)), y: Math.min(...bounds.map((rect) => rect.y1)) }
  }

  function screenStart(event: PointerEvent): ScreenStart {
    return { startClient: { x: event.clientX, y: event.clientY }, startScroll: { x: planViewport?.scrollLeft || 0, y: planViewport?.scrollTop || 0 } }
  }

  // The whole cells the pointer has travelled since the drag began, scrolling included.
  function pointerTravelCells(start: ScreenStart): IGridPoint {
    const cell = editor.cellSize
    return {
      x: Math.round((lastPointerClient.x - start.startClient.x + (planViewport?.scrollLeft || 0) - start.startScroll.x) / cell),
      y: Math.round((lastPointerClient.y - start.startClient.y + (planViewport?.scrollTop || 0) - start.startScroll.y) / cell),
    }
  }

  // A point on screen in the sheet's cells, fractional.
  function sheetPointAt(clientX: number, clientY: number): IGridPoint | undefined {
    const planBox = planViewport?.querySelector('svg')?.getBoundingClientRect()
    if (!planBox) return undefined
    return { x: (clientX - planBox.left) / editor.cellSize + canvasRegion.x1, y: (clientY - planBox.top) / editor.cellSize + canvasRegion.y1 }
  }

  // The handle under the point, for a single selected shape: a block's bottom-right corner, a line's ends.
  function handleAt(point: IGridPoint, start: ScreenStart): Drag | undefined {
    const shape = singleSelected
    if (!shape) return undefined
    const isNear = (x: number, y: number) => Math.hypot(point.x - x, point.y - y) <= HANDLE_HIT_DISTANCE
    if (isLineShape(shape)) {
      const [x1, y1, x2, y2] = shape.Points
      if (isNear(x1, y1)) return { ...start, kind: 'line-end', shapeID: shape.ShapeID, end: 1, startEnd: { x: x1, y: y1 } }
      if (isNear(x2, y2)) return { ...start, kind: 'line-end', shapeID: shape.ShapeID, end: 2, startEnd: { x: x2, y: y2 } }
      return undefined
    }
    const bounds = shapeBounds(shape)
    return isNear(bounds.x2, bounds.y2) ? { ...start, kind: 'resize', shapeID: shape.ShapeID, startCorner: { x: bounds.x2, y: bounds.y2 } } : undefined
  }

  // A palette element dropped on the plan: its top-left goes on the cell under the pointer (a wall
  // starts there), and it comes out selected so its row can be edited.
  function onPaletteDrop(event: DragEvent) {
    const draft = editor.draft
    const tool = event.dataTransfer?.getData(LAYOUT_TOOL_DRAG_TYPE) as LayoutTool | undefined
    const point = sheetPointAt(event.clientX, event.clientY)
    if (!draft || !tool || !point) return
    event.preventDefault()
    const shape = makeShape(tool, draft, cellAt(point))
    draft.Shapes = [...(draft.Shapes || []), shape]
    editor.selectedShapeIDs = [shape.ShapeID]
    editor.checkpoint()
  }

  // Only a palette element may be dropped: a file or a text dragged over the plan is refused.
  function onPaletteDragOver(event: DragEvent) {
    if (!event.dataTransfer?.types.includes(LAYOUT_TOOL_DRAG_TYPE)) return
    event.preventDefault()
    event.dataTransfer.dropEffect = 'copy'
  }

  function onPointerDown(event: PointerEvent, point: IGridPoint) {
    if (!editor.draft || event.button !== 0) return
    ;(event.currentTarget as Element).setPointerCapture(event.pointerId)
    lastPointerClient = { x: event.clientX, y: event.clientY }
    const handleDrag = handleAt(point, screenStart(event))
    if (handleDrag) { drag = handleDrag; return }
    const shape = shapeAt(editor.draft.Shapes || [], point)
    if (!shape) {
      if (!event.shiftKey) editor.selectedShapeIDs = []
      drag = { kind: 'band', origin: point, current: point }
      return
    }
    if (event.shiftKey) {
      editor.selectedShapeIDs = editor.selectedShapeIDs.includes(shape.ShapeID)
        ? editor.selectedShapeIDs.filter((shapeID) => shapeID !== shape.ShapeID)
        : [...editor.selectedShapeIDs, shape.ShapeID]
      return
    }
    if (!editor.selectedShapeIDs.includes(shape.ShapeID)) editor.selectedShapeIDs = [shape.ShapeID]
    drag = { ...screenStart(event), kind: 'move', startCorner: selectionCorner() }
  }

  // Applies the drag for the pointer where it is now. A shape pushed against an edge stays there while
  // the window slides (the other shapes move away from it), and grows once the shapes no longer fit.
  function applyDrag() {
    const draft = editor.draft
    if (!drag || !draft) return
    if (drag.kind === 'band') {
      drag.current = sheetPointAt(lastPointerClient.x, lastPointerClient.y) || drag.current
      return
    }
    const travel = pointerTravelCells(drag)
    if (drag.kind === 'move') {
      const corner = selectionCorner()
      moveShapes(draft, editor.selectedShapeIDs, { x: drag.startCorner.x + travel.x - corner.x, y: drag.startCorner.y + travel.y - corner.y })
      return
    }
    const shapeID = drag.shapeID
    const shape = draft.Shapes?.find((candidate) => candidate.ShapeID === shapeID)
    if (!shape) return
    if (drag.kind === 'resize') resizeShapeTo(shape, { x: drag.startCorner.x + travel.x, y: drag.startCorner.y + travel.y })
    else moveLineEnd(shape, drag.end, { x: drag.startEnd.x + travel.x, y: drag.startEnd.y + travel.y })
  }

  function onPointerMove(event: PointerEvent) {
    lastPointerClient = { x: event.clientX, y: event.clientY }
    applyDrag()
  }

  // While a drag is held at the viewport's edge, the plan scrolls a cell and the drag carries on by
  // that cell, where the plan grows again.
  function scrollAtViewportEdge() {
    const viewport = planViewport
    if (!drag || !viewport) return
    const viewportBox = viewport.getBoundingClientRect()
    const edgeDirection = (position: number, low: number, high: number) =>
      position > high - EDGE_SCROLL_ZONE_PX ? 1 : position < low + EDGE_SCROLL_ZONE_PX ? -1 : 0
    const direction = {
      x: edgeDirection(lastPointerClient.x, viewportBox.left, viewportBox.right),
      y: edgeDirection(lastPointerClient.y, viewportBox.top, viewportBox.bottom),
    }
    if (!direction.x && !direction.y) return
    viewport.scrollBy(direction.x * editor.cellSize, direction.y * editor.cellSize)
    applyDrag()
  }

  $effect(() => {
    if (!isDragging) return
    const edgeScrollTimer = setInterval(scrollAtViewportEdge, EDGE_SCROLL_TICK_MS)
    return () => clearInterval(edgeScrollTimer)
  })

  async function onPointerUp() {
    const draft = editor.draft
    const finishedDrag = drag
    drag = undefined
    if (!finishedDrag || !draft) return
    if (finishedDrag.kind === 'band') {
      const bandIDs = shapeIDsInRect(draft.Shapes || [], normalizeRect(finishedDrag.origin, finishedDrag.current))
      editor.selectedShapeIDs = [...new Set([...editor.selectedShapeIDs, ...bandIDs])]
      return
    }
    editor.checkpoint()
    await tick()
    revealSelection()
  }

  // A shape carried past the viewport's edge follows the pointer there, half hidden: once dropped, the
  // plan scrolls just enough to show it whole, with its margin cell.
  function revealSelection() {
    const viewport = planViewport
    if (!viewport || !editor.selectedShapeIDs.length) return
    const cell = editor.cellSize
    const bounds = editor.selectedShapes().map(shapeBounds)
    // In pixels from the window's top-left.
    const left = (Math.min(...bounds.map((rect) => rect.x1)) - GRID_MARGIN - canvasRegion.x1) * cell
    const top = (Math.min(...bounds.map((rect) => rect.y1)) - GRID_MARGIN - canvasRegion.y1) * cell
    const right = (Math.max(...bounds.map((rect) => rect.x2)) + GRID_MARGIN - canvasRegion.x1) * cell
    const bottom = (Math.max(...bounds.map((rect) => rect.y2)) + GRID_MARGIN - canvasRegion.y1) * cell
    viewport.scrollLeft = Math.min(left, Math.max(viewport.scrollLeft, right - viewport.clientWidth))
    viewport.scrollTop = Math.min(top, Math.max(viewport.scrollTop, bottom - viewport.clientHeight))
  }

  const ARROW_STEPS: Record<string, IGridPoint> = {
    ArrowLeft: { x: -1, y: 0 }, ArrowRight: { x: 1, y: 0 }, ArrowUp: { x: 0, y: -1 }, ArrowDown: { x: 0, y: 1 },
  }

  // Shortcuts act on the plan only while no form field has the focus.
  function onKeyDown(event: KeyboardEvent) {
    const target = event.target as HTMLElement
    if (!editor.draft || target.closest('input, textarea, select, [contenteditable="true"]')) return
    const isCommand = event.ctrlKey || event.metaKey
    const key = event.key.toLowerCase()
    if (isCommand && key === 'z') { event.preventDefault(); if (event.shiftKey) editor.redo(); else editor.undo(); return }
    if (isCommand && key === 'y') { event.preventDefault(); editor.redo(); return }
    if (isCommand && key === 'd') { event.preventDefault(); editor.duplicateSelection(); return }
    if (event.key === 'Delete' || event.key === 'Backspace') { if (editor.selectedShapeIDs.length) { event.preventDefault(); editor.deleteSelection() }; return }
    if (event.key === 'Escape') { editor.selectedShapeIDs = []; return }
    const step = ARROW_STEPS[event.key]
    if (step && editor.selectedShapeIDs.length) {
      event.preventDefault()
      moveShapes(editor.draft, editor.selectedShapeIDs, step)
      editor.checkpoint()
    }
  }

  const bandRect = $derived<IGridRect | undefined>(drag?.kind === 'band' ? normalizeRect(drag.origin, drag.current) : undefined)
</script>

<svelte:window onkeydown={onKeyDown} />

<div class="flex flex-col gap-8 min-w-0 flex-1 min-h-0">
  {#if editor.draft}
    {@const draft = editor.draft}
    <div bind:this={planViewport} bind:contentRect={viewportContentRect}
      class="flex-1 min-h-320 overflow-auto rounded-md border border-line bg-surface"
      role="region" aria-label="Floor plan: drop palette elements here" ondragover={onPaletteDragOver} ondrop={onPaletteDrop}>
      <LayoutCanvas layout={draft} cellSize={editor.cellSize} region={canvasRegion}
        selectedShapeIDs={editor.selectedShapeIDs} warnShapeIDs={overlapIDs} {onPointerDown} {onPointerMove} {onPointerUp}>
        {#snippet overlay()}
          {@const cell = editor.cellSize}
          {#if singleSelected}
            {#if isLineShape(singleSelected)}
              {#each [singleSelected.Points.slice(0, 2), singleSelected.Points.slice(2, 4)] as [handleX, handleY]}
                <circle cx={handleX * cell} cy={handleY * cell} r="6" class="fill-surface stroke-blue-solid cursor-move" stroke-width="2" />
              {/each}
            {:else}
              {@const bounds = shapeBounds(singleSelected)}
              <rect x={bounds.x2 * cell - 6} y={bounds.y2 * cell - 6} width="12" height="12"
                class="fill-surface stroke-blue-solid cursor-nwse-resize" stroke-width="2" />
            {/if}
          {/if}
          {#if bandRect}
            <rect x={bandRect.x1 * cell} y={bandRect.y1 * cell} width={(bandRect.x2 - bandRect.x1) * cell}
              height={(bandRect.y2 - bandRect.y1) * cell} class="fill-blue-bg/40 stroke-blue-solid" stroke-dasharray="4 3" />
          {/if}
        {/snippet}
      </LayoutCanvas>
    </div>
    <!-- Under the plan, so a warning appearing never shifts the canvas under the pointer. -->
    {#if overlapIDs.length}
      <div class="text-red-fg"><i class="icon-[fa--exclamation-triangle] mr-6"></i>
        {tr('Some tables overlap (red outline).|Algunas mesas se superponen (borde rojo).')}</div>
    {/if}
  {/if}
</div>
