import { bitmaskHas } from '#libs/bitmask.ts'

// Where a product is shown and sold (IProduct.SalesChannels, "Mostrar en"): option ID n is bit n-1
// of the mask, and 0 means every channel. Mirrors production.ProductChannel*.
export const PRODUCT_CHANNEL_PHYSICAL_STORE = 1
export const PRODUCT_CHANNEL_ONLINE_STORE = 2
export const PRODUCT_CHANNEL_RESTAURANT = 3

// Whether the product is shown and sold in the channel: no channel picked means all of them.
// Mirrors Product.IsSoldIn.
export function isProductSoldIn(product: { SalesChannels?: number }, channel: number): boolean {
  return !product.SalesChannels || bitmaskHas(product.SalesChannels, channel)
}
