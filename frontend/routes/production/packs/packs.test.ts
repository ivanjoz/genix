import { describe, expect, it } from 'vitest'
import { PACK_PRICE_TYPE, PACK_ROUNDING, type IProductPack } from '#core/product-packs.ts'
import {
  describePackComponents, describePackPrice, findPackError, makeEmptyPack, packComponentRowsOf, packComponentsOf,
} from './packs.ts'

const STARTER_CATEGORY_ID = 3
const MAIN_CATEGORY_ID = 4
const LEMONADE_ID = 20

const menuPack = (pack: Partial<IProductPack> = {}): IProductPack => ({
  ...makeEmptyPack(), ID: 1, Name: 'Menú del día', PriceValue: 1800,
  Components: [
    { ProductID: 0, CategoryID: STARTER_CATEGORY_ID, Quantity: 1 },
    { ProductID: 0, CategoryID: MAIN_CATEGORY_ID, Quantity: 1 },
    { ProductID: LEMONADE_ID, CategoryID: 0, Quantity: 1 },
  ],
  ...pack,
})

const translateToEnglish = (text: string) => text.split('|')[0]

describe('pack component rows', () => {
  it('round-trips the components through the single product-or-category key', () => {
    const pack = menuPack()
    const rows = packComponentRowsOf(pack)
    expect(rows.map((row) => row.ItemKey)).toEqual([`c${STARTER_CATEGORY_ID}`, `c${MAIN_CATEGORY_ID}`, `p${LEMONADE_ID}`])
    expect(packComponentsOf(rows)).toEqual(pack.Components)
  })
})

describe('findPackError', () => {
  const rowsOf = (pack: IProductPack) => packComponentRowsOf(pack)

  it('accepts a complete pack', () => {
    expect(findPackError(menuPack(), rowsOf(menuPack()))).toBeUndefined()
  })

  it('refuses an unpicked row, a repeated item and a single-unit pack', () => {
    const pack = menuPack()
    expect(findPackError(pack, [...rowsOf(pack), { ItemKey: '', Quantity: 1 }])).toMatch(/Pick the product/)
    expect(findPackError(pack, [...rowsOf(pack), { ItemKey: `c${STARTER_CATEGORY_ID}`, Quantity: 1 }])).toMatch(/repeated/)
    expect(findPackError(pack, [{ ItemKey: `p${LEMONADE_ID}`, Quantity: 1 }])).toMatch(/at least 2 units/)
  })

  it('refuses a missing price and a discount of 100% or more', () => {
    expect(findPackError(menuPack({ PriceValue: 0 }), rowsOf(menuPack()))).toMatch(/Enter the price/)
    const fullDiscount = menuPack({ PriceType: PACK_PRICE_TYPE.PERCENT_OFF, PriceValue: 10000 })
    expect(findPackError(fullDiscount, rowsOf(fullDiscount))).toMatch(/below 100%/)
  })
})

describe('pack descriptions', () => {
  it('lists the components by name', () => {
    const categoryNames = new Map([[STARTER_CATEGORY_ID, 'Entrada'], [MAIN_CATEGORY_ID, 'Plato Fondo 1']])
    expect(describePackComponents(menuPack().Components, () => 'Limonada', (categoryID) => categoryNames.get(categoryID) || ''))
      .toBe('1× Entrada + 1× Plato Fondo 1 + 1× Limonada')
  })

  it('shows the price rule, and the rounding only on a computed total', () => {
    expect(describePackPrice(menuPack({ RoundingMode: PACK_ROUNDING.UP }), translateToEnglish)).toBe('S/ 18.00')
    expect(describePackPrice(menuPack({ PriceType: PACK_PRICE_TYPE.PERCENT_OFF, PriceValue: 1500, RoundingMode: PACK_ROUNDING.DOWN }), translateToEnglish))
      .toBe('−15.00% · round down')
    expect(describePackPrice(menuPack({ PriceType: PACK_PRICE_TYPE.AMOUNT_OFF, PriceValue: 200 }), translateToEnglish)).toBe('−S/ 2.00')
  })
})
