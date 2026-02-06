import { GET, POST } from '$core/http.svelte';
import { accessHelper, checkIsLogin, registerReloadLogin } from '$core/lib/security';
import type { IUsuario, ILoginResult } from '$core/types/common';
import { Env } from '$core/env';

export interface ILogin {
  EmpresaID: number
  Usuario: string
  Password: string
  CipherKey: string
}

const makeRamdomString = (len?: number) => {
	return "123412341234123412341234123412341234".substring(0, len || 32)
}

export const sendUserLogin = async (data: ILogin): Promise<any> => {
  let loginInfo: ILoginResult
  data.CipherKey = makeRamdomString(32)

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
    await accessHelper.parseAccesos(loginInfo, data.CipherKey)
    if(!accessHelper.checkAcceso(1)){
      Env.clearAccesos?.()
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
    await accessHelper.parseAccesos(loginInfo, CipherKey)
    if(!accessHelper.checkAcceso(1)){
      Env.clearAccesos?.()
    }
  } catch (error) {
    console.log("error encriptando::")
    console.log(error)
  }

  return { result: loginInfo }
}

// Register the reload function with core to avoid circular dependency
registerReloadLogin(reloadLogin);

export const handleLogin = (login: ILoginResult) => {
  // Additional login handling if needed
}
