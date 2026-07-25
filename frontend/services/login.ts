import { GET, POST } from '$libs/http.svelte';
import { security } from '$libs/ui-runtime.svelte';
import type { IUser, ILoginResult } from '$core/types/common';
import { Env } from '$core/env';

export interface ILogin {
  CompanyID: number
  User: string
  Password: string
  CipherKey: string
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
      Env.navigate("/")
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

// Register the reload function with core to avoid circular dependency
security.setSessionRefresher(reloadLogin);

export const handleLogin = (login: ILoginResult) => {
  // Additional login handling if needed
}
