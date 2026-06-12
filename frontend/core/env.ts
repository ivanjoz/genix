declare global {
  var _isLocal: boolean;
}

import { browser } from '$app/environment';
import { goto } from '$app/navigation';
import { PUBLIC_ENDPOINTS, PUBLIC_FRONTEND_CDN, PUBLIC_LAMBDA_URL } from '$env/static/public';
export { browser };

export const IsClient = () => {
  return browser
}

const version = 1.11
console.log(version)
const selectedApiEndpointStorageKey = "genixSelectedApiEndpointRoute";

// Per-company storefront builds (prerender) pin the tenant via VITE_COMPANY_ID. These
// deploys have no login/endpoint selector: the API must always be PUBLIC_LAMBDA_URL,
// never the localStorage selection or the "Local" (localhost) option that gets added
// when the static build is previewed on localhost.
const isPrerenderStorefront = !!Number(import.meta.env.VITE_COMPANY_ID || 0)

if(browser){
  const host = window.location.host
  if((host.includes("localhost") || host.includes("127.0.0.1")) && host !== "localhost:8000"){
    globalThis._isLocal = true
  }
}

export interface IApiEndpointOption { name: string, route: string, hash: string }

const parsePublicApiEndpoints = (serializedEndpoints: string): IApiEndpointOption[] => {
  try {
    const rawEndpoints = JSON.parse(serializedEndpoints || "[]") as Partial<IApiEndpointOption>[]
		if (!Array.isArray(rawEndpoints)) { return [] }

    const parsedEndpoints = rawEndpoints
      .filter(endpointOption => endpointOption.name && endpointOption.route)
      // Only explicit credentials endpoints should be visible in the login selector.
      .map((endpointOption) => ({
        name: String(endpointOption.name || "").trim(),
        route: String(endpointOption.route || "").trim(),
        hash: ""
      }))

    console.info("[Env] Parsed API endpoints from PUBLIC_ENDPOINTS:", {
      configuredRoutes: parsedEndpoints.map((endpointOption) => endpointOption.route),
      configuredNames: parsedEndpoints.map((endpointOption) => endpointOption.name),
      lambdaUrl: PUBLIC_LAMBDA_URL || ""
    })

    // Never offer the localhost endpoint in a pinned storefront build, even when the
    // static output is previewed on localhost.
    if (globalThis._isLocal && !isPrerenderStorefront) {
      parsedEndpoints.unshift({ name: "Local", route: "http://localhost:3589/", hash: "" })
    }

    const usedHashes = new Set<string>()
    const endpointHashLength = 6
    const endpointHashSpace = 36 ** endpointHashLength

    // Keep the environment namespace short and stable across reloads.
		for (const endpointOption of parsedEndpoints) {
      let hashAccumulator = 0
      for (const char of endpointOption.route as string) {
        hashAccumulator = ((hashAccumulator * 31) + char.charCodeAt(0)) >>> 0
      }

      let hashValue = hashAccumulator % endpointHashSpace
      endpointOption.hash = hashValue.toString(36).padStart(endpointHashLength, "0").slice(-endpointHashLength)

      while (usedHashes.has(endpointOption.hash)) {
        hashValue = (hashValue + 1) % endpointHashSpace
        endpointOption.hash = hashValue.toString(36).padStart(endpointHashLength, "0").slice(-endpointHashLength)
      }
      usedHashes.add(endpointOption.hash)
		}

		return parsedEndpoints as IApiEndpointOption[]
  } catch (parseError) {
    console.error("[Env] Could not parse PUBLIC_ENDPOINTS:", parseError)
    return []
  }
}

const buildMainApiRoute = (baseRoute: string): string => {
  const normalizedBaseRoute = String(baseRoute || "").trim().replace(/\/+$/, "")
  if (!normalizedBaseRoute) { return "/api/" }
  return `${normalizedBaseRoute}/api/`
}

const ENPOINTS = parsePublicApiEndpoints(PUBLIC_ENDPOINTS || "")

const getSelectedApiEndpointRoute = (): string => {
  // Storefront build: always PUBLIC_LAMBDA_URL — ignore localStorage and any "Local" option.
  if (isPrerenderStorefront) { return PUBLIC_LAMBDA_URL || (ENPOINTS[0]?.route || "") }
  const endpointRoute = browser ? localStorage.getItem(selectedApiEndpointStorageKey) || "" : ""
  const persistedEndpointExists = ENPOINTS.some((endpointOption) => endpointOption.route === endpointRoute)
  return persistedEndpointExists ? endpointRoute : (ENPOINTS[0]?.route || "")
}

const getSelectedApiEndpoint = (selectedRoute: string): IApiEndpointOption => {
  const matchedEndpoint = ENPOINTS.find((endpointOption) => endpointOption.route === selectedRoute)
  if (matchedEndpoint) { return matchedEndpoint }
  // Storefront build pins PUBLIC_LAMBDA_URL even if it isn't one of PUBLIC_ENDPOINTS,
  // so synthesize an option for it rather than falling back to ENPOINTS[0].
  if (isPrerenderStorefront && selectedRoute) {
    return { name: "Lambda", route: selectedRoute, hash: "000000" }
  }
  return ENPOINTS[0] || {
    name: "",
    route: "",
    hash: "000000"
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

export interface ICompanyParams {
  CulqiLlave: string
  Name: string
  id: number
}

export const Env = {
  appId: "genix",
  CDN_URL: PUBLIC_FRONTEND_CDN,
  serviceWorker: "/sw.js",
  enviroment: getSelectedApiEndpoint(getSelectedApiEndpointRoute()).hash,
  counterID: 1,
  useTopMinimalMenu: false,
  fetchID: 1000,
  imageWorker: null as unknown as Worker,
  ImageWorkerClass: null as any,
  zoneOffset: (new Date()).getTimezoneOffset() * 60,
  dexieVersion: 1,
  cache: {} as {[e: string]: any},
  params: { fetchID: 1001, fetchProcesses: new Map() },
  pendingRequests: [] as any[],
  availableApiEndpoints: ENPOINTS,
  selectedApiEndpointRoute: getSelectedApiEndpointRoute(),
	API_ROUTES: {
		MAIN: buildMainApiRoute(getSelectedApiEndpointRoute())
	} as { [e: string]: string },
  screen: browser ? window.screen : { height: -1, width: -1 },
  language: browser ? window.navigator?.language || "" : "-",
  deviceMemory: browser ? (window.navigator as any)?.deviceMemory || 0 : 0,
  throttleTimer: null as NodeJS.Timeout | null,
  hostname: "",
  pathname: "",
  companyID: 0,
  empresa: {} as ICompanyParams,
  imageCounter: 10000,
  // Product-search debug and telemetry logs are centralized here.
  PRODUCT_SEARCH_FULL_DEBUG_LOG_ENABLED: false,
  // Top product search strategy: true → server text-search endpoint (p-product-text-search) + by-id
  // hydration; false → download the catalog snapshot/delta once and filter in memory (no per-id fetch).
  useLiveProductSearch: false,
  // Route-cache verification is expensive on mobile, so keep it opt-in from the client.
  DELTA_CACHE_VERIFY_ROUTE_MEMORY: false,
  clearAccesos: null as (() => void) | null,
  getToken: () => "",
  canUserAccessRoute: (_routeValue?: string | null) => true,
  navigate: goto,
  setSelectedApiEndpoint: (selectedRoute: string) => {
    const selectedEndpointOption = getSelectedApiEndpoint(selectedRoute)
    Env.selectedApiEndpointRoute = selectedEndpointOption?.route || ""
    Env.API_ROUTES.MAIN = buildMainApiRoute(Env.selectedApiEndpointRoute)
    Env.enviroment = selectedEndpointOption?.hash || "000000"

    if (browser && Env.selectedApiEndpointRoute) {
      localStorage.setItem(selectedApiEndpointStorageKey, Env.selectedApiEndpointRoute)
    }

    console.info("[Env] API endpoint selected:", {
      route: Env.selectedApiEndpointRoute,
      enviroment: Env.enviroment
    })
    return Env.API_ROUTES.MAIN
  },
  getPathname: () => {
    if(browser){ return window.location.pathname }
    return Env.pathname || ""
  },
  getCompanyID: (): number => {
    if(!Env.companyID){
      // Prerender/static build pins the tenant at build time (one build per company).
      // Vite inlines VITE_COMPANY_ID, so it resolves both in Node (build/SSR) and in
      // the deployed client bundle. Undefined in the admin app → falls through.
      const buildCompanyID = Number(import.meta.env.VITE_COMPANY_ID || 0)
      if(buildCompanyID){ return Env.companyID = buildCompanyID }

      const localCompanyID = browser ? localStorage.getItem(Env.appId + "CompanyID") : null
      if(localCompanyID){
        Env.companyID = parseInt(localCompanyID)
        return Env.companyID
      }

      let pathname = ""
      if(browser){
        pathname = document.head.querySelector(`meta[name="loc"]`)?.getAttribute("content") || ""
      }
      if(!pathname){ pathname = Env.getPathname() }
      pathname = pathname.replace(".html","")
      const paths = pathname.split("/").filter(x => x)
      if(paths[1] && paths[1].includes("-")){
        let companyID = paths[1].split("-")[0]
        if(!isNaN(companyID as unknown as number)){
          Env.companyID = parseInt(companyID)
        }
      } else if(!isNaN(paths[1] as unknown as number)){
        return Env.companyID = parseInt(paths[1])
      }
    }
    return Env.companyID
  },
  loadEmpresaConfig: () => {
    if(Env.empresa.id){ return }
    const companyID = Env.getCompanyID()
    if(companyID){
      fetch(Env.makeCDNRoute("empresas", `e-${companyID}.json`))
      .then(res => res.json())
      .then(res => {
        Env.empresa = res
      })
    } else {
      console.warn("No se encontró la empresa-id:", companyID)
    }
  },
  makeRoute: (route: string) => {
    const api = Env.API_ROUTES.MAIN
    if(route[0] === "/"){ route = route.substring(1) }
    const sep = route.includes("?") ? "&" : "?"
    return api + route + sep + `company-id=${Env.getCompanyID()}`
	},
	makeCDNRoute: (...segments: string[]) => {
		return makeRoute(Env.CDN_URL, ...segments)
	},
	componentIDCounter: 0,
	getComponentID: () => {
		Env.componentIDCounter++
		return Env.componentIDCounter
  }
}

Env.setSelectedApiEndpoint(Env.selectedApiEndpointRoute)

export const getInnerWidth = () => {
  if(browser){ return window.innerWidth }
  else { return 1200 }
}

export const makeRoute = (domain: string, ...segments: string[]) => {
	let prefix = "https"
	if (domain.includes("://")) {
		prefix = domain.split("://")[0]
		domain = domain.split("://")[1]
	}
	let route = [domain, ...segments]
		.filter(x => x).join("/").replaceAll(`//`, "/")
	
	console.debug("Route generated:", route)
	return prefix + "://" + route
}

export const LocalStorage = typeof window !== 'undefined'
  ? window.localStorage
  : {
      getItem: (k: string) => { return "" },
      setItem: (k: string, v: string) => { return "" },
      removeItem: (k: string) => { return "" }
    }
