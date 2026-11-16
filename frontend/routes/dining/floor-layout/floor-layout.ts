import {
  SHAPE_KIND, TABLE_OUTLINE, isLineShape, rectPoints, shapeBounds, shapeRect, type IDiningLayout, type IDiningStation,
  type IGridPoint, type IGridRect, type ILayoutShape,
} from '../dining.ts'

// The floor-layout editor's geometry (docs/DINING_PLAN.md §10): pure functions over shapes in grid
// cells, so the editor's rules are testable without a canvas. A point is in cells and may be
// fractional (the pointer); shapes snap to whole cells, lines to grid points. A shape's Points are
// always replaced by a new array, never edited in place, so a copy never shares them.

// What can be dragged from the palette onto the plan. The kind and, for tables, the outline.
export type LayoutTool = 'table-rect' | 'table-round' | 'rect' | 'line' | 'label' | 'station' | 'pickup'
// The drag-and-drop data type that carries a LayoutTool from the palette to the plan.
export const LAYOUT_TOOL_DRAG_TYPE = 'application/x-genix-layout-tool'

// The zoom: how many pixels a grid cell takes on screen.
export const MIN_CELL_SIZE = 8
export const MAX_CELL_SIZE = 24

// While editing, the plan is an open sheet: shapes go anywhere, even to negative cells, and the
// editor draws the part of it they need (editorCanvasRegion). What is stored is only the area they use
// plus GRID_MARGIN free cells on every side (trimGridToShapes); 8 is the backend's smallest grid.
export const MIN_GRID_SIZE = 8
export const GRID_MARGIN = 1

export const makeEmptyLayout = (): IDiningLayout => ({
  ID: 0, Name: '', WarehouseID: 0, GridWidth: MIN_GRID_SIZE, GridHeight: MIN_GRID_SIZE, Shapes: [],
  Stations: [{ ID: 1, Name: 'Cocina', ProductIDs: [], CategoryIDs: [], IsDefault: true }],
  QRRequiresConfirmation: true, WaitWarnMinutes: 10, WaitLateMinutes: 20, ss: 1, upd: 0,
})

// The cell under a point, and the grid point nearest to it.
export const cellAt = (point: IGridPoint): IGridPoint => ({ x: Math.floor(point.x), y: Math.floor(point.y) })
export const gridPointNear = (point: IGridPoint): IGridPoint => ({ x: Math.round(point.x), y: Math.round(point.y) })

// The shape moved by whole cells: every x by delta.x, every y by delta.y.
const shiftPoints = (points: number[], delta: IGridPoint) => points.map((value, index) => value + (index % 2 ? delta.y : delta.x))

export const normalizeRect = (from: IGridPoint, to: IGridPoint): IGridRect => ({
  x1: Math.min(from.x, to.x), y1: Math.min(from.y, to.y), x2: Math.max(from.x, to.x), y2: Math.max(from.y, to.y),
})

// The shapes a rubber band covers whole.
export function shapeIDsInRect(shapes: ILayoutShape[], rect: IGridRect): number[] {
  return shapes.filter((shape) => {
    const bounds = shapeBounds(shape)
    return bounds.x1 >= rect.x1 && bounds.y1 >= rect.y1 && bounds.x2 <= rect.x2 && bounds.y2 <= rect.y2
  }).map((shape) => shape.ShapeID)
}

// Moves the shapes of shapeIDs by whole cells.
export function moveShapes(layout: IDiningLayout, shapeIDs: number[], delta: IGridPoint): void {
  const wholeDelta = { x: Math.round(delta.x), y: Math.round(delta.y) }
  for (const shape of layout.Shapes || []) {
    if (shapeIDs.includes(shape.ShapeID)) shape.Points = shiftPoints(shape.Points, wholeDelta)
  }
}

// Resizes from the bottom-right corner to the cell under the point: at least one cell. The shape
// keeps its turn.
export function resizeShapeTo(shape: ILayoutShape, point: IGridPoint): void {
  const corner = gridPointNear(point)
  const rect = shapeRect(shape)
  rect.w = Math.max(1, corner.x - rect.x)
  rect.h = Math.max(1, corner.y - rect.y)
  shape.Points = rectPoints(rect)
}

// Moves one end of a line to the grid point nearest the point; a line never collapses to a point.
export function moveLineEnd(shape: ILayoutShape, end: 1 | 2, point: IGridPoint): void {
  const { x, y } = gridPointNear(point)
  const [x1, y1, x2, y2] = shape.Points
  const [otherX, otherY] = end === 1 ? [x2, y2] : [x1, y1]
  if (x === otherX && y === otherY) return
  shape.Points = end === 1 ? [x, y, x2, y2] : [x1, y1, x, y]
}

// A temporary ID for a new shape: negative and below every other, so the backend knows it is new.
export const nextTemporaryShapeID = (shapes: ILayoutShape[]): number =>
  Math.min(0, ...shapes.map((shape) => shape.ShapeID)) - 1

// The next free table name: "M" and one more than the highest number already used.
export function nextTableLabel(shapes: ILayoutShape[]): string {
  const usedNumbers = shapes
    .filter((shape) => shape.Kind === SHAPE_KIND.TABLE)
    .map((shape) => Number(/(\d+)$/.exec(shape.Label || '')?.[1] || 0))
  return `M${Math.max(0, ...usedNumbers) + 1}`
}

const defaultStationID = (stations: IDiningStation[]) =>
  (stations.find((station) => station.IsDefault) || stations[0])?.ID || 0

// The length a new wall gets, in cells: it is then stretched by its end handles.
const NEW_LINE_LENGTH = 10

// A new shape of the tool, its top-left on the cell (a line starts on that grid point, horizontal).
export function makeShape(tool: LayoutTool, layout: IDiningLayout, from: IGridPoint): ILayoutShape {
  const shapes = layout.Shapes || []
  const shapeID = nextTemporaryShapeID(shapes)
  // An unturned block of w × h cells from the clicked cell.
  const cellsFrom = (w: number, h: number) => rectPoints({ x: from.x, y: from.y, w, h, quarterTurns: 0 })
  switch (tool) {
    case 'table-rect':
    case 'table-round':
      return { ShapeID: shapeID, Kind: SHAPE_KIND.TABLE, Points: cellsFrom(3, 3), Seats: 4,
        Outline: tool === 'table-round' ? TABLE_OUTLINE.ROUND : TABLE_OUTLINE.RECT, Label: nextTableLabel(shapes) }
    case 'line':
      return { ShapeID: shapeID, Kind: SHAPE_KIND.LINE, Points: [from.x, from.y, from.x + NEW_LINE_LENGTH, from.y] }
    case 'label':
      return { ShapeID: shapeID, Kind: SHAPE_KIND.LABEL, Points: cellsFrom(6, 2), Label: 'Texto' }
    case 'station':
      return { ShapeID: shapeID, Kind: SHAPE_KIND.STATION, Points: cellsFrom(4, 2), StationID: defaultStationID(layout.Stations || []) }
    case 'pickup':
      return { ShapeID: shapeID, Kind: SHAPE_KIND.PICKUP, Points: cellsFrom(6, 2), StationID: defaultStationID(layout.Stations || []) }
    default:
      return { ShapeID: shapeID, Kind: SHAPE_KIND.RECT, Points: cellsFrom(3, 2) }
  }
}

// Copies of the shapes, one cell down and right, with new temporary IDs and,
// for tables, new names. Returns the copies' IDs, to select them.
export function duplicateShapes(layout: IDiningLayout, shapeIDs: number[]): number[] {
  const shapes = layout.Shapes || []
  const copies: ILayoutShape[] = []
  for (const shape of shapes.filter((candidate) => shapeIDs.includes(candidate.ShapeID))) {
    const copy: ILayoutShape = { ...shape, Points: [...shape.Points], ShapeID: nextTemporaryShapeID([...shapes, ...copies]) }
    if (copy.Kind === SHAPE_KIND.TABLE) copy.Label = nextTableLabel([...shapes, ...copies])
    copies.push(copy)
  }
  layout.Shapes = [...shapes, ...copies]
  const copyIDs = copies.map((copy) => copy.ShapeID)
  moveShapes(layout, copyIDs, { x: 1, y: 1 })
  return copyIDs
}

// Turns each shape 90° clockwise in place. A block turns about its center (its sides swap, its corner
// list starts one corner later); a line turns about its first end.
export function rotateShapes(layout: IDiningLayout, shapeIDs: number[]): void {
  for (const shape of (layout.Shapes || []).filter((candidate) => shapeIDs.includes(candidate.ShapeID))) {
    let turnedPoints: number[]
    if (isLineShape(shape)) {
      const [x1, y1, x2, y2] = shape.Points
      turnedPoints = [x1, y1, x1 - (y2 - y1), y1 + (x2 - x1)]
    } else {
      const rect = shapeRect(shape)
      turnedPoints = rectPoints({
        x: rect.x + Math.trunc((rect.w - rect.h) / 2), y: rect.y + Math.trunc((rect.h - rect.w) / 2),
        w: rect.h, h: rect.w, quarterTurns: rect.quarterTurns + 1,
      })
    }
    shape.Points = turnedPoints
  }
}

// The part of the open sheet the editor draws, in cells. It is the viewport's size, slid as little as
// possible from the anchor to hold every shape with GRID_MARGIN free cells around it, so a shape moving
// away from an edge gives that space back. It is larger only when the shapes do not fit the viewport,
// and then it is exactly what they need: that is when the plan scrolls.
export function editorCanvasRegion(shapes: ILayoutShape[], anchor: IGridPoint, viewWidth: number, viewHeight: number): IGridRect {
  if (!shapes.length) return { x1: anchor.x, y1: anchor.y, x2: anchor.x + viewWidth, y2: anchor.y + viewHeight }
  const bounds = shapes.map(shapeBounds)
  const regionAxis = (anchorStart: number, viewSize: number, low: number, high: number) => {
    if (high - low >= viewSize) return [low, high]
    const start = Math.min(Math.max(anchorStart, high - viewSize), low)
    return [start, start + viewSize]
  }
  const [x1, x2] = regionAxis(anchor.x, viewWidth,
    Math.min(...bounds.map((rect) => rect.x1)) - GRID_MARGIN, Math.max(...bounds.map((rect) => rect.x2)) + GRID_MARGIN)
  const [y1, y2] = regionAxis(anchor.y, viewHeight,
    Math.min(...bounds.map((rect) => rect.y1)) - GRID_MARGIN, Math.max(...bounds.map((rect) => rect.y2)) + GRID_MARGIN)
  return { x1, y1, x2, y2 }
}

// The grid as it is stored: the shapes shifted so the used area starts GRID_MARGIN cells in, and the
// grid ending GRID_MARGIN cells past it (at least MIN_GRID_SIZE). Returns the shift, so the editor can
// keep the plan where it was on screen.
export function trimGridToShapes(layout: IDiningLayout): IGridPoint {
  const shapes = layout.Shapes || []
  if (!shapes.length) {
    layout.GridWidth = MIN_GRID_SIZE
    layout.GridHeight = MIN_GRID_SIZE
    return { x: 0, y: 0 }
  }
  const bounds = shapes.map(shapeBounds)
  const shift = {
    x: GRID_MARGIN - Math.min(...bounds.map((rect) => rect.x1)),
    y: GRID_MARGIN - Math.min(...bounds.map((rect) => rect.y1)),
  }
  for (const shape of shapes) shape.Points = shiftPoints(shape.Points, shift)
  layout.GridWidth = Math.max(MIN_GRID_SIZE, Math.max(...bounds.map((rect) => rect.x2)) + shift.x + GRID_MARGIN)
  layout.GridHeight = Math.max(MIN_GRID_SIZE, Math.max(...bounds.map((rect) => rect.y2)) + shift.y + GRID_MARGIN)
  return shift
}

export function deleteShapes(layout: IDiningLayout, shapeIDs: number[]): void {
  layout.Shapes = (layout.Shapes || []).filter((shape) => !shapeIDs.includes(shape.ShapeID))
}

// Tables that overlap another table: allowed, but likely a mistake, so the editor warns.
export function overlappingTableIDs(shapes: ILayoutShape[]): number[] {
  const tables = shapes.filter((shape) => shape.Kind === SHAPE_KIND.TABLE)
  const overlapping = new Set<number>()
  for (let leftIndex = 0; leftIndex < tables.length; leftIndex++) {
    for (let rightIndex = leftIndex + 1; rightIndex < tables.length; rightIndex++) {
      const left = shapeBounds(tables[leftIndex]), right = shapeBounds(tables[rightIndex])
      if (left.x1 < right.x2 && right.x1 < left.x2 && left.y1 < right.y2 && right.y1 < left.y2) {
        overlapping.add(tables[leftIndex].ShapeID)
        overlapping.add(tables[rightIndex].ShapeID)
      }
    }
  }
  return [...overlapping]
}

// Exactly one station is the default: choosing one unsets the others.
export function setDefaultStation(stations: IDiningStation[], stationID: number): void {
  for (const station of stations) station.IsDefault = station.ID === stationID
}

// Why a station cannot be deleted, or '' when it can: the default one stays, and so does one a
// station or pickup mark still stands for.
export function stationDeleteBlocker(layout: IDiningLayout, stationID: number): string {
  const station = (layout.Stations || []).find((candidate) => candidate.ID === stationID)
  if (station?.IsDefault) return 'The default station cannot be deleted.|La estación por defecto no se puede eliminar.'
  if ((layout.Shapes || []).some((shape) => shape.StationID === stationID &&
    (shape.Kind === SHAPE_KIND.STATION || shape.Kind === SHAPE_KIND.PICKUP))) {
    return 'A mark on the plan stands for this station: delete it first.|Una marca del plano representa esta estación: elimínela primero.'
  }
  return ''
}

// A new station: the next ID, never the default (there is exactly one).
export const makeStation = (stations: IDiningStation[]): IDiningStation => ({
  ID: Math.max(0, ...stations.map((station) => station.ID)) + 1, Name: '', ProductIDs: [], CategoryIDs: [], IsDefault: false,
})
