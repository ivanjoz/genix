import { GET, POST } from "../../lib/http"
import { makeRamdomString } from "../../shared/main"
import { accessHelper, checkIsLogin, Env } from "../../lib/security"

export interface ILogin {
  EmpresaID: number
  Usuario: string
  Password: string
  CipherKey: string
}
  
export interface ILoginResult {
  UserID: number
  UserNames: string
  UserEmail: string
  UserToken: string
  UserInfo: string
  TokenExpTime: number
  EmpresaID: number
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

  let userInfo = ""
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
  
  console.log(userInfo)

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

  let userInfo = ""
  try {
    await accessHelper.parseAccesos(loginInfo, CipherKey)
    if(!accessHelper.checkAcceso(1)){
      Env.clearAccesos?.()
    }
  } catch (error) {
    console.log("error encriptando::")
    console.log(error)
  }
  
  console.log(userInfo)
  return { result: loginInfo }
}

export const handleLogin = (login: ILoginResult) => {
  // Additional login handling if needed
}

