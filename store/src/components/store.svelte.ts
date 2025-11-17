import type { IProducto } from "../services/productos.svelte";
import { SvelteMap } from 'svelte/reactivity';

export interface ICategoriaProducto {
  Name: string;
  Description?: string;
  Image: string;
}

export let layerOpenedState = $state({ id: 0 });

export function setLayerOpenedState(id: number) {
  layerOpenedState.id = id;
}

export interface ICartProducto {
  producto: IProducto
  cant: number
}

export const ProductsSelectedMap = $state<Map<number,ICartProducto>>(new SvelteMap())

export const addProductoCant = (
  producto: IProducto, fixedCant: number | null, incrementOrDecrement?: number
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