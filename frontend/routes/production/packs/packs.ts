import { PACK_PRICE_TYPE, PACK_ROUNDING, type IProductPack, type IProductPackComponent } from '#services/production/product-packs.ts'

// The rules of the packs page (docs/PACKS_PLAN.md §2): the editable component rows, what the form
// can refuse before the server does, and how a pack reads in the list.

export const makeEmptyPack = (): IProductPack => ({
  ID: 0, Name: '', Components: [], PriceType: PACK_PRICE_TYPE.FIXED, PriceValue: 0,
  RoundingMode: PACK_ROUNDING.NONE, ss: 1, upd: 0,
})

// One editable row of the components grid. A single picker lists categories and products together,
// so the row holds one key: "c<CategoryID>" or "p<ProductID>" ("" while not picked yet).
export interface IPackComponentRow {
  ItemKey: string
  Quantity: number
}

export const packItemKeyOf = (component: IProductPackComponent): string =>
  component.CategoryID > 0 ? `c${component.CategoryID}` : component.ProductID > 0 ? `p${component.ProductID}` : ''

export const packComponentRowsOf = (pack: IProductPack): IPackComponentRow[] =>
  (pack.Components || []).map((component) => ({ ItemKey: packItemKeyOf(component), Quantity: component.Quantity }))

export const packComponentsOf = (rows: IPackComponentRow[]): IProductPackComponent[] =>
  rows.map((row) => {
    const itemID = parseInt(row.ItemKey.slice(1)) || 0
    return {
      ProductID: row.ItemKey.startsWith('p') ? itemID : 0,
      CategoryID: row.ItemKey.startsWith('c') ? itemID : 0,
      Quantity: row.Quantity,
    }
  })

// The first reason the server would refuse the pack that the form can tell on its own.
export const findPackError = (pack: IProductPack, rows: IPackComponentRow[]): string | undefined => {
  if (pack.Name.trim().length < 2) return 'The name needs at least 2 characters.|El nombre debe tener al menos 2 caracteres.'
  if (!rows.length) return 'Add at least one component.|Agregue al menos un componente.'
  if (rows.some((row) => !row.ItemKey)) return 'Pick the product or category of every row.|Elija el producto o categoría de cada fila.'
  if (rows.some((row) => !(row.Quantity >= 1))) return 'Every component needs a quantity of 1 or more.|Cada componente necesita una cantidad de 1 o más.'
  if (new Set(rows.map((row) => row.ItemKey)).size !== rows.length) {
    return 'A product or category is repeated: add up its quantity in one row.|Un producto o categoría está repetido: sume su cantidad en una fila.'
  }
  if (rows.reduce((totalUnits, row) => totalUnits + row.Quantity, 0) < 2) {
    return 'A pack needs at least 2 units in total.|Un pack necesita al menos 2 unidades en total.'
  }
  if (!(pack.PriceValue > 0)) return 'Enter the price or the discount.|Ingrese el precio o el descuento.'
  if (pack.PriceType === PACK_PRICE_TYPE.PERCENT_OFF && pack.PriceValue >= 10000) {
    return 'The discount must be below 100%.|El descuento debe ser menor a 100%.'
  }
  return undefined
}

// "1× Entrada + 1× Plato Fondo 1", the way the pack is read in the list.
export const describePackComponents = (components: IProductPackComponent[],
  productNameOf: (productID: number) => string, categoryNameOf: (categoryID: number) => string): string =>
  (components || []).map((component) => {
    const itemName = component.CategoryID > 0 ? categoryNameOf(component.CategoryID) : productNameOf(component.ProductID)
    return `${component.Quantity}× ${itemName}`
  }).join(' + ')

const ROUNDING_LABELS: Record<number, string> = {
  [PACK_ROUNDING.UP]: 'round up|redondeo arriba',
  [PACK_ROUNDING.NEAREST]: 'round|redondeo',
  [PACK_ROUNDING.DOWN]: 'round down|redondeo abajo',
}

// "S/ 18.00", "−15.00%" or "−S/ 2.00", plus the rounding when the total is computed. `translate`
// picks the language of the "English|Spanish" rounding label.
export const describePackPrice = (pack: IProductPack, translate: (text: string) => string): string => {
  const amountText = (pack.PriceValue / 100).toFixed(2)
  if (pack.PriceType === PACK_PRICE_TYPE.FIXED) return `S/ ${amountText}`
  const discountText = pack.PriceType === PACK_PRICE_TYPE.PERCENT_OFF ? `−${amountText}%` : `−S/ ${amountText}`
  const roundingLabel = ROUNDING_LABELS[pack.RoundingMode]
  return roundingLabel ? `${discountText} · ${translate(roundingLabel)}` : discountText
}
