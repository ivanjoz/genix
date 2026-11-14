<script lang="ts">
  import Button from '#components/buttons/Button.svelte'
  import { tr } from '#core/store.svelte.ts'
  import LayoutCanvas from '../LayoutCanvas.svelte'
  import { SHAPE_KIND, isLineShape, shapeAt, shapeBounds, type IGridPoint, type IGridRect } from '../dining.ts'
  import {
    CELL_SIZES, cellAt, deleteShapes, duplicateShapes, fitShapeInGrid, gridPointNear, makeShape, moveLineEnd,
    moveShapes, normalizeRect, overlappingTableIDs, resizeShapeTo, rotateShapes, shapeIDsInRect, type LayoutTool,
  } from './floor-layout.ts'
  import type { FloorLayoutEditorState } from './floor-layout.svelte.ts'

  // The drawing surface of the floor-layout editor: tools, pointer gestures and keyboard shortcuts
  // over LayoutCanvas. The geometry is in floor-layout.ts; this only routes the gestures to it.
  let { editor }: { editor: FloorLayoutEditorState } = $props()

  const HANDLE_HIT_DISTANCE = 0.6 // cells

  const TOOLS: { tool: LayoutTool, icon: string, name: string, label: string }[] = [
    { tool: 'select', icon: 'icon-[fa--mouse-pointer]', name: 'Select|Seleccionar', label: 'Selects, moves and resizes shapes.' },
    { tool: 'table-rect', icon: 'icon-[fa--square-o]', name: 'Table|Mesa', label: 'Places a square table.' },
    { tool: 'table-round', icon: 'icon-[fa--circle-thin]', name: 'Round|Redonda', label: 'Places a round table.' },
    { tool: 'rect', icon: 'icon-[fa--square]', name: 'Block|Bloque', label: 'Places a block: bar, column, planter.' },
    { tool: 'line', icon: 'icon-[fa--minus]', name: 'Wall|Pared', label: 'Draws a wall or divider by dragging.' },
    { tool: 'label', icon: 'icon-[fa--font]', name: 'Text|Texto', label: 'Places a text.' },
    { tool: 'station', icon: 'icon-[fa--fire]', name: 'Station|Estación', label: 'Marks where a prep station is.' },
    { tool: 'pickup', icon: 'icon-[fa--hand-paper-o]', name: 'Pickup|Recojo', label: 'Marks where waiters collect a station\'s dishes.' },
  ]

  // The gesture in progress, from pointerdown to pointerup.
  type Drag =
    | { kind: 'move', origin: IGridPoint, applied: IGridPoint }
    | { kind: 'resize', shapeID: number }
    | { kind: 'line-end', shapeID: number, end: 1 | 2 }
    | { kind: 'band', origin: IGridPoint, current: IGridPoint }
    | { kind: 'new-line', origin: IGridPoint, current: IGridPoint }
  let drag = $state<Drag>()

  const overlapIDs = $derived(overlappingTableIDs(editor.draft?.Shapes || []))
  const singleSelected = $derived(editor.selectedShapeIDs.length === 1 ? editor.selectedShapes()[0] : undefined)

  // The handle under the point, for a single selected shape: a block's bottom-right corner, a line's ends.
  function handleAt(point: IGridPoint): Drag | undefined {
    const shape = singleSelected
    if (!shape) return undefined
    const isNear = (x: number, y: number) => Math.hypot(point.x - x, point.y - y) <= HANDLE_HIT_DISTANCE
    if (isLineShape(shape)) {
      const [x1, y1, x2, y2] = shape.Points
      if (isNear(x1, y1)) return { kind: 'line-end', shapeID: shape.ShapeID, end: 1 }
      if (isNear(x2, y2)) return { kind: 'line-end', shapeID: shape.ShapeID, end: 2 }
      return undefined
    }
    const bounds = shapeBounds(shape)
    return isNear(bounds.x2, bounds.y2) ? { kind: 'resize', shapeID: shape.ShapeID } : undefined
  }

  function placeShape(point: IGridPoint) {
    const draft = editor.draft!
    const shape = makeShape(editor.tool, draft, cellAt(point))
    fitShapeInGrid(shape, draft.GridWidth, draft.GridHeight)
    draft.Shapes = [...(draft.Shapes || []), shape]
    editor.selectedShapeIDs = [shape.ShapeID]
    editor.tool = 'select'
    editor.checkpoint()
  }

  function onPointerDown(event: PointerEvent, point: IGridPoint) {
    if (!editor.draft || event.button !== 0) return
    ;(event.currentTarget as Element).setPointerCapture(event.pointerId)
    if (editor.tool === 'line') {
      const origin = gridPointNear(point)
      drag = { kind: 'new-line', origin, current: origin }
      return
    }
    if (editor.tool !== 'select') { placeShape(point); return }

    const handleDrag = handleAt(point)
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
    drag = { kind: 'move', origin: point, applied: { x: 0, y: 0 } }
  }

  function onPointerMove(_event: PointerEvent, point: IGridPoint) {
    const draft = editor.draft
    if (!drag || !draft) return
    if (drag.kind === 'move') {
      // Whole cells only: the step not yet applied since the gesture began.
      const target = { x: Math.round(point.x - drag.origin.x), y: Math.round(point.y - drag.origin.y) }
      const step = { x: target.x - drag.applied.x, y: target.y - drag.applied.y }
      if (!step.x && !step.y) return
      const before = shapeBounds(editor.selectedShapes()[0])
      moveShapes(draft, editor.selectedShapeIDs, step)
      const after = shapeBounds(editor.selectedShapes()[0])
      // What the grid let the group move, so a clamped move does not drift from the pointer.
      drag.applied = { x: drag.applied.x + after.x1 - before.x1, y: drag.applied.y + after.y1 - before.y1 }
    } else if (drag.kind === 'resize') {
      const shape = draft.Shapes?.find((candidate) => candidate.ShapeID === (drag as { shapeID: number }).shapeID)
      if (shape) resizeShapeTo(shape, point, draft.GridWidth, draft.GridHeight)
    } else if (drag.kind === 'line-end') {
      const lineDrag = drag
      const shape = draft.Shapes?.find((candidate) => candidate.ShapeID === lineDrag.shapeID)
      if (shape) moveLineEnd(shape, lineDrag.end, point, draft.GridWidth, draft.GridHeight)
    } else if (drag.kind === 'band') {
      drag.current = point
    } else if (drag.kind === 'new-line') {
      const gridPoint = gridPointNear(point)
      drag.current = { x: Math.max(0, Math.min(draft.GridWidth, gridPoint.x)), y: Math.max(0, Math.min(draft.GridHeight, gridPoint.y)) }
    }
  }

  function onPointerUp() {
    const draft = editor.draft
    const finishedDrag = drag
    drag = undefined
    if (!finishedDrag || !draft) return
    if (finishedDrag.kind === 'band') {
      const bandIDs = shapeIDsInRect(draft.Shapes || [], normalizeRect(finishedDrag.origin, finishedDrag.current))
      editor.selectedShapeIDs = [...new Set([...editor.selectedShapeIDs, ...bandIDs])]
      return
    }
    if (finishedDrag.kind === 'new-line') {
      const { origin, current } = finishedDrag
      if (origin.x === current.x && origin.y === current.y) return
      const line = makeShape('line', draft, origin, current)
      draft.Shapes = [...(draft.Shapes || []), line]
      editor.selectedShapeIDs = [line.ShapeID]
      editor.tool = 'select'
    }
    editor.checkpoint()
  }

  // The actions on the selection, shared by the toolbar and the keyboard.
  function duplicateSelection() {
    if (!editor.draft || !editor.selectedShapeIDs.length) return
    editor.selectedShapeIDs = duplicateShapes(editor.draft, editor.selectedShapeIDs)
    editor.checkpoint()
  }
  function rotateSelection() {
    if (!editor.draft || !editor.selectedShapeIDs.length) return
    rotateShapes(editor.draft, editor.selectedShapeIDs)
    editor.checkpoint()
  }
  function deleteSelection() {
    if (!editor.draft || !editor.selectedShapeIDs.length) return
    deleteShapes(editor.draft, editor.selectedShapeIDs)
    editor.selectedShapeIDs = []
    editor.checkpoint()
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
    if (isCommand && key === 'd') { event.preventDefault(); duplicateSelection(); return }
    if (event.key === 'Delete' || event.key === 'Backspace') { if (editor.selectedShapeIDs.length) { event.preventDefault(); deleteSelection() }; return }
    if (event.key === 'Escape') { editor.selectedShapeIDs = []; editor.tool = 'select'; return }
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

<div class="flex flex-col gap-8 min-w-0">
  <div class="flex flex-wrap items-center gap-6" aria-label="Floor plan tools and actions">
    {#each TOOLS as toolOption (toolOption.tool)}
      <Button icon={toolOption.icon} name={toolOption.name} label={toolOption.label} height="compact" hideNameOnMobile
        color={editor.tool === toolOption.tool ? 'blue' : undefined}
        onClick={() => { editor.tool = toolOption.tool }} />
    {/each}
    <div class="w-1 h-24 bg-line mx-4"></div>
    <Button icon="icon-[fa--clone]" label="Duplicates the selected shapes (Ctrl+D)." height="compact"
      disabled={!editor.selectedShapeIDs.length} onClick={duplicateSelection} />
    <Button icon="icon-[fa--refresh]" label="Rotates the selected shapes 90°." height="compact"
      disabled={!editor.selectedShapeIDs.length} onClick={rotateSelection} />
    <Button icon="icon-[fa--trash]" label="Deletes the selected shapes (Delete)." height="compact" color="red" mode="subtle"
      disabled={!editor.selectedShapeIDs.length} onClick={deleteSelection} />
    <Button icon="icon-[fa--undo]" label="Undoes the last change (Ctrl+Z)." height="compact"
      disabled={!editor.canUndo} onClick={() => editor.undo()} />
    <Button icon="icon-[fa--repeat]" label="Redoes the undone change (Ctrl+Y)." height="compact"
      disabled={!editor.canRedo} onClick={() => editor.redo()} />
    <div class="ml-auto flex items-center gap-4" aria-label="Zoom">
      <i class="icon-[fa--search] text-fg-muted"></i>
      {#each CELL_SIZES as cellSize}
        <Button name={`${cellSize}px`} label="Sets the size of a grid cell on screen." height="compact"
          color={editor.cellSize === cellSize ? 'blue' : undefined} onClick={() => { editor.cellSize = cellSize }} />
      {/each}
    </div>
  </div>

  {#if editor.draft}
    {@const draft = editor.draft}
    <div class="overflow-auto max-h-[calc(100vh-var(--header-height)-150px)] rounded-md border border-line {editor.tool === 'select' ? '' : 'cursor-crosshair'}">
      <LayoutCanvas layout={draft} cellSize={editor.cellSize} selectedShapeIDs={editor.selectedShapeIDs} warnShapeIDs={overlapIDs}
        {onPointerDown} {onPointerMove} {onPointerUp}>
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
          {#if drag?.kind === 'new-line'}
            <line x1={drag.origin.x * cell} y1={drag.origin.y * cell} x2={drag.current.x * cell} y2={drag.current.y * cell}
              class="stroke-blue-solid" stroke-width={Math.max(3, cell / 3)} stroke-linecap="round" />
          {/if}
        {/snippet}
      </LayoutCanvas>
    </div>
    <!-- Under the plan, so a warning appearing never shifts the canvas under the pointer. -->
    <div class="text-fg-muted">
      {tr('Shift+click or drag over empty space to select several; arrows move 1 cell. Tables:|Shift+clic o arrastrar sobre el vacío para elegir varios; las flechas mueven 1 celda. Mesas:')}
      {(draft.Shapes || []).filter((shape) => shape.Kind === SHAPE_KIND.TABLE).length}
    </div>
    {#if overlapIDs.length}
      <div class="text-red-fg"><i class="icon-[fa--exclamation-triangle] mr-6"></i>
        {tr('Some tables overlap (red outline).|Algunas mesas se superponen (borde rojo).')}</div>
    {/if}
  {/if}
</div>
