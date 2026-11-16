import { describe, expect, it } from 'vitest'
import { SHAPE_KIND, TABLE_OUTLINE, rectPoints, shapeAt, shapeRect, type IDiningLayout } from '../dining.ts'
import {
  MIN_GRID_SIZE, duplicateShapes, editorCanvasRegion, makeEmptyLayout, makeShape, moveLineEnd,
  moveShapes, nextTableLabel, nextTemporaryShapeID, overlappingTableIDs, resizeShapeTo, rotateShapes,
  setDefaultStation, shapeIDsInRect, stationDeleteBlocker, trimGridToShapes,
} from './floor-layout.ts'

// An unturned block of w × h cells from (x, y).
const cells = (x: number, y: number, w: number, h: number) => rectPoints({ x, y, w, h, quarterTurns: 0 })

const makeLayout = (): IDiningLayout => ({
  ...makeEmptyLayout(), GridWidth: 20, GridHeight: 10,
  Shapes: [
    { ShapeID: 1, Kind: SHAPE_KIND.TABLE, Points: cells(2, 2, 2, 2), Label: 'M1', Seats: 4, Outline: TABLE_OUTLINE.RECT },
    { ShapeID: 2, Kind: SHAPE_KIND.TABLE, Points: cells(6, 2, 2, 2), Label: 'M7', Seats: 2, Outline: TABLE_OUTLINE.ROUND },
    { ShapeID: 3, Kind: SHAPE_KIND.LINE, Points: [0, 6, 20, 6] },
  ],
})

describe('shape points', () => {
  it('lists the corners clockwise from the content top-left, the turn being where the list starts', () => {
    expect(cells(2, 3, 4, 1)).toEqual([2, 3, 6, 3, 6, 4, 2, 4])
    const turned = rectPoints({ x: 2, y: 3, w: 4, h: 1, quarterTurns: 1 })
    expect(turned).toEqual([6, 3, 6, 4, 2, 4, 2, 3])
    expect(shapeRect({ ShapeID: 1, Kind: SHAPE_KIND.RECT, Points: turned })).toEqual({ x: 2, y: 3, w: 4, h: 1, quarterTurns: 1 })
  })
})

describe('floor layout geometry', () => {
  it('hits the topmost shape, and a line within half a cell', () => {
    const layout = makeLayout()
    expect(shapeAt(layout.Shapes!, { x: 3.5, y: 3.9 })?.ShapeID).toBe(1)
    expect(shapeAt(layout.Shapes!, { x: 4, y: 3 })).toBeUndefined()
    expect(shapeAt(layout.Shapes!, { x: 10, y: 6.4 })?.ShapeID).toBe(3)
  })

  it('selects with a rubber band only the shapes it covers whole', () => {
    expect(shapeIDsInRect(makeLayout().Shapes!, { x1: 1, y1: 1, x2: 5, y2: 7 })).toEqual([1])
  })

  it('moves a group by whole cells anywhere on the open sheet, even below 0', () => {
    const layout = makeLayout()
    moveShapes(layout, [1, 2], { x: 20, y: -0.6 })
    // The grid does not hold the group back; -0.6 rounds to -1.
    expect(layout.Shapes!.map((shape) => shape.Points.slice(0, 2))).toEqual([[22, 1], [26, 1], [0, 6]])
    moveShapes(layout, [1, 2], { x: -30, y: 0 })
    expect(layout.Shapes!.map((shape) => shape.Points.slice(0, 2))).toEqual([[-8, 1], [-4, 1], [0, 6]])
  })

  it('moves a line with its two points', () => {
    const layout = makeLayout()
    moveShapes(layout, [3], { x: 0, y: 2 })
    expect(layout.Shapes![2].Points).toEqual([0, 8, 20, 8])
  })

  it('resizes to the corner, keeping the turn, and moves a line end to a grid point', () => {
    const layout = makeLayout()
    const table = layout.Shapes![0]
    table.Points = rectPoints({ x: 2, y: 2, w: 2, h: 2, quarterTurns: 1 })
    resizeShapeTo(table, { x: 5.2, y: 1 })
    expect(shapeRect(table)).toEqual({ x: 2, y: 2, w: 3, h: 1, quarterTurns: 1 })
    const line = layout.Shapes![2]
    moveLineEnd(line, 2, { x: 0.2, y: 6.1 }) // would collapse the line: ignored
    expect(line.Points).toEqual([0, 6, 20, 6])
    moveLineEnd(line, 2, { x: 9.6, y: 2.4 })
    expect(line.Points).toEqual([0, 6, 10, 2])
  })

  it('names and numbers new shapes', () => {
    const layout = makeLayout()
    expect(nextTableLabel(layout.Shapes!)).toBe('M8')
    expect(nextTemporaryShapeID(layout.Shapes!)).toBe(-1)
    const table = makeShape('table-round', layout, { x: 1, y: 1 })
    expect([table.Kind, table.Outline, table.Label, table.ShapeID]).toEqual([SHAPE_KIND.TABLE, TABLE_OUTLINE.ROUND, 'M8', -1])
    expect(table.Points).toEqual(cells(1, 1, 3, 3))
    expect(makeShape('pickup', layout, { x: 1, y: 1 }).StationID).toBe(1)
  })

  it('duplicates one cell away, with new IDs, table names and points of their own', () => {
    const layout = makeLayout()
    const copyIDs = duplicateShapes(layout, [1, 2])
    expect(copyIDs).toEqual([-1, -2])
    const copies = layout.Shapes!.slice(3)
    expect(copies.map((copy) => [copy.Label, ...copy.Points.slice(0, 2)])).toEqual([['M8', 3, 3], ['M9', 7, 3]])
    expect(layout.Shapes![0].Points.slice(0, 2)).toEqual([2, 2])
  })

  it('turns a block about its center, its corner list starting one corner later', () => {
    const layout = makeLayout()
    layout.Shapes![0].Points = cells(2, 2, 4, 2)
    rotateShapes(layout, [1])
    expect(shapeRect(layout.Shapes![0])).toEqual({ x: 3, y: 1, w: 2, h: 4, quarterTurns: 1 })
    rotateShapes(layout, [1])
    rotateShapes(layout, [1])
    rotateShapes(layout, [1])
    expect(layout.Shapes![0].Points).toEqual(cells(2, 2, 4, 2))
  })

  it('turns a wall about its first end', () => {
    const layout = makeLayout()
    layout.Shapes![2].Points = [0, 0, 0, 5]
    rotateShapes(layout, [3])
    expect(layout.Shapes![2].Points).toEqual([0, 0, -5, 0])
  })

  it('makes a new wall 10 cells long', () => {
    expect(makeShape('line', makeLayout(), { x: 3, y: 4 }).Points).toEqual([3, 4, 13, 4])
  })

  it('draws the viewport, slid only as far as the shapes and their margin need', () => {
    // Shapes span x 0..20 and y 2..6: with their margin, -1..21 and 1..7.
    const shapes = makeLayout().Shapes!
    // A 30 × 10 viewport anchored at 0 slides one cell left to give the wall its margin.
    expect(editorCanvasRegion(shapes, { x: 0, y: 0 }, 30, 10)).toEqual({ x1: -1, y1: 0, x2: 29, y2: 10 })
    // Anchored far away, it slides back just enough to reach them: the space they left is dropped.
    expect(editorCanvasRegion(shapes, { x: -40, y: 5 }, 30, 10)).toEqual({ x1: -9, y1: 1, x2: 21, y2: 11 })
    // Narrower than the shapes: exactly what they need, so the plan scrolls.
    expect(editorCanvasRegion(shapes, { x: 0, y: 0 }, 12, 10)).toEqual({ x1: -1, y1: 0, x2: 21, y2: 10 })
    expect(editorCanvasRegion([], { x: 3, y: 4 }, 12, 10)).toEqual({ x1: 3, y1: 4, x2: 15, y2: 14 })
  })

  it('stores the used area with a margin on every side', () => {
    const layout = makeLayout()
    moveShapes(layout, [1, 2, 3], { x: -6, y: 3 })
    expect(trimGridToShapes(layout)).toEqual({ x: 7, y: -4 })
    // The wall goes to x 1 and M1 to the top margin, keeping their distance.
    expect(layout.Shapes![2].Points).toEqual([1, 5, 21, 5])
    expect(layout.Shapes![0].Points.slice(0, 2)).toEqual([3, 1])
    expect([layout.GridWidth, layout.GridHeight]).toEqual([22, 8])
  })

  it('keeps an empty layout at the smallest grid', () => {
    const layout = { ...makeEmptyLayout(), GridWidth: 40, GridHeight: 30 }
    trimGridToShapes(layout)
    expect([layout.GridWidth, layout.GridHeight]).toEqual([MIN_GRID_SIZE, MIN_GRID_SIZE])
  })

  it('keeps one default station, and the stations the plan still marks', () => {
    const layout = makeLayout()
    layout.Stations!.push({ ID: 2, Name: 'Bar' })
    layout.Shapes!.push(makeShape('pickup', layout, { x: 1, y: 8 }))
    layout.Shapes![3].StationID = 2
    expect(stationDeleteBlocker(layout, 1)).toContain('default')
    expect(stationDeleteBlocker(layout, 2)).toContain('mark')
    layout.Shapes!.pop()
    expect(stationDeleteBlocker(layout, 2)).toBe('')
    setDefaultStation(layout.Stations!, 2)
    expect(layout.Stations!.map((station) => station.IsDefault)).toEqual([false, true])
  })

  it('warns about overlapping tables', () => {
    const layout = makeLayout()
    expect(overlappingTableIDs(layout.Shapes!)).toEqual([])
    layout.Shapes![1].Points = cells(3, 2, 2, 2)
    expect(overlappingTableIDs(layout.Shapes!).sort()).toEqual([1, 2])
  })
})
