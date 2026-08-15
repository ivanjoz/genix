import { GET, POST, security } from '$libs/ui-runtime.svelte';
import type { IUser, ILoginResult } from '$core/types/common';
import { Env } from '$core/env';
import { getStaticRecordsByID } from '@genix/ui/cache';

export interface ILogin {
  CompanyID: number
  User: string
  Password: string
  CipherKey: string
}

export interface IPublicCompanyName {
  ID: number
  Name: string
}

// Company names are immutable for this login cache: memory → IndexedDB → selected server.
export const getPublicCompanyName = async (companyID: number): Promise<IPublicCompanyName | undefined> => {
  const apiRoute = 'p-company-names-by-ids'
  const companiesByID = await getStaticRecordsByID<IPublicCompanyName>(apiRoute, [companyID], {
    cacheNamespace: `${apiRoute}:${Env.enviroment || '000000'}`,
    databaseCompanyID: 0,
  })
  return companiesByID.get(companyID)
}

const makeRamdomString = (len?: number) => {
	return "123412341234123412341234123412341234".substring(0, len || 32)
}

export const sendUserLogin = async (data: ILogin): Promise<any> => {
  let loginInfo: ILoginResult
  data.CipherKey = makeRamdomString(32)

  console.log(data)
  
  try {
    loginInfo = await POST({
      data,
      route: `p-user-login`,
      apiName: 'MAIN',
      headers: { "Content-Type": "application/json" }
    })
  } catch (error) {
    console.log(error)
    return { error }
	}

	console.log("loginInfo", loginInfo)

  try {
		await security.parseLogin(loginInfo, data.CipherKey)
		const hasValidToken = security.isTokenValid()
		console.log("hasValidToken", hasValidToken)
		
		if (!hasValidToken) {
      security.clearSession()
    } else {
      Env.navigate(loginInfo.InitialDataPending ? "/initial-data" : "/")
    }
  } catch (error) {
    console.log("error encriptando::")
    console.log(error)
  }

  return { result: loginInfo }
}

export const reloadLogin = async (): Promise<any> => {
  let loginInfo: ILoginResult
  const CipherKey = makeRamdomString(32)

  try {
    loginInfo = await GET({
      route: `reload-login?cipher-key=${CipherKey}`,
      headers: { "Content-Type": "application/json" }
    })
  } catch (error) {
    console.log(error)
    return { error }
  }

  try {
    await security.parseLogin(loginInfo, CipherKey)
    if(!security.isTokenValid()){
      security.clearSession()
    }
  } catch (error) {
    console.log("error encriptando::")
    console.log(error)
  }

  return { result: loginInfo }
}

// applyDevLogin mints a password-less session for a "<companyID>:<userID>" pair through the
// p-dev-login route, which the backend only answers with is_local and from loopback. It exists so
// the headless dev browser (scripts/agent_browser) can attach to the app as any user: no test
// user's password is stored anywhere, so no automated tool could otherwise reach a logged-in page.
//
// Hydration goes through the same parseLogin as the real login on purpose — the session lives in
// six localStorage keys, one of them checksum-wrapped, and a second path to write them would
// eventually drift from the first.
export const applyDevLogin = async (companyAndUser: string): Promise<void> => {
  const [companyID, userID] = companyAndUser.split(':')
  const CipherKey = makeRamdomString(32)

  try {
    const loginInfo: ILoginResult = await GET({
      route: `p-dev-login?company=${companyID || 1}&user=${userID || 1}&cipher-key=${CipherKey}`,
      headers: { "Content-Type": "application/json" }
    })
    await security.parseLogin(loginInfo, CipherKey)
    if (!security.isTokenValid()) {
      security.clearSession()
      throw new Error("el token emitido por p-dev-login no es válido")
    }
    console.log("[dev-login] sesión iniciada", { companyID, userID })
  } catch (error) {
    console.error("[dev-login] no se pudo iniciar la sesión de desarrollo", error)
  }
}

// Register the reload function with core to avoid circular dependency
security.setSessionRefresher(reloadLogin);

export const handleLogin = (login: ILoginResult) => {
  // Additional login handling if needed
}
