import type { IProduct } from '#ecommerce/services/ecommerce-catalog.ts';
import { SvelteMap } from 'svelte/reactivity';
import { Env } from '#core/env.ts';

export interface IProductCategory {
  Name: string;
  Description?: string;
  Image: string;
}

export let layerOpenedState = $state({ id: 0 });

export function setLayerOpenedState(id: number) {
  layerOpenedState.id = id;
}

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
}

export const clearCart = () => {
  ProductsSelectedMap.clear()
  saveCart()
}

export const addProductoCant = (
  producto: IProduct, fixedCant: number | null, incrementOrDecrement?: number
) => {
  if(typeof fixedCant === 'number'){
    if(fixedCant === 0){
      ProductsSelectedMap.delete(producto.ID)
    } else {
      ProductsSelectedMap.set(producto.ID, { producto, cant: fixedCant })
    }
  } else if(typeof incrementOrDecrement === 'number'){
    const cant = ProductsSelectedMap.get(producto.ID)?.cant || 0
    ProductsSelectedMap.set(producto.ID, { producto, cant: cant + incrementOrDecrement })
  }
  saveCart()
}
