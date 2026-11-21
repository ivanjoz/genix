import { describe, expect, it } from 'vitest'
import {
  MENU_BLOCK, MENU_PRODUCTS_VARIANT, groupProductsByCategory, menuCatalogProducts, resolveDiningMenu,
  readableTextColor, restaurantPriceInSoles, type IDiningMenuBlock, type IMenuCatalogProduct,
} from './dining-menu.ts'

const dish = (ID: number, Name: string, CategoryIDs: number[]): IMenuCatalogProduct =>
  ({ ID, Name, Description: '', Price: 1000, ImageName: '', CategoryIDs })

const catalog = [dish(1, 'Ceviche', [10]), dish(2, 'Anticucho', [10, 20]), dish(3, 'Chicha', [20]), dish(4, 'Pan', [])]
const categories = [{ ID: 10, Name: 'Entradas' }, { ID: 20, Name: 'Bebidas' }, { ID: 30, Name: 'Postres' }]

describe('groupProductsByCategory', () => {
  it('first category wins, empty categories are skipped, uncategorized go last', () => {
    const groups = groupProductsByCategory(catalog, categories)
    expect(groups.map((group) => [group.heading, group.products.map((product) => product.Name)])).toEqual([
      ['Entradas', ['Anticucho', 'Ceviche']], ['Bebidas', ['Chicha']], ['Otros', ['Pan']],
    ])
  })
})

describe('resolveDiningMenu', () => {
  const allBlock = (RestOnly: boolean): IDiningMenuBlock =>
    ({ Kind: MENU_BLOCK.ALL_PRODUCTS, Variant: MENU_PRODUCTS_VARIANT.COMPACT, RestOnly })

  it('"all the rest" leaves out what a category or products block lists', () => {
    const blocks: IDiningMenuBlock[] = [
      { Kind: MENU_BLOCK.CATEGORY, CategoryID: 20 },
      { Kind: MENU_BLOCK.PRODUCTS, ProductIDs: [4] },
      allBlock(true),
    ]
    const rest = resolveDiningMenu(blocks, catalog, categories, 'builder')[2]
    expect(rest.groups.flatMap((group) => group.products.map((product) => product.ID))).toEqual([1])
  })

  it('a category block takes its heading from the category unless overridden', () => {
    const [plain, renamed] = resolveDiningMenu(
      [{ Kind: MENU_BLOCK.CATEGORY, CategoryID: 10 }, { Kind: MENU_BLOCK.CATEGORY, CategoryID: 10, Text: 'Para empezar' }],
      catalog, categories, 'builder')
    expect([plain.groups[0].heading, renamed.groups[0].heading]).toEqual(['Entradas', 'Para empezar'])
  })

  it('a products block keeps the picked order and skips products gone from the catalog', () => {
    const [resolved] = resolveDiningMenu([{ Kind: MENU_BLOCK.PRODUCTS, ProductIDs: [3, 99, 1] }], catalog, categories, 'builder')
    expect(resolved.groups[0].products.map((product) => product.ID)).toEqual([3, 1])
  })

  it('the public page drops product blocks with nothing to order; the builder keeps them', () => {
    const blocks: IDiningMenuBlock[] = [
      { Kind: MENU_BLOCK.TITLE, Text: 'Carta' }, { Kind: MENU_BLOCK.CATEGORY, CategoryID: 30 },
    ]
    expect(resolveDiningMenu(blocks, catalog, categories, 'public').map((resolved) => resolved.index)).toEqual([0])
    expect(resolveDiningMenu(blocks, catalog, categories, 'builder').map((resolved) => resolved.index)).toEqual([0, 1])
  })
})

describe('menuCatalogProducts', () => {
  const product = (overrides: object) => ({ ID: 1, Name: 'A', FinalPrice: 1000, CurrencyID: 1, ss: 1, SalesChannels: 0, CategoryIDs: [], ...overrides }) as any

  it('keeps active restaurant products, restating dollars into soles', () => {
    const products = [
      product({ ID: 1 }), product({ ID: 2, ss: 0 }), product({ ID: 3, SalesChannels: 1 }),
      product({ ID: 4, CurrencyID: 2, FinalPrice: 1000 }),
    ]
    const listed = menuCatalogProducts(products, 3500)
    expect(listed.map((entry) => [entry.ID, entry.Price])).toEqual([[1, 1000], [4, 3500]])
  })

  it('a dollar product with no rate cannot be priced', () => {
    expect(restaurantPriceInSoles({ FinalPrice: 1000, CurrencyID: 2 }, 0)).toBeNull()
    expect(menuCatalogProducts([product({ CurrencyID: 2 })], 0)).toEqual([])
  })

  it('the public catalog keeps only products with units left', () => {
    const listed = menuCatalogProducts([product({ ID: 1 }), product({ ID: 2 })], 0, new Map([[1, 5]]))
    expect(listed.map((entry) => [entry.ID, entry.Sellable])).toEqual([[1, 5]])
  })
})

describe('readableTextColor', () => {
  it('is dark on light backgrounds and light on dark ones', () => {
    expect(readableTextColor('#fffbeb')).toBe('#1c1917')
    expect(readableTextColor('#1c1917')).toBe('#fafaf9')
  })
})
