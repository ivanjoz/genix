import { createSignal } from "solid-js";
import { IListaRegistro } from "~/services/admin/listas-compartidas";
import { IProducto } from "~/services/operaciones/productos";
import { makeRoute } from "~/shared/http";
import { getToken } from "~/shared/security";

const maxCacheTime = 2 // 2 segundos

const productosPromiseMap: Map<string,Promise<any>> = new Map()

export const getEmpresaID = (): number => {
  const location = window.location.pathname.split("/").filter(x => x)
  if(location[1] && location[1].includes("-")){
    const empresaID = location[1].split("-")[0]
    if(!isNaN(empresaID as unknown as number)){ 
      return parseInt(empresaID)
    }
  } else if(!isNaN(location[1] as unknown as number)){
    return parseInt(location[1])
  }
  return 0
}

const parseProductos = (productos: IProducto[]) => {
  for(const p of productos){
    p._stock = 0
    for(const stock of p.Stock){ p._stock += (stock.c||0) }
  }
}

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
    const route = makeRoute("p-productos-cms") + `?empresa-id=${getEmpresaID()}`
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
        parseProductos(results.productos||[])
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