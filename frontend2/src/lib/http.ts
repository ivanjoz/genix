import { Notify } from "../core/helpers"
import type { IFetchEvent } from "../core/store.svelte"
import { formatN } from "../shared/main"
import type { CacheMode, serviceHttpProps } from "../workers/service-worker"
import { accessHelper, Env, getToken } from "./security"
import { fetchCache } from "./sw-cache"

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

export const makeRoute = (route: string) => {
  const apiUrl = Env.apiRoute
  return route.includes('://') ? route : apiUrl + route
}

export const buildHeaders = (contentType?: string) => {
  const cTs: {[e: string]: string } = { "json": "application/json" }
  /*
  const fromHeader = [
    location.pathname,
    (new Date()).getTimezoneOffset(),
    Env.language || "",
    [`${Env.screen.height}x${Env.screen.width}`,
    `ram:${Env.deviceMemory || 0}`,
    ].join("_"),
  ].join("|")
  */
  const headers = {} as {[k:string]: string}

  if (contentType && cTs[contentType]) {
    headers['Content-Type'] = cTs[contentType]
  }
  headers['Authorization'] = `Bearer ${getToken()}`
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
      headers: buildHeaders('json'),
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


let progressLastTime = 0
let progressTimeStart = 0
let progressBytes = 0
let fetchOnCourse = 0

export const setFetchProgress = (bytesLen: number) => {
  const nowTime = Date.now()
  if(!progressBytes){ 
    progressTimeStart = nowTime  
  }

  progressLastTime = nowTime
  progressBytes += bytesLen

  let mbps = 0
  const kb = progressBytes/1000
  const elapsed = nowTime - progressTimeStart

  if(elapsed > 50){
    mbps = kb / elapsed
  }
  
  let msg = `Descargando... ${formatN(kb)} kb`
  if(mbps){
    if(mbps > 10){ mbps = 10 }
    msg += ` (${formatN(mbps,2)} MB/s)`
  }

  const loadingMsgDiv = document.getElementById("NotiflixLoadingMessage")
  if(loadingMsgDiv){
    let nextElement = loadingMsgDiv.nextElementSibling
    if(!nextElement){
      nextElement = document.createElement("div")
      nextElement.setAttribute("id","NotifyProgressMessage")
      loadingMsgDiv.parentNode.insertBefore(nextElement, loadingMsgDiv.nextSibling)
    }
    nextElement.innerHTML = msg
  }
}


// Parsea los headers de la respuesta crear un reader
const parseResponseAsStream = async (fetchResponse: Response, status: any, props?: httpProps) => {

  const contentType = fetchResponse.headers.get("Content-Type")

  if (fetchResponse.status) {
    status.code = fetchResponse.status
    status.message = fetchResponse.statusText
  }

  if (fetchResponse.status === 200) { 
    const reader = fetchResponse.body.getReader()
    const stream = new ReadableStream({
      start(controller) {
        fetchOnCourse++
        return pump()
        function pump() {
          return reader.read().then(({ done, value }) => {
            // When no more data needs to be consumed, close the stream
            if (done) {
              controller.close()
              fetchOnCourse--
              if(fetchOnCourse <= 0){ progressBytes = 0 }
              return
            }
            // console.log("chunk obtenido:: ", value.length)
            setFetchProgress(value.length)
            // Enqueue the next data chunk into our target stream
            controller.enqueue(value)
            return pump()
          })
        }
      },
    })
    const responseStream = new Response(stream)
    const responseText = await responseStream.text()
    if(props){ props.contentLength = responseText.length }
    return Promise.resolve(JSON.parse(responseText))
  }
  else if (fetchResponse.status === 401) {
    document.dispatchEvent(new Event('userLogout'))
    console.warn('Error 401, la sesión ha expirado.')
    Notify.failure('La sesión ha expirado, vuelva a iniciar sesión.')
    Notify.remove()
  }
  else if (fetchResponse.status !== 200) {
    console.log(fetchResponse)
    if (!contentType || contentType.indexOf("/json") === -1) {
      console.log('Parseando como texto')
      return fetchResponse.text()
    } else {
      console.log('parseando como JSON')
      return fetchResponse.json()
    }
  }
}

export function GET(props: httpProps): Promise<any> {
  const status: IHttpStatus = { code: 200, message: "" }
  const route = makeRoute(props.route)
  
  return new Promise((resolve, reject) => {
    console.log("realizando fetch::", props)
    fetch(route, { headers: buildHeaders() })
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
  routeParsed = ""
  module = "a"

  useCache: { min: number, ver: number  } | undefined = undefined
  headers: { [k: string]: string } | undefined = undefined

  handler(e: any){}

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

  makeProps(cacheMode: CacheMode): serviceHttpProps {
    const props = {
      routeParsed: makeRoute(this.route),
      route: this.route,
      useCache: this.useCache,
      module: this.module,
      headers: buildHeaders('json'),
      cacheMode,
    } as serviceHttpProps
    return props
  }

  fetch(){
    if(this.route.length === 0){
      Notify.failure("No se especificó el route en productos.")
      return
    }

    fetchCache(this.makeProps('offline'))
    .then(cachedResponse => {
      if(cachedResponse){
        this.handler(cachedResponse)
      }
      return fetchCache(this.makeProps('refresh'))
    })
    .then(fetchedResponse => {
      if(fetchedResponse){
        this.handler(fetchedResponse)
      }
    })
  }
}