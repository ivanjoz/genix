import { Env, IsClient, LocalStorage } from '$core/env';
import { decrypt, Notify, throttle } from '$libs/helpers';
import type { IUsuario, ILoginResult } from '$core/types/common';
import { base64ToUInt16, checksum } from '$libs/funcs/parsers';
import { getAccessEntriesForRoute } from '../routes/configuracion/perfiles-accesos/access-list-catalog';

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
    if(!noError){ console.error('No se encontró la data del usuario. ¿Está logeado?:', Env.appId) }
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

export const isPublicFrontendRoute = (routeValue?: string | null): boolean => {
  const normalizedRoute = String(routeValue || "").trim()
  return normalizedRoute === '/' || normalizedRoute === '/login' || normalizedRoute.startsWith('/store')
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
  if(!IsClient()){ return }
  stopTokenRefreshCheck()
  accessHelper.clearStoredAccesosState()
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
    this.#loadAccesosComputedFromStorage()
    this.#setUserInfo()
  }

  #storedAccesos = ''
  #accesosComputed = new Uint16Array()
  #cachedResults: Map<number,boolean> = new Map()
  #userInfo: IUsuario = null as unknown as IUsuario

  #loadAccesosComputedFromStorage() {
    const storedAccesosComputed = LocalStorage.getItem(Env.appId + "Accesos") || ""
    if (storedAccesosComputed === this.#storedAccesos) {
      return
    }

    this.#storedAccesos = storedAccesosComputed
    this.#accesosComputed = new Uint16Array(decodeStoredAccesosComputed(storedAccesosComputed))
    this.#cachedResults.clear()
  }

  #setUserInfo(){
    const userInfoJson = LocalStorage?.getItem(Env.appId+ "UserInfo")
    if(!userInfoJson){
      this.#userInfo = null as unknown as IUsuario
      return
    }
    this.#userInfo = JSON.parse(userInfoJson)
  }
  clearStoredAccesosState(){
    this.#storedAccesos = ""
    this.#accesosComputed = new Uint16Array()
    this.#cachedResults.clear()
  }
  clearAccesos = Env.clearAccesos
  getUserInfo(){ return this.#userInfo }
  isTokenValid() {
    const empresaID = parseInt(LocalStorage.getItem(Env.appId + "EmpresaID") || "0")
    const tokenValue = getToken(true)
		const accesosComputed = decodeStoredAccesosComputed(LocalStorage.getItem(Env.appId + "Accesos") || "")
		debugger
    return empresaID > 0 && tokenValue.length > 0 && accesosComputed.length > 0
  }
  setUserInfo(userInfo: IUsuario){
    this.#userInfo = userInfo
    LocalStorage.setItem(Env.appId + "UserInfo", JSON.stringify(userInfo))
  }

  async parseAccesos(login: ILoginResult, cipherKey?: string) {
    const userInfoStr = await decrypt(login.UserInfo, cipherKey as string)
    const userInfo = JSON.parse(userInfoStr) as IUsuario

    const UnixTime = Math.floor(Date.now()/1000)
    LocalStorage.setItem(Env.appId + "TokenCreated", String(UnixTime))
    LocalStorage.setItem(Env.appId + "UserInfo", JSON.stringify(userInfo))
    LocalStorage.setItem(Env.appId + "UserToken", login.UserToken)
    // unix time un seconds expiration
    LocalStorage.setItem(Env.appId + "TokenExpTime", String(login.TokenExpTime))
    LocalStorage.setItem(Env.appId + "EmpresaID", String(login.EmpresaID))
    LocalStorage.setItem(Env.appId + "Accesos", wrapAccesosComputed(login.AccesosComputed || ""))
    this.#setUserInfo()
    this.#loadAccesosComputedFromStorage()
    // Start token refresh check after successful login
    startTokenRefreshCheck()
  }

  checkAcceso(accesoID: number, nivel?: number) {
    this.#loadAccesosComputedFromStorage()
    if (!this.#accesosComputed.length || accesoID <= 0) {
      return false
    }

    const requestedNivel = normalizeAccesoNivel(nivel)
    const cacheKey = accesoID * 10 + requestedNivel
    if (!this.#cachedResults.has(cacheKey)) {
      const [rangeStart, rangeEnd] = getAccesoNivelSearchRange(accesoID, requestedNivel)
      this.#cachedResults.set(cacheKey, hasPackedAccesoInRange(this.#accesosComputed, rangeStart, rangeEnd))
    }

    return this.#cachedResults.get(cacheKey) || false
  }

  canUserAccessRoute(routeValue?: string | null): boolean {
    const route = String(routeValue || '').trim() || '/'
    const normalizedRoute = route.replace(/^\//, '')

    if (isPublicFrontendRoute(route)) { return true }

    const matchedAccessEntries = getAccessEntriesForRoute(route)
    if (matchedAccessEntries.length === 0) {
      return true
		}

		return matchedAccessEntries.some((accessEntry) => {
      return this.checkAcceso(accessEntry.id, 1)
    })
  }
}

const normalizeAccesoNivel = (nivel?: number): number => {
  if (!nivel || nivel < 1 || nivel > 4) {
    return 1
  }

  return nivel
}

const makeAccesoNivelUint16 = (accesoID: number, nivel: number): number => {
  // Mirror the backend bit-packing so route checks behave exactly the same on both sides.
  return ((accesoID << 2) | (normalizeAccesoNivel(nivel) - 1)) >>> 0
}

const getAccesoNivelSearchRange = (accesoID: number, nivel: number): [number, number] => {
  // Require granted levels to be >= the requested level within the same access bucket.
  const normalizedNivel = normalizeAccesoNivel(nivel)
  const rangeStart = makeAccesoNivelUint16(accesoID, normalizedNivel)
  const rangeEnd = makeAccesoNivelUint16(accesoID, 4)
  return [rangeStart, rangeEnd]
}

const hasPackedAccesoInRange = (accesosComputed: Uint16Array, rangeStart: number, rangeEnd: number): boolean => {
  let leftIndex = 0
  let rightIndex = accesosComputed.length - 1

  while (leftIndex <= rightIndex) {
    const middleIndex = (leftIndex + rightIndex) >> 1
    const middleValue = accesosComputed[middleIndex]

    if (middleValue < rangeStart) {
      leftIndex = middleIndex + 1
    } else if (middleValue > rangeEnd) {
      rightIndex = middleIndex - 1
    } else {
      return true
    }
  }

  return false
}

const wrapAccesosComputed = (packedAccesosBase64: string): string => {
  if (!packedAccesosBase64) {
    return ""
  }

  const packedAccessHash = checksum(packedAccesosBase64)
  return `${packedAccessHash.substring(0, 2)}${packedAccesosBase64}${packedAccessHash.substring(2, 4)}`
}

const decodeStoredAccesosComputed = (storedAccesosComputed: string): Uint16Array => {
  if (!storedAccesosComputed) {
    return new Uint16Array()
  }
  if (storedAccesosComputed.length < 5) {
    console.warn("[AccessHelper] invalid accesos payload")
    return new Uint16Array()
  }

  const packedAccesosComputedBase64 = storedAccesosComputed.substring(2, storedAccesosComputed.length - 2)
  const storedHash = storedAccesosComputed.substring(0, 2) + storedAccesosComputed.substring(storedAccesosComputed.length - 2)
  if (checksum(packedAccesosComputedBase64) !== storedHash) {
    console.warn("[AccessHelper] invalid accesos payload")
    return new Uint16Array()
  }
  return base64ToUInt16(packedAccesosComputedBase64)
}

export const accessHelper = new AccessHelper()
export const canUserAccessRoute = (routeValue?: string | null) => accessHelper.canUserAccessRoute(routeValue)
export const isTokenValid = () => accessHelper.isTokenValid()

// Params
export const Params = {
  checkAcceso: (accesoID: number, nivel?: number) => accessHelper.checkAcceso(accesoID,nivel),
  canUserAccessRoute: (routeValue?: string | null) => accessHelper.canUserAccessRoute(routeValue),
  isTokenValid: () => accessHelper.isTokenValid(),
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
