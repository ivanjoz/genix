import { decrypt, throttle } from "./main"
import { ILoginResult } from "~/services/admin/login"
import { Loading, Notify } from "~/core/main"
import { createSignal } from "solid-js"

interface UserInfo {
  d: number // userID
  u: string // usuario
  r: number[] // roles ids
  m: number[] // modules ids
  a: number[] // accesos ids
}

interface UserInfoParsed {
  id: number
  user: string
  email: string
  names: string
}

const isClient = typeof window !== 'undefined'
if(isClient){ window.appId = "genix" }
const wd = typeof window !== 'undefined' ? window : { appId: "", _zoneOffset: 0 }

export const localStorage = typeof window !== 'undefined' 
  ? window.localStorage
  : {
      getItem: (k: string) => { return "" },
      setItem: (k: string, v: string) => { return "" },
      removeItem: (k: string) => { return "" }
    } as any

const clearAccesos = () => {
  if(typeof window !== 'undefined'){
    localStorage.removeItem(window.appId+ "Accesos")
    localStorage.removeItem(window.appId+ "UserInfo")
    setLoginStatus(false)
  }
}

// Función para obtener el Token
export const getToken = (noError?: boolean) => {
  const userToken = localStorage.getItem(window.appId + "UserToken")
  const expTime = parseInt(localStorage.getItem(window.appId + "TokenExpTime") || '0')
  const nowTime = Math.floor(Date.now()/1000)

  if (!userToken) {
    console.error('No se encontró la data del usuario. ¿Está logeado?:',window.appId)
    return
  }
  else if (!expTime || nowTime > expTime) {
    if(!noError){
      Notify.failure('La sesión ha expirado, vuelva a iniciar sesión.')
      clearAccesos()
    }
    Loading.remove()
    return
  }
  // revisa si la sesión está por expirar
  if ((expTime - nowTime) < (60 * 15)) {
    throttle(() => { Notify.warning(`La sesión expirará en 15 minutos`) }, 20)
  }
  else if ((expTime - nowTime) < (60 * 5)) {
    throttle(() => { Notify.warning(`La sesión expirará en 5 minutos`) }, 20)
  }
  return userToken
}

export const [loginStatus, setLoginStatus] = createSignal(isClient && !!getToken(true))

export class AccessHelper {
  constructor() {
    const b32l = []
    for (let i = 32; i < 36; i++) { b32l.push(i.toString(36)) }
    this.#b32l = b32l
    this.#b32ls = b32l.join(',')
    this.#setUserInfo()
  }

  #b32l: string[] = []
  #b32ls = ''
  #avoidCheckSum = false
  #accesos = ''
  #cachedResults: Map<string,boolean> = new Map()
  #userInfo: UserInfoParsed
  
  #setUserInfo(){
    const userInfoJson = localStorage?.getItem(wd.appId+ "UserInfo")
    if(!userInfoJson){ return }
    this.#userInfo = JSON.parse(userInfoJson)
  }
  clearAccesos = clearAccesos
  getUserInfo(){ return this.#userInfo }

  async parseAccesos(login: ILoginResult, cipherKey?: string) {
    const userInfoStr = await decrypt(login.UserInfo, cipherKey)
    const userInfo: UserInfo = JSON.parse(userInfoStr)
    let accesosIDs = [...(userInfo.a||[])]
    accesosIDs = accesosIDs.concat((userInfo.r||[]).map(x => x * 10 + 8))

    const userInfoParsed: UserInfoParsed = { 
      id: userInfo.d, user: userInfo.u, email: login.UserEmail, names: login.UserNames
    }
    
    localStorage.setItem(wd.appId + "UserInfo", JSON.stringify(userInfoParsed))
    localStorage.setItem(wd.appId + "UserToken", login.UserToken)
    localStorage.setItem(wd.appId + "TokenExpTime", String(login.TokenExpTime))
    this.#setUserInfo()
    
    const b32l = this.#b32l
    let parsedAccesos = b32l[b32l.length - 1]
    let i = 0
    for(let e of accesosIDs.map(x => x.toString(32))) {
      if(i > 3){  i = 0 }
      parsedAccesos += e
      parsedAccesos += b32l[i]
      i++
    }
    const hash = checksum(parsedAccesos)
    const hashParsed = `${hash.substring(0, 2)}${parsedAccesos}${hash.substring(2, 4)}`
    localStorage.setItem(wd.appId+ "Accesos", hashParsed)
  }

  checkAcceso(accesoID: number, nivel?: number) {
    nivel = nivel || 1
    const b32ls = this.#b32ls
    if(this.#accesos === ""){
      this.#accesos = localStorage.getItem(wd.appId + "Accesos") || ""
    }

    if(!this.#accesos) return false

    if (!this.#avoidCheckSum) {
      const passCheckSum = this.#checkAccesosCheckSum()
      if (!passCheckSum) {
        console.warn('Los accesos han sido modificados.');
        return false
      }
      // Evita que el checksum se calcule seguidamente
      this.#avoidCheckSum = true
      setTimeout(() => { this.#avoidCheckSum = false }, 50)
    }

    const check = (niveles: number[]) => {
      for (let nivel of niveles) {
        const code = (accesoID * 10 + nivel).toString(32)
        if(!this.#cachedResults.has(code)){
          const hasAccess = (new RegExp(`[${b32ls}]${code}[${b32ls}]`)).test(this.#accesos)
          this.#cachedResults.set(code,hasAccess)
        }
        const hasAccess = this.#cachedResults.get(code)
        if(hasAccess){ return hasAccess }
      }
      return false
    }

    if (!nivel || nivel === 1) return check([1, 7]) // VER
    else if (nivel === 2) return check([2, 7]) // CREA
    else if (nivel === 3) return check([3, 7]) // EDITA
    else if (nivel === 4) return check([4, 7]) // ELIMINA
    else if (nivel === 8) return check([8]) // ES UN ROL
    else return check([7])
  }

  checkRol(roleID?: number) {
    return this.checkAcceso(roleID, 8)
  }

  #checkAccesosCheckSum() {
    const accesos = this.#accesos
    const accesosInternal = accesos.substring(2, accesos.length - 2)
    const cks = accesos.substring(0, 2) + accesos.substring(accesos.length - 2)
    return (cks === checksum(accesosInternal))
  }
}

export const checksum = (string: string): string => {
  let seed = 888888
  for (let i = 0; i < string.length; i++) {
    const in3 = i % 1000
    const char = string[i]
    const code = char.charCodeAt(0)
    const ld = Math.abs(seed - code + in3) % 10
    if (ld > 6) seed += ((code + in3) * ld) + ld
    else if (ld > 3) seed -= ((code - in3) * ld) - in3
    else {
      seed += Math.abs((code - in3) * (ld + 1)) - (ld * (i % 10))
      if (seed >= 1000000) seed = Math.abs(seed) % 100000
    }
  }
  const rs = String(seed % 1000)
  for (let i = 0; i < rs.length; i++) {
    seed += Math.pow(parseInt(rs[0]), (6 - i))
  }
  let seedT = seed.toString(32).split('').reverse().join('')
  if (seedT.length > 4) seedT = seedT.substring(0, 4)
  else if (seedT.length < 4) {
    for (let i = seedT.length; i < 4; i++) { seedT += String(4 - i) }
  }
  return seedT
}

export const accessHelper = new AccessHelper()

// Params
export const Params = {
  checkAcceso: (accesoID: number, nivel?: number) => accessHelper.checkAcceso(accesoID,nivel),
  checkRol: (a: number) => accessHelper.checkRol(a),
  userInfo: () => accessHelper.getUserInfo(),
  setValue(key: string, value: string | number) {
    localStorage.setItem(key, String(value))
  },
  getValue(key: string): string {
    return localStorage.getItem(key) || ''
  },
  getValueInt(key: string): number {
    const value: string | number = localStorage.getItem(key) || '0'
    return parseInt(value as string)
  },
  getFechaUnix(){
    return Math.floor(((Date.now()/1000) - wd._zoneOffset) / 86400)
  }
}
