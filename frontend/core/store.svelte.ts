import { goto } from '$app/navigation';
import type { IMenuRecord, IModule } from '$core/types/modules';
import { SvelteMap } from 'svelte/reactivity';
import { Env, browser } from './env';
import type { IImageResult } from './types/common';

export const getDeviceType = () => {
  let view = 1 // Desktop
  if(!browser){ return view }
  if (window.innerWidth < 740) {
    view = 3 /* Mobile */
  } else if (window.innerWidth < 1140) { view = 2 /* Tablet */ }
  return view
}

// 1 = Spanish, 2 = English
export type ILanguaje = 1 | 2
const LANGUAJE_STORAGE_KEY = "genix_languaje"

const getStoredLanguaje = (): ILanguaje => {
  if(!browser){ return 1 }
  return localStorage.getItem(LANGUAJE_STORAGE_KEY) === "2" ? 2 : 1
}

// Resolves a "english|spanish" string for the active languaje.
// parts[0] = English, parts[1] = Spanish. Text without a pipe is returned as-is.
// Non-string values (numbers, undefined, etc.) are returned untouched so the
// helper is safe to wrap around any rendered value (e.g. table headers).
export function tr<V>(text: V, languaje?: ILanguaje): V {
  const lang = languaje ?? Core.languaje
  if(typeof text !== "string" || !text.includes("|")){ return text }
  const parts = text.split("|")
  const idx = lang === 2 ? 0 : 1
  return ((parts[idx] ?? parts[0]).trim() as V)
}

export const setLanguaje = (languaje: ILanguaje) => {
  Core.languaje = languaje
  if(browser){ localStorage.setItem(LANGUAJE_STORAGE_KEY, String(languaje)) }
}

export interface ITopSearchLayer {
  options: any[]
  keyName: string
  keyID: string | number
  onSelect: (e: any) => void
  onClear?: () => void
  onRemove?: (e: any) => void
}

export interface ITopDateLayer {
  selectedUnixDay: number
  focusedUnixDay?: number
  selectedMonthKey: number
  label?: string
  placeholder?: string
  onSelect: (unixDay: number) => void
  onClose?: () => void
}

export const Core = $state({
  module: { menus: [] as IMenuRecord[] } as IModule,
  openSearchLayer: 0 as number,
  deviceType: getDeviceType() as number,
	mobileMenuOpen: 0 as number,
	// 1 = Spanish, 2 = English
  languaje: getStoredLanguaje() as ILanguaje,
  // Keep layout-critical layer width in reactive global state so UI runes can subscribe to it.
  sideLayerSize: 0 as number,
  useTopMinimalMenu: false as boolean,
  popoverShowID: 0 as number | string,
  showSideLayer: 0 as number,
  headerSettingsOpen: false as boolean,
  isLoading: 1,
  pageTitle: "" as string,
  openLayers: [] as number[],
  pageOptions: [] as {id: number, name: string}[],
  pageOptionSelected: 1,
  showMobileSearchLayer: null as ITopSearchLayer | null,
  showMobileDateLayer: null as ITopDateLayer | null,
  toggleMobileMenu: (() => {}) as () => void,
  openSideLayer: (layerId: number) => {
    Core.showSideLayer = layerId
  },
  hideSideLayer: (() => { Core.showSideLayer = 0 }) as () => void,
  openHeaderSettings: (() => { Core.headerSettingsOpen = true }) as () => void,
  closeHeaderSettings: (() => { Core.headerSettingsOpen = false }) as () => void,
  openModal: (id: number) => {
    if (!openModals.includes(id)) { openModals.push(id); }
  },
  closeModal: (id: number) => {
    const index = openModals.indexOf(id);
    if (index > -1) { openModals.splice(index, 1); }
  },
  // Ecommerce state moved from pkg-store
  ecommerce: {
    cartOption: 1
  }
})

export const WeakSearchRef: WeakMap<any,{
  idToRecord: Map<string|number, any>
  valueToRecord: Map<string,any>
}> = new WeakMap()


export interface IFetchEvent {
  url: string
}

export const fetchOnCourse = $state<Map<number,IFetchEvent>>(new SvelteMap())

export const fetchEvent = (fetchID: number, props: IFetchEvent | 0) => {
  if (fetchID === 0) { // Para obtener un nuevo ID de FetchEvent
    Env.fetchID++
    return Env.fetchID
  }

  if (props === 0) { // Para eliminar el proceso del Map
    fetchOnCourse.delete(fetchID)
  }
  else { // Para setear el proceso
    fetchOnCourse.set(fetchID, props)
  }
}

// Global state for managing open modals
export const openModals = $state<number[]>([]);

// Helper functions to manage modals
export const openModal = (id: number) => {
	console.log("abriendo modal::",id, $state.snapshot(openModals))
  if (!openModals.includes(id)) { openModals.push(id); }
}

export const closeModal = (id: number) => {
  const index = openModals.indexOf(id);
  if (index > -1) { openModals.splice(index, 1); }
}

export const closeAllModals = () => {
  openModals.length = 0;
}

// Map to store images to upload (global state)
export const imagesToUpload = new Map<number, () => Promise<IImageResult>>();

// Store-specific exports
export const mainMenuOptions = [
  { name: "Iniciar Sesión",
    icon: "icon-user",
    onClick: () => {
      if(!Core.openLayers.includes(1)){
        Core.openLayers.push(1)
      }
    }
  },
  { name: "Regístrate",
    icon: "icon-doc",
  },
  { name: "Mis Pedidos",
    icon: "icon-box",
  },
  { name: "Tienda",
    icon: "icon-home",
  }
]

export interface NavFlag {
  id: number
  close: () => void
  updated?: number
  elementId: string
}

let navFlags: NavFlag[] = []
let navFlagCounter = 0
let navReturns = 0

export const suscribeUrlFlag = (elementId: string, callbackOnClose: (() => void)) => {
  navFlags = navFlags.filter(x => document.getElementById(x.elementId))

  let flag = navFlags.find(x => x.elementId === elementId)
  const isIncluded = !!flag

  if(!flag){
    flag = {
      id: navFlags.length + 1,
      close: callbackOnClose,
      elementId
    }
    navFlags.push(flag)
  }

  flag.updated = Date.now()

  let uriParams = window.location.search.substring(1).split("&").filter(x => x)
  const nf = (uriParams.find(x => x.substring(0,3) === "nf=")||"").replace("nf=","")

  if(navFlagCounter === 0 && nf){
    navFlagCounter = nf.split(",").map(x => parseInt(x))[0]
  }

  if(!isIncluded || navReturns > 0){
    navFlagCounter++
    if(navReturns > 0){ navReturns-- }

    uriParams = uriParams.filter(x => x.substring(0,3) !== "nf=")
    uriParams.push(`nf=${navFlagCounter},${navFlags.map(x => x.id).join(",")}`)

    goto(window.location.pathname +"?"+ uriParams.join("&"), { noScroll: true, replaceState: false })
  }
}

if(typeof window !== 'undefined'){
  window.addEventListener('popstate', () => {
    navFlags = navFlags.filter(x => document.getElementById(x.elementId))

    let flag: NavFlag | undefined = undefined;
    for(const e of navFlags){
      if(!flag || (e.updated ?? 0) > (flag.updated ?? 0)){
        flag = e
      }
    }
    if(flag){
      flag.close()
      navFlags = navFlags.filter(x => x.id !== flag.id)
    }
  })
}
