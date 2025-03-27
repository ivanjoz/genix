import { Navigator } from "@solidjs/router/dist/types.js"

export const IsClient = () => {
  return typeof window !== 'undefined'
}
const isClient = IsClient()
let api = "https://dnh72xkkh3junf57p3vexemlvm0emgys.lambda-url.us-east-1.on.aws/api/"

// const DEV_HOSTS = ["d16qwm950j0pjf.cloudfront.net","genix-dev.un.pe"]
if(isClient){
  if(window.location.host.includes("localhost") && window.location.host !== "localhost:8000"){
    api = "http://localhost:3589/api/"
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

export interface IEmpresaParams {
  CulqiLlave: string
  Nombre: string
  id: number
}

export const Env = {
  appId: "genix",
  S3_URL: "https://d16qwm950j0pjf.cloudfront.net/",
  counterID: 1,
  zoneOffset: (new Date()).getTimezoneOffset() * 60,
  dexieVersion: 1,
  cache: {} as {[e: string]: any},
  params: { fetchID: 1001, fetchProcesses: new Map() },
  pendingRequests: [] as any[],
  API_ROUTES: { MAIN: api } as {[e: string]: string},
  screen: isClient ? window.screen : { height: -1, width: -1 },
  language: isClient ? window.navigator?.language || "" : "-",
  deviceMemory: isClient ? (window.navigator as any)?.deviceMemory || 0 : 0,
  throttleTimer: null as NodeJS.Timeout,
  hostname: "",
  pathname: "",
  empresaID: 0,
  empresa: {} as IEmpresaParams,
  clearAccesos: null as (() => void),
  navigate: null as Navigator, 
  history: {
    pushState: (data: any, unused: string, url?: string | URL | null) => {
      console.log("Es server!!", data, unused, url)
    }
  },
  getPathname: () => {
    if(isClient){ return window.location.pathname }
    return Env.pathname || ""
  },
  getEmpresaID: (): number => {
    if(!Env.empresaID){ 
      const localEmpresaID = localStorage.getItem(Env.appId + "EmpresaID")
      if(localEmpresaID){
        Env.empresaID = parseInt(localEmpresaID)
        return Env.empresaID
      }

      let pathname = ""
      if(isClient){
        pathname = document.head.querySelector(`meta[name="loc"]`)?.getAttribute("content") || ""
      }
      if(!pathname){ pathname = Env.getPathname() }
      pathname = pathname.replace(".html","")
      const paths = pathname.split("/").filter(x => x)
      if(paths[1] && paths[1].includes("-")){
        let empresaID = paths[1].split("-")[0]
        if(!isNaN(empresaID as unknown as number)){ 
          Env.empresaID = parseInt(empresaID)
        }
      } else if(!isNaN(paths[1] as unknown as number)){
        return Env.empresaID = parseInt(paths[1])
      }
    }
    return Env.empresaID
  },
  loadEmpresaConfig: () => {
    if(Env.empresa.id){ return }
    const empresaID = Env.getEmpresaID()
    if(empresaID){
      fetch(Env.S3_URL +`empresas/e-${empresaID}.json`)
      .then(res => res.json())
      .then(res => {
        Env.empresa = res
      })
    } else {
      console.warn("No se encontró la empresa-id:", empresaID)
    }
  },
  // Extra
  productoSearchRefocusOnBlur: false,
  closeProductosSearchLayer: null as () => void
}

export const getInnerWidth = () => {
  if(isClient){ return window.innerWidth }
  else { return 1200 }
}

export const LocalStorage = typeof window !== 'undefined' 
  ? window.localStorage
  : {
      getItem: (k: string) => { return "" },
      setItem: (k: string, v: string) => { return "" },
      removeItem: (k: string) => { return "" }
    }


    