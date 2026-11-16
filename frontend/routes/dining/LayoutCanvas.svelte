<script lang="ts">
  import type { Snippet } from 'svelte'
  import { tr } from '#core/store.svelte.ts'
  import { SHAPE_KIND, TABLE_OUTLINE, shapeRect, type IDiningLayout, type IGridRect, type ILayoutShape } from './dining.ts'

  // Draws a layout as an SVG on its grid. It holds no editing logic: the editor and the floor map
  // get the pointer in grid cells, color the shapes and draw their own overlay on top.
  let {
    layout, cellSize, region, selectedShapeIDs = [],
    warnShapeIDs = [], showGrid = true, shapeFill, onPointerDown, onPointerMove, onPointerUp, overlay,
  }: {
    layout: IDiningLayout
    cellSize: number
    // The cells drawn, when not the layout's grid: the editor draws a window of its open sheet, which
    // may start at negative cells.
    region?: IGridRect
    selectedShapeIDs?: number[]
    // Tables drawn with a warning outline (the editor's overlaps).
    warnShapeIDs?: number[]
    showGrid?: boolean
    // The fill class of a shape, replacing its kind's (the floor map's states).
    shapeFill?: (shape: ILayoutShape) => string | undefined
    onPointerDown?: (event: PointerEvent, point: { x: number, y: number }) => void
    onPointerMove?: (event: PointerEvent, point: { x: number, y: number }) => void
    onPointerUp?: (event: PointerEvent, point: { x: number, y: number }) => void
    overlay?: Snippet
  } = $props()

  const shownRegion = $derived(region || { x1: 0, y1: 0, x2: layout.GridWidth, y2: layout.GridHeight })
  const originXPx = $derived(shownRegion.x1 * cellSize)
  const originYPx = $derived(shownRegion.y1 * cellSize)
  const widthPx = $derived((shownRegion.x2 - shownRegion.x1) * cellSize)
  const heightPx = $derived((shownRegion.y2 - shownRegion.y1) * cellSize)
  const stationNames = $derived(new Map((layout.Stations || []).map((station) => [station.ID, station.Name])))

  // The pointer in the layout's cells, fractional: the caller snaps it as the action needs.
  function gridPointOf(event: PointerEvent) {
    const bounds = (event.currentTarget as SVGSVGElement).getBoundingClientRect()
    return { x: (event.clientX - bounds.left) / cellSize + shownRegion.x1, y: (event.clientY - bounds.top) / cellSize + shownRegion.y1 }
  }

  // Fill and stroke apart: a selection or warning stroke replaces the kind's instead of competing
  // with it, and the floor map recolors the fill only.
  const KIND_FILL: Record<number, string> = {
    [SHAPE_KIND.TABLE]: 'fill-surface-raised', [SHAPE_KIND.RECT]: 'fill-surface-strong', [SHAPE_KIND.LABEL]: 'fill-transparent',
    [SHAPE_KIND.STATION]: 'fill-amber-bg', [SHAPE_KIND.PICKUP]: 'fill-green-bg',
  }
  const KIND_STROKE: Record<number, string> = {
    [SHAPE_KIND.TABLE]: 'stroke-line-strong', [SHAPE_KIND.RECT]: 'stroke-line-strong', [SHAPE_KIND.LABEL]: 'stroke-line',
    [SHAPE_KIND.STATION]: 'stroke-amber-border', [SHAPE_KIND.PICKUP]: 'stroke-green-border',
  }

  function shapeText(shape: ILayoutShape): string {
    if (shape.Kind === SHAPE_KIND.STATION) return stationNames.get(shape.StationID || 0) || tr('Station|Estación')
    if (shape.Kind === SHAPE_KIND.PICKUP) return `${tr('Pickup|Recojo')} · ${stationNames.get(shape.StationID || 0) || ''}`
    return shape.Label || ''
  }

  // Text turns with the shape a quarter turn either way; a half turn stays upright, never upside down.
  const TEXT_ROTATION_BY_QUARTER_TURNS = [0, 90, 0, -90]
</script>

<svg role="application" aria-label={tr(`Floor plan ${layout.Name}|Plano ${layout.Name}`)}
  width={widthPx} height={heightPx} viewBox="{originXPx} {originYPx} {widthPx} {heightPx}" class="block select-none touch-none bg-surface"
  onpointerdown={(event) => onPointerDown?.(event, gridPointOf(event))}
  onpointermove={(event) => onPointerMove?.(event, gridPointOf(event))}
  onpointerup={(event) => onPointerUp?.(event, gridPointOf(event))}
>
  {#if showGrid}
    <defs>
      <pattern id="layout-grid-{layout.ID}-{cellSize}" width={cellSize} height={cellSize} patternUnits="userSpaceOnUse">
        <path d="M {cellSize} 0 L 0 0 0 {cellSize}" class="fill-none stroke-line-soft" stroke-width="1" />
      </pattern>
    </defs>
    <rect x={originXPx} y={originYPx} width={widthPx} height={heightPx} fill="url(#layout-grid-{layout.ID}-{cellSize})" class="stroke-line-strong" />
  {/if}

  {#each layout.Shapes || [] as shape (shape.ShapeID)}
    {@const isSelected = selectedShapeIDs.includes(shape.ShapeID)}
    {@const isWarned = warnShapeIDs.includes(shape.ShapeID)}
    {#if shape.Kind === SHAPE_KIND.LINE}
      <line x1={shape.Points[0] * cellSize} y1={shape.Points[1] * cellSize} x2={shape.Points[2] * cellSize} y2={shape.Points[3] * cellSize}
        class={isSelected ? 'stroke-blue-solid' : 'stroke-fg-soft'} stroke-width={Math.max(3, cellSize / 3)} stroke-linecap="round" />
    {:else}
      {@const rect = shapeRect(shape)}
      {@const x = rect.x * cellSize}
      {@const y = rect.y * cellSize}
      {@const width = rect.w * cellSize}
      {@const height = rect.h * cellSize}
      {@const textRotation = TEXT_ROTATION_BY_QUARTER_TURNS[rect.quarterTurns]}
      {@const shapeCss = `${shapeFill?.(shape) || KIND_FILL[shape.Kind]} ${isSelected ? 'stroke-blue-solid' : isWarned ? 'stroke-red-solid' : KIND_STROKE[shape.Kind]}`}
      {@const isDashed = shape.Kind === SHAPE_KIND.STATION || shape.Kind === SHAPE_KIND.PICKUP || shape.Kind === SHAPE_KIND.LABEL}
      <!-- A table tall enough (along its turned text) shows its seats under its name. -->
      {@const showsSeats = shape.Kind === SHAPE_KIND.TABLE && !!shape.Seats && (textRotation ? width : height) >= 40}
      {@const shapeName = shapeText(shape)}
      {@const textBoxWidth = textRotation ? height : width}
      {@const textBoxHeight = textRotation ? width : height}
      <g class="dining-shape">
        {#if shape.Kind === SHAPE_KIND.TABLE && shape.Outline === TABLE_OUTLINE.ROUND}
          <ellipse cx={x + width / 2} cy={y + height / 2} rx={width / 2 - 1} ry={height / 2 - 1}
            class={shapeCss} stroke-width={isSelected || isWarned ? 3 : 1.5} />
        {:else}
          <rect x={x + 1} y={y + 1} width={width - 2} height={height - 2} rx={shape.Kind === SHAPE_KIND.TABLE ? 6 : 2}
            class={shapeCss}
            stroke-width={isSelected || isWarned ? 3 : 1.5} stroke-dasharray={isDashed ? '6 4' : undefined} />
        {/if}
        <!-- The name and the seats turn together, about the shape's center. They sit in an HTML box
             (SVG text cannot wrap), as long as the shape along the turned text, so a long name wraps
             onto several lines inside the shape. -->
        {#if shapeName || showsSeats}
          <g transform={textRotation ? `rotate(${textRotation} ${x + width / 2} ${y + height / 2})` : undefined}>
            <foreignObject x={x + (width - textBoxWidth) / 2} y={y + (height - textBoxHeight) / 2}
              width={textBoxWidth} height={textBoxHeight} class="pointer-events-none">
              <div class="w-full h-full flex flex-col items-center justify-center px-2 text-center text-[14px] leading-tight break-words">
                {#if shapeName}<div class="text-fg {shape.Kind === SHAPE_KIND.TABLE ? 'ff-bold' : ''}">{shapeName}</div>{/if}
                {#if showsSeats}<div class="text-fg-muted">{shape.Seats}p</div>{/if}
              </div>
            </foreignObject>
          </g>
        {/if}
      </g>
    {/if}
  {/each}

  {@render overlay?.()}
</svg>
