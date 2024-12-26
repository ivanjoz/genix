import { createSignal } from "solid-js";
import { IHeader1 } from "./headers";
import { IPageSection } from "./page";
import { makeRoute } from "~/shared/http";
import { getToken, Params } from "~/shared/security";
import { IProducto } from "~/services/operaciones/productos";
import { IListaRegistro } from "~/services/admin/listas-compartidas";

const maxCacheTime = 5 * 60 // 5 minutos

const productosPromiseMap: Map<string,Promise<any>> = new Map()

export const useProductosCmsAPI = (categoriasIDs?: number[]) => {

  const [fetchedRecords, setFetchedRecords] = createSignal({
    productos: [] as IProducto[],
    categorias: [] as IListaRegistro[],
    isFetching: true,
    updated: -1
  })

  const key = "productos_" + (categoriasIDs||[0]).join("_")
  const nowTime = Math.floor(Date.now()/1000)

  const cached = localStorage.getItem(key)
  if(cached){
    const productos = JSON.parse(cached)
    if((nowTime - productos.updated) < maxCacheTime){
      console.log("enviando productos del caché::", productos)
      setFetchedRecords(productos)
      return [fetchedRecords]
    }
  }

  if(!productosPromiseMap.has(key)){
    const route = makeRoute("p-productos-cms")
    const headers = new Headers()
    headers.append('Authorization', `Bearer ${getToken()}`)
  
    productosPromiseMap.set(key, new Promise((resolve, reject) => {
      fetch(route, { headers })
      .then(results => {
        return results.json()
      })
      .then(results => {
        console.log("productos obtenidos desde servidor:", results)
        results.updated = nowTime
        resolve(results)
        if(results.productos?.length > 0){
          localStorage.setItem(key, JSON.stringify(results))
        }
      })
      .catch(err => {
        reject(err)
      })
    }))
  }

  productosPromiseMap.get(key).then(results => {
    console.log("productos obtenidos promesa::", results)
    setFetchedRecords(results)
  })

  return [fetchedRecords]
}