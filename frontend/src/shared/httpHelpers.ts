import { keyID, keyUpdated, log1, log2 } from "./http"
import Dexie, { Table } from 'dexie';
import { camelToSnakeCase } from "./main";
import { Notify, Loading } from "~/core/main";

export type httpProps = {
  id?: number // window._params = { fetchID: 1001 }
  route: string
  apiName?: "MAIN"
  errorMessage?: string
  successMessage?: string
  useIndexDBCache?: string
  headers?: { [e: string]: string }
  headersExtra?: { [e: string]: string }
  collections?: { [e: string]: number }
  handleCache?: boolean
  clearCache?: boolean
  cacheMode?: 'offline' | 'refresh' | 'updateOnly' 
  removeCacheGroup?: string[] | string
  partition?: { 
    key: string, value: string | number, param: string, include?: string[]
  }
  emptyValue?: any
  refreshPartition?: (string | number) | (string | number)[]
  refreshIndexDBCache?: string | string[]
  idbTableSchema?: {[e: string]: string | number }
  noUpdateReceived?: boolean
  // mergeTables?: string
  cacheSyncTime?: number
  mergeRequest?: boolean
  data?: any
  filterIdb?: {[e: string]: string | number } | 
    ((tableName: string) => {[e: string]: string | number }),
  updatedQuery?: {[e: string]: string | number }
  status?: { code: number, message: string }
  startTime?: number
  startTimeMs?: number
  recordUpdated?: number
  routeQuery?: string
  makeTransform?: (e: any) => void
  resolver?: (e: any) => void
  rejecter?: (e: any) => void
  onUploadProgress?: (e: any) => void
  localRecordsToDelete?: string[]
  readyForFetch?: number
  resultCached?: any
}

export const getDexieDB = (): Dexie => {
  return window['DexieDB']
}

export class UriMerger {
  // Combina rutas en un solo query
  static mergeRoutesInQuery(routes: string[]){
    const parsedRoutes: string[] = []
    let idx = 0
    for(let route of routes){
      idx++
      const key = 'i--' + idx
      let [parsedRoute,params] = route.split('?')
      parsedRoute = parsedRoute.replace(/\//g,'--')
      parsedRoutes.push(key + '=' + parsedRoute)
      if(params){
        for(let param of params.split('&')){
          const [key,value] = param.split('=')
          parsedRoutes.push(`${idx}--${key}=${value}`)
        }
      }
    }
    const parsedUri = parsedRoutes.join('&')
    return parsedUri
  }
  // Obtiene las rutas de un string query
  static queryToRoutes(query: string, parsed?: boolean){
    const indexToRoute: Map<number,string> = new Map()
    const routes: { [key: string]: string[][] } = {}
    for(let param of query.split('&')){
      const [key,value] = param.split('=')
      if(key.includes('i--')){
        const idx = parseInt(key.replace('i--',''))
        const route = value.replace(/--/g,'/')
        indexToRoute.set(idx,route)
        routes[route] = []
        continue
      }
      else if(key.includes('--')){
        const [mergedKey,value] = param.split('=')
        const [idx,key] = mergedKey.split('--')
        const route = indexToRoute.get(parseInt(idx))
        if(!route){
          console.warn(`No se encontró la routa con idx:: ${idx} ${key}`)
          continue
        }
        routes[route].push([key,value])
      } 
      else {
        console.warn('No se reconocer la key del uri::',key)
      }
    }
    if(parsed){
      const obj:{[e: string]: {[e: string]: string | number}} = {}
      for(let route in routes){
        const params = routes[route]
        obj[route] = {}
        for(let e of params) obj[route][e[0]] = e[1]
      }
      return obj
    } else {
      const obj:{[e: string]: string} = {}
      for(let route in routes){
        const params = routes[route]
        obj[route] = params.map(arr => arr.join('=')).join('&')
      }
      return obj
    }
  }
}

// Obtiene los registros de IndexedDB
export const getRecordsFromIDB = (props: httpProps): Promise<any[]> => {
  const db = getDexieDB()
  const par = props.partition
  const tables = [props.useIndexDBCache]
  const filters = {} as any
  // Los obtiene de la IndexDB 
  const getTable = (table: string) =>{
    // Si el filtro es un objeto
    let filter
    if(typeof props.filterIdb === 'object'){
      filter = props.filterIdb   
    }
    // Si es filtro es una función que toma como parámetro la tabla
    if(typeof props.filterIdb === 'function'){
      filter = props.filterIdb(table)
    }

    // Si hay un partition key y un partition value
    if(par && par.key && par.value &&
      (!par.include || par.include.includes(table))
    ){
      if(!filter) filter = {}
      filter[par.key] = par.value
      return db.table(table).where(filter as {}).toArray()
    }
    filters[table] = filter

    if(!filter) return db.table(table).toArray()
    else return db.table(table).where(filter as {}).toArray()
  }

  return new Promise(resolve => {
    let route = props.route.split("?")[0]
    // Revisa si hay registros cacheados de una busqueda anterior
    if(props.cacheMode !== 'offline' && props.resultCached){
      log1(`Enviando registros pre-cacheados. ${tables.join(", ")} (${route})`)
      resolve(props.resultCached)
      return
    }

    log2(`Enviando desde IndexedDB: ${tables.join(", ")} (${route})`)
    
    // Guarda los datos en la IndexDB
    Promise.all(tables.map(table => getTable(table)))
    .then(results => {
      for(let i=0; i< tables.length; i++){
        const result = results[i] as any[]
        const idx1 = result.findIndex(x => x[keyID] === -1)
        if(idx1 !== -1){ result.splice(idx1,1) }
      }
      if(tables.length === 1) results = results[0] as any[]
      const elapsed = (Date.now() - (props.startTimeMs || 0))
      console.log(`Registros enviados desde IndexDB [${tables.join(',')}] en ${elapsed}ms`)
      for(let k of (props.localRecordsToDelete || [])){ 
        localStorage.removeItem(k)
      }
      resolve(results)
    })
    .catch(error => {
      console.error('Error al obtener los datos de DexieDB....')
      console.warn(error)
      console.warn("Filtros Utilizados::",filters)
      const blankResult = tables.length === 1 ? [] : tables.map(_ => []) 
      resolve(blankResult)
    })
  })
}

// Guarda los registros obtenidos en la IndexedDB
export const saveRecordsToIndexDB = 
async (props: httpProps, results: any[]): Promise<any> => {

  const db = getDexieDB()
  const idbTable = props.useIndexDBCache
  props.collections = {} as {[e: string]: number}

  // Revisa que los resultados sea un array
  if(!Array.isArray(results) && typeof results === 'object'){
    const keys = Object.keys(results).sort()

    const newResults = []
    for(let i = 0; i < keys.length; i++){
      const pk = i + 1
      const records = results[keys[i]] || []
      props.collections[keys[i]] = pk

      if(!Array.isArray(records)){
        console.warn('El objeto resultado no contiene arrays.',results)
        return Promise.reject([])     
      }
      for(let e of records as any[]){
        if(!e){ continue }
        if(Array.isArray(e)){
          console.warn('El array resultado contiene un array (esperando objeto).',records)
          return Promise.reject([])  
        }
        e._pk = pk
        newResults.push(e)
      }
    }
    results = newResults
  }

  // Revisa si ha recibido algún dato
  props.noUpdateReceived = true

  if(!results || !Array.isArray(results)){
    console.warn('No encontró un array[] en el resultado obtenido.', results)
    return Promise.reject([])
  }

  if(results.length > 0){ props.noUpdateReceived = false }
  if(props.noUpdateReceived){
    log1('No se obtuvieron registros nuevos. IndexedDB: ' + idbTable)
  } else {
    delete props.resultCached
  }
  
  // Itera sobre las tablas que corresponden a la data obtenida
  log1(`Registros obtenidos IndexedDB [${idbTable}] = ${results.length}`)

  // Itera sobre las tablas y parsea los registros
  const maxUpdated = (props.startTime || 0) - 20
  let updatedTimeMax = 0

  for(let e of results){
    if(!e) continue
    if(props.makeTransform) props.makeTransform(e)
    const updatedTime = e[keyUpdated]
    
    if(!updatedTime){
      Notify.failure(`El registro ${e.sk || e.id} no posee un campo [${keyUpdated}]. Tabla : ${idbTable}.`)
      console.log("Registros sin campo::",keyUpdated,results)
      return Promise.reject(null)
    }
    if(updatedTime > updatedTimeMax && updatedTime < maxUpdated){
      updatedTimeMax = updatedTime
    }
  }
  
  log1(`Registros a guardar IndexedDB [${idbTable}]: ${results.length}`)
  // Se agrega el objeto updated aunque no se haya recibido ningún objeto actualizado para guardar el momento de última de sincronización
  const updatedRecord = {...props.idbTableSchema, 
    collections: props.collections, upd: 0, _IS_META: true}
  updatedRecord[keyUpdated] = props.startTime
  results.unshift(updatedRecord)
  
  // Retorna la promesa que sigue luego del ingreso de la data
  try {
    await db.table(idbTable).bulkPut(results)
    // Aquí calcula el la cantidad de registros totales sumando los existentes
    const countByTable = []
    let keys = []
    let recordsCount = results.length
    keys.push(idbTable)
    countByTable.push([idbTable, results.length])
    recordsCount += results.length

    const reg = {
      key: keys.join("|"), countByTable, recordsCount, updatedTimeMax,
      syncCount: 1, updatedOn: Date.now(),
      tables: [idbTable]
    }

    // Revisa si hay un registros anterior con la misma key
    const prevReg = await db.table("fetchedIDBTables").get(reg.key)
    if(prevReg){
      for(let i = 0; i<prevReg.countByTable.length; i++){
        reg.countByTable[i][1] += prevReg.countByTable[i][1]
      }
      reg.recordsCount += prevReg.recordsCount
      reg.syncCount += prevReg.syncCount
      if(!updatedTimeMax){ reg.updatedTimeMax = prevReg.updatedTimeMax }
    }
    // Agrega el registro de con la metadad de la tabla
    await db.table("fetchedIDBTables").put(reg)

    return 1
  } catch (error) {
    console.warn('Hubo un error en el guardado de las Tablas',error)
    console.log('Detalle del Error:: ',props)
    console.log('Error::Indexed-DB Table: ', idbTable, " | Regs: ",results)
    return 0
  }
}

interface IAddToFetchedIDBTables {
  tables: string[], recordsCount: number, 
  key?: string, syncCount?: number, updatedOn?: number, updatedTimeMax?: number
}

export const addToFetchedIDBTables = async (args: IAddToFetchedIDBTables) => {
  const key = args.tables.map(x => camelToSnakeCase(x)).join("|")
  const db = window.DexieDB as Dexie
  args.key = key
  const prevReg = await db.table("fetchedIDBTables").get(key)
  if(prevReg){
    args.recordsCount += prevReg.recordsCount
    args.syncCount += prevReg.syncCount
  } else {
    args.syncCount = 1
  }
  args.updatedOn = Date.now()
  args.updatedTimeMax = Math.floor(Date.now()/1000)

  await db.table("fetchedIDBTables").put(args)
}