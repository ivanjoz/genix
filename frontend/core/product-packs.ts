// IProductPack mirrors production.ProductPack (docs/PACKS_PLAN.md §2): a bundle of products and/or
// categories priced as a whole, applied automatically by the sales engine.
export interface IProductPack {
  ID: number
  Name: string
  Components: IProductPackComponent[]
  PriceType: number
  // PEN cents for PACK_PRICE_TYPE.FIXED and AMOUNT_OFF; basis points (1500 = 15%) for PERCENT_OFF.
  PriceValue: number
  RoundingMode: number
  ss: number
  upd: number
}

// One slot of a pack: exactly one of ProductID / CategoryID is set.
export interface IProductPackComponent {
  ProductID: number
  CategoryID: number
  Quantity: number
}

export const PACK_PRICE_TYPE = { FIXED: 1, PERCENT_OFF: 2, AMOUNT_OFF: 3 } as const

// Applied to the pack total, to the whole sol. A fixed price is never rounded.
export const PACK_ROUNDING = { NONE: 0, UP: 1, NEAREST: 2, DOWN: 3 } as const
