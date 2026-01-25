declare global {
  var _isLocal: boolean;
}

import { browser } from '$app/environment';
import { goto } from '$app/navigation';

export { browser };

export const IsClient = () => {
  return browser
}

const apiPrd = "https://dnh72xkkh3junf57p3vexemlvm0emgys.lambda-url.us-east-1.on.aws/api/"
const apiLocal = "http://localhost:3589/api/"

if(browser){
  const host = window.location.host
  if((host.includes("localhost") || host.includes("127.0.0.1")) && host !== "localhost:8000"){
    globalThis._isLocal = true
  }
}

export const getWindow = () => {
  if(browser){ return window }
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
  serviceWorker: "/sw.js",
  enviroment: "dev",
  counterID: 1,
  sideLayerSize: 0,
  fetchID: 1000,
  imageWorker: null as unknown as Worker,
  ImageWorkerClass: null as any,
  zoneOffset: (new Date()).getTimezoneOffset() * 60,
  dexieVersion: 1,
  cache: {} as {[e: string]: any},
  params: { fetchID: 1001, fetchProcesses: new Map() },
  pendingRequests: [] as any[],
  API_ROUTES: { MAIN: globalThis._isLocal ? apiLocal : apiPrd } as {[e: string]: string},
  screen: browser ? window.screen : { height: -1, width: -1 },
  language: browser ? window.navigator?.language || "" : "-",
  deviceMemory: browser ? (window.navigator as any)?.deviceMemory || 0 : 0,
  throttleTimer: null as NodeJS.Timeout | null,
  hostname: "",
  pathname: "",
  empresaID: 0,
  empresa: {} as IEmpresaParams,
  imageCounter: 10000,
  clearAccesos: null as (() => void) | null,
  navigate: goto,
  history: {
    pushState: (data: any, unused: string, url?: string | URL | null) => {
      console.log("Es server!!", data, unused, url)
    }
  },
  getPathname: () => {
    if(browser){ return window.location.pathname }
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
      if(browser){
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

export const getInnerWidth = () => {
  if(browser){ return window.innerWidth }
  else { return 1200 }
}

export const LocalStorage = typeof window !== 'undefined'
  ? window.localStorage
  : {
      getItem: (k: string) => { return "" },
      setItem: (k: string, v: string) => { return "" },
      removeItem: (k: string) => { return "" }
    }
