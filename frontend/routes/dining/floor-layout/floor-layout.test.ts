import { describe, expect, it } from 'vitest'
import { SHAPE_KIND, TABLE_OUTLINE, rectPoints, shapeAt, shapeRect, type IDiningLayout } from '../dining.ts'
import {
  applyShapeGeometryFields, duplicateShapes, fitShapeInGrid, makeEmptyLayout, makeShape, moveLineEnd, moveShapes,
  nextTableLabel, nextTemporaryShapeID, overlappingTableIDs, resizeShapeTo, rotateShapes, setDefaultStation,
  shapeGeometryFields, shapeIDsInRect, stationDeleteBlocker,
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

  it('moves a group by whole cells without letting any shape leave the grid', () => {
    const layout = makeLayout()
    moveShapes(layout, [1, 2], { x: 20, y: -0.6 })
    // M7 ends at x 8: the group can go 12 cells right; -0.6 rounds to -1.
    expect(layout.Shapes!.map((shape) => shape.Points.slice(0, 2))).toEqual([[14, 1], [18, 1], [0, 6]])
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
    resizeShapeTo(table, { x: 5.2, y: 1 }, 20, 10)
    expect(shapeRect(table)).toEqual({ x: 2, y: 2, w: 3, h: 1, quarterTurns: 1 })
    const line = layout.Shapes![2]
    moveLineEnd(line, 2, { x: 0.2, y: 6.1 }, 20, 10) // would collapse the line: ignored
    expect(line.Points).toEqual([0, 6, 20, 6])
    moveLineEnd(line, 2, { x: 9.6, y: 2.4 }, 20, 10)
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

  it('turns a wall about its first end and shifts it back in whole: it never collapses', () => {
    const layout = makeLayout()
    // Against the left edge, pointing down: turned, it would point left, off the grid.
    layout.Shapes![2].Points = [0, 0, 0, 5]
    rotateShapes(layout, [3])
    expect(layout.Shapes![2].Points).toEqual([5, 0, 0, 0])
    // 20 long in a grid 10 high: it cannot stand up, so it stays as it is.
    layout.Shapes![2].Points = [0, 6, 20, 6]
    rotateShapes(layout, [3])
    expect(layout.Shapes![2].Points).toEqual([0, 6, 20, 6])
  })

  it('gives a wall typed with both ends on one point one cell of length', () => {
    const line = { ShapeID: 3, Kind: SHAPE_KIND.LINE, Points: [4, 4, 0, 0] }
    applyShapeGeometryFields(line, { ...shapeGeometryFields(line), X2: 4, Y2: 4 }, 20, 10)
    expect(line.Points).toEqual([4, 4, 5, 4])
    const atEdge = { ShapeID: 4, Kind: SHAPE_KIND.LINE, Points: [20, 3, 20, 3] }
    fitShapeInGrid(atEdge, 20, 10)
    expect(atEdge.Points).toEqual([20, 3, 19, 3])
  })

  it('writes typed block coordinates back keeping the turn, inside the grid', () => {
    const block = { ShapeID: 5, Kind: SHAPE_KIND.RECT, Points: rectPoints({ x: 1, y: 1, w: 2, h: 3, quarterTurns: 3 }) }
    applyShapeGeometryFields(block, { ...shapeGeometryFields(block), X: 19, W: Number.NaN }, 20, 10)
    expect(shapeRect(block)).toEqual({ x: 19, y: 1, w: 1, h: 3, quarterTurns: 3 })
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
