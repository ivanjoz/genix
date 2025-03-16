import Dexie from "dexie";
import { Loading, Notify } from "~/core/main";
import { IModule } from "~/core/modules";
import { Env, Params, checksum } from "./security";

export const joinb = (...args: any[]) => {
  return args.filter(x => x).join(" ")
}

export const camelToSnakeCase = (str: string) => {
  const arr = str.split(/(?=[A-Z0-9])/)
  let final = ""; let lastLeng = 0
  for(let letters of arr){ 
    if(lastLeng > 1) final += "_"; final += letters.toLowerCase()
    lastLeng = letters.length
  }
  return final
}

export function blobToBase64(file: Blob) {
  const reader = new FileReader()
  reader.readAsDataURL(file)
  return new Promise((resolve) => {
    reader.onload = () => {
      let encoded = (reader.result as string).replace(/^data:(.*,)?/, '')
      if((encoded.length % 4) > 0){
        encoded += '='.repeat(4 - (encoded.length % 4))
      }
      resolve(encoded);
    }
  })
}

export const uint32ToBase64 = (u8: Uint32Array) => {
  const decoder = new TextDecoder('utf8')
  return btoa(decoder.decode(u8))
}

export const makeRamdomString = (length: number) => {
  let str = ""
  while(str.length < length){
    str += (Math.random() + 1).toString(36).substring(2)
  }
  return str.substring(0,length)
}

const separators = new Set(["-","_"])
const capitals2: {[e: string]: string} = { 
  'á': 'a', 'é':'e', 'í':'i', 'ó':'o', 'ú':'u',
  '/': '_' , 'ñ':'n', '.':'.', '-':'-' 
}

export function normalizeStringN(string: string): string {
  if(typeof string === 'number') {
    return String(string)
  } else if (typeof string !== 'string') {
    return ''
  }
  let normalized = ""
  let charLast = ""
  const words = string.trim().toLowerCase().split(" ")
  for(let i = 0; i < words.length; i++){
    const word = words[i].trim()
    if(i > 0 && normalized && !separators.has(charLast)){ 
      normalized += "_";  charLast = "_" 
    }

    for(let i = 0; i < word.length; i++) {
      const char = word[i]
      let charToAdd
      if(separators.has(charLast) && separators.has(char)) continue

      const code = char.charCodeAt(0)
      if((code > 47 && code < 58) || (code > 96 && code < 123) || code == 95){
        charToAdd = char
      } else if(typeof capitals2[char] === 'string'){
        charToAdd = capitals2[char]
      }
      if(charToAdd){
        if(separators.has(charLast) && separators.has(charToAdd)) continue
        normalized += charToAdd
        charLast = charToAdd
      }
    }
  }
  return normalized
}

export function formatMo(moneda: number): string {
  if(typeof moneda !== 'number'){ return "" }
  return formatN(moneda / 100, 2)
}

export function formatN(
  x: number, decimal?: number, fixedLen?: number, charF?: string
){
  decimal = decimal || 0
  if (typeof x !== 'number') return x ? '-' : ''
  
  if(decimal === -1){
    if(x < 1) x = Math.round(x*10000)/10000  
    else if(x < 10) x = Math.round(x*1000)/1000
    else if(x >= 10) x = Math.round(x*100)/ 100
  }
  
  let xString
  if(typeof decimal === 'number' && decimal >= 0){
    if(decimal === 0){
      xString = Math.round(x).toString()
    } else {
      const pow = Math.pow(10, decimal)
      xString = (Math.round(x * pow) / pow).toFixed(decimal)
    }
  }
  else xString = x.toString()
  if(x >= 100) xString = xString.replace(/\B(?=(\d{3})+(?!\d))/g, ",")
  if(fixedLen){
    charF = charF || ' '
    while (xString.length < fixedLen) { xString = charF + xString }
  }
  return xString
}

export const decrypt = async (encryptedString: string, key: string) => {
  key = key.substring(0,32)
  // Decode the base64 string
  const encryptedData = Uint8Array.from(atob(encryptedString), c => c.charCodeAt(0))
  console.log("encrypted len::", encryptedData.length)

  // Convert the key to Uint8Array
  const keyBuffer = await crypto.subtle.importKey(
    "raw", new TextEncoder().encode(key), { name: "AES-GCM" }, false, ["decrypt"]
  )

  // Ensure the encrypted data is not too short
  if (encryptedData.length < 12) {
    throw new Error("Invalid encrypted data");
  }

  // Extract the nonce from the first 12 bytes
  const nonce = encryptedData.slice(0, 12)
  console.log("nonce:: " ,new TextDecoder().decode(nonce))

  const ciphertext = encryptedData.slice(12)

  console.log("desencriptando:: ", nonce.length, key.length, ciphertext.length)

  // Decrypt the data
  let decryptedData: ArrayBuffer
  try {
    decryptedData = await crypto.subtle.decrypt(
      { name: "AES-GCM", iv: nonce }, keyBuffer, ciphertext
    )
  } catch (error) { 
    console.log("Error desencriptando:: ", error)
    return ""
  }

  console.log("decripted data:: ", decryptedData)
  // Convert the decrypted data to a string
  const decryptedString = new TextDecoder().decode(decryptedData)
  return decryptedString
}

export const makeB64UrlEncode = (contentString: string): string => {
	contentString = contentString.replaceAll("/", "_")
  contentString = contentString.replaceAll("+", "-")
  contentString = contentString.replaceAll("=", "~")
	return contentString
}

export const makeB64UrlDecode = (contentString: string): string => {
	contentString = contentString.replaceAll("_", "/")
  contentString = contentString.replaceAll("-", "+")
  contentString = contentString.replaceAll("~", "=")
	return contentString
}

export const createIndexDB = (modules: IModule[]): Promise<any> => {
  const dbName = Env.appId
  const dexieVersion = Env.dexieVersion
  const lastDexieVersion = Params.getValueInt('dexieVersion')
  const indexedDBTables: {[m: string]: string } = {}
  for(let module of modules){
    for(let key in module.indexedDBTables){
      indexedDBTables[key] = module.indexedDBTables[key]
    }
  }
  
  const hash = checksum(JSON.stringify(indexedDBTables))
  const prevhash = Params.getValue(`dexie_idb_${dexieVersion}`)
  if (!prevhash) Params.setValue(`dexie_idb_${dexieVersion}`, hash)

  // IndexedDB principal
  let db = window.DexieDB as Dexie
  if (!db) {
    db = window.DexieDB = new Dexie(dbName)
    if (dexieVersion !== lastDexieVersion) {
      Loading.standard('Limpiando Datos...')
      db.delete().then(() => {
        Params.setValue('dexieVersion', dexieVersion)
        Params.setValue(`dexie_idb_${dexieVersion}`, hash)
        setTimeout(() => { Loading.remove(); window.location.reload() }, 100)
      })
    }
    else if (prevhash && hash !== prevhash) {
      Notify.warning(`El esquema de la Base de Datos local ha cambiando, re-inicializando...`)
      db.delete().then(() => {
        Params.setValue(`dexie_idb_${dexieVersion}`, hash)
        setTimeout(() => { Loading.remove(); window.location.reload() }, 100)
      })
    }
    else {
      db.version(dexieVersion).stores(indexedDBTables)
      Params.setValue('dexieVersion', dexieVersion)
    }
  }

  return new Promise((resolve, reject) => {
    if (db.isOpen()) resolve(true)
    else {
      db.open().then(() => {
        console.log('Base de datos IndexDB inicializada.')
        resolve(true)
      })
      .catch(error => { console.warn(error); reject(error) })
    }
  })
}

const DEXIE_ECOMMERCE_VERSION = 1

export const makeEcommerceDB = (): Promise<Dexie> => {
  if(typeof window === 'undefined'){ return null }
  // IndexedDB Ecommerce
  let db = window.DexieEcommerceDB as Dexie
  if(!db){
    db = window.DexieEcommerceDB = new Dexie("ecommerce")
    db.version(DEXIE_ECOMMERCE_VERSION).stores({
      cache: "key", forms: "[formID+key],formID"
    })
  }

  return new Promise((resolve, reject) => {
    if (db.isOpen()){ resolve(db) }
    else {
      db.open().then(() => {
        console.log('Base de datos IndexDB inicializada.')
        resolve(db)
      })
      .catch(error => { console.warn(error); reject(error) })
    }
  })
}

export const getCmsTable = async (name: string): Promise<Dexie.Table> => {
  const db = await makeEcommerceDB()
  return await db.table(name)
}

export const arrayToMapS = <T>(array: T[], keys?: string | string[]):
  Map<string, T> => {
  const map = new Map()
  if (typeof keys === 'string') { for (let e of array) { map.set(e[keys], e) } }
  else if (Array.isArray(keys)) {
    for (let e of array) {
      const keyGrouped = keys.map(key => (e[key] || "0")).join("_")
      map.set(keyGrouped, e)
    }
  }
  else { console.warn('No es un array::', array) }
  return map
}


export const arrayToMapN = <T>(array: T[], keys?: string | string[]):
  Map<number, T> => {
  const map = new Map()
  if (typeof keys === 'string') { 
    for (let e of array) { map.set(e[keys  as keyof T], e) } 
  }
  else if (Array.isArray(keys)) {
    for (let e of array) {
      const keyGrouped = keys.map(key => (e[key as keyof T] || "")).join("_")
      map.set(keyGrouped, e)
    }
  }
  else { console.warn('No es un array::', array) }
  return map
}


export const arrayToMapG = <T>(array: T[], keys?: string | string[]):
  Map<(string | number), T[]> => {
  const map = new Map()
  if (typeof keys === 'string') {
    for (let e of array||[]) {
      const keyValue = e[keys as keyof T]
      map.has(keyValue) ? map.get(keyValue).push(e) : map.set(keyValue, [e])
    }
  }
  else if (Array.isArray(keys)) {
    for (let e of array) {
      const keyGrouped = keys.map(key => (e[key  as keyof T] || "")).join("_")
      map.has(keyGrouped) ? map.get(keyGrouped).push(e) : map.set(keyGrouped, [e])
    }
  } else {
    console.warn('No es un array::', array)
  }
  return map
}