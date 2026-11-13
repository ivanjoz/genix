import { cartUnitsLimit, type IProduct } from '#ecommerce/services/ecommerce-catalog.ts';
import { ecommerceCatalog } from '#ecommerce/services/ecommerce-catalog.svelte.ts';
import { SvelteMap } from 'svelte/reactivity';
import { Env } from '#core/env.ts';
import { notifyWarning } from '@genix/ui/notify';

export interface IProductCategory {
  Name: string;
  Description?: string;
  Image: string;
}

export let layerOpenedState = $state({ id: 0 });

export function setLayerOpenedState(id: number) {
  layerOpenedState.id = id;
}

// The product shown in the detail side layer (ProductDetailLayer); 0 = closed. A grid card sets it.
export const productDetailLayer = $state({ productID: 0 });

export interface ICartProduct {
  producto: IProduct
  cant: number
}

export const ProductsSelectedMap = $state<Map<number,ICartProduct>>(new SvelteMap())

// The cart survives a page reload: it lives in localStorage per company, together with the
// product snapshot it shows (name, image, price), so it renders without waiting for the catalog.
const cartStorageKey = () => `genixWebCart-${Env.getCompanyID()}`

const saveCart = () => {
  localStorage.setItem(cartStorageKey(), JSON.stringify([...ProductsSelectedMap.values()]))
}

// restoreCart runs once on the client after mount: the prerendered HTML always has an empty cart.
export const restoreCart = () => {
  try {
    const storedCart: ICartProduct[] = JSON.parse(localStorage.getItem(cartStorageKey()) || '[]')
    for (const cartProduct of storedCart) {
      ProductsSelectedMap.set(cartProduct.producto.ID, cartProduct)
    }
  } catch (error) {
    console.warn('[cart] stored cart unreadable, discarded:', error)
    localStorage.removeItem(cartStorageKey())
  }
  capCartToStock()
}

export const clearCart = () => {
  ProductsSelectedMap.clear()
  saveCart()
}

// The units of a product the cart may hold, from the live catalog. Until the catalog is loaded there
// is no limit to apply; checkout validates the stock again either way.
export const cartUnitsLimitOf = (productID: number): number =>
  ecommerceCatalog.version > 0 ? cartUnitsLimit(ecommerceCatalog.getProduct(productID)) : Number.POSITIVE_INFINITY

const warnStockLimit = (producto: IProduct, unitsLimit: number) => {
  notifyWarning(unitsLimit > 0
    ? `Solo quedan ${unitsLimit} unidades de "${producto.Name}".`
    : `"${producto.Name}" ya no tiene stock disponible.`)
}

export const addProductoCant = (
  producto: IProduct, fixedCant: number | null, incrementOrDecrement?: number
) => {
  const currentQuantity = ProductsSelectedMap.get(producto.ID)?.cant || 0
  const requestedQuantity = typeof fixedCant === 'number' ? fixedCant : currentQuantity + (incrementOrDecrement || 0)
  // Never more than the store can sell; lowering a quantity is always allowed.
  const unitsLimit = cartUnitsLimitOf(producto.ID)
  const nextQuantity = requestedQuantity > currentQuantity ? Math.min(requestedQuantity, unitsLimit) : requestedQuantity
  if (nextQuantity < requestedQuantity) { warnStockLimit(producto, unitsLimit) }

  if (nextQuantity <= 0) {
    ProductsSelectedMap.delete(producto.ID)
  } else {
    ProductsSelectedMap.set(producto.ID, { producto, cant: nextQuantity })
  }
  saveCart()
}

// capCartToStock lowers every cart line to what the store can sell now: a cart restored from
// localStorage, or one whose products sold meanwhile. Runs on restore and on every catalog publish.
export const capCartToStock = () => {
  if (ecommerceCatalog.version === 0) { return }
  let isCartChanged = false
  for (const cartProduct of [...ProductsSelectedMap.values()]) {
    const unitsLimit = cartUnitsLimitOf(cartProduct.producto.ID)
    if (cartProduct.cant <= unitsLimit) { continue }
    warnStockLimit(cartProduct.producto, unitsLimit)
    if (unitsLimit > 0) {
      ProductsSelectedMap.set(cartProduct.producto.ID, { ...cartProduct, cant: unitsLimit })
    } else {
      ProductsSelectedMap.delete(cartProduct.producto.ID)
    }
    isCartChanged = true
  }
  if (isCartChanged) { saveCart() }
}
