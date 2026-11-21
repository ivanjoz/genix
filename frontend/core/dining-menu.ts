import { PRODUCT_CHANNEL_RESTAURANT, isProductSoldIn } from '#core/product-channels.ts'
import { CURRENCY_PEN, convertUnitPrice, normalizeCurrency } from '#routes/finance/exchange-rate/exchange-rate.ts'
import type { IProduct } from '#services/production/products.svelte.ts'

// The carta (docs/DINING_MENU_PLAN.md): the blocks a restaurant composes and how they resolve
// against the catalog. Pure, so the builder, the QR page and the tests share one set of rules.
// Constants mirror backend/dining/types/dining_menu.go.

export const MENU_BLOCK = {
  ALL_PRODUCTS: 1, CATEGORY: 2, PRODUCTS: 3, TITLE: 4, SUBTITLE: 5, TEXT: 6, IMAGE: 7, DIVIDER: 8,
} as const

export const MENU_PRODUCTS_VARIANT = { COMPACT: 1, WITH_PHOTO: 2, CARDS: 3 } as const
export const MENU_FONT = { SANS: 1, SERIF: 2, ROUNDED: 3 } as const
export const MENU_ALIGN = { LEFT: 1, CENTER: 2 } as const

// The block kinds that list products.
export const isProductBlock = (kind: number) =>
  kind === MENU_BLOCK.ALL_PRODUCTS || kind === MENU_BLOCK.CATEGORY || kind === MENU_BLOCK.PRODUCTS

//STRUCT:dining.DiningMenuBlock
export interface IDiningMenuBlock {
  Kind: number
  Text?: string
  CategoryID?: number
  ProductIDs?: number[]
  ImageURL?: string
  Variant?: number
  Align?: number
  RestOnly?: boolean
}

//STRUCT:dining.DiningMenuTheme
export interface IDiningMenuTheme {
  AccentColor: string
  BackgroundColor: string
  Font: number
}

//STRUCT:dining.DiningMenu
export interface IDiningMenu {
  ID: number
  Name: string
  Theme: IDiningMenuTheme
  Blocks: IDiningMenuBlock[]
  ss: number
  upd: number
}

export const DEFAULT_MENU_THEME: IDiningMenuTheme = { AccentColor: '#b45309', BackgroundColor: '#fffbeb', Font: MENU_FONT.SANS }

// A new carta starts as every product, so the user sees their catalog at once and builds from there.
export const defaultMenuBlocks = (): IDiningMenuBlock[] =>
  [{ Kind: MENU_BLOCK.ALL_PRODUCTS, Variant: MENU_PRODUCTS_VARIANT.WITH_PHOTO }]

// A product as the carta lists it: priced in soles (cents), with its main image's CDN name.
export interface IMenuCatalogProduct {
  ID: number
  Name: string
  Description: string
  Price: number
  ImageName: string
  CategoryIDs: number[]
  // Units that can be ordered now; absent in the builder, which shows every product whatever its stock.
  Sellable?: number
}

export interface IMenuCategory { ID: number, Name: string }

// The product's unit price in soles, or null when it is priced in dollars and there is no rate to
// restate it with (the product cannot be ordered then). Shared by the waiter's menu and the carta.
export function restaurantPriceInSoles(product: Pick<IProduct, 'FinalPrice' | 'CurrencyID'>, penRate: number): number | null {
  const productCurrency = normalizeCurrency(product.CurrencyID)
  if (productCurrency !== CURRENCY_PEN && !penRate) return null
  return convertUnitPrice(product.FinalPrice, productCurrency, CURRENCY_PEN, penRate)
}

// The builder's catalog: the active products sold in the restaurant. `sellableByProductID`, when
// given (the public page), keeps only the products with units left and records them.
export function menuCatalogProducts(products: IProduct[], penRate: number, sellableByProductID?: Map<number, number>): IMenuCatalogProduct[] {
  const catalog: IMenuCatalogProduct[] = []
  for (const product of products) {
    if (product.ss <= 0 || !isProductSoldIn(product, PRODUCT_CHANNEL_RESTAURANT)) continue
    const price = restaurantPriceInSoles(product, penRate)
    if (price === null) continue
    const sellable = sellableByProductID?.get(product.ID)
    if (sellableByProductID && !sellable) continue
    catalog.push({
      ID: product.ID, Name: product.Name, Description: product.Description || '', Price: price,
      ImageName: product.Image?.n || '', CategoryIDs: product.CategoryIDs || [], Sellable: sellable,
    })
  }
  return catalog
}

// One run of products under an optional heading.
export interface IResolvedMenuGroup { heading: string, products: IMenuCatalogProduct[] }

export interface IResolvedMenuBlock {
  // Index in the carta's Blocks, which the builder selects and reorders by.
  index: number
  block: IDiningMenuBlock
  groups: IResolvedMenuGroup[]
}

const OTHERS_HEADING = 'Otros'

// The products of the "Todos los productos" block: one group per category in the given order, a
// product under its first category, those with none last under "Otros"; products by name.
export function groupProductsByCategory(products: IMenuCatalogProduct[], categories: IMenuCategory[]): IResolvedMenuGroup[] {
  const categoryOrder = new Map(categories.map((category, position) => [category.ID, position]))
  const productsByCategory = new Map<number, IMenuCatalogProduct[]>()
  const uncategorized: IMenuCatalogProduct[] = []
  for (const product of products) {
    const firstCategoryID = product.CategoryIDs.find((categoryID) => categoryOrder.has(categoryID))
    if (firstCategoryID === undefined) { uncategorized.push(product); continue }
    productsByCategory.set(firstCategoryID, [...(productsByCategory.get(firstCategoryID) || []), product])
  }
  const byName = (left: IMenuCatalogProduct, right: IMenuCatalogProduct) => left.Name.localeCompare(right.Name)
  const groups = categories
    .filter((category) => productsByCategory.has(category.ID))
    .map((category) => ({ heading: category.Name, products: productsByCategory.get(category.ID)!.sort(byName) }))
  if (uncategorized.length) groups.push({ heading: OTHERS_HEADING, products: uncategorized.sort(byName) })
  return groups
}

// The product IDs the carta's other product blocks list, which a "Todos los demás" block leaves out.
function productIDsListedByOtherBlocks(blocks: IDiningMenuBlock[], catalog: IMenuCatalogProduct[]): Set<number> {
  const listedIDs = new Set<number>()
  for (const block of blocks) {
    if (block.Kind === MENU_BLOCK.PRODUCTS) (block.ProductIDs || []).forEach((productID) => listedIDs.add(productID))
    if (block.Kind === MENU_BLOCK.CATEGORY) {
      catalog.filter((product) => product.CategoryIDs.includes(block.CategoryID || 0)).forEach((product) => listedIDs.add(product.ID))
    }
  }
  return listedIDs
}

// Turns the carta into what is drawn. 'public' (the QR page) drops a product block, or a category
// heading, left with nothing to order; 'builder' keeps every block so it can be selected, an empty
// product block showing a placeholder.
export function resolveDiningMenu(
  blocks: IDiningMenuBlock[], catalog: IMenuCatalogProduct[], categories: IMenuCategory[], mode: 'builder' | 'public',
): IResolvedMenuBlock[] {
  const catalogByID = new Map(catalog.map((product) => [product.ID, product]))
  const categoryNameByID = new Map(categories.map((category) => [category.ID, category.Name]))
  const resolvedBlocks: IResolvedMenuBlock[] = []

  blocks.forEach((block, index) => {
    let groups: IResolvedMenuGroup[] = []
    if (block.Kind === MENU_BLOCK.ALL_PRODUCTS) {
      const listedElsewhere = block.RestOnly ? productIDsListedByOtherBlocks(blocks, catalog) : new Set<number>()
      groups = groupProductsByCategory(catalog.filter((product) => !listedElsewhere.has(product.ID)), categories)
    } else if (block.Kind === MENU_BLOCK.CATEGORY) {
      const categoryProducts = catalog.filter((product) => product.CategoryIDs.includes(block.CategoryID || 0))
        .sort((left, right) => left.Name.localeCompare(right.Name))
      const heading = block.Text || categoryNameByID.get(block.CategoryID || 0) || ''
      groups = categoryProducts.length ? [{ heading, products: categoryProducts }] : []
    } else if (block.Kind === MENU_BLOCK.PRODUCTS) {
      // The order the user gave, skipping what is no longer in the catalog.
      const pickedProducts = (block.ProductIDs || []).flatMap((productID) => catalogByID.get(productID) || [])
      groups = pickedProducts.length ? [{ heading: '', products: pickedProducts }] : []
    }
    if (mode === 'public' && isProductBlock(block.Kind) && groups.length === 0) return
    resolvedBlocks.push({ index, block, groups })
  })
  return resolvedBlocks
}

// The text color a carta shows on its background: dark on a light one, light on a dark one
// (the theme stores only the background and the accent).
export function readableTextColor(backgroundHex: string): string {
  const [red, green, blue] = [1, 3, 5].map((offset) => parseInt(backgroundHex.slice(offset, offset + 2), 16) || 0)
  const perceivedBrightness = (red * 299 + green * 587 + blue * 114) / 1000
  return perceivedBrightness > 150 ? '#1c1917' : '#fafaf9'
}

// The font stacks of MENU_FONT.
export const MENU_FONT_FAMILIES: Record<number, string> = {
  [MENU_FONT.SANS]: 'system-ui, -apple-system, "Segoe UI", sans-serif',
  [MENU_FONT.SERIF]: 'Georgia, "Times New Roman", serif',
  [MENU_FONT.ROUNDED]: 'ui-rounded, "Nunito", "Quicksand", system-ui, sans-serif',
}
