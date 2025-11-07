import { Notify } from "../core/helpers"
import { accessHelper, Env, getToken } from "./security"

export interface httpProps {
  data?: any
  route: string
  apiName?: string
  headers?: {[key: string]: string}
  successMessage?: string
  errorMessage?: string
}

interface IHttpStatus { 
  code: number
  message: string 
}

export const makeRoute = (route: string, apiName?: string) => {
  const apiUrl = apiName ? Env.API_ROUTES[apiName] : Env.API_ROUTES.MAIN
  return route.includes('://') ? route : apiUrl + route
}

export const buildHeaders = (props: httpProps, contentType?: string) => {
  const cTs: {[e: string]: string } = { "json": "application/json" }

  const fromHeader = [
    location.pathname,
    (new Date()).getTimezoneOffset(),
    Env.language || "",
    [`${Env.screen.height}x${Env.screen.width}`,
    `ram:${Env.deviceMemory || 0}`,
    ].join("_"),
  ].join("|")

  if (props.headers) {
    props.headers['x-api-key'] = fromHeader
    return props.headers
  }
  const headers = new Headers()

  if (contentType && cTs[contentType]) {
    headers.append('Content-Type', cTs[contentType])
  }
  headers.append('Authorization', `Bearer ${getToken()}`)
  // headers.append('x-api-key', fromHeader)
  return headers
}

const extractError = (result: any): string => {
  let errorJson
  let errorString = ""

  if(typeof result === 'string'){
    errorString = result.trim()
    if(errorString[0] === "{" || errorString[0] === "["){
      try {
        errorJson = JSON.parse(errorString)
      } catch {}
    }
  } else {
    errorJson = result
  }
  if(errorJson){
    if(Array.isArray(errorJson)){
      errorJson = errorJson[0]
    }
    if(errorJson.message || errorJson.error || errorJson.errorMessage){
      errorJson = errorJson.message || errorJson.error || errorJson.errorMessage
    }
    errorString = typeof errorJson === 'string' 
      ?  errorJson 
      : JSON.stringify(errorJson)
  }
  return errorString
}

const checkErrorResponse = (result: any, status: IHttpStatus) => {
  if (!status.code || status.code !== 200 || result.errorMessage) {
    console.warn(result)
    Notify.failure(extractError(result))
    return false
  } else {
    return true
  }
}

// Parsea los headers de la respuesta antes parseado el body
const parsePreResponse = (res: any, status: IHttpStatus): Promise<any> => {
  const contentType = res.headers.get("content-type")
  if (res.status) {
    status.code = res.status
    status.message = res.statusText
  }
  if (res.status === 200) { return res.json() }
  else if (res.status === 401) {
    accessHelper.clearAccesos?.()
    console.warn('Error 401, la sesión ha expirado.')
    Notify.failure('La sesión ha expirado, vuelva a iniciar sesión.')
  }
  else if (res.status !== 200) {
    if (!contentType || contentType.indexOf("/json") === -1) {
      return res.text()
    } else {
      return res.json()
    }
  }
}

// Parsea el body de la respuesta
function parseResponseBody(res: any, props: httpProps, status: IHttpStatus) {
  if (!res) { res = "Hubo un error desconocido en el servidor" }
  // Revisa si es un objeto
  else if (typeof res === 'string') { try { res = JSON.parse(res) } catch { } }
  // Revisa el Status Code
  if (!checkErrorResponse(res, status)) return false

  if (props.successMessage) Notify.success(props.successMessage)
  return true
}

const POST_PUT = (props: httpProps, method: string): Promise<any> => {
  const data = props.data
  if (typeof data !== 'object') {
    const err = 'The data provided is not a JSON'
    console.error(err)
    return Promise.reject(err)
  }
  
  const status: IHttpStatus = { code: 200, message: "" }
  const apiRoute = makeRoute(props.route, props.apiName)

  return new Promise((resolve, reject) => {
    console.log(`Fetching ${method} : ` + props.route)

    fetch(apiRoute, {
      method: method,
      headers: buildHeaders(props, 'json'),
      body: JSON.stringify(data)
    })
      .then(res => parsePreResponse(res, status))
      .then(res => {
        parseResponseBody(res, props, status) ? resolve(res) : reject(res)
      })
      .catch(error => {
        console.log('error::', error)
        if (props.errorMessage) {
          Notify.failure(props.errorMessage)
        } else {
          Notify.failure(String(error))
        }
        reject(error)
      })
  })
}

export function POST(props: httpProps) {
  return POST_PUT(props, 'POST')
}

export function PUT(props: httpProps) {
  return POST_PUT(props, 'PUT')
}

export function GET(props: httpProps): Promise<any> {
  const status: IHttpStatus = { code: 200, message: "" }
  const route = makeRoute(props.route, props.apiName)
  
  return new Promise((resolve, reject) => {
    console.log("realizando fetch::", props)
    fetch(route, { headers: buildHeaders(props) })
      .then(res => parsePreResponse(res, status))
      .then(res => {
        return parseResponseBody(res, props, status) ? resolve(res) : reject(res)
      })
      .catch(error => {
        console.warn(error)
        if (props.errorMessage) { Notify.failure(props.errorMessage) }
        reject(error)
      })
  })
}

export class GetHandler {

  route = ""
  useCache: { min: number, ver: number  } | undefined = undefined

  handler(e: any){

  }

  isTest: boolean = false
  Test(){
    alert(this.route)
    
    setTimeout(() => {
      this.handler({ message: "Message 1" })
      setTimeout(() => {
        this.handler({ message: "Message 2" })
      },1000)
    },1000)
  }

  fetch(){

  }
}