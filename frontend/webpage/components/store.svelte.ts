import type { IProduct } from '$ecommerce/services/products.svelte';
import { SvelteMap } from 'svelte/reactivity';

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
}
