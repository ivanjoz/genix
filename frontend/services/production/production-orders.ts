// Mirrors backend/production/types/production_order.go: one production, a manual order (Kind 1) or a
// daily close (Kind 2) (docs/MANUFACTURING_PLAN.md §3). Money is PEN cents; supply quantities are
// sub-units of the supply's divisor.

export const PRODUCTION_ORDER_KIND = { MANUAL: 1, DAILY_CLOSE: 2 } as const

export const PRODUCTION_ORDER_STATUS = { ANNULLED: 0, DRAFT: 1, RELEASED: 2, IN_PROGRESS: 3, FINISHED: 4 } as const

// Where a produced line's value came from.
export const COST_SOURCE = { STANDARD: 1, RECIPE: 2 } as const

export interface IProductionOrderLine {
  ProductID: number
  PresentationID: number
  PlannedUnits: number
  ProducedUnits: number
  ScrappedUnits: number
  Value: number
  CostSource: number
  ShortSupplies?: boolean
}

export interface IProductionOrderSupply {
  SupplyID: number
  SubDivisor: number
  Planned: number
  Available: number
  Consumed: number
  Reserved: number
  // Subcontracted: sent to the workshop and not returned or consumed yet (§8).
  AtWorkshop?: number
}

export interface IProductionOrderLabor {
  LaborTypeID: number
  PlannedMinutes: number
  PlannedCost: number
  RecordedMinutes: number
  RecordedCost: number
}

export interface IProductionReceiptLine {
  ProductID: number
  PresentationID: number
  Units: number
  Scrapped: number
  LotID: number
  Value: number
}

export interface IProductionReceiptConsumption {
  SupplyID: number
  LotID: number
  SerialNumber: string
  SubUnits: number
  SubDivisor: number
  Value: number
}

export interface IProductionReceipt {
  N: number
  Date: number
  Lines: IProductionReceiptLine[]
  Consumptions: IProductionReceiptConsumption[]
  SuppliesValue: number
  LaborValue: number
  SubcontractValue: number
  OverheadValue: number
  CreatedBy: number
}

export interface IProductionOrder {
  ID: number
  Kind: number
  TemplateID: number
  Series: number
  WarehouseID: number
  SuppliesWarehouseID: number
  SubcontractorID: number
  SubcontractPricePerUnit: number
  Date: number
  PlannedStart: number
  PlannedEnd: number
  StageID: number
  SupplyAdjustmentBasisPoints: number
  Lines: IProductionOrderLine[]
  Supplies: IProductionOrderSupply[]
  Labor: IProductionOrderLabor[]
  LaborRecordedAmount: number
  Receipts: IProductionReceipt[]
  SaleOrderIDs: number[]
  IsStrict: boolean
  Notes: string
  Created: number
  CreatedBy: number
  ss: number
  upd: number
  // The version a receipt must send back: one registered meanwhile makes the server refuse it.
  ver: number
}

// The units of one presentation (a size) in one series of a template.
export interface IPresentationQuantity {
  PresentationID: number
  Units: number
}

// One product of a template in its size run: 36×1, 37×2, 38×3… is one series.
export interface IProductionTemplateLine {
  ProductID: number
  Run: IPresentationQuantity[]
}

// IProductionTemplate mirrors production.ProductionTemplate (docs/MANUFACTURING_PLAN.md §4.1): a
// reusable list of products in their size runs. An order made from it is independent of it.
export interface IProductionTemplate {
  ID: number
  Name: string
  Lines: IProductionTemplateLine[]
  DefaultSeries: number
  WarehouseID: number
  SuppliesWarehouseID: number
  Notes: string
  ss: number
  upd: number
}

// The receipt the server registers (POST production-order-receipt).
export interface IProductionReceiptRequest {
  OrderID: number
  // The order's `ver` as loaded: a receipt registered meanwhile makes the server refuse this one.
  Version: number
  Lines: { ProductID: number, PresentationID: number, Units: number, Scrapped: number }[]
  // SubUnits at the order supply's SubDivisor.
  Consumptions: { SupplyID: number, SubUnits: number }[]
  // Finishes the order with this receipt, whatever is left unreceived.
  Complete: boolean
}

// IProductionLaborEntry mirrors production.ProductionLaborEntry (§9.2): time a worker spent on an
// order, of one labor type, valued by the server at the worker's real cost per hour.
export interface IProductionLaborEntry {
  ID: number
  OrderID: number
  EmployeeID: number
  LaborTypeID: number
  Date: number
  Period: number
  Minutes: number
  CostPerHour: number
  Amount: number
  ss: number
  upd: number
}
