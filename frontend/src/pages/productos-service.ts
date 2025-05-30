import { useLocation } from "@solidjs/router";
import { createEffect, createSignal, on } from "solid-js";
import { Env, IsClient, LocalStorage } from "~/env";
import { IListaRegistro } from "~/services/admin/listas-compartidas";
import { IProducto } from "~/services/operaciones/productos";
import { makeRoute } from "~/shared/http";
import { arrayToMapN, makeEcommerceDB } from "~/shared/main";
import { getToken } from "~/shared/security";
import { ICartProducto, ICartProductoCache, setCartProductos } from "./productos";

const maxCacheTime = 60 * 5 // 2 segundos
const productosPromiseMap: Map<string,Promise<any>> = new Map()

export interface IProductosResult {
  productos: IProducto[]
  categorias: IListaRegistro[]
  categoriasMap: Map<number,IListaRegistro>
  isFetching?: boolean
  updated?: number
}

const parseProductos = (productos: IProductosResult) => {
  productos.productos = productos.productos || []
  for(const p of productos.productos){
    p._stock = 0
    p._moneda = "s/."
    for(const stock of p.Stock){ p._stock += (stock.c||0) }
  }

  productos.categorias = productos.categorias || []
  productos.categoriasMap = arrayToMapN(productos.categorias,'ID')
}

export const useProductosCmsAPI = (categoriasIDs?: number[]) => {

  const [fetchedRecords, setFetchedRecords] = createSignal({
    productos: [], categorias: [], categoriasMap: new Map(),
    isFetching: true, updated: -1
  } as IProductosResult)

  createEffect(on(
    () => [ fetchedRecords() ],
    async () => {
      const productos = fetchedRecords().productos
      if(productos.length === 0){ return }
      // Obtiene los productos del caché del carrito y los setea
      const db = await makeEcommerceDB()
      if(!db){ return }
      const record = await db.table("cache").get("cart")
      const cartProductosCached = (record?.productos||[]) as ICartProductoCache[]
  
      const productosMap = arrayToMapN(productos,'ID')
      const cartProductos: Map<number,ICartProducto> = new Map()

      for(const cp of cartProductosCached){
        const producto = productosMap.get(cp.id)
        if(!producto){ continue }
        if(cp.cant > producto._stock){ cp.cant = producto._stock }
        cartProductos.set(cp.id,{
          cant: cp.cant, producto
        })
      }
      setCartProductos(cartProductos)
    }
  ))

  const key = "productos_" + (categoriasIDs||[0]).join("_")
  const nowTime = Math.floor(Date.now()/1000)

  const cached = LocalStorage.getItem(key)
  if(cached){
    const productos = JSON.parse(cached)

    if((nowTime - productos.updated) < maxCacheTime){
      console.log("enviando productos del caché::", productos)
      parseProductos(productos)
      setFetchedRecords(productos)
      return [fetchedRecords]
    }
  }
  
  if(IsClient()){
    if(!productosPromiseMap.has(key)){
      const route = makeRoute("p-productos-cms") + `?empresa-id=${Env.getEmpresaID()}`
      const headers = new Headers()
      headers.append('Authorization', `Bearer ${getToken(true)}`)
      
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
            LocalStorage.setItem(key, JSON.stringify(results))
          }
        })
        .catch(err => {
          reject(err)
        })
      }))
    }

    productosPromiseMap.get(key).then(results => {
      console.log("productos obtenidos promesa::", results)
      parseProductos(results)
      setFetchedRecords(results)
    })
  }

  return [fetchedRecords]
}