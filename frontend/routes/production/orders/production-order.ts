import { totalSubUnits, unpackQuantityLine } from '#core/quantity.ts'
import type { IProductPresentation } from '#services/production/products.svelte.ts'
import { PRODUCTION_ORDER_KIND, PRODUCTION_ORDER_STATUS, type IProductionOrder, type IProductionOrderLabor, type IProductionOrderLine, type IProductionOrderSupply, type IProductionTemplate, type IProductionTemplateLine } from '#services/production/production-orders.ts'
import { BASIS_POINTS_WHOLE, type IProductionFields } from '#routes/production/products/product-production.ts'

// The pure rules of the production orders page (docs/MANUFACTURING_PLAN.md §4.2, §6.2, §12): the size
// grid both orders and templates edit, template ↔ order conversion, the client-side recipe explosion,
// the consumption a receipt proposes and the planned-vs-actual figures of the Resumen tab.

// A made product or a semi-finished supply as these rules read it: its recipe and its sizes.
export type IRecipeProduct = IProductionFields & {
  ID: number
  Name: string
  Presentations?: IProductPresentation[]
}

// What an order line is before any receipt: a (product, presentation) and its planned units.
export type IPlannedLine = Pick<IProductionOrderLine, 'ProductID' | 'PresentationID' | 'PlannedUnits'>

// ---------------------------------------------------------------------------------------------------
// Size grid: one row per product (a template line, IProductionTemplateLine), one column per size.
// An order draft edits its lines in this same shape, with Run.Units holding the planned units.
// ---------------------------------------------------------------------------------------------------

// The column of the products with no active presentation: they are planned on presentation 0.
export const NO_PRESENTATION_COLUMN = ''

export interface ISizeColumn {
  // The presentation name ("38", "M"), or NO_PRESENTATION_COLUMN.
  key: string
  label: string
}

export const activePresentationsOf = (product: IRecipeProduct | undefined): IProductPresentation[] =>
  (product?.Presentations || []).filter((presentation) => presentation.ss > 0)

// The grid's columns: every active presentation name of the rows' products, in the order they first
// appear (sizes are entered in order: 36…41, S M L, so no sort is imposed), plus the no-presentation
// column when some product has none. Products share a column by presentation name.
export const sizeGridColumns = (productRuns: IProductionTemplateLine[], productsByID: Map<number, IRecipeProduct>): ISizeColumn[] => {
  const columns: ISizeColumn[] = []
  const seenKeys = new Set<string>()
  for (const productRun of productRuns) {
    const product = productsByID.get(productRun.ProductID)
    // A row whose product is not picked yet (or no longer made) adds no column.
    if (!product) continue
    const presentations = activePresentationsOf(product)
    const keys = presentations.length ? presentations.map((presentation) => presentation.nm) : [NO_PRESENTATION_COLUMN]
    for (const key of keys) {
      if (seenKeys.has(key)) continue
      seenKeys.add(key)
      columns.push({ key, label: key === NO_PRESENTATION_COLUMN ? 'Units|Unidades' : key })
    }
  }
  return columns
}

// The presentation of the product a column stands for, or undefined when the product has no such
// size, or is not picked yet (its cell is disabled).
export const presentationIDAtColumn = (product: IRecipeProduct | undefined, columnKey: string): number | undefined => {
  if (!product) return undefined
  const presentations = activePresentationsOf(product)
  if (!presentations.length) return columnKey === NO_PRESENTATION_COLUMN ? 0 : undefined
  return presentations.find((presentation) => presentation.nm === columnKey)?.id
}

export const runUnitsOf = (productRun: IProductionTemplateLine, presentationID: number): number =>
  productRun.Run.find((sizeUnits) => sizeUnits.PresentationID === presentationID)?.Units || 0

// Sets the units of one size; 0 or less removes it from the run.
export const setRunUnits = (productRun: IProductionTemplateLine, presentationID: number, units: number) => {
  const otherSizes = productRun.Run.filter((sizeUnits) => sizeUnits.PresentationID !== presentationID)
  productRun.Run = units > 0 ? [...otherSizes, { PresentationID: presentationID, Units: Math.floor(units) }] : otherSizes
}

export const runTotalUnits = (productRun: IProductionTemplateLine): number =>
  productRun.Run.reduce((total, sizeUnits) => total + sizeUnits.Units, 0)

// Order lines grouped into grid rows, products in the order they first appear.
export const orderLinesToProductRuns = (lines: IPlannedLine[]): IProductionTemplateLine[] => {
  const runsByProductID = new Map<number, IProductionTemplateLine>()
  for (const line of lines) {
    const productRun = runsByProductID.get(line.ProductID) || { ProductID: line.ProductID, Run: [] }
    productRun.Run.push({ PresentationID: line.PresentationID, Units: line.PlannedUnits })
    runsByProductID.set(line.ProductID, productRun)
  }
  return [...runsByProductID.values()]
}

// Grid rows back to order lines, each size × multiplier (the series count for a template; 1 for an
// order draft's own grid). Sizes with no units and rows with no product picked are left out.
export const productRunsToOrderLines = (productRuns: IProductionTemplateLine[], multiplier = 1): IProductionOrderLine[] =>
  productRuns.filter((productRun) => productRun.ProductID > 0).flatMap((productRun) => productRun.Run
    .filter((sizeUnits) => sizeUnits.Units > 0)
    .map((sizeUnits) => ({
      ProductID: productRun.ProductID, PresentationID: sizeUnits.PresentationID, PlannedUnits: sizeUnits.Units * multiplier,
      ProducedUnits: 0, ScrappedUnits: 0, Value: 0, CostSource: 0,
    })))

// "Nueva orden desde plantilla" (§4.2): one line per product and size, PlannedUnits = Run.Units × series.
export const expandTemplateToOrderLines = (templateLines: IProductionTemplateLine[], series: number): IProductionOrderLine[] =>
  productRunsToOrderLines(templateLines, Math.max(Math.floor(series), 1))

// "Guardar como plantilla" (§4.2): the order's lines as one series, units ÷ series (series 1 when the
// order has none). A size that does not divide evenly is rounded, never below 1 unit.
export const orderLinesToTemplateLines = (lines: IPlannedLine[], series: number): IProductionTemplateLine[] => {
  const seriesDivisor = series > 0 ? series : 1
  return orderLinesToProductRuns(lines.filter((line) => line.PlannedUnits > 0)).map((productRun) => ({
    ProductID: productRun.ProductID,
    Run: productRun.Run.map((sizeUnits) => ({
      PresentationID: sizeUnits.PresentationID, Units: Math.max(Math.round(sizeUnits.Units / seriesDivisor), 1),
    })),
  }))
}

// ---------------------------------------------------------------------------------------------------
// Explosion: the mirror of the backend's ExplodeProductionOrder, for estimates before a release.
// ---------------------------------------------------------------------------------------------------

export interface IExplodedSupply {
  SupplyID: number
  SubDivisor: number
  // Sub-units of SubDivisor, waste and adjustment included.
  Planned: number
}

export interface IExplodedLabor {
  LaborTypeID: number
  PlannedMinutes: number
}

// planned[s] = ceil( Σ_lines units × recipe.quantity[s] ÷ recipe.yield × (1 + waste[s]) × (1 + adjustment) )
// labor[t]   = round( Σ_lines units × recipe.minutes[t] ÷ recipe.yield )
// A product with a standard cost and no recipe asks nothing. Sorted by ID, as the backend's.
export const explodeProductionLines = (
  lines: Pick<IPlannedLine, 'ProductID' | 'PlannedUnits'>[], adjustmentBasisPoints: number, productsByID: Map<number, IRecipeProduct>,
): { supplies: IExplodedSupply[], labor: IExplodedLabor[] } => {
  const adjustmentFactor = 1 + (adjustmentBasisPoints || 0) / BASIS_POINTS_WHOLE
  const plannedBySupplyID = new Map<number, { subDivisor: number, subUnits: number }>()
  const minutesByLaborTypeID = new Map<number, number>()
  for (const line of lines) {
    const product = productsByID.get(line.ProductID)
    if (!product) continue
    const recipeYield = Math.max(product.ProductionYieldPortions || 0, 1)
    for (const recipeSupply of product.ProductionSupplies || []) {
      if (!recipeSupply.SupplyID) continue
      const recipeSubUnits = totalSubUnits(unpackQuantityLine(recipeSupply.Quantity), recipeSupply.SubDivisor)
      const wasteFactor = 1 + (recipeSupply.WasteBasisPoints || 0) / BASIS_POINTS_WHOLE
      const planned = plannedBySupplyID.get(recipeSupply.SupplyID) || { subDivisor: recipeSupply.SubDivisor, subUnits: 0 }
      planned.subUnits += line.PlannedUnits * recipeSubUnits / recipeYield * wasteFactor * adjustmentFactor
      plannedBySupplyID.set(recipeSupply.SupplyID, planned)
    }
    for (const recipeLabor of product.ProductionLabor || []) {
      if (!recipeLabor.LaborTypeID) continue
      minutesByLaborTypeID.set(recipeLabor.LaborTypeID,
        (minutesByLaborTypeID.get(recipeLabor.LaborTypeID) || 0) + line.PlannedUnits * recipeLabor.Minutes / recipeYield)
    }
  }

  const supplies: IExplodedSupply[] = []
  for (const supplyID of [...plannedBySupplyID.keys()].sort((leftID, rightID) => leftID - rightID)) {
    const planned = plannedBySupplyID.get(supplyID)!
    // The tolerance keeps a float sum like 1500.0000000002 from asking a whole extra sub-unit.
    const plannedSubUnits = Math.ceil(planned.subUnits - 1e-6)
    if (plannedSubUnits > 0) supplies.push({ SupplyID: supplyID, SubDivisor: planned.subDivisor, Planned: plannedSubUnits })
  }
  const labor = [...minutesByLaborTypeID.keys()].sort((leftID, rightID) => leftID - rightID)
    .map((laborTypeID) => ({ LaborTypeID: laborTypeID, PlannedMinutes: Math.round(minutesByLaborTypeID.get(laborTypeID)!) }))
  return { supplies, labor }
}

// ---------------------------------------------------------------------------------------------------
// Receipts (§6.2)
// ---------------------------------------------------------------------------------------------------

export const totalPlannedUnits = (lines: IPlannedLine[]): number =>
  lines.reduce((total, line) => total + Math.max(line.PlannedUnits, 0), 0)

// One (product, presentation) of a receipt being entered.
export interface IReceiptEntryLine {
  ProductID: number
  PresentationID: number
  Units: number
  Scrapped: number
}

// One supply of a receipt being entered, in sub-units of the order supply's divisor.
export interface IReceiptEntryConsumption {
  SupplyID: number
  SubDivisor: number
  SubUnits: number
}

// Every line of the order, with nothing received yet.
export const makeReceiptEntryLines = (lines: IPlannedLine[]): IReceiptEntryLine[] =>
  lines.map((line) => ({ ProductID: line.ProductID, PresentationID: line.PresentationID, Units: 0, Scrapped: 0 }))

// The consumption a receipt proposes per supply:
// - a normal receipt: Planned × (good + scrapped) ÷ total planned units, rounded;
// - the last one ("Completar"): the remainder Planned − Consumed, never below 0.
// Scrapped units count: they used the supplies too.
export const proposeReceiptConsumptions = (
  orderSupplies: IProductionOrderSupply[], orderLines: IPlannedLine[], receiptLines: IReceiptEntryLine[], isCompleting: boolean,
): IReceiptEntryConsumption[] => {
  const plannedUnits = totalPlannedUnits(orderLines)
  const receivedUnits = receiptLines.reduce((total, line) => total + Math.max(line.Units, 0) + Math.max(line.Scrapped, 0), 0)
  return orderSupplies.map((orderSupply) => ({
    SupplyID: orderSupply.SupplyID,
    SubDivisor: orderSupply.SubDivisor,
    SubUnits: isCompleting
      ? Math.max(orderSupply.Planned - (orderSupply.Consumed || 0), 0)
      : plannedUnits > 0 ? Math.round(orderSupply.Planned * receivedUnits / plannedUnits) : 0,
  }))
}

// The first reason the server would refuse the receipt, or undefined.
export const findReceiptError = (receiptLines: IReceiptEntryLine[], consumptions: IReceiptEntryConsumption[]): string | undefined => {
  if (receiptLines.some((line) => line.Units < 0 || line.Scrapped < 0)) {
    return 'Received units cannot be negative.|Las unidades del ingreso no pueden ser negativas.'
  }
  if (!receiptLines.some((line) => line.Units > 0)) {
    return 'A receipt needs at least one good unit.|El ingreso necesita al menos una unidad buena.'
  }
  if (consumptions.some((consumption) => consumption.SubUnits < 0)) {
    return 'A consumption cannot be negative.|El consumo de un insumo no puede ser negativo.'
  }
  return undefined
}

// ---------------------------------------------------------------------------------------------------
// Progress and the Resumen tab (§12)
// ---------------------------------------------------------------------------------------------------

// Good units received over planned units, as a whole percent. Over-production goes past 100.
export const orderProgressPercent = (lines: IProductionOrderLine[]): number => {
  const plannedUnits = totalPlannedUnits(lines)
  if (!plannedUnits) return 0
  const producedUnits = lines.reduce((total, line) => total + (line.ProducedUnits || 0), 0)
  return Math.round(producedUnits * 100 / plannedUnits)
}

// Units per product, its sizes summed: "24x Zapato Lino" on a board card.
export const unitsByProduct = (lines: IPlannedLine[]): { ProductID: number, Units: number }[] =>
  orderLinesToProductRuns(lines).map((productRun) => ({ ProductID: productRun.ProductID, Units: runTotalUnits(productRun) }))

export interface IProductSummaryRow {
  ProductID: number
  PlannedUnits: number
  ProducedUnits: number
  ScrappedUnits: number
  // Produced ÷ planned, percent; undefined with nothing planned.
  yieldPercent: number | undefined
  // The recipe's estimate (or the standard cost) of one unit, PEN cents; undefined without a cost source.
  estimatedUnitCost: number | undefined
  // What the receipts booked per good unit: Σ Value ÷ produced; undefined with nothing produced.
  bookedUnitCost: number | undefined
}

// Planned vs produced per product, its sizes summed, with the estimated vs booked unit cost.
export const summarizeOrderProducts = (lines: IProductionOrderLine[], estimatedUnitCostByProductID: Map<number, number>): IProductSummaryRow[] => {
  const rowsByProductID = new Map<number, IProductSummaryRow & { value: number }>()
  for (const line of lines) {
    const row = rowsByProductID.get(line.ProductID) || {
      ProductID: line.ProductID, PlannedUnits: 0, ProducedUnits: 0, ScrappedUnits: 0, value: 0,
      yieldPercent: undefined, estimatedUnitCost: estimatedUnitCostByProductID.get(line.ProductID), bookedUnitCost: undefined,
    }
    row.PlannedUnits += line.PlannedUnits || 0
    row.ProducedUnits += line.ProducedUnits || 0
    row.ScrappedUnits += line.ScrappedUnits || 0
    row.value += line.Value || 0
    rowsByProductID.set(line.ProductID, row)
  }
  return [...rowsByProductID.values()].map(({ value, ...row }) => ({
    ...row,
    yieldPercent: row.PlannedUnits > 0 ? row.ProducedUnits * 100 / row.PlannedUnits : undefined,
    bookedUnitCost: row.ProducedUnits > 0 ? value / row.ProducedUnits : undefined,
  }))
}

export interface ISupplySummaryRow {
  SupplyID: number
  SubDivisor: number
  Planned: number
  Consumed: number
  // The share of Planned that what was made (good + scrapped) should have used.
  expectedForOutput: number
  // Consumed ÷ expectedForOutput, percent: above 100 used more than the plan for what was made.
  usagePercent: number | undefined
}

// Planned vs consumed per supply. The fair comparison of a partly received order is against the
// planned share of what was made, not the whole plan.
export const summarizeOrderSupplies = (order: Pick<IProductionOrder, 'Lines' | 'Supplies'>): ISupplySummaryRow[] => {
  const lines = order.Lines || []
  const plannedUnits = totalPlannedUnits(lines)
  const madeUnits = lines.reduce((total, line) => total + (line.ProducedUnits || 0) + (line.ScrappedUnits || 0), 0)
  return (order.Supplies || []).map((orderSupply) => {
    const expectedForOutput = plannedUnits > 0 ? Math.round(orderSupply.Planned * madeUnits / plannedUnits) : 0
    return {
      SupplyID: orderSupply.SupplyID, SubDivisor: orderSupply.SubDivisor,
      Planned: orderSupply.Planned, Consumed: orderSupply.Consumed || 0, expectedForOutput,
      usagePercent: expectedForOutput > 0 ? (orderSupply.Consumed || 0) * 100 / expectedForOutput : undefined,
    }
  })
}

export interface ILaborSummaryRow extends IProductionOrderLabor {
  // Recorded ÷ planned minutes, percent; undefined with nothing planned.
  minutesPercent: number | undefined
}

export const summarizeOrderLabor = (labor: IProductionOrderLabor[]): ILaborSummaryRow[] =>
  (labor || []).map((laborLine) => ({
    ...laborLine,
    minutesPercent: laborLine.PlannedMinutes > 0 ? (laborLine.RecordedMinutes || 0) * 100 / laborLine.PlannedMinutes : undefined,
  }))

// ---------------------------------------------------------------------------------------------------
// Draft
// ---------------------------------------------------------------------------------------------------

export const ORDER_STATUS_LABELS: Record<number, string> = {
  [PRODUCTION_ORDER_STATUS.ANNULLED]: 'Annulled|Anulada',
  [PRODUCTION_ORDER_STATUS.DRAFT]: 'Draft|Borrador',
  [PRODUCTION_ORDER_STATUS.RELEASED]: 'Released|Liberada',
  [PRODUCTION_ORDER_STATUS.IN_PROGRESS]: 'In progress|En proceso',
  [PRODUCTION_ORDER_STATUS.FINISHED]: 'Finished|Terminada',
}

// A new manual draft, nothing planned yet.
export const makeEmptyOrder = (): IProductionOrder => ({
  ID: 0, Kind: PRODUCTION_ORDER_KIND.MANUAL, TemplateID: 0, Series: 0, WarehouseID: 0, SuppliesWarehouseID: 0,
  SubcontractorID: 0, SubcontractPricePerUnit: 0, Date: 0, PlannedStart: 0, PlannedEnd: 0, StageID: 0,
  SupplyAdjustmentBasisPoints: 0, Lines: [], Supplies: [], Labor: [], LaborRecordedAmount: 0, Receipts: [], SaleOrderIDs: [],
  IsStrict: false, Notes: '', Created: 0, CreatedBy: 0, ss: PRODUCTION_ORDER_STATUS.DRAFT, upd: 0, ver: 0,
})

// A new draft from a template (§4.2): its lines × series, its warehouses and notes. From then on the
// order is independent of the template.
export const makeOrderFromTemplate = (template: IProductionTemplate, series: number): IProductionOrder => ({
  ...makeEmptyOrder(),
  TemplateID: template.ID, Series: Math.max(Math.floor(series), 1),
  WarehouseID: template.WarehouseID, SuppliesWarehouseID: template.SuppliesWarehouseID, Notes: template.Notes,
  Lines: expandTemplateToOrderLines(template.Lines, series),
})

// The first reason the server would refuse saving the draft that the form can tell on its own.
export const findOrderDraftError = (
  order: Pick<IProductionOrder, 'WarehouseID' | 'SuppliesWarehouseID' | 'SupplyAdjustmentBasisPoints' | 'PlannedStart' | 'PlannedEnd'>,
  lines: IPlannedLine[],
): string | undefined => {
  if (!(order.WarehouseID > 0) || !(order.SuppliesWarehouseID > 0)) {
    return 'Pick the receiving and the supplies warehouse.|Elija el almacén de ingreso y el de insumos.'
  }
  if (order.SupplyAdjustmentBasisPoints <= -BASIS_POINTS_WHOLE || order.SupplyAdjustmentBasisPoints > BASIS_POINTS_WHOLE) {
    return 'The supply adjustment must be between -99.99% and +100%.|El ajuste de insumos debe estar entre -99.99% y +100%.'
  }
  if (order.PlannedStart > 0 && order.PlannedEnd > 0 && order.PlannedEnd < order.PlannedStart) {
    return 'The planned end is before the start.|La fecha de fin planificada es anterior a la de inicio.'
  }
  if (!lines.length) return 'Add at least one product with units.|Agregue al menos un producto con unidades.'
  return undefined
}

// What is still to send to a subcontracted order's workshop of a supply (§8): its plan minus what is
// already there and what was consumed, never below 0.
export const materialToSend = (orderSupply: Pick<IProductionOrderSupply, 'Planned' | 'AtWorkshop' | 'Consumed'>): number =>
  Math.max(orderSupply.Planned - (orderSupply.AtWorkshop || 0) - (orderSupply.Consumed || 0), 0)
