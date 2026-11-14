// Mirrors backend/dining/types (docs/DINING_PLAN.md): the restaurant's layouts, shared by the
// dining pages.

export const SHAPE_KIND = {
  TABLE: 1,
  RECT: 2,
  LINE: 3,
  LABEL: 4,
  STATION: 5,
  PICKUP: 6,
} as const

export const TABLE_OUTLINE = { RECT: 1, ROUND: 2 } as const

// One element of the plan. Points are grid points, x then y: a line's two ends {x1,y1,x2,y2}; any
// other shape's four corners {x1,y1,…,x4,y4}, clockwise from its content's top-left corner, so the
// corner order is its orientation (backend LayoutShape).
export interface ILayoutShape {
  ShapeID: number   // < 0 until the layout is saved: the backend assigns the real one
  Kind: number      // SHAPE_KIND
  Points: number[]
  Label?: string
  Seats?: number
  Outline?: number  // TABLE_OUTLINE, tables only
  StationID?: number
}

// A prep station. A product goes to the first station naming it, else the first naming one of its
// categories, else the default one.
export interface IDiningStation {
  ID: number
  Name: string
  ProductIDs?: number[]
  CategoryIDs?: number[]
  IsDefault?: boolean
}

export interface IDiningLayout {
  ID: number
  Name: string
  WarehouseID: number
  GridWidth: number
  GridHeight: number
  Shapes?: ILayoutShape[]
  Stations?: IDiningStation[]
  QRRequiresConfirmation?: boolean
  QRVersion?: number
  WaitWarnMinutes: number
  WaitLateMinutes: number
  NextShapeID?: number
  ss: number
  upd: number
}

// A point on the plan in cells, fractional under the pointer; a box by its two corners.
export interface IGridPoint { x: number, y: number }
export interface IGridRect { x1: number, y1: number, x2: number, y2: number }

export const isLineShape = (shape: ILayoutShape) => shape.Kind === SHAPE_KIND.LINE

// The box around a shape's points: a block's cells, a line's span.
export function shapeBounds(shape: ILayoutShape): IGridRect {
  const xs = shape.Points.filter((_, index) => index % 2 === 0), ys = shape.Points.filter((_, index) => index % 2 === 1)
  return { x1: Math.min(...xs), y1: Math.min(...ys), x2: Math.max(...xs), y2: Math.max(...ys) }
}

const LINE_HIT_DISTANCE = 0.5 // cells

function distanceToSegment(point: IGridPoint, from: IGridPoint, to: IGridPoint): number {
  const segmentX = to.x - from.x, segmentY = to.y - from.y
  const lengthSquared = segmentX * segmentX + segmentY * segmentY
  const along = lengthSquared ? Math.max(0, Math.min(1, ((point.x - from.x) * segmentX + (point.y - from.y) * segmentY) / lengthSquared)) : 0
  return Math.hypot(point.x - (from.x + along * segmentX), point.y - (from.y + along * segmentY))
}

// The topmost shape under the point: the last drawn wins, as on screen. A line is hit within half a cell.
export function shapeAt(shapes: ILayoutShape[], point: IGridPoint): ILayoutShape | undefined {
  for (let index = shapes.length - 1; index >= 0; index--) {
    const shape = shapes[index]
    if (isLineShape(shape)) {
      const [x1, y1, x2, y2] = shape.Points
      if (distanceToSegment(point, { x: x1, y: y1 }, { x: x2, y: y2 }) <= LINE_HIT_DISTANCE) return shape
      continue
    }
    const bounds = shapeBounds(shape)
    if (point.x >= bounds.x1 && point.x < bounds.x2 && point.y >= bounds.y1 && point.y < bounds.y2) return shape
  }
  return undefined
}

// The cells a non-line shape covers, and how many quarter turns (clockwise) its content is turned.
export interface IShapeRect { x: number, y: number, w: number, h: number, quarterTurns: number }

// A rectangle's corners clockwise from the top-left (y grows downward), as [x, y] pairs.
const clockwiseCorners = ({ x, y, w, h }: IShapeRect) => [[x, y], [x + w, y], [x + w, y + h], [x, y + h]]

export function shapeRect(shape: ILayoutShape): IShapeRect {
  const points = shape.Points
  const xs = [points[0], points[2], points[4], points[6]], ys = [points[1], points[3], points[5], points[7]]
  const x = Math.min(...xs), y = Math.min(...ys)
  const rect = { x, y, w: Math.max(...xs) - x, h: Math.max(...ys) - y, quarterTurns: 0 }
  // The corner the list starts at is how far the shape is turned.
  rect.quarterTurns = Math.max(0, clockwiseCorners(rect).findIndex(([cornerX, cornerY]) => cornerX === points[0] && cornerY === points[1]))
  return rect
}

// The Points of a rectangle turned quarterTurns times: its corners, starting that many corners later.
export function rectPoints(rect: IShapeRect): number[] {
  const corners = clockwiseCorners(rect)
  const firstCorner = ((rect.quarterTurns % 4) + 4) % 4
  return [...corners.slice(firstCorner), ...corners.slice(0, firstCorner)].flat()
}

// ── Table orders (backend TableOrder, docs/DINING_PLAN.md §2) ────────────────────────────────────

export const LINE_STATUS = { CANCELLED: 0, TO_CONFIRM: 1, ORDERED: 2, READY: 3, SERVED: 4 } as const

export const lineStatusLabels: Record<number, string> = {
  [LINE_STATUS.CANCELLED]: 'Cancelled|Anulado',
  [LINE_STATUS.TO_CONFIRM]: 'To confirm|Por confirmar',
  [LINE_STATUS.ORDERED]: 'In the kitchen|En cocina',
  [LINE_STATUS.READY]: 'Ready|Listo',
  [LINE_STATUS.SERVED]: 'Served|Servido',
}

export interface ITableOrderLine {
  LineID: number
  Round: number
  ProductID: number
  PresentationID?: number
  Quantity: number      // packed (#core/quantity.ts)
  SubDivisor?: number
  Price: number         // cents, soles, fixed when ordered
  SubPrice?: number
  StationID: number
  Note?: string
  FromQR?: boolean
  Status: number        // LINE_STATUS
  OrderedAt?: number    // SUnixTime
  ReadyAt?: number
  ServedAt?: number
}

export interface ITableOrder {
  ID: number
  LayoutID: number
  TableShapeID: number
  WarehouseID: number
  Lines: ITableOrderLine[]
  Guests?: number
  WaiterID?: number
  ss: number
  upd: number
  Created?: number
}

// SUnixTime ticks are 2 seconds.
const TICKS_PER_MINUTE = 30

// How long the table has waited for its kitchen: from the oldest line still in the kitchen, 0 when none is.
export function tableWaitTicks(order: ITableOrder | undefined, nowTime: number): number {
  const orderedTimes = (order?.Lines || []).filter((line) => line.Status === LINE_STATUS.ORDERED).map((line) => line.OrderedAt || nowTime)
  return orderedTimes.length ? nowTime - Math.min(...orderedTimes) : 0
}

export const waitMinutes = (ticks: number) => Math.floor(ticks / TICKS_PER_MINUTE)

// A table's state on the floor map (§9). The first rule that matches wins.
export const TABLE_STATE = { FREE: 0, EATING: 1, ORDERED: 2, WAITING: 3, TO_CONFIRM: 4, READY: 5, LATE: 6 } as const

export const tableStateLabels: Record<number, string> = {
  [TABLE_STATE.FREE]: 'Free|Libre',
  [TABLE_STATE.EATING]: 'Eating|Comiendo',
  [TABLE_STATE.ORDERED]: 'Ordered|Pedido en cocina',
  [TABLE_STATE.WAITING]: 'Waiting|Esperando',
  [TABLE_STATE.TO_CONFIRM]: 'QR to confirm|QR por confirmar',
  [TABLE_STATE.READY]: 'Ready to pick up|Listo para recoger',
  [TABLE_STATE.LATE]: 'Late|Retrasado',
}

export function tableState(order: ITableOrder | undefined, layout: IDiningLayout, nowTime: number): number {
  if (!order) return TABLE_STATE.FREE
  const lineStatuses = new Set(order.Lines.map((line) => line.Status))
  const waitedMinutes = waitMinutes(tableWaitTicks(order, nowTime))
  if (lineStatuses.has(LINE_STATUS.ORDERED) && waitedMinutes >= layout.WaitLateMinutes) return TABLE_STATE.LATE
  if (lineStatuses.has(LINE_STATUS.READY)) return TABLE_STATE.READY
  if (lineStatuses.has(LINE_STATUS.TO_CONFIRM)) return TABLE_STATE.TO_CONFIRM
  if (lineStatuses.has(LINE_STATUS.ORDERED) && waitedMinutes >= layout.WaitWarnMinutes) return TABLE_STATE.WAITING
  if (lineStatuses.has(LINE_STATUS.ORDERED)) return TABLE_STATE.ORDERED
  return TABLE_STATE.EATING
}

export const shapeKindLabels: Record<number, string> = {
  [SHAPE_KIND.TABLE]: 'Table|Mesa',
  [SHAPE_KIND.RECT]: 'Block|Bloque',
  [SHAPE_KIND.LINE]: 'Wall / divider|Pared / divisor',
  [SHAPE_KIND.LABEL]: 'Text|Texto',
  [SHAPE_KIND.STATION]: 'Prep station|Estación de preparación',
  [SHAPE_KIND.PICKUP]: 'Pickup point|Punto de recojo',
}
