import { PACK_PRICE_TYPE, PACK_ROUNDING, type IProductPack, type IProductPackComponent } from './product-packs.ts'
import { QUANTITY_LINE_SCALE, quantityAmount, unpackQuantityLine } from './quantity.ts'

// The pack pricing engine, a mirror of the backend's sales.ApplySalePacks
// (backend/sales/types/sale_packs.go, docs/PACKS_PLAN.md §3). The Go engine is authoritative and
// prices the sale; this one previews it before saving. Parity is held by the shared case file
// backend/sales/types/testdata/sale_packs_cases.json, which both test suites run.

// A slot naming one product is a deliberate offer and outranks a slot open to a whole category.
export const PACK_RELEVANCE_PRODUCT_SLOT = 5
export const PACK_RELEVANCE_CATEGORY_SLOT = 2

// A sale line as the engine reads it: Quantity packed (#core/quantity.ts), prices in cents.
export interface ISalePackLine {
  ProductID: number
  Price: number
  SubPrice?: number
  Quantity: number
  SubDivisor?: number
  PackID?: number
}

// One whole unit of a line, the grain packs are filled with. packID, packedPrice and
// applicationIndex are set once a pack takes it.
export interface IPackUnit {
  lineIndex: number
  productID: number
  unitPrice: number
  packID: number
  packedPrice: number
  applicationIndex: number
}

// One pack filled in the order: the units it took (indexes into the unit list) and their price.
export interface IPackApplication {
  pack: IProductPack
  packPrice: number
  catalogSum: number
  unitIndexes: number[]
}

export const packRelevance = (pack: IProductPack) => pack.Components.reduce((relevance, component) =>
  relevance + (component.ProductID > 0 ? PACK_RELEVANCE_PRODUCT_SLOT : PACK_RELEVANCE_CATEGORY_SLOT), 0)

// What the filled units cost as a pack, from catalogSum, the sum of their unit prices. A fixed price
// ignores the rounding. Math.trunc mirrors Go's integer division.
export function packPrice(pack: IProductPack, catalogSum: number): number {
  if (pack.PriceType === PACK_PRICE_TYPE.FIXED) return pack.PriceValue
  let price = catalogSum
  if (pack.PriceType === PACK_PRICE_TYPE.PERCENT_OFF) price = Math.trunc(catalogSum * (10000 - pack.PriceValue) / 10000)
  if (pack.PriceType === PACK_PRICE_TYPE.AMOUNT_OFF) price = catalogSum - pack.PriceValue
  // Rounding is to the whole sol: the prices are in cents.
  if (pack.RoundingMode === PACK_ROUNDING.UP) return Math.trunc((price + 99) / 100) * 100
  if (pack.RoundingMode === PACK_ROUNDING.NEAREST) return Math.trunc((price + 50) / 100) * 100
  if (pack.RoundingMode === PACK_ROUNDING.DOWN) return Math.trunc(price / 100) * 100
  return price
}

// Finds the packs in the lines: greedy and deterministic, as the Go engine. While some pack can be
// filled with a positive saving, apply the one with the highest relevance; a tie goes to the larger
// saving, then to the lower pack ID. Only whole units take part.
export function packSaleUnits(
  lines: ISalePackLine[], categoryIDsOf: (productID: number) => number[], packs: IProductPack[],
): { units: IPackUnit[], applications: IPackApplication[] } {
  const activePacks = packs.filter((pack) => pack.ss === 1)
  const units: IPackUnit[] = []
  const applications: IPackApplication[] = []
  if (!activePacks.length) return { units, applications }

  lines.forEach((line, lineIndex) => {
    for (let unit = 0; unit < unpackQuantityLine(line.Quantity).units; unit++) {
      units.push({ lineIndex, productID: line.ProductID, unitPrice: line.Price, packID: 0, packedPrice: 0, applicationIndex: 0 })
    }
  })
  // A slot takes the most expensive eligible unit first: the largest saving for the customer.
  units.sort((left, right) => right.unitPrice - left.unitPrice)

  for (let applicationIndex = 1; ; applicationIndex++) {
    let bestFill: IPackApplication | undefined
    for (const pack of activePacks) {
      const fill = fillPack(pack, units, categoryIDsOf)
      if (!fill || fill.packPrice <= 0 || fill.packPrice >= fill.catalogSum) continue
      if (!bestFill || isBetterPackFill(fill, bestFill)) bestFill = fill
    }
    if (!bestFill) break
    prorateFillPrice(bestFill, units, applicationIndex)
    applications.push(bestFill)
  }
  return { units, applications }
}

function isBetterPackFill(candidate: IPackApplication, current: IPackApplication): boolean {
  const candidateRelevance = packRelevance(candidate.pack), currentRelevance = packRelevance(current.pack)
  if (candidateRelevance !== currentRelevance) return candidateRelevance > currentRelevance
  const candidateSaving = candidate.catalogSum - candidate.packPrice, currentSaving = current.catalogSum - current.packPrice
  if (candidateSaving !== currentSaving) return candidateSaving > currentSaving
  return candidate.pack.ID < current.pack.ID
}

// Takes the free units a pack needs, product slots before category slots: a slot naming the product
// must not lose its unit to a slot that would take any product of a category.
function fillPack(pack: IProductPack, units: IPackUnit[], categoryIDsOf: (productID: number) => number[]): IPackApplication | undefined {
  const fill: IPackApplication = { pack, packPrice: 0, catalogSum: 0, unitIndexes: [] }
  const takeUnits = (component: IProductPackComponent) => {
    let missingUnits = component.Quantity
    for (let unitIndex = 0; unitIndex < units.length && missingUnits > 0; unitIndex++) {
      const unit = units[unitIndex]
      if (unit.packID > 0 || fill.unitIndexes.includes(unitIndex)) continue
      const isEligible = component.CategoryID > 0
        ? categoryIDsOf(unit.productID).includes(component.CategoryID)
        : unit.productID === component.ProductID
      if (isEligible) {
        fill.unitIndexes.push(unitIndex)
        fill.catalogSum += unit.unitPrice
        missingUnits--
      }
    }
    return missingUnits === 0
  }
  for (const isProductPass of [true, false]) {
    for (const component of pack.Components) {
      if ((component.ProductID > 0) === isProductPass && !takeUnits(component)) return undefined
    }
  }
  fill.packPrice = packPrice(pack, fill.catalogSum)
  return fill
}

// Spreads the pack price over its units in proportion to their catalog price. The cents the floors
// leave go to the most expensive unit, so the units sum to the pack price exactly.
function prorateFillPrice(fill: IPackApplication, units: IPackUnit[], applicationIndex: number) {
  let assignedTotal = 0, mostExpensiveIndex = fill.unitIndexes[0]
  for (const unitIndex of fill.unitIndexes) {
    const unit = units[unitIndex]
    unit.packID = fill.pack.ID
    unit.applicationIndex = applicationIndex
    unit.packedPrice = Math.floor(unit.unitPrice * fill.packPrice / fill.catalogSum)
    assignedTotal += unit.packedPrice
    if (unit.unitPrice > units[mostExpensiveIndex].unitPrice) mostExpensiveIndex = unitIndex
  }
  units[mostExpensiveIndex].packedPrice += fill.packPrice - assignedTotal
}

// The lines as the backend saves them (ApplySalePacks): a line whose units end up at different
// prices is split, one line per (pack, prorated price) in the order the packs were applied, then
// its unpacked units and sub-units at catalog price.
export function applySalePacks<T extends ISalePackLine>(lines: T[], categoryIDsOf: (productID: number) => number[], packs: IProductPack[]): T[] {
  const { units } = packSaleUnits(lines, categoryIDsOf, packs)
  const unitsByLine: IPackUnit[][] = lines.map(() => [])
  for (const unit of units) if (unit.packID > 0) unitsByLine[unit.lineIndex].push(unit)

  const packedLines: T[] = []
  lines.forEach((line, lineIndex) => {
    const lineUnits = unitsByLine[lineIndex].sort((left, right) => left.applicationIndex - right.applicationIndex)
    const firstSplitIndex = packedLines.length
    for (const unit of lineUnits) {
      // Units of this line taken by the same pack at the same price stay on one line.
      const lastLine = packedLines.length > firstSplitIndex ? packedLines[packedLines.length - 1] : undefined
      if (lastLine && lastLine.PackID === unit.packID && lastLine.Price === unit.packedPrice) {
        lastLine.Quantity += QUANTITY_LINE_SCALE
        continue
      }
      packedLines.push({ ...line, Quantity: QUANTITY_LINE_SCALE, Price: unit.packedPrice, PackID: unit.packID })
    }
    // What no pack took stays at catalog price: the leftover whole units and the sub-units.
    const remainderQuantity = line.Quantity - lineUnits.length * QUANTITY_LINE_SCALE
    if (remainderQuantity > 0) packedLines.push(lineUnits.length ? { ...line, Quantity: remainderQuantity } : line)
  })
  return packedLines
}

// One row of a table that shows its packs: a pack's header (pack set) or a line — the part of it no
// pack took (isPacked false), or the units one pack took (isPacked true).
export interface IPackGroupRow<T> {
  key: string
  pack?: IProductPack
  line?: T
  // The packed quantity the row shows (#core/quantity.ts).
  Quantity: number
  // In cents. A header carries the pack price.
  Amount: number
  isPacked: boolean
}

// The lines grouped as the waiter reads them: first what no pack took, at catalog price, then each
// pack application as a header ("Pack <name> = S/ 30") over the units it took. A fixed-price pack
// shows its whole price on its first row and 0 on the rest; any other pack shows each row's prorated
// share. Either way the rows add up to what the backend will charge. A line no pack touched always
// shows, even with 0 units, so the waiter can still edit it.
export function packGroupRows<T extends ISalePackLine>(
  lines: T[], categoryIDsOf: (productID: number) => number[], packs: IProductPack[],
): IPackGroupRow<T>[] {
  const { units, applications } = packSaleUnits(lines, categoryIDsOf, packs)
  const rows: IPackGroupRow<T>[] = []
  lines.forEach((line, lineIndex) => {
    const packedUnits = units.filter((unit) => unit.lineIndex === lineIndex && unit.packID > 0).length
    const remainderQuantity = unpackQuantityLine(line.Quantity - packedUnits * QUANTITY_LINE_SCALE)
    if (packedUnits && !remainderQuantity.units && !remainderQuantity.sub) return
    rows.push({
      key: `line_${lineIndex}`, line, Quantity: line.Quantity - packedUnits * QUANTITY_LINE_SCALE,
      Amount: quantityAmount(remainderQuantity, line.Price, line.SubPrice || 0), isPacked: false,
    })
  })
  applications.forEach((application, applicationIndex) => {
    rows.push({ key: `pack_${applicationIndex}`, pack: application.pack, Quantity: 0, Amount: application.packPrice, isPacked: true })
    // The pack's items in the order of their lines, one row per line however many units it gave.
    const lineIndexes = [...new Set(application.unitIndexes.map((unitIndex) => units[unitIndex].lineIndex))].sort((left, right) => left - right)
    const isFixedPrice = application.pack.PriceType === PACK_PRICE_TYPE.FIXED
    lineIndexes.forEach((lineIndex, itemIndex) => {
      const lineUnits = application.unitIndexes.map((unitIndex) => units[unitIndex]).filter((unit) => unit.lineIndex === lineIndex)
      const proratedAmount = lineUnits.reduce((amount, unit) => amount + unit.packedPrice, 0)
      rows.push({
        key: `pack_${applicationIndex}_${lineIndex}`, line: lines[lineIndex], Quantity: lineUnits.length * QUANTITY_LINE_SCALE,
        Amount: isFixedPrice ? (itemIndex === 0 ? application.packPrice : 0) : proratedAmount, isPacked: true,
      })
    })
  })
  return rows
}

// What the rows charge: their items, never the headers (each repeats its items' sum).
export const packGroupRowsTotal = <T>(rows: IPackGroupRow<T>[]) =>
  rows.reduce((total, row) => total + (row.pack ? 0 : row.Amount), 0)
