import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'
import type { IProductPack } from './product-packs.ts'
import { PACK_PRICE_TYPE, PACK_ROUNDING } from './product-packs.ts'
import { applySalePacks, packGroupRows, packGroupRowsTotal, type ISalePackLine } from './sale-packs.ts'

// The shared case file the Go engine runs too (docs/PACKS_PLAN.md §4): if the two disagree, a case
// fails on one side.
interface ISalePackCase {
  name: string
  products: { ID: number, CategoryIDs?: number[] }[]
  packs: IProductPack[]
  lines: ISalePackLine[]
  expected: ISalePackLine[]
}

const casesPath = new URL('../../backend/sales/types/testdata/sale_packs_cases.json', import.meta.url)
const packCases = JSON.parse(readFileSync(casesPath, 'utf8')) as ISalePackCase[]

describe('applySalePacks — shared cases with the Go engine', () => {
  for (const packCase of packCases) {
    it(packCase.name, () => {
      const categoryIDsByProduct = new Map(packCase.products.map((product) => [product.ID, product.CategoryIDs || []]))
      const packedLines = applySalePacks(packCase.lines, (productID) => categoryIDsByProduct.get(productID) || [], packCase.packs)
      expect(packedLines).toEqual(packCase.expected)
    })
  }
})

describe('packGroupRows', () => {
  // Arroz con pato (S/ 25) + any Postre (category 30), 15% off rounded down to the sol: S/ 29 → S/ 24.
  const ARROZ = 1, SUSPIRO = 2, CAUSA = 3
  const categoryIDsOf = (productID: number) => productID === SUSPIRO ? [30] : []
  const makePack = (PriceType: number, PriceValue: number): IProductPack => ({
    ID: 9, Name: 'Arroz con pato + postre', PriceType, PriceValue, RoundingMode: PACK_ROUNDING.DOWN, ss: 1, upd: 0,
    Components: [{ ProductID: ARROZ, CategoryID: 0, Quantity: 1 }, { ProductID: 0, CategoryID: 30, Quantity: 1 }],
  })
  const lines: ISalePackLine[] = [
    { ProductID: CAUSA, Price: 500, Quantity: 1000 },
    { ProductID: ARROZ, Price: 2500, Quantity: 2000 },
    { ProductID: SUSPIRO, Price: 400, Quantity: 1000 },
  ]
  const shape = (rows: ReturnType<typeof packGroupRows<ISalePackLine>>) =>
    rows.map((row) => row.pack ? ['pack', row.Amount] : [row.line?.ProductID, row.Quantity, row.Amount, row.isPacked])

  it('lists the unpacked part first, then the pack header over its items with their prorated share', () => {
    const rows = packGroupRows(lines, categoryIDsOf, [makePack(PACK_PRICE_TYPE.PERCENT_OFF, 1500)])
    expect(shape(rows)).toEqual([
      [CAUSA, 1000, 500, false],
      [ARROZ, 1000, 2500, false],
      ['pack', 2400],
      [ARROZ, 1000, 2069, true],
      [SUSPIRO, 1000, 331, true],
    ])
    expect(packGroupRowsTotal(rows)).toBe(500 + 2500 + 2400)
  })

  it('puts a fixed pack price on its first item only', () => {
    const rows = packGroupRows(lines, categoryIDsOf, [makePack(PACK_PRICE_TYPE.FIXED, 2000)])
    expect(shape(rows).slice(2)).toEqual([['pack', 2000], [ARROZ, 1000, 2000, true], [SUSPIRO, 1000, 0, true]])
    expect(packGroupRowsTotal(rows)).toBe(500 + 2500 + 2000)
  })

  it('keeps a line no pack touched, even with 0 units', () => {
    const rows = packGroupRows([{ ProductID: CAUSA, Price: 500, Quantity: 0 }], categoryIDsOf, [makePack(PACK_PRICE_TYPE.FIXED, 2000)])
    expect(shape(rows)).toEqual([[CAUSA, 0, 0, false]])
  })
})
