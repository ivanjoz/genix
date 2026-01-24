export const IsClient = () => {
  return typeof window !== 'undefined'
}
const isClient = IsClient()
const apiPrd = "https://dnh72xkkh3junf57p3vexemlvm0emgys.lambda-url.us-east-1.on.aws/api/"
const apiLocal = "http://localhost:3589/api/"

// const DEV_HOSTS = ["d16qwm950j0pjf.cloudfront.net","genix-dev.un.pe"]
if(isClient){
  if(window.location.host.includes("localhost") && window.location.host !== "localhost:8000"){
    globalThis._isLocal = true
  }
}

export const getWindow = () => {
  if(isClient){ return window }
  else return {
    scrollY: 0,
    innerHeight: 800,
    innerWidth: 1200,
    addEventListener: (event: string, func: () => {}) => {
      console.log("No es window!. Event:", event)
    }
  } as Window
}

export const Env = {
  appId: "genix",
  S3_URL: "https://d16qwm950j0pjf.cloudfront.net/",
  api: globalThis._isLocal ? apiLocal : apiPrd,
  empresaID: 1,
  counterID: 1,
  zoneOffset: (new Date()).getTimezoneOffset() * 60,
  cache: {} as {[e: string]: any},
  params: { fetchID: 1001, fetchProcesses: new Map() },
  pendingRequests: [] as any[],
  makeRoute: (route: string) => {
    const api = globalThis._isLocal ? apiLocal : apiPrd
    if(route[0] === "/"){ route = route.substring(1) }
    const sep = route.includes("?") ? "&" : "?"
    return api + route + sep + `empresa-id=${Env.empresaID}`
  },
  makeImageRoute: (route: string) => {
    if(route.substring(0,6) !== "http//" && route.substring(0,7) !== "https//"){
      route = Env.S3_URL + "img-productos/" + route
    }
    return route
  }
}