import {
  SHAPE_KIND, TABLE_OUTLINE, isLineShape, rectPoints, shapeBounds, shapeRect, type IDiningLayout, type IDiningStation,
  type IGridPoint, type IGridRect, type ILayoutShape,
} from '../dining.ts'

// The floor-layout editor's geometry (docs/DINING_PLAN.md §10): pure functions over shapes in grid
// cells, so the editor's rules are testable without a canvas. A point is in cells and may be
// fractional (the pointer); shapes snap to whole cells, lines to grid points. A shape's Points are
// always replaced by a new array, never edited in place, so a copy never shares them.

// What the editor can place. The kind and, for tables, the outline.
export type LayoutTool = 'select' | 'table-rect' | 'table-round' | 'rect' | 'line' | 'label' | 'station' | 'pickup'

export const CELL_SIZES = [8, 12, 16] as const

export const makeEmptyLayout = (): IDiningLayout => ({
  ID: 0, Name: '', WarehouseID: 0, GridWidth: 40, GridHeight: 30, Shapes: [],
  Stations: [{ ID: 1, Name: 'Cocina', ProductIDs: [], CategoryIDs: [], IsDefault: true }],
  QRRequiresConfirmation: true, WaitWarnMinutes: 10, WaitLateMinutes: 20, ss: 1, upd: 0,
})

// The cell under a point, and the grid point nearest to it.
export const cellAt = (point: IGridPoint): IGridPoint => ({ x: Math.floor(point.x), y: Math.floor(point.y) })
export const gridPointNear = (point: IGridPoint): IGridPoint => ({ x: Math.round(point.x), y: Math.round(point.y) })

// The shape moved by whole cells: every x by delta.x, every y by delta.y.
const shiftPoints = (points: number[], delta: IGridPoint) => points.map((value, index) => value + (index % 2 ? delta.y : delta.x))

// The shift that brings a span low..high inside 0..limit; one longer than the limit starts at 0.
const shiftIntoRange = (low: number, high: number, limit: number) =>
  low < 0 || high - low > limit ? -low : Math.min(0, limit - high)

// Whether the box fits the grid at all, wherever it is placed.
const fitsGrid = (bounds: IGridRect, gridWidth: number, gridHeight: number) =>
  bounds.x2 - bounds.x1 <= gridWidth && bounds.y2 - bounds.y1 <= gridHeight

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

// The move a group can make without any of its shapes leaving the grid.
export function clampGroupDelta(shapes: ILayoutShape[], delta: IGridPoint, gridWidth: number, gridHeight: number): IGridPoint {
  if (!shapes.length) return { x: 0, y: 0 }
  const bounds = shapes.map(shapeBounds)
  const minX = Math.min(...bounds.map((rect) => rect.x1)), maxX = Math.max(...bounds.map((rect) => rect.x2))
  const minY = Math.min(...bounds.map((rect) => rect.y1)), maxY = Math.max(...bounds.map((rect) => rect.y2))
  return {
    x: Math.max(-minX, Math.min(gridWidth - maxX, delta.x)),
    y: Math.max(-minY, Math.min(gridHeight - maxY, delta.y)),
  }
}

// Moves the shapes of shapeIDs by whole cells, as a group kept inside the grid.
export function moveShapes(layout: IDiningLayout, shapeIDs: number[], delta: IGridPoint): void {
  const movedShapes = (layout.Shapes || []).filter((shape) => shapeIDs.includes(shape.ShapeID))
  const clampedDelta = clampGroupDelta(movedShapes, { x: Math.round(delta.x), y: Math.round(delta.y) }, layout.GridWidth, layout.GridHeight)
  for (const shape of movedShapes) shape.Points = shiftPoints(shape.Points, clampedDelta)
}

// Resizes from the bottom-right corner to the cell under the point: at least one cell, inside the
// grid. The shape keeps its turn.
export function resizeShapeTo(shape: ILayoutShape, point: IGridPoint, gridWidth: number, gridHeight: number): void {
  const corner = gridPointNear(point)
  const rect = shapeRect(shape)
  rect.w = Math.max(1, Math.min(gridWidth, corner.x) - rect.x)
  rect.h = Math.max(1, Math.min(gridHeight, corner.y) - rect.y)
  shape.Points = rectPoints(rect)
}

// Moves one end of a line to the grid point nearest the pointer; a line never collapses to a point.
export function moveLineEnd(shape: ILayoutShape, end: 1 | 2, point: IGridPoint, gridWidth: number, gridHeight: number): void {
  const gridPoint = gridPointNear(point)
  const x = Math.max(0, Math.min(gridWidth, gridPoint.x)), y = Math.max(0, Math.min(gridHeight, gridPoint.y))
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

// A new shape of the tool, its top-left on the cell (a line from one grid point to another).
export function makeShape(tool: LayoutTool, layout: IDiningLayout, from: IGridPoint, to?: IGridPoint): ILayoutShape {
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
      return { ShapeID: shapeID, Kind: SHAPE_KIND.LINE, Points: [from.x, from.y, to?.x ?? from.x + 4, to?.y ?? from.y] }
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

// Keeps a new or edited shape inside the grid. A block shifts back in, shrunk only if larger than the
// grid. A line shifts back in whole, so it keeps its length (only one longer than the grid is cut at
// its edge), and one typed with both ends on one point gets one cell of length: a line never
// collapses to a point the backend would refuse.
export function fitShapeInGrid(shape: ILayoutShape, gridWidth: number, gridHeight: number): void {
  if (!isLineShape(shape)) {
    const rect = shapeRect(shape)
    rect.w = Math.max(1, Math.min(gridWidth, rect.w))
    rect.h = Math.max(1, Math.min(gridHeight, rect.h))
    rect.x = Math.max(0, Math.min(gridWidth - rect.w, rect.x))
    rect.y = Math.max(0, Math.min(gridHeight - rect.h, rect.y))
    shape.Points = rectPoints(rect)
    return
  }
  const bounds = shapeBounds(shape)
  const shift = { x: shiftIntoRange(bounds.x1, bounds.x2, gridWidth), y: shiftIntoRange(bounds.y1, bounds.y2, gridHeight) }
  const [x1, y1, x2, y2] = shiftPoints(shape.Points, shift)
    .map((value, index) => Math.max(0, Math.min(index % 2 ? gridHeight : gridWidth, value)))
  const isCollapsed = x1 === x2 && y1 === y2
  shape.Points = isCollapsed ? [x1, y1, x1 < gridWidth ? x1 + 1 : x1 - 1, y1] : [x1, y1, x2, y2]
}

// The coordinates the side panel types: a block's top-left cell and size (X, Y, W, H), a line's two
// ends (X, Y and X2, Y2).
export interface IShapeGeometryFields { X: number, Y: number, W: number, H: number, X2: number, Y2: number }

export function shapeGeometryFields(shape: ILayoutShape): IShapeGeometryFields {
  if (isLineShape(shape)) {
    const [x1, y1, x2, y2] = shape.Points
    return { X: x1, Y: y1, X2: x2, Y2: y2, W: 0, H: 0 }
  }
  const rect = shapeRect(shape)
  return { X: rect.x, Y: rect.y, W: rect.w, H: rect.h, X2: 0, Y2: 0 }
}

// Writes typed coordinates back into the shape, kept inside the grid. A block keeps its turn and at
// least one cell per side (a side of 0 would leave its corners unable to tell the turn); an emptied
// or non-numeric field counts as 0.
export function applyShapeGeometryFields(shape: ILayoutShape, fields: IShapeGeometryFields, gridWidth: number, gridHeight: number): void {
  const typed = (value: number) => Math.round(Number(value)) || 0
  shape.Points = isLineShape(shape)
    ? [typed(fields.X), typed(fields.Y), typed(fields.X2), typed(fields.Y2)]
    : rectPoints({ x: typed(fields.X), y: typed(fields.Y), w: Math.max(1, typed(fields.W)), h: Math.max(1, typed(fields.H)),
      quarterTurns: shapeRect(shape).quarterTurns })
  fitShapeInGrid(shape, gridWidth, gridHeight)
}

// Copies of the shapes, one cell down and right (kept inside the grid), with new temporary IDs and,
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

// Turns each shape 90° clockwise in place, then shifts it back inside the grid whole. A block turns
// about its center (its sides swap, its corner list starts one corner later); a line turns about its
// first end. A shape that would not fit the grid once turned (a wall longer than the grid's other
// side) stays as it is, rather than being cut down.
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
    const turnedShape = { ...shape, Points: turnedPoints }
    if (!fitsGrid(shapeBounds(turnedShape), layout.GridWidth, layout.GridHeight)) continue
    fitShapeInGrid(turnedShape, layout.GridWidth, layout.GridHeight)
    shape.Points = turnedShape.Points
  }
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
