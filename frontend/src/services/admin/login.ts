import { POST } from "~/shared/http"
import { makeRamdomString } from "~/shared/main"
import { accessHelper, checkIsLogin, Env, setIsLogin_ } from "~/shared/security"

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

export async function sendUserLogin(data: ILogin): Promise<any> {
  let result: ILoginResult
  data.CipherKey = makeRamdomString(32)

  try {
    result = await POST({
      data, 
      route: `p-user-login`, apiName: 'MAIN',
      headers: { "Content-Type": "application/json" }
    })
  } catch (error) {
    console.log(error)
    return { error }
  }

  let userInfo = ""
  try {
    await accessHelper.parseAccesos(result, data.CipherKey)
    if(!accessHelper.checkAcceso(1)){
      Env.clearAccesos()
    } else {
      setIsLogin_(checkIsLogin())
      Env.navigate("/")
    }
  } catch (error) {
    console.log("error encriptando::")
    console.log(error)
  }
  
  console.log(userInfo)

  return { result }
}

export const handleLogin = (login: ILoginResult) => {



}