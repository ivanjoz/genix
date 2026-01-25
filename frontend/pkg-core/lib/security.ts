import { Env, IsClient, LocalStorage } from '$core/env';
import { decrypt, Notify, throttle } from '$core/helpers';
import type { IUsuario, ILoginResult } from '$core/types/common';

// Token refresh constants (all in seconds)
const TOKEN_REFRESH_THRESHOLD = 40 * 60 // 40 minutes in seconds
const TOKEN_CHECK_INTERVAL = 4 * 60 // 4 minutes in seconds
const REFRESH_LOCK_DURATION = 30 // 30 seconds lock duration
const REFRESH_LOCK_KEY = Env.appId + "TokenRefreshLock"

// Global registry for reloadLogin function to avoid circular dependencies
let reloadLoginFn: () => Promise<any> = async () => {
  console.warn('reloadLogin function not registered');
};

export const registerReloadLogin = (fn: () => Promise<any>) => {
  reloadLoginFn = fn;
};

interface UserInfo {
  d: number // userID
  u: string // usuario
  r: number[] // roles ids
  m: number[] // modules ids
  a: number[] // accesos ids
}

export interface UserInfoParsed {
  id: number
  user: string
  email: string
  names: string
}

// Función para obtener el Token
export const getToken = (noError?: boolean) => {
  const userToken = LocalStorage.getItem(Env.appId + "UserToken")
  const expTime = parseInt(LocalStorage.getItem(Env.appId + "TokenExpTime") || '0')
  const nowTime = Math.floor(Date.now()/1000)

  if (!userToken) {
    console.error('No se encontró la data del usuario. ¿Está logeado?:',Env.appId)
    return ""
  }
  else if (!expTime || nowTime > expTime) {
    if(!noError){
      Notify.failure('La sesión ha expirado, vuelva a iniciar sesión.')
      Env.clearAccesos?.()
    }
    return ""
  }
  // revisa si la sesión está por expirar
  if ((expTime - nowTime) < (60 * 15)) {
    throttle(() => { Notify.warning(`La sesión expirará en 15 minutos`) }, 20)
  }
  else if ((expTime - nowTime) < (60 * 5)) {
    throttle(() => { Notify.warning(`La sesión expirará en 5 minutos`) }, 20)
  }
  return userToken || ""
}

export const checkIsLogin = () => {
  // if(Env.getEmpresaID() > 0){ return 1 }
  if(!IsClient()){ return 0 }
  else if(IsClient() && !!getToken(true)){ return 2 }
  else { return 3 }
}

export const isLogged = (): boolean => {
  return Env.getEmpresaID() > 0 && getToken(true)?.length > 0
}

export const getShowStore = (pathname?: string): number => {
  pathname = pathname || Env.getPathname()
  console.log("evaluando pathname::", pathname)
  if(pathname === "/login"){
    return -1
  } else if(pathname === "/" || pathname.substring(0,5) === "/page"){
    return Env.getEmpresaID() || -1
  }
  return 0
}

// TOKEN REFRESH MANAGEMENT
const acquireRefreshLock = (): boolean => {
  if (!IsClient()) return false

  const lockTime = parseInt(LocalStorage.getItem(REFRESH_LOCK_KEY)||"0")
  const nowUnix = Math.floor(Date.now() / 1000)
  // Check if lock has expired
  if (lockTime && (nowUnix - lockTime < REFRESH_LOCK_DURATION)) {
    console.log('Token refresh already in progress in another tab')
    return false
  }

  // Acquire lock
  LocalStorage.setItem(REFRESH_LOCK_KEY, String(nowUnix))
  return true
}

// Check if token needs refresh
const shouldRefreshToken = (): boolean => {
  const tokenCreated = parseInt(LocalStorage.getItem(Env.appId + "TokenCreated") || '0')
  const tokenAge =  Math.floor(Date.now() / 1000) - tokenCreated

  return tokenCreated > 0 && tokenAge >= TOKEN_REFRESH_THRESHOLD
}

// Token refresh checker - will be called every 4 minutes
let tokenRefreshInterval: number | null = null

const checkAndRefreshToken = async () => {
  if (!IsClient()) return

  // Check if user is still logged in
  if (!getToken(true)) {
    stopTokenRefreshCheck()
    return
  }

  // Check if token needs refresh and Try to acquire lock to prevent parallel execution
  if (!shouldRefreshToken() || !acquireRefreshLock()) { return }
  console.log('Token refresh initiated - token is older than 40 minutes')

  try {
    await reloadLoginFn()
    console.log('Token refreshed successfully')
  } catch (error) {
    console.error('Error refreshing token:', error)
  } finally {
    LocalStorage.removeItem(REFRESH_LOCK_KEY) // Release the lock
  }
}

// Start the token refresh check interval
export const startTokenRefreshCheck = () => {
  if (!IsClient()) return

  // Clear any existing interval
  if (tokenRefreshInterval !== null) { clearInterval(tokenRefreshInterval) }

  // Start new interval (convert seconds to milliseconds for setInterval)
  tokenRefreshInterval = window.setInterval(checkAndRefreshToken, TOKEN_CHECK_INTERVAL * 1000)
  console.log('Token refresh check started - will check every 4 minutes')
}

// Stop the token refresh check interval
export const stopTokenRefreshCheck = () => {
  if (tokenRefreshInterval !== null) {
    clearInterval(tokenRefreshInterval)
    tokenRefreshInterval = null
    console.log('Token refresh check stopped')
  }
}

// Clear accesos and logout
Env.clearAccesos = () => {
  if(!IsClient){ return }
  stopTokenRefreshCheck()
  LocalStorage.removeItem(Env.appId+ "Accesos")
  LocalStorage.removeItem(Env.appId+ "UserInfo")
  LocalStorage.removeItem(Env.appId+ "UserToken")
  LocalStorage.removeItem(Env.appId+ "TokenExpTime")
  LocalStorage.removeItem(Env.appId+ "TokenCreated")
  LocalStorage.removeItem(REFRESH_LOCK_KEY)
  Env.navigate("/login")
}

// Initialize token refresh check on page load if user is logged in
export const initTokenRefreshCheck = () => {
  if (!IsClient()) return

  // Check if user is logged in
  const hasToken = getToken(true)
  const tokenCreated = LocalStorage.getItem(Env.appId + "TokenCreated")

  if (hasToken && tokenCreated) {
    startTokenRefreshCheck()
  }
}

// Auto-initialize on client side
if (IsClient()) {
  // Wait a bit for the app to initialize (1 second)
  setTimeout(initTokenRefreshCheck, 1 * 1000)
}



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
  #userInfo: IUsuario = null as unknown as IUsuario

  #setUserInfo(){
    const userInfoJson = LocalStorage?.getItem(Env.appId+ "UserInfo")
    if(!userInfoJson){ return }
    this.#userInfo = JSON.parse(userInfoJson)
  }
  clearAccesos = Env.clearAccesos
  getUserInfo(){ return this.#userInfo }
  setUserInfo(userInfo: IUsuario){
    this.#userInfo = userInfo
    LocalStorage.setItem(Env.appId + "UserInfo", JSON.stringify(userInfo))
  }

  async parseAccesos(login: ILoginResult, cipherKey?: string) {
    debugger
    const userInfoStr = await decrypt(login.UserInfo, cipherKey as string)
    const userInfo = JSON.parse(userInfoStr) as IUsuario

    const rolesIDsParsed = (userInfo.rolesIDs||[]).map(x => x * 10 + 8)
    const accesosIDs = (userInfo.accesosIDs||[]).concat(rolesIDsParsed)

    const UnixTime = Math.floor(Date.now()/1000)
    LocalStorage.setItem(Env.appId + "TokenCreated", String(UnixTime))
    LocalStorage.setItem(Env.appId + "UserInfo", JSON.stringify(userInfo))
    LocalStorage.setItem(Env.appId + "UserToken", login.UserToken)
    // unix time un seconds expiration
    LocalStorage.setItem(Env.appId + "TokenExpTime", String(login.TokenExpTime))
    LocalStorage.setItem(Env.appId + "EmpresaID", String(login.EmpresaID))
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
    LocalStorage.setItem(Env.appId+ "Accesos", hashParsed)
    debugger
    // Start token refresh check after successful login
    startTokenRefreshCheck()
  }

  checkAcceso(accesoID: number, nivel?: number) {
    nivel = nivel || 1
    const b32ls = this.#b32ls
    if(this.#accesos === ""){
      this.#accesos = LocalStorage.getItem(Env.appId + "Accesos") || ""
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
    return this.checkAcceso(roleID as number, 8)
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
    LocalStorage.setItem(key, String(value))
  },
  getValue(key: string): string {
    return LocalStorage.getItem(key) || ''
  },
  getValueInt(key: string | number): number {
    const value: string | number = LocalStorage.getItem(String(key)) || '0'
    return parseInt(value as string)
  },
  getFechaUnix(){
    return Math.floor(((Date.now()/1000) - Env.zoneOffset) / 86400)
  },
  toSunix(fechaHoraUnix: number){
    return Math.floor((fechaHoraUnix - (10**9)) / 2)
  },
  sunixTime(){
    const fechaHora = Math.floor(Date.now()/1000)
    return Params.toSunix(fechaHora)
  }
}
